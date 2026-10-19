import { Link } from '@tanstack/react-router'
import { useState, type ReactNode } from 'react'
import { errorMessage, http, type Schemas } from '@/api/client'
import { Icon } from '@/components/icons/Icon'
import { Page } from '@/components/shell/Page'
import { Badge } from '@/components/ui/Badge'
import { Button, ButtonAnchor, ButtonLink } from '@/components/ui/Button'
import { KeyValues } from '@/components/ui/Data'
import { Dialog } from '@/components/ui/Dialog'
import { Skeleton } from '@/components/ui/Feedback'
import { Note } from '@/components/ui/Note'
import { toast } from '@/components/ui/Toast'
import { T, formatList, useT } from '@/i18n'
import { ago, clock, dayMonthDate } from '@/lib/format'
import { useWorkspace } from '@/lib/session'
import { WooBadge, WooSection } from '@/features/commerce/wordpress'
import { ConnectDialog, Footer } from './ConnectDialog'
import { DisconnectDialog, RotateDialog, TakeOverDialog } from './KeyDialogs'
import { m } from './messages'
import { helpersText, KNOWN_PLUGIN, longDay, maintainer, PermChips, PLUGIN_URL, stamp, StatusBadge, tags, Tile, useRefreshHub, useWordPress, type Site, type WordPressPage as Data } from './shared'

type Check = Schemas['WordPressCheck']
type SiteDialog = { kind: 'rotate' | 'take-over' | 'disconnect' | 'rule'; site: Site } | null

/** Settings › Integrations › WordPress: the sites, their health, forms, post drafts and keys (IntegrationsWordPress). */
export function WordPressPage({ connect }: { connect?: boolean }) {
  const { slug } = useWorkspace()
  const t = useT()
  const q = useWordPress()
  const [connecting, setConnecting] = useState(Boolean(connect))
  const [dialog, setDialog] = useState<SiteDialog>(null)
  const name = t(m.wordpress)
  const data = q.data
  return (
    <Page
      title={name}
      documentTitle={`${name} · ${t(m.title)}`}
      breadcrumb={[
        { label: t(m.settings), to: '/w/$workspace/settings', params: { workspace: slug } },
        { label: t(m.title), to: '/w/$workspace/settings/integrations', params: { workspace: slug } },
        { label: name },
      ]}
    >
      {q.isError ? (
        <SitesError detail={errorMessage(q.error)} onRetry={() => void q.refetch()} />
      ) : !data ? (
        <SitesSkeleton />
      ) : (
        <>
          <Header data={data} onConnect={() => setConnecting(true)} />
          {!data.canManage ? (
            <div className="note n" role="status">
              <Badge tone="neutral">{t(m.readOnly)}</Badge>
              <span>{data.helpers.length ? t(m.readOnlyWp, { helpers: helpersText(data.helpers) }) : t(m.readOnlyWpAlone)}</span>
            </div>
          ) : null}
          {data.sites.length === 0 ? (
            <NoSites canManage={data.canManage} onConnect={() => setConnecting(true)} />
          ) : (
            <>
              <div className="row gap-x-3 gap-y-1">
                <h2 className="h2">{t(m.sitesTitle, { count: data.sites.length })}</h2>
                <span className="k">{t(m.sitesSub)}</span>
              </div>
              {data.sites.map((s, i) => (
                <SiteCard key={s.id} data={data} site={s} defaultOpen={data.sites.length <= 2 || i === 0} onDialog={setDialog} />
              ))}
            </>
          )}
        </>
      )}
      {connecting ? <ConnectDialog kind="wordpress" open onOpenChange={(o) => !o && setConnecting(false)} onDone={() => setConnecting(false)} /> : null}
      {dialog?.kind === 'rotate' ? (
        <RotateDialog connectionId={dialog.site.id} kind="wordpress" name={dialog.site.host} oldMasked={dialog.site.key?.masked ?? ''} open onOpenChange={(o) => !o && setDialog(null)} />
      ) : null}
      {dialog?.kind === 'take-over' ? <TakeOverDialog connectionId={dialog.site.id} name={dialog.site.host} open onOpenChange={(o) => !o && setDialog(null)} /> : null}
      {dialog?.kind === 'disconnect' ? (
        <DisconnectDialog connectionId={dialog.site.id} kind="wordpress" name={dialog.site.host} open onOpenChange={(o) => !o && setDialog(null)} />
      ) : null}
      {dialog?.kind === 'rule' && data ? <RuleDialog data={data} site={dialog.site} onClose={() => setDialog(null)} /> : null}
    </Page>
  )
}

