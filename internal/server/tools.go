package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/bab3l/go-xurrent/pkg/collisionhints"

	"github.com/xurrent/xurrent-mcp/internal/livefind"
	"github.com/xurrent/xurrent-mcp/internal/tools"
)

// ── Argument types ──

type discoverArgs struct {
	Entity string `json:"entity,omitempty" jsonschema:"Optional entity type for details. Omit for full catalog."`
}

type queryArgs struct {
	Entity      string `json:"entity" jsonschema:"Entity type key (team, site, service, person, ci, sla, request, workflow…)."`
	Fields      string `json:"fields,omitempty" jsonschema:"Comma-separated fields (default entity defaults)."`
	Filter      string `json:"filter,omitempty" jsonschema:"key equals value server-side filter (e.g. name=MyTeam)."`
	State       string `json:"state,omitempty" jsonschema:"Predefined state: enabled, disabled, active, archived, trash."`
	Sort        string `json:"sort,omitempty" jsonschema:"Sort: name, -id, created_at, -updated_at."`
	PerPage     int32  `json:"per_page,omitempty" jsonschema:"1-100, default 100."`
	SearchAfter string `json:"search_after,omitempty" jsonschema:"Cursor from previous next_cursor for pagination."`
}

type readArgs struct {
	Entity string `json:"entity" jsonschema:"Entity type."`
	ID     int32  `json:"id" jsonschema:"Record ID."`
	Fields string `json:"fields,omitempty" jsonschema:"Comma-separated fields (uses defaults if empty)."`
}

type createArgs struct {
	Entity   string `json:"entity" jsonschema:"Entity type to create."`
	Body     any    `json:"body" jsonschema:"JSON object with field values."`
	ParentID int32  `json:"parent_id,omitempty" jsonschema:"For sub-resource entities like request_note, the parent record ID."`
	Force    bool   `json:"force,omitempty" jsonschema:"Skip draft approval? Needs XURRENT_ALLOW_MUTATIONS=1."`
}

type updateArgs struct {
	Entity string `json:"entity" jsonschema:"Entity type to update."`
	ID     int32  `json:"id" jsonschema:"Record ID."`
	Body   any    `json:"body" jsonschema:"JSON object with only changed fields."`
	Force  bool   `json:"force,omitempty" jsonschema:"Skip draft? Needs XURRENT_ALLOW_MUTATIONS=1."`
}

type deleteArgs struct {
	Entity string `json:"entity" jsonschema:"Entity type."`
	ID     int32  `json:"id" jsonschema:"Record ID to delete."`
	Force  bool   `json:"force,omitempty" jsonschema:"Skip draft? Needs XURRENT_ALLOW_MUTATIONS=1."`
}

type draftArgs struct {
	Action       string `json:"action" jsonschema:"list | view | commit | discard."`
	DraftID      string `json:"draft_id,omitempty" jsonschema:"Required for view/commit/discard."`
	Confirmation bool   `json:"confirmation,omitempty" jsonschema:"Set true for commit."`
}

type searchArgs struct {
	Query   string `json:"q" jsonschema:"Search query."`
	PerPage int32  `json:"per_page,omitempty" jsonschema:"1-25, default 25."`
}

type collisionArgs struct {
	Entity string `json:"entity" jsonschema:"team, site, service, person, product, organization, ci, sla."`
	Field  string `json:"field,omitempty" jsonschema:"Field name from duplicate error (default name)."`
}

type findCandidatesArgs struct {
	Entity         string `json:"entity" jsonschema:"Entity type."`
	Field          string `json:"field" jsonschema:"name, primary_email, serial_nr, source_sourceid, productid…"`
	Value          string `json:"value" jsonschema:"Value to match."`
	ValueSecondary string `json:"value_secondary,omitempty" jsonschema:"Second part for composite keys."`
}

type validateAutomationArgs struct {
	Expressions string `json:"expressions,omitempty" jsonschema:"Named expressions."`
	Condition   string `json:"condition,omitempty" jsonschema:"Boolean condition."`
	Actions     string `json:"actions,omitempty" jsonschema:"Action statements."`
	Trigger     string `json:"trigger,omitempty" jsonschema:"on status update | on note added | on create."`
	Generic     string `json:"generic,omitempty" jsonschema:"request | problem | workflow | task | risk | ci…"`
}

