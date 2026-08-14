import React from 'react'
import ReactDOM from 'react-dom/client'
import App from './App'
import { createAuthProvider } from '@ark-iam/auth'

// login-web 既是 OP 跳转的集中登录页（/login 凭证表单），
// 也是受 SSO 保护的门户 App（/ 门户页 + 全局登出）。
const AuthProvider = createAuthProvider({
  clientId: import.meta.env.VITE_OIDC_CLIENT_ID || 'login-web',
})

ReactDOM.createRoot(document.getElementById('root')!).render(
  <React.StrictMode>
    <AuthProvider>
      <App />
    </AuthProvider>
  </React.StrictMode>,
)
