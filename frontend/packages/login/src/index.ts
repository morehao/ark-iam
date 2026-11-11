// @ark-iam/login 包入口：导出 OP 账密登录页 + 运行时配置 / 认证 API / install 辅助 + 类型。
//
// 消费方（门面包 `@morehao/ark-iam-react/login`、login-web）从这里导入：
//   - 组件：LoginPage（4 mode 账密页）
//   - runtime config：ensureRuntimeConfig / loadRuntimeConfig / setAppBase / getRuntimeConfig /
//     getAppBase / appPath / detectBasePath / RuntimeConfig
//   - 认证 API：oidcLogin / oidcSelectTenant / oidcChangePassword / registerPerson / createTenant /
//     getLoginConfig / install 相关（getInstallStatus / initializeInstall / installRedirectForError …）
//   - 类型：LoginPageProps / OIDCLoginResp / TenantSelectionItem / Install* 等

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
  installApi,
  oidcLogin,
  oidcSelectTenant,
  oidcChangePassword,
  registerPerson,
  createTenant,
  getLoginConfig,
  getInstallStatus,
  initializeInstall,
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

export {
  INSTALL_ERROR_CODE,
  describeInstallError,
  codeFromHTTPStatus,
  decodeErrorEnvelope,
  resolveInstallError,
  statusUnavailableMessage,
  unwrapEnvelope,
} from './install/errors'
export type {
  InstallErrorField,
  InstallErrorBehavior,
  UnwrapResult,
} from './install/errors'
export { INSTALL_MESSAGES } from './install/messages'
export { DEFAULT_TENANT_NAME, createEmptyInstallForm } from './install/types'
export type {
  InstallConsoles,
  InstallStatus,
  InstallInitializeReq,
  InstallInitializeResp,
  InstallChangeItem,
  InstallReport,
  InstallForm,
  InstallField,
  InstallFieldErrors,
} from './install/types'
