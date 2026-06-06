import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

export default defineConfig({
  plugins: [react()],
  server: {
    port: 3003,
    proxy: {
      '/v1/iam': {
        target: 'http://localhost:8099',
        changeOrigin: true,
      },
    },
  },
})
