import { Tag } from 'antd'

/**
 * 租户 API 密钥状态推导（参考平台端 platform-admin-web KeyStateTag 风格）。
 * 优先级：已吊销 > 已过期 > 有效。字段来自 @ark-iam/types TenantApiKeyItem（可空时间）。
 */
export interface KeyStateInput {
  revokedAt?: number | null
  expiresAt?: number | null
}

export function keyStateOf(r: KeyStateInput): { label: string; color: string } {
  if (r.revokedAt) return { label: '已吊销', color: 'error' }
  if (r.expiresAt != null && r.expiresAt > 0 && r.expiresAt * 1000 <= Date.now()) return { label: '已过期', color: 'warning' }
  return { label: '有效', color: 'success' }
}

export function KeyStateTag(r: KeyStateInput) {
  const st = keyStateOf(r)
  return <Tag color={st.color}>{st.label}</Tag>
}
