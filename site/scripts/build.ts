/**
 * Builds reputecast.com as static files in dist/:
 *
 *   1. the client bundle (vite build)
 *   2. a server bundle of the same page (vite build --ssr)
 *   3. prerender: renderToString into dist/index.html, so the page is
 *      complete HTML for search engines and readers without JavaScript;
 *      the client bundle hydrates it
 *   4. sitemap.xml and robots.txt for SITE_URL
 *
 *   node scripts/build.ts
 *
 * Environment
 *   SITE_URL             canonical origin (default https://reputecast.com)
 *   SITE_GITHUB_STARS    star count for the header button; when unset the
 *                        build asks the GitHub API once (3 s timeout) and
 *                        falls back to a plain "GitHub" label
 *   SITE_OFFLINE=1       never call the network
 *   SOURCE_DATE_EPOCH    build date for the sitemap (reproducible builds)
 */
import { readdirSync, readFileSync, rmSync, writeFileSync } from 'node:fs'
import { join, resolve } from 'node:path'
import { pathToFileURL } from 'node:url'
import { build } from 'vite'

const root = resolve(import.meta.dirname, '..')
const dist = join(root, 'dist')
const ssrOut = join(root, '.ssr')
const siteUrl = (process.env.SITE_URL ?? 'https://reputecast.com').replace(/\/+$/, '')

async function githubStars(): Promise<string> {
  if (process.env.SITE_GITHUB_STARS) return process.env.SITE_GITHUB_STARS
  if (process.env.SITE_OFFLINE === '1') return ''
  try {
    const r = await fetch('https://api.github.com/repos/hostrepute/reputecast', {
      headers: { Accept: 'application/vnd.github+json', 'User-Agent': 'reputecast-site-build' },
      signal: AbortSignal.timeout(3000),
    })
    if (!r.ok) return ''
    const body = (await r.json()) as { stargazers_count?: number }
    return typeof body.stargazers_count === 'number' ? String(body.stargazers_count) : ''
  } catch {
    return ''
  }
}

/** <link rel=preload> for the fonts the first screen needs. */
function fontPreloads(): string {
  const assets = readdirSync(join(dist, 'assets'))
  const wanted = [/^ibm-plex-sans-latin-400-normal-.*\.woff2$/, /^ibm-plex-sans-latin-600-normal-.*\.woff2$/, /^ibm-plex-sans-latin-700-normal-.*\.woff2$/]
  return wanted
    .map((re) => assets.find((f) => re.test(f)))
    .filter((f): f is string => Boolean(f))
    .map((f) => `<link rel="preload" href="/assets/${f}" as="font" type="font/woff2" crossorigin />`)
    .join('\n    ')
}

function isoDate(): string {
  const epoch = Number(process.env.SOURCE_DATE_EPOCH)
  const d = Number.isFinite(epoch) && epoch > 0 ? new Date(epoch * 1000) : new Date()
  return d.toISOString().slice(0, 10)
}

async function main() {
  const stars = await githubStars()
  process.env.VITE_GITHUB_STARS = stars

  await build({ root, logLevel: 'warn' })
  await build({ root, logLevel: 'warn', build: { ssr: 'src/entry-server.tsx', outDir: ssrOut, emptyOutDir: true } })

  const entry = (await import(pathToFileURL(join(ssrOut, 'entry-server.js')).href)) as { render: () => string }
  const html = entry.render()
  const indexPath = join(dist, 'index.html')
  const template = readFileSync(indexPath, 'utf8')
  if (!template.includes('<!--app-html-->')) throw new Error('dist/index.html has no <!--app-html--> placeholder')
  const page = template.replace('<!--app-html-->', html).replace('</title>', `</title>\n    ${fontPreloads()}`)
  writeFileSync(indexPath, page)

  writeFileSync(
    join(dist, 'sitemap.xml'),
    `<?xml version="1.0" encoding="UTF-8"?>
<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
  <url>
    <loc>${siteUrl}/</loc>
    <lastmod>${isoDate()}</lastmod>
    <changefreq>weekly</changefreq>
    <priority>1.0</priority>
  </url>
</urlset>
`,
  )
  writeFileSync(join(dist, 'robots.txt'), `User-agent: *\nAllow: /\n\nSitemap: ${siteUrl}/sitemap.xml\n`)

  rmSync(ssrOut, { recursive: true, force: true })
  const kb = (Buffer.byteLength(page) / 1024).toFixed(1)
  console.log(`prerendered dist/index.html (${kb} KB)${stars ? `, GitHub stars ${stars}` : ''}`)
}

main().catch((e) => {
  console.error(e)
  process.exit(1)
})
