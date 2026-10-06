package server

// Static content for MCP resources and prompts.
// These are extracted to a separate file to keep resources.go focused on registration.

func domainGraphJSON() string {
	return `{
  "description": "Xurrent entity relationship graph — key entities and their cardinality",
  "entities": {
    "organization": {
      "relations": [
        {"entity": "person", "cardinality": "1:N", "via": "organization_id"},
        {"entity": "service", "cardinality": "1:N", "via": "provider_id"},
        {"entity": "sla", "cardinality": "1:N", "via": "customer_id"}
      ]
    },
    "person": {
      "relations": [
        {"entity": "organization", "cardinality": "N:1", "via": "organization_id"},
        {"entity": "site", "cardinality": "N:1", "via": "site_id"},
        {"entity": "team", "cardinality": "N:M", "via": "members"},
        {"entity": "request", "cardinality": "1:N", "via": "requested_by / requested_for"}
      ]
    },
    "site": {
      "relations": [
        {"entity": "person", "cardinality": "1:N", "via": "site_id"},
        {"entity": "ci", "cardinality": "1:N", "via": "site_id"}
      ]
    },
    "team": {
      "relations": [
        {"entity": "person", "cardinality": "N:M", "via": "members"},
        {"entity": "service_instance", "cardinality": "N:M", "via": "support_team_id"},
        {"entity": "calendar", "cardinality": "1:N", "via": "team"}
      ]
    },
    "service": {
      "relations": [
        {"entity": "organization", "cardinality": "N:1", "via": "provider_id"},
        {"entity": "service_offering", "cardinality": "1:N", "via": "service_id"},
        {"entity": "service_instance", "cardinality": "1:N", "via": "service_id"}
      ]
    },
    "service_offering": {
      "relations": [
        {"entity": "service", "cardinality": "N:1", "via": "service_id"},
        {"entity": "sla", "cardinality": "1:N", "via": "service_offering_id"}
      ]
    },
    "service_instance": {
      "relations": [
        {"entity": "service", "cardinality": "N:1", "via": "service_id"},
        {"entity": "team", "cardinality": "N:1", "via": "support_team_id"},
        {"entity": "ci", "cardinality": "N:M", "via": "cis"},
        {"entity": "sla", "cardinality": "N:M", "via": "slas"}
      ]
    },
    "sla": {
      "relations": [
        {"entity": "organization", "cardinality": "N:1", "via": "customer_id"},
        {"entity": "service_offering", "cardinality": "N:1", "via": "service_offering_id"},
        {"entity": "service_instance", "cardinality": "N:M", "via": "service_instances"},
        {"entity": "coverage_group", "cardinality": "1:N", "via": "coverage_groups"}
      ]
    },
    "request": {
      "relations": [
        {"entity": "person", "cardinality": "N:1", "via": "requested_by / requested_for"},
        {"entity": "service_instance", "cardinality": "N:1", "via": "service_instance_id"},
        {"entity": "request_template", "cardinality": "N:1", "via": "template_id"},
        {"entity": "workflow", "cardinality": "1:N", "via": "request_id"},
        {"entity": "ci", "cardinality": "N:M", "via": "cis"},
        {"entity": "note", "cardinality": "1:N", "via": "notes"}
      ]
    },
    "workflow": {
      "relations": [
        {"entity": "workflow_template", "cardinality": "N:1", "via": "template_id"},
        {"entity": "task", "cardinality": "1:N", "via": "tasks"},
        {"entity": "request", "cardinality": "N:1", "via": "requests"},
        {"entity": "problem", "cardinality": "N:1", "via": "problems"}
      ]
    },
    "task": {
      "relations": [
        {"entity": "workflow", "cardinality": "N:1", "via": "workflow_id"},
        {"entity": "team", "cardinality": "N:1", "via": "assigned_team_id"},
        {"entity": "person", "cardinality": "N:1", "via": "assigned_to_id"}
      ]
    },
    "ci": {
      "relations": [
        {"entity": "product", "cardinality": "N:1", "via": "product_id"},
        {"entity": "site", "cardinality": "N:1", "via": "site_id"},
        {"entity": "service_instance", "cardinality": "N:M", "via": "service_instances"},
        {"entity": "request", "cardinality": "N:M", "via": "requests"}
      ]
    },
    "automation_rule": {
      "relations": [
        {"entity": "request", "cardinality": "N:1", "via": "request_id (optional)"},
        {"entity": "workflow", "cardinality": "N:1", "via": "workflow_id (optional)"},
        {"entity": "task", "cardinality": "N:1", "via": "task_id (optional)"}
      ]
    },
    "ui_extension": {
      "relations": [
        {"entity": "ui_extension_version", "cardinality": "1:N", "via": "versions"}
      ]
    }
  },
  "key_paths": {
    "customer_workflow": "organization → sla → service_instance → request → workflow → task",
    "cmdb": "product → ci ↔ service_instance → service → service_offering → sla → organization",
    "support": "team → service_instance ← ci / request → workflow → task → team"
  }
}`
}

