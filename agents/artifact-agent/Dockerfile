FROM golang:1.26-alpine AS build
WORKDIR /app

# Copy module files first for better caching
COPY go.mod go.sum* ./
RUN go mod download || true

# Copy all source
COPY adk/ ./adk/
COPY artifact-agent/ ./artifact-agent/

# Build the artifact-agent binary
WORKDIR /app/artifact-agent
RUN go build -o /artifact-agent .

# Runtime stage
FROM alpine:3.20
RUN apk --no-cache add ca-certificates
COPY --from=build /artifact-agent /artifact-agent
COPY --from=build /app/artifact-agent/config.yaml /config.yaml
WORKDIR /
EXPOSE 8096
CMD ["/artifact-agent"]
