import type { ReactNode } from 'react'
import { App, Button, Dropdown, Popconfirm, Space } from 'antd'
import type { MenuProps } from 'antd'
import { DownOutlined } from '@ant-design/icons'

/** 表格操作列中的单个操作。 */
export interface RowAction {
  /** 操作唯一标识（React key / 菜单 key） */
  key: string
  /** 操作文案 */
  label: ReactNode
  /** 点击回调 */
  onClick?: () => void
  /** 危险操作：按钮/菜单项渲染为语义红 */
  danger?: boolean
  /** 禁用 */
  disabled?: boolean
  /** 二次确认文案；提供时点击后先确认再执行 onClick */
  confirm?: ReactNode
  /** 可选图标（下拉菜单项会展示） */
  icon?: ReactNode
  /** 运行时隐藏该操作（如已吊销的密钥不再展示「吊销」） */
  hidden?: boolean
}

export interface RowActionsProps {
  /** 操作列表，按重要程度从高到低排列 */
  actions: RowAction[]
  /** 横排上限，超过则收起为「更多」下拉；默认 3 */
  max?: number
}

/**
 * 统一的表格「操作列」渲染器（列表页操作列规范，见 DESIGN.md §7.3）：
 * - 操作数 ≤ max（默认 3）：全部以 `Button type="link" size="small"` 横排；
 * - 操作数 > max：保留前 `max - 1` 个高频操作横排，其余收进「更多」下拉（菜单项纵向排列），
 *   避免操作列被撑宽、行内按钮挤成一团。
 *
 * 二次确认：横排操作用 Popconfirm 就地气泡确认；下拉菜单项用 Modal.confirm
 * （菜单点击会先关闭下拉，气泡无法稳定锚定）。
 */
export function RowActions({ actions, max = 3 }: RowActionsProps) {
  const { modal } = App.useApp()
  const visible = actions.filter((action) => !action.hidden)
  if (visible.length === 0) return null

  const renderInline = (action: RowAction) => {
    if (!action.confirm) {
      return (
        <Button
          key={action.key}
          type="link"
          size="small"
          danger={action.danger}
          disabled={action.disabled}
          icon={action.icon}
          onClick={action.onClick}
        >
          {action.label}
        </Button>
      )
    }
    return (
      <Popconfirm
        key={action.key}
        title={action.confirm}
        disabled={action.disabled}
        onConfirm={action.onClick}
      >
        <Button type="link" size="small" danger={action.danger} disabled={action.disabled} icon={action.icon}>
          {action.label}
        </Button>
      </Popconfirm>
    )
  }

  if (visible.length <= max) {
    return <Space size={4}>{visible.map(renderInline)}</Space>
  }

  const inline = visible.slice(0, Math.max(1, max - 1))
  const overflow = visible.slice(Math.max(1, max - 1))

  const items: MenuProps['items'] = overflow.map((action) => ({
    key: action.key,
    label: action.label,
    icon: action.icon,
    danger: action.danger,
    disabled: action.disabled,
  }))

  const handleMenuClick: MenuProps['onClick'] = ({ key }) => {
    const action = overflow.find((item) => item.key === key)
    if (!action) return
    if (action.confirm) {
      modal.confirm({
        title: action.confirm,
        okButtonProps: { danger: action.danger },
        onOk: () => action.onClick?.(),
      })
      return
    }
    action.onClick?.()
  }

  return (
    <Space size={4}>
      {inline.map(renderInline)}
      <Dropdown menu={{ items, onClick: handleMenuClick }} trigger={['click']}>
        <Button type="link" size="small">
          更多 <DownOutlined style={{ fontSize: 10 }} />
        </Button>
      </Dropdown>
    </Space>
  )
}
