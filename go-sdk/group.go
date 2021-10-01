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

type Group struct {
	ApplicationID             string    `json:"application_id"`
	ChannelID                 string    `json:"channel_id"`
	CreatedTs                 time.Time `json:"created_ts"`
	Description               string    `json:"description"`
	Id                        string    `json:"id"`
	Name                      string    `json:"name"`
	PolicyMaxUpdatesPerPeriod int       `json:"policy_max_updates_per_period"`
	PolicyOfficeHours         bool      `json:"policy_office_hours"`
	PolicyPeriodInterval      string    `json:"policy_period_interval"`
	PolicySafeMode            bool      `json:"policy_safe_mode"`
	PolicyTimezone            string    `json:"policy_timezone"`
	PolicyUpdateTimeout       string    `json:"policy_update_timeout"`
	PolicyUpdatesEnabled      bool      `json:"policy_updates_enabled"`
	RolloutInProgress         bool      `json:"rollout_in_progress"`
	Track                     string    `json:"track"`
}

type GroupConfig codegen.CreateGroupJSONBody

type GroupResponse interface {
	Update(ctx context.Context, config GroupConfig, reqEditors ...RequestEditorFn) (GroupResponse, error)
	Delete(ctx context.Context, appID string, groupID string, reqEditors ...RequestEditorFn) error
	Props() Group
}

type group struct {
	Group
	n *Nebraska
}

func (group *group) Props() Group {
	return group.Group
}

func (group *group) Update(ctx context.Context, config GroupConfig, reqEditors ...RequestEditorFn) (GroupResponse, error) {
	return group.n.UpdateGroup(ctx, group.ApplicationID, group.Id, config, reqEditors...)
}

func (group *group) Delete(ctx context.Context, appID string, groupID string, reqEditors ...RequestEditorFn) error {
	return group.n.DeleteGroup(ctx, group.ApplicationID, group.Id, reqEditors...)
}

type groupWithResponse struct {
	group
	rawResponse
}

type GroupsResponse struct {
	TotalCount int
	Groups     []GroupResponse
	n          *Nebraska
	rawResponse
}

func (n *Nebraska) PaginateGroups(ctx context.Context, appID string, page int, perPage int, reqEditors ...RequestEditorFn) (*GroupsResponse, error) {
	params := codegen.PaginateGroupsParams{
		Page:    &page,
		Perpage: &perPage,
	}
	resp, err := n.client.PaginateGroups(ctx, appID, &params, convertReqEditors(reqEditors...)...)
	if err != nil {
		return nil, fmt.Errorf("paginate groups of appID %q : %w", appID, err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("PaginateGroups of appID: %q returned invalid response code: %d", appID, resp.StatusCode)
	}

	return n.parseGroups(resp)
}

func (n *Nebraska) CreateGroup(ctx context.Context, appID string, config GroupConfig, reqEditors ...RequestEditorFn) (GroupResponse, error) {
	resp, err := n.client.CreateGroup(ctx, appID, codegen.CreateGroupJSONRequestBody(config), convertReqEditors(reqEditors...)...)
	if err != nil {
		return nil, fmt.Errorf("creating group in app %q: %w", appID, err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("CreateGroup returned invalid response code: %d", resp.StatusCode)
	}
	return n.parseGroup(resp)
}

func (n *Nebraska) UpdateGroup(ctx context.Context, appID string, groupID string, config GroupConfig, reqEditors ...RequestEditorFn) (GroupResponse, error) {
	resp, err := n.client.UpdateGroup(ctx, appID, groupID, codegen.UpdateGroupJSONRequestBody(config), convertReqEditors(reqEditors...)...)
	if err != nil {
		return nil, fmt.Errorf("updating group in app %q: %w", appID, err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("UpdateGroup returned invalid response code: %d", resp.StatusCode)
	}
	return n.parseGroup(resp)
}

func (n *Nebraska) DeleteGroup(ctx context.Context, appID string, groupID string, reqEditors ...RequestEditorFn) error {
	resp, err := n.client.DeleteGroupWithResponse(ctx, appID, groupID, convertReqEditors(reqEditors...)...)
	if err != nil {
		return fmt.Errorf("deleting group %q: %w", appID, err)
	}
	if resp.StatusCode() != http.StatusNoContent {
		return fmt.Errorf("DeleteGroup %q returned invalid response code: %d", appID, resp.StatusCode())
	}
	return nil
}

func (n *Nebraska) GetGroup(ctx context.Context, appID string, groupID string, reqEditors ...RequestEditorFn) (GroupResponse, error) {
	resp, err := n.client.GetGroup(ctx, appID, groupID, convertReqEditors(reqEditors...)...)
	if err != nil {
		return nil, fmt.Errorf("fetching group %q: %w", groupID, err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GetGroup %q returned invalid response code: %d", groupID, resp.StatusCode)
	}
	return n.parseGroup(resp)
}

func (n *Nebraska) parseGroup(resp *http.Response) (GroupResponse, error) {
	var group groupWithResponse

	if !strings.Contains(resp.Header.Get("Content-Type"), "json") {
		return nil, fmt.Errorf("invalid group response content-type: %q", resp.Header.Get("Content-Type"))
	}

	decoder := json.NewDecoder(resp.Body)
	err := decoder.Decode(&group)
	if err != nil {
		return nil, fmt.Errorf("group decode: %w", err)
	}

	group.n = n
	group.rawResponse.resp = resp
	return &group, nil
}

func (n *Nebraska) parseGroups(resp *http.Response) (*GroupsResponse, error) {
	type groupsPage struct {
		Groups     []json.RawMessage `json:"groups"`
		Count      int               `json:"count"`
		TotalCount int               `json:"totalCount"`
	}

	var rawGroups groupsPage

	if !strings.Contains(resp.Header.Get("Content-Type"), "json") {
		return nil, fmt.Errorf("invalid groups response content-type: %q", resp.Header.Get("Content-Type"))
	}

	decoder := json.NewDecoder(resp.Body)
	err := decoder.Decode(&rawGroups)
	if err != nil {
		return nil, fmt.Errorf("groupPage response decode: %w", err)
	}
	var groups []GroupResponse

	for _, rawGroup := range rawGroups.Groups {
		var group group
		decoder := json.NewDecoder(bytes.NewReader(rawGroup))
		err := decoder.Decode(&group)
		if err != nil {
			return nil, fmt.Errorf("application decoding: %w", err)
		}
		group.n = n
		groups = append(groups, &group)
	}
	return &GroupsResponse{Groups: groups, TotalCount: rawGroups.TotalCount}, nil
}
