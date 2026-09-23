import { describe, expect, it } from 'vitest'
import {
  ADMIN_USERNAME_MAX_LENGTH,
  PASSWORD_MAX_LENGTH,
  hasContact,
  isPasswordStrong,
  runeLength,
  validateAdminStep,
  validateBootstrapStep,
  validateTenantStep,
} from './validation'
import { createEmptyInstallForm, type InstallForm } from './types'

function form(overrides: Partial<InstallForm> = {}): InstallForm {
  return { ...createEmptyInstallForm(), ...overrides }
}

/** 各用例的最小合法表单：用户名 + 强口令 + 确认 + 邮箱。 */
function validForm(overrides: Partial<InstallForm> = {}): InstallForm {
  return form({
    adminUsername: 'admin',
    adminPassword: 'Admin123',
    confirmPassword: 'Admin123',
    adminEmail: 'admin@example.com',
    ...overrides,
  })
}

describe('isPasswordStrong（与后端口令强度对齐：8~128 位且含大小写与数字）', () => {
  it('接受大小写 + 数字的 8 位口令', () => {
    expect(isPasswordStrong('Admin123')).toBe(true)
  })

  it('拒绝缺少大写、小写或数字的口令', () => {
    expect(isPasswordStrong('admin123')).toBe(false)
    expect(isPasswordStrong('ADMIN123')).toBe(false)
    expect(isPasswordStrong('AdminAbc')).toBe(false)
  })

  it('按字符数而非字节数判定长度：7 位拒绝、8 位接受', () => {
    expect(isPasswordStrong('Admin12')).toBe(false)
    expect(isPasswordStrong('Admin123')).toBe(true)
  })

  it('拒绝超长口令（>128 字符）', () => {
    const long = `A1a${'x'.repeat(PASSWORD_MAX_LENGTH)}`
    expect(runeLength(long)).toBeGreaterThan(PASSWORD_MAX_LENGTH)
    expect(isPasswordStrong(long)).toBe(false)
  })

  it('中文等非 ASCII 字符不计入大写/小写/数字要求', () => {
    expect(isPasswordStrong('管理员Admin123')).toBe(true)
    expect(isPasswordStrong('管理员密码测试一二')).toBe(false)
  })
})

describe('hasContact（邮箱或手机号至少一个）', () => {
  it('任一非空即通过，仅空白不算填写', () => {
    expect(hasContact('a@b.com', '')).toBe(true)
    expect(hasContact('', '13800000000')).toBe(true)
    expect(hasContact('   ', '\t')).toBe(false)
  })
})

describe('validateAdminStep', () => {
  it('合法表单无错误', () => {
    const result = validateAdminStep(validForm())
    expect(result.ok).toBe(true)
    expect(result.message).toBe('')
    expect(result.fields).toEqual({})
  })

  it('用户名为空时报错', () => {
    const result = validateAdminStep(validForm({ adminUsername: '  ' }))
    expect(result.ok).toBe(false)
    expect(result.fields.adminUsername).toBe('请输入管理员用户名')
  })

  it('用户名超过 128 字符时报错', () => {
    const result = validateAdminStep(validForm({ adminUsername: 'u'.repeat(ADMIN_USERNAME_MAX_LENGTH + 1) }))
    expect(result.ok).toBe(false)
    expect(result.fields.adminUsername).toContain(String(ADMIN_USERNAME_MAX_LENGTH))
  })

  it('口令强度不足时报在密码字段', () => {
    const result = validateAdminStep(validForm({ adminPassword: 'weakpass', confirmPassword: 'weakpass' }))
    expect(result.ok).toBe(false)
    expect(result.fields.adminPassword).toContain('密码强度不足')
  })

  it('两次口令不一致时报在确认密码字段', () => {
    const result = validateAdminStep(validForm({ confirmPassword: 'Admin124' }))
    expect(result.ok).toBe(false)
    expect(result.fields.confirmPassword).toBe('两次输入的密码不一致')
  })

  it('邮箱与手机号都为空时字段级与表单级同时给出提示', () => {
    const result = validateAdminStep(validForm({ adminEmail: '', adminPhone: '' }))
    expect(result.ok).toBe(false)
    expect(result.fields.adminEmail).toBe('邮箱与手机号至少填写一个')
    expect(result.fields.adminPhone).toBe('邮箱与手机号至少填写一个')
    expect(result.message).toBe('邮箱与手机号至少填写一个')
  })

  it('只填手机号即通过（不要求邮箱必填）', () => {
    const result = validateAdminStep(validForm({ adminEmail: '', adminPhone: '13800000000' }))
    expect(result.ok).toBe(true)
  })

  it('多条错误时 message 取第一条，且按字段顺序返回', () => {
    const result = validateAdminStep(form())
    expect(result.ok).toBe(false)
    expect(result.message).toBe('请输入管理员用户名')
    expect(Object.keys(result.fields)).toEqual(['adminUsername', 'adminPassword', 'confirmPassword', 'adminEmail', 'adminPhone'])
  })
})

describe('validateTenantStep / validateBootstrapStep', () => {
  it('租户名可留空（后端取默认）', () => {
    expect(validateTenantStep(form({ tenantName: '' })).ok).toBe(true)
  })

  it('令牌空白视为未填写', () => {
    expect(validateBootstrapStep('   ').ok).toBe(false)
    expect(validateBootstrapStep('   ').fields.bootstrapToken).toBe('请输入初始化令牌（BOOTSTRAP_TOKEN）')
    expect(validateBootstrapStep('t-123').ok).toBe(true)
  })
})
