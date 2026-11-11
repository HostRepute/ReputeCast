import { Link, useNavigate } from '@tanstack/react-router'
import { useEffect, useRef, useState, type ReactNode } from 'react'
import { $api, errorMessage } from '@/api/client'
import { Icon } from '@/components/icons/Icon'
import { Page } from '@/components/shell/Page'
import { Badge, type Tone } from '@/components/ui/Badge'
import { Button, ButtonLink } from '@/components/ui/Button'
import { KeyValues } from '@/components/ui/Data'
import { Dialog } from '@/components/ui/Dialog'
import { Skeleton } from '@/components/ui/Feedback'
import { Menu, MenuItem, MenuSeparator } from '@/components/ui/Menu'
import { Table, Th } from '@/components/ui/Table'
import { toast } from '@/components/ui/Toast'
import { AttentionIcon, Tile } from '@/features/integrations/shared'
import '@/features/integrations/integrations.css'
import { formatList, T, useT, type MessageDescriptor } from '@/i18n'
import { ago, clock, count, dayMonthDate, shortDate } from '@/lib/format'
import { useMe, useWorkspace } from '@/lib/session'
import { formsOf, leadsBase, useActivity, useConnection, useRefreshLeads, type ActivityItem, type Connection, type LeadForm, type Outcome } from './api'
import { ChoosePagesDialog, ConnectLeadAdsDialog, ReplaceCredentialsDialog } from './ConnectLeadAdsDialog'
import { MappingDialog } from './MappingDialog'
import { ml } from './messages'
import './leads.css'

type T_ = ReturnType<typeof useT>
type DialogKind = { kind: 'credentials' } | { kind: 'pages' } | { kind: 'disconnect' } | { kind: 'map'; form: LeadForm } | { kind: 'connect' }

const mono = (c: ReactNode[]) => <span className="mono text-[12.5px]">{c}</span>

/** "14:08" today, "Tue 14 Oct 17:30" before (Recent leads). */
function when(at: string): string {
  return dayMonthDate(at) === dayMonthDate(new Date()) ? clock(at) : `${shortDate(at)} ${clock(at)}`
}

/** The header's badge: how leads arrive, or why they don't. */
export function connectionStatus(t: T_, c: Pick<Connection, 'status' | 'receiving'>): { tone: Tone; label: string } {
  if (c.status === 'disconnected') return { tone: 'neutral', label: t(ml.statusDisconnected) }
  if (c.status === 'attention') return { tone: 'warn', label: t(ml.statusAttention) }
  if (c.receiving.mode === 'polling') return { tone: 'warn', label: t(ml.statusPolling) }
  return { tone: 'ok', label: t(ml.statusReceiving) }
}

const outcomes: Record<Outcome, { tone: Tone; msg: MessageDescriptor }> = {
  created: { tone: 'ok', msg: ml.outcomeCreated },
  updated: { tone: 'info', msg: ml.outcomeUpdated },
  pending: { tone: 'info', msg: ml.outcomePending },
  blocked: { tone: 'neutral', msg: ml.outcomeBlocked },
  invalid: { tone: 'warn', msg: ml.outcomeInvalid },
  declined: { tone: 'neutral', msg: ml.outcomeDeclined },
  test: { tone: 'neutral', msg: ml.outcomeTest },
}

