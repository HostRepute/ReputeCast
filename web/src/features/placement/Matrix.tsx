import { Link } from '@tanstack/react-router'
import { useState, type ReactNode } from 'react'
import { useNow } from '@/lib/hooks'
import { useT, T } from '@/i18n'
import { clock, clockSeconds, shortDate } from '@/lib/format'
import { useWorkspace } from '@/lib/session'
import { m } from './messages'
import { arrive, providerColor, resultLine, useCopy, type CopyView, type ProviderResult, type TestDetail } from './shared'

// ── Folder marks ───────────────────────────────────────────────────

function FolderIcon({ kind }: { kind: 'inbox' | 'tab' | 'spam' | 'none' }) {
  const common = { width: 12, height: 12, viewBox: '0 0 16 16', fill: 'none', stroke: 'currentColor', strokeWidth: 1.7, strokeLinecap: 'round' as const, strokeLinejoin: 'round' as const, 'aria-hidden': true }
  switch (kind) {
    case 'inbox':
      return (
        <svg {...common}>
          <path d="M2.5 9.2h3.1l1 1.8h2.8l1-1.8h3.1" />
          <path d="M4.3 3.5h7.4l1.8 5.7v3.3h-11V9.2z" />
        </svg>
      )
    case 'tab':
      return (
        <svg {...common}>
          <path d="M8.6 2.6h4.8v4.8l-6.2 6.2-4.8-4.8z" />
          <circle cx="10.9" cy="5.1" r=".7" />
        </svg>
      )
    case 'spam':
      return (
        <svg {...common}>
          <path d="M8 2.3 14.1 13.2H1.9z" />
          <path d="M8 6.6v3M8 11.3v.1" />
        </svg>
      )
  }
  return (
    <svg {...common}>
      <circle cx="8" cy="8" r="5.6" strokeDasharray="2.1 2.1" />
    </svg>
  )
}

/** Inbox · Primary, Promotions tab, Spam · Junk Email, Not arrived… */
export function FolderBadge({ c }: { c: CopyView }) {
  const t = useT()
  if (c.status === 'found') {
    if (c.folder === 'spam')
      return (
        <span className="badge bad fb">
          <FolderIcon kind="spam" />
          {t(m.cellSpam, { folder: c.folderName ?? '' })}
        </span>
      )
    if (c.folder === 'tab')
      return (
        <span className="badge info fb">
          <FolderIcon kind="tab" />
          {t(m.cellTab, { tab: c.tab || c.folderName || '' })}
        </span>
      )
    // Gmail's Primary, Outlook's Focused and Other; a mailbox's own name for its inbox ("Posteingang") stays Inbox.
    const named = c.tab
    return (
      <span className="badge ok fb">
        <FolderIcon kind="inbox" />
        {named ? t(m.cellInboxTab, { tab: named }) : t(m.cellInbox)}
        {c.late ? ` · ${t(m.cellLate)}` : null}
      </span>
    )
  }
  const word =
    c.status === 'not_arrived'
      ? t(m.cellNotArrived)
      : c.status === 'not_checked'
        ? c.reason === 'unsent'
          ? t(m.cellUnsent)
          : t(m.cellNotChecked)
        : c.status === 'failed'
          ? t(m.cellRefused)
          : c.status === 'queued'
            ? t(m.cellQueued)
            : t(m.cellWaiting)
  return (
    <span className={c.status === 'failed' ? 'badge bad fb' : 'badge mut fb'}>
      <FolderIcon kind="none" />
      {word}
    </span>
  )
}

function Check({ ok, size = 11 }: { ok: boolean; size?: number }) {
  return ok ? (
    <svg width={size} height={size} viewBox="0 0 16 16" fill="none" stroke="#15803D" strokeWidth="2.4" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
      <path d="m3 8 3.5 3.5L13 5" />
    </svg>
  ) : (
    <svg width={size} height={size} viewBox="0 0 16 16" fill="none" stroke="#B91C1C" strokeWidth="2.4" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
      <path d="m4 4 8 8M12 4l-8 8" />
    </svg>
  )
}

