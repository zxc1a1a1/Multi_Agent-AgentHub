FROM golang:1.26-alpine AS build
WORKDIR /app

# Copy module files first for better caching
COPY go.mod go.sum* ./
RUN go mod download || true

# Copy all source
COPY adk/ ./adk/
COPY file-agent/ ./file-agent/

# Build the file-agent binary
WORKDIR /app/file-agent
RUN go build -o /file-agent .

# Runtime stage
FROM alpine:3.20
RUN apk --no-cache add ca-certificates
COPY --from=build /file-agent /file-agent
COPY --from=build /app/file-agent/config.yaml /config.yaml
WORKDIR /
EXPOSE 8101
CMD ["/file-agent"]
