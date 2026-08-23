import { Navigate, Route, Routes } from 'react-router-dom'
import LoginPage from './pages/LoginPage'
import RegisterOrgPage from './pages/RegisterOrgPage'
import JoinTenantPage from './pages/JoinTenantPage'

// login-web 仅承载 OP 的登录 UI：/login 凭证表单（+ 多租户选择）。
// 自助开通租户（/register/org）与凭邀请加入租户（/join）作为登录页补充入口。
// 所有其它路径统一回到 /login。
// 全局登出由各业务应用（OIDC Client）内部触发，登录页自身不再持有会话状态。
function App() {
  return (
    <Routes>
      <Route path="/login" element={<LoginPage />} />
      <Route path="/register/org" element={<RegisterOrgPage />} />
      <Route path="/join" element={<JoinTenantPage />} />
      <Route path="*" element={<Navigate to="/login" replace />} />
    </Routes>
  )
}

export default App