/** SPF, DKIM and DMARC as the mailbox's Authentication-Results reported them. */
export function AuthMarks({ spf, dkim, dmarc }: { spf?: string; dkim?: string; dmarc?: string }) {
  const t = useT()
  const one = (name: string, r?: string) =>
    r ? (
      <span className={r === 'pass' ? 'pl-am' : 'pl-am f'} key={name}>
        <Check ok={r === 'pass'} />
        {name}
        <span className="vh"> {r === 'pass' ? t(m.pass) : r}</span>
      </span>
    ) : null
  return (
    <span className="pl-ams">
      {one('SPF', spf)}
      {one('DKIM', dkim)}
      {one('DMARC', dmarc)}
    </span>
  )
}

/** The line under a cell: deferrals, a late arrival, a seed not checked. */
function CellNote({ c, test, provider }: { c: CopyView; test: TestDetail; provider?: ProviderResult }) {
  const t = useT()
  const { slug } = useWorkspace()
  const e = c.lastEvent
  if (c.status === 'found' && c.late) {
    const minutes = c.handedAt && c.receivedAt ? Math.round((new Date(c.receivedAt).getTime() - new Date(c.handedAt).getTime()) / 60000) : 0
    return <span className="ev2">{c.deferrals ? t(m.cellLateNote, { deferrals: c.deferrals, minutes }) : t(m.cellLateNoteNoDeferrals, { minutes })}</span>
  }
  if (c.status === 'not_arrived') {
    if (c.reason === 'stopped' && c.reasonAt) return <span className="ev2">{t(m.cellNotFoundStop, { time: clock(c.reasonAt) })}</span>
    if (c.deferrals && e) return <span className="ev2">{t(m.cellDeferrals, { deferrals: c.deferrals, mailbox: c.seedProviderName, code: e.code ?? '', text: e.diagnostic ?? '' })}</span>
    return null
  }
  if (c.status === 'sent' && e && (e.type === 'deferred' || e.type === 'soft_bounce')) {
    return <span className="ev2">{t(m.cellDeferred, { provider: provider?.short ?? c.providerName, mailbox: c.seedProviderName, code: e.code ?? '' })}</span>
  }
  if (c.status === 'not_checked') {
    if (c.reason === 'unsent') return <span className="ev2">{t(m.cellUnsentNote)}</span>
    if (c.reason === 'seed_removed') return <span className="ev2">{t(m.cellSeedRemoved)}</span>
    if (c.reasonAt) {
      const msg = c.reason === 'signin_expired' ? m.cellSigninExpired : m.cellSigninFailed
      return (
        <span className="ev2">
          {t(msg, { date: shortDate(c.reasonAt), time: clock(c.reasonAt) })}{' '}
          {c.seedId && test.seeds.some((s) => s.id === c.seedId && (s.status === 'failed' || s.status === 'expired')) ? (
            <Link to="/w/$workspace/deliverability/placement/seeds" params={{ workspace: slug }} search={{ reconnect: c.seedId }} className="font-semibold">
              {t(m.reconnect)}
            </Link>
          ) : null}
        </span>
      )
    }
  }
  if (c.status === 'failed' && c.error) return <span className="ev2">{c.error}</span>
  return null
}

function CellBody({ c, test, provider }: { c: CopyView; test: TestDetail; provider?: ProviderResult }) {
  return (
    <div className="pl-cl">
      <div className="t1">
        <FolderBadge c={c} />
        <span className="pl-tm">{c.status === 'found' ? arrive(c.arriveMs) : '–'}</span>
      </div>
      {c.status === 'found' ? <AuthMarks spf={c.spf} dkim={c.dkim} dmarc={c.dmarc} /> : null}
      <CellNote c={c} test={test} provider={provider} />
    </div>
  )
}

const tone = (c: CopyView | undefined) => (!c ? '' : c.status === 'found' && c.folder === 'spam' ? 'cs' : c.status === 'not_arrived' || c.status === 'not_checked' ? 'cn' : '')

// ── The matrix ────────────────────────────────────────────────────

type Group = { provider: string; name: string; seeds: { key: string; email: string; seedId?: string }[] }

/** Seeds grouped by mailbox provider, in the test's order. */
function groups(test: TestDetail): Group[] {
  const out: Group[] = []
  const seen = new Set<string>()
  const add = (provider: string, name: string, email: string, seedId?: string) => {
    const key = seedId ?? email
    if (seen.has(key)) return
    seen.add(key)
    let g = out.find((x) => x.provider === provider)
    if (!g) {
      g = { provider, name, seeds: [] }
      out.push(g)
    }
    g.seeds.push({ key, email, seedId })
  }
  for (const s of [...test.seeds].sort((a, b) => a.position - b.position)) add(s.provider, s.providerName, s.email, s.id)
  for (const c of test.copies) add(c.seedProvider, c.seedProviderName, c.seedEmail, c.seedId)
  return out
}

