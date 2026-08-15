# Ark IAM 设计文档索引

本目录存放 Ark IAM（统一身份认证与访问管理服务）的设计与使用文档，覆盖 **SSO/OIDC 协议概念**、**系统总体设计**、**API 参考**、**应用接入**、**配置与部署** 等主题。

> 文档中的图形优先使用 [Mermaid](https://mermaid.js.org/) 绘制（GitHub / GitLab 等平台可直接渲染），少数场景使用 ASCII 图。

## 快速导航

| 读者 / 场景 | 想了解什么 | 阅读文档 |
|---|---|---|
| 新同学、整体认知 | IAM 是什么、SSO/OIDC 是什么、总体架构 | [sso-oidc-concepts.md](sso-oidc-concepts.md) → [system-design.md](system-design.md) |
| 产品 / 架构评审 | 背景、架构、数据模型、核心流程、安全 | [system-design.md](system-design.md) 全篇 |
| 业务应用（RP）开发 | 如何把新应用接入 SSO / OIDC | [application-integration-guide.md](application-integration-guide.md) |
| 前端开发对接 | 登录页 / 管理台调用哪些接口、OIDC 端点契约 | [api-reference.md](api-reference.md) |
| 后端开发 | 路由规范、接口清单、权限模型 | [api-reference.md](api-reference.md) |
| 部署运维 | 配置项含义、构建运行、Docker | [run-and-deploy.md](run-and-deploy.md) |
| 快速查词 | 术语、缩写、角色定义 | [glossary.md](glossary.md) |

## 文档清单

| 文档 | 职责 |
|---|---|
| [sso-oidc-concepts.md](sso-oidc-concepts.md) | **SSO 与 OIDC 概念、协议、流程说明**（建议先读）。回答：什么是 SSO、OIDC 为什么出现、核心概念（Client / Authorization Server / Token / Scope / Claims / 授权码+PKCE / 刷新令牌 / 单点登出）、协议端点与令牌生命周期 |
| [system-design.md](system-design.md) | **系统设计文档（核心主文档）**。背景与目标、总体架构、应用划分（auth / platformadmin / tenantadmin / gateway）、技术栈、数据库设计（ER 图 + 表说明）、核心业务流程（注册 / 登录 / SSO / 登出 / 授权 / 令牌签发）、安全设计、演进方向 |
| [application-integration-guide.md](application-integration-guide.md) | **新应用接入指南**。从零把业务应用接入 IAM：前置准备、创建应用与 OAuth 客户端、RP 侧 OIDC 配置示例、SSO 单点登录体验、单点登出接入、API Key / client_credentials 机器凭证接入、验收清单 |
| [api-reference.md](api-reference.md) | **API 参考**。四类端点总览（OIDC 协议端点 / auth 认证端点 / platform 平台管理端点 / tenant 租户自服务端点）、路由规范摘要、认证与鉴权方式、通用响应信封 |
| [configuration-reference.md](configuration-reference.md) | **配置参考**。各应用 `config.yaml` 全量配置项说明（server / log / trace / db / redis / security / oidc / jwt） |
| [run-and-deploy.md](run-and-deploy.md) | **运行与部署**。本地开发环境准备、构建运行、测试、Docker、多环境关键配置与安全注意事项 |
| [glossary.md](glossary.md) | **术语表**。SSO / OIDC / IAM / 租户 / 自然人 / 应用 / Client / 令牌等术语的统一定义 |

## 核心概念速览

- **Ark IAM** 是一个多租户、多应用的统一身份认证服务，对外提供 **OIDC Provider**（基于 [zitadel/oidc](https://github.com/zitadel/oidc)）能力，业务应用作为 **RP（Relying Party）** 通过标准 OIDC 协议接入，实现 **一次登录、处处通行（SSO）** 与 **一处登出、处处登出（SLO）**。
- 后端由 4 个应用组成：`auth`（认证网关 / OIDC Provider，:8081）、`platformadmin`（平台管理，:8082）、`tenantadmin`（租户自服务，:8083）、`gateway`（单体聚合部署，:8100），共享 `backend/pkg` 公共层，Go workspace 管理。
- 身份模型为 **自然人（person）+ 租户成员（user）**：person 是跨租户的全局身份，user 是 person 在某个租户内的成员记录，一个人可同时属于多个租户。
- 前端为 React 18 + Vite + Ant Design 5 的 pnpm monorepo，包含登录门户（login-web）、平台管理台（platform-admin-web）、租户管理台（tenant-admin-web）。

## 目录约定

本文档面向中文读者，术语首次出现时给出英文原文。所有 mermaid 图使用 `mermaid` 代码块包裹；无 mermaid 渲染环境时，图下方附简要文字说明。
