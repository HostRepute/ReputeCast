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

const LAYER_ORDER = '@layer properties, theme, base, components, utilities;'

/**
 * Names Tailwind's cascade layers, in order, before any stylesheet. The build
 * links the CSS of chunks the entry imports before index.css, and a chunk that
 * opened @layer components first would rank it under base: base's resets
 * would then undo every component style. The build fails if the order isn't
 * first in dist/index.html.
 */
function layerOrder(): Plugin {
  let build = false
  return {
    name: 'reputecast-layer-order',
    configResolved(c) {
      build = c.command === 'build'
    },
    transformIndexHtml: () => [{ tag: 'style', children: LAYER_ORDER, injectTo: 'head-prepend' }],
    closeBundle() {
      if (!build) return
      const html = readFileSync(fileURLToPath(new URL('./dist/index.html', import.meta.url)), 'utf8')
      const order = html.indexOf(LAYER_ORDER)
      const sheet = html.search(/<link rel="stylesheet"/)
      if (order < 0 || (sheet >= 0 && order > sheet)) throw new Error('dist/index.html must name the cascade layers before its first stylesheet')
    },
  }
}

export default defineConfig({
  plugins: [
    tanstackRouter({ target: 'react', autoCodeSplitting: true, routesDirectory: './src/routes', generatedRouteTree: './src/routeTree.gen.ts' }),
    react(),
    tailwindcss(),
    layerOrder(),
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
      // Shopify's App URL and allowed redirection URL (pasted into the store's custom app).
      '/integrations/shopify/': { target: backend, changeOrigin: false },
      // Web push (X22): the prompt script, the workers and the click link.
      '/push/': { target: backend, changeOrigin: false },
      '/t/p/': { target: backend, changeOrigin: false },
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
