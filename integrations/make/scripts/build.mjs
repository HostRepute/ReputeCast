#!/usr/bin/env node
// Bundles the app into dist/reputecast-make.json: app.json's metadata with
// every component's sections inlined, after validating it. It's what goes
// into Make (README › Publish), one file to review or hand to tooling.

import { mkdirSync, readFileSync, writeFileSync, existsSync } from 'node:fs'
import { dirname, join } from 'node:path'
import { fileURLToPath } from 'node:url'

import { validateApp } from './validate.mjs'

const root = join(dirname(fileURLToPath(import.meta.url)), '..')
const src = join(root, 'src')
const json = (p) => JSON.parse(readFileSync(p, 'utf8'))

const SECTIONS = {
  connections: ['api', 'parameters'],
  webhooks: ['api', 'parameters', 'attach', 'detach'],
  rpcs: ['api', 'parameters'],
  modules: ['api', 'parameters', 'expect', 'interface', 'samples'],
}

export function build() {
  const problems = validateApp(root)
  if (problems.length) {
    throw new Error(`The Make app isn't valid:\n- ${problems.join('\n- ')}`)
  }
  const app = json(join(root, 'app.json'))
  const out = {
    app: { name: app.name, label: app.label, description: app.description, version: app.version, release: app.release, theme: app.theme, language: app.language, audience: app.audience },
    base: json(join(src, 'base.imljson')),
    groups: json(join(src, 'groups.imljson')),
    readme: readFileSync(join(src, 'readme.md'), 'utf8'),
  }
  for (const [kind, sections] of Object.entries(SECTIONS)) {
    out[kind] = {}
    for (const [name, meta] of Object.entries(app[kind] ?? {})) {
      const c = { ...meta }
      for (const s of sections) {
        const p = join(src, kind, name, `${s}.imljson`)
        if (existsSync(p)) c[s] = json(p)
      }
      out[kind][name] = c
    }
  }
  mkdirSync(join(root, 'dist'), { recursive: true })
  const file = join(root, 'dist', 'reputecast-make.json')
  writeFileSync(file, `${JSON.stringify(out, null, 2)}\n`)
  return { file, modules: Object.keys(out.modules).length, webhooks: Object.keys(out.webhooks).length, rpcs: Object.keys(out.rpcs).length }
}

if (import.meta.url === `file://${process.argv[1]}`) {
  try {
    const r = build()
    console.log(`Built ${r.file.replace(`${root}/`, '')}: ${r.modules} modules, ${r.webhooks} webhooks, ${r.rpcs} RPCs.`)
  } catch (err) {
    console.error(err.message)
    process.exit(1)
  }
}
