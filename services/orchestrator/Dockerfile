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

RUN go mod download

# Layer 2: source code (only what orchestrator needs)
COPY pkg                         ./pkg
COPY services/orchestrator       ./services/orchestrator

RUN --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=linux \
    go build -o /out/orchestrator ./services/orchestrator/cmd/orchestrator

FROM alpine:3.20
RUN addgroup -S app && adduser -S -G app app
COPY --from=builder /out/orchestrator /usr/local/bin/orchestrator
USER app
EXPOSE 8080
CMD ["orchestrator"]
