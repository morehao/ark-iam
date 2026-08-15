import { useCallback, useEffect, useState } from 'react'
import { Table, Button, Modal, Form, Input, message, Popconfirm, Space, Select, Tag } from 'antd'
import { PlusOutlined, ReloadOutlined } from '@ant-design/icons'
import type { ColumnsType } from 'antd/es/table'
import { PageContainer } from '@ark-iam/ui'
import {
  createOrganizationRole,
  deleteOrganizationRole,
  getOrganizationPage,
  getOrganizationRolePage,
  updateOrganizationRole,
} from '../../api/organization'
import type { OrganizationItem, OrganizationRoleItem } from '@ark-iam/types'

export default function OrganizationRoleList() {
  const [list, setList] = useState<OrganizationRoleItem[]>([])
  const [organizations, setOrganizations] = useState<OrganizationItem[]>([])
  const [loading, setLoading] = useState(false)
  const [modalOpen, setModalOpen] = useState(false)
  const [editing, setEditing] = useState<OrganizationRoleItem | null>(null)
  const [submitLoading, setSubmitLoading] = useState(false)
  const [form] = Form.useForm()

  const load = useCallback(async () => {
    setLoading(true)
    try {
      const [roles, orgs] = await Promise.all([getOrganizationRolePage({ page: 1, pageSize: 50 }), getOrganizationPage({ page: 1, pageSize: 100 })])
      setList(roles.list || [])
      setOrganizations(orgs.list || [])
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    void load()
  }, [load])

  const orgName = (id: string) => organizations.find((o) => o.organizationID === id)?.name ?? `#${id}`

  const submit = async () => {
    try {
      const values = await form.validateFields()
      setSubmitLoading(true)
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
    } catch {
      /* 校验或请求失败 */
    } finally {
      setSubmitLoading(false)
    }
  }

  const remove = async (id: string) => {
    await deleteOrganizationRole(id)
    message.success('删除成功')
    void load()
  }

  const columns: ColumnsType<OrganizationRoleItem> = [
    { title: '角色ID', dataIndex: 'organizationRoleID', key: 'organizationRoleID', width: 100 },
    { title: '组织', key: 'organization', width: 180, render: (_, r) => orgName(r.organizationID) },
    { title: '名称', dataIndex: 'name', key: 'name' },
    { title: '描述', dataIndex: 'description', key: 'description', ellipsis: true },
    {
      title: '类型',
      dataIndex: 'type',
      key: 'type',
      width: 120,
      render: (v: string) => (v === 'organization' ? <Tag color="blue">组织角色</Tag> : v === 'department' ? <Tag color="geekblue">部门角色</Tag> : <Tag>{v}</Tag>),
    },
    {
      title: '操作',
      key: 'action',
      width: 160,
      render: (_: unknown, record: OrganizationRoleItem) => (
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
          <Popconfirm title="确认删除？" onConfirm={() => void remove(record.organizationRoleID)}>
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
      title="组织角色"
      description="管理组织内的角色定义"
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
            新建组织角色
          </Button>
        </Space>
      }
    >
      <Table<OrganizationRoleItem> rowKey="organizationRoleID" loading={loading} columns={columns} dataSource={list} pagination={false} />
      <Modal
        title={editing ? '编辑组织角色' : '新建组织角色'}
        open={modalOpen}
        onCancel={() => setModalOpen(false)}
        onOk={() => void submit()}
        confirmLoading={submitLoading}
        destroyOnClose
      >
        <Form form={form} layout="vertical">
          <Form.Item name="organizationID" label="所属组织" rules={[{ required: true, message: '请选择组织' }]}>
            <Select
              placeholder="选择组织"
              showSearch
              optionFilterProp="label"
              options={organizations.map((o) => ({ label: o.name, value: o.organizationID }))}
            />
          </Form.Item>
          <Form.Item name="name" label="角色名称" rules={[{ required: true, message: '请输入名称' }]}>
            <Input placeholder="如：组织管理员" />
          </Form.Item>
          <Form.Item name="type" label="类型" rules={[{ required: true, message: '请选择类型' }]} initialValue="organization">
            <Select
              options={[
                { label: '组织角色', value: 'organization' },
                { label: '部门角色', value: 'department' },
              ]}
            />
          </Form.Item>
          <Form.Item name="description" label="描述">
            <Input.TextArea rows={3} />
          </Form.Item>
        </Form>
      </Modal>
    </PageContainer>
  )
}
