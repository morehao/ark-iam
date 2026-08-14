import { useCallback, useEffect, useState } from 'react'
import { AutoComplete, Button, Descriptions, Drawer, Form, Input, message, Modal, Popconfirm, Select, Space, Table, Tag, Typography } from 'antd'
import { PlusOutlined, ReloadOutlined, SearchOutlined, UserAddOutlined } from '@ant-design/icons'
import type { ColumnsType } from 'antd/es/table'
import { PageContainer } from '@ark-iam/ui'
import { assignRoleUsers, createRole, deleteRole, getRolePageList, getRoleUsers, getUserPageList, removeRoleUser, updateRole } from '@ark-iam/api'
import type { RoleItem, RoleUserItem, UserItem } from '@ark-iam/types'
import { fmtTime } from '../../components/common'

/** 角色类型渲染：User→用户(蓝)，Admin→管理员(紫)，其他原样展示 */
function renderRoleType(type?: string) {
  switch (type) {
    case 'User':
      return <Tag color="blue">用户</Tag>
    case 'Admin':
      return <Tag color="purple">管理员</Tag>
    default:
      return <Tag>{type || '-'}</Tag>
  }
}

export default function RoleList() {
  const [data, setData] = useState<RoleItem[]>([])
  const [loading, setLoading] = useState(false)
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(10)
  const [total, setTotal] = useState(0)
  const [keyword, setKeyword] = useState('')

  const [modalOpen, setModalOpen] = useState(false)
  const [editing, setEditing] = useState<RoleItem | null>(null)
  const [form] = Form.useForm()
  const [submitLoading, setSubmitLoading] = useState(false)

  // 详情 Drawer
  const [detailOpen, setDetailOpen] = useState(false)
  const [detailRole, setDetailRole] = useState<RoleItem | null>(null)

  // 成员管理 Drawer
  const [memberOpen, setMemberOpen] = useState(false)
  const [memberRole, setMemberRole] = useState<RoleItem | null>(null)
  const [members, setMembers] = useState<RoleUserItem[]>([])
  const [membersTotal, setMembersTotal] = useState(0)
  const [membersLoading, setMembersLoading] = useState(false)

  // 添加成员
  const [addOpen, setAddOpen] = useState(false)
  const [userOptions, setUserOptions] = useState<UserItem[]>([])
  const [usersLoading, setUsersLoading] = useState(false)
  const [selectedUserIds, setSelectedUserIds] = useState<number[]>([])
  const [assignLoading, setAssignLoading] = useState(false)

  const fetchData = useCallback(async () => {
    setLoading(true)
    try {
      const resp = await getRolePageList({ page, pageSize, name: keyword })
      setData(resp?.list || [])
      setTotal(resp?.total || 0)
    } catch (error) {
      console.error('获取角色列表失败:', error)
    } finally {
      setLoading(false)
    }
  }, [page, pageSize, keyword])

  useEffect(() => {
    void fetchData()
  }, [fetchData])

  const fetchMembers = useCallback(async (roleID: number) => {
    setMembersLoading(true)
    try {
      const resp = await getRoleUsers(roleID)
      setMembers(resp?.users || [])
      setMembersTotal(resp?.total || 0)
    } catch (error) {
      console.error('获取角色成员失败:', error)
    } finally {
      setMembersLoading(false)
    }
  }, [])

  const handleCreate = () => {
    setEditing(null)
    form.resetFields()
    setModalOpen(true)
  }

  const handleEdit = (record: RoleItem) => {
    setEditing(record)
    form.setFieldsValue({
      name: record.name,
      code: record.code,
      type: record.type,
      description: record.description,
    })
    setModalOpen(true)
  }

  const handleSubmit = async () => {
    try {
      const values = await form.validateFields()
      setSubmitLoading(true)
      if (editing) {
        await updateRole({ roleID: editing.roleID, ...values })
        message.success('修改成功')
      } else {
        await createRole(values)
        message.success('创建成功')
      }
      setModalOpen(false)
      void fetchData()
    } catch (error) {
      /* 校验失败或请求失败，拦截器已提示 */
      console.error('保存角色失败:', error)
    } finally {
      setSubmitLoading(false)
    }
  }

  const handleDelete = async (record: RoleItem) => {
    try {
      await deleteRole(record.roleID)
      message.success('删除成功')
      void fetchData()
    } catch (error) {
      console.error('删除角色失败:', error)
    }
  }

  const handleOpenDetail = (record: RoleItem) => {
    setMemberOpen(false)
    setDetailRole(record)
    setDetailOpen(true)
    void fetchMembers(record.roleID)
  }

  const handleOpenMembers = (record: RoleItem) => {
    setDetailOpen(false)
    setMemberRole(record)
    setMemberOpen(true)
    void fetchMembers(record.roleID)
  }

  const handleOpenAdd = async () => {
    setSelectedUserIds([])
    setAddOpen(true)
    if (userOptions.length === 0) {
      setUsersLoading(true)
      try {
        const resp = await getUserPageList({ page: 1, pageSize: 50 })
        setUserOptions(resp?.list || [])
      } catch (error) {
        console.error('加载用户列表失败:', error)
      } finally {
        setUsersLoading(false)
      }
    }
  }

  const handleAssign = async () => {
    if (!memberRole) return
    if (selectedUserIds.length === 0) {
      message.warning('请选择要添加的用户')
      return
    }
    setAssignLoading(true)
    try {
      await assignRoleUsers(memberRole.roleID, selectedUserIds)
      message.success('添加成功')
      setAddOpen(false)
      void fetchMembers(memberRole.roleID)
    } catch (error) {
      console.error('添加成员失败:', error)
    } finally {
      setAssignLoading(false)
    }
  }

  const handleRemoveMember = async (record: RoleUserItem) => {
    if (!memberRole) return
    try {
      await removeRoleUser(memberRole.roleID, record.userId)
      message.success('移除成功')
      void fetchMembers(memberRole.roleID)
    } catch (error) {
      console.error('移除成员失败:', error)
    }
  }

  const columns: ColumnsType<RoleItem> = [
    { title: '角色ID', dataIndex: 'roleID', key: 'roleID', width: 80 },
    { title: '角色名称', dataIndex: 'name', key: 'name', width: 160, render: (v: string) => v || '-' },
    { title: '角色编码', dataIndex: 'code', key: 'code', width: 160, render: (v: string) => v || '-' },
    {
      title: '类型',
      dataIndex: 'type',
      key: 'type',
      width: 110,
      render: (v: string) => renderRoleType(v),
    },
    { title: '描述', dataIndex: 'description', key: 'description', ellipsis: true, render: (v: string) => v || '-' },
    {
      title: '创建时间',
      dataIndex: 'createdAt',
      key: 'createdAt',
      width: 160,
      render: (_, r) => fmtTime(r.createdAt),
    },
    {
      title: '操作',
      key: 'action',
      width: 250,
      render: (_, r) => (
        <Space size={4}>
          <Button type="link" size="small" onClick={() => handleOpenDetail(r)}>
            详情
          </Button>
          <Button type="link" size="small" onClick={() => handleOpenMembers(r)}>
            成员
          </Button>
          <Button type="link" size="small" onClick={() => handleEdit(r)}>
            编辑
          </Button>
          <Popconfirm title="确认删除该角色？" onConfirm={() => void handleDelete(r)}>
            <Button type="link" size="small" danger>
              删除
            </Button>
          </Popconfirm>
        </Space>
      ),
    },
  ]

  const readOnlyMemberColumns: ColumnsType<RoleUserItem> = [
    { title: '用户ID', dataIndex: 'userId', key: 'userId', width: 80 },
    { title: '姓名', dataIndex: 'name', key: 'name', width: 120, render: (v: string) => v || '-' },
    { title: '用户名', dataIndex: 'username', key: 'username', width: 120, render: (v: string) => v || '-' },
    { title: '邮箱', dataIndex: 'email', key: 'email', ellipsis: true, render: (v: string) => v || '-' },
    {
      title: '加入时间',
      dataIndex: 'createdAt',
      key: 'createdAt',
      width: 170,
      render: (v: string) => fmtTime(v),
    },
  ]

  const memberColumns: ColumnsType<RoleUserItem> = [
    ...readOnlyMemberColumns,
    {
      title: '操作',
      key: 'action',
      width: 90,
      render: (_, r) => (
        <Popconfirm title="确认将该成员移出角色？" onConfirm={() => void handleRemoveMember(r)}>
          <Button type="link" size="small" danger>
            移除
          </Button>
        </Popconfirm>
      ),
    },
  ]

  return (
    <PageContainer
      title="角色管理"
      description="管理角色及其成员"
      extra={
        <Space>
          <Button icon={<ReloadOutlined />} onClick={() => void fetchData()}>
            刷新
          </Button>
          <Button type="primary" icon={<PlusOutlined />} onClick={handleCreate}>
            新建角色
          </Button>
        </Space>
      }
    >
      <div style={{ marginBottom: 16 }}>
        <Input.Search
          allowClear
          placeholder="按角色名称搜索"
          prefix={<SearchOutlined />}
          style={{ width: 240 }}
          onSearch={(v) => {
            setKeyword(v)
            setPage(1)
          }}
        />
      </div>

      <Table<RoleItem>
        rowKey="roleID"
        columns={columns}
        dataSource={data}
        loading={loading}
        scroll={{ x: 1000 }}
        pagination={{
          current: page,
          pageSize,
          total,
          showSizeChanger: true,
          showTotal: (t) => `共 ${t} 条`,
          onChange: (p, ps) => {
            setPage(p)
            setPageSize(ps)
          },
        }}
      />

      {/* 新建 / 编辑角色 */}
      <Modal
        title={editing ? '编辑角色' : '新建角色'}
        open={modalOpen}
        onOk={() => void handleSubmit()}
        onCancel={() => setModalOpen(false)}
        confirmLoading={submitLoading}
        destroyOnClose
        width={560}
      >
        <Form form={form} layout="vertical">
          <Form.Item name="name" label="角色名称" rules={[{ required: true, message: '请输入角色名称' }]}>
            <Input placeholder="如：平台管理员" />
          </Form.Item>
          <Form.Item name="code" label="角色编码" rules={[{ required: true, message: '请输入角色编码' }]}>
            <Input placeholder="如：platform_admin" />
          </Form.Item>
          <Form.Item name="type" label="类型" initialValue="User">
            <AutoComplete
              placeholder="输入或选择类型，如 User / Admin / Guest"
              options={[{ value: 'User' }, { value: 'Admin' }, { value: 'Guest' }]}
              filterOption={(input, option) => String(option?.value ?? '').toLowerCase().includes(input.toLowerCase())}
            />
          </Form.Item>
          <Form.Item name="description" label="描述">
            <Input.TextArea rows={3} placeholder="选填" />
          </Form.Item>
        </Form>
      </Modal>

      {/* 角色详情 Drawer */}
      <Drawer
        title={`角色详情 - ${detailRole?.name || ''}`}
        width={520}
        open={detailOpen}
        onClose={() => setDetailOpen(false)}
      >
        {detailRole && (
          <Descriptions column={2} size="small" bordered>
            <Descriptions.Item label="角色ID">{detailRole.roleID}</Descriptions.Item>
            <Descriptions.Item label="角色名称">{detailRole.name}</Descriptions.Item>
            <Descriptions.Item label="角色编码">{detailRole.code}</Descriptions.Item>
            <Descriptions.Item label="类型">{renderRoleType(detailRole.type)}</Descriptions.Item>
            <Descriptions.Item label="默认角色">{detailRole.isDefault === 1 ? '是' : '否'}</Descriptions.Item>
            <Descriptions.Item label="创建时间">{fmtTime(detailRole.createdAt)}</Descriptions.Item>
            <Descriptions.Item label="描述" span={2}>
              {detailRole.description || '-'}
            </Descriptions.Item>
          </Descriptions>
        )}
        <div style={{ margin: '16px 0 12px' }}>
          <Typography.Text strong>成员列表</Typography.Text>
          <Typography.Text type="secondary" style={{ marginLeft: 8 }}>
            共 {membersTotal} 位成员
          </Typography.Text>
        </div>
        <Table<RoleUserItem>
          rowKey="userId"
          columns={readOnlyMemberColumns}
          dataSource={members}
          loading={membersLoading}
          size="small"
          pagination={false}
          scroll={{ x: 520 }}
        />
      </Drawer>

      {/* 成员管理 Drawer */}
      <Drawer
        title={`成员管理 - ${memberRole?.name || ''}`}
        width={560}
        open={memberOpen}
        onClose={() => setMemberOpen(false)}
        footer={
          <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
            <Typography.Text type="secondary">共 {membersTotal} 位成员</Typography.Text>
            <Button type="primary" icon={<UserAddOutlined />} onClick={() => void handleOpenAdd()}>
              添加成员
            </Button>
          </div>
        }
      >
        <Table<RoleUserItem>
          rowKey="userId"
          columns={memberColumns}
          dataSource={members}
          loading={membersLoading}
          size="small"
          pagination={false}
          scroll={{ x: 560 }}
        />
      </Drawer>

      {/* 添加成员 Modal */}
      <Modal
        title={`添加成员 - ${memberRole?.name || ''}`}
        open={addOpen}
        onOk={() => void handleAssign()}
        onCancel={() => setAddOpen(false)}
        confirmLoading={assignLoading}
        destroyOnClose
      >
        <Select
          mode="multiple"
          allowClear
          showSearch
          optionFilterProp="label"
          placeholder="搜索并选择要添加的用户"
          loading={usersLoading}
          style={{ width: '100%' }}
          value={selectedUserIds}
          onChange={(v: number[]) => setSelectedUserIds(v)}
          options={userOptions.map((u) => ({ label: `${u.name}(${u.username})`, value: u.userID }))}
        />
      </Modal>
    </PageContainer>
  )
}
