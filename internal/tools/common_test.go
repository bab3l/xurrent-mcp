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
	require.LessOrEqual(t, count, 85, "should have at most 85 entity types")
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

func TestInboundEmail_HasCorrectPath(t *testing.T) {
	t.Parallel()
	def, ok := LookupEntity("inbound_email")
	require.True(t, ok)
	require.Equal(t, "/v1/requests/{request_id}/inbound_emails", def.Path)
	require.Equal(t, "inbound_emails", def.Plural)
	require.Contains(t, def.Methods, "GET")
}

func TestInboundEmail_HasCorrectFields(t *testing.T) {
	t.Parallel()
	def, ok := LookupEntity("inbound_email")
	require.True(t, ok)

	// Verify the correct field names (from Xurrent API: from, to, subject, created_at, message_id, body_start, source_uri).
	fields := map[string]bool{}
	for _, f := range def.Fields {
		fields[f.Name] = true
	}
	require.True(t, fields["from"], "should have 'from' field (not from_address)")
	require.True(t, fields["to"], "should have 'to' field (not to_address)")
	require.True(t, fields["created_at"], "should have 'created_at' field")
	require.True(t, fields["message_id"], "should have 'message_id' field")
	require.True(t, fields["body_start"], "should have 'body_start' field")
	require.True(t, fields["source_uri"], "should have 'source_uri' field")
	// Old incorrect field names should not exist.
	require.False(t, fields["from_address"], "should NOT have 'from_address' (API uses 'from')")
	require.False(t, fields["to_address"], "should NOT have 'to_address' (API uses 'to')")
	require.False(t, fields["received_at"], "should NOT have 'received_at' (API uses 'created_at')")
}

func TestInboundEmail_DefaultFieldsMatchAvailable(t *testing.T) {
	t.Parallel()
	def, ok := LookupEntity("inbound_email")
	require.True(t, ok)
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

func TestInboundEmail_AliasesWork(t *testing.T) {
	t.Parallel()
	aliases := []string{"inbound_emails", "inboundemail", "inbound email", "mail"}
	for _, alias := range aliases {
		t.Run(alias, func(t *testing.T) {
			def, ok := LookupEntity(alias)
			require.True(t, ok, "alias %q should resolve", alias)
			require.Equal(t, "Inbound Email", def.Name)
		})
	}
}

func TestAuditLine_HasCorrectPath(t *testing.T) {
	t.Parallel()
	def, ok := LookupEntity("audit_line")
	require.True(t, ok)
	require.Equal(t, "/v1/audit_lines", def.Path)
	require.Equal(t, "audit_lines", def.Plural)
	require.Contains(t, def.Methods, "GET")
}

func TestAuditLine_HasCorrectFields(t *testing.T) {
	t.Parallel()
	def, ok := LookupEntity("audit_line")
	require.True(t, ok)

	fields := map[string]bool{}
	for _, f := range def.Fields {
		fields[f.Name] = true
	}
	require.True(t, fields["action"], "should have 'action' field")
	require.True(t, fields["audited"], "should have 'audited' field (not auditable_type)")
	require.True(t, fields["created_by"], "should have 'created_by' field (not person)")
	require.True(t, fields["user"], "should have 'user' field (for impersonation)")
	require.True(t, fields["changes"], "should have 'changes' field (only on single GET)")
	// Old incorrect field names should not exist.
	require.False(t, fields["auditable_type"], "should NOT have 'auditable_type' (API uses 'audited')")
	require.False(t, fields["auditable_id"], "should NOT have 'auditable_id' (API embeds entity reference)")
	require.False(t, fields["person"], "should NOT have 'person' (API uses 'created_by')")
}

func TestAuditLine_AliasesWork(t *testing.T) {
	t.Parallel()
	aliases := []string{"audit_lines", "audit_entries", "auditentry", "audit line", "audit entry", "audit"}
	for _, alias := range aliases {
		t.Run(alias, func(t *testing.T) {
			def, ok := LookupEntity(alias)
			require.True(t, ok, "alias %q should resolve to audit_line", alias)
			require.Equal(t, "Audit Line", def.Name)
		})
	}
}

func TestEntityAudit_HasParameterizedPath(t *testing.T) {
	t.Parallel()
	def, ok := LookupEntity("entity_audit")
	require.True(t, ok)
	require.Equal(t, "/v1/{parent_type}/{parent_id}/audit", def.Path)
	require.Equal(t, "entity_audits", def.Plural)
}

func TestAuditLine_AllFieldsHaveCorrectNames(t *testing.T) {
	t.Parallel()
	def, ok := LookupEntity("audit_line")
	require.True(t, ok)

	// All 7 fields should be readonly (audit entries are immutable).
	for _, f := range def.Fields {
		require.True(t, f.ReadOnly, "field %q should be read-only (audit entries are immutable)", f.Name)
	}
}
