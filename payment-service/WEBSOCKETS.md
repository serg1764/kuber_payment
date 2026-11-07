# WebSocket Demo в `payment-service`

Этот файл описывает, как работает демо WebSocket, добавленный в сервис.

## Что реализовано

- `GET /ws` — WebSocket endpoint.
- `GET /ws/status` — состояние WebSocket (есть ли клиенты, сколько, текущая пара).
- `POST /test/change-pair` — HTTP-команда смены пары для подключенных WS-клиентов.

## Логика работы

1. Клиент подключается к `GET /ws`.
2. Сервер запускает:
- цикл чтения входящих WS-сообщений;
- цикл отправки котировки (`quote`) раз в 1 секунду.
3. Если приходит сообщение:
```json
{"type":"change_pair","pair":"GBPUSD","requestId":"req-1"}
```
сервер:
- меняет текущую пару;
- пишет это в лог;
- отвечает подтверждением:
```json
{"type":"ack","status":"ok","pair":"GBPUSD","requestId":"req-1"}
```

## Пошаговый flow (точка входа -> что происходит дальше)

1. Точка входа HTTP-сервера:
- [`cmd/payment-service/main.go`](cmd/payment-service/main.go)
- Роутер `chi` регистрирует:
  - `GET /ws`
  - `GET /ws/status`
  - `POST /test/change-pair`

2. При вызове `GET /ws` попадаем в метод:
- [`WS(...)`](internal/handler/ws_handler.go)
- Делается `HTTP -> WebSocket upgrade`.
- Если upgrade успешен, соединение сохраняется в `clients`.

3. После подключения стартуют 2 параллельных процесса:
- `readLoop(conn)`:
  - постоянно читает входящие сообщения от клиента;
  - логирует raw JSON;
  - если `type=change_pair`, меняет `currentPair` и шлет `ack`.
- `writeLoop(conn)`:
  - раз в 1 секунду берет `currentPair`;
  - генерирует тестовую цену;
  - отправляет клиенту сообщение `quote`.

4. При вызове `POST /test/change-pair` попадаем в метод:
- [`ChangePair(...)`](internal/handler/ws_handler.go)
- Читаем JSON `{ "pair": "GBPUSD" }`.
- Формируем WS-команду `change_pair` с `requestId`.
- Делаем broadcast этой команды всем активным WS-клиентам.

5. Что дальше после `POST /test/change-pair`:
- WS-клиент получает команду `change_pair`.
- Если этот же сервер читает это сообщение через `readLoop`, он:
  - обновляет `currentPair` на новую пару;
  - пишет лог `ws pair changed ...`;
  - отправляет `ack`.
- Начиная со следующего тика в `writeLoop`, в поток идут `quote` уже по новой паре.

6. При вызове `GET /ws/status`:
- попадаем в `Status(...)`;
- возвращаем:
  - `connected` (есть ли хотя бы 1 клиент),
  - `clients` (количество),
  - `current_pair` (текущая активная пара).

7. При закрытии сокета или ошибке:
- `readLoop`/`writeLoop` завершаются;
- соединение удаляется из `clients`;
- пишется лог `ws disconnected`.

## Формат сообщений

### Котировка от сервера
```json
{
  "type": "quote",
  "pair": "EURUSD",
  "price": 1.08342,
  "ts": "2026-05-27T10:15:30Z"
}
```

### Команда смены пары от клиента
```json
{
  "type": "change_pair",
  "pair": "GBPUSD",
  "requestId": "req-123"
}
```

### Ответ-подтверждение от сервера
```json
{
  "type": "ack",
  "status": "ok",
  "pair": "GBPUSD",
  "requestId": "req-123"
}
```

## Быстрая проверка локально

1. Запуск сервиса:
```bash
go run ./cmd/payment-service
```

2. Подключение WS-клиентом (пример через `wscat`):
```bash
wscat -c ws://localhost:8080/ws
```

3. Проверка статуса:
```bash
curl http://localhost:8080/ws/status
```

4. Смена пары через HTTP:
```bash
curl -X POST http://localhost:8080/test/change-pair \
  -H "Content-Type: application/json" \
  -d '{"pair":"GBPUSD"}'
```

После этого в WS-потоке пойдут котировки с новой парой.

## Про подтверждения (ack)

- `ack` в этой демо-версии подтверждает, что сервер **получил и обработал** команду.
- Если `ack` не пришел в нужный таймаут, клиент может отправить повторно с тем же `requestId`.
- Для защиты от дублей в реальной системе добавляют идемпотентность по `requestId`.

## Может ли быть несколько клиентов

