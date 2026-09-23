import type { InstallField, InstallFieldErrors, InstallForm } from './types'

/**
 * 向导客户端校验。
 *
 * 规则与后端逐条对齐（不引入后端没有的限制，也不漏掉后端会拒的明显输入）：
 *   - 用户名：必填、≤128 字符（后端 max 128）
 *   - 密码：8~128 字符，且同时含大写、小写、数字（backend/pkg/credential.ValidateStrength）
 *   - 确认密码：必须一致
 *   - 邮箱 / 手机号：至少填一个（后端 107005 的典型触发条件）
 * 客户端校验只为即时反馈，后端 400（107005/107006）仍按字段回填。
 */

export const ADMIN_USERNAME_MAX_LENGTH = 128
export const PASSWORD_MIN_LENGTH = 8
export const PASSWORD_MAX_LENGTH = 128
export const PASSWORD_STRENGTH_HINT = '至少 8 位，且同时包含大写字母、小写字母与数字'

const HAS_UPPER = /\p{Lu}/u
const HAS_LOWER = /\p{Ll}/u
const HAS_DIGIT = /\p{Nd}/u

/** 按 Unicode 码点计长度，与后端 utf8.RuneCountInString 对齐。 */
export function runeLength(value: string): number {
  return [...value].length
}

/** 口令强度：长度 8~128 且含大写、小写、数字。 */
export function isPasswordStrong(password: string): boolean {
  const length = runeLength(password)
  if (length < PASSWORD_MIN_LENGTH || length > PASSWORD_MAX_LENGTH) return false
  return HAS_UPPER.test(password) && HAS_LOWER.test(password) && HAS_DIGIT.test(password)
}

/** 联系方式：邮箱或手机号至少一个非空。 */
export function hasContact(email: string, phone: string): boolean {
  return email.trim() !== '' || phone.trim() !== ''
}

export interface StepValidation {
  ok: boolean
  /** 字段级错误（用于输入框下方红字） */
  fields: InstallFieldErrors
  /** 表单级提示（第一条错误；字段级错误也会在横幅给出，保证"点下一步必有反馈"） */
  message: string
}

function buildValidation(entries: Array<[InstallField, string]>): StepValidation {
  const fields: InstallFieldErrors = {}
  const messages: string[] = []
  for (const [field, message] of entries) {
    if (!(field in fields)) fields[field] = message
    if (!messages.includes(message)) messages.push(message)
  }
  return { ok: entries.length === 0, fields, message: messages[0] ?? '' }
}

/** 第 ① 步：租户名可留空（后端取默认），无需校验。 */
export function validateTenantStep(_form: InstallForm): StepValidation {
  return { ok: true, fields: {}, message: '' }
}

/** 第 ② 步：管理员账号。 */
export function validateAdminStep(form: InstallForm): StepValidation {
  const entries: Array<[InstallField, string]> = []
  const username = form.adminUsername.trim()
  if (!username) {
    entries.push(['adminUsername', '请输入管理员用户名'])
  } else if (runeLength(username) > ADMIN_USERNAME_MAX_LENGTH) {
    entries.push(['adminUsername', `管理员用户名不能超过 ${ADMIN_USERNAME_MAX_LENGTH} 个字符`])
  }

  if (!form.adminPassword) {
    entries.push(['adminPassword', '请输入管理员密码'])
  } else if (!isPasswordStrong(form.adminPassword)) {
    entries.push(['adminPassword', `密码强度不足：${PASSWORD_STRENGTH_HINT}`])
  }

  if (!form.confirmPassword) {
    entries.push(['confirmPassword', '请再次输入管理员密码'])
  } else if (form.confirmPassword !== form.adminPassword) {
    entries.push(['confirmPassword', '两次输入的密码不一致'])
  }

  if (!hasContact(form.adminEmail, form.adminPhone)) {
    // 两个字段同时标红：这条约束是"二选一"，只挂在邮箱下面会误导成"邮箱必填"。
    entries.push(['adminEmail', '邮箱与手机号至少填写一个'])
    entries.push(['adminPhone', '邮箱与手机号至少填写一个'])
  }

  return buildValidation(entries)
}

/** 第 ③ 步：bootstrap token 必填（值来自服务端环境变量 BOOTSTRAP_TOKEN）。 */
export function validateBootstrapStep(bootstrapToken: string): StepValidation {
  if (bootstrapToken.trim() === '') {
    return buildValidation([['bootstrapToken', '请输入初始化令牌（BOOTSTRAP_TOKEN）']])
  }
  return { ok: true, fields: {}, message: '' }
}
