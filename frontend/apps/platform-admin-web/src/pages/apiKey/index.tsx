import { useCallback, useEffect, useState, type CSSProperties, type ReactNode } from 'react'
import { Alert, Button, DatePicker, Form, Input, Modal, Popconfirm, Space, Table, Tabs, Tag, message } from 'antd'
import type { ColumnsType } from 'antd/es/table'
import { CopyOutlined, PlusOutlined, ReloadOutlined, SearchOutlined } from '@ant-design/icons'
import { EllipsisCell, fmtTime, IDCell, PageContainer, tokens } from '@ark-iam/ui'
import { createApiKey, deleteApiKey, getApiKeyPageList, getApiKeySupervisionPageList, revokeApiKey } from '@ark-iam/api'
import type { ApiKeyCreateResp, ApiKeyItem, ApiKeySupervisionItem } from '@ark-iam/types'

interface ApiKeyFormValues {
  name: string
  scope?: string
  expiresAt?: { valueOf: () => number }
}

const monospaceStyle: CSSProperties = { fontFamily: 'Consolas, Monaco, monospace' }

// 统一的状态推导：已吊销 > 已过期 > 有效
function keyStateOf(r: { revokedAt: number; expiresAt: number }): { label: string; color: string } {
  if (r.revokedAt) return { label: '已吊销', color: 'error' }
  if (r.expiresAt > 0 && r.expiresAt * 1000 <= Date.now()) return { label: '已过期', color: 'warning' }
  return { label: '有效', color: 'success' }
}

function KeyStateTag(r: { revokedAt: number; expiresAt: number }) {
  const st = keyStateOf(r)
  return <Tag color={st.color}>{st.label}</Tag>
}

function ToolbarRow({
  searchPlaceholder,
  onSearch,
  onRefresh,
  extra,
}: {
  searchPlaceholder: string
  onSearch: (v: string) => void
  onRefresh: () => void
  extra?: ReactNode
}) {
  const [draft, setDraft] = useState('')
  return (
    <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 16, flexWrap: 'wrap', gap: 8 }}>
      <Input.Search
        allowClear
        placeholder={searchPlaceholder}
        prefix={<SearchOutlined />}
        style={{ width: 240 }}
        value={draft}
        onChange={(e) => {
          const v = e.target.value
          setDraft(v)
          // 清空搜索框时立即回到全量列表
          if (v === '') onSearch('')
        }}
        onSearch={onSearch}
      />
      <Space>
        <Button icon={<ReloadOutlined />} onClick={onRefresh}>
          刷新
        </Button>
        {extra}
      </Space>
    </div>
  )
}

