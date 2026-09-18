import { Tag } from 'antd'

/**
 * 状态标签规范（详见 frontend/DESIGN.md「Components → 状态 Tag」）
 *
 * 两类字典严格区分：
 * - 语义状态（enable/disable/挂起/验证）→ 语义色 success/default/error/warning；
 * - 分类标识（类型/来源等）→ 固定分类色，不做状态语义。
 * 禁止在页面内联用 green/red/orange 等传统色名表示状态，避免跨 app 配色漂移。
 */

export type StatusValue = string | number | null | undefined

/** 启用/停用状态：enable → 启用(success)；disable → 停用(default)；其余原样回显 */
export function EnableTag({ value }: { value?: string }) {
  if (value === 'enable') {
    return <Tag color="success">启用</Tag>
  }
  if (value === 'disable') {
    return <Tag color="default">停用</Tag>
  }
  return <Tag>{value || '-'}</Tag>
}

/**
 * 挂起状态：只认 status 枚举（后端 model.UserStatus / PersonStatus），
 * 'suspended' → 挂起(error)；'active' → 正常(success)。
 *
 * **不再兼容 isSuspended 的 1/true/0/false**：P3 起 user/person 状态统一为 status 枚举，
 * 保留对布尔/数字的兼容会掩盖「读了已下线字段」的缺陷（值恒为 undefined → 恒显示「正常」）。
 */
export function SuspendedTag({ value }: { value?: StatusValue | boolean }) {
  return value === 'suspended' ? <Tag color="error">挂起</Tag> : <Tag color="success">正常</Tag>
}

/**
 * 验证状态（后端具名类型 model.DomainVerificationStatus）：只认 'verified' → 已验证(success)；
 * 'unverified' 及其它取值 → 未验证(warning)。不再兼容 isVerified 的 1/0。
 */
export function VerifiedTag({ value }: { value?: StatusValue }) {
  return value === 'verified' ? <Tag color="success">已验证</Tag> : <Tag color="warning">未验证</Tag>
}

/** 类型标识（分类色，非状态语义）：租户类型 platform→geekblue、customer→cyan；其余原样回显 */
const TYPE_META: Record<string, { label: string; color: string }> = {
  platform: { label: '平台', color: 'geekblue' },
  customer: { label: '客户', color: 'cyan' },
}

/** 应用/租户类型标识（分类色，非状态语义） */
export function TypeTag({ value }: { value?: string }) {
  const meta = TYPE_META[value ?? '']
  return meta ? <Tag color={meta.color}>{meta.label}</Tag> : <Tag>{value || '-'}</Tag>
}

/**
 * 来源标识（分类色，非状态语义），一套映射覆盖两种来源字典：
 * - 应用 / OAuth 客户端 `source`（`model.AppSource`）：builtin→内置(gold)、first_party→第一方(blue)、third_party→第三方(orange)；
 * - 角色 `source`（`model.RoleSource`）：builtin→内置(gold)、custom→自定义(blue)。
 */
const SOURCE_META: Record<string, { label: string; color: string }> = {
  builtin: { label: '内置', color: 'gold' },
  first_party: { label: '第一方', color: 'blue' },
  third_party: { label: '第三方', color: 'orange' },
  custom: { label: '自定义', color: 'blue' },
}

/** 来源标识：应用/客户端 source 与角色 source 共用 */
export function SourceTag({ value }: { value?: string }) {
  const meta = SOURCE_META[value ?? '']
  return meta ? <Tag color={meta.color}>{meta.label}</Tag> : <Tag>{value || '-'}</Tag>
}