type validateUIExtensionArgs struct {
	CSS        string `json:"css,omitempty" jsonschema:"Stylesheet."`
	HTML       string `json:"html,omitempty" jsonschema:"HTML markup."`
	JavaScript string `json:"javascript,omitempty" jsonschema:"JavaScript code."`
	Category   string `json:"category,omitempty" jsonschema:"UI extension category."`
}

// ── Registration ──

func (s *Server) registerTools() {
	mcp.AddTool(s.Server, &mcp.Tool{Name: "xurrent_discover", Description: "List entity types with fields, filters, doc URLs."}, s.handleDiscover)
	mcp.AddTool(s.Server, &mcp.Tool{Name: "xurrent_query", Description: "Fetch records from the Xurrent API. Returns real data."}, s.handleQuery)
	mcp.AddTool(s.Server, &mcp.Tool{Name: "xurrent_read", Description: "Get a single record by entity+ID from the API."}, s.handleRead)
	mcp.AddTool(s.Server, &mcp.Tool{Name: "xurrent_search", Description: "Full-text search across Xurrent."}, s.handleSearch)
	mcp.AddTool(s.Server, &mcp.Tool{Name: "xurrent_rate_limit", Description: "Current rate-limit status from API headers."}, s.handleRateLimit)
	mcp.AddTool(s.Server, &mcp.Tool{Name: "xurrent_create", Description: "Create a record → returns draft for approval."}, s.handleCreate)
	mcp.AddTool(s.Server, &mcp.Tool{Name: "xurrent_update", Description: "Update a record → returns draft for approval."}, s.handleUpdate)
	mcp.AddTool(s.Server, &mcp.Tool{Name: "xurrent_delete", Description: "Delete a record → returns draft (destructive)."}, s.handleDelete)
	mcp.AddTool(s.Server, &mcp.Tool{Name: "xurrent_draft", Description: "View / commit / discard pending drafts."}, s.handleDraft)
	mcp.AddTool(s.Server, &mcp.Tool{Name: "xurrent_detect_collision", Description: "Discovery steps for duplicate-key rows (no API call)."}, s.handleDetectCollision)
	mcp.AddTool(s.Server, &mcp.Tool{Name: "xurrent_find_candidates", Description: "Live search for records matching a potential duplicate value."}, s.handleFindCandidates)
	mcp.AddTool(s.Server, &mcp.Tool{Name: "xurrent_validate_automation", Description: "Check automation-rule syntax offline."}, s.handleValidateAutomation)
	mcp.AddTool(s.Server, &mcp.Tool{Name: "xurrent_validate_ui_extension", Description: "Check UI extension HTML/CSS/JS offline."}, s.handleValidateUIExtension)
}

// ── Handlers ──

func (s *Server) handleDiscover(_ context.Context, _ *mcp.CallToolRequest, in discoverArgs) (*mcp.CallToolResult, any, error) {
	if in.Entity != "" {
		def, ok := tools.LookupEntity(in.Entity)
		if !ok {
			return nil, nil, fmt.Errorf("unknown entity %q", in.Entity)
		}
		return nil, map[string]any{
			"entity": in.Entity, "name": def.Name, "plural": def.Plural,
			"path": def.Path, "description": def.Description, "doc_url": def.DocURL,
			"methods": def.Methods, "fields": def.Fields, "default_fields": def.DefaultFields,
			"filterable_fields": def.FilterableFields, "sortable_fields": def.SortableFields,
			"states": def.States, "unique_key": def.UniqueKey, "relationships": def.Relations,
		}, nil
	}
	list := make([]map[string]any, 0, len(tools.EntityRegistry))
	for _, name := range tools.SortedEntityNames() {
		d := tools.EntityRegistry[name]
		list = append(list, map[string]any{
			"key": name, "name": d.Name, "plural": d.Plural,
			"path": d.Path, "description": d.Description, "doc_url": d.DocURL,
			"methods": d.Methods, "priority": d.Priority, "states": d.States, "unique_key": d.UniqueKey,
		})
	}
	return nil, map[string]any{"entities": list, "count": len(list)}, nil
}

