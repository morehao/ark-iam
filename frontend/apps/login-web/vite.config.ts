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
      // 初始化向导的两个接口（GET /install/status、POST /install/initialize）由 auth 应用提供，
      // 与 /v1、/oidc 一样经 gateway(:8100) 代理。
      //
      // **必须逐条列出，不能用 `/install` 前缀**：`/install` 同时是**本应用的页面路由**
      // （`<Route path="/install">`）。前缀代理会把浏览器对 http://localhost:4000/install 的
      // 导航也转发给后端，后端只注册了 /install/status 与 /install/initialize，
      // 于是安装页永远打不开、直接看到网关的 404（真实踩过：e2e 页面用例拿到 "404 page not found"）。
      // 生产反代的规则同理——页面路由归前端，只有这两个接口路径归后端。
      '/install/status': {
        target: 'http://localhost:8100',
        changeOrigin: true,
      },
      '/install/initialize': {
        target: 'http://localhost:8100',
        changeOrigin: true,
      },
    },
  },
})
