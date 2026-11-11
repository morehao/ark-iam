import React from 'react'
import ReactDOM from 'react-dom/client'
import { BrowserRouter } from 'react-router-dom'
import { getAppBase, loadRuntimeConfig } from '@ark-iam/login'
import App from './App'
import './base.css'

// login-web 是 IAM OP 的登录 UI（凭证表单 + 多租户选择），不是 OIDC Client/RP：
// 凭证直接提交到 OP 内部端点 {base}/oidc/login，不参与授权码流程，无 client_id / redirect_uri。
// 因此这里不挂任何 OIDC AuthProvider，业务应用（platform-admin-web / tenant-admin-web）才是 OIDC Client。
//
// 内嵌形态自适应挂载前缀：先探测自身 basePath 并拉取 {basePath}/config.json（运行时下发
// oidcBase/installBase 等），再据此配置路由 basename 与 axios 基址；根路径挂载时 basePath=""。
async function bootstrap() {
  await loadRuntimeConfig()
  const base = getAppBase() // "" 或 "/iam" 等
  ReactDOM.createRoot(document.getElementById('root')!).render(
    <React.StrictMode>
      <BrowserRouter basename={base || undefined}>
        <App />
      </BrowserRouter>
    </React.StrictMode>,
  )
}

void bootstrap()
