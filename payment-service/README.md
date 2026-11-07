# payment-service

Учебный backend-проект на Go с доменом `payments`, PostgreSQL 16 и Docker Compose.

## Стек

- Go
- PostgreSQL 16
- Docker
- Docker Compose

## Структура

```text
payment-service/
  cmd/
    payment-service/
      main.go
  internal/
    config/
      config.go
    db/
      postgres.go
    handler/
      payment_handler.go
      health_handler.go
    repository/
      payment_repository.go
    service/
      payment_service.go
  db/
    init.sql
  Dockerfile
  docker-compose.yml
  README.md
  go.mod
  go.sum
```

## Запуск

```bash
docker compose up -d --build
Что делает команда:

docker compose — современная версия Docker Compose без дефиса.
up — создаёт и запускает сервисы из docker-compose.yml.
-d — запускает контейнеры в фоне.
--build — пересобирает image перед запуском, если в проекте есть build:.
```
```md
docker-compose up
старый вариант Compose V1. Сейчас лучше использовать `docker compose up`.
```

## Проверка контейнеров

```bash
docker compose ps
```

## Логи

```bash
docker compose logs -f payment-service
```

## Проверка API

```bash
curl http://localhost:8080/health
```

```bash
curl http://localhost:8080/ready
```

```bash
curl http://localhost:8080/payments
```

```bash
curl http://localhost:8080/payments/1
```

```bash
curl -X POST http://localhost:8080/payments \
  -H "Content-Type: application/json" \
  -d '{"order_id":1005,"provider":"paypal","status":"created","amount":150.75,"currency":"EUR"}'
```

## Остановка

```bash
docker compose down
```

## Полная очистка с volume

```bash
docker compose down -v
```

## Переменные окружения

Поддерживаются переменные с дефолтами:

- `APP_PORT` (default `8080`)
- `DB_HOST` (default `localhost`)
- `DB_PORT` (default `5432`)
- `DB_NAME` (default `payments_db`)
- `DB_USER` (default `payment_user`)
- `DB_PASSWORD` (default `payment_password`)

## API

- `GET /health` -> `{ "status": "ok" }`
- `GET /ready` -> ping PostgreSQL, если БД недоступна возвращает `503`
- `GET /payments` -> список платежей
- `GET /payments/{id}` -> платеж по id, `404` если не найден
- `POST /payments` -> создание платежа

## Lab 04.3 - First Kubernetes Pod

1. Собрать локальный image:

```bash
docker build -t payment-service:local .
```

2. Применить Pod:

```bash
kubectl apply -f k8s/payment-pod.yaml
```

3. Проверить Pod:

```bash
kubectl get pods -n payments
```

4. Посмотреть подробности:

```bash
kubectl describe pod payment-service-pod -n payments
```

5. Посмотреть логи:

```bash
kubectl logs payment-service-pod -n payments
```

6. Удалить Pod:

```bash
kubectl delete -f k8s/payment-pod.yaml
```

## Lab 04.4 - Deployment, ReplicaSet, Scaling

```bash
docker build -t payment-service:local .
kubectl apply -f k8s/09-payment-deployment.yaml
kubectl get deployments -n payments
kubectl get rs -n payments
kubectl get pods -n payments
kubectl describe deployment payment-service -n payments
kubectl describe pod <pod-name> -n payments
kubectl logs <pod-name> -n payments
kubectl delete pod <pod-name> -n payments
kubectl scale deployment payment-service --replicas=3 -n payments
kubectl get pods -n payments
kubectl scale deployment payment-service --replicas=1 -n payments
kubectl get pods -n payments
kubectl delete -f k8s/09-payment-deployment.yaml
```

## Lab 05.4 - Rolling update (v1 -> v2 -> v3)

Цель: обновлять приложение постепенно, без одновременной остановки всех Pod'ов.

### Что подготовлено в коде

