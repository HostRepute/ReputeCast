import type { ComposerCtxData } from '../context'

/** Variables ReputeCast fills in per recipient, beside the workspace's fields. */
const FILLED = ['unsubscribe_url', 'preferences_url', 'view_in_browser_url', 'upgrade_url', 'manage_url']

/**
 * What an email may print (the server's composer checks and previews say
 * the same, internal/campaigns): the workspace's tokens, the links filled
 * in per recipient, contact.*, poll and survey links, and a digest
 * template's feed. Anything else is an unknown field, flagged even with a
 * fallback (CO-4): it prints the fallback, or nothing, for everyone.
 */
export function knownFields(tokens: ComposerCtxData['tokens'], digestTemplate = false): Set<string> {
  const out = new Set<string>([...tokens.map((t) => t.key), ...FILLED])
  if (digestTemplate) {
    out.add('issue')
    out.add('feed')
  }
  return out
}

export function isKnownField(key: string, known: Set<string>): boolean {
  const root = key.split('.')[0] ?? key
  if (known.has(key) || known.has(root) || root === 'contact' || root === 'forloop') return true
  return root.startsWith('rc_poll_') || root.startsWith('rc_survey_')
}

/** Single-character edits between two words (with swaps), for "did you mean". */
function distance(a: string, b: string): number {
  const prev2 = Array<number>(b.length + 1).fill(0)
  let prev = Array.from({ length: b.length + 1 }, (_, j) => j)
  for (let i = 1; i <= a.length; i++) {
    const cur = [i]
    for (let j = 1; j <= b.length; j++) {
      cur[j] = Math.min(prev[j]! + 1, cur[j - 1]! + 1, prev[j - 1]! + (a[i - 1] === b[j - 1] ? 0 : 1))
      if (i > 1 && j > 1 && a[i - 1] === b[j - 2] && a[i - 2] === b[j - 1]) cur[j] = Math.min(cur[j]!, prev2[j - 2]! + 1)
    }
    prev2.splice(0, prev2.length, ...prev)
    prev = cur
  }
  return prev[b.length]!
}

/** The known field a mistyped key most likely meant ("frist_name" → "first_name"). */
export function suggestField(key: string, known: Set<string>): string | undefined {
  const limit = key.length <= 4 ? 1 : 2
  let best: string | undefined
  let bestD = Infinity
  for (const k of known) {
    const d = distance(key.toLowerCase(), k.toLowerCase())
    if (d <= limit && d < bestD) {
      best = k
      bestD = d
    }
  }
  return best
}
