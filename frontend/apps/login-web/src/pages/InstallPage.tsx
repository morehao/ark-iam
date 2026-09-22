import { useCallback, useEffect, useState, type FormEvent } from 'react'
import { INSTALL_PENDING_REDIRECT, getInstallStatus, initializeInstall } from '../api'
import { resolveInstallError, statusUnavailableMessage } from '../install/errors'
import { INSTALL_MESSAGES } from '../install/messages'
import {
  INSTALL_VIEW_TEXT,
  consoleEntries,
  resolveInstallView,
  resolveLoginURL,
  type InstallView,
} from '../install/status'
import {
  FIRST_WIZARD_STEP,
  LAST_WIZARD_STEP,
  WIZARD_STEPS,
  buildInitializeRequest,
  canAdvance,
  goBack,
  goNext,
  isSubmitEnabled,
  type WizardStep,
} from '../install/wizard'
import { PASSWORD_STRENGTH_HINT, validateAdminStep, validateBootstrapStep } from '../install/validation'
import { DEFAULT_TENANT_NAME, createEmptyInstallForm } from '../install/types'
import type {
  InstallConsoles,
  InstallField,
  InstallFieldErrors,
  InstallForm,
  InstallInitializeResp,
  InstallStatus,
} from '../install/types'
import '../LoginPage.css'
import '../InstallPage.css'

/**
 * /install 三步初始化向导。
 *
 * 只有第 ③ 步会发写请求（POST /install/initialize）；第 ①② 步纯前端 state。
 * 进入页面先读 GET /install/status，按 initialized / schemaReady / tokenRequired
 * 决定渲染向导还是"已初始化 / 数据库未就绪 / 令牌未配置"三种阻塞视图。
 */
