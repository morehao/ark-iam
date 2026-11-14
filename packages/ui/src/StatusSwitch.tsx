import { useState, type ReactNode } from 'react'
import { App, Switch } from 'antd'

/** 状态开关的方向语义：`on` 对应「正常/启用」，`off` 对应「挂起/停用」。 */
export interface StatusSwitchProps {
  /** 当前是否处于「正常/启用」态（on） */
  checked: boolean
  /** 开态文案（checkedChildren），默认「正常」 */
  checkedChildren?: ReactNode
  /** 关态文案（unCheckedChildren），默认「挂起」 */
  unCheckedChildren?: ReactNode
  /**
   * 切换回调：**在用户于确认框点确定后**触发，参数为目标的 on 态布尔值。
   * 组件本身不提交请求——由调用方在回调里发请求，成功后更新受控 `checked`；
   * 失败则不更新 state，开关自然回退到原值（受控）。
   */
  onConfirmChange: (nextChecked: boolean) => void | Promise<void>
  /** 禁用（如平台内置对象不可改） */
  disabled?: boolean
  /** 确认框标题（可选，缺省用通用文案） */
  confirmTitle?: ReactNode
  /** 确认框正文（按对象定制影响说明，如「挂起后该用户无法登录、会话将失效」） */
  confirmContent?: ReactNode
  /** 确认框确定按钮文案（负向操作默认「确认挂起」） */
  okText?: string
  /** 负向（转为 off，即挂起/停用）时确定按钮是否为危险色，默认 true */
  dangerOnOff?: boolean
}

/**
 * 状态列内联开关（列表页「停用/启用、挂起/恢复」统一交互，见 DESIGN.md §7.3）。
 *
 * 设计要点：
 * - **两个方向都要二次确认**（正向=恢复/启用、负向=挂起/停用，文案各异），确认后才回调；
 * - 受控组件：`checked` 由外部状态决定，只有调用方在请求成功后更新它，开关才会真正翻转；
 *   请求失败时外部不更新，开关回退——天然「失败回滚」；
 * - 提交中显示 loading 并禁点，避免快速连点。
 *
 * 用法（调用方负责发请求 + 更新行状态）：
 * ```tsx
 * <StatusSwitch
 *   checked={r.status === 'active'}
 *   checkedChildren="正常"
 *   unCheckedChildren="挂起"
 *   confirmTitle={next ? '恢复该用户？' : '挂起该用户？'}
 *   confirmContent="挂起后该用户无法登录，已签发的会话会立即失效。"
 *   onConfirmChange={(next) => void toggleSuspended(r, next ? 'active' : 'suspended')}
 * />
 * ```
 */
export function StatusSwitch({
  checked,
  checkedChildren = '正常',
  unCheckedChildren = '挂起',
  onConfirmChange,
  disabled,
  confirmTitle,
  confirmContent,
  okText,
  dangerOnOff = true,
}: StatusSwitchProps) {
  const { modal, message } = App.useApp()
  const [loading, setLoading] = useState(false)

  const handleChange = (next: boolean) => {
    const toOff = !next
    modal.confirm({
      title: confirmTitle ?? (next ? `确认恢复？` : `确认停用？`),
      content: confirmContent,
      okText: okText ?? (next ? '确认' : '确认停用'),
      cancelText: '取消',
      okButtonProps: toOff && dangerOnOff ? { danger: true } : undefined,
      onOk: async () => {
        setLoading(true)
        try {
          await onConfirmChange(next)
          message.success(next ? '已恢复' : '已停用')
        } catch {
          // 请求失败：拦截器已提示；不更新受控 checked，开关回退
        } finally {
          setLoading(false)
        }
      },
    })
  }

  return (
    <Switch
      size="small"
      checked={checked}
      loading={loading}
      disabled={disabled}
      checkedChildren={checkedChildren}
      unCheckedChildren={unCheckedChildren}
      onChange={handleChange}
    />
  )
}
