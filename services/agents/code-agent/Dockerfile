FROM golang:1.26-alpine AS builder

WORKDIR /src

COPY go.work go.work.sum ./

COPY agents ./agents
COPY server ./server
COPY pkg ./pkg
COPY services/gateway ./services/gateway
COPY services/orchestrator ./services/orchestrator
COPY services/agents/code-agent ./services/agents/code-agent
COPY services/agents/web-agent ./services/agents/web-agent

WORKDIR /src/services/agents/code-agent
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/code-agent ./cmd/code-agent

FROM alpine:3.20

RUN addgroup -S app && adduser -S -G app app
COPY --from=builder /out/code-agent /usr/local/bin/code-agent

USER app
EXPOSE 8080
CMD ["code-agent"]

