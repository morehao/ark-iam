import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import { arkIamAlias } from '../../vite.alias'

export default defineConfig({
  plugins: [react()],
  resolve: {
    alias: arkIamAlias,
  },
  server: {
    port: 4002,
    proxy: {
      '/v1': {
        target: 'http://localhost:8100',
        changeOrigin: true,
      },
      '/oidc': {
        target: 'http://localhost:8100',
        changeOrigin: true,
      },
      // /config.json 是后端下发的运行时配置（见 backend/iam/app_frontend.go），AuthProvider 启动先拉它。
      '/config.json': {
        target: 'http://localhost:8100',
        changeOrigin: true,
      },
    },
  },
})
