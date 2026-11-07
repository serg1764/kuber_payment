FROM golang:1.25-alpine AS builder

WORKDIR /app

COPY go.mod go.sum* ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /bin/notification-service ./cmd/notification-service

FROM alpine:3.20

WORKDIR /app
COPY --from=builder /bin/notification-service /notification-service

EXPOSE 9090
ENTRYPOINT ["/notification-service"]
