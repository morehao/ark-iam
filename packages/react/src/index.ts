// @morehao/ark-iam-react —— 聚合入口。
//
// 根入口只暴露「基础层」types / api / auth / ui（零页面实现）；页面级能力一律走显式子路径：
//   import { BizCode } from '@morehao/ark-iam-react/types'          // 零重依赖
//   import { createApiClient } from '@morehao/ark-iam-react/api'
//   import { createAuthProvider } from '@morehao/ark-iam-react/auth'
//   import { actionColumn, themeConfig } from '@morehao/ark-iam-react/ui'  // 含 antd
//   import { LoginPage } from '@morehao/ark-iam-react/login'         // OP 账密页（含 axios/表单）
//   import { InstallPage 协议/工具 } from '@morehao/ark-iam-react/install'  // /install 初始化协议
//   import { IamConsoleApp } from '@morehao/ark-iam-react/console'   // 页面级控制台
//
// 为保持根入口体积可控、并避免 login/console 的自足类型与 base 层同名符号二义（TS2308），
// 根入口不再 re-export login / console / install。
export * from '@ark-iam/types'
export * from '@ark-iam/api'
export * from '@ark-iam/auth'
export * from '@ark-iam/ui'
