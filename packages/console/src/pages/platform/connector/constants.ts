import type { ConnectorCapability, ConnectorProvider } from '@ark-iam/types'

/** 连接器提供商展示名（后端 model.ConnectorProvider 字典，与 connector_factory 一致）。 */
export const PROVIDER_LABEL: Record<ConnectorProvider, string> = {
  google: 'Google',
  github: 'GitHub',
  microsoft: 'Microsoft Entra ID',
  wechat: '微信',
}

/** 提供商 Tag 颜色（分类色，非状态语义）。 */
export const PROVIDER_COLOR: Record<ConnectorProvider, string> = {
  google: 'blue',
  github: 'default',
  microsoft: 'cyan',
  wechat: 'green',
}

/** 连接器能力展示名（后端 model.ConnectorCapability）。 */
export const CAPABILITY_LABEL: Record<ConnectorCapability, string> = {
  authorize: '发起授权',
  callback: '处理回调',
  claim_mapping: '声明映射',
  domain_policy: '域策略',
  profile_sync: '资料同步',
}