// ==================== 视角一：本租户（平台租户）密钥管理 ====================
function OwnedApiKeysPane() {
  const [data, setData] = useState<ApiKeyItem[]>([])
  const [loading, setLoading] = useState(false)
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(10)
  const [total, setTotal] = useState(0)
  const [keyword, setKeyword] = useState('')

  const [modalOpen, setModalOpen] = useState(false)
  const [form] = Form.useForm<ApiKeyFormValues>()
  const [submitLoading, setSubmitLoading] = useState(false)

  const [createdKey, setCreatedKey] = useState<ApiKeyCreateResp | null>(null)
  const [keyModalOpen, setKeyModalOpen] = useState(false)

  const fetchData = useCallback(async () => {
    setLoading(true)
    try {
      const resp = await getApiKeyPageList({ page, pageSize, name: keyword })
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
    form.resetFields()
    setModalOpen(true)
  }

  const handleSubmit = async () => {
    try {
      const values = await form.validateFields()
      setSubmitLoading(true)
      const resp = await createApiKey({
        name: values.name,
        scope: values.scope || undefined,
        expiresAt: values.expiresAt ? Math.floor(values.expiresAt.valueOf() / 1000) : undefined,
      })
      message.success('创建成功')
      setModalOpen(false)
      setCreatedKey(resp)
      setKeyModalOpen(true)
      void fetchData()
    } catch {
      /* 校验或请求失败，拦截器已提示 */
    } finally {
      setSubmitLoading(false)
    }
  }

  const handleCopyKey = async () => {
    if (!createdKey) return
    try {
      await navigator.clipboard.writeText(createdKey.key)
      message.success('已复制')
    } catch {
      message.error('复制失败，请手动复制')
    }
  }

  const handleRevoke = async (record: ApiKeyItem) => {
    try {
      await revokeApiKey(record.id)
      message.success('已吊销')
      void fetchData()
    } catch {
      /* 拦截器已提示 */
    }
  }

  const handleDelete = async (record: ApiKeyItem) => {
    try {
      await deleteApiKey(record.id)
      message.success('删除成功')
      void fetchData()
    } catch {
      /* 拦截器已提示 */
    }
  }

  const columns: ColumnsType<ApiKeyItem> = [
    { title: 'ID', dataIndex: 'id', key: 'id', width: 150, render: (v: string) => <IDCell value={v} /> },
    { title: '名称', dataIndex: 'name', key: 'name', render: (v: string) => <EllipsisCell value={v} /> },
    {
      title: '前缀',
      dataIndex: 'keyPrefix',
      key: 'keyPrefix',
      width: 180,
      render: (v: string) => <span style={{ ...monospaceStyle, fontSize: 12 }}>{v || '-'}</span>,
    },
    { title: '范围', dataIndex: 'scope', key: 'scope', render: (v: string) => <EllipsisCell value={v} /> },
    { title: '过期时间', dataIndex: 'expiresAt', key: 'expiresAt', width: 160, render: (v: number) => fmtTime(v) },
    { title: '最后使用', dataIndex: 'lastUsedAt', key: 'lastUsedAt', width: 160, render: (v: number) => fmtTime(v) },
    {
      title: '状态',
      key: 'status',
      width: 90,
      render: (_, r) => KeyStateTag(r),
    },
    { title: '创建时间', dataIndex: 'createdAt', key: 'createdAt', width: 160, render: (v: number) => fmtTime(v) },
    {
      title: '操作',
      key: 'action',
      width: 150,
      render: (_, r) => (
        <Space size={4}>
          {!r.revokedAt && (
            <Popconfirm title="确认吊销该 API Key？吊销后立即失效" onConfirm={() => void handleRevoke(r)}>
              <Button type="link" size="small" danger>
                吊销
              </Button>
            </Popconfirm>
          )}
          <Popconfirm title="确认删除该 API Key？" onConfirm={() => void handleDelete(r)}>
            <Button type="link" size="small" danger>
              删除
            </Button>
          </Popconfirm>
        </Space>
      ),
    },
  ]

  return (
    <>
      <ToolbarRow
        searchPlaceholder="按名称搜索"
        onSearch={(v) => {
          setKeyword(v)
          setPage(1)
        }}
        onRefresh={() => void fetchData()}
        extra={
          <Button type="primary" icon={<PlusOutlined />} onClick={handleCreate}>
            新建 API Key
          </Button>
        }
      />
      <Table<ApiKeyItem>
        rowKey="id"
        columns={columns}
        dataSource={data}
        loading={loading}
        scroll={{ x: 1100 }}
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
        title="新建 API Key"
        open={modalOpen}
        onOk={() => void handleSubmit()}
        onCancel={() => setModalOpen(false)}
        confirmLoading={submitLoading}
        destroyOnClose
        width={520}
      >
        <Form form={form} layout="vertical">
          <Form.Item name="name" label="名称" rules={[{ required: true, message: '请输入名称' }]}>
            <Input placeholder="如：CI/CD 部署凭证" />
          </Form.Item>
          <Form.Item name="scope" label="范围">
            <Input placeholder="选填，如 read:user write:org" />
          </Form.Item>
          <Form.Item name="expiresAt" label="过期时间">
            <DatePicker showTime style={{ width: '100%' }} placeholder="选填，留空表示永不过期" />
          </Form.Item>
        </Form>
      </Modal>

      <Modal
        title="API Key 创建成功"
        open={keyModalOpen}
        onCancel={() => setKeyModalOpen(false)}
        footer={
          <Button type="primary" onClick={() => setKeyModalOpen(false)}>
            我已保存
          </Button>
        }
        destroyOnClose
        width={560}
      >
        <Alert
          type="warning"
          showIcon
          message="请立即保存，关闭后不再显示"
          description={
            <Space direction="vertical" size={12} style={{ width: '100%' }}>
              <div
                style={{
                  ...monospaceStyle,
                  wordBreak: 'break-all',
                  background: tokens.warningBg,
                  border: `1px solid ${tokens.warningBorder}`,
                  borderRadius: 8,
                  padding: '10px 12px',
                }}
              >
                {createdKey?.key}
              </div>
              <Button size="small" icon={<CopyOutlined />} onClick={() => void handleCopyKey()}>
                复制
              </Button>
            </Space>
          }
        />
      </Modal>
    </>
  )
}

// ==================== 视角二：全租户只读监督（平台排查） ====================
function SupervisionApiKeysPane() {
  const [data, setData] = useState<ApiKeySupervisionItem[]>([])
  const [loading, setLoading] = useState(false)
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(10)
  const [total, setTotal] = useState(0)
  const [keyword, setKeyword] = useState('')

  const fetchData = useCallback(async () => {
    setLoading(true)
    try {
      const resp = await getApiKeySupervisionPageList({ page, pageSize, name: keyword })
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

  const columns: ColumnsType<ApiKeySupervisionItem> = [
    { title: '租户', dataIndex: 'tenantName', key: 'tenantName', width: 160, render: (v: string, r) => <EllipsisCell value={v || r.tenantID} /> },
    { title: '租户ID', dataIndex: 'tenantID', key: 'tenantID', width: 150, render: (v: string) => <IDCell value={v} /> },
    { title: '名称', dataIndex: 'name', key: 'name', render: (v: string) => <EllipsisCell value={v} /> },
    {
      title: '前缀',
      dataIndex: 'keyPrefix',
      key: 'keyPrefix',
      width: 180,
      render: (v: string) => <span style={{ ...monospaceStyle, fontSize: 12 }}>{v || '-'}</span>,
    },
    { title: '创建人', dataIndex: 'creatorName', key: 'creatorName', width: 140, render: (v: string, r) => <EllipsisCell value={v || r.createdBy} /> },
    {
      title: '状态',
      key: 'status',
      width: 90,
      render: (_, r) => KeyStateTag(r),
    },
    { title: '最后使用', dataIndex: 'lastUsedAt', key: 'lastUsedAt', width: 160, render: (v: number) => fmtTime(v) },
    { title: '过期时间', dataIndex: 'expiresAt', key: 'expiresAt', width: 160, render: (v: number) => fmtTime(v) },
    { title: '创建时间', dataIndex: 'createdAt', key: 'createdAt', width: 160, render: (v: number) => fmtTime(v) },
  ]

  return (
    <>
      <Alert
        type="info"
        showIcon
        style={{ marginBottom: 16 }}
        message="平台全租户只读监督视图"
        description="用于密钥泄漏/僵尸密钥排查：明文密钥永不可见（仅展示前缀），本视图不提供吊销与删除（平台应急吊销暂未纳入）。若发现可疑密钥请联系对应租户管理员处理。"
      />
      <ToolbarRow
        searchPlaceholder="按密钥名称搜索"
        onSearch={(v) => {
          setKeyword(v)
          setPage(1)
        }}
        onRefresh={() => void fetchData()}
      />
      <Table<ApiKeySupervisionItem>
        rowKey="id"
        columns={columns}
        dataSource={data}
        loading={loading}
        scroll={{ x: 1200 }}
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
    </>
  )
}

export default function ApiKeyList() {
  return (
    <PageContainer title="API 密钥" description="机器访问凭证：本租户密钥管理；全租户监督为平台只读排查视图">
      <Tabs
        defaultActiveKey="owned"
        items={[
          { key: 'owned', label: '本租户密钥', children: <OwnedApiKeysPane /> },
          { key: 'supervision', label: '全租户监督', children: <SupervisionApiKeysPane /> },
        ]}
      />
    </PageContainer>
  )
}
