import type { Schemas } from '@/api/client'
import { Badge } from '@/components/ui/Badge'
import { toast } from '@/components/ui/Toast'
import { count } from '@/lib/format'
import { T, useT } from '@/i18n'
import { useComposer } from './context'
import { replaceField } from './fixes'
import { mk } from './code.messages'
import { mp } from './preview.messages'

type Preview = Schemas['ComposerPreview']
type Token = Preview['tokens'][number]

/** Summary badge: unknown fields first (they print nothing for everyone), then fields this contact lacks with no fallback. */
export function personalizationBadge(tokens: Token[]): { tone: 'warn' | 'ok'; unknown: number; missing: number } {
  const unknown = tokens.filter((t) => t.unknown).length
  const missing = tokens.filter((t) => !t.unknown && !t.perRecipient && t.missing && !t.fallbackUsed).length
  return { tone: unknown || missing ? 'warn' : 'ok', unknown, missing }
}

/**
 * The preview's Personalization card (ComposerPreview, CO-5): what each
 * field prints for the contact previewed, fields that aren't contact fields
 * ("frist_name isn't a contact field (did you mean first_name?)") with a
 * fix, and how many of the audience have no value for each field.
 */
export function PersonalizationCard({ preview }: { preview: Preview | null }) {
  const c = useComposer()
  const tr = useT()
  const tokens = preview?.tokens ?? []
  const badge = personalizationBadge(tokens)
  const replace = (from: string, to: string) => {
    const n = replaceField(c, from, to)
    toast({ kind: n ? 'success' : 'info', title: tr(mp.replaced, { from, to }), detail: tr(mp.replacedDetail, { count: n }) })
  }
  // Each field the audience lacks, once (a field used twice is counted once).
  const counted = preview?.counted
  const seen = new Set<string>()
  const lacking = tokens.filter((t) => {
    if (t.audienceMissing === undefined || t.audienceMissing === null || !t.audienceMissing || seen.has(t.key)) return false
    seen.add(t.key)
    return true
  })
  return (
    <div className="flex flex-col gap-2.5 border-b border-line-2" style={{ padding: '14px 16px' }}>
      <div className="prow">
        <h2 className="m-0 text-[14px] font-semibold">{tr(mp.personalization)}</h2>
        {preview && tokens.length ? (
          <Badge tone={badge.tone} dot={badge.tone === 'warn'}>
            {badge.unknown ? tr(mp.unknownFields, { count: badge.unknown }) : badge.missing ? tr(mp.missingFields, { count: badge.missing }) : tr(mp.noMissing)}
          </Badge>
        ) : null}
      </div>
      <div className="flex flex-col">
        {tokens.map((t) => (
          <div key={t.token} className="tokr">
            <code>{t.token}</code>
            <span className="cmp-token-value">
              → <b>{t.perRecipient ? tr(mp.perRecipient) : t.value || tr(mp.emptyValue)}</b>
            </span>
            {t.unknown ? (
              <div className="w" role="note">
                <span>
                  <T msg={mp.unknownField} values={{ key: t.key, b: (chunks) => <b style={{ fontWeight: 600 }}>{chunks}</b> }} />
                  {t.suggestion ? ` ${tr(mp.didYouMean, { suggestion: t.suggestion })}` : ''} {t.fallbackUsed ? tr(mp.unknownFallback, { fallback: t.value }) : tr(mp.unknownNothing)}
                </span>
                {t.suggestion && c.canEdit ? (
                  <button type="button" className="btn sm" onClick={() => replace(t.key, t.suggestion!)}>
                    {tr(mp.replaceWith, { suggestion: t.suggestion })}
                  </button>
                ) : null}
              </div>
            ) : null}
          </div>
        ))}
        {preview && !tokens.length ? <span className="k">{tr(mp.noTokens)}</span> : null}
      </div>
      {counted !== undefined && counted !== null
        ? lacking.map((t) => (
            <span key={t.key} className="k">
              {t.token.includes('default:')
                ? tr(mp.missingOf, { missing: t.audienceMissing!, counted: count(counted), key: t.key, fallback: fallbackOf(t.token) })
                : tr(mp.missingOfNothing, { missing: t.audienceMissing!, counted: count(counted), key: t.key })}
            </span>
          ))
        : null}
      {counted !== undefined && counted !== null && preview?.countedListsOnly && lacking.length ? <span className="k">{tr(mp.countedListsOnly)}</span> : null}
      {preview?.problems.length ? (
        <span className="k" style={{ color: 'var(--bad-fg)' }}>
          {tr(mk.lineError, { line: preview.problems[0]!.line, message: preview.problems[0]!.message })}
        </span>
      ) : null}
    </div>
  )
}

/** The fallback a token prints ({{ first_name | default: "there" }} → there). */
export function fallbackOf(token: string): string {
  return /default:\s*"([^"]*)"/.exec(token)?.[1] ?? /default:\s*'([^']*)'/.exec(token)?.[1] ?? ''
}
