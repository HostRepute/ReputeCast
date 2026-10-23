import { useQueryClient } from '@tanstack/react-query'
import { useState } from 'react'
import { $api, http, type Schemas } from '@/api/client'
import { Icon } from '@/components/icons/Icon'
import { Badge } from '@/components/ui/Badge'
import { ButtonLink, Button } from '@/components/ui/Button'
import { ConfirmDialog } from '@/components/ui/ConfirmDialog'
import { Dialog } from '@/components/ui/Dialog'
import { toast } from '@/components/ui/Toast'
import { t, useT } from '@/i18n'
import { clock, count, dayMonthDate, dayMonthYear, monthName } from '@/lib/format'
import { useCan, useWorkspace } from '@/lib/session'
import { bare, useWebsite } from './api'
import { me } from './elsewhere.messages'
import { mp } from './posts.messages'
import { dateTime } from './postFormat'
import { SiteFrame } from './SiteFrame'
import './website.css'

type WebState = Schemas['WebsiteCampaignWebState']

/**
 * The Campaigns list's Web column (WebsiteStates · elsewhere): the posts of
 * the listed campaigns, while the website is on (null otherwise).
 */
export function useCampaignWebStates(ids: string[]): Map<string, WebState> | null {
  const { slug } = useWorkspace()
  const can = useCan('website:view')
  const q = $api.useQuery('get', '/api/v1/workspaces/{workspace}/campaigns-web', { params: { path: { workspace: slug }, query: { ids: ids.join(',') } } }, { enabled: can && ids.length > 0, staleTime: 30_000 })
  if (!q.data?.siteOn) return null
  return new Map(q.data.states.map((s) => [s.campaignId, s]))
}

/** "Published · featured on home", "Scheduled · with the send" or "– · not on the web". */
export function WebCell({ state, status }: { state?: WebState; status: string }) {
  const t = useT()
  if (!state || state.chip === 'off') {
    return (
      <span className="ws-stk">
        <span className="k">–</span>
        {status === 'draft' || status === 'approval' || status === 'approved' ? <span className="k">{t(me.webNotOn)}</span> : null}
      </span>
    )
  }
  if (state.chip === 'unpublished') {
    return (
      <span className="ws-stk">
        <span className="k">–</span>
        <span className="k">{t(me.webUnpublishedLine, { date: state.unpublishedAt ? dayMonthDate(state.unpublishedAt) : '' })}</span>
      </span>
    )
  }
  if (state.chip === 'published') {
    return (
      <span className="ws-stk">
        <Badge tone="ok">{t(me.webPublished)}</Badge>
        <span className="k">{state.featured ? t(me.webFeatured) : state.slug ? `/posts/${state.slug}` : ''}</span>
      </span>
    )
  }
  return (
    <span className="ws-stk">
      <Badge tone="info">{t(me.webScheduled)}</Badge>
      <span className="k">{state.chip === 'with_send' || !state.publishAt ? t(me.webWithSendLine) : t(me.webAtLine, { date: dateTime(state.publishAt) })}</span>
    </span>
  )
}

/** Review & send's Web line: "Also publishes news.acme.io/posts/…, public, with the send." */
export function ReviewWebLine({ campaignId }: { campaignId: string }) {
  const t = useT()
  const { slug } = useWorkspace()
  const can = useCan('website:view')
  const q = $api.useQuery('get', '/api/v1/workspaces/{workspace}/campaigns/{campaignId}/web', { params: { path: { workspace: slug, campaignId } } }, { enabled: can })
  const d = q.data
  if (!d || !d.siteOn || !d.post.webOn) return null
  const p = d.post
  const vis = p.visibility === 'paid' ? t(me.visPaid) : p.visibility === 'members' ? t(me.visMembers) : t(me.visPublic)
  const when = p.publishMode === 'with_send' ? t(me.webWithSend) : p.publishMode === 'at' && p.publishAt ? t(me.webAt, { date: dateTime(p.publishAt) }) : t(me.webNow)
  return (
    <section className="card flex items-start gap-3 px-[18px] py-3.5" style={{ borderColor: '#BFDBFE', background: '#F8FBFF' }} aria-labelledby="rv-web-h">
      <span className="inline-flex h-[28px] w-[28px] flex-none items-center justify-center rounded-full bg-[#DBEAFE] text-[#1E40AF]" aria-hidden="true">
        <Icon name="globe" size={15} />
      </span>
      <div className="flex min-w-0 flex-1 flex-col gap-0.5">
        <b id="rv-web-h">{t(me.webLine)}</b>
        <span className="text-[13px]">{t(me.webLineText, { url: bare(p.url), visibility: vis, when })}</span>
      </div>
      <ButtonLink size="sm" to="/w/$workspace/campaigns/$campaignId/edit" params={{ workspace: slug, campaignId }} search={{ panel: 'web' } as never}>
        {t(me.edit)}
      </ButtonLink>
    </section>
  )
}

