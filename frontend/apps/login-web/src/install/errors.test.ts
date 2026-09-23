import { describe, expect, it } from 'vitest'
import {
  INSTALL_ERROR_CODE,
  codeFromHTTPStatus,
  decodeErrorEnvelope,
  describeInstallError,
  resolveInstallError,
  statusUnavailableMessage,
  unwrapEnvelope,
} from './errors'
import { INSTALL_MESSAGES } from './messages'

describe('describeInstallError：错误码 -> 文案与处置', () => {
  it('107000 已初始化：终态屏 + 控制台入口，不再重试', () => {
    const behavior = describeInstallError(INSTALL_ERROR_CODE.alreadyInitialized)
    expect(behavior.alreadyInitialized).toBe(true)
    expect(behavior.redirectToInstall).toBe(false)
    expect(behavior.disableSubmit).toBe(false)
    expect(behavior.field).toBe('form')
    expect(behavior.message).toBe(INSTALL_MESSAGES.alreadyInitialized)
  })

  it('107001 令牌错误：保留表单并定位到令牌字段', () => {
    const behavior = describeInstallError(INSTALL_ERROR_CODE.tokenInvalid)
    expect(behavior.field).toBe('bootstrapToken')
    expect(behavior.message).toBe(INSTALL_MESSAGES.tokenInvalid)
    expect(behavior.alreadyInitialized).toBe(false)
    expect(behavior.redirectToInstall).toBe(false)
  })

  it('107002 未配置 BOOTSTRAP_TOKEN：禁用提交并给出可执行文案', () => {
    const behavior = describeInstallError(INSTALL_ERROR_CODE.tokenNotConfigured)
    expect(behavior.disableSubmit).toBe(true)
    expect(behavior.message).toBe(INSTALL_MESSAGES.tokenNotConfigured)
    expect(behavior.message).toContain('BOOTSTRAP_TOKEN')
    expect(behavior.message).toContain('重启后端')
  })

  it('107003 系统错误：保留表单并提示查后端日志', () => {
    const behavior = describeInstallError(INSTALL_ERROR_CODE.initializeFailed)
    expect(behavior.field).toBe('form')
    expect(behavior.message).toContain('后端日志')
    expect(behavior.disableSubmit).toBe(false)
  })

  it('107004 守卫未初始化：跳 /install，不停留在表单', () => {
    const behavior = describeInstallError(INSTALL_ERROR_CODE.pending)
    expect(behavior.redirectToInstall).toBe(true)
    expect(behavior.alreadyInitialized).toBe(false)
  })

  it('107005 入参非法：表单级错误', () => {
    const behavior = describeInstallError(INSTALL_ERROR_CODE.badRequest)
    expect(behavior.field).toBe('form')
    expect(behavior.message).toBe(INSTALL_MESSAGES.badRequest)
  })

  it('107006 口令过弱：定位到密码字段', () => {
    const behavior = describeInstallError(INSTALL_ERROR_CODE.passwordWeak)
    expect(behavior.field).toBe('adminPassword')
    expect(behavior.message).toContain('大写字母')
  })

  it('107001/107005/107006 优先回显后端更具体的 msg', () => {
    expect(describeInstallError(INSTALL_ERROR_CODE.tokenInvalid, '令牌已过期').message).toBe('令牌已过期')
    expect(describeInstallError(INSTALL_ERROR_CODE.badRequest, '用户名不合法').message).toBe('用户名不合法')
    expect(describeInstallError(INSTALL_ERROR_CODE.passwordWeak, '口令太短').message).toBe('口令太短')
  })

  it('环境/终态类错误忽略后端 msg（否则会丢掉"下一步做什么"）', () => {
    expect(describeInstallError(INSTALL_ERROR_CODE.tokenNotConfigured, 'boom').message).toBe(
      INSTALL_MESSAGES.tokenNotConfigured,
    )
    expect(describeInstallError(INSTALL_ERROR_CODE.alreadyInitialized, '已初始化').message).toBe(
      INSTALL_MESSAGES.alreadyInitialized,
    )
  })

  it('未知错误码按系统错误兜底且标记为未知', () => {
    const behavior = describeInstallError(999999)
    expect(behavior.known).toBe(false)
    expect(behavior.field).toBe('form')
    expect(behavior.message).toBe(INSTALL_MESSAGES.unknown)
  })
})

describe('codeFromHTTPStatus：无响应体时按 HTTP 状态兜底', () => {
  it('映射 401/503/500/400', () => {
    expect(codeFromHTTPStatus(401)).toBe(INSTALL_ERROR_CODE.tokenInvalid)
    expect(codeFromHTTPStatus(503)).toBe(INSTALL_ERROR_CODE.tokenNotConfigured)
    expect(codeFromHTTPStatus(500)).toBe(INSTALL_ERROR_CODE.initializeFailed)
    expect(codeFromHTTPStatus(400)).toBe(INSTALL_ERROR_CODE.badRequest)
  })

  it('409 两义时保守取"已初始化"（不把可能已初始化的系统推回表单）', () => {
    expect(codeFromHTTPStatus(409)).toBe(INSTALL_ERROR_CODE.alreadyInitialized)
  })

  it('未登记状态码返回 undefined', () => {
    expect(codeFromHTTPStatus(418)).toBeUndefined()
    expect(codeFromHTTPStatus(undefined)).toBeUndefined()
  })
})

