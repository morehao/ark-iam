import { beforeEach, describe, expect, test, vi } from 'vitest'
import { BizCode } from './response'

beforeEach(() => {
  vi.clearAllMocks()
})

describe('BizCode', () => {
  test('has correct success code', () => {
    expect(BizCode.Success).toBe(0)
  })

  test('has correct auth error codes', () => {
    expect(BizCode.Unauthorized).toBe(110000)
    expect(BizCode.Forbidden).toBe(110001)
    expect(BizCode.TokenInvalid).toBe(110002)
    expect(BizCode.TokenExpired).toBe(110003)
    expect(BizCode.PermissionDenied).toBe(110004)
  })

  test('has correct param invalid code', () => {
    expect(BizCode.ParamInvalid).toBe(100104)
  })
})
