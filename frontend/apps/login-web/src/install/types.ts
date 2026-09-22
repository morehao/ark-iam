/**
 * /install/* 接口契约（由 auth 应用提供，前缀 /install）。
 *
 * login-web 不依赖 @ark-iam/types（刻意保持零 @ark-iam/* 依赖），
 * 故与 src/types.ts 的既有做法一致：按后端同名同形在本包内定义一份。
 * 后端事实源：apps/auth/internal/controller/ctrinstall + pkg/code/install.go。
 */

/** 内置控制台的部署地址（后端 oidc.consoles 回显，用于"下一步去哪登录"）。 */
export interface InstallConsoles {
  platformAdminWeb?: string
  tenantAdminWeb?: string
}

/** GET /install/status 响应（公开只读，无鉴权）。 */
export interface InstallStatus {
  /** false = 需要初始化；true = 已初始化（终态，不可重复初始化） */
  initialized: boolean
  /** true = 服务端已配置 BOOTSTRAP_TOKEN；false = /install/initialize 整体不可用（fail-closed） */
  tokenRequired: boolean
  /** false = 数据库缺表（后端可能以 db.auto_migrate: false 启动） */
  schemaReady: boolean
  /** 内置控制台地址，供页面展示"稍后从哪登录" */
  consoles: InstallConsoles
}

/** POST /install/initialize 请求体（可选字段留空即不提交，由后端取默认值）。 */
export interface InstallInitializeReq {
  /** 可选；留空 -> 后端默认租户名 */
  tenantName?: string
  /** 必填，最长 128 字符 */
  adminUsername: string
  /** 必填，须包含大写字母、小写字母与数字 */
  adminPassword: string
  adminName?: string
  /** 可选，但与 adminPhone 至少填一个 */
  adminEmail?: string
  /** 可选，但与 adminEmail 至少填一个 */
  adminPhone?: string
  issuer?: string
}

/** 初始化写入报告中的单条变更。 */
export interface InstallChangeItem {
  entity: string
  key: string
  action: string
}

export interface InstallReport {
  tenantId: string
  changes: InstallChangeItem[]
}

/** POST /install/initialize 成功响应。 */
export interface InstallInitializeResp {
  report: InstallReport
  adminUsername: string
  /** 管理端登录入口（页面回显；缺失时回退到本页登录路由） */
  loginURL: string
  /** 与 GET /install/status 同源的控制台地址（dtoinstall.InstallationInitializeResp 也回传） */
  consoles?: InstallConsoles
}

/** 向导表单：纯前端状态，第 ①② 步不产生任何网络请求。 */
export interface InstallForm {
  tenantName: string
  adminUsername: string
  adminPassword: string
  confirmPassword: string
  adminName: string
  adminEmail: string
  adminPhone: string
}

/** 可承载字段级错误的字段（含不在表单里的 bootstrap token）。 */
export type InstallField = keyof InstallForm | 'bootstrapToken'

export type InstallFieldErrors = Partial<Record<InstallField, string>>

/** 后端内置的默认租户名（pkg/seed），第 ① 步预填并允许直接沿用。 */
export const DEFAULT_TENANT_NAME = '平台运营中心'

/**
 * 初始表单：仅租户名按后端默认值预填（可留空/改写），
 * 管理员相关字段一律留空，避免运维在无意识下沿用示例值。
 */
export function createEmptyInstallForm(): InstallForm {
  return {
    tenantName: DEFAULT_TENANT_NAME,
    adminUsername: '',
    adminPassword: '',
    confirmPassword: '',
    adminName: '',
    adminEmail: '',
    adminPhone: '',
  }
}
