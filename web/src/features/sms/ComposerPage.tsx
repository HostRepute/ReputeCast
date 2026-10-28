import { useQueryClient } from '@tanstack/react-query'
import { Link } from '@tanstack/react-router'
import { useEffect, useMemo, useRef, useState, type ReactNode } from 'react'
import { $api, ApiError, errorMessage, type Schemas } from '@/api/client'
import { Icon } from '@/components/icons/Icon'
import { problemReference } from '@/components/shell/SystemScreens'
import { Avatar } from '@/components/ui/Avatar'
import { Badge } from '@/components/ui/Badge'
import { Button, ButtonLink } from '@/components/ui/Button'
import { Dialog } from '@/components/ui/Dialog'
import { ErrorPanel, Loading } from '@/components/ui/Feedback'
import { Select } from '@/components/ui/Field'
import { Note } from '@/components/ui/Note'
import { toast } from '@/components/ui/Toast'
import { channelIcon } from '@/features/channels/channel'
import { formatList, T, useT, type MessageDescriptor } from '@/i18n'
import { useDocumentTitle } from '@/lib/hooks'
import { clock, count, dayMonthDate, dayMonthYear, monthName, money } from '@/lib/format'
import { useWorkspace } from '@/lib/session'
import { BudgetMeter, CountryName, countryName, holdWord, price, regionLabel, regionSentence, throwIfForbidden } from './common'
import { m } from './composer.messages'
import { m as sm } from './messages'
import { fixEncoding, inGSM7, type OutsideKind } from './segments'
import { findLinks, measure, renderWith } from './measure'
import { EditorTools } from './EditorTools'
import './composer.css'
import './sms.css'

type Campaign = Schemas['SmsCampaign']
type CampaignKey = ['get', '/api/v1/workspaces/{workspace}/sms-campaigns/{campaignId}', { params: { path: { workspace: string; campaignId: string } } }]

const kindWord: Record<OutsideKind, MessageDescriptor> = {
  curly_apostrophe: m.kCurly_apostrophe,
  curly_quote: m.kCurly_quote,
  en_dash: m.kEn_dash,
  em_dash: m.kEm_dash,
  minus: m.kMinus,
  ellipsis: m.kEllipsis,
  non_breaking_space: m.kNon_breaking_space,
  thin_space: m.kThin_space,
  zero_width_space: m.kZero_width_space,
  prime: m.kPrime,
  backtick: m.kBacktick,
  emoji: m.kEmoji,
  other: m.kOther,
}

const statusWord: Partial<Record<Campaign['status'], MessageDescriptor>> = {
  approval: m.statusApproval,
  approved: m.statusApproved,
  scheduled: m.statusScheduled,
  sending: m.statusSending,
  paused: m.statusPaused,
  sent: m.statusSent,
  cancelled: m.statusCancelled,
  failed: m.statusFailed,
}

/** The SMS composer: full screen, outside the app shell (SmsComposer). */
export function SmsComposerPage({ campaignId }: { campaignId: string }) {
  const t = useT()
  const { slug } = useWorkspace()
  const q = $api.useQuery('get', '/api/v1/workspaces/{workspace}/sms-campaigns/{campaignId}', { params: { path: { workspace: slug, campaignId } } })
  throwIfForbidden(q.error)
  if (q.data) return <Composer key={q.data.id} data={q.data} />
  return (
    <div className="smc">
      <main className="flex flex-1 items-start justify-center p-10" id="main">
        {q.isError ? (
          <ErrorPanel
            className="max-w-[560px]"
            title={t(m.errorTitle)}
            reference={problemReference(q.error)}
            actions={
              <>
                <Button onClick={() => void q.refetch()}>{t(m.retry)}</Button>
                <ButtonLink to="/w/$workspace/campaigns" params={{ workspace: slug }}>
                  {t(m.back)}
                </ButtonLink>
              </>
            }
          >
            {errorMessage(q.error)}
          </ErrorPanel>
        ) : (
          <Loading label={t(m.loading)}>
            <span className="k">{t(m.loading)}</span>
          </Loading>
        )}
      </main>
    </div>
  )
}

