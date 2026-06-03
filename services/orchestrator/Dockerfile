FROM golang:1.26-alpine AS builder

WORKDIR /src

COPY go.work go.work.sum ./
COPY services/orchestrator ./services/orchestrator
COPY pkg/adk ./pkg/adk

WORKDIR /src/services/orchestrator
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/orchestrator ./cmd/orchestrator

FROM alpine:3.20

RUN addgroup -S app && adduser -S -G app app
COPY --from=builder /out/orchestrator /usr/local/bin/orchestrator

USER app
EXPOSE 8080
CMD ["orchestrator"]