/** The report's Web section: views, signups and upgrades from the post, with its referrers. */
export function ReportWebSection({ campaignId }: { campaignId: string }) {
  const t = useT()
  const { slug } = useWorkspace()
  const can = useCan('website:view')
  const q = $api.useQuery('get', '/api/v1/workspaces/{workspace}/campaigns/{campaignId}/web-report', { params: { path: { workspace: slug, campaignId } } }, { enabled: can })
  const r = q.data?.report
  if (!r) return null
  const max = Math.max(1, ...r.referrers.map((x) => x.views))
  const status =
    r.status === 'published' && r.publishedAt
      ? t(me.statusPublished, { date: dayMonthDate(r.publishedAt) })
      : r.status === 'scheduled'
        ? t(me.statusScheduled)
        : r.status === 'unpublished'
          ? t(me.statusUnpublished)
          : t(me.statusDraft)
  return (
    <section className="card" aria-labelledby="rep-web-h">
      <div className="sec">
        <div className="flex min-w-0 flex-col gap-0.5">
          <h2 className="h2 m-0" id="rep-web-h">
            {t(me.reportTitle)}
          </h2>
          <span className="k">{t(me.reportSub, { url: bare(r.url), status })}</span>
        </div>
        {r.status === 'published' ? (
          <a className="btn sm" href={r.url} target="_blank" rel="noreferrer">
            <Icon name="external" size={13} />
            {t(me.openPost)}
          </a>
        ) : null}
      </div>
      <div className="pad flex flex-col gap-4">
        <div className="ws-stats" style={{ gridTemplateColumns: 'repeat(auto-fit,minmax(170px,1fr))' }}>
          <div className="card ws-stat">
            <span className="k">{t(me.reportViews)}</span>
            <span className="v">{count(r.views)}</span>
            <span className="s">{t(me.reportViewsSub, { total: count(r.viewsAllTime) })}</span>
          </div>
          <div className="card ws-stat">
            <span className="k">{t(me.reportSignups)}</span>
            <span className="v">{count(r.signups)}</span>
            <span className="s">{t(me.reportSignupsSub)}</span>
          </div>
          <div className="card ws-stat">
            <span className="k">{t(me.reportUpgrades)}</span>
            <span className="v">{count(r.upgrades)}</span>
            <span className="s">{t(me.reportUpgradesSub)}</span>
          </div>
        </div>
        <div className="flex flex-col gap-2.5">
          <b>{t(me.topReferrers)}</b>
          {r.referrers.length ? (
            <ul className="m-0 flex list-none flex-col gap-2.5 p-0" aria-label={t(me.referrersLabel)}>
              {r.referrers.map((x) => (
                <li key={x.key} className="flex flex-col gap-1">
                  <span className="flex justify-between gap-3 text-[13px]">
                    <span>{x.label}</span>
                    <b className="num">{count(x.views)}</b>
                  </span>
                  <span className="block h-[6px] overflow-hidden rounded-full bg-[#EEF0F3]" aria-hidden="true">
                    <span className="block h-full rounded-full bg-accent" style={{ width: `${Math.max(2, Math.round((x.views / max) * 100))}%` }} />
                  </span>
                </li>
              ))}
            </ul>
          ) : (
            <span className="k">{t(me.noViews)}</span>
          )}
          <span className="k">{r.utm.length ? t(me.utm, { sources: r.utm.map((u) => `${u.label.startsWith('utm_source=') ? u.label : `utm_source=${u.label}`} ${count(u.views)}`).join(' · ') }) : t(me.utmNone)}</span>
        </div>
      </div>
    </section>
  )
}

