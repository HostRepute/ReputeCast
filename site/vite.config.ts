import { fileURLToPath } from 'node:url'
import tailwindcss from '@tailwindcss/vite'
import react from '@vitejs/plugin-react'
import type { Plugin } from 'vite'
import { defineConfig } from 'vitest/config'

/** Canonical origin used in meta tags, the sitemap and JSON-LD. */
const siteUrl = (process.env.SITE_URL ?? 'https://reputecast.com').replace(/\/+$/, '')

/** Replaces %SITE_URL% in index.html so previews can build for another origin. */
function siteUrlPlugin(): Plugin {
  return {
    name: 'reputecast-site-url',
    transformIndexHtml: (html) => html.replaceAll('%SITE_URL%', siteUrl),
  }
}

export default defineConfig({
  plugins: [react(), tailwindcss(), siteUrlPlugin()],
  server: {
    port: Number(process.env.SITE_PORT ?? 5288),
    strictPort: true,
    // The design tokens live in the web app (generated from design/design-system/tokens.json).
    fs: { allow: [fileURLToPath(new URL('..', import.meta.url))] },
  },
  preview: {
    port: Number(process.env.SITE_PREVIEW_PORT ?? 5289),
    strictPort: true,
  },
  build: {
    target: 'es2023',
    outDir: 'dist',
    emptyOutDir: true,
    // Fonts and images are separate files so the browser can cache them.
    assetsInlineLimit: 0,
  },
  test: {
    environment: 'jsdom',
    globals: true,
    setupFiles: ['./src/test/setup.ts'],
    include: ['src/**/*.test.{ts,tsx}'],
    css: false,
  },
})
