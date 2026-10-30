import { Link, useNavigate } from '@tanstack/react-router'
import { useState, type ReactNode } from 'react'
import { $api, errorMessage, http } from '@/api/client'
import { Icon } from '@/components/icons/Icon'
import { Page } from '@/components/shell/Page'
import { Badge } from '@/components/ui/Badge'
import { Button, ButtonLink } from '@/components/ui/Button'
import { ConfirmDialog } from '@/components/ui/ConfirmDialog'
import { Dialog } from '@/components/ui/Dialog'
import { Note } from '@/components/ui/Note'
import { toast } from '@/components/ui/Toast'
import { T, useT } from '@/i18n'
import { ago, andList, clock, dayMonthDate, dayMonthYear, count } from '@/lib/format'
import { useCan, useWorkspace } from '@/lib/session'
import { FormsTabs, useInvalidateForms, usePublicPages } from '@/features/forms/common'
import { bare, useInvalidateWebsite, type Overview } from './api'
import { m } from './messages'
import { ReadOnlyNote, ViewSite } from './WebsiteFrame'
import './website.css'

type Layout = 'featured' | 'list' | 'grid'
type SignupList = Overview['members']['signupLists'][number]

/** The three home layouts as small wireframes (turn-on page, Home & navigation). */
export function LayoutThumbs({ value, onChange, disabled, label }: { value: Layout; onChange: (l: Layout) => void; disabled?: boolean; label: string }) {
  const t = useT()
  const items: { v: Layout; label: string; wf: ReactNode }[] = [
    {
      v: 'featured',
      label: t(m.layoutFeatured),
      wf: (
        <>
          <span className="ws-wf i" style={{ left: 8, top: 8, width: 30, height: 4 }} />
          <span className="ws-wf w" style={{ left: 8, top: 18, width: 58, height: 30 }} />
          <span className="ws-wf w" style={{ left: 72, top: 18, width: 58, height: 30 }} />
          <span className="ws-wf" style={{ left: 8, top: 56, width: 120, height: 3 }} />
          <span className="ws-wf" style={{ left: 8, top: 64, width: 96, height: 3 }} />
          <span className="ws-wf" style={{ left: 8, top: 72, width: 108, height: 3 }} />
        </>
      ),
    },
    {
      v: 'list',
      label: t(m.layoutList),
      wf: (
        <>
          <span className="ws-wf i" style={{ left: 8, top: 8, width: 30, height: 4 }} />
          {[18, 34, 50, 66].map((y) => (
            <span key={y} className="ws-wf" style={{ left: 8, top: y + 2, width: 104, height: 3 }} />
          ))}
          {[16, 32, 48, 64].map((y) => (
            <span key={y} className="ws-wf w" style={{ right: 8, top: y, width: 18, height: 12 }} />
          ))}
        </>
      ),
    },
    {
      v: 'grid',
      label: t(m.layoutGrid),
      wf: (
        <>
          <span className="ws-wf i" style={{ left: 8, top: 8, width: 30, height: 4 }} />
          {[0, 1, 2].map((c) =>
            [18, 50].map((y) => <span key={`${c}-${y}`} className="ws-wf w" style={{ left: `calc(8px + ${c} * ((100% - 16px) / 3))`, top: y, width: 'calc((100% - 28px) / 3)', height: 26 }} />),
          )}
        </>
      ),
    },
  ]
  return (
    <div className="ws-thumbs" role="radiogroup" aria-label={label}>
      {items.map((it) => (
        <button key={it.v} type="button" role="radio" aria-checked={value === it.v} className="ws-thumb" disabled={disabled} onClick={() => onChange(it.v)}>
          <span className="tv" aria-hidden="true">
            {it.wf}
          </span>
          <span className="lb">
            {value === it.v ? <Icon name="check" size={13} className="text-accent" /> : null}
            {it.label}
          </span>
        </button>
      ))}
    </div>
  )
}

/**
 * Website off (WebsiteStates): the invitation to turn the archive into a
 * website, and today's archive settings until then.
 */
