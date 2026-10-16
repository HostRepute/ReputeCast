/**
 * The translation check behind `pnpm run i18n:check` (scripts/i18n-check.ts).
 *
 * It reads the app's messages without running the app:
 * - English sources: every `defineMessages('namespace', { key: 'text' })`
 *   under web/src;
 * - translations: the `translate(m, { key: 'text' })` default export of a
 *   `*messages.<lang>.ts` file (or one translate() per group of messages,
 *   spread together), matched to its source through the import;
 * - the server's catalogs: internal/<package>/messages/<lang>.json, with
 *   en.json as the source.
 *
 * Errors (exit 1): a message that isn't valid ICU MessageFormat, a
 * translation that uses an argument or a tag its source doesn't have, a
 * translation key the source doesn't have, or one message ID defined twice
 * with different English. Missing translations aren't errors: they show in
 * English, and the language stays out of the picker until it is complete.
 */
import { existsSync, readdirSync, readFileSync, statSync } from 'node:fs'
import { join, relative, resolve, dirname } from 'node:path'
import { parse, TYPE, type MessageFormatElement } from '@formatjs/icu-messageformat-parser'
import ts from 'typescript'

/** A file to check: its path (relative to the repository) and text. */
export interface SourceFile {
  path: string
  text: string
}

export interface Message {
  id: string
  namespace: string
  key: string
  text: string
  file: string
  side: 'web' | 'server'
}

export interface LanguageStatus {
  code: string
  name: string
  total: number
  translated: number
  /** Missing message IDs by the file that defines their English. */
  missing: Map<string, string[]>
}

/** Text a person reads that skipped web/src/i18n, so it's English in every language. */
export interface HardCoded {
  file: string
  line: number
  text: string
}

export interface Report {
  messages: Message[]
  languages: LanguageStatus[]
  /** Hard-coded text in the app's screens. */
  hardCoded: HardCoded[]
  errors: string[]
  /**
   * The languages the picker offers in a release: English, and every
   * translation with nothing missing once no hard-coded text is left.
   */
  complete: string[]
}

// ---------------------------------------------------------------- ICU

interface Shape {
  args: Set<string>
  tags: Set<string>
}

function shapeOf(els: MessageFormatElement[], out: Shape = { args: new Set(), tags: new Set() }): Shape {
  for (const el of els) {
    switch (el.type) {
      case TYPE.argument:
      case TYPE.number:
      case TYPE.date:
      case TYPE.time:
        out.args.add(el.value)
        break
      case TYPE.select:
      case TYPE.plural:
        out.args.add(el.value)
        for (const opt of Object.values(el.options)) shapeOf(opt.value, out)
        break
      case TYPE.tag:
        out.tags.add(el.value)
        shapeOf(el.children, out)
        break
    }
  }
  return out
}

/** Parses an ICU message; returns its arguments and tags, or the parse error. */
export function icuShape(text: string): Shape | string {
  try {
    return shapeOf(parse(text, { requiresOtherClause: true }))
  } catch (e) {
    return e instanceof Error ? e.message : String(e)
  }
}

// ---------------------------------------------------------------- TypeScript sources

function stringValue(node: ts.Expression): string | undefined {
  if (ts.isStringLiteral(node) || ts.isNoSubstitutionTemplateLiteral(node)) return node.text
  if (ts.isParenthesizedExpression(node)) return stringValue(node.expression)
  if (ts.isBinaryExpression(node) && node.operatorToken.kind === ts.SyntaxKind.PlusToken) {
    const a = stringValue(node.left)
    const b = stringValue(node.right)
    return a !== undefined && b !== undefined ? a + b : undefined
  }
  return undefined
}

function propName(p: ts.ObjectLiteralElementLike): string | undefined {
  if (!ts.isPropertyAssignment(p)) return undefined
  if (ts.isIdentifier(p.name) || ts.isStringLiteral(p.name) || ts.isNumericLiteral(p.name)) return p.name.text
  return undefined
}

interface Definition {
  namespace: string
  /** The variable the descriptors are exported as (m, common…). */
  binding?: string
  keys: Map<string, string>
}

function parseTs(f: SourceFile): ts.SourceFile {
  return ts.createSourceFile(f.path, f.text, ts.ScriptTarget.Latest, true, f.path.endsWith('.tsx') ? ts.ScriptKind.TSX : ts.ScriptKind.TS)
}

