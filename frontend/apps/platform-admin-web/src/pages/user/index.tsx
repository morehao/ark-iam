import { useCallback, useEffect, useState } from 'react'
import { Table, Button, Space, Input, Modal, Form, Switch, message, Avatar, Tooltip } from 'antd'
import { ReloadOutlined, SearchOutlined } from '@ant-design/icons'
import type { ColumnsType } from 'antd/es/table'
import { brand, EllipsisCell, fmtTime, PageContainer, SuspendedTag } from '@ark-iam/ui'
import { getUserPageList, updateUserPassword, updateUserStatus } from '@ark-iam/api'
import type { UserItem } from '@ark-iam/types'
import { useNavigate } from 'react-router-dom'

export default function UserList() {
  const navigate = useNavigate()
  const [data, setData] = useState<UserItem[]>([])
  const [loading, setLoading] = useState(false)
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(10)
  const [total, setTotal] = useState(0)
  const [keyword, setKeyword] = useState('')

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
    { title: 'ID', dataIndex: 'userID', key: 'userID', width: 150, render: (v: string) => <EllipsisCell value={v} /> },
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
      width: 100,
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
      width: 200,
      render: (_, r) => (
        <Space size={4}>
          <Button type="link" size="small" onClick={() => navigate(`/user/${r.userID}`)}>
            详情
          </Button>
          <Button type="link" size="small" onClick={() => { setPwdTarget(r); pwdForm.resetFields(); setPwdModalOpen(true) }}>
            重置密码
          </Button>
          <Tooltip title={r.isSuspended === 1 ? '恢复账号' : '挂起账号'}>
            <Switch size="small" checked={r.isSuspended !== 1} onChange={(c) => void handleToggleStatus(r, !c)} />
          </Tooltip>
        </Space>
      ),
    },
  ]

  return (
    <PageContainer
      title="用户管理（平台视角）"
      description="跨租户用户目录排查：只读查看 + 挂起/恢复 + 重置密码；租户内账号管理（创建/组织归属/角色）请使用租户自服务控制台"
      extra={
        <Button icon={<ReloadOutlined />} onClick={() => void fetchData()}>
          刷新
        </Button>
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
