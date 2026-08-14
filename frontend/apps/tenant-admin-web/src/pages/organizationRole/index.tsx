import { useEffect, useState } from 'react'
import { Card, Table, Button, Modal, Form, Input, message, Popconfirm, Select } from 'antd'
import type { ColumnsType } from 'antd/es/table'
import { getOrganizationRolePage, createOrganizationRole, updateOrganizationRole, deleteOrganizationRole } from '../../api/organization'
import type { OrganizationRoleItem } from '@ark-iam/types'

export default function OrganizationRoleList() {
  const [list, setList] = useState<OrganizationRoleItem[]>([])
  const [loading, setLoading] = useState(false)
  const [modalOpen, setModalOpen] = useState(false)
  const [editing, setEditing] = useState<OrganizationRoleItem | null>(null)
  const [form] = Form.useForm()

  const load = async () => {
    setLoading(true)
    try {
      const resp = await getOrganizationRolePage({ page: 1, pageSize: 50 })
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
      await updateOrganizationRole({ ...values, organizationRoleID: editing.organizationRoleID })
    } else {
      await createOrganizationRole(values)
    }
    message.success('保存成功')
    setModalOpen(false)
    setEditing(null)
    form.resetFields()
    void load()
  }

  const remove = async (id: number) => {
    await deleteOrganizationRole(id)
    message.success('删除成功')
    void load()
  }

  const columns: ColumnsType<OrganizationRoleItem> = [
    { title: '组织角色ID', dataIndex: 'organizationRoleID', key: 'organizationRoleID' },
    { title: '名称', dataIndex: 'name', key: 'name' },
    { title: '描述', dataIndex: 'description', key: 'description' },
    { title: '类型', dataIndex: 'type', key: 'type' },
    {
      title: '操作',
      key: 'action',
      render: (_: unknown, record: OrganizationRoleItem) => (
        <>
          <Button size="small" onClick={() => { setEditing(record); form.setFieldsValue(record); setModalOpen(true) }}>编辑</Button>
          <Popconfirm title="确认删除？" onConfirm={() => void remove(record.organizationRoleID)}>
            <Button size="small" danger style={{ marginLeft: 8 }}>删除</Button>
          </Popconfirm>
        </>
      ),
    },
  ]

  return (
    <Card title="组织角色">
      <Button type="primary" style={{ marginBottom: 16 }} onClick={() => { setEditing(null); form.resetFields(); setModalOpen(true) }}>
        新建组织角色
      </Button>
      <Table<OrganizationRoleItem> rowKey="organizationRoleID" loading={loading} columns={columns} dataSource={list} pagination={false} />
      <Modal title={editing ? '编辑组织角色' : '新建组织角色'} open={modalOpen} onCancel={() => setModalOpen(false)} onOk={() => void submit()}>
        <Form form={form} layout="vertical">
          <Form.Item name="name" label="名称" rules={[{ required: true, message: '请输入名称' }]}>
            <Input />
          </Form.Item>
          <Form.Item name="description" label="描述">
            <Input.TextArea />
          </Form.Item>
          <Form.Item name="type" label="类型" rules={[{ required: true, message: '请选择类型' }]}>
            <Select
              options={[
                { label: '组织角色', value: 'organization' },
                { label: '部门角色', value: 'department' },
              ]}
            />
          </Form.Item>
        </Form>
      </Modal>
    </Card>
  )
}
