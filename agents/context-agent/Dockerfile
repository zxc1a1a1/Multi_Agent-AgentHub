FROM golang:1.26-alpine AS build
WORKDIR /app

# Copy module files first for better caching
COPY go.mod go.sum* ./
RUN go mod download || true

# Copy all source
COPY adk/ ./adk/
COPY context-agent/ ./context-agent/

# Build the context-agent binary
WORKDIR /app/context-agent
RUN go build -o /context-agent .

# Runtime stage
FROM alpine:3.20
RUN apk --no-cache add ca-certificates
COPY --from=build /context-agent /context-agent
COPY --from=build /app/context-agent/config.yaml /config.yaml
WORKDIR /
EXPOSE 8091
CMD ["/context-agent"]
