import { useCallback, useEffect, useState } from 'react'
import { Table, Button, Modal, Form, message, Popconfirm, Space, Select, Tag } from 'antd'
import { PlusOutlined, ReloadOutlined } from '@ant-design/icons'
import type { ColumnsType } from 'antd/es/table'
import { PageContainer } from '@ark-iam/ui'
import { getUserPageList } from '@ark-iam/api'
import { createOrganizationUser, deleteOrganizationUser, getOrganizationPage, getOrganizationUserPage } from '../../api/organization'
import type { OrganizationItem, OrganizationUserItem, UserItem } from '@ark-iam/types'

export default function OrganizationUserList() {
  const [list, setList] = useState<OrganizationUserItem[]>([])
  const [organizations, setOrganizations] = useState<OrganizationItem[]>([])
  const [users, setUsers] = useState<UserItem[]>([])
  const [loading, setLoading] = useState(false)
  const [modalOpen, setModalOpen] = useState(false)
  const [submitLoading, setSubmitLoading] = useState(false)
  const [form] = Form.useForm()

  const load = useCallback(async () => {
    setLoading(true)
    try {
      const [rels, orgs, us] = await Promise.all([
        getOrganizationUserPage({ page: 1, pageSize: 100 }),
        getOrganizationPage({ page: 1, pageSize: 100 }),
        getUserPageList({ page: 1, pageSize: 100 }).catch(() => ({ list: [], total: 0 })),
      ])
      setList(rels.list || [])
      setOrganizations(orgs.list || [])
      setUsers(us.list || [])
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    void load()
  }, [load])

  const orgName = (id: number) => organizations.find((o) => o.organizationID === id)?.name ?? `#${id}`
  const userName = (id: number) => {
    const u = users.find((x) => x.userID === id)
    return u ? (u.name ? `${u.name}（@${u.username || id}）` : `@${u.username || id}`) : `#${id}`
  }

  const submit = async () => {
    try {
      const values = await form.validateFields()
      setSubmitLoading(true)
      await createOrganizationUser(values)
      message.success('创建成功')
      setModalOpen(false)
      form.resetFields()
      void load()
    } catch {
      /* 校验或请求失败 */
    } finally {
      setSubmitLoading(false)
    }
  }

  const remove = async (record: OrganizationUserItem) => {
    await deleteOrganizationUser({ organizationID: record.organizationID, userID: record.userID })
    message.success('删除成功')
    void load()
  }

  const columns: ColumnsType<OrganizationUserItem> = [
    { title: '组织', key: 'org', width: 220, render: (_, r) => <Tag color="blue">{orgName(r.organizationID)}</Tag> },
    { title: '用户', key: 'user', render: (_, r) => userName(r.userID) },
    {
      title: '操作',
      key: 'action',
      width: 100,
      render: (_: unknown, record: OrganizationUserItem) => (
        <Popconfirm title="确认移除？" onConfirm={() => void remove(record)}>
          <Button type="link" size="small" danger>
            移除
          </Button>
        </Popconfirm>
      ),
    },
  ]

  return (
    <PageContainer
      title="组织用户"
      description="将用户加入组织"
      extra={
        <Space>
          <Button icon={<ReloadOutlined />} onClick={() => void load()}>
            刷新
          </Button>
          <Button
            type="primary"
            icon={<PlusOutlined />}
            onClick={() => {
              form.resetFields()
              setModalOpen(true)
            }}
          >
            新建组织用户
          </Button>
        </Space>
      }
    >
      <Table<OrganizationUserItem>
        rowKey={(r) => `${r.organizationID}-${r.userID}`}
        loading={loading}
        columns={columns}
        dataSource={list}
        pagination={false}
      />
      <Modal title="新建组织用户" open={modalOpen} onCancel={() => setModalOpen(false)} onOk={() => void submit()} confirmLoading={submitLoading} destroyOnClose>
        <Form form={form} layout="vertical">
          <Form.Item name="organizationID" label="组织" rules={[{ required: true, message: '请选择组织' }]}>
            <Select
              showSearch
              optionFilterProp="label"
              placeholder="选择组织"
              options={organizations.map((o) => ({ label: o.name, value: o.organizationID }))}
            />
          </Form.Item>
          <Form.Item name="userID" label="用户" rules={[{ required: true, message: '请选择用户' }]}>
            <Select
              showSearch
              optionFilterProp="label"
              placeholder="选择用户"
              options={users.map((u) => ({ label: u.name ? `${u.name}（@${u.username || u.userID}）` : `@${u.username || u.userID}`, value: u.userID }))}
            />
          </Form.Item>
        </Form>
      </Modal>
    </PageContainer>
  )
}
