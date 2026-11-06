import { defineConfig } from 'vitest/config'
import react from '@vitejs/plugin-react'
import { arkIamAlias } from '../../vite.alias'

export default defineConfig({
  plugins: [react()],
  // 与 vite.config.ts / dev 保持一致：测试同样把 @ark-iam/* 与 @morehao/ark-iam-react 指向 packages/src。
  resolve: { alias: arkIamAlias },
  test: {
    environment: 'jsdom',
    setupFiles: './src/test/setup.ts',
    globals: true,
  },
})
