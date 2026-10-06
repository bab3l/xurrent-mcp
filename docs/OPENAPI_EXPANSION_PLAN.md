# go-xurrent OpenAPI Expansion Plan

## Overview

The current `api/openapi.yaml` covers ~130 paths but is missing ~38 endpoints that exist in the Xurrent REST API (confirmed returning 401 — real endpoints, just not documented in the spec). This plan adds the most valuable ones first.

## Verified Endpoints (401 = exists but needs auth)

```
/v1/email_templates             /v1/pdf_designs               /v1/broadcasts
/v1/closure_codes               /v1/effort_classes            /v1/holidays
/v1/knowledge_articles          /v1/releases                  /v1/rfc_types
/v1/risk_severities             /v1/short_urls               /v1/sla_coverage_groups
/v1/sla_notification_schemes    /v1/sprints                   /v1/survey_responses
/v1/time_allocations            /v1/timesheet_settings        /v1/time_entries
/v1/translations                /v1/trusted_organizations     /v1/waiting_for_customer_rules
/v1/webhook_policies            /v1/workflow_types            /v1/shop_articles
/v1/agile_boards                /v1/app_offerings             /v1/out_of_office_periods
/v1/product_backlogs            /v1/project_categories        /v1/project_risk_levels
/v1/reservation_offerings       /v1/service_categories        /v1/skill_pools
/v1/standard_service_requests   /v1/affected_slas             /v1/scrum_workspaces
/v1/cti                         /v1/mail_handlers
```

## Batch 1: Email Triage & Fault Finding (7 endpoints)

| Endpoint | Priority | Rationale |
|----------|----------|-----------|
| `/v1/email_templates` | Critical | Email triage — see what templates fire on notifications |
| `/v1/knowledge_articles` | Critical | KB lookup during investigations |
| `/v1/workflow_types` | High | Understand workflow categorization |
| `/v1/closure_codes` | High | Request closure reason analysis |
| `/v1/sla_notification_schemes` | High | SLA notification timing analysis |
| `/v1/holidays` | Medium | Calendar/coverage analysis |
| `/v1/effort_classes` | Medium | Time tracking configuration |

For each endpoint, the Xurrent API docs show:
- Email Templates: GET /v1/email_templates, GET /v1/email_templates/{id}
- Knowledge Articles: GET /v1/knowledge_articles, GET /v1/knowledge_articles/{id}
- Workflow Types: GET /v1/workflow_types, GET /v1/workflow_types/{id}
- Closure Codes: GET /v1/closure_codes, GET /v1/closure_codes/{id}
- SLA Notification Schemes: GET /v1/sla_notification_schemes, GET .../{id}, nested rules
- Holidays: GET /v1/holidays, GET /v1/holidays/{id}
- Effort Classes: GET /v1/effort_classes, GET /v1/effort_classes/{id}

## Implementation Pattern

For each new endpoint:

### Step 1: OpenAPI Spec Addition
Add path entry to `api/openapi.yaml` following the existing pattern:
```yaml
  /v1/email_templates:
    get:
      operationId: get_email_templates
      parameters:
        - $ref: '#/components/parameters/per_page'
        - $ref: '#/components/parameters/fields'
        - $ref: '#/components/parameters/sort'
      responses:
        '200':
          description: OK
          content:
            application/json:
              schema:
                type: array
                items:
                  type: object
  /v1/email_templates/{id}:
    get:
      operationId: get_email_templates_id
      parameters:
        - name: id
          in: path
          required: true
          schema:
            type: integer
        - $ref: '#/components/parameters/fields'
```

### Step 2: Generate Go Client File
Run the OpenAPI generator to create `api_email_templates.go` and model files.

### Step 3: Add Tests
Follow the existing test pattern from `test/api_teams_test.go` etc.

### Step 4: Add to xurrent-mcp Entity Registry
Add entity definitions with correct fields, paths, methods, and documentation URLs.

### Step 5: Endpoint Verification
Test against production API to confirm 200 responses with real data.

## Batch 2: Read-Only Reference Data (15 endpoints)
Lower priority — config/reference data that agents may need occasionally:
- rfc_types, risk_severities, project_categories, project_risk_levels
- service_categories, reservation_offerings, standard_service_requests
- short_urls, translations, trusted_organizations, workflow_types
- sla_coverage_groups, shop_articles, product_backlogs

## Batch 3: Admin Configuration (16 endpoints)
Lowest priority — pure admin config rarely needed by agents:
- broadcasts, pdf_designs, timesheet_settings, time_allocations
- time_entries, surveys, survey_responses, agile_boards
- app_offerings, scrum_workspaces, out_of_office_periods
- skill_pools, waiting_for_customer_rules, webhook_policies
- affected_slas, releases

## Testing Strategy

For each new endpoint:
1. **Unit**: Generated client compiles and type-checks
2. **Contract**: OpenAPI spec validates against Xurrent schema
3. **Live**: Test against `trimble-netops` production account with read-only token
4. **Entity**: xurrent-mcp entity integration test fetches real data

## Estimated Effort

- Batch 1 (7 endpoints): ~4 hours
- Batch 2 (15 endpoints): ~6 hours  
- Batch 3 (16 endpoints): ~6 hours
- Total: ~16 hours for full coverage