- Добавлена переменная окружения `APP_VERSION` (default: `v1`).
- API теперь возвращает версию в ответах:
  - `GET /health` -> `{ "status": "ok", "version": "v1|v2|v3" }`
  - `GET /ready` -> `{ "status": "ready", "version": "v1|v2|v3" }`

### Сборка 3 версий image

```bash
docker build --build-arg APP_VERSION=v1 -t payment-service:local .
docker build --build-arg APP_VERSION=v2 -t payment-service:v2 .
docker build --build-arg APP_VERSION=v3 -t payment-service:v3 .
```

### Таблица команд (шпаргалка)

| Шаг | Команда |
|---|---|
| Deploy базовой версии | `kubectl apply -f k8s/09-payment-deployment.yaml` |
| Проверка Pod'ов | `kubectl get pods -n payments` |
| Обновить на v2 | `kubectl set image deployment/payment-service payment-service=payment-service:v2 -n payments` |
| Статус rollout | `kubectl rollout status deployment/payment-service -n payments` |
| История rollout | `kubectl rollout history deployment/payment-service -n payments` |
| Проверка ReplicaSet | `kubectl get rs -n payments` |
| Проверка Pod'ов после обновления | `kubectl get pods -n payments` |
| Обновить на v3 | `kubectl set image deployment/payment-service payment-service=payment-service:v3 -n payments` |
| Статус rollout (v3) | `kubectl rollout status deployment/payment-service -n payments` |
| История rollout (v3) | `kubectl rollout history deployment/payment-service -n payments` |
| Проверка API (NodePort) | `curl http://localhost:30080/health` |
| Проверка API ready (NodePort) | `curl http://localhost:30080/ready` |

### Что важно помнить

- `set image` создает новую ревизию Deployment.
- В rolling update появляется новый ReplicaSet, старый уменьшается постепенно.
- Для `imagePullPolicy: Never` образ с нужным тегом должен существовать локально.
- Тег должен совпадать точно: `v3`, а не `3`.

## Lab 05.5 - Rollout history и rollback

Цель: посмотреть ревизии Deployment и потренировать откаты.

### Команды

```bash
kubectl rollout history deployment/payment-service -n payments
kubectl rollout undo deployment/payment-service -n payments
kubectl rollout status deployment/payment-service -n payments
kubectl rollout undo deployment/payment-service --to-revision=1 -n payments
```

### Что важно понимать

- `kubectl rollout undo deployment/payment-service -n payments` откатывает Deployment на предыдущую ревизию.
- `--to-revision=<N>` откатывает Deployment на конкретную ревизию (например, `--to-revision=1`).
- `revisionHistoryLimit` в Deployment ограничивает, сколько старых `ReplicaSet` хранится для rollback.

## Lab 05.6 - Probes (startup, readiness, liveness)

Цель: добавить health-пробы, чтобы Kubernetes корректно понимал старт, готовность и "живость" приложения.

### Команды

```bash
kubectl apply -f k8s/09-payment-deployment.yaml
kubectl get pods -n payments
kubectl describe pod <pod-name> -n payments
kubectl logs <pod-name> -n payments
kubectl logs <pod-name> -n payments --previous
```

### Что важно понимать

- `startupProbe` проверяет старт приложения.
- `readinessProbe` проверяет готовность принимать трафик.
- `livenessProbe` проверяет, что приложение живо.

## Lab 05.7 - Resources (requests/limits)

Цель: задать минимальные и максимальные ресурсы контейнера.

### Команды

```bash
kubectl apply -f k8s/09-payment-deployment.yaml
kubectl describe pod <pod-name> -n payments
kubectl describe node docker-desktop
```

### Что важно понимать

- `requests` нужны scheduler для размещения Pod.
- `limits` ограничивают максимальное потребление.
- `cpu: 50m` означает 50 millicpu, то есть `0.05` CPU.
- `memory: 64Mi` означает 64 mebibytes памяти.
- Если контейнер превысит memory limit, возможен `OOMKilled`.

