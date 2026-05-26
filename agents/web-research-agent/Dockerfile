FROM golang:1.26-alpine AS build
WORKDIR /app

# Copy module files first for better caching
COPY go.mod go.sum* ./
RUN go mod download || true

# Copy all source
COPY adk/ ./adk/
COPY web-research-agent/ ./web-research-agent/

# Build the web-research-agent binary
WORKDIR /app/web-research-agent
RUN go build -o /web-research-agent .

# Runtime stage
FROM alpine:3.20
RUN apk --no-cache add ca-certificates
COPY --from=build /web-research-agent /web-research-agent
COPY --from=build /app/web-research-agent/config.yaml /config.yaml
WORKDIR /
EXPOSE 8102
CMD ["/web-research-agent"]
