import { Routes, Route, Navigate } from 'react-router'
import MainLayout from './components/MainLayout'
import Login from './pages/auth/Login'
import Register from './pages/auth/Register'
import SelectTenant from './pages/auth/SelectTenant'
import Dashboard from './pages/dashboard'
import UserList from './pages/user'
import RoleList from './pages/role'
import DepartmentList from './pages/department'
import ApplicationList from './pages/application'
import { useAuthStore } from './stores/authStore'

function App() {
  const { personToken, tenantToken } = useAuthStore()

  return (
    <Routes>
      <Route
        path="/login"
        element={tenantToken ? <Navigate to="/" replace /> : personToken ? <Navigate to="/select-tenant" replace /> : <Login />}
      />
      <Route path="/register" element={<Register />} />
      <Route
        path="/select-tenant"
        element={
          tenantToken ? (
            <Navigate to="/" replace />
          ) : personToken ? (
            <SelectTenant />
          ) : (
            <Navigate to="/login" replace />
          )
        }
      />
      <Route
        path="/"
        element={
          tenantToken ? <MainLayout /> : personToken ? <Navigate to="/select-tenant" replace /> : <Navigate to="/login" replace />
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
