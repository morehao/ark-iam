import { useEffect, useState } from 'react'
import { Card, Col, Row, Statistic, Typography, Spin } from 'antd'
import {
  AppstoreOutlined,
  GlobalOutlined,
  SafetyCertificateOutlined,
  TeamOutlined,
  UserOutlined,
} from '@ant-design/icons'
import { PageContainer, brand } from '@ark-iam/ui'
import { getUserPageList, getRolePageList, getApplicationPageList, getTenantPageList } from '@ark-iam/api'

interface Stat {
  title: string
  value: number | null
  icon: React.ReactNode
  color: string
}

export default function Dashboard() {
  const [stats, setStats] = useState<Stat[]>([
    { title: '用户总数', value: null, icon: <UserOutlined />, color: '#4f6ef7' },
    { title: '角色总数', value: null, icon: <SafetyCertificateOutlined />, color: '#7a5af8' },
    { title: '应用总数', value: null, icon: <AppstoreOutlined />, color: '#f59e0b' },
    { title: '租户总数', value: null, icon: <GlobalOutlined />, color: '#22c55e' },
  ])
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    let mounted = true
    const load = async () => {
      try {
        const [users, roles, apps, tenants] = await Promise.allSettled([
          getUserPageList({ page: 1, pageSize: 1 }),
          getRolePageList({ page: 1, pageSize: 1 }),
          getApplicationPageList({ page: 1, pageSize: 1 }),
          getTenantPageList({ page: 1, pageSize: 1 }),
        ])
        if (!mounted) return
        const count = (r: PromiseSettledResult<unknown>) => (r.status === 'fulfilled' ? ((r.value as { total?: number; list?: unknown[] }).total ?? (Array.isArray((r.value as { list?: unknown[] }).list) ? (r.value as { list: unknown[] }).list.length : 0)) : 0)
        setStats([
          { title: '用户总数', value: count(users), icon: <UserOutlined />, color: '#4f6ef7' },
          { title: '角色总数', value: count(roles), icon: <SafetyCertificateOutlined />, color: '#7a5af8' },
          { title: '应用总数', value: count(apps), icon: <AppstoreOutlined />, color: '#f59e0b' },
          { title: '租户总数', value: count(tenants), icon: <GlobalOutlined />, color: '#22c55e' },
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
                style={{ borderRadius: 14, border: '1px solid #f0f0f0' }}
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
                      color: '#fff',
                      background: `linear-gradient(135deg, ${s.color} 0%, ${s.color}cc 100%)`,
                      boxShadow: `0 8px 18px ${s.color}40`,
                    }}
                  >
                    {s.icon}
                  </div>
                  <div>
                    <Statistic title={s.title} value={s.value ?? 0} valueStyle={{ fontSize: 26, fontWeight: 700, color: brand.text }} />
                  </div>
                </div>
              </Card>
            </div>
          ))}
        </div>

        <Card
          style={{ marginTop: 20, borderRadius: 14, border: '1px solid #f0f0f0' }}
          styles={{ body: { padding: '24px 28px' } }}
        >
          <Typography.Title level={5} style={{ marginTop: 0 }}>
            平台能力
          </Typography.Title>
          <Row gutter={[16, 16]}>
            {[
              { icon: <TeamOutlined />, title: '统一身份', desc: '用户 · 角色 · 组织，一套身份体系' },
              { icon: <SafetyCertificateOutlined />, title: '权限体系', desc: '菜单 · 权限域 · 资源 · 角色授权' },
              { icon: <AppstoreOutlined />, title: '应用接入', desc: '应用 · OAuth 客户端 · 域名 · 租户应用' },
              { icon: <GlobalOutlined />, title: '多租户', desc: '租户隔离 · 自助开通 · 租户切换' },
            ].map((f) => (
              <Col xs={24} sm={12} lg={6} key={f.title}>
                <div
                  style={{
                    padding: '18px 20px',
                    borderRadius: 12,
                    background: brand.gradientSoft,
                    border: '1px solid #ece9ff',
                  }}
                >
                  <div style={{ fontSize: 22, color: brand.primary, marginBottom: 8 }}>{f.icon}</div>
                  <div style={{ fontWeight: 600, marginBottom: 4 }}>{f.title}</div>
                  <div style={{ fontSize: 12, color: brand.textSecondary }}>{f.desc}</div>
                </div>
              </Col>
            ))}
          </Row>
        </Card>
      </Spin>
    </PageContainer>
  )
}
