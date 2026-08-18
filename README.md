# goph-profile

## Запуск приложения

### Требования

- Docker и Docker Compose
- Go 1.26.2 или новее, если нужен локальный запуск без контейнера
- Свободные порты:
  - `8080` - HTTP API приложения
  - `5435` - PostgreSQL на хосте
  - `9000`, `9001` - MinIO API и консоль
  - `29092`, `9092` - Kafka
  - `3000` - Grafana
  - `9090` - Prometheus
  - `3100` - Loki
  - `4317` - OTLP collector
  - `16686` - Jaeger UI

### Быстрый запуск через Docker Compose

Из корня проекта выполните:

```powershell
docker compose up --build
```

Команда поднимет:

- `goph` - HTTP-сервер приложения
- `goph-outbox` - воркер публикации событий
- `goph-inbox` - воркер обработки событий
- PostgreSQL
- MinIO
- Kafka и Zookeeper
- OpenTelemetry Collector, Prometheus, Loki, Grafana, Jaeger

После запуска приложение будет доступно по адресам:

- Web/API: http://localhost:8080
- Health check: http://localhost:8080/health
- Swagger UI: http://localhost:8080/docs
- OpenAPI spec: http://localhost:8080/openapi.yaml
- MinIO console: http://localhost:9001
- Grafana: http://localhost:3000
- Prometheus: http://localhost:9090
- Jaeger: http://localhost:16686

Данные для MinIO:

```text
login: admin
password: admin12345
```

PostgreSQL с хоста доступен по строке подключения:

```text
postgres://dev:dev@localhost:5435/goph?sslmode=disable
```

Внутри Docker-сети приложение использует:

```text
postgres://dev:dev@psql:5432/goph?sslmode=disable
```

### Проверка запуска

В отдельном терминале выполните:

```powershell
curl http://localhost:8080/health
```

Ожидаемый ответ:

```json
{
  "server": true,
  "db": true,
  "s3": true
}
```

### Остановка

Остановить контейнеры:

```powershell
docker compose down
```

Остановить контейнеры и удалить локальные volumes с данными PostgreSQL, MinIO, Loki и Grafana:

```powershell
docker compose down -v
```

### Локальный запуск без контейнера приложения

Сначала поднимите инфраструктуру:

```powershell
docker compose up -d psql minio minio-init zookeeper kafka kafka-init otel-collector loki prometheus grafana jaeger
```

Затем задайте переменные окружения для локального процесса:

```powershell
$env:GOPH_ADDR=":8080"
$env:GOPH_DATABASE="postgres://dev:dev@localhost:5435/goph?sslmode=disable"
$env:GOPH_BUCKET="goph"
$env:GOPH_REGION="eu-east-1"
$env:GOPH_ENDPOINT="http://localhost:9000"
$env:GOPH_ACCESS_KEY="admin"
$env:GOPH_SECRET_KEY="admin12345"
$env:GOPH_USE_SSL="false"
$env:GOPH_BROKERS="localhost:29092"
$env:OTEL_SERVICE_NAME="goph-server"
$env:OTEL_RESOURCE_ATTRIBUTES="deployment.environment=development"
$env:OTEL_EXPORTER_OTLP_ENDPOINT="localhost:4317"
$env:OTEL_EXPORTER_OTLP_PROTOCOL="grpc"
$env:OTEL_EXPORTER_OTLP_INSECURE="true"
```

Запустите HTTP-сервер:

```powershell
go run ./cmd/server
```

Для запуска воркеров в отдельных терминалах используйте те же переменные окружения и команды:

```powershell
$env:OTEL_SERVICE_NAME="goph-outbox"
go run ./cmd/outbox
```

```powershell
$env:OTEL_SERVICE_NAME="goph-inbox"
go run ./cmd/inbox
```

### Полезные команды разработки

Запуск тестов:

```powershell
go test -v ./...
```

Если установлен `task`, можно использовать команды из `Taskfile.yml`:

```powershell
task go-test
task go-test-coverage
task go-fmt
task go-lint
```

## Запуск через Helm chart

Chart находится в каталоге `goph-profile`.

### Требования

- Kubernetes-кластер
- `kubectl`
- Helm 3
- Доступный StorageClass для PVC PostgreSQL, MinIO, Loki и Grafana
- Vault Agent Injector, если используются стандартные аннотации из `values.yaml`

Chart устанавливает приложение и инфраструктурные зависимости:

- namespace `goph-profile` для приложения
- namespace `postgres` для PostgreSQL
- namespace `kafka` для Kafka и Zookeeper
- namespace `minio` для MinIO
- namespace `monitoring` для Prometheus, Loki, Grafana, Jaeger, OpenTelemetry Collector и node-exporter
- Traefik из dependency chart

### Подготовка dependency chart

Если зависимости ещё не загружены, выполните:

```powershell
helm dependency build .\goph-profile
```

В репозитории уже есть архив `goph-profile/charts/traefik-40.0.0.tgz`, поэтому при обычном локальном запуске этот шаг может быть не нужен.

### Секреты Vault

По умолчанию deployments приложения запускаются через:

```sh
. /vault/secrets/config && exec ./app
```

Поэтому в кластере должен быть настроен Vault Agent Injector, а по пути `secret/data/goph-profile` должны быть значения:

```text
goph_access_key=admin
goph_secret_key=admin12345
goph_database=postgres://postgres:goph@postgres.postgres:5432/goph?sslmode=disable
```

Эти значения используются как переменные окружения:

- `GOPH_ACCESS_KEY`
- `GOPH_SECRET_KEY`
- `GOPH_DATABASE`

Остальные переменные задаются в `goph-profile/values.yaml`.

### Установка chart

Из корня проекта:

```powershell
helm upgrade --install goph-profile .\goph-profile
```

Если нужно переопределить тег образов:

```powershell
helm upgrade --install goph-profile .\goph-profile --set app.tag=v0.0.3-unstable
```

По умолчанию используются образы:

```text
d1dog/goph-server:<app.tag>
d1dog/goph-outbox:<app.tag>
d1dog/goph-inbox:<app.tag>
```

### Проверка статуса

```powershell
kubectl get pods -n goph-profile
kubectl get pods -n postgres
kubectl get pods -n kafka
kubectl get pods -n minio
kubectl get pods -n monitoring
```

Проверить сервис приложения:

```powershell
kubectl get svc -n goph-profile
```

Проверить ingress:

```powershell
kubectl get ingress -n goph-profile
```

Посмотреть логи:

```powershell
kubectl logs -n goph-profile deploy/goph-profile-server
kubectl logs -n goph-profile deploy/goph-profile-outbox
kubectl logs -n goph-profile deploy/goph-profile-inbox
```

### Локальный доступ к приложению

Если ingress недоступен локально, используйте port-forward:

```powershell
kubectl port-forward -n goph-profile svc/goph-profile-server 8080:80
```

После этого:

- Web/API: http://localhost:8080
- Health check: http://localhost:8080/health
- Swagger UI: http://localhost:8080/docs

Если ingress работает, стандартные host из `values.yaml`:

```text
goph-avatar
localhost
```

Для доступа через host `goph-avatar` добавьте локальную DNS-запись, например в `hosts`:

```text
127.0.0.1 goph-avatar
```

### Удаление chart

```powershell
helm uninstall goph-profile
```

Namespaces и PVC могут остаться в кластере. Перед удалением данных проверьте, что они больше не нужны:

```powershell
kubectl get pvc -A
kubectl get ns
```
