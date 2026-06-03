# OIDC Provider 联调验证指南

提供从自动化测试到手工 curl/浏览器验证的完整步骤，确保 OIDC Provider 实现正确。

## 1. 前置条件

| 组件 | 启动命令 | 端口 |
|------|----------|------|
| MySQL | 外部启动（系统服务） | 3306 |
| Redis | 外部启动（系统服务） | 6379 |
| 后端 IAM | `make run APP=iam` | 8099 |
| 前端 | `npm run dev`（frontend/ 目录） | 3000 |

确认配置文件 `backend/apps/iam/config/config.yaml` 中：

```yaml
oidc:
  issuer: "http://localhost:8099/v1/iam/oidc"
  frontendLoginURL: "http://localhost:3000/oidc/login"
  signingKeyID: "dev-oidc-key"
  signingPrivateKeyPath: "config/oidc-dev-key.pem"
  signingPrivateKeyPEM: ""
  encryptionKey: "oidc-dev-encryption-key-32bytes"
  encryptionKeyID: "dev-enc-key"
  allowInsecure: true
  authRequestTTL: 600
  authCodeTTL: 300
  spentCodeTTL: 86400
```

确认签名私钥文件 `backend/apps/iam/config/oidc-dev-key.pem` 存在。

## 2. 运行自动化测试

```bash
# 后端 OIDC 相关测试（不依赖数据库）
cd backend/apps/iam
go test -count=1 -v ./internal/service/svcoidc/... |
  grep -E '(PASS|FAIL|---)'

go test -count=1 -v ./internal/controller/ctroidc/... |
  grep -E '(PASS|FAIL|---)'

go test -count=1 -v ./internal/router/... |
  grep -E '(PASS|FAIL|---)'

# 后端编译检查
go vet ./...
go build -o /dev/null ./...

# 前端测试
cd frontend
npx vitest run
```

预期结果：所有 OIDC 相关测试 PASS，编译无错误。前端 17 tests PASS。

## 3. 准备 OAuth Client 测试数据

以下 SQL 插入一个测试 RP（Relying Party）客户端，用于验证完整授权码流程。

### 3.1 插入 oauth_client

```sql
INSERT INTO `oauth_client`
  (`created_at`, `updated_at`, `tenant_id`, `app_id`, `client_id`, `name`,
   `redirect_uris`, `post_logout_redirect_uris`, `grant_types`, `response_types`,
   `token_endpoint_auth_method`, `allowed_origins`, `require_pkce`,
   `require_auth_time`, `default_scopes`, `access_token_ttl`, `refresh_token_ttl`,
   `type`, `is_third_party`, `status`, `created_by`, `updated_by`, `deleted_by`)
VALUES
  (NOW(), NOW(), 1, 1, 'test-rp-client', '测试 RP',
   '["https://client.example.com/callback"]', '[]',
   '["authorization_code","refresh_token"]', '["code"]',
   'client_secret_basic', '[]', 0, 0,
   '["openid","profile","email"]', 3600, 2592000,
   'first_party', 0, 'enable', 1, 1, 0);
```

> `client_id = test-rp-client`，这个 ID 在 curl 和浏览器测试中会用到。

### 3.2 客户端密钥计算

后端验证密钥的逻辑（`persistent_store.go:46-48`）：

```go
secretHash := sha256.Sum256([]byte(clientSecret))
clientHash := hex.EncodeToString(secretHash[:])
```

选择一个测试密钥并计算 SHA256 十六进制串。

**方法 A：命令行计算**

```bash
echo -n "my-test-client-secret" | shasum -a 256
# 输出: 5449e4aa114c00349163d0825e5b4cdd2f72642c1c31f625e6a67ec52e85fa0d
```

**方法 B：Go 代码计算**

```go
package main
import (
    "crypto/sha256"
    "encoding/hex"
    "fmt"
)
func main() {
    h := sha256.Sum256([]byte("my-test-client-secret"))
    fmt.Println(hex.EncodeToString(h[:]))
}
```

### 3.3 插入 oauth_client_secret

