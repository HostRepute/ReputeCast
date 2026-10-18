import { existsSync, readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import { describe, expect, it } from 'vitest'
import { check, collect, icuShape, statusModule, summary, type SourceFile } from './i18n-check.ts'

const languages: SourceFile = {
  path: 'web/src/i18n/languages.ts',
  text: `export const APP_LANGUAGES: readonly AppLanguage[] = [
  { code: 'en', name: 'English' },
  { code: 'vi', name: 'Tiếng Việt' },
]`,
}

const source: SourceFile = {
  path: 'web/src/features/transactional/messages.ts',
  text: `import { defineMessages } from '@/i18n'
export const m = defineMessages('transactional', {
  title: 'Transactional',
  sent: '{count, plural, one {# message} other {# messages}} sent to <b>{name}</b>',
  long: 'Two ' + 'parts',
})`,
}

function vi(body: string): SourceFile {
  return { path: 'web/src/features/transactional/messages.vi.ts', text: `import { translate } from '@/i18n'\nimport { m } from './messages'\nexport default translate(m, {\n${body}\n})` }
}

describe('i18n:check', () => {
  it('lists what a language misses, by file', () => {
    const r = check([languages, source, vi(`title: 'Giao dịch'`)])
    expect(r.errors).toEqual([])
    expect(r.messages.map((x) => x.id)).toEqual(['transactional.title', 'transactional.sent', 'transactional.long'])
    const l = r.languages.find((x) => x.code === 'vi')!
    expect(l.translated).toBe(1)
    expect(l.missing.get(source.path)).toEqual(['transactional.sent', 'transactional.long'])
    expect(r.complete).toEqual(['en'])
    expect(summary(r)).toContain('Tiếng Việt (vi): 1 of 3 translated (33.3%), 2 missing')
  })

  it('offers a language in the picker once it is complete', () => {
    const r = check([languages, source, vi(`title: 'Giao dịch', sent: 'Đã gửi {count, plural, other {# thư}} cho <b>{name}</b>', long: 'Hai phần'`)])
    expect(r.errors).toEqual([])
    expect(r.complete).toEqual(['en', 'vi'])
    expect(statusModule(r)).toContain("COMPLETE_LANGUAGES: readonly string[] = ['en', 'vi']")
  })

  it('reports broken messages, unknown arguments, tags and keys', () => {
    const r = check([
      languages,
      source,
      vi(`title: 'Giao dịch {oops', sent: 'Đã gửi {total} cho <i>{name}</i>', extra: 'Thêm'`),
      { path: 'web/src/features/other/messages.ts', text: `export const m = defineMessages('transactional', { title: 'Something else' })` },
    ])
    expect(r.errors.some((e) => e.includes('transactional.title') && e.includes('messages.vi.ts'))).toBe(true)
    expect(r.errors).toContain('web/src/features/transactional/messages.vi.ts: transactional.sent uses {total}, which the English doesn\'t')
    expect(r.errors).toContain('web/src/features/transactional/messages.vi.ts: transactional.sent uses <i>, which the English doesn\'t')
    expect(r.errors).toContain("web/src/features/transactional/messages.vi.ts: transactional.extra isn't in the English")
    expect(r.errors).toContain('web/src/features/other/messages.ts: transactional.title is also defined in web/src/features/transactional/messages.ts with different English')
  })

  it('counts hard-coded text, and keeps a complete translation out of the picker while some is left', () => {
    const screen: SourceFile = {
      path: 'web/src/features/transactional/Page.tsx',
      text: `export function Page() {
  return (
    <section title="Templates" data-x="ignored">
      <h1>{t(m.title)}</h1>
      <p>
        Nothing here yet
      </p>
      <span>·</span>
      <b>{'xxxx-xxxx'}</b>
      <i>ReputeCast</i>
      <input placeholder="Search templates" aria-label={label} />
    </section>
  )
}`,
    }
    const complete = vi(`title: 'Giao dịch', sent: 'Đã gửi {count, plural, other {# thư}} cho <b>{name}</b>', long: 'Hai phần'`)
    const r = check([languages, source, complete, screen])
    expect(r.hardCoded).toEqual([
      { file: screen.path, line: 3, text: 'Templates' },
      { file: screen.path, line: 6, text: 'Nothing here yet' },
      { file: screen.path, line: 11, text: 'Search templates' },
    ])
    expect(r.complete).toEqual(['en'])
    const short = summary(r, { verbose: false })
    expect(short).toContain('Hard-coded text: 3 strings in 1 file, in English in every language')
    expect(short).toContain('In the language picker: en (Tiếng Việt waits for the hard-coded text)')
    expect(summary(r, { hardCoded: true })).toContain('    6: Nothing here yet')
    // Tests aren't screens.
    expect(check([languages, source, complete, { ...screen, path: 'web/src/features/transactional/Page.test.tsx' }]).complete).toEqual(['en', 'vi'])
  })

  it('reads every group a translation file translates', () => {
    const two: SourceFile = {
      path: 'web/src/features/editor/messages.ts',
      text: `import { defineMessages } from '@/i18n'
export const mb = defineMessages('editor.blocks', { footer: 'Footer' })
export const m = defineMessages('editor', { title: 'Editor', save: 'Save' })`,
    }
    const r = check([
      languages,
      two,
      {
        path: 'web/src/features/editor/messages.vi.ts',
        text: `import { translate } from '@/i18n'
import { m, mb } from './messages'
export default { ...translate(mb, { footer: 'Chân trang' }), ...translate(m, { title: 'Trình soạn thảo', nope: 'Không' }) }`,
      },
    ])
    expect(r.errors).toEqual(["web/src/features/editor/messages.vi.ts: editor.nope isn't in the English"])
    const l = r.languages.find((x) => x.code === 'vi')!
    expect(l.translated).toBe(2)
    expect(l.missing.get(two.path)).toEqual(['editor.save'])
  })

  it('checks the server catalogs too', () => {
    const r = check([
      languages,
      { path: 'internal/membermail/messages/en.json', text: '{ "invite.subject": "{inviter} invited you", "reset.action": { "message": "Set a new password", "note": "a button" } }' },
      { path: 'internal/membermail/messages/vi.json', text: '{ "invite.subject": "{inviter} đã mời bạn", "unknown": "?" }' },
    ])
    expect(r.errors).toEqual(["internal/membermail/messages/vi.json: unknown isn't in en.json"])
    expect(r.languages.find((x) => x.code === 'vi')!.missing.get('internal/membermail/messages/en.json')).toEqual(['membermail.reset.action'])
  })

  it('parses ICU arguments and tags', () => {
    expect(icuShape('Ask <b>{name}</b> · {n, plural, one {# day} other {# days}}')).toEqual({ args: new Set(['name', 'n']), tags: new Set(['b']) })
    expect(typeof icuShape('{broken')).toBe('string')
  })

  // It reads every catalog in the repository, which takes longer than
  // Vitest's 5 seconds on a busy machine.
  it('keeps status.gen.ts up to date with the catalogs', () => {
    // Vitest runs from web/ (its config); the files are read from the repository.
    const web = existsSync('scripts/lib/i18n-check.ts') ? process.cwd() : resolve(process.cwd(), 'web')
    const repo = resolve(web, '..')
    const r = check(collect(repo))
    expect(r.errors).toEqual([])
    const committed = readFileSync(resolve(web, 'src/i18n/status.gen.ts'), 'utf8')
    expect(committed, 'run pnpm run i18n:check and commit src/i18n/status.gen.ts').toBe(statusModule(r))
  }, 60_000)
})
