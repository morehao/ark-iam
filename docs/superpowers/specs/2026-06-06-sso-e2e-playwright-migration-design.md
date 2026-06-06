# SSO E2E Playwright 迁移 & 前端应用精简 设计文档

**日期**: 2026-06-06  
**状态**: 已批准

## 目标

SSO 登录测试只需 3 个前端应用：login-web、platform-admin-web、sso-test-app（RP1）。精简多余应用，将 e2e 从 puppeteer-core 迁移到 Playwright，并确保测试结束后自动清理进程。

## 1. 删除 sso-test-app-2

### 变更内容

| 操作 | 文件/目录 |
|------|-----------|
| 删除目录 | `frontend/apps/sso-test-app-2/` |
| 移除脚本 | `frontend/package.json` 中的 `dev:sso2` |
| 更新脚本 | `frontend/package.json` 中的 `dev:all`，改为只启动 3 个应用 |

### 不影响的文件

- `frontend/pnpm-workspace.yaml` — sso-test-app-2 不在其中（apps/ 目录下的应用通过通配符匹配）
- 共享包 `@ark-iam/shared` 和 `@ark-iam/tsconfig` — sso-test-app-2 不引用它们

## 2. E2E 迁移到 Playwright

### 技术选型

- 使用 `@playwright/test`（Playwright Test Runner）
- 测试文件：TypeScript（`.spec.ts`）
- 浏览器：Chromium（headless）

### 测试场景

1. **RP1 首次登录**
   - 访问 sso-test-app（`localhost:3001`）
   - 点击登录 → 跳转到 login-web（`localhost:3003`）
   - 填写用户名/密码 → 提交
   - 验证回调到 RP1 后显示项目管理面板
   - 验证 Token 详情展示正确

2. **管理平台 SSO 自动登录**
   - 在同一浏览器上下文访问 platform-admin-web（`localhost:3000`）
   - 点击"IAM 账号登录"触发 OIDC Authorize 流程
   - 验证因 iam_sso_session cookie 存在而自动登录
   - 验证跳转到管理平台仪表盘

### 进程管理

- 使用 Playwright `globalSetup` 启动后端 + 前端服务
- 使用 Playwright `globalTeardown` 停止所有服务
- 测试失败时 teardown 也会执行（确保进程清理）

### 文件结构

```
e2e/
├── package.json
├── playwright.config.ts      # Playwright 配置
├── global-setup.ts           # 启动服务
├── global-teardown.ts        # 停止服务
├── helpers/
│   └── oidc-helpers.ts       # OIDC 测试辅助函数（PKCE、等待回调等）
├── tests/
│   └── oidc-sso.spec.ts      # OIDC SSO 测试用例
└── README.md
```

## 3. Makefile 更新

新增 `e2e` 目标，封装完整 e2e 测试流程：

```makefile
.PHONY: e2e
e2e:
	cd e2e && npx playwright test
```

## 4. 文档更新

### OIDC_SELF_TEST.md

需更新的章节：
- **Step 4**（第 69-96 行）：移除 sso-test-app-2 启动说明
- **Step 5**（第 103-142 行）：删除"双 RP"流程图中的 RP2 部分
- **Step 6**（第 169-189 行）：将"双 RP SSO 验证"改为"管理平台 SSO 验证"
- **Step 8**（第 208-288 行）：更新 E2E 测试说明，指向 Playwright
- **环境速查表**：移除 sso-test-app-2 / 3002 条目

### e2e/README.md

- 移除端口 3002 引用
- 移除"双 RP"概念
- 更新为 Playwright 的安装和运行说明

## 5. 不涉及的内容

- 不修改后端代码
- 不修改 login-web、platform-admin-web、sso-test-app 的功能逻辑
- 不修改 `docs/oidc-verification-guide.md`、`docs/oidc-sso-integration.md`、`docs/oidc-frontend-contract.md`、`docs/iam-login-flow.md`