/** Settings › Integrations › Meta Lead Ads · Acme Marketing (IntegrationsLeadAds, LeadAdsStates). */
export function LeadAdsPage({ id }: { id: string }) {
  const { slug } = useWorkspace()
  const t = useT()
  const q = useConnection(id)
  const c = q.data
  const [dialog, setDialog] = useState<DialogKind | null>(null)
  const close = () => setDialog(null)
  useImportToasts(c)
  return (
    <Page
      title={c ? t(ml.crumb, { name: c.name }) : t(ml.metaLeadAds)}
      documentTitle={c ? t(ml.documentTitle, { name: c.name }) : t(ml.metaLeadAds)}
      breadcrumb={[
        { label: t(ml.settings), to: '/w/$workspace/settings', params: { workspace: slug } },
        { label: t(ml.integrations), to: '/w/$workspace/settings/integrations', params: { workspace: slug } },
        { label: c ? t(ml.crumb, { name: c.name }) : '…' },
      ]}
    >
      {q.isError ? (
        <section className="card flex flex-col items-center gap-2.5 p-7 text-center" role="alert">
          <AttentionIcon size={22} color="var(--bad-fg)" />
          <span className="text-[16px] font-bold">{t(ml.loadErrorTitle)}</span>
          <span className="k max-w-[440px] text-ink-2">{errorMessage(q.error) ? `${errorMessage(q.error)} ${t(ml.loadErrorBody)}` : t(ml.loadErrorBody)}</span>
          <div className="row justify-center gap-2">
            <Button variant="primary" onClick={() => void q.refetch()}>
              {t(ml.tryAgain)}
            </Button>
          </div>
        </section>
      ) : !c ? (
        <PageSkeleton />
      ) : (
        <ConnectionView c={c} onDialog={setDialog} />
      )}
      {c && dialog?.kind === 'credentials' ? <ReplaceCredentialsDialog connection={c} onOpenChange={(o) => !o && close()} /> : null}
      {c && dialog?.kind === 'pages' ? <ChoosePagesDialog connection={c} onOpenChange={(o) => !o && close()} /> : null}
      {c && dialog?.kind === 'disconnect' ? <DisconnectDialog connection={c} onOpenChange={(o) => !o && close()} /> : null}
      {c && dialog?.kind === 'map' ? <MappingDialog form={dialog.form} onOpenChange={(o) => !o && close()} /> : null}
      {dialog?.kind === 'connect' ? <ConnectLeadAdsDialog onOpenChange={(o) => !o && close()} /> : null}
    </Page>
  )
}

/** "Imported 86 leads from Field Guide download" once an import the page watched finishes. */
function useImportToasts(c: Connection | undefined) {
  const t = useT()
  const running = useRef(new Set<string>())
  const refresh = useRefreshLeads()
  useEffect(() => {
    if (!c) return
    let finished = false
    for (const f of formsOf(c)) {
      if (f.import.state === 'running') {
        running.current.add(f.id)
      } else if (running.current.delete(f.id)) {
        finished = true
        if (f.import.state === 'done') toast({ kind: 'success', title: t(ml.importDone, { count: f.import.count, form: f.name }) })
        else if (f.import.state === 'failed') toast({ kind: 'error', title: t(ml.importFailed) })
      }
    }
    if (finished) void refresh()
  }, [c, t, refresh])
}

function PageSkeleton() {
  const line = (
    <div className="flex gap-[18px]">
      <Skeleton w={90} h={10} />
      <Skeleton w={220} h={10} />
    </div>
  )
  const row = (first: boolean) => (
    <div className={`flex items-center gap-3.5 py-2.5 ${first ? '' : 'border-t border-line-2'}`}>
      <Skeleton w={160} h={12} />
      <Skeleton w={80} h={12} />
      <span className="flex-1" />
      <Skeleton w={70} h={22} r={999} />
    </div>
  )
  return (
    <div className="la-grid" aria-busy="true">
      <div className="la-col">
        <div className="card flex items-center gap-4 px-5 py-[18px]">
          <Skeleton w={52} h={52} r={12} />
          <div className="flex flex-1 flex-col gap-2">
            <Skeleton w={180} h={18} />
            <Skeleton w={260} h={10} />
          </div>
          <Skeleton w={120} h={36} r={8} />
        </div>
        <div className="card flex flex-col gap-2.5 px-[18px] py-4">
          <Skeleton w={140} h={14} />
          {line}
          {line}
          {line}
          {line}
        </div>
        <div className="card flex flex-col gap-1.5 px-[18px] py-4">
          <Skeleton w={140} h={14} />
          {row(true)}
          {row(false)}
          {row(false)}
        </div>
      </div>
      <div className="la-col">
        <div className="card flex flex-col gap-2.5 px-[18px] py-4">
          <Skeleton w={180} h={14} />
          <Skeleton w="90%" h={10} />
          <Skeleton w="80%" h={10} />
          <Skeleton w="70%" h={10} />
        </div>
      </div>
    </div>
  )
}

