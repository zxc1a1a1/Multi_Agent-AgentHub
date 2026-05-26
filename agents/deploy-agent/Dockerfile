FROM golang:1.26-alpine AS build
WORKDIR /app

# Copy module files first for better caching
COPY go.mod go.sum* ./
RUN go mod download || true

# Copy all source
COPY adk/ ./adk/
COPY deploy-agent/ ./deploy-agent/

# Build the deploy-agent binary
WORKDIR /app/deploy-agent
RUN go build -o /deploy-agent .

# Runtime stage
FROM alpine:3.20
RUN apk --no-cache add ca-certificates
COPY --from=build /deploy-agent /deploy-agent
COPY --from=build /app/deploy-agent/config.yaml /config.yaml
WORKDIR /
EXPOSE 8104
CMD ["/deploy-agent"]
