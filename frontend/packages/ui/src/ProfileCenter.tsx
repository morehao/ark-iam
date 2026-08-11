import { useEffect, useState } from 'react'
import { Modal, Tabs, Descriptions, Form, Input, Button, Table, message, Popconfirm, Space } from 'antd'
import { getPersonDetail, updatePassword, getSessionList, revokeSession, revokeAllSessions } from '@ark-iam/api'
import type { PersonDetailResp, SessionResp } from '@ark-iam/types'

interface Props {
  open: boolean
  onClose: () => void
}

export function ProfileCenter({ open, onClose }: Props) {
  const [person, setPerson] = useState<PersonDetailResp | null>(null)
  const [sessions, setSessions] = useState<SessionResp[]>([])
  const [form] = Form.useForm()

  const loadPerson = async () => {
    try {
      setPerson(await getPersonDetail())
    } catch {
      message.error('获取个人信息失败')
    }
  }

  const loadSessions = async () => {
    try {
      const resp = await getSessionList()
      setSessions(resp.list || [])
    } catch {
      message.error('获取会话列表失败')
    }
  }

  useEffect(() => {
    if (open) {
      void loadPerson()
      void loadSessions()
    }
  }, [open])

  const submitPassword = async () => {
    try {
      const values = await form.validateFields()
      await updatePassword(values)
      message.success('密码已修改')
      form.resetFields()
    } catch {
      // 校验或请求失败，错误已由 request 拦截器提示
    }
  }

  const columns = [
    { title: '会话ID', dataIndex: 'sessionId', key: 'sessionId' },
    { title: '客户端', dataIndex: 'clientType', key: 'clientType' },
    { title: 'IP', dataIndex: 'clientIP', key: 'clientIP' },
    { title: '创建时间', dataIndex: 'createdAt', key: 'createdAt' },
    {
      title: '操作',
      key: 'action',
      render: (_: unknown, record: SessionResp) => (
        <Popconfirm
          title="确认撤销该会话？"
          onConfirm={() =>
            void revokeSession(record.id)
              .then(() => loadSessions())
              .catch(() => message.error('撤销会话失败'))
          }
        >
          <Button size="small" danger>
            撤销
          </Button>
        </Popconfirm>
      ),
    },
  ]

  return (
    <Modal title="个人中心" open={open} onCancel={onClose} footer={null} width={640}>
      <Tabs
        items={[
          {
            key: 'info',
            label: '个人信息',
            children: (
              <Descriptions column={1} bordered>
                <Descriptions.Item label="用户名">{person?.name ?? '-'}</Descriptions.Item>
                <Descriptions.Item label="账号">{person?.username ?? '-'}</Descriptions.Item>
                <Descriptions.Item label="邮箱">{person?.primaryEmail ?? '-'}</Descriptions.Item>
                <Descriptions.Item label="手机号">{person?.primaryPhone ?? '-'}</Descriptions.Item>
              </Descriptions>
            ),
          },
          {
            key: 'password',
            label: '修改密码',
            children: (
              <Form form={form} layout="vertical">
                <Form.Item name="oldPassword" label="原密码" rules={[{ required: true, message: '请输入原密码' }]}>
                  <Input.Password />
                </Form.Item>
                <Form.Item name="newPassword" label="新密码" rules={[{ required: true, message: '请输入新密码' }, { min: 6, message: '至少 6 位' }]}>
                  <Input.Password />
                </Form.Item>
                <Button type="primary" onClick={() => void submitPassword()}>
                  确认修改
                </Button>
              </Form>
            ),
          },
          {
            key: 'session',
            label: '会话管理',
            children: (
              <>
                <Space style={{ marginBottom: 12 }}>
                  <Button
                    danger
                    onClick={() =>
                      void revokeAllSessions()
                        .then(() => loadSessions())
                        .catch(() => message.error('撤销全部会话失败'))
                    }
                  >
                    撤销全部会话
                  </Button>
                </Space>
                <Table<SessionResp> rowKey="id" columns={columns} dataSource={sessions} pagination={false} />
              </>
            ),
          },
        ]}
      />
    </Modal>
  )
}
