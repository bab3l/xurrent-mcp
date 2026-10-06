// Package tools provides MCP tool handlers for the xurrent-mcp server.
//
// The entity registry (common.go) defines all Xurrent API entity types with
// their paths, fields, filtering capabilities, and documentation URLs.
// Tools delegate to this registry for entity-aware dispatch.
package tools

import (
	"fmt"
	"sort"
	"strings"
)

// EntityDef describes a Xurrent API entity/resource type with its metadata.
type EntityDef struct {
	// API path prefix (e.g. "/v1/teams").
	Path string
	// Human-readable name (e.g. "Team").
	Name string
	// Plural name for lists (e.g. "teams").
	Plural string
	// Description for discovery.
	Description string
	// DocURL is the canonical documentation page.
	DocURL string
	// Methods supported: "GET", "POST", "PATCH", "DELETE".
	Methods []string
	// Fields that can be used with ?fields= parameter.
	Fields []FieldDef
	// DefaultFields are the fields returned when no fields= is specified.
	DefaultFields []string
	// FilterableFields lists fields that accept server-side filters per Xurrent filtering docs.
	FilterableFields []string
	// SortableFields lists fields accepted by ?sort=.
	SortableFields []string
	// State-based lists (e.g. "disabled", "enabled" for teams).
	States []string
	// UniqueKey describes the field(s) that must be unique.
	UniqueKey string
	// Priority: 0=core, 1=important, 2=auxiliary.
	Priority int
	// Relationships to other entities for domain graph.
	Relations []EntityRelation
}

// FieldDef describes a field on an entity.
type FieldDef struct {
	Name        string
	Type        string
	Description string
	Required    bool
	ReadOnly    bool
	Default     string
}

// EntityRelation describes a relationship between entities.
type EntityRelation struct {
	Entity      string // Target entity name.
	Cardinality string // "1:1", "1:N", "N:M".
	Via         string // The field name that holds the reference.
	Description string // Brief description of the relationship.
}

