import { describe, expect, it } from 'vitest'
import { render, screen } from '@testing-library/react'
import type { ReactNode } from 'react'
import { App as AntdApp } from 'antd'
import type { ColumnType } from 'antd/es/table'
import {
  actionColumn,
  actionColumnWidth,
  CODE_COL_WIDTH,
  EllipsisCell,
  ID_COL_WIDTH,
  idColumn,
  nameColumn,
  STATUS_COL_WIDTH,
  tableScrollX,
  textColumn,
  timeColumn,
} from '@ark-iam/ui'

/**
 * 列表列宽 / 操作列回归测试。
 *
 * 背景（2026-09 修复）：各页手写列宽偏大 + 手写 `scroll.x`，导致列表一进来就有横向滚动条，
 * 「操作」列要横滑很久才看得到。收敛后的三条不变量：
 * 1. 列宽来自具名常量，ID 列收窄到 `ID_COL_WIDTH`（仍能放下 IDCell 的 8+…+4 等宽片段）；
 * 2. 操作列一律 `fixed: 'right'`，宽度由 `actionColumnWidth(max)` 统一给出且窄于旧的手写 240；
 * 3. `tableScrollX(columns)` 由列宽求和得出，取代会漂移的手写数字。
 */
interface Row {
  id: string
  name: string
  code: string
  createdAt: number
}

const row: Row = { id: '0190e2a4-1111-2222-3333-444455556666', name: '示例', code: 't_platform', createdAt: 1789142400 }

/**
 * 渲染某个列的单元格。`ColumnType.render` 的返回类型含 `RenderedCell` 变体
 * （用于 `colSpan`/`rowSpan` 合并），本测试只走普通 ReactNode 分支。
 */
function renderCell<T>(column: ColumnType<T>, value: unknown, record: T): ReactNode {
  return column.render?.(value, record, 0) as ReactNode
}

describe('列宽常量', () => {
  it('ID 列宽仍能容纳 IDCell 的「首 8 + … + 尾 4」等宽片段 + 单元格内边距', () => {
    // 13 个字符 × 12px 等宽（≈7.2px/字符）≈ 93.6px，加 antd 单元格左右 16×2 = 32px
    expect(ID_COL_WIDTH).toBeGreaterThanOrEqual(125.6)
    // 回归：旧实现各页手写 150，普遍偏宽
    expect(ID_COL_WIDTH).toBeLessThan(150)
  })

  it('列宽常量互相协调（状态列窄于文本列，编码列窄于长文本列）', () => {
    expect(STATUS_COL_WIDTH).toBeLessThan(CODE_COL_WIDTH)
    expect(CODE_COL_WIDTH).toBeGreaterThan(0)
  })
})

describe('actionColumn', () => {
  it('操作列固定在右侧，宽度由 max 推导且窄于旧的手写 240', () => {
    const column = actionColumn<Row>({ actions: () => [{ key: 'edit', label: '编辑' }] })

    expect(column.title).toBe('操作')
    expect(column.fixed).toBe('right')
    expect(column.width).toBe(actionColumnWidth(3))
    expect(actionColumnWidth(3)).toBeLessThan(240)
  })

  it('actionColumnWidth 随横排数量单调递增，且允许显式覆盖', () => {
    expect(actionColumnWidth(1)).toBeLessThan(actionColumnWidth(2))
    expect(actionColumnWidth(2)).toBeLessThan(actionColumnWidth(3))

    const pinned = actionColumn<Row>({ actions: () => [], max: 1, width: 88 })
    expect(pinned.width).toBe(88)
  })

  it('操作数 ≤ max 时全部横排，> max 时收进「更多」下拉', () => {
    const two = actionColumn<Row>({
      max: 2,
      actions: () => [
        { key: 'edit', label: '编辑' },
        { key: 'delete', label: '删除' },
      ],
    })
    render(<AntdApp>{renderCell(two, undefined, row)}</AntdApp>)
    expect(screen.getByText('编辑')).toBeInTheDocument()
    expect(screen.getByText('删除')).toBeInTheDocument()
    expect(screen.queryByText(/更多/)).toBeNull()
  })

  it('超过 max 的长文案操作收进「更多」，不撑宽操作列', () => {
    const three = actionColumn<Row>({
      max: 2,
      actions: () => [
        { key: 'edit', label: '编辑' },
        { key: 'reset', label: '重置管理员密码' },
        { key: 'delete', label: '删除' },
      ],
    })
    render(<AntdApp>{renderCell(three, undefined, row)}</AntdApp>)
    // 横排仅保留最高优先级操作 + 「更多」
    expect(screen.getByText('编辑')).toBeInTheDocument()
    expect(screen.getByText(/更多/)).toBeInTheDocument()
    // 次要/危险操作在收起状态下不渲染
    expect(screen.queryByText('重置管理员密码')).toBeNull()
  })
})

describe('列工厂', () => {
  it('idColumn / textColumn / nameColumn 给出标准列宽与渲染器', () => {
    const id = idColumn<Row>({ dataIndex: 'id' })
    expect(id.width).toBe(ID_COL_WIDTH)
    render(<>{renderCell(id, row.id, row)}</>)
    // 长 ID 展示为「首 8…尾 4」
    expect(screen.getByText('0190e2a4…6666')).toBeInTheDocument()

    const text = textColumn<Row>({ title: '编码', dataIndex: 'code', width: CODE_COL_WIDTH, monospace: true })
    expect(text.width).toBe(CODE_COL_WIDTH)
    render(<>{renderCell(text, row.code, row)}</>)
    expect(screen.getByText('t_platform')).toBeInTheDocument()

    const name = nameColumn<Row>({ title: '名称', dataIndex: 'name' })
    render(<>{renderCell(name, row.name, row)}</>)
    expect(screen.getByText('示例')).toBeInTheDocument()
  })

  it('textColumn 空值走 placeholder（EllipsisCell 语义）', () => {
    const text = textColumn<Row>({ title: '描述', dataIndex: 'name', placeholder: '未设置' })
    render(<>{renderCell(text, null, row)}</>)
    expect(screen.getByText('未设置')).toBeInTheDocument()
    // 默认占位仍是 -
    render(<EllipsisCell value={null} />)
    expect(screen.getAllByText('-').length).toBeGreaterThan(0)
  })
})

describe('tableScrollX', () => {
  it('等于各列宽之和（取代会漂移的手写 scroll.x）', () => {
    const columns: ColumnType<Row>[] = [
      idColumn<Row>({ dataIndex: 'id' }),
      textColumn<Row>({ title: '名称', dataIndex: 'name', width: 160 }),
      timeColumn<Row>({ title: '创建时间', dataIndex: 'createdAt' }),
      actionColumn<Row>({ max: 2, actions: () => [] }),
    ]

    expect(tableScrollX(columns)).toEqual({
      x: ID_COL_WIDTH + 160 + 180 + actionColumnWidth(2),
    })
  })

  it('递归统计列分组的叶子列宽', () => {
    const grouped = {
      title: '分组',
      children: [
        textColumn<Row>({ title: '名称', dataIndex: 'name', width: 100 }),
        textColumn<Row>({ title: '编码', dataIndex: 'code', width: 120 }),
      ],
    }

    expect(tableScrollX([grouped])).toEqual({ x: 220 })
  })
})
