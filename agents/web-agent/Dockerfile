FROM golang:1.26-alpine AS build
WORKDIR /app

# Copy module files first for better caching
COPY go.mod go.sum* ./
RUN go mod download || true

# Copy all source
COPY adk/ ./adk/
COPY web-agent/ ./web-agent/

# Build the web-agent binary
WORKDIR /app/web-agent
RUN go build -o /web-agent .

# Runtime stage
FROM alpine:3.20
RUN apk --no-cache add ca-certificates
COPY --from=build /web-agent /web-agent
COPY --from=build /app/web-agent/config.yaml /config.yaml
WORKDIR /
EXPOSE 8082
CMD ["/web-agent"]
