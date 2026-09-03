import { useCallback, useEffect, useState } from 'react'
import { Alert, Button, Input, Space, Table, Tag } from 'antd'
import type { ColumnsType } from 'antd/es/table'
import { ReloadOutlined, SearchOutlined } from '@ant-design/icons'
import { EllipsisCell, fmtTime, IDCell, PageContainer } from '@ark-iam/ui'
import { getApiKeySupervisionPageList } from '@ark-iam/api'
import type { ApiKeySupervisionItem } from '@ark-iam/types'

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

// 密钥前缀：无值展示 -，超长省略展示（悬浮完整，等宽字体）
function KeyPrefixCell({ value }: { value: string }) {
  if (!value) return <span>-</span>
  return <EllipsisCell value={value} monospace limit={24} />
}

// 归属主体：ownerName + ownerType 徽标；ownerName 空则展示 -
function OwnerCell({ item }: { item: ApiKeySupervisionItem }) {
  if (!item.ownerName) return <span>-</span>
  const badge = item.ownerType === 'machine' ? <Tag color="geekblue">服务账号</Tag> : <Tag>用户</Tag>
  return (
    <Space size={4}>
      {badge}
      <EllipsisCell value={item.ownerName} />
    </Space>
  )
}

export default function ApiKeySupervisionList() {
  const [data, setData] = useState<ApiKeySupervisionItem[]>([])
  const [loading, setLoading] = useState(false)
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(10)
  const [total, setTotal] = useState(0)

  const [tenantID, setTenantID] = useState('')
  const [keyword, setKeyword] = useState('')
  const [tenantDraft, setTenantDraft] = useState('')
  const [nameDraft, setNameDraft] = useState('')

  const fetchData = useCallback(async () => {
    setLoading(true)
    try {
      const resp = await getApiKeySupervisionPageList({
        page,
        pageSize,
        name: keyword || undefined,
        tenantID: tenantID || undefined,
      })
      setData(resp?.list || [])
      setTotal(resp?.total || 0)
    } catch {
      /* 拦截器已提示 */
    } finally {
      setLoading(false)
    }
  }, [page, pageSize, keyword, tenantID])

  useEffect(() => {
    void fetchData()
  }, [fetchData])

  const columns: ColumnsType<ApiKeySupervisionItem> = [
    { title: '名称', dataIndex: 'name', key: 'name', width: 200, render: (v: string) => <EllipsisCell value={v} /> },
    { title: '密钥前缀', dataIndex: 'keyPrefix', key: 'keyPrefix', width: 170, render: (v: string) => <KeyPrefixCell value={v} /> },
    { title: '范围', dataIndex: 'scope', key: 'scope', width: 180, render: (v: string) => <EllipsisCell value={v} /> },
    {
      title: '归属租户',
      key: 'tenant',
      width: 220,
      render: (_, r) => (
        <div style={{ lineHeight: 1.6 }}>
          <EllipsisCell value={r.tenantName || r.tenantID} />
          <div style={{ fontSize: 12 }}>
            <IDCell value={r.tenantID} />
          </div>
        </div>
      ),
    },
    {
      title: '归属主体',
      key: 'owner',
      width: 200,
      render: (_, r) => <OwnerCell item={r} />,
    },
    { title: '创建人', dataIndex: 'creatorName', key: 'creatorName', width: 160, render: (v: string, r) => <EllipsisCell value={v || r.createdBy} /> },
    {
      title: '状态',
      key: 'status',
      width: 90,
      render: (_, r) => KeyStateTag(r),
    },
    { title: '过期时间', dataIndex: 'expiresAt', key: 'expiresAt', width: 160, render: (v: number) => fmtTime(v) },
    { title: '最后使用', dataIndex: 'lastUsedAt', key: 'lastUsedAt', width: 160, render: (v: number) => fmtTime(v) },
    { title: '创建时间', dataIndex: 'createdAt', key: 'createdAt', width: 160, render: (v: number) => fmtTime(v) },
  ]

  return (
    <PageContainer
      title="API密钥监督"
      description="平台全租户只读监督视图：明文密钥永不可见（仅前缀），用于密钥泄漏/僵尸密钥排查。"
    >
      <Alert
        type="info"
        showIcon
        style={{ marginBottom: 16 }}
        message="平台全租户只读监督视图"
        description="用于密钥泄漏/僵尸密钥排查：明文密钥永不可见（仅展示前缀），本视图不提供吊销与删除（平台应急吊销暂未纳入）。密钥的创建与生命周期管理在租户控制台「API密钥」模块。若发现可疑密钥请联系对应租户管理员处理。"
      />
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 16, flexWrap: 'wrap', gap: 8 }}>
        <Space wrap>
          <Input.Search
            allowClear
            placeholder="租户ID"
            prefix={<SearchOutlined />}
            style={{ width: 220 }}
            value={tenantDraft}
            onChange={(e) => {
              const v = e.target.value
              setTenantDraft(v)
              // 清空搜索框时立即回到全量列表
              if (v === '') {
                setTenantID('')
                setPage(1)
              }
            }}
            onSearch={(v) => {
              setTenantID(v.trim())
              setPage(1)
            }}
          />
          <Input.Search
            allowClear
            placeholder="密钥名称关键词"
            prefix={<SearchOutlined />}
            style={{ width: 220 }}
            value={nameDraft}
            onChange={(e) => {
              const v = e.target.value
              setNameDraft(v)
              // 清空搜索框时立即回到全量列表
              if (v === '') {
                setKeyword('')
                setPage(1)
              }
            }}
            onSearch={(v) => {
              setKeyword(v.trim())
              setPage(1)
            }}
          />
        </Space>
        <Button icon={<ReloadOutlined />} onClick={() => void fetchData()}>
          刷新
        </Button>
      </div>
      <Table<ApiKeySupervisionItem>
        rowKey="id"
        columns={columns}
        dataSource={data}
        loading={loading}
        scroll={{ x: 1600 }}
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
    </PageContainer>
  )
}
