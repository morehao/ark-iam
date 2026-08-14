import { Tag } from 'antd'

/** 通用状态标签：enable/disable、isSuspended、isVerified 等字典渲染 */
export function StatusTag({ value }: { value?: string | number }) {
  const v = String(value ?? '')
  if (v === 'enable' || v === '1' || v === 'active') {
    return <Tag color="success">启用</Tag>
  }
  if (v === 'disable' || v === '0' || v === 'inactive' || v === 'suspended') {
    return <Tag color="default">停用</Tag>
  }
  return <Tag>{v || '-'}</Tag>
}

export function SuspendedTag({ value }: { value?: number }) {
  return value === 1 ? <Tag color="error">挂起</Tag> : <Tag color="success">正常</Tag>
}

export function VerifiedTag({ value }: { value?: number }) {
  return value === 1 ? <Tag color="success">已验证</Tag> : <Tag color="warning">未验证</Tag>
}

export function TypeTag({ value }: { value?: string }) {
  switch (value) {
    case 'platform':
      return <Tag color="geekblue">平台</Tag>
    case 'customer':
      return <Tag color="cyan">客户</Tag>
    case 'first_party':
      return <Tag color="blue">第一方</Tag>
    case 'third_party':
      return <Tag color="orange">第三方</Tag>
    default:
      return <Tag>{value || '-'}</Tag>
  }
}

/** 时间渲染：兼容秒级时间戳与字符串 */
export function fmtTime(value?: number | string | null): string {
  if (value == null || value === '' || value === 0) return '-'
  if (typeof value === 'number') {
    const ms = value < 1e12 ? value * 1000 : value
    const d = new Date(ms)
    if (Number.isNaN(d.getTime())) return String(value)
    const pad = (n: number) => String(n).padStart(2, '0')
    return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())} ${pad(d.getHours())}:${pad(d.getMinutes())}:${pad(d.getSeconds())}`
  }
  return value
}
