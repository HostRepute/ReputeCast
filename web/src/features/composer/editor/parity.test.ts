import type { Editor } from 'grapesjs'
import { existsSync, readdirSync, readFileSync, writeFileSync } from 'node:fs'
import { resolve } from 'node:path'
import { afterAll, beforeAll, describe, expect, it } from 'vitest'
import { installMjmlHooks, type MjmlCompile } from './mjmlHooks'

/**
 * gomjml ↔ mjml-browser parity (CO-3). The server checks the HTML the
 * composer stores against its own compile (internal/mjml.Verify): links,
 * images, text and conditional markers must agree. These fixtures are
 * compiled here exactly as the composer compiles (the mjml-browser bundled
 * with grapesjs-mjml, through installMjmlHooks), and the results are kept
 * next to the fixtures; the Go side checks that gomjml agrees with each.
 * Run with UPDATE_PARITY=1 to write the results after an MJML upgrade.
 */
// Tests run from web/.
const dir = `${resolve(process.cwd(), '../internal/mjml/testdata/parity')}/`
const update = process.env.UPDATE_PARITY === '1'

let editor: Editor
let compile: MjmlCompile

beforeAll(async () => {
  const { default: grapesjs } = await import('grapesjs')
  const mjmlModule = await import('grapesjs-mjml')
  const mjmlPlugin = ((mjmlModule.default as unknown as { default?: typeof mjmlModule.default }).default ?? mjmlModule.default) as typeof mjmlModule.default
  const container = document.createElement('div')
  document.body.appendChild(container)
  editor = grapesjs.init({
    container,
    headless: true,
    storageManager: false,
    telemetry: false,
    plugins: [
      (ed: Editor) =>
        mjmlPlugin(ed, {
          blocks: [],
          resetBlocks: true,
          customComponents: [
            (_e, o) => {
              compile = installMjmlHooks(ed, o, () => '')
            },
          ],
        }),
    ],
  })
})

afterAll(() => editor?.destroy())

const fixtures = readdirSync(dir)
  .filter((f) => f.endsWith('.mjml'))
  .sort()

describe('mjml-browser output of the parity fixtures', () => {
  it('has fixtures', () => {
    expect(fixtures.length).toBeGreaterThan(5)
  })
  for (const name of fixtures) {
    it(name, () => {
      const src = readFileSync(`${dir}${name}`, 'utf8')
      const out = compile(src)
      expect(out.errors.filter((e) => !/attribute|has invalid value/i.test(e.message))).toEqual([])
      const file = `${dir}browser/${name.replace(/\.mjml$/, '.html')}`
      if (update) writeFileSync(file, out.html)
      // The kept output is what the Go parity test checks gomjml against.
      expect(existsSync(file), `${file} is missing: run with UPDATE_PARITY=1`).toBe(true)
      expect(out.html).toBe(readFileSync(file, 'utf8'))
    })
  }
})
