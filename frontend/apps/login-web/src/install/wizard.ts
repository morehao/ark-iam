import type { InstallForm, InstallInitializeReq } from './types'
import { validateAdminStep, validateBootstrapStep, type StepValidation } from './validation'

/**
 * 三步向导的纯状态机。
 *
 * 关键约束：**只有第 ③ 步会发起写请求**（POST /install/initialize）。
 * 第 ①② 步只改本地 state，不存在任何按步的后端端点；因此 goNext 只做本地校验，
 * 第 ③ 步的"下一步"不存在，提交走 isSubmitEnabled + buildInitializeRequest。
 */

export type WizardStep = 1 | 2 | 3

export interface WizardStepMeta {
  step: WizardStep
  title: string
  description: string
}

export const WIZARD_STEPS: readonly WizardStepMeta[] = [
  { step: 1, title: '租户', description: '平台租户名称' },
  { step: 2, title: '管理员', description: '初始管理员账号' },
  { step: 3, title: '内置数据确认', description: '确认并提交初始化' },
]

export const FIRST_WIZARD_STEP: WizardStep = 1
export const LAST_WIZARD_STEP: WizardStep = 3

const OK: StepValidation = { ok: true, fields: {}, message: '' }

/** 该步是否满足"前进"条件（第 ③ 步不前进，恒返回 ok）。 */
export function canAdvance(step: WizardStep, form: InstallForm): StepValidation {
  switch (step) {
    case 1:
      // 租户名可留空（后端取默认值），恒可前进
      return OK
    case 2:
      return validateAdminStep(form)
    case 3:
      return OK
  }
}

/** 前进：校验不通过时停在原步（调用方用返回的 StepValidation 展示错误）。 */
export function goNext(step: WizardStep, form: InstallForm): WizardStep {
  if (step >= LAST_WIZARD_STEP) return step
  return canAdvance(step, form).ok ? ((step + 1) as WizardStep) : step
}

/** 后退：第 ① 步不再后退。 */
export function goBack(step: WizardStep): WizardStep {
  return step <= FIRST_WIZARD_STEP ? step : ((step - 1) as WizardStep)
}

export interface SubmitGateInput {
  step: WizardStep
  form: InstallForm
  bootstrapToken: string
  submitting: boolean
  /** 服务端声明 token 不可用（status.tokenRequired=false 或 107002）时禁止提交 */
  blocked: boolean
}

/** 提交按钮可用性：必须停在第 ③ 步、未在提交中、token 可用、且全表单校验通过。 */
export function isSubmitEnabled(input: SubmitGateInput): boolean {
  const { step, form, bootstrapToken, submitting, blocked } = input
  if (submitting || blocked) return false
  if (step !== LAST_WIZARD_STEP) return false
  return validateAdminStep(form).ok && validateBootstrapStep(bootstrapToken).ok
}

/**
 * 组装 POST /install/initialize 请求体：可选字段留空即不提交（后端取默认），
 * 必填字段去首尾空白。
 *
 * bootstrap token 刻意**不出现在 body**：它只经 X-Bootstrap-Token 头传递，
 * 且只存在于组件内存（不落 localStorage/sessionStorage/cookie）。
 */
export function buildInitializeRequest(form: InstallForm): InstallInitializeReq {
  const req: InstallInitializeReq = {
    adminUsername: form.adminUsername.trim(),
    adminPassword: form.adminPassword,
  }
  const tenantName = form.tenantName.trim()
  if (tenantName) req.tenantName = tenantName
  const adminName = form.adminName.trim()
  if (adminName) req.adminName = adminName
  const adminEmail = form.adminEmail.trim()
  if (adminEmail) req.adminEmail = adminEmail
  const adminPhone = form.adminPhone.trim()
  if (adminPhone) req.adminPhone = adminPhone
  return req
}
