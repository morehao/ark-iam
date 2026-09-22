import { describe, expect, it } from 'vitest'
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
} from './wizard'
import { createEmptyInstallForm, type InstallForm } from './types'

function validForm(overrides: Partial<InstallForm> = {}): InstallForm {
  return {
    ...createEmptyInstallForm(),
    adminUsername: 'admin',
    adminPassword: 'Admin123',
    confirmPassword: 'Admin123',
    adminEmail: 'admin@example.com',
    ...overrides,
  }
}

describe('向导步骤状态机', () => {
  it('三步定义齐全且顺序为 租户 -> 管理员 -> 内置数据确认', () => {
    expect(WIZARD_STEPS.map((s) => s.step)).toEqual([1, 2, 3])
    expect(WIZARD_STEPS.map((s) => s.title)).toEqual(['租户', '管理员', '内置数据确认'])
    expect(FIRST_WIZARD_STEP).toBe(1)
    expect(LAST_WIZARD_STEP).toBe(3)
  })

  it('第 ① 步不校验即可前进（租户名可留空）', () => {
    const form = validForm({ tenantName: '' })
    expect(canAdvance(1, form).ok).toBe(true)
    expect(goNext(1, form)).toBe(2)
  })

  it('第 ② 步校验失败时停在原步', () => {
    const form = validForm({ adminPassword: 'weak', confirmPassword: 'weak' })
    expect(goNext(2, form)).toBe(2)
    expect(canAdvance(2, form).ok).toBe(false)
  })

  it('第 ② 步校验通过后进入第 ③ 步', () => {
    expect(goNext(2, validForm())).toBe(3)
  })

  it('第 ③ 步不前进（提交动作由提交按钮承担）', () => {
    expect(goNext(3, validForm())).toBe(3)
  })

  it('后退不越过第 ① 步', () => {
    const steps: WizardStep[] = [1, 2, 3]
    expect(steps.map((s) => goBack(s))).toEqual([1, 1, 2])
  })
})

describe('isSubmitEnabled：只有第 ③ 步 + token 可用 + 全表单合法才可提交', () => {
  const base = { form: validForm(), bootstrapToken: 't-123', submitting: false, blocked: false }

  it('第 ③ 步且一切就绪时可用', () => {
    expect(isSubmitEnabled({ ...base, step: 3 })).toBe(true)
  })

  it('第 ①② 步不可提交（这两步不发任何请求）', () => {
    expect(isSubmitEnabled({ ...base, step: 1 })).toBe(false)
    expect(isSubmitEnabled({ ...base, step: 2 })).toBe(false)
  })

  it('提交中不可重复提交', () => {
    expect(isSubmitEnabled({ ...base, step: 3, submitting: true })).toBe(false)
  })

  it('服务端声明 token 不可用（未配置 BOOTSTRAP_TOKEN）时置灰', () => {
    expect(isSubmitEnabled({ ...base, step: 3, blocked: true })).toBe(false)
  })

  it('令牌为空不可提交', () => {
    expect(isSubmitEnabled({ ...base, step: 3, bootstrapToken: '  ' })).toBe(false)
  })

  it('管理员表单非法时不提交（即使已到第 ③ 步）', () => {
    expect(isSubmitEnabled({ ...base, step: 3, form: validForm({ adminUsername: '' }) })).toBe(false)
    expect(isSubmitEnabled({ ...base, step: 3, form: validForm({ adminEmail: '', adminPhone: '' }) })).toBe(false)
  })
})

describe('buildInitializeRequest', () => {
  it('只输出契约字段，可选字段留空即省略', () => {
    const req = buildInitializeRequest(validForm({ tenantName: '', adminName: '', adminEmail: '', adminPhone: '' }))
    expect(req).toEqual({ adminUsername: 'admin', adminPassword: 'Admin123' })
  })

  it('去除首尾空白并保留已填的可选字段', () => {
    const req = buildInitializeRequest(
      validForm({
        tenantName: '  平台运营中心  ',
        adminUsername: '  admin  ',
        adminName: ' 系统管理员 ',
        adminEmail: ' admin@example.com ',
        adminPhone: ' 13800000000 ',
      }),
    )
    expect(req).toEqual({
      tenantName: '平台运营中心',
      adminUsername: 'admin',
      adminPassword: 'Admin123',
      adminName: '系统管理员',
      adminEmail: 'admin@example.com',
      adminPhone: '13800000000',
    })
  })

  it('口令不做 trim（空白是口令的一部分，改了会与确认口令不一致）', () => {
    const req = buildInitializeRequest(validForm({ adminPassword: ' Admin123 ', confirmPassword: ' Admin123 ' }))
    expect(req.adminPassword).toBe(' Admin123 ')
  })

  it('bootstrap token 绝不进入请求体（只走 X-Bootstrap-Token 头）', () => {
    const req = buildInitializeRequest(validForm()) as unknown as Record<string, unknown>
    expect(Object.keys(req)).not.toContain('bootstrapToken')
    expect(Object.keys(req)).not.toContain('token')
    expect(JSON.stringify(req)).not.toContain('t-123')
  })
})
