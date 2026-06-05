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
| 回调地址 | `http://localhost:3001/`, `http://localhost:3002/` |

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

## Step 3 — 启动独立登录页服务

`log-web` 是 monorepo 下的子应用（`frontend/apps/log-web/`），基于 Vite + React，提供独立的登录页面。

```bash
cd frontend
pnpm dev:log
```

log-web 监听 `:3003`，提供 OIDC 授权码流程所需的登录表单。

---

## Step 4 — 启动 SSO 测试应用

`sso-test-app` 和 `sso-test-app-2` 是 monorepo 下的子应用（`frontend/apps/sso-test-app/`、`frontend/apps/sso-test-app-2/`），基于 Vite 开发。

**从 monorepo 根目录同时启动 RP1 + RP2：**

```bash
cd frontend
pnpm dev:sso
pnpm dev:sso2
```

**或从子应用目录单独启动：**

```bash
cd frontend/apps/sso-test-app
pnpm dev

# 另一个终端
cd frontend/apps/sso-test-app-2
pnpm dev
```

| 应用 | 端口 | 说明 |
|------|------|------|
| sso-test-app | 3001 | SSO 测试 RP（首次手动登录） |
| sso-test-app-2 | 3002 | SSO 测试 RP（自动 SSO 登录） |

sso-test-app 和 sso-test-app-2 均无 API 代理，直接请求后端 `:8099`。

---

## Step 5 — 完整 OIDC 授权码流程测试

### 流程示意图

```
[测试RP1:3001]                      [测试RP2:3002]                       [IAM后端:8099]
     |                                    |                                    |
     |-- 1. 点击"使用IAM登录" ------------>|                                    |
     |                                    |-- 2. GET /authorize -------------->|
     |                                    |                                    |
     |                                    |<-- 3. 302 → /sso-login -----------|
     |                                    |    (无session cookie)              |
     |                                    |                                    |
|-- 4. 302 → :3003/login ---------->|                                    |
|    ?authRequestID=ar-xxx           |                                    |
     |                                    |                                    |
     |-- 5. POST /oidc/login ------------>|                                    |
     |    (admin/admin123)                |                                    |
     |                                    |                                    |
     |<-- 6. {continueURL, sessionID} ---|                                    |
     |    + Set-Cookie: iam_sso_session   |                                    |
     |                                    |                                    |
     |-- 7. 302 → continueURL ----------->|                                    |
     |<-- 8. 302 → :3001/?code=xxx ------|                                    |
     |                                    |                                    |
     |-- 9. POST /oauth/token ----------->|                                    |
     |<-- 10. {access_token, id_token} ---|                                    |
      |-- 10b. 展示"项目管理面板"主页       |                                    |
     |                                    |                                    |
     |                                    |-- (用户在RP1页面浏览中)           |
     |                                    |                                    |
     |                                    |-- 11. 打开RP2:3002，自动登录 ---->|
     |                                    |                                    |
     |                                    |-- 12. GET /authorize ------------->|
     |                                    |                                    |
     |                                    |<-- 13. 302 → /sso-login ---------|
     |                                    |    (带session cookie!)            |
     |                                    |                                    |
     |                                    |-- 14. 自动完成认证，302 → continue|
     |                                    |<-- 15. 302 → :3002/?code=yyy ----|
     |                                    |                                    |
     |                                    |-- 16. POST /oauth/token ---------->|
     |                                    |<-- 17. {access_token, id_token} --|
     |                                    |-- 17b. 展示"数据分析面板"主页    |
```

### 操作步骤（RP1 首次登录）

1. 浏览器打开 `http://localhost:3001/`
2. 点击 **"使用 IAM 登录"**
3. 浏览器跳转到独立登录页 `http://localhost:3003/login?authRequestID=ar-xxx`
4. 输入凭据：
   - 用户名: `admin`
   - 密码: `admin123`
5. 自动登录
6. 浏览器自动重定向回测试 RP `http://localhost:3001/?code=xxx&state=yyy`
7. 测试 RP 页面自动完成令牌交换，展示 **"项目管理面板"主页**

### 验证要点（RP1）

