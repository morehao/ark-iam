import { Tag } from 'antd'

/**
 * 状态标签规范（详见 frontend/DESIGN.md「Components → 状态 Tag」）
 *
 * 两类字典严格区分：
 * - 语义状态（enable/disable/挂起/验证）→ 语义色 success/default/error/warning；
 * - 分类标识（类型/内置来源等）→ 固定分类色，不做状态语义。
 * 禁止在页面内联用 green/red/orange 等传统色名表示状态，避免跨 app 配色漂移。
 */

export type StatusValue = string | number | null | undefined

/** 启用/停用状态：enable|1|active → 启用(success)；disable|0|inactive → 停用(default)；suspended → 挂起(error) */
export function StatusTag({ value }: { value?: StatusValue }) {
  const v = String(value ?? '')
  if (v === 'enable' || v === '1' || v === 'active') {
    return <Tag color="success">启用</Tag>
  }
  if (v === 'disable' || v === '0' || v === 'inactive') {
    return <Tag color="default">停用</Tag>
  }
  if (v === 'suspended') {
    return <Tag color="error">挂起</Tag>
  }
  return <Tag>{v || '-'}</Tag>
}

/** 挂起状态：isSuspended 字段，1/true → 挂起(error)；0/false → 正常(success) */
export function SuspendedTag({ value }: { value?: StatusValue | boolean }) {
  return value === 1 || value === true ? <Tag color="error">挂起</Tag> : <Tag color="success">正常</Tag>
}

/** 验证状态：isVerified 字段，1 → 已验证(success)；0 → 未验证(warning) */
export function VerifiedTag({ value }: { value?: StatusValue }) {
  return value === 1 ? <Tag color="success">已验证</Tag> : <Tag color="warning">未验证</Tag>
}

const TYPE_META: Record<string, { label: string; color: string }> = {
  platform: { label: '平台', color: 'geekblue' },
  customer: { label: '客户', color: 'cyan' },
  first_party: { label: '第一方', color: 'blue' },
  third_party: { label: '第三方', color: 'orange' },
}

/** 应用/租户类型标识（分类色，非状态语义） */
export function TypeTag({ value }: { value?: string }) {
  const meta = TYPE_META[value ?? '']
  return meta ? <Tag color={meta.color}>{meta.label}</Tag> : <Tag>{value || '-'}</Tag>
}

/** 角色来源标识：builtin → 内置(gold)；其余 → 自定义(blue) */
export function SourceTag({ value }: { value?: string }) {
  return value === 'builtin' ? <Tag color="gold">内置</Tag> : <Tag color="blue">自定义</Tag>
}
