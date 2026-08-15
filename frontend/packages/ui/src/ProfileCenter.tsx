import { useCallback, useEffect, useState } from 'react'
import { Modal, Tabs, Descriptions, Form, Input, Button, Table, message, Popconfirm, Space, Avatar, Tag, Tooltip } from 'antd'
import { getPersonDetail, updatePassword, getSessionList, revokeSession, revokeAllSessions } from '@ark-iam/api'
import type { PersonDetailResp, SessionResp } from '@ark-iam/types'
import { brand } from './theme'

interface Props {
  open: boolean
  onClose: () => void
}

export function ProfileCenter({ open, onClose }: Props) {
  const [person, setPerson] = useState<PersonDetailResp | null>(null)
  const [sessions, setSessions] = useState<SessionResp[]>([])
  const [loading, setLoading] = useState(false)
  const [form] = Form.useForm()

  const loadPerson = useCallback(async () => {
    try {
      setPerson(await getPersonDetail())
    } catch {
      /* 拦截器已提示 */
    }
  }, [])

  // 会话接口默认 10 条/页，这里拉取前 5 页以便展示更多会话
  const loadSessions = useCallback(async () => {
    setLoading(true)
    try {
      const pages = await Promise.all(
        [1, 2, 3, 4, 5].map((p) => getSessionList({ page: p, pageSize: 10 }).catch(() => ({ list: [], total: 0 }))),
      )
      const seen = new Set<number>()
      const merged: SessionResp[] = []
      for (const resp of pages) {
        for (const s of resp.list || []) {
          if (!seen.has(s.id)) {
            seen.add(s.id)
            merged.push(s)
          }
        }
      }
      setSessions(merged)
    } catch {
      /* 拦截器已提示 */
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    if (open) {
      void loadPerson()
      void loadSessions()
    }
  }, [open, loadPerson, loadSessions])

  const submitPassword = async () => {
    try {
      const values = await form.validateFields()
      if (values.newPassword !== values.confirmPassword) {
        message.warning('两次输入的新密码不一致')
        return
      }
      await updatePassword({ oldPassword: values.oldPassword, newPassword: values.newPassword })
      message.success('密码已修改')
      form.resetFields()
    } catch {
      // 校验或请求失败，错误已由拦截器提示
    }
  }

  const columns = [
    { title: '会话ID', dataIndex: 'sessionID', key: 'sessionID', ellipsis: true },
    { title: '客户端', dataIndex: 'clientType', key: 'clientType', width: 110 },
    { title: 'IP', dataIndex: 'clientIP', key: 'clientIP', width: 130 },
    {
      title: '状态',
      dataIndex: 'isActive',
      key: 'isActive',
      width: 90,
      render: (v: boolean) => (v ? <Tag color="success">活跃</Tag> : <Tag color="default">已失效</Tag>),
    },
    {
      title: 'UserAgent',
      dataIndex: 'userAgent',
      key: 'userAgent',
      ellipsis: true,
      render: (v: string) => <Tooltip title={v}>{v || '-'}</Tooltip>,
    },
    { title: '创建时间', dataIndex: 'createdAt', key: 'createdAt', width: 160 },
    {
      title: '操作',
      key: 'action',
      width: 90,
      render: (_: unknown, record: SessionResp) => (
        <Popconfirm
          title="确认撤销该会话？"
          onConfirm={() =>
            void revokeSession(record.id)
              .then(() => {
                message.success('已撤销')
                void loadSessions()
              })
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
    <Modal title="个人中心" open={open} onCancel={onClose} footer={null} width={760} styles={{ body: { paddingTop: 12 } }}>
      <Tabs
        items={[
          {
            key: 'info',
            label: '个人信息',
            children: (
              <div>
                <Space size={20} align="center" style={{ marginBottom: 20 }}>
                  <Avatar size={56} style={{ background: brand.gradient, fontSize: 22 }}>
                    {person?.name?.charAt(0)?.toUpperCase() || 'U'}
                  </Avatar>
                  <div>
                    <div style={{ fontSize: 18, fontWeight: 700 }}>
                      {person?.name || '-'}
                      {person?.isSuspended === 1 && <Tag color="error" style={{ marginLeft: 8 }}>已挂起</Tag>}
                    </div>
                    <div style={{ color: brand.textSecondary }}>@{person?.username || '-'}</div>
                  </div>
                </Space>
                <Descriptions column={1} bordered size="small">
                  <Descriptions.Item label="账号">{person?.username ?? '-'}</Descriptions.Item>
                  <Descriptions.Item label="姓名">{person?.name ?? '-'}</Descriptions.Item>
                  <Descriptions.Item label="邮箱">{person?.primaryEmail ?? '-'}</Descriptions.Item>
                  <Descriptions.Item label="手机号">{person?.primaryPhone ?? '-'}</Descriptions.Item>
                </Descriptions>
              </div>
            ),
          },
          {
            key: 'password',
            label: '修改密码',
            children: (
              <Form form={form} layout="vertical" style={{ maxWidth: 420 }}>
                <Form.Item name="oldPassword" label="原密码" rules={[{ required: true, message: '请输入原密码' }]}>
                  <Input.Password />
                </Form.Item>
                <Form.Item
                  name="newPassword"
                  label="新密码"
                  rules={[
                    { required: true, message: '请输入新密码' },
                    { min: 8, message: '至少 8 位，需含大小写字母和数字' },
                  ]}
                >
                  <Input.Password />
                </Form.Item>
                <Form.Item
                  name="confirmPassword"
                  label="确认新密码"
                  rules={[{ required: true, message: '请再次输入新密码' }]}
                >
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
                        .then(() => {
                          message.success('已撤销全部会话')
                          void loadSessions()
                        })
                        .catch(() => message.error('撤销全部会话失败'))
                    }
                  >
                    撤销全部会话
                  </Button>
                  <span style={{ color: brand.textSecondary, fontSize: 12 }}>共 {sessions.length} 条（含历史失效会话）</span>
                </Space>
                <Table<SessionResp>
                  rowKey="id"
                  columns={columns}
                  dataSource={sessions}
                  loading={loading}
                  pagination={{ pageSize: 10, showSizeChanger: false, showTotal: (t) => `共 ${t} 条` }}
                  scroll={{ x: 800 }}
                />
              </>
            ),
          },
        ]}
      />
    </Modal>
  )
}
