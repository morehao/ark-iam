import { useState, useEffect, type FormEvent } from 'react'
import { Link } from 'react-router-dom'
import { oidcLogin, oidcSelectTenant } from '../api'
import '../LoginPage.css'

export default function LoginPage() {
  const authRequestID = new URLSearchParams(window.location.search).get('authRequestID')

  const [identifier, setIdentifier] = useState('')
  const [password, setPassword] = useState('')
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(false)
  const [tenants, setTenants] = useState<{ tenantID: string; name: string }[]>([])
  const [pendingAuthRequestID, setPendingAuthRequestID] = useState('')

  useEffect(() => {
    if (!authRequestID) {
      setError('缺少认证请求 ID，请从应用重新发起登录')
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
      if (resp.continueURL) {
        window.location.href = resp.continueURL
      } else {
        setError('登录响应异常，请重试')
      }
    } catch (err: any) {
      setError(err?.message || '登录失败，请重试')
    } finally {
      setLoading(false)
    }
  }

  const selectTenant = async (tenantID: string) => {
    if (!pendingAuthRequestID) return
    setLoading(true)
    setError('')
    try {
      const resp = await oidcSelectTenant({ authRequestID: pendingAuthRequestID, tenantID })
      if (resp.continueURL) {
        window.location.href = resp.continueURL
      } else {
        setError('选择租户响应异常，请重试')
      }
    } catch (err: any) {
      setError(err?.message || '选择租户失败，请重试')
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="login-page">
      <div className="login-brand">
        <div className="brand-orbit orbit-1" />
        <div className="brand-orbit orbit-2" />
        <div className="brand-inner">
          <div className="brand-logo">IAM</div>
          <h1 className="brand-title">统一身份认证</h1>
          <p className="brand-desc">
            一个平台管理多应用的账号、权限与安全策略。
            <br />
            基于标准 OIDC/OAuth2 协议，支持单点登录与多租户切换。
          </p>
          <div className="brand-features">
            <span>单点登录 SSO</span>
            <span>多租户隔离</span>
            <span>细粒度授权</span>
          </div>
        </div>
      </div>

      <div className="login-main">
        <div className="login-card">
          <h2 className="card-title">欢迎回来</h2>
          <p className="card-subtitle">使用统一身份账号登录</p>

          {error && <div className="error-msg">{error}</div>}

          {tenants.length > 0 ? (
            <div className="tenant-selection">
              <h3>请选择要登录的组织/租户</h3>
              <ul className="tenant-list">
                {tenants.map((t) => (
                  <li key={t.tenantID}>
                    <button className="login-btn" onClick={() => selectTenant(t.tenantID)} disabled={loading}>
                      {t.name}
                    </button>
                  </li>
                ))}
              </ul>
            </div>
          ) : authRequestID ? (
            <>
              <form onSubmit={handleSubmit}>
                <div className="form-group">
                  <label htmlFor="identifier">用户名 / 邮箱 / 手机号</label>
                  <input
                    id="identifier"
                    type="text"
                    value={identifier}
                    onChange={(e) => setIdentifier(e.target.value)}
                    placeholder="请输入用户名"
                    autoFocus
                    autoComplete="username"
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
                    autoComplete="current-password"
                  />
                </div>

                <button type="submit" className="login-btn" disabled={loading}>
                  {loading ? '登录中...' : '登录'}
                </button>
              </form>

              <p style={{ textAlign: 'center', marginTop: 20, fontSize: 13, color: '#8b93a7' }}>
                没有账号？<Link to="/register/org" style={{ color: '#4f6ef7', textDecoration: 'none' }}>开通新租户</Link>
                <span style={{ margin: '0 8px' }}>·</span>
                <Link to="/join" style={{ color: '#4f6ef7', textDecoration: 'none' }}>凭邀请加入</Link>
              </p>
            </>
          ) : (
            <div className="error-msg">认证请求无效，请重新发起登录</div>
          )}
        </div>
      </div>
    </div>
  )
}