export function WebsiteOff({ overview }: { overview: Overview }) {
  const t = useT()
  const { slug } = useWorkspace()
  return (
    <Page title={t(m.screenTitle)} documentTitle={`${t(m.turnOn)} · ${t(m.screenTitle)}`} filters={<FormsTabs />}>
      <div className="ws-main">
        <section className="ws-cta" aria-labelledby="ws-off-h">
          <div className="hd">
            <span className="ic" aria-hidden="true">
              <Icon name="globe" size={20} />
            </span>
            <h2 id="ws-off-h">{t(m.offTitle)}</h2>
          </div>
          <p>{t(m.offText, { host: overview.address.domain || overview.address.host })}</p>
          <div className="acts">
            <ButtonLink variant="primary" to="/w/$workspace/forms/website/turn-on" params={{ workspace: slug }}>
              {t(m.turnOn)}
            </ButtonLink>
            <span className="k">{t(m.offFree)}</span>
          </div>
        </section>
        <PublicArchiveCard overview={overview} />
      </div>
    </Page>
  )
}

/** "Public archive · Today's archive settings, until the website is turned on." */
function PublicArchiveCard({ overview }: { overview: Overview }) {
  const t = useT()
  const { slug } = useWorkspace()
  const pages = usePublicPages()
  const invalidateForms = useInvalidateForms()
  const invalidate = useInvalidateWebsite()
  const canEdit = useCan('forms:edit')
  const save = $api.useMutation('put', '/api/v1/workspaces/{workspace}/public-pages/pages')
  const a = overview.archive
  const archive = pages.data?.pages.archive
  const ready = Boolean(archive) && canEdit && !save.isPending
  const listed = (id: string) => a.lists.find((l) => l.id === id)?.archived ?? false
  const put = (patch: Partial<NonNullable<typeof archive>>) => {
    if (!archive) return
    save.mutate(
      { params: { path: { workspace: slug } }, body: { archive: { ...archive, lists: a.lists.filter((l) => l.archived).map((l) => l.id), ...patch } } },
      {
        onSuccess: () => {
          toast({ kind: 'success', title: t(m.archiveSaved) })
          void invalidateForms()
          void invalidate()
        },
        onError: (e) => toast({ kind: 'error', title: t(m.archiveSaveFailed), detail: errorMessage(e) }),
      },
    )
  }
  return (
    <section className="card" aria-labelledby="ws-arch-h">
      <div className="ws-sec">
        <div className="tx">
          <h2 className="ws-h2" id="ws-arch-h">
            {t(m.archiveTitle)}
          </h2>
          <span className="k">{t(m.archiveSub)}</span>
        </div>
      </div>
      <div className="ws-setr" style={{ borderTop: 0 }}>
        <div className="tx">
          <b id="ws-arch-pub">{t(m.archivePublic)}</b>
          <span className="k">{a.issues ? t(m.archivePublicText, { url: bare(a.url), count: a.issues, year: a.since }) : t(m.archivePublicTextNone, { url: bare(a.url) })}</span>
        </div>
        <div className="ct">
          <input type="checkbox" role="switch" className="tg" aria-labelledby="ws-arch-pub" checked={a.enabled} disabled={!ready} onChange={(e) => put({ enabled: e.target.checked })} />
        </div>
      </div>
      <div className="ws-sub">
        <b className="ws-sub-h">{t(m.archiveLists)}</b>
        <div className="ws-lchecks">
          {a.lists.map((l) => (
            <label key={l.id} className="ws-lcheck">
              <input
                type="checkbox"
                className="cb"
                checked={l.archived}
                disabled={!ready}
                onChange={(e) => {
                  const ids = a.lists.filter((x) => (x.id === l.id ? e.target.checked : listed(x.id))).map((x) => x.id)
                  put({ lists: ids })
                }}
              />
              <span className="tx">
                {l.name}
                <span className="k">{!l.public && !l.archived ? t(m.archiveListPrivate) : t(m.archiveListIssues, { count: l.issues })}</span>
              </span>
            </label>
          ))}
        </div>
      </div>
      <div className="ws-setr">
        <div className="tx">
          <b id="ws-arch-ix">{t(m.archiveIndex)}</b>
          <span className="k">{t(m.archiveIndexText)}</span>
        </div>
        <div className="ct">
          <input type="checkbox" role="switch" className="tg" aria-labelledby="ws-arch-ix" checked={a.index} disabled={!ready} onChange={(e) => put({ index: e.target.checked })} />
        </div>
      </div>
      <div className="ws-foot">
        <span className="k">
          {t(m.archiveRss)} <span className="mono">{bare(a.feed)}</span>
        </span>
        <a className="btn sm" href={a.url} target="_blank" rel="noreferrer">
          <Icon name="external" size={13} />
          {t(m.openArchive)}
        </a>
      </div>
    </section>
  )
}

