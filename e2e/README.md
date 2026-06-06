# E2E 自动化测试

本目录存放基于 Playwright 的 OIDC SSO 端到端测试，模拟真实浏览器操作验证完整业务流程。

## 测试场景

| 测试用例 | 覆盖内容 |
|----------|----------|
| RP1 首次登录 | 打开测试 RP → 跳转登录页 → 输入凭据 → 回调展示项目管理面板 |
| RP1 Token 详情 | 查看 Token → 获取 UserInfo → 刷新 Token → 返回主页 |
| RP1 本地退出 | 退出当前应用 → 登录页 → 点"使用 IAM 登录" → SSO 免密进入主页 |
| RP1 全局退出 | 从所有应用退出 → 登录页 → 需重新填写凭证 |
| 管理平台 SSO 自动登录 | RP1 登录后 → 打开管理平台 → 点击 IAM 账号登录 → 自动认证进仪表盘 |
| 管理平台登出后 SSO 已清除 | 登录 → 登出 → 再点登录应显示登录表单而非自动认证 |

测试基于 Playwright 的 browser context 自动共享 cookie，模拟真实的 SSO session 行为。

## 前置条件

- Node.js 18+
- MySQL + Redis 已运行
- 后端种子数据已导入（`admin` / `admin123` + `test-rp-client`）

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
- `globalSetup` — 检查并启动所需服务（IAM 后端 :8099、platform-admin-web :3000、sso-test-app :3001、login-web :3003）
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
