# OIDC 协议状态存储重构设计

- **日期**: 2026-06-02
- **状态**: 草案
- **作者**: AI 辅助设计

## 1. 背景与问题

### 1.1 当前问题

OIDC 授权码流程中有两类需要跨 HTTP 请求共享的短期协议状态：

1. **AuthRequest** — `/authorize` 端点创建，前端登录页提交 `/oidc/login` 时读取并完成，后续 `/authorize/callback` 换码阶段继续使用
2. **Authorization Code** — `CompleteAuthRequest` 后生成，客户端用 code 换取 token，一次性消费

当前这两个状态存储在 `OIDCStorage` 的进程内 map 中（`svcoidc/storage.go:25`），带来以下风险：

- 多实例部署时，`/authorize` 落在 A 实例，`/oidc/login` 落在 B 实例 → 状态不存在
- 实例重启/滚动发布后，用户留在前端登录页再提交 → 状态丢失
- 无法设置 TTL，异常中断产生脏数据长期残留
- 协议状态与持久化业务数据（DAO）混在同一个 `OIDCStorage` 中，职责边界模糊

### 1.2 目标

将 OIDC 授权流程的短期协议状态从进程内存迁移到 Redis，同时重构 `OIDCStorage` 按职责分层，为后续 OIDC 功能扩展打好基础。

### 1.3 非目标

- 不涉及 refresh token 的存储方案变更（已走 DB）
- 不涉及 OIDC 协议本身的逻辑变更（只改存储层）
- 不涉及前端 UI 的变更

## 2. 架构设计

### 2.1 三层存储架构

将当前单一的 `OIDCStorage` 按职责拆分为三层：

```
┌──────────────────────────────────────────────────────┐
│                   OIDCStorage (Adapter)                │
│    实现 op.Storage 接口，将方法路由到下层存储             │
│    负责签名密钥、算法等与存储无关的固定信息               │
├────────────────────┬─────────────────────────────────┤
│  PersistentStore    │    ProtocolStateStore            │
│  (长期业务数据)      │    (短期协议状态)                  │
│                    │                                  │
│  - 读取 OAuth client│  - AuthRequest CRUD              │
│  - 校验 client secret│  - Authorization Code 消费       │
│  - 读取 person/user  │  - TTL 过期清理                  │
│  - 读写 refresh token│  - 防重放                        │
│                    │                                  │
│  后端: DAO → DB    │  后端: Redis (唯一权威)            │
└────────────────────┴─────────────────────────────────┘
```

决策：

- **三层定位**：`OIDCStorage` 只做协议适配，不持有状态；`PersistentStore` 处理所有与 DAO/DB 相关的长期数据；`ProtocolStateStore` 处理所有短期 OIDC 协议状态
- **Redis 唯一权威**：`ProtocolStateStore` 只使用 Redis，不回退内存，Redis 不可用时严格失败
- **无 mock 层**：`ProtocolStateStore` 测试直接依赖真实 Redis

### 2.2 分层方法映射

#### OIDCStorage（适配层——自己持有的方法）

| 方法 | 说明 |
|------|------|
| `SigningKey` | 返回 RSA 私钥 |
| `KeySet` | 返回 JWKS 公钥 |
| `SignatureAlgorithms` | 返回 RS256 |
| `Health` | 委托协议存储检查 Redis 健康状态 |

#### ProtocolStateStore（委托到这里的方法）

| 方法 | 说明 |
|------|------|
| `CreateAuthRequest` | 创建并保存授权请求 |
| `AuthRequestByID` | 按 ID 读取授权请求 |
| `AuthRequestByCode` | 按授权码读取授权请求 |
| `SaveAuthCode` | 保存授权码 → 请求的映射 |
| `CompleteAuthRequest` | 标记授权请求为已完成 |
| `DeleteAuthRequest` | 删除授权请求及相关 code |

#### PersistentStore（委托到这里的方法）

