import { fileURLToPath } from 'node:url'
import { dirname, resolve } from 'node:path'

// 仓内 dev 免 build 的共享 alias（供各 app 的 vite.config 复用）。
//
// 把 @morehao/ark-iam-react 的各子路径直接指向对应内部包的 TS 源码，开发期无需先 build packages。
// 对外发布时消费者走 @morehao/ark-iam-react 的 dist 产物（见 packages/react/package.json 的 exports），
// 二者互不影响。另把内部包 @ark-iam/* 也 alias 到 src，兼容子包内部对彼此的 import。
const PKG = resolve(dirname(fileURLToPath(import.meta.url)), 'packages')

export const arkIamAlias = [
  { find: /^@morehao\/ark-iam-react\/types$/, replacement: resolve(PKG, 'types/src/index.ts') },
  { find: /^@morehao\/ark-iam-react\/api$/, replacement: resolve(PKG, 'api/src/index.ts') },
  { find: /^@morehao\/ark-iam-react\/auth$/, replacement: resolve(PKG, 'auth/src/index.ts') },
  { find: /^@morehao\/ark-iam-react\/ui$/, replacement: resolve(PKG, 'ui/src/index.ts') },
  { find: /^@morehao\/ark-iam-react$/, replacement: resolve(PKG, 'react/src/index.ts') },
  // 内部包自身引用（api/auth/ui 内部 import @ark-iam/{types,api,auth,ui}）
  { find: /^@ark-iam\/types$/, replacement: resolve(PKG, 'types/src/index.ts') },
  { find: /^@ark-iam\/api$/, replacement: resolve(PKG, 'api/src/index.ts') },
  { find: /^@ark-iam\/auth$/, replacement: resolve(PKG, 'auth/src/index.ts') },
  { find: /^@ark-iam\/ui$/, replacement: resolve(PKG, 'ui/src/index.ts') },
]
