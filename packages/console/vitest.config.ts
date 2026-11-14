import { defineConfig } from 'vitest/config'
import react from '@vitejs/plugin-react'
import { resolve } from 'node:path'

// console 包单测配置：把 @ark-iam/{ui,api,auth,types} 指向各自 src，避免依赖已构建 dist。
const PKG = resolve(__dirname, '..')
const alias = [
  { find: /^@ark-iam\/types$/, replacement: resolve(PKG, 'types/src/index.ts') },
  { find: /^@ark-iam\/api$/, replacement: resolve(PKG, 'api/src/index.ts') },
  { find: /^@ark-iam\/auth$/, replacement: resolve(PKG, 'auth/src/index.ts') },
  { find: /^@ark-iam\/ui$/, replacement: resolve(PKG, 'ui/src/index.ts') },
]

export default defineConfig({
  plugins: [react()],
  resolve: { alias },
  test: {
    environment: 'jsdom',
    setupFiles: './src/test/setup.ts',
    globals: true,
  },
})
