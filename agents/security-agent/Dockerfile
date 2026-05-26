FROM golang:1.26-alpine AS build
WORKDIR /app

# Copy module files first for better caching
COPY go.mod go.sum* ./
RUN go mod download || true

# Copy all source
COPY adk/ ./adk/
COPY security-agent/ ./security-agent/

# Build the security-agent binary
WORKDIR /app/security-agent
RUN go build -o /security-agent .

# Runtime stage
FROM alpine:3.20
RUN apk --no-cache add ca-certificates
COPY --from=build /security-agent /security-agent
COPY --from=build /app/security-agent/config.yaml /config.yaml
WORKDIR /
EXPOSE 8097
CMD ["/security-agent"]
