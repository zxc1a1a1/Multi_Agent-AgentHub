FROM golang:1.26-alpine AS build
WORKDIR /app

# Copy module files first for better caching
COPY go.mod go.sum* ./
RUN go mod download || true

# Copy all source
COPY adk/ ./adk/
COPY document-agent/ ./document-agent/

# Build the document-agent binary
WORKDIR /app/document-agent
RUN go build -o /document-agent .

# Runtime stage
FROM alpine:3.20
RUN apk --no-cache add ca-certificates
COPY --from=build /document-agent /document-agent
COPY --from=build /app/document-agent/config.yaml /config.yaml
WORKDIR /
EXPOSE 8083
CMD ["/document-agent"]
