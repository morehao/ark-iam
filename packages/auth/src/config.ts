// OIDC 客户端共享配置。
//
// 集中 OIDC 相关默认值，供 AuthProvider 与各业务 App（main.tsx）复用，
// 避免 issuer / 默认路径等在多个文件间漂移。
//
// 配置优先级（越靠前越优先）：
//   1. createAuthProvider(opts) 显式传入的 props（issuer / clientId / redirectPath ...）
//   2. /config.json 运行时配置（createAuthProvider 启动先 fetch，见 fetchRuntimeConfig）
//   3. import.meta.env.VITE_* 构建期常量（仅仓内 / 自托管构建期的便捷覆盖）
//   4. 下方相对默认值（开发环境经 Vite 代理到后端）
//
// 外部接入方通常无需设任何 VITE_*：只需提供 clientID，或让后端 /config.json 下发。

// defaultIssuer 是开发环境的默认 issuer（相对路径，经 Vite 代理到后端）。
export const defaultIssuer = '/oidc'

// defaultRedirectPath 是 OIDC 授权回调路径（拼接在 window.location.origin 后）。
export const defaultRedirectPath = '/auth/callback'

// defaultPostLogoutRedirectPath 是登出后回跳路径。
export const defaultPostLogoutRedirectPath = '/login'

// defaultScope 是申请的标准 OIDC scope；offline_access 用于签发 refresh token。
export const defaultScope = 'openid profile email offline_access'

/**
 * RuntimeConfig 是后端 /config.json 下发的运行时配置（见 backend/iam/app_frontend.go）。
 * 字段全部可选：缺失时逐字段回退到 VITE_* / 相对默认值。
 */
export interface RuntimeConfig {
  issuer?: string
  apiBase?: string
  oidcBase?: string
  callbackPath?: string
  clientId?: string
}

/**
 * fetchRuntimeConfig 从 `{base}/config.json` 拉取运行时配置。
 *
 * - base 默认取 `import.meta.env.BASE_URL`（Vite 注入；自托管常为 '/'），保证子路径部署也能命中。
 * - 任一环节失败（网络、非 2xx、非 JSON）都 resolve 为 null，由调用方回退默认值，绝不 reject，
 *   避免因配置端点不可用而阻断整个应用启动。
 */
export async function fetchRuntimeConfig(base?: string): Promise<RuntimeConfig | null> {
  // import.meta.env.BASE_URL 在 Vite 下恒为 '/'（或显式 base），保证末尾带 '/'。
  const root: string = base ?? ((import.meta as any)?.env?.BASE_URL ?? '/')
  try {
    const res = await fetch(`${root}${root.endsWith('/') ? '' : '/'}config.json`, {
      credentials: 'same-origin',
      headers: { Accept: 'application/json' },
    })
    if (!res.ok) return null
    return (await res.json()) as RuntimeConfig
  } catch {
    return null
  }
}