Да.  
В этом демо один сервер может держать много одновременных WebSocket-подключений:
- все активные подключения хранятся в `clients map[*websocket.Conn]struct{}`;
- `GET /ws/status` показывает текущее количество в поле `clients`;
- `POST /test/change-pair` рассылает команду сразу всем подключенным клиентам (broadcast).

## Как работает Gorilla WebSocket под капотом (в контексте этого проекта)

`github.com/gorilla/websocket` **не поднимает отдельный сервер сам по себе**.  
HTTP-сервер поднимаем мы в `main.go` (`net/http` + `chi`).

Что делает Gorilla:

1. В методе `WS(...)` вызывается `upgrader.Upgrade(w, r, nil)`.
2. Библиотека проверяет заголовки WebSocket-handshake и переводит HTTP-соединение в протокол WebSocket.
3. После upgrade она дает объект `*websocket.Conn`.
4. Дальше библиотека **сама ничего не генерирует периодически**:
- чтение делает наш `readLoop` через `ReadMessage()`;
- отправку раз в секунду делает наш `writeLoop` через `WriteMessage()`.

Итог: генерация котировок, ack, смена пары, логи, retry-логика — это код приложения, не “автоматика” Gorilla.

## Как запустить в Docker

### Вариант 1: через docker compose (рекомендуется)

Файл: `payment-service/docker-compose.yml`.

1. Перейти в папку:
```bash
cd payment-service
```

2. Поднять сервисы:
```bash
docker compose up --build
```

3. Проверить health:
```bash
curl http://localhost:8080/health
```

4. Проверить WS-статус:
```bash
curl http://localhost:8080/ws/status
```

5. Подключить WebSocket-клиент:
```bash
wscat -c ws://localhost:8080/ws
```

6. Сменить пару через HTTP:
```bash
curl -X POST http://localhost:8080/test/change-pair \
  -H "Content-Type: application/json" \
  -d '{"pair":"GBPUSD"}'
```

7. Остановить:
```bash
docker compose down
```

### Вариант 2: отдельно `docker build` + `docker run`

1. Сборка image:
```bash
docker build -t payment-service:local .
```

2. Запуск PostgreSQL (пример):
```bash
docker run --name payment-postgres \
  -e POSTGRES_DB=payments_db \
  -e POSTGRES_USER=payment_user \
  -e POSTGRES_PASSWORD=payment_password \
  -p 5432:5432 \
  -d postgres:16
```

3. Запуск `payment-service`:
```bash
docker run --name payment-service \
  -p 8080:8080 \
  -e APP_PORT=8080 \
  -e DB_HOST=host.docker.internal \
  -e DB_PORT=5432 \
  -e DB_NAME=payments_db \
  -e DB_USER=payment_user \
  -e DB_PASSWORD=payment_password \
  -e NOTIFICATION_GRPC_ADDR=host.docker.internal:9090 \
  payment-service:local
```

## Краткая шпаргалка (для собеседования)

`github.com/gorilla/websocket` не поднимает отдельный сервер сам по себе.  
HTTP-сервер поднимаем мы в `main.go` (`net/http` + `chi`).

Что делает Gorilla в нашем коде:

1. В методе `WS(...)` вызывается `upgrader.Upgrade(w, r, nil)`.
2. Библиотека проверяет заголовки WebSocket-handshake и переводит HTTP-соединение в WebSocket.
3. После upgrade возвращается `*websocket.Conn` — это и есть живое WS-соединение.

Что важно дальше:

1. Gorilla сама не генерирует периодические сообщения.
2. Чтение делает наш `readLoop` через `ReadMessage()`.
3. Отправку раз в секунду делает наш `writeLoop` через `WriteMessage()`.
4. `ack`, смена пары, генерация котировок, логи, retry-логика — это логика приложения.

Правильное понимание flow:

1. HTTP-запрос приходит в `WS(...)`.
2. `Upgrade(...)` переключает протокол на WebSocket.
3. Получаем `*websocket.Conn`.
4. Запускаем:
   - `go h.writeLoop(conn)` — отдельная goroutine на отправку;
   - `h.readLoop(conn)` — текущая goroutine на чтение.
5. Соединение работает в full-duplex: одновременно читаем и пишем.

Почему `writeLoop` запускается в отдельной goroutine:

1. `readLoop` блокирующий: он занимает текущий поток до закрытия сокета.
2. Если не вынести запись в `go`:
   - либо чтение не запустится,
   - либо запись не запустится,
   - двунаправленности не будет.
3. С двумя goroutine получаем корректную модель для WebSocket:
   - одна goroutine читает,
   - другая пишет.
