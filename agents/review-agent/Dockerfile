FROM golang:1.26-alpine AS build
WORKDIR /app

# Copy module files first for better caching
COPY go.mod go.sum* ./
RUN go mod download || true

# Copy all source
COPY adk/ ./adk/
COPY review-agent/ ./review-agent/

# Build the review-agent binary
WORKDIR /app/review-agent
RUN go build -o /review-agent .

# Runtime stage
FROM alpine:3.20
RUN apk --no-cache add ca-certificates
COPY --from=build /review-agent /review-agent
COPY --from=build /app/review-agent/config.yaml /config.yaml
WORKDIR /
EXPOSE 8094
CMD ["/review-agent"]
