# xurrent-mcp

MCP (Model Context Protocol) server for the [Xurrent](https://xurrent.com/) ITSM platform. Provides AI agents with read/write access to 35+ Xurrent entity types through a small set of powerful, layered tools.

## Credentials

You need a Xurrent personal access token. Create one at **My Profile → Personal Access Tokens** in Xurrent, or follow [the docs](https://developer.xurrent.com/v1/#personal-access-tokens).

### Configuration by Client Type

#### Docker (HTTP transport)

```bash
docker run -d \
  -e XURRENT_TOKEN=your-personal-access-token \
  -e XURRENT_ACCOUNT=your-account-id \
  -e XURRENT_ALLOW_MUTATIONS=0 \
  -p 8080:8080 \
  --name xurrent-mcp \
  ghcr.io/bab3l/xurrent-mcp:latest
```

#### Docker Compose

```yaml
services:
  xurrent-mcp:
    image: ghcr.io/bab3l/xurrent-mcp:latest
    ports:
      - "8080:8080"
    environment:
      XURRENT_TOKEN: ${XURRENT_TOKEN}
      XURRENT_ACCOUNT: ${XURRENT_ACCOUNT}
      XURRENT_ALLOW_MUTATIONS: "0"
      XURRENT_HTTP_MIN_INTERVAL: "0.4"
```
Then: `XURRENT_TOKEN=... XURRENT_ACCOUNT=... docker compose up -d`

#### VS Code / Copilot

Add to `.vscode/mcp.json`:
```json
{
  "servers": {
    "xurrent": {
      "type": "streamableHttp",
      "url": "http://localhost:8080/mcp"
    }
  }
}
```
The Docker container passes credentials via environment — no additional config needed.

#### Cursor

In Cursor Settings → MCP → Add Server:
```json
{
  "mcpServers": {
    "xurrent": {
      "url": "http://localhost:8080/mcp"
    }
  }
}
```

#### Claude Desktop (stdio transport)

In `claude_desktop_config.json`:
```json
{
  "mcpServers": {
    "xurrent": {
      "command": "docker",
      "args": [
        "run", "-i", "--rm",
        "-e", "XURRENT_TOKEN",
        "-e", "XURRENT_ACCOUNT",
        "-e", "XURRENT_ALLOW_MUTATIONS",
        "-e", "XURRENT_HTTP_MIN_INTERVAL",
        "ghcr.io/bab3l/xurrent-mcp:latest"
      ],
      "env": {
        "XURRENT_TOKEN": "your-personal-access-token",
        "XURRENT_ACCOUNT": "your-account-id",
        "XURRENT_ALLOW_MUTATIONS": "0",
        "XURRENT_HTTP_MIN_INTERVAL": "0.4"
      }
    }
  }
}
```

#### Local development (stdio transport)

```bash
export XURRENT_TOKEN=your-personal-access-token
export XURRENT_ACCOUNT=your-account-id
go run ./cmd/xurrent-mcp
```

### How Credentials Flow

| Transport | How credentials reach the server |
|-----------|----------------------------------|
| HTTP (Docker) | Container environment variables (`docker run -e`) |
| stdio (Claude Desktop) | Process environment inherited from launcher config `env` block |
| stdio (Cursor local) | Process environment from shell that launched Cursor |
| stdio (dev) | Shell environment (`export`) |

The server reads `XURRENT_TOKEN` and `XURRENT_ACCOUNT` once at startup. All client sessions share the same credentials. To use different accounts, run separate server instances.

## Tools

| Tool | Type | Description |
|------|------|-------------|
| `xurrent_discover` | Read | List 35 entity types with fields, filters, relationships, docs |
| `xurrent_query` | Read | Fetch records from Xurrent API with filtering, sorting, pagination |
| `xurrent_read` | Read | Get single record by entity type + ID |
| `xurrent_search` | Read | Full-text search across all records |
| `xurrent_rate_limit` | Read | Check API rate limit status |
| `xurrent_create` | Write | Create record → returns draft for approval |
| `xurrent_update` | Write | Update record → returns draft for approval |
| `xurrent_delete` | Write | Delete/archive record → returns draft (destructive) |
| `xurrent_draft` | Meta | View, commit, or discard pending drafts |
| `xurrent_detect_collision` | Read | Discovery steps for duplicate-key rows |
| `xurrent_find_candidates` | Read | Live API search for duplicate records |
| `xurrent_validate_automation` | Read | Offline syntax check for automation rules |
| `xurrent_validate_ui_extension` | Read | Offline check for UI extension HTML/CSS/JS |

## Resources & Skills

| Resource/Skill | Description |
|----------------|-------------|
| `xurrent://domain/graph` | Entity relationship map (services, SLAs, CIs, requests, workflows) |
| `xurrent://docs/index` | Documentation URL index by topic |
| `xurrent://docs/ui-extensions` | UI Extension JavaScript API and CSS constraints |
| `xurrent://docs/automation` | Automation rule expression syntax and patterns |
| `itsm-relationships` | How ITSM entities relate — service delivery chain, request flow, CMDB |
| `ui-extension-development` | Guide to building Xurrent UI extensions |
| `automation-rule-development` | Guide to creating automation rules |
| `email-triage` | Step-by-step guide for investigating inbound email issues |

## Entity Coverage

### Priority 0 (core — full API access)
teams, sites, services, service_instances, people, organizations, configuration_items, SLAs, requests, workflows, UI extensions, automation_rules

### Priority 1 (important)
service_offerings, request_templates, workflow_templates, tasks, task_templates, problems, projects, calendars, products, notes (via request_note), events, audit_entries, inbound_emails

### Priority 2 (auxiliary / sub-resource only)
account, contracts, invoices, surveys, reservations, product_categories, custom_collections, webhooks

## Environment Variables

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `XURRENT_TOKEN` | Yes | — | Personal access token from Xurrent |
| `XURRENT_ACCOUNT` | Yes | — | Xurrent account ID |
| `XURRENT_ALLOW_MUTATIONS` | No | `0` | Set to `1` to enable create/update/delete |
| `XURRENT_HTTP_MIN_INTERVAL` | No | `0.4` | Seconds between API calls (rate limit) |
| `XURRENT_API_BASE` | No | `https://api.xurrent.com` | API endpoint override |
| `MCP_TRANSPORT` | No | `http` | `stdio` for local, `http` for Docker |
| `PORT` | No | `8080` | HTTP listen port |
| `LOG_LEVEL` | No | `info` | `debug`, `info`, `warn`, `error` |

## Safety

- **Draft-first mutations**: All writes generate a draft that must be explicitly committed
- **Read-only by default**: `XURRENT_ALLOW_MUTATIONS=0` blocks all creates/updates/deletes
- **Rate limiting**: Configurable minimum interval between API calls with 429 retry and exponential backoff
- **Response shaping**: Automatic JSON compaction and truncation to preserve context window

## Development

```bash
# Clone both repos side by side
git clone https://github.com/xurrent/go-xurrent.git
git clone https://github.com/xurrent/xurrent-mcp.git

cd xurrent-mcp
cp .env.example .env   # edit .env with your credentials

# Run tests
make test              # unit tests
make test-integration  # live API tests (needs credentials)

# Run locally (stdio mode)
make run

# Build Docker image (from workspace root containing both repos)
make docker

# Pre-commit hooks
git config core.hooksPath .githooks
```

## License

Apache 2.0 — see [LICENSE](LICENSE)