func docsIndexJSON() string {
	return `{
  "topics": {
    "api_introduction": {"url": "https://developer.xurrent.com/v1/", "description": "REST API overview, auth, rate limiting, error codes"},
    "data_types": {"url": "https://developer.xurrent.com/v1/general/data_types/", "description": "JSON data type conventions"},
    "pagination": {"url": "https://developer.xurrent.com/v1/general/pagination/", "description": "per_page, search_after, search_before, Link headers"},
    "filtering": {"url": "https://developer.xurrent.com/v1/general/filtering/", "description": "Server-side filters, predefined filters, state"},
    "ordering": {"url": "https://developer.xurrent.com/v1/general/ordering/", "description": "sort parameter and field ordering"},
    "field_selection": {"url": "https://developer.xurrent.com/v1/general/field_selection/", "description": "fields= parameter for response trimming"},
    "source_sourceid": {"url": "https://developer.xurrent.com/v1/general/source/", "description": "Source and Source ID for integration keys"},
    "rate_limit": {"url": "https://developer.xurrent.com/v1/#rate-limiting", "description": "Rate limit headers, 429 handling, Retry-After"},
    "search": {"url": "https://developer.xurrent.com/v1/search/", "description": "Full-text search API"},
    "import": {"url": "https://developer.xurrent.com/v1/import/", "description": "Bulk import API"},
    "export": {"url": "https://developer.xurrent.com/v1/export/", "description": "Bulk export API"},
    "webhooks": {"url": "https://developer.xurrent.com/v1/webhooks/", "description": "Webhook event subscriptions"},
    "scim": {"url": "https://developer.xurrent.com/v1/scim/", "description": "SCIM provisioning API"},
    "sso": {"url": "https://developer.xurrent.com/v1/sso/", "description": "Single Sign-On configuration"},
    "oauth": {"url": "https://developer.xurrent.com/v1/oauth/", "description": "OAuth 2.0 authentication"},
    "ui_extensions": {"url": "https://developer.xurrent.com/v1/ui_extensions/", "description": "UI Extensions API HTML/CSS/JS"},
    "ui_extensions_js": {"url": "https://developer.xurrent.com/v1/ui_extensions/js_api/", "description": "JavaScript API for UI extensions"},
    "automation_rules": {"url": "https://developer.xurrent.com/v1/automation_rules/", "description": "Automation Rules API"},
    "text_formatting": {"url": "https://developer.xurrent.com/v1/general/text_formatting/", "description": "Text formatting for descriptions"},
    "changelog": {"url": "https://developer.xurrent.com/v1/general/changelog/", "description": "API changelog"}
  }
}`
}

