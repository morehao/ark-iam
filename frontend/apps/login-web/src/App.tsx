import { Navigate, Route, Routes } from 'react-router-dom'
import LoginPage from './pages/LoginPage'

// login-web 仅承载 OP 的登录 UI：/login 凭证表单（+ 多租户选择）。
// 所有其它路径（如登出后回跳、直接访问根路径）统一回到 /login。
// 全局登出由各业务应用（OIDC Client）内部触发，登录页自身不再持有会话状态。
function App() {
  return (
    <Routes>
      <Route path="/login" element={<LoginPage />} />
      <Route path="*" element={<Navigate to="/login" replace />} />
    </Routes>
  )
}

export default App
