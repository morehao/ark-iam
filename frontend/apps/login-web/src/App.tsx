import { Navigate, Route, Routes } from 'react-router-dom'
import LoginPage from './pages/LoginPage'

// login-web 仅承载 OP 的登录 UI：/login 凭证表单（+ 多租户选择 + 按应用开关的注册 person / 创建租户）。
// 认证/注册均在 OIDC authorize 流程内完成，见 /oidc/registerPerson、/oidc/createTenant。
function App() {
  return (
    <Routes>
      <Route path="/login" element={<LoginPage />} />
      <Route path="*" element={<Navigate to="/login" replace />} />
    </Routes>
  )
}

export default App