function Header({ data, onConnect }: { data: Data; onConnect: () => void }) {
  const t = useT()
  const status = data.sites.length === 0 ? 'not_connected' : data.status
  return (
    <section className="card ih-dhead" aria-label={t(m.wordpress)}>
      <div className="ih-dmain">
        <Tile k="wordpress" size="lg" />
        <div className="ih-dtext">
          <div className="row gap-2.5">
            <h2 className="ih-ttl">{t(m.wordpress)}</h2>
            <StatusBadge status={status} />
          </div>
          <span className="text-[13px] text-ink-2">
            {data.sites.length
              ? t(m.wpSub, { maintainer: t(maintainer.wordpress), count: data.sites.length, name: data.connectedBy ?? '' })
              : t(m.wpSubNone, { maintainer: t(maintainer.wordpress) })}
          </span>
          <p>{t(m.wpLead)}</p>
        </div>
      </div>
      <div className="ih-dact">
        {data.canManage ? (
          <Button variant="primary" icon="plus" onClick={onConnect}>
            {t(m.connectSite)}
          </Button>
        ) : null}
        <ButtonAnchor href={PLUGIN_URL} iconRight="external">
          {t(m.getPlugin)}
          <span className="vh">{t(m.newTab)}</span>
        </ButtonAnchor>
      </div>
    </section>
  )
}

/** "Timed out after 10 s at 14:02." and the other outcomes of Check now. */
function checkSentence(t: ReturnType<typeof useT>, c: Check): string {
  const time = stamp(c.checkedAt)
  switch (c.outcome) {
    case 'timeout':
      return t(m.checkTimeout, { seconds: c.seconds ?? 10, time })
    case 'answered':
      return t(m.checkAnswered, { ms: c.ms ?? 0, time })
    case 'status':
      return t(m.checkStatus, { status: c.status ?? '', time })
    case 'blocked':
      return t(m.checkBlocked, { time, detail: c.detail.replace(/^Blocked:\s*/, '') })
  }
  return t(m.checkUnreachable, { time, detail: c.detail })
}

/** The Health list's "Check now" value: "14:02 · timed out after 10 s". */
function checkValue(t: ReturnType<typeof useT>, c: Check): string {
  const time = stamp(c.checkedAt)
  switch (c.outcome) {
    case 'timeout':
      return t(m.checkValueTimeout, { time, seconds: c.seconds ?? 10 })
    case 'answered':
      return t(m.checkValueAnswered, { time, ms: c.ms ?? 0 })
    case 'status':
      return t(m.checkValueStatus, { time, status: c.status ?? '' })
  }
  return t(m.checkValueOther, { time, detail: c.detail })
}

