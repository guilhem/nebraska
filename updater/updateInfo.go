package updater

import (
	"github.com/kinvolk/go-omaha/omaha"
)

// UpdateInfo is a wrapper around CheckForUpdates response
// and it provides helper functions.
type UpdateInfo struct {
	HasUpdate bool
	app       *omaha.AppResponse
	omahaResp *omaha.Response
}

// NewUpdateInfo creates and returns *UpdateInfo from omaha.Response and appID.
func NewUpdateInfo(resp *omaha.Response, appID string) *UpdateInfo {
	if resp == nil {
		return nil
	}
	app := resp.GetApp(appID)
	if app == nil {
		return nil
	}
	return &UpdateInfo{
		HasUpdate: app != nil && app.Status == omaha.AppOK && app.UpdateCheck.Status == "ok",
		app:       app,
		omahaResp: resp,
	}
}

// GetVersion retuns the manifest version of the UpdateInfo.
func (u *UpdateInfo) GetVersion() string {
	if u.app == nil || u.app.UpdateCheck == nil || u.app.UpdateCheck.Manifest == nil {
		return ""
	}
	return u.app.UpdateCheck.Manifest.Version
}

// GetURLs returns an array of update check urls from the omaha response.
func (u *UpdateInfo) GetURLs() []string {
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

// GetURL returns the first update url from the omaha response.
func (u *UpdateInfo) GetURL() string {
	urls := u.GetURLs()
	if urls == nil || len(urls) == 0 {
		return ""
	}
	return urls[0]
}

// GetPackages returns an array of packages from the omaha response.
func (u *UpdateInfo) GetPackages() []*omaha.Package {
	if u.app == nil || u.app.UpdateCheck == nil || u.app.UpdateCheck.Manifest == nil {
		return nil
	}
	return u.app.UpdateCheck.Manifest.Packages
}

// GetPackage returns the first package from the omaha response.
func (u *UpdateInfo) GetPackage() *omaha.Package {
	pkgs := u.GetPackages()
	if pkgs == nil || len(pkgs) == 0 {
		return nil
	}
	return pkgs[0]
}

// GetUpdateStatus returns the update status from the omaha response.
func (u *UpdateInfo) GetUpdateStatus() string {
	if u.app == nil || u.app.UpdateCheck == nil {
		return ""
	}
	return string(u.app.UpdateCheck.Status)
}

// GetOmahaReponse returns the raw omaha response.
func (u *UpdateInfo) GetOmahaResponse() *omaha.Response {
	return u.omahaResp
}
