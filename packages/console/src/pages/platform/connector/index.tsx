import { useCallback, useEffect, useState } from 'react'
import { Button, Input, Segmented, Select, Space, Table, Tag, message } from 'antd'
import type { ColumnsType } from 'antd/es/table'
import {
  DeleteOutlined,
  EditOutlined,
  PlusOutlined,
  ReloadOutlined,
  SearchOutlined,
} from '@ant-design/icons'
import {
  actionColumn,
  EnableTag,
  nameColumn,
  PageContainer,
  StatCard,
  StatRow,
  STATUS_COL_WIDTH,
  tableScrollX,
  textColumn,
  timeColumn,
  tokens,
} from '@ark-iam/ui'
import { deleteConnector, getConnectorPageList, getTenantPageList } from '@ark-iam/api'
import type { ConnectorItem, TenantItem } from '@ark-iam/types'
import { PROVIDER_COLOR, PROVIDER_LABEL } from './constants'
import ConnectorWizard from './ConnectorWizard'
import ConnectorDetail from './ConnectorDetail'

type StatusFilter = 'all' | 'enable' | 'disable'

/**
 * 身份连接器列表（平台管理后台）。
 *
 * 连接器是平台级全局对象（由平台运营方统一管理），列表含全部租户的连接器；
 * 「默认落租户」是「自动创建用户」时新用户默认落到的租户，与协议级 tenant 配置无关。
 * 列宽/操作列/时间列均走 @ark-iam/ui 的工厂，scroll 由 tableScrollX 求和，不手写。
 */
