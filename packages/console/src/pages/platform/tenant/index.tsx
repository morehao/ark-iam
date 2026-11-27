import { useCallback, useEffect, useState } from 'react'
import { Button, Divider, Form, Input, message, Modal, Select, Space, Switch, Table, Tooltip } from 'antd'
import { DeleteOutlined, EditOutlined, LogoutOutlined, PlusOutlined, QuestionCircleOutlined, ReloadOutlined, RollbackOutlined, SearchOutlined } from '@ant-design/icons'
import type { ColumnsType } from 'antd/es/table'
import { actionColumn, CODE_COL_WIDTH, idColumn, InitialPasswordModal, NAME_COL_WIDTH, PageContainer, StatCard, StatRow, STATUS_COL_WIDTH, tableScrollX, TAG_COL_WIDTH, TenantStatusTag, textColumn, timeColumn, TypeTag } from '@ark-iam/ui'
import { createTenant, deleteTenant, getTenantPageList, offboardTenant, resetTenantAdminPassword, updateTenant } from '@ark-iam/api'
import type { TenantIsolationMode, TenantItem, TenantStatus } from '@ark-iam/types'

// 租户状态筛选项（后端 model.TenantStatus 三值：provisioning-建租中 / active-正常 / suspended-已挂起）。
// 「全部」由 allowClear 的空值表达，无需单独一项。
const TENANT_STATUS_OPTIONS = [
  { value: 'provisioning', label: '建租中' },
  { value: 'active', label: '正常' },
  { value: 'suspended', label: '挂起' },
]

// 租户隔离模式（后端 model.TenantIsolationMode）：dedicated-专属（自定义域+独立 issuer）/ shared-共享。
const TENANT_ISOLATION_OPTIONS = [
  { value: 'shared' as TenantIsolationMode, label: '共享（shared）' },
  { value: 'dedicated' as TenantIsolationMode, label: '专属（dedicated）' },
]

// 租户类型（后端 model.TenantType：customer-客户租户 / platform-平台租户）：
// 用于区分「外部客户租户」与「平台自运营租户」，当前仅作分类标识，
// 不参与数据隔离与权限判定（隔离一律按 tenant_id）。
const TENANT_TYPE_OPTIONS = [
  { value: 'customer', label: '客户租户', title: '外部客户/合作方的独立租户' },
  { value: 'platform', label: '平台租户', title: '平台自运营租户（种子数据“平台运营中心”即平台租户）' },
]

const TENANT_TYPE_TIP =
  '客户租户（customer）：外部客户/合作方的独立数据与权限边界；平台租户（platform）：平台自运营租户（如“平台运营中心”）。当前该字段仅作分类标识，不参与数据隔离与权限判定。'

/**
 * 平台自运营租户编码（后端 model.SeedPlatformTenantCode）：种子写入的平台控制台自身所在租户，
 * 既不可挂起（挂起即整栈失联），也不可删除。判定按编码而非 type——控制台可创建 type=platform 的
 * 普通租户，只有种子租户是产品内置。
 */
