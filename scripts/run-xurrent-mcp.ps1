# Run the stdio MCP server (tools: detect_identifier_collision, find_identifier_candidates).
# Loads .env so XURRENT_TOKEN / XURRENT_ACCOUNT are set for live API tools.
# Cursor: command = go, args = run, ./cmd/xurrent-mcp, cwd = this repo (with go.mod).

$ErrorActionPreference = "Stop"
$xurrentMcpRoot = Split-Path -Parent $PSScriptRoot
$envFile = Join-Path $xurrentMcpRoot ".env"
if (Test-Path $envFile) {
	Get-Content $envFile | ForEach-Object {
		if ($_ -match '^\s*([^#][^=]*)=(.*)$') {
			$k = $matches[1].Trim()
			$v = $matches[2].Trim()
			Set-Item -Path "env:$k" -Value $v
		}
	}
}
Set-Location $xurrentMcpRoot

go run ./cmd/xurrent-mcp @args