```sql
INSERT INTO `oauth_client_secret`
  (`created_at`, `updated_at`, `oauth_client_id`, `name`, `value_hash`,
   `value_prefix`, `expired_at`, `revoked_at`, `created_by`, `updated_by`, `deleted_by`)
VALUES
  (NOW(), NOW(), 1, '测试密钥',
   '5449e4aa114c00349163d0825e5b4cdd2f72642c1c31f625e6a67ec52e85fa0d',
   'my-test', NULL, NULL, 1, 1, 0);
```

> 注意：`oauth_client_id` 是上一步插入的 oauth_client 记录的自增 ID。如果数据库有历史数据，请替换为正确的 ID。
>
> 密钥原文：`my-test-client-secret`（此后 curl 中会用到）

### 3.4 测试用户

本验证使用后端已有的用户认证逻辑。如果数据库中没有可用的 `person` + `user` 记录，请先插入：

```sql
-- 先确定你已有的测试用户
SELECT id, username, primary_email FROM person LIMIT 5;
SELECT id, person_id, tenant_id FROM user WHERE person_id = <上面查到的person.id>;
```

记录下你用于测试的 `identifier`（用户名/邮箱/手机号）和密码。

## 4. Curl 验证协议端点

以下步骤逐步执行，覆盖完整授权码流。

### Step 1: Discovery 端点

```bash
curl -s http://localhost:8099/v1/iam/oidc/.well-known/openid-configuration | jq .
```

**确认要点：**
- `issuer` 为 `http://localhost:8099/v1/iam/oidc`
- `authorization_endpoint` 存在
- `token_endpoint` 存在
- `jwks_uri` 存在
- `userinfo_endpoint` 存在
- `response_types_supported` 包含 `code`
- `grant_types_supported` 包含 `authorization_code`, `refresh_token`
- **不应包含** `device_authorization_endpoint`

### Step 2: JWKS 端点

```bash
curl -s http://localhost:8099/v1/iam/oidc/.well-known/jwks.json | jq .
```

**确认要点：**
- `keys` 数组中只有 **RSA 公钥**（`kty` 应为 `RSA`）
- `alg` 为 `RS256`
- 不包含对称密钥（不应出现 `kty: oct`）

### Step 3: 构造 Authorize URL（模拟 RP 跳转）

手动构造 authorize 请求。在浏览器中打开或使用 curl 跟随重定向：

```
http://localhost:8099/v1/iam/oidc/authorize
  ?client_id=test-rp-client
  &redirect_uri=https://client.example.com/callback
  &response_type=code
  &scope=openid%20profile%20email
  &state=test-state-123
  &nonce=test-nonce-456
```

**Curl 方式（不跟随 302）：**

```bash
curl -v 'http://localhost:8099/v1/iam/oidc/authorize?client_id=test-rp-client&redirect_uri=https://client.example.com/callback&response_type=code&scope=openid%20profile%20email&state=test-state-123&nonce=test-nonce-456'
```

**确认要点：**
- 返回 HTTP 302
- `Location` 形如 `http://localhost:3000/oidc/login?authRequestID=ar-xxx`
- `authRequestID` 是后端创建的唯一 ID

记录下 `authRequestID` 的值（下一步需要）。

### Step 4: OIDC Login（服务器端认证）

拿到 Step 3 的 `authRequestID`，替换到下面请求中：

```bash
curl -s -X POST http://localhost:8099/v1/iam/oidc/login \
  -H 'Content-Type: application/json' \
  -d '{
    "authRequestID": "ar-1741234567890123000",
    "identifier": "person@example.com",
    "password": "YourPassword"
  }' | jq .
```

**确认要点：**
- `code` 为 `0`
- `data.continueURL` 形如 `http://localhost:8099/v1/iam/oidc/authorize/callback?id=ar-1741234567890123000`

将 `continueURL` 复制到浏览器打开（或下一步通过 curl 跟随）。

### Step 5: 跟随 continueURL 获取 Code

**Curl 方式（不跟随 302）：**