/** The defineMessages calls of a file. */
export function definitionsIn(f: SourceFile, errors: string[]): Definition[] {
  const out: Definition[] = []
  const visit = (node: ts.Node) => {
    if (ts.isCallExpression(node) && ts.isIdentifier(node.expression) && node.expression.text === 'defineMessages') {
      const [ns, obj] = node.arguments
      const namespace = ns ? stringValue(ns) : undefined
      if (namespace !== undefined && obj && ts.isObjectLiteralExpression(obj)) {
        const keys = new Map<string, string>()
        for (const p of obj.properties) {
          const name = propName(p)
          const text = name !== undefined && ts.isPropertyAssignment(p) ? stringValue(p.initializer) : undefined
          if (name === undefined || text === undefined) {
            errors.push(`${f.path}: ${namespace}: every message must be a plain string`)
            continue
          }
          keys.set(name, text)
        }
        const decl = node.parent
        const binding = ts.isVariableDeclaration(decl) && ts.isIdentifier(decl.name) ? decl.name.text : undefined
        out.push({ namespace, binding, keys })
      }
    }
    ts.forEachChild(node, visit)
  }
  visit(parseTs(f))
  return out
}

interface Translation {
  /** The imported binding passed to translate(). */
  binding: string
  /** Where it is imported from, resolved to a repository path. */
  from?: string
  keys: Map<string, string>
}

const TS_EXT = ['.ts', '.tsx', '/index.ts', '/index.tsx']

function resolveImport(fromFile: string, spec: string, files: Map<string, SourceFile>): string | undefined {
  let base: string
  if (spec.startsWith('@/')) base = join('web/src', spec.slice(2))
  else if (spec.startsWith('.')) base = join(dirname(fromFile), spec)
  else return undefined
  if (files.has(base)) return base
  for (const ext of TS_EXT) if (files.has(base + ext)) return base + ext
  return undefined
}

/**
 * The translate() calls of a translation file: one per group of messages
 * its source defines (`export default { ...translate(mb, {…}), ...translate(m, {…}) }`).
 */
export function translationsIn(f: SourceFile, files: Map<string, SourceFile>, errors: string[]): Translation[] {
  const sf = parseTs(f)
  const imports = new Map<string, { from: string; name: string }>()
  for (const st of sf.statements) {
    if (!ts.isImportDeclaration(st) || !ts.isStringLiteral(st.moduleSpecifier)) continue
    const named = st.importClause?.namedBindings
    if (!named || !ts.isNamedImports(named)) continue
    for (const el of named.elements) imports.set(el.name.text, { from: st.moduleSpecifier.text, name: (el.propertyName ?? el.name).text })
  }
  const found: Translation[] = []
  const visit = (node: ts.Node) => {
    if (ts.isCallExpression(node) && ts.isIdentifier(node.expression) && node.expression.text === 'translate') {
      const [src, obj] = node.arguments
      if (src && ts.isIdentifier(src) && obj && ts.isObjectLiteralExpression(obj)) {
        const imp = imports.get(src.text)
        const keys = new Map<string, string>()
        for (const p of obj.properties) {
          const name = propName(p)
          const text = name !== undefined && ts.isPropertyAssignment(p) ? stringValue(p.initializer) : undefined
          if (name === undefined || text === undefined) {
            errors.push(`${f.path}: every translation must be a plain string`)
            continue
          }
          keys.set(name, text)
        }
        found.push({ binding: imp?.name ?? src.text, from: imp ? resolveImport(f.path, imp.from, files) : f.path, keys })
      }
    }
    ts.forEachChild(node, visit)
  }
  visit(sf)
  return found
}

// ---------------------------------------------------------------- hard-coded text

/** Attributes whose plain string is text a person reads. */
const TEXT_ATTRIBUTES = new Set([
  'title',
  'label',
  'caption',
  'description',
  'placeholder',
  'aria-label',
  'alt',
  'subtitle',
  'eyebrow',
  'detail',
  'disabledReason',
  'hint',
  'help',
  'tooltip',
])

/** Names that read the same in every language and stand alone in the markup. */
const NAMES = new Set(['ReputeCast', 'HostRepute'])

function readable(text: string): boolean {
  return /\p{L}{2,}/u.test(text) && !NAMES.has(text)
}

/**
 * The JSX text and the plain strings in text attributes of a screen (.tsx):
 * what a screen converted to messages no longer has. A literal that isn't
 * text to translate, such as the format "xxxx-xxxx-xxxx", goes in an
 * expression: placeholder={'xxxx-xxxx-xxxx'}.
 */
