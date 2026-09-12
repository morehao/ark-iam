import type { CSSProperties } from 'react'
import { Alert, Button, Modal, Space, message } from 'antd'
import { CopyOutlined } from '@ant-design/icons'
import { tokens } from './theme'

const monospaceStyle: CSSProperties = { fontFamily: 'Consolas, Monaco, monospace' }

// TODO(delivery): 本组件是"一次性回显临时密码"的前端出口（系统尚无邮件/短信通道）；
// 接入通道后临时密码改为直接下发给账号本人，本组件应改为"已通过邮件/短信发送"的提示，
// 或整体下线。见 docs/design/system-design.md §5.8。
export interface InitialPasswordModalProps {
  /**
   * 一次性初始/临时密码明文。为空表示未生成（如复用了既有自然人，其密码未被改动），
   * 此时 Modal 不展示。
   */
  password: string
  /** 弹窗标题，默认「初始密码已生成」。 */
  title?: string
  /** 密码上方的补充说明（如「该成员首次登录必须修改密码」）。 */
  description?: string
  onClose: () => void
}

/**
 * 一次性初始密码展示 Modal：密码由服务端生成、仅在本次响应返回一次，关闭后不可再查。
 * 平台侧（建租户/重置内置管理员密码）与租户侧（建成员/重置成员密码）共用同一交互，
 * 保证两边"密码相关交互一致"（见 docs/design/system-design.md §5.8）。
 */
export function InitialPasswordModal({ password, title, description, onClose }: InitialPasswordModalProps) {
  const handleCopy = async () => {
    try {
      await navigator.clipboard.writeText(password)
      message.success('已复制')
    } catch {
      message.error('复制失败，请手动复制')
    }
  }
  return (
    <Modal
      title={title || '初始密码已生成'}
      open={!!password}
      onCancel={onClose}
      footer={
        <Button type="primary" onClick={onClose}>
          我已保存
        </Button>
      }
      destroyOnClose
      width={560}
    >
      <Alert
        type="warning"
        showIcon
        message="请立即保存，关闭后不再显示"
        description={
          <Space direction="vertical" size={12} style={{ width: '100%' }}>
            {description && <div style={{ color: tokens.textSecondary }}>{description}</div>}
            <div
              style={{
                ...monospaceStyle,
                wordBreak: 'break-all',
                background: tokens.warningBg,
                border: `1px solid ${tokens.warningBorder}`,
                borderRadius: 8,
                padding: '10px 12px',
              }}
            >
              {password}
            </div>
            <Button size="small" icon={<CopyOutlined />} onClick={() => void handleCopy()}>
              复制
            </Button>
          </Space>
        }
      />
    </Modal>
  )
}
