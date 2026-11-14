import { resolve } from 'node:path'

// @morehao/ark-iam-react 门面包的多入口库构建。
//
// 关键：为做到「对外只有一个 npm 包」，@ark-iam/{types,api,auth,ui} 必须**打进 dist**（不 external），
// 否则 publish 后 dependencies 会残留这四个内部包、消费者装不到 → 违反单包目标。
// 仅第三方依赖（react/antd/axios/oidc-client-ts/react-oidc-context/react-router-dom 等）external，
// 由 consumers 提供（peer）或由本包 dependencies 带入。
const THIRD_PARTY = [
  'react',
  'react-dom',
  'react/jsx-runtime',
  'react-router-dom',
  'antd',
  'antd/', // antd 子路径（locale 等）
  '@ant-design/icons',
  '@ant-design/',
  'dayjs',
  'oidc-client-ts',
  'react-oidc-context',
  'axios',
]

function isExternal(id: string): boolean {
  // 内部包一律打包进来。
  if (id.startsWith('@ark-iam/')) return false
  // 相对/绝对路径是本地源码。
  if (id.startsWith('.') || id.startsWith('/')) return false
  // node 内建。
  if (id.startsWith('node:')) return true
  // 第三方裸模块名 external。
  return THIRD_PARTY.some((p) => id === p || id.startsWith(p))
}

export default {
  build: {
    lib: {
      entry: {
        index: resolve(process.cwd(), 'src/index.ts'),
        types: resolve(process.cwd(), 'src/types.ts'),
        api: resolve(process.cwd(), 'src/api.ts'),
        auth: resolve(process.cwd(), 'src/auth.ts'),
        ui: resolve(process.cwd(), 'src/ui.ts'),
        login: resolve(process.cwd(), 'src/login.ts'),
        console: resolve(process.cwd(), 'src/console.ts'),
      },
      formats: ['es'] as const,
      fileName: (_fmt: string, entryName: string) => `${entryName}.js`,
    },
    rollupOptions: {
      external: isExternal,
    },
    sourcemap: true,
    minify: false,
    outDir: 'dist',
    emptyOutDir: true,
  },
}
