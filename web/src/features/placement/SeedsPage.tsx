import { Link, useNavigate } from '@tanstack/react-router'
import { useEffect, useState, type ReactNode } from 'react'
import { errorMessage, http } from '@/api/client'
import { Icon } from '@/components/icons/Icon'
import { Badge } from '@/components/ui/Badge'
import { Button } from '@/components/ui/Button'
import { Loading, Skeleton } from '@/components/ui/Feedback'
import { toast } from '@/components/ui/Toast'
import { T, useT } from '@/i18n'
import { clock, orList, shortDate } from '@/lib/format'
import { useMe, useWorkspace } from '@/lib/session'
import { m } from './messages'
import { PlacementShell } from './PlacementShell'
import { AddSeedDialog, inboxNames, ReconnectDialog, RemoveSeedDialog, type AddNote } from './SeedDialogs'
import { base, dayTime, LoadError, SeedAttention, useRefreshPlacement, useSeeds, type SeedList, type SeedView } from './shared'

export type SeedsSearch = {
  add?: boolean
  reconnect?: string
  oauth?: 'cancelled' | 'failed' | 'forbidden'
  reason?: string
  provider?: string
  added?: string
}

/** Deliverability › Placement tests › Seed mailboxes. */
export function SeedsPage({ search }: { search: SeedsSearch }) {
  const t = useT()
  const q = useSeeds()
  return (
    <PlacementShell current="seeds" documentTitle={t(m.documentSeeds)}>
      {q.isPending ? (
        <SeedsLoading />
      ) : q.isError ? (
        <LoadError title={t(m.loadSeedsError)} error={q.error} body={(status) => t(m.loadSeedsBody, { status })} onRetry={() => void q.refetch()} />
      ) : (
        <SeedsBody d={q.data} search={search} />
      )}
    </PlacementShell>
  )
}

type Open = { kind: 'add'; provider?: string; note?: AddNote } | { kind: 'reconnect'; seed: SeedView } | { kind: 'remove'; seed: SeedView }

function SeedsBody({ d, search }: { d: SeedList; search: SeedsSearch }) {
  const t = useT()
  const { slug } = useWorkspace()
  const navigate = useNavigate()
  // Links from elsewhere (Reconnect, Add seed mailbox) and the OAuth return
  // open a dialog when the page opens.
  const [open, setOpen] = useState<Open | undefined>(() => opened(search, d))

  useEffect(() => {
    if (!search.add && !search.reconnect && !search.oauth && !search.added) return
    if (search.oauth === 'forbidden') toast({ kind: 'error', title: t(m.oauthForbidden) })
    if (search.added) {
      const s = d.seeds.find((x) => x.id === search.added)
      if (s) toast({ kind: 'success', title: t(m.oauthAdded, { email: s.email }) })
    }
    void navigate({ to: '/w/$workspace/deliverability/placement/seeds', params: { workspace: slug }, search: {}, replace: true })
  }, [search, d, navigate, slug, t])

  const attention = d.seeds.filter((s) => s.status === 'failed' || s.status === 'expired')
  const expired = attention.find((s) => s.status === 'expired' && s.auth === 'oauth')
  return (
    <>
      {d.canManage && expired ? <ExpiredCard seed={expired} onReconnect={() => setOpen({ kind: 'reconnect', seed: expired })} /> : null}
      {d.canManage && attention.filter((s) => s !== expired).length ? (
        <SeedAttention seeds={attention.filter((s) => s !== expired)} onReconnect={(s) => setOpen({ kind: 'reconnect', seed: s })} />
      ) : null}
      {d.canManage ? null : <CantManage helpers={d.manageHelpers} />}
      {d.seeds.length === 0 ? (
        <section className="card pl-empty">
          <svg width="72" height="56" viewBox="0 0 72 56" aria-hidden="true">
            <rect x="4" y="12" width="30" height="22" rx="3" fill="none" stroke="#9AA3B2" strokeWidth="2" />
            <path d="m5 14 14 10 14-10" fill="none" stroke="#9AA3B2" strokeWidth="2" />
            <rect x="38" y="22" width="30" height="22" rx="3" fill="none" stroke="var(--accent)" strokeWidth="2" />
            <path d="m39 24 14 10 14-10" fill="none" stroke="var(--accent)" strokeWidth="2" />
          </svg>
          <h2 className="tt">{t(m.seedsEmptyTitle)}</h2>
          <span className="ds">{t(m.seedsEmptyBody)}</span>
          {d.canManage ? (
            <Button variant="primary" icon="plus" className="min-h-10" onClick={() => setOpen({ kind: 'add' })}>
              {t(m.addSeed)}
            </Button>
          ) : null}
          <span className="k">{t(m.seedsEmptyHint)}</span>
        </section>
      ) : (
        <SeedsCard d={d} onAdd={() => setOpen({ kind: 'add' })} onReconnect={(s) => setOpen({ kind: 'reconnect', seed: s })} onRemove={(s) => setOpen({ kind: 'remove', seed: s })} />
      )}
      <div className="flex flex-wrap items-start gap-4">
        <HowSeedsConnect />
        <DeletionCard d={d} />
      </div>
      {open?.kind === 'add' ? <AddSeedDialog d={d} open onOpenChange={(o) => !o && setOpen(undefined)} preset={open.provider} note={open.note} /> : null}
      {open?.kind === 'reconnect' ? <ReconnectDialog d={d} seed={open.seed} open onOpenChange={(o) => !o && setOpen(undefined)} /> : null}
      {open?.kind === 'remove' ? <RemoveSeedDialog seed={open.seed} open onOpenChange={(o) => !o && setOpen(undefined)} /> : null}
    </>
  )
}