// EntityRegistry is a lookup table for all supported Xurrent entity types.
var EntityRegistry = map[string]EntityDef{
	"team": {
		Path:        "/v1/teams",
		Name:        "Team",
		Plural:      "teams",
		Description: "A team of people in Xurrent. Teams can be members of service instances, assigned to requests/tasks, and have calendars.",
		DocURL:      "https://developer.xurrent.com/v1/teams/",
		Methods:     []string{"GET", "POST", "PATCH"},
		Fields: []FieldDef{
			{Name: "id", Type: "integer", Description: "Unique ID", ReadOnly: true},
			{Name: "name", Type: "string", Description: "Team name (unique per account)", Required: true},
			{Name: "disabled", Type: "boolean", Description: "Whether the team is disabled", Default: "false"},
			{Name: "remarks", Type: "text", Description: "Remarks/notes"},
			{Name: "manager", Type: "reference", Description: "Team manager (person)"},
			{Name: "auto_assign", Type: "boolean", Description: "Auto-assign members to requests"},
			{Name: "source", Type: "string", Description: "Source system identifier"},
			{Name: "sourceID", Type: "string", Description: "Source system record ID"},
		},
		DefaultFields:   []string{"id", "name", "disabled"},
		FilterableFields: []string{"id", "name", "source", "sourceID"},
		SortableFields:  []string{"id", "name"},
		States:          []string{"enabled", "disabled"},
		UniqueKey:       "name",
		Priority:        0,
		Relations: []EntityRelation{
			{Entity: "person", Cardinality: "N:M", Via: "members", Description: "Team members are people"},
			{Entity: "service_instance", Cardinality: "N:M", Via: "service_instances", Description: "Teams support service instances"},
			{Entity: "calendar", Cardinality: "1:N", Via: "calendars", Description: "Team has calendars"},
		},
	},
	"site": {
		Path:        "/v1/sites",
		Name:        "Site",
		Plural:      "sites",
		Description: "A physical or logical location. People, CIs, and service instances can be linked to sites.",
		DocURL:      "https://developer.xurrent.com/v1/sites/",
		Methods:     []string{"GET", "POST", "PATCH"},
		Fields: []FieldDef{
			{Name: "id", Type: "integer", Description: "Unique ID", ReadOnly: true},
			{Name: "name", Type: "string", Description: "Site name (unique per account)", Required: true},
			{Name: "disabled", Type: "boolean", Description: "Whether the site is disabled", Default: "false"},
			{Name: "address", Type: "string", Description: "Physical address"},
			{Name: "source", Type: "string", Description: "Source system identifier"},
			{Name: "sourceID", Type: "string", Description: "Source system record ID"},
		},
		DefaultFields:   []string{"id", "name", "disabled"},
		FilterableFields: []string{"id", "name", "source", "sourceID"},
		SortableFields:  []string{"id", "name"},
		States:          []string{"enabled", "disabled"},
		UniqueKey:       "name",
		Priority:        0,
		Relations: []EntityRelation{
			{Entity: "person", Cardinality: "1:N", Via: "site", Description: "People belong to a site"},
			{Entity: "configuration_item", Cardinality: "1:N", Via: "site", Description: "CIs located at a site"},
		},
	},
	"service": {
		Path:        "/v1/services",
		Name:        "Service",
		Plural:      "services",
		Description: "A service delivered to customers. Services have service offerings, instances, and SLAs.",
		DocURL:      "https://developer.xurrent.com/v1/services/",
		Methods:     []string{"GET", "POST", "PATCH"},
		Fields: []FieldDef{
			{Name: "id", Type: "integer", Description: "Unique ID", ReadOnly: true},
			{Name: "name", Type: "string", Description: "Service name (unique per account)", Required: true},
			{Name: "disabled", Type: "boolean", Description: "Whether the service is disabled", Default: "false"},
			{Name: "provider", Type: "reference", Description: "Provider organization"},
			{Name: "source", Type: "string", Description: "Source system identifier"},
			{Name: "sourceID", Type: "string", Description: "Source system record ID"},
		},
		DefaultFields:   []string{"id", "name", "disabled"},
		FilterableFields: []string{"id", "name", "source", "sourceID"},
		SortableFields:  []string{"id", "name"},
		States:          []string{"enabled", "disabled"},
		UniqueKey:       "name",
		Priority:        0,
		Relations: []EntityRelation{
			{Entity: "organization", Cardinality: "1:N", Via: "provider", Description: "Service provided by an organization"},
			{Entity: "service_offering", Cardinality: "1:N", Via: "service", Description: "Service has offerings"},
			{Entity: "service_instance", Cardinality: "1:N", Via: "service", Description: "Service has instances"},
			{Entity: "service_level_agreement", Cardinality: "N:M", Via: "service", Description: "Service covered by SLAs"},
		},
	},
	"service_instance": {
		Path:        "/v1/service_instances",
		Name:        "Service Instance",
		Plural:      "service_instances",
		Description: "A specific instance of a service, typically linked to a CI and supported by a team.",
		DocURL:      "https://developer.xurrent.com/v1/service_instances/",
		Methods:     []string{"GET", "POST", "PATCH"},
		Fields: []FieldDef{
			{Name: "id", Type: "integer", Description: "Unique ID", ReadOnly: true},
			{Name: "name", Type: "string", Description: "Service instance name (unique per account)", Required: true},
			{Name: "service", Type: "reference", Description: "Parent service", Required: true},
			{Name: "support_team", Type: "reference", Description: "Support team"},
			{Name: "status", Type: "string", Description: "Status: active, discontinued, etc."},
			{Name: "source", Type: "string", Description: "Source system identifier"},
			{Name: "sourceID", Type: "string", Description: "Source system record ID"},
		},
		DefaultFields:   []string{"id", "name", "status"},
		FilterableFields: []string{"id", "name", "source", "sourceID"},
		SortableFields:  []string{"id", "name"},
		UniqueKey:       "name",
		Priority:        0,
		Relations: []EntityRelation{
			{Entity: "service", Cardinality: "N:1", Via: "service", Description: "Instance belongs to a service"},
			{Entity: "team", Cardinality: "N:1", Via: "support_team", Description: "Supported by a team"},
			{Entity: "configuration_item", Cardinality: "1:N", Via: "cis", Description: "CIs linked to this instance"},
			{Entity: "service_level_agreement", Cardinality: "N:M", Via: "slas", Description: "SLAs covering this instance"},
		},
	},
	"service_offering": {
		Path:        "/v1/service_offerings",
		Name:        "Service Offering",
		Plural:      "service_offerings",
		Description: "Defines how a service is offered (e.g., Gold/Silver/Bronze tiers), with rates and SLAs.",
		DocURL:      "https://developer.xurrent.com/v1/service_offerings/",
		Methods:     []string{"GET", "POST", "PATCH"},
		Fields: []FieldDef{
			{Name: "id", Type: "integer", Description: "Unique ID", ReadOnly: true},
			{Name: "name", Type: "string", Description: "Offering name (unique per account)", Required: true},
			{Name: "service", Type: "reference", Description: "Parent service", Required: true},
			{Name: "source", Type: "string", Description: "Source system identifier"},
			{Name: "sourceID", Type: "string", Description: "Source system record ID"},
		},
		DefaultFields:   []string{"id", "name", "service"},
		FilterableFields: []string{"id", "name", "source", "sourceID"},
		SortableFields:  []string{"id", "name"},
		UniqueKey:       "name",
		Priority:        1,
		Relations: []EntityRelation{
			{Entity: "service", Cardinality: "N:1", Via: "service", Description: "Offering belongs to a service"},
			{Entity: "service_level_agreement", Cardinality: "1:N", Via: "slas", Description: "SLAs for this offering"},
		},
	},
	"person": {
		Path:        "/v1/people",
		Name:        "Person",
		Plural:      "people",
		Description: "A person (user or contact) in Xurrent. People can be requesters, approvers, team members, etc.",
		DocURL:      "https://developer.xurrent.com/v1/people/",
		Methods:     []string{"GET", "POST", "PATCH"},
		Fields: []FieldDef{
			{Name: "id", Type: "integer", Description: "Unique ID", ReadOnly: true},
			{Name: "name", Type: "string", Description: "Display name (unique per account)"},
			{Name: "primary_email", Type: "string", Description: "Primary email (unique identifier per account)", Required: true},
			{Name: "disabled", Type: "boolean", Description: "Whether the person is disabled", Default: "false"},
			{Name: "organization", Type: "reference", Description: "Organization person belongs to"},
			{Name: "site", Type: "reference", Description: "Site person is located at"},
			{Name: "source", Type: "string", Description: "Source system identifier"},
			{Name: "sourceID", Type: "string", Description: "Source system record ID (unique per account)"},
			{Name: "authenticationID", Type: "string", Description: "SSO authentication ID"},
		},
		DefaultFields:   []string{"id", "name", "primary_email", "disabled"},
		FilterableFields: []string{"id", "name", "primary_email", "source", "sourceID"},
		SortableFields:  []string{"id", "name", "primary_email"},
		States:          []string{"enabled", "disabled"},
		UniqueKey:       "primary_email (or source+sourceID)",
		Priority:        0,
		Relations: []EntityRelation{
			{Entity: "organization", Cardinality: "N:1", Via: "organization", Description: "Person belongs to an organization"},
			{Entity: "site", Cardinality: "N:1", Via: "site", Description: "Person is at a site"},
			{Entity: "team", Cardinality: "N:M", Via: "teams", Description: "Person is a member of teams"},
			{Entity: "configuration_item", Cardinality: "N:M", Via: "cis", Description: "Person covered for CIs"},
		},
	},
	"organization": {
		Path:        "/v1/organizations",
		Name:        "Organization",
		Plural:      "organizations",
		Description: "An organization (customer, provider, department) in Xurrent.",
		DocURL:      "https://developer.xurrent.com/v1/organizations/",
		Methods:     []string{"GET", "POST", "PATCH"},
		Fields: []FieldDef{
			{Name: "id", Type: "integer", Description: "Unique ID", ReadOnly: true},
			{Name: "name", Type: "string", Description: "Organization name (unique per account)", Required: true},
			{Name: "disabled", Type: "boolean", Description: "Whether the organization is disabled", Default: "false"},
			{Name: "financialID", Type: "string", Description: "Financial system identifier"},
			{Name: "source", Type: "string", Description: "Source system identifier"},
			{Name: "sourceID", Type: "string", Description: "Source system record ID"},
		},
		DefaultFields:   []string{"id", "name", "disabled"},
		FilterableFields: []string{"id", "name", "source", "sourceID", "financialID"},
		SortableFields:  []string{"id", "name"},
		States:          []string{"enabled", "disabled"},
		UniqueKey:       "name",
		Priority:        0,
		Relations: []EntityRelation{
			{Entity: "person", Cardinality: "1:N", Via: "organization", Description: "Organization has people"},
			{Entity: "service", Cardinality: "1:N", Via: "provider", Description: "Organization provides services"},
			{Entity: "service_level_agreement", Cardinality: "1:N", Via: "customer", Description: "Organization has SLAs as customer"},
		},
	},
	"configuration_item": {
		Path:        "/v1/cis",
		Name:        "Configuration Item",
		Plural:      "cis",
		Description: "A configuration item (CI) represents an asset, device, or component tracked in the CMDB.",
		DocURL:      "https://developer.xurrent.com/v1/configuration_items/",
		Methods:     []string{"GET", "POST", "PATCH"},
		Fields: []FieldDef{
			{Name: "id", Type: "integer", Description: "Unique ID", ReadOnly: true},
			{Name: "label", Type: "string", Description: "CI label/name (unique per account)"},
			{Name: "serial_nr", Type: "string", Description: "Serial number (unique per brand per account)"},
			{Name: "status", Type: "string", Description: "Status: active, archived, trash"},
			{Name: "product", Type: "reference", Description: "Product this CI is an instance of"},
			{Name: "site", Type: "reference", Description: "Site where CI is located"},
			{Name: "source", Type: "string", Description: "Source system identifier"},
			{Name: "sourceID", Type: "string", Description: "Source system record ID"},
		},
		DefaultFields:   []string{"id", "label", "serial_nr", "status"},
		FilterableFields: []string{"id", "label", "serial_nr", "source", "sourceID"},
		SortableFields:  []string{"id", "label"},
		States:          []string{"active", "archived", "trash"},
		UniqueKey:       "brand + serial_nr",
		Priority:        0,
		Relations: []EntityRelation{
			{Entity: "product", Cardinality: "N:1", Via: "product", Description: "CI is an instance of a product"},
			{Entity: "site", Cardinality: "N:1", Via: "site", Description: "CI is located at a site"},
			{Entity: "service_instance", Cardinality: "N:M", Via: "service_instances", Description: "CI linked to service instances"},
			{Entity: "request", Cardinality: "N:M", Via: "requests", Description: "CI referenced in requests"},
		},
	},
	"service_level_agreement": {
		Path:        "/v1/slas",
		Name:        "Service Level Agreement",
		Plural:      "slas",
		Description: "An SLA defines service commitments between a provider and customer, including coverage and response times.",
		DocURL:      "https://developer.xurrent.com/v1/service_level_agreements/",
		Methods:     []string{"GET", "POST", "PATCH"},
		Fields: []FieldDef{
			{Name: "id", Type: "integer", Description: "Unique ID", ReadOnly: true},
			{Name: "name", Type: "string", Description: "SLA name (unique per account)", Required: true},
			{Name: "customer", Type: "reference", Description: "Customer organization", Required: true},
			{Name: "service_offering", Type: "reference", Description: "Service offering"},
			{Name: "source", Type: "string", Description: "Source system identifier"},
			{Name: "sourceID", Type: "string", Description: "Source system record ID"},
		},
		DefaultFields:   []string{"id", "name", "customer"},
		FilterableFields: []string{"id", "name"},
		SortableFields:  []string{"id", "name"},
		States:          []string{"active", "inactive"},
		UniqueKey:       "name",
		Priority:        0,
		Relations: []EntityRelation{
			{Entity: "organization", Cardinality: "N:1", Via: "customer", Description: "SLA with a customer organization"},
			{Entity: "service_offering", Cardinality: "N:1", Via: "service_offering", Description: "SLA based on a service offering"},
			{Entity: "service_instance", Cardinality: "N:M", Via: "service_instances", Description: "SLA covers service instances"},
			{Entity: "coverage_group", Cardinality: "1:N", Via: "coverage_groups", Description: "SLA has coverage groups"},
		},
	},
	"request": {
		Path:        "/v1/requests",
		Name:        "Request",
		Plural:      "requests",
		Description: "A service request (ticket/incident/change). The core work item in Xurrent.",
		DocURL:      "https://developer.xurrent.com/v1/requests/",
		Methods:     []string{"GET", "POST", "PATCH", "PUT"},
		Fields: []FieldDef{
			{Name: "id", Type: "integer", Description: "Unique ID", ReadOnly: true},
			{Name: "subject", Type: "string", Description: "Request subject/title", Required: true},
			{Name: "category", Type: "string", Description: "Request category"},
			{Name: "status", Type: "string", Description: "Status (assigned, accepted, declined, completed, etc.)"},
			{Name: "impact", Type: "string", Description: "Impact level"},
			{Name: "urgency", Type: "string", Description: "Urgency level"},
			{Name: "requested_by", Type: "reference", Description: "Person who requested"},
			{Name: "requested_for", Type: "reference", Description: "Person this is for"},
			{Name: "service_instance", Type: "reference", Description: "Related service instance"},
			{Name: "template", Type: "reference", Description: "Request template used"},
			{Name: "source", Type: "string", Description: "Source system identifier (e.g. email, web, api)"},
			{Name: "sourceID", Type: "string", Description: "Source system record ID"},
			{Name: "member", Type: "reference", Description: "Team assigned to the request"},
			{Name: "created_at", Type: "datetime", Description: "When the request was created", ReadOnly: true},
			{Name: "updated_at", Type: "datetime", Description: "Last update time", ReadOnly: true},
			{Name: "completed_at", Type: "datetime", Description: "When the request was completed", ReadOnly: true},
			{Name: "completion_reason", Type: "string", Description: "Why the request was completed"},
			{Name: "note", Type: "text", Description: "Most recent note text (for filtering)"},
		},
		DefaultFields:   []string{"id", "subject", "status", "category", "created_at", "source"},
		FilterableFields: []string{"id", "subject", "status", "category", "source", "sourceID", "requested_by", "requested_for", "created_at", "updated_at", "completed_at"},
		SortableFields:  []string{"id", "subject", "created_at", "updated_at", "completed_at"},
		UniqueKey:       "",
		Priority:        0,
		Relations: []EntityRelation{
			{Entity: "person", Cardinality: "N:1", Via: "requested_by", Description: "Requested by a person"},
			{Entity: "person", Cardinality: "N:1", Via: "requested_for", Description: "Requested for a person"},
			{Entity: "service_instance", Cardinality: "N:1", Via: "service_instance", Description: "Related to a service instance"},
			{Entity: "request_template", Cardinality: "N:1", Via: "template", Description: "Created from a template"},
			{Entity: "workflow", Cardinality: "1:N", Via: "workflows", Description: "Request may have workflows"},
			{Entity: "request_note", Cardinality: "1:N", Via: "notes", Description: "Request has notes"},
			{Entity: "configuration_item", Cardinality: "N:M", Via: "cis", Description: "CIs linked to request"},
		},
	},
	"workflow": {
		Path:        "/v1/workflows",
		Name:        "Workflow",
		Plural:      "workflows",
		Description: "A workflow tracks a multi-step process (change, project, etc.) with phases and tasks.",
		DocURL:      "https://developer.xurrent.com/v1/workflows/",
		Methods:     []string{"GET", "POST", "PATCH"},
		Fields: []FieldDef{
			{Name: "id", Type: "integer", Description: "Unique ID", ReadOnly: true},
			{Name: "subject", Type: "string", Description: "Workflow subject", Required: true},
			{Name: "status", Type: "string", Description: "Status"},
			{Name: "template", Type: "reference", Description: "Workflow template"},
			{Name: "service", Type: "reference", Description: "Related service"},
			{Name: "source", Type: "string", Description: "Source system identifier"},
			{Name: "sourceID", Type: "string", Description: "Source system record ID"},
		},
		DefaultFields:   []string{"id", "subject", "status"},
		FilterableFields: []string{"id", "subject", "status", "source", "sourceID"},
		SortableFields:  []string{"id", "subject", "created_at"},
		UniqueKey:       "",
		Priority:        0,
		Relations: []EntityRelation{
			{Entity: "workflow_template", Cardinality: "N:1", Via: "template", Description: "Created from a template"},
			{Entity: "service", Cardinality: "N:1", Via: "service", Description: "Related to a service"},
			{Entity: "task", Cardinality: "1:N", Via: "tasks", Description: "Workflow contains tasks"},
			{Entity: "request", Cardinality: "N:1", Via: "requests", Description: "Workflow on a request"},
			{Entity: "problem", Cardinality: "N:1", Via: "problems", Description: "Workflow on a problem"},
		},
	},
	"ui_extension": {
		Path:        "/v1/ui_extensions",
		Name:        "UI Extension",
		Plural:      "ui_extensions",
		Description: "Custom HTML/CSS/JavaScript injected into Xurrent forms. Versions manage active/prepared states.",
		DocURL:      "https://developer.xurrent.com/v1/ui_extensions/",
		Methods:     []string{"GET", "POST", "PATCH"},
		Fields: []FieldDef{
			{Name: "id", Type: "integer", Description: "Unique ID", ReadOnly: true},
			{Name: "name", Type: "string", Description: "UI extension name", Required: true},
			{Name: "category", Type: "string", Description: "Record type this UI extension applies to", Required: true},
			{Name: "disabled", Type: "boolean", Description: "Whether disabled", Default: "false"},
			{Name: "css", Type: "text", Description: "CSS stylesheet (64KB max)"},
			{Name: "html", Type: "text", Description: "HTML code (64KB max)"},
			{Name: "javascript", Type: "text", Description: "JavaScript code (64KB max)"},
			{Name: "active_version", Type: "reference", Description: "Active version"},
			{Name: "prepared_version", Type: "reference", Description: "Prepared version"},
			{Name: "source", Type: "string", Description: "Source system identifier"},
			{Name: "sourceID", Type: "string", Description: "Source system record ID"},
		},
		DefaultFields:   []string{"id", "name", "category", "disabled"},
		FilterableFields: []string{"id", "name", "source", "sourceID"},
		SortableFields:  []string{"id", "name"},
		UniqueKey:       "name",
		Priority:        0,
		Relations: []EntityRelation{
			{Entity: "ui_extension_version", Cardinality: "1:N", Via: "versions", Description: "UI extension has versions"},
		},
	},
	"automation_rule": {
		Path:        "/v1/automation_rules",
		Name:        "Automation Rule",
		Plural:      "automation_rules",
		Description: "Automation rules define conditions that execute when records are created/updated. Read-only via API — create/update in Xurrent UI.",
		DocURL:      "https://developer.xurrent.com/v1/automation_rules/",
		Methods:     []string{"GET"},
		Fields: []FieldDef{
			{Name: "id", Type: "integer", Description: "Unique ID", ReadOnly: true},
			{Name: "name", Type: "string", Description: "Rule name", Required: true},
			{Name: "trigger", Type: "string", Description: "When the rule triggers", Required: true},
			{Name: "disabled", Type: "boolean", Description: "Whether disabled", Default: "false"},
			{Name: "condition", Type: "string", Description: "Condition expression"},
			{Name: "expressions", Type: "string", Description: "Named expression definitions"},
			{Name: "actions", Type: "string", Description: "Actions to execute when condition met"},
			{Name: "generic", Type: "string", Description: "Record type if not linked to specific record"},
			{Name: "position", Type: "integer", Description: "Execution order"},
			{Name: "source", Type: "string", Description: "Source system identifier"},
			{Name: "sourceID", Type: "string", Description: "Source system record ID"},
		},
		DefaultFields:   []string{"id", "name", "trigger", "disabled"},
		FilterableFields: []string{"id", "sourceID"},
		SortableFields:  []string{"id", "position"},
		UniqueKey:       "",
		Priority:        0,
		Relations: []EntityRelation{
			{Entity: "request", Cardinality: "N:1", Via: "request", Description: "Rule on a specific request"},
			{Entity: "workflow", Cardinality: "N:1", Via: "workflow", Description: "Rule on a specific workflow"},
			{Entity: "task", Cardinality: "N:1", Via: "task", Description: "Rule on a specific task"},
		},
	},
	"request_template": {
		Path:        "/v1/request_templates",
		Name:        "Request Template",
		Plural:      "request_templates",
		Description: "Predefined request templates that populate defaults and attach UI extensions.",
		DocURL:      "https://developer.xurrent.com/v1/request_templates/",
		Methods:     []string{"GET", "POST", "PATCH"},
		Priority:    1,
	},
	"workflow_template": {
		Path:        "/v1/workflow_templates",
		Name:        "Workflow Template",
		Plural:      "workflow_templates",
		Description: "Template for creating workflows with predefined phases, tasks, and automation rules.",
		DocURL:      "https://developer.xurrent.com/v1/workflow_templates/",
		Methods:     []string{"GET", "POST", "PATCH"},
		Priority:    1,
	},
	"task": {
		Path:        "/v1/tasks",
		Name:        "Task",
		Plural:      "tasks",
		Description: "A task within a workflow. Tasks have approvals, notes, and can be assigned to teams.",
		DocURL:      "https://developer.xurrent.com/v1/tasks/",
		Methods:     []string{"GET", "POST", "PATCH"},
		Priority:    1,
	},
	"task_template": {
		Path:        "/v1/task_templates",
		Name:        "Task Template",
		Plural:      "task_templates",
		Description: "Task templates define default task attributes within workflow templates.",
		DocURL:      "https://developer.xurrent.com/v1/task_templates/",
		Methods:     []string{"GET", "POST", "PATCH"},
		Priority:    2,
	},
	"problem": {
		Path:        "/v1/problems",
		Name:        "Problem",
		Plural:      "problems",
		Description: "A problem record for root-cause analysis, linked to requests, CIs, and workflows.",
		DocURL:      "https://developer.xurrent.com/v1/problems/",
		Methods:     []string{"GET", "POST", "PATCH"},
		Priority:    1,
	},
	"project": {
		Path:        "/v1/projects",
		Name:        "Project",
		Plural:      "projects",
		Description: "A project with phases, tasks, risks, and linked requests/workflows.",
		DocURL:      "https://developer.xurrent.com/v1/projects/",
		Methods:     []string{"GET", "POST", "PATCH"},
		Priority:    1,
	},
	"calendar": {
		Path: "/v1/calendars", Name: "Calendar", Plural: "calendars",
		Description: "Defines working hours and holidays for teams and SLAs.",
		DocURL:  "https://developer.xurrent.com/v1/calendars/",
		Methods: []string{"GET", "POST", "PATCH"},
		Priority: 1,
	},
	"contract": {
		Path: "", Name: "Contract", Plural: "contracts",
		Description: "Contracts linked to CIs (sub-resource only: audit, cis). Use xurrent_query on configuration_item to find linked contracts.",
		DocURL:  "https://developer.xurrent.com/v1/contracts/",
		Methods: []string{},
		Priority: 2,
	},
	"request_note": {
		Path: "/v1/requests/{request_id}/notes", Name: "Request Note", Plural: "request_notes",
		Description: "A note on a request. Use xurrent_create with entity=request_note and parent_id set to the request ID. Notes can be internal or public. The note text is in the 'text' field with optional 'internal' flag.",
		DocURL:  "https://developer.xurrent.com/v1/requests/notes/",
		Methods: []string{"GET", "POST"},
		Fields: []FieldDef{
			{Name: "id", Type: "integer", Description: "Note ID", ReadOnly: true},
			{Name: "text", Type: "text", Description: "Note content"},
			{Name: "internal", Type: "boolean", Description: "Internal note (not visible to requester)"},
			{Name: "created_at", Type: "datetime", Description: "When note was created", ReadOnly: true},
			{Name: "person", Type: "reference", Description: "Person who created the note", ReadOnly: true},
		},
		DefaultFields:   []string{"id", "text", "internal", "created_at"},
		FilterableFields: []string{"id", "text"},
		SortableFields:  []string{"id", "created_at"},
		Priority: 1,
	},
	"product": {
		Path: "/v1/products", Name: "Product", Plural: "products",
		Description: "A product definition; CIs are instances of products. Brand+productID must be unique.",
		DocURL:  "https://developer.xurrent.com/v1/products/",
		Methods: []string{"GET", "POST", "PATCH"},
		Priority: 1,
	},
	"product_category": {
		Path: "/v1/product_categories", Name: "Product Category", Plural: "product_categories",
		Description: "Hierarchical categories for products.",
		DocURL:  "https://developer.xurrent.com/v1/product_categories/",
		Methods: []string{"GET", "POST", "PATCH"},
		Priority: 2,
	},
	"configuration_item_relation": {
		Path: "", Name: "CI Relation", Plural: "ci_relations",
		Description: "Relationships between configuration items. Requires a parent CI ID — use xurrent_read on a CI to see its relations field.",
		DocURL:  "https://developer.xurrent.com/v1/configuration_items/ci_relations/",
		Methods: []string{},
		Priority: 1,
	},
	"account": {
		Path: "/v1/account", Name: "Account", Plural: "account",
		Description: "Account-level information including usage statements.",
		DocURL:  "https://developer.xurrent.com/v1/account/",
		Methods: []string{"GET"},
		Priority: 2,
	},
	"invoice": {
		Path: "", Name: "Invoice", Plural: "invoices",
		Description: "Invoice records (sub-resource only: audit, cis). Use xurrent_query on configuration_item to find linked invoices.",
		DocURL:  "https://developer.xurrent.com/v1/invoices/",
		Methods: []string{},
		Priority: 2,
	},
	"survey": {
		Path: "", Name: "Survey", Plural: "surveys",
		Description: "Customer satisfaction surveys (sub-resource only: questions). Access via survey response entities.",
		DocURL:  "https://developer.xurrent.com/v1/surveys/",
		Methods: []string{},
		Priority: 2,
	},
	"reservation": {
		Path: "/v1/reservations", Name: "Reservation", Plural: "reservations",
		Description: "Resource reservations (e.g., rooms, equipment).",
		DocURL:  "https://developer.xurrent.com/v1/reservations/",
		Methods: []string{"GET", "POST", "PATCH"},
		Priority: 2,
	},
	"first_line_support_agreement": {
		Path: "", Name: "First Line Support Agreement", Plural: "flsas",
		Description: "First line support agreements (sub-resource only: audit). Managed in Xurrent UI.",
		DocURL:  "https://developer.xurrent.com/v1/first_line_support_agreements/",
		Methods: []string{},
		Priority: 2,
	},
	"custom_collection": {
		Path: "/v1/custom_collections", Name: "Custom Collection", Plural: "custom_collections",
		Description: "Custom collections for organizing records.",
		DocURL:  "https://developer.xurrent.com/v1/custom_collections/",
		Methods: []string{"GET", "POST", "PATCH"},
		Priority: 2,
	},
	"inbound_email": {
		Path: "/v1/requests/{request_id}/inbound_emails", Name: "Inbound Email", Plural: "inbound_emails",
		Description: "Inbound emails received by Xurrent for a specific request. Requires parent_id (the request ID). Inspects sender (from), recipient (to), subject, and body for email triage.",
		DocURL:  "https://developer.xurrent.com/v1/requests/inbound_emails/",
		Methods: []string{"GET"},
		Fields: []FieldDef{
			{Name: "id", Type: "integer", Description: "Unique ID", ReadOnly: true},
			{Name: "from", Type: "string", Description: "Sender email address (e.g. corrections@trimble.com)"},
			{Name: "to", Type: "string", Description: "Recipient email address (e.g. account@mail.4me.com)"},
			{Name: "subject", Type: "string", Description: "Email subject line"},
			{Name: "created_at", Type: "datetime", Description: "When the email was received (ISO 8601)", ReadOnly: true},
			{Name: "message_id", Type: "string", Description: "RFC 5322 Message-ID header"},
			{Name: "body_start", Type: "text", Description: "Start of the email body (visible in collection)"},
			{Name: "source_uri", Type: "string", Description: "URL to download the full raw email source"},
			{Name: "note", Type: "reference", Description: "Note created from this email"},
		},
		DefaultFields:   []string{"id", "from", "to", "subject", "created_at"},
		FilterableFields: []string{"id", "from", "to", "subject"},
		SortableFields:  []string{"id", "created_at"},
		Priority: 1,
	},
	"event": {
		Path: "/v1/events", Name: "Event", Plural: "events",
		Description: "Request events (status changes, assignments, notes added). Use to trace the audit trail of a request — what happened and when.",
		DocURL:  "https://developer.xurrent.com/v1/requests/events/",
		Methods: []string{"GET"},
		Fields: []FieldDef{
			{Name: "id", Type: "integer", Description: "Unique ID", ReadOnly: true},
			{Name: "event", Type: "string", Description: "Event type (status_update, note_added, assigned, etc.)"},
			{Name: "created_at", Type: "datetime", Description: "When the event occurred"},
			{Name: "person", Type: "reference", Description: "Person who triggered the event"},
			{Name: "request", Type: "reference", Description: "Related request"},
		},
		DefaultFields:   []string{"id", "event", "created_at"},
		FilterableFields: []string{"id", "event"},
		SortableFields:  []string{"id", "created_at"},
		Priority: 1,
	},
	"audit_line": {
		Path: "/v1/audit_lines", Name: "Audit Line", Plural: "audit_lines",
		Description: "Audit trail entries showing who changed what and when. Each entry records an action (create/update/destroy/info) on a record, with the person who performed it and the changes made. Filter by entity type (audited field) or access per-entity audit at /v1/{entity}/{id}/audit.",
		DocURL:  "https://developer.xurrent.com/v1/audit_entries/",
		Methods: []string{"GET"},
		Fields: []FieldDef{
			{Name: "id", Type: "integer", Description: "Unique ID", ReadOnly: true},
			{Name: "action", Type: "string", Description: "Action: create, update, destroy, or info", ReadOnly: true},
			{Name: "audited", Type: "string", Description: "Entity type audited (e.g. request, problem, ci, workflow)", ReadOnly: true},
			{Name: "created_at", Type: "datetime", Description: "When the action occurred (ISO 8601)", ReadOnly: true},
			{Name: "created_by", Type: "reference", Description: "Person who performed the action", ReadOnly: true},
			{Name: "user", Type: "reference", Description: "Person whose session was used (may differ from created_by for impersonation)", ReadOnly: true},
			{Name: "changes", Type: "text", Description: "JSON describing what changed (only in single-record GET)", ReadOnly: true},
		},
		DefaultFields:   []string{"id", "action", "audited", "created_at"},
		FilterableFields: []string{"id", "created_at"},
		SortableFields:  []string{"id"},
		Priority: 1,
	},
	"request_audit": {
		Path: "/v1/requests/{request_id}/audit", Name: "Request Audit", Plural: "request_audits",
		Description: "Audit trail for a specific request. Shows all actions (create, update, automation triggers, notifications) on the request. Requires parent_id (the request ID).",
		DocURL:  "https://developer.xurrent.com/v1/requests/audit/",
		Methods: []string{"GET"},
		Fields: []FieldDef{
			{Name: "id", Type: "integer", Description: "Unique ID", ReadOnly: true},
			{Name: "action", Type: "string", Description: "Action: create, update, destroy, info", ReadOnly: true},
			{Name: "created_at", Type: "datetime", Description: "When the action occurred", ReadOnly: true},
			{Name: "created_by", Type: "reference", Description: "Person who performed the action", ReadOnly: true},
			{Name: "user", Type: "reference", Description: "Person whose session was used", ReadOnly: true},
			{Name: "changes", Type: "text", Description: "JSON describing what changed (single GET)", ReadOnly: true},
		},
		DefaultFields:   []string{"id", "action", "created_at"},
		FilterableFields: []string{"id"},
		SortableFields:  []string{"id"},
		Priority: 1,
	},
	"automation_rule_audit": {
		Path: "/v1/automation_rules/{automation_rule_id}/audit", Name: "Automation Rule Audit", Plural: "automation_rule_audits",
		Description: "Audit trail for a specific automation rule. Shows when the rule was created, updated, or disabled. Requires parent_id.",
		DocURL:  "https://developer.xurrent.com/v1/automation_rules/audit/",
		Methods: []string{"GET"},
		Fields: []FieldDef{
			{Name: "id", Type: "integer", Description: "Unique ID", ReadOnly: true},
			{Name: "action", Type: "string", Description: "Action: create, update, destroy, info", ReadOnly: true},
			{Name: "created_at", Type: "datetime", Description: "When the action occurred", ReadOnly: true},
			{Name: "created_by", Type: "reference", Description: "Person who performed the action", ReadOnly: true},
			{Name: "changes", Type: "text", Description: "JSON describing what changed", ReadOnly: true},
		},
		DefaultFields:   []string{"id", "action", "created_at"},
		FilterableFields: []string{"id"},
		SortableFields:  []string{"id"},
		Priority: 1,
	},
	"entity_audit": {
		Path: "/v1/{parent_type}/{parent_id}/audit", Name: "Entity Audit", Plural: "entity_audits",
		Description: "Audit trail for any entity type. Provide parent_type (e.g. people, teams, services, workflows) and parent_id. Covers all 25+ entity types that support audit: people, teams, services, service_instances, service_offerings, slas, sites, organizations, problems, tasks, workflows, calendars, products, product_categories, cis, custom_collections, projects, project_tasks, risks, surveys, ui_extensions, webhooks, workflow_templates, task_templates, request_templates, effort_classes, and more.",
		DocURL:  "https://developer.xurrent.com/v1/audit_entries/",
		Methods: []string{"GET"},
		Fields: []FieldDef{
			{Name: "id", Type: "integer", Description: "Unique ID", ReadOnly: true},
			{Name: "action", Type: "string", Description: "Action: create, update, destroy, info", ReadOnly: true},
			{Name: "created_at", Type: "datetime", Description: "When the action occurred", ReadOnly: true},
			{Name: "created_by", Type: "reference", Description: "Person who performed the action", ReadOnly: true},
			{Name: "user", Type: "reference", Description: "Person whose session was used", ReadOnly: true},
			{Name: "changes", Type: "text", Description: "JSON describing what changed (single GET)", ReadOnly: true},
		},
		DefaultFields:   []string{"id", "action", "created_at"},
		FilterableFields: []string{"id"},
		SortableFields:  []string{"id"},
		Priority: 1,
	},
	"webhook": {
		Path: "/v1/webhooks", Name: "Webhook", Plural: "webhooks",
		Description: "Webhook event subscriptions for real-time notifications.",
		DocURL:  "https://developer.xurrent.com/v1/webhooks/",
		Methods: []string{"GET"},
		Priority: 2,
	},
}

