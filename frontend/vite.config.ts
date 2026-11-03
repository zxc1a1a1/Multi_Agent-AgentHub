import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

export default defineConfig(({ mode }) => ({
  plugins: [react()],
  // R3-03: opt-in pilot. `vite build --mode streamdown` enables the lazy
  // Streamdown branch; every other mode keeps the legacy renderer.
  define: {
    __STREAMDOWN_ENABLED__: JSON.stringify(mode === 'streamdown'),
  },
  test: {
    environment: 'jsdom',
    globals: true,
    setupFiles: './src/test/setup.ts',
    exclude: ['e2e/**', 'node_modules/**'],
  },
  server: {
    port: 5173,
    host: '0.0.0.0',
    proxy: {
      '/api': {
        target: process.env.VITE_API_URL || 'http://localhost:8080',
        changeOrigin: true,
      },
    },
  },
}))
