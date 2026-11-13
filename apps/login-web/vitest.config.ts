import { defineConfig } from 'vitest/config'
import react from '@vitejs/plugin-react'
import { resolve } from 'node:path'

// login-web 的测试基础设施（与 platform-admin-web 的 vitest.config.ts 保持一致）。
// 账密页与 runtime config/auth API 已迁入 @ark-iam/login；本 app 测试覆盖 install 向导的纯逻辑
// （校验规则、向导状态机、错误码 -> 视图映射），并从 @ark-iam/login 复用共享类型/辅助。
// 测试与 dev 一致走包 src（alias），故 import @ark-iam/login 命中源码。
const loginSrc = resolve(__dirname, '../../packages/login/src/index.ts')

export default defineConfig({
  plugins: [react()],
  resolve: {
    alias: [{ find: /^@ark-iam\/login$/, replacement: loginSrc }],
  },
  test: {
    environment: 'jsdom',
    include: ['src/**/*.test.{ts,tsx}'],
  },
})