```bash
curl -v 'http://localhost:8099/v1/iam/oidc/authorize/callback?id=ar-1741234567890123000'
```

**确认要点：**
- 返回 HTTP 302
- `Location` 应按 RP 注册的 `redirect_uri` 跳转
- URL 中包含 `?code=xxx&state=test-state-123`
- `state` 与 Step 3 传入的一致

记录下 `code` 的值（下一步 token 交换需要）。

### Step 6: Token Endpoint（code 换 token）

```bash
curl -s -X POST http://localhost:8099/v1/iam/oidc/oauth/token \
  -H 'Content-Type: application/x-www-form-urlencoded' \
  -d 'grant_type=authorization_code' \
  -d 'code=上一步获取的code' \
  -d 'redirect_uri=https://client.example.com/callback' \
  -d 'client_id=test-rp-client' \
  -d 'client_secret=my-test-client-secret' | jq .
```

**确认要点：**
- `access_token` 不为空
- `token_type` 为 `Bearer`
- `expires_in` 为正整数
- `id_token` 为 JWT 格式（三段式）
- `refresh_token` 不为空

**可选：验证 id_token 签名**

```bash
# 先获取 JWKS，用 jose 工具验证 id_token 的 RSA 签名
# 安装 jose: https://github.com/lestrrat-go/jwx
# 或使用 https://jwt.io 手动粘贴 id_token 和 JWKS 公钥验证
```

确认 id_token 的 payload 中：
- `iss` = `http://localhost:8099/v1/iam/oidc`
- `sub` = `person:<id>` 格式
- `aud` = `test-rp-client`
- `exp` > `iat`

### Step 7: UserInfo 端点

```bash
curl -s http://localhost:8099/v1/iam/oidc/userinfo \
  -H 'Authorization: Bearer 上一步获取的access_token' | jq .
```

**确认要点：**
- `sub` 为 `person:<id>`
- 包含 `name`、`preferred_username` 等个人信息
- `email` 不为空

### Step 8: Refresh Token（可选）

```bash
curl -s -X POST http://localhost:8099/v1/iam/oidc/oauth/token \
  -H 'Content-Type: application/x-www-form-urlencoded' \
  -d 'grant_type=refresh_token' \
  -d 'refresh_token=上一步获取的refresh_token' \
  -d 'client_id=test-rp-client' \
  -d 'client_secret=my-test-client-secret' | jq .
```

**确认要点：**
- 返回新的 `access_token` 和 `id_token`
- 可选：返回新的 `refresh_token`

### Step 9: 错误场景验证

**场景 A：错误的密码**

```bash
curl -s -X POST http://localhost:8099/v1/iam/oidc/login \
  -H 'Content-Type: application/json' \
  -d '{
    "authRequestID": "ar-xxx",
    "identifier": "person@example.com",
    "password": "wrong-password"
  }' | jq .
```

预期：`code != 0`

**场景 B：无效的 authRequestID**

```bash
curl -s -X POST http://localhost:8099/v1/iam/oidc/login \
  -H 'Content-Type: application/json' \
  -d '{
    "authRequestID": "non-existent-id",
    "identifier": "person@example.com",
    "password": "Password1"
  }' | jq .
```

预期：`code = 100799`（OIDC session not found）

**场景 C：无效 client_id（token 端点）**

```bash
curl -s -X POST http://localhost:8099/v1/iam/oidc/oauth/token \
  -H 'Content-Type: application/x-www-form-urlencoded' \
  -d 'grant_type=authorization_code' \
  -d 'code=invalid' \
  -d 'redirect_uri=https://evil.com/' \
  -d 'client_id=invalid-client' \
  -d 'client_secret=xxx'
```

预期：返回 401/400

## 5. 浏览器全流程验证

如果选择优先通过前端页面验证，可直接使用本节流程，Section 4 的 curl 步骤可跳过或后续按需补充。前端认证失败时，页面底部会 toast 显示后端返回的错误提示（如 "OIDC session not found"），同时可在浏览器 Console 面板查看完整错误信息。

### 5.1 入口方式

**方式 A：直接粘贴 URL（推荐）**

