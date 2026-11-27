import { useCallback, useEffect, useState } from 'react'
import { Button, Form, Input, Modal, Select, Space, Table, message } from 'antd'
import {
  CheckCircleOutlined,
  DeleteOutlined,
  LockOutlined,
  PlusOutlined,
  ReloadOutlined,
  RollbackOutlined,
  SafetyCertificateOutlined,
  SearchOutlined,
  StopOutlined,
  ThunderboltOutlined,
} from '@ant-design/icons'
import type { ColumnsType } from 'antd/es/table'
import {
  actionColumn,
  CODE_COL_WIDTH,
  DomainStatusTag,
  idColumn,
  PageContainer,
  StatCard,
  StatRow,
  STATUS_COL_WIDTH,
  TAG_COL_WIDTH,
  tableScrollX,
  textColumn,
  timeColumn,
  TypeTag,
} from '@ark-iam/ui'
import {
  activateDomain,
  createDomain,
  deleteDomain,
  disableDomain,
  enableDomain,
  getDomainPageList,
  releaseDomain,
  retireDomain,
  tlsReadyDomain,
  verifyDomain,
} from '@ark-iam/api'
import type {
  DomainItem,
  DomainRole,
  DomainStatus,
  DomainUsage,
  DomainVerificationMethod,
} from '@ark-iam/types'

// 域名状态筛选项（后端 model.DomainStatus 7 态）。
const DOMAIN_STATUS_OPTIONS: { value: DomainStatus; label: string }[] = [
  { value: 'draft', label: '草稿' },
  { value: 'verified', label: '已验证' },
  { value: 'tls_ready', label: '证书就绪' },
  { value: 'active', label: '启用' },
  { value: 'disabled', label: '停用' },
  { value: 'retired', label: '已退役' },
  { value: 'released', label: '已释放' },
]

// 域名角色（后端 model.DomainRole）：branded 仅 P1/P2 预留，创建表单不提供。
const DOMAIN_ROLE_OPTIONS: { value: DomainRole; label: string }[] = [
  { value: 'primary', label: '主域（primary）' },
  { value: 'alias', label: '别名（alias）' },
]

// 域名用途（后端 model.DomainUsage）。
const DOMAIN_USAGE_OPTIONS: { value: DomainUsage; label: string }[] = [
  { value: 'login', label: '登录（login）' },
  { value: 'email', label: '邮箱（email）' },
  { value: 'both', label: '登录 + 邮箱（both）' },
]

// 域控证明方式（后端 model.DomainVerificationMethod）。
const DOMAIN_METHOD_OPTIONS: { value: DomainVerificationMethod; label: string }[] = [
  { value: 'manual', label: '人工留证（manual）' },
  { value: 'dns_txt', label: 'DNS TXT（dns_txt）' },
]

/**
 * 按状态机推导某域名**当前可执行**的动作集合（与后端 pkg/core/tenantdomain 动作端点一一对应，
 * 前端只做「按钮可见性」的乐观裁剪，真正校验以后端为准）：
 * - draft → verify；
 * - verified → tls-ready；alias/email-only 可跳过 tls_ready 直接 activate；
 * - tls_ready → activate；
 * - active（alias）→ disable；（primary 不可直接 disable，只能走 offboard）；
 * - disabled（alias）→ enable / retire；（primary disabled → retire，offboard 之后）；
 * - retired → release（需审批单）；
 * - released / disabled 之外的终态无动作。
 *
 * 注意：由于动作数量会随状态变化（≤3），交由 actionColumn 在行内横排，超过 3 个自动收「更多」。
 */
