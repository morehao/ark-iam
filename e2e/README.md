# E2E 自动化测试

本目录存放基于 Playwright 的 OIDC SSO 端到端测试，模拟真实浏览器操作验证完整业务流程。

## 认证流程

三个前端应用基于 **react-oidc-context** + **oidc-client-ts** 实现 OIDC 认证，应用会自动重定向到 IAM OIDC Provider。

### SSO 单点登录流程

```
访问 SP 应用 → 自动 signinRedirect → OIDC authorize 端点
  ├── 有 iam_sso_session cookie → 静默认证 → 回调 SP 应用
  └── 无 SSO session → 重定向到 login-web → 填写凭证 → 回调 SP 应用
```

## 测试场景

| 测试用例 | 覆盖内容 |
|----------|----------|
| RP1 首次登录 | 访问租户管理平台 → 自动跳转 login-web → 填写凭证 → 回调展示首页 |
| Admin 直接登录 | 访问管理平台 → 自动跳转 login-web → 填写凭证 → 进入仪表盘 |
| Admin 登录后 SSO 免密 | 先登录 Admin → 租户管理平台（同 context）自动免密登录 |
| Admin 登出后自身需重认证 | Admin 登录 → 登出 → 访问需重新跳转 login-web |
| RP1 登录后 Admin SSO | RP1 登录后 → Admin 管理平台静默 SSO 免密登录 |
| RP1→Admin→Admin 登出→RP1 | RP1 登录 → Admin SSO → Admin 登出 → 兄弟应用需重新认证 |
| 双向 SSO | Admin 登录 → RP1 SSO → Admin 登出 → RP1 需重新认证 |
| Admin 登录→登出→重新登录 | 完整认证流程验证 |
| Cookie 跨 context 隔离 | 独立 browser context 各自维护独立 session |
| 全局登出（SLO） | 全局登出清除 SSO 会话，兄弟应用不再共享免密 SSO |

测试基于 Playwright 的 browser context 自动共享 cookie，模拟真实的 SSO session 行为。

## 服务映射

| 服务 | 端口 | OAuth client_id | 说明 |
|------|------|-----------------|------|
| gateway | 8100 | - | IAM 后端（`/oidc`） |
| login-web | 3000 | - | 登录页（凭证表单） |
| platform-admin-web | 3001 | `platform-admin-web` | 管理平台（Admin） |
| tenant-admin-web | 3002 | `tenant-admin-web` | 租户管理平台（RP1） |

## 前置条件

- Node.js 18+
- MySQL + Redis 已运行
- 后端种子数据已导入（`admin` / `admin123` + OAuth 客户端 `platform-admin-web` / `tenant-admin-web`）
- 后端需使用 `config.yaml` 启动（OIDC 端点前缀 `/oidc`）

## 安装

```bash
cd e2e
npm install
npx playwright install chromium
```

## 运行

```bash
# headless 模式（CI 推荐）
npm test

# 有头模式（可视化调试）
npm run test:headed

# 单步调试
npm run test:debug
```

测试自动管理服务生命周期：
- `globalSetup` — 检查并启动所需服务（IAM 后端 :8100、platform-admin-web :3001、tenant-admin-web :3002、login-web :3000）
- `globalTeardown` — 测试结束后强制清理所有进程（无论成功/失败）

## 配置

配置文件：`playwright.config.ts`

| 配置项 | 值 | 说明 |
|--------|-----|------|
| headless | true | headless 运行 |
| browser | chromium | 浏览器类型 |
| workers | 1 | 单 worker（确保服务状态一致） |
| timeout | 120s | 单个测试超时 |

## 项目集成

通过 Makefile：

```bash
make e2e
```