FROM golang:1.26-alpine AS builder
WORKDIR /src
ENV GOPROXY=https://goproxy.cn,direct

# Layer 1: workspace + module files (cached until any go.mod changes)
COPY go.work go.work.sum ./
COPY pkg/adk/go.mod            ./pkg/adk/
COPY pkg/runtime/go.mod         ./pkg/runtime/
COPY services/gateway/go.mod services/gateway/go.sum ./services/gateway/
COPY services/orchestrator/go.mod                     ./services/orchestrator/
COPY services/agents/code-agent/go.mod                ./services/agents/code-agent/
COPY services/agents/web-agent/go.mod                 ./services/agents/web-agent/
COPY services/agents/document-agent/go.mod            ./services/agents/document-agent/
COPY services/agents/vision-agent/go.mod              ./services/agents/vision-agent/
COPY services/agents/context-agent/go.mod             ./services/agents/context-agent/
COPY services/agents/test-agent/go.mod                ./services/agents/test-agent/
COPY services/agents/review-agent/go.mod              ./services/agents/review-agent/
COPY services/agents/security-agent/go.mod            ./services/agents/security-agent/
COPY services/agents/deploy-agent/go.mod              ./services/agents/deploy-agent/
COPY services/agents/diff-agent/go.mod                ./services/agents/diff-agent/
COPY services/agents/markdown-agent/go.mod         ./services/agents/markdown-agent/
COPY pkg/runtime/go.sum            ./pkg/runtime/
COPY services/orchestrator/go.sum  ./services/orchestrator/
COPY services/agents/markdown-agent/go.sum ./services/agents/markdown-agent/

RUN go mod download

# Layer 2: source code (only what orchestrator needs)
COPY pkg                         ./pkg
COPY services/orchestrator       ./services/orchestrator

RUN --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=linux \
    go build -o /out/orchestrator ./services/orchestrator/cmd/orchestrator

FROM mcr.microsoft.com/playwright:v1.55.0-jammy@sha256:55db7b712a981e9cc1ec757a4511cd01c5ed4e20bd7c84b2dee4e1d320102b9d

# The Playwright base image already ships ca-certificates for TLS.
ENV PLAYWRIGHT_BROWSERS_PATH=/ms-playwright \
    PLAYWRIGHT_SKIP_BROWSER_DOWNLOAD=1 \
    NODE_PATH=/app/node_modules \
    PLAYWRIGHT_RENDER_SCRIPT=/app/render.cjs

WORKDIR /app
COPY services/orchestrator/playwright/render.cjs /app/render.cjs
RUN npm install --omit=dev --no-audit --no-fund playwright@1.55.0 \
    && groupadd -r app && useradd -r -g app -d /app -s /usr/sbin/nologin app \
    && mkdir -p /data/artifacts /app && chown -R app:app /app /data

COPY --from=builder /out/orchestrator /usr/local/bin/orchestrator
USER app
EXPOSE 8080
CMD ["orchestrator"]
