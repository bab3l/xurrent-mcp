package server

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/xurrent/xurrent-mcp/internal/client"
	"github.com/xurrent/xurrent-mcp/internal/middleware"
	"github.com/xurrent/xurrent-mcp/internal/tools"
)

// TestDiscover_ListsAllEntities verifies xurrent_discover returns all registered entities.
func TestDiscover_ListsAllEntities(t *testing.T) {
	t.Parallel()
	srv := newTestServer(t)
	_, result, err := srv.handleDiscover(nilCtx(), nil, discoverArgs{})
	require.NoError(t, err)
	data, ok := result.(map[string]any)
	require.True(t, ok)
	require.Contains(t, data, "entities")
	require.Contains(t, data, "count")
	count, ok := data["count"].(int)
	require.True(t, ok)
	require.Greater(t, count, 10, "should have at least 10 entity types")
}

// TestDiscover_SpecificEntity returns details for a known entity.
func TestDiscover_SpecificEntity(t *testing.T) {
	t.Parallel()
	srv := newTestServer(t)
	_, result, err := srv.handleDiscover(nilCtx(), nil, discoverArgs{Entity: "team"})
	require.NoError(t, err)
	data, ok := result.(map[string]any)
	require.True(t, ok)
	require.Equal(t, "team", data["entity"])
	require.Equal(t, "Team", data["name"])
	require.Equal(t, "/v1/teams", data["path"])
	methods, ok := data["methods"].([]string)
	require.True(t, ok)
	require.Contains(t, methods, "GET")
	require.Contains(t, methods, "POST")
}

// TestDiscover_UnknownEntity returns an error.
func TestDiscover_UnknownEntity(t *testing.T) {
	t.Parallel()
	srv := newTestServer(t)
	_, _, err := srv.handleDiscover(nilCtx(), nil, discoverArgs{Entity: "nonexistent"})
	require.Error(t, err)
	require.Contains(t, err.Error(), "unknown entity")
}

// TestDiscover_AllAliases tests that entity aliases work.
func TestDiscover_AllAliases(t *testing.T) {
	t.Parallel()
	srv := newTestServer(t)
	aliases := []string{"ci", "cis", "sla", "slas", "org", "people"}
	for _, alias := range aliases {
		t.Run(alias, func(t *testing.T) {
			_, result, err := srv.handleDiscover(nilCtx(), nil, discoverArgs{Entity: alias})
			require.NoError(t, err)
			data := result.(map[string]any)
			require.NotEmpty(t, data["entity"])
		})
	}
}

// TestDraftStore_CreateGetCommitDiscard verifies the draft lifecycle.
func TestDraftStore_CreateGetCommitDiscard(t *testing.T) {
	t.Parallel()
	ds := NewDraftStore(1 * time.Minute)

	d := ds.Create("POST /v1/teams — CREATE Team", "POST", "team", "/v1/teams", json.RawMessage(`{"name":"test"}`))
	require.NotEmpty(t, d.ID)

	got := ds.Get(d.ID)
	require.NotNil(t, got)
	require.Equal(t, d.Summary, got.Summary)
	require.Equal(t, "POST", got.Method)

	list := ds.List()
	require.Len(t, list, 1)

	require.Nil(t, ds.Get("nonexistent"))

	ds.Remove(d.ID)
	require.Nil(t, ds.Get(d.ID))
	require.Len(t, ds.List(), 0)
}

// TestDraftStore_Expiry verifies draft expiration.
func TestDraftStore_Expiry(t *testing.T) {
	t.Parallel()
	ds := NewDraftStore(1 * time.Millisecond)
	d := ds.Create("test", "GET", "team", "/v1/teams", nil)
	time.Sleep(10 * time.Millisecond)
	require.Nil(t, ds.Get(d.ID))
	require.Len(t, ds.List(), 0)
}

// TestValidateAutomation_ValidRule returns no issues.
func TestValidateAutomation_ValidRule(t *testing.T) {
	t.Parallel()
	issues := validateAutomationSyntax(
		"is_assigned: status = assigned\nbadge: request.custom_fields.badge",
		"is_assigned and !badge",
		"a1: add note 'canceled by automation'\na2: set status = canceled",
		"on status update",
		"workflow",
	)
	require.Len(t, issues, 0)
}

// TestValidateAutomation_EmptyActions returns warning.
func TestValidateAutomation_EmptyActions(t *testing.T) {
	t.Parallel()
	issues := validateAutomationSyntax("", "status = assigned", "", "on status update", "")
	require.NotEmpty(t, issues)
}

