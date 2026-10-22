import { useQueryClient } from '@tanstack/react-query'
import type { ReactNode } from 'react'
import { $api, errorMessage, type Schemas } from '@/api/client'
import { Badge } from '@/components/ui/Badge'
import { Button, ButtonLink } from '@/components/ui/Button'
import { toast } from '@/components/ui/Toast'
import { Skeleton } from '@/components/ui/Feedback'
import { Grid, Row } from '@/components/ui/Layout'
import { t as tt, useT } from '@/i18n'
import { cn } from '@/lib/cn'
import { clock, count, trimPct } from '@/lib/format'
import { usePermissionReason, useWorkspace } from '@/lib/session'
import { m } from './messages'
import { authLine, cap, filterWord, reputationWord, shortSource, toneOf, usedBy, type AssetHealth, type Overview } from './words'

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
  return b.listed
    ? tt(m.listingLine, { listed: count(b.listed), total: count(b.sources), names: b.listings.map(shortSource).join(', ') })
    : tt(m.listingNone, { total: count(b.sources) })
}

function warmupLine(a: AssetHealth): string {
  const parts = [tt(m.dedicatedIp)]
  if (a.provider) parts.push(a.provider)
  if (a.warmup) parts.push(tt(m.warmingUpDay, { day: a.warmup.day, days: a.warmup.days }))
  else if (a.providerStatus === 'warming_up') parts.push(tt(m.warmingUp))
  return parts.join(' · ')
}

function rdnsLine(a: AssetHealth): string {
  if (!a.reverseDns) return tt(m.notCheckedYet)
  if (!a.reverseDns.ptr) return tt(m.rdnsMissing)
  return a.reverseDns.matches ? tt(m.rdnsMatches, { ptr: a.reverseDns.ptr }) : tt(m.rdnsNoMatch, { ptr: a.reverseDns.ptr })
}

/** The SNDS line of an IP: its filter result and trap hits, or whether it's current. */
function sndsLine(a: AssetHealth, state?: 'current' | 'stale'): string {
  if (!a.snds) return tt(m.noDataYetLower)
  const filter = filterWord(a.snds.filterResult)
  if (state === 'current') return tt(m.sndsCurrent, { filter })
  if (state === 'stale') return tt(m.sndsStale, { filter })
  return tt(m.sndsTraps, { filter, count: a.snds.trapHits })
}

/** The badge of a domain or IP card. */
function assetBadge(a: AssetHealth, connected: boolean): ReactNode {
  const tone = toneOf(a.tone)
  if (connected && a.score != null) {
    return (
      <Badge tone={tone} dot>
        {a.blocklists.listed
          ? tt(m.badgeScoreListed, { score: a.score, listed: a.blocklists.listed })
          : tt(m.badgeScore, { score: a.score, verdict: a.verdict ? tt(m.verdict, { verdict: a.verdict }) : tt(m.verdictGood) })}
      </Badge>
    )
  }
  if (a.blocklists.listed) {
    return (
      <Badge tone={tone} dot>
        {tt(m.badgeListed, { listed: a.blocklists.listed })}
      </Badge>
    )
  }
  if (a.type === 'ip') {
    return (
      <Badge tone={a.blocklists.via === 'none' ? 'neutral' : 'ok'} dot>
        {a.blocklists.via === 'none' ? tt(m.badgeNotChecked) : tt(m.badgeCleanOn, { sources: a.blocklists.sources })}
      </Badge>
    )
  }
  return (
    <Badge tone={tone} dot>
      {tone === 'ok' ? tt(m.verdictGood) : tt(m.needsAttention)}
    </Badge>
  )
}

/**
 * A domain HostRepute doesn't watch yet (notify.acme.io, brief §6): it
 * comes after the monitored ones, with Add to monitoring.
 */
