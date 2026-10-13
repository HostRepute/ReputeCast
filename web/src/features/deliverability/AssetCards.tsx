import type { ReactNode } from 'react'
import type { Schemas } from '@/api/client'
import { Badge } from '@/components/ui/Badge'
import { Button, ButtonLink } from '@/components/ui/Button'
import { Skeleton } from '@/components/ui/Feedback'
import { Grid, Row } from '@/components/ui/Layout'
import { cn } from '@/lib/cn'
import { clock, count, trimPct } from '@/lib/format'
import { useWorkspace } from '@/lib/session'
import { authLine, cap, shortSource, toneOf, usedBy, type AssetHealth, type Overview } from './words'

/** A label and a value on one line (the board's `.kv` on Deliverability). */
export function KV({ label, children, labelClass, valueClass, className }: { label: ReactNode; children: ReactNode; labelClass?: string; valueClass?: string; className?: string }) {
  return (
    <div className={cn('flex justify-between gap-2.5 text-[13px]', className)}>
      <span className={labelClass}>{label}</span>
      <span className={cn('n', valueClass)}>{children}</span>
    </div>
  )
}

function CardHead({ name, mono, badge }: { name: string; mono?: boolean; badge: ReactNode }) {
  return (
    <Row justify="space-between" gap={8}>
      <span className={cn('text-[15px] font-bold', mono && 'mono')}>{name}</span>
      {badge}
    </Row>
  )
}

function Card({ children, className, label }: { children: ReactNode; className?: string; label: string }) {
  return (
    <article className={cn('card flex flex-col gap-2.5 px-4.5 py-4', className)} aria-label={label}>
      {children}
    </article>
  )
}

/** "1 of 84 · UCEPROTECT L3" */
function listingLine(b: Schemas['DeliverabilityAssetBlocklists']): string {
  return b.listed ? `${b.listed} of ${b.sources} · ${b.listings.map(shortSource).join(', ')}` : `0 of ${b.sources} listed`
}

function warmupLine(a: AssetHealth): string {
  const parts = ['Dedicated IP']
  if (a.provider) parts.push(a.provider)
  if (a.warmup) parts.push(`warming up, day ${a.warmup.day} of ${a.warmup.days}`)
  else if (a.providerStatus === 'warming_up') parts.push('warming up')
  return parts.join(' · ')
}

function rdnsLine(a: AssetHealth): string {
  if (!a.reverseDns) return 'not checked yet'
  if (!a.reverseDns.ptr) return 'missing'
  return `${a.reverseDns.ptr} · ${a.reverseDns.matches ? 'matches' : "doesn't resolve back"}`
}

function sndsLine(a: AssetHealth, suffix?: string): string {
  if (!a.snds) return 'no data yet'
  return `${cap(a.snds.filterResult)} · ${suffix ?? `${count(a.snds.trapHits)} trap ${a.snds.trapHits === 1 ? 'hit' : 'hits'}`}`
}

/** The badge of a domain or IP card. */
function assetBadge(a: AssetHealth, connected: boolean): ReactNode {
  const tone = toneOf(a.tone)
  if (connected && a.score != null) {
    const word = a.blocklists.listed ? `Listed on ${a.blocklists.listed}` : a.verdict || 'Good'
    return (
      <Badge tone={tone} dot>
        {a.score} · {word}
      </Badge>
    )
  }
  if (a.blocklists.listed) {
    return (
      <Badge tone={tone} dot>
        Listed on {a.blocklists.listed}
      </Badge>
    )
  }
  if (a.type === 'ip') {
    return (
      <Badge tone={a.blocklists.via === 'none' ? 'neutral' : 'ok'} dot>
        {a.blocklists.via === 'none' ? 'Not checked yet' : `Clean on ${a.blocklists.sources}`}
      </Badge>
    )
  }
  return (
    <Badge tone={tone} dot>
      {tone === 'ok' ? 'Good' : 'Needs attention'}
    </Badge>
  )
}

