import type { ReactNode } from 'react'

/**
 * A preset hint as on the boards: the IdP's menu path in bold ("open
 * Token configuration → Add groups claim") and URLs in mono. The server
 * writes the hint in the app language, with the path right after the word
 * that opens it and up to the word that goes on: "open … and/to" in
 * English, "mở … và/rồi" in Vietnamese, "abre … y/para" in Spanish,
 * "ouvrez … et/pour" in French, "zu … und/um" in German.
 */
export function richHint(text: string): ReactNode {
  const parts: ReactNode[] = []
  let key = 0
  const pushText = (s: string) => {
    const m = /(?:open|mở|abre|ouvrez|zu) ((?:[^.→]+ → )+[^.→]+?)(?= and | to | và | rồi | y | para | et | pour | und | um |,|\.)/.exec(s)
    if (!m) {
      parts.push(s)
      return
    }
    const start = m.index + m[0].length - m[1]!.length
    parts.push(s.slice(0, start), <b key={key++}>{m[1]}</b>, s.slice(start + m[1]!.length))
  }
  let last = 0
  for (const m of text.matchAll(/https?:\/\/[^\s,]+[^\s,.]/g)) {
    pushText(text.slice(last, m.index))
    parts.push(
      <span key={key++} className="mono">
        {m[0]}
      </span>,
    )
    last = m.index + m[0].length
  }
  pushText(text.slice(last))
  return <>{parts}</>
}

/** "…/acme/v2.0/.well-known/openid-configuration" */
export function shortURL(u?: string) {
  if (!u) return ''
  try {
    const x = new URL(u)
    return `…${x.pathname}`
  } catch {
    return u
  }
}
