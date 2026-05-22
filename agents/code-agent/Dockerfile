FROM golang:1.23-alpine AS build
WORKDIR /app
COPY . .
RUN go build -o /code-agent .

FROM alpine:3.20
COPY --from=build /code-agent /code-agent
CMD ["/code-agent"]
