import type { ColumnType } from 'antd/es/table'
import { Tooltip } from 'antd'
import { fmtRelativeTime, fmtTime } from './format'
import { tokens } from './theme'

/**
 * 时间列标准宽度（px）。
 *
 * 依据：完整时间串 `2026-09-03 17:19:46` 在 antd 表格 14px 字号 + 全站 `tabular-nums`
 * 下实测 146.06px（macOS / Chrome / 系统字体栈，为常见平台里较宽的一种）；
 * antd v5 单元格水平内边距为 `token.padding` × 2 = 32px，故单行显示至少需要 178.06px，
 * 取 180 留余量。低于该值的列宽会让时间恰好在唯一的空格处折行成「日期 / 时间」两行
 * （规范见 frontend/DESIGN.md §7.3）。
 */
export const TIME_COL_WIDTH = 180

/**
 * 相对时间列宽度（px）：如「3 天前」实测 40.47px、最长档「10 个月前」约 63px、
 * 占位文案「从未使用」56px，加 32px 内边距后 120 有充足余量。
 */
export const TIME_COL_WIDTH_RELATIVE = 120

export interface TimeCellProps {
  /** 秒级时间戳（约定见 AGENTS.md），兼容字符串与毫秒值 */
  value?: number | string | null
  /** 空值占位文案（默认 '-'），如「永不过期」「从未使用」 */
  placeholder?: string
  /** 相对时间模式：展示「3 天前」，悬浮给出完整 `YYYY-MM-DD HH:mm:ss` */
  relative?: boolean
}

/**
 * 表格时间单元格：统一 `YYYY-MM-DD HH:mm:ss`，并强制单行不折行。
 *
 * 所有时间列都必须走本组件（或 `timeColumn` 工厂），禁止页面直接 `fmtTime` 且自定列宽 —— 
 * 列宽与 `whiteSpace: nowrap` 是一对，缺任何一半都会出现"折行"或"溢出串列"。
 */
export function TimeCell({ value, placeholder = '-', relative = false }: TimeCellProps) {
  if (value == null || value === '' || value === 0) {
    return <span style={{ color: tokens.textPlaceholder }}>{placeholder}</span>
  }
  const absolute = fmtTime(value)
  if (!relative) return <span style={{ whiteSpace: 'nowrap' }}>{absolute}</span>
  const text = fmtRelativeTime(value)
  return (
    <Tooltip title={absolute} mouseEnterDelay={0.3}>
      <span style={{ whiteSpace: 'nowrap' }}>{text || absolute}</span>
    </Tooltip>
  )
}

export interface TimeColumnConfig<T> {
  title: string
  /** 记录字段名；同时作为列的 key */
  dataIndex: keyof T & string
  /** 覆盖默认列宽（默认绝对时间 180 / 相对时间 120） */
  width?: number
  placeholder?: string
  /** 次要时间字段（最后使用 / 登录时间等）用相对时间 + Tooltip */
  relative?: boolean
}

/**
 * 时间列工厂：一次给出「标题 + 字段 + 标准列宽 + TimeCell 渲染」，
 * 避免各页面手写列宽（历史 bug 根因就是每页各写 150/160/170）。
 *
 * ```tsx
 * const columns: ColumnsType<TenantItem> = [
 *   timeColumn<TenantItem>({ title: '创建时间', dataIndex: 'createdAt' }),
 *   timeColumn<TenantItem>({ title: '最后使用', dataIndex: 'lastUsedAt', relative: true, placeholder: '从未使用' }),
 * ]
 * ```
 */
export function timeColumn<T>({
  title,
  dataIndex,
  width,
  placeholder,
  relative = false,
}: TimeColumnConfig<T>): ColumnType<T> {
  return {
    title,
    dataIndex,
    key: dataIndex,
    width: width ?? (relative ? TIME_COL_WIDTH_RELATIVE : TIME_COL_WIDTH),
    render: (value: unknown) => (
      <TimeCell value={value as number | string | null | undefined} placeholder={placeholder} relative={relative} />
    ),
  }
}
