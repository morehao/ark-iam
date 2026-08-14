import type { ReactNode } from 'react'
import { Card, Space, Typography } from 'antd'
import { brand } from './theme'

interface Props {
  title: string
  description?: string
  extra?: ReactNode
  children: ReactNode
  loading?: boolean
}

/**
 * 统一页面容器：页面标题区（标题 + 描述 + 右侧操作按钮）+ 内容卡片。
 * 所有管理页共用，保证页面骨架一致。
 */
export function PageContainer({ title, description, extra, children, loading }: Props) {
  return (
    <div>
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start', marginBottom: 16 }}>
        <div>
          <Typography.Title level={4} style={{ margin: 0, fontWeight: 700 }}>
            {title}
          </Typography.Title>
          {description && (
            <Typography.Text type="secondary" style={{ fontSize: 13 }}>
              {description}
            </Typography.Text>
          )}
        </div>
        {extra && <Space>{extra}</Space>}
      </div>
      <Card
        loading={loading}
        styles={{
          body: { padding: 20 },
        }}
        style={{ borderRadius: 12, boxShadow: '0 2px 12px rgba(15,23,42,0.05)', border: '1px solid #f0f0f0' }}
      >
        {children}
      </Card>
    </div>
  )
}

export function pageTitleStyle(): React.CSSProperties {
  return { color: brand.text }
}
