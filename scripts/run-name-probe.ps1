# Run name-uniqueness integration probes (go-xurrent).
# Usage: .\scripts\run-name-probe.ps1   (from xurrent-mcp repo root)
# Optional in .env: XURRENT_TRUSTED_ACCOUNT_IDS=id1,id2 when resource.account.id != XURRENT_ACCOUNT (trust).

$ErrorActionPreference = "Stop"
$xurrentMcpRoot = Split-Path -Parent $PSScriptRoot
$envFile = Join-Path $xurrentMcpRoot ".env"
if (-not (Test-Path $envFile)) {
    Write-Error "Missing .env at $envFile"
}

$env:XURRENT_ALLOW_MUTATIONS = "1"
if (-not $env:XURRENT_HTTP_MIN_INTERVAL) {
    $env:XURRENT_HTTP_MIN_INTERVAL = "0.35"
}

Get-Content $envFile | ForEach-Object {
    if ($_ -match '^\s*([^#][^=]*)=(.*)$') {
        $k = $matches[1].Trim()
        $v = $matches[2].Trim()
        Set-Item -Path "env:$k" -Value $v
    }
}

$gitRoot = Split-Path -Parent $xurrentMcpRoot
$goXurrent = Join-Path $gitRoot "go-xurrent"
Set-Location $goXurrent

go test -tags=integration ./test/integration/ -run "TestProbe_(NameUniqueness|CompositeUniqueness)" -v -count=1 @args
