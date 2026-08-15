import { Tooltip } from 'antd'

export interface IDCellProps {
  /** ID 值；空值展示为 - */
  value?: string | number | null
  /** 保留的前缀长度，默认 8 */
  prefix?: number
  /** 保留的后缀长度，默认 4 */
  suffix?: number
}

/**
 * 长 ID 单元格（UUID v7 等）：
 * - 默认仅展示首尾片段，如 `0190e2a4…c3d4`
 * - 鼠标悬浮展示完整 ID（可拖选直接复制，无独立复制按钮）
 */
export function IDCell({ value, prefix = 8, suffix = 4 }: IDCellProps) {
  const text = value == null ? '' : String(value)
  if (!text) return <span>-</span>
  const short = text.length > prefix + suffix + 1 ? `${text.slice(0, prefix)}…${text.slice(-suffix)}` : text
  return (
    <Tooltip
      title={text}
      mouseEnterDelay={0.3}
      mouseLeaveDelay={0.6}
      styles={{ body: { userSelect: 'text' } }}
    >
      <span style={{ fontFamily: 'ui-monospace, SFMono-Regular, Menlo, monospace', fontSize: 12 }}>{short}</span>
    </Tooltip>
  )
}
