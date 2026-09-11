import { useCallback, useEffect, useState } from 'react'
import { Button, Form, Input, Modal, Popconfirm, Space, Table, message } from 'antd'
import { PlusOutlined, ReloadOutlined } from '@ant-design/icons'
import type { ColumnsType } from 'antd/es/table'
import { IDCell, timeColumn, tokens } from '@ark-iam/ui'
import type { TenantUserIdentityItem } from '@ark-iam/types'
import { createTenantUserIdentity, deleteTenantUserIdentity, getTenantUserIdentities } from '../api/user'

export interface UserIdentityTabProps {
  userID: string
}

/**
 * 用户第三方身份：列表 + 绑定 + 解绑。
 * 身份实体按自然人归属，租户范围由后端从登录上下文判定，前端不传 tenantID。
 */
export default function UserIdentityTab({ userID }: UserIdentityTabProps) {
  const [data, setData] = useState<TenantUserIdentityItem[]>([])
  const [loading, setLoading] = useState(true)
  const [modalOpen, setModalOpen] = useState(false)
  const [saving, setSaving] = useState(false)
  const [form] = Form.useForm()

  const fetchData = useCallback(async () => {
    setLoading(true)
    try {
      const resp = await getTenantUserIdentities(userID)
      setData(resp?.list || [])
    } catch {
      /* 拦截器已提示 */
    } finally {
      setLoading(false)
    }
  }, [userID])

  useEffect(() => {
    void fetchData()
  }, [fetchData])

  const submit = async () => {
    let values: { issuer: string; identityID: string }
    try {
      values = await form.validateFields()
    } catch {
      return // 表单校验未通过，提示由 Form 展示
    }
    setSaving(true)
    try {
      await createTenantUserIdentity({ userID, issuer: values.issuer, identityID: values.identityID })
      message.success('绑定成功')
      setModalOpen(false)
      form.resetFields()
      void fetchData()
    } catch {
      /* 拦截器已提示 */
    } finally {
      setSaving(false)
    }
  }

  const unbind = async (userIdentityID: string) => {
    try {
      await deleteTenantUserIdentity(userID, userIdentityID)
      message.success('解绑成功')
      void fetchData()
    } catch {
      /* 拦截器已提示 */
    }
  }

  const columns: ColumnsType<TenantUserIdentityItem> = [
    { title: 'ID', dataIndex: 'userIdentityID', key: 'userIdentityID', width: 140, render: (v: string) => <IDCell value={v} /> },
    { title: '身份提供商', dataIndex: 'issuer', key: 'issuer', render: (v: string) => v || '-' },
    { title: '第三方用户ID', dataIndex: 'identityID', key: 'identityID', render: (v: string) => v || '-' },
    timeColumn<TenantUserIdentityItem>({ title: '创建时间', dataIndex: 'createdAt' }),
    {
      title: '操作',
      key: 'action',
      width: 80,
      render: (_, r) => (
        <Popconfirm title="确认解绑该身份？" onConfirm={() => void unbind(r.userIdentityID)}>
          <Button type="link" size="small" danger>
            解绑
          </Button>
        </Popconfirm>
      ),
    },
  ]

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 12 }}>
      <Space>
        <Button type="primary" icon={<PlusOutlined />} onClick={() => setModalOpen(true)}>
          绑定身份
        </Button>
        <Button icon={<ReloadOutlined />} onClick={() => void fetchData()}>
          刷新
        </Button>
      </Space>

      <Table<TenantUserIdentityItem>
        rowKey="userIdentityID"
        loading={loading}
        columns={columns}
        dataSource={data}
        pagination={false}
        scroll={{ x: 660 }}
      />

      <div style={{ color: tokens.textPlaceholder, fontSize: 12 }}>
        同一第三方账号（身份提供商 + 第三方用户ID）在系统内只能绑定到一个自然人。
      </div>

      <Modal
        title="绑定第三方身份"
        open={modalOpen}
        confirmLoading={saving}
        onOk={() => void submit()}
        onCancel={() => setModalOpen(false)}
        destroyOnClose
      >
        <Form form={form} layout="vertical">
          <Form.Item name="issuer" label="身份提供商" rules={[{ required: true, message: '请输入身份提供商' }]}>
            <Input placeholder="如 https://accounts.google.com" />
          </Form.Item>
          <Form.Item name="identityID" label="第三方用户ID" rules={[{ required: true, message: '请输入第三方用户ID' }]}>
            <Input />
          </Form.Item>
        </Form>
      </Modal>
    </div>
  )
}
