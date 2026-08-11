import React from 'react'
import { BrowserRouter } from 'react-router-dom'
import { AuthProvider as OidcAuthProvider } from 'react-oidc-context'
import { WebStorageStateStore } from 'oidc-client-ts'
import { oidcExtraQueryParams } from './tenant'

export interface CreateAuthProviderOptions {
  clientId: string
  clientSecret?: string
  getExtraQueryParams?: () => Record<string, string | number | boolean>
  redirectPath?: string
}

export function createAuthProvider(opts: CreateAuthProviderOptions) {
  const { clientId, clientSecret, redirectPath = '/auth/callback' } = opts

  const oidcConfig = {
    authority: import.meta.env.VITE_OIDC_ISSUER || '/oidc',
    client_id: clientId,
    ...(clientSecret ? { client_secret: clientSecret } : {}),
    redirect_uri: window.location.origin + redirectPath,
    post_logout_redirect_uri: window.location.origin + '/login',
    scope: 'openid profile email offline_access',
    automaticSilentRenew: true,
    monitorSession: false,
    loadUserInfo: true,
    userStore: new WebStorageStateStore({ store: localStorage }),
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
