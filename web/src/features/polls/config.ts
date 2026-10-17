/**
 * The Poll block's settings and MJML (design expansion, X11). Keep in step
 * with internal/polls/config.go: the server reads the settings from the
 * section's rc-poll attribute, and both sides write the same MJML.
 */

export type PollKind = 'choice' | 'scale' | 'emoji'

export interface PollAnswerConfig {
  label: string
  emoji?: string
  tag?: string
}

export interface PollClose {
  mode: 'days' | 'date' | 'manual'
  days?: number
  /** "2026-10-28": the poll closes at the end of that day. */
  date?: string
}

export interface PollConfig {
  id: string
  kind: PollKind
  question: string
  answers: PollAnswerConfig[]
  layout: 'stacked' | 'row'
  save: 'none' | 'tag' | 'field'
  field?: string
  close: PollClose
  after: 'thanks' | 'survey'
  thanks?: string
  survey?: string
  nonContacts: 'count' | 'refuse'
  results: boolean
  nps?: boolean
  low?: string
  high?: string
  faces?: number
}

export interface PollAnswer {
  key: number
  label: string
  emoji?: string
  tag?: string
}

export const POLL_ACCENT = '#0F766E'
const SERIF = "Georgia, 'Times New Roman', serif"
const SANS = "'IBM Plex Sans', Helvetica, Arial, sans-serif"

const FACE_WORDS: Record<number, string[]> = {
  3: ['unhappy', 'neutral', 'happy'],
  5: ['very unhappy', 'unhappy', 'neutral', 'happy', 'very happy'],
}

/** The words of an emoji answer ("very unhappy"). */
export function faceWord(faces: number, key: number): string {
  return FACE_WORDS[faces]?.[key - 1] ?? ''
}

const ID = /^[a-z0-9]{1,24}$/

/** A fresh block id: six characters of [a-z0-9]. */
export function newPollId(): string {
  const alphabet = 'abcdefghijklmnopqrstuvwxyz0123456789'
  const b = new Uint8Array(6)
  crypto.getRandomValues(b)
  return Array.from(b, (x) => alphabet[x % alphabet.length]).join('')
}

/** Fills in defaults, as the server does. */
export function normalizePoll(c: Partial<PollConfig>): PollConfig {
  const kind: PollKind = c.kind === 'scale' || c.kind === 'emoji' ? c.kind : 'choice'
  const close: PollClose = c.close?.mode === 'date' || c.close?.mode === 'manual' ? { ...c.close } : { mode: 'days', days: c.close?.days && c.close.days > 0 ? c.close.days : 7 }
  const out: PollConfig = {
    id: c.id && ID.test(c.id) ? c.id : newPollId(),
    kind,
    question: c.question ?? '',
    answers: c.answers ?? [],
    layout: c.layout === 'row' ? 'row' : 'stacked',
    save: c.save === 'tag' || c.save === 'field' ? c.save : 'none',
    field: c.field,
    close,
    after: c.after === 'survey' ? 'survey' : 'thanks',
    thanks: c.thanks,
    survey: c.survey,
    nonContacts: c.nonContacts === 'refuse' ? 'refuse' : 'count',
    results: !!c.results,
    nps: c.nps,
    low: c.low,
    high: c.high,
    faces: c.faces,
  }
  if (kind === 'emoji' && out.faces !== 3) out.faces = 5
  if (kind === 'scale') {
    out.low = out.low || 'Not likely'
    out.high = out.high || 'Very likely'
  }
  return out
}

/** Reads an rc-poll attribute value; null when it isn't a poll. */
export function parsePoll(raw: string | undefined | null): PollConfig | null {
  if (!raw || !raw.trim()) return null
  try {
    return normalizePoll(JSON.parse(raw) as Partial<PollConfig>)
  } catch {
    return null
  }
}

/** The attribute value, with the server's field order. */
export function encodePoll(c: PollConfig): string {
  const n = normalizePoll(c)
  const o: Record<string, unknown> = { id: n.id, kind: n.kind, question: n.question, answers: n.answers.map((a) => ({ label: a.label, ...(a.emoji ? { emoji: a.emoji } : {}), ...(a.tag ? { tag: a.tag } : {}) })) }
  o.layout = n.layout
  o.save = n.save
  if (n.field) o.field = n.field
  o.close = { mode: n.close.mode, ...(n.close.days ? { days: n.close.days } : {}), ...(n.close.date ? { date: n.close.date } : {}) }
  o.after = n.after
  if (n.thanks) o.thanks = n.thanks
  if (n.survey) o.survey = n.survey
  o.nonContacts = n.nonContacts
  o.results = n.results
  if (n.nps) o.nps = n.nps
  if (n.low) o.low = n.low
  if (n.high) o.high = n.high
  if (n.faces) o.faces = n.faces
  return JSON.stringify(o)
}

/** The answers a poll takes: 1… for choices and faces, 0–10 for a scale. */
export function pollAnswers(c: PollConfig): PollAnswer[] {
  if (c.kind === 'scale') return Array.from({ length: 11 }, (_, i) => ({ key: i, label: String(i) }))
  if (c.kind === 'emoji') {
    const faces = c.faces === 3 ? 3 : 5
    return Array.from({ length: faces }, (_, i) => ({ key: i + 1, label: `${i + 1} of ${faces}, ${faceWord(faces, i + 1)}` }))
  }
  return c.answers.map((a, i) => ({ key: i + 1, label: a.label.trim(), emoji: a.emoji, tag: a.tag?.trim() }))
}

/** The Liquid variable holding a recipient's answer link base: an answer links to {{ rc_poll_<id> }}/<key>. */
export function pollVar(id: string): string {
  return `rc_poll_${id}`
}

