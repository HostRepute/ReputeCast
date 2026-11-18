import { createContext, memo, useContext, useEffect, useState, type ComponentProps, type ReactElement, type ReactNode } from 'react'
import Markdown, { type Components } from 'react-markdown'
import remarkGfm from 'remark-gfm'
import { useT } from '@/i18n'
import { useScrollRegion } from '@/lib/hooks'
import type { highlight, Piece, supported } from './highlight'
import { AsstIcon } from './icons'
import { m } from './messages'

/**
 * An answer's Markdown (assistant brief §3.2, D5): paragraphs, lists, GFM
 * tables and code with Copy. Raw HTML shows as the text it is
 * (react-markdown never renders it), Liquid tokens stay as written (as code),
 * remote images are never loaded (each is a link with its domain) and
 * links show their domain and open in a new tab without a referrer.
 */

function domainOf(href: string | undefined): string {
  if (!href) return ''
  try {
    const u = new URL(href, window.location.origin)
    return u.protocol === 'http:' || u.protocol === 'https:' ? u.hostname : ''
  } catch {
    return ''
  }
}

/** Inside a link: an image in its text can't be a second link. */
const InLink = createContext(false)

/**
 * A link: its domain shown, a new tab without a referrer. A link whose
 * scheme isn't safe (javascript:, data:, vbscript:…) arrives without an
 * href (react-markdown's URL transform) and is plain text.
 */
function Anchor({ href, children }: ComponentProps<'a'>) {
  const t = useT()
  const domain = domainOf(href)
  if (!href) return <span>{children}</span>
  return (
    <a href={href} target="_blank" rel="noopener noreferrer" referrerPolicy="no-referrer">
      <InLink.Provider value>{children}</InLink.Provider>
      {domain ? <span className="asst-dom"> ({domain})</span> : null}
      <span className="vh"> {t(m.newTab)}</span>
    </a>
  )
}

/** An image is never fetched: it becomes a link that names its domain (text, inside a link). */
function Img({ src, alt }: ComponentProps<'img'>) {
  const t = useT()
  const inLink = useContext(InLink)
  const href = typeof src === 'string' ? src : ''
  const domain = domainOf(href) || href
  const label = alt ? t(m.image, { alt, domain }) : t(m.imageNoAlt, { domain })
  if (!href || inLink) return <span>{label}</span>
  return (
    <a href={href} target="_blank" rel="noopener noreferrer" referrerPolicy="no-referrer" className="asst-img">
      {label}
      <span className="vh"> {t(m.newTab)}</span>
    </a>
  )
}

function textOf(node: ReactNode): string {
  if (typeof node === 'string' || typeof node === 'number') return String(node)
  if (Array.isArray(node)) return node.map(textOf).join('')
  if (node && typeof node === 'object' && 'props' in node) return textOf((node as ReactElement<{ children?: ReactNode }>).props.children)
  return ''
}

let highlighter: Promise<{ highlight: typeof highlight; supported: typeof supported }> | undefined

/** A code block: highlighted once its language's chunk is in, with Copy. */
export function CodeBlock({ code, lang }: { code: string; lang: string }) {
  const t = useT()
  const [pieces, setPieces] = useState<{ code: string; lang: string; pieces: Piece[] | null }>()
  const [copied, setCopied] = useState(false)
  const [pre, region] = useScrollRegion<HTMLPreElement>(lang ? t(m.codeLabel, { lang }) : t(m.codeLabelPlain))
  useEffect(() => {
    if (!lang) return
    let live = true
    highlighter ??= import('./highlight')
    void highlighter.then((h) => {
      if (live && h.supported(lang)) setPieces({ code, lang, pieces: h.highlight(code, lang) })
    })
    return () => {
      live = false
    }
  }, [code, lang])
  const shown = pieces && pieces.code === code && pieces.lang === lang ? pieces.pieces : null
  const copy = () => {
    void navigator.clipboard?.writeText(code).then(
      () => {
        setCopied(true)
        window.setTimeout(() => setCopied(false), 2000)
      },
      () => setCopied(false),
    )
  }
  return (
    <div className="asst-code">
      <div className="asst-code-bar">
        <span className="asst-code-lang">{lang}</span>
        <button type="button" className="asst-copy" onClick={copy} aria-label={t(m.copy)}>
          <AsstIcon name={copied ? 'check' : 'copy'} size={14} />
          <span aria-hidden="true">{copied ? t(m.copied) : t(m.copy)}</span>
        </button>
        <span className="vh" role="status">
          {copied ? t(m.copied) : ''}
        </span>
      </div>
      <pre ref={pre} {...region}>
        <code>{shown ? shown.map((p, i) => (p.cls ? <span key={i} className={p.cls}>{p.text}</span> : p.text)) : code}</code>
      </pre>
    </div>
  )
}

