// @morehao/ark-iam-react —— 聚合入口（一把梭：从根入口 import 全部）。
//
// 需要按需减小体积时，请优先用子路径：
//   import { BizCode } from '@morehao/ark-iam-react/types'          // 零重依赖
//   import { createApiClient } from '@morehao/ark-iam-react/api'
//   import { createAuthProvider } from '@morehao/ark-iam-react/auth'
//   import { actionColumn, themeConfig } from '@morehao/ark-iam-react/ui'  // 含 antd
//
// 根入口会把四层全部 re-export（含 ui 的 antd 依赖），仅供便利；追求体积请用子路径。
export * from '@ark-iam/types'
export * from '@ark-iam/api'
export * from '@ark-iam/auth'
export * from '@ark-iam/ui'
