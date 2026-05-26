FROM golang:1.26-alpine AS build
WORKDIR /app

# Copy module files first for better caching
COPY go.mod go.sum* ./
RUN go mod download || true

# Copy all source
COPY adk/ ./adk/
COPY agent-builder-agent/ ./agent-builder-agent/

# Build the agent-builder-agent binary
WORKDIR /app/agent-builder-agent
RUN go build -o /agent-builder-agent .

# Runtime stage
FROM alpine:3.20
RUN apk --no-cache add ca-certificates
COPY --from=build /agent-builder-agent /agent-builder-agent
COPY --from=build /app/agent-builder-agent/config.yaml /config.yaml
WORKDIR /
EXPOSE 8100
CMD ["/agent-builder-agent"]
