import { useState, type FormEvent } from 'react'
import { Link } from 'react-router-dom'
import { joinWithInvite } from '../api'
import '../LoginPage.css'

export default function JoinTenantPage() {
  const [inviteCode, setInviteCode] = useState('')
  const [error, setError] = useState('')
  const [success, setSuccess] = useState('')
  const [loading, setLoading] = useState(false)

  const handleSubmit = async (e: FormEvent) => {
    e.preventDefault()
    if (!inviteCode.trim()) {
      setError('请输入邀请码')
      return
    }
    setLoading(true)
    setError('')
    setSuccess('')
    try {
      await joinWithInvite({ inviteCode: inviteCode.trim() })
      setSuccess('加入成功。请返回登录后使用新加入的租户登录。')
    } catch (err: any) {
      setError(err?.message || '加入租户失败，请重试')
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
          <h1 className="brand-title">加入租户</h1>
          <p className="brand-desc">
            凭管理员发放的邀请码加入一个已有租户，你将作为普通成员接受角色授权。
          </p>
          <div className="brand-features">
            <span>邀请制</span>
            <span>普通成员</span>
          </div>
        </div>
      </div>

      <div className="login-main">
        <div className="login-card">
          <h2 className="card-title">加入租户</h2>
          <p className="card-subtitle">请向你的租户管理员索取邀请码</p>

          {error && <div className="error-msg">{error}</div>}
          {success && (
            <div className="error-msg" style={{ background: '#f6ffed', borderColor: '#b7eb8f', color: '#389e0d' }}>
              {success}
            </div>
          )}

          <form onSubmit={handleSubmit}>
            <div className="form-group">
              <label htmlFor="inviteCode">邀请码</label>
              <input id="inviteCode" type="text" value={inviteCode} onChange={(e) => setInviteCode(e.target.value)} placeholder="例如：invite-xxxx" required />
            </div>

            <button type="submit" className="login-btn" disabled={loading}>
              {loading ? '加入中...' : '加入租户'}
            </button>
          </form>

          <p style={{ textAlign: 'center', marginTop: 20, fontSize: 13, color: '#8b93a7' }}>
            <Link to="/login" style={{ color: '#4f6ef7', textDecoration: 'none' }}>返回登录</Link>
          </p>
        </div>
      </div>
    </div>
  )
}
