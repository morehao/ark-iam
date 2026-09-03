import { useCallback, useEffect, useState, type CSSProperties } from 'react'
import { Alert, Button, DatePicker, Form, Input, Modal, Popconfirm, Select, Space, Table, message } from 'antd'
import type { ColumnsType } from 'antd/es/table'
import { CopyOutlined, PlusOutlined, ReloadOutlined, SearchOutlined } from '@ant-design/icons'
import { EllipsisCell, fmtTime, PageContainer, tokens } from '@ark-iam/ui'
import type { TenantApiKeyCreateResp, TenantApiKeyItem, TenantMachineUserItem } from '@ark-iam/types'
import { createApiKey, deleteApiKey, getApiKeyPageList, revokeApiKey } from '../../api/apiKey'
import { getMachineUserPageList } from '../../api/machineUser'
import { KeyStateTag } from './KeyState'

interface ApiKeyFormValues {
  name: string
  machineUserID: string
  expiresAt?: { valueOf: () => number }
}

const monospaceStyle: CSSProperties = { fontFamily: 'Consolas, Monaco, monospace' }

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

/** API 密钥管理（单一视图）：密钥一律归属服务账号，供服务端集成；列表/创建/吊销/删除均需系统管理能力。 */
function ApiKeysPane() {
  const [data, setData] = useState<TenantApiKeyItem[]>([])
  const [loading, setLoading] = useState(false)
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(10)
  const [total, setTotal] = useState(0)
  const [keyword, setKeyword] = useState('')
  // 归属服务账号筛选：空=租户全部密钥
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
    setLoading(true)
    try {
      const resp = await getApiKeyPageList({
        page,
        pageSize,
        name: keyword || undefined,
        machineUserID: machineUserID || undefined,
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

  const openCreate = () => {
    form.resetFields()
    // 便捷：默认带入当前筛选的服务账号，仍可改选
    form.setFieldsValue({ machineUserID: machineUserID || undefined })
    setModalOpen(true)
  }

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
      title: '密钥前缀',
      dataIndex: 'keyPrefix',
      key: 'keyPrefix',
      width: 180,
      render: (v: string) => <span style={{ ...monospaceStyle, fontSize: 12 }}>{v || '-'}</span>,
    },
    { title: '归属服务账号', dataIndex: 'ownerName', key: 'ownerName', width: 170, render: (v: string, r) => `${r.ownerType === 'machine' ? '服务账号 · ' : ''}${v || '-'}` },
    { title: '状态', key: 'status', width: 90, render: (_: unknown, r) => <KeyStateTag {...r} /> },
    { title: '过期时间', dataIndex: 'expiredAt', key: 'expiredAt', width: 160, render: (v: number | null) => fmtTime(v) },
    { title: '最近使用', dataIndex: 'lastUsedAt', key: 'lastUsedAt', width: 160, render: (v: number | null) => fmtTime(v) },
    { title: '创建人', dataIndex: 'creatorName', key: 'creatorName', width: 160, render: (v: string) => v || '-' },
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
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 16, flexWrap: 'wrap', gap: 8 }}>
        <Space wrap>
          <Select
            allowClear
            showSearch
            optionFilterProp="label"
            placeholder="归属服务账号（不选=租户全部）"
            style={{ width: 300 }}
            value={machineUserID}
            onChange={(v) => {
              setMachineUserID(v)
              setPage(1)
              setData([])
            }}
            options={machineOptions.map((m) => ({ label: m.name, value: m.machineUserID }))}
          />
          <Input.Search
            allowClear
            placeholder="按密钥名称搜索"
            prefix={<SearchOutlined />}
            style={{ width: 240 }}
            onSearch={(v) => {
              setKeyword(v)
              setPage(1)
            }}
          />
        </Space>
        <Space>
          <Button icon={<ReloadOutlined />} onClick={() => void fetchData()}>
            刷新
          </Button>
          <Button type="primary" icon={<PlusOutlined />} onClick={openCreate}>
            新建 API 密钥
          </Button>
        </Space>
      </div>

      <Table<TenantApiKeyItem>
        rowKey="keyID"
        columns={columns}
        dataSource={data}
        loading={loading}
        scroll={{ x: 1180 }}
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
        width={540}
      >
        <Form form={form} layout="vertical">
          <div style={{ marginBottom: 16, color: tokens.textPlaceholder, fontSize: 12 }}>
            密钥归属于所选服务账号，供服务端集成调用平台 API；创建后明文仅展示一次，请立即保存。创建与管理密钥需系统管理能力。
          </div>
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
    <PageContainer title="API 密钥" description="服务端集成凭证：密钥归属服务账号，创建与管理需系统管理能力">
      <Alert
        type="info"
        showIcon
        style={{ marginBottom: 16 }}
        message="密钥一律归属于服务账号（机器主体），供开发者/服务端集成调用平台 API；可设置有效期，使用前可在到期前吊销。本页密钥的查看、创建、吊销与删除均需系统管理能力。"
      />
      <ApiKeysPane />
    </PageContainer>
  )
}