const PLATFORM_TENANT_CODE = 't_platform'

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
    form.setFieldsValue({ type: 'customer', isolationMode: 'shared' })
    setModalOpen(true)
  }

  const handleEdit = (record: TenantItem) => {
    setEditing(record)
    form.setFieldsValue({
      name: record.name,
      type: record.type,
      tag: record.tag,
      dbUser: record.dbUser,
      // D 编辑不提交 status（表单隐藏状态项）；shared 编辑仍可切正常/挂起。
      status: record.status,
    })
    setModalOpen(true)
  }

  const handleSubmit = async () => {
    try {
      const values = await form.validateFields()
      setSubmitLoading(true)
      if (editing) {
        // D 编辑不提交 status（状态只经 offboard/activate 动作流转）：从 payload 中剔除。
        const { status, ...rest } = values
        const payload = editing.isolationMode === 'dedicated' ? rest : { ...rest, status }
        await updateTenant({ tenantID: editing.tenantID, ...payload })
        message.success('修改成功')
      } else {
        const created = await createTenant({
          name: values.name,
          type: values.type,
          tag: values.tag,
          dbUser: values.dbUser,
          isolationMode: values.isolationMode,
          // D 新建带 primaryDomain（同事务创建 draft primary 域名）。
          primaryDomain: values.isolationMode === 'dedicated' ? values.primaryDomain : undefined,
          admin: {
            name: values.adminName,
            username: values.adminUsername,
            primaryEmail: values.adminPrimaryEmail,
            primaryPhone: values.adminPrimaryPhone,
          },
        })
        setModalOpen(false)
        void fetchData()
        if (created.delivered) {
          // 通道启用且投递成功：管理员不经手明文，仅提示已发送。
          message.success('创建成功')
          Modal.success({
            title: '租户已创建',
            content: '管理员初始密码已通过邮件/短信发送至账号本人，首次登录必须修改密码。若对方未收到，可在租户列表执行「重置管理员密码」重发。',
          })
        } else if (created.adminInitialPassword) {
          // 通道未启用或投递失败回退：明文仅此一次返回；该管理员首次登录必须改密
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
      if (resp.delivered) {
        Modal.success({
          title: '重置成功',
          content: '新密码已通过邮件/短信发送至管理员本人，下次登录必须修改密码。',
        })
      } else {
        // 通道未启用或投递失败回退：明文仅此一次展示
        setInitialPassword(resp.initialPassword)
      }
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

  /** offboard（D 唯一停用入口）：tenant→suspended、primary disabled、清 issuer、撤 session。 */
  const handleOffboard = (record: TenantItem) => {
    Modal.confirm({
      title: 'offboard 该租户？',
      content:
        'offboard 是该专属（D）租户的唯一停用入口：租户将置为「挂起」、primary 域名停用、issuer 清空、成员会话与刷新令牌撤销。该操作不可直接撤销（需重新激活域名与应用），确认继续？',
      okButtonProps: { danger: true },
      onOk: async () => {
        try {
          await offboardTenant(record.tenantID)
          message.success('已 offboard')
          void fetchData()
        } catch {
          /* 拦截器已提示 */
        }
      },
    })
  }

  const columns: ColumnsType<TenantItem> = [
    idColumn<TenantItem>({ dataIndex: 'tenantID' }),
    textColumn<TenantItem>({ title: '租户名', dataIndex: 'name', width: NAME_COL_WIDTH }),
    textColumn<TenantItem>({ title: '编码', dataIndex: 'code', width: CODE_COL_WIDTH, monospace: true }),
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
      width: TAG_COL_WIDTH,
      render: (v: string) => <TypeTag value={v} />,
    },
    textColumn<TenantItem>({ title: '标签', dataIndex: 'tag', width: TAG_COL_WIDTH }),
    {
      title: '状态',
      dataIndex: 'status',
      key: 'status',
      width: STATUS_COL_WIDTH,
      render: (v: string) => <TenantStatusTag value={v} />,
    },
    timeColumn<TenantItem>({ title: '创建时间', dataIndex: 'createdAt' }),
    timeColumn<TenantItem>({ title: '更新时间', dataIndex: 'updatedAt' }),
    // 3 个操作中「重置管理员密码」文案较长（7 字），降为横排 1 个 + 「更多」，
    // 操作列因此收窄到 150 且 fixed: 'right'，任何窗口下都无需横滑即可操作。
    actionColumn<TenantItem>({
      max: 2,
      variant: 'icon',
      actions: (r) => [
        { key: 'edit', label: '编辑', icon: <EditOutlined />, onClick: () => handleEdit(r) },
        {
          key: 'resetAdminPassword',
          label: '重置管理员密码',
          icon: <RollbackOutlined />,
          confirm: '将重置该租户内置管理员的密码：新临时密码仅展示一次，其既有会话立即失效。确认重置？',
          onClick: () => void handleResetAdminPassword(r),
        },
        // D（dedicated）租户：停用唯一入口是 offboard，不提供删除（后端 DELETE 一律 403）。
        {
          key: 'offboard',
          label: 'offboard',
          icon: <LogoutOutlined />,
          danger: true,
          hidden: r.isolationMode !== 'dedicated',
          onClick: () => handleOffboard(r),
        },
        {
          key: 'delete',
          label: '删除',
          danger: true,
          icon: <DeleteOutlined />,
          // D 租户不显示删除（只能 offboard）；平台自运营种子租户禁删（后端兜底 TenantBuiltInDeleteForbiddenError）。
          hidden: r.isolationMode === 'dedicated',
          disabled: r.code === PLATFORM_TENANT_CODE,
          confirm: '确认删除该租户？',
          onClick: () => void handleDelete(r),
        },
      ],
    }),
  ]

  // 平台自运营租户（种子租户 t_platform）不可挂起：它是平台控制台自身所在租户，挂起即整栈失联；
  // 与后端 svctenant.Update 的拒写规则同源。平台租户改名仍归运维（migrate_once 不覆盖自定义值）。
  const platformTenant = editing?.code === PLATFORM_TENANT_CODE

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
      summary={
        <StatRow>
          <StatCard label="租户总数" value={total} hint="当前筛选下返回总数" />
          <StatCard label="本页数量" value={data.length} hint={`第 ${page} 页 · 每页 ${pageSize} 条`} />
          <StatCard label="当前筛选" value={status ?? '全部'} hint={status ? '' : '未设置状态筛选'} />
        </StatRow>
      }
      toolbar={
        <Space wrap>
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
      }
    >
      <Table<TenantItem>
        rowKey="tenantID"
        columns={columns}
        dataSource={data}
        loading={loading}
        tableLayout="fixed"
        scroll={tableScrollX(columns)}
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
          {!editing ? (
            <>
              <Form.Item
                name="isolationMode"
                label="隔离模式"
                tooltip="专属（dedicated，D 租户）：自定义域名 + 独立 issuer；共享（shared）：中央 issuer。create-only，创建后不可改。"
              >
                <Select options={TENANT_ISOLATION_OPTIONS} />
              </Form.Item>
              <Form.Item
                noStyle
                shouldUpdate={(prev, cur) => prev.isolationMode !== cur.isolationMode}
              >
                {({ getFieldValue }) =>
                  getFieldValue('isolationMode') === 'dedicated' ? (
                    <Form.Item
                      name="primaryDomain"
                      label="主域名"
                      rules={[{ required: true, message: '专属租户必须指定主域名' }]}
                      extra="同事务创建该 primary（draft）；激活后作为独立 issuer 来源"
                    >
                      <Input placeholder="如：tenant.example.com" />
                    </Form.Item>
                  ) : null
                }
              </Form.Item>
            </>
          ) : (
            <Form.Item label="隔离模式" extra="隔离模式 create-only，创建后不可修改">
              <Input value={editing.isolationMode || 'shared'} disabled />
            </Form.Item>
          )}
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
              <Divider titlePlacement="left" plain>
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
          {/* D（dedicated）编辑不提交 status：状态只经 activate/offboard 动作流转，此处隐藏状态项。
              仅共享（shared）租户编辑保留「正常/挂起」开关。 */}
          {editing && editing.isolationMode !== 'dedicated' && (
            <Form.Item
              name="status"
              label="状态"
              tooltip={
                platformTenant
                  ? '平台自运营租户不可挂起（挂起后平台控制台会整体失联）'
                  : '挂起后该租户成员无法登录、已签发会话会被撤销；不能挂起你当前所在的租户'
              }
              valuePropName="checked"
              getValueFromEvent={(checked: boolean) => (checked ? 'active' : 'suspended')}
              getValueProps={(v?: string) => ({ checked: v === 'active' })}
            >
              <Switch checkedChildren="正常" unCheckedChildren="挂起" disabled={platformTenant} />
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
