import { useEffect, useState } from 'react'
import { Card, Col, Row, Statistic, Typography, Spin } from 'antd'
import {
  AppstoreOutlined,
  FileSearchOutlined,
  GlobalOutlined,
  KeyOutlined,
} from '@ant-design/icons'
import { PageContainer, tokens } from '@ark-iam/ui'
import { getApiKeySupervisionPageList, getApplicationPageList, getAuditLogPageList, getTenantPageList } from '@ark-iam/api'

interface Stat {
  title: string
  value: number | null
  icon: React.ReactNode
}

const STAT_CARDS: Stat[] = [
  { title: '应用总数', value: null, icon: <AppstoreOutlined /> },
  { title: '租户总数', value: null, icon: <GlobalOutlined /> },
  { title: 'API 密钥', value: null, icon: <KeyOutlined /> },
  { title: '审计日志', value: null, icon: <FileSearchOutlined /> },
]

export default function Dashboard() {
  const [stats, setStats] = useState<Stat[]>(STAT_CARDS)
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    let mounted = true
    const load = async () => {
      try {
        const [apps, tenants, apiKeys, logs] = await Promise.allSettled([
          getApplicationPageList({ page: 1, pageSize: 1 }),
          getTenantPageList({ page: 1, pageSize: 1 }),
          getApiKeySupervisionPageList({ page: 1, pageSize: 1 }),
          getAuditLogPageList({ page: 1, pageSize: 1 }),
        ])
        if (!mounted) return
        const count = (r: PromiseSettledResult<unknown>) =>
          r.status === 'fulfilled'
            ? ((r.value as { total?: number; list?: unknown[] }).total ??
              (Array.isArray((r.value as { list?: unknown[] }).list) ? (r.value as { list: unknown[] }).list.length : 0))
            : 0
        setStats([
          { title: '应用总数', value: count(apps), icon: <AppstoreOutlined /> },
          { title: '租户总数', value: count(tenants), icon: <GlobalOutlined /> },
          { title: 'API 密钥', value: count(apiKeys), icon: <KeyOutlined /> },
          { title: '审计日志', value: count(logs), icon: <FileSearchOutlined /> },
        ])
      } finally {
        if (mounted) setLoading(false)
      }
    }
    void load()
    return () => {
      mounted = false
    }
  }, [])

  return (
    <PageContainer title="仪表盘" description="平台整体运行概览">
      <Spin spinning={loading}>
        <div style={{ display: 'flex', flexWrap: 'wrap', gap: 20 }}>
          {stats.map((s) => (
            <div key={s.title} style={{ flex: '1 1 200px', minWidth: 200 }}>
              <Card
                hoverable
                styles={{ body: { padding: '22px 24px' } }}
                style={{ borderRadius: 14, border: `1px solid ${tokens.border}` }}
              >
                <div style={{ display: 'flex', alignItems: 'center', gap: 16 }}>
                  <div
                    style={{
                      width: 52,
                      height: 52,
                      borderRadius: 14,
                      display: 'flex',
                      alignItems: 'center',
                      justifyContent: 'center',
                      fontSize: 24,
                      color: tokens.primary,
                      background: tokens.softFill,
                    }}
                  >
                    {s.icon}
                  </div>
                  <div>
                    <Statistic title={s.title} value={s.value ?? 0} valueStyle={{ fontSize: 26, fontWeight: 700, color: tokens.text }} />
                  </div>
                </div>
              </Card>
            </div>
          ))}
        </div>

        <Card
          style={{ marginTop: 20, borderRadius: 14, border: `1px solid ${tokens.border}` }}
          styles={{ body: { padding: '24px 28px' } }}
        >
          <Typography.Title level={5} style={{ marginTop: 0 }}>
            平台能力
          </Typography.Title>
          <Row gutter={[16, 16]}>
            {[
              { icon: <GlobalOutlined />, title: '多租户治理', desc: '租户状态 · 应用订阅 · 自定义域名' },
              { icon: <AppstoreOutlined />, title: '应用接入', desc: '应用 · OAuth 客户端 · 租户应用' },
              { icon: <KeyOutlined />, title: '密钥监督', desc: '全租户 API 密钥只读监督（仅前缀）' },
              { icon: <FileSearchOutlined />, title: '平台治理', desc: '菜单字典 · 审计日志' },
            ].map((f) => (
              <Col xs={24} sm={12} lg={6} key={f.title}>
                <div
                  style={{
                    padding: '18px 20px',
                    borderRadius: 12,
                    background: tokens.tableHeaderBg,
                    border: `1px solid ${tokens.border}`,
                  }}
                >
                  <div style={{ fontSize: 22, color: tokens.primary, marginBottom: 8 }}>{f.icon}</div>
                  <div style={{ fontWeight: 600, marginBottom: 4 }}>{f.title}</div>
                  <div style={{ fontSize: 12, color: tokens.textSecondary }}>{f.desc}</div>
                </div>
              </Col>
            ))}
          </Row>
        </Card>
      </Spin>
    </PageContainer>
  )
}
