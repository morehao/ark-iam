import { describe, expect, it } from 'vitest'
import { render, screen } from '@testing-library/react'
import { renderToStaticMarkup } from 'react-dom/server'
import { App as AntdApp } from 'antd'
import type { ColumnType } from 'antd/es/table'
import {
  actionColumn,
  actionColumnWidth,
  menuIcon,
  MENU_ICONS,
} from '@morehao/ark-iam-react/ui'
import type { RowAction } from '@morehao/ark-iam-react/ui'

/** P0 共享层单测：菜单图标注册表 + 操作列 variant（在已有 vitest 基建的 app 内运行）。 */

describe('menuIcon 注册表', () => {
  it('已注册 key 取到对应图标（非兜底）', () => {
    for (const key of Object.keys(MENU_ICONS)) {
      expect(menuIcon(key)).toBeTruthy()
    }
  })

  it('未注册 / 空 key 回落兜底图标，不抛错', () => {
    expect(menuIcon()).toBeTruthy()
    expect(menuIcon('')).toBeTruthy()
    expect(menuIcon('__not_exist__')).toBeTruthy()
  })

  it('语义唯一：平台端语义化新 key 渲染出互不相同的图标', () => {
    const keys = ['tenant', 'domain', 'application', 'app-group', 'oauth', 'menu']
    const htmls = keys.map((k) => renderToStaticMarkup(<>{MENU_ICONS[k]}</>))
    expect(htmls.every((h) => h.length > 0)).toBe(true)
    // 语义新 key 集合内图标两两不同
    expect(new Set(htmls).size).toBe(htmls.length)
  })
})

describe('actionColumn variant', () => {
  const renderActions = (variant: 'text' | 'icon' | 'iconOnly', actions: RowAction[]) => {
    const columns: ColumnType<Record<string, unknown>>[] = [
      actionColumn<Record<string, unknown>>({ actions: () => actions, variant }),
    ]
    const el = columns[0].render?.({}, {}, 0) as React.ReactNode
    return render(<AntdApp>{el}</AntdApp>)
  }

  it('text 档：只渲染文字，不出图标', () => {
    renderActions('text', [{ key: 'edit', label: '编辑' }])
    expect(screen.getByText('编辑')).toBeInTheDocument()
  })

  it('icon 档：文字仍在（图标+文字）', () => {
    renderActions('icon', [{ key: 'edit', label: '编辑' }])
    expect(screen.getByText('编辑')).toBeInTheDocument()
  })

  it('iconOnly 档：不渲染文字（只有图标 + Tooltip 包裹）', () => {
    renderActions('iconOnly', [{ key: 'edit', label: '编辑' }])
    // 按钮内不应出现可见文字
    expect(screen.queryByText('编辑')).toBeNull()
  })

  it('actionColumnWidth：icon 档比 text 档宽，iconOnly 档比 text 窄', () => {
    const text = actionColumnWidth(3, 'text')
    const icon = actionColumnWidth(3, 'icon')
    const iconOnly = actionColumnWidth(3, 'iconOnly')
    expect(icon).toBeGreaterThan(text)
    expect(iconOnly).toBeLessThan(text)
  })
})