function ConnectionView({ c, onDialog }: { c: Connection; onDialog: (d: DialogKind) => void }) {
  const live = c.status !== 'disconnected'
  return (
    <>
      <Banners c={c} onDialog={onDialog} />
      <Header c={c} onDialog={onDialog} />
      <div className="la-grid">
        <div className="la-col">
          <HowCard c={c} />
          {live ? <FormsCard c={c} onDialog={onDialog} /> : null}
          <ActivityCard id={c.id} />
        </div>
        <div className="la-col">
          <ReceivesCard />
          <WhoCard c={c} />
        </div>
      </div>
    </>
  )
}

/** The reason buttons are disabled for people who can't manage integrations. */
function useReadOnlyReason(c: Connection): string | undefined {
  const t = useT()
  if (c.canManage) return undefined
  return c.helpers.length ? t(ml.readOnlyBody, { helpers: formatList(c.helpers, 'disjunction') }) : t(ml.readOnlyAlone)
}

function Banners({ c, onDialog }: { c: Connection; onDialog: (d: DialogKind) => void }) {
  const t = useT()
  const { slug } = useWorkspace()
  const reason = useReadOnlyReason(c)
  if (c.status === 'disconnected') {
    return (
      <div className="ih-banner" role="status">
        <div className="ih-bi">
          <AttentionIcon size={16} />
          <span className="ih-bt">{t(ml.bannerDisconnected, { name: c.name, date: c.disconnectedAt ? shortDate(c.disconnectedAt) : '' })}</span>
        </div>
      </div>
    )
  }
  const out: ReactNode[] = []
  if (c.status === 'attention') {
    const msg = { token_rejected: ml.bannerRefused, token_expired: ml.bannerExpired, app_rejected: ml.bannerApp, permission_missing: ml.bannerPermission }[c.problem || 'token_rejected']
    out.push(
      <div key="attention" className="ih-banner b" role="alert">
        <div className="ih-bi">
          <AttentionIcon size={16} color="var(--bad-fg)" />
          <span className="ih-bt">{t(msg, { name: c.name })}</span>
        </div>
        <Button size="sm" disabledReason={reason} onClick={() => onDialog({ kind: 'credentials' })}>
          {t(ml.replaceCredentials)}
        </Button>
      </div>,
    )
  } else if (c.receiving.mode === 'polling') {
    const r = c.receiving.reason
    out.push(
      <div key="polling" className="ih-banner" role="status">
        <div className="ih-bi">
          <AttentionIcon size={16} />
          <span className="ih-bt">
            {r === 'unreachable' ? <T msg={ml.bannerUnreachable} values={{ url: c.receiving.url, mono }} /> : t(r === 'quiet' ? ml.bannerQuiet : ml.bannerNotSubscribed)}
          </span>
        </div>
        <ButtonLink size="sm" to="/w/$workspace/admin/instance" params={{ workspace: slug }}>
          {t(ml.checkPublicUrl)}
        </ButtonLink>
      </div>,
    )
  }
  if (c.rateLimitedUntil) {
    out.push(
      <div key="rate" className="ih-banner border-info-line bg-info-tint text-info-ink" role="status">
        <div className="ih-bi">
          <Icon name="info" size={16} className="text-info-fg" />
          <span className="ih-bt">{t(ml.bannerRate, { time: clock(c.rateLimitedUntil) })}</span>
        </div>
      </div>,
    )
  }
  return <>{out}</>
}

