import { defineConfig } from 'vitest/config'
import react from '@vitejs/plugin-react'

// @ark-iam/login 的测试基础设施：jsdom + React 插件，用于组件测试（testing-library）。
export default defineConfig({
  plugins: [react()],
  test: {
    environment: 'jsdom',
    include: ['src/**/*.test.{ts,tsx}'],
    setupFiles: ['src/test/setup.ts'],
  },
})