/**
 * Turn on the website (WebsiteStates · ready, address problems, read-only):
 * four short checks and one button; then the progress, then "live".
 */
export function TurnOnPage({ overview }: { overview: Overview }) {
  const t = useT()
  const [justLive, setJustLive] = useState(false)
  if (overview.status === 'turning_on' || (justLive && overview.status === 'on')) {
    return (
      <Page title={t(m.screenTitle)} documentTitle={`${t(m.turnOn)} · ${t(m.screenTitle)}`} filters={<FormsTabs />}>
        <div className="ws-states">
          <TurningOn overview={overview} />
          {overview.status === 'on' ? <LiveCard overview={overview} /> : null}
        </div>
      </Page>
    )
  }
  if (overview.status === 'on') {
    return (
      <Page title={t(m.screenTitle)} documentTitle={`${t(m.turnOn)} · ${t(m.screenTitle)}`} filters={<FormsTabs />}>
        <div className="ws-states">
          <LiveCard overview={overview} />
        </div>
      </Page>
    )
  }
  return (
    <Page title={t(m.screenTitle)} documentTitle={`${t(m.turnOn)} · ${t(m.screenTitle)}`} filters={<FormsTabs />}>
      <TurnOnForm overview={overview} onStarted={() => setJustLive(true)} />
    </Page>
  )
}

