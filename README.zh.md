[English](./README.md) | [简体中文](./README.zh.md)

# 项目概述

`ark-iam` 是一个前后端分离的全栈项目，后端基于 [Gin](https://github.com/gin-gonic/gin)，提供分层、可维护、可扩展的服务结构，支持多个应用模块。

IAM 后端拆分为 `backend/apps/` 下的四个应用（auth、platformadmin、tenantadmin、gateway）。其中 auth/platformadmin/tenantadmin 是**可 import 的库包**（各带 `cmd/` 进程入口，保留分体部署）；gateway 是**纯 `cmd/main`** 聚合器（绝不被 import）。原只读目录应用 `rpapi` 已下线——其能力并入 `sdk/server` 的 `DomainDirectory` 域（路由仍在 `/v1/rp/directory/*`）。可复用的能力放在**仓库根级**作为独立 Go 模块——`contract/`（框架无关的数据契约模块）、`pkg/`（公共包）、`sdk/`（RP 侧 SDK + `sdk/server` 门面，委托 apps 各域库）——统一由根 `go.work` 管理，并可各自通过 `go get` 引依赖使用。

> **分层倒置（单向 DAG）**：进程入口 → `sdk/server` → `sdk/server/internal` → `apps/*` 各域库 → `pkg` → `contract`。`sdk/rp` 与 `apps/*` 只依赖 `contract`；`pkg` 只依赖 `contract`（验签经 `contract.Verifier`/`contract.KeySource` 接口，由 `sdk/rp` 实现、`sdk/server/internal` 注入）——因此不存在 `pkg → sdk`、也不存在 `apps → sdk` 依赖边。原独立 `iam/` 模块已删除，域实现全部并入 `apps/`；Go 侧不再托管前端静态资源（`MountFrontend` 已删），登录/管理台前端由 npm 包 `@morehao/ark-iam-react` 提供。

---

# 项目特点

* **清晰的项目结构**: 借鉴 [project-layout](https://github.com/golang-standards/project-layout)，遵循分层架构原则，适合团队协作和长期维护。
* **前后端分离**: React 前端 + Vite 构建工具。
* **通用组件集成**: 后端内置 PostgreSQL、Redis 和 Elasticsearch 示例。
* **全链路日志**: 基于 `zap` 的自定义日志包 `glog`，支持 PostgreSQL、Redis、ES 和 HTTP 调用的全链路 trace ID 传播。
* **代码生成工具**: 配套命令行工具 `gocli`，可根据配置生成标准化代码（包括 model、dao、object、dto、code、service、controller、router 各层）。
* **Swagger API 文档**: 使用 `swaggo` 自动生成交互式 API 文档，方便前后端协作和测试。
* **Docker 支持**: 包含基础的 `Dockerfile` 用于容器化部署。
* **Makefile 工具链**: 提供丰富的 make 命令，简化代码构建、运行、生成、Swagger 文档和 Docker 部署。
* **不断完善的 Golib 库**: 通用工具组件通过 [golib](https://github.com/morehao/golib) 包抽象和复用。

---

# 项目结构

```
ark-iam/
├── go.work                # 根 Go workspace（contract/ pkg/ sdk/ + backend/apps/*）
├── contract/              # 框架无关的数据契约模块（github.com/morehao/ark-iam/contract）：claims / Identity / KeySource+Verifier 接口 / 目录 DTO
├── pkg/                   # 公共包模块（github.com/morehao/ark-iam/pkg）— 仅依赖 contract
├── sdk/                   # RP 侧 SDK 模块（github.com/morehao/ark-iam/sdk）— 唯一 go get 目标
│   ├── rp/                # RP 客户端：验签 / M2M / introspect / revoke / userinfo / 登出 / 目录（实现 contract.Verifier/KeySource）
│   └── server/            # 薄门面：New(Domains)/Mount/MountConfig/RunMigrations；公开 API 只用 contract 类型
│       └── internal/      # 唯一 import apps/* + pkg 的地方；装配验签器、DB/Redis、迁移
├── backend/               # Go 后端 (基于 project-layout, go.work 多模块)
│   ├── apps/
│   │   ├── auth/          # 认证域库 + cmd（登录/会话/OIDC）, :8081
│   │   ├── platformadmin/ # 平台域库 + cmd（user/role/menu/tenant）, :8082
│   │   ├── tenantadmin/   # 租户域库 + cmd（department/user/role）, :8083
│   │   └── gateway/       # 纯 cmd 聚合器：server.New(Domains=nil) + MountDirectory, :8100
├── frontend/              # React 前端 (Vite + React)
├── docs/                  # 文档目录
├── Makefile               # 根目录 Makefile
├── AGENTS.md              # AI 代理开发规范
└── README.md
```

---

# 核心功能

## 后端

### 代码生成

安装 CLI 工具：

```bash
go install github.com/morehao/gocli@latest
```

确保应用目录下存在 `code_gen.yaml` 配置文件，例如 `backend/apps/auth/config/code_gen.yaml`。

运行代码生成命令：

```bash
# 根据表生成完整模块
make codegen APP=auth COMMAND=module

# 仅生成 model 代码
make codegen APP=auth COMMAND=model

# 生成 API 端点代码
make codegen APP=auth COMMAND=api
```

详细文档请参阅 [generate](https://github.com/morehao/gocli?tab=readme-ov-file#generate)。

### API 文档

安装 Swagger 工具：

```bash
go install github.com/swaggo/swag/cmd/swag@latest
```

生成 Swagger 文档：

```bash
make swag APP=auth
```

访问文档（开发模式）：

```
http://localhost:8081/auth/redocs
```

### 项目部署

构建 / 运行 / 测试指定应用。有效 `APP` 取值为 `auth | platformadmin | tenantadmin | gateway`（共享层 `pkg` 无需单独构建）：

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

应用端口映射：

| 应用 | 端口 |
|------|------|
| auth | 8081 |
| platformadmin | 8082 |
| tenantadmin | 8083 |
| gateway（聚合） | 8100 |

所有应用统一使用 `/v1/{模块}/{操作}` 路由前缀（不含应用名段），OIDC Provider 端点为 `/oidc/*`（在 auth / gateway 上）。

构建 Docker 镜像：

```bash
make docker-build APP=gateway
```

运行容器：

```bash
make docker-run APP=gateway
```

### 快速脚手架新后端项目

安装 `cutter` 工具：

```bash
go install github.com/morehao/gocli@latest
```

在**模板项目根目录（例如 `./`）**下运行：

```bash
gocli cutter -d /goProject/yourAppName
```

这将在 `/goProject` 下创建一个基于当前模板的新项目 `yourAppName`。

更多用法请参阅 [cutter](https://github.com/morehao/gocli)。

---

## 相关库

所有后端相关组件都在 [golib](https://github.com/morehao/golib) 包中实现。