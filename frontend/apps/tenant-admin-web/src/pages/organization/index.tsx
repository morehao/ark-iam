import { useCallback, useEffect, useState } from 'react'
import { Table, Button, Modal, Form, Input, Switch, message, Popconfirm, Space } from 'antd'
import { PlusOutlined, ReloadOutlined } from '@ant-design/icons'
import type { ColumnsType } from 'antd/es/table'
import { PageContainer } from '@ark-iam/ui'
import { createOrganization, deleteOrganization, getOrganizationPage, updateOrganization } from '../../api/organization'
import type { OrganizationItem } from '@ark-iam/types'

export default function OrganizationList() {
  const [list, setList] = useState<OrganizationItem[]>([])
  const [loading, setLoading] = useState(false)
  const [modalOpen, setModalOpen] = useState(false)
  const [editing, setEditing] = useState<OrganizationItem | null>(null)
  const [submitLoading, setSubmitLoading] = useState(false)
  const [form] = Form.useForm()

  const load = useCallback(async () => {
    setLoading(true)
    try {
      const resp = await getOrganizationPage({ page: 1, pageSize: 50 })
      setList(resp.list || [])
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    void load()
  }, [load])

  const submit = async () => {
    try {
      const values = await form.validateFields()
      setSubmitLoading(true)
      if (editing) {
        await updateOrganization({ ...values, organizationID: editing.organizationID })
      } else {
        await createOrganization(values)
      }
      message.success('保存成功')
      setModalOpen(false)
      setEditing(null)
      form.resetFields()
      void load()
    } catch {
      /* 校验或请求失败 */
    } finally {
      setSubmitLoading(false)
    }
  }

  const remove = async (id: string) => {
    await deleteOrganization(id)
    message.success('删除成功')
    void load()
  }

  const columns: ColumnsType<OrganizationItem> = [
    { title: '组织ID', dataIndex: 'organizationID', key: 'organizationID', width: 100 },
    { title: '组织名称', dataIndex: 'name', key: 'name' },
    { title: '描述', dataIndex: 'description', key: 'description', ellipsis: true },
    {
      title: '需要 MFA',
      dataIndex: 'isMFARequired',
      key: 'isMFARequired',
      width: 110,
      render: (v: number) => (v === 1 ? <span style={{ color: '#f59e0b' }}>是</span> : <span style={{ color: '#94a3b8' }}>否</span>),
    },
    {
      title: '操作',
      key: 'action',
      width: 160,
      render: (_: unknown, record: OrganizationItem) => (
        <Space size={4}>
          <Button
            type="link"
            size="small"
            onClick={() => {
              setEditing(record)
              form.setFieldsValue(record)
              setModalOpen(true)
            }}
          >
            编辑
          </Button>
          <Popconfirm title="确认删除？" onConfirm={() => void remove(record.organizationID)}>
            <Button type="link" size="small" danger>
              删除
            </Button>
          </Popconfirm>
        </Space>
      ),
    },
  ]

  return (
    <PageContainer
      title="组织管理"
      description="管理当前租户下的组织结构"
      extra={
        <Space>
          <Button icon={<ReloadOutlined />} onClick={() => void load()}>
            刷新
          </Button>
          <Button
            type="primary"
            icon={<PlusOutlined />}
            onClick={() => {
              setEditing(null)
              form.resetFields()
              setModalOpen(true)
            }}
          >
            新建组织
          </Button>
        </Space>
      }
    >
      <Table<OrganizationItem> rowKey="organizationID" loading={loading} columns={columns} dataSource={list} pagination={false} />
      <Modal
        title={editing ? '编辑组织' : '新建组织'}
        open={modalOpen}
        onCancel={() => setModalOpen(false)}
        onOk={() => void submit()}
        confirmLoading={submitLoading}
        destroyOnClose
      >
        <Form form={form} layout="vertical">
          <Form.Item name="name" label="组织名称" rules={[{ required: true, message: '请输入组织名称' }]}>
            <Input placeholder="如：产品研发部" />
          </Form.Item>
          <Form.Item name="description" label="描述">
            <Input.TextArea rows={3} />
          </Form.Item>
          <Form.Item name="isMFARequired" label="需要 MFA" valuePropName="checked" getValueFromEvent={(c: boolean) => (c ? 1 : 0)}>
            <Switch />
          </Form.Item>
        </Form>
      </Modal>
    </PageContainer>
  )
}
