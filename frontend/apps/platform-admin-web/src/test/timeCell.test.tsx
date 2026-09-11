import { describe, expect, it } from 'vitest'
import { render, screen } from '@testing-library/react'
import {
  fmtRelativeTime,
  fmtTime,
  TIME_COL_WIDTH,
  TIME_COL_WIDTH_RELATIVE,
  TimeCell,
  timeColumn,
} from '@ark-iam/ui'

/**
 * 时间列回归测试（2026-09 列表页时间折行问题）。
 *
 * 为什么放在 app 侧：`packages/ui` 目前只有源码，未配置 jsdom 测试环境；
 * 这里复用 platform-admin-web 已有的 vitest(jsdom) + testing-library。
 *
 * 锁定三条不变量：
 * 1. 时间列宽 ≥ 完整时间串实测宽度 + antd 单元格内边距（146.06 + 16×2 = 178.06 → 180）；
 * 2. 单元格 `white-space: nowrap` —— 折行的结构性保证；
 * 3. 空值一律走 placeholder，不再散落 `v ? fmtTime(v) : '自定义文案'` 三元表达式。
 */
const SAMPLE = 1789142400 // 2026-09-11 08:00:00 (UTC+8)

describe('TimeCell / timeColumn', () => {
  it('列宽常量覆盖完整时间串所需的最小宽度', () => {
    // 14px + 全站 tabular-nums 实测 146.06px；antd v5 单元格左右内边距各 16px
    expect(TIME_COL_WIDTH).toBeGreaterThanOrEqual(178.06)
    // 相对时间列（「3 天前」/「从未使用」）可以明显更窄
    expect(TIME_COL_WIDTH_RELATIVE).toBeLessThan(TIME_COL_WIDTH)
  })

  it('绝对时间单行展示并强制 nowrap', () => {
    render(<TimeCell value={SAMPLE} />)
    const el = screen.getByText(fmtTime(SAMPLE))
    expect(el).toBeInTheDocument()
    expect(el).toHaveStyle({ whiteSpace: 'nowrap' })
  })

  it('空值走 placeholder：默认 -、永不过期、从未使用', () => {
    const { unmount } = render(<TimeCell value={null} />)
    expect(screen.getByText('-')).toBeInTheDocument()
    unmount()

    render(<TimeCell value={0} placeholder="永不过期" />)
    expect(screen.getByText('永不过期')).toBeInTheDocument()
  })

  it('relative 模式展示相对时间（完整时间由 Tooltip 兜底）', () => {
    const threeDaysAgo = Math.floor(Date.now() / 1000) - 3 * 86400
    render(<TimeCell value={threeDaysAgo} relative />)
    const el = screen.getByText('3 天前')
    expect(el).toBeInTheDocument()
    expect(el).toHaveStyle({ whiteSpace: 'nowrap' })
  })

  it('timeColumn 统一列宽与渲染，禁止各页自写 150/160/170', () => {
    const absolute = timeColumn<{ createdAt: number }>({ title: '创建时间', dataIndex: 'createdAt' })
    const relative = timeColumn<{ lastUsedAt: number }>({ title: '最后使用', dataIndex: 'lastUsedAt', relative: true })

    expect(absolute.title).toBe('创建时间')
    expect(absolute.key).toBe('createdAt')
    expect(absolute.width).toBe(TIME_COL_WIDTH)
    expect(relative.width).toBe(TIME_COL_WIDTH_RELATIVE)

    // 工厂产出的 render 必须仍是 nowrap 的 TimeCell
    render(<>{absolute.render?.(SAMPLE, { createdAt: SAMPLE }, 0)}</>)
    expect(screen.getByText(fmtTime(SAMPLE))).toHaveStyle({ whiteSpace: 'nowrap' })
  })

  it('fmtRelativeTime 分档、空值与未来时间回退', () => {
    const now = 1_800_000_000_000 // 毫秒基准
    const at = (secondsAgo: number) => Math.floor(now / 1000) - secondsAgo

    expect(fmtRelativeTime(at(30), now)).toBe('刚刚')
    expect(fmtRelativeTime(at(5 * 60), now)).toBe('5 分钟前')
    expect(fmtRelativeTime(at(3 * 3600), now)).toBe('3 小时前')
    expect(fmtRelativeTime(at(3 * 86400), now)).toBe('3 天前')
    expect(fmtRelativeTime(at(3 * 30 * 86400), now)).toBe('3 个月前')
    expect(fmtRelativeTime(at(3 * 365 * 86400), now)).toBe('3 年前')

    // 空值返回空串，由 TimeCell 决定占位文案
    expect(fmtRelativeTime(0, now)).toBe('')
    expect(fmtRelativeTime(null, now)).toBe('')
    // 未来时间（时钟漂移）不猜「N 分钟后」，回退绝对时间
    expect(fmtRelativeTime(at(-3600), now)).toBe(fmtTime(at(-3600)))
  })
})