export function hardCodedIn(f: SourceFile): HardCoded[] {
  const out: HardCoded[] = []
  const sf = parseTs(f)
  const lineOf = (pos: number) => sf.getLineAndCharacterOfPosition(pos).line + 1
  const visit = (node: ts.Node) => {
    if (ts.isJsxText(node)) {
      const text = node.text.replace(/\s+/g, ' ').trim()
      if (text && readable(text)) out.push({ file: f.path, line: lineOf(node.getStart(sf) + node.text.length - node.text.trimStart().length), text })
    } else if (ts.isJsxAttribute(node) && node.initializer && ts.isStringLiteral(node.initializer)) {
      const text = node.initializer.text.trim()
      if (TEXT_ATTRIBUTES.has(node.name.getText(sf)) && readable(text)) out.push({ file: f.path, line: lineOf(node.getStart(sf)), text })
    }
    ts.forEachChild(node, visit)
  }
  visit(sf)
  return out
}

/** The screens the hard-coded scan reads: the app's .tsx files, not its tests. */
function isScreen(path: string): boolean {
  return path.startsWith('web/src/') && path.endsWith('.tsx') && !/\.test\.tsx$/.test(path) && !path.startsWith('web/src/test/')
}

// ---------------------------------------------------------------- the check

/** `feature.messages.vi.ts` and `messages.vi.ts` are Vietnamese. */
const TRANSLATION_FILE = /(?:^|[./])messages\.([a-z]{2,3}(?:-[A-Za-z0-9]+)?)\.ts$/

/** The app languages, from APP_LANGUAGES in web/src/i18n/languages.ts. */
export function appLanguages(f: SourceFile | undefined): { code: string; name: string }[] {
  const out: { code: string; name: string }[] = []
  if (!f) return [{ code: 'en', name: 'English' }]
  const visit = (node: ts.Node) => {
    if (ts.isVariableDeclaration(node) && ts.isIdentifier(node.name) && node.name.text === 'APP_LANGUAGES' && node.initializer) {
      let init: ts.Expression = node.initializer
      while (ts.isAsExpression(init) || ts.isSatisfiesExpression(init)) init = init.expression
      if (ts.isArrayLiteralExpression(init)) {
        for (const el of init.elements) {
          if (!ts.isObjectLiteralExpression(el)) continue
          let code: string | undefined
          let name: string | undefined
          for (const p of el.properties) {
            if (!ts.isPropertyAssignment(p)) continue
            if (propName(p) === 'code') code = stringValue(p.initializer)
            if (propName(p) === 'name') name = stringValue(p.initializer)
          }
          if (code) out.push({ code, name: name ?? code })
        }
      }
    }
    ts.forEachChild(node, visit)
  }
  visit(parseTs(f))
  return out.length ? out : [{ code: 'en', name: 'English' }]
}

/**
 * Checks the messages of the given files. Paths are relative to the
 * repository: web/src/… for the app, internal/<pkg>/messages/<lang>.json
 * for the server.
 */