/** The contact profile's Website card: membership, sign-ins, sessions, the meter, and Sign out everywhere. */
export function ContactWebsiteCard({ contactId, name }: { contactId: string; name: string }) {
  const t = useT()
  const { slug } = useWorkspace()
  const qc = useQueryClient()
  const can = useCan('subscribers:view')
  const q = $api.useQuery('get', '/api/v1/workspaces/{workspace}/contacts/{contactId}/website', { params: { path: { workspace: slug, contactId } } }, { enabled: can })
  const [confirming, setConfirming] = useState(false)
  const d = q.data
  if (!d || !d.siteOn) return null
  const method = (m?: string) => (m === 'code' ? t(me.methodCode) : m === 'signup' ? t(me.methodSignup) : t(me.methodLink))
  const kind = d.paid ? t(me.memberPaid, { tiers: d.tiers.join(', ') }) : t(me.memberFree)
  return (
    <section className="card" aria-labelledby="ct-web-h">
      <div className="sec">
        <div className="flex min-w-0 flex-col gap-0.5">
          <h2 className="h2 m-0" id="ct-web-h">
            {t(me.cardTitle)}
          </h2>
          <span className="k">{t(me.cardSub, { host: d.host })}</span>
        </div>
      </div>
      <div className="pad flex flex-col gap-3">
        {d.member ? (
          <dl className="ws-kv">
            <dt>{t(me.member)}</dt>
            <dd>{t(me.memberSince, { date: d.since ? dayMonthYear(d.since) : '', kind })}</dd>
            <dt>{t(me.lastSignin)}</dt>
            <dd>{d.lastSignIn ? t(me.lastSigninText, { date: `${dayMonthDate(d.lastSignIn)}, ${clock(d.lastSignIn)}`, method: method(d.lastMethod) }) : '–'}</dd>
            <dt>{t(me.sessions)}</dt>
            <dd>{d.sessions ? t(me.sessionsText, { count: d.sessions, ends: d.sessionEnds ? dayMonthYear(d.sessionEnds) : '' }) : t(me.sessionsNone)}</dd>
            {d.meter && d.meter.limit > 0 ? (
              <>
                <dt>{t(me.meter)}</dt>
                <dd>{t(me.meterText, { used: d.meter.used, limit: d.meter.limit, month: monthName(d.meter.month, 'long') })}</dd>
              </>
            ) : null}
          </dl>
        ) : (
          <span className="k">{t(me.notMember, { host: d.host })}</span>
        )}
        {d.member ? (
          <div className="flex flex-wrap items-center gap-2.5">
            <Button size="sm" disabledReason={d.canSignOut ? (d.sessions ? undefined : t(me.sessionsNone)) : t(me.signOutHint)} onClick={() => setConfirming(true)}>
              {t(me.signOutEverywhere)}
            </Button>
            <span className="k">{t(me.signOutHint)}</span>
          </div>
        ) : null}
        {d.events.length ? (
          <ul className="m-0 flex list-none flex-col p-0" style={{ borderTop: '1px solid var(--line-2)' }}>
            {d.events.slice(0, 5).map((e) => (
              <li key={e.id} className="ws-chkl" style={{ alignItems: 'center' }}>
                <span className="inline-flex h-[26px] w-[26px] flex-none items-center justify-center rounded-full bg-ok-tint text-ok-fg" aria-hidden="true">
                  <Icon name={e.kind === 'signed_out_everywhere' ? 'signOut' : e.kind === 'link_refused' ? 'ban' : 'check'} size={13} />
                </span>
                <span className="tx">
                  <b>
                    {e.kind === 'signed_in'
                      ? t(me.eventSignedIn, { host: d.host })
                      : e.kind === 'link_refused'
                        ? t(me.eventRefused)
                        : e.kind === 'signed_out_everywhere'
                          ? t(me.eventSignedOut)
                          : e.kind === 'joined'
                            ? t(me.eventJoined, { host: d.host })
                            : t(me.eventEmailChanged)}
                  </b>
                  <span className="k">
                    {e.kind === 'signed_in'
                      ? t(me.eventSignedInDetail, { time: `${dayMonthDate(e.at)}, ${clock(e.at)}`, method: method(e.method), same: e.sameBrowser ? 'yes' : 'no' })
                      : e.kind === 'signed_out_everywhere' && e.actor
                        ? `${dayMonthDate(e.at)}, ${clock(e.at)} · ${t(me.eventSignedOutBy, { actor: e.actor })}`
                        : `${dayMonthDate(e.at)}, ${clock(e.at)}${e.detail ? ` · ${e.detail}` : ''}`}
                  </span>
                </span>
                <span className="mono k">{dayMonthDate(e.at)}</span>
              </li>
            ))}
          </ul>
        ) : null}
      </div>
      <ConfirmDialog
        open={confirming}
        onOpenChange={setConfirming}
        tone="warn"
        badge={t(me.signOutBadge)}
        title={t(me.signOutTitle, { name })}
        description={t(me.signOutText, { host: d.host, count: d.sessions })}
        confirmLabel={t(me.signOutBadge)}
        onConfirm={async () => {
          await http.POST('/api/v1/workspaces/{workspace}/contacts/{contactId}/website/sign-out', { params: { path: { workspace: slug, contactId } } })
          toast({ kind: 'success', title: t(me.signedOut, { name }) })
          await qc.invalidateQueries({ queryKey: ['get', '/api/v1/workspaces/{workspace}/contacts/{contactId}/website'] })
        }}
      />
    </section>
  )
}

