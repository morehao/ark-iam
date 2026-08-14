import { useEffect, useState } from 'react'
import { Card, Table, Button, Modal, Form, Input, Switch, message, Popconfirm } from 'antd'
import type { ColumnsType } from 'antd/es/table'
import { getOrganizationPage, createOrganization, updateOrganization, deleteOrganization } from '../../api/organization'
import type { OrganizationItem } from '@ark-iam/types'

export default function OrganizationList() {
  const [list, setList] = useState<OrganizationItem[]>([])
  const [loading, setLoading] = useState(false)
  const [modalOpen, setModalOpen] = useState(false)
  const [editing, setEditing] = useState<OrganizationItem | null>(null)
  const [form] = Form.useForm()

  const load = async () => {
    setLoading(true)
    try {
      const resp = await getOrganizationPage({ page: 1, pageSize: 50 })
      setList(resp.list || [])
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    void load()
  }, [])

  const submit = async () => {
    const values = await form.validateFields()
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
  }

  const remove = async (id: number) => {
    await deleteOrganization(id)
    message.success('删除成功')
    void load()
  }

  const columns: ColumnsType<OrganizationItem> = [
    { title: '组织ID', dataIndex: 'organizationID', key: 'organizationID' },
    { title: '组织名称', dataIndex: 'name', key: 'name' },
    { title: '描述', dataIndex: 'description', key: 'description' },
    {
      title: '操作',
      key: 'action',
      render: (_: unknown, record: OrganizationItem) => (
        <>
          <Button size="small" onClick={() => { setEditing(record); form.setFieldsValue(record); setModalOpen(true) }}>编辑</Button>
          <Popconfirm title="确认删除？" onConfirm={() => void remove(record.organizationID)}>
            <Button size="small" danger style={{ marginLeft: 8 }}>删除</Button>
          </Popconfirm>
        </>
      ),
    },
  ]

  return (
    <Card title="组织管理">
      <Button type="primary" style={{ marginBottom: 16 }} onClick={() => { setEditing(null); form.resetFields(); setModalOpen(true) }}>
        新建组织
      </Button>
      <Table<OrganizationItem> rowKey="organizationID" loading={loading} columns={columns} dataSource={list} pagination={false} />
      <Modal title={editing ? '编辑组织' : '新建组织'} open={modalOpen} onCancel={() => setModalOpen(false)} onOk={() => void submit()}>
        <Form form={form} layout="vertical">
          <Form.Item name="name" label="组织名称" rules={[{ required: true, message: '请输入组织名称' }]}>
            <Input />
          </Form.Item>
          <Form.Item name="description" label="描述">
            <Input.TextArea />
          </Form.Item>
          <Form.Item name="isMFARequired" label="需要 MFA" valuePropName="checked" getValueFromEvent={checked => (checked ? 1 : 0)}>
            <Switch />
          </Form.Item>
        </Form>
      </Modal>
    </Card>
  )
}
