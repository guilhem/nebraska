package nebraska

import (
	"context"
	"fmt"
	"net/http"
	"net/url"

	"github.com/kinvolk/nebraska/backend/pkg/codegen"
)

type Nebraska struct {
	config *Config
	client *codegen.ClientWithResponses
}

type Config struct {
	ServerURL string
	UserName  *string
	Password  *string
	Debug     bool
}

type rawResponse struct {
	resp *http.Response
}

func (r *rawResponse) Response() *http.Response {
	return r.resp
}

type RequestEditorFn codegen.RequestEditorFn

func New(conf Config) (*Nebraska, error) {
	// Validate Server URL
	_, err := url.Parse(conf.ServerURL)
	if err != nil {
		return nil, fmt.Errorf("invalid server url: %w", err)
	}

	// If the authMode is not noop UserName and Password is necessary.
	client, err := codegen.NewClientWithResponses(conf.ServerURL)
	if err != nil {
		return nil, fmt.Errorf("couldn't init client: %w", err)
	}

	serverConfig, err := client.GetConfigWithResponse(context.TODO())
	if err != nil || serverConfig.JSON200 == nil {
		return nil, fmt.Errorf("couldn't fetch nebraska config: %w", err)
	}

	if serverConfig.JSON200.AuthMode != "noop" && (conf.UserName == nil || conf.Password == nil) {
		return nil, fmt.Errorf("username and password required for auth mode: %s", serverConfig.JSON200.AuthMode)
	}

	return &Nebraska{
		config: &conf,
		client: client,
	}, nil
}