function SiteCard({ data, site: s, defaultOpen, onDialog }: { data: Data; site: Site; defaultOpen: boolean; onDialog: (d: SiteDialog) => void }) {
  const { slug } = useWorkspace()
  const t = useT()
  const [open, setOpen] = useState(defaultOpen)
  const [checking, setChecking] = useState(false)
  const refresh = useRefreshHub()
  const silent = s.reasons.find((r) => r.code === 'site_silent')
  const rejected = s.forms.filter((f) => !f.allowed && !f.missing).length
  const url = s.siteUrl || `https://${s.host}`
  const latest = data.latestPlugin || KNOWN_PLUGIN

  async function checkNow() {
    setChecking(true)
    try {
      await http.POST('/api/v1/workspaces/{workspace}/integrations/connections/{id}/check', { params: { path: { workspace: slug, id: s.id } } })
      await refresh()
    } catch (e) {
      toast({ kind: 'error', title: t(m.checkError, { host: s.host }), detail: errorMessage(e) })
    } finally {
      setChecking(false)
    }
  }

  const facts: [string, ReactNode][] = [
    [t(m.sitePlugin), s.pluginVersion ?? '—'],
    [t(m.siteLastHeard), s.lastReportAt ? (silent ? stamp(s.lastReportAt) : ago(s.lastReportAt)) : t(m.never)],
    [t(m.siteForms), rejected ? t(m.formsRejected, { count: s.forms.length }) : String(s.forms.length)],
    [t(m.siteDrafts), s.rule.set && s.rule.enabled ? t(m.draftsThisWeek, { count: s.draftsThisWeek }) : t(m.off)],
  ]
  const checkButton = data.canManage ? (
    <Button size="sm" icon="refresh" onClick={checkNow} loading={checking}>
      {t(m.checkNow)}
    </Button>
  ) : null
  const disconnectButton = data.canManage ? (
    <Button size="sm" variant="danger" onClick={() => onDialog({ kind: 'disconnect', site: s })}>
      {t(m.disconnectSite)}
    </Button>
  ) : null
  return (
    <section className="card" aria-label={s.host}>
      <div className={`ih-site-head${open ? ' open' : ''}`}>
        <div className="flex min-w-0 flex-[1_1_520px] flex-col gap-1.5">
          <div className="row gap-2.5">
            <h3>
              <a href={url} target="_blank" rel="noopener noreferrer">
                {s.host}
                <span className="vh">{t(m.newTab)}</span>
              </a>
            </h3>
            {silent ? (
              <Badge tone="warn" dot>
                {t(m.notHeardFrom)}
              </Badge>
            ) : s.status === 'waiting' ? (
              <Badge tone="info" dot>
                {t(m.waitingReport)}
              </Badge>
            ) : (
              <StatusBadge status={s.status} />
            )}
            {s.updateAvailable ? <Badge tone="info">{t(m.statusUpdate)}</Badge> : null}
            {/* WooCommerce on the site (X20). */}
            <WooBadge woo={s.woocommerce} />
          </div>
          <div className="ih-facts">
            {facts.map(([k, v]) => (
              <span key={k}>
                <span className="k">{k}</span> <span>{v}</span>
              </span>
            ))}
          </div>
        </div>
        <Button size="sm" aria-expanded={open} icon={open ? 'chevronUp' : 'chevronDown'} onClick={() => setOpen(!open)}>
          {open ? t(m.collapse) : t(m.expand)}
        </Button>
      </div>
      {open ? (
        <div className="pad flex flex-col gap-[18px]">
          {s.updateAvailable ? (
            <Note tone="info">
              {latest === '1.3.0' ? t(m.updateNote130, { version: latest }) : latest === '1.4.0' ? t(m.updateNote140, { version: latest }) : t(m.updateNote, { version: latest })}
            </Note>
          ) : null}
          {silent?.since ? (
            <Note tone="warn">
              <span className="font-semibold">{t(m.silentTitle, { since: stamp(silent.since) })}</span>
              <div>
                {s.lastCheck && !s.lastCheck.ok
                  ? t(m.silentBodyChecked, { check: checkSentence(t, s.lastCheck), since: stamp(silent.since) })
                  : t(m.silentBody, { since: stamp(silent.since) })}
                {data.canManage ? (
                  <div className="row mt-2 gap-2">
                    {checkButton}
                    {disconnectButton}
                  </div>
                ) : null}
              </div>
            </Note>
          ) : null}
          <div className="ih-site-body">
            <div className="flex min-w-0 flex-col gap-[22px]">
              <div className="ih-sect">
                <h3>{t(m.health)}</h3>
                <KeyValues
                  items={[
                    [t(m.sitePlugin), s.pluginVersion ? (s.updateAvailable ? t(m.pluginAvailable, { version: s.pluginVersion, latest }) : t(m.pluginLatest, { version: s.pluginVersion })) : '—'],
                    [t(m.wpVersion), s.wordpressVersion ?? '—'],
                    [t(m.phpVersion), s.phpVersion ?? '—'],
                    [t(m.siteLastHeard), s.lastReportAt ? t(m.lastHeardValue, { time: stamp(s.lastReportAt), ago: ago(s.lastReportAt) }) : t(m.never)],
                    ...(silent && s.lastCheck ? ([[t(m.checkNow), checkValue(t, s.lastCheck)]] as [string, ReactNode][]) : []),
                  ]}
                />
                {!silent ? (
                  <div className="row gap-2.5">
                    {checkButton}
                    <span className="k">{t(m.reportsEvery)}</span>
                  </div>
                ) : null}
              </div>
              <div className="ih-sect">
                <h3>{t(m.feed)}</h3>
                {s.feedUrl ? (
                  <>
                    <div className="row gap-2">
                      <span className="mono text-[12.5px]">{s.feedUrl}</span>
                      <Badge tone="neutral">{t(m.notUsedYet)}</Badge>
                    </div>
                    <span className="k">{t(m.feedLine)}</span>
                  </>
                ) : (
                  <span className="k">{t(m.noFeed)}</span>
                )}
              </div>
              <WooSection woo={s.woocommerce} canManage={data.canManage} />
              {s.key ? (
                <div className="ih-sect">
                  <h3>{t(m.key)}</h3>
                  <div className="flex flex-col gap-2.5">
                    <span>
                      <span className="font-semibold">{s.key.name}</span> <span className="mono k">{s.key.masked}</span>
                    </span>
                    <span className="k">
                      {s.key.lastUsedAt
                        ? t(m.keyCreatedLine, { date: longDay(s.key.createdAt), name: s.key.owner.name, time: stamp(s.key.lastUsedAt) })
                        : t(m.keyCreatedNever, { date: longDay(s.key.createdAt), name: s.key.owner.name })}
                    </span>
                    <PermChips perms={s.key.permissions} notes={{ 'subscribers:view': t(m.onlyListForms), 'campaigns:edit': t(m.onlyCreateDrafts) }} />
                    {data.canManage ? (
                      <div className="row gap-2">
                        <Button size="sm" onClick={() => onDialog({ kind: 'rotate', site: s })}>
                          {t(m.rotate)}
                        </Button>
                        <Button size="sm" onClick={() => onDialog({ kind: 'take-over', site: s })}>
                          {t(m.takeOver)}
                        </Button>
                        {disconnectButton}
                      </div>
                    ) : null}
                  </div>
                </div>
              ) : null}
            </div>
            <div className="flex min-w-0 flex-col gap-[22px]">
              <Forms data={data} site={s} />
              <Drafts data={data} site={s} onRule={() => onDialog({ kind: 'rule', site: s })} />
            </div>
          </div>
        </div>
      ) : null}
    </section>
  )
}

