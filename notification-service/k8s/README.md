# notification-service Kubernetes manifests

## Порядок применения

```bash
kubectl apply -f payment-service/k8s/01-namespace.yaml
kubectl apply -f payment-service/k8s/08-postgres.yaml
kubectl apply -f notification-service/k8s/02-notification-db-init-job.yaml
kubectl apply -f notification-service/k8s/03-notification-deployment.yaml
kubectl apply -f notification-service/k8s/04-notification-services.yaml
```

## Проверка Job (инициализация таблицы)

```bash
kubectl get jobs -n payments
kubectl logs job/notification-db-init -n payments
```

## Проверка Deployment

```bash
kubectl get deployment notification-service -n payments
kubectl rollout status deployment/notification-service -n payments
kubectl get pods -n payments -l app=notification-service
kubectl describe pod -n payments -l app=notification-service
```

## Проверка Service

```bash
kubectl get svc -n payments
kubectl describe svc notification-service -n payments
kubectl describe svc notification-service-external -n payments
```

## Полезно для отладки

```bash
kubectl logs -n payments -l app=notification-service --tail=200
kubectl get events -n payments --sort-by=.metadata.creationTimestamp
```
