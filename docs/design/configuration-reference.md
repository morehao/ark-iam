# 配置参考（Configuration Reference）

> 本文说明 Ark IAM 各应用 `config.yaml` 的配置项。四个应用（auth / platformadmin / tenantadmin / gateway）共享同一套 `pkg/config.Config` 结构，差异主要在 `server.name/port` 与 OIDC 相关项。
>
> 配置加载顺序：环境变量 `APP_CONFIG_PATH` 指定路径 → `../config/config.yaml` → 可执行文件同级 `../config/config.yaml`。

---

## 目录

1. [配置总览](#1-配置总览)
2. [server（服务）](#2-server服务)
3. [log（日志）](#3-log日志)
4. [trace（链路追踪）](#4-trace链路追踪)
5. [db_configs（数据库）](#5-db_configs数据库)
6. [redis_config（缓存）](#6-redis_config缓存)
7. [security（登录安全）](#7-security登录安全)
8. [oidc（OIDC/SSO）](#8-oidcoidcssso)
9. [其他（jwt / client / masterKey）](#9-其他jwt--client--masterkey)
10. [环境差异要点](#10-环境差异要点)

---

## 1. 配置总览

```yaml
server:
  name: auth            # 服务名（auth/platformadmin/tenantadmin/gateway）
  port: 8081
  env: dev              # dev / prod 等

log:
  default: {...}        # 应用日志
  gorm: {...}           # GORM SQL 日志
  redis: {...}          # Redis 调用日志

trace:
  enable: true
  otlp:
    endpoint: "127.0.0.1:4317"

db:
  auto_migrate: true   # 启动时基于 GORM AutoMigrate 自动建表/增量同步（幂等）
  seed: true           # 启动时幂等写入基础种子数据（租户/角色/权限/菜单/管理员/应用客户端）

db_configs:
  - url: "postgres://postgres:123456@127.0.0.1:5432/iam?sslmode=disable&TimeZone=Asia/Shanghai"
    service: iam

redis_config:
  service: iam
  addr: 127.0.0.1:6379
  db: 0

security:
  login:
    maxFailures: 5
    windowSec: 300
    lockSec: 900

jwt:
  signKey: "your-jwt-secret-key"

oidc:
  issuer: "http://localhost:8081/oidc"
  frontendLoginURL: "http://localhost:3000/login"
  signingPrivateKeyPath: "config/oidc-dev-key.pem"
  signingPrivateKeyPEM: ""
  signingKeyID: "dev-oidc-key"
  encryptionKey: "oidc-dev-encryption-key-32bytes"
  encryptionKeyID: "dev-enc-key"
  allowInsecure: true
  authRequestTTL: 600
  authCodeTTL: 300
  spentCodeTTL: 86400
  sessionTTL: 86400
  cookieSecure: false
  cookieSameSite: lax
  enableSSOSessionValidation: false
  backChannelLogoutPath: "/bc-logout/platform"
```

---

## 2. server（服务）

| 配置项 | 说明 | 默认 |
|---|---|---|
| `name` | 服务名，用于日志/追踪标识 | auth 等 |
| `port` | HTTP 监听端口（8081/8082/8083/8100） | - |
| `env` | 环境标识：`dev`/`prod`。`dev` 启用 Swagger 文档、允许临时密钥、允许 insecure OIDC | 空 |

---

## 3. log（日志）

基于 `glog`（zap 内核）。每个命名日志（`default`/`gorm`/`redis`）独立配置：

| 配置项 | 说明 |
|---|---|
| `service` / `module` | 日志归属标识 |
| `level` | debug / info / warn / error |
| `logger_type` | zap |
| `writers` | `console`（控制台）、`file`（文件，`dir` 指定目录） |
| `enable_otel_trace` | 是否输出 OpenTelemetry trace 关联字段 |
| `extra_keys` | 附加 context key（如 `requestID`） |

---

## 4. trace（链路追踪）

| 配置项 | 说明 |
|---|---|
| `enable` | 是否启用 |
| `service_version` | 服务版本标签 |
| `sampler` | `traceidratio`（按比例采样）等 |
| `trace_id_ratio` | 采样比例（1.0 = 全采样） |
| `otlp.endpoint` | OTLP gRPC Collector 地址（默认 `127.0.0.1:4317`） |
| `otlp.insecure` | 是否明文传输 |
| `otlp.timeout` | 上报超时 |

> 初始化失败自动降级为 disabled 模式，不影响服务启动。

---

## 5. db_configs（数据库）

| 配置项 | 说明 |
|---|---|
| `url` | PostgreSQL DSN（`postgres://user:pass@host:port/db?sslmode=disable&TimeZone=Asia/Shanghai`） |
| `service` | 库服务名（本系统 `iam`），应用通过 `dbclient.IamDB(ctx)` 访问 |

## 5.1 db（启动行为）

| 配置项 | 说明 |
|---|---|
| `auto_migrate` | 启动时基于 GORM AutoMigrate 自动创建/增量同步全部数据表（幂等，只增不改不删）；关闭后需自行执行 schema 初始化 |
| `seed` | 启动时幂等写入基础种子数据（平台租户、角色、资源、权限、菜单、管理员账号 admin/admin123、OIDC 测试客户端）；基于唯一键查重，可安全重复执行 |

> 多租户预留：`tenant.db_user` 字段支持按租户路由数据库用户（当前未启用分库）。

---

## 6. redis_config（缓存）

| 配置项 | 说明 |
|---|---|
| `service` | `iam` |
| `addr` | `host:port` |
| `password` / `db` | 认证与库号 |
| `dial_timeout` / `read_timeout` / `write_timeout` | 连接超时 |

> **重要**：SSO 会话、授权状态、令牌元数据、SLO 队列都依赖 Redis。启用"登出即失效"（`enableSSOSessionValidation`）的业务应用必须与 auth **共享同一认证 Redis**。

---

## 7. security（登录安全）

| 配置项 | 说明 | 默认 |
|---|---|---|
| `login.maxFailures` | 窗口内最大失败次数 | 5 |
| `login.windowSec` | 失败计数窗口（秒） | 300 |
| `login.lockSec` | 锁定时间（秒） | 900 |

达到阈值后该 IP + person 登录锁定，返回 `LoginLockedError`；成功登录清零计数。

---

## 8. oidc（OIDC/SSO）

| 配置项 | 说明 | 默认 |
|---|---|---|
| `issuer` | OIDC issuer（派生全部端点）。生产必须为正式域名 | `http://localhost:{port}/oidc` |
| `frontendLoginURL` | 登录门户地址（跳转与 logged-out 落地） | `http://localhost:3000/login` |
| `signingKeyID` | 签名密钥 kid | dev-oidc-key |
| `signingPrivateKeyPath` | 签名私钥 PEM 文件路径（PKCS#1/PKCS#8） | - |
| `signingPrivateKeyPEM` | 签名私钥 PEM 内联（与 path 二选一） | - |
| `encryptionKey` | 授权码加密密钥（非 dev 必须显式配置，fail-closed） | - |
| `encryptionKeyID` | 加密密钥 kid | enc-key-1 |
| `allowInsecure` | 允许非 HTTPS（dev 才应开启） | false |
| `authRequestTTL` | 授权请求有效期（秒） | 600 |
| `authCodeTTL` | 授权码有效期（秒） | 300 |
| `spentCodeTTL` | 已消费授权码留存（秒） | 86400 |
| `sessionTTL` | SSO 会话 TTL（秒，同时是 Cookie Max-Age） | 86400 |
| `cookieSecure` | SSO Cookie `Secure` 标志，生产（HTTPS）必须 true | false |
| `cookieSameSite` | SSO Cookie SameSite：`lax`/`strict`/`none`（跨站 SSO 需 none + Secure） | lax |
| `cookieDomain` | SSO Cookie Domain（跨子域共享时设置） | 空 |
| `enableSSOSessionValidation` | 是否开启请求粒度 SSO 会话活性校验（需共享 Redis） | false |
| `backChannelLogoutPath` | 本应用 back-channel logout 接收端基础路径 | `/bc-logout/{app}` |

### 8.1 签名密钥说明

- 密钥加载优先级：`signingPrivateKeyPath`（文件）→ `signingPrivateKeyPEM`（内联）→ dev 自动生成（kid=`auto-key`）；
- **fail-closed 规则**：非 dev 环境未配置密钥或文件缺失时，服务**拒绝启动**（避免重启后 kid 变化导致全部令牌失效与 RP 公钥失同步）；
- RP 侧通过 `oidcauth.LoadSigningPublicKey(Conf)` 从同一配置加载公钥；也可从 `GET {issuer}/keys`（JWKS）动态获取。

### 8.2 SSO Cookie 拓扑选择

```mermaid
flowchart TB
    DEPLOY{"部署拓扑"}
    DEPLOY -->|"同站（同域名不同端口）"| LAX["SameSite=Lax<br/>Secure=false（HTTP dev）"]
    DEPLOY -->|"同站 HTTPS"| LAX2["SameSite=Lax + Secure=true"]
    DEPLOY -->|"跨站（不同域名）"| NONE["SameSite=None + Secure=true<br/>（必须 HTTPS）"]
    DEPLOY -->|"跨子域"| DOMAIN["cookieDomain=.example.com<br/>+ Secure"]
```

---

## 9. 其他（jwt / client / masterKey）

| 配置项 | 说明 |
|---|---|
| `jwt.signKey` | 预留的 JWT 密钥（当前令牌体系以 OIDC RS256 为主，此项保留兼容） |
| `client.httpbingo` | 示例 HTTP 客户端配置（host/module/retry/timeout） |
| `masterKey` | 预留主密钥（加密敏感配置用） |

---

## 10. 环境差异要点

| 项 | dev | prod |
|---|---|---|
| OIDC issuer | `http://localhost:8081/oidc` | 正式域名（HTTPS） |
| 签名/加密密钥 | 可自动生成临时密钥 | **必须显式配置**（否则启动失败） |
| `allowInsecure` | 可 true | false |
| `cookieSecure` | false | **true** |
| `cookieSameSite` | lax | 同站 lax / 跨站 none |
| Swagger 文档 | 开启（`/auth/redocs` 等） | 关闭 |
| Gin 模式 | debug | release（`env: prod`） |