/** Signup forms on the site: where each is used, and whether the site is one of its allowed domains. */
function Forms({ data, site: s }: { data: Data; site: Site }) {
  const { slug } = useWorkspace()
  const t = useT()
  const refresh = useRefreshHub()
  const [busy, setBusy] = useState<string | null>(null)
  async function allow(formId: string, formName: string) {
    setBusy(formId)
    try {
      await http.POST('/api/v1/workspaces/{workspace}/forms/{form}/allowed-origins', { params: { path: { workspace: slug, form: formId } }, body: { origin: s.host } })
      await refresh()
      toast({ title: t(m.allowedToast, { host: s.host, form: formName }) })
    } catch (e) {
      toast({ kind: 'error', title: t(m.allowError, { host: s.host }), detail: errorMessage(e) })
    } finally {
      setBusy(null)
    }
  }
  const places = (f: Site['forms'][number]) => (f.places.length > 2 ? t(m.formPlaces, { count: f.places.length }) : f.places.length ? formatList(f.places.map((p) => p.title)) : t(m.formPlaces, { count: 0 }))
  return (
    <div className="ih-sect">
      <h3>{t(m.formsTitle)}</h3>
      {s.forms.length === 0 ? (
        <span className="k">{t(m.noForms)}</span>
      ) : (
        <div className="ih-frows">
          {s.forms.map((f) => (
            <div key={f.publicId}>
              <div className="flex min-w-0 flex-[1_1_220px] flex-col gap-0.5">
                <span className="font-semibold">
                  {f.name} <span className="mono k">{f.publicId}</span>
                </span>
                <span className="k text-ink-2">{t(m.formKind, { kind: f.kind, places: places(f) })}</span>
                {f.missing ? <span className="mt-1 text-[13px] text-ink-nav">{t(m.formGone)}</span> : null}
                {!f.allowed && !f.missing ? (
                  <div className="mt-1.5 flex flex-col gap-2">
                    <span className="text-[13px] text-ink-nav">{t(m.formRejected, { host: s.host })}</span>
                    <div className="row gap-2">
                      {data.canEditForms ? (
                        <Button size="sm" variant="primary" onClick={() => allow(f.id, f.name)} loading={busy === f.id}>
                          {t(m.allowHost, { host: s.host })}
                        </Button>
                      ) : null}
                      <ButtonLink size="sm" to="/w/$workspace/forms" params={{ workspace: slug }}>
                        {t(m.formSettings)}
                      </ButtonLink>
                    </div>
                  </div>
                ) : null}
              </div>
              {f.missing ? (
                <Badge tone="warn">{t(m.formMissing)}</Badge>
              ) : f.allowed ? (
                <Badge tone="ok">{t(m.allowedDomain)}</Badge>
              ) : (
                <Badge tone="bad">{t(m.notAllowed)}</Badge>
              )}
            </div>
          ))}
        </div>
      )}
    </div>
  )
}

