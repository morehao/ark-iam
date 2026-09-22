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
| RP1 首次登录 | 访问租户管理后台 → 自动跳转 login-web → 填写凭证 → 回调展示首页 |
| Admin 直接登录 | 访问管理平台 → 自动跳转 login-web → 填写凭证 → 进入仪表盘 |
| Admin 登录后 SSO 免密 | 先登录 Admin → 租户管理后台（同 context）自动免密登录 |
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
| login-web | 4000 | - | 登录页（凭证表单，非 OIDC Client） |
| platform-admin-web | 4001 | `platform_admin_web` | 平台管理后台（Admin） |
| tenant-admin-web | 4002 | `tenant_admin_web` | 租户管理后台（RP1） |

## 前置条件

- Node.js 18+
- PostgreSQL + Redis 已运行
- 后端**不再有启动期种子数据**：全新库必须先完成一次性初始化。`globalSetup` 会自动调 `GET /install/status` + `POST /install/initialize` 完成它，因此后端要**带初始化令牌启动**——环境变量 `BOOTSTRAP_TOKEN` 必须等于 `e2e/config.ts` 的 `bootstrapToken`（`e2e-bootstrap-token`），否则端点整体不可用（fail-closed，e2e 直接失败并给出提示）
- 初始化写入的内置数据：管理员 `admin` / 口令 `Admin123`（`e2e/config.ts` 的 `password`，满足 8–128 位含大小写与数字）+ 内置 OAuth 客户端 `platform_admin_web` / `tenant_admin_web`。库**已初始化**时 `/install/initialize` 返回 409 且永久自锁，e2e 会跳过初始化直接跑；要重来只能删库重建（见 `docs/design/run-and-deploy.md` §2.3 / §2.4）
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
- `globalSetup` — 检查并启动所需服务（IAM 后端 :8100、platform-admin-web :4001、tenant-admin-web :4002、login-web :4000）
- `globalTeardown` — 测试结束后强制清理所有进程（无论成功/失败）
- 首次初始化 — `globalSetup` 在服务就绪后调用 `GET /install/status`，未初始化则用 `BOOTSTRAP_TOKEN` 调 `POST /install/initialize`，随后再确认一次状态；已初始化直接跳过（端点自锁返回 409）

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