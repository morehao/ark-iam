import { defineConfig } from 'vitest/config'

// @ark-iam/install 的测试基础设施：纯 TS 协议层，无需 jsdom/React，node 环境即可。
export default defineConfig({
  test: {
    environment: 'node',
    include: ['src/**/*.test.{ts,tsx}'],
  },
})
