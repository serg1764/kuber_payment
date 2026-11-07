
# gRPC Client/Server Demo (`payment-service` -> `notification-service`)

Этот файл описывает, как работает текущая интеграция:

- `payment-service` выступает как **gRPC client**;
- `notification-service` выступает как **gRPC server**.

## Что реализовано

- В `payment-service`:
  - REST endpoint `POST /payments` создает платеж;
  - после создания платежа вызывается gRPC метод `SendPaymentStatus`.
- В `notification-service`:
  - gRPC метод `SendPaymentStatus` сохраняет статус в БД;
  - gRPC метод `GetPaymentStatusesByPaymentId` возвращает историю статусов.

## Логика работы

1. Клиент вызывает `POST /payments` у `payment-service`.
2. `payment-service` пишет платеж в `payments_db`.
3. `payment-service` вызывает `notification-service` по gRPC:
   - `NotificationService.SendPaymentStatus(...)`.
4. `notification-service` сохраняет запись в `payment_status_notifications`.
5. В ответ REST возвращается поле `notification_status`:
   - `sent` — gRPC отправка успешна;
   - `failed` — gRPC вызов завершился ошибкой;
   - `skipped` — gRPC клиент не инициализирован.

## Пошаговый flow (точка входа -> что происходит дальше)

1. Точка входа `payment-service`:
- [`cmd/payment-service/main.go`](cmd/payment-service/main.go)
- Создается gRPC клиент:
  - `grpcclient.NewNotificationClient(cfg.NotificationGRPCAddr)`
- Клиент передается в `PaymentService`.

2. REST вход:
- [`internal/handler/payment_handler.go`](internal/handler/payment_handler.go)
- Метод `CreatePayment(...)` принимает JSON и вызывает сервисный слой.

3. Сервисный слой `payment-service`:
- [`internal/service/payment_service.go`](internal/service/payment_service.go)
- `CreatePayment(...)`:
  - сначала `repo.CreatePayment(...)`;
  - затем (если клиент есть) `notificationClient.SendPaymentStatus(...)`;
  - выставляет `notification_status`.

4. gRPC client в `payment-service`:
- [`internal/grpcclient/notification_client.go`](internal/grpcclient/notification_client.go)
- Формирует `SendPaymentStatusRequest` и вызывает RPC.

5. Точка входа `notification-service`:
- [`cmd/notification-service/main.go`](../notification-service/cmd/notification-service/main.go)
- Поднимается `grpc.NewServer()` на `:9090`;
- Регистрируется `NotificationService`;
- Включен reflection для Postman/grpcurl.

6. gRPC server в `notification-service`:
- [`internal/grpcserver/notification_server.go`](../notification-service/internal/grpcserver/notification_server.go)
- `SendPaymentStatus(...)`:
  - валидирует вход;
  - сохраняет статус через repository.
- `GetPaymentStatusesByPaymentId(...)`:
  - валидирует `payment_id`;
  - читает список статусов по `payment_id`.

7. SQL слой `notification-service`:
- [`internal/repository/notification_repository.go`](../notification-service/internal/repository/notification_repository.go)
- `INSERT` в `payment_status_notifications`;
- `SELECT ... WHERE payment_id = $1 ORDER BY id`.

## Контракт gRPC

- [`notification-service/proto/notification.proto`](../notification-service/proto/notification.proto)
- Сервис:
  - `SendPaymentStatus`
  - `GetPaymentStatusesByPaymentId`

## Формат ключевых данных

### REST request в `payment-service`
```json
{
  "order_id": 3001,
  "provider": "stripe",
  "status": "paid",
  "amount": 120.5,
  "currency": "EUR"
}
```

### REST response из `payment-service`
```json
{
  "id": 12,
  "order_id": 3001,
  "provider": "stripe",
  "status": "paid",
  "amount": 120.5,
  "currency": "EUR",
  "created_at": "2026-05-28 10:00:00",
  "notification_status": "sent"
}
```

