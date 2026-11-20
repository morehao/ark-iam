import { useEffect, useState } from 'react'
import { Modal, Space, Spin, Typography } from 'antd'
import { CheckCircleTwoTone, CloseCircleTwoTone } from '@ant-design/icons'
import { tokens } from '@ark-iam/ui'
import { testConnector } from '@ark-iam/api'
import type { ConnectorItem } from '@ark-iam/types'

interface Props {
  open: boolean
  connector: ConnectorItem | null
  onClose: () => void
}

type Phase = 'running' | 'ok' | 'fail'

/**
 * 连通性测试（后端 testConnector 只返回 { success, message } 一态）。
 * 前端用固定步骤展示「进行中」，最终以返回结论渲染成功/失败与后端 message。
 */
export default function ConnectorTestModal({ open, connector, onClose }: Props) {
  const [phase, setPhase] = useState<Phase>('running')
  const [msg, setMsg] = useState('')

  useEffect(() => {
    if (!open || !connector) return
    setPhase('running')
    setMsg('')
    let alive = true
    testConnector(connector.connectorID)
      .then((resp) => {
        if (!alive) return
        if (resp?.success) {
          setPhase('ok')
          setMsg(resp.message || '')
        } else {
          setPhase('fail')
          setMsg(resp?.message || '连通性测试失败')
        }
      })
      .catch(() => {
        if (!alive) return
        // 请求失败（含 4xx/5xx）也归为测试失败；拦截器可能已提示
        setPhase('fail')
        setMsg('测试请求失败，请检查配置或稍后重试')
      })
    return () => {
      alive = false
    }
  }, [open, connector])

  if (!connector) return null

  return (
    <Modal
      title={`连通性测试 · ${connector.displayName}`}
      open={open}
      onCancel={onClose}
      footer={null}
      width={520}
      destroyOnHidden
    >
      {phase === 'running' && (
        <Space>
          <Spin size="small" />
          <Typography.Text type="secondary">正在探测连接器配置与端点…</Typography.Text>
        </Space>
      )}
      {phase === 'ok' && (
        <Space align="start">
          <CheckCircleTwoTone twoToneColor={tokens.success} style={{ fontSize: 18 }} />
          <div>
            <Typography.Text strong style={{ color: tokens.success }}>
              连接正常
            </Typography.Text>
            {msg && <div style={{ color: tokens.textSecondary, fontSize: 12 }}>{msg}</div>}
          </div>
        </Space>
      )}
      {phase === 'fail' && (
        <Space align="start">
          <CloseCircleTwoTone twoToneColor={tokens.error} style={{ fontSize: 18 }} />
          <div>
            <Typography.Text strong style={{ color: tokens.error }}>
              测试失败
            </Typography.Text>
            <div style={{ color: tokens.textSecondary, fontSize: 12 }}>{msg}</div>
          </div>
        </Space>
      )}
    </Modal>
  )
}
