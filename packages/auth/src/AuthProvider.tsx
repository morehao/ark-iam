import React, { useEffect, useState } from 'react'
import { BrowserRouter } from 'react-router-dom'
import { AuthProvider as OidcAuthProvider } from 'react-oidc-context'
import { WebStorageStateStore } from 'oidc-client-ts'
import {
  defaultIssuer,
  defaultRedirectPath,
  defaultPostLogoutRedirectPath,
  defaultScope,
  fetchRuntimeConfig,
  type RuntimeConfig,
} from './config'
import { oidcExtraQueryParams } from './tenant'

export interface CreateAuthProviderOptions {
  /**
   * OIDC client_id（控制台应用的 application_client.code）。
   * 未传时优先用 /config.json 的 clientId，再退回 import.meta.env.VITE_OIDC_CLIENT_ID。
   */
  clientID?: string
  clientSecret?: string
  /**
   * OIDC issuer（OP 地址）。未传时优先级：/config.json.issuer > VITE_OIDC_ISSUER > '/oidc'。
   * 多源部署时可显式传 OP 的绝对地址（如 https://iam.example.com/oidc）。
   */
  issuer?: string
  redirectPath?: string
  postLogoutRedirectPath?: string
  getExtraQueryParams?: () => Record<string, string | number | boolean>
  /**
   * 是否在启动时拉取 `{base}/config.json` 作为运行时配置来源（默认 true）。
   * 置 false 则完全走 props / VITE_* / 默认值，适用于无 IAM 托管 / 无 config.json 的场景。
   */
  useRuntimeConfig?: boolean
  /** config.json 的拉取基址；默认 import.meta.env.BASE_URL。 */
  runtimeConfigBase?: string
}

const ENV = (import.meta as any)?.env ?? {}

// buildConfig 把「props > /config.json > VITE_* > 相对默认值」的优先级收敛为一层。
function buildConfig(opts: CreateAuthProviderOptions, runtime: RuntimeConfig | null) {
  const issuer = opts.issuer || runtime?.issuer || ENV.VITE_OIDC_ISSUER || defaultIssuer
  const clientId =
    opts.clientID || runtime?.clientId || ENV.VITE_OIDC_CLIENT_ID || ''
  const redirectPath = opts.redirectPath || runtime?.callbackPath || defaultRedirectPath
  const postLogoutRedirectPath = opts.postLogoutRedirectPath || defaultPostLogoutRedirectPath
  return { issuer, clientId, redirectPath, postLogoutRedirectPath }
}

export function createAuthProvider(opts: CreateAuthProviderOptions = {}) {
  return function AuthProvider({ children }: { children: React.ReactNode }) {
    const useRuntime = opts.useRuntimeConfig !== false
    // runtime 为 undefined 表示「尚未就绪」（加载中）；null 表示「已尝试但不可用」，走回退。
    const [runtime, setRuntime] = useState<RuntimeConfig | null | undefined>(
      useRuntime ? undefined : null,
    )

    useEffect(() => {
      if (!useRuntime) return
      let cancelled = false
      void fetchRuntimeConfig(opts.runtimeConfigBase).then((cfg) => {
        if (!cancelled) setRuntime(cfg)
      })
      return () => {
        cancelled = true
      }
      // 仅挂载时拉取一次：运行时配置在应用生命周期内不变。
      // eslint-disable-next-line react-hooks/exhaustive-deps
    }, [])

    // 加载中：先渲染 children 会拿到未就绪配置（issuer/clientId 可能为默认值），
    // 故等待 config.json 拉取完成（成功或失败都落定）再接 AuthProvider，避免一次错误的授权跳转。
    if (runtime === undefined) return null

    const { issuer, clientId, redirectPath, postLogoutRedirectPath } = buildConfig(opts, runtime)

    const oidcConfig = {
      authority: issuer,
      client_id: clientId,
      ...(opts.clientSecret ? { client_secret: opts.clientSecret } : {}),
      redirect_uri: window.location.origin + redirectPath,
      post_logout_redirect_uri: window.location.origin + postLogoutRedirectPath,
      scope: defaultScope,
      automaticSilentRenew: true,
      // monitorSession 需要 OP 在 discovery 暴露 check_session_iframe（OIDC Session
      // Management）。当前 OP（zitadel）未启用该端点，oidc-client-ts 会自动降级为不监控。
      // "一处登出、处处登出"的即时性主要由后端请求粒度 SSO 活性校验保障
      // （WithOIDCSSOValidation + HasActiveSession），此配置仅作跨标签页监控的尽力而为。
      monitorSession: true,
      loadUserInfo: true,
      // 令牌存储：使用 sessionStorage（标签页级，关闭即清）而非 localStorage，
      // 缩小 XSS 窃取令牌的持续暴露窗口。SSO 认证态本身在 OP 的 iam_sso_session
      // cookie，不依赖前端存储；页面刷新/静默续期不受影响。
      userStore: new WebStorageStateStore({ store: sessionStorage }),
      extraQueryParams: {
        ...oidcExtraQueryParams,
        ...(opts.getExtraQueryParams ? opts.getExtraQueryParams() : {}),
      },
    }

    return (
      <BrowserRouter>
        <OidcAuthProvider {...oidcConfig}>{children}</OidcAuthProvider>
      </BrowserRouter>
    )
  }
}
