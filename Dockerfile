# syntax=docker/dockerfile:1

# ── Stage 1: Build ──
FROM golang:1.26-alpine AS builder

RUN apk add --no-cache ca-certificates git

WORKDIR /src

# Copy both repos for Go module replace directive.
# In CI (GitHub Actions), both are checked out side-by-side.
# For local builds: docker build -f Dockerfile .. (from the parent of both repos).
COPY go-xurrent/go.mod go-xurrent/go.sum go-xurrent/
COPY go-xurrent/ go-xurrent/
COPY xurrent-mcp/go.mod xurrent-mcp/go.sum xurrent-mcp/
ENV GOPRIVATE=github.com/xurrent/*
ENV GONOSUMDB=*
ENV GONOSUMCHECK=*
RUN cd xurrent-mcp && go mod download

COPY xurrent-mcp/ xurrent-mcp/

RUN cd xurrent-mcp && GOPRIVATE=github.com/xurrent/* GONOSUMDB=* GONOSUMCHECK=* CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -trimpath -ldflags="-s -w" \
    -o /xurrent-mcp ./cmd/xurrent-mcp

# ── Stage 2: Runtime ──
FROM alpine:3.21

RUN apk add --no-cache ca-certificates tzdata wget && \
    adduser -D -H -s /sbin/nologin xurrent

USER xurrent

ENV MCP_TRANSPORT=http
ENV PORT=8080
ENV LOG_LEVEL=info
ENV XURRENT_HTTP_MIN_INTERVAL=0.4
ENV XURRENT_TOOL_MAX_RESPONSE_CHARS=120000
ENV XURRENT_TOOL_JSON_COMPACT=1

EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=5s --start-period=5s --retries=3 \
    CMD wget -qO- http://localhost:8080/health || exit 1

COPY --from=builder /xurrent-mcp /usr/local/bin/xurrent-mcp

ENTRYPOINT ["/usr/local/bin/xurrent-mcp"]
