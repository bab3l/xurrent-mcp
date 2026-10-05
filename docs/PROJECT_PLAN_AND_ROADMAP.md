# xurrent-mcp — project plan and roadmap



**Status:** active — **`cmd/xurrent-mcp`** (stdio MCP) and **`internal/livefind`** ship read-only collision tools; see **`PROBE_RESULTS.md`**.

**Related:** [go-xurrent](https://github.com/xurrent/go-xurrent) (OpenAPI-derived REST client; **`docs/ROADMAP.md`** includes REST vs optional **GraphQL/BFF** strategy), [Xurrent developer docs](https://developer.xurrent.com/).



---



## 1. Goals



- Ship a **Docker-friendly** MCP server for [Xurrent](https://xurrent.com/) (ITSM / service management) that integrates safely with **very low REST rate limits** (see [Rate limit](https://developer.xurrent.com/v1/rate_limit/) — core context includes a small short window, e.g. 20 requests, plus a larger window; `429` + `retry-after` behavior).

- Expose **curated capabilities** (not a raw dump of every REST endpoint) with **draft → review → confirm** flows for mutations, matching Xurrent’s auditability and avoiding “vibed” production changes.

- Provide **product and domain context** (what Xurrent is, how entities relate — **services, service offerings, service instances, SLAs, CIs, requests, workflows,** etc.) in a **token-efficient** way, with **links to live developer documentation** so the model can **retrieve** detail only when needed.

- Add a **research track** for **UI extensions and automation** (CSS, Xurrent’s JavaScript subset, automation/rule syntax), grounded in official docs and community sources (e.g. forums), with honest gaps where the platform is under-specified.



---



## 2. Lessons from “non-default” MCP patterns (Medium, Mar 2026)



The article *“Datadog, Block, and Cloudflare All Abandoned the Default MCP Pattern…”* benchmarks **three** implementations against the **same** upstream API (HackerNews):  



| Version | Shape | What it is |

|--------|--------|------------|

| **v1 — Traditional** | ~10 tools, **one per endpoint** | What most tutorials prescribe; works in isolation but scales poorly. |

| **v2 — Code mode** | **1 tool** (e.g. `execute_code`) | The model writes **Python** against a **predefined API object**; chaining, filtering, and computation run **server-side** (article ties this to **Cloudflare**). Collapses multi-hop fan-out to **one** agent step (fewer **LLM round-trips**), but **simple** queries can pay a **token penalty** vs structured tools because the code string is bulky. Requires a **sandbox** and hardening. |

| **v3 — Layered + query** | **3 tools**: `discover()`, **`query()`** (field selection, filters, sort, aggregation, **format**), **`compare()`** | **Progressive disclosure** (article: **Block** — discovery → planning → execution; hundreds of endpoints conceptually compressed into a small surface), plus **context-efficient** returns (article: **Datadog** — trim fields, prefer lean formats). May still need **multiple** upstream HTTP calls for fan-out, but each response is **trimmed** (e.g. explicit `fields=…`) so total tokens and latency stay low. **`compare()`-style** aggregation can beat asking the model to average raw rows (**~98%** token reduction in the article’s aggregation scenario). |



**Core thesis (quote in spirit):** treat the MCP server as an **agent-optimized interface** whose upstream is just a **data source**. The REST/GraphQL API can stay unchanged; the server **re-serializes** (trim JSON, optional **CSV/tabular** default, caps) so the agent is not forced to ingest full blobs or do mental math over raw lists.



**Industry techniques called out:**



- **Datadog-style:** default **lean** outputs (**CSV / trimmed fields**), **fuzzy** “did you mean …?” errors, hints like **“Showing 5 of 500”** to cut **retry** loops (not just token count).

- **Block-style:** **layered** tools so the model discovers what exists before planning execution — avoids stuffing the system prompt with every capability.

- **Cloudflare-style:** **code mode** for arbitrary multi-hop logic when parameters alone are awkward — best as an **opt-in** (article: **~90%** structured tools, **~10%** code escape hatch).



**Implications for Xurrent (rate limits *and* agent latency):**



- **Two costs:** (1) **Xurrent HTTP** quota — each outbound call counts. (2) **MCP / LLM** latency — the article stresses **~0.5–3 s per tool call** in real chats, so **11 sequential tools** add tens of seconds *before* counting API time. Designs that reduce **agent-visible tool invocations** or **trim each response** help both **tokens** and **clock time**.

- Prefer **v3-style** primitives mapped to Xurrent: e.g. **`discover`** (catalog of safe read templates / entity types), **`query`** (wrap REST `fields=` + filters + pagination; **GraphQL** only if you operate a separate BFF — see **`go-xurrent/docs/ROADMAP.md`** *GraphQL and other query interfaces*), **`compare` / aggregate** for common ITSM questions — implemented **inside** the server with **throttling** and **Retry-After** on the wire.

- **Code mode (v2)** only where query cannot be expressed safely as parameters — **read-only** sandbox, **no** shortcut around **draft → confirm** for writes; heavy security review.

- **80/20 checklist** from the article applied here: (1) **measure** response-token footprint vs what the agent needed; (2) add **one** high-value **aggregation** tool early; (3) observe **real** multi-step sessions, not only per-tool unit tests.



**API efficiency (operational, not only tokens):**



- Centralize an **outbound HTTP scheduler**: global min interval, **respect `Retry-After`**, exponential backoff on `429`, and optional jitter.

- Prefer **GraphQL** or **batched** read strategies for paths that would otherwise cause **N+1** REST GETs; keep mutations on REST (or GraphQL mutations only if documented and aligned with audit flows).

- Use **GET `/rate_limit`** sparingly (documented as **once per second** max); derive remaining budget from response headers on normal calls when possible.

- **Cache** (see §8) and **aggregate** server-side so repeated agent turns do not repeat full API fan-out.



---



## 3. Safety, auditing, and MCP consumer signaling



- **Default stance:** all create/update/delete operations go through **draft → user confirmation → execute**, with explicit summaries (subject, team, template, workflow type, etc.).

- **MCP tool annotations:** use the protocol’s hints so clients can show warnings or extra friction (e.g. **destructive** / **non-idempotent** / **writes** vs read-only). Map “risky” tools (production mutations, bulk operations, automation changes, **collision-fixing PATCHes**) to the strongest appropriate hints.

- **Idempotency:** where the API supports idempotent keys or safe replays, document and use them to reduce duplicate creates on retries.



---



## 4. Language, runtime, and Go MCP SDK



| Option | Pros | Cons |

|--------|------|------|

| **Go** using **go-xurrent** | Single static binary, low RAM vs typical Node/Python servers, excellent fit for Docker; client already tracks Xurrent conventions (`collection_query`, pagination, `fields`) | GraphQL client is extra |

| **TypeScript (Node)** | Rich MCP ecosystem | Higher baseline memory; more care needed for Docker resource limits |

| **Rust** | Extreme efficiency | Higher build complexity; no in-repo client today |



**Recommendation:** **Go** for the production server image.



**MCP SDK (investigated):** the **official** Go implementation is **[github.com/modelcontextprotocol/go-sdk](https://github.com/modelcontextprotocol/go-sdk)** (`import …/mcp`), maintained with **Google**, Apache-2.0/MIT-style licensing, broad **MCP spec version** support (see repo compatibility table), **stdio** and **streamable HTTP** transports, docs on **security** topics (e.g. SSRF, confused deputy). It is the **default choice** for a compiled, enterprise-oriented server unless requirements force another stack. Pin a **released** version and track upstream for spec/security fixes.



Use **openapi.yaml from go-xurrent** only to generate **helpers** — not to auto-expose hundreds of tools.



---



## 5. Domain model, relationships, and documentation (low token footprint)



**Problem:** agents need to understand how **Configuration Items**, **Services**, **Service offerings**, **Service instances**, **SLAs**, **Requests**, **Workflows**, **Problems**, **People**, **Organizations**, **Sites**, etc. relate — but loading long prose into the **system prompt** or default tool descriptions burns **context** that should stay available for **tool results**, **user messages**, and reasoning.



**Approach:**



1. **Thin static baseline:** one **compact** MCP **resource** (or tiny set), e.g. `xurrent://domain/graph` — a **structured outline** (bullet or table) of **entity types** and **key relationships** (cardinality and direction only), **under 1–2 k tokens** if possible. No long prose; optional **one-line** notes per edge.

2. **Live documentation URIs:** the same resource (or `xurrent://docs/index`) lists **stable URLs** to [developer.xurrent.com](https://developer.xurrent.com/) sections — REST topics, GraphQL, [general filtering](https://developer.xurrent.com/v1/general/filtering/), [source / sourceID](https://developer.xurrent.com/v1/general/source/), rate limiting, etc. The agent is instructed to use **`fetch`/browser tools** (when available) or **`resource` read** for detail, not to memorize pages.

3. **Curated summaries:** short excerpts **only** where they prevent systematic mistakes (e.g. “service instance vs service offering,” “SLA attaches to …”). Each excerpt should cite the **canonical URL**. Regenerate from docs in CI when APIs change (versioned changelog).

4. **Discover tool:** extends the graph with **what’s queryable** (filters, `fields` presets) without pasting full OpenAPI into the prompt.

5. **Optional:** ship a **separate** “extended domain pack” resource for advanced users (disabled by default) to keep default enrollment minimal.



**Relationship research task:** derive the graph from **official** REST/GraphQL field references and help center articles; validate against a **non-production** tenant.



---



## 6. Response shaping, compression hooks, and token budget for tool replies



**Goals:** maximize room for **user content** and **model reasoning**; treat tool outputs as **expensive**.



- **Hooks (internal API):** a **single** response pipeline for all tools: normalize JSON (**no redundant whitespace**), optional **stable key ordering**, **truncate** with explicit `… truncated` and **counts**, optional **CSV** for tabular data.

- **Whitespace:** strip decorative indentation in machine responses; keep human-readable **only** in draft summaries intended for user review.

- **Size ceiling:** configurable **max characters** per tool result; split or summarize beyond the cap.

- **Telemetry:** optional metrics on **output size** per tool name (redacted) for tuning.

- **Compression:** for very large payloads, prefer **fewer fields** and **pagination** over generic gzip of text inside MCP (clients may not benefit); if HTTP transport adds binary channels later, revisit.



---



## 7. Caching outside the hot path, staleness, and aggregation



**Motivation:** preserve **rate limits**; avoid re-fetching unchanged catalog and metadata on every agent turn.



- **Process-external cache (recommended for production):** pluggable backend (e.g. **Redis**, **filesystem**, or **embedded** DB) keyed by **account id** + **resource type** + **query fingerprint**, with **TTL** and **ETag**/last-modified style invalidation where the API exposes `updated_at` or list ordering.

- **Modes:** `fresh` (bypass cache), `auto` (use cache, refresh if stale), `cache_only` (offline / dry-run — return 409 or explicit error if missing).

- **What to cache:** read-only, non-secret metadata: service catalog slices, enum definitions, team lists, template ids — **never** tokens, full PII blobs, or unredacted audit payloads.

- **Invalidation:** user-triggered `invalidate_cache` tool (gated); TTL backoff on **429**.

- **Aggregation:** implement **compare/summary** tools that **read from cache + one** refresh when needed, instead of N uncached GETs.



---



## 8. Unique identifiers, collisions, and “revived” names



**Validated product behavior (test account):** **`name`** is effectively **unique per entity type** (e.g. `/services`, `/teams`, `/cis`, `/service_instances`): a **removed, archived, trashed, or disabled** row **still reserves** that name for that **type**. **Different types** may reuse the same string (e.g. a CI and a service instance both named `foo`). Other keys remain as in [RESEARCH_UNIQUE_FIELDS_AND_ENTITIES.md](RESEARCH_UNIQUE_FIELDS_AND_ENTITIES.md). Further API error capture: [INVESTIGATION_PLAN.md](INVESTIGATION_PLAN.md).



**Plan:**



1. **Documentation:** add explicit **callouts** in MCP **instructions** and in **`draft_*` tool** descriptions: which operations are sensitive (e.g. **name**, **`sourceID`**, **`financialID`**, SKU-style fields — **confirm per entity** against [API field docs](https://developer.xurrent.com/v1/) and help center).

2. **Research:** evidence-backed matrix in [RESEARCH_UNIQUE_FIELDS_AND_ENTITIES.md](RESEARCH_UNIQUE_FIELDS_AND_ENTITIES.md) (API + help citations, P0/P1 collision priorities, gaps). Ongoing: sandbox validation of error payloads and “name-only” collisions. Canonical integration topic: [Source and Source ID](https://developer.xurrent.com/v1/general/source/).

3. **Helpers (draft + confirm):**

   - **`detect_identifier_collision` (read):** given entity type + proposed key fields, search **including** inactive/archived/trashed where the API allows filters — **rate-aware**.

   - **`propose_identifier_resolution`:** if collision with **inactive** record, propose **PATCH** to append a **deterministic or random suffix** to the **old** record’s field (e.g. `name` / `sourceID`) to free the identifier — output a **draft diff** only.

   - **`commit`** applies only after explicit user approval; **destructive** annotations on any tool that mutates existing rows to free a name.

4. **Safety:** suffix strategy must be **auditable** (log **what** changed, **why**); avoid silent renames in **production** without confirmation.



---



## 9. Architecture outline



1. **Transport:** stdio for local/agents; optional **streamable HTTP** for container deployments (align with **go-sdk** and target MCP clients).

2. **API layer:**

   - **REST** via go-xurrent for mutations and endpoints without good GraphQL coverage.

   - **GraphQL** for aggregated reads (single round-trip, explicit field sets).

3. **Tool surface (illustrative, not final):** align with **layered + query** (article v3), not endpoint-per-tool (v1).

   - **Discovery / read:** **`discover`** + **`query`** with mandatory **fields**, **limits**, **format**; optional **`compare` / aggregate**.

   - **Draft workflows:** `draft_request`, `draft_workflow` → **human-readable plan + machine payload**.

   - **Commit:** `commit_draft` (draft id / checksum).

   - **Meta:** `rate_limit_status` (throttled); **`invalidate_cache`** (optional, gated).

   - **Identifiers:** collision detection + **propose resolution** (§8).

   - **Optional (later):** single **sandboxed** **read-only** **code** tool for awkward fan-out (article v2).

4. **Resources:** **compact** `domain/graph`, **`docs/index`** URLs, optional extended pack (§5).

5. **Prompts:** short task-specific prompts; avoid duplicating resources.



---



## 10. UI, automation, and customization (research phase)



**Objective:** tools and resources that help admins and developers work with **UI extensions** and **automation**, without pretending the platform is fully formalized.



- **Official:** [Xurrent GraphQL](https://developer.xurrent.com/graphql/), [REST](https://developer.xurrent.com/v1/), UI extension and automation rule endpoints in OpenAPI (see go-xurrent).

- **Community:** Xurrent forums / discussions for real-world patterns and limitations.

- **Deliverables:** a living **syntax and constraint guide** (CSS subset, JS restrictions, automation patterns), **examples** with warnings where behavior is empirical; optional **lint-style checks** (best-effort) for UI extension assets.



---



## 11. Docker-oriented release (plan)



- Multi-stage **Dockerfile**: compile static or mostly static binary; **non-root** user; **read-only** root filesystem where possible; configurable **memory** limits documented.

- **Health:** HTTP transport exposes `/healthz`; stdio-only images document process supervision.

- **Secrets:** env or mounted files; never bake tokens.

- **Sidecar / external cache:** document **Redis** (or equivalent) for multi-replica deployments.



---



## 12. Competitive landscape (ITSM / ITOM MCP ideas)



Useful patterns seen in broad ITSM MCP servers (e.g. ServiceNow-oriented servers and docs): **persona- or package-based tool grouping**, **read vs write** separation, **schema/discovery** helpers, **incident/problem/change** workflows, **CMDB** queries, **attachment** handling, **knowledge/search**, and **multi-environment** config. For Xurrent, map these to: requests, problems, workflows, tasks, CIs/services/offerings, SLAs, knowledge-related requests, and **templates**—always with **draft-first** mutations and rate-aware reads.



---



## 13. Roadmap (phased)



### Phase 0 — Foundation



- Go module; depend on **modelcontextprotocol/go-sdk** (pinned release) and **go-xurrent**.

- MCP server skeleton (**stdio** + config for HTTP if needed).

- HTTP middleware: **rate limit**, **retry-after**, **min interval**; optional **external cache** interface (in-memory stub first).

- **Single** compact **resource:** `docs/index` (URLs only) + **minimal** `domain/graph` stub.

- **Response pipeline hook:** JSON compact + whitespace strip + max-size guard.



### Phase 1 — Read path MVP



- **`discover`** + **`query`** with mandatory **fields**, caps, **lean** defaults; wire **cache** (`auto`/`fresh`) for list/metadata endpoints.

- **One** **aggregation/compare** tool for a high-traffic question.

- Optional **GraphQL** for one-shot graphs.

- **Actionable errors** (unknown field → suggestions) where feasible.

- **Telemetry:** tool-call counts, response size histograms (redacted).



### Phase 2 — Domain knowledge & doc ergonomics



- Flesh out **`domain/graph`** from official docs (relationships CI ↔ service ↔ SLA ↔ request ↔ workflow); keep **under agreed token budget**; add **changelog** process when Xurrent updates APIs.

- **Optional** extended resource pack (off by default).



### Phase 3 — Draft-first create flows



- **Draft/commit** for requests and workflows; strong MCP annotations on **commit**.

- **Identifier collision** **detection** + **propose resolution** (draft rename of inactive records); **never** silent rename.



### Phase 4 — Customization assistant (incremental)



- Resources + prompts for UI extension and automation; validators (best-effort).



### Phase 5 — Enterprise hardening



- **Security** review (tokens, SSRF, cache poisoning, multi-tenant isolation if ever applicable).

- **OAuth** / enterprise auth modes if required; audit logging; **Docker** + **cache** runbooks.

- Load and **soak** testing with **realistic** agent sessions.



---



## 14. Resolved / open decisions



**Resolved:** use **Go** with **github.com/modelcontextprotocol/go-sdk** for MCP protocol implementation.



**Still open:**



- Minimum **MCP spec** version for supported clients.

- **GraphQL mutations** vs **REST-only** for writes.

- **OAuth** vs PAT-only for enterprise.

- **Cache backend** default (embedded vs Redis) per deployment size.

- **Exact** per-entity **uniqueness** rules — **research** against live docs and tenant behavior (§8).



---



## 15. Open questions & research items (enterprise-grade, security)



**Product / data model**



- Precise rules for **unique** fields per entity (names, `sourceID`, financial IDs, product fields, etc.) including **archived / trashed / disabled** records.

- **Directory vs account** scoping (see API changelog notes on organizations/sites) — how it affects queries.



**Security & compliance**



- **Token** storage and rotation; **no** logging of secrets; **PII** minimization in caches and logs.

- **SSRF** if any URL-fetching or webhook tools are added; **go-sdk** security guidance.

- **Tenant isolation** if the server ever multiplexes accounts.

- **Data residency** and **cache location** for regulated customers.

- **Sandbox** for optional code mode: syscall restrictions, CPU/memory/time limits.



**Operations**



- **Cache consistency** after external changes (UI edits while agent runs).

- **429** storms: coordinated backoff across tools and user-visible **degraded** mode.

- **Observability:** structured logs, metrics, **no** sensitive payload dumps.



**Legal / docs**



- **Copyright** for embedded doc excerpts; prefer **links** + **short** fair-use summaries.

- **Terms** of use for connecting AI agents to production ITSM.



**Testing**



- **Conformance** tests (go-sdk scripts) + **contract** tests against Xurrent **sandbox** tenants.

- **Red-team** prompts for **tool injection** and **approval bypass** attempts.



---



## 16. Next step for you



Copy `.env.example` to `.env`, set **`XURRENT_TOKEN`** and **`XURRENT_ACCOUNT`**, and optionally **`XURRENT_HTTP_MIN_INTERVAL`** for safe testing. When implementation starts, use this document as the acceptance baseline for **efficiency**, **draft-first** behavior, **risk signaling**, **domain literacy without prompt bloat**, **response shaping**, **external caching**, and **identifier collision** handling.


