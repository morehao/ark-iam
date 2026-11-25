// @ark-iam/install —— /install 初始化协议包（从 @ark-iam/login 拆出的独立能力层）。
//
// 内容：install 协议契约类型（types）、用户可见文案（messages）、领域错误码与错误映射（errors）、
// 以及 status/initialize 两个接口调用（api）。
//
// 依赖方向：install → login（复用运行时 base 解析与 ApiError）；login 不反向依赖本包。
// 账密登录页（LoginPage）仍留在 @ark-iam/login；宿主做「首次部署初始化向导」时从这里按需引入。

// 协议契约类型 + 表单工具
export { DEFAULT_TENANT_NAME, createEmptyInstallForm } from './types'
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
} from './types'

// 用户可见文案
export { INSTALL_MESSAGES } from './messages'

// 领域错误码与错误映射
export {
  INSTALL_ERROR_CODE,
  describeInstallError,
  codeFromHTTPStatus,
  decodeErrorEnvelope,
  resolveInstallError,
  statusUnavailableMessage,
  unwrapEnvelope,
} from './errors'
export type { InstallErrorField, InstallErrorBehavior, UnwrapResult } from './errors'

// 接口调用
export { installApi, getInstallStatus, initializeInstall } from './api'

// "系统未初始化"守卫：跳出 /install 的判定与目标路径（实现留在 @ark-iam/login 的 ApiError 体系内，
// 统一从本包对外暴露，便于 install 域消费方（含登录侧跳转）只依赖单一入口）。
export { INSTALL_PENDING_REDIRECT, isInstallPendingError, installRedirectForError, ApiError } from '@ark-iam/login'