function Header({ c, onDialog }: { c: Connection; onDialog: (d: DialogKind) => void }) {
  const t = useT()
  const st = connectionStatus(t, c)
  const reason = useReadOnlyReason(c)
  const refresh = useRefreshForms(c)
  const live = c.status !== 'disconnected'
  return (
    <section className="card ih-dhead" aria-label={c.name}>
      <div className="ih-dmain">
        <Tile k="meta_lead_ads" size="lg" />
        <div className="ih-dtext gap-1.5">
          <div className="row gap-2.5">
            <h2 className="ih-ttl">{c.name}</h2>
            <Badge tone={st.tone} dot>
              {st.label}
            </Badge>
          </div>
          <span className="text-[13px] text-ink-2">{t(ml.sub)}</span>
        </div>
      </div>
      {live ? (
        <div className="ih-dact">
          <Button icon="refresh" disabledReason={reason} loading={refresh.busy} onClick={refresh.run}>
            {t(ml.refreshForms)}
          </Button>
          {reason ? (
            <Button iconOnly icon="more" className="ibtn" aria-haspopup="menu" aria-label={t(ml.moreActions, { name: c.name })} disabledReason={reason} />
          ) : (
            <Menu
              label={c.name}
              trigger={
                <button type="button" className="ibtn" aria-haspopup="menu" aria-label={t(ml.moreActions, { name: c.name })}>
                  <Icon name="more" size={16} />
                </button>
              }
            >
              <MenuItem onSelect={() => onDialog({ kind: 'credentials' })}>{t(ml.replaceCredentials)}</MenuItem>
              <MenuSeparator />
              <MenuItem className="text-bad-fg" onSelect={() => onDialog({ kind: 'disconnect' })}>
                {t(ml.disconnect)}
              </MenuItem>
            </Menu>
          )}
        </div>
      ) : null}
    </section>
  )
}

function useRefreshForms(c: Connection) {
  const t = useT()
  const { slug } = useWorkspace()
  const refresh = useRefreshLeads()
  const m = $api.useMutation('post', `${leadsBase}/connections/{id}/refresh`)
  return {
    busy: m.isPending,
    run: () =>
      m.mutate(
        { params: { path: { workspace: slug, id: c.id } } },
        {
          onSuccess: () => {
            toast({ kind: 'success', title: t(ml.refreshed) })
            void refresh()
          },
          onError: (e) => toast({ kind: 'error', title: errorMessage(e) }),
        },
      ),
  }
}

/** How leads arrive: the webhook (or polling), the Meta app, the token, who connected it. */
function HowCard({ c }: { c: Connection }) {
  const t = useT()
  const r = c.receiving
  const live = c.status !== 'disconnected'
  const webhook = !live
    ? t(ml.webhookOff, { date: c.disconnectedAt ? shortDate(c.disconnectedAt) : '' })
    : r.mode === 'polling'
      ? r.lastPolledAt
        ? t(ml.webhookPolling, { ago: ago(r.lastPolledAt) })
        : t(ml.webhookPollingNever)
      : r.lastLeadAt
        ? t(ml.webhookReceiving, { ago: ago(r.lastLeadAt) })
        : t(ml.webhookReceivingNone)
  const dot = !live ? 'var(--muted-ink)' : r.mode === 'polling' ? 'var(--warn-bar)' : 'var(--ok-fg)'
  const token = !live
    ? t(ml.tokenDeleted)
    : c.problem === 'token_rejected' || c.problem === 'token_expired'
      ? t(ml.tokenRefused)
      : c.checkedAt
        ? t(ml.tokenChecked, { ago: ago(c.checkedAt) })
        : t(ml.tokenUnchecked)
  const date = shortDate(c.createdAt)
  return (
    <section className="card" aria-labelledby="la-how">
      <div className="sec">
        <div className="flex min-w-0 flex-col gap-0.5">
          <h2 className="h2" id="la-how">
            {t(ml.howTitle)}
          </h2>
        </div>
      </div>
      <div className="pad">
        <KeyValues
          className="cm-kv gap-x-[18px] gap-y-[7px]"
          items={[
            [
              t(ml.kvWebhook),
              <>
                <span className="dot mr-1.5 align-[1px]" style={{ color: dot }} aria-hidden="true" />
                {webhook}
              </>,
            ],
            [t(ml.kvMetaApp), <span className="mono text-[12.5px]">{c.appId}</span>],
            [t(ml.kvToken), token],
            [t(ml.kvConnected), c.createdBy ? t(ml.connectedBy, { date, name: c.createdBy }) : t(ml.connectedOn, { date })],
          ]}
        />
      </div>
    </section>
  )
}

