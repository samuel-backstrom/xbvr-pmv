import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

export default defineConfig({
  plugins: [react()],
  base: '/ui/',
  publicDir: '../ui/public',
  server: {
    proxy: {
      '/api': 'http://localhost:9999',
      '/img': 'http://localhost:9999',
      '/ws': {
        target: 'http://localhost:9999',
        ws: true,
      },
    },
  },
  build: {
    outDir: '../ui/dist',
    emptyOutDir: true,
  },
})