function TurnOnForm({ overview, onStarted }: { overview: Overview; onStarted: () => void }) {
  const t = useT()
  const { slug } = useWorkspace()
  const canManage = useCan('website:manage')
  const invalidate = useInvalidateWebsite()
  const [layout, setLayout] = useState<Layout>(overview.homeLayout)
  const [signinOn, setSigninOn] = useState(overview.members.signinOn || overview.status === 'off')
  const [lists, setLists] = useState<SignupList[]>(() => overview.members.signupLists.map((l) => ({ ...l })))
  const [listsOpen, setListsOpen] = useState(false)
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState<string>()
  const a = overview.address
  const certFailing = Boolean(a.domain) && a.domainStatus === 'verified' && a.certificate.status === 'failing'
  const blocked = certFailing
  const offered = lists.filter((l) => l.offered)
  const turnOn = async () => {
    setBusy(true)
    setError(undefined)
    try {
      await http.POST('/api/v1/workspaces/{workspace}/website/turn-on', {
        params: { path: { workspace: slug } },
        body: { layout, signinOn, lists: lists.filter((l) => l.offered).map((l) => ({ listId: l.id, checked: l.checked, consent: l.consent })) },
      })
      onStarted()
      await invalidate()
    } catch (e) {
      setError(errorMessage(e))
    } finally {
      setBusy(false)
    }
  }
  if (!canManage) {
    return (
      <div className="ws-main" style={{ maxWidth: 880, margin: '0 auto', width: '100%' }}>
        <ReadOnlyNote msg={m.readOnlyTurnOn} helpers={overview.helpers.manage} />
        <section className="card ws-ton" aria-labelledby="ws-ton-h">
          <h2 className="t" id="ws-ton-h" style={{ fontSize: 16 }}>
            {t(m.turnOnTitle)}
          </h2>
          <div className="flex flex-col gap-2 pt-3">
            <ShortStep n={1} tone={addressTone(overview)} title={t(m.stepAddress)} text={addressShort(t, overview)} />
            <ShortStep n={2} title={t(m.stepChanges)} text={t(m.changesShort, { count: overview.archive.issues })} />
            <ShortStep n={3} title={t(m.stepLayout)} text={layoutName(t, overview.homeLayout)} />
            <ShortStep
              n={4}
              title={t(m.stepMembers)}
              text={overview.members.signinOn || overview.status === 'off' ? (offered.length ? t(m.membersShortOn, { lists: andList(offered.map((l) => l.name)) }) : t(m.membersShortOnNoLists)) : t(m.membersShortOff)}
            />
          </div>
          <div className="flex flex-wrap items-center gap-3 pt-4">
            <Button variant="primary" disabled>
              {t(m.turnOn)}
            </Button>
            <span className="k">
              {t(m.needsManage)}
            </span>
          </div>
        </section>
      </div>
    )
  }
  return (
    <section className="card ws-ton" aria-labelledby="ws-ton-h">
      <h2 className="t" id="ws-ton-h">
        {t(m.turnOnTitle)}
      </h2>
      <span className="k" style={{ paddingBottom: 6 }}>
        {t(m.turnOnSub)}
      </span>
      <div className="ws-step" style={{ borderTop: '1px solid var(--line-2)' }}>
        <StepNumber n={1} tone={addressTone(overview)} />
        <div className="bd">
          <b>{t(m.stepAddress)}</b>
          <AddressStep overview={overview} />
        </div>
      </div>
      <div className="ws-step">
        <StepNumber n={2} />
        <div className="bd">
          <b>{t(m.stepChanges)}</b>
          <ul>
            <li>{t(m.changeRoot, { host: a.domain || a.host })}</li>
            <li>
              {overview.archive.issues ? (
                <>
                  <b>{t(m.changePosts, { count: overview.archive.issues })}</b> {t(m.changeRedirect)}
                </>
              ) : (
                t(m.changePostsNone)
              )}
            </li>
            <li>{overview.archive.tags.length ? t(m.changeTags, { lists: andList(overview.archive.tags) }) : t(m.changeTagsNone)}</li>
            <li>{t(m.changeSignin)}</li>
            <li>{t(m.changeArchive)}</li>
            <li>{t(m.changePaywall)}</li>
          </ul>
        </div>
      </div>
      <div className="ws-step">
        <StepNumber n={3} />
        <div className="bd">
          <b id="ws-ton-layout">{t(m.stepLayout)}</b>
          <LayoutThumbs value={layout} onChange={setLayout} label={t(m.layoutLabel)} />
          <span className="k">{t(m.layoutLater)}</span>
        </div>
      </div>
      <div className="ws-step">
        <StepNumber n={4} />
        <div className="bd">
          <b>{t(m.stepMembers)}</b>
          <div className="ws-optbox">
            <div className="tx">
              <b id="ws-ton-signin">{t(m.membersSignin)}</b>
              <span className="k">{t(m.membersSigninText, { days: overview.members.sessionDays })}</span>
            </div>
            <input type="checkbox" role="switch" className="tg" aria-labelledby="ws-ton-signin" checked={signinOn} onChange={(e) => setSigninOn(e.target.checked)} />
          </div>
          <div className="ws-optbox">
            <div className="tx">
              <b>{t(m.signupLists)}</b>
              <span className="k">{offered.length ? offered.map((l) => (l.checked ? t(m.signupChecked, { name: l.name }) : l.name)).join(' · ') : t(m.signupListsNone)}</span>
            </div>
            <Button size="sm" onClick={() => setListsOpen(true)}>
              {t(m.change)}
            </Button>
          </div>
        </div>
      </div>
      <div className="ws-step" style={{ alignItems: 'center', flexWrap: 'wrap' }}>
        <Button variant="primary" loading={busy} disabledReason={blocked ? t(m.certUnavailable) : undefined} onClick={() => void turnOn()}>
          {t(m.turnOn)}
        </Button>
        <span className="k">{t(m.turnOnNote)}</span>
      </div>
      {error ? (
        <Note tone="bad">
          {t(m.turningOnFailed)}: {error}
        </Note>
      ) : null}
      <ListsDialog open={listsOpen} lists={lists} onClose={() => setListsOpen(false)} onChange={setLists} />
    </section>
  )
}

