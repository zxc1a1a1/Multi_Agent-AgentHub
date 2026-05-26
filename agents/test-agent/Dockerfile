FROM golang:1.26-alpine AS build
WORKDIR /app

# Copy module files first for better caching
COPY go.mod go.sum* ./
RUN go mod download || true

# Copy all source
COPY adk/ ./adk/
COPY test-agent/ ./test-agent/

# Build the test-agent binary
WORKDIR /app/test-agent
RUN go build -o /test-agent .

# Runtime stage
FROM alpine:3.20
RUN apk --no-cache add ca-certificates
COPY --from=build /test-agent /test-agent
COPY --from=build /app/test-agent/config.yaml /config.yaml
WORKDIR /
EXPOSE 8095
CMD ["/test-agent"]
