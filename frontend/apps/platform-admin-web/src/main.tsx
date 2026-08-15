import React from 'react'
import ReactDOM from 'react-dom/client'
import App from './App'
import { createAuthProvider } from '@ark-iam/auth'
import { AppShell } from '@ark-iam/ui'

const AuthProvider = createAuthProvider({
  clientID: import.meta.env.VITE_OIDC_CLIENT_ID || 'platform-admin-web',
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
