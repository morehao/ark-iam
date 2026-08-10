import { useState, useEffect, type FormEvent } from 'react'
import { oidcLogin, oidcSelectTenant } from '../api'
import '../LoginPage.css'

export default function LoginPage() {
  const authRequestID = new URLSearchParams(window.location.search).get('authRequestID')

  const [identifier, setIdentifier] = useState('')
  const [password, setPassword] = useState('')
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(false)
  const [tenants, setTenants] = useState<{ tenantID: number; name: string }[]>([])
  const [pendingAuthRequestID, setPendingAuthRequestID] = useState('')

  useEffect(() => {
    if (!authRequestID) {
      setError('缺少认证请求 ID')
    }
  }, [authRequestID])

  const handleSubmit = async (e: FormEvent) => {
    e.preventDefault()
    if (!authRequestID) {
      setError('缺少认证请求 ID')
      return
    }
    if (!identifier.trim() || !password.trim()) {
      setError('请输入用户名和密码')
      return
    }

    setLoading(true)
    setError('')
    try {
      const resp = await oidcLogin({ authRequestID, identifier, password })
      if (resp.requiresTenantSelection && resp.tenants?.length) {
        setTenants(resp.tenants)
        setPendingAuthRequestID(authRequestID)
        return
      }
      window.location.href = resp.continueURL
    } catch (err: any) {
      const msg = err?.response?.data?.message || err?.message || '登录失败，请重试'
      setError(msg)
    } finally {
      setLoading(false)
    }
  }

  const selectTenant = async (tenantID: number) => {
    if (!pendingAuthRequestID) return
    setLoading(true)
    setError('')
    try {
      const resp = await oidcSelectTenant({ authRequestID: pendingAuthRequestID, tenantID })
      window.location.href = resp.continueURL
    } catch (err: any) {
      const msg = err?.response?.data?.message || err?.message || '选择租户失败，请重试'
      setError(msg)
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="login-container">
      <div className="login-card">
        <h1>IAM 登录</h1>
        <p className="subtitle">使用 IAM 身份提供者登录</p>

        {error && <div className="error-msg">{error}</div>}

        {tenants.length > 0 ? (
          <div className="tenant-selection">
            <h2>请选择要登录的组织/租户</h2>
            <ul className="tenant-list">
              {tenants.map((t) => (
                <li key={t.tenantID}>
                  <button
                    className="login-btn"
                    onClick={() => selectTenant(t.tenantID)}
                    disabled={loading}
                  >
                    {t.name}
                  </button>
                </li>
              ))}
            </ul>
          </div>
        ) : authRequestID ? (
          <form onSubmit={handleSubmit}>
            <div className="form-group">
              <label htmlFor="identifier">用户名/邮箱/手机号</label>
              <input
                id="identifier"
                type="text"
                value={identifier}
                onChange={(e) => setIdentifier(e.target.value)}
                placeholder="请输入用户名"
                autoFocus
              />
            </div>

            <div className="form-group">
              <label htmlFor="password">密码</label>
              <input
                id="password"
                type="password"
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                placeholder="请输入密码"
              />
            </div>

            <button type="submit" className="login-btn" disabled={loading}>
              {loading ? '登录中...' : '登录'}
            </button>
          </form>
        ) : (
          <div className="error-msg">认证请求无效，请重新发起登录</div>
        )}

        <div className="social-section">
          <div className="divider-text">第三方登录（即将支持）</div>
          <div className="social-buttons">
            <button className="social-btn" disabled>
              Google 登录
            </button>
            <button className="social-btn" disabled>
              GitHub 登录
            </button>
          </div>
        </div>
      </div>
    </div>
  )
}
