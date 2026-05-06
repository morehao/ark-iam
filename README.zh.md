[English](./README.md) | [简体中文](./README.zh.md)

# 项目概述

`ark-iam` 是一个前后端分离的全栈项目，后端基于 [Gin](https://github.com/gin-gonic/gin)，提供分层、可维护、可扩展的服务结构，支持多个应用模块。

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
├── backend/               # Go 后端 (基于 project-layout)
│   ├── apps/
│   │   ├── demo/          # Demo 应用
│   │   └── iam/           # IAM 应用
│   ├── pkg/               # 公共包
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

确保应用目录下存在 `code_gen.yaml` 配置文件，例如 `backend/apps/demo/config/code_gen.yaml`。

运行代码生成命令：

```bash
# 根据表生成完整模块
make codegen APP=demo COMMAND=module

# 仅生成 model 代码
make codegen APP=demo COMMAND=model

# 生成 API 端点代码
make codegen APP=demo COMMAND=api
```

详细文档请参阅 [generate](https://github.com/morehao/gocli?tab=readme-ov-file#generate)。

### API 文档

安装 Swagger 工具：

```bash
go install github.com/swaggo/swag/cmd/swag@latest
```

生成 Swagger 文档：

```bash
make swag APP=demo
```

访问文档（开发模式）：

```
http://localhost:8099/demo/redocs
```

### 项目部署

构建 Docker 镜像：

```bash
make docker-build APP=demo
```

运行容器：

```bash
make docker-run APP=demo
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