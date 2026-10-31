import type { Schemas } from '@/api/client'
import { Badge } from '@/components/ui/Badge'
import { Bar } from '@/components/ui/Data'
import { useT } from '@/i18n'
import { cn } from '@/lib/cn'
import { trimPct } from '@/lib/format'
import { KV } from './AssetCards'
import { m } from './messages'
import { barColor, toneOf } from './words'

/**
 * A rate against the provider's thresholds. While there's too little mail to
 * judge it (judged=false), it shows without the bar that would compare it.
 */
function RateRow({ r, judged = true }: { r: Schemas['DeliverabilityRateLimit']; judged?: boolean }) {
  const t = useT()
  const label = r.thresholds ? `${r.label} · ${r.thresholds}` : r.label
  return (
    <div>
      <KV label={label} labelClass="k">
        {trimPct(r.value)}
      </KV>
      {judged ? <Bar value={r.usage} color={barColor[r.tone] ?? barColor.ok} label={t(m.rateBarLabel, { label, value: trimPct(r.value) })} /> : null}
    </div>
  )
}

/** Account health per sending provider, against each provider's own thresholds. */
export function ProviderHealthCard({ providers, connected }: { providers: Schemas['DeliverabilityProviderHealth'][]; connected: boolean }) {
  const t = useT()
  if (!providers.length) return null
  return (
    <section className="card" aria-labelledby="h-provider-health">
      <div className="flex flex-wrap items-center justify-between gap-2 border-b border-line-2 px-4.5 py-3.5">
        <h2 className="h2" id="h-provider-health">
          {t(m.providerHealthTitle)}
        </h2>
        <span className="k">{connected ? t(m.providerHealthConnected) : t(m.providerHealthDirect)}</span>
      </div>
      <div className="grid gap-0 [grid-template-columns:repeat(auto-fit,minmax(min(320px,100%),1fr))]">
        {providers.map((p, i) => (
          <div key={p.providerId} className={cn('flex flex-col gap-2.5 px-4.5 py-4', i < providers.length - 1 && 'border-r border-line-2')}>
            <div className="flex items-center justify-between">
              <span className="font-semibold">{p.name}</span>
              <Badge tone={toneOf(p.tone)} dot>
                {p.badge}
              </Badge>
            </div>
            <RateRow r={p.bounce} judged={!p.notJudged} />
            <RateRow r={p.complaint} judged={!p.notJudged} />
            {/* The server's sentence: the same minimum volume as Send Readiness. */}
            {p.notJudged ? <span className="k">{p.notJudged}</span> : null}
            {connected && p.note ? (
              <KV label={p.note.label} labelClass="k" valueClass="font-normal [font-variant-numeric:normal]">
                {p.note.value}
              </KV>
            ) : null}
          </div>
        ))}
      </div>
    </section>
  )
}
