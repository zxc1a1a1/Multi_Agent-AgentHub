FROM golang:1.26-alpine AS build
WORKDIR /app

# Copy module files first for better caching
COPY go.mod go.sum* ./
RUN go mod download || true

# Copy all source
COPY adk/ ./adk/
COPY release-agent/ ./release-agent/

# Build the release-agent binary
WORKDIR /app/release-agent
RUN go build -o /release-agent .

# Runtime stage
FROM alpine:3.20
RUN apk --no-cache add ca-certificates
COPY --from=build /release-agent /release-agent
COPY --from=build /app/release-agent/config.yaml /config.yaml
WORKDIR /
EXPOSE 8106
CMD ["/release-agent"]
