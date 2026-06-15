# job-service-go

> Scheduled job runner skeleton for Go 1.25. Connects to `backend-go` to
> consume PostgreSQL, Redis, and RabbitMQ.

A clean, idiomatic boilerplate for running cron-scheduled jobs in Go — no HTTP
layer, no auth, no audit, no PDF, no storage. Just jobs.

## Stack

- **Go 1.25** (toolchain auto-download if older)
- **PostgreSQL** via `github.com/jackc/pgx/v5/pgxpool` (no ORM, no migrations)
- **Redis** via `github.com/redis/go-redis/v9`
- **RabbitMQ** via `github.com/rabbitmq/amqp091-go`
- **Cron** via `github.com/robfig/cron/v3`
- **Logging** via `log/slog` (stdlib)
- **Tests** via `github.com/stretchr/testify` + `alicebob/miniredis/v2`

## Architecture

```
cmd/
└── jobservice/main.go              # Worker bootstrap (DI + graceful shutdown)
internal/
├── core/
│   ├── job.go                      # BaseJob interface + JobContext/JobResult
│   ├── scheduler.go                # IHostedService equivalent (goroutine-based)
│   └── cronadapter.go              # RobfigAdapter (5-field cron)
├── infra/
│   ├── database/pgx_provider.go    # Singleton pgxpool.Pool
│   ├── redis/redis_provider.go     # Singleton go-redis Client
│   ├── messaging/rabbit_provider.go # Publisher + IsOpen() check
│   └── health/                     # DefaultHealthChecker (PG/Redis/Rabbit)
├── jobs/
│   ├── health_check_job.go         # Example: status a cada minuto
│   └── register_jobs.go            # Central registration
└── shared/
    ├── config/env.go               # AppSettings + env loaders
    ├── errors/errors.go            # AppError hierarchy
    └── utils/                      # logger, shutdown, signals
```

## Quick start

### 1. Subir infra local

```bash
make infra-up
```

### 2. Configurar `.env`

```bash
cp .env.example .env
# editar DATABASE_URL, RABBIT_URL, etc.
```

### 3. Rodar em dev

```bash
make dev
```

### 4. Adicionar um job

```go
// internal/jobs/cleanup_job.go
package jobs

import (
    "context"
    "job-service-go/internal/core"
    "job-service-go/internal/shared/config"
    "log/slog"
)

type CleanupJob struct {
    enabled bool
}

func NewCleanupJob(settings *config.AppSettings) *CleanupJob {
    return &CleanupJob{enabled: true}
}

func (j *CleanupJob) Name() string        { return "cleanup" }
func (j *CleanupJob) Schedule() string     { return "0 3 * * *" }
func (j *CleanupJob) Description() string { return "Remove registros antigos" }
func (j *CleanupJob) Enabled() bool        { return j.enabled }

func (j *CleanupJob) Run(ctx context.Context, jc core.JobContext) error {
    jc.Logger.Info("running cleanup")
    // ... sua lógica aqui
    return nil
}
```

```go
// internal/jobs/register_jobs.go
func RegisterJobs(checker health.IHealthChecker, settings *config.AppSettings) []core.BaseJob {
    return []core.BaseJob{
        NewHealthCheckJob(checker, settings),
        NewCleanupJob(settings),
    }
}
```

## Comandos

```bash
make dev          # GOTOOLCHAIN=go1.25.0 go run ./cmd/jobservice
make test         # go test ./... -count=1
make coverage     # go test ./... -coverprofile=coverage.out
make lint         # go vet ./...
make check        # lint + test
make build        # CGO_ENABLED=0 go build -o bin/jobservice
make docker       # build image
make run          # go run
make infra-up     # docker compose up (PG+Redis+Rabbit)
make infra-down   # docker compose down
make sonar        # SonarQube scan
make clean
```

## Configuração (env vars)

| Var | Default | Descrição |
|---|---|---|
| `ENVIRONMENT` | `local` | dev / staging / production |
| `LOG_LEVEL` | `Information` | slog level (debug/info/warn/error) |
| `SHUTDOWN_TIMEOUT_SECONDS` | `30` | Max wait for cleanup on SIGTERM |
| `JOB_EXECUTION_TIMEOUT_SECONDS` | `300` | Per-job timeout |
| `DATABASE_URL` | (required) | PostgreSQL DSN |
| `DATABASE_COMMAND_TIMEOUT_SECONDS` | `10` | SELECT timeout |
| `REDIS_URL` | `redis://localhost:6379/0` | URL (preferred) |
| `REDIS_HOST` / `REDIS_PORT` | `localhost` / `6379` | (if no URL) |
| `REDIS_PASSWORD` | (empty) | |
| `REDIS_DB` | `0` | |
| `MESSAGING_ENABLED` | `false` | Enable RabbitMQ publisher |
| `RABBIT_URL` | `amqp://guest:guest@localhost:5672/` | |
| `RABBIT_USER` / `RABBIT_PASSWORD` | `guest` | |
| `RABBITMQ_PUBLISH_TIMEOUT` | `5` | seconds |
| `HEALTH_CHECK_CRON` | `*/1 * * * *` | 5-field cron |
| `HEALTH_CHECK_ENABLED` | `true` | Disable health check |

## Princípios

- **S** Single Responsibility — cada job tem um único propósito
- **O** Open/Closed — adicionar um job = criar um struct + 1 linha de registro
- **L** Liskov Substitution — todo `BaseJob` é intercambiável
- **I** Interface Segregation — dependências injetadas via construtor
- **D** Dependency Inversion — jobs dependem de abstrações (`health.PgPinger`, `health.RabbitChecker`), não de implementações
- **DRY** — lógica compartilhada fica em `ExecuteJob`
- **Clean Code** — nomes expressivos, funções curtas, sem side-effects

## Testes

```bash
GOTOOLCHAIN=go1.25.0 go test ./... -coverprofile=coverage.out
```

Coverage (statements):
- `core` — 92.2%
- `infra/database` — 45.5% (success path needs real PG)
- `infra/health` — 95.2%
- `infra/messaging` — 74.6% (test seam for Conn/Channel)
- `infra/redis` — 90.0%
- `jobs` — 100%
- `shared/config` — 94.7%
- `shared/errors` — 100%
- `shared/utils` — 79.2%

## CI

GitHub Actions roda em push/PR para `develop` e `main`:
- `go vet ./...`
- `go build ./...`
- `go test ./...` com coverage
- Coverage gate: ≥85% line

## License

Same as the parent `mage-boilerplates` repository.
