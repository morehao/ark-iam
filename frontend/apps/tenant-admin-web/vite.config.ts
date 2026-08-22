import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

export default defineConfig({
  plugins: [react()],
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
    },
  },
})