export default function ConnectorList() {
  const [data, setData] = useState<ConnectorItem[]>([])
  const [loading, setLoading] = useState(false)
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(10)
  const [total, setTotal] = useState(0)
  const [keyword, setKeyword] = useState('')
  const [statusFilter, setStatusFilter] = useState<StatusFilter>('all')
  const [defaultTenantFilter, setDefaultTenantFilter] = useState<string | undefined>(undefined)
  const [tenants, setTenants] = useState<TenantItem[]>([])

  const [wizardOpen, setWizardOpen] = useState(false)
  const [editing, setEditing] = useState<ConnectorItem | null>(null)

  const [detailOpen, setDetailOpen] = useState(false)
  const [detail, setDetail] = useState<ConnectorItem | null>(null)


  // 平台租户列表（供「默认落租户」筛选下拉）
  useEffect(() => {
    getTenantPageList({ page: 1, pageSize: 200 })
      .then((resp) => setTenants(resp?.list || []))
      .catch(() => {
        /* 拦截器已提示 */
      })
  }, [])

  const fetchData = useCallback(async () => {
    setLoading(true)
    try {
      const resp = await getConnectorPageList({
        page,
        pageSize,
        name: keyword || undefined,
        status: statusFilter === 'all' ? undefined : statusFilter,
        defaultTenantID: defaultTenantFilter || undefined,
      })
      setData(resp?.list || [])
      setTotal(resp?.total || 0)
    } catch {
      /* 拦截器已提示 */
    } finally {
      setLoading(false)
    }
  }, [page, pageSize, keyword, statusFilter, defaultTenantFilter])

  useEffect(() => {
    void fetchData()
  }, [fetchData])

  const handleCreate = () => {
    setEditing(null)
    setWizardOpen(true)
  }
  const handleEdit = (r: ConnectorItem) => {
    setEditing(r)
    setDetailOpen(false)
    setWizardOpen(true)
  }
  const handleOpenDetail = (r: ConnectorItem) => {
    setDetail(r)
    setDetailOpen(true)
  }
  const handleDelete = async (r: ConnectorItem) => {
    try {
      await deleteConnector(r.connectorID)
      message.success('删除成功')
      void fetchData()
    } catch {
      /* 拦截器已提示 */
    }
  }

  const columns: ColumnsType<ConnectorItem> = [
    nameColumn<ConnectorItem>({
      title: '名称',
      dataIndex: 'displayName',
      onClick: (r) => handleOpenDetail(r),
    }),
    textColumn<ConnectorItem>({
      title: '协议',
      dataIndex: 'protocol',
      width: STATUS_COL_WIDTH,
    }),
    {
      title: '提供商',
      dataIndex: 'provider',
      key: 'provider',
      width: 150,
      render: (v: ConnectorItem['provider']) => <Tag color={PROVIDER_COLOR[v]}>{PROVIDER_LABEL[v] ?? v}</Tag>,
    },
    {
      title: '默认落租户',
      dataIndex: 'defaultTenantID',
      key: 'defaultTenantID',
      width: 180,
      ellipsis: true,
      render: (_: unknown, r) => (
        <span style={{ color: r.defaultTenantName || r.defaultTenantID ? undefined : tokens.textTertiary }}>
          {r.defaultTenantName || r.defaultTenantID || '-'}
        </span>
      ),
    },
    {
      title: '状态',
      dataIndex: 'status',
      key: 'status',
      width: STATUS_COL_WIDTH,
      render: (v: string) => <EnableTag value={v} />,
    },
    timeColumn<ConnectorItem>({ title: '创建时间', dataIndex: 'createdAt' }),
    timeColumn<ConnectorItem>({ title: '更新时间', dataIndex: 'updatedAt' }),
    actionColumn<ConnectorItem>({
      max: 3,
      variant: 'icon',
      actions: (r) => [
        { key: 'edit', label: '编辑', icon: <EditOutlined />, onClick: () => handleEdit(r) },
        {
          key: 'delete',
          label: '删除',
          icon: <DeleteOutlined />,
          danger: true,
          confirm: `确认删除连接器「${r.displayName}」？删除后该登录入口立即失效，已绑定用户保留但无法再用其登录。`,
          onClick: () => void handleDelete(r),
        },
      ],
    }),
  ]

  const enabledTotal = data.filter((c) => c.status === 'enable').length

  return (
    <PageContainer
      title="身份连接器"
      description="平台级身份提供商（Google / GitHub / Microsoft Entra ID 等）统一接入管理，开启第三方登录、账号关联与资料同步。可指定自动创建用户的默认落租户。"
      extra={
        <Space>
          <Button icon={<ReloadOutlined />} onClick={() => void fetchData()}>
            刷新
          </Button>
          <Button type="primary" icon={<PlusOutlined />} onClick={handleCreate}>
            新建连接器
          </Button>
        </Space>
      }
      summary={
        <StatRow>
          <StatCard label="连接器总数" value={total} hint="平台全局" />
          <StatCard label="本页启用" value={enabledTotal} hint="参与授权与回调" />
          <StatCard label="本页停用" value={data.length - enabledTotal} hint="保留配置不参与授权" />
          <StatCard label="当前页" value={data.length} hint={`第 ${page} 页 · 每页 ${pageSize} 条`} />
        </StatRow>
      }
      toolbar={
        <Space wrap>
          <Input.Search
            allowClear
            placeholder="按显示名 / 标识名搜索"
            prefix={<SearchOutlined />}
            style={{ width: 260 }}
            onSearch={(v) => {
              setKeyword(v)
              setPage(1)
            }}
          />
          <Select
            allowClear
            showSearch
            optionFilterProp="label"
            placeholder="默认落租户"
            style={{ width: 200 }}
            value={defaultTenantFilter}
            options={tenants.map((t) => ({ value: t.tenantID, label: t.name }))}
            onChange={(v) => {
              setDefaultTenantFilter(v)
              setPage(1)
            }}
          />
          <Segmented
            value={statusFilter}
            onChange={(v) => {
              setStatusFilter(v as StatusFilter)
              setPage(1)
            }}
            options={[
              { value: 'all', label: '全部' },
              { value: 'enable', label: '启用' },
              { value: 'disable', label: '停用' },
            ]}
          />
        </Space>
      }
    >
      <Table<ConnectorItem>
        rowKey="connectorID"
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

      {wizardOpen && (
        <ConnectorWizard
          open={wizardOpen}
          editing={editing}
          onClose={() => setWizardOpen(false)}
          onSaved={() => {
            setWizardOpen(false)
            void fetchData()
          }}
        />
      )}

      <ConnectorDetail
        open={detailOpen}
        connector={detail}
        onClose={() => setDetailOpen(false)}
        onEdit={handleEdit}
      />

    </PageContainer>
  )
}
