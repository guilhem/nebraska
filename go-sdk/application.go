package nebraska

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/kinvolk/nebraska/backend/pkg/codegen"
)

type AppConfig codegen.CreateAppJSONBody
type ApplicationResponse interface {
	Update(ctx context.Context, conf AppConfig, reqEditors ...RequestEditorFn) (ApplicationResponse, error)
	Delete(ctx context.Context, reqEditors ...RequestEditorFn) error
	Props() Application
	Groups(ctx context.Context, reqEditors ...RequestEditorFn) ([]GroupResponse, error)
	CreateGroup(ctx context.Context, conf GroupConfig, reqEditors ...RequestEditorFn) (GroupResponse, error)
}

type Application struct {
	CreatedTs   time.Time `json:"created_ts"`
	Description string    `json:"description"`
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	ProductId   string    `json:"product_id"`
}

type application struct {
	Application
	n *Nebraska
}

type applicationWithResponse struct {
	application
	rawResponse
}

func (app *application) Update(ctx context.Context, conf AppConfig, reqEditors ...RequestEditorFn) (ApplicationResponse, error) {
	return app.n.UpdateApp(ctx, app.ID, conf, reqEditors...)
}

func (app *application) Delete(ctx context.Context, reqEditors ...RequestEditorFn) error {
	return app.n.DeleteApp(ctx, app.ID)
}

func (app *application) Props() Application {
	return app.Application
}

func (app *application) Groups(ctx context.Context, reqEditors ...RequestEditorFn) ([]GroupResponse, error) {
	var groups []GroupResponse
	count := 1
	for {
		groupsResp, err := app.n.PaginateGroups(ctx, app.ID, count, 10, reqEditors...)
		if err != nil {
			return nil, fmt.Errorf("fetching groups page %d: %w", count, err)
		}
		groups = append(groups, groupsResp.Groups...)
		if groupsResp.TotalCount == len(groups) {
			break
		}
		count += 1
	}
	return groups, nil
}

func (app *application) CreateGroup(ctx context.Context, conf GroupConfig, reqEditors ...RequestEditorFn) (GroupResponse, error) {
	return app.n.CreateGroup(ctx, app.ID, conf, reqEditors...)
}

type ApplicationsResponse struct {
	TotalCount int
	Apps       []ApplicationResponse
	rawResponse
}

type UpdateApplication struct {
	Name string
}

func (n *Nebraska) GetApp(ctx context.Context, appID string, reqEditors ...RequestEditorFn) (ApplicationResponse, error) {
	resp, err := n.client.GetApp(ctx, appID, convertReqEditors(reqEditors...)...)
	if err != nil {
		return nil, fmt.Errorf("fetching app %q: %w", appID, err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GetApp %q returned invalid response code: %d", appID, resp.StatusCode)
	}
	return n.parseApplication(resp)
}

func (n *Nebraska) CreateApp(ctx context.Context, info AppConfig, cloneFrom *string, reqEditors ...RequestEditorFn) (ApplicationResponse, error) {

	var params codegen.CreateAppParams
	if cloneFrom != nil {
		params.CloneFrom = cloneFrom
	}
	resp, err := n.client.CreateApp(ctx, &params, codegen.CreateAppJSONRequestBody(info), convertReqEditors(reqEditors...)...)
	if err != nil {
		return nil, fmt.Errorf("creating app: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("CreateApp returned invalid response code: %d", resp.StatusCode)
	}

	return n.parseApplication(resp)
}

func (n *Nebraska) UpdateApp(ctx context.Context, appID string, info AppConfig, reqEditors ...RequestEditorFn) (ApplicationResponse, error) {

	resp, err := n.client.UpdateApp(ctx, appID, codegen.UpdateAppJSONRequestBody(info), convertReqEditors(reqEditors...)...)
	if err != nil {
		return nil, fmt.Errorf("updating app id %q: %w", appID, err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Update App %q returned invalid response code: %d", appID, resp.StatusCode)
	}

	return n.parseApplication(resp)
}

func (n *Nebraska) DeleteApp(ctx context.Context, appID string, reqEditors ...RequestEditorFn) error {
	resp, err := n.client.DeleteAppWithResponse(ctx, appID, convertReqEditors(reqEditors...)...)
	if err != nil {
		return fmt.Errorf("deleting app %q: %w", appID, err)
	}
	if resp.StatusCode() != http.StatusNoContent {
		return fmt.Errorf("DeleteApp %q returned invalid response code: %d", appID, resp.StatusCode())
	}
	return nil
}

func (n *Nebraska) PaginateApps(ctx context.Context, page int, perPage int, reqEditors ...RequestEditorFn) (*ApplicationsResponse, error) {
	params := codegen.PaginateAppsParams{
		Page:    &page,
		Perpage: &perPage,
	}
	resp, err := n.client.PaginateApps(ctx, &params, convertReqEditors(reqEditors...)...)
	if err != nil {
		return nil, fmt.Errorf("paginate app: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("PaginateApps returned invalid response code: %d", resp.StatusCode)
	}
	return n.parseApplications(resp)
}

func (n *Nebraska) parseApplication(resp *http.Response) (ApplicationResponse, error) {
	var application applicationWithResponse

	if !strings.Contains(resp.Header.Get("Content-Type"), "json") {
		return nil, fmt.Errorf("invalid application response content-type: %q", resp.Header.Get("Content-Type"))
	}

	decoder := json.NewDecoder(resp.Body)
	err := decoder.Decode(&application)
	if err != nil {
		return nil, fmt.Errorf("application decode: %w", err)
	}
	application.n = n
	application.rawResponse.resp = resp

	return &application, nil
}

func (n *Nebraska) parseApplications(resp *http.Response) (*ApplicationsResponse, error) {

	type appsPage struct {
		Applications []json.RawMessage `json:"applications"`
		Count        int               `json:"count"`
		TotalCount   int               `json:"totalCount"`
	}

	var rawApps appsPage

	if !strings.Contains(resp.Header.Get("Content-Type"), "json") {
		return nil, fmt.Errorf("invalid applications response content-type: %q", resp.Header.Get("Content-Type"))
	}

	decoder := json.NewDecoder(resp.Body)
	err := decoder.Decode(&rawApps)
	if err != nil {
		return nil, fmt.Errorf("appPage response decode: %w", err)
	}
	var apps []ApplicationResponse

	for _, rawApp := range rawApps.Applications {
		var app application
		decoder := json.NewDecoder(bytes.NewReader(rawApp))
		err := decoder.Decode(&app)
		if err != nil {
			return nil, fmt.Errorf("application decoding: %w", err)
		}
		app.n = n
		// TODO: Figure out raw response for apps
		apps = append(apps, &app)
	}
	return &ApplicationsResponse{Apps: apps, TotalCount: rawApps.TotalCount}, nil
}