describe('decodeErrorEnvelope：兼容顶层与嵌套信封', () => {
  it('顶层 {code,msg}', () => {
    expect(decodeErrorEnvelope({ code: 107001, msg: '令牌不对' })).toEqual({ code: 107001, message: '令牌不对' })
  })

  it('嵌套在 data / error 下', () => {
    expect(decodeErrorEnvelope({ data: { code: 107000, msg: '已初始化' } })).toEqual({
      code: 107000,
      message: '已初始化',
    })
    expect(decodeErrorEnvelope({ error: { code: 107002, message: '未配置' } })).toEqual({
      code: 107002,
      message: '未配置',
    })
  })

  it('code 为数字字符串时也识别', () => {
    expect(decodeErrorEnvelope({ code: '107005', msg: '参数错' })).toEqual({ code: 107005, message: '参数错' })
  })

  it('非对象/无 code 返回 undefined', () => {
    expect(decodeErrorEnvelope('boom')).toBeUndefined()
    expect(decodeErrorEnvelope({ msg: 'no code' })).toBeUndefined()
    expect(decodeErrorEnvelope(null)).toBeUndefined()
  })
})

describe('resolveInstallError：把 axios 异常归一为页面处置', () => {
  it('非 2xx + 响应体业务码（正常路径）', () => {
    const axiosLike = { isAxiosError: true, response: { status: 409, data: { code: 107000, msg: '系统已完成初始化' } } }
    const behavior = resolveInstallError(axiosLike)
    expect(behavior.code).toBe(107000)
    expect(behavior.alreadyInitialized).toBe(true)
  })

  it('非 2xx 且响应体为空时按状态码兜底', () => {
    expect(resolveInstallError({ response: { status: 503, data: '' } }).code).toBe(107002)
    expect(resolveInstallError({ response: { status: 401, data: null } }).code).toBe(107001)
  })

  it('恒 200 信封（后端用 Fail helper）也能识别错误码', () => {
    expect(resolveInstallError({ response: { status: 200, data: { code: 107004, msg: '尚未初始化' } } }).redirectToInstall).toBe(
      true,
    )
  })

  it('携带 code 的 ApiError 同样可归一', () => {
    const apiError = Object.assign(new Error('初始化执行失败'), { code: 107003 })
    const behavior = resolveInstallError(apiError)
    expect(behavior.code).toBe(107003)
    expect(behavior.message).toContain('后端日志')
  })

  it('纯网络错误（无 response）落入未知兜底', () => {
    const behavior = resolveInstallError(new Error('Network Error'))
    expect(behavior.code).toBe(0)
    expect(behavior.known).toBe(false)
    expect(behavior.message).toBe(INSTALL_MESSAGES.unknown)
  })
})

describe('unwrapEnvelope：成功响应两种形态都兼容', () => {  it('{code:0,data} 信封取 data', () => {
    expect(unwrapEnvelope<{ initialized: boolean }>({ code: 0, msg: 'success', data: { initialized: false } })).toEqual({
      ok: true,
      data: { initialized: false },
    })
  })

  it('裸对象原样返回（契约示例即裸对象）', () => {
    expect(unwrapEnvelope<{ initialized: boolean }>({ initialized: true })).toEqual({
      ok: true,
      data: { initialized: true },
    })
  })

  it('code!==0 返回失败与业务码', () => {
    expect(unwrapEnvelope({ code: 107002, msg: '未配置' })).toEqual({ ok: false, code: 107002, message: '未配置' })
  })
})

describe('statusUnavailableMessage：状态查询失败的回显', () => {
  it('取后端信封文案（503 系统错误，code=-1）', () => {
    const err = { response: { status: 503, data: { code: -1, msg: '数据库不可用' } } }
    expect(statusUnavailableMessage(err)).toBe('数据库不可用')
  })

  it('状态查询的 503 不冒用 107002「未配置 BOOTSTRAP_TOKEN」文案', () => {
    const err = { response: { status: 503, data: { code: -1, msg: '' } } }
    expect(statusUnavailableMessage(err)).toBe(INSTALL_MESSAGES.statusUnavailable)
    expect(statusUnavailableMessage(err)).not.toContain('BOOTSTRAP_TOKEN')
  })

  it('网络错误不让 axios 英文原文冒到界面上', () => {
    expect(statusUnavailableMessage(new Error('Network Error'))).toBe(INSTALL_MESSAGES.statusUnavailable)
    expect(statusUnavailableMessage(new Error('Request failed with status code 502'))).toBe(
      INSTALL_MESSAGES.statusUnavailable,
    )
  })
})