## Lab 05.8 - Kubernetes QoS (Quality of Service)

Цель: понять, как Kubernetes назначает Pod класс QoS на основе `requests/limits`.

### QoS классы

1. `BestEffort` — у контейнеров не заданы `requests` и `limits`.
2. `Burstable` — `requests` и `limits` заданы не полностью или отличаются.
3. `Guaranteed` — для `cpu` и `memory` заданы и `requests`, и `limits`, и они равны друг другу.

### Команды

```bash
kubectl describe pod <pod-name> -n payments
kubectl get pod <pod-name> -n payments -o yaml
```

### Где смотреть QoS Class

- В выводе `kubectl describe pod <pod-name> -n payments` есть отдельная строка `QoS Class: ...`.
- В YAML (`-o yaml`) можно смотреть поля `resources.requests` и `resources.limits`, по которым Kubernetes определяет класс QoS.

## Lab 05.9 - Kubelet, Node Diagnostics и kubectl explain

Цель: понять роль `kubelet` на node, посмотреть диагностику node и научиться читать встроенную справку по полям Kubernetes API.

### Команды

```bash
kubectl describe node docker-desktop
kubectl get events -n payments
kubectl explain deployment
kubectl explain deployment.spec
kubectl explain deployment.spec.template
kubectl explain deployment.spec.template.spec.containers
```

### Kubelet: что важно понимать

- `kubelet` работает на каждой node.
- `kubelet` получает `PodSpec` от control plane (через Kubernetes API).
- `kubelet` просит container runtime (например, containerd) запустить контейнеры.
- `kubelet` проверяет `startupProbe`, `readinessProbe` и `livenessProbe`.
- При `livenessProbe` failure `kubelet` перезапускает контейнеры по `restartPolicy` Pod.
- При нехватке ресурсов на node `kubelet` участвует в eviction (выселении) Pod.

### Node diagnostics

- `kubectl describe node docker-desktop` показывает состояние node: `Conditions`, `Capacity/Allocatable`, запущенные Pod и события.
- `kubectl get events -n payments` показывает события namespace `payments`: проблемы с запуском, probes, перезапуски, scheduling и другое.

### kubectl explain

- `kubectl explain` — это встроенная справка по Kubernetes API fields.
- Команда показывает описание поля, его тип и вложенные поля.
- Удобно идти сверху вниз: `deployment` -> `deployment.spec` -> `deployment.spec.template` -> `deployment.spec.template.spec.containers`.

## Дополнительно: Kiosk и multi-tenancy

Kiosk — сторонний open-source инструмент для multi-tenancy в Kubernetes.

Он не входит в Kubernetes по умолчанию и устанавливается отдельно в кластер.

Kiosk нужен, когда один Kubernetes-кластер используют несколько команд или проектов.

Пример:

- payments team
- analytics team
- frontend team

Каждая команда может получить своё изолированное пространство, namespace, права доступа и лимиты ресурсов.

Kiosk строится поверх стандартных механизмов Kubernetes:

- Namespace
- RBAC
- ResourceQuota
- LimitRange
- NetworkPolicy

Основные термины:

| Термин | Описание |
|---|---|
| multi-tenancy | Один кластер используют несколько команд или проектов |
| tenant | Арендатор: команда, проект или группа пользователей |
| account | Логическая группа пользователей в Kiosk |
| space | Изолированное рабочее пространство |
| RBAC | Управление правами доступа |
| ResourceQuota | Ограничение ресурсов |
| LimitRange | Дефолтные requests/limits |
| NetworkPolicy | Сетевая изоляция |

Важно:

Kiosk — не обязательная часть Kubernetes.
Для начала важнее понимать Namespace, RBAC, ResourceQuota, LimitRange и NetworkPolicy.
Kiosk можно изучать позже как дополнительный инструмент для multi-tenancy.
