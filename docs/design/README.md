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
| 业务应用接入 SSO/RP | 如何作为 RP 接入 OIDC | [oidc-sso-integration.md](oidc-sso-integration.md) |
| 联调验证、问题排查 | 怎么测、怎么排障 | [oidc-verification-guide.md](oidc-verification-guide.md) |

## 文档清单

| 文档 | 职责 |
|---|---|
| [iam-design.md](iam-design.md) | **核心主文档**（建议首读）。系统总体设计：架构、登录态与 token、数据模型、API/OIDC 端点、安全、演进与落地核对清单 |
| [iam_data_model_overview.md](iam_data_model_overview.md) | 按业务域拆解全部数据表：字段含义与典型业务场景 |
| [iam-login-flow.md](iam-login-flow.md) | 覆盖密码登录、SSO、租户选择、注册、API Key、Connector、刷新等全认证流程的时序与代码路径 |
| [oidc-frontend-contract.md](oidc-frontend-contract.md) | 面向前端开发的接口契约：`POST /oidc/login` 请求/响应与消费流程 |
| [oidc-sso-integration.md](oidc-sso-integration.md) | 面向业务应用（RP）的接入指南：OIDC 概念、端点说明、接入步骤、代码示例 |
| [oidc-verification-guide.md](oidc-verification-guide.md) | 联调验证指南：自动化测试、curl 协议验证、浏览器全流程、常见问题排查 |