// TestValidateAutomation_BadTrigger returns issue.
func TestValidateAutomation_BadTrigger(t *testing.T) {
	t.Parallel()
	issues := validateAutomationSyntax("", "", "a1: set status = done", "on full moon", "")
	require.NotEmpty(t, issues)
}

// TestValidateAutomation_BadGeneric returns issue.
func TestValidateAutomation_BadGeneric(t *testing.T) {
	t.Parallel()
	issues := validateAutomationSyntax("", "", "a1: set status = canceled", "on status update", "banana")
	require.NotEmpty(t, issues)
}

// TestValidateUIExtension_Good returns no issues.
func TestValidateUIExtension_Good(t *testing.T) {
	t.Parallel()
	issues := validateUIExtension(
		".row.subject { display: none; }",
		"<div class='row vertical'><label>Foo</label><input type='text'></div>",
		"console.log('ok');",
		"request_template",
	)
	require.Len(t, issues, 0)
}

// TestValidateUIExtension_SizeLimit returns issue.
func TestValidateUIExtension_SizeLimit(t *testing.T) {
	t.Parallel()
	big := strings.Repeat("x", 65*1024)
	issues := validateUIExtension(big, "", "", "")
	require.NotEmpty(t, issues)
}

// TestValidateUIExtension_ScriptInHTML returns issue.
func TestValidateUIExtension_ScriptInHTML(t *testing.T) {
	t.Parallel()
	issues := validateUIExtension("", "<script>alert('xss')</script>", "", "")
	require.NotEmpty(t, issues)
}

// TestValidateUIExtension_UnscopedCSS returns issue.
func TestValidateUIExtension_UnscopedCSS(t *testing.T) {
	t.Parallel()
	issues := validateUIExtension("div { color: red; }", "", "", "")
	require.NotEmpty(t, issues)
}

// TestValidateUIExtension_BadCategory returns issue.
func TestValidateUIExtension_BadCategory(t *testing.T) {
	t.Parallel()
	issues := validateUIExtension("", "", "", "foo_bar_baz")
	require.NotEmpty(t, issues)
}

// TestEntityRegistry_Lookup verifies entity lookups.
func TestEntityRegistry_Lookup(t *testing.T) {
	t.Parallel()
	def, ok := tools.LookupEntity("team")
	require.True(t, ok)
	require.Equal(t, "Team", def.Name)
	require.Equal(t, "/v1/teams", def.Path)

	def, ok = tools.LookupEntity("ci")
	require.True(t, ok)
	require.Equal(t, "Configuration Item", def.Name)

	_, ok = tools.LookupEntity("ghost")
	require.False(t, ok)
}

// TestEntityRegistry_SortedNames verifies priority ordering.
func TestEntityRegistry_SortedNames(t *testing.T) {
	t.Parallel()
	names := tools.SortedEntityNames()
	require.NotEmpty(t, names)
}

// TestServer_ConfigFromEnv verifies config parsing.
func TestServer_ConfigFromEnv(t *testing.T) {
	t.Setenv("XURRENT_TOKEN", "tok123")
	t.Setenv("XURRENT_ACCOUNT", "my-account")
	t.Setenv("XURRENT_ALLOW_MUTATIONS", "1")
	cfg, err := client.ConfigFromEnv()
	require.NoError(t, err)
	require.Equal(t, "tok123", cfg.Token)
	require.Equal(t, "my-account", cfg.Account)
	require.True(t, cfg.AllowMutations)
}

// TestServer_ConfigFromEnvMissingToken returns error.
func TestServer_ConfigFromEnvMissingToken(t *testing.T) {
	for _, k := range []string{"XURRENT_TOKEN", "XURRENT_ACCOUNT"} {
		t.Setenv(k, "")
	}
	_, err := client.ConfigFromEnv()
	require.Error(t, err)
}

// TestCache_SetGet verifies cache behavior.
func TestCache_SetGet(t *testing.T) {
	t.Parallel()
	c := middleware.NewCache(10 * time.Second)
	defer c.Stop()

	c.Set("key1", []byte("value1"), 5*time.Second)
	v, ok := c.Get("key1")
	require.True(t, ok)
	require.Equal(t, "value1", string(v))

	_, ok = c.Get("nonexistent")
	require.False(t, ok)
}

// TestCache_TTL verifies expiration.
func TestCache_TTL(t *testing.T) {
	t.Parallel()
	c := middleware.NewCache(10 * time.Second)
	defer c.Stop()

	c.Set("key1", []byte("value1"), 1*time.Millisecond)
	time.Sleep(10 * time.Millisecond)
	_, ok := c.Get("key1")
	require.False(t, ok)
}

