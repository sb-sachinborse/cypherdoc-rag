import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'

// In dev, run the backend on :8081 (PORT=8081) and Vite proxies /api to it.
export default defineConfig({
  plugins: [vue()],
  server: { port: 5173, proxy: { '/api': 'http://localhost:8081' } },
})
