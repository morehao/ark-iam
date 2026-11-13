import type { CSSProperties, ReactNode } from 'react'
import { Tooltip, Typography } from 'antd'

export interface NameLinkProps {
  /** 名称内容；空值展示为 - */
  value?: ReactNode
  /** 点击查看详情；未提供时退化为普通文本 */
  onClick?: () => void
  /** 悬浮提示；默认取 value 的字符串形式 */
  title?: string
  /** 使用等宽字体（日志键 / 编码类名称） */
  monospace?: boolean
}

/**
 * 名称单元格：列表行的「名称即详情入口」（列表页操作列规范，见 DESIGN.md §7.3）。
 *
 * 详情不再占用操作列，而是通过点击名称打开；因此名称渲染为主色链接
 * （hover 下划线 + 手型光标），让用户一眼看出可点击。名称过长时省略号截断并悬浮展示全称。
 */
export function NameLink({ value, onClick, title, monospace = false }: NameLinkProps) {
  const empty = value == null || value === ''
  const text: ReactNode = empty ? '-' : value
  const style: CSSProperties = {
    display: 'inline-block',
    fontWeight: 500,
    maxWidth: '100%',
    overflow: 'hidden',
    textOverflow: 'ellipsis',
    whiteSpace: 'nowrap',
    verticalAlign: 'bottom',
    ...(monospace ? { fontFamily: 'ui-monospace, SFMono-Regular, Menlo, monospace', fontSize: 12 } : {}),
  }

  if (empty || !onClick) {
    return <span style={style}>{text}</span>
  }

  const tooltip =
    title ?? (typeof value === 'string' || typeof value === 'number' ? String(value) : undefined)

  return (
    <Tooltip title={tooltip} mouseEnterDelay={0.3}>
      <Typography.Link onClick={onClick} style={style}>
        {text}
      </Typography.Link>
    </Tooltip>
  )
}
