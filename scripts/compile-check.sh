#!/bin/sh
# compile-check.sh — Quick compilation check for all packages
# Run: bash compile-check.sh
# Requires: go 1.26+

set -e

echo "=== Compile check for xurrent-mcp ==="
echo ""

# Download deps first
echo "[1] Downloading dependencies..."
cd /xurrentmcp-workspace/xurrent-mcp
go mod download 2>&1 || true
go mod tidy 2>&1 || true

echo ""
echo "[2] go vet ./internal/..."
go vet ./internal/... 2>&1 || echo "  (vet found issues — see above)"

echo ""
echo "[3] go build ./cmd/xurrent-mcp"
go build ./cmd/xurrent-mcp 2>&1

echo ""
echo "[4] go test -run=NONE ./... (compile check only)"
go test -run=NONE ./... 2>&1

echo ""
echo "[5] go test -short ./internal/... (unit tests)"
go test -short ./internal/... 2>&1

echo ""
echo "=== Done ==="
