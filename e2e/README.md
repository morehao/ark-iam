# E2E 自动化测试

本目录存放项目的端到端自动化测试脚本，使用 puppeteer 驱动真实 Chrome 浏览器，模拟用户操作验证完整业务流程。

## OIDC 端到端测试

`oidc-e2e.js` 对应 `OIDC_SELF_TEST.md` 中 Step 5 (RP1 首次登录) 和 Step 6 (双 RP SSO) 的所有验证要点，共 **25 个测试用例**。

### 前置条件

- 已完成 `OIDC_SELF_TEST.md` 的 Step 1 ~ Step 4（数据库初始化、后端、前端服务全部启动）
- macOS 已安装 Google Chrome（`/Applications/Google Chrome.app/`）

### 运行

```bash
cd e2e

# 安装依赖
npm install

# 启动测试（确保后端 8099、log-web 3003、sso-test-app 3001/3002 都在运行）
npm run test:oidc
```

### 输出

测试逐项打印每个验证点的 ✅/❌，最后给出汇总：

```
========== 测试结果汇总 ==========
✅ RP1 首页加载
✅ 找到并点击"使用 IAM 登录"按钮
✅ 跳转到 log-web 登录页
...
总计: 25 通过: 25 失败: 0
```

退出码：全部通过返回 0，有失败返回 1（可用于 CI 集成）。

### 跨平台 Chrome 路径

脚本默认 macOS Chrome 路径：

```js
chromePath: '/Applications/Google Chrome.app/Contents/MacOS/Google Chrome',
```

如需在 Linux/Windows 环境下运行，编辑 `oidc-e2e.js` 修改 `CONFIG.chromePath`：

| 平台 | Chrome 可执行路径 |
|------|-----------------|
| macOS | `/Applications/Google Chrome.app/Contents/MacOS/Google Chrome` |
| Linux | `/usr/bin/google-chrome` |
| Windows | `C:\Program Files\Google\Chrome\Application\chrome.exe` |