- ✅ 页面展示 **模拟的"项目管理面板"主页**，包含用户头像、姓名、邮箱和 SSO 登录徽标
- ✅ 展示 4 个统计卡片：项目数、任务数、消息数、团队数
- ✅ SSO 徽标显示 **"✅ 已通过 IAM SSO 登录"**
- ✅ 点击 **"查看 Token 详情"** 可切换查看 access_token、id_token、refresh_token
- ✅ Token 详情页点击 **"获取 UserInfo"** 展示用户信息（name、email、username）
- ✅ Token 详情页点击 **"刷新 Token"** 成功刷新 access_token
- ✅ Token 详情页点击 **"返回主页"** 回到模拟面板

---

## Step 6 — 双 RP SSO 验证

### 流程示意图

见上方 Step 5 流程图（步骤 11-17）。

### 操作步骤

1. ✅ 完成 Step 5（RP1 已成功登录，浏览器已有 `iam_sso_session` cookie）
2. 打开新标签页 `http://localhost:3002/`
3. **页面自动重定向到 IAM OIDC authorize 端点**（无需任何点击操作）
4. IAM 检测到已有 session cookie → **自动签发授权码**，重定向回 RP2
5. RP2 自动完成令牌交换，展示 **"数据分析面板"主页**

### 验证要点

- ✅ RP1 首次登录：跳转到 IAM 登录页 → 输入凭据 → 展示"项目管理面板"主页
- ✅ RP2 二次登录：**无登录表单出现，页面自动完成跳转**，直接展示"数据分析面板"主页
- ✅ 两个 RP 的 SSO 徽标一致（同一用户身份），但主页面板数据不同
- ✅ 点击 **"查看 Token 详情"** 可看到有效的 access_token 和 id_token
- ✅ 两个 RP 的 id_token 中 sub（personID）一致

---

## Step 7 — SSO 原理说明

本次测试验证了 IAM 作为**中心身份提供者（IdP）**的 SSO 能力：

| 验证点 | 说明 | 结果 |
|--------|------|------|
| 首次登录需凭据 | RP1 首次访问 IAM IdP，无 session cookie，显示登录页 | ✅ |
| SSO 自动认证 | RP2 访问 IAM IdP，携带 `iam_sso_session` cookie，自动完成认证 | ✅ |
| 身份令牌签发 | IAM 签发 id_token，含 sub(personID) 等声明 | ✅ |
| 跨应用令牌 | 同一用户在不同 RP 获取不同的授权码和令牌 | ✅ |

**SSO 的本质** — 用户只需要在 IAM（IdP）进行一次认证后，浏览器获得 `iam_sso_session` cookie。后续任意 RP（RP1、RP2 乃至更多）发起 OIDC 授权请求时，服务端 `/sso-login` 端点检测到有效 cookie，自动完成用户认证并生成授权码，用户全程无需干预。

---

## 环境速查表

| 服务 | 地址 | 端口 | 说明 |
|------|------|------|------|
| MySQL | `127.0.0.1:3306` | 3306 | 数据库 iam |
| Redis | `127.0.0.1:6379` | 6379 | 缓存 |
| IAM 后端 | `http://localhost:8099` | 8099 | Gin HTTP 服务 |
| IAM 前端 | `http://localhost:3000` | 3000 | React SPA（管理端） |
| 独立登录页 | `http://localhost:3003` | 3003 | Vite + React（独立 IdP 登录页） |
| SSO 测试 RP1 | `http://localhost:3001` | 3001 | 静态 HTML 测试页 |
| SSO 测试 RP2 | `http://localhost:3002` | 3002 | 静态 HTML 测试页（验证 SSO） |
| OIDC Issuer | `http://localhost:8099/v1/iam/oidc` | - | OIDC Provider 根路径 |
| OIDC 登录页 | `http://localhost:3003/login` | - | 独立登录页服务（log-web） |
| SSO Session | `iam_sso_session` cookie | - | HTTP-only，Redis 存储，24h 过期 |

---

## 常见问题

**Q: 打开 RP 页面直接跳过了登录表单？**
A: 浏览器缓存了前次登录的 `iam_sso_session` cookie，这是 SSO 正常行为。如需重新输入密码：
- **Chrome**: DevTools (F12) → Application → Cookies → `http://localhost:8099` → 删除 `iam_sso_session`
- **Safari**: 设置 → 隐私 → 管理网站数据 → 搜索 `localhost` → 删除
- **Firefox**: DevTools (F12) → 存储 → Cookie → `http://localhost:8099` → 删除 `iam_sso_session`
- 最简方案：使用**无痕/隐私窗口**测试

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
