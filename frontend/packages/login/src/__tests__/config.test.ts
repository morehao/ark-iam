import { describe, it, expect } from 'vitest'
import { detectBasePath, appPath } from '../config'

// 说明：detectBasePath/appPath 是纯函数，覆盖"根挂载"与"非空前缀内嵌"两类，锁住终局方案的
// 前缀自适应行为。appPath 依赖模块级 appBase（由 loadRuntimeConfig 设置），此处通过直接
// 断言 detectBasePath 保证解析正确；appPath 在 appBase 为空（未初始化）时退化为无前缀。

describe('detectBasePath', () => {
  it('根挂载 /login => 空前缀', () => {
    expect(detectBasePath('/login')).toBe('')
  })
  it('根挂载 /install => 空前缀', () => {
    expect(detectBasePath('/install')).toBe('')
  })
  it('内嵌 /iam/login => /iam', () => {
    expect(detectBasePath('/iam/login')).toBe('/iam')
  })
  it('内嵌 /iam/install => /iam', () => {
    expect(detectBasePath('/iam/install')).toBe('/iam')
  })
  it('多段前缀 /a/b/login => /a/b', () => {
    expect(detectBasePath('/a/b/login')).toBe('/a/b')
  })
  it('带尾随子路径 /iam/login/xxx => /iam', () => {
    expect(detectBasePath('/iam/login/xxx')).toBe('/iam')
  })
})

describe('appPath（未初始化时 appBase 为空，退化为无前缀）', () => {
  it('/install => /install', () => {
    expect(appPath('/install')).toBe('/install')
  })
  it('/login => /login', () => {
    expect(appPath('/login')).toBe('/login')
  })
  it('无前导斜杠自动补', () => {
    expect(appPath('install')).toBe('/install')
  })
})
