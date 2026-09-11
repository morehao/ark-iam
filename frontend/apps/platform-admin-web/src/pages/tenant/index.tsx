import { useCallback, useEffect, useState } from 'react'
import { Button, Divider, Form, Input, message, Modal, Select, Space, Switch, Table, Tooltip } from 'antd'
import { PlusOutlined, QuestionCircleOutlined, ReloadOutlined, SearchOutlined } from '@ant-design/icons'
import type { ColumnsType } from 'antd/es/table'
import { EllipsisCell, IDCell, InitialPasswordModal, PageContainer, RowActions, SuspendedTag, timeColumn, TypeTag } from '@ark-iam/ui'
import { createTenant, deleteTenant, getTenantPageList, resetTenantAdminPassword, updateTenant } from '@ark-iam/api'
import type { TenantItem, TenantStatus } from '@ark-iam/types'

// 租户状态筛选项（后端 model.TenantStatus：active-正常 / suspended-已挂起）。
// 「全部」由 allowClear 的空值表达，无需单独一项。
const TENANT_STATUS_OPTIONS = [
  { value: 'active', label: '正常' },
  { value: 'suspended', label: '挂起' },
]

// 租户类型（后端 model.TenantType：customer-客户租户 / platform-平台租户）：
// 用于区分「外部客户租户」与「平台自运营租户」，当前仅作分类标识，
// 不参与数据隔离与权限判定（隔离一律按 tenant_id）。
const TENANT_TYPE_OPTIONS = [
  { value: 'customer', label: '客户租户', title: '外部客户/合作方的独立租户' },
  { value: 'platform', label: '平台租户', title: '平台自运营租户（种子数据 Default Tenant 即平台租户）' },
]

const TENANT_TYPE_TIP =
  '客户租户（customer）：外部客户/合作方的独立数据与权限边界；平台租户（platform）：平台自运营租户（如 Default Tenant）。当前该字段仅作分类标识，不参与数据隔离与权限判定。'

