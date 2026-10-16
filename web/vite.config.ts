import { readdirSync, readFileSync, statSync, writeFileSync } from 'node:fs'
import { join } from 'node:path'
import { brotliCompressSync, constants, gzipSync } from 'node:zlib'
import { fileURLToPath } from 'node:url'
import tailwindcss from '@tailwindcss/vite'
import { tanstackRouter } from '@tanstack/router-plugin/vite'
import react from '@vitejs/plugin-react'
import type { Plugin } from 'vite'
import { defineConfig } from 'vitest/config'

const backend = process.env.REPUTECAST_BACKEND ?? 'http://localhost:8080'

/**
 * Writes .br and .gz next to every text asset so the Go server can serve
 * pre-compressed files, and keeps dist/.gitkeep so `go:embed all:dist`
 * compiles before the first frontend build.
 */
function precompress(): Plugin {
  return {
    name: 'reputecast-precompress',
    apply: 'build',
    closeBundle() {
      const dist = fileURLToPath(new URL('./dist', import.meta.url))
      const walk = (dir: string) => {
        for (const name of readdirSync(dir)) {
          const p = join(dir, name)
          if (statSync(p).isDirectory()) {
            walk(p)
            continue
          }
          if (!/\.(js|css|html|svg|json|txt|xml|webmanifest)$/.test(name)) continue
          const buf = readFileSync(p)
          if (buf.length < 1024) continue
          writeFileSync(`${p}.br`, brotliCompressSync(buf, { params: { [constants.BROTLI_PARAM_QUALITY]: 11 } }))
          writeFileSync(`${p}.gz`, gzipSync(buf, { level: 9 }))
        }
      }
      walk(dist)
      writeFileSync(join(dist, '.gitkeep'), '')
    },
  }
}

export default defineConfig({
  plugins: [
    tanstackRouter({ target: 'react', autoCodeSplitting: true, routesDirectory: './src/routes', generatedRouteTree: './src/routeTree.gen.ts' }),
    react(),
    tailwindcss(),
    precompress(),
  ],
  resolve: {
    alias: { '@': fileURLToPath(new URL('./src', import.meta.url)) },
  },
  server: {
    port: Number(process.env.WEB_PORT ?? 5173),
    strictPort: true,
    proxy: {
      '/api': { target: backend, changeOrigin: false },
      '/hooks': { target: backend, changeOrigin: false },
      '/p/': { target: backend, changeOrigin: false },
      '/media/': { target: backend, changeOrigin: false },
      // SSO and social sign-in callbacks (their URLs are registered at the IdP).
      '/auth/': { target: backend, changeOrigin: false },
    },
  },
  build: {
    target: 'es2023',
    outDir: 'dist',
    emptyOutDir: true,
    assetsInlineLimit: 2048,
    chunkSizeWarningLimit: 1200,
  },
  test: {
    environment: 'jsdom',
    globals: true,
    setupFiles: ['./src/test/setup.ts'],
    include: ['src/**/*.test.{ts,tsx}', 'scripts/**/*.test.ts'],
    css: false,
  },
})
