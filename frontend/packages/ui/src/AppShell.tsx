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
 */
export function AppShell({ children }: Props) {
  return (
    <ConfigProvider locale={zhCN} theme={themeConfig}>
      <AntdApp>{children}</AntdApp>
    </ConfigProvider>
  )
}
