import React from 'react'
import ReactDOM from 'react-dom/client'
import App from './App'
import { createAuthProvider } from '@ark-iam/auth'

const AuthProvider = createAuthProvider({
  clientId: import.meta.env.VITE_OIDC_CLIENT_ID || 'test-rp-client',
  clientSecret: 'my-test-client-secret',
})

ReactDOM.createRoot(document.getElementById('root')!).render(
  <React.StrictMode>
    <AuthProvider>
      <App />
    </AuthProvider>
  </React.StrictMode>,
)
