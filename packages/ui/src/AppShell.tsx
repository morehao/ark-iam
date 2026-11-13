import type { ReactNode } from 'react'
import { ConfigProvider, App as AntdApp } from 'antd'
import zhCN from 'antd/locale/zh_CN'
import dayjs from 'dayjs'
import 'dayjs/locale/zh-cn'
import { themeConfig } from './theme'

dayjs.locale('zh-cn')

interface Props {
  children: ReactNode
}

/**
 * 应用外壳：注入 Ant Design 全局主题（中文语言包 + 设计令牌）。
 * 每个前端 app 的 main.tsx 用 <AppShell> 包裹。
 * 根容器开启 tabular-nums，让 ID/时间/数字等宽对齐（Stripe 数据细节，见 DESIGN.md）。
 *
 * 全局 antd「结构级」微修正也集中在此：AppShell 是各 app 唯一公共挂载点，
 * 而这类修正针对 antd 内部 DOM / portal 渲染的下拉等，无法用组件 inline style 表达
 * （约定与清单见 DESIGN.md §7.1）。禁止各 app 新增 css 文件或在页面散落重复 <style>。
 */
export function AppShell({ children }: Props) {
  return (
    <div style={{ fontVariantNumeric: 'tabular-nums' }}>
      {/* TreeSelect 下拉树：去掉顶层节点的左侧空白。antd 树即使对最顶层节点也会渲染
          固定的“展开/收起箭头列” .ant-select-tree-switcher（默认宽 24px），把宽度归 0、
          隐藏溢出后顶层节点即与下拉框左对齐；子级缩进由 .ant-select-tree-indent-unit
          （每个 24px）独立控制，不受影响。 */}
      <style>{`
        .ant-select-dropdown .ant-select-tree .ant-select-tree-switcher {
          width: 0;
          overflow: hidden;
        }
      `}</style>
      <ConfigProvider locale={zhCN} theme={themeConfig}>
        <AntdApp>{children}</AntdApp>
      </ConfigProvider>
    </div>
  )
}