/** Post drafts: the rule, on or off, and the latest drafts with links. */
function Drafts({ data, site: s, onRule }: { data: Data; site: Site; onRule: () => void }) {
  const { slug } = useWorkspace()
  const t = useT()
  const refresh = useRefreshHub()
  const [busy, setBusy] = useState(false)
  const r = s.rule
  async function setEnabled(enabled: boolean) {
    setBusy(true)
    try {
      await http.PUT('/api/v1/workspaces/{workspace}/integrations/connections/{id}/rule', {
        params: { path: { workspace: slug, id: s.id } },
        body: {
          enabled,
          categories: r.categories.map((c) => c.slug),
          listIds: r.listIds,
          templateId: r.template?.id,
          senderId: r.sender?.id,
          notifyUserId: r.notify?.id,
        },
      })
      await refresh()
      toast({ title: t(m.ruleSavedToast) })
    } catch (e) {
      toast({ kind: 'error', title: t(m.ruleError), detail: errorMessage(e) })
    } finally {
      setBusy(false)
    }
  }
  if (!r.set) {
    return (
      <div className="ih-sect">
        <div className="row gap-2">
          <h3 className="ih-h3x">{t(m.draftsTitle)}</h3>
          <Badge tone="neutral">{t(m.off)}</Badge>
        </div>
        <div className="row gap-2.5">
          <span className="k">{t(m.noRule)}</span>
          {data.canManage ? (
            <Button size="sm" onClick={onRule}>
              {t(m.setUpRule)}
            </Button>
          ) : null}
        </div>
      </div>
    )
  }
  const chips = (
    <>
      {r.categories.map((c, i) => (
        <span key={c.slug}>
          {i > 0 ? ' ' : null}
          <span className="chip h-[22px]">{c.name}</span>
        </span>
      ))}
    </>
  )
  const sender = r.sender ? (r.sender.detail ? `${r.sender.name} <${r.sender.detail}>` : r.sender.name) : ''
  const values = { ...tags, categories: chips, list: r.list?.name ?? '', template: r.template?.name ?? '', sender, notify: r.notify?.name ?? '' }
  return (
    <div className="ih-sect">
      <div className="row justify-between gap-2">
        <h3 className="ih-h3x">{t(m.draftsTitle)}</h3>
        <span className="k">{t(m.draftsOnly)}</span>
      </div>
      <div className="row items-start justify-between gap-3">
        <span className="min-w-0 flex-[1_1_260px] leading-[1.7] text-ink-nav">
          <T msg={r.notify ? m.ruleSentence : m.ruleSentenceNoNotify} values={values} />
        </span>
        {data.canManage ? (
          <div className="row flex-nowrap gap-2">
            <input type="checkbox" role="switch" className="tg" checked={r.enabled} disabled={busy} aria-label={t(m.ruleOnLabel)} onChange={(e) => setEnabled(e.target.checked)} />
            <Button size="sm" onClick={onRule}>
              {t(m.editRule)}
            </Button>
          </div>
        ) : null}
      </div>
      {!r.enabled ? <span className="k">{t(m.ruleOff)}</span> : null}
      <span className="k mt-0.5">{t(m.latestDrafts)}</span>
      {s.drafts.length === 0 ? (
        <span className="k">{t(m.noDraftsYet)}</span>
      ) : (
        s.drafts.map((d, i) => (
          <div key={i} className="ih-drow">
            <div className="flex min-w-0 flex-[1_1_220px] flex-col gap-0.5">
              <span className="font-semibold">
                {d.title} <Badge tone="neutral">{t(m.draftBadge)}</Badge>
              </span>
              <span className="k text-ink-2">
                {d.notified
                  ? t(m.draftLineNotified, { published: d.publishedAt ? dayStampShort(d.publishedAt) : '', created: clock(d.createdAt), list: d.audience ?? '', notified: d.notified })
                  : t(m.draftLine, { published: d.publishedAt ? dayStampShort(d.publishedAt) : '', created: clock(d.createdAt), list: d.audience ?? '' })}
              </span>
            </div>
            {d.campaignId ? (
              <Link className="btn sm" to="/w/$workspace/campaigns/$campaignId/edit" params={{ workspace: slug, campaignId: d.campaignId }}>
                {t(m.openDraft)}
              </Link>
            ) : null}
          </div>
        ))
      )}
    </div>
  )
}

