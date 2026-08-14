import React from 'react'
import ReactDOM from 'react-dom/client'
import { BrowserRouter } from 'react-router-dom'
import App from './App'

// login-web 是 IAM OP 的登录 UI（凭证表单 + 多租户选择），不是 OIDC Client/RP：
// 凭证直接提交到 OP 内部端点 /oidc/login，不参与授权码流程，无 client_id / redirect_uri。
// 因此这里不挂任何 OIDC AuthProvider，业务应用（platform-admin-web / tenant-admin-web）
// 才是 OIDC Client。
ReactDOM.createRoot(document.getElementById('root')!).render(
  <React.StrictMode>
    <BrowserRouter>
      <App />
    </BrowserRouter>
  </React.StrictMode>,
)
