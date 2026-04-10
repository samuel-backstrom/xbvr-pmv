import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

export default defineConfig({
  plugins: [react()],
  base: '/ui-new/',
  server: {
    proxy: {
      '/api': 'http://localhost:9999',
      '/img': 'http://localhost:9999',
    },
  },
  build: {
    outDir: 'dist',
  },
})
