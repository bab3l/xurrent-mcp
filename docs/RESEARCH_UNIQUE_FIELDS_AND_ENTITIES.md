# Research spike: entities, uniqueness, and inactive records

**Date:** 2026-04-11  
**Sources:** [Xurrent REST API](https://developer.xurrent.com/v1/) (field descriptions), [Xurrent Help](https://help.xurrent.com/) (user-facing field guides), product blog where cited.

This note supports MCP **collision detection** and **draft → confirm** rename helpers. Scope is **per Xurrent account** unless stated otherwise.

---

## 1. Confirmed in official API field text (developer.xurrent.com)

These are **explicitly documented** as unique or as the account-scoped identifier:

| Entity / object | Field(s) | Rule (as documented) | Primary URL |
|-----------------|----------|----------------------|-------------|
| **Person** | `primary_email` | “This email address acts as the **unique identifier for the person within the Xurrent account**.” Also used as login when the person is a user. | [People API — Fields](https://developer.xurrent.com/v1/people/) |
| **Product** | `brand` + `productID` | “The concatenation of **Brand and Product ID** must be **unique within a Xurrent account**.” (`productID` is optional in API; when used with brand, pair must be unique.) | [Products API — Fields](https://developer.xurrent.com/v1/products/) |
| **Configuration item (CI)** | product **brand** (via product) + `serial_nr` | “The concatenation of **Product Brand and Serial Number** must be **unique within a Xurrent account**.” (`serial_nr` filter is case-insensitive.) | [Configuration Items API — Fields](https://developer.xurrent.com/v1/configuration_items/) |
| **Integration / external keys** | `source` + `sourceID` | “The combination of these two fields is used to **uniquely identify** records **in the correct source system**.” Applies to many record types that expose both fields. | [Source and Source ID](https://developer.xurrent.com/v1/general/source/) |
| **Person (SSO)** | `authenticationID` | May be used to “**uniquely identify** a user” when the IdP cannot supply primary email (see SSO docs). | [People API — Fields](https://developer.xurrent.com/v1/people/), [SSO / Authentication ID](https://developer.xurrent.com/v1/sso/) |

**Help center (not API prose):** [Configuration Item Fields](https://help.xurrent.com/help/configuration-item-fields/) states serial behavior in plain language: “A serial number must be **unique in Xurrent for each brand**… **concatenation of Brand and Serial Number** must be **unique within a Xurrent account**,” aligned with the API.

---

## 2. Strong semantics but weaker “uniqueness” wording in API

| Entity | Field | Notes | URL |
|--------|-------|--------|-----|
| **Organization** | `financialID` | “The **unique identifier** by which the organization is known **in the financial system**.” Implies one ID per org in finance; **not** spelled out as “unique across all org rows in Xurrent” in the API text — treat as **high-risk for integration collisions** and **verify** if create fails. | [Organizations API — Fields](https://developer.xurrent.com/v1/organizations/) |
| **SLA** | `agreementID`, `activityID_*` | Described as unique **in the billing system** of the service provider — integration semantics, not a general “SLA name unique” rule. | [Service Level Agreements API — Fields](https://developer.xurrent.com/v1/service_level_agreements/) |

---

## 3. `name` uniqueness — tenant validation (supplements official docs)

Official **API field** pages often **do not** spell out “`name` is unique,” but **operational behavior** on our test account matches:

- **`name` is unique per entity (record) type** within the account for the types we care about (service, team, site, organization, SLA, product, etc.).
- If a row is **removed, archived, trashed, or disabled**, the **`name` is still reserved** for that **same** entity type — a **new** row of that type **cannot** reuse the name until the old row is renamed or destroyed per product rules.
- **Cross-type reuse is allowed:** the **same string** can appear on **different** entity types (example: a **configuration item** and a **service instance** may both be named `foo`).

**MCP implication:** collision detection must be **scoped by entity type** (and by API resource name: `/services`, `/teams`, `/cis`, `/service_instances`, etc.). Do **not** assume one global “name registry” across all objects.

**Phase A automation:** see [PROBE_RESULTS.md](PROBE_RESULTS.md) — `TestProbe_NameUniqueness_*` covers teams, sites, cross-type, services, calendars, products, service offerings, SLAs, request templates, organizations, service instances, optional workflows. **Phase B (`TestProbe_CompositeUniqueness_*` in the same file):** `primary_email`, product `brand`+`productID`, CI `serial_nr`+product — see Phase B table in [PROBE_RESULTS.md](PROBE_RESULTS.md). Duplicate `name` / composite keys → often **422**; some probes **SKIP** when uniqueness is not enforced or trusted-account rules apply.

---

## 4. Inactive, disabled, archive, and trash — impact on “reserved” values

| Mechanism | API / product notes | Implication for collisions |
|-----------|---------------------|------------------------------|
| **`disabled: true`** | Many objects (e.g. [products `/disabled`](https://developer.xurrent.com/v1/products/), [services](https://developer.xurrent.com/v1/services/), [sites](https://developer.xurrent.com/v1/sites/), [teams](https://developer.xurrent.com/v1/teams/), [people](https://developer.xurrent.com/v1/people/), [organizations](https://developer.xurrent.com/v1/organizations/)) — “may no longer be related to other records” but row **still exists**. | **Likely** still occupies **unique** fields documented in §1 (e.g. same `primary_email` cannot be reused on another person until changed). |
| **CI archive / trash** | [POST `/cis/:id/archive`](https://developer.xurrent.com/v1/configuration_items/), [POST `/cis/:id/trash`](https://developer.xurrent.com/v1/configuration_items/), [POST `/cis/:id/restore`](https://developer.xurrent.com/v1/configuration_items/); blog: [Archive and Trash Configuration Items](https://www.xurrent.com/blog/archive-and-trash-configuration-items/). | CI **remains a record**; **brand + serial** uniqueness (§1) is expected to **still apply** — you cannot register a **second** live CI with the same brand+serial while the first remains in trash/archive without changing fields or restoring/deleting per product rules. **Confirm** with API error messages in a sandbox. |

**API lists:** use predefined filters such as `/products/disabled`, `/people/disabled`, `/organizations/disabled`, `/sites/disabled`, `/teams/disabled`, and CI-specific queries/filters for archived/trashed states as documented on [Configuration Items](https://developer.xurrent.com/v1/configuration_items/) to find **conflicting** rows **before** create.

---

## 5. Suggested MCP collision matrix (priority for tooling)

| Priority | Key | Detection strategy (read-only) |
|----------|-----|--------------------------------|
| **P0** | Person `primary_email` | `GET /people?primary_email=` (filter not case sensitive per API) + check `disabled` |
| **P0** | CI brand + `serial_nr` | Filter `serial_nr` + join product `brand` (or resolve product id); include archived/trash via CI state APIs |
| **P0** | Product `brand` + `productID` | List/search products with filters on `name`, `sourceID`, `disabled` as needed |
| **P1** | `source` + `sourceID` per entity | Entity-specific GET/list filters |
| **P2** | Org `financialID` | `GET /organizations` with `financialID` filter if supported; else search |
| **P1** | **`name`** per **entity type** (service, team, site, SLA, org, product, …) | List with filters for `disabled` / state; search; **tenant rule:** reserved after archive/trash/delete — see §3 |

---

## 6. Rename / suffix strategy (draft-only, user-approved)

1. **Prefer** official lifecycle: restore, merge, or change fields in UI/API per **Account Administrator** / role rules (e.g. [source/sourceID updates](https://developer.xurrent.com/v1/general/source/) require **Account Administrator** in many cases).  
2. **If** freeing a value: **PATCH** the **existing** inactive row to change `primary_email`, `serial_nr`, `productID`, `name`, or `sourceID` as applicable — **destructive** to audit history; must be **explicitly confirmed**.  
3. **Random suffix:** only after confirming **no** integration relies on the old value; document in MCP output for auditors.

---

## 7. Open questions (next research steps)

See [INVESTIGATION_PLAN.md](INVESTIGATION_PLAN.md) for phased procedures and **test hygiene** (random suffixes, cleanup).

1. **Exact HTTP validation errors** when duplicating each key (names + P0 fields) — capture JSON body for MCP error mapping.  
2. **Team** `inbound_email_local_part` — uniqueness across teams / mail routing collisions.  
3. **GraphQL** — parity with REST for uniqueness constraints on mutations.  
4. **Directory vs support domain** org/site lists — [changelog](https://developer.xurrent.com/v1/general/changelog/) notes future alignment; re-check for **duplicate** detection across linked accounts.  
5. **Case sensitivity** for `name` uniqueness (if any).

---

## 8. Related plan document

See [PROJECT_PLAN_AND_ROADMAP.md §8](PROJECT_PLAN_AND_ROADMAP.md) (Unique identifiers & collision helpers) for product requirements; this file is the **evidence-backed** research input.