function copyFor(test: TestDetail, seed: { email: string; seedId?: string }, providerId: string): CopyView | undefined {
  return test.copies.find((c) => (seed.seedId ? c.seedId === seed.seedId : c.seedEmail === seed.email) && c.providerId === providerId)
}

export function Matrix({ test, selected, onSelect, live }: { test: TestDetail; selected?: string; onSelect: (id: string) => void; live: boolean }) {
  const t = useT()
  const gs = groups(test)
  const cell = (seed: Group['seeds'][number], p: ProviderResult) => {
    const c = copyFor(test, seed, p.id)
    if (!c) return <td className="cell" key={p.id} />
    const sel = c.id === selected
    return (
      <td key={p.id} className={['cell', tone(c), sel ? 'sel' : ''].filter(Boolean).join(' ')}>
        <button type="button" className="pl-cellbtn" aria-pressed={sel} onClick={() => onSelect(c.id)}>
          <span className="vh">{t(m.cellContext, { seed: seed.email, provider: p.name })} </span>
          <CellBody c={c} test={test} provider={p} />
        </button>
      </td>
    )
  }
  return (
    <>
      <div className="pl-no-phone">
        <div className="relative overflow-x-auto">
          <table className="pl-mx" aria-label={t(m.matrixLabel)}>
            <thead>
              <tr>
                <th scope="col">{t(m.colMailboxProvider)}</th>
                <th scope="col">{t(m.colSeed)}</th>
                {test.providers.map((p) => (
                  <th scope="col" key={p.id}>
                    <div className="flex flex-col gap-[3px]">
                      <span className="pl-pv">
                        <span className="pl-sw" style={{ background: providerColor(p.adapter) }} />
                        {p.name}
                      </span>
                      <span className="k whitespace-normal font-medium normal-case tracking-normal">
                        {live ? t(m.inboxSoFar, { found: p.counts.found, copies: p.counts.copies }) : resultLine(p.counts)}
                      </span>
                    </div>
                  </th>
                ))}
              </tr>
            </thead>
            <tbody>
              {gs.map((g) =>
                g.seeds.map((s, i) => (
                  <tr key={s.key}>
                    {i === 0 ? (
                      <th scope="row" className="mp" rowSpan={g.seeds.length}>
                        {g.name}
                      </th>
                    ) : null}
                    <td className="sd">{s.email}</td>
                    {test.providers.map((p) => cell(s, p))}
                  </tr>
                )),
              )}
            </tbody>
          </table>
        </div>
      </div>
      <div className="pl-only-phone">
        {gs.flatMap((g) =>
          g.seeds.map((s) => (
            <div className="pl-seedc" key={s.key}>
              <div className="flex flex-col gap-px">
                <span className="font-semibold">{g.name}</span>
                <span className="mono k">{s.email}</span>
              </div>
              {test.providers.map((p) => {
                const c = copyFor(test, s, p.id)
                if (!c) return null
                const sel = c.id === selected
                return (
                  <button
                    type="button"
                    key={p.id}
                    className={['pr', tone(c), sel ? 'sel' : ''].filter(Boolean).join(' ')}
                    aria-pressed={sel}
                    onClick={() => onSelect(c.id)}
                  >
                    <span className="vh">{t(m.cellContext, { seed: s.email, provider: p.name })} </span>
                    <span className="pl-pv">
                      <span className="pl-sw" style={{ background: providerColor(p.adapter) }} />
                      {p.short}
                    </span>
                    <CellBody c={c} test={test} provider={p} />
                  </button>
                )
              })}
            </div>
          )),
        )}
      </div>
    </>
  )
}

// ── Copy detail ────────────────────────────────────────────────────

/** Header lines with their field names marked, as the mailbox sent them. */
function HeaderText({ text }: { text: string }) {
  const lines = text.replace(/\r\n/g, '\n').split('\n')
  const out: ReactNode[] = []
  lines.forEach((ln, i) => {
    const mm = /^([A-Za-z0-9-]+:)(.*)$/.exec(ln)
    if (mm && !/^\s/.test(ln)) {
      out.push(
        <span key={i}>
          <span className="c1">{mm[1]}</span>
          {mm[2]}
          {'\n'}
        </span>,
      )
    } else out.push(<span key={i}>{`${ln}\n`}</span>)
  })
  return <pre className="pl-hdr">{out}</pre>
}

