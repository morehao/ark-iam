# 签名密钥轮换运行手册（Signing Key Rotation Runbook）

适用范围：OP（`apps/auth`）的 ID token / access token / logout token 签名密钥。
相关实现：`backend/apps/auth/internal/service/svcoidc/provider.go`、`backend/pkg/config/config.go` 的 `OIDC.Keys`、`backend/sdk/rp/keys.go`（RP 侧 key set 消费）。

## 0. 结论速览

> **先分清两件事**：本文只讲**签名密钥**轮换（`oidc.keys`，影响"令牌能否被验签"）。
> 另有一条**刷新令牌**轮换（refresh token rotation，RFC 9706）——它由
> `oidcop.PersistentStore` 负责，与签名密钥无关：刷新成功后旧 refresh token 立即撤销，
> 但保留 **10 秒并发宽限窗口**，窗口内同一把旧 token 的重试幂等返回同一把新 token，
> 窗口外按复用攻击撤销整个 token 家族。它**不受**本文任何步骤影响。

| 场景 | 操作 | 生效时延 | 存量 token |
|---|---|---|---|
| 例行轮换 | 加新 key（`active: false`）→ 等 RP 刷新 → 新 key 置 `active: true` → 观察后摘除旧 key | 到期即生效（分钟级） | 过渡期内继续可用 |
| 紧急轮换（key 泄露） | 直接移除泄露 key（或换新 key 并设为唯一 `active`） | **立即** | 泄露 key 签发的 token 立刻失效 |
| 回滚 | 把配置改回上一份并重启 | 立即 | 视是否已摘除旧 key 而定 |

## 1. 密钥模型（最小多 key）

`oidc.keys` 是签名密钥列表，每项 `SigningKeyConfig`：

```yaml
oidc:
  keys:
    # 过渡期旧 key：仍在 JWKS 里发布，但不再签发新 token
    - kid: "2026-09-a"
      privateKeyPath: "config/oidc-2026-09-a.pem"
    # 当前签发 key：有且仅有一个 active
    - kid: "2026-10-b"
      privateKeyPath: "config/oidc-2026-10-b.pem"
      active: true
```

规则（启动期校验，违反即启动失败，绝不"猜一个"）：

- **恰好一个 `active: true`**：0 个或多个都是配置错误。
- **`kid` 必须唯一**：重复直接拒绝。
- **`kid` 可省略**：省略时按公钥派生 `base64url(sha256(publicKey.N)[:16])`，可复现。
- **单 key 三元组仍可用**（零破坏迁移）：`signingKeyID` / `signingPrivateKeyPath` / `signingPrivateKeyPEM`
  等价于列表里只有一项且 `active: true`。**不要与 `keys` 同时配置**。
- **非 dev 环境没有可用密钥即启动失败**（fail-closed）；dev 允许生成临时密钥。

发布语义：

- `JWKS`（`{issuer}/keys`，即 discovery 文档声明的 `jwks_uri`）发布**全部** key（含过渡期旧 key），按 `kid` 升序。
- **只有 `active` key 签发新 token**。
- 每个 token 头部必须带 `kid`；RP 按 `kid` 在 key set 里取公钥。
  RP 侧拒绝 `jwk` / `jku` / `x5u` 头（防 SSRF 与密钥替换），算法白名单默认只有 `RS256`。

## 2. RP 侧取键时序（决定轮换时延）

`backend/sdk/rp/keys.go` 的策略：

| 参数 | 默认值 | 作用 |
|---|---|---|
| startup prefetch | 启动即拉 | 启动拿不到 JWKS 且无磁盘兜底 → 启动失败（fail-closed） |
| 内存 kid 查表 | 命中零网络 | 业务请求路径**不访问 OP** |
| unknown-kid 刷新 | 限频 1 次/分钟 | 遇到没见过的 `kid` 主动刷新一次，防止无脑打爆 OP |
| 后台刷新 | 10 分钟 | 周期性同步 key set（新增 key 的主要发现路径） |
| 最大 key 年龄 | `WithMaxKeyAge`，默认 24h | 超过即视为不可用 → `ErrKeySourceUnavailable`（fail-closed） |
| 磁盘兜底 | `WithJWKSFallback(path)`（可选） | OP 不可达时用最近一次成功结果 |

因此**例行轮换的等待时间 = max(后台刷新周期 10min, unknown-kid 限频 1min)**。
实践上建议：加新 key 后**等待 ≥15 分钟**再切换 `active`。

## 3. 例行轮换（零中断）

1. **生成新密钥对**（不要覆盖旧文件）：

   ```bash
   openssl genrsa -out config/oidc-2026-10-b.pem 2048
   ```

2. **把新 key 加入配置，但 `active` 仍指向旧 key**（`active: false` 或省略）。

