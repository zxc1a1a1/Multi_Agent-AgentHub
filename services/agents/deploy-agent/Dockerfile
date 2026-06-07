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

RUN go mod download

# Layer 2: source code (only what deploy-agent needs)
COPY pkg                               ./pkg
COPY services/agents/deploy-agent      ./services/agents/deploy-agent

RUN --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=linux \
    go build -o /out/deploy-agent ./services/agents/deploy-agent/cmd/deploy-agent

FROM alpine:3.20
RUN addgroup -S app && adduser -S -G app app
COPY --from=builder /out/deploy-agent /usr/local/bin/deploy-agent
USER app
EXPOSE 8080
CMD ["deploy-agent"]