function DomainCard({ a, connected }: { a: AssetHealth; connected: boolean }) {
  const builtin = a.blocklists.via !== 'hostrepute'
  return (
    <Card label={a.asset} className={a.tone === 'warn' ? 'border-[#FCD34D]' : a.tone === 'bad' ? 'border-[#FCA5A5]' : undefined}>
      <CardHead name={a.asset} badge={assetBadge(a, connected)} />
      <span className="k">{['Sending domain', builtin ? 'built-in checks' : usedBy(a.usedBy)].filter(Boolean).join(' · ')}</span>
      {builtin ? (
        <KV label={`Blocklists · built-in, ${a.blocklists.sources} public DNSBLs`}>{a.blocklists.via === 'none' ? 'not checked yet' : listingLine(a.blocklists)}</KV>
      ) : (
        <KV label="Domain blocklists">{listingLine(a.blocklists)}</KV>
      )}
      <KV label="SPF · DKIM · DMARC">{a.auth ? authLine(a.auth) : 'not checked yet'}</KV>
      {connected && a.dmarcAligned != null ? <KV label="DMARC alignment, 7 d">{trimPct(a.dmarcAligned)}</KV> : null}
      {a.gmailReputation ? <KV label="Gmail domain reputation">{cap(a.gmailReputation)}</KV> : null}
      {!connected && a.dmarcAligned != null ? <KV label="DMARC alignment, 7 d">{trimPct(a.dmarcAligned)}</KV> : null}
      {!connected && a.dmarcAligned == null ? (
        <KV label="DMARC alignment" labelClass="k" valueClass="k font-medium [font-variant-numeric:normal]">
          needs reports: add rua to HostRepute or your own mailbox
        </KV>
      ) : null}
    </Card>
  )
}

function IPCard({ a, connected, onDelist, delistReason, delisting }: { a: AssetHealth; connected: boolean; onDelist: () => void; delistReason?: string; delisting: boolean }) {
  const { slug } = useWorkspace()
  const builtin = a.blocklists.via !== 'hostrepute'
  return (
    <Card label={a.asset} className={a.tone === 'warn' ? 'border-[#FCD34D]' : a.tone === 'bad' ? 'border-[#FCA5A5]' : undefined}>
      <CardHead name={a.asset} mono badge={assetBadge(a, connected)} />
      <span className="k">{warmupLine(a)}</span>
      {builtin ? (
        <KV label={`Blocklists · built-in, ${a.blocklists.sources} public DNSBLs`} valueClass={a.blocklists.listed ? 'text-warn-fg' : undefined}>
          {a.blocklists.via === 'none' ? 'not checked yet' : a.blocklists.listed ? listingLine(a.blocklists) : `0 of ${a.blocklists.sources} · daily`}
        </KV>
      ) : (
        <KV label="IP blocklists" valueClass={a.blocklists.listed ? 'text-warn-fg' : undefined}>
          {listingLine(a.blocklists)}
        </KV>
      )}
      <KV label="Reverse DNS" valueClass="mono">
        {rdnsLine(a)}
      </KV>
      <KV label="Microsoft SNDS">{sndsLine(a)}</KV>
      {connected && (a.blocklists.listed > 0 || a.warmup) ? (
        <div className="flex gap-2">
          {a.blocklists.listed > 0 ? (
            <Button size="sm" onClick={onDelist} disabledReason={delistReason} loading={delisting}>
              Request delisting
            </Button>
          ) : null}
          {a.warmup ? (
            <ButtonLink size="sm" to="/w/$workspace/providers/warmup" params={{ workspace: slug }}>
              Warm-up plan
            </ButtonLink>
          ) : null}
        </div>
      ) : null}
      {!connected ? <span className="k">HostRepute would also check 76 more sources hourly, including UCEPROTECT.</span> : null}
    </Card>
  )
}

function PoolCard({ p, connected, ipPool }: { p: Schemas['DeliverabilityPoolHealth']; connected: boolean; ipPool?: string }) {
  const caption = connected
    ? [`Shared pool${p.region ? `, ${p.region}` : ''}`, p.onlyStream ? `${cap(p.onlyStream)} stream only` : null, `checked at domain level${p.managedReputation ? ', Amazon manages IP reputation' : ''}`]
    : ['Shared pool', p.onlyStream ? `${cap(p.onlyStream)} stream only` : 'checked at domain level']
  return (
    <Card label={`${p.provider} pool`}>
      <CardHead name={`${p.provider} pool`} badge={<Badge tone="neutral">Shared IPs</Badge>} />
      <span className="k">{caption.filter(Boolean).join(' · ')}</span>
      <KV label={p.managedReputation ? 'Account reputation' : 'Provider status'}>{p.accountStatus}</KV>
      {p.gmailIpReputation && ipPool === p.short ? <KV label="Gmail IP reputation, pool">{cap(p.gmailIpReputation)}</KV> : null}
      {connected && p.dedicatedIpOption ? <KV label="Dedicated IP option">{p.dedicatedIpOption}</KV> : null}
      {connected && p.volumeThisMonth != null ? <KV label="Volume this month">{count(p.volumeThisMonth)}</KV> : null}
    </Card>
  )
}

