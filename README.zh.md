[English](./README.md) | [简体中文](./README.zh.md)

# 项目概述

`ark-iam` 是一个前后端分离的全栈项目，后端基于 [Gin](https://github.com/gin-gonic/gin)，提供分层、可维护、可扩展的服务结构，支持多个应用模块。

IAM 后端拆分为 `backend/apps/` 下的四个应用（auth、platformadmin、tenantadmin、gateway），共享公共层 `backend/pkg`，以 Go workspace（`backend/go.work`）管理。

---

# 项目特点

* **清晰的项目结构**: 借鉴 [project-layout](https://github.com/golang-standards/project-layout)，遵循分层架构原则，适合团队协作和长期维护。
* **前后端分离**: React 前端 + Vite 构建工具。
* **通用组件集成**: 后端内置 MySQL、Redis 和 Elasticsearch 示例。
* **全链路日志**: 基于 `zap` 的自定义日志包 `glog`，支持 MySQL、Redis、ES 和 HTTP 调用的全链路 trace ID 传播。
* **代码生成工具**: 配套命令行工具 `gocli`，可根据配置生成标准化代码（包括 model、dao、object、dto、code、service、controller、router 各层）。
* **Swagger API 文档**: 使用 `swaggo` 自动生成交互式 API 文档，方便前后端协作和测试。
* **Docker 支持**: 包含基础的 `Dockerfile` 用于容器化部署。
* **Makefile 工具链**: 提供丰富的 make 命令，简化代码构建、运行、生成、Swagger 文档和 Docker 部署。
* **不断完善的 Golib 库**: 通用工具组件通过 [golib](https://github.com/morehao/golib) 包抽象和复用。

---

# 项目结构

```
ark-iam/
├── backend/               # Go 后端 (基于 project-layout, go.work 多模块)
│   ├── apps/
│   │   ├── auth/          # 认证网关（登录/注册/token/OIDC）, :8081
│   │   ├── platformadmin/ # 平台管理（user/role/menu/tenant）, :8082
│   │   ├── tenantadmin/   # 租户自服务（organization/orgRole）, :8083
│   │   └── gateway/       # 聚合应用，挂载上述三个, :8100
│   ├── pkg/               # 公共包（多应用共享）
│   ├── scripts/           # 脚本
│   └── Makefile
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