# Ark IAM 设计文档索引

本目录存放 Ark IAM（统一身份认证与访问管理服务）的设计与使用文档，覆盖 **SSO/OIDC 协议概念**、**系统总体设计**、**API 参考**、**应用接入**、**配置与部署** 等主题。

> 图形优先使用 [Mermaid](https://mermaid.js.org/) 绘制（GitHub / GitLab 等平台可直接渲染），少数场景使用 ASCII 图。

## 快速导航

| 读者 / 场景 | 想了解什么 | 阅读文档 |
|---|---|---|
| 新同学、整体认知 | IAM 是什么、SSO/OIDC 是什么、总体架构 | [sso-oidc-concepts.md](sso-oidc-concepts.md) → [system-design.md](system-design.md) |
| 产品 / 架构评审 | 背景、架构、数据模型、核心流程、安全 | [system-design.md](system-design.md) 全篇 |
| 业务应用（RP）开发 | 如何把新应用接入 SSO / OIDC | [application-integration-guide.md](application-integration-guide.md) |
| 前端开发对接 | 登录页 / 管理台调用哪些接口、OIDC 端点契约 | [api-reference.md](api-reference.md) |
| 后端开发 | 路由规范、接口清单、权限模型、种子与字段权威 | [api-reference.md](api-reference.md) → [system-design.md](system-design.md) §4.5 |
| 部署运维 | 配置项含义、构建运行、Docker | [configuration-reference.md](configuration-reference.md) + [run-and-deploy.md](run-and-deploy.md) |
| 快速查词 | 术语、缩写、角色定义 | [glossary.md](glossary.md) |

## 文档清单

| 文档 | 职责 |
|---|---|
| [sso-oidc-concepts.md](sso-oidc-concepts.md) | **SSO 与 OIDC 概念、协议、流程说明**（建议先读）。回答：什么是 SSO、OIDC 为什么出现、核心概念（Client / Authorization Server / Token / Scope / Claims / 授权码+PKCE / 刷新令牌 / 单点登出）、协议端点与令牌生命周期 |
| [system-design.md](system-design.md) | **系统设计文档（核心主文档，单一事实源）**。背景与目标、总体架构、应用划分（auth / platformadmin / tenantadmin / gateway）与内部分层、技术栈、数据库设计（ER 图 + 表说明 + Redis Key + **种子数据与字段权威矩阵**）、核心业务流程（自助开通租户 / 登录 / SSO / 登出 / 授权 / 令牌签发 / Connector / **建租户与内置管理员**）、路由规范、安全设计、演进方向 |
| [application-integration-guide.md](application-integration-guide.md) | **新应用接入指南**。从零把业务应用接入 IAM：前置准备、创建应用与 OAuth 客户端、RP 侧 OIDC 配置示例、SSO 单点登录体验、单点登出接入、API Key / client_credentials 机器凭证接入、验收清单 |
| [api-reference.md](api-reference.md) | **API 参考**。四类端点总览（OIDC 协议端点 / auth 认证端点 / platform 平台管理端点 / tenant 租户自服务端点）、认证与鉴权方式、通用响应信封、路由规范摘要 |
| [configuration-reference.md](configuration-reference.md) | **配置参考**。各应用 `config.yaml` 全量配置项说明（server / log / trace / db_configs / redis_config / security / oidc / jwt / client / password / es_configs / masterKey） |
| [run-and-deploy.md](run-and-deploy.md) | **运行与部署**。本地开发环境准备、构建运行、测试、Docker、多环境部署拓扑、常见排障 |
| [glossary.md](glossary.md) | **术语表**。SSO / OIDC / IAM / 租户 / 自然人 / 应用 / Client / 令牌等术语的统一定义 |
| [tenant-custom-domain-redesign.md](tenant-custom-domain-redesign.md) | **租户级自定义域名实施方案（待实施）**。按域名识别租户、登录预设与品牌化、`domain` 表重做。**当前实现状态：`domain` 表与控制台 CRUD 已落地，但 auth 侧尚未按域名解析租户，本文方案的主体仍未实施** |

> **过程性设计稿（spec / plan）不入库**：本仓库用本地 SDD 工作流产出的方案与计划文档统一放在 `docs/superpowers/`（已 gitignore）。已落地的改造不再在 `docs/design/` 留过程文档——结论一律收敛进上表的长期维护文档：字段权威矩阵与种子身份键见 `system-design.md` §4.5、租户开通与内置管理员见 §5.8、共享层分层见 §2.3、schema 变更策略见 §4.1、运行与升级见 `run-and-deploy.md`。

## 核心概念速览

- **Ark IAM** 是一个多租户、多应用的统一身份认证服务，对外提供 **OIDC Provider**（基于 [zitadel/oidc](https://github.com/zitadel/oidc)）能力，业务应用作为 **RP（Relying Party）** 通过标准 OIDC 协议接入，实现 **一次登录、处处通行（SSO）** 与 **一处登出、处处登出（SLO）**。
- 后端由 4 个应用组成：`auth`（认证网关 / OIDC Provider，:8081）、`platformadmin`（平台管理，:8082）、`tenantadmin`（租户自服务，:8083）、`gateway`（单体聚合部署，:8100），共享 `backend/pkg` 公共层，Go workspace（`backend/go.work`）管理 5 个模块。
- 身份模型为 **自然人（person）+ 租户成员（user，表 `tenant_user`）**：person 是跨租户的全局身份，user 是 person 在某个租户内的成员记录，一个人可同时属于多个租户。
- 授权模型到「**角色—菜单**」粒度（角色按租户 + 应用作用域，`user_role` / `role_menu` 两张关联表）；资源级权限不在 IAM 承载。
- 前端为 React 18 + Vite + Ant Design 5 的 pnpm monorepo（`frontend/`），包含登录门户（login-web :4000）、平台管理台（platform-admin-web :4001）、租户管理台（tenant-admin-web :4002）。

## 目录约定

- 文档面向中文读者，术语首次出现时给出英文原文；
- 所有 mermaid 图使用 `mermaid` 代码块包裹；无 mermaid 渲染环境时，图下方附简要文字说明；
- 本目录文档必须与当前代码一致：表格/枚举/端口/默认值以代码为事实源，不从旧文档转抄；
- 路径参数统一写作 `{xxxID}`（与代码里 gin 的 `:xxxID` 对应）。