在浏览器地址栏直接粘贴 Authorize URL，回车触发流程：

```
http://localhost:8099/v1/iam/oidc/authorize?client_id=test-rp-client&redirect_uri=https://client.example.com/callback&response_type=code&scope=openid%20profile%20email&state=test-state-123&nonce=test-nonce-456
```

浏览器会自动跟随 302 跳转到 OIDC 登录页。**建议操作前先打开浏览器 DevTools（F12）→ Network 面板，勾选 Preserve log**，否则 302 跳转后会丢失之前的请求记录。

**方式 B：使用假 RP 页面（观察回调）**

如果想同时确认 code 参数落到了哪个页面，创建一个测试 HTML 文件：

```html
<!DOCTYPE html>
<html>
<body>
  <h1>Test RP</h1>
  <a href="http://localhost:8099/v1/iam/oidc/authorize?client_id=test-rp-client&redirect_uri=https://client.example.com/callback&response_type=code&scope=openid%20profile%20email&state=test-state-123&nonce=test-nonce-456">
    发起 OIDC 登录
  </a>
  <div id="result"></div>
  <script>
    const params = new URLSearchParams(window.location.search);
    if (params.has('code')) {
      document.getElementById('result').innerHTML = '<h3>Callback 成功</h3>' +
        '<p>code: ' + params.get('code') + '</p>' +
        '<p>state: ' + params.get('state') + '</p>';
    }
  </script>
</body>
</html>
```

用浏览器直接用 `file://` 协议打开这个 HTML，点击"发起 OIDC 登录"。`file://` 协议下最终 callback 重定向可能被浏览器拦截，此时仍需到 Network 面板查看最后一条 302 响应的 Location。

### 5.2 预期流程

打开 DevTools → Network 面板并勾选 **Preserve log**，然后开始操作：

```
1. 浏览器访问 authorize URL → 后端返回 302（Network 面板可见）
2. 浏览器跳转到 http://localhost:3000/oidc/login?authRequestID=ar-xxx
3. 页面显示 OIDC 登录表单
4. 输入 identifier 和 password，点击登录
5. 前端 POST /v1/iam/oidc/login（axios baseURL /v1/iam + Vite proxy 转发到后端 8099）
6. 后端返回 { code: 0, data: { continueURL: "..." } }
7. 前端执行 window.location.href = resp.continueURL
8. 浏览器跳转到 /authorize/callback?id=ar-xxx，后端完成认证并 302 到 redirect_uri
9. redirect_uri 携带 code 和 state 参数（Network 面板可见）
```

> **关键验证点**：步骤 9 即使浏览器无法加载目标页面（如 `https://client.example.com/callback`），也能在 Network 面板中找到最后一条 302 响应，其 Location 头包含 `?code=xxx&state=test-state-123`。确认 `state` 与步骤 1 传入的一致。

> **查看完整回调**：如果想看到浏览器加载回调页面，可将 oauth_client 的 `redirect_uris` 改为 `["http://localhost:9999/callback"]`，然后启动 HTTP 服务器：
> ```bash
> python3 -m http.server 9999
> ```
> 修改后需重新插入测试数据（Step 3）。浏览器会加载 `http://localhost:9999/callback?code=xxx&state=test-state-123`，页面 404 无所谓，地址栏中能看到 code 和 state 即验证通过。

### 5.3 DevTools 调试技巧

| 场景 | 操作 |
|------|------|
| 302 跳转后请求记录丢失 | **勾选 Preserve log**，Network 面板顶部勾选框 |
| 查看 302 跳转目标 | 找到 Status 302 的请求 → Response Headers → `Location` 字段 |
| 后端返回错误但页面无提示 | Console 面板查看错误对象（`console.error` 输出） |
| 查看最终 callback 地址 | 最后一条 302 的 Location，或浏览器地址栏实际跳转的 URL |
| 确认 code/state 参数 | 在 Network 面板搜索 `?code=` 或 `state=` 定位请求 |

