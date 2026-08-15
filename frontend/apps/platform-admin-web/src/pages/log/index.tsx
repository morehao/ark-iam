import { useCallback, useEffect, useState } from 'react'
import { Button, Descriptions, Drawer, Input, Space, Table, Tooltip } from 'antd'
import { ReloadOutlined, SearchOutlined } from '@ant-design/icons'
import type { ColumnsType } from 'antd/es/table'
import { EllipsisCell, IDCell, PageContainer } from '@ark-iam/ui'
import { getAuditLogDetail, getAuditLogPageList } from '@ark-iam/api'
import type { AuditLogItem } from '@ark-iam/types'
import { fmtTime } from '../../components/common'

/** 序列化 payload：对象 JSON 化，其余转字符串 */
function formatPayload(value: unknown): string {
  if (value == null) return '-'
  if (typeof value === 'object') {
    try {
      return JSON.stringify(value) ?? String(value)
    } catch {
      return String(value)
    }
  }
  return String(value)
}

/** 超长文本截断 */
function truncate(text: string, max: number): string {
  return text.length > max ? `${text.slice(0, max)}...` : text
}

export default function AuditLogList() {
  const [data, setData] = useState<AuditLogItem[]>([])
  const [loading, setLoading] = useState(false)
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(10)
  const [total, setTotal] = useState(0)
  const [keyword, setKeyword] = useState('')

  const [drawerOpen, setDrawerOpen] = useState(false)
  const [detail, setDetail] = useState<AuditLogItem | null>(null)

  const fetchData = useCallback(async () => {
    setLoading(true)
    try {
      const resp = await getAuditLogPageList({ page, pageSize, key: keyword })
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

  const handleDetail = async (record: AuditLogItem) => {
    try {
      const resp = await getAuditLogDetail(record.logID)
      setDetail(resp)
      setDrawerOpen(true)
    } catch {
      /* 拦截器已提示 */
    }
  }

  const columns: ColumnsType<AuditLogItem> = [
    { title: 'ID', dataIndex: 'logID', key: 'logID', width: 150, render: (v: string) => <IDCell value={v} /> },
    {
      title: '日志键',
      dataIndex: 'key',
      key: 'key',
      width: 220,
      render: (v: string) => <EllipsisCell value={v} monospace />,
    },
    {
      title: '内容',
      dataIndex: 'payload',
      key: 'payload',
      ellipsis: { showTitle: false },
      render: (_, r) => {
        const text = formatPayload(r.payload)
        return (
          <Tooltip title={text}>
            <span style={{ fontFamily: 'monospace' }}>{truncate(text, 60)}</span>
          </Tooltip>
        )
      },
    },
    { title: '租户ID', dataIndex: 'tenantID', key: 'tenantID', width: 150, render: (v: string) => <IDCell value={v} /> },
    {
      title: '创建时间',
      key: 'createdAt',
      width: 160,
      render: (_, r) => fmtTime(r.createdAt),
    },
    {
      title: '操作',
      key: 'action',
      width: 100,
      render: (_, r) => (
        <Button type="link" size="small" onClick={() => void handleDetail(r)}>
          详情
        </Button>
      ),
    },
  ]

  return (
    <PageContainer
      title="审计日志"
      description="平台操作审计记录（只读）"
      extra={
        <Space>
          <Button icon={<ReloadOutlined />} onClick={() => void fetchData()}>
            刷新
          </Button>
        </Space>
      }
    >
      <div style={{ marginBottom: 16 }}>
        <Input.Search
          allowClear
          placeholder="按日志键搜索"
          prefix={<SearchOutlined />}
          style={{ width: 240 }}
          onSearch={(v) => {
            setKeyword(v)
            setPage(1)
          }}
        />
      </div>
      <Table<AuditLogItem>
        rowKey="logID"
        columns={columns}
        dataSource={data}
        loading={loading}
        scroll={{ x: 800 }}
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

      <Drawer title="审计日志详情" width={560} open={drawerOpen} onClose={() => setDrawerOpen(false)}>
        {detail && (
          <>
            <Descriptions column={1} bordered size="small">
              <Descriptions.Item label="ID"><IDCell value={detail.logID} /></Descriptions.Item>
              <Descriptions.Item label="日志键">{detail.key}</Descriptions.Item>
              <Descriptions.Item label="租户ID"><IDCell value={detail.tenantID} /></Descriptions.Item>
              <Descriptions.Item label="创建时间">{fmtTime(detail.createdAt)}</Descriptions.Item>
            </Descriptions>
            <div style={{ marginTop: 16, marginBottom: 8, fontWeight: 500 }}>内容</div>
            <pre
              style={{
                margin: 0,
                padding: 12,
                overflow: 'auto',
                maxHeight: 400,
                whiteSpace: 'pre-wrap',
                wordBreak: 'break-all',
                fontFamily: 'monospace',
                fontSize: 12,
                lineHeight: 1.6,
                background: '#fafafa',
                border: '1px solid #f0f0f0',
                borderRadius: 8,
              }}
            >
              {typeof detail.payload === 'object' && detail.payload !== null
                ? (JSON.stringify(detail.payload, null, 2) ?? '')
                : String(detail.payload ?? '')}
            </pre>
          </>
        )}
      </Drawer>
    </PageContainer>
  )
}
