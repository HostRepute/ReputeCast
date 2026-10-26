import { Badge } from '@/components/ui/Badge'
import { clock, count, pct } from '@/lib/format'
import { useT } from '@/i18n'
import { useComposer, type ABTest, type Checks } from './context'
import { mp } from './preview.messages'

const PERCENTS = [10, 20, 30, 40, 50]
const HOURS = [1, 2, 4, 8, 24]

type Variant = ABTest['variants'][number]

/**
 * The decision rules of an A/B test (C14): R05 adds minSample, confidence
 * and noWinner to the test; until then the engine's defaults apply, and
 * the rule line says so.
 */
export function decisionRules(t: ABTest): { minSample: number; confidence: number; noWinner: string } {
  const r = t as ABTest & { minSample?: number; confidence?: number; noWinner?: string }
  return { minSample: r.minSample ?? 500, confidence: r.confidence ?? 0.95, noWinner: r.noWinner ?? 'default' }
}

/** A new test, or a new variant: a subject variant starts from A's subject. */
export function newTest(kind: ABTest['kind'], subject: string, mjml: () => string, base?: Partial<ABTest>): ABTest {
  const b = { testPercent: base?.testPercent ?? 20, metric: base?.metric ?? 'clicks', durationHours: base?.durationHours ?? 2 } as const
  if (kind === 'content') return { ...b, kind, variants: [{ key: 'A' }, { key: 'B', mjml: mjml() }] }
  if (kind === 'send_time') return { ...b, kind, variants: [{ key: 'A', sendTime: '09:00' }, { key: 'B', sendTime: '14:00' }] }
  return { ...b, kind, variants: [{ key: 'A' }, { key: 'B', subject }] }
}

/**
 * The subject and preheader variant `key` sends: a subject test's variant
 * its own subject (an empty one is a check finding, not A's), any other
 * variant its own when it has one, else A's.
 */
export function variantLines(t: ABTest | null, key: string, a: { subject: string; preheader: string }): { subject: string; preheader: string } {
  const v = key === 'A' ? undefined : t?.variants.find((x) => x.key === key)
  if (!v || !t) return a
  return { subject: t.kind === 'subject' ? (v.subject ?? '') : v.subject || a.subject, preheader: v.preheader || a.preheader }
}

export function withVariant(t: ABTest, subject: string, mjml: () => string): ABTest {
  if (t.variants.length >= 3) return t
  const key = t.variants.some((v) => v.key === 'B') ? 'C' : 'B'
  const v: Variant = t.kind === 'content' ? { key, mjml: mjml() } : t.kind === 'send_time' ? { key, sendTime: '17:00' } : { key, subject }
  return { ...t, variants: [...t.variants, v] }
}

/** What a variant's own check found (B, C), from the composer's checks. */
function VariantCheck({ k, checks }: { k: string; checks: Checks | undefined }) {
  const tr = useT()
  const v = checks?.variants?.find((x) => x.key === k)
  if (!v) return null
  if (!v.issues.length) return <span className="k">{tr(mp.variantClean, { key: k })}</span>
  return (
    <div className="flex flex-col gap-1">
      <span className="flex flex-wrap items-center gap-1.5">
        {v.blocks ? (
          <Badge tone="bad" dot>
            {tr(mp.variantBlocking, { count: v.blocks })}
          </Badge>
        ) : null}
        {v.warnings ? (
          <Badge tone="warn" dot>
            {tr(mp.variantWarnings, { count: v.warnings })}
          </Badge>
        ) : null}
      </span>
      <ul className="k m-0 flex flex-col gap-0.5 pl-4">
        {v.issues.map((is, i) => (
          <li key={`${is.id}-${i}`}>{is.title}</li>
        ))}
      </ul>
    </div>
  )
}

/**
 * The A/B test in the preview (ComposerPreview, CO-6): each variant with
 * its subject (a new one starts from A's) and what its check found, the
 * sample, the test length, the winner metric with why clicks are
 * recommended, the decision rule and who gets what when.
 */
