# Further investigation plan (test account + MCP design)

**Purpose:** validate **name** and **key** uniqueness per **entity type**, document **API error shapes**, and define safe **automation** behavior (draft → confirm, suffix rename, cleanup).

**Tenant observation (2026):** For our account, **`name` is unique per record type**: a removed/archived/disabled row **still reserves** that name for that type. **Different types** may reuse the same string (e.g. a **CI** and a **service instance** can both be named `foo`). Scope investigation around this rule; capture exceptions if any.

---

## 1. Safety and hygiene (mandatory for all API probes)

- **Suffix:** append a **random** tag to any user-visible `name` or identifier created during tests (e.g. `-mcp-a7f3`), so production-linked naming is never polluted.
- **Account:** use only the **dedicated test account** credentials from `xurrent-mcp/.env` (not committed); rotate token if leaked.
- **Cleanup:** every create must have a **teardown** path (delete, trash, or rename-back) documented in the run; if the API forbids delete, **rename** the test row to free the name and record the limitation.
- **Rate limits:** throttle calls; prefer `fields=` and tight filters; batch discovery with caching per [PROJECT_PLAN_AND_ROADMAP.md](PROJECT_PLAN_AND_ROADMAP.md).

---

## 2. Phase A — Name uniqueness matrix (empirical)

For each **entity type** the MCP will create or search by name, run a minimal script or manual sequence:

1. **Create** row with `name = "UNIQ-TEST-<random>"`.
2. **Archive / disable / trash** (whichever applies) per API.
3. **Attempt second create** with the **same** `name` and record HTTP status + JSON body.
4. **Optional:** create **another entity type** with the same **string** to confirm cross-type allowance (e.g. CI vs SI).

**Priority types:** `teams`, `services`, `sites`, `organizations`, `products` (name field), `service_instances`, `slas`, `request_templates`, `workflows` / templates as needed.

**Output:** table: `entity` | `name reserved after archive?` | `sample error` | `notes`.

**Automated probes:** integration tests `TestProbe_NameUniqueness_*` in [go-xurrent](https://github.com/xurrent/go-xurrent) (`test/integration/name_uniqueness_probe_test.go`) — teams, sites, cross-type, services, calendars, products, service offerings, SLAs, request templates, organizations, service instances, workflows (optional `XURRENT_TEST_WORKFLOW_TEMPLATE_ID`). Run from `go-xurrent` with `XURRENT_ALLOW_MUTATIONS=1` or use [scripts/run-name-probe.ps1](../scripts/run-name-probe.ps1). Results: [PROBE_RESULTS.md](PROBE_RESULTS.md).

---

## 3. Phase B — Error payload catalog

- **Duplicate `name` (team/site):** captured in [PROBE_RESULTS.md](PROBE_RESULTS.md) — `422` with `errors: [["name","Name has already been taken"]]`.
- **Composite probes (started):** `TestProbe_CompositeUniqueness_*` in `go-xurrent` (`name_uniqueness_probe_test.go`) — `primary_email`, `brand`+`productID`, CI `serial_nr` (`+` product). See **Phase B** table in [PROBE_RESULTS.md](PROBE_RESULTS.md); tenant may **SKIP** when uniqueness is not enforced or trusted-account blocks person create.
- **Duplicate `source`+`sourceID` (person):** `TestProbe_CompositeUniqueness_SourceSourceID` — same account guard as `primary_email`; see [PROBE_RESULTS.md](PROBE_RESULTS.md) Phase B.
- **Richer 422 samples:** log bodies from `t.Logf` when probes run on a home-account tenant (not SKIP).
- Map to MCP **user-facing** messages and **fuzzy** hints (per Datadog-style UX in the plan).

---

## 4. Phase C — List filters for “ghost” rows

For each entity, confirm how to **find** the blocking row:

- Predefined filters (`/teams/disabled`, etc.).
- CI: archived vs trashed vs active ([Configuration Items API](https://developer.xurrent.com/v1/configuration_items/)).
- Search API if applicable.

**Documented starting points:** [PROBE_RESULTS.md § Phase C](PROBE_RESULTS.md) (table of GET paths and filters).

**Implementation:** Go package **`github.com/xurrent/go-xurrent/pkg/collisionhints`** — `DetectIdentifierCollision` (ordered GET steps + `RenamePolicy`). See [PROBE_RESULTS.md — Implementation](PROBE_RESULTS.md).

---

## 5. Phase D — Rename-to-free workflow

- **Probes:** `TestProbe_NameUniqueness_*` already perform PATCH rename-to-free after duplicate **422** (see [PROBE_RESULTS.md § Phase D](PROBE_RESULTS.md)).
- **Policy:** when rename is inappropriate (regulated data, external sync on `sourceID`, production CIs) — documented in **PROBE_RESULTS Phase D** and in **`collisionhints.RenamePolicy`**.

---

## 6. Deliverables

- Updated [RESEARCH_UNIQUE_FIELDS_AND_ENTITIES.md](RESEARCH_UNIQUE_FIELDS_AND_ENTITIES.md) with Phase A results.
- Optional `docs/sandbox_runs/` (gitignored or redacted templates) for repeatable test recipes — only if the team wants checked-in procedures.

---

## 7. Open questions

- Does **case** normalization apply to name uniqueness (CI `label`/`serial` are documented as case-insensitive filters — names may differ)?
- **Multi-account** / directory: same name in directory vs support domain?
- **GraphQL** mutation errors vs REST parity?
