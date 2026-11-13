// @morehao/ark-iam-react —— 聚合入口（一把梭：从根入口 import 全部）。
//
// 需要按需减小体积时，请优先用子路径：
//   import { BizCode } from '@morehao/ark-iam-react/types'          // 零重依赖
//   import { createApiClient } from '@morehao/ark-iam-react/api'
//   import { createAuthProvider } from '@morehao/ark-iam-react/auth'
//   import { actionColumn, themeConfig } from '@morehao/ark-iam-react/ui'  // 含 antd
//   import { LoginPage } from '@morehao/ark-iam-react/login'         // OP 账密页（含 axios/表单）
//
// 根入口会把五层全部 re-export（含 ui 的 antd 与 login 的 axios/表单依赖），仅供便利；
// 追求体积请用子路径。
export * from '@ark-iam/types'
export * from '@ark-iam/api'
export * from '@ark-iam/auth'
export * from '@ark-iam/ui'

// login 层单独显式 re-export：它自带的自足类型中，`createTenant` / `RuntimeConfig` /
// `ApiResponse` / `OwnerType` 与上面 api/auth/types 已导出的同名符号冲突（`export *`
// 会触发 TS2308 二义性）。这 4 个符号在根入口保留 api/auth/types 的原版；login 独有的
// 页面、runtime config、认证/install API 与协议类型在此补齐，需要 login 版同名类型时
// 请从子路径 `@morehao/ark-iam-react/login` 引入。
export {
  LoginPage,
  LoginPageDefault,
  ensureRuntimeConfig,
  loadRuntimeConfig,
  setAppBase,
  getRuntimeConfig,
  getAppBase,
  appPath,
  detectBasePath,
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
  getLoginConfig,
  getInstallStatus,
  initializeInstall,
  INSTALL_ERROR_CODE,
  describeInstallError,
  codeFromHTTPStatus,
  decodeErrorEnvelope,
  resolveInstallError,
  statusUnavailableMessage,
  unwrapEnvelope,
  INSTALL_MESSAGES,
  DEFAULT_TENANT_NAME,
  createEmptyInstallForm,
} from '@ark-iam/login'
export type {
  LoginPageProps,
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
  InstallErrorField,
  InstallErrorBehavior,
  UnwrapResult,
  InstallConsoles,
  InstallStatus,
  InstallInitializeReq,
  InstallInitializeResp,
  InstallChangeItem,
  InstallReport,
  InstallForm,
  InstallField,
  InstallFieldErrors,
} from '@ark-iam/login'