export default function InstallPage() {
  const [status, setStatus] = useState<InstallStatus | null>(null)
  const [statusLoading, setStatusLoading] = useState(true)
  const [statusError, setStatusError] = useState('')

  const [step, setStep] = useState<WizardStep>(FIRST_WIZARD_STEP)
  const [form, setForm] = useState<InstallForm>(createEmptyInstallForm)
  const [bootstrapToken, setBootstrapToken] = useState('')
  const [fieldErrors, setFieldErrors] = useState<InstallFieldErrors>({})
  const [formError, setFormError] = useState('')
  const [submitting, setSubmitting] = useState(false)
  const [blocked, setBlocked] = useState(false)
  const [result, setResult] = useState<InstallInitializeResp | null>(null)

  const loadStatus = useCallback(async () => {
    setStatusLoading(true)
    setStatusError('')
    try {
      const next = await getInstallStatus()
      setStatus(next)
      setBlocked(!next.tokenRequired)
    } catch (err) {
      setStatus(null)
      setStatusError(statusUnavailableMessage(err))
    } finally {
      setStatusLoading(false)
    }
  }, [])

  useEffect(() => {
    void loadStatus()
  }, [loadStatus])

  const clearFieldError = (field: InstallField) => {
    setFieldErrors((prev) => {
      if (!(field in prev)) return prev
      const next = { ...prev }
      delete next[field]
      return next
    })
  }

  const updateField = (field: keyof InstallForm, value: string) => {
    setForm((prev) => ({ ...prev, [field]: value }))
    clearFieldError(field)
  }

  const updateBootstrapToken = (value: string) => {
    setBootstrapToken(value)
    clearFieldError('bootstrapToken')
  }

  const handleNext = () => {
    const next = goNext(step, form)
    if (next === step) {
      const validation = canAdvance(step, form)
      setFieldErrors(validation.fields)
      setFormError(validation.message)
      return
    }
    setFieldErrors({})
    setFormError('')
    setStep(next)
  }

  const handleBack = () => {
    setFieldErrors({})
    setFormError('')
    setStep((current) => goBack(current))
  }

  const submit = async (e: FormEvent) => {
    e.preventDefault()
    // 第 ①② 步回车/误触提交：只前进，不发请求（"只有第 ③ 步发请求"）
    if (step !== LAST_WIZARD_STEP) {
      handleNext()
      return
    }

    const adminValidation = validateAdminStep(form)
    if (!adminValidation.ok) {
      setStep(2)
      setFieldErrors(adminValidation.fields)
      setFormError(adminValidation.message)
      return
    }
    const tokenValidation = validateBootstrapStep(bootstrapToken)
    if (!tokenValidation.ok) {
      setFieldErrors(tokenValidation.fields)
      setFormError(tokenValidation.message)
      return
    }

    setSubmitting(true)
    setFieldErrors({})
    setFormError('')
    try {
      const resp = await initializeInstall(buildInitializeRequest(form), bootstrapToken)
      // 成功后立即清掉内存中的 bootstrap token，缩短敏感值生命周期
      setBootstrapToken('')
      setResult(resp)
    } catch (err) {
      const behavior = resolveInstallError(err)
      if (behavior.alreadyInitialized) {
        setBootstrapToken('')
        setStatus((prev) =>
          prev
            ? { ...prev, initialized: true }
            : { initialized: true, tokenRequired: true, schemaReady: true, consoles: {} },
        )
        return
      }
      if (behavior.redirectToInstall) {
        window.location.href = INSTALL_PENDING_REDIRECT
        return
      }
      if (behavior.disableSubmit) {
        // 服务端此刻才知道 token 未配置：直接切到"初始化接口不可用"阻塞屏，
        // 那里有可执行文案 + "重新检查"（后端配好 env 重启后一键复核），而不是让表单一直置灰。
        setBlocked(true)
        setStatus((prev) => (prev ? { ...prev, tokenRequired: false } : prev))
        return
      }
      if (behavior.field === 'form') {
        setFieldErrors({})
        setFormError(behavior.message)
      } else {
        setFieldErrors({ [behavior.field]: behavior.message })
      }
    } finally {
      setSubmitting(false)
    }
  }

  const submitEnabled = isSubmitEnabled({ step, form, bootstrapToken, submitting, blocked })

  const renderBody = () => {
    if (statusLoading) {
      return (
        <div className="install-state">
          <h2 className="card-title">正在检查系统状态</h2>
          <p className="card-subtitle">正在读取 /install/status…</p>
        </div>
      )
    }
    if (result) {
      // 控制台入口优先取 initialize 响应（与 status 同源），其次用进入页面时读到的 status
      return <CompletionScreen result={result} consoles={result.consoles ?? status?.consoles} />
    }
    if (!status) {
      return (
        <div className="install-state">
          <h2 className="card-title">无法获取初始化状态</h2>
          <p className="install-state-msg">{statusError || INSTALL_MESSAGES.statusUnavailable}</p>
          <button type="button" className="login-btn" onClick={() => void loadStatus()}>
            重新检查
          </button>
        </div>
      )
    }
    const view = resolveInstallView(status)
    if (view !== 'form') {
      return <BlockedScreen view={view} consoles={status.consoles} onRetry={() => void loadStatus()} />
    }

    return (
      <form className="install-form" onSubmit={submit} autoComplete="off">
        <ol className="install-steps">
          {WIZARD_STEPS.map((meta) => (
            <li
              key={meta.step}
              className={[
                'install-step',
                meta.step === step ? 'is-active' : '',
                meta.step < step ? 'is-done' : '',
              ]
                .filter(Boolean)
                .join(' ')}
            >
              <span className="install-step-index">{meta.step < step ? '✓' : meta.step}</span>
              <span className="install-step-text">
                <strong>{meta.title}</strong>
                <em>{meta.description}</em>
              </span>
            </li>
          ))}
        </ol>

        {formError && <div className="error-msg">{formError}</div>}

        {step === 1 && (
          <div className="install-step-body">
            <div className="form-group">
              <label htmlFor="install-tenant-name">租户名称</label>
              <input
                id="install-tenant-name"
                type="text"
                value={form.tenantName}
                onChange={(e) => updateField('tenantName', e.target.value)}
                placeholder={DEFAULT_TENANT_NAME}
                autoComplete="off"
              />
              <p className="field-hint">
                默认写入后端内置的平台租户「{DEFAULT_TENANT_NAME}」，可直接沿用；也可按需修改，留空则由后端取默认值。
              </p>
            </div>
          </div>
        )}

        {step === 2 && (
          <div className="install-step-body">
            <div className="form-group">
              <label htmlFor="install-admin-username">管理员用户名</label>
              <input
                id="install-admin-username"
                type="text"
                value={form.adminUsername}
                onChange={(e) => updateField('adminUsername', e.target.value)}
                placeholder="请输入登录用户名"
                autoComplete="off"
              />
              {fieldErrors.adminUsername && <p className="field-error">{fieldErrors.adminUsername}</p>}
            </div>
            <div className="form-group">
              <label htmlFor="install-admin-password">管理员密码</label>
              <input
                id="install-admin-password"
                type="password"
                value={form.adminPassword}
                onChange={(e) => updateField('adminPassword', e.target.value)}
                placeholder={PASSWORD_STRENGTH_HINT}
                autoComplete="new-password"
              />
              {fieldErrors.adminPassword ? (
                <p className="field-error">{fieldErrors.adminPassword}</p>
              ) : (
                <p className="field-hint">{PASSWORD_STRENGTH_HINT}</p>
              )}
            </div>
            <div className="form-group">
              <label htmlFor="install-admin-confirm">确认密码</label>
              <input
                id="install-admin-confirm"
                type="password"
                value={form.confirmPassword}
                onChange={(e) => updateField('confirmPassword', e.target.value)}
                placeholder="请再次输入管理员密码"
                autoComplete="new-password"
              />
              {fieldErrors.confirmPassword && <p className="field-error">{fieldErrors.confirmPassword}</p>}
            </div>
            <div className="form-group">
              <label htmlFor="install-admin-name">管理员姓名</label>
              <input
                id="install-admin-name"
                type="text"
                value={form.adminName}
                onChange={(e) => updateField('adminName', e.target.value)}
                placeholder="可选，例如：系统管理员"
                autoComplete="off"
              />
            </div>
            <div className="form-group">
              <label htmlFor="install-admin-email">管理员邮箱</label>
              <input
                id="install-admin-email"
                type="email"
                value={form.adminEmail}
                onChange={(e) => updateField('adminEmail', e.target.value)}
                placeholder="与手机号至少填一个"
                autoComplete="off"
              />
              {fieldErrors.adminEmail && <p className="field-error">{fieldErrors.adminEmail}</p>}
            </div>
            <div className="form-group">
              <label htmlFor="install-admin-phone">管理员手机号</label>
              <input
                id="install-admin-phone"
                type="tel"
                value={form.adminPhone}
                onChange={(e) => updateField('adminPhone', e.target.value)}
                placeholder="与邮箱至少填一个"
                autoComplete="off"
              />
              {fieldErrors.adminPhone && <p className="field-error">{fieldErrors.adminPhone}</p>}
            </div>
          </div>
        )}

        {step === 3 && (
          <div className="install-step-body">
            <div className="install-summary">
              <div className="install-summary-title">将写入的内置数据（仅写入一次）</div>
              <ul>
                <li>
                  <span>平台租户</span>
                  <strong>{form.tenantName.trim() || DEFAULT_TENANT_NAME}</strong>
                </li>
                <li>
                  <span>管理员用户名</span>
                  <strong>{form.adminUsername.trim()}</strong>
                </li>
                <li>
                  <span>管理员姓名</span>
                  <strong>{form.adminName.trim() || '-'}</strong>
                </li>
                <li>
                  <span>邮箱</span>
                  <strong>{form.adminEmail.trim() || '-'}</strong>
                </li>
                <li>
                  <span>手机号</span>
                  <strong>{form.adminPhone.trim() || '-'}</strong>
                </li>
              </ul>
              <p className="field-hint">
                同时创建内置应用、内置菜单与内置 OAuth 客户端；初始化成功后再次触发会被拒绝。
              </p>
            </div>
            <div className="form-group">
              <label htmlFor="install-bootstrap-token">初始化令牌 BOOTSTRAP_TOKEN</label>
              <input
                id="install-bootstrap-token"
                type="password"
                value={bootstrapToken}
                onChange={(e) => updateBootstrapToken(e.target.value)}
                placeholder="粘贴服务端环境变量 BOOTSTRAP_TOKEN 的值"
                autoComplete="off"
              />
              {fieldErrors.bootstrapToken ? (
                <p className="field-error">{fieldErrors.bootstrapToken}</p>
              ) : (
                <p className="field-hint">
                  向部署人员索取该值。仅用于本次提交（请求头 X-Bootstrap-Token），不会写入浏览器存储。
                </p>
              )}
            </div>
          </div>
        )}

        <div className="install-actions">
          {step > FIRST_WIZARD_STEP && (
            <button type="button" className="ghost-btn" onClick={handleBack} disabled={submitting}>
              上一步
            </button>
          )}
          {step < LAST_WIZARD_STEP ? (
            <button type="button" className="login-btn" onClick={handleNext}>
              下一步
            </button>
          ) : (
            <button type="submit" className="login-btn" disabled={!submitEnabled}>
              {submitting ? '初始化中…' : '确认初始化'}
            </button>
          )}
        </div>
      </form>
    )
  }

  return (
    <div className="login-page">
      <div className="login-brand">
        <div className="brand-orbit orbit-1" />
        <div className="brand-orbit orbit-2" />
        <div className="brand-inner">
          <div className="brand-logo">IAM</div>
          <h1 className="brand-title">系统初始化</h1>
          <p className="brand-desc">
            全新部署只需完成一次初始化。
            <br />
            将创建平台租户、初始管理员与内置应用 / 菜单 / OAuth 客户端，这些数据仅在本次写入。
          </p>
          <div className="brand-features">
            <span>令牌校验</span>
            <span>内置数据只写一次</span>
            <span>完成后即可登录控制台</span>
          </div>
        </div>
      </div>

      <div className="login-main">
        <div className="login-card install-card">{renderBody()}</div>
      </div>
    </div>
  )
}