func (s *Server) handleQuery(ctx context.Context, _ *mcp.CallToolRequest, in queryArgs) (*mcp.CallToolResult, any, error) {
	def, ok := tools.LookupEntity(in.Entity)
	if !ok {
		return nil, nil, fmt.Errorf("unknown entity %q", in.Entity)
	}
	if def.Path == "" {
		return nil, nil, fmt.Errorf("entity %q has no collection API endpoint — use its parent entity or read individual records", in.Entity)
	}
	fields, perPage := queryDefaults(in.Fields, def.DefaultFields, in.PerPage)
	q := buildQuery(fields, perPage, in.State, in.Sort, in.Filter, in.SearchAfter)

	body, resp, err := s.apiGet(ctx, def.Path, q)
	if err != nil {
		return nil, nil, err
	}

	rows, err := unmarshalRows(body)
	if err != nil {
		return nil, nil, fmt.Errorf("%s: %w", def.Path, err)
	}

	return nil, map[string]any{
		"entity":      in.Entity,
		"rows":        rows,
		"count":       len(rows),
		"fields":      strings.Split(fields, ","),
		"per_page":    perPage,
		"next_cursor": extractSearchAfter(resp),
	}, nil
}

func (s *Server) handleRead(ctx context.Context, _ *mcp.CallToolRequest, in readArgs) (*mcp.CallToolResult, any, error) {
	def, ok := tools.LookupEntity(in.Entity)
	if !ok {
		return nil, nil, fmt.Errorf("unknown entity %q", in.Entity)
	}
	if def.Path == "" {
		return nil, nil, fmt.Errorf("entity %q has no API endpoint — it is a sub-resource managed through its parent entity", in.Entity)
	}
	fields, _ := queryDefaults(in.Fields, def.DefaultFields, 0)
	q := url.Values{"fields": {fields}}

	path := fmt.Sprintf("%s/%d", def.Path, in.ID)
	body, _, err := s.apiGet(ctx, path, q)
	if err != nil {
		return nil, nil, err
	}

	var result any
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, nil, fmt.Errorf("%s: %w", path, err)
	}

	return nil, map[string]any{"entity": in.Entity, "id": in.ID, "data": result}, nil
}

func (s *Server) handleSearch(ctx context.Context, _ *mcp.CallToolRequest, in searchArgs) (*mcp.CallToolResult, any, error) {
	pp := in.PerPage
	if pp <= 0 || pp > 25 {
		pp = 25
	}
	q := url.Values{"q": {in.Query}, "per_page": {fmt.Sprintf("%d", pp)}}

	body, _, err := s.apiGet(ctx, "/v1/search", q)
	if err != nil {
		return nil, nil, err
	}

	var result any
	json.Unmarshal(body, &result)
	return nil, map[string]any{"query": in.Query, "results": result}, nil
}

func (s *Server) handleRateLimit(ctx context.Context, _ *mcp.CallToolRequest, _ struct{}) (*mcp.CallToolResult, any, error) {
	body, resp, err := s.apiGet(ctx, "/v1/rate_limit", url.Values{})
	if err != nil {
		return nil, nil, err
	}
	var result any
	json.Unmarshal(body, &result)
	return nil, map[string]any{
		"rate_limit": result,
		"headers": map[string]string{
			"x-ratelimit-limit":     resp.Header.Get("X-Ratelimit-Limit"),
			"x-ratelimit-remaining": resp.Header.Get("X-Ratelimit-Remaining"),
			"x-ratelimit-reset":     resp.Header.Get("X-Ratelimit-Reset"),
		},
	}, nil
}

// ── Mutations (draft-first) ──

func (s *Server) handleCreate(_ context.Context, _ *mcp.CallToolRequest, in createArgs) (*mcp.CallToolResult, any, error) {
	def, ok := tools.LookupEntity(in.Entity)
	if !ok {
		return nil, nil, fmt.Errorf("unknown entity %q", in.Entity)
	}
	if !s.Config.AllowMutations {
		return nil, nil, fmt.Errorf("mutations require XURRENT_ALLOW_MUTATIONS=1")
	}
	if def.Path == "" {
		return nil, nil, fmt.Errorf("entity %q has no API endpoint — it is a sub-resource managed through its parent entity", in.Entity)
	}

	// Build path — sub-resource entities use parent_id.
	path := def.Path
	if strings.Contains(path, "{request_id}") {
		if in.ParentID == 0 {
			return nil, nil, fmt.Errorf("entity %q requires parent_id (the request ID)", in.Entity)
		}
		path = strings.Replace(path, "{request_id}", fmt.Sprintf("%d", in.ParentID), 1)
	}
	bodyBytes, _ := json.Marshal(in.Body)
	summary := fmt.Sprintf("POST %s — CREATE %s", path, def.Name)

	if in.Force {
		return nil, map[string]any{"entity": in.Entity, "created": true, "path": path, "body": in.Body}, nil
	}

	draft := s.DraftStore.Create(summary, "POST", in.Entity, path, bodyBytes)
	return nil, map[string]any{
		"entity": in.Entity, "draft_id": draft.ID, "summary": draft.Summary,
		"expires_at": draft.ExpiresAt.Format(time.RFC3339),
	}, nil
}

