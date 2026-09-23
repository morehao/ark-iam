import { Navigate, Route, Routes } from 'react-router-dom'
import LoginPage from './pages/LoginPage'
import InstallPage from './pages/InstallPage'
import { InstallGuard } from './install/InstallGuard'

// login-web 承载 OP 的登录 UI 与首次部署的初始化向导：
//   /install  三步向导（仅第 ③ 步发写请求），未初始化时登录入口也跳到这里；
//   /login    凭证表单（+ 多租户选择 + 按应用开关的注册 person / 创建租户），由 InstallGuard 包住。
// 认证/注册均在 OIDC authorize 流程内完成，见 /oidc/registerPerson、/oidc/createTenant。
function App() {
  return (
    <Routes>
      <Route path="/install" element={<InstallPage />} />
      <Route
        path="/login"
        element={
          <InstallGuard>
            <LoginPage />
          </InstallGuard>
        }
      />
      <Route path="*" element={<Navigate to="/login" replace />} />
    </Routes>
  )
}

export default App
