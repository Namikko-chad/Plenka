import { readFileSync } from 'node:fs'
import { fileURLToPath, URL } from 'node:url'
import vue from '@vitejs/plugin-vue'
import { defineConfig } from 'vitest/config'

const pkg = JSON.parse(readFileSync(new URL('./package.json', import.meta.url), 'utf8')) as { version: string }
const target = process.env.PLENKA_SERVER ?? 'http://localhost:8095'

// Server paths proxied in dev. Exact regexes so they never shadow SPA routes: the
// server owns only /api/*, /img/* and /health, and no SPA route uses those names.
const proxied = ['^/api/', '^/img/', '^/health$']

export default defineConfig({
  base: '/',
  plugins: [vue()],
  define: {
    __APP_VERSION__: JSON.stringify(pkg.version),
  },
  resolve: {
    alias: { '@': fileURLToPath(new URL('./src', import.meta.url)) },
  },
  server: {
    port: 5173,
    host: true,
    proxy: Object.fromEntries(proxied.map((p) => [p, { target, changeOrigin: true }])),
  },
  build: {
    outDir: 'dist',
    target: 'es2020',
    assetsInlineLimit: 0,
    rolldownOptions: {
      output: {
        advancedChunks: { groups: [{ name: 'vendor', test: /node_modules/ }] },
      },
    },
  },
  test: {
    environment: 'happy-dom',
    include: ['src/**/*.test.ts'],
  },
})