export function check(input: SourceFile[]): Report {
  const errors: string[] = []
  const files = new Map(input.map((f) => [f.path, f]))
  const messages: Message[] = []
  const byId = new Map<string, Message>()
  /** Source definitions by file and binding, for translate(). */
  const bindings = new Map<string, Definition>()
  const sourceShapes = new Map<string, Shape>()
  /** Translated IDs per language. */
  const translated = new Map<string, Set<string>>()
  const mark = (lang: string, id: string) => {
    let s = translated.get(lang)
    if (!s) translated.set(lang, (s = new Set()))
    s.add(id)
  }

  const add = (m: Message) => {
    const known = byId.get(m.id)
    if (known && known.text !== m.text) errors.push(`${m.file}: ${m.id} is also defined in ${known.file} with different English`)
    if (known) return
    byId.set(m.id, m)
    messages.push(m)
    const shape = icuShape(m.text)
    if (typeof shape === 'string') errors.push(`${m.file}: ${m.id}: ${shape}`)
    else sourceShapes.set(m.id, shape)
  }

  const checkTranslation = (file: string, lang: string, id: string, text: string) => {
    const shape = icuShape(text)
    if (typeof shape === 'string') {
      errors.push(`${file}: ${id}: ${shape}`)
      return
    }
    const src = sourceShapes.get(id)
    if (src) {
      for (const a of shape.args) if (!src.args.has(a)) errors.push(`${file}: ${id} uses {${a}}, which the English doesn't`)
      for (const t of shape.tags) if (!src.tags.has(t)) errors.push(`${file}: ${id} uses <${t}>, which the English doesn't`)
    }
    mark(lang, id)
  }

  // The app's English sources.
  for (const f of input) {
    if (!f.path.startsWith('web/src/') || !/\.tsx?$/.test(f.path) || /\.test\.tsx?$|\.d\.ts$|\.gen\.ts$/.test(f.path) || TRANSLATION_FILE.test(f.path)) continue
    if (!f.text.includes('defineMessages(')) continue
    for (const d of definitionsIn(f, errors)) {
      if (d.binding) bindings.set(`${f.path}#${d.binding}`, d)
      for (const [key, text] of d.keys) add({ id: `${d.namespace}.${key}`, namespace: d.namespace, key, text, file: f.path, side: 'web' })
    }
  }

  // The app's translations.
  for (const f of input) {
    const lang = f.path.startsWith('web/src/') ? TRANSLATION_FILE.exec(f.path)?.[1] : undefined
    if (!lang || lang === 'en') continue
    const trs = translationsIn(f, files, errors)
    if (!trs.length) {
      errors.push(`${f.path}: no translate(m, { … }) call`)
      continue
    }
    for (const tr of trs) {
      const def = tr.from ? bindings.get(`${tr.from}#${tr.binding}`) : undefined
      if (!def) {
        errors.push(`${f.path}: can't find the messages ${tr.binding} it translates`)
        continue
      }
      for (const [key, text] of tr.keys) {
        if (!def.keys.has(key)) {
          errors.push(`${f.path}: ${def.namespace}.${key} isn't in the English`)
          continue
        }
        checkTranslation(f.path, lang, `${def.namespace}.${key}`, text)
      }
    }
  }

  // The server's catalogs.
  const catalogs = new Map<string, Map<string, SourceFile>>()
  for (const f of input) {
    const m = /^internal\/(.+)\/messages\/([a-zA-Z-]+)\.json$/.exec(f.path)
    if (!m) continue
    let c = catalogs.get(m[1]!)
    if (!c) catalogs.set(m[1]!, (c = new Map()))
    c.set(m[2]!, f)
  }
  const entries = (f: SourceFile): Map<string, string> => {
    const out = new Map<string, string>()
    let raw: Record<string, unknown>
    try {
      raw = JSON.parse(f.text) as Record<string, unknown>
    } catch (e) {
      errors.push(`${f.path}: ${e instanceof Error ? e.message : String(e)}`)
      return out
    }
    for (const [k, v] of Object.entries(raw)) {
      const text = typeof v === 'string' ? v : v && typeof v === 'object' && typeof (v as { message?: unknown }).message === 'string' ? (v as { message: string }).message : undefined
      if (text === undefined) errors.push(`${f.path}: ${k} must be a string or { "message": … }`)
      else out.set(k, text)
    }
    return out
  }
  for (const [pkg, langs] of catalogs) {
    const en = langs.get('en')
    if (!en) {
      errors.push(`internal/${pkg}/messages: en.json is missing`)
      continue
    }
    const namespace = pkg.split('/').pop()!
    for (const [key, text] of entries(en)) add({ id: `server:${pkg}.${key}`, namespace, key, text, file: en.path, side: 'server' })
    for (const [lang, f] of langs) {
      if (lang === 'en') continue
      for (const [key, text] of entries(f)) {
        const id = `server:${pkg}.${key}`
        if (!byId.has(id)) {
          errors.push(`${f.path}: ${key} isn't in en.json`)
          continue
        }
        checkTranslation(f.path, lang, id, text)
      }
    }
  }

  const languages: LanguageStatus[] = []
  const app = appLanguages(files.get('web/src/i18n/languages.ts'))
  const codes = new Set([...app.map((l) => l.code), ...translated.keys()])
  for (const code of codes) {
    if (code === 'en') continue
    const done = translated.get(code) ?? new Set<string>()
    const missing = new Map<string, string[]>()
    for (const m of messages) {
      if (done.has(m.id)) continue
      const list = missing.get(m.file) ?? []
      list.push(m.side === 'server' ? `${m.namespace}.${m.key}` : m.id)
      missing.set(m.file, list)
    }
    languages.push({ code, name: app.find((l) => l.code === code)?.name ?? code, total: messages.length, translated: done.size, missing })
  }
  // Hard-coded text reads in English whatever the language, so no
  // translation is complete while some is left.
  const hardCoded = input.filter((f) => isScreen(f.path)).flatMap(hardCodedIn)
  const complete = [
    'en',
    ...(hardCoded.length ? [] : languages.filter((l) => l.translated === l.total && app.some((a) => a.code === l.code)).map((l) => l.code)),
  ]
  return { messages, languages, hardCoded, errors, complete }
}