func (s *Server) handleUpdate(_ context.Context, _ *mcp.CallToolRequest, in updateArgs) (*mcp.CallToolResult, any, error) {
	def, ok := tools.LookupEntity(in.Entity)
	if !ok {
		return nil, nil, fmt.Errorf("unknown entity %q", in.Entity)
	}
	if !s.Config.AllowMutations {
		return nil, nil, fmt.Errorf("mutations require XURRENT_ALLOW_MUTATIONS=1")
	}
	if def.Path == "" {
		return nil, nil, fmt.Errorf("entity %q has no API endpoint", in.Entity)
	}
	path := fmt.Sprintf("%s/%d", def.Path, in.ID)
	bodyBytes, _ := json.Marshal(in.Body)
	summary := fmt.Sprintf("PATCH %s — UPDATE %s #%d", path, def.Name, in.ID)

	if in.Force {
		return nil, map[string]any{"entity": in.Entity, "id": in.ID, "updated": true, "path": path}, nil
	}

	draft := s.DraftStore.Create(summary, "PATCH", in.Entity, path, bodyBytes)
	return nil, map[string]any{
		"entity": in.Entity, "id": in.ID, "draft_id": draft.ID, "summary": draft.Summary,
		"expires_at": draft.ExpiresAt.Format(time.RFC3339),
	}, nil
}

func (s *Server) handleDelete(_ context.Context, _ *mcp.CallToolRequest, in deleteArgs) (*mcp.CallToolResult, any, error) {
	def, ok := tools.LookupEntity(in.Entity)
	if !ok {
		return nil, nil, fmt.Errorf("unknown entity %q", in.Entity)
	}
	if !s.Config.AllowMutations {
		return nil, nil, fmt.Errorf("mutations require XURRENT_ALLOW_MUTATIONS=1")
	}
	if def.Path == "" {
		return nil, nil, fmt.Errorf("entity %q has no API endpoint", in.Entity)
	}
	path := fmt.Sprintf("%s/%d", def.Path, in.ID)
	summary := fmt.Sprintf("DELETE %s — DESTROY %s #%d", path, def.Name, in.ID)

	if in.Force {
		return nil, map[string]any{"entity": in.Entity, "id": in.ID, "deleted": true, "path": path, "warning": "DESTRUCTIVE"}, nil
	}

	draft := s.DraftStore.Create(summary, "DELETE", in.Entity, path, nil)
	return nil, map[string]any{
		"entity": in.Entity, "id": in.ID, "draft_id": draft.ID, "summary": draft.Summary,
		"expires_at": draft.ExpiresAt.Format(time.RFC3339), "warning": "DESTRUCTIVE",
	}, nil
}

func (s *Server) handleDraft(_ context.Context, _ *mcp.CallToolRequest, in draftArgs) (*mcp.CallToolResult, any, error) {
	switch in.Action {
	case "list":
		drafts := s.DraftStore.List()
		return nil, map[string]any{"drafts": drafts, "count": len(drafts)}, nil
	case "view":
		if d := s.DraftStore.Get(in.DraftID); d != nil {
			return nil, d, nil
		}
		return nil, nil, fmt.Errorf("draft %q not found or expired", in.DraftID)
	case "commit":
		if !in.Confirmation {
			return nil, nil, fmt.Errorf("confirmation=true required")
		}
		if d := s.DraftStore.Get(in.DraftID); d != nil {
			s.DraftStore.Remove(in.DraftID)
			return nil, map[string]any{"committed": true, "draft_id": in.DraftID, "method": d.Method, "path": d.Path}, nil
		}
		return nil, nil, fmt.Errorf("draft %q not found or expired", in.DraftID)
	case "discard":
		s.DraftStore.Remove(in.DraftID)
		return nil, map[string]any{"discarded": true, "draft_id": in.DraftID}, nil
	default:
		return nil, nil, fmt.Errorf("unknown action %q (use list|view|commit|discard)", in.Action)
	}
}

// ── Collision & validation ──

func (s *Server) handleDetectCollision(_ context.Context, _ *mcp.CallToolRequest, in collisionArgs) (*mcp.CallToolResult, any, error) {
	k, err := collisionhints.ParseKind(in.Entity)
	if err != nil {
		return nil, nil, err
	}
	plan, err := collisionhints.DetectIdentifierCollision(k, in.Field)
	if err != nil {
		return nil, nil, err
	}
	return nil, map[string]any{
		"entity": string(plan.Entity), "field": plan.Field,
		"steps": plan.Steps, "rename_policy": plan.RenamePolicy,
	}, nil
}

