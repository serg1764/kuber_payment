# gRPC Integration Overview

## Что в итоге происходит

1. Клиент делает `POST /payments` в `payment-service` (старый REST сценарий).
2. `payment-service` сохраняет платеж в своей БД (`payments_db`).
3. После успешного сохранения `payment-service` вызывает gRPC метод `SendPaymentStatus` у `notification-service`.
4. `notification-service` сохраняет статус в своей БД (`notifications_db`, таблица `payment_status_notifications`).
5. Историю статусов можно получить gRPC-методом `GetPaymentStatusesByPaymentId`.

Идея: REST остается только на входе в `payment-service`, а межсервисный вызов сделан через gRPC.

## Что добавлено в notification-service

### Контракт

- `proto/notification.proto`
  - `NotificationService.SendPaymentStatus`
  - `NotificationService.GetPaymentStatusesByPaymentId`

### Сгенерированный gRPC код

- `gen/notification-service/gen/notification/notification.pb.go`
- `gen/notification-service/gen/notification/notification_grpc.pb.go`

Это служебный generated code для protobuf/gRPC.

### Серверная реализация

- `internal/grpcserver/notification_server.go`
  - валидация входа
  - реализация двух gRPC методов
  - преобразование моделей БД в protobuf response

### Работа с БД

- `internal/repository/notification_repository.go`
  - `CreatePaymentStatus` -> `INSERT ... RETURNING id`
  - `GetPaymentStatusesByPaymentID` -> `SELECT ... WHERE payment_id = $1 ORDER BY id`

### Точка входа

- `cmd/notification-service/main.go`
  - подключение к БД
  - запуск `grpc.NewServer()` на `:9090`
  - регистрация `NotificationService`
  - включен gRPC reflection (для `grpcurl`)

### Инфраструктура

- `db/init.sql` (таблица `payment_status_notifications`)
- `docker-compose.yml` (notification-service + notification-postgres)
- `Dockerfile` (сборка сервиса)

## Что добавлено в payment-service

### gRPC контракт для клиента

- `proto/notification.proto`
- `gen/notification/notification.pb.go`
- `gen/notification/notification_grpc.pb.go`

Почему это есть в `payment-service`: клиенту нужны protobuf/gRPC типы для компиляции вызова.
Это не дублирование логики, это локальная копия контракта.

### gRPC клиент

- `internal/grpcclient/notification_client.go`
  - создание gRPC подключения
  - метод `SendPaymentStatus(...)`
  - закрытие соединения `Close()`

### Конфиг

- `internal/config/config.go`
  - добавлен `NOTIFICATION_GRPC_ADDR`
  - default: `localhost:9090`

### Инициализация в main

- `cmd/payment-service/main.go`
  - создается gRPC клиент к `notification-service`
  - клиент передается в `PaymentService`
  - если подключение не удалось, сервис продолжает работать без падения

### Бизнес-логика

- `internal/service/payment_service.go`
  - после `CreatePayment` вызывается `notificationClient.SendPaymentStatus(...)`
  - режим мягкий: если gRPC вызов не удался, платеж уже создан и не откатывается, ошибка только логируется

### Docker

- `docker-compose.yml`
  - добавлен `NOTIFICATION_GRPC_ADDR=host.docker.internal:9090` для доступа из контейнера `payment-service` к `notification-service`

## Почему контракт есть в двух сервисах

Коротко:

1. `notification-service` использует его как серверный контракт.
2. `payment-service` использует его как клиентский контракт.

Технически возможно вынести это в общий `shared-proto` модуль, чтобы не копировать. Сейчас сделан простой учебный вариант для быстрого старта.

## Как проверить руками

1. Создать платеж:

```bash
curl -X POST http://localhost:8080/payments \
  -H "Content-Type: application/json" \
  -d '{"order_id":2001,"provider":"stripe","status":"paid","amount":99.90,"currency":"EUR"}'
```

2. Проверить историю статусов через gRPC:

```bash
docker run --rm --network host fullstorydev/grpcurl:latest \
  -plaintext -d '{"paymentId":11}' \
  localhost:9090 notification.v1.NotificationService/GetPaymentStatusesByPaymentId
```

Важно: для `grpcurl` используй `camelCase` поля (`paymentId`), не `snake_case`.