function Composer({ data }: { data: Campaign }) {
  const t = useT()
  const { slug, me } = useWorkspace()
  const qc = useQueryClient()
  const key: CampaignKey = ['get', '/api/v1/workspaces/{workspace}/sms-campaigns/{campaignId}', { params: { path: { workspace: slug, campaignId: data.id } } }]
  const editable = data.access.canEdit && data.status === 'draft'
  const [body, setBody] = useState(data.body)
  const [saving, setSaving] = useState<'idle' | 'saving' | 'error'>('idle')
  const [saveError, setSaveError] = useState<string>()
  const [replaced, setReplaced] = useState<{ before: string; chars: string; with: string; segments: number; was: number } | null>(null)
  const [audienceOpen, setAudienceOpen] = useState(false)
  const [historyOpen, setHistoryOpen] = useState(false)
  const version = useRef(data.version)
  const [lastSaved, setLastSaved] = useState(data.body)
  useDocumentTitle(`${data.name} · ${me.workspace.name} · ReputeCast`)
  const update = $api.useMutation('patch', '/api/v1/workspaces/{workspace}/sms-campaigns/{campaignId}')

  // Autosave: a moment after the last keystroke.
  useEffect(() => {
    if (!editable || body === lastSaved) return
    const id = window.setTimeout(() => {
      setSaving('saving')
      update.mutate(
        { params: { path: { workspace: slug, campaignId: data.id } }, body: { body, version: version.current } },
        {
          onSuccess: (next) => {
            version.current = next.version
            setLastSaved(body)
            qc.setQueryData(key, next)
            setSaving('idle')
          },
          onError: (e) => {
            if (e instanceof ApiError && e.status === 409) {
              toast({ kind: 'info', title: t(m.conflict) })
              void qc.invalidateQueries({ queryKey: key })
              return
            }
            setSaving('error')
            setSaveError(errorMessage(e))
          },
        },
      )
    }, 900)
    return () => window.clearTimeout(id)
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [body, editable, lastSaved])

  const live = useMemo(() => measure(data.prefix, body, data.optOut, data.shortDomain), [data.prefix, body, data.optOut, data.shortDomain])
  const savedNow = body === lastSaved
  const status =
    data.status === 'draft'
      ? saving === 'saving'
        ? t(m.saving)
        : data.autosavedAt
          ? t(m.draftSaved, { time: clock(data.autosavedAt), version: data.version })
          : t(m.draftNew)
      : t(statusWord[data.status] ?? m.draftNew)
  const audienceName = data.audience.map((a) => a.name).join(' + ')
  const replace = () => {
    const fixed = fixEncoding(body)
    const after = measure(data.prefix, fixed, data.optOut, data.shortDomain).count.segments
    const chars = live.count.outside.filter((o) => o.replacement !== '' || o.removable)
    setReplaced({ before: body, chars: chars.map((c) => c.char).join(' '), with: chars.map((c) => c.replacement || '∅').join(' '), segments: after, was: live.count.segments })
    setBody(fixed)
  }
  return (
    <div className="smc">
      <a href="#main" className="skip">
        {t(m.skip)}
      </a>
      <header className="smc-top">
        <ButtonLink iconOnly to="/w/$workspace/campaigns" params={{ workspace: slug }} aria-label={t(m.back)}>
          <Icon name="chevronLeft" size={16} />
        </ButtonLink>
        <div className="flex min-w-0 flex-col">
          <span className="k text-[11px]">
            <Link to="/w/$workspace/campaigns" params={{ workspace: slug }} className="text-ink-3">
              {t(m.campaigns)}
            </Link>{' '}
            › {t(m.kind)}
          </span>
          <h1 className="m-0 text-[15px] font-bold leading-7">{data.name}</h1>
        </div>
        <Badge tone="neutral" className="gap-[5px]">
          {channelIcon.sms}
          {t(m.sms)}
        </Badge>
        <Badge tone={saving === 'error' ? 'bad' : 'neutral'} title={saving === 'error' ? t(m.saveFailed, { error: saveError ?? '' }) : undefined}>
          {saving === 'error' ? t(m.saveFailed, { error: '' }).replace(/:\s*$/, '') : status}
        </Badge>
        {!editable && !data.access.canEdit ? <Badge tone="neutral">{t(m.readOnly)}</Badge> : null}
        <div className="ml-2 flex flex-wrap gap-2">
          <button type="button" className="btn" onClick={() => setAudienceOpen(true)} disabled={!editable}>
            <Icon name="user" size={14} />
            {data.audience.length ? (
              <T msg={m.audienceBtn} values={{ name: audienceName, count: count(data.funnel.canReceive), b: (c) => <b className="n">{c}</b> }} />
            ) : (
              t(m.audienceNone)
            )}
          </button>
        </div>
        <div className="ml-auto flex flex-wrap items-center gap-2">
          {data.status === 'sent' || data.status === 'sending' || data.status === 'paused' ? (
            <ButtonLink variant="primary" to="/w/$workspace/campaigns/$campaignId/sms-report" params={{ workspace: slug, campaignId: data.id }}>
              {t(m.report)}
            </ButtonLink>
          ) : (
            <>
              <SendTestButton data={data} disabled={!savedNow} />
              <ButtonLink variant="primary" to="/w/$workspace/campaigns/$campaignId/sms-review" params={{ workspace: slug, campaignId: data.id }}>
                {data.access.canSend ? t(m.review) : t(m.reviewApproval)}
              </ButtonLink>
            </>
          )}
        </div>
      </header>
      <RouteBar data={data} />
      <div className="smc-cols">
        <aside className="smc-panel l" aria-label={t(m.text)}>
          <div className="smc-sec">
            <div className="smc-h">
              <h2>{t(m.text)}</h2>
              {editable ? <EditorTools body={body} setBody={setBody} /> : null}
            </div>
            <Editor data={data} body={body} setBody={setBody} editable={editable} />
            <span className="k">
              {data.optOut ? (
                <T
                  msg={m.lockedNote}
                  values={{
                    prefix: data.prefix,
                    optOut: data.optOut,
                    mono: (c) => <span className="mono">{c}</span>,
                    link: (c) => (
                      <Link to="/w/$workspace/providers/sms/rules" params={{ workspace: slug }}>
                        {c}
                      </Link>
                    ),
                  }}
                />
              ) : (
                <T
                  msg={m.lockedNotePrefix}
                  values={{
                    prefix: data.prefix,
                    mono: (c) => <span className="mono">{c}</span>,
                    link: (c) => (
                      <Link to="/w/$workspace/providers/sms/rules" params={{ workspace: slug }}>
                        {c}
                      </Link>
                    ),
                  }}
                />
              )}
            </span>
            {!data.access.canEdit ? (
              <Note tone="neutral">{t(m.readOnlyText, { helpers: data.access.helpers.length ? formatList(data.access.helpers, 'disjunction') : 'none' })}</Note>
            ) : null}
          </div>
          <LengthSection data={data} live={live} editable={editable} onReplace={replace} replaced={replaced} onUndo={() => {
            if (replaced) setBody(replaced.before)
            setReplaced(null)
          }} onDismiss={() => setReplaced(null)} />
          <LinksSection data={data} body={body} setBody={setBody} editable={editable} />
          <PersonalizationSection data={data} live={live} />
          <TestsSection data={data} disabled={!savedNow} />
        </aside>
        <Preview data={data} body={body} />
        <aside className="smc-panel r" aria-label={t(m.costTitle)}>
          <div className="smc-rcols">
            <CostSection data={data} />
            <BudgetSection data={data} />
            <WhoSection data={data} />
          </div>
        </aside>
      </div>
      <footer className="smc-foot">
        <span className="k">
          {t(m.footerFacts, {
            encoding: live.count.encoding === 'gsm7' ? 'GSM-7' : 'UCS-2',
            outside: live.count.outside.reduce((n, o) => n + o.count, 0),
            links: live.links.length,
            fields: live.fields.length,
          })}
        </span>
        <span className="k ml-auto">{data.retentionDays ? t(m.keptDays, { days: data.retentionDays }) : t(m.keptNone)}</span>
        {data.blocks ? (
          <Badge tone="bad" dot>
            {t(m.blocks, { count: data.blocks })}
          </Badge>
        ) : data.warnings ? (
          <Badge tone="warn" dot>
            {t(m.warnings, { count: data.warnings })}
          </Badge>
        ) : (
          <Badge tone="ok" dot>
            {t(m.noWarnings)}
          </Badge>
        )}
        <button type="button" className="sms-lbtn k" onClick={() => setHistoryOpen(true)}>
          {t(m.history, { count: data.versions })}
        </button>
      </footer>
      {audienceOpen ? <AudienceDialog data={data} onClose={() => setAudienceOpen(false)} onSaved={(next) => {
        version.current = next.version
        qc.setQueryData(key, next)
      }} /> : null}
      {historyOpen ? <HistoryDialog data={data} editable={editable} onClose={() => setHistoryOpen(false)} onRestore={(b) => setBody(b)} /> : null}
    </div>
  )
}

/** Route: US and Canada via Twilio +1 888 555 0142 · Ireland skipped (pending) · 20. */
function RouteBar({ data }: { data: Campaign }) {
  const t = useT()
  const { slug } = useWorkspace()
  const order = [...new Set(data.routes.map((r) => r.provider))]
  const color = (p: string) => ['var(--chart-gmail)', 'var(--chart-outlook)', 'var(--chart-yahoo)'][order.indexOf(p)] ?? 'var(--chart-other)'
  return (
    <div className="smc-rbar" aria-label={t(m.route)}>
      <span className="k">{t(m.route)}</span>
      {data.routes.map((r) =>
        r.skipped ? (
          <span key={r.countries.join()} className="sms-pill" style={{ background: '#FFFBEB', borderColor: '#FCD34D', color: '#92400E' }}>
            {t(m.routeSkipped, { region: regionLabel(r.countries), reason: holdWord(t, r.skipped), count: count(r.recipients) })}
          </span>
        ) : (
          <span key={r.countries.join()} className="sms-pill">
            <span className="dot" style={{ color: color(r.provider) }} aria-hidden="true" />
            {t(m.routeVia, { region: regionSentence(r.countries), provider: r.provider })}
            <span className={r.sender.startsWith('+') ? 'mono font-medium' : undefined}>{r.sender}</span>
          </span>
        ),
      )}
      <Link to="/w/$workspace/providers/sms/rules" params={{ workspace: slug }} className="sms-lnk ml-auto text-[12.5px]">
        {t(m.routesAndRules)}
      </Link>
    </div>
  )
}

/** The text: the locked prefix, the body over its highlighted mirror, the opt-out text. */
function Editor({ data, body, setBody, editable }: { data: Campaign; body: string; setBody: (b: string) => void; editable: boolean }) {
  const t = useT()
  const area = useRef<HTMLTextAreaElement>(null)
  const parts = useMemo(() => highlight(body), [body])
  return (
    <div className={`smc-ed${editable ? '' : ' ro'}`}>
      {data.prefix ? (
        <span className="fx" title={t(m.fromRules)}>
          {data.prefix}
        </span>
      ) : null}
      <div className="smc-wrap">
        <div className="smc-mirror" aria-hidden="true">
          {parts.map((p, i) =>
            p.kind === 'text' ? (
              <span key={i}>{p.text}</span>
            ) : (
              <span key={i} className={p.kind === 'link' ? 'lt' : p.kind === 'shortener' ? 'lt e' : p.kind === 'field' ? 'tk' : 'uc'}>
                {p.text}
              </span>
            ),
          )}
          {/* A trailing line break needs a character to keep its height. */}
          {'​'}
        </div>
        <textarea
          ref={area}
          aria-label={t(m.editorLabel)}
          value={body}
          readOnly={!editable}
          spellCheck
          onChange={(e) => setBody(e.target.value)}
        />
      </div>
      {data.optOut ? (
        <span className="fx" title={t(m.fromRules)}>
          {data.optOut}
        </span>
      ) : null}
    </div>
  )
}

type Part = { kind: 'text' | 'link' | 'shortener' | 'field' | 'outside'; text: string }

/** Splits the body into plain text, links, fields and characters outside GSM-7. */
function highlight(body: string): Part[] {
  const marks: { start: number; end: number; kind: Part['kind'] }[] = []
  for (const l of findLinks(body)) marks.push({ start: l.start, end: l.end, kind: l.shortener ? 'shortener' : 'link' })
  for (const mt of body.matchAll(/\{\{.*?\}\}/g)) marks.push({ start: mt.index ?? 0, end: (mt.index ?? 0) + mt[0].length, kind: 'field' })
  let i = 0
  for (const ch of Array.from(body)) {
    if (!inGSM7(ch)) marks.push({ start: i, end: i + ch.length, kind: 'outside' })
    i += ch.length
  }
  marks.sort((a, b) => a.start - b.start)
  const out: Part[] = []
  let pos = 0
  for (const mk of marks) {
    if (mk.start < pos) continue
    if (mk.start > pos) out.push({ kind: 'text', text: body.slice(pos, mk.start) })
    out.push({ kind: mk.kind, text: body.slice(mk.start, mk.end) })
    pos = mk.end
  }
  if (pos < body.length) out.push({ kind: 'text', text: body.slice(pos) })
  return out
}

function Section({ title, aside, children }: { title: ReactNode; aside?: ReactNode; children: ReactNode }) {
  return (
    <div className="smc-sec">
      <div className="smc-h">
        <h2>{title}</h2>
        {aside}
      </div>
      {children}
    </div>
  )
}

/** Length: the counter, characters outside GSM-7 and their fix, over 3 segments. */
function LengthSection({
  data,
  live,
  editable,
  onReplace,
  replaced,
  onUndo,
  onDismiss,
}: {
  data: Campaign
  live: ReturnType<typeof measure>
  editable: boolean
  onReplace: () => void
  replaced: { before: string; chars: string; with: string; segments: number; was: number } | null
  onUndo: () => void
  onDismiss: () => void
}) {
  const t = useT()
  const c = live.count
  const segments = Math.max(c.segments, 1)
  const fieldsUsed = live.fields.length > 0
  const spread = data.counter.spread
  const minSeg = spread.length ? Math.min(...spread.map((s) => s.segments)) : segments
  const maxSeg = spread.length ? Math.max(...spread.map((s) => s.segments)) : segments
  const outside = c.outside
  const gsm = measure(data.prefix, fixEncoding(data.body), data.optOut, data.shortDomain).count.segments
  return (
    <Section title={t(m.length)}>
      <div className="sms-segc">
        <div className="sms-segc-h">
          <span>
            {fieldsUsed && minSeg !== maxSeg ? (
              <T
                msg={m.counterHeadFields}
                values={{ characters: live.base, field: t(m.fieldFirstName).toLowerCase(), min: minSeg, max: maxSeg, b: (x) => <b className="n">{x}</b> }}
              />
            ) : (
              <T msg={m.counterHead} values={{ characters: c.characters, segments, b: (x) => <b className="n">{x}</b> }} />
            )}
          </span>
          <Badge tone={c.encoding === 'gsm7' ? 'neutral' : 'warn'}>{c.encoding === 'gsm7' ? 'GSM-7' : 'UCS-2'}</Badge>
        </div>
        <div
          className={`sms-segc-b${c.encoding === 'ucs2' || segments > 3 ? ' w' : ''}`}
          role="img"
          aria-label={t(m.counterAria, { characters: c.characters, segments, encoding: c.encoding === 'gsm7' ? 'GSM-7' : 'UCS-2' })}
        >
          {(c.fill.length ? c.fill : [0]).map((f, i) => (
            <span key={i}>
              <i style={{ width: `${Math.min(100, (f / c.capacity) * 100).toFixed(1)}%` }} />
            </span>
          ))}
        </div>
        <div className="sms-segc-x" aria-hidden="true">
          {(c.fill.length ? c.fill : [0]).map((_, i) => (
            <span key={i}>{segments === 1 ? t(m.segUpTo, { n: i + 1, capacity: c.capacity }) : t(m.segPart, { n: i + 1, capacity: c.capacity })}</span>
          ))}
        </div>
        <span className="k">
          {c.encoding === 'ucs2' ? t(m.ucs2Note) : segments === 1 ? t(m.leftSingle, { left: c.left, part: 153 }) : t(m.leftParts, { left: c.left })}
        </span>
      </div>
      {outside.length ? (
        <Note tone="warn" role="note">
          <span className="flex flex-col gap-1.5">
            <span className="font-semibold">{outside.length === 1 ? t(m.outsideOne) : t(m.outsideMany, { count: outside.length })}</span>
            {outside.map((o) => (
              <span key={o.char}>
                <T
                  msg={o.replacement || o.removable ? m.outsideWhy : m.outsideWhyNoFix}
                  values={{ char: o.char, kind: t(kindWord[o.kind]), segments, gsm, b: (x) => <b className="sms-uc">{x}</b> }}
                />
              </span>
            ))}
            {editable && outside.some((o) => o.replacement || o.removable) ? (
              <span className="flex flex-wrap gap-2">
                <Button size="sm" variant="primary" onClick={onReplace}>
                  {outside.length === 1 ? (outside[0]!.removable ? t(m.removeIt) : t(m.replaceWith, { with: outside[0]!.replacement })) : t(m.replaceAll)}
                </Button>
              </span>
            ) : null}
          </span>
        </Note>
      ) : null}
      {replaced ? (
        <div className="flex flex-wrap items-center gap-2 rounded-lg border border-line bg-[#f9fafb] px-3 py-2 text-[13px]" role="status">
          <Icon name="check" size={14} />
          <span className="font-semibold">{t(m.replaced, { char: replaced.chars, with: replaced.with })}</span>
          <span className="k">· {t(m.replacedSegments, { segments: replaced.segments, before: replaced.was })}</span>
          <button type="button" className="sms-lbtn ml-auto text-[13px]" onClick={onUndo}>
            {t(m.undo)}
          </button>
          <button type="button" className="btn sm ic border-transparent" aria-label={t(m.dismiss)} onClick={onDismiss}>
            <Icon name="close" size={12} />
          </button>
        </div>
      ) : outside.length ? (
        <span className="k">{t(m.pastedNote)}</span>
      ) : null}
      {segments > 3 ? (
        <Note tone="warn" role="note">
          <span className="flex flex-col gap-1">
            <span className="font-semibold">{t(m.overTitle)}</span>
            <span>{t(m.overText, { times: segments, cost: money(data.cost), single: money(data.segments ? data.cost / Math.max(1, data.segments / Math.max(1, data.texts)) : data.cost) })}</span>
          </span>
        </Note>
      ) : null}
    </Section>
  )
}

/** Links: each one shortened on the workspace's domain; public shorteners blocked. */
function LinksSection({ data, body, setBody, editable }: { data: Campaign; body: string; setBody: (b: string) => void; editable: boolean }) {
  const t = useT()
  const links = findLinks(body)
  const short = data.shortDomain
  const utm = data.links[0]?.utmCampaign ?? ''
  return (
    <Section title={t(m.links, { count: links.length })} aside={short && links.length ? <span className="k">{t(m.shortenedOn, { domain: short })}</span> : null}>
      {links.length === 0 ? <span className="k">{t(m.noLinks)}</span> : null}
      {links.map((l) =>
        l.shortener ? (
          <Note key={l.start} tone="bad">
            <span className="flex flex-col gap-1.5">
              <span className="font-semibold">{t(m.shortenerTitle)}</span>
              <span>{t(m.shortenerText, { host: l.host, domain: short || '…' })}</span>
              {editable ? (
                <span>
                  <Button size="sm" onClick={() => setBody(`${body.slice(0, l.start)}${body.slice(l.end)}`.replace(/\s{2,}/g, ' '))}>
                    {t(m.removeLink)}
                  </Button>
                </span>
              ) : null}
            </span>
          </Note>
        ) : (
          <div key={l.start} className="smc-lrow">
            <Icon name="link" size={14} />
            <div className="flex min-w-0 flex-1 flex-col gap-[3px]">
              <span className="mono text-[12.5px] [overflow-wrap:anywhere]">{l.url}</span>
              {short ? (
                <span className="text-[12.5px]">
                  <T msg={m.becomes} values={{ domain: short, length: short.length + 7, mono: (c) => <span className="mono">{c}</span>, b: (c) => <b>{c}</b> }} />
                </span>
              ) : null}
              {utm ? <span className="k">{t(m.linkUtm, { campaign: utm })}</span> : null}
            </div>
          </div>
        ),
      )}
      {links.length && data.shortDomainStatus !== 'verified' && short ? <Note tone="warn">{t(m.domainPending, { domain: short })}</Note> : null}
      <span className="k">{t(m.shortenerNote)}</span>
    </Section>
  )
}

/** Personalization: none, or how length varies with the fields. */
function PersonalizationSection({ data, live }: { data: Campaign; live: ReturnType<typeof measure> }) {
  const t = useT()
  const c = data.counter
  const total = c.spread.reduce((n, s) => n + s.recipients, 0)
  if (!live.fields.length) {
    return (
      <Section title={t(m.personalization)}>
        <span className="k text-[#374151]">{t(m.noFields, { characters: live.count.characters })}</span>
      </Section>
    )
  }
  const field = t(m.fieldFirstName).toLowerCase()
  return (
    <Section title={t(m.fieldsUsed)}>
      {c.spread.length > 1 && total ? (
        <span className="text-[13px]">
          {t(m.spread, {
            field,
            parts: formatList(c.spread.map((s) => t(m.spreadPart, { segments: s.segments, pct: `${Math.round((s.recipients / total) * 100)}%` }))),
          })}
        </span>
      ) : null}
      {c.longest && c.longest.segments > 1 ? (
        <span className="k text-[#374151]">
          {t(m.longest, {
            count: c.spread.filter((s) => s.segments > 1).reduce((n, s) => n + s.recipients, 0),
            field,
            segments: c.longest.segments,
            example: c.longest.value,
            cost: money(data.cost),
            base: money(c.longest.cost),
          })}
        </span>
      ) : null}
    </Section>
  )
}

/** Test texts: each test number with its last test. */
function TestsSection({ data, disabled }: { data: Campaign; disabled: boolean }) {
  const t = useT()
  const tests = data.tests
  return (
    <Section title={t(m.tests)} aside={tests.length ? <SendTestButton data={data} disabled={disabled} small /> : null}>
      {tests.length === 0 ? <span className="k">{t(m.noTests)}</span> : null}
      <div>
        {tests.map((n) => {
          const r = n.lastTest
          return (
            <div key={n.id} className="smc-crow items-center">
              <div className="cl" style={{ flexDirection: 'row', alignItems: 'center', gap: 8 }}>
                <Avatar name={n.name} color={n.color} size={26} fontSize={11} />
                <span className="flex flex-col">
                  <span className="font-semibold">{n.name}</span>
                  <span className="mono k">{n.display}</span>
                </span>
              </div>
              {r ? (
                <div className="cr">
                  <Badge tone={r.status === 'delivered' ? 'ok' : r.status === 'failed' || r.status === 'undelivered' ? 'bad' : 'info'}>
                    {r.status === 'delivered' ? t(m.testDelivered) : r.status === 'failed' || r.status === 'undelivered' ? t(m.testFailed) : t(m.testSending)}
                  </Badge>
                  <span className="k">
                    {t(m.testVia, {
                      time: clock(r.at),
                      provider: r.provider,
                      sender: r.senderId ? r.sender : 'none',
                      seconds: r.seconds ? String(Math.round(r.seconds)) : 'none',
                    })}
                  </span>
                </div>
              ) : null}
            </div>
          )
        })}
      </div>
      <span className="k">{t(m.testsNote)}</span>
    </Section>
  )
}

function SendTestButton({ data, disabled, small }: { data: Campaign; disabled?: boolean; small?: boolean }) {
  const t = useT()
  const { slug } = useWorkspace()
  const qc = useQueryClient()
  const send = $api.useMutation('post', '/api/v1/workspaces/{workspace}/sms-campaigns/{campaignId}/test', {
    onSuccess: (d) => {
      toast({ kind: 'success', title: t(m.testsSent, { count: d.items.length }) })
      void qc.invalidateQueries({ queryKey: ['get', '/api/v1/workspaces/{workspace}/sms-campaigns/{campaignId}'] })
    },
    onError: (e) => toast({ kind: 'error', title: errorMessage(e) }),
  })
  return (
    <Button
      size={small ? 'sm' : undefined}
      disabled={disabled || !data.tests.length}
      loading={send.isPending}
      onClick={() => send.mutate({ params: { path: { workspace: slug, campaignId: data.id } }, body: {} })}
    >
      {small ? t(m.sendTestAll, { count: data.tests.length }) : t(m.sendTest)}
    </Button>
  )
}

/** The phone preview, as a contact in a country sees it. */
function Preview({ data, body }: { data: Campaign; body: string }) {
  const t = useT()
  const { slug } = useWorkspace()
  const contacts = $api.useQuery('get', '/api/v1/workspaces/{workspace}/sms-campaigns/{campaignId}/preview-contacts', {
    params: { path: { workspace: slug, campaignId: data.id } },
  })
  const [contactId, setContactId] = useState<string>()
  const [country, setCountry] = useState<string>()
  const preview = $api.useMutation('post', '/api/v1/workspaces/{workspace}/sms-campaigns/{campaignId}/preview')
  const who = contactId ?? contacts.data?.items[0]?.id
  const contact = contacts.data?.items.find((c) => c.id === who)
  const where = country ?? contact?.country
  useEffect(() => {
    if (!who && !where) return
    preview.mutate({ params: { path: { workspace: slug, campaignId: data.id } }, body: { contactId: who, country: where } })
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [who, where, data.version])
  const p = preview.data
  const countries = [...new Set(data.costs.map((c) => c.country))]
  const empty = data.funnel.canReceive === 0 && data.audience.length > 0
  return (
    <section className="smc-pv" id="main" aria-label={t(m.previewAs)}>
      <div className="smc-pvbar">
        <label>
          {t(m.previewAs)}
          <Select
            value={who ?? ''}
            onChange={(e) => {
              setContactId(e.target.value)
              setCountry(undefined)
            }}
          >
            {(contacts.data?.items ?? []).map((c) => (
              <option key={c.id} value={c.id}>
                {c.name} · {c.country}
              </option>
            ))}
          </Select>
        </label>
        <label>
          {t(m.country)}
          <Select value={where ?? ''} onChange={(e) => setCountry(e.target.value)}>
            {countries.map((c) => (
              <option key={c} value={c}>
                {c} {countryName(c)}
              </option>
            ))}
          </Select>
        </label>
        <span className="k ml-auto">
          {p?.arrives ? t(m.arrives, { date: dayMonthDate(p.arrives), time: data.localTime ?? clock(p.arrives) }) : t(m.arrivesNow, { window: data.window })}
        </span>
      </div>
      <div className="flex flex-1 flex-col items-center gap-3.5 px-5 pb-8 pt-7">
        {empty ? (
          <EmptyAudience data={data} />
        ) : p ? (
          <>
            <Phone p={p} body={body} data={data} />
            {p.skipped ? <Note tone="warn">{t(m.previewSkipped, { country: countryName(p.country), reason: holdWord(t, p.skipped) })}</Note> : null}
            <PreviewCaption data={data} p={p} />
          </>
        ) : (
          <Loading label={t(m.loading)}>
            <span className="k">{t(m.loading)}</span>
          </Loading>
        )}
      </div>
    </section>
  )
}

function Phone({ p, body, data }: { p: Schemas['SmsPreview']; body: string; data: Campaign }) {
  const t = useT()
  // The saved text's parts, or the text as typed until it's saved.
  const typed = body !== data.body
  const typedText = renderWith(body, { first_name: p.contact.split(' ')[0] ?? '' })
  return (
    <div className="sms-phn" role="img" aria-label={t(m.phoneAria, { sender: p.sender })}>
      <div className="sms-phn-h">
        <span className="sms-phn-a">{p.sender.startsWith('+') ? <Icon name="user" size={18} /> : p.sender.slice(0, 2)}</span>
        <span className="sms-phn-n">{p.sender}</span>
        <span className="k">{t(m.textMessage)}</span>
      </div>
      <div className="sms-phn-b">
        {p.confirmation ? (
          <>
            <span className="sms-phn-tm">{`${dayMonthDate(p.confirmation.at)} · ${clock(p.confirmation.at)}`}</span>
            <div className="sms-bub">{p.confirmation.text}</div>
          </>
        ) : null}
        {p.arrives ? <span className="sms-phn-tm">{`${dayMonthDate(p.arrives)} · ${data.localTime ?? clock(p.arrives)}`}</span> : null}
        <div className="sms-bub">
          {typed ? (
            <>
              {data.prefix ? <span className="fx">{data.prefix} </span> : null}
              {typedText}
              {data.optOut ? <span className="fx"> {data.optOut}</span> : null}
            </>
          ) : (
            p.parts.map((part, i) =>
              part.kind === 'fixed' ? (
                <span key={i} className="fx">
                  {i > 0 ? ' ' : ''}
                  {part.text}
                  {i === 0 ? ' ' : ''}
                </span>
              ) : part.kind === 'link' ? (
                <span key={i} className="sl">
                  {part.text}
                </span>
              ) : (
                <span key={i}>{part.text}</span>
              ),
            )
          )}
        </div>
      </div>
      <div className="sms-phn-f">{p.canReply ? <span className="sms-phn-in">{t(m.textMessage)}</span> : <span className="sms-phn-no">{t(m.cantReply)}</span>}</div>
    </div>
  )
}

function PreviewCaption({ data, p }: { data: Campaign; p: Schemas['SmsPreview'] }) {
  const t = useT()
  const numbers = data.routes.filter((r) => !r.skipped && r.sender.startsWith('+'))
  const ids = data.routes.filter((r) => !r.skipped && !r.sender.startsWith('+'))
  if (numbers.length === 1 && ids.length === 1) {
    return (
      <span className="k max-w-[340px] text-center">
        <T
          msg={m.previewNote}
          values={{
            replyRegion: regionSentence(numbers[0]!.countries),
            number: numbers[0]!.sender,
            idRegion: regionSentence(ids[0]!.countries),
            id: ids[0]!.sender,
            mono: (c) => <span className="mono">{c}</span>,
          }}
        />
      </span>
    )
  }
  if (!p.canReply) {
    return (
      <div className="flex max-w-[340px] flex-col items-center gap-2 text-center">
        <CountryName code={p.country} />
        <span className="k">{t(m.ukNote, { region: regionSentence(ids[0]?.countries ?? [p.country]), sender: p.sender, provider: p.provider })}</span>
        <span className="k">
          {t(m.priceIn, { country: countryName(p.country), price: price(p.price) })}
        </span>
      </div>
    )
  }
  return (
    <span className="k max-w-[340px] text-center">
      <T msg={m.previewNoteOne} values={{ sender: p.sender, reply: p.canReply ? 'yes' : 'no', mono: (c) => <span className="mono">{c}</span> }} />
    </span>
  )
}

function EmptyAudience({ data }: { data: Campaign }) {
  const t = useT()
  const { slug } = useWorkspace()
  return (
    <div className="card sms-empty w-full max-w-[520px]">
      <span className="tt">{t(m.emptyTitle, { list: data.audience.map((a) => a.name).join(' + ') })}</span>
      <span className="ds">{t(m.emptyText)}</span>
      <div className="flex flex-wrap justify-center gap-2.5">
        <ButtonLink variant="primary" to="/w/$workspace/forms" params={{ workspace: slug }}>
          {t(m.openForms)}
        </ButtonLink>
      </div>
    </div>
  )
}

/** Cost by country: recipients, sender, segments × price; skipped countries. */
function CostSection({ data }: { data: Campaign }) {
  const t = useT()
  const segments = data.costs.reduce((n, c) => n + c.segments * (c.recipients - c.skipped), 0)
  return (
    <Section title={t(m.costTitle)}>
      <div className="sms-sum k" style={{ marginBottom: -4 }}>
        <span>{t(m.costHead)}</span>
        <span>{t(m.costHeadRight)}</span>
      </div>
      <div>
        {data.costs.map((c) =>
          c.skipped && c.skipped >= c.recipients ? (
            <div key={c.country} className="smc-crow" style={{ background: '#FFFBEB', margin: '0 -16px', padding: '9px 16px' }}>
              <div className="cl">
                <CountryName code={c.country} />
                <span className="k" style={{ color: '#92400E' }}>
                  <T msg={m.skippedLine} values={{ count: count(c.skipped), sender: c.sender, reason: holdWord(t, c.skipReason ?? ''), b: (x) => <b className="n">{x}</b> }} />
                </span>
              </div>
              <div className="cr">
                <span className="n">{money(0)}</span>
                {c.pendingPrice ? (
                  <span className="k" style={{ color: '#92400E' }}>
                    {t(m.skippedPrice, { price: price(c.pendingPrice) })}
                  </span>
                ) : null}
              </div>
            </div>
          ) : (
            <div key={c.country} className="smc-crow">
              <div className="cl">
                <CountryName code={c.country} />
                <span className="k">
                  <b className="n text-ink">{count(c.recipients)}</b> · {c.sender}
                </span>
              </div>
              <div className="cr">
                <span className="n">{money(c.cost)}</span>
                <span className="k">{t(m.costLine, { segments: c.segments, price: price(c.price) })}</span>
              </div>
            </div>
          ),
        )}
      </div>
      <div className="sms-sum border-t border-line pt-2.5 text-[14px]">
        <span className="font-semibold">{t(m.totalLine, { texts: data.texts, segments: segments || data.segments })}</span>
        <span className="n text-[18px]">{money(data.cost)}</span>
      </div>
      {data.pricedOn ? (
        <span className="k">
          {data.costs.some((c) => c.country === 'US' || c.country === 'CA')
            ? t(m.pricedOnUs, { providers: possessive(data.priceProviders, t), date: dayMonthYear(data.pricedOn) })
            : t(m.pricedOn, { providers: possessive(data.priceProviders, t), date: dayMonthYear(data.pricedOn) })}
        </span>
      ) : null}
    </Section>
  )
}

/** "Twilio's and Vonage's". */
function possessive(names: string[], t: ReturnType<typeof useT>): string {
  return formatList(names.map((provider) => t(sm.pricingOf, { provider })))
}

function BudgetSection({ data }: { data: Campaign }) {
  const t = useT()
  const { slug } = useWorkspace()
  const b = data.budget
  const month = monthName(b.month, 'long')
  return (
    <Section
      title={t(m.budget, { month })}
      aside={
        b.budget ? (
          <Link to="/w/$workspace/providers/sms/rules" params={{ workspace: slug }} className="k" hash="budget">
            {t(m.budgetLink, { amount: money(b.budget, Number.isInteger(b.budget) ? 0 : 2) })}
          </Link>
        ) : null
      }
    >
      {b.budget ? (
        <div className="sms-bm">
          <div className="sms-sum">
            <span className="k">{t(m.beforeAfter)}</span>
            <span className="n">
              <T msg={m.beforeAfterValue} values={{ after: money(b.spent), total: money(b.after), budget: money(b.budget), k: (c) => <span className="k">{c}</span> }} />
            </span>
          </div>
          <BudgetMeter spent={b.spent} budget={b.budget} alertPct={b.alertPct} send={data.cost} month={t(m.beforeAfter)} legend />
        </div>
      ) : (
        <span className="k">{t(m.noBudget)}</span>
      )}
    </Section>
  )
}

function WhoSection({ data }: { data: Campaign }) {
  const t = useT()
  const f = data.funnel
  return (
    <Section title={t(m.whoGets)}>
      <div className="sms-sum">
        <span>{data.audience.map((a) => a.name).join(' + ')}</span>
        <span className="n">{count(f.base)}</span>
      </div>
      <div className="sms-sum text-ink-3">
        <span>{t(m.noConsent)}</span>
        <span className="n">−{count(f.noConsent)}</span>
      </div>
      <div className="sms-sum text-ink-3">
        <span>{t(m.suppressed)}</span>
        <span className="n">−{count(f.suppressed)}</span>
      </div>
      <div className="sms-sum border-t border-line pt-2.5 text-[15px]">
        <span className="font-semibold">{t(m.canReceive)}</span>
        <span className="n text-[20px]">{count(f.canReceive)}</span>
      </div>
      <span className="k">
        {t(m.consentNote)}
        {f.skipped ? ` ${t(m.skippedNote, { count: count(f.skipped), total: count(f.canReceive), region: regionSentence(f.skippedIn) })}` : ''}
      </span>
    </Section>
  )
}

/** Who gets this text: lists and segments. */
function AudienceDialog({ data, onClose, onSaved }: { data: Campaign; onClose: () => void; onSaved: (c: Campaign) => void }) {
  const t = useT()
  const { slug } = useWorkspace()
  const lists = $api.useQuery('get', '/api/v1/workspaces/{workspace}/lists', { params: { path: { workspace: slug } } })
  const segments = $api.useQuery('get', '/api/v1/workspaces/{workspace}/segments', { params: { path: { workspace: slug } } })
  const [picked, setPicked] = useState<string[]>(data.audience.map((a) => a.id))
  const update = $api.useMutation('patch', '/api/v1/workspaces/{workspace}/sms-campaigns/{campaignId}', {
    onSuccess: (next) => {
      onSaved(next)
      onClose()
    },
    onError: (e) => toast({ kind: 'error', title: errorMessage(e) }),
  })
  const listIds = (lists.data?.items ?? []).map((l) => l.id)
  const toggle = (id: string, on: boolean) => setPicked(on ? [...picked, id] : picked.filter((x) => x !== id))
  return (
    <Dialog
      open
      onOpenChange={(o) => !o && onClose()}
      title={t(m.audienceTitle)}
      description={t(m.audienceText)}
      footer={
        <>
          <Button onClick={onClose}>{t(m.close)}</Button>
          <Button
            variant="primary"
            loading={update.isPending}
            disabled={!picked.length}
            onClick={() =>
              update.mutate({
                params: { path: { workspace: slug, campaignId: data.id } },
                body: { version: data.version, lists: picked.filter((id) => listIds.includes(id)), segments: picked.filter((id) => !listIds.includes(id)) },
              })
            }
          >
            {t(m.audienceSave)}
          </Button>
        </>
      }
    >
      <div className="flex flex-col gap-3">
        <span className="lbl">{t(m.lists)}</span>
        {(lists.data?.items ?? []).map((l) => (
          <label key={l.id} className="flex items-center gap-2 text-[13px]">
            <input type="checkbox" checked={picked.includes(l.id)} onChange={(e) => toggle(l.id, e.target.checked)} />
            {l.name}
            <span className="k">{count(l.subscribed)}</span>
          </label>
        ))}
        {segments.data?.items.length ? <span className="lbl mt-2">{t(m.segments)}</span> : null}
        {(segments.data?.items ?? []).map((s) => (
          <label key={s.id} className="flex items-center gap-2 text-[13px]">
            <input type="checkbox" checked={picked.includes(s.id)} onChange={(e) => toggle(s.id, e.target.checked)} />
            {s.name}
          </label>
        ))}
      </div>
    </Dialog>
  )
}

function HistoryDialog({ data, editable, onClose, onRestore }: { data: Campaign; editable: boolean; onClose: () => void; onRestore: (body: string) => void }) {
  const t = useT()
  const { slug } = useWorkspace()
  const q = $api.useQuery('get', '/api/v1/workspaces/{workspace}/sms-campaigns/{campaignId}/versions', { params: { path: { workspace: slug, campaignId: data.id } } })
  return (
    <Dialog open onOpenChange={(o) => !o && onClose()} title={t(m.historyTitle)} description={t(m.historyText)} size="lg" footer={<Button onClick={onClose}>{t(m.close)}</Button>}>
      <div className="flex flex-col">
        {(q.data?.items ?? []).map((v) => (
          <div key={v.version} className="smc-crow">
            <div className="cl">
              <span className="font-semibold">
                v{v.version} · {dayMonthDate(v.savedAt)} {clock(v.savedAt)} · {v.savedBy}
              </span>
              <span className="k">{v.body}</span>
            </div>
            {editable && v.version !== data.version ? (
              <div className="cr">
                <Button
                  size="sm"
                  onClick={() => {
                    onRestore(v.body)
                    toast({ kind: 'success', title: t(m.restored, { version: v.version }) })
                    onClose()
                  }}
                >
                  {t(m.restore)}
                </Button>
              </div>
            ) : null}
          </div>
        ))}
      </div>
    </Dialog>
  )
}
