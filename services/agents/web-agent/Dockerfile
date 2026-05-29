FROM golang:1.22-alpine AS builder

WORKDIR /src

COPY go.work go.work.sum ./
COPY pkg/adk ./pkg/adk
COPY services/agents/web-agent ./services/agents/web-agent

WORKDIR /src/services/agents/web-agent
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/web-agent ./cmd/web-agent

FROM alpine:3.20

RUN addgroup -S app && adduser -S -G app app
COPY --from=builder /out/web-agent /usr/local/bin/web-agent

USER app
EXPOSE 8080
CMD ["web-agent"]

