import { useCallback, useEffect, useState, type CSSProperties, type ReactNode } from 'react'
import { Alert, Button, DatePicker, Form, Input, Modal, Popconfirm, Select, Space, Table, Tabs, message } from 'antd'
import type { ColumnsType } from 'antd/es/table'
import { CopyOutlined, PlusOutlined, ReloadOutlined, SearchOutlined } from '@ant-design/icons'
import { EllipsisCell, fmtTime, PageContainer, tokens } from '@ark-iam/ui'
import type { PageListResp, TenantApiKeyCreateResp, TenantApiKeyItem, TenantMachineUserItem } from '@ark-iam/types'
import { createApiKey, deleteApiKey, getApiKeyPageList, revokeApiKey } from '../../api/apiKey'
import { getMachineUserPageList } from '../../api/machineUser'
import { KeyStateTag } from './KeyState'

interface ApiKeyFormValues {
  name: string
  machineUserID?: string
  expiresAt?: { valueOf: () => number }
}

const monospaceStyle: CSSProperties = { fontFamily: 'Consolas, Monaco, monospace' }

/** 状态推导（已吊销 > 已过期 > 有效），复用 KeyState 模块 */
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

/** 创建成功的一次性明文密钥展示 Modal（关闭后不再显示） */
function CreatedKeyModal({
  created,
  onClose,
}: {
  created: TenantApiKeyCreateResp | null
  onClose: () => void
}) {
  const handleCopy = async () => {
    if (!created) return
    try {
      await navigator.clipboard.writeText(created.key)
      message.success('已复制')
    } catch {
      message.error('复制失败，请手动复制')
    }
  }
  return (
    <Modal
      title="API 密钥创建成功"
      open={!!created}
      onCancel={onClose}
      footer={
        <Button type="primary" onClick={onClose}>
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
              {created?.key}
            </div>
            <Button size="small" icon={<CopyOutlined />} onClick={() => void handleCopy()}>
              复制
            </Button>
          </Space>
        }
      />
    </Modal>
  )
}

