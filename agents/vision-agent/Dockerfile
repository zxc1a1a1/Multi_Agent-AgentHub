FROM golang:1.26-alpine AS build
WORKDIR /app

# Copy module files first for better caching
COPY go.mod go.sum* ./
RUN go mod download || true

# Copy all source
COPY adk/ ./adk/
COPY vision-agent/ ./vision-agent/

# Build the vision-agent binary
WORKDIR /app/vision-agent
RUN go build -o /vision-agent .

# Runtime stage
FROM alpine:3.20
RUN apk --no-cache add ca-certificates
COPY --from=build /vision-agent /vision-agent
COPY --from=build /app/vision-agent/config.yaml /config.yaml
WORKDIR /
EXPOSE 8092
CMD ["/vision-agent"]
