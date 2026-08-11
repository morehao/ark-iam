import { useEffect, useState } from 'react'
import { Card, Table, Button, Modal, Form, InputNumber, message, Popconfirm } from 'antd'
import type { ColumnsType } from 'antd/es/table'
import { getOrganizationRoleUserPage, createOrganizationRoleUser, deleteOrganizationRoleUser } from '../../api/organization'
import type { OrganizationRoleUserItem } from '@ark-iam/types'

export default function OrganizationRoleUserList() {
  const [list, setList] = useState<OrganizationRoleUserItem[]>([])
  const [loading, setLoading] = useState(false)
  const [modalOpen, setModalOpen] = useState(false)
  const [form] = Form.useForm()

  const load = async () => {
    setLoading(true)
    try {
      const resp = await getOrganizationRoleUserPage({ page: 1, pageSize: 50 })
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
    await createOrganizationRoleUser(values)
    message.success('创建成功')
    setModalOpen(false)
    form.resetFields()
    void load()
  }

  const remove = async (record: OrganizationRoleUserItem) => {
    await deleteOrganizationRoleUser({
      organizationID: record.organizationID,
      organizationRoleID: record.organizationRoleID,
      userID: record.userID,
    })
    message.success('删除成功')
    void load()
  }

  const columns: ColumnsType<OrganizationRoleUserItem> = [
    { title: '组织ID', dataIndex: 'organizationID', key: 'organizationID' },
    { title: '组织角色ID', dataIndex: 'organizationRoleID', key: 'organizationRoleID' },
    { title: '用户ID', dataIndex: 'userID', key: 'userID' },
    {
      title: '操作',
      key: 'action',
      render: (_: unknown, record: OrganizationRoleUserItem) => (
        <Popconfirm title="确认删除？" onConfirm={() => void remove(record)}>
          <Button size="small" danger>删除</Button>
        </Popconfirm>
      ),
    },
  ]

  return (
    <Card title="组织角色用户">
      <Button type="primary" style={{ marginBottom: 16 }} onClick={() => { form.resetFields(); setModalOpen(true) }}>
        新建组织角色用户
      </Button>
      <Table<OrganizationRoleUserItem> rowKey={(r) => `${r.organizationID}-${r.organizationRoleID}-${r.userID}`} loading={loading} columns={columns} dataSource={list} pagination={false} />
      <Modal title="新建组织角色用户" open={modalOpen} onCancel={() => setModalOpen(false)} onOk={() => void submit()}>
        <Form form={form} layout="vertical">
          <Form.Item name="organizationID" label="组织ID" rules={[{ required: true, message: '请输入组织ID' }]}>
            <InputNumber style={{ width: '100%' }} />
          </Form.Item>
          <Form.Item name="organizationRoleID" label="组织角色ID" rules={[{ required: true, message: '请输入组织角色ID' }]}>
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
