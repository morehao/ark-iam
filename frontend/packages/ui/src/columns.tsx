import type { ReactNode } from 'react'
import type { ColumnType } from 'antd/es/table'
import { EllipsisCell } from './EllipsisCell'
import { IDCell } from './IDCell'
import { NameLink } from './NameLink'
import { RowActions, type RowAction } from './RowActions'

/**
 * 列表列宽标准（px）与列工厂（规范见 frontend/DESIGN.md §7.3）。
 *
 * 背景：历史 bug 是各页手写列宽 + 手写 `scroll.x`，列宽普遍偏大（ID 150、编码 180…）
 * 叠加后总宽轻松超过内容区，导致「一进列表就有横向滚动条、操作列要横滑很久才看得到」。
 * 这里把所有列宽收口成具名常量与工厂，页面不再出现魔法数字。
 *
 * 列宽 = 内容实测宽 + antd v5 单元格左右内边距（16 × 2 = 32）。
 */

/** ID 列：`IDCell` 默认展示「首 8 + … + 尾 4」共 13 个 12px 等宽字符（≈93.6px）+ 内边距。 */
export const ID_COL_WIDTH = 130
/** 名称列：约 10 个汉字；过长由 `NameLink` / `EllipsisCell` 省略号截断。 */
export const NAME_COL_WIDTH = 180
/** 编码 / key / 域名等等宽短标识列。 */
export const CODE_COL_WIDTH = 150
/** 分类 Tag 列（类型 / 可见性 / 来源等）。 */
export const TAG_COL_WIDTH = 110
/** 状态 Tag 列（正常 / 挂起 / 启用 / 停用）。 */
export const STATUS_COL_WIDTH = 100
/** 数字计数列（成员数、角色数、排序）。 */
export const COUNT_COL_WIDTH = 90
/** 自由文本列（描述、备注等），一律省略号截断。 */
export const TEXT_COL_WIDTH = 200
/** 长文本列（日志 payload、长描述）：比 `TEXT_COL_WIDTH` 宽一档，仍省略号截断。 */
export const LONG_TEXT_COL_WIDTH = 320

/** antd v5 单元格左右内边距合计（16 × 2）。 */
const CELL_PADDING = 32
/** 操作列横排按钮宽度预算：约 3 个汉字（42px）+ link 按钮左右内边距（14px）。 */
const ACTION_BUTTON_BUDGET = 56
/** `RowActions` 横排 `Space` 间隔。 */
const ACTION_GAP = 4

/* ------------------------------------------------------------------ *
 * 列工厂
 * ------------------------------------------------------------------ */

export interface IDColumnConfig<T> {
  /** 记录中的 ID 字段 */
  dataIndex: keyof T & string
  /** 列标题，默认 `ID` */
  title?: string
  /** 覆盖默认列宽（默认 `ID_COL_WIDTH`） */
  width?: number
}

/**
 * ID 列：`IDCell`（等宽 + 首 8 尾 4 + Tooltip 可拖选复制）。
 *
 * 同一行已有对外可读业务键（编码 / key / 域名 / 客户端ID）时，内部 UUID 列通常是噪声，
 * 可按需省略；保留时一律用本工厂（默认 130，而非手写 150）。
 */
export function idColumn<T>({ dataIndex, title = 'ID', width }: IDColumnConfig<T>): ColumnType<T> {
  return {
    title,
    dataIndex,
    key: dataIndex,
    width: width ?? ID_COL_WIDTH,
    render: (value: unknown) => <IDCell value={value as string | number | null | undefined} />,
  }
}

export interface TextColumnConfig<T> {
  title: ReactNode
  /** 记录字段名；同时作为列的 key */
  dataIndex: keyof T & string
  /** 列宽；不传时取 `TEXT_COL_WIDTH` */
  width?: number
  /** 空值占位文案，默认 `-` */
  placeholder?: string
  /** 等宽字体（编码 / key / 路径等） */
  monospace?: boolean
}

/**
 * 文本列：`EllipsisCell`（超出省略号截断 + 悬浮展示全文），列宽固定不随内容膨胀。
 */
export function textColumn<T>({
  title,
  dataIndex,
  width,
  placeholder,
  monospace,
}: TextColumnConfig<T>): ColumnType<T> {
  return {
    title,
    dataIndex,
    key: dataIndex,
    width: width ?? TEXT_COL_WIDTH,
    render: (value: unknown) => (
      <EllipsisCell
        value={value as string | number | null | undefined}
        placeholder={placeholder}
        monospace={monospace}
      />
    ),
  }
}

