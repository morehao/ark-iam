import { INSTALL_MESSAGES } from './messages'
import type { InstallConsoles, InstallStatus } from './types'

/**
 * GET /install/status 的结果 -> 页面该渲染哪个视图。
 *
 * 判定顺序即优先级：已初始化是终态（即使 schemaReady/tokenRequired 异常也无意义），
 * 其次是"环境不具备初始化条件"（缺表、缺 token），最后才是正常向导。
 */
export type InstallView = 'form' | 'alreadyInitialized' | 'schemaNotReady' | 'tokenNotConfigured'

export function resolveInstallView(status: InstallStatus): InstallView {
  if (status.initialized) return 'alreadyInitialized'
  if (!status.schemaReady) return 'schemaNotReady'
  if (!status.tokenRequired) return 'tokenNotConfigured'
  return 'form'
}

/** 未初始化时，登录入口应把用户送到 /install。 */
export function shouldRedirectToInstall(status: InstallStatus): boolean {
  return !status.initialized
}

/** 环境阻塞视图的标题与说明（供页面与测试共用同一份文案）。 */
export const INSTALL_VIEW_TEXT: Record<Exclude<InstallView, 'form'>, { title: string; message: string }> = {
  alreadyInitialized: { title: '系统已初始化', message: INSTALL_MESSAGES.alreadyInitialized },
  schemaNotReady: { title: '数据库尚未就绪', message: INSTALL_MESSAGES.schemaNotReady },
  tokenNotConfigured: { title: '初始化接口不可用', message: INSTALL_MESSAGES.tokenNotConfigured },
}

export interface InstallConsoleEntry {
  key: keyof InstallConsoles
  label: string
  url: string
}

/** 控制台入口（地址由后端 status.consoles 回显；缺失则不下发入口，不臆造地址）。 */
export function consoleEntries(consoles?: InstallConsoles): InstallConsoleEntry[] {
  const entries: InstallConsoleEntry[] = []
  if (consoles?.platformAdminWeb) {
    entries.push({ key: 'platformAdminWeb', label: '平台管理后台', url: consoles.platformAdminWeb })
  }
  if (consoles?.tenantAdminWeb) {
    entries.push({ key: 'tenantAdminWeb', label: '租户管理后台', url: consoles.tenantAdminWeb })
  }
  return entries
}

/** 成功后的登录入口：优先后端回显的 loginURL，缺失时回退到本页登录路由。 */
export const FALLBACK_LOGIN_URL = '/login'

export function resolveLoginURL(loginURL?: string): string {
  return (loginURL ?? '').trim() || FALLBACK_LOGIN_URL
}