function consentLabel(t: T_, f: LeadForm): string {
  return t(f.consentMode === 'single' ? ml.consentSingle : ml.consentDouble)
}

function FormStatus({ f }: { f: LeadForm }) {
  const t = useT()
  if (!f.mapped)
    return (
      <Badge tone="warn" dot>
        {t(ml.formNotMapped)}
      </Badge>
    )
  return f.status === 'active' ? (
    <Badge tone="ok" dot>
      {t(ml.formActive)}
    </Badge>
  ) : (
    <Badge tone="neutral" dot>
      {t(ml.formPaused)}
    </Badge>
  )
}

/** The line under a form's name: leads waiting at Meta, or the import running. */
function FormNote({ f }: { f: LeadForm }) {
  const t = useT()
  if (f.import.state === 'running') {
    return (
      <span className="inline-flex items-center gap-2 text-[12.5px] text-ink-nav" role="status">
        <span className="spin" aria-hidden="true" />
        {t(ml.importing, { count: count(f.import.count) })}
      </span>
    )
  }
  if (f.waiting > 0 && (!f.mapped || f.status === 'paused')) {
    return <span className="k text-warn-fg">{t(f.mapped ? ml.waitingPaused : ml.waitingUnmapped, { count: f.waiting })}</span>
  }
  return null
}

function FormActions({ f, c, onDialog }: { f: LeadForm; c: Connection; onDialog: (d: DialogKind) => void }) {
  const t = useT()
  const { slug } = useWorkspace()
  const reason = useReadOnlyReason(c)
  const refresh = useRefreshLeads()
  const pause = $api.useMutation('post', `${leadsBase}/forms/{formId}/pause`)
  const resume = $api.useMutation('post', `${leadsBase}/forms/{formId}/resume`)
  const imp = $api.useMutation('post', `${leadsBase}/forms/{formId}/import`)
  const path = { params: { path: { workspace: slug, formId: f.id } } }
  const done = (title: string) => () => {
    toast({ kind: 'success', title })
    void refresh()
  }
  const fail = (e: unknown) => toast({ kind: 'error', title: errorMessage(e) })
  const label = t(ml.formMenu, { name: f.name })
  return (
    <div className="la-acts">
      <Button size="sm" variant={f.mapped ? undefined : 'primary'} disabledReason={reason} onClick={() => onDialog({ kind: 'map', form: f })}>
        {t(ml.map)}
      </Button>
      {reason ? (
        <Button iconOnly icon="more" className="ibtn" aria-haspopup="menu" aria-label={label} disabledReason={reason} />
      ) : (
        <Menu
          label={f.name}
          trigger={
            <button type="button" className="ibtn" aria-haspopup="menu" aria-label={label}>
              <Icon name="more" size={16} />
            </button>
          }
        >
          {f.status === 'active' ? (
            <MenuItem disabled={!f.mapped} onSelect={() => pause.mutate(path, { onSuccess: done(t(ml.paused, { form: f.name })), onError: fail })}>
              {t(ml.pause)}
            </MenuItem>
          ) : (
            <MenuItem onSelect={() => resume.mutate(path, { onSuccess: done(t(ml.resumed, { form: f.name })), onError: fail })}>{t(ml.resume)}</MenuItem>
          )}
          <MenuItem
            disabled={!f.mapped || f.import.state === 'running'}
            onSelect={() => imp.mutate(path, { onSuccess: done(t(ml.importStarted, { form: f.name })), onError: fail })}
          >
            {t(ml.importPast)}
          </MenuItem>
        </Menu>
      )}
    </div>
  )
}

