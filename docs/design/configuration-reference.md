# 配置参考（Configuration Reference）

> 本文说明 Ark IAM 各应用 `config.yaml` 的配置项。五个应用（auth / platformadmin / tenantadmin / rpapi / gateway）共享同一套 `pkg/config.Config` 结构，差异主要在 `server.name/port`、OIDC 相关项，以及**仅 auth 生效**的登录限流（`security.login.ratePerMinute` / `burst`）与 `server.trustedProxies`、**仅 platformadmin / tenantadmin 生效**的 `oidc.backChannelLogoutPath`。RP 侧（含外部应用）的 OIDC/目录能力另由 `backend/sdk` 提供，旋钮见 §11。
>
> 配置加载顺序：环境变量 `APP_CONFIG_PATH` 指定路径 → 相对**当前工作目录**的 `../config/config.yaml` → **可执行文件所在目录的上一级** `config/config.yaml`；三者皆无时仍按相对路径加载并启动失败（panic）。

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
11. [RP SDK 客户端配置（sdk/rp）](#11-rp-sdk-客户端配置sdkrp)

---

## 1. 配置总览

```yaml
server:
  name: auth            # 服务名（auth/platformadmin/tenantadmin/rpapi/gateway）
  port: 8081
  env: dev              # dev / prod 等
  trustedProxies: []    # 可信反向代理 CIDR（仅 auth 生效；留空 = 不信任任何代理）

log:
  default: {...}        # 应用日志
  gorm: {...}           # GORM SQL 日志
  redis: {...}          # Redis 调用日志

trace:
  enable: true
  otlp:
    endpoint: "127.0.0.1:4317"

db:
  auto_migrate: true   # 启动时基于 GORM AutoMigrate 自动建表/增量同步（幂等，只增不删）
  seed: true           # 启动时幂等写入基础种子数据（seed_key 认行 + 字段权威矩阵）

db_configs:
  - url: "postgres://postgres:123456@127.0.0.1:5432/iam?sslmode=disable&TimeZone=Asia/Shanghai"
    service: iam
    max_open_conns: 100
    max_idle_conns: 10

redis_config:
  service: iam
  addr: 127.0.0.1:6379
  db: 0

security:
  login:
    maxFailures: 5
    windowSec: 300
    lockSec: 900
    ratePerMinute: 30   # 登录/改密接口按 IP 每分钟令牌数（仅 auth 生效）
    burst: 10           # 令牌桶突发容量（仅 auth 生效）

jwt:
  signKey: ""           # 已废弃（令牌统一为 OIDC RS256），仅保留结构兼容，可安全移除

oidc:
  issuer: "http://localhost:8081/oidc"
  # jwksURL 是 RP 侧取公钥的**显式端点**；留空时按 issuer 解析（标准 discovery 的
  # jwks_uri 优先，本仓 OP 即 {issuer}/keys）
  jwksURL: ""
  # audiences 是本应用接受的 aud 白名单；留空＝不校验 aud
  # 注意 aud 是 per-app 的：gateway 聚合了某个应用，就要在 gateway 配置里为它
  # 声明同样的白名单（如 rpapi → ["rpapi"]）
  audiences: []
  # 多签名密钥（配置后优先于下方 legacy 单 key 三元组）
  keys:
    - kid: "dev-oidc-key"                      # 留空则按公钥派生
      privateKeyPath: "config/oidc-dev-key.pem"
      privateKeyPEM: ""
      active: true                             # 恰好一个 active
  frontendLoginURL: "http://localhost:4000/login"
  # legacy 单 key（与上面的 keys 二选一，等价于 keys 只有一项且 active: true）
  # signingPrivateKeyPath: "config/oidc-dev-key.pem"
  # signingPrivateKeyPEM: ""
  # signingKeyID: "dev-oidc-key"
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
| `port` | HTTP 监听端口（auth 8081 / platformadmin 8082 / tenantadmin 8083 / rpapi 8084 / gateway 8100） | - |
| `env` | 环境标识：`dev`/`prod`。`dev` 启用 Swagger 文档、允许临时密钥、允许 insecure OIDC | 空 |
| `trustedProxies` | 可信反向代理 CIDR 列表（如 `["10.0.0.0/8"]`）。配置后 gin 仅从这些代理透传的 `X-Forwarded-For` 取客户端 IP；**未配置则不信任任何代理**，直接使用 `RemoteAddr`，防止伪造 `X-Forwarded-For` 绕过按 IP 的限流与登录锁定。**仅 auth 应用读取并生效**，gateway 聚合部署下该项不生效 | 空 |

---

## 3. log（日志）

基于 `glog`（zap 内核）。每个命名日志（`default`/`gorm`/`redis`）独立配置：

| 配置项 | 说明 |
|---|---|
| `service` / `module` | 日志归属标识 |
| `level` | debug / info / warn / error |
| `logger_type` | zap |
| `writers` | 输出目标列表：`type: console`（控制台）/ `type: file`（文件）。`file` 支持 `dir`、`file_name`、`level`（覆盖全局级别）、`max_size`、`max_backups`、`max_age`、`compress`、`wf_only`（仅 warn/fatal/error 落盘） |
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
| `max_idle_conns` / `max_open_conns` | 连接池空闲 / 最大连接数 |
| `conn_max_lifetime` | 连接最大存活时长 |
| `max_sql_len` | SQL 日志截断长度 |
| `slow_threshold` | 慢 SQL 阈值（超过按慢查询记录） |

## 5.1 db（启动行为）

| 配置项 | 说明 |
|---|---|
| `auto_migrate` | 启动时基于 GORM AutoMigrate 自动创建/增量同步全部数据表（幂等）。**只新增缺失的表/列/索引，不删列、不改名**；但 GORM 会同步既有列的类型/默认值/非空约束。列 / 表下线一律删代码 + 删库重建（见 `system-design.md` §4.1） |
| `seed` | 启动时幂等写入基础种子数据（平台租户、根部门、两个内置应用 platform_admin/tenant_admin、内置角色与菜单授权、租户应用订阅、管理员账号 admin/admin123、两个内置 OAuth 客户端 platform_admin_web/tenant_admin_web）。按**种子身份键 `seed_key`** 认行、字段按**权威矩阵**收敛（见 `system-design.md` §4.5），可安全重复执行 |

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
| `login.ratePerMinute` | 登录 / 改密接口按 IP 每分钟令牌数（**仅 auth 生效**） | 30 |
| `login.burst` | 令牌桶突发容量（**仅 auth 生效**） | 10 |

达到阈值后锁定，返回 `LoginLockedError`；成功登录清零计数。**IP 维度无条件锁定；person 维度只在触发锁定的来源 IP 上被拒绝**（避免攻击者用错误口令把他人账号锁死）。

`ratePerMinute` / `burst` 用 golib 令牌桶限流挂载在 `POST /oidc/login` 与 `POST /v1/auth/me/changePassword` 上；Redis 不可用时 **fail-open**（放行不报错）。

---

## 8. oidc（OIDC/SSO）

| 配置项 | 说明 | 默认 |
|---|---|---|
| `issuer` | OIDC issuer（派生全部端点）。生产必须为正式域名 | `http://localhost:{port}/oidc` |
| `frontendLoginURL` | 登录门户地址（跳转与 logged-out 落地） | `http://localhost:4000/login` |
| `signingKeyID` | 签名密钥 kid（**legacy 单 key**，等价于 `keys` 只有一项） | 留空时按公钥派生 |
| `signingPrivateKeyPath` | 签名私钥 PEM 文件路径（PKCS#1/PKCS#8，**legacy 单 key**） | - |
| `signingPrivateKeyPEM` | 签名私钥 PEM 内联（**legacy 单 key**，与 path 二选一，PEM 优先） | - |
| `keys` | **签名密钥列表（多 key）**：每项 `kid`（可省）/`privateKeyPath`/`privateKeyPEM`/`active`；配置后**优先于**上面三个单 key 字段 | 空（回退单 key 三元组） |
| `jwksURL` | RP 侧（含内置应用）获取 OP 公钥的**显式端点**（末段为 `keys`/`jwks` 或以 `.json` 结尾）；留空时按 `issuer` 解析：标准 discovery 的 `jwks_uri` → `{issuer}/keys` → `{issuer}/.well-known/jwks.json`（**本仓 OP 发布在 `{issuer}/keys`**）；gateway 单体部署下注入进程内 key set，完全无网络调用 | 空 |
| `audiences` | 本应用接受的 `aud` 白名单；留空＝**不校验 aud**（auth 的 `/v1/auth/*` 即是）；`rpapi` 声明 `["rpapi"]`。**per-app**：gateway 聚合部署时须在 gateway 配置里同样声明 | 空 |
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
| `backChannelLogoutPath` | 本应用 back-channel logout 接收端基础路径（挂载在 `/oidc` 组下） | platformadmin 兜底 `/bc-logout/platform`、tenantadmin 兜底 `/bc-logout/tenant`；`pkg/oidckit` 通用兜底 `/oidc/bc-logout`；auth / gateway 未配置 |

### 8.1 签名密钥（多 key、轮换与启动校验）

`oidc.keys` 是签名密钥列表；**配置了 `keys` 就忽略三个单 key 字段**。每项字段：

| 字段 | 说明 | 默认 |
|---|---|---|
| `kid` | 密钥标识（JWKS 的 `kid`）。留空时按公钥派生 `base64url(sha256(publicKey.N)[:16])`（RFC 7638 thumbprint 风格），避免"换 key 忘改 kid" | 派生 |
| `privateKeyPath` | RSA 私钥 PEM 文件路径（PKCS#1 / PKCS#8） | - |
| `privateKeyPEM` | 内联 RSA 私钥 PEM（与 `privateKeyPath` 二选一，**PEM 优先**） | - |
| `active` | 是否为当前签发用 key | false |

例行零中断轮换（追加 → 切 `active` → 等 ≥2×JWKS 缓存 TTL（默认 10 分钟，见 §11.1）→ 摘除旧 key）：

```yaml
oidc:
  issuer: "https://iam.example.com/oidc"
  keys:
    - kid: "2026-09"
      privateKeyPath: "config/oidc-2026-09.pem"
      active: true               # 新令牌用该 key 签发
    - kid: "2026-03"
      privateKeyPath: "config/oidc-2026-03.pem"
      active: false              # 过渡期继续发布，供存量令牌验签
```

紧急轮换（疑似私钥泄露）= 追加新 key + 切 `active` + **立即删除旧 key 条目**（旧令牌当场失效）。`/oidc/keys` 发布**全部**配置的公钥（含 `active: false` 的过渡 key），SDK 按 `kid` 取键，因此轮换不需要重启 RP。

**零破坏迁移**：三个单 key 字段仍然可用，等价于"`keys` 只有一项且 `active: true`"（`signingKeyID` 给 kid，`signingPrivateKeyPath` / `signingPrivateKeyPEM` 二选一给私钥）：

```yaml
oidc:                                  # 与上面的 keys 列表等价
  signingKeyID: "dev-oidc-key"
  signingPrivateKeyPath: "config/oidc-dev-key.pem"
  signingPrivateKeyPEM: ""
```

**启动期校验（配置错误即拒绝启动，fail-closed）**：

| 规则 | 说明 |
|---|---|
| 恰好一个 active | `keys` 中 0 个或多个 `active: true` 都是配置错误（不猜测） |
| kid 唯一 | 列表内 kid（含派生值）重复即报错——否则验签方会按 kid 取到错误公钥 |
| 每项必须有私钥 | 每项须给 `privateKeyPath` 或 `privateKeyPEM`，两者皆空即报错 |
| 非 dev 必须配置 | 非 dev 环境未配置任何密钥时**拒绝启动**；dev 会自动生成临时密钥（显式配了 `privateKeyPath` 时写回该路径） |
| 非 dev 不自动生成 | 显式配了路径但文件缺失时，非 dev 同样拒绝启动（dev 才自动生成）——避免重启换 kid 使全部令牌失效、各 RP 公钥失同步 |
| 加密密钥 | `encryptionKey` 非 dev 必须显式配置（缺省测试密钥是公开常量）；`encryptionKeyID` 缺省 `enc-key-1` |

RP 侧取公钥的正路是 `oidckit.ResolveKeySource(injected, conf)` / `oidckit.NewKeySourceFromConfig(conf)`（装配在 `pkg/oidckit/keys.go`）：gateway 单体部署下由 auth 注入**进程内 key set**（`OIDCStorage.PublishedKeys()` → `rp.NewKeysFromSet`，零网络调用、跟随多 key 轮换）；独立部署则优先 `oidc.jwksURL`（显式端点，原样使用），其次 `issuer`（SDK 解析端点并**同步预取**：标准 discovery 的 `jwks_uri` → `{issuer}/keys` → `{issuer}/.well-known/jwks.json`；**本仓 OP（zitadel/oidc）发布在 `{issuer}/keys`**，它不提供 `{issuer}/.well-known/jwks.json`），最后回退到本地配置签名密钥导出的快照（会告警：OP 轮换后需重启才生效，生产应配 `jwksURL`）；三者皆无则启动失败（fail-closed）。客户端**必须**在 JWT 头带 `kid`，并拒绝 `jwk`/`jku`/`x5u` 头。（历史上曾有「启动时钉死一把公钥」的兼容入口，已随本轮收敛删除；本地签名密钥快照只作无 JWKS 端点时的兜底。）

### 8.2 SSO Cookie 拓扑选择

```mermaid
flowchart TB
    DEPLOY{"部署拓扑"}
    DEPLOY -->|"同站（同域名不同端口）"| LAX["SameSite=Lax<br/>Secure=false（HTTP dev）"]
    DEPLOY -->|"同站 HTTPS"| LAX2["SameSite=Lax + Secure=true"]
    DEPLOY -->|"跨站（不同域名）"| NONE["SameSite=None + Secure=true<br/>（必须 HTTPS）"]
    DEPLOY -->|"跨子域"| DOMAIN["cookieDomain=.example.com<br/>+ Secure"]
```

### 8.3 aud 白名单与 JWKS 地址（以 rpapi 为例）

`oidc.audiences` 是 **per-app** 的 `aud` 白名单（按应用声明，不硬编码 client_id）：留空表示**不校验 aud**（`auth` 的 `/v1/auth/*` 就是这种情形——它服务多个内置控制台客户端）；一旦声明，access token 的 `aud` 必须命中其中之一，否则按无效令牌拒绝。

`rpapi`（独立进程，校验 M2M 令牌时**本地验签、不查库**）的 `oidc` 配置即范例：

```yaml
oidc:
  issuer: "http://localhost:8081/oidc"          # 指向 OP（auth）
  jwksURL: "http://localhost:8081/oidc/keys"    # 显式端点可省一次 discovery；留空则由 SDK 解析
  audiences:
    - "rpapi"                                   # 只接受 aud 命中 rpapi 自身 client_id 的 M2M 令牌
  enableSSOSessionValidation: true
  backChannelLogoutPath: ""
```

> gateway 单体部署时由 auth 注入**进程内 key set**，不产生网络调用；此时 `jwksURL` 不生效也不需要配置。
> **但 `audiences` 仍要配**：它是 per-app 的，gateway 配置里为 rpapi 声明 `["rpapi"]`，否则 `/v1/rp/directory/*` 在 gateway 下会跳过 aud 收紧。rpapi 独立部署时才走 `jwksURL`。

---

## 9. 其他（jwt / client / masterKey）

| 配置项 | 说明 |
|---|---|
| `jwt.signKey` | 预留的 JWT 密钥（当前令牌体系以 OIDC RS256 为主，此项保留兼容） |
| `client.httpbingo` | 示例 HTTP 客户端配置（`host` / `retry` / `timeout`，对应 golib `ghttp.Client`；**无 `module` 字段**——YAML 为非严格解析，写了会被静默忽略；当前代码未消费） |
| `password.prefix` | 口令前缀配置位（`pkg/config.PasswordConfig`）；当前全仓无消费者，属死配置 |
| `es_configs` | Elasticsearch 配置位（`[]dbes.ESConfig`）；应用侧未配置、未消费，仅 `pkg/testsetup` 构造测试配置时读取 |
| `masterKey` | 预留主密钥（加密敏感配置用） |

---

## 10. 环境差异要点

| 项 | dev | prod |
|---|---|---|
| OIDC issuer | `http://localhost:8081/oidc` | 正式域名（HTTPS） |
| 签名/加密密钥 | 签名密钥可自动生成（或显式配 `oidc.keys` 多 key）；加密密钥用固定测试常量（仅供 dev） | **必须显式配置**（否则启动失败） |
| `allowInsecure` | 可 true | false |
| `cookieSecure` | false | **true** |
| `cookieSameSite` | lax | 同站 lax / 跨站 none |
| `oidc.audiences` | 可留空（不校验 aud） | 建议按应用声明白名单 |
| Swagger 文档 | 开启（`/auth/redocs` 等） | 关闭 |
| Gin 模式 | debug | release（`env: prod`） |

---

## 11. RP SDK 客户端配置（`sdk/rp`）

外部应用（RP）经独立 module `backend/sdk` 接入（安装与用法见 `backend/sdk/README.md`，依赖白名单由 `make sdk-check-deps` 强制）。以下旋钮是**程序化参数**（Go option / `Config` 字段），**不在 `config.yaml`**；本仓内置应用把同一批能力经 `pkg/middleware` 暴露给 `oidc.*` 配置（§8）。

### 11.1 验签（`rp.NewKeys` / `rp.NewVerifier`）

| 旋钮 | 说明 | 默认 |
|---|---|---|
| `WithJWKSTTL` | JWKS 内存缓存的后台刷新周期 | 10 分钟 |
| `WithMinRefreshInterval` | 未知 `kid` 触发的即时刷新限速窗口 | 1 次/分钟 |
| `WithMaxKeyAge` | 拉取失败时保留旧 key 的最长容忍时长；超过即视为密钥源不可用（fail-closed） | 24 小时 |
| `WithJWKSFallback(path)` | 落盘 JWKS 兜底文件：远端不可用时读取，成功刷新后写回（0600） | 关闭 |
| `WithHTTPClient` | 注入自定义 HTTP 客户端（JWKS 拉取单次超时默认 3s） | 默认客户端 |
| `WithLogger` | 最小日志接口（`Debugf`/`Warnf`），未注入时静默 | 静默 |
| `WithIssuer` | 期望的 OP issuer，设置后 `iss` 必须精确匹配 | 不校验 |
| `WithAudiences` | 认可的 `aud` 集合，设置后必须命中其一 | 不校验 |
| `WithLeeway` | 时钟容差（`exp`/`nbf`） | 30s |
| `WithAlgorithms` | 允许的签名算法白名单 | RS256 |
| `WithKeySource` | 注入密钥来源（`NewVerifier` 必填，缺失即全部拒绝） | 必填 |

`rp.NewKeys` **启动即同步预取一次**；失败时若配置了 `WithJWKSFallback` 且文件有效则用它，否则返回错误（fail-fast，避免"服务起来了但全部 401"）。注意 `WithJWKSFallback` 缓存的是**公钥**，与目录客户端的 `StaleOnError`（缓存目录数据）不是一回事。

### 11.2 只读目录客户端（`sdk/rp/directory.Config`）

对应 `/v1/rp/directory/*`（见 `api-reference.md` §8）：

| 字段 | 说明 | 默认 |
|---|---|---|
| `BaseURL` | 目录 API 根地址（如 `http://iam:8084/v1/rp`，不带尾斜杠） | 必填 |
| `Tokens` | M2M 令牌来源（`rp.TokenClient`，距 `exp` 60s 自动换新） | 必填 |
| `HTTPClient` | 自定义 HTTP 客户端 | connect 1s / 总超时 3s |
| `MemberTTL` | 成员缓存 TTL | 60s |
| `DepartmentTTL` | 部门树缓存 TTL | 300s |
| `RoleTTL` | 角色清单缓存 TTL | 300s |
| `NegativeTTL` | 404 负缓存 TTL（挡穿透） | 30s |
| `StaleOnError` | **仅**连接失败/超时/5xx 时允许返回过期缓存（**401/403/404 一律透传，绝不降级**） | 0（关闭），建议上限 5 分钟 |
| `Retry` | GET 失败后的重试次数（带退避）。字段注释声称默认 1，但 `directory.New` **未填默认值**，实际不配置时为 0（即不重试，`attempts = Retry+1`） | 0（字段注释写作 1） |
| `Metrics` | 可选计数器（缓存命中 / stale / 未命中 / 不可用；额外实现 `NotModifiedMetrics` 可收 304 计数） | nil |

客户端侧批量上限 `directory.MaxBatchIDs = 100`（与服务端一致，超出本地即报 `ErrTooManyIDs`，不静默截断）；目录读取要求的 scope 是 `directory.ScopeRead = "directory.read"`。缓存无命中且目录不可用时返回 `ErrUnavailable`；`Member.Status` **不得**用于放行/拒绝判定（只服务展示）。
