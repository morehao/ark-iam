import { Tooltip } from 'antd'

export interface EllipsisCellProps {
  /** 文本内容；空值展示为 - */
  value?: string | number | null
  /** 展示前先截断到该字符数（仅影响展示，悬浮提示仍为完整内容） */
  limit?: number
  /** 使用等宽字体 */
  monospace?: boolean
}

/**
 * 可能展示不全的文本单元格：
 * - 超出单元格宽度即省略号截断
 * - 鼠标悬浮展示完整内容（可拖选直接复制）
 */
export function EllipsisCell({ value, limit, monospace = false }: EllipsisCellProps) {
  const text = value == null ? '' : String(value)
  if (!text) return <span>-</span>
  const display = limit && text.length > limit ? `${text.slice(0, limit)}…` : text
  return (
    <Tooltip
      title={text}
      mouseEnterDelay={0.3}
      mouseLeaveDelay={0.6}
      styles={{ body: { userSelect: 'text' } }}
    >
      <span
        style={{
          display: 'inline-block',
          maxWidth: '100%',
          overflow: 'hidden',
          textOverflow: 'ellipsis',
          whiteSpace: 'nowrap',
          verticalAlign: 'bottom',
          ...(monospace ? { fontFamily: 'ui-monospace, SFMono-Regular, Menlo, monospace', fontSize: 12 } : {}),
        }}
      >
        {display}
      </span>
    </Tooltip>
  )
}