| 方法 | 说明 |
|------|------|
| `GetClientByClientID` | 读取 OAuth 客户端 |
| `AuthorizeClientIDSecret` | 校验客户端密钥 |
| `SetUserinfoFromScopes` | 按 scope 填充用户信息 |
| `SetUserinfoFromToken` | 按 token 填充用户信息 |
| `CreateAccessAndRefreshTokens` | 签发访问令牌和刷新令牌 |
| `TokenRequestByRefreshToken` | 按刷新令牌查询 |
| `SetIntrospectionFromToken` | token introspection |
| `GetPrivateClaimsFromScopes` | 获取私有声明（目前 stub） |
| `ValidateJWTProfileScopes` | 验证 JWT Profile 范围（目前 stub） |

## 3. ProtocolStateStore 数据模型

### 3.1 Redis Key 设计

| Key | Value | TTL | 说明 |
|-----|-------|-----|------|
| `iam:oidc:auth_req:{id}` | JSON serialized AuthRequest | 10 分钟 | 授权请求完整状态 |
| `iam:oidc:auth_code:{code}` | request ID（纯字符串） | 5 分钟 | 授权码到请求的映射 |
| `iam:oidc:auth_code:spent:{code}` | 空值标记 | 24 小时 | 防重放，授权码用后标记 |

### 3.2 AuthRequest 序列化字段

```go
type AuthRequest struct {
    ID            string    `json:"id"`
    ClientID      string    `json:"client_id"`
    RedirectURI   string    `json:"redirect_uri"`
    State         string    `json:"state"`
    Scopes        []string  `json:"scopes"`
    ResponseType  string    `json:"response_type"`
    ResponseMode  string    `json:"response_mode"`
    Nonce         string    `json:"nonce"`
    CodeChallenge *CodeChallenge `json:"code_challenge,omitempty"`
    Subject       string    `json:"subject"`
    AuthTime      time.Time `json:"auth_time"`
    AMR           []string  `json:"amr"`
    ACR           string    `json:"acr"`
    Audience      []string  `json:"audience"`
    DoneFlag      bool      `json:"done_flag"`
    ExpiresAt     time.Time `json:"expires_at"`
}
```

`ExpiresAt` 用于代码层显式过期校验，与 Redis TTL 形成双重保障。

### 3.3 消费语义

**授权码消费（`ConsumeAuthCode`）**：

1. `GetDel` 读取 `iam:oidc:auth_code:{code}` → 不存在返回 `ErrCodeInvalid`
2. 用返回的 requestID 读取 `iam:oidc:auth_req:{id}` → 不存在返回 `ErrCodeInvalid`
3. 校验 `AuthRequest.DoneFlag == true` → 否则返回 `ErrSessionNotCompleted`
4. 校验 `AuthRequest.ExpiresAt` → 已过期返回 `ErrCodeExpired`
5. `SET iam:oidc:auth_code:spent:{code} "" EX 86400` → 标记防重放
6. 返回 `AuthRequest`

**SaveAuthCode**：

使用 `SETNX` 设置 `iam:oidc:auth_code:{code}`，确保同一 code 不会被覆盖。

### 3.4 TTL 说明

| 对象 | TTL | 可配置 |
|------|-----|--------|
| AuthRequest | 10 分钟 | 是（OIDC 配置 `authRequestTTL`） |
| Authorization Code | 5 分钟 | 是（OIDC 配置 `authCodeTTL`） |
| Spent 标记 | 24 小时 | 是（OIDC 配置 `spentCodeTTL`） |

## 4. Redis 操作与错误处理

### 4.1 操作原子性

| 操作 | 原子性保证 | 说明 |
|------|-----------|------|
| `SaveAuthRequest` | `SET` | 单 key 写入 |
| `AuthRequestByID` | `GET` | 单 key 读取 |
| `CompleteAuthRequest` | `GET` + `SET` | 读旧值 + 写新值（单 key） |
| `SaveAuthCode` | `SETNX` | 防止 code 碰撞 |
| `ConsumeAuthCode` | `GetDel` | 原子读取并删除 |
| `DeleteAuthRequest` | `DEL` | 删除 request 和关联 code |
| `MarkSpent` | `SET` | 单独的 spent key |