/** Pages and forms: a table at 1440, a list of cards at 1100 px and below. */
function FormsCard({ c, onDialog }: { c: Connection; onDialog: (d: DialogKind) => void }) {
  const t = useT()
  const reason = useReadOnlyReason(c)
  const refresh = useRefreshForms(c)
  const forms = formsOf(c)
  // List names are workspace content, separated by a comma as on the board.
  const lists = (f: LeadForm) => (f.mapped ? f.lists.map((l) => l.name).join(', ') || '—' : '—')
  const consent = (f: LeadForm) => (f.mapped ? consentLabel(t, f) : '—')
  const empty = c.pages.filter((p) => p.forms.length === 0)
  return (
    <section className="card" aria-labelledby="la-forms">
      <div className="sec">
        <div className="flex min-w-0 flex-col gap-0.5">
          <h2 className="h2" id="la-forms">
            {t(ml.formsTitle)}
          </h2>
        </div>
        <div className="flex flex-wrap items-center gap-2">
          <Button size="sm" disabledReason={reason} onClick={() => onDialog({ kind: 'pages' })}>
            {t(ml.choosePages)}
          </Button>
          <Button size="sm" icon="refresh" disabledReason={reason} loading={refresh.busy} onClick={refresh.run}>
            {t(ml.refreshForms)}
          </Button>
        </div>
      </div>
      {c.pages.length === 0 ? (
        <div className="flex flex-wrap items-center justify-between gap-3 border-t border-line-2 px-[18px] py-3.5">
          <span className="text-[13px] text-ink-nav">{t(ml.noPages)}</span>
          <Button size="sm" disabledReason={reason} onClick={() => onDialog({ kind: 'pages' })}>
            {t(ml.choosePages)}
          </Button>
        </div>
      ) : null}
      {forms.length ? (
        <>
          <div className="ih-no-narrow">
            <Table caption={t(ml.formsTitle)}>
              <thead>
                <tr>
                  <Th>{t(ml.colForm)}</Th>
                  <Th>{t(ml.colPage)}</Th>
                  <Th className="r min-w-16 whitespace-normal">
                    {t(ml.colLeads)}
                  </Th>
                  <Th>{t(ml.colLists)}</Th>
                  <Th>{t(ml.colConsent)}</Th>
                  <Th>{t(ml.colStatus)}</Th>
                  <th scope="col">
                    <span className="vh">{t(ml.colActions)}</span>
                  </th>
                </tr>
              </thead>
              <tbody>
                {forms.map((f) => {
                  const note = <FormNote f={f} />
                  const hasNote = f.import.state === 'running' || (f.waiting > 0 && (!f.mapped || f.status === 'paused'))
                  return (
                    <tr key={f.id}>
                      <td>
                        {hasNote ? (
                          <div className="flex min-w-60 flex-col gap-[3px]">
                            <span className="font-semibold">{f.name}</span>
                            {note}
                          </div>
                        ) : (
                          <span className="inline-block min-w-[150px] font-semibold">{f.name}</span>
                        )}
                      </td>
                      <td className="nw">{f.page}</td>
                      <td className="r num">{count(f.mapped ? f.leads30d : f.waiting)}</td>
                      <td className="min-w-[110px]">{lists(f)}</td>
                      <td className="nw">{consent(f)}</td>
                      <td>
                        <FormStatus f={f} />
                      </td>
                      <td className="r">
                        <FormActions f={f} c={c} onDialog={onDialog} />
                      </td>
                    </tr>
                  )
                })}
              </tbody>
            </Table>
          </div>
          <div className="ih-only-narrow">
            {forms.map((f) => (
              <div key={f.id} className="la-item">
                <span className="la-top">
                  <b className="font-semibold">{f.name}</b>
                  <FormStatus f={f} />
                </span>
                <span className="k">{t(ml.formLine, { page: f.page, count: f.mapped ? f.leads30d : f.waiting })}</span>
                {f.mapped ? <span className="text-[12.5px] text-ink-nav">{t(ml.formListsLine, { lists: lists(f), consent: consent(f) })}</span> : null}
                <FormNote f={f} />
                <div className="flex flex-nowrap items-center gap-2">
                  <FormActions f={f} c={c} onDialog={onDialog} />
                </div>
              </div>
            ))}
          </div>
        </>
      ) : null}
      {empty.map((p) => (
        <div key={p.id} className="flex flex-wrap items-center justify-between gap-3 border-t border-line-2 px-[18px] py-3">
          <span className="k text-ink-2">{t(ml.noForms, { page: p.name })}</span>
          <Button size="sm" icon="refresh" disabledReason={reason} loading={refresh.busy} onClick={refresh.run}>
            {t(ml.refreshForms)}
          </Button>
        </div>
      ))}
      <div className="ih-foot">
        <span className="k">{t(ml.formsFoot)}</span>
      </div>
    </section>
  )
}