function addressTone(ov: Overview): 'ok' | 'warn' | 'bad' {
  const a = ov.address
  if (!a.domain) return 'ok'
  if (a.domainStatus !== 'verified') return a.domainStatus === 'failed' ? 'bad' : 'warn'
  if (a.certificate.status === 'failing') return 'bad'
  if (a.certificate.status !== 'valid') return 'warn'
  return 'ok'
}

function addressShort(t: ReturnType<typeof useT>, ov: Overview): string {
  const a = ov.address
  if (!a.domain) return t(m.addressInstanceShort, { host: a.host })
  if (a.domainStatus !== 'verified') return t(m.domainPendingTitle)
  if (a.certificate.status === 'failing') return t(m.certFailingTitle)
  return t(m.addressOkShort, { domain: a.domain })
}

function layoutName(t: ReturnType<typeof useT>, l: string): string {
  return l === 'grid' ? t(m.layoutGrid) : l === 'list' ? t(m.layoutList) : t(m.layoutFeatured)
}

function StepNumber({ n, tone }: { n: number; tone?: 'ok' | 'warn' | 'bad' }) {
  if (tone === 'ok') {
    return (
      <span className="n ok" aria-hidden="true">
        <Icon name="check" size={14} />
      </span>
    )
  }
  return (
    <span className={`n${tone ? ` ${tone}` : ''}`} aria-hidden="true">
      {n}
    </span>
  )
}

function ShortStep({ n, tone, title, text }: { n: number; tone?: 'ok' | 'warn' | 'bad'; title: string; text: string }) {
  return (
    <div className="ws-step-short">
      <span className="ws-step" style={{ padding: 0, border: 0 }}>
        <StepNumber n={n} tone={tone} />
      </span>
      <div className="tx">
        <b>{title}</b>
        <span className="k">{text}</span>
      </div>
    </div>
  )
}

