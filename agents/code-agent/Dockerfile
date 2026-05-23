FROM golang:1.26-alpine AS build
WORKDIR /app

# Copy module files first for better caching
COPY go.mod go.sum* ./
RUN go mod download || true

# Copy all source
COPY adk/ ./adk/
COPY code-agent/ ./code-agent/

# Build the code-agent binary
WORKDIR /app/code-agent
RUN go build -o /code-agent .

# Runtime stage
FROM alpine:3.20
RUN apk --no-cache add ca-certificates
COPY --from=build /code-agent /code-agent
COPY --from=build /app/code-agent/config.yaml /config.yaml
WORKDIR /
EXPOSE 8081
CMD ["/code-agent"]
