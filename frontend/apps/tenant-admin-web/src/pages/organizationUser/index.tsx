import { useEffect, useState } from 'react'
import { Card, Table, Button, Modal, Form, InputNumber, message, Popconfirm } from 'antd'
import type { ColumnsType } from 'antd/es/table'
import { getOrganizationUserPage, createOrganizationUser, deleteOrganizationUser } from '../../api/organization'
import type { OrganizationUserItem } from '@ark-iam/types'

export default function OrganizationUserList() {
  const [list, setList] = useState<OrganizationUserItem[]>([])
  const [loading, setLoading] = useState(false)
  const [modalOpen, setModalOpen] = useState(false)
  const [form] = Form.useForm()

  const load = async () => {
    setLoading(true)
    try {
      const resp = await getOrganizationUserPage({ page: 1, pageSize: 50 })
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
    await createOrganizationUser(values)
    message.success('创建成功')
    setModalOpen(false)
    form.resetFields()
    void load()
  }

  const remove = async (record: OrganizationUserItem) => {
    await deleteOrganizationUser({ organizationID: record.organizationID, userID: record.userID })
    message.success('删除成功')
    void load()
  }

  const columns: ColumnsType<OrganizationUserItem> = [
    { title: '组织ID', dataIndex: 'organizationID', key: 'organizationID' },
    { title: '用户ID', dataIndex: 'userID', key: 'userID' },
    {
      title: '操作',
      key: 'action',
      render: (_: unknown, record: OrganizationUserItem) => (
        <Popconfirm title="确认删除？" onConfirm={() => void remove(record)}>
          <Button size="small" danger>删除</Button>
        </Popconfirm>
      ),
    },
  ]

  return (
    <Card title="组织用户">
      <Button type="primary" style={{ marginBottom: 16 }} onClick={() => { form.resetFields(); setModalOpen(true) }}>
        新建组织用户
      </Button>
      <Table<OrganizationUserItem> rowKey={(r) => `${r.organizationID}-${r.userID}`} loading={loading} columns={columns} dataSource={list} pagination={false} />
      <Modal title="新建组织用户" open={modalOpen} onCancel={() => setModalOpen(false)} onOk={() => void submit()}>
        <Form form={form} layout="vertical">
          <Form.Item name="organizationID" label="组织ID" rules={[{ required: true, message: '请输入组织ID' }]}>
            <InputNumber style={{ width: '100%' }} />
          </Form.Item>
          <Form.Item name="userID" label="用户ID" rules={[{ required: true, message: '请输入用户ID' }]}>
            <InputNumber style={{ width: '100%' }} />
          </Form.Item>
        </Form>
      </Modal>
    </Card>
  )
}