function NotMonitoredCard({ a }: { a: AssetHealth }) {
  const t = useT()
  const { slug } = useWorkspace()
  const qc = useQueryClient()
  const add = $api.useMutation('post', '/api/v1/workspaces/{workspace}/integrations/hostrepute/assets')
  const reason = usePermissionReason('integrations:hostrepute')
  return (
    <Card label={a.asset}>
      <CardHead name={a.asset} badge={<Badge tone="neutral">{t(m.notMonitored)}</Badge>} />
      <span className="k">{[t(m.sendingDomain), usedBy(a.usedBy) || undefined, t(m.notMonitoredYet)].filter(Boolean).join(' · ')}</span>
      <KV label={'SPF · DKIM · DMARC'}>{a.auth ? authLine(a.auth) : t(m.notCheckedYet)}</KV>
      <KV label={t(m.blocklists)} valueClass="k font-medium [font-variant-numeric:normal]">
        {t(m.checkedOnceMonitored)}
      </KV>
      <Row gap={8}>
        <Button
          size="sm"
          loading={add.isPending}
          disabledReason={reason}
          onClick={() =>
            add.mutate(
              { params: { path: { workspace: slug } }, body: { asset: a.asset } },
              {
                onSuccess: () => {
                  toast({ kind: 'success', title: t(m.monitoringAdded, { asset: a.asset }), detail: t(m.monitoringAddedDetail) })
                  void qc.invalidateQueries()
                },
                onError: (e) => toast({ kind: 'error', title: errorMessage(e) }),
              },
            )
          }
        >
          {t(m.addToMonitoring)}
        </Button>
      </Row>
    </Card>
  )
}

function DomainCard({ a, connected }: { a: AssetHealth; connected: boolean }) {
  const t = useT()
  const builtin = a.blocklists.via !== 'hostrepute'
  if (connected && a.notMonitored) return <NotMonitoredCard a={a} />
  return (
    <Card label={a.asset} className={a.tone === 'warn' ? 'border-[#FCD34D]' : a.tone === 'bad' ? 'border-[#FCA5A5]' : undefined}>
      <CardHead name={a.asset} badge={assetBadge(a, connected)} />
      <span className="k">{[t(m.sendingDomain), builtin ? t(m.builtinChecks) : usedBy(a.usedBy)].filter(Boolean).join(' · ')}</span>
      {builtin ? (
        <KV label={t(m.blocklistsBuiltin, { sources: a.blocklists.sources })}>{a.blocklists.via === 'none' ? t(m.notCheckedYet) : listingLine(a.blocklists)}</KV>
      ) : (
        <KV label={t(m.domainBlocklists)}>{listingLine(a.blocklists)}</KV>
      )}
      <KV label={'SPF · DKIM · DMARC'}>{a.auth ? authLine(a.auth) : t(m.notCheckedYet)}</KV>
      {connected && a.dmarcAligned != null ? <KV label={t(m.dmarcAlignment7d)}>{trimPct(a.dmarcAligned)}</KV> : null}
      {a.gmailReputation ? <KV label={t(m.gmailDomainReputation)}>{reputationWord(a.gmailReputation)}</KV> : null}
      {!connected && a.dmarcAligned != null ? <KV label={t(m.dmarcAlignment7d)}>{trimPct(a.dmarcAligned)}</KV> : null}
      {!connected && a.dmarcAligned == null ? (
        <KV label={t(m.dmarcAlignment)} labelClass="k" valueClass="k font-medium [font-variant-numeric:normal]">
          {t(m.needsReports)}
        </KV>
      ) : null}
    </Card>
  )
}

function IPCard({ a, connected, onDelist, delistReason, delisting }: { a: AssetHealth; connected: boolean; onDelist: () => void; delistReason?: string; delisting: boolean }) {
  const t = useT()
  const { slug } = useWorkspace()
  const builtin = a.blocklists.via !== 'hostrepute'
  return (
    <Card label={a.asset} className={a.tone === 'warn' ? 'border-[#FCD34D]' : a.tone === 'bad' ? 'border-[#FCA5A5]' : undefined}>
      <CardHead name={a.asset} mono badge={assetBadge(a, connected)} />
      <span className="k">{warmupLine(a)}</span>
      {builtin ? (
        <KV label={t(m.blocklistsBuiltin, { sources: a.blocklists.sources })} valueClass={a.blocklists.listed ? 'text-warn-fg' : undefined}>
          {a.blocklists.via === 'none' ? t(m.notCheckedYet) : a.blocklists.listed ? listingLine(a.blocklists) : t(m.noneOfDaily, { total: count(a.blocklists.sources) })}
        </KV>
      ) : (
        <KV label={t(m.ipBlocklists)} valueClass={a.blocklists.listed ? 'text-warn-fg' : undefined}>
          {listingLine(a.blocklists)}
        </KV>
      )}
      <KV label={t(m.reverseDns)} valueClass="mono">
        {rdnsLine(a)}
      </KV>
      <KV label={'Microsoft SNDS'}>{sndsLine(a)}</KV>
      {connected && (a.blocklists.listed > 0 || a.warmup) ? (
        <div className="flex gap-2">
          {a.blocklists.listed > 0 ? (
            <Button size="sm" onClick={onDelist} disabledReason={delistReason} loading={delisting}>
              {t(m.requestDelisting)}
            </Button>
          ) : null}
          {a.warmup ? (
            <ButtonLink size="sm" to="/w/$workspace/providers/warmup" params={{ workspace: slug }}>
              {t(m.warmupPlan)}
            </ButtonLink>
          ) : null}
        </div>
      ) : null}
      {!connected ? <span className="k">{t(m.moreSources)}</span> : null}
    </Card>
  )
}

