// vite.config.ts
// Vite 开发配置说明：
// - 启动前端开发服务器在 3000 端口
// - 通过代理将 `/api` 前缀的请求转发到后端 `http://localhost:8080`
//   保持与生产路径一致，避免跨域问题
import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'

export default defineConfig({
  plugins: [vue()],
  server: {
    port: 3000,
    proxy: {
      '/api': {
        target: 'http://localhost:8080',
        changeOrigin: true,
        rewrite: (path) => path.replace(/^\/api/, '/api')
      }
    }
  }
})