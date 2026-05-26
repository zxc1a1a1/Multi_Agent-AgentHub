FROM golang:1.26-alpine AS build
WORKDIR /app

# Copy module files first for better caching
COPY go.mod go.sum* ./
RUN go mod download || true

# Copy all source
COPY adk/ ./adk/
COPY diff-agent/ ./diff-agent/

# Build the diff-agent binary
WORKDIR /app/diff-agent
RUN go build -o /diff-agent .

# Runtime stage
FROM alpine:3.20
RUN apk --no-cache add ca-certificates
COPY --from=build /diff-agent /diff-agent
COPY --from=build /app/diff-agent/config.yaml /config.yaml
WORKDIR /
EXPOSE 8093
CMD ["/diff-agent"]
