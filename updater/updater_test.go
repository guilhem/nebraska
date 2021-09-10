package updater_test

import (
	"bytes"
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"log"
	"os"
	"testing"

	"github.com/google/uuid"
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

func newTestHandler(api *api.API) *testOmahaHandler {
	return &testOmahaHandler{
		handler: omaha.NewHandler(api),
	}
}

func (h *testOmahaHandler) Handle(ctx context.Context, url string, req *omahaSpec.Request) (*omahaSpec.Response, error) {
	requestBuf := bytes.NewBuffer(nil)
	encoder := xml.NewEncoder(requestBuf)
	err := encoder.Encode(req)
	if err != nil {
		return nil, err
	}

	omahaRespXML := new(bytes.Buffer)
	if err = h.handler.Handle(requestBuf, omahaRespXML, "0.1.0.0"); err != nil {
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
	api, err := api.New(api.OptionInitDB)

	require.NoError(t, err)
	require.NotNil(t, api)

	return api
}

func TestNewUpdater(t *testing.T) {
	conf := updater.Config{
		OmahaURL:        "http://localhost:8000",
		AppID:           "io.phony.App",
		Channel:         "stable",
		InstanceID:      "instance001",
		InstanceVersion: "0.1.0",
	}
	// Valid Config
	_, err := updater.New(conf)
	assert.NoError(t, err)

	// Invalid Config
	conf.OmahaURL = "http://invalidurl.test\\"
	updater, err := updater.New(conf)
	assert.Nil(t, updater)
	assert.Error(t, err)
}

func TestCheckForUpdates(t *testing.T) {
	apiInstance := newForTest(t)

	t.Cleanup(func() {
		apiInstance.Close()
	})

	appID, track, tChannel := setup(&config{t: t, api: apiInstance, pkgVersion: "0.1.0", policySafeMode: true, policyMaxUpdatesPerPeriod: 2})

	u, err := updater.New(updater.Config{
		OmahaURL:        "http://localhost:8000",
		AppID:           appID,
		Channel:         track,
		InstanceID:      "instance001",
		InstanceVersion: "0.2.0",
		OmahaReqHandler: newTestHandler(apiInstance),
	})
	require.NoError(t, err)

	info, err := u.CheckForUpdates(context.TODO())
	require.NoError(t, err)
	assert.False(t, info.HasUpdate())
	assert.Equal(t, "", info.Version())

	newPkg, err := apiInstance.AddPackage(&api.Package{Type: api.PkgTypeOther, URL: "http://sample.url/pkg", Version: "0.3.0", ApplicationID: appID, Arch: api.ArchAMD64, Filename: null.StringFrom("updatefile.txt")})
	require.NoError(t, err)
	tChannel.PackageID = null.StringFrom(newPkg.ID)
	err = apiInstance.UpdateChannel(tChannel)
	require.NoError(t, err)

	info, err = u.CheckForUpdates(context.TODO())
	require.NoError(t, err)
	assert.True(t, info.HasUpdate())

	version := info.Version()
	assert.Equal(t, "0.3.0", version)

	urls := info.URLs()
	require.NotNil(t, urls)
	assert.Equal(t, 1, len(urls))
	assert.Equal(t, urls[len(urls)-1], info.URL())
	assert.Equal(t, "http://sample.url/pkg", info.URL())

	pkg := info.Package()
	require.NotNil(t, pkg)
	assert.Equal(t, "updatefile.txt", pkg.Name)
}

type updateTestHandler struct {
	fetchUpdateResult error
	applyUpdateResult error
}

func (u updateTestHandler) FetchUpdate(ctx context.Context, info updater.UpdateInfo) error {
	return u.fetchUpdateResult
}

func (u updateTestHandler) ApplyUpdate(ctx context.Context, info updater.UpdateInfo) error {
	return u.applyUpdateResult
}

type config struct {
	t                         *testing.T
	api                       *api.API
	pkgVersion                string
	policySafeMode            bool
	policyMaxUpdatesPerPeriod int
}

func setup(cnf *config) (string, string, *api.Channel) {
	cnf.t.Helper()
	tTeam, err := cnf.api.AddTeam(&api.Team{Name: "test_team"})
	require.NoError(cnf.t, err)
	tApp, err := cnf.api.AddApp(&api.Application{Name: "io.phony.App", TeamID: tTeam.ID})
	require.NoError(cnf.t, err)
	tPkg, err := cnf.api.AddPackage(&api.Package{Type: api.PkgTypeOther, URL: "http://sample.url/pkg", Version: cnf.pkgVersion, ApplicationID: tApp.ID, Arch: api.ArchAMD64})
	require.NoError(cnf.t, err)
	tChannel, err := cnf.api.AddChannel(&api.Channel{Name: "channel1", Color: "blue", ApplicationID: tApp.ID, PackageID: null.StringFrom(tPkg.ID), Arch: api.ArchAMD64})
	require.NoError(cnf.t, err)
	tGroup, err := cnf.api.AddGroup(&api.Group{Name: "group1", ApplicationID: tApp.ID, ChannelID: null.StringFrom(tChannel.ID), PolicyUpdatesEnabled: true, PolicySafeMode: cnf.policySafeMode, PolicyPeriodInterval: "15 minutes", PolicyMaxUpdatesPerPeriod: cnf.policyMaxUpdatesPerPeriod, PolicyUpdateTimeout: "60 minutes", Track: "stable"})
	require.NoError(cnf.t, err)
	return tApp.ID, tGroup.Track, tChannel
}

func TestTryUpdate(t *testing.T) {
	api := newForTest(t)

	t.Cleanup(func() {
		api.Close()
	})

	oldVersion := "0.2.0"
	pkgVersion := "0.4.0"
	appID, track, _ := setup(&config{t: t, api: api, pkgVersion: pkgVersion, policySafeMode: false, policyMaxUpdatesPerPeriod: 10})

	tests := []struct {
		name              string
		fetchUpdateResult error
		applyUpdateResult error
		isErr             bool
	}{
		{
			name:              "error_fetching_update",
			fetchUpdateResult: errors.New("something went wrong fetching the update"),
			applyUpdateResult: nil,
			isErr:             true,
		},
		{
			name:              "error_applying_update",
			fetchUpdateResult: nil,
			applyUpdateResult: errors.New("something went wrong fetching the update"),
			isErr:             true,
		},
		{
			name:              "success_try_update",
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
				OmahaReqHandler: newTestHandler(api),
				Debug:           false,
			})
			require.NoError(t, err)

			assert.Equal(t, oldVersion, u.InstanceVersion())

			err = u.TryUpdate(context.TODO(), &updateTestHandler{
				fetchUpdateResult: tc.fetchUpdateResult,
				applyUpdateResult: tc.applyUpdateResult,
			})
			if tc.isErr {
				assert.Error(t, err)
				assert.Equal(t, oldVersion, u.InstanceVersion())
			} else {
				assert.NoError(t, err)
				assert.Equal(t, pkgVersion, u.InstanceVersion())
			}
		})
	}
}