### 4.2 错误分类

| 错误 | 条件 | HTTP 映射 |
|------|------|-----------|
| `ErrStoreUnavailable` | Redis 连接失败/PONG 不通过 | 503 |
| `ErrSessionNotFound` | AuthRequest key 不存在或已过期 | 404 |
| `ErrCodeInvalid` | Code key 不存在或已过期 | 400 |
| `ErrCodeAlreadyUsed` | Spent 标记已存在 | 400 |
| `ErrCodeCollision` | SETNX 返回 0（理论上极低概率） | 500 |
| `ErrSessionNotCompleted` | AuthRequest DoneFlag 为 false | 400 |

### 4.3 Redis 不可用行为链

- Redis 连接建立时（`NewProtocolStateStore`）验证可达性
- 每次读写操作如果 Redis 返回连接错误，直接返回 `ErrStoreUnavailable`
- `Health()` 方法显式检查 Redis PONG
- **不允许**静默降级回内存或其他 fallback

## 5. 测试策略

### 5.1 测试模式

遵循项目现有模式：每个测试函数独立调用 `testsetup.Initialize(testsetup.AppNameIam)` + `defer testsetup.Done(testsetup.AppNameIam)`。

Redis client 通过 `dbclient.RedisCli` 全局变量获取，由 testsetup 自动初始化。

### 5.2 ProtocolStateStore 测试

直接使用真实 Redis，覆盖：

1. **正向流程**
   - `SaveAuthRequest` → `AuthRequestByID` 读写一致
   - `SaveAuthCode` → `ConsumeAuthCode` 全流程
   - `CompleteAuthRequest` 后 DoneFlag 置 true

2. **消费语义**
   - `ConsumeAuthCode` 原子消费：消费后再次消费返回 `ErrCodeInvalid` 或 `ErrCodeAlreadyUsed`
   - `SaveAuthCode` 重复 code 不被覆盖

3. **过期校验**
   - 构造已过期的 `ExpiresAt`，读取后应返回错误
   - 可配合 Redis TTL 验证（调整短 TTL 等待过期）

4. **删除**
   - `DeleteAuthRequest` 删除后 `AuthRequestByID` 返回 `ErrSessionNotFound`
   - 关联的 code 也一并不可用

5. **错误处理**
   - 不存在的 ID → `ErrSessionNotFound`
   - 未完成请求的 code → `ErrSessionNotCompleted`

### 5.3 完整流程测试

基于现有的 `TestFullOIDCCodeFlow`（`provider_flow_test.go`），底层存储由内存换为 Redis 后，同测试用例应继续通过。

### 5.4 不需要新增的测试

- `PersistentStore` 的方法不需要另写集成测试（DAO 已有，接口封装无新逻辑）
- `OIDCStorage` 适配层的测试覆盖由现有测试维持

## 6. 配置文件变更

OIDC 配置新增三个 TTL 配置项：

```yaml
oidc:
  authRequestTTL: 10m      # AuthRequest 过期时间
  authCodeTTL: 5m          # 授权码过期时间
  spentCodeTTL: 24h        # 已消费 code 防重放标记保留时间
```

默认值与当前设计一致，配置缺失时回退默认值。

## 7. 实施步骤

1. 定义 `ProtocolStateStore` 接口和 `RedisProtocolStateStore` 实现
2. 定义 `PersistentStore` 结构体，从 `OIDCStorage` 抽离 DAO 相关方法
3. 重构 `OIDCStorage`，注入两个 Store，移除进程内 map
4. 补全测试（`ProtocolStateStore` 真实 Redis 测试 + 完整流程回归）
5. 添加 OIDC 配置项并接入
6. 验证 `provider_flow_test.go` 全流程通过
7. 清理 `OIDCStorage` 中移除的 `newOAuthClientDao`、`newOAuthClientSecretDao` 函数变量（由 PersistentStore 接管）