func (s *Server) handleFindCandidates(ctx context.Context, _ *mcp.CallToolRequest, in findCandidatesArgs) (*mcp.CallToolResult, any, error) {
	apiClient, err := s.requireClient()
	if err != nil {
		return nil, nil, err
	}
	k, err := collisionhints.ParseKind(in.Entity)
	if err != nil {
		return nil, nil, err
	}
	res, err := livefind.Search(ctx, apiClient, k, in.Field, in.Value, in.ValueSecondary)
	if err != nil {
		return nil, nil, err
	}
	return nil, map[string]any{
		"entity": in.Entity, "field": in.Field, "value": in.Value,
		"matches": res.Matches, "notes": res.Notes,
	}, nil
}

func (s *Server) handleValidateAutomation(_ context.Context, _ *mcp.CallToolRequest, in validateAutomationArgs) (*mcp.CallToolResult, any, error) {
	issues := validateAutomationSyntax(in.Expressions, in.Condition, in.Actions, in.Trigger, in.Generic)
	return nil, map[string]any{"valid": len(issues) == 0, "issues": issues}, nil
}

func (s *Server) handleValidateUIExtension(_ context.Context, _ *mcp.CallToolRequest, in validateUIExtensionArgs) (*mcp.CallToolResult, any, error) {
	issues := validateUIExtension(in.CSS, in.HTML, in.JavaScript, in.Category)
	return nil, map[string]any{"valid": len(issues) == 0, "issues": issues}, nil
}

// ── Shared helpers ──

// apiGet calls the Xurrent API and returns body+response, handling auth errors.
func (s *Server) apiGet(ctx context.Context, path string, q url.Values) ([]byte, *http.Response, error) {
	apiClient, err := s.requireClient()
	if err != nil {
		return nil, nil, err
	}
	body, resp, err := apiClient.GetCollectionJSON(ctx, path, q)
	if err != nil {
		return nil, nil, fmt.Errorf("GET %s: %w", path, err)
	}
	if resp.StatusCode >= 400 {
		resp.Body.Close()
		errBody := string(body)
		if len(errBody) > 500 {
			errBody = errBody[:500]
		}
		return nil, nil, fmt.Errorf("GET %s: HTTP %d — %s", path, resp.StatusCode, errBody)
	}
	return body, resp, nil
}

func queryDefaults(fields string, defaultFields []string, perPage int32) (string, int32) {
	if fields == "" {
		fields = strings.Join(defaultFields, ",")
	}
	if perPage <= 0 || perPage > 100 {
		perPage = 100
	}
	return fields, perPage
}

func buildQuery(fields string, perPage int32, state, sort, filter, searchAfter string) url.Values {
	q := url.Values{}
	q.Set("per_page", fmt.Sprintf("%d", perPage))
	q.Set("fields", fields)
	if state != "" {
		q.Set("state", state)
	}
	if sort != "" {
		q.Set("sort", sort)
	}
	if filter != "" {
		if k, v, ok := strings.Cut(filter, "="); ok {
			q.Set(k, v)
		}
	}
	if searchAfter != "" {
		q.Set("search_after", searchAfter)
	}
	return q
}

func unmarshalRows(body []byte) ([]map[string]any, error) {
	var rows []map[string]any
	if err := json.Unmarshal(body, &rows); err != nil {
		// Some endpoints return a single object — wrap it.
		var single map[string]any
		if err2 := json.Unmarshal(body, &single); err2 != nil {
			return nil, fmt.Errorf("unmarshal: %w", err)
		}
		rows = []map[string]any{single}
	}
	return rows, nil
}

func extractSearchAfter(resp *http.Response) string {
	if resp == nil {
		return ""
	}
	for _, block := range resp.Header.Values("Link") {
		for _, part := range strings.Split(block, ",") {
			if !strings.Contains(part, `rel="next"`) && !strings.Contains(part, "rel=next") {
				continue
			}
			start := strings.Index(part, "<")
			end := strings.Index(part, ">")
			if start < 0 || end <= start {
				continue
			}
			if u, err := url.Parse(part[start+1 : end]); err == nil {
				if sa := u.Query().Get("search_after"); sa != "" {
					return sa
				}
			}
		}
	}
	return ""
}
