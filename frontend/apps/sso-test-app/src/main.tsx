import React from 'react'
import ReactDOM from 'react-dom/client'
import { BrowserRouter } from 'react-router-dom'
import { AuthProvider } from 'react-oidc-context'
import { WebStorageStateStore } from 'oidc-client-ts'
import App from './App'

const oidcConfig = {
  authority: import.meta.env.VITE_OIDC_ISSUER || '/oidc',
  client_id: import.meta.env.VITE_OIDC_CLIENT_ID || 'test-rp-client',
  client_secret: 'my-test-client-secret',
  redirect_uri: window.location.origin + '/auth/callback',
  post_logout_redirect_uri: window.location.origin + '/login',
  scope: 'openid profile email offline_access',
  automaticSilentRenew: true,
  monitorSession: false,
  loadUserInfo: true,
  userStore: new WebStorageStateStore({ store: localStorage }),
}

ReactDOM.createRoot(document.getElementById('root')!).render(
  <React.StrictMode>
    <BrowserRouter>
      <AuthProvider {...oidcConfig}>
        <App />
      </AuthProvider>
    </BrowserRouter>
  </React.StrictMode>,
)
