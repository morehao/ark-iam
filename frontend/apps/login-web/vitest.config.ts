import { defineConfig } from 'vitest/config'
import react from '@vitejs/plugin-react'

// login-web 的测试基础设施（与 platform-admin-web 的 vitest.config.ts 保持一致）。
// 本 app 刻意不依赖 @ark-iam/*，测试也只覆盖纯逻辑（校验规则、向导状态机、
// 错误码 -> 文案/处置映射、信封解码），因此不需要 jest-dom / testing-library。
export default defineConfig({
  plugins: [react()],
  test: {
    environment: 'jsdom',
    include: ['src/**/*.test.{ts,tsx}'],
  },
})