function domainActions(r: DomainItem): {
  canVerify: boolean
  canTLSReady: boolean
  canActivate: boolean
  canDisable: boolean
  canEnable: boolean
  canRetire: boolean
  canRelease: boolean
} {
  const isAlias = r.domainRole === 'alias'
  const isPrimary = r.domainRole === 'primary'
  const emailOnlyAlias = isAlias && r.domainUsage === 'email'
  return {
    canVerify: r.status === 'draft',
    canTLSReady: r.status === 'verified' && !emailOnlyAlias,
    // primary/alias(login/both) 从 tls_ready 激活；email-only alias 从 verified 直接激活。
    canActivate: r.status === 'tls_ready' || (emailOnlyAlias && r.status === 'verified'),
    // 仅 alias active 可 disable；primary 只能走 offboard。
    canDisable: isAlias && r.status === 'active',
    canEnable: isAlias && r.status === 'disabled',
    // alias active/disabled → retire；primary disabled（offboard 后）→ retire。
    canRetire: (isAlias && (r.status === 'active' || r.status === 'disabled')) || (isPrimary && r.status === 'disabled'),
    canRelease: r.status === 'retired',
  }
}

export default function DomainList() {
  const [data, setData] = useState<DomainItem[]>([])
  const [loading, setLoading] = useState(false)
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(10)
  const [total, setTotal] = useState(0)
  const [keyword, setKeyword] = useState('')
  const [tenantIDFilter, setTenantIDFilter] = useState<string | undefined>(undefined)
  const [statusFilter, setStatusFilter] = useState<DomainStatus | undefined>(undefined)

  const [modalOpen, setModalOpen] = useState(false)
  const [form] = Form.useForm()
  const [submitLoading, setSubmitLoading] = useState(false)

  const fetchData = useCallback(async () => {
    setLoading(true)
    try {
      const resp = await getDomainPageList({ page, pageSize, domain: keyword, tenantID: tenantIDFilter, status: statusFilter })
      setData(resp?.list || [])
      setTotal(resp?.total || 0)
    } catch {
      /* 拦截器已提示 */
    } finally {
      setLoading(false)
    }
  }, [page, pageSize, keyword, tenantIDFilter, statusFilter])

  useEffect(() => {
    void fetchData()
  }, [fetchData])

  const handleCreate = () => {
    form.resetFields()
    form.setFieldsValue({ domainRole: 'primary', domainUsage: 'login', verificationMethod: 'manual' })
    setModalOpen(true)
  }

  const handleSubmit = async () => {
    try {
      const values = await form.validateFields()
      setSubmitLoading(true)
      await createDomain({
        tenantID: values.tenantID,
        domain: values.domain,
        domainRole: values.domainRole,
        domainUsage: values.domainUsage,
        targetDomain: values.targetDomain,
        verificationMethod: values.verificationMethod,
      })
      message.success('创建成功')
      setModalOpen(false)
      void fetchData()
    } catch {
      /* 校验或请求失败 */
    } finally {
      setSubmitLoading(false)
    }
  }

  /** 统一动作执行器：调端点、回显、刷新。 */
  const runAction = async (_key: string, fn: () => Promise<unknown>, okMsg: string) => {
    try {
      await fn()
      message.success(okMsg)
      void fetchData()
    } catch {
      /* 拦截器已提示 */
    }
  }

  const handleVerify = (r: DomainItem) =>
    runAction(`verify:${r.id}`, () => verifyDomain(r.id, { method: 'manual' }), '已验证')

  const handleTLSReady = (r: DomainItem) =>
    runAction(`tls:${r.id}`, () => tlsReadyDomain(r.id), '证书已就绪')

  const handleActivate = (r: DomainItem) =>
    runAction(`activate:${r.id}`, () => activateDomain(r.id), '已激活')

  const handleDisable = (r: DomainItem) =>
    runAction(`disable:${r.id}`, () => disableDomain(r.id), '已停用')

  const handleEnable = (r: DomainItem) =>
    runAction(`enable:${r.id}`, () => enableDomain(r.id), '已恢复')

  const handleRetire = (r: DomainItem) => {
    let reason = ''
    Modal.confirm({
      title: '退役该域名？',
      content: (
        <Input.TextArea
          rows={2}
          placeholder="退役原因（可选）"
          onChange={(e) => {
            reason = e.target.value
          }}
        />
      ),
      okButtonProps: { danger: true },
      onOk: () => runAction(`retire:${r.id}`, () => retireDomain(r.id, reason), '已退役'),
    })
  }

  const handleRelease = (r: DomainItem) => {
    let ticket = ''
    Modal.confirm({
      title: '正式释放该域名？',
      content: (
        <Space direction="vertical" style={{ width: '100%' }}>
          <span>释放需满 90 天隔离期且有审批单，释放后 host 可被重新登记（终态）。</span>
          <Input
            placeholder="审批单号（必填）"
            onChange={(e) => {
              ticket = e.target.value
            }}
          />
        </Space>
      ),
      okButtonProps: { danger: true },
      onOk: async () => {
        if (!ticket.trim()) {
          message.error('请填写审批单号')
          return Promise.reject(new Error('missing ticket'))
        }
        await runAction(`release:${r.id}`, () => releaseDomain(r.id, ticket.trim()), '已释放')
      },
    })
  }

  const handleDelete = async (r: DomainItem) => {
    try {
      await deleteDomain(r.id)
      message.success('删除成功')
      void fetchData()
    } catch {
      /* 拦截器已提示 */
    }
  }

  const columns: ColumnsType<DomainItem> = [
    idColumn<DomainItem>({ dataIndex: 'id' }),
    textColumn<DomainItem>({ title: '域名', dataIndex: 'domain', width: CODE_COL_WIDTH, monospace: true }),
    {
      title: '角色',
      dataIndex: 'domainRole',
      key: 'domainRole',
      width: TAG_COL_WIDTH,
      render: (v: DomainRole) => <TypeTag value={v === 'primary' ? 'platform' : 'customer'} />,
    },
    {
      title: '用途',
      dataIndex: 'domainUsage',
      key: 'domainUsage',
      width: STATUS_COL_WIDTH,
      render: (v: DomainUsage) => <span>{DOMAIN_USAGE_OPTIONS.find((o) => o.value === v)?.label.split('（')[0] ?? v}</span>,
    },
    {
      title: '状态',
      dataIndex: 'status',
      key: 'status',
      width: STATUS_COL_WIDTH,
      render: (v: DomainStatus) => <DomainStatusTag value={v} />,
    },
    timeColumn<DomainItem>({ title: '创建时间', dataIndex: 'createdAt' }),
    timeColumn<DomainItem>({ title: '更新时间', dataIndex: 'updatedAt' }),
    // 状态机驱动的动作按钮：按当前状态裁剪可见动作（hidden），由 actionColumn 统一横排/收「更多」。
    // draft/verified 阶段可同时「删除」（后端仅允许 draft/verified/tls_ready 删除）。
    actionColumn<DomainItem>({
      max: 3,
      variant: 'icon',
      actions: (r) => {
        const a = domainActions(r)
        return [
          { key: 'verify', label: '验证', icon: <CheckCircleOutlined />, hidden: !a.canVerify, onClick: () => void handleVerify(r) },
          { key: 'tlsReady', label: '证书就绪', icon: <SafetyCertificateOutlined />, hidden: !a.canTLSReady, onClick: () => void handleTLSReady(r) },
          { key: 'activate', label: '激活', icon: <ThunderboltOutlined />, hidden: !a.canActivate, onClick: () => void handleActivate(r) },
          { key: 'disable', label: '停用', icon: <StopOutlined />, hidden: !a.canDisable, onClick: () => void handleDisable(r) },
          { key: 'enable', label: '恢复', icon: <CheckCircleOutlined />, hidden: !a.canEnable, onClick: () => void handleEnable(r) },
          { key: 'retire', label: '退役', icon: <LockOutlined />, hidden: !a.canRetire, danger: true, onClick: () => handleRetire(r) },
          { key: 'release', label: '释放', icon: <RollbackOutlined />, hidden: !a.canRelease, danger: true, onClick: () => handleRelease(r) },
          {
            key: 'delete',
            label: '删除',
            icon: <DeleteOutlined />,
            danger: true,
            // 仅 draft/verified/tls_ready 可删除（active 必须 disable/offboard，retired/released 不可删）。
            hidden: !(r.status === 'draft' || r.status === 'verified' || r.status === 'tls_ready'),
            confirm: '确认删除该域名？',
            onClick: () => void handleDelete(r),
          },
        ]
      },
    }),
  ]

  return (
    <PageContainer
      title="域名管理"
      description="管理租户自定义域名，按 7 态状态机流转（验证 / 证书 / 激活 / 停用 / 退役 / 释放）"
      extra={
        <Space>
          <Button icon={<ReloadOutlined />} onClick={() => void fetchData()}>
            刷新
          </Button>
          <Button type="primary" icon={<PlusOutlined />} onClick={handleCreate}>
            新建域名
          </Button>
        </Space>
      }
      summary={
        <StatRow>
          <StatCard label="域名总数" value={total} hint="当前筛选下返回总数" />
          <StatCard label="本页数量" value={data.length} hint={`第 ${page} 页 · 每页 ${pageSize} 条`} />
          <StatCard label="当前状态" value={statusFilter ?? '全部'} hint={statusFilter ? '' : '未设置状态筛选'} />
        </StatRow>
      }
      toolbar={
        <Space wrap>
          <Input.Search
            allowClear
            placeholder="按域名搜索"
            prefix={<SearchOutlined />}
            style={{ width: 220 }}
            onSearch={(v) => {
              setKeyword(v)
              setPage(1)
            }}
          />
          <Input
            allowClear
            placeholder="按租户 ID 过滤"
            style={{ width: 220 }}
            onChange={(e) => {
              setTenantIDFilter(e.target.value || undefined)
              setPage(1)
            }}
          />
          <Select
            allowClear
            placeholder="全部状态"
            style={{ width: 140 }}
            value={statusFilter}
            onChange={(v: DomainStatus | undefined) => {
              setStatusFilter(v)
              setPage(1)
            }}
            options={DOMAIN_STATUS_OPTIONS}
          />
        </Space>
      }
    >
      <Table<DomainItem>
        rowKey="id"
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
        title="新建域名"
        open={modalOpen}
        onOk={() => void handleSubmit()}
        onCancel={() => setModalOpen(false)}
        confirmLoading={submitLoading}
        destroyOnClose
        width={520}
      >
        <Form form={form} layout="vertical">
          <Form.Item name="tenantID" label="所属租户 ID" rules={[{ required: true, message: '请输入租户 ID' }]}>
            <Input placeholder="目标租户 ID（UUID）" />
          </Form.Item>
          <Form.Item name="domain" label="域名" rules={[{ required: true, message: '请输入域名' }]}>
            <Input placeholder="如：example.com" />
          </Form.Item>
          <Form.Item name="domainRole" label="角色" rules={[{ required: true }]}>
            <Select options={DOMAIN_ROLE_OPTIONS} />
          </Form.Item>
          <Form.Item
            name="domainUsage"
            label="用途"
            rules={[{ required: true }]}
            tooltip="primary 只允许 login/both；alias/email 需指定 targetDomain"
          >
            <Select options={DOMAIN_USAGE_OPTIONS} />
          </Form.Item>
          <Form.Item
            noStyle
            shouldUpdate={(prev, cur) => prev.domainRole !== cur.domainRole}
          >
            {({ getFieldValue }) =>
              getFieldValue('domainRole') === 'alias' ? (
                <Form.Item
                  name="targetDomain"
                  label="目标主域 host"
                  rules={[{ required: true, message: '别名域名必须指定目标 primary 的 host' }]}
                  extra="同租户 primary 的 host，服务端解析为 target_domain_id"
                >
                  <Input placeholder="如：primary.example.com" />
                </Form.Item>
              ) : null
            }
          </Form.Item>
          <Form.Item name="verificationMethod" label="验证方式" initialValue="manual">
            <Select options={DOMAIN_METHOD_OPTIONS} />
          </Form.Item>
        </Form>
      </Modal>
    </PageContainer>
  )
}
