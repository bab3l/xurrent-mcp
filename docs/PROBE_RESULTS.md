# Name uniqueness probe results (Phase A / B)

**Run:** integration tests in `go-xurrent` (`TestProbe_NameUniqueness*`, `TestProbe_CompositeUniqueness*`). **Date:** 2026-04-11 (representative run on test tenant).

## Commands

From repo root with `XURRENT_TOKEN`, `XURRENT_ACCOUNT` set (e.g. load `xurrent-mcp/.env`):

```powershell
$env:XURRENT_ALLOW_MUTATIONS = "1"
$env:XURRENT_HTTP_MIN_INTERVAL = "0.35"   # optional, rate limits
cd C:\GitRoot\go-xurrent
go test -tags=integration ./test/integration/ -run 'TestProbe_(NameUniqueness|CompositeUniqueness)' -v -count=1
```

Or: `xurrent-mcp/scripts/run-name-probe.ps1` (loads env and runs the same).

Source: `go-xurrent/test/integration/name_uniqueness_probe_test.go`, helpers in `probe_helpers.go`. Collision discovery: `go-xurrent/pkg/collisionhints`.

## Outcomes

| Probe | Result |
|-------|--------|
| **Team** — duplicate `name` after `disabled: true` | **422** — `Name has already been taken` |
| **Site** — same | **422** |
| **Cross-type** — same string as team **and** site | **201** for both (allowed) |
| **Service** — duplicate after disable | **422** — use `provider_id` (enabled org); nested `provider` alone may not set `provider_id` |
| **Calendar** | **422** |
| **Product** (same name/brand/model) | **422** |
| **Service offering** | **422** — POST body must include **`service_id`** (top-level); nested `service: { id }` yields `service_id can't be blank` |
| **SLA** | **422** on duplicate `name` — create path: **two enabled orgs** (provider vs customer), service → offering, then `PostSlas` with **`customer_id`** + **`service_offering_id`** (top-level). Nested `customer` / `service_offering` alone led to **400 BadRequest** on this tenant. |
| **Request template** | **SKIP** if a second POST with the same `subject` + `service_id` returns **201** (this tenant does not enforce unique `subject`). |
| **Organization** | **422** — `PostOrganizations` uses the **`Organization`** model (`SetName`); success may be **200** or **201** depending on spec/client. |
| **Service instance** | **422** on duplicate `name` after `status: discontinued` — probe creates a **temporary team** (support team must be **enabled**), then service + SI. Body uses nested `service` + `support_team` (same pattern as `postServiceInstanceForCITest`). |
| **Workflow** | **SKIP** without `XURRENT_TEST_WORKFLOW_TEMPLATE_ID`. With template id: duplicate **`subject`** after **`PostWorkflowsIdTrash`** — **SKIP** if second POST returns **201** (no uniqueness on subject after trash); otherwise **422** logged. |

## Write-shape notes (JSON bodies)

- **POST /v1/services:** `provider_id` + enabled organization.
- **POST /v1/service_offerings:** `service_id` (integer).
- **POST /v1/request_templates:** `service_id` (integer).
- **POST /v1/slas:** Prefer `customer_id` + `service_offering_id` when the API rejects nested hashes.
- **POST /v1/organizations:** JSON body via generated `Organization` (e.g. `name` only for minimal create).
- **POST /v1/service_instances:** Nested `service` + `support_team` (not only `service_id`/`support_team_id` on all tenants); support team must be an **enabled** team — probe creates a dedicated team.
- **POST /v1/workflows:** Optional env `XURRENT_TEST_WORKFLOW_TYPE` (default `application_change`); `service_id` + `workflow_template` id.

## Error payload catalog (duplicate `name`)

HTTP **422 Unprocessable Entity**, typical body:

```json
{
  "message": "Validation Failed",
  "errors": [["name", "Name has already been taken"]]
}
```

**MCP mapping:** parse `errors` as `[field, message]` pairs; surface **"Name has already been taken"** and suggest finding disabled/archived row or renaming.

## Rename-to-free

After duplicate **422**, **PATCH** the first row’s `name` (or `subject` where applicable and unique), then **POST** again with the original value returns **201** — validates suffix/rename for freeing a label (where uniqueness is enforced).

## Phase D — rename policy, audit, and when *not* to rename

Empirical rename-to-free flows are already exercised by **`TestProbe_NameUniqueness_*`** (teams, sites, services, calendars, products, organizations, etc.): disable or soft-delete the first row, assert duplicate **422**, **PATCH** a suffix onto the unique field, then recreate with the original value (**201**).

**Generally OK:** disposable automation rows (random suffix in `name`), display names with no external sync.

**Avoid or escalate:** regulated personal data; **person** `source` / `sourceID` tied to HR or directory sync; production **CIs** under change control; any identifier referenced outside Xurrent — prefer formal archive/trash or data-owner approval before PATCH.

Structured guidance (same cautions) is exported as **`RenamePolicy`** from the Go package below.

## Phase B — composite keys (same test file; `TestProbe_CompositeUniqueness_*`)

| Probe | Result |
|-------|--------|
| **Person `primary_email`** | **SKIP** when `PostPeople` creates a person in a different account id than `XURRENT_ACCOUNT` (trusted-account / directory). Otherwise duplicate email while first person is **disabled** → **422** (logged). |
| **Product `brand` + `productID`** | **SKIP** if a second POST with the same `brand` + `productID` returns **201** while the first product is still **enabled** (no composite enforcement on this tenant). |
| **CI `serial_nr` (+ product)** | After **`archiveCreatedCI`** (PostCisIdArchive or Patch `status=archived`), **SKIP** if second `PostCis` with same `serial_nr` returns **201** (serial reusable after archive on this tenant). |
| **Person `source` + `sourceID`** | **SKIP** on trusted-account mismatch (same as `primary_email` probe). Otherwise duplicate pair while first person is **disabled** → **422** or **SKIP** if second `PostPeople` returns **201**. |

