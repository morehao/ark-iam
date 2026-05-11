import { Routes, Route, Navigate } from 'react-router'
import MainLayout from './components/MainLayout'
import Login from './pages/auth/Login'
import Register from './pages/auth/Register'
import Dashboard from './pages/dashboard'
import UserList from './pages/user'
import RoleList from './pages/role'
import DepartmentList from './pages/department'
import ApplicationList from './pages/application'
import { useAuthStore } from './stores/authStore'

function App() {
  const { accessToken } = useAuthStore()

  return (
    <Routes>
      <Route path="/login" element={<Login />} />
      <Route path="/register" element={<Register />} />
      <Route
        path="/"
        element={
          accessToken ? <MainLayout /> : <Navigate to="/login" replace />
        }
      >
        <Route index element={<Dashboard />} />
        <Route path="user" element={<UserList />} />
        <Route path="role" element={<RoleList />} />
        <Route path="department" element={<DepartmentList />} />
        <Route path="application" element={<ApplicationList />} />
      </Route>
    </Routes>
  )
}

export default App