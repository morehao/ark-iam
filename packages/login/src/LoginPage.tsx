// LoginPage —— Ark IAM OP（OpenID Provider）账密认证页。
//
// 职责：接收 identifier/password → POST {base}/oidc/login → 拿到 continueURL 整页跳回业务应用。
// 覆盖 4 个 mode：login（账密登录）/ register（自助注册 person）/ createTenant（零租户建租户）
// / changePassword（临时密码首次登录强制改密）；并处理 OIDC 授权码流的分支：
//   - requiresPasswordChange：先改密再回登录；
//   - requiresTenantSelection：选租户后继续；
//   - personID 且无租户（零租户已登录 person）：允许则引导建租户，否则明确报错；
//   - 107004（系统未初始化）：跳 {basePath}/install。
//
// 设计约束（对齐 packages 规范）：
//   - antd Form/Input/Button/Alert/List/Radio + 内联样式，**不引入任何 CSS 文件**；
//   - 品牌区复用 `@ark-iam/ui` 的 `brand`/`tokens`（渐变/卡片/字号/阴影），布局与本仓 SSO 入口页同构；
//   - `@media (max-width:900px)` 隐藏品牌区，用 antd `Grid.useBreakpoint()` 判定；
//   - 不依赖路由 / Provider（仅依赖 antd + 自身 api/config），可挂载时自加载 runtime config。

import { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import {
  Alert,
  Button,
  Divider,
  Form,
  Grid,
  Input,
  List,
  Radio,
  Typography,
} from 'antd'
import { brand, tokens } from '@ark-iam/ui'
import {
  createTenant,
  getLoginConfig,
  installRedirectForError,
  oidcChangePassword,
  oidcLogin,
  oidcSelectTenant,
  registerPerson,
} from './api'
import { appPath, ensureRuntimeConfig, setAppBase } from './config'
import type { TenantSelectionItem } from './types'

const { Title, Text, Paragraph } = Typography

/** 页面 4 个 mode。 */
type Mode = 'login' | 'register' | 'createTenant' | 'changePassword'

/** LoginPage props。 */
export interface LoginPageProps {
  /** 品牌区主标题（默认「统一身份认证」）。 */
  title?: string
  /** 品牌区副标题 / 右侧卡片说明前缀（默认「统一身份认证平台」）。 */
  subtitle?: string
  /** IAM 挂载前缀；默认从 window.location.pathname 推断（/iam/login → /iam）。 */
  basePath?: string
  /** 登录成功（拿到 continueURL）后的回调，默认 window.location.assign(continueURL)。 */
  onLoggedIn?: (continueURL: string) => void
}

/** 从查询串读取 authRequestID。 */
function readAuthRequestID(): string | null {
  if (typeof window === 'undefined') return null
  return new URLSearchParams(window.location.search).get('authRequestID')
}

/** 品牌区（左侧紫色渐变）。移动端（<900px）由父组件判定后不渲染。 */
function BrandPanel({ title, subtitle }: { title: string; subtitle: string }) {
  const features = ['单点登录 SSO', '多租户隔离', '细粒度授权']
  return (
    <div
      style={{
        flex: 1.2,
        display: 'flex',
        flexDirection: 'column',
        justifyContent: 'center',
        padding: '0 8vw',
        background: brand.gradient,
        color: '#fff',
        position: 'relative',
        overflow: 'hidden',
      }}
    >
      <div
        style={{
          position: 'absolute',
          width: 420,
          height: 420,
          borderRadius: '50%',
          background: 'rgba(255,255,255,0.08)',
          top: -120,
          right: -120,
        }}
      />
      <div
        style={{
          position: 'absolute',
          width: 260,
          height: 260,
          borderRadius: '50%',
          background: 'rgba(255,255,255,0.06)',
          bottom: -60,
          left: -60,
        }}
      />
      <div
        style={{
          position: 'relative',
          display: 'inline-flex',
          alignItems: 'center',
          justifyContent: 'center',
          width: 56,
          height: 56,
          borderRadius: 14,
          background: 'rgba(255,255,255,0.18)',
          backdropFilter: 'blur(4px)',
          fontSize: 20,
          fontWeight: 700,
          letterSpacing: 1,
          marginBottom: 28,
        }}
      >
        IAM
      </div>
      <Title level={2} style={{ color: '#fff', margin: '0 0 14px', position: 'relative' }}>
        {title}
      </Title>
      <Paragraph
        style={{ color: 'rgba(255,255,255,0.85)', fontSize: 15, lineHeight: 1.8, maxWidth: 460, position: 'relative' }}
      >
        一个平台管理多应用的账号、权限与安全策略。基于标准 OIDC/OAuth2 协议，支持单点登录与多租户切换。
      </Paragraph>
      <div style={{ display: 'flex', gap: 12, flexWrap: 'wrap', marginTop: 18, position: 'relative' }}>
        {features.map((f) => (
          <span
            key={f}
            style={{
              padding: '8px 16px',
              borderRadius: 20,
              background: 'rgba(255,255,255,0.14)',
              fontSize: 13,
              backdropFilter: 'blur(4px)',
            }}
          >
            {f}
          </span>
        ))}
      </div>
      <Text style={{ color: 'rgba(255,255,255,0.7)', fontSize: 12, marginTop: 28, position: 'relative' }}>
        {subtitle}
      </Text>
    </div>
  )
}

/**
 * Ark IAM OP 账密登录页（4 mode，内联样式，无 CSS，不依赖路由/Provider）。
 */
export function LoginPage({ title, subtitle, basePath, onLoggedIn }: LoginPageProps) {
  const screens = Grid.useBreakpoint()

  const brandTitle = title ?? '统一身份认证'
  const brandSubtitle = subtitle ?? '统一身份认证平台'

  const authRequestID = useMemo(() => readAuthRequestID(), [])

  const [mode, setMode] = useState<Mode>('login')
  const [error, setError] = useState('')
  const [notice, setNotice] = useState('')
  const [loading, setLoading] = useState(false)

  // 租户选择
  const [tenants, setTenants] = useState<TenantSelectionItem[]>([])
  const [pendingAuthRequestID, setPendingAuthRequestID] = useState('')

  // 登录 / 改密「当前密码」共用
  const [identifier, setIdentifier] = useState('')
  const [password, setPassword] = useState('')
  const [newPassword, setNewPassword] = useState('')
  const [confirmPassword, setConfirmPassword] = useState('')

  // 注册表单
  const [regUsername, setRegUsername] = useState('')
  const [regEmail, setRegEmail] = useState('')
  const [regPhone, setRegPhone] = useState('')
  const [regName, setRegName] = useState('')
  const [regPassword, setRegPassword] = useState('')

  // 建租户表单
  const [tenantName, setTenantName] = useState('')
  const [tenantCode, setTenantCode] = useState('')

  // 应用是否开放自助注册
  const [allowRegister, setAllowRegister] = useState(false)

  const mountedRef = useRef(true)

  const redirectTo = useCallback((path: string) => {
    window.location.href = appPath(path)
  }, [])

  const failWith = useCallback(
    (err: unknown, fallback: string) => {
      const redirect = installRedirectForError(err)
      if (redirect) {
        redirectTo(redirect)
        return
      }
      setError(err instanceof Error && err.message ? err.message : fallback)
    },
    [redirectTo],
  )

  // 挂载时：自加载 runtime config（幂等），并在缺失 authRequestID 时提示。
  useEffect(() => {
    mountedRef.current = true
    if (basePath !== undefined) {
      // 宿主显式传入 basePath 时优先使用（覆盖自动探测），随后再加载 runtime config。
      setAppBase(basePath)
    }
    const run = async () => {
      await ensureRuntimeConfig()
      if (!mountedRef.current) return
      if (!authRequestID) {
        setError('缺少认证请求 ID，请从应用重新发起登录')
        return
      }
      try {
        const resp = await getLoginConfig({ authRequestID })
        if (mountedRef.current) setAllowRegister(!!resp.allowPersonCreateTenant)
      } catch (err) {
        if (!mountedRef.current) return
        setAllowRegister(false)
        if (installRedirectForError(err)) redirectTo('/install')
      }
    }
    void run()
    return () => {
      mountedRef.current = false
    }
  }, [authRequestID, basePath, redirectTo])

  const finishLogin = useCallback(
    (continueURL: string) => {
      if (onLoggedIn) {
        onLoggedIn(continueURL)
      } else {
        window.location.assign(continueURL)
      }
    },
    [onLoggedIn],
  )

  const resetToLogin = () => {
    setError('')
    setTenants([])
    setPendingAuthRequestID('')
    setMode('login')
  }

  // ---- login ----
  const handleLogin = async () => {
    if (!authRequestID) {
      setError('缺少认证请求 ID，请从应用重新发起登录')
      return
    }
    if (!identifier.trim() || !password.trim()) {
      setError('请输入用户名和密码')
      return
    }
    setLoading(true)
    setError('')
    setNotice('')
    try {
      const resp = await oidcLogin({ authRequestID, identifier, password })
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
        finishLogin(resp.continueURL)
      } else {
        setError('登录响应异常，请重试')
      }
    } catch (err) {
      failWith(err, '登录失败，请重试')
    } finally {
      setLoading(false)
    }
  }

  // ---- 选租户 ----
  const handleSelectTenant = async (tenantID: string) => {
    if (!pendingAuthRequestID) return
    setLoading(true)
    setError('')
    try {
      const resp = await oidcSelectTenant({ authRequestID: pendingAuthRequestID, tenantID })
      if (resp.continueURL) {
        finishLogin(resp.continueURL)
      } else {
        setError('选择租户响应异常，请重试')
      }
    } catch (err) {
      failWith(err, '选择租户失败，请重试')
    } finally {
      setLoading(false)
    }
  }

  // ---- register ----
  const enableRegister = () => {
    setError('')
    setNotice('')
    setTenants([])
    setPendingAuthRequestID('')
    setMode('register')
  }

  const handleRegister = async () => {
    if (!authRequestID) {
      setError('缺少认证请求 ID，请从应用重新发起登录')
      return
    }
    if (!regUsername.trim() && !regEmail.trim() && !regPhone.trim()) {
      setError('用户名 / 邮箱 / 手机号至少填写一个')
      return
    }
    if (regPassword.length < 8) {
      setError('密码至少 8 位')
      return
    }
    setLoading(true)
    setError('')
    try {
      const resp = await registerPerson({
        authRequestID,
        username: regUsername.trim() || undefined,
        primaryEmail: regEmail.trim() || undefined,
        primaryPhone: regPhone.trim() || undefined,
        name: regName.trim(),
        password: regPassword,
      })
      if (resp.requiresPasswordLogin) {
        setTenants([])
        setPendingAuthRequestID('')
        setMode('login')
        setError('该账号已存在，请使用密码登录')
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
    } catch (err) {
      failWith(err, '注册失败，请重试')
    } finally {
      setLoading(false)
    }
  }

  // ---- changePassword ----
  const handleChangePassword = async () => {
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
      setPassword('')
      setNewPassword('')
      setConfirmPassword('')
      setMode('login')
      setNotice('密码修改成功，请使用新密码登录')
    } catch (err) {
      failWith(err, '修改密码失败，请重试')
    } finally {
      setLoading(false)
    }
  }

  // ---- createTenant ----
  const handleCreateTenant = async () => {
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
        finishLogin(sel.continueURL)
      } else {
        setError('创建租户响应异常，请重试')
      }
    } catch (err) {
      failWith(err, '创建租户失败，请重试')
    } finally {
      setLoading(false)
    }
  }

  const primaryBtnStyle = {
    height: 46,
    borderRadius: 10,
    fontWeight: 600,
    background: brand.gradient,
    border: 'none',
    boxShadow: '0 8px 20px rgba(79,110,247,0.35)',
  } as const

  const titleText =
    mode === 'register'
      ? '注册账号'
      : mode === 'createTenant'
        ? '创建租户'
        : mode === 'changePassword'
          ? '设置新密码'
          : '欢迎回来'
  const subtitleText =
    mode === 'register'
      ? '创建你的统一身份账号'
      : mode === 'createTenant'
        ? '注册成功，请创建你的租户'
        : mode === 'changePassword'
          ? '首次登录需修改初始密码'
          : '使用统一身份账号登录'

  // 窄屏（≤900px）隐藏品牌区：等价旧 CSS 的 `@media (max-width:900px)`。
  // antd 的 Grid 断点没有 900px 档位，故用 useBreakpoint 取 `md`(≥768) 作 SSR 兜底，
  // 再叠加一个精确的 window.innerWidth 监听得到真实的 900px 判定。
  const [narrow, setNarrow] = useState(() =>
    typeof window !== 'undefined' ? window.innerWidth < 900 : screens.md === false,
  )
  useEffect(() => {
    if (typeof window === 'undefined') return
    const onResize = () => setNarrow(window.innerWidth < 900)
    onResize()
    window.addEventListener('resize', onResize)
    return () => window.removeEventListener('resize', onResize)
  }, [])
  const brandVisible = !narrow && screens.md !== false

  return (
    <div
      style={{
        minHeight: '100vh',
        display: 'flex',
        alignItems: 'stretch',
        background: brand.gradientSoft,
      }}
    >
      {brandVisible && <BrandPanel title={brandTitle} subtitle={brandSubtitle} />}

      <div
        style={{
          flex: 1,
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'center',
          padding: 40,
        }}
      >
        <div
          style={{
            width: 400,
            background: tokens.cardBg,
            borderRadius: 16,
            boxShadow: '0 20px 60px rgba(15,23,42,0.12)',
            padding: '40px 36px',
          }}
        >
          <div style={{ fontSize: 20, fontWeight: 700, marginBottom: 8 }}>{titleText}</div>
          <div style={{ fontSize: 14, color: brand.textSecondary, marginBottom: 24 }}>{subtitleText}</div>

          {error && !tenants.length && (
            <Alert type="error" showIcon message={error} style={{ marginBottom: 16 }} />
          )}
          {!error && notice && (
            <Alert type="info" showIcon message={notice} style={{ marginBottom: 16 }} />
          )}

          {tenants.length > 0 ? (
            <TenantSelection tenants={tenants} loading={loading} onSelect={handleSelectTenant} />
          ) : !authRequestID ? (
            <Alert type="error" showIcon message="认证请求无效，请重新发起登录" />
          ) : mode === 'register' ? (
            <Form layout="vertical" onFinish={handleRegister} requiredMark={false}>
              <Form.Item label="用户名（可选）">
                <Input value={regUsername} onChange={(e) => setRegUsername(e.target.value)} placeholder="登录用户名" autoComplete="username" />
              </Form.Item>
              <Form.Item label="邮箱（可选）">
                <Input type="email" value={regEmail} onChange={(e) => setRegEmail(e.target.value)} placeholder="主要邮箱" />
              </Form.Item>
              <Form.Item label="手机号（可选）">
                <Input type="tel" value={regPhone} onChange={(e) => setRegPhone(e.target.value)} placeholder="主要手机号" />
              </Form.Item>
              <Form.Item label="姓名（可选）">
                <Input value={regName} onChange={(e) => setRegName(e.target.value)} placeholder="姓名" />
              </Form.Item>
              <Form.Item label="密码" required>
                <Input.Password value={regPassword} onChange={(e) => setRegPassword(e.target.value)} placeholder="至少 8 位" autoComplete="new-password" />
              </Form.Item>
              <Button type="primary" htmlType="submit" block loading={loading} style={primaryBtnStyle}>
                注册
              </Button>
              <Divider plain style={{ margin: '16px 0' }} />
              <Button type="link" block onClick={resetToLogin} style={{ padding: 0 }}>
                已有账号？返回登录
              </Button>
            </Form>
          ) : mode === 'createTenant' ? (
            <Form layout="vertical" onFinish={handleCreateTenant} requiredMark={false}>
              <Form.Item label="租户名称" required>
                <Input value={tenantName} onChange={(e) => setTenantName(e.target.value)} placeholder="例如：Acme 公司" />
              </Form.Item>
              <Form.Item label="租户编码（可选）">
                <Input value={tenantCode} onChange={(e) => setTenantCode(e.target.value)} placeholder="留空则自动生成" />
              </Form.Item>
              <Button type="primary" htmlType="submit" block loading={loading} style={primaryBtnStyle}>
                创建租户
              </Button>
            </Form>
          ) : mode === 'changePassword' ? (
            <Form layout="vertical" onFinish={handleChangePassword} requiredMark={false}>
              <Form.Item label="当前密码" required>
                <Input.Password value={password} onChange={(e) => setPassword(e.target.value)} placeholder="请输入下发的临时密码" autoComplete="current-password" />
              </Form.Item>
              <Form.Item label="新密码" required>
                <Input.Password value={newPassword} onChange={(e) => setNewPassword(e.target.value)} placeholder="至少 8 位，含大小写与数字" autoComplete="new-password" />
              </Form.Item>
              <Form.Item label="确认新密码" required>
                <Input.Password value={confirmPassword} onChange={(e) => setConfirmPassword(e.target.value)} placeholder="再次输入新密码" autoComplete="new-password" />
              </Form.Item>
              <Button type="primary" htmlType="submit" block loading={loading} style={primaryBtnStyle}>
                设置新密码
              </Button>
              <div style={{ textAlign: 'center', marginTop: 16, fontSize: 12, color: brand.textSecondary }}>
                修改成功后需用新密码重新登录
              </div>
            </Form>
          ) : (
            <>
              <Form layout="vertical" onFinish={handleLogin} requiredMark={false}>
                <Form.Item label="用户名 / 邮箱 / 手机号" required>
                  <Input
                    value={identifier}
                    onChange={(e) => setIdentifier(e.target.value)}
                    placeholder="请输入用户名"
                    autoFocus
                    autoComplete="username"
                  />
                </Form.Item>
                <Form.Item label="密码" required>
                  <Input.Password
                    value={password}
                    onChange={(e) => setPassword(e.target.value)}
                    placeholder="请输入密码"
                    autoComplete="current-password"
                  />
                </Form.Item>
                <Button type="primary" htmlType="submit" block loading={loading} style={primaryBtnStyle}>
                  登录
                </Button>
              </Form>
              {allowRegister && (
                <div style={{ textAlign: 'center', marginTop: 20, fontSize: 13, color: brand.textSecondary }}>
                  没有账号？
                  <Button type="link" onClick={enableRegister} style={{ padding: '0 4px', fontSize: 13 }}>
                    注册账号
                  </Button>
                </div>
              )}
            </>
          )}
        </div>
      </div>
    </div>
  )
}

