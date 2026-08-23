import { useState, type FormEvent } from 'react'
import { Link } from 'react-router-dom'
import { registerOrg } from '../api'
import '../LoginPage.css'

export default function RegisterOrgPage() {
  const [tenantName, setTenantName] = useState('')
  const [tenantCode, setTenantCode] = useState('')
  const [username, setUsername] = useState('')
  const [primaryEmail, setPrimaryEmail] = useState('')
  const [primaryPhone, setPrimaryPhone] = useState('')
  const [name, setName] = useState('')
  const [password, setPassword] = useState('')
  const [confirm, setConfirm] = useState('')
  const [error, setError] = useState('')
  const [success, setSuccess] = useState('')
  const [loading, setLoading] = useState(false)

  const handleSubmit = async (e: FormEvent) => {
    e.preventDefault()
    if (!tenantName.trim()) {
      setError('请填写租户名称')
      return
    }
    if (!username.trim() && !primaryEmail.trim() && !primaryPhone.trim()) {
      setError('用户名 / 邮箱 / 手机号至少填写一个')
      return
    }
    if (password.length < 8) {
      setError('密码至少 8 位')
      return
    }
    if (password !== confirm) {
      setError('两次输入的密码不一致')
      return
    }

    setLoading(true)
    setError('')
    setSuccess('')
    try {
      const resp = await registerOrg({
        tenantName: tenantName.trim(),
        tenantCode: tenantCode.trim() || undefined,
        username: username.trim() || undefined,
        primaryEmail: primaryEmail.trim() || undefined,
        primaryPhone: primaryPhone.trim() || undefined,
        name: name.trim(),
        password,
      })
      setSuccess(`注册成功，已开通租户 "${resp.tenantID ? '（ID ' + resp.tenantID + '）' : ''}"，请返回登录。`)
    } catch (err: any) {
      setError(err?.message || '注册失败，请重试')
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
          <h1 className="brand-title">开通新租户</h1>
          <p className="brand-desc">
            注册一个新租户，你将自动成为该租户的拥有者（Owner），
            可配置 SSO、组织架构与用户。
          </p>
          <div className="brand-features">
            <span>租户隔离</span>
            <span>拥有者权限</span>
            <span>组织管理</span>
          </div>
        </div>
      </div>

      <div className="login-main">
        <div className="login-card">
          <h2 className="card-title">开通租户</h2>
          <p className="card-subtitle">填写租户与管理员信息</p>

          {error && <div className="error-msg">{error}</div>}
          {success && (
            <div className="error-msg" style={{ background: '#f6ffed', borderColor: '#b7eb8f', color: '#389e0d' }}>
              {success}
            </div>
          )}

          <form onSubmit={handleSubmit}>
            <div className="form-group">
              <label htmlFor="tenantName">租户名称</label>
              <input id="tenantName" type="text" value={tenantName} onChange={(e) => setTenantName(e.target.value)} placeholder="例如：Acme 公司" required />
            </div>

            <div className="form-group">
              <label htmlFor="tenantCode">租户编码（可选）</label>
              <input id="tenantCode" type="text" value={tenantCode} onChange={(e) => setTenantCode(e.target.value)} placeholder="留空则自动生成" />
            </div>

            <div className="form-group">
              <label htmlFor="name">姓名</label>
              <input id="name" type="text" value={name} onChange={(e) => setName(e.target.value)} placeholder="管理员姓名" />
            </div>

            <div className="form-group">
              <label htmlFor="username">用户名</label>
              <input id="username" type="text" value={username} onChange={(e) => setUsername(e.target.value)} placeholder="登录用户名（可选）" />
            </div>

            <div className="form-group">
              <label htmlFor="primaryEmail">邮箱</label>
              <input id="primaryEmail" type="email" value={primaryEmail} onChange={(e) => setPrimaryEmail(e.target.value)} placeholder="主要邮箱（可选）" />
            </div>

            <div className="form-group">
              <label htmlFor="primaryPhone">手机号</label>
              <input id="primaryPhone" type="tel" value={primaryPhone} onChange={(e) => setPrimaryPhone(e.target.value)} placeholder="主要手机号（可选）" />
            </div>

            <div className="form-group">
              <label htmlFor="password">密码</label>
              <input id="password" type="password" value={password} onChange={(e) => setPassword(e.target.value)} placeholder="至少 8 位，包含大小写与数字" autoComplete="new-password" required />
            </div>

            <div className="form-group">
              <label htmlFor="confirm">确认密码</label>
              <input id="confirm" type="password" value={confirm} onChange={(e) => setConfirm(e.target.value)} placeholder="再次输入密码" autoComplete="new-password" required />
            </div>

            <button type="submit" className="login-btn" disabled={loading}>
              {loading ? '开通中...' : '开通租户'}
            </button>
          </form>

          <p style={{ textAlign: 'center', marginTop: 20, fontSize: 13, color: '#8b93a7' }}>
            已有账号？<Link to="/login" style={{ color: '#4f6ef7', textDecoration: 'none' }}>返回登录</Link>
          </p>
        </div>
      </div>
    </div>
  )
}