/** The contacts filters' members: "Signed in to the website · 2,804" and "Paid members · 1,184". */
export function MembersChips({
  signedIn,
  onSignedIn,
  paid,
  onPaid,
}: {
  signedIn: boolean
  onSignedIn: (on: boolean) => void
  /** Earn's "Paid status: with access" filter (X16). */
  paid: boolean
  onPaid: (on: boolean) => void
}) {
  const t = useT()
  const ov = useWebsite()
  const d = ov.data
  if (!d || d.status !== 'on') return null
  return (
    <div className="flex flex-wrap items-center gap-2" role="group" aria-label={t(me.filterLabel)}>
      <span className="k mr-1">{t(me.filterLabel)}</span>
      <button type="button" className="vchip" aria-pressed={signedIn} onClick={() => onSignedIn(!signedIn)}>
        {t(me.filterSignedIn, { count: count(d.members.signedIn) })}
      </button>
      {d.members.paid ? (
        <button type="button" className="vchip" aria-pressed={paid} onClick={() => onPaid(!paid)}>
          {t(me.filterPaid, { count: count(d.members.paid) })}
        </button>
      ) : null}
    </div>
  )
}

/**
 * The segment condition "Signed in to the website · since · [7 days ago]":
 * a number of days ago ("7d") or a date ("2026-09-29").
 */
export function SignedInWhen({ value, onChange }: { value: string; onChange: (v: string) => void }) {
  const t = useT()
  const preset = ['7d', '30d', '90d'].includes(value)
  const [dated, setDated] = useState(!preset && /^\d{4}-\d{2}-\d{2}$/.test(value))
  return (
    <>
      <select
        aria-label={t(me.condWhen)}
        className="in"
        value={dated ? 'date' : preset ? value : '7d'}
        onChange={(e) => {
          if (e.target.value === 'date') {
            setDated(true)
            onChange(new Date(Date.now() - 30 * 86_400_000).toISOString().slice(0, 10))
          } else {
            setDated(false)
            onChange(e.target.value)
          }
        }}
      >
        <option value="7d">{t(me.days7)}</option>
        <option value="30d">{t(me.days30)}</option>
        <option value="90d">{t(me.days90)}</option>
        <option value="date">{dated && /^\d{4}-\d{2}-\d{2}$/.test(value) ? dayMonthYear(`${value}T12:00:00Z`) : t(me.onDate)}</option>
      </select>
      {dated ? <input className="in" type="date" aria-label={t(me.condWhen)} value={value} onChange={(e) => onChange(e.target.value)} /> : null}
    </>
  )
}