// SortedEntityNames returns entity names sorted by priority then alphabetically.
func SortedEntityNames() []string {
	var names []string
	for k := range EntityRegistry {
		names = append(names, k)
	}
	sort.Slice(names, func(i, j int) bool {
		a, b := EntityRegistry[names[i]], EntityRegistry[names[j]]
		if a.Priority != b.Priority {
			return a.Priority < b.Priority
		}
		return names[i] < names[j]
	})
	return names
}

// LookupEntity finds an entity by name (case-insensitive, accepting plurals and aliases).
func LookupEntity(name string) (EntityDef, bool) {
	n := strings.ToLower(strings.TrimSpace(name))
	// Also accept some alternate forms.
	aliases := map[string]string{
		"teams":                       "team",
		"sites":                       "site",
		"services":                    "service",
		"service_instances":           "service_instance",
		"serviceinstance":             "service_instance",
		"service instance":            "service_instance",
		"service_offerings":           "service_offering",
		"serviceoffering":             "service_offering",
		"service offering":            "service_offering",
		"people":                      "person",
		"organizations":               "organization",
		"org":                         "organization",
		"orgs":                        "organization",
		"configuration_items":         "configuration_item",
		"configuration_item":          "configuration_item",
		"configurationitems":          "configuration_item",
		"ci":                          "configuration_item",
		"cis":                         "configuration_item",
		"service_level_agreements":    "service_level_agreement",
		"servicelevelagreement":       "service_level_agreement",
		"sla":                         "service_level_agreement",
		"slas":                        "service_level_agreement",
		"requests":                    "request",
		"workflows":                   "workflow",
		"ui_extensions":               "ui_extension",
		"uiextensions":                "ui_extension",
		"ui extension":                "ui_extension",
		"automation_rules":            "automation_rule",
		"automationrules":             "automation_rule",
		"automationrule":              "automation_rule",
		"automation rule":             "automation_rule",
		"request_templates":           "request_template",
		"requesttemplate":             "request_template",
		"workflow_templates":          "workflow_template",
		"workflowtemplate":            "workflow_template",
		"tasks":                       "task",
		"task_templates":              "task_template",
		"tasktemplate":                "task_template",
		"problems":                    "problem",
		"projects":                    "project",
		"calendars":                   "calendar",
		"contracts":                   "contract",
		"notes":                       "request_note",
		"note":                        "request_note",
		"request_notes":               "request_note",
		"request_note":                "request_note",
		"products":                    "product",
		"product_categories":          "product_category",
		"productcategory":             "product_category",
		"configuration_item_relations": "configuration_item_relation",
		"ci_relations":                "configuration_item_relation",
		"cirelation":                  "configuration_item_relation",
		"invoices":                    "invoice",
		"surveys":                     "survey",
		"reservations":                "reservation",
		"first_line_support_agreements": "first_line_support_agreement",
		"flsas":                       "first_line_support_agreement",
		"custom_collections":          "custom_collection",
		"customcollection":            "custom_collection",
		"inbound_emails":              "inbound_email",
		"inboundemail":                "inbound_email",
		"inbound email":               "inbound_email",
		"mail":                        "inbound_email",
		"events":                      "event",
		"audit_lines":                 "audit_line",
		"audit_entries":               "audit_line",
		"auditentry":                  "audit_line",
		"audit line":                  "audit_line",
		"audit entry":                 "audit_line",
		"audit":                       "audit_line",
		"request_audits":              "request_audit",
		"request_audit":               "request_audit",
		"request audit":               "request_audit",
		"automation_rule_audits":      "automation_rule_audit",
		"automation_rule_audit":       "automation_rule_audit",
		"automation rule audit":       "automation_rule_audit",
		"entity_audits":               "entity_audit",
		"entityaudit":                 "entity_audit",
		"entity audit":                "entity_audit",
		"webhooks":                    "webhook",
	}

	if canonical, ok := aliases[n]; ok {
		n = canonical
	}
	def, ok := EntityRegistry[n]
	return def, ok
}

// EntitySummary returns a compact summary string for an entity.
func EntitySummary(name string) string {
	def, ok := EntityRegistry[name]
	if !ok {
		return fmt.Sprintf("Unknown entity: %s", name)
	}
	methods := strings.Join(def.Methods, "/")
	states := ""
	if len(def.States) > 0 {
		states = fmt.Sprintf(" states=%s", strings.Join(def.States, ","))
	}
	return fmt.Sprintf("%s (%s) — %s — methods=%s%s — %s", def.Name, def.Plural, def.Description, methods, states, def.DocURL)
}
