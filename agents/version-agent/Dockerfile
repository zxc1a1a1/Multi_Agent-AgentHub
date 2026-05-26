FROM golang:1.26-alpine AS build
WORKDIR /app

# Copy module files first for better caching
COPY go.mod go.sum* ./
RUN go mod download || true

# Copy all source
COPY adk/ ./adk/
COPY version-agent/ ./version-agent/

# Build the version-agent binary
WORKDIR /app/version-agent
RUN go build -o /version-agent .

# Runtime stage
FROM alpine:3.20
RUN apk --no-cache add ca-certificates
COPY --from=build /version-agent /version-agent
COPY --from=build /app/version-agent/config.yaml /config.yaml
WORKDIR /
EXPOSE 8099
CMD ["/version-agent"]