/** Health per sending domain and IP, and the shared pools. */
export function AssetGrid({ o, onDelist, delistReason, delisting }: { o: Overview; onDelist: (asset: string) => void; delistReason?: string; delisting?: string }) {
  const connected = o.hostRepute.status === 'connected' && !o.hostRepute.unreachable
  return (
    <section className="flex flex-col gap-3" aria-labelledby="h-assets">
      <h2 className="h2" id="h-assets">
        Health per sending domain and IP
      </h2>
      <Grid min={300} gap={16}>
        {o.assets.map((a) =>
          a.type === 'domain' ? (
            <DomainCard key={a.asset} a={a} connected={connected} />
          ) : (
            <IPCard key={a.asset} a={a} connected={connected} onDelist={() => onDelist(a.asset)} delistReason={delistReason} delisting={delisting === a.asset} />
          ),
        )}
        {o.pools.map((p) => (
          <PoolCard key={p.providerId} p={p} connected={connected} ipPool={o.postmaster?.ipReputationPool ?? o.pools[0]?.short} />
        ))}
      </Grid>
    </section>
  )
}

/**
 * The cards while HostRepute can't be refreshed (DeliverabilityStates ›
 * error): HostRepute's results greyed with the time they're from, the
 * built-in fallback next to them, and the provider data that is current.
 */
export function StaleGrid({ o }: { o: Overview }) {
  const pool = o.pools[0]
  const domain = o.assets.find((a) => a.type === 'domain')
  return (
    <Grid min={300} gap={16}>
      {o.assets.map((a) => {
        const kind = a.type === 'domain' ? 'Domain' : 'IP'
        const stale = a.stale ? clock(a.stale) : null
        return (
          <Card key={a.asset} label={a.asset} className="border-[#FCA5A5]">
            <CardHead
              name={a.asset}
              mono={a.type === 'ip'}
              badge={
                <Badge tone="bad" dot>
                  Stale{stale ? ` · ${stale}` : ''}
                </Badge>
              }
            />
            <KV label={`${kind} blocklists · ${a.blocklists.sources} via HostRepute`} className="opacity-70">
              {listingLine(a.blocklists)}
            </KV>
            <KV label={`${kind} blocklists · built-in fallback`}>
              {a.fallback?.checkedAt ? `${a.fallback.listed} of ${a.fallback.sources} · ${clock(a.fallback.checkedAt)}` : 'running now'}
            </KV>
            {a.type === 'domain' ? (
              <KV label="SPF · DKIM · DMARC">{a.auth ? authLine(a.auth) : 'not checked yet'}</KV>
            ) : (
              <KV label="Microsoft SNDS">{sndsLine(a, o.snds && !o.snds.stale ? 'current' : 'stale')}</KV>
            )}
          </Card>
        )
      })}
      {pool ? (
        <Card label={pool.provider}>
          <CardHead
            name={pool.provider}
            badge={
              <Badge tone="ok" dot>
                Current
              </Badge>
            }
          />
          <KV label="Bounce · complaint">
            {trimPct(pool.bounceRate)} · {trimPct(pool.complaintRate)}
          </KV>
          {domain?.gmailReputation ? (
            <KV label="Gmail domain reputation">
              {cap(domain.gmailReputation)} · {o.postmaster && !o.postmaster.stale ? 'current' : 'stale'}
            </KV>
          ) : null}
        </Card>
      ) : null}
    </Grid>
  )
}

/** Skeleton cards shaped like the asset cards (DeliverabilityStates › loading). */
export function SkeletonGrid() {
  const shapes: [number, number, (number | string)[]][] = [
    [120, 200, ['100%', '90%', '70%']],
    [120, 220, ['100%', '85%', '60%']],
    [140, 180, ['90%', '70%']],
  ]
  return (
    <Grid min={300} gap={16} aria-busy="true">
      {shapes.map(([title, sub, rows], i) => (
        <div key={i} className="card flex flex-col gap-2.5 px-4.5 py-4">
          <div className="flex justify-between">
            <Skeleton w={title} h={18} />
            <Skeleton w={80} h={22} r={999} />
          </div>
          <Skeleton w={sub} h={12} />
          {rows.map((w, j) => (
            <Skeleton key={j} w={w} h={12} />
          ))}
        </div>
      ))}
    </Grid>
  )
}
