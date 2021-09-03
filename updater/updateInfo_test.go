package updater_test

import (
	"encoding/xml"
	"testing"

	"github.com/kinvolk/go-omaha/omaha"
	"github.com/kinvolk/nebraska/updater"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	updateExistsResponse = `<?xml version="1.0" encoding="UTF-8"?>
	<response protocol="3.0" server="nebraska">
	   <daystart elapsed_seconds="0" />
	   <app appid="e96281a6-d1af-4bde-9a0a-97b76e56dc57" status="ok">
		  <updatecheck status="ok">
			 <urls>
				<url codebase="https://kinvolk.io/test/response" />
			 </urls>
			 <manifest version="2191.5.0">
				<packages>
				   <package name="flatcar_production_update.gz" hash="test+x2zIoeClk=" size="465881871" required="true" />
				</packages>
				<actions>
				   <action event="postinstall" sha256="test/FodbjVgqkyF/y8=" DisablePayloadBackoff="true" />
				</actions>
			 </manifest>
		  </updatecheck>
	   </app>
	</response>
	`

	noUpdateResponse = `<?xml version="1.0" encoding="UTF-8"?>
	<response protocol="3.0" server="nebraska">
	   <daystart elapsed_seconds="0" />
	   <app appid="e96281a6-d1af-4bde-9a0a-97b76e56dc57" status="ok">
		  <updatecheck status="noupdate">
			 <urls />
		  </updatecheck>
	   </app>
	</response>`

	errorResponse = `<?xml version="1.0" encoding="UTF-8"?>
	<response protocol="3.0" server="nebraska">
	   <daystart elapsed_seconds="0" />
	   <app appid="h96281a6-d1af-4bde-9a0a-97b76e56dc57" status="error-failedToRetrieveUpdatePackageInfo">
	      <updatecheck status="error-internal">
	         <urls />
	      </updatecheck>
	   </app>
	</response>`

	nonUpdateCheckResponse = `<?xml version="1.0" encoding="UTF-8"?>
	<response protocol="3.0" server="nebraska">
	   <daystart elapsed_seconds="0" />
	   <app appid="e96281a6-d1af-4bde-9a0a-97b76e56dc57" status="error-internal">
	   </app>
	</response>`
	appID      = "e96281a6-d1af-4bde-9a0a-97b76e56dc57"
	errorAppID = "h96281a6-d1af-4bde-9a0a-97b76e56dc57"
)

func TestUpdateInfo(t *testing.T) {

	type test struct {
		name          string
		response      *omaha.Response
		appID         string
		isNil         bool
		hasUpate      bool
		updateStatus  string
		packagesCount int
		urlCount      int
		version       string
	}

	// update exists response
	var updateExistsOmahaResponse omaha.Response
	err := xml.Unmarshal([]byte(updateExistsResponse), &updateExistsOmahaResponse)
	require.NoError(t, err)

	// no update response
	var noUpdateOmahaResponse omaha.Response
	err = xml.Unmarshal([]byte(noUpdateResponse), &noUpdateOmahaResponse)
	require.NoError(t, err)

	// error response
	var errorOmahaResponse omaha.Response
	err = xml.Unmarshal([]byte(errorResponse), &errorOmahaResponse)
	require.NoError(t, err)

	// non update check response
	var nonUpdateCheckOmahaResponse omaha.Response
	err = xml.Unmarshal([]byte(nonUpdateCheckResponse), &nonUpdateCheckOmahaResponse)
	require.NoError(t, err)

	tests := []test{
		{
			name:          "update exists",
			response:      &updateExistsOmahaResponse,
			appID:         appID,
			isNil:         false,
			hasUpate:      true,
			updateStatus:  "ok",
			packagesCount: 1,
			urlCount:      1,
			version:       "2191.5.0",
		},
		{
			name:     "invalid app id",
			response: &updateExistsOmahaResponse,
			appID:    errorAppID,
			isNil:    true,
		},
		{
			name:          "no update exists",
			response:      &noUpdateOmahaResponse,
			appID:         appID,
			isNil:         false,
			hasUpate:      false,
			updateStatus:  "noupdate",
			packagesCount: 0,
			urlCount:      0,
		},
		{
			name:          "error response",
			response:      &errorOmahaResponse,
			appID:         errorAppID,
			isNil:         false,
			hasUpate:      false,
			updateStatus:  "error-internal",
			packagesCount: 0,
			urlCount:      0,
		},
		{
			name:     "nil response",
			response: nil,
			appID:    appID,
			isNil:    true,
		},
		{
			name:          "non update check response",
			response:      &nonUpdateCheckOmahaResponse,
			appID:         appID,
			isNil:         false,
			hasUpate:      false,
			updateStatus:  "",
			packagesCount: 0,
			urlCount:      0,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			updateInfo := updater.NewUpdateInfo(tc.response, tc.appID)
			if tc.isNil {
				assert.Nil(t, updateInfo)
			} else {
				assert.NotNil(t, updateInfo)
				assert.Equal(t, tc.hasUpate, updateInfo.HasUpdate)
				assert.Equal(t, tc.updateStatus, updateInfo.GetUpdateStatus())
				assert.Equal(t, tc.version, updateInfo.GetVersion())
				assert.Equal(t, tc.urlCount, len(updateInfo.GetURLs()))
				if tc.urlCount > 0 {
					assert.NotEqual(t, "", updateInfo.GetURL())
				} else {
					assert.Equal(t, "", updateInfo.GetURL())
				}
				assert.Equal(t, tc.packagesCount, len(updateInfo.GetPackages()))
				if tc.packagesCount > 0 {
					assert.NotNil(t, updateInfo.GetPackage())
				} else {
					assert.Nil(t, updateInfo.GetPackage())
				}
				assert.Equal(t, tc.response, updateInfo.GetOmahaResponse())
			}
		})
	}

}