/** Section 1 of the turn-on page: the address, or what keeps it from working. */
function AddressStep({ overview }: { overview: Overview }) {
  const t = useT()
  const { slug } = useWorkspace()
  const invalidate = useInvalidateWebsite()
  const invalidateForms = useInvalidateForms()
  const [checking, setChecking] = useState(false)
  const a = overview.address
  const check = async () => {
    setChecking(true)
    try {
      await http.POST('/api/v1/workspaces/{workspace}/public-pages/domain/check', { params: { path: { workspace: slug } } })
    } catch (e) {
      toast({ kind: 'error', title: t(m.checkAgain), detail: errorMessage(e) })
    } finally {
      setChecking(false)
      void invalidate()
      void invalidateForms()
    }
  }
  const actions = (note: string) => (
    <div className="flex flex-wrap items-center gap-2">
      <Button size="sm" loading={checking} busyLabel={t(m.checking)} onClick={() => void check()}>
        {t(m.checkAgain)}
      </Button>
      <ButtonLink size="sm" to="/w/$workspace/forms/branding" params={{ workspace: slug }}>
        {t(m.brandingDomain)}
      </ButtonLink>
      <span className="k">{note}</span>
    </div>
  )
  if (!a.domain) return <span>{t(m.addressInstance, { host: a.host })}</span>
  if (a.domainStatus !== 'verified') {
    return (
      <>
        <span>
          <T msg={m.domainPendingText} values={{ domain: a.domain, fallback: <span className="mono">{a.fallback}</span> }} />
        </span>
        <table className="ws-dns" aria-label={t(m.dnsLabel)}>
          <thead>
            <tr>
              <th scope="col">{t(m.dnsType)}</th>
              <th scope="col">{t(m.dnsName)}</th>
              <th scope="col">{t(m.dnsValue)}</th>
              <th scope="col">{t(m.dnsStatus)}</th>
            </tr>
          </thead>
          <tbody>
            <tr>
              <td>{'CNAME'}</td>
              <td>{a.domain.split('.')[0]}</td>
              <td>{a.target}</td>
              <td style={{ fontFamily: 'inherit' }}>
                <Badge tone="warn">{t(m.dnsNotFound)}</Badge>
              </td>
            </tr>
          </tbody>
        </table>
        {actions(a.checkedAt ? t(m.checkedAgo, { ago: ago(a.checkedAt) }) : t(m.checkedNever))}
      </>
    )
  }
  if (a.certificate.status === 'failing') {
    return (
      <>
        <Note tone="bad">
          <div className="flex flex-col gap-1">
            <b>{t(m.certFailingTitle)}</b>
            <span>{t(m.certFailingText, { domain: a.domain, detail: a.certificate.detail || t(m.certFailingDefault) })}</span>
            <span className="mono text-[12.5px]">
              {t(m.certNeeded)}: {a.domain} → {a.target}
            </span>
          </div>
        </Note>
        {actions(t(m.certRetried))}
      </>
    )
  }
  return (
    <span>
      {t(m.addressOk, {
        domain: a.domain,
        verified: a.verifiedAt ? dayMonthYear(a.verifiedAt) : '',
        issuer: a.certificate.issuer || "Let's Encrypt",
        renews: a.certificate.renewsAt ? dayMonthYear(a.certificate.renewsAt) : '',
      })}
    </span>
  )
}

/** "Lists offered on sign-up": Offered and Checked per list. */
function ListsDialog({ open, lists, onClose, onChange }: { open: boolean; lists: SignupList[]; onClose: () => void; onChange: (l: SignupList[]) => void }) {
  const t = useT()
  const set = (id: string, patch: Partial<SignupList>) => onChange(lists.map((l) => (l.id === id ? { ...l, ...patch, checked: patch.offered === false ? false : (patch.checked ?? l.checked) } : l)))
  return (
    <Dialog open={open} onOpenChange={(o) => !o && onClose()} title={t(m.listsDialogTitle)} description={t(m.listsDialogText)} footer={<Button variant="primary" onClick={onClose}>{t(m.done)}</Button>}>
      <div className="flex flex-col gap-2">
        {lists.map((l) => (
          <div key={l.id} className="ws-optbox">
            <div className="tx">
              <b>{l.name}</b>
              {l.public ? null : <span className="k">{t(m.listNotPublic)}</span>}
            </div>
            <div className="flex items-center gap-4">
              <label className="flex items-center gap-2 font-medium">
                <input type="checkbox" checked={l.offered} onChange={(e) => set(l.id, { offered: e.target.checked })} />
                {t(m.listOffered)}
              </label>
              <label className="flex items-center gap-2 font-medium">
                <input type="checkbox" checked={l.checked} disabled={!l.offered} onChange={(e) => set(l.id, { checked: e.target.checked })} />
                {t(m.listChecked)}
              </label>
            </div>
          </div>
        ))}
      </div>
    </Dialog>
  )
}

