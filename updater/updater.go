package updater

import (
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"net/url"
	"sync"

	"github.com/google/uuid"
	"github.com/kinvolk/go-omaha/omaha"
)

const defaultClientVersion = "go-omaha"

type Progress int

const (
	ProgressDownloadStarted Progress = iota
	ProgressDownloadFinished
	ProgressInstallationStarted
	ProgressInstallationFinished
	ProgressUpdateComplete
	ProgressUpdateCompleteAndRestarted
	ProgressError
)

var progressEventMap = map[Progress]*omaha.EventRequest{
	ProgressDownloadStarted: {
		Type:   omaha.EventTypeUpdateDownloadStarted,
		Result: omaha.EventResultSuccess,
	},
	ProgressDownloadFinished: {
		Type:   omaha.EventTypeUpdateDownloadFinished,
		Result: omaha.EventResultSuccess,
	},
	ProgressUpdateComplete: {
		Type:   omaha.EventTypeUpdateComplete,
		Result: omaha.EventResultSuccess,
	},
	ProgressUpdateCompleteAndRestarted: {
		Type:   omaha.EventTypeUpdateComplete,
		Result: omaha.EventResultSuccessReboot,
	},
	ProgressInstallationStarted: {
		Type:   omaha.EventTypeInstallStarted,
		Result: omaha.EventResultSuccess,
	},
	ProgressInstallationFinished: {
		Type:   omaha.EventTypeInstallStarted,
		Result: omaha.EventResultSuccess,
	},
	ProgressError: {
		Type:   omaha.EventTypeUpdateComplete,
		Result: omaha.EventResultError,
	},
}

type OmahaRequestHandler interface {
	Handle(req *omaha.Request) (*omaha.Response, error)
}

type Updater struct {
	omahaURL      string
	clientVersion string

	instanceID      string
	instanceVersion string
	sessionID       string

	appID   string
	channel string

	debug           bool
	omahaReqHandler OmahaRequestHandler

	mu sync.RWMutex
}

type Config struct {
	OmahaURL        string
	AppID           string
	Channel         string
	InstanceID      string
	InstanceVersion string
	Debug           bool
	OmahaReqHandler OmahaRequestHandler
}

func New(config Config) (*Updater, error) {
	if _, err := url.Parse(config.OmahaURL); err != nil {
		return nil, fmt.Errorf("parsing URL %q: %w", config.OmahaURL, err)
	}
	updater := Updater{
		omahaURL:        config.OmahaURL,
		clientVersion:   defaultClientVersion,
		instanceID:      config.InstanceID,
		sessionID:       uuid.New().String(),
		appID:           config.AppID,
		instanceVersion: config.InstanceVersion,
		channel:         config.Channel,
		debug:           config.Debug,
	}
	if config.OmahaReqHandler == nil {
		updater.omahaReqHandler = NewDefaultOmahaRequestHandler(config.OmahaURL)
	} else {
		updater.omahaReqHandler = config.OmahaReqHandler
	}
	return &updater, nil
}

func NewAppRequest(u *Updater) *omaha.Request {
	req := omaha.NewRequest()
	req.Version = u.clientVersion
	req.UserID = u.instanceID
	req.SessionID = u.sessionID

	app := req.AddApp(u.appID, u.GetInstanceVersion())
	app.MachineID = u.instanceID
	app.BootID = u.sessionID
	app.Track = u.channel

	return req
}

func (u *Updater) SendOmahaRequest(req *omaha.Request) (*omaha.Response, error) {
	if u.debug {
		requestByte, err := xml.Marshal(req)
		if err == nil {
			fmt.Println("Raw Request:\n", string(requestByte))
		}
	}
	resp, err := u.omahaReqHandler.Handle(req)
	if u.debug {
		responseByte, err := xml.Marshal(resp)
		if err == nil {
			fmt.Println("Raw Response:\n", string(responseByte))
		}
	}
	return resp, err
}

func (u *Updater) CheckForUpdates(ctx context.Context) (*UpdateInfo, error) {
	req := NewAppRequest(u)
	app := req.GetApp(u.appID)
	app.AddUpdateCheck()

	resp, err := u.SendOmahaRequest(req)
	if err != nil {
		return nil, err
	}
	info := NewUpdateInfo(resp, u.appID)

	return info, nil
}

func (u *Updater) ReportProgress(ctx context.Context, progress Progress) error {
	val, ok := progressEventMap[progress]
	if !ok {
		return errors.New("invalid Progress value")
	}
	resp, err := u.SendOmahaEvent(ctx, val)
	if err != nil {
		return err
	}

	app := resp.GetApp(u.appID)
	if app.Status != "ok" {
		return fmt.Errorf("reporting progress to omaha server, got response %q", app.Status)
	}

	return nil
}

func (u *Updater) SendOmahaEvent(ctx context.Context, event *omaha.EventRequest) (*omaha.Response, error) {
	req := NewAppRequest(u)
	app := req.GetApp(u.appID)
	app.Events = append(app.Events, event)

	return u.SendOmahaRequest(req)
}

func (u *Updater) GetInstanceVersion() string {
	u.mu.RLock()
	defer u.mu.RUnlock()
	return u.instanceVersion
}

func (u *Updater) SetInstanceVersion(version string) {
	u.mu.Lock()
	defer u.mu.Unlock()
	u.instanceVersion = version
}

func (u *Updater) TryUpdate(ctx context.Context, handler UpdateHandler) error {

	info, err := u.CheckForUpdates(ctx)
	if err != nil {
		return err
	}

	if !info.HasUpdate {
		return fmt.Errorf("no update available for app %v, channel %v: %v", u.appID, u.channel, info.GetUpdateStatus())
	}

	if err := handler.FetchUpdate(ctx, info); err != nil {
		if progressErr := u.ReportProgress(ctx, ProgressError); progressErr != nil {
			fmt.Println("error reporting ProgressError to omaha server:", progressErr)
		}
		return err
	}

	err = u.ReportProgress(ctx, ProgressDownloadFinished)
	if err != nil {
		return err
	}

	if err := handler.ApplyUpdate(ctx, info); err != nil {
		if progressErr := u.ReportProgress(ctx, ProgressError); progressErr != nil {
			fmt.Println("error reporting ProgressError to omaha server:", progressErr)
		}
		return err
	}

	if err := u.ReportProgress(ctx, ProgressInstallationFinished); err != nil {
		return err
	}

	version := info.GetVersion()
	u.SetInstanceVersion(version)

	if err := u.ReportProgress(ctx, ProgressUpdateComplete); err != nil {
		return err
	}

	return nil
}
