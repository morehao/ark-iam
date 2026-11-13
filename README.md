[English](./README.md) | [简体中文](./README.zh.md)

# Project Overview

`ark-iam` is a full-stack project with Go backend and React frontend. The backend is based on [Gin](https://github.com/gin-gonic/gin), providing a layered, maintainable, and scalable service structure with multiple app modules.

The IAM backend is split into four apps under `apps/` (auth, platformadmin, tenantadmin, gateway). `auth`/`platformadmin`/`tenantadmin` are importable **library packages** (each with a `cmd/` process entry for standalone deployment); `gateway` is a **pure `cmd/main`** aggregator that is never imported. The former read-only directory app `rpapi` has been retired — its capability now lives in the `sdk/server` `DomainDirectory` domain (still served at `/v1/rp/directory/*`). The reusable layers live at the repository root as standalone Go modules — `contract/` (the framework-free data-contract module), `pkg/` (common packages), `sdk/` (RP-side SDK plus the `sdk/server` facade that orchestrates the domain libraries) — all managed by the root `go.work` and individually consumable via `go get`.

> **Layered inversion (single-direction DAG)**: process entry → `sdk/server` → `sdk/server/internal` → `apps/*` domain libraries → `pkg` → `contract`. `sdk/rp` and `apps/*` depend only on `contract`; `pkg` depends only on `contract` (its verification path goes through the `contract.Verifier`/`contract.KeySource` interfaces, implemented by `sdk/rp` and injected by `sdk/server/internal`) — so there is no `pkg → sdk` and no `apps → sdk` edge. The former standalone `iam/` module has been removed; its domain implementations now live in `apps/`, and Go-side frontend embedding (`MountFrontend`) is gone — the login/admin frontend is served from the npm package `@morehao/ark-iam-react`.

---

# Features

* **Clear Project Structure**: Inspired by [project-layout](https://github.com/golang-standards/project-layout), follows layered architecture principles, organized for team collaboration and long-term maintenance.
* **Frontend-Backend Separation**: React frontend with Vite build tool.
* **Common Component Integration**: Backend includes built-in examples for PostgreSQL, Redis, and Elasticsearch.
* **Full Link Logging**: Provides a custom logging package `glog` based on `zap`, supporting full trace ID propagation across PostgreSQL, Redis, ES, and HTTP calls.
* **Code Generation Tool**: Comes with a command-line tool `gocli` that can generate standardized code (including model, dao, object, dto, code, service, controller, router layers) based on config.
* **Swagger API Documentation**: Automatically generate interactive API docs using `swaggo` for easier frontend-backend collaboration and testing.
* **Docker Support**: Includes a basic `Dockerfile` for containerized deployment.
* **Makefile Toolchain**: Provides a rich set of make commands to simplify code build, run, generation, Swagger docs, and Docker deployment.
* **Growing Golib Library**: Common utility components are abstracted and reusable via the [golib](https://github.com/morehao/golib) package.

---

# Project Structure

```
ark-iam/
├── apps/                  # All apps (Go backend + React frontend), flattened to the repo root
│   ├── auth/              # Auth domain library + cmd (login/session/OIDC), :8081
│   ├── platformadmin/     # Platform domain library + cmd (user/role/menu/tenant), :8082
│   ├── tenantadmin/       # Tenant domain library + cmd (department/user/role), :8083
│   ├── gateway/           # Pure cmd aggregator: server.New(Domains=nil) + MountDirectory, :8100
│   ├── login-web/         # Login portal frontend, :4000
│   ├── platform-admin-web/# Platform admin console frontend, :4001
│   └── tenant-admin-web/  # Tenant admin console frontend, :4002
├── packages/              # Shared frontend packages (pnpm workspace: types/api/auth/ui)
├── go.work                # Root Go workspace (covers contract/ pkg/ sdk/ + apps/*)
├── contract/              # Framework-free data-contract module (github.com/morehao/ark-iam/contract): claims / Identity / KeySource+Verifier interfaces / directory DTOs
├── pkg/                   # Common packages module (github.com/morehao/ark-iam/pkg) — depends only on contract
├── sdk/                   # RP-side SDK module (github.com/morehao/ark-iam/sdk) — the single go get target
│   ├── rp/                # RP client: verify / M2M / introspect / revoke / userinfo / logout / directory (implements contract.Verifier/KeySource)
│   └── server/            # Thin facade: New(Domains)/Mount/MountConfig/RunMigrations; public API uses contract types only
│       └── internal/      # The only place that imports apps/* + pkg; wires the verifier, DB/Redis, migrations
├── examples/              # Minimal embeddable-IAM example project
├── e2e/                   # End-to-end tests
├── docs/                  # Documentation
├── config/                # Configuration
├── package.json           # Frontend workspace root package (pnpm)
├── pnpm-workspace.yaml    # pnpm workspace config
├── Makefile               # Root Makefile
├── AGENTS.md              # Development guide for AI agents
└── README.md
```

---

# Core Features

## Backend

### Code Generation

Install the CLI tool:

```bash
go install github.com/morehao/gocli@latest
```

Ensure a `code_gen.yaml` config file exists under the application directory, e.g., `apps/auth/config/code_gen.yaml`.

Run code generation commands:

```bash
# Generate full module based on table
make codegen APP=auth COMMAND=module

# Generate only model code
make codegen APP=auth COMMAND=model

# Generate API endpoint code
make codegen APP=auth COMMAND=api
```

See [generate](https://github.com/morehao/gocli?tab=readme-ov-file#generate) for full documentation.

### API Documentation

Install Swagger tool:

```bash
go install github.com/swaggo/swag/cmd/swag@latest
```

Generate Swagger docs:

```bash
make swag APP=auth
```

Access docs at (dev mode):

```
http://localhost:8081/auth/redocs
```

### Project Deployment

Build / run / test an app. Valid `APP` values are `auth | platformadmin | tenantadmin | gateway` (the shared `pkg` layer needs no app build):

```bash
# 列出所有可用应用
make list-apps

# 构建指定应用
make build APP=auth
make build APP=gateway

# 运行指定应用（开发调试）
make run APP=auth
make run APP=platformadmin
make run APP=tenantadmin
make run APP=gateway     # 单进程聚合 auth + platformadmin + tenantadmin

# 运行指定应用的测试
make test APP=gateway
```

App-port mapping:

| App | Port |
|-----|------|
| auth | 8081 |
| platformadmin | 8082 |
| tenantadmin | 8083 |
| gateway (aggregate) | 8100 |

Routes are namespaced per app as `/v1/{service}/{module}/{operation}` (`service`: `auth` / `platform` / `tenant`), plus the OIDC provider prefix `/oidc/*` on auth/gateway.

Build Docker image:

```bash
make docker-build APP=gateway
```

Run container:

```bash
make docker-run APP=gateway
```

### Quickly Scaffold a New Backend Project

Install the `cutter` tool:

```bash
go install github.com/morehao/gocli@latest
```

Run under **the root of the template project (e.g., `./`)**:

```bash
gocli cutter -d /goProject/yourAppName
```

This will scaffold a new project named `yourAppName` under `/goProject` based on the current template.

See [cutter](https://github.com/morehao/gocli) for more usage details.

---

## Related Libraries

All related backend components are implemented in the [golib](https://github.com/morehao/golib) package.