type Ev = { at: string; title: string; sub?: ReactNode }

export function CopyDetailPanel({ test, copyId, onClose }: { test: TestDetail; copyId: string; onClose: () => void }) {
  const t = useT()
  const q = useCopy(test.id, copyId)
  const now = useNow(30_000)
  const [all, setAll] = useState(false)
  const base = test.copies.find((c) => c.id === copyId)
  const d = q.data
  const c = d?.copy ?? base
  if (!c) return null
  const provider = test.providers.find((p) => p.id === c.providerId)
  const pname = provider?.name ?? c.providerName
  const events: Ev[] = []
  if (c.handedAt) {
    events.push({
      at: c.handedAt,
      title: t(m.evHanded, { provider: pname }),
      sub: test.through === 'compare' ? (c.first ? t(m.evFirst, { provider: pname }) : t(m.evSecond, { round: c.round, gap: test.gapMinutes })) : undefined,
    })
  }
  let deferred = 0
  for (const e of d?.events ?? []) {
    const sub = e.code || e.diagnostic ? <span className="mono">{[e.code, e.diagnostic].filter(Boolean).join(' ')}</span> : undefined
    switch (e.type) {
      case 'delivered':
        events.push({ at: e.at, title: t(m.evAccepted, { mailbox: c.seedProviderName }), sub })
        break
      case 'deferred':
      case 'soft_bounce':
        events.push({ at: e.at, title: deferred++ ? t(m.evDeferredAgain) : t(m.evDeferred, { host: e.host || c.seedProviderName }), sub })
        break
      case 'bounce':
      case 'hard_bounce':
        events.push({ at: e.at, title: t(m.evBounced, { host: e.host || c.seedProviderName }), sub })
        break
      case 'complaint':
        events.push({ at: e.at, title: t(m.evComplaint, { mailbox: c.seedProviderName }), sub })
        break
      case 'rejected':
      case 'failed':
        events.push({ at: e.at, title: t(m.evRejected, { provider: pname }), sub })
        break
      default:
        events.push({ at: e.at, title: e.type, sub })
    }
  }
  if (c.status === 'found' && c.receivedAt) {
    events.push({ at: c.receivedAt, title: t(m.evFound, { folder: c.folderName || c.folder || '' }), sub: t(m.evFoundAfter, { time: arrive(c.arriveMs) }) })
  }
  if (c.status === 'not_arrived' && c.reason === 'stopped' && c.reasonAt) {
    events.push({ at: c.reasonAt, title: t(m.evStopped), sub: t(m.evStoppedNote, { name: test.stoppedBy ?? '', time: clock(c.reasonAt) }) })
  } else if (c.status === 'not_arrived' && test.waitUntil) {
    const until = c.lateUntil ?? test.lateUntil
    const still = until && new Date(until).getTime() > now.getTime()
    events.push({
      at: test.waitUntil,
      title: t(m.evNotArrived),
      sub: until ? (still ? t(m.evNotArrivedNote, { time: clock(until) }) : t(m.evNotArrivedDone, { time: clock(until) })) : undefined,
    })
  }
  events.sort((a, b) => new Date(a.at).getTime() - new Date(b.at).getTime())

  const authRow = (name: string, a?: { result: string; domain?: string; selector?: string; ip?: string; aligned?: boolean }) =>
    a ? (
      <div className="grid items-baseline gap-2 text-[13px]" style={{ gridTemplateColumns: '70px minmax(0,1fr)' }} key={name}>
        <span className="inline-flex items-center gap-[5px] font-semibold">
          <Check ok={a.result === 'pass'} size={13} />
          {name}
        </span>
        <span>
          <span className={a.result === 'pass' ? 'font-semibold text-ok-fg' : 'font-semibold text-bad-fg'}>{a.result === 'pass' ? t(m.pass) : a.result}</span>{' '}
          <span className="k text-ink-nav!">
            {[
              a.domain,
              name === 'SPF' ? a.ip : undefined,
              name === 'DKIM' && a.selector ? t(m.selector, { selector: a.selector }) : undefined,
              name === 'DMARC' && a.aligned ? t(m.aligned) : undefined,
            ]
              .filter(Boolean)
              .join(' · ')}
          </span>
        </span>
      </div>
    ) : null

  const found = c.status === 'found'
  return (
    <section className="card pl-copyd" aria-label={t(m.copyDetailLabel)}>
      <div className="dh2">
        <div className="flex min-w-0 flex-1 flex-col gap-1.5">
          <span className="k">{t(m.copyDetailTitle, { provider: pname, mailbox: c.seedProviderName })}</span>
          <h2 className="mono m-0 text-[17px] font-bold tracking-[-0.01em] [overflow-wrap:anywhere]">{c.seedEmail}</h2>
          <div className="flex flex-wrap items-center gap-2.5">
            <FolderBadge c={c} />
            <span className="k">{found ? t(m.toArrive, { time: arrive(c.arriveMs) }) : c.status === 'not_arrived' ? t(m.notFoundAfter) : null}</span>
          </div>
        </div>
        <button type="button" className="ibtn flex-none" style={{ width: 32, height: 32 }} aria-label={t(m.closeCopy)} onClick={onClose}>
          <svg width="14" height="14" viewBox="0 0 16 16" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
            <path d="m4 4 8 8M12 4l-8 8" />
          </svg>
        </button>
      </div>
      <div className="db2">
        {found ? (
          <div className="flex min-w-0 flex-col gap-2">
            <h3>{t(m.whereLanded)}</h3>
            <div className="row gap-1.5! text-[13px]">
              <span className="mono text-[12.5px]">{c.seedEmail}</span>
              <span className="k">›</span>
              <span className="font-semibold">{c.folderName || c.folder}</span>
            </div>
            <span className="k">
              {d?.readVia === 'graph' ? t(m.readGraph, { mailbox: c.seedProviderName }) : d?.readVia === 'gmail' ? t(m.readGmail, { mailbox: c.seedProviderName }) : t(m.readImap, { mailbox: c.seedProviderName })}
            </span>
          </div>
        ) : c.status === 'not_arrived' || c.status === 'sent' ? (
          <div className="flex min-w-0 flex-col gap-2">
            <h3>{t(m.whatHappened)}</h3>
            <span className="text-[13px] text-ink-nav">
              {c.deferrals ? t(m.whatHappenedDeferred, { provider: pname, mailbox: c.seedProviderName, count: c.deferrals }) : c.status === 'sent' ? t(m.noHeaders) : t(m.whatHappenedMissing)}
            </span>
          </div>
        ) : null}
        {found && d ? (
          <div className="flex min-w-0 flex-col gap-2">
            <h3>{t(m.authTitle)}</h3>
            <div className="flex flex-col gap-1.5">
              {authRow('SPF', d.spf)}
              {authRow('DKIM', d.dkim)}
              {authRow('DMARC', d.dmarc)}
            </div>
          </div>
        ) : null}
        {found && d && (d.keyHeaders || d.headers) ? (
          <div className="flex min-w-0 flex-col gap-2">
            <div className="flex items-center justify-between gap-2">
              <h3>{t(m.headersTitle)}</h3>
              {d.headers && d.keyHeaders && d.headers !== d.keyHeaders ? (
                <button type="button" className="pl-lnk cursor-pointer border-0 bg-transparent p-0 text-accent text-[12px]!" aria-pressed={all} onClick={() => setAll(!all)}>
                  {all ? t(m.showKeyHeaders) : t(m.showAllHeaders)}
                </button>
              ) : null}
            </div>
            <HeaderText text={all ? d.headers : d.keyHeaders || d.headers} />
          </div>
        ) : null}
        {events.length ? (
          <div className="flex min-w-0 flex-col gap-2">
            <h3>{t(m.eventsTitle, { provider: pname })}</h3>
            <div className="flex flex-col">
              {events.map((e, i) => (
                <div className="pl-vline items-start" key={i}>
                  <span className="mono k w-16 flex-none pt-px">{clockSeconds(e.at)}</span>
                  <span className="flex min-w-0 flex-col gap-px">
                    <span className="font-semibold">{e.title}</span>
                    {e.sub ? <span className="k text-ink-2!">{e.sub}</span> : null}
                  </span>
                </div>
              ))}
            </div>
            {d?.providerMessageId ? (
              <span className="k">
                <T msg={m.providerMessageId} values={{ provider: pname, id: d.providerMessageId, mono: (ch) => <span className="mono">{ch}</span> }} />
              </span>
            ) : null}
          </div>
        ) : null}
      </div>
    </section>
  )
}