function BlockedScreen({
  view,
  consoles,
  onRetry,
}: {
  view: Exclude<InstallView, 'form'>
  consoles?: InstallConsoles
  onRetry: () => void
}) {
  const text = INSTALL_VIEW_TEXT[view]
  return (
    <div className="install-state">
      <h2 className="card-title">{text.title}</h2>
      <p className="install-state-msg">{text.message}</p>
      {view === 'alreadyInitialized' && <ConsoleLinks consoles={consoles} />}
      <button type="button" className="login-btn" onClick={onRetry}>
        重新检查
      </button>
    </div>
  )
}

function CompletionScreen({ result, consoles }: { result: InstallInitializeResp; consoles?: InstallConsoles }) {
  return (
    <div className="install-state">
      <div className="install-done-badge">✓</div>
      <h2 className="card-title">初始化完成</h2>
      <p className="card-subtitle">内置数据已写入，后续启动不会再重复写入。</p>
      <p className="install-summary-line">
        管理员账号：<strong>{result.adminUsername || '-'}</strong>
      </p>
      <a className="login-btn install-link-btn" href={resolveLoginURL(result.loginURL)}>
        前往登录
      </a>
      <ConsoleLinks consoles={consoles} />
    </div>
  )
}

function ConsoleLinks({ consoles }: { consoles?: InstallConsoles }) {
  const entries = consoleEntries(consoles)
  if (entries.length === 0) return null
  return (
    <div className="install-consoles">
      <div className="install-consoles-title">控制台入口</div>
      <ul>
        {entries.map((entry) => (
          <li key={entry.key}>
            <a href={entry.url} target="_blank" rel="noreferrer">
              <strong>{entry.label}</strong>
              <span>{entry.url}</span>
            </a>
          </li>
        ))}
      </ul>
    </div>
  )
}
