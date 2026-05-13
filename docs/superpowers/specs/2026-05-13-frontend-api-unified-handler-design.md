# 前端 API 统一处理层设计

## 背景

当前前端项目（React + Axios）的响应拦截器仅处理了 HTTP 层面的错误（4xx/5xx），对于后端返回的业务错误（code != 0）没有任何统一处理。所有调用方假设请求必定成功，直接消费 `resp.data`，没有检查 `resp.code`。这导致业务错误静默失败，用户体验差。

## 目标

1. 响应拦截器统一拦截 `code != 0` 的业务错误，按错误类别自动处理
2. 实现 token 过期自动刷新重试机制
3. API 函数补充泛型类型，获得 IDE 类型提示
4. 现有调用方代码零改动

## 特殊错误码分类处理

| Code | 含义 | 处理方式 |
|------|------|---------|
| 0 | 成功 | resolve，返回完整 { code, msg, data } |
| 110003 (TokenExpiredErr) | Token 过期 | 自动刷新 token，重试原请求；刷新失败则清会话 |
| 110000 (UnauthorizedErr) | 未认证 | 清会话（回退到 person 阶段） |
| 110002 (TokenInvalidErr) | Token 无效 | 清会话 |
| 110001 (ForbiddenErr) | 禁止访问 | message.warning('暂无权限访问') |
| 110004 (PermissionDeniedErr) | 无权限 | message.warning('暂无权限访问') |
| 100104 (ParamInvalidErr) | 参数校验失败 | message.error(后端返回的 msg) |
| 其他任意业务 code | 普通业务失败 | message.error(后端返回的 msg) |

## 架构

```
页面/组件
    ↓ POST/GET（async/await）
API 函数 (src/api/*.ts)
    ↓ request.post<UserReq, ApiResponse<UserResp>>
Axios 实例 (src/utils/request.ts)
    ├── 请求拦截器
    │   └── 注入 Bearer token（不变）
    └── 响应拦截器（重构）
         ├── code === 0 → resolve，返回完整 { code, msg, data }
         ├── TokenExpired → handleTokenExpired(config)
         │   ├── isRefreshing=false → 发 refresh 请求 → 更新 token → 重试所有排队请求 → 重试当前请求
         │   └── isRefreshing=true  → 排队等待，refresh 成功后批量重试
         ├── TokenInvalid / Unauthorized → clearTenantSession → reject
         ├── Forbidden / PermissionDenied → message.warning('暂无权限') → reject
         └── 其他 → message.error(msg) → reject
```

## 文件变更

### 1. 新增 `src/utils/response.ts`

定义 `ApiResponse<T>` 接口和 `BizCode` 枚举。

### 2. 修改 `src/utils/request.ts`

- 新增 `handleTokenExpired` 函数，包含 `isRefreshing` 锁和 `pendingRequests` 队列
- 重构响应拦截器的 `fulfilled` 回调，按 BizCode 分发处理
- HTTP 错误分支（4xx/5xx）保持原逻辑不变

### 3. 修改 `src/api/*.ts`

所有 API 函数补充泛型签名：`request.post<Req, ApiResponse<Resp>>(...)`

涉及文件：`auth.ts`, `user.ts`, `role.ts`, `department.ts`, `application.ts`

### 4. 新增 `src/utils/request.test.ts`

单元测试覆盖：
- code === 0 正常 resolve
- TokenExpired → refresh 成功 → 重试成功
- TokenExpired → 无 refreshToken → 清会话
- TokenExpired → refresh 失败 → 清会话
- Forbidden → message.warning + reject
- ParamInvalid → message.error + reject
- HTTP 401 → clearTenantSession

## 调用方影响

零改动。现有页面代码保持不变：

```typescript
// 之前
const resp = await getUserPageList(params)
setData(resp.data.list)

// 之后 — 完全不变
const resp = await getUserPageList(params)
setData(resp.data.list)
```

业务错误由拦截器统一弹窗处理，调用方不需要 catch。

## 不做的功能

- 不引入 react-query / swr 等请求缓存库
- 不引入 useRequest 等自定义 hook
- 不对现有页面组件做任何改动
- 不重构 authStore 或路由守卫