export default function TenantList() {
  const [data, setData] = useState<TenantItem[]>([])
  const [loading, setLoading] = useState(false)
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(10)
  const [total, setTotal] = useState(0)
  const [keyword, setKeyword] = useState('')
  const [status, setStatus] = useState<TenantStatus | undefined>(undefined)

  const [modalOpen, setModalOpen] = useState(false)
  const [editing, setEditing] = useState<TenantItem | null>(null)
  const [form] = Form.useForm()
  const [submitLoading, setSubmitLoading] = useState(false)

  // 内置管理员初始/临时密码（建租户后或重置后展示一次，关闭即不可再查）
  const [initialPassword, setInitialPassword] = useState('')

  const fetchData = useCallback(async () => {
    setLoading(true)
    try {
      const resp = await getTenantPageList({ page, pageSize, name: keyword, status })
      setData(resp?.list || [])
      setTotal(resp?.total || 0)
    } catch {
      /* 拦截器已提示 */
    } finally {
      setLoading(false)
    }
  }, [page, pageSize, keyword, status])

  useEffect(() => {
    void fetchData()
  }, [fetchData])

  const handleCreate = () => {
    setEditing(null)
    form.resetFields()
    form.setFieldsValue({ type: 'customer' })
    setModalOpen(true)
  }

  const handleEdit = (record: TenantItem) => {
    setEditing(record)
    form.setFieldsValue({
      name: record.name,
      type: record.type,
      tag: record.tag,
      dbUser: record.dbUser,
      status: record.status,
    })
    setModalOpen(true)
  }

  const handleSubmit = async () => {
    try {
      const values = await form.validateFields()
      setSubmitLoading(true)
      if (editing) {
        await updateTenant({ tenantID: editing.tenantID, ...values })
        message.success('修改成功')
      } else {
        const created = await createTenant({
          name: values.name,
          type: values.type,
          tag: values.tag,
          dbUser: values.dbUser,
          admin: {
            name: values.adminName,
            username: values.adminUsername,
            primaryEmail: values.adminPrimaryEmail,
            primaryPhone: values.adminPrimaryPhone,
          },
        })
        setModalOpen(false)
        void fetchData()
        if (created.adminInitialPassword) {
          // 管理员初始临时密码仅此一次返回；该管理员首次登录必须改密
          message.success('创建成功')
          setInitialPassword(created.adminInitialPassword)
        } else {
          Modal.info({
            title: '租户已创建',
            content: '该管理员的邮箱/手机已存在统一身份账号，密码未被改动；如需登录凭据，可在租户列表执行「重置管理员密码」。',
          })
        }
        return
      }
      setModalOpen(false)
      void fetchData()
    } catch {
      /* 校验或请求失败 */
    } finally {
      setSubmitLoading(false)
    }
  }

  /** 重置内置管理员密码：兜底路径，仅作用于建租户时由平台创建的管理员（source=builtin） */
  const handleResetAdminPassword = async (record: TenantItem) => {
    try {
      const resp = await resetTenantAdminPassword(record.tenantID)
      setInitialPassword(resp.initialPassword)
    } catch {
      /* 拦截器已提示 */
    }
  }

  const handleDelete = async (record: TenantItem) => {
    try {
      await deleteTenant(record.tenantID)
      message.success('删除成功')
      void fetchData()
    } catch {
      /* 拦截器已提示 */
    }
  }

  const columns: ColumnsType<TenantItem> = [
    { title: 'ID', dataIndex: 'tenantID', key: 'tenantID', width: 150, render: (v: string) => <IDCell value={v} /> },
    { title: '租户名', dataIndex: 'name', key: 'name', width: 180, render: (v: string) => <EllipsisCell value={v} /> },
    {
      title: '编码',
      dataIndex: 'code',
      key: 'code',
      width: 180,
      render: (v: string) => <span style={{ fontFamily: 'monospace' }}>{v || '-'}</span>,
    },
    {
      title: (
        <Space size={4}>
          类型
          <Tooltip title={TENANT_TYPE_TIP}>
            <QuestionCircleOutlined style={{ color: 'rgba(0, 0, 0, 0.45)' }} />
          </Tooltip>
        </Space>
      ),
      dataIndex: 'type',
      key: 'type',
      width: 120,
      render: (v: string) => <TypeTag value={v} />,
    },
    { title: '标签', dataIndex: 'tag', key: 'tag', width: 140, render: (v: string) => v || '-' },
    {
      title: '状态',
      dataIndex: 'status',
      key: 'status',
      width: 100,
      render: (v: string) => <SuspendedTag value={v} />,
    },
    timeColumn<TenantItem>({ title: '创建时间', dataIndex: 'createdAt' }),
    timeColumn<TenantItem>({ title: '更新时间', dataIndex: 'updatedAt' }),
    {
      title: '操作',
      key: 'action',
      width: 240,
      render: (_, r) => (
        <RowActions
          actions={[
            { key: 'edit', label: '编辑', onClick: () => handleEdit(r) },
            {
              key: 'resetAdminPassword',
              label: '重置管理员密码',
              confirm: '将重置该租户内置管理员的密码：新临时密码仅展示一次，其既有会话立即失效。确认重置？',
              onClick: () => void handleResetAdminPassword(r),
            },
            { key: 'delete', label: '删除', danger: true, confirm: '确认删除该租户？', onClick: () => void handleDelete(r) },
          ]}
        />
      ),
    },
  ]

  return (
    <PageContainer
      title="租户管理"
      description="平台租户生命周期管理"
      extra={
        <Space>
          <Button icon={<ReloadOutlined />} onClick={() => void fetchData()}>
            刷新
          </Button>
          <Button type="primary" icon={<PlusOutlined />} onClick={handleCreate}>
            新建租户
          </Button>
        </Space>
      }
    >
      <Space style={{ marginBottom: 16 }} wrap>
        <Input.Search
          allowClear
          placeholder="按租户名搜索"
          prefix={<SearchOutlined />}
          style={{ width: 240 }}
          onSearch={(v) => {
            setKeyword(v)
            setPage(1)
          }}
        />
        <Select
          allowClear
          placeholder="全部状态"
          style={{ width: 140 }}
          value={status}
          onChange={(v: TenantStatus | undefined) => {
            setStatus(v)
            setPage(1)
          }}
          options={TENANT_STATUS_OPTIONS}
        />
      </Space>
      <Table<TenantItem>
        rowKey="tenantID"
        columns={columns}
        dataSource={data}
        loading={loading}
        scroll={{ x: 1490 }}
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
        title={editing ? '编辑租户' : '新建租户'}
        open={modalOpen}
        onOk={() => void handleSubmit()}
        onCancel={() => setModalOpen(false)}
        confirmLoading={submitLoading}
        destroyOnClose
        width={560}
      >
        <Form form={form} layout="vertical">
          <Form.Item name="name" label="租户名称" rules={[{ required: true, message: '请输入租户名称' }]}>
            <Input placeholder="租户名称" />
          </Form.Item>
          <Form.Item name="type" label="类型" tooltip={TENANT_TYPE_TIP}>
            <Select options={TENANT_TYPE_OPTIONS} />
          </Form.Item>
          <Form.Item name="tag" label="标签">
            <Input placeholder="选填" />
          </Form.Item>
          <Form.Item
            label="编码"
            extra={editing ? '编码由服务端生成，创建后不可修改' : '保存后由服务端自动生成（t_随机段）'}
          >
            <Input value={editing?.code || ''} disabled placeholder="保存后自动生成" />
          </Form.Item>
          <Form.Item name="dbUser" label="数据库用户">
            <Input placeholder="选填" />
          </Form.Item>
          {!editing && (
            <>
              <Divider orientation="left" plain>
                租户管理员
              </Divider>
              <Form.Item
                name="adminName"
                label="管理员姓名"
                rules={[{ required: true, message: '请输入管理员姓名' }]}
                extra="建租户即创建该租户的初始管理员；初始密码由系统生成，仅在创建后展示一次，首次登录必须修改"
              >
                <Input placeholder="管理员姓名" />
              </Form.Item>
              <Form.Item name="adminUsername" label="管理员用户名">
                <Input placeholder="选填，全局用户名" />
              </Form.Item>
              <Form.Item
                name="adminPrimaryEmail"
                label="管理员邮箱"
                dependencies={['adminPrimaryPhone']}
                rules={[
                  ({ getFieldValue }) => ({
                    validator(_r, value: string) {
                      if (value || getFieldValue('adminPrimaryPhone')) return Promise.resolve()
                      return Promise.reject(new Error('邮箱与手机号至少填写一个'))
                    },
                  }),
                ]}
              >
                <Input placeholder="用于登录与密码交接" />
              </Form.Item>
              <Form.Item name="adminPrimaryPhone" label="管理员手机号" dependencies={['adminPrimaryEmail']}>
                <Input placeholder="选填（与邮箱至少一个）" />
              </Form.Item>
            </>
          )}
          {editing && (
            <Form.Item
              name="status"
              label="状态"
              tooltip="挂起后该租户成员无法登录、已签发会话会被撤销；不能挂起你当前所在的租户"
              valuePropName="checked"
              getValueFromEvent={(checked: boolean) => (checked ? 'active' : 'suspended')}
              getValueProps={(v?: string) => ({ checked: v !== 'suspended' })}
            >
              <Switch checkedChildren="正常" unCheckedChildren="挂起" />
            </Form.Item>
          )}
        </Form>
      </Modal>

      {/* 内置管理员初始/临时密码：仅此一次展示 */}
      <InitialPasswordModal
        password={initialPassword}
        description="该管理员首次登录必须修改密码；如需再次获取凭据，可在租户列表执行「重置管理员密码」。"
        onClose={() => setInitialPassword('')}
      />
    </PageContainer>
  )
}
