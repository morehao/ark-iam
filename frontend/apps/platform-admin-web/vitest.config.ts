import { defineConfig } from 'vitest/config'
import react from '@vitejs/plugin-react'
import { arkIamAlias } from '../../vite.alias'

export default defineConfig({
  plugins: [react()],
  // 与 vite.config.ts / dev 保持一致：测试同样把 @ark-iam/* 与 @morehao/ark-iam-react 指向 packages/src，
  // 否则会解析到已构建的 dist（dist 外置了 dayjs 等第三方依赖，测试环境无法解析）。
  resolve: { alias: arkIamAlias },
  test: {
    environment: 'jsdom',
    setupFiles: './src/test/setup.ts',
    globals: true,
  },
})