/** 租户选择：List + Radio 单选，选中即提交继续登录。 */
function TenantSelection({
  tenants,
  loading,
  onSelect,
}: {
  tenants: TenantSelectionItem[]
  loading: boolean
  onSelect: (tenantID: string) => void
}) {
  const [selected, setSelected] = useState<string | undefined>(undefined)
  return (
    <div>
      <Title level={5} style={{ textAlign: 'center', marginBottom: 16 }}>
        请选择要登录的组织/租户
      </Title>
      <Radio.Group
        value={selected}
        onChange={(e) => setSelected(e.target.value as string)}
        style={{ width: '100%' }}
        disabled={loading}
      >
        <List
          dataSource={tenants}
          renderItem={(t) => (
            <List.Item style={{ padding: 0, border: 'none', marginBottom: 12 }}>
              <Radio value={t.tenantID} style={{ width: '100%' }}>
                {t.name}
              </Radio>
            </List.Item>
          )}
        />
      </Radio.Group>
      <Button
        type="primary"
        block
        disabled={!selected}
        loading={loading}
        onClick={() => selected && onSelect(selected)}
        style={{ height: 46, borderRadius: 10, fontWeight: 600, background: brand.gradient, border: 'none', boxShadow: '0 8px 20px rgba(79,110,247,0.35)', marginTop: 4 }}
      >
        进入
      </Button>
    </div>
  )
}

export default LoginPage
