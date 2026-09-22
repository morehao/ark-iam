import { useState, useEffect, type FormEvent } from 'react'
import {
  oidcLogin,
  oidcSelectTenant,
  registerPerson,
  createTenant,
  getLoginConfig,
  oidcChangePassword,
  installRedirectForError,
} from '../api'
import '../LoginPage.css'

type Mode = 'login' | 'register' | 'createTenant' | 'changePassword'

export default function LoginPage() {
  const authRequestID = new URLSearchParams(window.location.search).get('authRequestID')

  const [identifier, setIdentifier] = useState('')
  const [password, setPassword] = useState('')
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(false)
  const [tenants, setTenants] = useState<{ tenantID: string; name: string }[]>([])
  const [pendingAuthRequestID, setPendingAuthRequestID] = useState('')

  // 应用是否开放自助注册：登录前查询，决定是否展示"注册账号"入口
  const [allowRegister, setAllowRegister] = useState(false)

  // 注册 person / 创建租户两步表单
  const [mode, setMode] = useState<Mode>('login')
  const [username, setUsername] = useState('')
  const [primaryEmail, setPrimaryEmail] = useState('')
  const [primaryPhone, setPrimaryPhone] = useState('')
  const [name, setName] = useState('')
  const [registerPassword, setRegisterPassword] = useState('')
  const [tenantName, setTenantName] = useState('')
  const [tenantCode, setTenantCode] = useState('')

  // 首次登录强制改密（临时密码）：当前密码 + 新密码 + 确认新密码
  const [newPassword, setNewPassword] = useState('')
  const [confirmPassword, setConfirmPassword] = useState('')
  const [notice, setNotice] = useState('')

  // 系统尚未初始化（107004）时业务端点整体不可用：把用户送回 /install，而不是停在登录表单反复失败。
  const failWith = (err: unknown, fallback: string) => {
    const redirect = installRedirectForError(err)
    if (redirect) {
      window.location.href = redirect
      return
    }
    setError(err instanceof Error && err.message ? err.message : fallback)
  }

  useEffect(() => {
    if (!authRequestID) {
      setError('缺少认证请求 ID，请从应用重新发起登录')
      return
    }
    getLoginConfig({ authRequestID })
      .then((resp) => setAllowRegister(resp.allowPersonCreateTenant))
      .catch((err) => {
        setAllowRegister(false)
        if (installRedirectForError(err)) {
          window.location.href = '/install'
        }
      })
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
      // 临时密码首次登录：先把密码改掉，再回登录页用新密码登录（会话在改密时已全局撤销）
      if (resp.requiresPasswordChange) {
        setMode('changePassword')
        setNotice('该账号正在使用初始临时密码，请先设置新密码')
        return
      }
      if (resp.requiresTenantSelection && resp.tenants?.length) {
        setTenants(resp.tenants)
        setPendingAuthRequestID(authRequestID)
        return
      }
      // 零租户已登录 person：无租户成员（响应带 personID）。应用允许则引导建租户，否则给出明确提示。
      if (resp.personID && !resp.tenants?.length) {
        if (resp.allowPersonCreateTenant) {
          setTenants([])
          setPendingAuthRequestID('')
          setMode('createTenant')
        } else {
          setError('该账号暂无可用租户，且当前应用未开放创建租户')
        }
        return
      }
      if (resp.continueURL) {
        window.location.href = resp.continueURL
      } else {
        setError('登录响应异常，请重试')
      }
    } catch (err: any) {
      failWith(err, '登录失败，请重试')
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
      failWith(err, '选择租户失败，请重试')
    } finally {
      setLoading(false)
    }
  }

  const enableRegister = () => {
    setError('')
    setNotice('')
    setTenants([])
    setPendingAuthRequestID('')
    setMode('register')
  }

  const submitRegister = async (e: FormEvent) => {
    e.preventDefault()
    if (!authRequestID) {
      setError('缺少认证请求 ID，请从应用重新发起登录')
      return
    }
    if (!username.trim() && !primaryEmail.trim() && !primaryPhone.trim()) {
      setError('用户名 / 邮箱 / 手机号至少填写一个')
      return
    }
    if (registerPassword.length < 8) {
      setError('密码至少 8 位')
      return
    }

    setLoading(true)
    setError('')
    try {
      const resp = await registerPerson({
        authRequestID,
        username: username.trim() || undefined,
        primaryEmail: primaryEmail.trim() || undefined,
        primaryPhone: primaryPhone.trim() || undefined,
        name: name.trim(),
        password: registerPassword,
      })
      if (resp.requiresPasswordLogin) {
        setError('该账号已存在，请使用密码登录')
        setTenants([])
        setPendingAuthRequestID('')
        setMode('login')
        return
      }
      if (resp.tenants?.length) {
        setTenants(resp.tenants.map((t) => ({ tenantID: t.tenantID, name: t.name })))
        setPendingAuthRequestID(authRequestID)
        setMode('login')
        return
      }
      if (resp.allowPersonCreateTenant) {
        setMode('createTenant')
        return
      }
      setError('当前应用未开放自助注册')
    } catch (err: any) {
      failWith(err, '注册失败，请重试')
    } finally {
      setLoading(false)
    }
  }

  const submitChangePassword = async (e: FormEvent) => {
    e.preventDefault()
    if (!authRequestID) {
      setError('缺少认证请求 ID，请从应用重新发起登录')
      return
    }
    if (!password.trim() || !newPassword.trim()) {
      setError('请填写当前密码与新密码')
      return
    }
    if (newPassword.length < 8) {
      setError('新密码至少 8 位')
      return
    }
    if (newPassword !== confirmPassword) {
      setError('两次输入的新密码不一致')
      return
    }
    if (newPassword === password) {
      setError('新密码不能与当前密码相同')
      return
    }

    setLoading(true)
    setError('')
    try {
      await oidcChangePassword({ authRequestID, currentPassword: password, newPassword })
      // 改密成功：清空口令并回到登录表单，用新密码重新登录
      setPassword('')
      setNewPassword('')
      setConfirmPassword('')
      setMode('login')
      setNotice('密码修改成功，请使用新密码登录')
    } catch (err: any) {
      failWith(err, '修改密码失败，请重试')
    } finally {
      setLoading(false)
    }
  }

  const submitCreateTenant = async (e: FormEvent) => {
    e.preventDefault()
    if (!authRequestID) {
      setError('缺少认证请求 ID，请从应用重新发起登录')
      return
    }
    if (!tenantName.trim()) {
      setError('请填写租户名称')
      return
    }

    setLoading(true)
    setError('')
    try {
      const resp = await createTenant({
        authRequestID,
        tenantName: tenantName.trim(),
        tenantCode: tenantCode.trim() || undefined,
      })
      const sel = await oidcSelectTenant({ authRequestID, tenantID: resp.tenantID })
      if (sel.continueURL) {
        window.location.href = sel.continueURL
      } else {
        setError('创建租户响应异常，请重试')
      }
    } catch (err: any) {
      failWith(err, '创建租户失败，请重试')
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
          <h2 className="card-title">
            {mode === 'register'
              ? '注册账号'
              : mode === 'createTenant'
                ? '创建租户'
                : mode === 'changePassword'
                  ? '设置新密码'
                  : '欢迎回来'}
          </h2>
          <p className="card-subtitle">
            {mode === 'register'
              ? '创建你的统一身份账号'
              : mode === 'createTenant'
                ? '注册成功，请创建你的租户'
                : mode === 'changePassword'
                  ? '首次登录需修改初始密码'
                  : '使用统一身份账号登录'}
          </p>

          {error && <div className="error-msg">{error}</div>}
          {!error && notice && <div className="notice-msg">{notice}</div>}

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
            mode === 'register' ? (
              <form onSubmit={submitRegister}>
                <div className="form-group">
                  <label htmlFor="reg-username">用户名（可选）</label>
                  <input id="reg-username" type="text" value={username} onChange={(e) => setUsername(e.target.value)} placeholder="登录用户名" autoComplete="username" />
                </div>
                <div className="form-group">
                  <label htmlFor="reg-email">邮箱（可选）</label>
                  <input id="reg-email" type="email" value={primaryEmail} onChange={(e) => setPrimaryEmail(e.target.value)} placeholder="主要邮箱" />
                </div>
                <div className="form-group">
                  <label htmlFor="reg-phone">手机号（可选）</label>
                  <input id="reg-phone" type="tel" value={primaryPhone} onChange={(e) => setPrimaryPhone(e.target.value)} placeholder="主要手机号" />
                </div>
                <div className="form-group">
                  <label htmlFor="reg-name">姓名（可选）</label>
                  <input id="reg-name" type="text" value={name} onChange={(e) => setName(e.target.value)} placeholder="姓名" />
                </div>
                <div className="form-group">
                  <label htmlFor="reg-password">密码</label>
                  <input id="reg-password" type="password" value={registerPassword} onChange={(e) => setRegisterPassword(e.target.value)} placeholder="至少 8 位" autoComplete="new-password" required />
                </div>
                <button type="submit" className="login-btn" disabled={loading}>
                  {loading ? '注册中...' : '注册'}
                </button>
              </form>
            ) : mode === 'createTenant' ? (
              <form onSubmit={submitCreateTenant}>
                <div className="form-group">
                  <label htmlFor="tenantName">租户名称</label>
                  <input id="tenantName" type="text" value={tenantName} onChange={(e) => setTenantName(e.target.value)} placeholder="例如：Acme 公司" required />
                </div>
                <div className="form-group">
                  <label htmlFor="tenantCode">租户编码（可选）</label>
                  <input id="tenantCode" type="text" value={tenantCode} onChange={(e) => setTenantCode(e.target.value)} placeholder="留空则自动生成" />
                </div>
                <button type="submit" className="login-btn" disabled={loading}>
                  {loading ? '创建中...' : '创建租户'}
                </button>
              </form>
            ) : mode === 'changePassword' ? (
              <form onSubmit={submitChangePassword}>
                <div className="form-group">
                  <label htmlFor="current-password">当前密码</label>
                  <input
                    id="current-password"
                    type="password"
                    value={password}
                    onChange={(e) => setPassword(e.target.value)}
                    placeholder="请输入下发的临时密码"
                    autoComplete="current-password"
                  />
                </div>
                <div className="form-group">
                  <label htmlFor="new-password">新密码</label>
                  <input
                    id="new-password"
                    type="password"
                    value={newPassword}
                    onChange={(e) => setNewPassword(e.target.value)}
                    placeholder="至少 8 位，含大小写与数字"
                    autoComplete="new-password"
                  />
                </div>
                <div className="form-group">
                  <label htmlFor="confirm-password">确认新密码</label>
                  <input
                    id="confirm-password"
                    type="password"
                    value={confirmPassword}
                    onChange={(e) => setConfirmPassword(e.target.value)}
                    placeholder="再次输入新密码"
                    autoComplete="new-password"
                  />
                </div>
                <button type="submit" className="login-btn" disabled={loading}>
                  {loading ? '提交中...' : '设置新密码'}
                </button>
                <p style={{ textAlign: 'center', marginTop: 16, fontSize: 12, color: '#8b93a7' }}>
                  修改成功后需用新密码重新登录
                </p>
              </form>
            ) : (
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

                {allowRegister && (
                  <p style={{ textAlign: 'center', marginTop: 20, fontSize: 13, color: '#8b93a7' }}>
                    没有账号？
                    <button type="button" onClick={enableRegister} style={{ color: '#4f6ef7', textDecoration: 'none', background: 'none', border: 'none', cursor: 'pointer', fontSize: 13, padding: 0 }}>
                      注册账号
                    </button>
                  </p>
                )}
              </>
            )
          ) : (
            <div className="error-msg">认证请求无效，请重新发起登录</div>
          )}
        </div>
      </div>
    </div>
  )
}
