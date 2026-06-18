# SSO Test App 精简重构设计

## 目标

将 `frontend/apps/sso-test-app` 从功能复杂的 SSO 测试应用精简为仅包含登录、首页、登出三个功能的轻量应用。改造方案参照 `frontend/apps/platform-admin-web` 的代码架构和模式。

## 设计决策

| 决策项 | 选择 | 说明 |
|--------|------|------|
| 布局 | Header + Content（无 Sider） | 只有登录和首页两个页面，不需要侧边菜单 |
| 首页内容 | 仅显示用户基本信息 | 头像、姓名、用户 ID，不保留统计卡片和 Token 调试功能 |
| 登出流程 | 单一退出 | 参照 platform-admin-web 模式：调后端 API → 清除 store → OIDC end_session |
| HTTP 客户端 | axios | 引入 axios + 请求/响应拦截器，对齐 platform-admin-web |
| 改造方式 | 文件对齐 platform-admin-web | 核心文件（authStore、oidc、request）直接对齐 platform-admin-web 模式 |

## 文件变更清单

### 新增文件

| 文件 | 说明 |
|------|------|
| `src/api/auth.ts` | 认证 API：`getUserinfo()`（GET `/v1/iam/auth/userinfo`）、`logoutAPI(refreshToken)`（POST `/v1/iam/auth/logout`） |
| `src/utils/request.ts` | Axios 实例 + 请求/响应拦截器：token 注入、TokenExpired 刷新、401 跳转登录 |
| `src/utils/response.ts` | `ApiResponse<T>` 类型 + `BizCode` 枚举 |
| `src/components/MainLayout.tsx` | Header（标题 + 头像下拉退出）+ Content（`<Outlet />`） |

### 改写文件

| 文件 | 说明 |
|------|------|
| `src/stores/authStore.ts` | 添加 `personInfo`、`tenantInfo` 字段，添加 `setPersonInfo`、`setTenantInfo` 方法 |
| `src/utils/oidc.ts` | 对齐 platform-admin-web 的 OIDC 配置结构、函数签名 |
| `src/App.tsx` | 路由守卫 + 静默登录逻辑，对齐 platform-admin-web |
| `src/pages/auth/Login.tsx` | 简化：仅 Card + "IAM 账号登录"按钮，对齐 platform-admin-web 风格 |
| `src/pages/auth/AuthCallback.tsx` | 对齐 platform-admin-web 的错误处理分支 |
| `src/pages/home/index.tsx` | 极简版：仅 Card 内展示用户 Avatar、姓名、ID |

### 目录重命名

| 原路径 | 新路径 |
|--------|--------|
| `src/pages/Login.tsx` | `src/pages/auth/Login.tsx` |
| `src/pages/AuthCallback.tsx` | `src/pages/auth/AuthCallback.tsx` |
| `src/pages/Home.tsx` | `src/pages/home/index.tsx` |

### 新增依赖

`package.json` 添加 `axios` 依赖。

## 组件设计

### MainLayout

```
Layout (minHeight: 100vh)
└── Layout
    ├── Header（白色背景，flex 左右两端对齐，paddingInline: 24）
    │   ├── 左侧：应用标题「SSO 测试应用」
    │   └── 右侧：Avatar + Dropdown
    │       └── Dropdown 菜单项：「退出登录」
    └── Content（margin: 24, padding: 24, 白色背景, borderRadius: 8）
        └── <Outlet />
```

- `useEffect` 中在 `authStage === 'authenticated'` 时调用 `getUserinfo()` 获取用户信息
- 退出：调用 `logoutAPI(refreshToken)` → 设置 `sessionStorage.logged_out = '1'` → 调用 `logout()` 清除 store → 加载 OIDC end_session URL

### Login

```
全屏居中布局，灰色背景 #f0f2f5
└── Card（width: 400, 标题「SSO 测试应用」）
    └── Button「IAM 账号登录」（KeyOutlined 图标, type: primary, block）
        点击 → generatePKCEParams() → storePKCEParams() → buildAuthorizeURL() → window.location.assign(url)
```

### AuthCallback

```
全屏 Spin 加载中
└── useEffect：
    - 解析 URL query params（code, state, error）
    - 有 error → 处理错误分支，跳转 /login
    - 有 code → 校验 state → exchangeCodeForTokens() → setSession() → navigate /（replace）
    - 清理 PKCE params
```

### Home

```
Card（title「用户信息」）
└── Avatar（src=personInfo.avatar, size=64）
    姓名：{personInfo.name}
    用户ID：{personInfo.personID}
```

## 数据流

### 认证流程

```
用户点击登录
  → 生成 PKCE params → 存 sessionStorage
  → 跳转 OIDC authorize（含 code_challenge, state）
  → IAM 认证后回调 /auth/callback?code=xxx&state=xxx
  → AuthCallback 校验 state → exchangeCodeForTokens
  → setSession 存储 token → navigate /
  → MainLayout useEffect 调用 getUserinfo → setPersonInfo
```

### 静默登录

```
App 加载
  → authStage !== 'authenticated'？
    → 跳过条件：logged_out='1' || oidc_silent_failed='1' || 已在 /auth/callback
    → 生成 PKCE params → buildSilentAuthorizeURL（prompt=none）
    → window.location.replace(url)
    → IAM 检查已有会话，成功则回调 /auth/callback 换 token
    → 失败（error=login_required）则标记 oidc_silent_failed，显示登录页
```

### Token 刷新

```
请求拦截器自动附加 Authorization: Bearer {accessToken}
  → 响应拦截器检测 BizCode.TokenExpired
    → handleTokenExpired：
      - 取 refreshToken
      - refreshTokens(refreshToken)
      - 成功 → updateTokens → 重试原请求
      - 失败 → logout → 跳转 /login
    - 并发刷新队列：isRefreshing 标志 + pendingRequests 等待队列
```

### 登出流程

```
用户点击「退出登录」
  → handleLogout：
    1. logoutAPI(refreshToken) — 通知后端销毁 token（失败忽略）
    2. sessionStorage.setItem('logged_out', '1')
    3. authStore.logout() — 清除所有状态
    4. 若有 idToken → 构造 end_session URL → 动态插入 script 标签加载
```

## 路由设计

| 路径 | 守卫 | 组件 |
|------|------|------|
| `/login` | `authStage === 'authenticated'` → Navigate to `/` | Login |
| `/auth/callback` | 无 | AuthCallback |
| `/` | `authStage !== 'authenticated'` → Navigate to `/login` | MainLayout |
| `/` (index) | 继承父路由 | Home |

## 错误处理

- **BizCode.TokenExpired (110003)**：触发 token 刷新
- **BizCode.TokenInvalid (110002) / Unauthorized (110000)**：清除 store，跳转 `/login`
- **HTTP 401**：清除 store，跳转 `/login`
- **其他业务错误**：显示 message 提示
- **logoutAPI 失败**：忽略错误，继续完成前端登出
