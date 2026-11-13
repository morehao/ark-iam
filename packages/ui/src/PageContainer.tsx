import type { ReactNode } from 'react'
import { Typography } from 'antd'
import { tokens } from './theme'

interface Props {
  title: string
  description?: string
  extra?: ReactNode
  children: ReactNode
  loading?: boolean
  /** 标题与内容卡之间的概要区（如指标卡条 StatRow），页面自备内容 */
  summary?: ReactNode
  /** 内容卡顶部工具条（分段 Tab / 搜索筛选区），与内容卡共享边框 */
  toolbar?: ReactNode
}

/**
 * 统一页面容器：页面标题区（标题 + 描述 + 右侧操作按钮）+ 可选概要区 + 内容卡。
 * 内容卡为 hairline-only（1px 边框 + 8px 圆角 + 20px 内边距），不使用阴影。
 * 所有管理页共用，保证页面骨架一致（见 DESIGN.md §4 / §7.3，对齐 ark-go-starter）。
 */
export function PageContainer({ title, description, extra, children, summary, toolbar }: Props) {
  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 16 }}>
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start', gap: 16 }}>
        <div>
          <Typography.Title level={4} style={{ margin: 0, fontWeight: 600 }}>
            {title}
          </Typography.Title>
          {description && (
            <Typography.Text type="secondary" style={{ fontSize: 13 }}>
              {description}
            </Typography.Text>
          )}
        </div>
        {extra}
      </div>
      {summary}
      <div
        style={{
          background: tokens.cardBg,
          border: `1px solid ${tokens.border}`,
          borderRadius: tokens.radiusLg,
          overflow: toolbar ? 'hidden' : undefined,
        }}
      >
        {toolbar ? (
          <div style={{ padding: '10px 16px', borderBottom: `1px solid ${tokens.border}` }}>{toolbar}</div>
        ) : null}
        <div style={{ padding: 20 }}>{children}</div>
      </div>
    </div>
  )
}

export function pageTitleStyle(): React.CSSProperties {
  return { color: tokens.text }
}
