import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

export default defineConfig({
  plugins: [react()],
  server: {
    port: 5173,
    proxy: {
      '/api': {
        target: process.env.VITE_API_URL || 'http://localhost:8080',
        changeOrigin: true,
      },
    },
  },
  build: {
    rollupOptions: {
      output: {
        manualChunks(id) {
          if (id.includes('node_modules/react/') || id.includes('node_modules/react-dom/')) return 'vendor-react'
          if (id.includes('node_modules/@mui/') || id.includes('node_modules/@emotion/')) return 'vendor-mui'
          if (id.includes('node_modules/@tiptap/') || id.includes('node_modules/tiptap')) return 'vendor-tiptap'
          if (id.includes('node_modules/html2pdf')) return 'vendor-pdf'
        },
      },
    },
    chunkSizeWarningLimit: 600,
  },
})
