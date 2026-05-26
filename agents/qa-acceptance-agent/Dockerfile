FROM golang:1.26-alpine AS build
WORKDIR /app

# Copy module files first for better caching
COPY go.mod go.sum* ./
RUN go mod download || true

# Copy all source
COPY adk/ ./adk/
COPY qa-acceptance-agent/ ./qa-acceptance-agent/

# Build the qa-acceptance-agent binary
WORKDIR /app/qa-acceptance-agent
RUN go build -o /qa-acceptance-agent .

# Runtime stage
FROM alpine:3.20
RUN apk --no-cache add ca-certificates
COPY --from=build /qa-acceptance-agent /qa-acceptance-agent
COPY --from=build /app/qa-acceptance-agent/config.yaml /config.yaml
WORKDIR /
EXPOSE 8103
CMD ["/qa-acceptance-agent"]