function opened(search: SeedsSearch, d: SeedList): Open | undefined {
  if (!d.canManage) return undefined
  if (search.oauth === 'cancelled' || search.oauth === 'failed') return { kind: 'add', provider: search.provider, note: { kind: search.oauth, reason: search.reason } }
  const s = search.reconnect ? d.seeds.find((x) => x.id === search.reconnect) : undefined
  if (s) return { kind: 'reconnect', seed: s }
  return search.add ? { kind: 'add' } : undefined
}

function ExpiredCard({ seed, onReconnect }: { seed: SeedView; onReconnect: () => void }) {
  const t = useT()
  const at = seed.statusAt ?? seed.addedAt
  const today = shortDate(at) === shortDate(new Date())
  const app = seed.provider === 'outlook' || seed.provider === 'microsoft_365' ? 'Microsoft' : 'Google'
  return (
    <div className="note w" style={{ flexWrap: 'nowrap' }} role="note">
      <Icon name="alertCircle" size={16} className="mt-px flex-none text-warn-bar" />
      <div className="flex min-w-0 flex-[1_1_320px] flex-col gap-1">
        <span className="font-bold">{t(m.expiredCardTitle, { provider: seed.providerName })}</span>
        <span>
          <T
            msg={m.expiredCardBody}
            values={{
              company: seed.company,
              email: seed.email,
              when: today ? t(m.whenToday, { time: clock(at) }) : t(m.whenOn, { date: shortDate(at), time: clock(at) }),
              mono: (c) => <span className="mono">{c}</span>,
            }}
          />
        </span>
        <Button size="sm" variant="primary" className="mt-1.5 w-fit" icon="external" onClick={onReconnect}>
          {t(m.reconnectWith, { app })}
        </Button>
      </div>
    </div>
  )
}

function CantManage({ helpers }: { helpers: string[] }) {
  const t = useT()
  const me = useMe().data
  const { me: ws } = useWorkspace()
  return (
    <div className="note n" role="note">
      <Icon name="info" size={16} className="mt-px flex-none text-ink-3" />
      <div className="flex min-w-0 flex-col gap-1">
        {me ? <span className="font-bold">{t(m.readOnlyWho, { name: me.user.name, role: ws.workspace.roleName })}</span> : null}
        <span>
          <T msg={helpers.length ? m.cantManage : m.cantManageNobody} values={{ helpers: orList(helpers), code: (c) => <span className="mono">{c}</span> }} />
        </span>
      </div>
    </div>
  )
}

