// OIDC 客户端共享配置。
//
// 集中 OIDC 相关默认值，供 AuthProvider 与各业务 App（main.tsx）复用，
// 避免 issuer / 默认路径等在多个文件间漂移。生产多源部署时可通过
// VITE_OIDC_ISSUER 覆盖为 OP 的绝对地址（如 https://iam.example.com/oidc），
// 并配合后端 SSOCookieDomain 使 iam_sso_session 跨域共享。

// defaultIssuer 是开发环境的默认 issuer（相对路径，经 Vite 代理到后端）。
// 生产环境必须用 VITE_OIDC_ISSUER 覆盖为绝对地址。
export const defaultIssuer = '/oidc'

// defaultRedirectPath 是 OIDC 授权回调路径（拼接在 window.location.origin 后）。
export const defaultRedirectPath = '/auth/callback'

// defaultPostLogoutRedirectPath 是登出后回跳路径。
export const defaultPostLogoutRedirectPath = '/login'

// defaultScope 是申请的标准 OIDC scope；offline_access 用于签发 refresh token。
export const defaultScope = 'openid profile email offline_access'