### gRPC request `GetPaymentStatusesByPaymentId` (Postman)
```json
{
  "payment_id": 12
}
```

## Быстрая проверка локально

1. Убедиться, что поднят `notification-service`:
```bash
cd notification-service
docker compose up -d --build
```

2. Поднять `payment-service`:
```bash
cd payment-service
docker compose up -d --build
```

3. Создать платеж:
```bash
curl -X POST http://localhost:8080/payments \
  -H "Content-Type: application/json" \
  -d '{"order_id":3001,"provider":"stripe","status":"paid","amount":120.5,"currency":"EUR"}'
```

4. Проверить историю статусов через `grpcurl`:
```bash
docker run --rm --network host fullstorydev/grpcurl:latest \
  -plaintext -d '{"payment_id":12}' \
  localhost:9090 notification.v1.NotificationService/GetPaymentStatusesByPaymentId
```

## Проверка в Postman (gRPC)

1. Создать `gRPC Request`.
2. Подключиться к `localhost:9090`.
3. Выбрать метод `NotificationService/GetPaymentStatusesByPaymentId`.
4. Отправить:
```json
{
  "payment_id": 12
}
```

Если получаешь `INVALID_ARGUMENT`, проверь:
- `payment_id > 0`;
- правильное имя поля (`payment_id`).

## Краткая шпаргалка (для собеседования)

1. Внешний вход остается REST (`POST /payments`).
2. Межсервисная синхронная коммуникация сделана через gRPC.
3. `payment-service` — клиент, `notification-service` — сервер.
4. gRPC контракт описан в `.proto`.
5. Ответ REST возвращает `notification_status`, чтобы явно видеть, сработал ли gRPC вызов.

## Подробная шпаргалка (что именно делали)

1. Описали контракт в `notification.proto`:
   - сервис `NotificationService`;
   - методы `SendPaymentStatus` и `GetPaymentStatusesByPaymentId`;
   - request/response message-модели.

2. Сгенерировали код из `.proto` через Docker с `protoc` внутри контейнера:
```bash
docker run --rm -v ${PWD}:/defs namely/protoc-all:1.51_2 -f proto/notification.proto -l go -o gen
```

3. После генерации появились generated-файлы:
   - `notification.pb.go` — protobuf-модели сообщений;
   - `notification_grpc.pb.go` — gRPC интерфейсы/регистрация/клиентские вызовы.

4. На стороне сервера (`notification-service`) реализовали методы, которые удовлетворяют сгенерированному интерфейсу `NotificationServiceServer`:
   - `SendPaymentStatus(ctx, req)`:
     - валидирует вход;
     - сохраняет запись в `payment_status_notifications`;
     - возвращает `ok` и `notification_id`.
   - `GetPaymentStatusesByPaymentId(ctx, req)`:
     - валидирует `payment_id`;
     - читает список из БД;
     - возвращает `items` (0..N записей).

5. В `main.go` сервера:
   - подняли `grpc.NewServer()`;
   - зарегистрировали `RegisterNotificationServiceServer(...)`;
   - включили `reflection.Register(...)` для Postman/grpcurl.

6. На стороне клиента (`payment-service`):
   - использовали сгенерированный `NotificationServiceClient`;
   - завернули его в `internal/grpcclient/notification_client.go`;
   - после успешного `CreatePayment` вызываем `SendPaymentStatus`.

7. В REST-ответ `POST /payments` добавили `notification_status`:
   - `sent` / `failed` / `skipped`,
   чтобы сразу видеть результат межсервисного gRPC-вызова.

8. Отдельно проверили gRPC API напрямую через Postman:
   - создали `gRPC Request` на `localhost:9090`;
   - выбрали метод `NotificationService/GetPaymentStatusesByPaymentId`;
   - отправили message с `payment_id`;
     - получили `0 OK` и список `items` из `notification-service`.
тест 1234567890 git commit 1234567890