func uiExtensionsGuide() string {
	return `# Xurrent UI Extensions Guide

## Overview
UI Extensions inject custom HTML, CSS, and JavaScript into Xurrent forms.
Each UI extension has an Active Version (live) and Prepared Version (draft).

## HTML Constraints
- Max 64KB
- Use Xurrent's standard row/column CSS classes for layout
- Inputs with class add-to-subject append their value to the request subject
- Inputs with class required are validated client-side
- Do NOT use <script> tags — JavaScript goes in the separate field
- Phrases wrapped in double braces {{phrase}} are auto-extracted for translation

## CSS Constraints
- Max 64KB
- Must use row-level scoping (e.g., .row.subject) to avoid breaking page layout
- Xurrent compiles CSS server-side; avoid @import
- Respect dark mode (check dark_mode_safe field)

## JavaScript Constraints
- Max 64KB
- Xurrent provides a restricted JS API (see developer.xurrent.com/v1/ui_extensions/js_api/)
- No DOM access outside the UI extension container
- Key APIs: xurrent.getFieldValue(), xurrent.setFieldValue(), xurrent.addNote()

## Version Management
- Edit html, css, javascript fields on the UI extension — this updates the Prepared Version
- Set activate=true on PATCH to promote Prepared to Active (archives previous)
- Cannot directly edit an Active Version

## Common Categories
request_template, problem, workflow_template, task_template, project,
project_task_template, service, service_instance, product, organization,
team, person, site, risk, contract, custom_collection,
knowledge_article_template, shop_article, app_offering, scim_user
`
}

func automationGuide() string {
	return `# Xurrent Automation Rules Guide

## Overview
Automation rules define conditions and actions that execute when records are
created, updated, or have notes added. Each rule has:

- Trigger — when the rule executes: on status update, on note added, on create
- Expressions — named definitions that simplify condition/action references
- Condition — boolean expression; rule actions execute when true
- Actions — what to do (update fields, add notes, change status, etc.)
- Generic — if set, applies to all records of that type

## Expressions Syntax
name: path.to.value
is_assigned: status = assigned
request: workflow.requests[first]
badge: request.custom_fields.badge

## Condition Syntax
Boolean: is_assigned and !badge
Operators: =, !=, and, or, ! (not)

## Actions Syntax
a1: update record set field = value
a2: add note 'Note text here'
a3: set status = canceled

## Generic Types
request, problem, workflow, task, risk, ci, scim_user, scim_group

## Predefined Filters
/automation_rules/for_requests, /automation_rules/for_workflows,
/automation_rules/for_tasks, /automation_rules/disabled,
/automation_rules/enabled, etc.
`
}

func itsmRelationshipsPrompt() string {
	return `You are helping work with Xurrent ITSM. Here's how key entities relate:

Service Delivery Chain:
- Organization (customer) → SLA → Service Offering → Service
- SLA covers specific Service Instances, backed by Configuration Items (CIs)
- Service Instances are supported by Teams

Request Flow:
- Person creates Request (optionally from Request Template)
- Request links to Service Instance and CIs
- Request can spawn Workflows with Tasks
- Tasks are assigned to Teams/People and may require Approvals

CMDB:
- Products define CI types
- CIs are instances of Products, located at Sites
- CIs relate to Service Instances, Requests, and people (as users)

Workflow Automation:
- Workflow Templates define phases and task templates
- Automation Rules can be attached to requests, workflows, tasks, or be generic

Unique Fields (name reservation persists after disable/archive):
- Team, Site, Service: name is unique per type
- Person: primary_email unique; source+sourceID unique
- CI: brand+serial_nr unique per account
- Product: brand+productID unique per account
- SLA, Organization, Service Instance: name unique per type

When creating records, always check for collisions on these unique fields.
Use xurrent_detect_collision for a read-only search plan.
Use xurrent_find_candidates for a live API search.
`
}

func uiExtensionDevPrompt() string {
	return `You are helping develop a Xurrent UI Extension. Key facts:

- UI Extensions inject HTML/CSS/JS into Xurrent forms
- Each has an Active Version (live) and Prepared Version (draft)
- Edit the Prepared Version, then activate it
- Max 64KB per field (html, css, javascript)
- JavaScript has a restricted API — see xurrent://docs/ui-extensions resource
- Use row-level CSS classes to avoid breaking page layout
- Add phrases as {{double-brace}} text for translation
- Use class="add-to-subject" on inputs to append to request subject
- Use class="required" for validated fields
- No <script> tags in HTML — JS goes in the javascript field
- dark_mode_safe: set true if your CSS handles dark mode

Before creating a UI extension:
1. Read xurrent://docs/ui-extensions resource for full API reference
2. Use xurrent_discover to see the ui_extension entity fields
3. Use xurrent_query entity=ui_extension to see existing extensions
4. Create with xurrent_create entity=ui_extension
5. Test with xurrent_read to verify
`
}

