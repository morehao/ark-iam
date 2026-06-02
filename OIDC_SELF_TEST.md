# OIDC 本地自测指南

## 前置条件

- MySQL（root:123456@127.0.0.1:3306）
- Redis（127.0.0.1:6379, password: 123456）
- Go 1.21+
- Node.js 18+
- pnpm 11+

---

## Step 1 — 初始化数据库

```bash
# 创建数据库
mysql -uroot -p123456 -e "CREATE DATABASE IF NOT EXISTS iam CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci;"

# 建表
mysql -uroot -p123456 iam < backend/scripts/sql/iam_schema.sql

# 导入种子数据（包含 admin 用户 + test-rp-client）
mysql -uroot -p123456 iam < backend/scripts/sql/iam_seed_data.sql
```

**种子数据关键内容：**

| 数据 | 值 |
|------|-----|
| 管理员账号 | `admin` / `admin123` |
| OAuth ClientID | `test-rp-client` |
| Client Secret | `my-test-client-secret` |
| 回调地址 | `http://localhost:3001/` |

---

## Step 2 — 启动后端

```bash
# 项目根目录
make run APP=iam
```

服务监听 `:8099`。验证：

```bash
curl -s http://localhost:8099/v1/iam/oidc/.well-known/openid-configuration | python3 -m json.tool
```

> 如果 OpenTelemetry 报错不影响启动，代码有 graceful fallback。

---

## Step 3 — 启动前端

```bash
cd frontend
pnpm dev
```

前端监听 `:3000`，API 代理到 `:8099`。

---

## Step 4 — 启动 SSO 测试 RP

```bash
cd frontend/sso-test-app
python3 -m http.server 3001
```

> 简单静态服务器即可，因为 OIDC 路由组已配置 CORS 中间件。

---

## Step 5 — 完整 OIDC 授权码流程测试

### 流程示意图

```
[测试RP:3001]                         [IAM前端:3000]                     [IAM后端:8099]
     |                                      |                                |
     |-- 1. 点击"使用IAM登录" -------------->|                                |
     |                                      |-- 2. /authorize --------------->|
     |                                      |                                |
     |                                      |<-- 3. 302 → /oidc/login -------|
     |                                      |                                |
     |                                      |-- 4. 展示OIDC登录表单           |
     |                                      |                                |
     |                                      |-- 5. POST /oidc/login --------->|
     |                                      |   (admin/admin123)             |
     |                                      |                                |
     |                                      |<-- 6. {continueURL} -----------|
     |                                      |                                |
     |                                      |-- 7. 302 → continueURL ------->|
     |                                      |                                |
     |<-- 8. 302 → :3001/?code=xxx ---------|                                |
     |                                      |                                |
     |-- 9. POST /token ------------------->|                                |
     |   (code + secret + verifier)         |                                |
     |                                      |                                |
     |<-- 10. {access_token, id_token, ...}  |                                |
     |                                      |                                |
     |-- 11. GET /userinfo ----------------->|                                |
     |   (Bearer access_token)              |                                |
     |                                      |                                |
     |<-- 12. {name, email, ...}            |                                |
```

### 操作步骤

1. 浏览器打开 `http://localhost:3001/`
2. 点击 **"使用 IAM 登录"**
3. 浏览器跳转到 IAM 登录页 `http://localhost:3000/oidc/login?authRequestID=ar-xxx`
4. 输入凭据：
   - 用户名: `admin`
   - 密码: `admin123`
5. 点击登录
6. 浏览器自动重定向回测试 RP `http://localhost:3001/?code=xxx&state=yyy`
7. 测试 RP 页面自动完成令牌交换，展示结果

### 验证要点

- ✅ 页面展示 **"OIDC 登录成功"**
- ✅ 展示 `access_token`（JWT 格式）
- ✅ 展示 `id_token`（解码后含 sub、iss、aud 等字段）
- ✅ 展示 `refresh_token`
- ✅ 点击 **"获取 UserInfo"** 展示用户信息（name、email、username）
- ✅ 点击 **"刷新 Token"** 成功刷新 access_token

---

## Step 6 — SSO 验证（概念验证）

因为 OIDC 本身就是 SSO 协议，本次测试已经验证了 IAM 作为**中心身份提供者（IdP）**的能力：

| 验证点 | 说明 | 结果 |
|--------|------|------|
| RP 通过 IAM 认证用户 | 测试 RP 使用 IAM 的 OIDC 端点完成登录 | ✅ |
| 身份令牌签发 | IAM 签发 id_token，含 sub(personID) 等声明 | ✅ |
| 访问令牌 | access_token 可用于调用 userinfo 等资源 | ✅ |
| 刷新令牌 | refresh_token 可换取新的 access_token | ✅ |

**SSO 的本质** — 用户只需要在 IAM（IdP）进行一次认证，IAM 向任意第三方应用（RP）签发身份令牌。测试 RP 作为第一个"第三方应用"验证了此能力。未来可以创建第二个测试 RP 在 `:3002`，使用同一个 IAM IdP，无需重复输入凭据即可登录。

---

## 环境速查表

| 服务 | 地址 | 端口 | 说明 |
|------|------|------|------|
| MySQL | `127.0.0.1:3306` | 3306 | 数据库 iam |
| Redis | `127.0.0.1:6379` | 6379 | 缓存 |
| IAM 后端 | `http://localhost:8099` | 8099 | Gin HTTP 服务 |
| IAM 前端 | `http://localhost:3000` | 3000 | React SPA |
| SSO 测试 RP | `http://localhost:3001` | 3001 | 静态 HTML 测试页 |
| OIDC Issuer | `http://localhost:8099/v1/iam/oidc` | - | OIDC Provider 根路径 |
| OIDC 登录页 | `http://localhost:3000/oidc/login` | - | 前端 OIDC 登录表单 |

---

## 常见问题

**Q: 授权请求返回 404？**
A: 检查后端是否已启动且端口正确。

**Q: OIDCLogin 跳转后白屏？**
A: 确认前端 `pnpm dev` 正常，Vite 代理生效。

**Q: 令牌交换返回 401？**
A: 确认 `OAuth Client` 种子数据已导入，client_id 和 secret 正确。

**Q: CORS 报错？**
A: OIDC 路由组已配置 CORS 中间件，检查浏览器是否拦截。

**Q: trace 初始化报错？**
A: 不影响功能，可以临时将 `config.yaml` 的 `trace.enable` 设为 `false`。
