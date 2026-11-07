FROM golang:1.25-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /bin/payment-service ./cmd/payment-service

FROM alpine:3.20

WORKDIR /app
RUN adduser -D -g '' appuser
ARG APP_VERSION=v4-Dockerfile
ENV APP_VERSION=$APP_VERSION

COPY --from=builder /bin/payment-service /app/payment-service

EXPOSE 8080

USER appuser

CMD ["/app/payment-service"]
