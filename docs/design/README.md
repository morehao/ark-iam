# IAM 设计文档索引

本目录存放 IAM（统一认证服务）的落地设计文档。以 [iam-design.md](iam-design.md) 为核心主文档，按读者角色与目的分为不同主题文档。

核心要点：基于 OIDC 标准协议、支持多租户、多应用共享登录态的认证服务。

## 读者导航

| 读者 / 场景 | 想解决 | 阅读文档 |
|---|---|---|
| 新同学、整体认知 | IAM 是什么、总体架构、核心概念 | [iam-design.md](iam-design.md)（先读 §0/§1） |
| 架构评审、技术选型 | 整体设计细节、安全、演进 | [iam-design.md](iam-design.md) 全篇 |
| 数据库建模、SQL 评审 | 表结构、字段含义、业务场景 | [iam_data_model_overview.md](iam_data_model_overview.md) |
| 理解各类登录流程 | 端到端时序、代码路径 | [iam-login-flow.md](iam-login-flow.md) |
| 前端开发对接 | `/oidc/login` 等接口契约 | [oidc-frontend-contract.md](oidc-frontend-contract.md) |
| 后端/前端开发 | 路由怎么写、新接口如何设计 | [api-routing-convention.md](api-routing-convention.md) |
| 路由改造执行 | 旧→新路由对照、分步迁移计划 | [api-route-migration.md](api-route-migration.md) |
| 业务应用接入 SSO/RP | 如何作为 RP 接入 OIDC | [oidc-sso-integration.md](oidc-sso-integration.md) |
| 统一登出 / SLO / back-channel | 登出不同步、标准 Back-Channel Logout、sid、多副本 | [oidc-slo-unified-logout.md](oidc-slo-unified-logout.md) |
| 联调验证、问题排查 | 怎么测、怎么排障 | [oidc-verification-guide.md](oidc-verification-guide.md) |

## 文档清单

| 文档 | 职责 |
|---|---|
| [iam-design.md](iam-design.md) | **核心主文档**（建议首读）。系统总体设计：架构、登录态与 token、数据模型、API/OIDC 端点、安全、演进与落地核对清单 |
| [iam_data_model_overview.md](iam_data_model_overview.md) | 按业务域拆解全部数据表：字段含义与典型业务场景 |
| [iam-login-flow.md](iam-login-flow.md) | 覆盖密码登录、SSO、租户选择、注册、API Key、Connector、刷新等全认证流程的时序与代码路径 |
| [oidc-frontend-contract.md](oidc-frontend-contract.md) | 面向前端开发的接口契约：`POST /oidc/login` 请求/响应与消费流程 |
| [oidc-sso-integration.md](oidc-sso-integration.md) | 面向业务应用（RP）的接入指南：OIDC 概念、端点说明、接入步骤、代码示例 |
| [oidc-slo-unified-logout.md](oidc-slo-unified-logout.md) | **统一登出设计变更**：登出不同步的根因、OP/RP 无状态分层、标准 Back-Channel Logout、sid 会话锚点、auth 多副本共享认证 Redis、分步实施（M1–M5）|
| [oidc-verification-guide.md](oidc-verification-guide.md) | 联调验证指南：自动化测试、curl 协议验证、浏览器全流程、常见问题排查 |
| [api-routing-convention.md](api-routing-convention.md) | **API 路由规范**：规则化混合（R1 资源 CRUD → REST / R2 业务动作 → `:action` / R3 协议端点不动），资源命名、方法语义、子资源与关联建模、认证域边界 |
| [api-route-migration.md](api-route-migration.md) | **路由改造方案**：134 条 `/v1` 路由旧→新对照表、迁移原则、配套变更、分步执行计划与验证方式 |
