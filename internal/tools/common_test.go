package tools

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLookupEntity_Exact(t *testing.T) {
	t.Parallel()
	def, ok := LookupEntity("team")
	require.True(t, ok)
	require.Equal(t, "Team", def.Name)
	require.Equal(t, "/v1/teams", def.Path)
	require.Contains(t, def.Methods, "GET")
	require.Contains(t, def.Methods, "POST")
	require.Contains(t, def.Methods, "PATCH")
}

func TestLookupEntity_CaseInsensitive(t *testing.T) {
	t.Parallel()
	_, ok := LookupEntity("TEAM")
	require.True(t, ok)
	_, ok = LookupEntity("Team")
	require.True(t, ok)
	_, ok = LookupEntity("  team  ")
	require.True(t, ok)
}

func TestLookupEntity_Plural(t *testing.T) {
	t.Parallel()
	def, ok := LookupEntity("teams")
	require.True(t, ok)
	require.Equal(t, "Team", def.Name)
}

func TestLookupEntity_Aliases(t *testing.T) {
	t.Parallel()
	cases := []string{
		"ci", "cis", "configuration_items", "configuration_item",
		"sla", "slas", "service_level_agreements",
		"org", "orgs", "organizations",
		"people", "person",
		"automation_rules", "automationrule", "automation rule",
		"ui_extensions", "uiextensions", "ui extension",
		"requests", "workflows", "problems", "projects",
		"service_instances", "serviceinstance",
		"service_offerings", "serviceoffering",
		"request_templates", "requesttemplate",
		"workflow_templates", "workflowtemplate",
		"tasks", "task_templates", "tasktemplate",
	}
	for _, alias := range cases {
		t.Run(alias, func(t *testing.T) {
			_, ok := LookupEntity(alias)
			require.True(t, ok, "alias %q should resolve", alias)
		})
	}
}

func TestLookupEntity_NotFound(t *testing.T) {
	t.Parallel()
	_, ok := LookupEntity("nonexistent_entity")
	require.False(t, ok)
	_, ok = LookupEntity("")
	require.False(t, ok)
}

func TestSortedEntityNames_Ordered(t *testing.T) {
	t.Parallel()
	names := SortedEntityNames()
	require.NotEmpty(t, names)
	require.Greater(t, len(names), 15, "should have at least 15 entity types")

	// Verify priority ordering: all P0 before P1, all P1 before P2.
	lastPrio := -1
	for _, name := range names {
		def := EntityRegistry[name]
		require.GreaterOrEqual(t, def.Priority, lastPrio)
		lastPrio = def.Priority
	}
}

func TestEntityRegistry_AllHavePaths(t *testing.T) {
	t.Parallel()
	for name, def := range EntityRegistry {
		// Entities with API methods must have a Path; informational entities may not.
		if len(def.Methods) > 0 {
			require.NotEmpty(t, def.Path, "entity %q has methods but empty Path", name)
		}
		require.NotEmpty(t, def.Name, "entity %q has empty Name", name)
		require.NotEmpty(t, def.Plural, "entity %q has empty Plural", name)
		require.NotEmpty(t, def.Description, "entity %q has empty Description", name)
		require.NotEmpty(t, def.DocURL, "entity %q has empty DocURL", name)
	}
}

func TestEntityRegistry_Priority0HasFullMethods(t *testing.T) {
	t.Parallel()
	p0Entities := []string{"team", "site", "service", "service_instance", "person",
		"organization", "configuration_item", "service_level_agreement",
		"request", "workflow", "ui_extension", "automation_rule"}
	for _, name := range p0Entities {
		t.Run(name, func(t *testing.T) {
			def, ok := EntityRegistry[name]
			require.True(t, ok)
			require.Equal(t, 0, def.Priority)
			require.LessOrEqual(t, len(def.Fields), 20) // sanity: not too many fields
			require.GreaterOrEqual(t, len(def.Fields), 1)
		})
	}
}

func TestEntitySummary(t *testing.T) {
	t.Parallel()
	s := EntitySummary("team")
	require.Contains(t, s, "Team")
	require.Contains(t, s, "/v1/teams")
	require.Contains(t, s, "GET")

	s = EntitySummary("nonexistent")
	require.Contains(t, s, "Unknown")
}

func TestLookupEntity_AllRegisteredCount(t *testing.T) {
	t.Parallel()
	count := len(EntityRegistry)
	require.GreaterOrEqual(t, count, 28, "should have at least 28 entity types")
	require.LessOrEqual(t, count, 40, "should have at most 40 entity types")
}

func TestFieldDef_ReadOnlyDefault(t *testing.T) {
	t.Parallel()
	def, _ := LookupEntity("team")
	for _, f := range def.Fields {
		if f.Name == "id" {
			require.True(t, f.ReadOnly, "id should be readonly")
		}
	}
}

func TestEntityRegistry_DefaultFieldsExist(t *testing.T) {
	t.Parallel()
	def, _ := LookupEntity("team")
	for _, df := range def.DefaultFields {
		found := false
		for _, f := range def.Fields {
			if f.Name == df {
				found = true
				break
			}
		}
		require.True(t, found, "default field %q not defined in fields", df)
	}
}
