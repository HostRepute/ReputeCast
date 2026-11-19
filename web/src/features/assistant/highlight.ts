import { htmlLanguage } from '@codemirror/lang-html'
import { liquidLanguage } from '@codemirror/lang-liquid'
import { PostgreSQL } from '@codemirror/lang-sql'
import { classHighlighter, highlightCode } from '@lezer/highlight'

/**
 * Highlights a code block of an answer with the CodeMirror languages the
 * app already has (assistant brief D5): HTML and MJML, Liquid, SQL. A
 * separate chunk, loaded the first time an answer shows code; other
 * languages stay plain.
 */
export interface Piece {
  text: string
  /** classHighlighter's classes ("tok-keyword"), or "" for plain text. */
  cls: string
}

const parsers = {
  html: htmlLanguage.parser,
  liquid: liquidLanguage.parser,
  sql: PostgreSQL.language.parser,
}

const aliases: Record<string, keyof typeof parsers> = {
  html: 'html',
  htm: 'html',
  xml: 'html',
  mjml: 'html',
  liquid: 'liquid',
  sql: 'sql',
  postgres: 'sql',
  postgresql: 'sql',
  pgsql: 'sql',
}

export function supported(lang: string): boolean {
  return lang.toLowerCase() in aliases
}

/** The code's pieces, line breaks as "\n" pieces; null for a language the app doesn't know. */
export function highlight(code: string, lang: string): Piece[] | null {
  const key = aliases[lang.toLowerCase()]
  if (!key) return null
  const parser = parsers[key]
  const out: Piece[] = []
  highlightCode(
    code,
    parser.parse(code),
    classHighlighter,
    (text, cls) => out.push({ text, cls }),
    () => out.push({ text: '\n', cls: '' }),
  )
  return out
}