function OutcomeBadge({ o }: { o: Outcome }) {
  const t = useT()
  const v = outcomes[o]
  return (
    <Badge tone={v.tone} dot>
      {t(v.msg)}
    </Badge>
  )
}

function ContactCell({ it }: { it: ActivityItem }) {
  const t = useT()
  const { slug } = useWorkspace()
  if (it.contact) {
    if (it.contact.erased) return <span className="k">{t(ml.erasedContact)}</span>
    return (
      <Link to="/w/$workspace/audience/contacts/$contactId" params={{ workspace: slug, contactId: it.contact.id }}>
        {it.contact.name || it.contact.email}
      </Link>
    )
  }
  return <span className="k">{it.hidden ? t(ml.aContact) : '—'}</span>
}

/** Recent leads: when, which form, what happened and the contact. No answers. */
function ActivityCard({ id }: { id: string }) {
  const t = useT()
  const q = useActivity(id)
  const items = q.data?.items
  return (
    <section className="card" aria-labelledby="la-act">
      <div className="sec">
        <div className="flex min-w-0 flex-col gap-0.5">
          <h2 className="h2" id="la-act">
            {t(ml.activityTitle)}
          </h2>
        </div>
      </div>
      {q.isError ? (
        <div className="flex flex-wrap items-center justify-between gap-3 border-t border-line-2 px-[18px] py-3" role="alert">
          <span className="text-[13px]">{errorMessage(q.error)}</span>
          <Button size="sm" onClick={() => void q.refetch()}>
            {t(ml.tryAgain)}
          </Button>
        </div>
      ) : !items ? (
        <div className="flex flex-col gap-3 border-t border-line-2 px-[18px] py-3.5" aria-busy="true">
          <Skeleton w="80%" h={12} />
          <Skeleton w="70%" h={12} />
          <Skeleton w="75%" h={12} />
        </div>
      ) : items.length === 0 ? (
        <div className="border-t border-line-2 px-[18px] py-3.5 text-[13px] text-ink-2">{t(ml.activityEmpty)}</div>
      ) : (
        <>
          <div className="ih-no-phone">
            <Table caption={t(ml.activityTitle)}>
              <thead>
                <tr>
                  <Th>{t(ml.colTime)}</Th>
                  <Th>{t(ml.colForm)}</Th>
                  <Th>{t(ml.colOutcome)}</Th>
                  <Th>{t(ml.colContact)}</Th>
                </tr>
              </thead>
              <tbody>
                {items.map((it) => (
                  <tr key={it.id}>
                    <td className="nw num">{when(it.at)}</td>
                    <td>{it.formName}</td>
                    <td>
                      <OutcomeBadge o={it.outcome} />
                    </td>
                    <td className="nw">
                      <ContactCell it={it} />
                    </td>
                  </tr>
                ))}
              </tbody>
            </Table>
          </div>
          <div className="ih-only-phone">
            {items.map((it) => (
              <div key={it.id} className="la-item">
                <span className="flex items-baseline justify-between gap-2.5">
                  <b className="font-semibold">{it.formName}</b>
                  <span className="k num whitespace-nowrap">{when(it.at)}</span>
                </span>
                <span className="flex items-center justify-between gap-2.5">
                  <OutcomeBadge o={it.outcome} />
                  <span className="text-[13px]">
                    <ContactCell it={it} />
                  </span>
                </span>
              </div>
            ))}
          </div>
        </>
      )}
      <div className="ih-foot">
        <span className="k">{t(ml.activityFoot)}</span>
      </div>
    </section>
  )
}

