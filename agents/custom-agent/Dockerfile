FROM golang:1.26-alpine AS build
WORKDIR /app

# Copy module files first for better caching
COPY go.mod go.sum* ./
RUN go mod download || true

# Copy all source
COPY adk/ ./adk/
COPY custom-agent/ ./custom-agent/

# Build the custom-agent binary
WORKDIR /app/custom-agent
RUN go build -o /custom-agent .

# Runtime stage
FROM alpine:3.20
RUN apk --no-cache add ca-certificates
COPY --from=build /custom-agent /custom-agent
COPY --from=build /app/custom-agent/config.yaml /config.yaml
WORKDIR /
EXPOSE 8084
CMD ["/custom-agent"]