/** "Turning on news.acme.io…": tags, posts, redirects, then the pages. */
export function TurningOn({ overview }: { overview: Overview }) {
  const t = useT()
  const p = overview.progress
  const host = overview.address.domain || overview.address.host
  // Once the website is on every step is done, including steps with nothing to do.
  const finished = overview.status === 'on' || Boolean(p?.finishedAt)
  const step = (label: string, s: { done: number; total: number; names?: string[] } | undefined, extra?: string) => {
    const done = finished || Boolean(s && s.total > 0 && s.done >= s.total)
    const pct = done ? 100 : s && s.total ? Math.min(100, Math.round((s.done / s.total) * 100)) : 0
    return (
      <div className="ws-prog">
        <div className="hd">
          <b>
            {done ? <Icon name="check" size={14} className="text-accent" /> : <span className="spin sm" aria-hidden="true" />}
            {label}
          </b>
          <span className="k">{extra ?? (finished && !s?.total ? t(m.progressNothing) : s ? t(m.progressOf, { done: count(s.done), total: count(s.total) }) : t(m.progressWaiting))}</span>
        </div>
        <div className="ws-track" role="progressbar" aria-label={label} aria-valuemin={0} aria-valuemax={100} aria-valuenow={pct}>
          <span style={{ width: `${pct}%` }} />
        </div>
      </div>
    )
  }
  const started = p?.startedAt ?? overview.turnedOnAt
  return (
    <section className="card pad flex flex-col gap-4" aria-labelledby="ws-turning-h" aria-live="polite">
      <div className="flex flex-col gap-0.5">
        <h2 className="ws-h2" id="ws-turning-h">
          {finished ? t(m.turnedTitle, { host }) : t(m.turningTitle, { host })}
        </h2>
        {started ? (
          <span className="k">
            {finished && p?.finishedAt
              ? t(m.turnedSub, { time: `${dayMonthDate(started)}, ${clock(started)}`, end: clock(p.finishedAt) })
              : t(m.turningSub, { time: `${dayMonthDate(started)}, ${clock(started)}` })}
          </span>
        ) : null}
      </div>
      {step(t(m.progressTags), p?.tags, p?.tags.names?.length ? p.tags.names.join(', ') : undefined)}
      {step(t(m.progressPosts), p?.posts)}
      {step(t(m.progressRedirects), p?.redirects)}
      <div className="ws-prog">
        <div className="hd">
          <b>
            {finished || p?.pages === 'done' ? <Icon name="check" size={14} className="text-accent" /> : <span className="dot" style={{ color: 'var(--ink-4)' }} aria-hidden="true" />}
            {t(m.progressPages)}
          </b>
          <span className="k">{finished || p?.pages === 'done' ? t(m.progressDone) : t(m.progressWaiting)}</span>
        </div>
        <div className="ws-track">
          <span style={{ width: finished || p?.pages === 'done' ? '100%' : '0%' }} />
        </div>
      </div>
    </section>
  )
}

/** "Your website is live at news.acme.io" with the three next steps. */
export function LiveCard({ overview }: { overview: Overview }) {
  const t = useT()
  const { slug } = useWorkspace()
  const p = overview.progress
  const host = overview.address.domain || overview.address.host
  const params = { workspace: slug }
  const tags = p?.tags.names ?? overview.archive.tags
  return (
    <section className="card pad flex flex-col gap-3" aria-labelledby="ws-live-h">
      <div className="flex items-center gap-3">
        <span className="ws-step" style={{ padding: 0, border: 0 }}>
          <span className="n ok" aria-hidden="true" style={{ width: 36, height: 36 }}>
            <Icon name="check" size={18} />
          </span>
        </span>
        <div className="flex flex-col gap-0.5">
          <h2 className="m-0 text-[18px] font-bold" id="ws-live-h">
            {t(m.liveTitle, { host })}
          </h2>
          {p ? (
            <span className="k">
              {t(m.liveSub, { posts: p.posts.total, redirects: p.redirects.total, tags: p.tags.total, time: p.finishedAt ? clock(p.finishedAt) : '' })}
            </span>
          ) : null}
        </div>
      </div>
      <Link to="/w/$workspace/forms/website/access" params={params} className="ws-next">
        <span className="n">1</span>
        <span className="tx">
                {t(m.next1)}
          <span className="k">{t(m.next1Text)}</span>
        </span>
        <Icon name="chevronRight" size={14} />
      </Link>
      <Link to="/w/$workspace/forms/website/home" params={params} className="ws-next">
        <span className="n">2</span>
        <span className="tx">
                {t(m.next2)}
          <span className="k">{tags.length ? t(m.next2Text, { tags: andList(tags) }) : t(m.next2TextNone)}</span>
        </span>
        <Icon name="chevronRight" size={14} />
      </Link>
      <Link to="/w/$workspace/forms/pages" params={params} className="ws-next">
        <span className="n">3</span>
        <span className="tx">
                {t(m.next3)}
          <span className="k">{t(m.next3Text)}</span>
        </span>
        <Icon name="chevronRight" size={14} />
      </Link>
      <div className="flex flex-wrap gap-2 pt-1">
        <ViewSite address={overview.address} primary size="md" />
        <ButtonLink to="/w/$workspace/forms/website" params={params}>
          {t(m.goToPosts)}
        </ButtonLink>
      </div>
    </section>
  )
}