// ExampleUpdater shows how to use the updater package to
// update an application manually.
func ExampleUpdater() error {

	conf := updater.Config{
		OmahaURL:        "http://test.omahaserver.com/v1/update/",
		AppID:           "application_id",
		Channel:         "stable",
		InstanceID:      uuid.NewString(),
		InstanceVersion: "0.0.1",
	}

	appUpdater, err := updater.New(conf)
	if err != nil {
		return err
	}

	ctx := context.TODO()

	updateInfo, err := appUpdater.CheckForUpdates(ctx)
	if err != nil {
		return fmt.Errorf("error checking updates for app: %s, err: %w", conf.AppID, err)
	}

	if !updateInfo.HasUpdate() {
		return fmt.Errorf("No update exists for the application")
	}

	// So we got an update, let's report we'll start downloading it.
	if err := appUpdater.ReportProgress(ctx, updater.ProgressDownloadStarted); err != nil {
		if progressErr := appUpdater.ReportError(ctx, nil); progressErr != nil {
			fmt.Println("reporting progress error:", progressErr)
		}
		return err
	}

	// This should be implemented by the user.
	// download, err := someFunctionThatDownloadsAFile(ctx, info.GetURL())
	// if err != nil {
	// 	// Oops something went wrong
	// 	if progressErr := appUpdater.ReportError(ctx, nil); progressErr != nil {
	// 		fmt.Println("reporting error:", progressErr)
	// 	}
	// 	return err
	// }

	// The download was successful, let's inform that to the omaha server
	if err := appUpdater.ReportProgress(ctx, updater.ProgressDownloadFinished); err != nil {
		if progressErr := appUpdater.ReportError(ctx, nil); progressErr != nil {
			fmt.Println("reporting progress error:", progressErr)
		}
		return err
	}

	// We got our update file, let's install it!
	if err := appUpdater.ReportProgress(ctx, updater.ProgressInstallationStarted); err != nil {
		if progressErr := appUpdater.ReportError(ctx, nil); progressErr != nil {
			fmt.Println("reporting progress error:", progressErr)
		}
		return err
	}

	// This should be your own implementation
	// err := someFunctionThatExtractsTheUpdateAndInstallIt(ctx, download)
	// if err != nil {
	// 	// Oops something went wrong
	// 	if progressErr := appUpdater.ReportError(ctx, nil); progressErr != nil {
	// 		fmt.Println("reporting error:", progressErr)
	// 	}
	// 	return err
	// }

	if err := appUpdater.CompleteUpdate(ctx, updateInfo); err != nil {
		if progressErr := appUpdater.ReportError(ctx, nil); progressErr != nil {
			fmt.Println("reporting progress error:", progressErr)
		}
		return err
	}

	return nil
}

type exampleUpdateHandler struct {
}

func (e exampleUpdateHandler) FetchUpdate(ctx context.Context, info updater.UpdateInfo) error {
	// download, err := someFunctionThatDownloadsAFile(ctx, info.GetURL())
	// if err != nil {
	// 	return err
	// }
	return nil
}

func (e exampleUpdateHandler) ApplyUpdate(ctx context.Context, info updater.UpdateInfo) error {
	// err := someFunctionThatExtractsTheUpdateAndInstallIt(ctx, getDownloadFile(ctx))
	// if err != nil {
	// 	// Oops something went wrong
	// 	return err
	// }

	// err := someFunctionThatExitsAndRerunsTheApp(ctx)
	// if err != nil {
	// 	// Oops something went wrong
	// 	return err
	// }
	return nil
}

// ExampleUpdaterWithUpdateHandler shows how to use the updater package to
// update an application automatically using exampleUpdateHandler.
func ExampleUpdaterWithUpdateHandler() error {

	conf := updater.Config{
		OmahaURL:        "http://test.omahaserver.com/v1/update/",
		AppID:           "application_id",
		Channel:         "stable",
		InstanceID:      uuid.NewString(),
		InstanceVersion: "0.0.1",
	}

	appUpdater, err := updater.New(conf)
	if err != nil {
		return err
	}

	ctx := context.TODO()

	if err := appUpdater.TryUpdate(ctx, exampleUpdateHandler{}); err != nil {
		return err
	}

	return nil
}
