// @ark-iam/login 包入口：OP 账密登录页 + 运行时配置 / 认证 API + 类型。
//
// install 初始化协议已拆到独立包 `@ark-iam/install`（账密页不再承载 /install 协议）；
// 消费方需要初始化向导能力（InstallStatus / getInstallStatus / initializeInstall / 错误映射）时
// 从 `@ark-iam/install` 引入。
//
// 消费方（门面包 `@morehao/ark-iam-react/login`、login-web）从这里导入：
//   - 组件：LoginPage（4 mode 账密页）
//   - runtime config：ensureRuntimeConfig / loadRuntimeConfig / setAppBase / getRuntimeConfig /
//     getAppBase / appPath / detectBasePath / RuntimeConfig
//   - 认证 API：oidcLogin / oidcSelectTenant / oidcChangePassword / registerPerson / createTenant /
//     getLoginConfig / ApiError / 107004 守卫（isInstallPendingError / installRedirectForError）
//   - 类型：LoginPageProps / OIDCLoginResp / TenantSelectionItem …

export { LoginPage, default as LoginPageDefault } from './LoginPage'
export type { LoginPageProps } from './LoginPage'

export {
  ensureRuntimeConfig,
  loadRuntimeConfig,
  setAppBase,
  getRuntimeConfig,
  getAppBase,
  appPath,
  detectBasePath,
} from './config'
export type { RuntimeConfig } from './config'

export {
  ApiError,
  INSTALL_PENDING_REDIRECT,
  isInstallPendingError,
  installRedirectForError,
  oidcApi,
  oidcLogin,
  oidcSelectTenant,
  oidcChangePassword,
  registerPerson,
  createTenant,
  getLoginConfig,
} from './api'

export type {
  OwnerType,
  TenantSelectionItem,
  ApiResponse,
  OIDCLoginReq,
  OIDCLoginResp,
  OIDCChangePasswordReq,
  OIDCSelectTenantReq,
  RegisterPersonReq,
  RegisterPersonResp,
  CreateTenantReq,
  CreateTenantResp,
  OIDCLoginConfigReq,
  OIDCLoginConfigResp,
} from './types'