> **Preserve log 不生效的替代方案**：如果忘记勾选 Preserve log，可在 Console 面板执行 `window.performance.getEntriesByType('navigation')` 查看当前页面的导航来源，或在 Network 面板开启 **"Show all content"** 筛选。

## 6. 常见问题排查

### 6.1 后端启动失败

```bash
# 查看启动日志
cd backend/apps/iam && make run
```

常见错误：
- `signingPrivateKeyPath` 指定的 PEM 文件不存在 → 检查 `config/oidc-dev-key.pem`
- MySQL/Redis 连接失败 → 检查 `config.yaml` 中的 `db_configs` 和 `redis_config`

### 6.2 CORS 问题

如果前端请求 `POST /v1/iam/oidc/login` 出现跨域错误：

- 确认 `router/oidc.go` 中 OIDC 路由组已添加 `ginmiddleware.CORS()`
- 检查请求是从 `http://localhost:3000` 发出
- 生产环境应配置具体域名白名单：`ginmiddleware.WithAddAllowOrigins("https://console.example.com")`

### 6.3 302 跳转不到前端

如果 authorize 端点没有跳转到 `http://localhost:3000/oidc/login`：

- 检查 `config.yaml` 中 `frontendLoginURL` 是否配置正确
- 检查 `client.go` 中 `LoginURL` 方法的 `authRequestID` 参数拼接

### 6.4 OIDC Login 返回 100799 OIDC session not found

- 确认 `authRequestID` 来自刚创建的 authorize 请求
- AuthRequest 存储在内存中，服务重启后之前的 request 丢失
- 每次测试前重新执行 Step 3 获取新的 `authRequestID`

### 6.5 Token 端点返回 Invalid Client

- 确认 client_id（`test-rp-client`）在 `oauth_client` 表中存在且 `status = enable`
- 确认 client_secret 的 SHA256 哈希与库中存储一致
- 尝试用 `client_secret_basic` 方式（将 client_id:client_secret base64 放入 Authorization header）

```bash
# client_secret_basic 方式
AUTH=$(echo -n "test-rp-client:my-test-client-secret" | base64)
curl -s -X POST http://localhost:8099/v1/iam/oidc/oauth/token \
  -H 'Content-Type: application/x-www-form-urlencoded' \
  -H "Authorization: Basic $AUTH" \
  -d 'grant_type=authorization_code' \
  -d 'code=xxx' \
  -d 'redirect_uri=https://client.example.com/callback'
```

### 6.6 端口配置不匹配

| 配置文件 | 配置项 | 预期值 |
|----------|--------|--------|
| `backend/config.yaml` | `server.port` | 8099 |
| `frontend/vite.config.ts` | `server.proxy./v1.target` | `http://localhost:8099` |
| `backend/config.yaml` | `oidc.issuer` | `http://localhost:8099/v1/iam/oidc` |
| `frontend` 浏览器访问 | 前端页面 | `http://localhost:3000` |

### 6.7 签名密钥问题

启动时日志中应有类似输出：

```
loaded signing key from config/oidc-dev-key.pem (keyID=dev-oidc-key)
```

如果看到 `auto-generated signing key`，说明私钥文件未找到，使用了自动生成密钥。这不会影响功能，但会导致重启后 id_token 签名变化（JWKS 也变）。

## 7. 验证清单（逐项打勾）

```
[ ] 启动后端（make run APP=iam）
[ ] 启动前端（npm run dev）
[ ] 后端自动化测试通过
[ ] 前端自动化测试通过
[ ] OAuth client 测试数据就绪
[ ] Discovery 端点返回完整配置
[ ] JWKS 仅暴露 RSA 公钥
[ ] Authorize 返回 302 + 前端登录 URL
[ ] OIDC Login 返回 continueURL
[ ] ContinueURL 回调完成，获取到 code
[ ] Token 端点 code 换 token 成功
[ ] id_token sub 为 person:<id> 格式
[ ] UserInfo 端点返回用户信息
[ ] Refresh Token 可用
[ ] 密码错误场景返回正确错误码
[ ] 无效 authRequestID 场景返回 100799
```
