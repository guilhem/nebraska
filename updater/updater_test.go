package updater_test

import (
	"bytes"
	"context"
	"encoding/xml"
	"errors"
	"log"
	"os"
	"testing"

	omahaSpec "github.com/kinvolk/go-omaha/omaha"
	"github.com/kinvolk/nebraska/backend/pkg/api"
	"github.com/kinvolk/nebraska/backend/pkg/omaha"
	"github.com/kinvolk/nebraska/updater"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/guregu/null.v4"
)

const (
	defaultTestDbURL string = "postgres://postgres:nebraska@127.0.0.1:5432/nebraska_tests?sslmode=disable&connect_timeout=10"
)

type testOmahaHandler struct {
	handler *omaha.Handler
}

func newTestHandler(a *api.API) *testOmahaHandler {
	return &testOmahaHandler{
		handler: omaha.NewHandler(a),
	}
}

func (h *testOmahaHandler) Handle(req *omahaSpec.Request) (*omahaSpec.Response, error) {
	omahaReqXML, err := xml.Marshal(req)
	if err != nil {
		return nil, err
	}

	omahaRespXML := new(bytes.Buffer)
	if err = h.handler.Handle(bytes.NewReader(omahaReqXML), omahaRespXML, "0.1.0.0"); err != nil {
		return nil, err
	}

	var omahaResp *omahaSpec.Response
	err = xml.NewDecoder(omahaRespXML).Decode(&omahaResp)
	if err != nil {
		return nil, err
	}

	return omahaResp, nil
}

func newForTest(t *testing.T) *api.API {
	if _, ok := os.LookupEnv("NEBRASKA_DB_URL"); !ok {
		log.Printf("NEBRASKA_DB_URL not set, setting to default %q\n", defaultTestDbURL)
		_ = os.Setenv("NEBRASKA_DB_URL", defaultTestDbURL)
	}
	a, err := api.New(api.OptionInitDB)

	require.NoError(t, err)
	require.NotNil(t, a)

	return a
}

func TestNewUpdater(t *testing.T) {

	conf := updater.Config{
		OmahaURL:        "http://localhost:8000",
		AppID:           "io.phony.App",
		Channel:         "stable",
		InstanceID:      "instance001",
		InstanceVersion: "0.1.0",
	}

	_, err := updater.New(conf)
	assert.NoError(t, err)
}

func TestCheckForUpdates(t *testing.T) {
	a := newForTest(t)
	defer a.Close()

	appID, track, tChannel := setup(t, a, "0.1.0", true, 2)

	u, err := updater.New(updater.Config{
		OmahaURL:        "http://localhost:8000",
		AppID:           appID,
		Channel:         track,
		InstanceID:      "instance001",
		InstanceVersion: "0.2.0",
		OmahaReqHandler: newTestHandler(a),
	})
	require.NoError(t, err)

	info, err := u.CheckForUpdates(context.TODO())
	assert.NoError(t, err)
	assert.False(t, info.HasUpdate)
	assert.Equal(t, "", info.GetVersion())

	newPkg, _ := a.AddPackage(&api.Package{Type: api.PkgTypeOther, URL: "http://sample.url/pkg", Version: "0.3.0", ApplicationID: appID, Arch: api.ArchAMD64, Filename: null.StringFrom("updatefile.txt")})
	tChannel.PackageID = null.StringFrom(newPkg.ID)
	err = a.UpdateChannel(tChannel)
	assert.NoError(t, err)

	info, err = u.CheckForUpdates(context.TODO())
	assert.NoError(t, err)
	assert.True(t, info.HasUpdate)

	version := info.GetVersion()
	assert.Equal(t, "0.3.0", version)

	urls := info.GetURLs()
	assert.NotNil(t, urls)
	assert.Equal(t, 1, len(urls))
	assert.Equal(t, urls[len(urls)-1], info.GetURL())
	assert.Equal(t, "http://sample.url/pkg", info.GetURL())

	pkg := info.GetPackage()
	assert.NotNil(t, pkg)
	assert.Equal(t, "updatefile.txt", pkg.Name)
}

type updateTestHandler struct {
	fetchUpdateResult error
	applyUpdateResult error
}

func (u updateTestHandler) FetchUpdate(ctx context.Context, info *updater.UpdateInfo) error {
	return u.fetchUpdateResult
}

func (u updateTestHandler) ApplyUpdate(ctx context.Context, info *updater.UpdateInfo) error {
	return u.applyUpdateResult
}

func setup(t *testing.T, a *api.API, version string, policySafeMode bool, policyMaxUpdatesPerPeriod int) (string, string, *api.Channel) {
	t.Helper()
	tTeam, err := a.AddTeam(&api.Team{Name: "test_team"})
	require.NoError(t, err)
	tApp, err := a.AddApp(&api.Application{Name: "io.phony.App", TeamID: tTeam.ID})
	require.NoError(t, err)
	tPkg, err := a.AddPackage(&api.Package{Type: api.PkgTypeOther, URL: "http://sample.url/pkg", Version: version, ApplicationID: tApp.ID, Arch: api.ArchAMD64})
	require.NoError(t, err)
	tChannel, err := a.AddChannel(&api.Channel{Name: "channel1", Color: "blue", ApplicationID: tApp.ID, PackageID: null.StringFrom(tPkg.ID), Arch: api.ArchAMD64})
	require.NoError(t, err)
	tGroup, err := a.AddGroup(&api.Group{Name: "group1", ApplicationID: tApp.ID, ChannelID: null.StringFrom(tChannel.ID), PolicyUpdatesEnabled: true, PolicySafeMode: policySafeMode, PolicyPeriodInterval: "15 minutes", PolicyMaxUpdatesPerPeriod: policyMaxUpdatesPerPeriod, PolicyUpdateTimeout: "60 minutes", Track: "stable"})
	require.NoError(t, err)
	return tApp.ID, tGroup.Track, tChannel
}

func TestTryUpdate(t *testing.T) {
	a := newForTest(t)
	defer a.Close()

	oldVersion := "0.2.0"
	pkgVersion := "0.4.0"
	appID, track, _ := setup(t, a, pkgVersion, false, 10)

	type test struct {
		name              string
		fetchUpdateResult error
		applyUpdateResult error
		isErr             bool
	}

	tests := []test{
		{
			name:              "error fetching update",
			fetchUpdateResult: errors.New("something went wrong fetching the update"),
			applyUpdateResult: nil,
			isErr:             true,
		},
		{
			name:              "error applying update",
			fetchUpdateResult: nil,
			applyUpdateResult: errors.New("something went wrong fetching the update"),
			isErr:             true,
		}, {
			name:              "success try update",
			fetchUpdateResult: nil,
			applyUpdateResult: nil,
			isErr:             false,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {

			u, err := updater.New(updater.Config{
				OmahaURL:        "http://localhost:8000",
				AppID:           appID,
				Channel:         track,
				InstanceID:      "instance001",
				InstanceVersion: "0.2.0",
				OmahaReqHandler: newTestHandler(a),
				Debug:           true,
			})
			require.NoError(t, err)

			assert.Equal(t, oldVersion, u.GetInstanceVersion())

			err = u.TryUpdate(context.TODO(), &updateTestHandler{
				fetchUpdateResult: tc.fetchUpdateResult,
				applyUpdateResult: tc.applyUpdateResult,
			})
			if tc.isErr {
				assert.Error(t, err)
				assert.Equal(t, oldVersion, u.GetInstanceVersion())
			} else {
				assert.NoError(t, err)
				assert.Equal(t, pkgVersion, u.GetInstanceVersion())
			}
		})
	}
}