// ==================== Tab 一：我的密钥（代表用户本人） ====================
function MyKeysPane() {
  const [data, setData] = useState<TenantApiKeyItem[]>([])
  const [loading, setLoading] = useState(false)
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(10)
  const [total, setTotal] = useState(0)
  const [keyword, setKeyword] = useState('')

  const [modalOpen, setModalOpen] = useState(false)
  const [form] = Form.useForm<ApiKeyFormValues>()
  const [submitLoading, setSubmitLoading] = useState(false)
  const [created, setCreated] = useState<TenantApiKeyCreateResp | null>(null)

  const fetchData = useCallback(async () => {
    setLoading(true)
    try {
      const resp = await getApiKeyPageList({ page, pageSize, name: keyword || undefined })
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

  const handleSubmit = async () => {
    try {
      const values = await form.validateFields()
      setSubmitLoading(true)
      const resp = await createApiKey({
        name: values.name,
        expiredAt: values.expiresAt ? Math.floor(values.expiresAt.valueOf() / 1000) : undefined,
      })
      message.success('创建成功')
      setModalOpen(false)
      setCreated(resp)
      void fetchData()
    } catch {
      /* 校验或请求失败，拦截器已提示 */
    } finally {
      setSubmitLoading(false)
    }
  }

  const handleRevoke = async (record: TenantApiKeyItem) => {
    try {
      await revokeApiKey(record.keyID)
      message.success('已吊销')
      void fetchData()
    } catch {
      /* 拦截器已提示 */
    }
  }

  const handleDelete = async (record: TenantApiKeyItem) => {
    try {
      await deleteApiKey(record.keyID)
      message.success('删除成功')
      void fetchData()
    } catch {
      /* 拦截器已提示 */
    }
  }

  const columns: ColumnsType<TenantApiKeyItem> = [
    { title: '名称', dataIndex: 'name', key: 'name', render: (v: string) => <EllipsisCell value={v} /> },
    {
      title: '前缀',
      dataIndex: 'keyPrefix',
      key: 'keyPrefix',
      width: 180,
      render: (v: string) => <span style={{ ...monospaceStyle, fontSize: 12 }}>{v || '-'}</span>,
    },
    { title: '状态', key: 'status', width: 90, render: (_: unknown, r) => <KeyStateTag {...r} /> },
    { title: '过期时间', dataIndex: 'expiredAt', key: 'expiredAt', width: 160, render: (v: number | null) => fmtTime(v) },
    { title: '最近使用', dataIndex: 'lastUsedAt', key: 'lastUsedAt', width: 160, render: (v: number | null) => fmtTime(v) },
    { title: '创建时间', dataIndex: 'createdAt', key: 'createdAt', width: 160, render: (v: number) => fmtTime(v) },
    {
      title: '操作',
      key: 'action',
      width: 160,
      render: (_, r) => (
        <Space size={4}>
          {!r.revokedAt && (
            <Popconfirm title="确认吊销该密钥？吊销后立即失效" onConfirm={() => void handleRevoke(r)}>
              <Button type="link" size="small" danger>
                吊销
              </Button>
            </Popconfirm>
          )}
          <Popconfirm title="确认删除该密钥？" onConfirm={() => void handleDelete(r)}>
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
        searchPlaceholder="按密钥名称搜索"
        onSearch={(v) => {
          setKeyword(v)
          setPage(1)
        }}
        onRefresh={() => void fetchData()}
        extra={
          <Button
            type="primary"
            icon={<PlusOutlined />}
            onClick={() => {
              form.resetFields()
              setModalOpen(true)
            }}
          >
            新建 API 密钥
          </Button>
        }
      />
      <Table<TenantApiKeyItem>
        rowKey="keyID"
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

      <Modal
        title="新建 API 密钥"
        open={modalOpen}
        onOk={() => void handleSubmit()}
        onCancel={() => setModalOpen(false)}
        confirmLoading={submitLoading}
        destroyOnClose
        width={520}
      >
        <Form form={form} layout="vertical">
          <div style={{ marginBottom: 16, color: tokens.textPlaceholder, fontSize: 12 }}>归属为当前用户本人，代表你本人调用平台 API。</div>
          <Form.Item name="name" label="名称" rules={[{ required: true, message: '请输入名称' }]}>
            <Input placeholder="如：我的命令行凭证" />
          </Form.Item>
          <Form.Item name="expiresAt" label="过期时间">
            <DatePicker showTime style={{ width: '100%' }} placeholder="选填，留空表示永不过期" />
          </Form.Item>
        </Form>
      </Modal>

      <CreatedKeyModal created={created} onClose={() => setCreated(null)} />
    </>
  )
}

// ==================== Tab 二：服务账号密钥（管理） ====================
function MachineKeysPane() {
  const [data, setData] = useState<TenantApiKeyItem[]>([])
  const [loading, setLoading] = useState(false)
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(10)
  const [total, setTotal] = useState(0)
  const [keyword, setKeyword] = useState('')
  const [machineUserID, setMachineUserID] = useState<string>()

  const [machineOptions, setMachineOptions] = useState<TenantMachineUserItem[]>([])

  const [modalOpen, setModalOpen] = useState(false)
  const [form] = Form.useForm<ApiKeyFormValues>()
  const [submitLoading, setSubmitLoading] = useState(false)
  const [created, setCreated] = useState<TenantApiKeyCreateResp | null>(null)

  useEffect(() => {
    getMachineUserPageList({ page: 1, pageSize: 100 })
      .then((resp) => setMachineOptions(resp?.list || []))
      .catch(() => {})
  }, [])

  const fetchData = useCallback(async () => {
    if (!machineUserID) return
    setLoading(true)
    try {
      const resp: PageListResp<TenantApiKeyItem> = await getApiKeyPageList({
        page,
        pageSize,
        name: keyword || undefined,
        machineUserID,
      })
      setData(resp?.list || [])
      setTotal(resp?.total || 0)
    } catch {
      /* 拦截器已提示 */
    } finally {
      setLoading(false)
    }
  }, [page, pageSize, keyword, machineUserID])

  useEffect(() => {
    void fetchData()
  }, [fetchData])

  const handleSubmit = async () => {
    try {
      const values = await form.validateFields()
      setSubmitLoading(true)
      const resp = await createApiKey({
        name: values.name,
        machineUserID: values.machineUserID,
        expiredAt: values.expiresAt ? Math.floor(values.expiresAt.valueOf() / 1000) : undefined,
      })
      message.success('创建成功')
      setModalOpen(false)
      setCreated(resp)
      void fetchData()
    } catch {
      /* 校验或请求失败，拦截器已提示 */
    } finally {
      setSubmitLoading(false)
    }
  }

  const handleRevoke = async (record: TenantApiKeyItem) => {
    try {
      await revokeApiKey(record.keyID)
      message.success('已吊销')
      void fetchData()
    } catch {
      /* 拦截器已提示 */
    }
  }

  const handleDelete = async (record: TenantApiKeyItem) => {
    try {
      await deleteApiKey(record.keyID)
      message.success('删除成功')
      void fetchData()
    } catch {
      /* 拦截器已提示 */
    }
  }

  const columns: ColumnsType<TenantApiKeyItem> = [
    { title: '名称', dataIndex: 'name', key: 'name', render: (v: string) => <EllipsisCell value={v} /> },
    {
      title: '前缀',
      dataIndex: 'keyPrefix',
      key: 'keyPrefix',
      width: 180,
      render: (v: string) => <span style={{ ...monospaceStyle, fontSize: 12 }}>{v || '-'}</span>,
    },
    { title: '归属', dataIndex: 'ownerName', key: 'ownerName', width: 150, render: (v: string, r) => `${r.ownerType === 'machine' ? '服务账号 · ' : ''}${v || '-'}` },
    { title: '状态', key: 'status', width: 90, render: (_: unknown, r) => <KeyStateTag {...r} /> },
    { title: '过期时间', dataIndex: 'expiredAt', key: 'expiredAt', width: 160, render: (v: number | null) => fmtTime(v) },
    { title: '最近使用', dataIndex: 'lastUsedAt', key: 'lastUsedAt', width: 160, render: (v: number | null) => fmtTime(v) },
    { title: '创建时间', dataIndex: 'createdAt', key: 'createdAt', width: 160, render: (v: number) => fmtTime(v) },
    {
      title: '操作',
      key: 'action',
      width: 160,
      render: (_, r) => (
        <Space size={4}>
          {!r.revokedAt && (
            <Popconfirm title="确认吊销该密钥？吊销后立即失效" onConfirm={() => void handleRevoke(r)}>
              <Button type="link" size="small" danger>
                吊销
              </Button>
            </Popconfirm>
          )}
          <Popconfirm title="确认删除该密钥？" onConfirm={() => void handleDelete(r)}>
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
      <Space style={{ marginBottom: 16 }} wrap>
        <Select
          allowClear
          showSearch
          optionFilterProp="label"
          placeholder="选择要管理的服务账号"
          style={{ width: 320 }}
          value={machineUserID}
          onChange={(v) => {
            setMachineUserID(v)
            setPage(1)
            setData([])
          }}
          options={machineOptions.map((m) => ({ label: m.name, value: m.machineUserID }))}
        />
        <Button
          type="primary"
          icon={<PlusOutlined />}
          disabled={!machineUserID}
          onClick={() => {
            form.resetFields()
            form.setFieldsValue({ machineUserID })
            setModalOpen(true)
          }}
        >
          新建 API 密钥
        </Button>
      </Space>
      <ToolbarRow
        searchPlaceholder="按密钥名称搜索"
        onSearch={(v) => {
          setKeyword(v)
          setPage(1)
        }}
        onRefresh={() => void fetchData()}
      />
      <Table<TenantApiKeyItem>
        rowKey="keyID"
        columns={columns}
        dataSource={data}
        loading={loading}
        scroll={{ x: 1150 }}
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
        title="新建服务账号 API 密钥"
        open={modalOpen}
        onOk={() => void handleSubmit()}
        onCancel={() => setModalOpen(false)}
        confirmLoading={submitLoading}
        destroyOnClose
        width={540}
      >
        <Form form={form} layout="vertical">
          <Form.Item name="machineUserID" label="归属服务账号" rules={[{ required: true, message: '请选择归属服务账号' }]}>
            <Select
              showSearch
              optionFilterProp="label"
              placeholder="选择该密钥归属的服务账号"
              options={machineOptions.map((m) => ({ label: m.name, value: m.machineUserID }))}
            />
          </Form.Item>
          <Form.Item name="name" label="名称" rules={[{ required: true, message: '请输入名称' }]}>
            <Input placeholder="如：CI 构建凭证" />
          </Form.Item>
          <Form.Item name="expiresAt" label="过期时间">
            <DatePicker showTime style={{ width: '100%' }} placeholder="选填，留空表示永不过期" />
          </Form.Item>
        </Form>
      </Modal>

      <CreatedKeyModal created={created} onClose={() => setCreated(null)} />
    </>
  )
}

export default function ApiKeyPage() {
  return (
    <PageContainer title="API 密钥" description="机器访问凭证：个人密钥代表用户本人，服务账号密钥供开发者/服务端集成">
      <Alert
        type="info"
        showIcon
        style={{ marginBottom: 16 }}
        message="服务账号密钥面向开发者/服务端集成；个人密钥代表用户本人。两者均可设置有效期并在到期前吊销。"
      />
      <Tabs
        defaultActiveKey="mine"
        items={[
          { key: 'mine', label: '我的密钥', children: <MyKeysPane /> },
          { key: 'machine', label: '服务账号密钥', children: <MachineKeysPane /> },
        ]}
      />
    </PageContainer>
  )
}
