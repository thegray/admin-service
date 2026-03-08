# Go Project Template
## Template from https://github.com/thegray/go-project-template
This template implements a **Package-Oriented Architecture**. It prioritizes Go idioms over rigid enterprise patterns, following the lead of major projects like **Kubernetes** and **Terraform**.


## Core Philosophy
- **Accept Interfaces, Return Structs:** We define interfaces where they are consumed (in ports.go), not where they are implemented.
- **Encapsulation over Layering:** Logic is grouped by business domain (e.g., user, order) rather than technical role (service, controller).
- **Dependency Orchestration:** Cross-package logic is handled at the api layer to prevent circular dependencies.


## Project Structure 
```text
/myapp  
    ├── cmd/  
    │    └── server/
    |         └── main.go        // App entry point & dependency injection (wiring)  
    ├── internal/  
    │    ├── domain/               // Shared DTOs and Entities (prevents circular deps)  
    │    │    ├── user.go  
    │    │    └── order.go  
    │    ├── user/                 // Domain-specific package  
    │    │    ├── service.go        // Core business logic (Concrete Structs)  
    │    │    ├── service_test.go  
    │    │    ├── ports.go          // Requirements defined as Interfaces  
    │    │    └── repository/       // Infrastructure implementations (Postgres, Mock, etc.)  
    │    │        └── pg.go  
    │    └── order/  
    ├── pkg/                        // Reusable library code (Logger, Auth, Crypto)  
    └── api/                        // Transport layer (REST Handlers, gRPC, Middleware)  
         └── rest/  
               └── user_handler.go   // Orchestrates calls between domain services  
```


## Key Design Rules
**1. Avoiding Circular Dependencies**
To prevent the common ```import cycle not allowed``` error:
**A. Shared Types:** All structs used by more than one package live in ```internal/domain```.
**B. Orchestration:** If a feature requires calling both ```user``` and ```order``` services, that logic lives in the ```api/rest``` handler or a dedicated orchestrator.

**2. Mocking and Testability**
Testability is achieved through ```Constructor Injection```. Each service in ```internal/``` defines its dependencies as interfaces in ```ports.go```. 
During testing, you simply pass a mock implementation into the service constructor.

**3. The ```internal/``` Boundary**
Code inside ```internal/``` cannot be imported by any code outside this project. 
This ensures your core business logic remains private and cannot be "leaked" into external tools or libraries.

### Code Example for ```Orchestration```
This example demonstrates a **"Checkout"** flow. The ```api``` layer orchestrates the logic, keeping the ```user``` and ```order``` packages decoupled.

#### 1. The Service (internal/order/service.go)
Each service is "pure" and only manages its own domain logic.
```Go
package order

import (
    "context"
    "myproject/internal/domain"
)

type Service struct {
    repo OrderRepository
}

func (s *Service) CreateOrder(ctx context.Context, userID int, items []domain.Item) (*domain.Order, error) {
    // Domain-specific logic only
    return s.repo.Save(ctx, &domain.Order{UserID: userID, Items: items})
}
```

#### 2. The Orchestrator (api/rest/order_handler.go)
The handler imports multiple services to coordinate a multi-step workflow.
```Go
package rest

import (
    "myproject/internal/user"
    "myproject/internal/order"
)

type OrderHandler struct {
    userSvc  *user.Service
    orderSvc *order.Service
}

func (h *OrderHandler) HandleCheckout(w http.ResponseWriter, r *http.Request) {
    ctx := r.Context()
    userID := getIDFromToken(r)

    // 1. check user status (call User Domain)
    u, _ := h.userSvc.GetProfile(ctx, userID)
    if u.IsBanned {
        http.Error(w, "User is banned", http.StatusForbidden)
        return
    }

    // 2. create the order (call Order Domain)
    newOrder, _ := h.orderSvc.CreateOrder(ctx, userID, items)

    // 3. respond
    renderJSON(w, newOrder)
}
```

## Runtime configuration

| Variable | Default | Purpose |
|----------|---------|---------|
| `PORT` | `8000` | Defaults to `config.yaml`; override to change the listener |
| `APP_ENV` | `development` | `config.yaml` default; production/staging routes secrets through Google Secret Manager |
| `LOG_LEVEL` | `info` | Override the zap level (debug/info/warn/error) |
| `TOKEN_SECRET` | `dev-default-secret` | Local fallback; production/staging pulls from Secret Manager |
| `TOKEN_SECRET_NAME` | `projects/<GCP_PROJECT>/secrets/token-secret/versions/latest` | Optional override when Secret Manager path differs |
| `RATE_LIMIT_RPS` | `5` | Requests per second for auth endpoints |
| `RATE_LIMIT_BURST` | `10` | Burst size for the rate limiter |
| `READ_TIMEOUT_SECONDS` | `5` | Base HTTP read timeout (seconds) defined in `config.yaml`, override via ConfigMap/env |
| `WRITE_TIMEOUT_SECONDS` | `10` | HTTP write timeout |
| `IDLE_TIMEOUT_SECONDS` | `120` | HTTP idle timeout |
| `SHUTDOWN_TIMEOUT_SECONDS` | `5` | Graceful shutdown window used by `pkg/server` |
| `GCP_PROJECT` | n/a | Required when `APP_ENV` is `staging` or `production` so the Secret Manager client can resolve secrets. |

When `APP_ENV` resolves to `staging` or `production`, runtime secrets are fetched from GCP Secret Manager using the `TOKEN_SECRET_NAME` (or a sane default derived from `TOKEN_SECRET`). Local development continues to read `TOKEN_SECRET` from the environment or falls back to the `dev-default-secret`.

## Observability

- All Gin requests run through a trace middleware that emits `X-Trace-ID` and zaps each span along with status/duration.
- Prometheus metrics are exposed at `/metrics` through `pkg/observability`, including request counters and durations.
- Rate limiting binaries log `Too Many Requests` responses when limits are exceeded and still expose trace IDs for investigation.

## Domain implementation

- The current services under `internal/domain/example` and the in-memory repository are illustrative examples that exercise the API surface.

## Docker Compose (development)

1. Build and run the stack (dev-only compose helper):

   ```sh
   cp .env.example .env    # keep secrets out of version control
   docker compose -f docker-compose.local.yml up --build
   ```

2. Compose reads `.env` via `env_file`, so local overrides stay outside the image while the container reuses the `config.yaml` defaults.

3. Do NOT use this compose setup in production; Kubernetes manifests live outside this repo and supply overrides via ConfigMaps/Secrets plus Secret Manager for sensitive data.
