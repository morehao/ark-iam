# OIDC 登录接口 — 前后端对接契约

## 接口

`POST /v1/iam/oidc/login`

### 请求

```json
{
  "authRequestID": "ar-1741234567890123000",
  "identifier": "person@example.com",
  "password": "Password1"
}
```

### 成功响应

```json
{
  "code": 0,
  "msg": "success",
  "data": {
    "continueURL": "http://iam.example.com/v1/iam/oidc/authorize/callback?id=ar-1741234567890123000"
  }
}
```

### 失败响应

```json
{
  "code": 100799,
  "msg": "OIDC session not found",
  "data": {}
}
```

## 前端消费流程

1. 浏览器被 OIDC Provider 重定向到前端登录页，URL 中包含 `authRequestID`
2. 前端收集用户名密码后，调用 `POST /v1/iam/oidc/login`
3. 成功后，执行 `window.location.href = response.data.continueURL`
4. 浏览器跳转到 Provider 的 `/authorize/callback`，自动完成 OIDC 授权码流程

## 关键约束

- `continueURL` 必须由后端返回，前端不能自行拼接
- 前端收到 `continueURL` 后应**立即跳转**，不应额外停留或展示中间页面
- `continueURL` 中 `id` 参数即为 `authRequestID`，但前端不应依赖其格式
