import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

// https://vite.dev/config/
export default defineConfig({
  plugins: [react()],
  server: {
    port: 5173,
    // Every request to /api goes to the Go server. For the browser the API
    // and the page then share one address (localhost:5173), so the refresh
    // cookie (SameSite=Strict) is sent and no CORS is needed.
    proxy: {
      '/api': 'http://127.0.0.1:5002',
    },
  },
})