3. **滚动重启 auth**（`gateway` 单体部署则重启 gateway）。此时 JWKS 已包含新 key，
   但签发仍用旧 key —— **存量与新增 token 都不受影响**，这是本步骤的意义。

4. **等待 ≥15 分钟**，让所有 RP 完成一次后台刷新（或经 unknown-kid 路径刷新）。

5. **把新 key 置 `active: true`，旧 key 去掉 `active`**，再次滚动重启。
   新签发的 token 开始用新 `kid`；旧 key 仍在 JWKS 中，存量 token 继续可用。

6. **观察一个 access token 生命周期 + 余量**（建议 ≥ 24h，覆盖 refresh token 与
   最长的 ID token 有效期）后，**从配置中移除旧 key**，再次滚动重启。

7. **验证**：

   ```bash
   # JWKS 只应包含预期数量与 kid
   curl -s http://localhost:8081/oidc/keys | jq '.keys[].kid'

   # 新签发的 token 必须带新 kid
   # （登录一次，解出 header 看 kid）
   ```

> **gateway 单体部署的特别说明**：`auth`、`platformadmin`、`tenantadmin` 与 `auth` 同进程时，
> 内置应用走**进程内 key set**（`OIDCStorage.PublishedKeys()` → `rp.NewKeysFromSet`），
> 没有网络调用、也没有 10 分钟刷新延迟——配置变更随重启立即生效。
> 只有真正独立的 RP 进程才走上面的刷新时序。

## 4. 紧急轮换（key 泄露）

目标：**立即**让泄露 key 签发的 token 失效。

1. 生成新密钥对并按上面的方式配置为唯一 `active` key。
2. **从 `keys` 列表中直接删除泄露的 key**（或把它的文件替换为新密钥内容——
   但更清晰的做法是删除条目 + 新增条目，避免 `kid` 与内容不一致）。
3. 滚动重启 auth/gateway。

重启后 JWKS 不再包含泄露 key，所有 RP 的下一次校验就会因 `ErrUnknownKid` 失败 →
401。注意两点：

- **存量 token 立即失效**，用户需要重新登录；refresh token 也会因签名校验失败而不可用，
  这是紧急轮换的预期代价。
- 如果配置了 `WithJWKSFallback(path)`，**必须同时清掉该磁盘缓存文件**，否则 OP 不可达时
  RP 会回落到包含泄露 key 的旧 key set（见第 6 节的两条"轮换不生效"路径）。

## 5. 回滚

- **摘除旧 key 之前**：把配置改回上一份并重启即可，无副作用。
- **已摘除旧 key 之后**：无法恢复此前签发的 token（签名密钥已不在发布列表中）。
  回滚只能让新 token 重新用旧 `kid`，需要把密钥文件与条目一起恢复。

## 6. 常见误操作

| 误操作 | 后果 |
|---|---|
| 配置多个 `active: true` | 启动失败（不是"随机选一个"） |
| 两个条目 `kid` 重复 | 启动失败 |
| 同时配置 `keys` 与单 key 三元组 | 语义歧义；只保留一种 |
| 加新 key 后立即切 `active` | 部分 RP 还没拿到新 key → 新 token 出现大量 401 |
| 紧急轮换只换文件内容、不换 `kid` | RP 缓存里同 `kid` 的旧公钥继续被使用（`kid` 查表命中即不再刷新）→ **轮换不生效**。必须让 `kid` 变化（或删除条目让派生 kid 变化） |
| 紧急轮换后忘记清 `WithJWKSFallback` 磁盘缓存 | OP 不可达时回落到含泄露 key 的旧 key set |
| 认为 key 年龄兜底会"自动"救场 | `WithMaxKeyAge` 是断网时的 fail-closed 保护，不是轮换机制 |

## 7. 相关测试（回归锚点）

- `backend/apps/auth/internal/service/svcoidc/provider_keys_test.go`
  `TestLoadSigningKeysMultiKey`：active 唯一性、kid 派生稳定性、重复/缺失/多个 active 均报错。
- `backend/pkg/middleware/oidc_auth_test.go`
  `TestOIDCMultiKeyRotation`：多 key 并存时存量 token 可用；**摘除旧 key 后立即失效**。
- `backend/sdk/rp/verifier_test.go`：unknown-kid 触发刷新（`TestKeys_UnknownKidTriggersRefresh`）、
  刷新限频 1/分钟（`TestKeys_RefreshRateLimited`）、拉取失败保留旧 key
  （`TestKeys_RefreshFailureKeepsOldKeys`）、陈旧密钥 fail-closed（`TestKeys_MaxKeyAgeFailClosed`）、
  多 key key set 基准（`BenchmarkVerify_Keyset`）。