/** "Inbox · 4 tabs · Spam", "Focused · Other · Junk Email". */
function foldersLine(s: SeedView, t: ReturnType<typeof useT>): string {
  const tabs = s.folders.filter((f) => f.role === 'tab').length
  const spam = s.folders.find((f) => f.role === 'spam')?.label
  return [...inboxNames(s.folders), tabs ? t(m.tabsCount, { count: tabs }) : '', spam ?? ''].filter(Boolean).join(' · ')
}

function StatusBadge({ s }: { s: SeedView }) {
  const t = useT()
  switch (s.status) {
    case 'connected':
      return (
        <Badge tone="ok" dot>
          {t(m.seedConnected)}
        </Badge>
      )
    case 'failed':
      return (
        <Badge tone="bad" dot>
          {t(m.seedFailed)}
        </Badge>
      )
    case 'expired':
      return (
        <Badge tone="warn" dot>
          {t(m.seedExpired)}
        </Badge>
      )
  }
  return <Badge tone="neutral">{t(m.seedUnchecked)}</Badge>
}

function since(s: SeedView, t: ReturnType<typeof useT>): string {
  if (s.status === 'connected') return s.checkedAt ? t(m.checkedOn, { date: shortDate(s.checkedAt) }) : ''
  const at = s.statusAt ?? s.checkedAt
  if (!at) return ''
  return shortDate(at) === shortDate(new Date()) ? t(m.sinceToday, { time: clock(at) }) : t(m.sinceOn, { date: shortDate(at), time: clock(at) })
}

const rowTint = (s: SeedView) => (s.status === 'failed' ? 'var(--bad-tint)' : s.status === 'expired' ? '#FFFBEB' : undefined)