/** "Turn off the website?" (WebsiteStates · turning off): what goes offline, what stays, the typed address. */
export function TurnOffDialog({ overview, open, onOpenChange }: { overview: Overview; open: boolean; onOpenChange: (o: boolean) => void }) {
  const t = useT()
  const { slug } = useWorkspace()
  const navigate = useNavigate()
  const invalidate = useInvalidateWebsite()
  const host = overview.address.domain || overview.address.host
  const rows: [string, boolean][] = [
    [t(m.offRowHome), false],
    [t(m.offRowSignin), false],
    [t(m.offRowGated), false],
    [t(m.offRowPublic), true],
    [t(m.offRowCampaigns), true],
  ]
  return (
    <ConfirmDialog
      open={open}
      onOpenChange={onOpenChange}
      tone="bad"
      badge={t(m.turnOffTitle)}
      eyebrow={t(m.turnOffEyebrow, { host })}
      title={t(m.turnOffQuestion)}
      description={t(m.turnOffImpact, { webOnly: overview.off.webOnly, paid: overview.off.paidMembers, pages: overview.off.publicPages })}
      confirmLabel={t(m.turnOffTitle)}
      typedConfirmation={{ label: t(m.typeToConfirm, { host }), expected: host, hint: t(m.typeHint) }}
      onConfirm={async () => {
        await http.POST('/api/v1/workspaces/{workspace}/website/turn-off', { params: { path: { workspace: slug } }, body: { confirm: host } })
        toast({ kind: 'success', title: t(m.turnedOff), detail: t(m.turnedOffDetail) })
        await invalidate()
        void navigate({ to: '/w/$workspace/forms/website', params: { workspace: slug } })
      }}
    >
      <div className="flex flex-col gap-1.5">
        {rows.map(([label, stays]) => (
          <div key={label} className="ws-optbox" style={{ padding: '9px 12px' }}>
            <span>{label}</span>
            <Badge tone={stays ? 'ok' : 'bad'}>{stays ? t(m.stayOnline) : t(m.offline)}</Badge>
          </div>
        ))}
      </div>
      <span className="k">{t(m.turnOffKept)}</span>
    </ConfirmDialog>
  )
}

/** The danger card at the end of Home & navigation. */
export function TurnOffCard({ overview }: { overview: Overview }) {
  const t = useT()
  const canManage = useCan('website:manage')
  const [open, setOpen] = useState(false)
  if (!canManage) return null
  return (
    <section className="card ws-dz pad flex flex-wrap items-center justify-between gap-4" aria-labelledby="ws-off-dz">
      <div className="flex min-w-0 flex-[1_1_480px] flex-col gap-1">
        <h2 className="m-0 text-[15px] font-semibold text-bad-fg" id="ws-off-dz">
          {t(m.turnOffTitle)}
        </h2>
        <span className="k">{t(m.turnOffText)}</span>
      </div>
      <Button variant="danger" onClick={() => setOpen(true)}>
        {t(m.turnOffButton)}
      </Button>
      <TurnOffDialog overview={overview} open={open} onOpenChange={setOpen} />
    </section>
  )
}
