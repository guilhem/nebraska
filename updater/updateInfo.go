package updater

import (
	"github.com/kinvolk/go-omaha/omaha"
)

// UpdateInfo interface wraps the helper functions
// to fetch specific values from the omaha response
// that was recieved for check if any new update
// exists request.
type UpdateInfo interface {
	HasUpdate() bool
	Version() string
	URLs() []string
	URL() string
	Packages() []*omaha.Package
	Package() *omaha.Package
	UpdateStatus() string
	OmahaResponse() omaha.Response
}

// updateInfo implements the UpdateInfo interface.
type updateInfo struct {
	app       *omaha.AppResponse
	omahaResp omaha.Response
}

// NewUpdateInfo returns UpdateInfo from omaha.Response and appID.
func NewUpdateInfo(resp omaha.Response, appID string) UpdateInfo {
	app := resp.GetApp(appID)
	if app == nil {
		return nil
	}
	return &updateInfo{
		app:       app,
		omahaResp: resp,
	}
}

// HasUpdate returns true if an update exists.
func (u *updateInfo) HasUpdate() bool {
	return u.app != nil && u.app.Status == omaha.AppOK && u.app.UpdateCheck.Status == "ok"
}

// GetVersion returns the manifest version of the UpdateInfo,
// returns "" if the version is not present in the omaha response.
func (u *updateInfo) Version() string {
	if u.app == nil || u.app.UpdateCheck == nil || u.app.UpdateCheck.Manifest == nil {
		return ""
	}
	return u.app.UpdateCheck.Manifest.Version
}

// GetURLs returns an array of URLs present in the omaha response,
// returns nil if the URLs are not present in the omaha response.
func (u *updateInfo) URLs() []string {
	if u.app == nil || u.app.UpdateCheck == nil {
		return nil
	}
	omahaURLs := u.app.UpdateCheck.URLs
	urls := make([]string, len(omahaURLs))
	for i, url := range omahaURLs {
		urls[i] = url.CodeBase
	}
	return urls
}

// GetURL returns the first update URL in the omaha response,
// returns "" if the URL is not present in the omaha response.
func (u *updateInfo) URL() string {
	urls := u.URLs()
	if urls == nil || len(urls) == 0 {
		return ""
	}
	return urls[0]
}

// GetPackages returns an array of packages present in the omaha response,
// returns nil if the Packages are not present in the omaha response.
func (u *updateInfo) Packages() []*omaha.Package {
	if u.app == nil || u.app.UpdateCheck == nil || u.app.UpdateCheck.Manifest == nil {
		return nil
	}
	return u.app.UpdateCheck.Manifest.Packages
}

// GetPackage returns the first package from the omaha response,
// returns nil if the package is not present in the omaha response.
func (u *updateInfo) Package() *omaha.Package {
	pkgs := u.Packages()
	if pkgs == nil || len(pkgs) == 0 {
		return nil
	}
	return pkgs[0]
}

// GetUpdateStatus returns the update status from the omaha response,
// returns "" if the status is not present in the omaha response.
func (u *updateInfo) UpdateStatus() string {
	if u.app == nil || u.app.UpdateCheck == nil {
		return ""
	}
	return string(u.app.UpdateCheck.Status)
}

// GetOmahaReponse returns the raw omaha response.
func (u *updateInfo) OmahaResponse() omaha.Response {
	return u.omahaResp
}
