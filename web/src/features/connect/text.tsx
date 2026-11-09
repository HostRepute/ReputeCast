import { Fragment, type ReactNode } from 'react'

/**
 * Server text marks code with backticks ("Created SNS topic `reputecast-events`").
 * Ticks renders those spans in the mono face, as the boards do.
 */
export function Ticks({ text, codeClassName = 'mono' }: { text: string; codeClassName?: string }) {
  if (!text.includes('`')) return <>{text}</>
  const parts = text.split('`')
  return (
    <>
      {parts.map((p, i) =>
        i % 2 === 1 ? (
          <span key={i} className={codeClassName}>
            {p}
          </span>
        ) : (
          <Fragment key={i}>{p}</Fragment>
        ),
      )}
    </>
  )
}

/** Plain text without the backtick marks (for titles and labels). */
export function plain(text: string): string {
  return text.replaceAll('`', '')
}

/** Joins nodes with " · ". */
export function dotted(items: ReactNode[]): ReactNode {
  return items.map((it, i) => (
    <Fragment key={i}>
      {i > 0 ? ' · ' : null}
      {it}
    </Fragment>
  ))
}

/**
 * A DNS name or value with a line-break chance after each dot, colon,
 * semicolon and equals sign and before an at sign
 * ("7kq2n3x.|_domainkey.|news.|acme.io"), so a narrow column breaks a record
 * between its parts, not in the middle of "include" or "_domainkey".
 */
export function Breakable({ text }: { text: string }) {
  const parts = text.split(/(?<=[.:;=])|(?=@)/)
  if (parts.length < 2) return <>{text}</>
  return (
    <>
      {parts.map((p, i) => (
        <Fragment key={i}>
          {i > 0 ? <wbr /> : null}
          {p}
        </Fragment>
      ))}
    </>
  )
}
