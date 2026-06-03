FROM golang:1.26-alpine AS builder

WORKDIR /src

COPY go.work go.work.sum ./
COPY agents/go.mod ./agents/go.mod
COPY server/go.mod ./server/go.mod
COPY services/agents/code-agent/go.mod ./services/agents/code-agent/go.mod
COPY services/agents/web-agent/go.mod ./services/agents/web-agent/go.mod

COPY pkg/adk ./pkg/adk
COPY pkg/runtime ./pkg/runtime
COPY services/gateway ./services/gateway

WORKDIR /src/services/gateway
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/gateway ./cmd/gateway

FROM alpine:3.20

RUN addgroup -S app && adduser -S -G app app
COPY --from=builder /out/gateway /usr/local/bin/gateway

USER app
EXPOSE 8080
CMD ["gateway"]
