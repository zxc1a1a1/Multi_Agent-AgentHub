FROM golang:1.26-alpine AS build
WORKDIR /app

# Copy module files first for better caching
COPY go.mod go.sum* ./
RUN go mod download || true

# Copy all source
COPY adk/ ./adk/
COPY ppt-agent/ ./ppt-agent/

# Build the ppt-agent binary
WORKDIR /app/ppt-agent
RUN go build -o /ppt-agent .

# Runtime stage
FROM alpine:3.20
RUN apk --no-cache add ca-certificates
COPY --from=build /ppt-agent /ppt-agent
COPY --from=build /app/ppt-agent/config.yaml /config.yaml
WORKDIR /
EXPOSE 8105
CMD ["/ppt-agent"]
