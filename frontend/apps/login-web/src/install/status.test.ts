import { describe, expect, it } from 'vitest'
import {
  FALLBACK_LOGIN_URL,
  INSTALL_VIEW_TEXT,
  consoleEntries,
  resolveInstallView,
  resolveLoginURL,
  shouldRedirectToInstall,
} from './status'
import type { InstallStatus } from './types'

function status(overrides: Partial<InstallStatus> = {}): InstallStatus {
  return {
    initialized: false,
    tokenRequired: true,
    schemaReady: true,
    consoles: { platformAdminWeb: 'http://localhost:4001', tenantAdminWeb: 'http://localhost:4002' },
    ...overrides,
  }
}

describe('resolveInstallView：status -> 视图', () => {
  it('未初始化 + 表就绪 + token 可用 -> 正常向导', () => {
    expect(resolveInstallView(status())).toBe('form')
  })

  it('已初始化 -> 已初始化屏（即使其它字段异常）', () => {
    expect(resolveInstallView(status({ initialized: true, schemaReady: false, tokenRequired: false }))).toBe(
      'alreadyInitialized',
    )
  })

  it('未初始化但缺表 -> 数据库未就绪（优先于 token 判定）', () => {
    expect(resolveInstallView(status({ schemaReady: false, tokenRequired: false }))).toBe('schemaNotReady')
  })

  it('未初始化且表就绪但未配置 BOOTSTRAP_TOKEN -> 令牌不可用', () => {
    expect(resolveInstallView(status({ tokenRequired: false }))).toBe('tokenNotConfigured')
  })

  it('三种阻塞视图都带可执行文案（不是只描述现象）', () => {
    expect(INSTALL_VIEW_TEXT.schemaNotReady.message).toContain('db.auto_migrate')
    expect(INSTALL_VIEW_TEXT.tokenNotConfigured.message).toContain('BOOTSTRAP_TOKEN')
    expect(INSTALL_VIEW_TEXT.alreadyInitialized.message).toContain('控制台入口')
  })
})

describe('shouldRedirectToInstall', () => {
  it('仅未初始化时跳转 /install', () => {
    expect(shouldRedirectToInstall(status({ initialized: false }))).toBe(true)
    expect(shouldRedirectToInstall(status({ initialized: true }))).toBe(false)
  })
})

describe('consoleEntries：控制台入口', () => {
  it('按 platformAdminWeb / tenantAdminWeb 生成中文入口名', () => {
    const entries = consoleEntries({ platformAdminWeb: 'http://a', tenantAdminWeb: 'http://b' })
    expect(entries).toEqual([
      { key: 'platformAdminWeb', label: '平台管理后台', url: 'http://a' },
      { key: 'tenantAdminWeb', label: '租户管理后台', url: 'http://b' },
    ])
  })

  it('缺失地址时不下发入口，不臆造 URL', () => {
    expect(consoleEntries({})).toEqual([])
    expect(consoleEntries(undefined)).toEqual([])
    expect(consoleEntries({ tenantAdminWeb: 'http://b' })).toHaveLength(1)
  })
})

describe('resolveLoginURL', () => {
  it('优先用后端回显的 loginURL', () => {
    expect(resolveLoginURL('http://localhost:4001/login')).toBe('http://localhost:4001/login')
  })

  it('缺失或空白时回退本页登录路由', () => {
    expect(resolveLoginURL(undefined)).toBe(FALLBACK_LOGIN_URL)
    expect(resolveLoginURL('  ')).toBe(FALLBACK_LOGIN_URL)
    expect(FALLBACK_LOGIN_URL).toBe('/login')
  })
})