export function ABPanel() {
  const c = useComposer()
  const tr = useT()
  const t = c.abTest
  const audience = c.campaign.audienceCount ?? 0
  const start = (kind: ABTest['kind']) => {
    if (!c.canEdit) return
    c.setABTest(newTest(kind, c.subject, c.currentMjml, t ?? undefined))
  }
  const update = (next: Partial<ABTest>) => t && c.setABTest({ ...t, ...next })
  const setVariant = (key: string, v: Partial<Variant>) => t && c.setABTest({ ...t, variants: t.variants.map((x) => (x.key === key ? { ...x, ...v } : x)) })
  const addVariant = () => (t ? c.setABTest(withVariant(t, c.subject, c.currentMjml)) : start('subject'))
  const share = pct(t ? Math.round(100 / t.variants.length) : 50, 0)
  const sample = Math.round((audience * (t?.testPercent ?? 20)) / 100)
  const perVariant = t ? Math.round(sample / t.variants.length) : 0
  const rest = Math.max(0, audience - sample)
  const kinds: { kind: ABTest['kind']; title: string; text: string }[] = [
    { kind: 'subject', title: tr(mp.kindSubject), text: tr(mp.kindSubjectText) },
    { kind: 'content', title: tr(mp.kindContent), text: tr(mp.kindContentText) },
    { kind: 'send_time', title: tr(mp.kindSendTime), text: tr(mp.kindSendTimeText) },
  ]
  const winnerAt = c.campaign.scheduledAt && t ? clock(new Date(new Date(c.campaign.scheduledAt).getTime() + t.durationHours * 3600_000)) : null
  const hours = t ? Array.from(new Set([...HOURS, t.durationHours])).sort((a, b) => a - b) : HOURS
  const rules = t ? decisionRules(t) : null
  return (
    <div className="flex flex-col gap-2.5" style={{ padding: '14px 16px' }}>
      <div className="prow">
        <h2 className="m-0 text-[14px] font-semibold">{tr(mp.abTitle)}</h2>
        {c.canEdit ? (
          <button type="button" className="btn sm" onClick={addVariant} disabled={!!t && t.variants.length >= 3}>
            {tr(mp.addVariant)}
          </button>
        ) : null}
      </div>
      {kinds.map((k) =>
        t && t.kind === k.kind ? (
          <div key={k.kind} className="card flex flex-col gap-2" style={{ padding: '10px 12px' }}>
            <div className="prow">
              <span className="font-semibold">{k.title}</span>
              <Badge tone="info">{tr(mp.active)}</Badge>
            </div>
            {t.variants.map((v) =>
              t.kind === 'subject' ? (
                <div key={v.key} className="flex flex-col gap-1">
                  <label className="lbl">
                    {tr(mp.ofSample, { key: v.key, share })}
                    <input
                      className="in"
                      value={v.key === 'A' ? c.subject : (v.subject ?? '')}
                      readOnly={!c.canEdit}
                      onChange={(e) => (v.key === 'A' ? c.setSubject(e.target.value) : setVariant(v.key, { subject: e.target.value }))}
                      placeholder={tr(mp.anotherSubject)}
                    />
                  </label>
                  {v.key !== 'A' ? <VariantCheck k={v.key} checks={c.checks} /> : null}
                </div>
              ) : t.kind === 'send_time' ? (
                <label key={v.key} className="lbl">
                  {tr(mp.ofSampleLocal, { key: v.key, share })}
                  <input className="in" type="time" value={v.sendTime ?? ''} readOnly={!c.canEdit} onChange={(e) => setVariant(v.key, { sendTime: e.target.value })} />
                </label>
              ) : (
                <div key={v.key} className="flex flex-col gap-1">
                  <div className="flex items-center justify-between gap-2">
                    <span className="lbl" style={{ flexDirection: 'row' }}>
                      {tr(mp.ofSample, { key: v.key, share })}
                    </span>
                    <button
                      type="button"
                      className="btn sm"
                      onClick={() => {
                        c.setVariant(v.key)
                        c.setView('visual')
                      }}
                    >
                      {tr(mp.editOnCanvas)}
                    </button>
                  </div>
                  {v.key !== 'A' ? <VariantCheck k={v.key} checks={c.checks} /> : null}
                </div>
              ),
            )}
            {t.kind === 'send_time' ? (
              <label className="chk items-center">
                <input type="checkbox" checked={!!t.bestTime} disabled={!c.canEdit} onChange={(e) => update({ bestTime: e.target.checked })} />
                {tr(mp.bestTime)}
              </label>
            ) : null}
            <label className="lbl">
              {tr(mp.sampleSize)}
              <select className="in" value={t.testPercent} disabled={!c.canEdit} onChange={(e) => update({ testPercent: Number(e.target.value) })}>
                {PERCENTS.map((p) => (
                  <option key={p} value={p}>
                    {audience ? tr(mp.sampleOption, { share: pct(p, 0), count: count(Math.round((audience * p) / 100)) }) : pct(p, 0)}
                  </option>
                ))}
              </select>
            </label>
            <label className="lbl">
              {tr(mp.testLength)}
              <select className="in" value={t.durationHours} disabled={!c.canEdit} onChange={(e) => update({ durationHours: Number(e.target.value) })}>
                {hours.map((h) => (
                  <option key={h} value={h}>
                    {tr(mp.hoursOption, { hours: h })}
                  </option>
                ))}
              </select>
            </label>
            <label className="lbl">
              {tr(mp.winnerBy)}
              <select className="in" value={t.metric} disabled={!c.canEdit} onChange={(e) => update({ metric: e.target.value as ABTest['metric'] })}>
                <option value="clicks">{tr(mp.winnerClicksRec)}</option>
                <option value="human_opens">{tr(mp.winnerOpensPlain)}</option>
                <option value="manual">{tr(mp.winnerByHand)}</option>
              </select>
              <span className="k">{tr(mp.opensOverstate)}</span>
            </label>
            {t.metric !== 'manual' && rules ? (
              <div className="ab-rule" role="note">
                <svg width="15" height="15" viewBox="0 0 16 16" fill="none" stroke="currentColor" strokeWidth="1.6" aria-hidden="true">
                  <path d="M8 1.5 13.5 3.5v4.5c0 3-2.3 5.3-5.5 6.5C4.8 13.3 2.5 11 2.5 8V3.5z" strokeLinejoin="round" />
                  <path d="m5.5 8 1.8 1.8L10.8 6" />
                </svg>
                <span>{tr(mp.ruleLine, { confidence: pct(rules.confidence * 100, 0), minSample: count(rules.minSample) })}</span>
              </div>
            ) : null}
            <span className="k">
              {t.metric === 'manual'
                ? tr(mp.perVariantManual, { perVariant: count(perVariant), rest: count(rest) })
                : winnerAt
                  ? tr(mp.perVariantAt, { perVariant: count(perVariant), time: winnerAt, rest: count(rest) })
                  : tr(mp.perVariantAfter, { perVariant: count(perVariant), hours: t.durationHours, rest: count(rest) })}
            </span>
            {c.canEdit ? (
              <button type="button" className="k self-start" style={{ border: 0, background: 'none', padding: 0, minHeight: 24, cursor: 'pointer', textDecoration: 'underline', textUnderlineOffset: 2 }} onClick={() => c.setABTest(null)}>
                {tr(mp.stopTesting)}
              </button>
            ) : null}
          </div>
        ) : (
          <div key={k.kind} className="card flex flex-col gap-1.5" style={{ padding: '10px 12px', borderStyle: 'dashed' }}>
            <div className="prow">
              <span className="font-semibold">{k.title}</span>
              {c.canEdit ? (
                <button type="button" className="btn sm" onClick={() => start(k.kind)}>
                  {tr(mp.setUp)}
                </button>
              ) : null}
            </div>
            <span className="k">{k.text}</span>
          </div>
        ),
      )}
    </div>
  )
}

/** "Previewing: Subject A | Subject B": which variant the preview shows. */
export function PreviewingSwitch() {
  const c = useComposer()
  const tr = useT()
  const t = c.abTest
  if (!t || t.kind === 'send_time') return null
  return (
    <div className="vsw">
      <span className="lbt" id="vsw-l">
        {tr(mp.previewing)}
      </span>
      <div className="seg" role="group" aria-labelledby="vsw-l">
        {t.variants.map((v) => (
          <button key={v.key} type="button" aria-pressed={c.variant === v.key} onClick={() => c.setVariant(v.key)}>
            {t.kind === 'subject' ? tr(mp.subjectKey, { key: v.key }) : tr(mp.variantKey, { key: v.key })}
          </button>
        ))}
      </div>
    </div>
  )
}