function Pre({ children }: ComponentProps<'pre'>) {
  const child = (Array.isArray(children) ? children[0] : children) as ReactElement<{ className?: string; children?: ReactNode }> | undefined
  const lang = /language-([\w-]+)/.exec(child?.props?.className ?? '')?.[1] ?? ''
  return <CodeBlock code={textOf(child?.props?.children).replace(/\n$/, '')} lang={lang} />
}

/** A table scrolls inside the answer when it is wider than the panel. */
function Table({ children }: ComponentProps<'table'>) {
  const t = useT()
  const [ref, props] = useScrollRegion<HTMLDivElement>(t(m.tableLabel))
  return (
    <div className="asst-table tscroll" ref={ref} {...props}>
      <table>{children}</table>
    </div>
  )
}

const components: Components = {
  a: ({ href, children }) => <Anchor href={href}>{children}</Anchor>,
  img: ({ src, alt }) => <Img src={src} alt={alt} />,
  pre: ({ children }) => <Pre>{children}</Pre>,
  table: ({ children }) => <Table>{children}</Table>,
  // Headings in an answer sit under the message's own (h3): never page headings.
  h1: ({ children }) => <h4 className="asst-h">{children}</h4>,
  h2: ({ children }) => <h4 className="asst-h">{children}</h4>,
  h3: ({ children }) => <h4 className="asst-h">{children}</h4>,
  h4: ({ children }) => <h4 className="asst-h">{children}</h4>,
  h5: ({ children }) => <h4 className="asst-h">{children}</h4>,
  h6: ({ children }) => <h4 className="asst-h">{children}</h4>,
}

const plugins = [remarkGfm]

const liquidToken = /\{\{[^\n]*?\}\}|\{%[^\n]*?%\}/g

/**
 * Liquid in prose shows as code, exactly as written: Markdown never reads
 * inside it, so a tag holding HTML, an address or asterisks stays the text
 * it is (no link, emphasis or markup). Fenced code and code spans are left
 * as they are.
 */
export function literalLiquid(text: string): string {
  let fence = ''
  return text
    .split('\n')
    .map((line) => {
      const open = /^\s{0,3}(`{3,}|~{3,})/.exec(line)
      if (open) {
        const marker = open[1]!
        if (!fence) fence = marker
        else if (marker[0] === fence[0] && marker.length >= fence.length) fence = ''
        return line
      }
      if (fence || !line.includes('{')) return line
      return line
        .split(/(`+[^`]*`+)/)
        .map((part, i) => (i % 2 ? part : part.replace(liquidToken, (tok) => (tok.includes('`') ? `\`\` ${tok} \`\`` : `\`${tok}\``))))
        .join('')
    })
    .join('\n')
}

/** One Markdown text, parsed again only when it changes. */
export const MarkdownText = memo(function MarkdownText({ text }: { text: string }) {
  return (
    <Markdown remarkPlugins={plugins} components={components}>
      {literalLiquid(text)}
    </Markdown>
  )
})

/**
 * The blocks of a text as it streams: split at blank lines outside code
 * fences, so finished blocks keep their parse and only the last one is
 * parsed again as text arrives.
 */
export function streamBlocks(text: string): string[] {
  const out: string[] = []
  let fence = ''
  let start = 0
  const lines = text.split('\n')
  let at = 0
  for (let i = 0; i < lines.length; i++) {
    const line = lines[i]!
    const open = /^\s{0,3}(`{3,}|~{3,})/.exec(line)
    if (open) {
      const marker = open[1]!
      if (!fence) fence = marker
      else if (marker[0] === fence[0] && marker.length >= fence.length) fence = ''
    }
    at += line.length + 1
    if (!fence && line.trim() === '' && i < lines.length - 1) {
      const block = text.slice(start, at)
      if (block.trim()) out.push(block)
      start = at
    }
  }
  const rest = text.slice(start)
  if (rest.trim()) out.push(rest)
  return out
}

/** A streaming answer: its finished blocks memoized, the last one live. */
export function StreamingMarkdown({ text }: { text: string }) {
  const blocks = streamBlocks(text)
  return (
    <>
      {blocks.map((b, i) => (
        <MarkdownText key={i} text={b} />
      ))}
    </>
  )
}
