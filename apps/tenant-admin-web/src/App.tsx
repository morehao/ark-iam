import { IamConsoleApp } from '@morehao/ark-iam-react'

// 租户管理台：整站独占、无外层 Router → 用自托管入口（内部 BrowserRouter + Provider + Routes）。
// 页面与门面逻辑全部来自 @morehao/ark-iam-react/console（见 packages/console）。
export default function App() {
  return <IamConsoleApp config={{ variant: 'tenant' }} />
}