/** The Liquid variable of a survey's link for a recipient. */
export function surveyVar(n: number): string {
  return `rc_survey_${n}`
}

/** html.EscapeString, as the server writes it. */
function esc(s: string): string {
  return s.replace(/&/g, '&amp;').replace(/'/g, '&#39;').replace(/</g, '&lt;').replace(/>/g, '&gt;').replace(/"/g, '&#34;')
}

function capitalize(s: string): string {
  return s ? s[0]!.toUpperCase() + s.slice(1) : s
}

/**
 * The Poll block's MJML: the question, the answers as plain links (MJML
 * buttons, or cells for a scale), the ends' words and the note.
 * `faceUrl` gives an emoji face's image.
 */
export function pollSection(cfg: PollConfig, faceUrl?: (faces: number, key: number) => string): string {
  const c = normalizePoll(cfg)
  let b = '<!-- rc:poll -->\n'
  b += `<mj-section rc-poll="${esc(encodePoll(c))}" rc-name="Poll" background-color="#ffffff" padding="12px 24px 20px">\n<mj-column>\n`
  b += `<mj-text font-family="${esc(SERIF)}" font-size="20px" line-height="1.3" font-weight="700" color="#0B1220" padding="0px 0px 12px">${esc(c.question)}</mj-text>\n`
  const link = (key: number) => `{{ ${pollVar(c.id)} }}/${key}`
  if (c.kind === 'scale') {
    let cells = ''
    for (let i = 0; i <= 10; i++) {
      cells +=
        `<a href="${link(i)}" style="display:inline-block;width:40px;height:40px;margin:0 6px 6px 0;border:2px solid ${POLL_ACCENT};border-radius:6px;` +
        `color:${POLL_ACCENT};text-align:center;font-family:${SANS};font-size:14px;font-weight:600;line-height:40px;text-decoration:none">${i}</a>`
    }
    b +=
      `<mj-text css-class="rc-poll-scale" padding="0px"><div style="font-size:0;line-height:0">${cells}</div>` +
      `<table role="presentation" width="100%" cellpadding="0" cellspacing="0" style="font-family:${SANS};font-size:12px;color:#5B6472"><tr>` +
      `<td style="padding:4px 0 0">0 · ${esc(c.low ?? '')}</td><td align="right" style="padding:4px 0 0">10 · ${esc(c.high ?? '')}</td></tr></table></mj-text>\n`
  } else if (c.kind === 'emoji') {
    let cells = ''
    for (const a of pollAnswers(c)) {
      const src = faceUrl ? faceUrl(c.faces ?? 5, a.key) : ''
      cells +=
        `<a href="${link(a.key)}" style="display:inline-block;width:96px;height:48px;margin:0 8px 8px 0;border:2px solid ${POLL_ACCENT};border-radius:6px;text-align:center;text-decoration:none">` +
        `<img src="${esc(src)}" alt="${esc(a.label)}" width="28" height="28" style="display:inline-block;margin-top:10px;border:0"></a>`
    }
    const faces = c.faces ?? 5
    b +=
      `<mj-text css-class="rc-poll-emoji" padding="0px"><div style="font-size:0;line-height:0">${cells}</div>` +
      `<table role="presentation" width="100%" cellpadding="0" cellspacing="0" style="font-family:${SANS};font-size:12px;color:#5B6472"><tr>` +
      `<td style="padding:2px 0 0">${esc(capitalize(faceWord(faces, 1)))}</td><td align="right" style="padding:2px 0 0">${esc(capitalize(faceWord(faces, faces)))}</td></tr></table></mj-text>\n`
  } else if (c.layout === 'row') {
    let cells = ''
    for (const a of pollAnswers(c)) {
      const label = a.emoji ? `${a.emoji} ${a.label}` : a.label
      cells +=
        `<a href="${link(a.key)}" style="display:inline-block;min-width:96px;margin:0 8px 8px 0;padding:11px 16px;border:2px solid ${POLL_ACCENT};border-radius:6px;` +
        `color:${POLL_ACCENT};text-align:center;font-family:${SANS};font-size:14px;font-weight:600;line-height:20px;text-decoration:none">${esc(label)}</a>`
    }
    b += `<mj-text css-class="rc-poll-row" padding="0px"><div style="font-size:0;line-height:0">${cells}</div></mj-text>\n`
  } else {
    for (const a of pollAnswers(c)) {
      const label = a.emoji ? `${a.emoji} ${a.label}` : a.label
      b +=
        `<mj-button href="${link(a.key)}" css-class="rc-poll-answer" width="100%" align="center" background-color="#ffffff" color="${POLL_ACCENT}" border="2px solid ${POLL_ACCENT}" ` +
        `border-radius="6px" font-family="${esc(SANS)}" font-size="14px" font-weight="600" inner-padding="11px 16px" padding="0px 0px 8px">${esc(label)}</mj-button>\n`
    }
  }
  let note = 'One click opens a page that saves your answer.'
  if (c.results) note += " You'll see the results there."
  b += `<mj-text font-family="${esc(SANS)}" font-size="12px" line-height="1.5" color="#5B6472" padding="4px 0px 0px">${esc(note)}</mj-text>\n`
  b += '</mj-column>\n</mj-section>'
  return b
}

/** A new Poll block's settings: a Choice poll with two answers. */
export function newPoll(): PollConfig {
  return normalizePoll({
    kind: 'choice',
    question: 'What should we write about next?',
    answers: [{ label: 'Option one' }, { label: 'Option two' }],
    save: 'none',
    close: { mode: 'days', days: 7 },
    after: 'thanks',
    nonContacts: 'count',
    results: false,
  })
}
