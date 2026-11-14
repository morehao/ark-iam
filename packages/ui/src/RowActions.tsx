import type { ReactNode } from 'react'
import { App, Button, Dropdown, Popconfirm, Space, Tooltip } from 'antd'
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
  /** 可选图标（`icon`/`iconOnly` 档按钮与「更多」下拉菜单项会展示） */
  icon?: ReactNode
  /** 运行时隐藏该操作（如已吊销的密钥不再展示「吊销」） */
  hidden?: boolean
}

/**
 * 操作列按钮呈现样式：
 * - `text`：纯文字链接（历史形态，`Button type="link"`）；
 * - `icon`：图标 + 文字（`Button type="link"` 带 icon，图标在左，扫视更快）；
 * - `iconOnly`：纯图标（`Button type="text"`，Tooltip 悬浮显示动作名），操作列最紧凑。
 */
export type RowActionsVariant = 'text' | 'icon' | 'iconOnly'

export interface RowActionsProps {
  /** 操作列表，按重要程度从高到低排列 */
  actions: RowAction[]
  /** 横排上限，超过则收起为「更多」下拉；默认 3 */
  max?: number
  /** 呈现样式，默认 `text`（纯文字，向后兼容）；需要图标时显式传 `icon`/`iconOnly` */
  variant?: RowActionsVariant
}

/**
 * 统一的表格「操作列」渲染器（列表页操作列规范，见 DESIGN.md §7.3）：
 * - 操作数 ≤ max（默认 3）：全部横排（`variant` 决定文字/图标+文字/纯图标）；
 * - 操作数 > max：保留前 `max - 1` 个高频操作横排，其余收进「更多」下拉（菜单项纵向排列），
 *   避免操作列被撑宽、行内按钮挤成一团。
 *
 * 二次确认：横排操作用 Popconfirm 就地气泡确认；下拉菜单项用 Modal.confirm
 * （菜单点击会先关闭下拉，气泡无法稳定锚定）。
 */
export function RowActions({ actions, max = 3, variant = 'text' }: RowActionsProps) {
  const { modal } = App.useApp()
  const visible = actions.filter((action) => !action.hidden)
  if (visible.length === 0) return null

  const iconOnly = variant === 'iconOnly'
  const showIcon = variant !== 'text'
  // iconOnly 用紧凑的 text 按钮（无边框无底色），其余用链接样式
  const buttonType = iconOnly ? 'text' : 'link'

  const renderButton = (action: RowAction) => (
    <Button
      type={buttonType}
      size="small"
      danger={action.danger}
      disabled={action.disabled}
      icon={showIcon ? action.icon : undefined}
      onClick={action.onClick}
    >
      {iconOnly ? undefined : action.label}
    </Button>
  )

  const wrap = (action: RowAction, node: ReactNode) => {
    if (!iconOnly) return node
    // 纯图标：hover 显示动作名与禁用原因，避免无字按钮语义丢失
    const tip = typeof action.label === 'string' ? action.label : undefined
    return (
      <Tooltip key={action.key} title={tip} mouseEnterDelay={0.3}>
        {node as React.ReactElement}
      </Tooltip>
    )
  }

  const renderInline = (action: RowAction) => {
    if (!action.confirm) {
      const btn = renderButton(action)
      return iconOnly ? wrap(action, btn) : <span key={action.key}>{btn}</span>
    }
    return (
      <Popconfirm
        key={action.key}
        title={action.confirm}
        disabled={action.disabled}
        onConfirm={action.onClick}
      >
        <Button
          type={buttonType}
          size="small"
          danger={action.danger}
          disabled={action.disabled}
          icon={showIcon ? action.icon : undefined}
        >
          {iconOnly ? undefined : action.label}
        </Button>
      </Popconfirm>
    )
  }

  if (visible.length <= max) {
    return <Space size={iconOnly ? 0 : 4}>{visible.map(renderInline)}</Space>
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
    <Space size={iconOnly ? 0 : 4}>
      {inline.map(renderInline)}
      <Dropdown menu={{ items, onClick: handleMenuClick }} trigger={['click']}>
        <Button type={buttonType} size="small">
          {iconOnly ? <DownOutlined /> : <>更多 <DownOutlined style={{ fontSize: 10 }} /></>}
        </Button>
      </Dropdown>
    </Space>
  )
}