// ---------------------------------------------------------------- files and output

/** Reads the files the check looks at, under the repository root. */
export function collect(repo: string): SourceFile[] {
  const out: SourceFile[] = []
  const walk = (dir: string, keep: (p: string) => boolean) => {
    if (!existsSync(dir)) return
    for (const name of readdirSync(dir)) {
      if (name === 'node_modules' || name.startsWith('.')) continue
      const p = join(dir, name)
      if (statSync(p).isDirectory()) walk(p, keep)
      else {
        const rel = relative(repo, p).split('\\').join('/')
        if (keep(rel)) out.push({ path: rel, text: readFileSync(p, 'utf8') })
      }
    }
  }
  walk(resolve(repo, 'web/src'), (p) => /\.tsx?$/.test(p))
  walk(resolve(repo, 'internal'), (p) => /\/messages\/[a-zA-Z-]+\.json$/.test(p))
  return out
}

/** The generated status module (web/src/i18n/status.gen.ts). */
export function statusModule(report: Report): string {
  const list = report.complete.map((c) => `'${c}'`).join(', ')
  return [
    "// Generated by scripts/i18n-check.ts (`pnpm run i18n:check`, and every build). Don't edit it.",
    '',
    '/** Languages whose translation is complete in this release; the picker shows only these outside development. */',
    `export const COMPLETE_LANGUAGES: readonly string[] = [${list}]`,
    '',
  ].join('\n')
}

/**
 * A readable summary: per language, what is missing, by file; then the
 * hard-coded text, by file (each string with `hardCoded`).
 */
export function summary(report: Report, { verbose = true, hardCoded = false } = {}): string {
  const web = report.messages.filter((m) => m.side === 'web').length
  const lines = [`${report.messages.length} messages: ${web} in the app, ${report.messages.length - web} on the server.`]
  for (const l of report.languages) {
    const pct = l.total ? Math.floor((l.translated / l.total) * 1000) / 10 : 100
    const left = l.total - l.translated
    lines.push('', `${l.name} (${l.code}): ${l.translated} of ${l.total} translated (${pct}%)${left ? `, ${left} missing` : ', complete'}`)
    if (!verbose) continue
    for (const [file, ids] of [...l.missing].sort(([a], [b]) => a.localeCompare(b))) {
      lines.push(`  ${file} · ${ids.length}`)
      for (const id of ids) lines.push(`    ${id}`)
    }
  }
  const hc = report.hardCoded
  if (!hc.length) lines.push('', 'Hard-coded text: none')
  else {
    const byFile = new Map<string, HardCoded[]>()
    for (const h of hc) byFile.set(h.file, [...(byFile.get(h.file) ?? []), h])
    const strings = `${hc.length} ${hc.length === 1 ? 'string' : 'strings'} in ${byFile.size} ${byFile.size === 1 ? 'file' : 'files'}`
    lines.push('', `Hard-coded text: ${strings}, in English in every language${hardCoded ? '' : ' (--hardcoded lists them)'}`)
    if (verbose || hardCoded) {
      for (const [file, list] of [...byFile].sort(([x], [y]) => x.localeCompare(y))) {
        lines.push(`  ${file} · ${list.length}`)
        if (hardCoded) for (const h of list) lines.push(`    ${h.line}: ${h.text}`)
      }
    }
  }
  const waiting = report.languages.filter((l) => l.translated === l.total && !report.complete.includes(l.code)).map((l) => l.name)
  const why = waiting.length ? ` (${waiting.join(', ')} ${waiting.length === 1 ? 'waits' : 'wait'} for the hard-coded text)` : ''
  lines.push('', `In the language picker: ${report.complete.join(', ')}${why}`)
  if (report.errors.length) lines.push('', `${report.errors.length} error${report.errors.length === 1 ? '' : 's'}:`, ...report.errors.map((e) => `  ${e}`))
  return lines.join('\n')
}