function ReceivesCard() {
  const t = useT()
  const tick = <Icon name="check" size={15} strokeWidth={1.8} className="text-accent" />
  return (
    <section className="card" aria-labelledby="la-rec">
      <div className="sec">
        <div className="flex min-w-0 flex-col gap-0.5">
          <h2 className="h2" id="la-rec">
            {t(ml.receivesTitle)}
          </h2>
        </div>
      </div>
      <div className="la-wd">
        <ul>
          <li>
            {tick}
            <span>{t(ml.receivesAnswers)}</span>
          </li>
          <li>
            {tick}
            <span>{t(ml.receivesConsent)}</span>
          </li>
          <li>
            {tick}
            <span>{t(ml.receivesSource)}</span>
          </li>
        </ul>
      </div>
    </section>
  )
}

function WhoCard({ c }: { c: Connection }) {
  const t = useT()
  const { me } = useWorkspace()
  const user = useMe().data
  const helpers = formatList(c.helpers, 'disjunction')
  return (
    <section className="card" aria-labelledby="la-who">
      <div className="sec">
        <div className="flex min-w-0 flex-col gap-0.5">
          <h2 className="h2" id="la-who">
            {t(ml.whoTitle)}
          </h2>
        </div>
      </div>
      <div className="pad flex flex-col gap-3">
        <p className="m-0 text-[13px] text-ink-nav">{c.helpers.length ? t(ml.whoBody, { helpers }) : t(ml.whoBodyAlone)}</p>
        {c.canManage ? null : (
          <div className="note i" role="note">
            <Icon name="info" size={16} className="mt-px flex-none" />
            <div className="flex min-w-0 flex-[1_1_200px] flex-col gap-1">
              <b className="font-semibold">{t(ml.readOnlyTitle, { name: user?.user.name ?? '', role: me.workspace.roleName })}</b>
              <span>{c.helpers.length ? t(ml.readOnlyBody, { helpers }) : t(ml.readOnlyAlone)}</span>
            </div>
          </div>
        )}
      </div>
    </section>
  )
}

/** Disconnect: what stops, what's deleted, what stays, and the safer alternative (LeadAdsStates › 8). */
function DisconnectDialog({ connection: c, onOpenChange }: { connection: Connection; onOpenChange: (o: boolean) => void }) {
  const t = useT()
  const { slug } = useWorkspace()
  const navigate = useNavigate()
  const refresh = useRefreshLeads()
  const m = $api.useMutation('post', `${leadsBase}/connections/{id}/disconnect`)
  return (
    <Dialog
      open
      onOpenChange={onOpenChange}
      title={t(ml.dcTitle, { name: c.name })}
      eyebrow={<span className="k">{t(ml.eyebrow)}</span>}
      size="sm"
      className="max-w-[520px]"
      hideClose
      dismissible={!m.isPending}
      footer={
        <>
          <Button onClick={() => onOpenChange(false)}>{t(ml.cancel)}</Button>
          <Button
            variant="danger-primary"
            loading={m.isPending}
            onClick={() =>
              m.mutate(
                { params: { path: { workspace: slug, id: c.id } } },
                {
                  onSuccess: () => {
                    toast({ kind: 'success', title: t(ml.dcDone, { name: c.name }) })
                    void refresh()
                    onOpenChange(false)
                    void navigate({ to: '/w/$workspace/settings/integrations', params: { workspace: slug } })
                  },
                  onError: (e) => toast({ kind: 'error', title: errorMessage(e) }),
                },
              )
            }
          >
            {t(ml.dcButton)}
          </Button>
        </>
      }
    >
      <ul className="m-0 flex flex-col gap-1 pl-[18px] text-[13.5px] text-ink-nav">
        <li>{t(ml.dcPages, { count: c.pages.length })}</li>
        <li>{t(ml.dcCredentials)}</li>
        <li>{t(ml.dcContacts)}</li>
      </ul>
      <div className="note i" role="note">
        <Icon name="info" size={16} className="mt-px flex-none" />
        <div className="min-w-0 flex-[1_1_320px]">{t(ml.dcSafer)}</div>
      </div>
    </Dialog>
  )
}