func automationDevPrompt() string {
	return `You are helping create a Xurrent Automation Rule. Key facts:

- Rules have: trigger, expressions, condition, actions
- Trigger: "on status update", "on note added", "on create"
- Expressions define named paths:  name: path.to.value
- Condition is a boolean:  is_assigned and !badge
- Actions execute sequentially:  a1: update record set field = value

Common patterns:
- Adding notes on status change
- Auto-assigning based on custom fields
- Canceling tasks when conditions change
- Cascading status updates

Before creating an automation rule:
1. Read xurrent://docs/automation resource for full syntax reference
2. Use xurrent_discover to see automation_rule entity
3. Use xurrent_validate_automation to check your syntax
4. Create with xurrent_create entity=automation_rule
5. Test by triggering the rule's condition

Generic rules (field: generic) apply to all records of a type.
Specific rules attach to one request/workflow/task.
Position controls execution order.
`
}

func emailTriagePrompt() string {
	return `You are investigating why inbound emails that should be blocked are creating tickets in Xurrent.

## Investigation Steps

### 1. Find recent requests created via email
Use xurrent_query to find requests with source=email, sorted by newest:
  xurrent_query entity=request filter=source=email fields=id,subject,status,created_at,source,note sort=-created_at per_page=20

### 2. Identify the automation rule that should be blocking
Search for automation rules that mention "close", "reject", "block", "suppress", or "notification":
  xurrent_query entity=automation_rule fields=id,name,trigger,disabled,condition,actions,generic

### 3. Inspect the blocking rule in detail
  xurrent_read entity=automation_rule id=<rule-id> 

Look at:
- condition — what must be true for the rule to fire
- actions — what the rule does (close, add note, suppress notification)
- trigger — "on create", "on status update", "on note added"
- generic — if set, applies to ALL records of that type

### 4. Check if the rule is disabled
Disabled rules don't fire. Check the 'disabled' field on the automation rule.

### 5. Check the rule's expressions
Named expressions may reference custom fields, from-addresses, or subject patterns.
If the remote system changed its "from" address or subject line format, expressions may no longer match.

### 6. Inspect a sample ticket that should have been blocked
  xurrent_read entity=request id=<sample-id> fields=id,subject,source,sourceID,created_at,status,note

Look at:
- The subject line — did it change format?
- sourceID — this may contain the inbound email ID
- Notes — do they show the rule firing?

### 7. Check inbound emails
  xurrent_query entity=inbound_email fields=id,from,to,subject,created_at per_page=20 sort=-created_at

Compare the "from" address against what the automation rule expects.

### 8. Check request events for the sample ticket
Look at the event timeline to see what actually happened:
  The events API at /v1/events shows status changes, assignments, and note additions.
  xurrent_query entity=event fields=id,event,created_at per_page=20 sort=-created_at

### 9. Check audit entries for the automation rule
  xurrent_query entity=audit_entry filter=auditable_type=automation_rule fields=id,auditable_id,action,created_at sort=-created_at

This shows if anyone changed the rule recently.

## Common Causes

1. **Rule disabled** — someone (or a software update) disabled the rule
2. **From address changed** — remote system now sends from a different address
3. **Subject format changed** — no longer matches the rule's condition
4. **Condition too narrow** — rule only matches specific statuses or categories
5. **Position/order changed** — another rule fires first and changes state
6. **Generic field mismatch** — rule set to fire on wrong record type
7. **Xurrent update** — check audit entries for system-level changes around the time behavior changed

## Follow-up Actions

Once root cause is identified:
- If rule is disabled: re-enable it (xurrent_update entity=automation_rule id=<id>)
- If from-address changed: update the rule's expressions
- If condition doesn't match: update the condition
- Consider adding a note to affected tickets explaining the investigation
`
}