export interface NameColumnConfig<T> {
  title: ReactNode
  /** 记录字段名；同时作为列的 key */
  dataIndex: keyof T & string
  /** 列宽；不传时取 `NAME_COL_WIDTH` */
  width?: number
  /** 等宽字体（日志键 / 编码类名称） */
  monospace?: boolean
  /** 点击名称的回调（详情 / 下级入口）；不传时退化为普通文本 */
  onClick?: (record: T) => void
}

/**
 * 名称列：`NameLink`（主色链接，详情入口；见 DESIGN.md §7.3 R2）。
 */
export function nameColumn<T>({
  title,
  dataIndex,
  width,
  monospace,
  onClick,
}: NameColumnConfig<T>): ColumnType<T> {
  return {
    title,
    dataIndex,
    key: dataIndex,
    width: width ?? NAME_COL_WIDTH,
    render: (value: unknown, record: T) => (
      <NameLink
        value={value as ReactNode}
        monospace={monospace}
        onClick={onClick ? () => onClick(record) : undefined}
      />
    ),
  }
}

export interface ActionColumnConfig<T> {
  /** 由记录生成操作列表，按重要程度从高到低排列 */
  actions: (record: T) => RowAction[]
  /** 横排上限，超过则收起为「更多」下拉；默认 3 */
  max?: number
  /** 覆盖自动计算的列宽 */
  width?: number
}

/**
 * 操作列工厂（列表页操作列规范，DESIGN.md §7.3 R1）：
 * - 统一 `RowActions` 渲染（≤ max 横排，> max 收「更多」）；
 * - **永远 `fixed: 'right'`**：即使表格出现横向滚动，操作列也钉在右侧可见，
 *   不会再出现「横向滑动很久才看到操作」；
 * - 列宽由 `actionColumnWidth(max)` 统一给出，页面禁止再手写 120/200/240。
 */
export function actionColumn<T>({ actions, max = 3, width }: ActionColumnConfig<T>): ColumnType<T> {
  return {
    title: '操作',
    key: 'action',
    fixed: 'right',
    width: width ?? actionColumnWidth(max),
    render: (_, record) => <RowActions actions={actions(record)} max={max} />,
  }
}

/**
 * 操作列宽度估算（px）：`单元格内边距 32 + 横排数 × 56 + 间距 4`，向上取整到 10。
 *
 * 56 = 约 3 个汉字（42px）+ link 按钮左右内边距（14px）。**横排按钮文案应保持简短
 * （≤ 4 个汉字）**；文案更长时请降低 `max` 把它收进「更多」下拉，或显式传入 `width`。
 */
export function actionColumnWidth(max: number): number {
  const inline = Math.max(1, max)
  const raw = CELL_PADDING + inline * ACTION_BUTTON_BUDGET + (inline - 1) * ACTION_GAP
  return Math.ceil(raw / 10) * 10
}

/* ------------------------------------------------------------------ *
 * 横向滚动
 * ------------------------------------------------------------------ */

type WidthableColumn<T> = ColumnType<T> & { children?: WidthableColumn<T>[] }

/**
 * 由列定义计算表格 `scroll.x`：所有列宽之和。
 *
 * 取代各页手写的 `scroll={{ x: 1490 }}`——手写值会随列增删漂移（历史上 apiKey 页
 * 实际列宽合计已到 1580 却仍写 1520），且写大了会让本可放下的表格也强制出现横滚条。
 *
 * 配合 `tableLayout="fixed"`：列宽严格生效（`EllipsisCell` / `TimeCell` 才能正确截断），
 * 总宽小于容器时表格按 `min-width: 100%` 铺满，不会留白也不会出现滚动条。
 *
 * ```tsx
 * <Table columns={columns} scroll={tableScrollX(columns)} tableLayout="fixed" />
 * ```
 */
export function tableScrollX<T>(columns: readonly WidthableColumn<T>[]): { x: number } {
  let total = 0
  const walk = (list: readonly WidthableColumn<T>[]) => {
    for (const column of list) {
      if (column.children?.length) {
        walk(column.children)
        continue
      }
      total += typeof column.width === 'number' ? column.width : 0
    }
  }
  walk(columns)
  return { x: total }
}