/** "15 Oct 10:04" */
function dayStampShort(d: string): string {
  return `${dayMonthDate(d)} ${clock(d)}`
}

/** The post-draft rule: categories (from the plugin), audience, template, sender, who's notified. */
function RuleDialog({ data, site: s, onClose }: { data: Data; site: Site; onClose: () => void }) {
  const { slug } = useWorkspace()
  const t = useT()
  const r = s.rule
  const [cats, setCats] = useState(new Set(r.categories.map((c) => c.slug)))
  const [listId, setListId] = useState(r.listIds[0] ?? '')
  const [templateId, setTemplateId] = useState(r.template?.id ?? '')
  const [senderId, setSenderId] = useState(r.sender?.id ?? '')
  const [notify, setNotify] = useState(r.notify?.id ?? '')
  const [enabled, setEnabled] = useState(r.set ? r.enabled : true)
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState<string>()
  const refresh = useRefreshHub()
  const complete = cats.size > 0 && listId && templateId && senderId

  async function save() {
    setBusy(true)
    setError(undefined)
    try {
      await http.PUT('/api/v1/workspaces/{workspace}/integrations/connections/{id}/rule', {
        params: { path: { workspace: slug, id: s.id } },
        body: { enabled, categories: [...cats], listIds: listId ? [listId] : [], templateId: templateId || undefined, senderId: senderId || undefined, notifyUserId: notify || undefined },
      })
      await refresh()
      toast({ title: t(m.ruleSavedToast) })
      onClose()
    } catch (e) {
      setError(errorMessage(e))
    } finally {
      setBusy(false)
    }
  }

  return (
    <Dialog open onOpenChange={(o) => !o && onClose()} eyebrow={<span className="k">{t(m.ruleEyebrow, { host: s.host })}</span>} title={t(m.ruleTitle, { host: s.host })} hideClose>
      <div className="flex flex-col gap-2">
        <span className="lbl">{t(m.whenPublished)}</span>
        {s.categories.length ? (
          <div className="ih-chips" role="group" aria-label={t(m.categoriesLabel)}>
            {s.categories.map((c) => {
              const on = cats.has(c.slug)
              return (
                <button
                  key={c.slug}
                  type="button"
                  className="ih-chip"
                  aria-pressed={on}
                  onClick={() =>
                    setCats((prev) => {
                      const next = new Set(prev)
                      if (on) next.delete(c.slug)
                      else next.add(c.slug)
                      return next
                    })
                  }
                >
                  {on ? <Icon name="check" size={13} strokeWidth={2.2} /> : null}
                  {c.name}
                </button>
              )
            })}
          </div>
        ) : null}
        <span className="k">{s.categories.length ? t(m.categoriesFrom, { count: s.categories.length, host: s.host }) : t(m.noCategories)}</span>
      </div>
      <div className="grid gap-3 sm:grid-cols-2">
        <label className="lbl">
          {t(m.audience)}
          <select className="in" value={listId} onChange={(e) => setListId(e.target.value)}>
            <option value="">{t(m.pick)}</option>
            {data.lists.map((l) => (
              <option key={l.id} value={l.id}>
                {t(m.listOption, { name: l.name, count: l.contacts })}
              </option>
            ))}
          </select>
        </label>
        <label className="lbl">
          {t(m.template)}
          <select className="in" value={templateId} onChange={(e) => setTemplateId(e.target.value)}>
            <option value="">{t(m.pick)}</option>
            {data.templates.map((o) => (
              <option key={o.id} value={o.id}>
                {o.name}
              </option>
            ))}
          </select>
        </label>
        <label className="lbl">
          {t(m.sender)}
          <select className="in" value={senderId} onChange={(e) => setSenderId(e.target.value)}>
            <option value="">{t(m.pick)}</option>
            {data.senders.map((o) => (
              <option key={o.id} value={o.id}>
                {o.detail ? `${o.name} <${o.detail}>` : o.name}
              </option>
            ))}
          </select>
        </label>
        <label className="lbl">
          {t(m.notify)}
          <select className="in" value={notify} onChange={(e) => setNotify(e.target.value)}>
            <option value="">{t(m.nobody)}</option>
            {data.members.map((o) => (
              <option key={o.id} value={o.id}>
                {o.name}
              </option>
            ))}
          </select>
          <span className="k font-medium">{t(m.notifyHint)}</span>
        </label>
      </div>
      <span className="k">{t(m.draftTakes)}</span>
      <Note tone="neutral" className="font-normal [font-variant-numeric:normal]">
        {t(m.draftsOnlyNote)}
      </Note>
      {error ? (
        <Note tone="bad" role="alert">
          {error}
        </Note>
      ) : null}
      <Footer>
        <label className="row mr-auto gap-2 text-[13px]">
          <input type="checkbox" role="switch" className="tg" checked={enabled} onChange={(e) => setEnabled(e.target.checked)} />
          {t(m.ruleOn)}
        </label>
        <Button onClick={onClose}>{t(m.cancel)}</Button>
        <Button variant="primary" onClick={save} loading={busy} disabledReason={complete ? undefined : t(m.ruleMissing)}>
          {t(m.saveRule)}
        </Button>
      </Footer>
    </Dialog>
  )
}

