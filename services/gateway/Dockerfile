FROM golang:1.26-alpine AS builder

WORKDIR /src

ENV GOPROXY=https://goproxy.cn,direct

COPY go.work go.work.sum ./

COPY agents ./agents
COPY server ./server
COPY pkg ./pkg
COPY services/gateway ./services/gateway
COPY services/orchestrator ./services/orchestrator
COPY services/agents/code-agent ./services/agents/code-agent
COPY services/agents/web-agent ./services/agents/web-agent

WORKDIR /src/services/gateway
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/gateway ./cmd/gateway

FROM alpine:3.20

RUN addgroup -S app && adduser -S -G app app
RUN mkdir -p /data && chown -R app:app /data
COPY --from=builder /out/gateway /usr/local/bin/gateway

USER app
EXPOSE 8080
CMD ["gateway"]