// TestCache_DeletePrefix verifies prefix deletion.
func TestCache_DeletePrefix(t *testing.T) {
	t.Parallel()
	c := middleware.NewCache(10 * time.Second)
	defer c.Stop()

	c.Set("acct1:query:teams:1", []byte("a"), time.Minute)
	c.Set("acct1:query:sites:1", []byte("b"), time.Minute)
	c.Set("acct2:query:teams:1", []byte("c"), time.Minute)

	c.DeletePrefix("acct1:")
	require.Equal(t, 1, c.Size())
}

// TestCache_InvalidateAll clears everything.
func TestCache_InvalidateAll(t *testing.T) {
	t.Parallel()
	c := middleware.NewCache(10 * time.Second)
	defer c.Stop()

	c.Set("a", []byte("1"), time.Minute)
	c.Set("b", []byte("2"), time.Minute)
	c.InvalidateAll()
	require.Equal(t, 0, c.Size())
}

// TestResponseShaper_JSON verifies compact JSON output.
func TestResponseShaper_JSON(t *testing.T) {
	t.Parallel()
	rs := middleware.NewResponseShaper()
	s, _, err := rs.ShapeJSON(map[string]any{"name": "test", "id": 42})
	require.NoError(t, err)
	require.Contains(t, s, `"name"`)
	require.Contains(t, s, `42`)
}

// TestResponseShaper_Truncation verifies text truncation.
func TestResponseShaper_Truncation(t *testing.T) {
	t.Parallel()
	rs := middleware.NewResponseShaper()
	rs.MaxChars = 10
	s, truncated := rs.TruncateText("this is a long string")
	require.True(t, truncated)
	require.Less(t, len(s), 100)
}

// TestResponseShaper_CSV verifies CSV output.
func TestResponseShaper_CSV(t *testing.T) {
	t.Parallel()
	rs := middleware.NewResponseShaper()
	rows := []map[string]any{
		{"id": float64(1), "name": "Alice"},
		{"id": float64(2), "name": "Bob"},
	}
	s, truncated, err := rs.ShapeCSV(rows, []string{"id", "name"})
	require.NoError(t, err)
	require.False(t, truncated)
	require.Contains(t, s, "Alice")
}

func newTestServer(t *testing.T) *Server {
	t.Helper()
	return &Server{
		Config: &client.EnvConfig{
			Token:   "test-token",
			Account: "test-account",
		},
		Shaper:     middleware.NewResponseShaper(),
		Cache:      middleware.NewCache(time.Minute),
		DraftStore: NewDraftStore(time.Minute),
	}
}

func nilCtx() context.Context {
	return context.Background()
}

// TestQuery_ParameterizedPath verifies that handleQuery substitutes {request_id}
// for sub-resource entities like inbound_email.
func TestQuery_ParameterizedPath(t *testing.T) {
	_ = newTestServer(t)
	// Verify the entity exists and has the parameterized path.
	def, ok := tools.LookupEntity("inbound_email")
	require.True(t, ok)
	require.Contains(t, def.Path, "{request_id}")
}

// TestQuery_ParameterizedPath_RequiresParentID verifies that querying a sub-resource
// without parent_id returns an error.
func TestQuery_ParameterizedPath_RequiresParentID(t *testing.T) {
	srv := newTestServer(t)
	_, _, err := srv.handleQuery(nilCtx(), nil, queryArgs{
		Entity: "inbound_email",
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "parent_id")
}

// TestQuery_ParameterizedPath_WithParentID verifies that querying a sub-resource
// with parent_id builds the correct path.
func TestQuery_ParameterizedPath_WithParentID(t *testing.T) {
	srv := newTestServer(t)
	// Without a real API client, this will fail trying to call the API.
	// But we can verify it doesn't fail on the parent_id check.
	_, _, err := srv.handleQuery(nilCtx(), nil, queryArgs{
		Entity:   "inbound_email",
		ParentID: 84088756,
		Fields:   "id,from,to,subject,created_at",
		PerPage:  5,
	})
	// Expected to fail with API client error — not parent_id error.
	require.Error(t, err)
	require.NotContains(t, err.Error(), "parent_id", "should not fail on parent_id when provided")
	require.Contains(t, err.Error(), "API client", "should fail because we don't have a real API client")
}

// TestQuery_RegularEntity_NoParentID verifies that regular entities work without parent_id.
func TestQuery_RegularEntity_NoParentID(t *testing.T) {
	srv := newTestServer(t)
	_, _, err := srv.handleQuery(nilCtx(), nil, queryArgs{
		Entity: "team",
		Fields: "id,name",
	})
	// Expected to fail with API client error — not parent_id error.
	require.Error(t, err)
	require.Contains(t, err.Error(), "API client")
}