function PoolCard({ p, connected, ipPool }: { p: Schemas['DeliverabilityPoolHealth']; connected: boolean; ipPool?: string }) {
  const t = useT()
  const only = p.onlyStream ? t(m.streamOnly, { stream: p.onlyStream, name: cap(p.onlyStream) }) : null
  const caption = connected
    ? [p.region ? t(m.sharedPoolRegion, { region: p.region }) : t(m.sharedPool), only, p.managedReputation ? t(m.checkedDomainLevelAmazon) : t(m.checkedDomainLevel)]
    : [t(m.sharedPool), only ?? t(m.checkedDomainLevel)]
  const name = t(m.poolName, { provider: p.provider })
  return (
    <Card label={name}>
      <CardHead name={name} badge={<Badge tone="neutral">{t(m.sharedIps)}</Badge>} />
      <span className="k">{caption.filter(Boolean).join(' · ')}</span>
      <KV label={p.managedReputation ? t(m.accountReputation) : t(m.providerStatus)}>{p.accountStatus}</KV>
      {p.gmailIpReputation && ipPool === p.short ? <KV label={t(m.gmailIpReputationPool)}>{reputationWord(p.gmailIpReputation)}</KV> : null}
      {connected && p.dedicatedIpOption ? <KV label={t(m.dedicatedIpOption)}>{p.dedicatedIpOption}</KV> : null}
      {connected && p.volumeThisMonth != null ? <KV label={t(m.volumeThisMonth)}>{count(p.volumeThisMonth)}</KV> : null}
    </Card>
  )
}

/** Health per sending domain and IP, and the shared pools. */
export function AssetGrid({ o, onDelist, delistReason, delisting }: { o: Overview; onDelist: (asset: string) => void; delistReason?: string; delisting?: string }) {
  const t = useT()
  const connected = o.hostRepute.status === 'connected' && !o.hostRepute.unreachable
  return (
    <section className="flex flex-col gap-3" aria-labelledby="h-assets">
      <h2 className="h2" id="h-assets">
        {t(m.assetsTitle)}
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
  const t = useT()
  const pool = o.pools[0]
  const domain = o.assets.find((a) => a.type === 'domain')
  return (
    <Grid min={300} gap={16}>
      {o.assets.map((a) => {
        const kind = a.type === 'domain' ? 'domain' : 'ip'
        const stale = a.stale ? clock(a.stale) : null
        return (
          <Card key={a.asset} label={a.asset} className="border-[#FCA5A5]">
            <CardHead
              name={a.asset}
              mono={a.type === 'ip'}
              badge={
                <Badge tone="bad" dot>
                  {stale ? t(m.staleAt, { time: stale }) : t(m.stale)}
                </Badge>
              }
            />
            <KV label={t(m.staleBlocklistsVia, { kind, sources: a.blocklists.sources })} className="opacity-70">
              {listingLine(a.blocklists)}
            </KV>
            <KV label={t(m.staleBlocklistsFallback, { kind })}>
              {a.fallback?.checkedAt
                ? t(m.fallbackLine, { listed: count(a.fallback.listed), total: count(a.fallback.sources), time: clock(a.fallback.checkedAt) })
                : t(m.runningNow)}
            </KV>
            {a.type === 'domain' ? (
              <KV label={'SPF · DKIM · DMARC'}>{a.auth ? authLine(a.auth) : t(m.notCheckedYet)}</KV>
            ) : (
              <KV label={'Microsoft SNDS'}>{sndsLine(a, o.snds && !o.snds.stale ? 'current' : 'stale')}</KV>
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
                {t(m.current)}
              </Badge>
            }
          />
          <KV label={t(m.bounceComplaint)}>
            {trimPct(pool.bounceRate)} · {trimPct(pool.complaintRate)}
          </KV>
          {domain?.gmailReputation ? (
            <KV label={t(m.gmailDomainReputation)}>
              {o.postmaster && !o.postmaster.stale
                ? t(m.reputationCurrent, { reputation: reputationWord(domain.gmailReputation) })
                : t(m.reputationStale, { reputation: reputationWord(domain.gmailReputation) })}
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
