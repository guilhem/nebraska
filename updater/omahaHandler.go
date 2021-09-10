package updater

import (
	"bytes"
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"

	"github.com/kinvolk/go-omaha/omaha"
)

// HttpDo interface wraps the Do function which takes
// http.Request and returns http.Response
// and error. HttpDo interface allows the user to
// create their custom implementation to handle proxies or
// retries etc.
type HttpDo interface {
	Do(*http.Request) (*http.Response, error)
}

// httpOmahaHandler implements the OmahaRequestHandler using the
// HttpDo interface to handle the network calls.
type httpOmahaReqHandler struct {
	httpClient HttpDo
}

// NewOmahaRequestHandler returns a OmahaRequestHandler which uses the HttpDo client
// to handle the post request to the omaha server.
func NewOmahaRequestHandler(client HttpDo) OmahaRequestHandler {
	omahaRequestHandler := httpOmahaReqHandler{
		httpClient: client,
	}
	if omahaRequestHandler.httpClient == nil {
		omahaRequestHandler.httpClient = http.DefaultClient
	}
	return &omahaRequestHandler
}

// Handle uses the httpClient to send the omaha request to the url and
// returns omaha response and error.
func (h *httpOmahaReqHandler) Handle(ctx context.Context, url string, req *omaha.Request) (*omaha.Response, error) {
	requestBuf := bytes.NewBuffer(nil)
	encoder := xml.NewEncoder(requestBuf)
	err := encoder.Encode(req)
	if err != nil {
		return nil, fmt.Errorf("encoding request as XML: %w", err)
	}

	request, err := http.NewRequest("POST", url, requestBuf)
	if err != nil {
		return nil, fmt.Errorf("http new request: %w", err)
	}
	request.WithContext(ctx)
	request.Header.Set("Content-Type", "text/xml")

	resp, err := h.httpClient.Do(request)
	if err != nil {
		return nil, fmt.Errorf("http post request: %w", err)
	}
	defer resp.Body.Close()

	// A response over 1M in size is certainly bogus.
	respBody := &io.LimitedReader{R: resp.Body, N: 1024 * 1024}
	contentType := resp.Header.Get("Content-Type")
	omahaResp, err := omaha.ParseResponse(contentType, respBody)
	if err != nil {
		return nil, fmt.Errorf("parse response to omaha response: %w", err)
	}

	return omahaResp, nil
}