**MCP mapping:** parse `errors` arrays; field names may be `primary_email`, `productID`, `serial_nr`, `sourceID`, etc., not only `name`.

## Phase C — finding “ghost” rows (collision detection reads)

Use collection **GET**s + **state** / **fields** / **filters** (see [Filtering](https://developer.xurrent.com/v1/general/filtering/)) to locate the row that still holds a unique value.

| Entity | Starting points |
|--------|-----------------|
| **Team** | `GET /v1/teams` with `state=disabled` or predefined **`/v1/teams/disabled`** (per spec); `fields=id,name,disabled`. |
| **Site** | `GET /v1/sites` + `state=disabled` or **`/v1/sites/disabled`**. |
| **Service** | `GET /v1/services` + provider / `state` / `disabled` filters as documented. |
| **Person** | **`GET /v1/people/disabled`** / **`GET /v1/people/enabled`**: MCP tries **`name=`** / **`primary_email=`** when supported, else full scan. **`sourceID`**: field **`sourceid`** (value = sourceID) or **`source_sourceid`** with **`value_secondary`** (list rows lack `source` — confirm with **`GET /v1/people/{id}`**). Optional **`GET /v1/search`** for fragments (see **`go-xurrent` `APIClient.GetSearchJSON`** for **`page=`** continuation). |
| **Product** | **`GET /v1/products/disabled`**, **`/v1/products/enabled`**; MCP tries **`productID=`** when supported, else paginated scan; **`name=`** for name; **`brand_productid`** + **`value_secondary`** for brand+productID (composite scan emphasizes enabled products). |
| **Organization** | `GET /v1/organizations` + `state=disabled` / `enabled`. |
| **CI** | **`GET /v1/cis`**: MCP tries **`serial_nr=`** per state (**`active`**, **`archived`**, **`trash`**, and default) when supported, else full scan; **`GET /v1/cis/{id}`** after id discovery. |
| **SLA** | `GET /v1/slas` + inactive/active collections per spec. |

**MCP `detect_identifier_collision`:** narrow by **entity type + field** (e.g. `name` only for teams), then **list + filter** before suggesting a destructive rename.

### Implementation (`detect_identifier_collision`)

Use **`github.com/xurrent/go-xurrent/pkg/collisionhints`**: **`DetectIdentifierCollision(entity Kind, field string)`** returns ordered **GET** steps (`Step.Path`, `Step.QueryOrBody`) plus **`RenamePolicy`** (Phase D). Entity constants: `KindTeam`, `KindSite`, `KindService`, `KindPerson`, `KindProduct`, `KindOrganization`, `KindCI`, `KindSLA`. Pass the duplicate field from the **422** payload (e.g. `name`, `primary_email`, `serial_nr`, `productID`).

Example:

```go
plan, err := collisionhints.DetectIdentifierCollision(collisionhints.KindTeam, "name")
// plan.Steps — discovery; plan.RenamePolicy — when rename is risky
```

**CLI (no MCP):** from `go-xurrent`, `go run ./cmd/collision-hints -entity team -field name` prints the same plan as JSON (`-list` shows entity keywords).

**Stdio MCP server:** sibling repo **`xurrent-mcp`** (module `github.com/xurrent/xurrent-mcp`) builds **`cmd/xurrent-mcp`** — tools **`detect_identifier_collision`** (`entity`, `field`) and **`find_identifier_candidates`** (`entity`, `field`, `value`, optional **`value_secondary`** for composites — live API, needs `XURRENT_TOKEN` + `XURRENT_ACCOUNT`). Requires a local checkout with `replace github.com/xurrent/go-xurrent => ../go-xurrent` or a published `go-xurrent` version. Run: `go run ./cmd/xurrent-mcp` from `xurrent-mcp` (logs on stderr; JSON-RPC on stdin/stdout). **`scripts/run-xurrent-mcp.ps1`** loads `.env` before starting.

## Follow-ups

- **Request templates:** if `subject` uniqueness is required for MCP UX, enforce in the server layer or document tenant variance (probe skips when duplicates are accepted).
- **MCP `find_identifier_candidates`:** pagination uses **`Link: rel="next"`** → **`search_after`** on collection GETs (default **50** pages × 100 rows; override **`XURRENT_LIVEFIND_MAX_PAGES`**, max **500**). Parses quoted/unquoted `rel=next`, multiple `Link` headers, and relative URLs.
- **Implemented livefind filters (fall back to full scan on 400/422 where noted):** composite **`brand`+`productID`** / **`source`+`sourceID`**; single-field **`sourceid`** (sourceID only); single-field **`productid`** (**`productID=`** on **`/v1/products/disabled|enabled`**); **`serial_nr=`** on **`GET /v1/cis`** per state; **server-side `name=`** on **`GET /v1/teams`**, **`/sites`**, **`/services`**, **`/organizations`**, **`/slas`**; **`name=`** on **`GET /v1/products/disabled|enabled`** — all via **`go-xurrent` `APIClient.GetCollectionJSON`** (or typed **`Execute`**) when a filter is not modeled on the generated request type.
- **`GET /v1/search`:** continuation uses **`page=`** (per Search API docs), not **`search_after`**; use **`APIClient.GetSearchJSON`**, **`SearchNextPageToken`** / **`WithSearchPage`**, or **`ForEachSearchPage`** (**`go-xurrent`**, **`search_get.go`** / **`search_pagination.go`**).
