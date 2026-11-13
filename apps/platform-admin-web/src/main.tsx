import React from 'react'
import ReactDOM from 'react-dom/client'
import App from './App'
import { createAuthProvider } from '@morehao/ark-iam-react/auth'
import { AppShell } from '@morehao/ark-iam-react/ui'

const AuthProvider = createAuthProvider({
  // clientID 只是「兜底默认」：运行时 AuthProvider 会先拉 /config.json，其 clientId/issuer 优先；
  // 拉取失败再回退到这里（clientID → VITE_OIDC_CLIENT_ID → 该字面量）。
  // 客户端编码（= OIDC client_id）遵循下划线口径，与 pkg/model.SeedBuiltinClientPlatformAdminWeb 一致。
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
