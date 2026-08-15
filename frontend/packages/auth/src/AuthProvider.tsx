import React from 'react'
import { BrowserRouter } from 'react-router-dom'
import { AuthProvider as OidcAuthProvider } from 'react-oidc-context'
import { WebStorageStateStore } from 'oidc-client-ts'
import { defaultIssuer, defaultRedirectPath, defaultPostLogoutRedirectPath, defaultScope } from './config'
import { oidcExtraQueryParams } from './tenant'

export interface CreateAuthProviderOptions {
  clientID: string
  clientSecret?: string
  getExtraQueryParams?: () => Record<string, string | number | boolean>
  redirectPath?: string
}

export function createAuthProvider(opts: CreateAuthProviderOptions) {
  const { clientID, clientSecret, redirectPath = defaultRedirectPath } = opts

  const oidcConfig = {
    authority: import.meta.env.VITE_OIDC_ISSUER || defaultIssuer,
    client_id: clientID,
    ...(clientSecret ? { client_secret: clientSecret } : {}),
    redirect_uri: window.location.origin + redirectPath,
    post_logout_redirect_uri: window.location.origin + defaultPostLogoutRedirectPath,
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
    extraQueryParams: { ...oidcExtraQueryParams, ...(opts.getExtraQueryParams ? opts.getExtraQueryParams() : {}) },
  }

  return function AuthProvider({ children }: { children: React.ReactNode }) {
    return (
      <BrowserRouter>
        <OidcAuthProvider {...oidcConfig}>{children}</OidcAuthProvider>
      </BrowserRouter>
    )
  }
}
