import { useCallback, useEffect, useState } from 'react'
import { Table, Button, Space, Input, Modal, Form, Switch, Popconfirm, message, Avatar, Tooltip } from 'antd'
import { PlusOutlined, ReloadOutlined, SearchOutlined } from '@ant-design/icons'
import type { ColumnsType } from 'antd/es/table'
import { PageContainer } from '@ark-iam/ui'
import { createUser, deleteUser, getUserPageList, updateUser, updateUserPassword, updateUserStatus } from '@ark-iam/api'
import type { UserItem } from '@ark-iam/types'
import { fmtTime, SuspendedTag } from '../../components/common'
import { useNavigate } from 'react-router-dom'
import { brand, EllipsisCell, IDCell } from '@ark-iam/ui'

export default function UserList() {
  const navigate = useNavigate()
  const [data, setData] = useState<UserItem[]>([])
  const [loading, setLoading] = useState(false)
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(10)
  const [total, setTotal] = useState(0)
  const [keyword, setKeyword] = useState('')

  const [modalOpen, setModalOpen] = useState(false)
  const [editing, setEditing] = useState<UserItem | null>(null)
  const [form] = Form.useForm()
  const [pwdModalOpen, setPwdModalOpen] = useState(false)
  const [pwdTarget, setPwdTarget] = useState<UserItem | null>(null)
  const [pwdForm] = Form.useForm()
  const [submitLoading, setSubmitLoading] = useState(false)

  const fetchData = useCallback(async () => {
    setLoading(true)
    try {
      const resp = await getUserPageList({ page, pageSize, name: keyword })
      setData(resp?.list || [])
      setTotal(resp?.total || 0)
    } catch {
      /* 拦截器已提示 */
    } finally {
      setLoading(false)
    }
  }, [page, pageSize, keyword])

  useEffect(() => {
    void fetchData()
  }, [fetchData])

  const handleCreate = () => {
    setEditing(null)
    form.resetFields()
    setModalOpen(true)
  }

  const handleEdit = (record: UserItem) => {
    setEditing(record)
    form.setFieldsValue({ name: record.name, avatar: record.avatar, isSuspended: record.isSuspended })
    setModalOpen(true)
  }

  const handleSubmit = async () => {
    try {
      const values = await form.validateFields()
      setSubmitLoading(true)
      if (editing) {
        await updateUser({ userID: editing.userID, ...values })
        message.success('修改成功')
      } else {
        await createUser(values)
        message.success('创建成功')
      }
      setModalOpen(false)
      void fetchData()
    } catch {
      /* 校验或请求失败 */
    } finally {
      setSubmitLoading(false)
    }
  }

  const handleToggleStatus = async (record: UserItem, checked: boolean) => {
    try {
      await updateUserStatus({ userID: record.userID, isSuspended: checked ? 1 : 0 })
      message.success(checked ? '已挂起' : '已恢复')
      void fetchData()
    } catch {
      /* 拦截器已提示 */
    }
  }

  const handleResetPassword = async () => {
    try {
      const values = await pwdForm.validateFields()
      if (!pwdTarget) return
      setSubmitLoading(true)
      await updateUserPassword({ userID: pwdTarget.userID, password: values.password })
      message.success('密码已重置')
      setPwdModalOpen(false)
      pwdForm.resetFields()
    } catch {
      /* 校验或请求失败 */
    } finally {
      setSubmitLoading(false)
    }
  }

  const columns: ColumnsType<UserItem> = [
    { title: 'ID', dataIndex: 'userID', key: 'userID', width: 150, render: (v: string) => <IDCell value={v} /> },
    {
      title: '用户',
      key: 'user',
      width: 200,
      render: (_, r) => (
        <Space>
          <Avatar size={30} style={{ background: brand.gradient }}>{r.name?.charAt(0)?.toUpperCase() || 'U'}</Avatar>
          <Space direction="vertical" size={0}>
            <span style={{ fontWeight: 500 }}>{r.name || '-'}</span>
            <span style={{ fontSize: 12, color: brand.textSecondary }}>@{r.username || '-'}</span>
          </Space>
        </Space>
      ),
    },
    { title: '邮箱', dataIndex: 'primaryEmail', key: 'primaryEmail', render: (v: string) => <EllipsisCell value={v} /> },
    { title: '手机号', dataIndex: 'primaryPhone', key: 'primaryPhone', width: 140 },
    {
      title: '状态',
      dataIndex: 'isSuspended',
      key: 'isSuspended',
      width: 120,
      render: (v: number) => <SuspendedTag value={v} />,
    },
    {
      title: '创建时间',
      key: 'createdAt',
      width: 160,
      render: (_, r) => fmtTime((r as UserItem & { createdAt?: number }).createdAt),
    },
    {
      title: '操作',
      key: 'action',
      width: 260,
      render: (_, r) => (
        <Space size={4}>
          <Button type="link" size="small" onClick={() => navigate(`/user/${r.userID}`)}>
            详情
          </Button>
          <Button type="link" size="small" onClick={() => handleEdit(r)}>
            编辑
          </Button>
          <Button type="link" size="small" onClick={() => { setPwdTarget(r); pwdForm.resetFields(); setPwdModalOpen(true) }}>
            重置密码
          </Button>
          <Tooltip title={r.isSuspended === 1 ? '恢复账号' : '挂起账号'}>
            <Switch size="small" checked={r.isSuspended !== 1} onChange={(c) => void handleToggleStatus(r, !c)} />
          </Tooltip>
          <Popconfirm
            title="确认删除该用户？"
            onConfirm={async () => {
              await deleteUser(r.userID)
              message.success('删除成功')
              void fetchData()
            }}
          >
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
      title="用户管理"
      description="管理租户内的用户账号、状态与登录凭证"
      extra={
        <Space>
          <Button icon={<ReloadOutlined />} onClick={() => void fetchData()}>
            刷新
          </Button>
          <Button type="primary" icon={<PlusOutlined />} onClick={handleCreate}>
            新建用户
          </Button>
        </Space>
      }
    >
      <div style={{ marginBottom: 16 }}>
        <Input.Search
          allowClear
          placeholder="按姓名搜索"
          prefix={<SearchOutlined />}
          style={{ width: 240 }}
          onSearch={(v) => { setKeyword(v); setPage(1) }}
        />
      </div>
      <Table<UserItem>
        rowKey="userID"
        columns={columns}
        dataSource={data}
        loading={loading}
        scroll={{ x: 900 }}
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

      <Modal
        title={editing ? '编辑用户' : '新建用户'}
        open={modalOpen}
        onOk={() => void handleSubmit()}
        onCancel={() => setModalOpen(false)}
        confirmLoading={submitLoading}
        destroyOnClose
        width={560}
      >
        <Form form={form} layout="vertical">
          {!editing && (
            <>
              <Form.Item name="username" label="用户名" rules={[{ required: true, message: '请输入用户名' }]}>
                <Input placeholder="登录账号" />
              </Form.Item>
              <Form.Item name="password" label="初始密码" rules={[{ required: true, message: '请输入初始密码' }, { min: 8, message: '至少 8 位，需含大小写字母和数字' }]}>
                <Input.Password placeholder="至少 8 位，需含大小写字母和数字" />
              </Form.Item>
              <Form.Item name="primaryEmail" label="邮箱" rules={[{ type: 'email', message: '邮箱格式不正确' }]}>
                <Input placeholder="选填" />
              </Form.Item>
              <Form.Item name="primaryPhone" label="手机号">
                <Input placeholder="选填" />
              </Form.Item>
            </>
          )}
          <Form.Item name="name" label="姓名" rules={[{ required: true, message: '请输入姓名' }]}>
            <Input placeholder="用户姓名" />
          </Form.Item>
          <Form.Item name="avatar" label="头像地址">
            <Input placeholder="https://... 选填" />
          </Form.Item>
          {editing && (
            <Form.Item name="isSuspended" label="状态" valuePropName="checked" getValueFromEvent={(c: boolean) => (c ? 0 : 1)} initialValue={0}>
              <Switch checkedChildren="正常" unCheckedChildren="挂起" />
            </Form.Item>
          )}
        </Form>
      </Modal>

      <Modal
        title={`重置密码 - ${pwdTarget?.name || ''}`}
        open={pwdModalOpen}
        onOk={() => void handleResetPassword()}
        onCancel={() => setPwdModalOpen(false)}
        confirmLoading={submitLoading}
        destroyOnClose
      >
        <Form form={pwdForm} layout="vertical">
          <Form.Item name="password" label="新密码" rules={[{ required: true, message: '请输入新密码' }, { min: 8, message: '至少 8 位，需含大小写字母和数字' }]}>
            <Input.Password placeholder="至少 8 位，需含大小写字母和数字" />
          </Form.Item>
        </Form>
      </Modal>
    </PageContainer>
  )
}
