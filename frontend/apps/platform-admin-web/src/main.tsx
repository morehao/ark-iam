import React from 'react'
import ReactDOM from 'react-dom/client'
import App from './App'
import { createAuthProvider } from '@ark-iam/auth'
import { AppShell } from '@ark-iam/ui'

const AuthProvider = createAuthProvider({
  // 客户端编码（= OIDC client_id）遵循下划线口径，与 pkg/model.SeedBuiltinClientPlatformAdminWeb 一致
  clientID: import.meta.env.VITE_OIDC_CLIENT_ID || 'platform_admin_web',
})

ReactDOM.createRoot(document.getElementById('root')!).render(
  <React.StrictMode>
    <AppShell>
      <AuthProvider>
        <App />
      </AuthProvider>
    </AppShell>
  </React.StrictMode>,
)