type PostSnapshot = Schemas['ApprovalWebsiteSnapshot']

/** "Post" next to a request's name in the approval inbox. */
export function PostBadge() {
  const t = useT()
  return <span className="badge sm pur">{t(me.inboxPost)}</span>
}

/** "Post · Members · /posts/q4-roadmap-deep-dive": what a post's request covers, in the inbox. */
export function postLine(w: PostSnapshot, rules: string): string {
  return [t(me.inboxPost), visibilityWord(w.visibility), w.path, rules].filter(Boolean).join(' · ')
}

/** A request's visibility in words, in the app's language (the snapshot's own label is English). */
function visibilityWord(v: string): string {
  return v === 'paid' ? t(mp.paid) : v === 'members' ? t(mp.members) : t(mp.public)
}

/** The route of a post: the website. */
export function postRoute(): string {
  return t(me.webLine)
}

/** When a post goes live once approved. */
export function postWhen(w: PostSnapshot): string {
  return w.when === 'at' && w.publishAt ? dateTime(w.publishAt) : t(me.whenApproved)
}

/**
 * A post's approval (ApprovalReview · posts): what the approval covers and
 * the page as a reader with access sees it.
 */
export function PostApprovalCard({ approvalId, name, snapshot: w }: { approvalId: string; name: string; snapshot: PostSnapshot }) {
  const tt = useT()
  const { slug } = useWorkspace()
  const [open, setOpen] = useState(false)
  const preview = $api.useQuery('get', '/api/v1/workspaces/{workspace}/approvals/{approvalId}/preview', { params: { path: { workspace: slug, approvalId } } })
  return (
    <section className="card" aria-labelledby="ap-post">
      <div className="sec">
        <div className="flex min-w-0 flex-col gap-0.5">
          <h2 className="h2 m-0" id="ap-post">
            {tt(me.reviewWeb)}
          </h2>
          <span className="k">{tt(me.reviewWebSub, { url: w.path })}</span>
        </div>
      </div>
      <div className="pad flex flex-col gap-3">
        <dl className="kv">
          <dt>{tt(me.reviewTitle)}</dt>
          <dd className="font-semibold">{w.title || name}</dd>
          <dt>{tt(me.reviewAddress)}</dt>
          <dd className="mono">{w.path}</dd>
          <dt>{tt(me.reviewVisibility)}</dt>
          <dd>{visibilityWord(w.visibility)}</dd>
          <dt>{tt(me.reviewWhen)}</dt>
          <dd>{postWhen(w)}</dd>
          {w.excerpt ? (
            <>
              <dt>{tt(me.reviewExcerpt)}</dt>
              <dd>{w.excerpt}</dd>
            </>
          ) : null}
        </dl>
        <div className="overflow-hidden rounded-[8px] border border-line">
          {preview.data ? <SiteFrame html={preview.data.html} title={tt(me.reviewFrame, { title: w.title || name })} width={1100} fit minHeight={700} maxHeight={1600} /> : <div className="h-[260px] bg-ground" aria-hidden="true" />}
        </div>
        <span>
          <Button size="sm" disabledReason={preview.data ? undefined : tt(me.reviewLoading)} onClick={() => setOpen(true)}>
            {tt(me.reviewOpen)}
          </Button>
        </span>
      </div>
      {open && preview.data ? (
        <Dialog open onOpenChange={setOpen} size="lg" title={w.title || name} description={w.path}>
          <div className="max-h-[70vh] overflow-auto rounded-[8px] border border-line">
            <SiteFrame html={preview.data.html} title={tt(me.reviewFrame, { title: w.title || name })} minHeight={600} />
          </div>
        </Dialog>
      ) : null}
    </section>
  )
}