function SeedsCard({ d, onAdd, onReconnect, onRemove }: { d: SeedList; onAdd: () => void; onReconnect: (s: SeedView) => void; onRemove: (s: SeedView) => void }) {
  const t = useT()
  const conn = (s: SeedView) =>
    s.auth === 'oauth' ? (
      <span className="flex flex-col gap-0.5">
        <span>{t(m.connOAuth)}</span>
        <span className="k">{s.provider === 'outlook' || s.provider === 'microsoft_365' ? t(m.connMicrosoft) : t(m.connGoogle)}</span>
      </span>
    ) : (
      <span>{t(m.connImap, { password: s.passwordLabel.toLowerCase() })}</span>
    )
  const actions = (s: SeedView) =>
    d.canManage ? (
      <span className="flex flex-nowrap gap-2 whitespace-nowrap lg:justify-end">
        {s.status === 'failed' || s.status === 'expired' ? (
          <Button size="sm" variant="primary" onClick={() => onReconnect(s)}>
            {t(m.reconnect)}
          </Button>
        ) : null}
        <Button size="sm" variant="danger" onClick={() => onRemove(s)} aria-label={t(m.removeNamed, { email: s.email })}>
          {t(m.remove)}
        </Button>
      </span>
    ) : null
  const used = (s: SeedView) =>
    s.lastUsedAt ? (
      <span className="flex flex-col gap-0.5">
        <span className="whitespace-nowrap">{dayTime(s.lastUsedAt)}</span>
        {s.lastTest ? <span className="k whitespace-nowrap">{s.lastTest}</span> : null}
      </span>
    ) : (
      <span className="k">{t(m.notUsedYet)}</span>
    )
  return (
    <section className="card">
      <div className="sec">
        <div className="flex min-w-0 flex-col gap-0.5">
          <h2 className="h2">{t(m.seedsTitle, { count: d.seeds.length })}</h2>
          <span className="k">{t(m.seedsLead)}</span>
        </div>
        {d.canManage ? (
          <Button variant="primary" icon="plus" onClick={onAdd}>
            {t(m.addSeed)}
          </Button>
        ) : null}
      </div>
      <div className="pl-lg-only">
        <div className="relative overflow-x-auto">
          <table className="tbl" aria-label={t(m.seedsTableLabel)}>
            <thead>
              <tr>
                <th scope="col">{t(m.colSeedMailbox)}</th>
                <th scope="col">{t(m.colProviderFolders)}</th>
                <th scope="col">{t(m.colConnectedWith)}</th>
                <th scope="col">{t(m.colStatus)}</th>
                <th scope="col">{t(m.colLastUsed)}</th>
                <th scope="col">{t(m.colAddedBy)}</th>
                {d.canManage ? (
                  <th scope="col">
                    <span className="vh">{t(m.colActions)}</span>
                  </th>
                ) : null}
              </tr>
            </thead>
            <tbody>
              {d.seeds.map((s) => (
                <tr key={s.id} style={{ background: rowTint(s) }}>
                  <td className="mono nw text-[12.5px]">{s.email}</td>
                  <td>
                    <span className="flex flex-col gap-0.5">
                      <b className="font-semibold">{s.providerName}</b>
                      <span className="k">{foldersLine(s, t)}</span>
                    </span>
                  </td>
                  <td>{conn(s)}</td>
                  <td>
                    <span className="flex flex-col items-start gap-1">
                      <StatusBadge s={s} />
                      <span className="k whitespace-nowrap">{since(s, t)}</span>
                    </span>
                  </td>
                  <td>{used(s)}</td>
                  <td>
                    <span className="flex flex-col gap-0.5">
                      <span className="whitespace-nowrap">{s.addedBy}</span>
                      <span className="k">{shortDate(s.addedAt)}</span>
                    </span>
                  </td>
                  {d.canManage ? <td className="r">{actions(s)}</td> : null}
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </div>
      <div className="pl-md-down">
        {d.seeds.map((s) => {
          const at = s.statusAt ?? s.checkedAt
          const auth = s.auth === 'oauth' ? t(m.connOAuth) : t(m.connImap, { password: s.passwordLabel.toLowerCase() })
          const added = { name: s.addedBy, date: shortDate(s.addedAt) }
          const used = s.lastUsedAt ? t(m.phoneUsedLine, { when: dayTime(s.lastUsedAt), ...added }) : t(m.phoneUsedNever, added)
          const problem =
            at && s.status === 'failed'
              ? t(m.phoneFailedSince, { date: shortDate(at), time: clock(at) })
              : at && s.status === 'expired'
                ? t(m.phoneExpiredSince, { date: shortDate(at), time: clock(at) })
                : ''
          const last = s.lastUsedAt ? t(m.phoneUsedOnly, { when: dayTime(s.lastUsedAt) }) : t(m.notUsedYet)
          return (
            <div key={s.id} className="flex flex-col gap-[5px] border-t border-line-2 px-4 py-3" style={{ background: rowTint(s) }}>
              <span className="flex flex-wrap items-center justify-between gap-2">
                <b className="font-semibold">{s.providerName}</b>
                <StatusBadge s={s} />
              </span>
              <span className="mono text-[12.5px] [overflow-wrap:anywhere]">{s.email}</span>
              <span className="k text-ink-2!">{t(m.phoneSeedLine, { auth, folders: foldersLine(s, t) })}</span>
              <span className="k">{problem ? `${last} · ${problem}` : used}</span>
              {actions(s)}
            </div>
          )
        })}
      </div>
      <div className="pl-foot">
        <span />
        <span className="k">{t(m.writeOnly)}</span>
      </div>
    </section>
  )
}

function HowSeedsConnect() {
  const t = useT()
  const { slug } = useWorkspace()
  const items: [string, ReactNode][] = [
    [
      t(m.howBigFour),
      <T
        key="a"
        msg={m.howBigFourBody}
        values={{
          link: (c) => (
            <Link to="/w/$workspace/admin/authentication" params={{ workspace: slug }} className="underline">
              {c}
            </Link>
          ),
        }}
      />,
    ],
    [t(m.howOthers), t(m.howOthersBody)],
    [t(m.howGmail), t(m.howGmailBody)],
    [t(m.howFocused), t(m.howFocusedBody)],
  ]
  return (
    <section className="card min-w-0 flex-[1.3_1_520px]" id="how-seeds-connect">
      <div className="sec">
        <h2 className="h2">{t(m.howSeedsTitle)}</h2>
      </div>
      <div className="pad flex flex-col gap-3">
        <div className="note i" role="note">
          <Icon name="info" size={16} className="mt-px flex-none" />
          <span>{t(m.howSeedsNote)}</span>
        </div>
        <div className="flex flex-col">
          {items.map(([title, body], i) => (
            <div key={title} className="flex flex-col gap-1 py-2.5 text-[13px]" style={{ borderTop: i ? '1px solid var(--line-2)' : 0 }}>
              <b className="font-semibold">{title}</b>
              <span className="text-ink-nav">{body}</span>
            </div>
          ))}
        </div>
      </div>
    </section>
  )
}

function DeletionCard({ d }: { d: SeedList }) {
  const t = useT()
  const { slug } = useWorkspace()
  const refresh = useRefreshPlacement()
  const [draft, setDraft] = useState<SeedList['deleteAfter']>()
  const value = draft ?? d.deleteAfter
  const setValue = setDraft
  const [busy, setBusy] = useState(false)
  const save = async () => {
    setBusy(true)
    const res = await http.PUT(`${base}/settings`, { params: { path: { workspace: slug } }, body: { deleteAfter: value } })
    setBusy(false)
    if (!res.data) {
      toast({ kind: 'error', title: errorMessage(res.error), retry: () => void save() })
      return
    }
    await refresh()
    setDraft(undefined)
    toast({ kind: 'success', title: t(m.saved) })
  }
  const opts: ['24h' | 'finished' | 'never', string, string][] = [
    ['24h', t(m.delete24h), t(m.delete24hBody)],
    ['finished', t(m.deleteFinished), t(m.deleteFinishedBody)],
    ['never', t(m.deleteNever), t(m.deleteNeverBody)],
  ]
  return (
    <section className="card min-w-0 flex-[1_1_400px]" id="deletion">
      <div className="sec">
        <div className="flex min-w-0 flex-col gap-0.5">
          <h2 className="h2">{t(m.deleteTitle)}</h2>
          <span className="k">{t(m.deleteLead)}</span>
        </div>
      </div>
      <div className="pad flex flex-col gap-2" role="radiogroup" aria-label={t(m.deleteTitle)}>
        {opts.map(([v, title, body]) => (
          <div key={v} className={value === v ? 'pl-opt on' : 'pl-opt'}>
            <input id={`pl-delete-${v}`} type="radio" name="pl-delete" value={v} checked={value === v} disabled={!d.canManage} onChange={() => setValue(v)} />
            <label htmlFor={`pl-delete-${v}`} className="ot cursor-pointer">
              <b className="font-semibold">{title}</b>
              <span className="k text-ink-2!">{body}</span>
            </label>
          </div>
        ))}
      </div>
      <div className="pl-foot">
        <span className="k flex-[1_1_240px]">{t(m.deleteFoot)}</span>
        {d.canManage ? (
          <Button variant="primary" onClick={() => void save()} loading={busy}>
            {t(m.save)}
          </Button>
        ) : null}
      </div>
    </section>
  )
}

function SeedsLoading() {
  const t = useT()
  return (
    <Loading label={t(m.loadingSeeds)} className="card flex flex-col gap-3 p-[18px]">
      <span className="k" aria-hidden="true">
        {t(m.loadingSeeds)}
      </span>
      {[0, 1, 2, 3, 4].map((i) => (
        <div key={i} className="flex items-center gap-3.5 border-t border-line-2 py-2.5">
          <Skeleton w={130} h={12} />
          <Skeleton w={90} h={12} />
          <Skeleton w={70} h={20} r={999} />
        </div>
      ))}
    </Loading>
  )
}