function NoSites({ canManage, onConnect }: { canManage: boolean; onConnect: () => void }) {
  const t = useT()
  return (
    <section className="card ih-empty">
      <Tile k="wordpress" size="lg" />
      <span className="ih-et">{t(m.noSitesTitle)}</span>
      <span className="ih-ed">{t(m.noSitesBody)}</span>
      <div className="row justify-center gap-2">
        {canManage ? (
          <Button variant="primary" onClick={onConnect}>
            {t(m.connectSite)}
          </Button>
        ) : null}
        <ButtonAnchor href={PLUGIN_URL} iconRight="external">
          {t(m.getPlugin)}
          <span className="vh">{t(m.newTab)}</span>
        </ButtonAnchor>
      </div>
      <span className="k">{t(m.noSitesFoot)}</span>
    </section>
  )
}

function SitesSkeleton() {
  const t = useT()
  return (
    <section className="card flex flex-col gap-4 p-5" aria-busy="true">
      <span className="vh" role="status">
        {t(m.loadingSites)}
      </span>
      {[0, 1, 2].map((i) => (
        <div key={i} className="row gap-3">
          <Skeleton w={140} h={16} />
          <Skeleton w={90} h={22} r={999} />
          <Skeleton w={220} />
        </div>
      ))}
    </section>
  )
}

function SitesError({ detail, onRetry }: { detail: string; onRetry: () => void }) {
  const { slug } = useWorkspace()
  const t = useT()
  return (
    <section className="card flex flex-col items-start gap-2.5 px-5 py-4.5" role="alert">
      <span className="row gap-2 font-semibold">
        <Icon name="alertCircle" size={18} className="text-bad-fg" />
        {t(m.loadSitesError)}
      </span>
      <span className="k">{t(m.loadSitesErrorBody, { detail })}</span>
      <div className="row gap-2">
        <Button variant="primary" onClick={onRetry}>
          {t(m.retry)}
        </Button>
        <ButtonLink to="/w/$workspace" params={{ workspace: slug }}>
          {t(m.statusOnOverview)}
        </ButtonLink>
      </div>
    </section>
  )
}
