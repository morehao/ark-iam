import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

export default defineConfig({
  plugins: [react()],
  server: {
    port: 4000,
    proxy: {
      '/v1': {
        target: 'http://localhost:8100',
        changeOrigin: true,
      },
      '/oidc': {
        target: 'http://localhost:8100',
        changeOrigin: true,
      },
      // 初始化向导（GET /install/status、POST /install/initialize）由 auth 应用提供，
      // 与 /v1、/oidc 一样经 gateway(:8100) 代理。
      '/install': {
        target: 'http://localhost:8100',
        changeOrigin: true,
      },
    },
  },
})
