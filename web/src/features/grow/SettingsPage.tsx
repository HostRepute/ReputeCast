import { Link } from '@tanstack/react-router'
import { useMemo, useState, type ReactNode } from 'react'
import { $api, ApiError, errorMessage, http, type Schemas } from '@/api/client'
import { Badge } from '@/components/ui/Badge'
import { Button } from '@/components/ui/Button'
import { CardSection } from '@/components/ui/Card'
import { ConfirmDialog, UnsavedChangesGuard } from '@/components/ui/ConfirmDialog'
import { ErrorSummary, type FormError } from '@/components/ui/ErrorSummary'
import { Loading, Skeleton } from '@/components/ui/Feedback'
import { Field, Input, Select, Textarea } from '@/components/ui/Field'
import { Note } from '@/components/ui/Note'
import { toast } from '@/components/ui/Toast'
import { T, useT } from '@/i18n'
import { dayMonthDate, weekdayDate } from '@/lib/format'
import { useCan, useWorkspace } from '@/lib/session'
import { OffBanner, useProgramSwitch, type Program } from './ProgramHeader'
import { PreviewButton } from './Previews'
import { m } from './messages'
import { mr } from './referrals.messages'
import { ms } from './settings.messages'
import { GROW, GrowPage, LoadError, ProgramNav, ReadOnlyNote, mono, orNames, useGrowRefresh, useReferralPrograms } from './shared'

type Counting = Schemas['ReferralCounting']
type Pages = Schemas['ReferralPageSettings']

interface Draft {
  counting: Counting
  pages: Pages
  periods: ('month' | 'all')[]
  rows: 5 | 10 | 25
  senderId: string
  replyTo: string
}

function draftOf(p: Program): Draft {
  return {
    counting: structuredClone(p.counting),
    pages: structuredClone(p.pages),
    periods: [...p.leaderboard.periods],
    rows: p.leaderboard.rows,
    senderId: p.emails.senderId,
    replyTo: p.emails.replyTo,
  }
}

const BUTTONS: Pages['shareButtons'] = ['copy', 'email', 'x', 'linkedin', 'whatsapp']

/**
 * Referrals › Settings (ReferralSettings, ReferralStates): the program, the
 * counting rules with what a change does to referrals in hold, links and
 * pages, the leaderboard, milestone emails' sender and the danger zone.
 */
export function SettingsPage({ search }: { search: { program?: string } }) {
  const t = useT()
  const { slug } = useWorkspace()
  const { list, program, q: pq } = useReferralPrograms(search.program)
  const q = $api.useQuery('get', `${GROW}/referrals/{program}`, { params: { path: { workspace: slug, program: program?.id ?? '' } } }, { enabled: Boolean(program) })
  const p = q.data
  const err = pq.error ?? q.error
  return (
    <GrowPage tab="referrals" documentTitle={t(m.docReferrals)}>
      <div className="gw">
        {!list || (program && !p) ? (
          <>
            <ProgramNav current="settings" programId={search.program} />
            {err ? (
              <LoadError error={err} title={t(ms.loadError)} onRetry={() => void (pq.isError ? pq.refetch() : q.refetch())} />
            ) : (
              <Loading label={t(ms.loadingLabel)}>
                <Skeleton w="100%" h={520} r={10} />
              </Loading>
            )}
          </>
        ) : !program || !p ? (
          <ProgramNav current="settings" programId={search.program} />
        ) : (
          <Settings key={p.updatedAt} p={p} programId={search.program} />
        )}
      </div>
    </GrowPage>
  )
}

function Settings({ p, programId }: { p: Program; programId?: string }) {
  const t = useT()
  const { slug } = useWorkspace()
  const canEdit = useCan('grow:edit')
  const refresh = useGrowRefresh()
  const saved = useMemo(() => draftOf(p), [p])
  const [d, setD] = useState<Draft>(saved)
  const [errors, setErrors] = useState<FormError[]>([])
  const [saving, setSaving] = useState(false)
  const [conflict, setConflict] = useState(false)
  const dirty = JSON.stringify(d) !== JSON.stringify(saved)
  const holdChanged = d.counting.holdDays !== saved.counting.holdDays
  const impact = $api.useQuery(
    'post',
    `${GROW}/referrals/{program}/settings/impact`,
    { params: { path: { workspace: slug, program: p.id } }, body: d.counting },
    { enabled: holdChanged, placeholderData: (prev) => prev },
  )
  const on = p.status === 'on'
  const disabled = !canEdit
  const setCounting = (c: Partial<Counting>) => setD((x) => ({ ...x, counting: { ...x.counting, ...c } }))
  const setPages = (c: Partial<Pages>) => setD((x) => ({ ...x, pages: { ...x.pages, ...c } }))

  const validate = (): FormError[] => {
    const out: FormError[] = []
    if (d.counting.burst.on && !(d.counting.burst.threshold >= 2 && d.counting.burst.threshold <= 100)) out.push({ field: 'rs-burst', message: t(ms.errBurst) })
    if (d.counting.sharedIp.on && !(d.counting.sharedIp.threshold >= 2 && d.counting.sharedIp.threshold <= 1000)) out.push({ field: 'rs-shared', message: t(ms.errShared) })
    if (!d.pages.shareMessage.includes('{link}')) out.push({ field: 'rs-share', message: t(ms.errShareLink) })
    return out
  }
  const save = async () => {
    const bad = validate()
    setErrors(bad)
    if (bad.length) return
    setSaving(true)
    try {
      await http.PUT('/api/v1/workspaces/{workspace}/grow/referrals/{program}/settings', {
        params: { path: { workspace: slug, program: p.id } },
        body: { baseUpdatedAt: p.updatedAt, counting: d.counting, pages: d.pages, leaderboardPeriods: d.periods, leaderboardRows: d.rows, replyTo: d.replyTo, senderId: d.senderId || undefined },
      })
      toast({
        kind: 'success',
        title: t(ms.saved),
        detail: holdChanged ? t(ms.savedHold, { days: d.counting.holdDays }) : t(ms.savedDetail),
      })
      await refresh()
    } catch (e) {
      if (e instanceof ApiError && e.status === 409) setConflict(true)
      else if (e instanceof ApiError && e.problem.errors?.length) setErrors(e.problem.errors.map((x) => ({ field: 'rs-share', message: x.message ?? '' })))
      else toast({ kind: 'error', title: errorMessage(e) })
    } finally {
      setSaving(false)
    }
  }
  const err = (field: string) => errors.find((x) => x.field === field)?.message
  return (
    <>
      <div className="flex flex-wrap items-center justify-between gap-x-4 gap-y-2.5">
        <ProgramNav current="settings" held={p.held} programId={programId} />
        <span className="flex items-center gap-2">
          <Badge tone={on ? 'ok' : 'neutral'} dot>
            {on ? t(m.on) : t(m.off)}
          </Badge>
          <span className="k">{on && p.startedAt ? t(ms.statusLine, { list: p.listName, date: dayMonthDate(p.startedAt) }) : t(ms.statusLineOff, { list: p.listName })}</span>
        </span>
      </div>
      <OffBanner p={p} settings={false} />
      {disabled ? (
        <ReadOnlyNote>
          <T msg={p.helpers.edit.length ? ms.readOnlyText : ms.readOnlyTextNoHelpers} values={{ helpers: orNames(p.helpers.edit), mono }} />
        </ReadOnlyNote>
      ) : null}
      {conflict ? (
        <Note tone="warn">
          <span className="flex flex-wrap items-center gap-2.5">
            {t(ms.conflict)}
            <Button size="sm" onClick={() => void refresh().then(() => setConflict(false))}>
              {t(ms.reload)}
            </Button>
          </span>
        </Note>
      ) : null}
      <ErrorSummary errors={errors} title={t(ms.problems, { count: errors.length })} />
      <div className="gw-g2">
        <div className="flex min-w-0 flex-col gap-4">
          <ProgramCard p={p} />
          <section className="card">
            <CardSection title={t(ms.countingTitle)} caption={t(ms.countingCaption)} />
            <Row title={t(ms.confirmedOnly)} text={t(ms.confirmedOnlyText)}>
              <Badge tone="neutral">{t(ms.alwaysOn)}</Badge>
            </Row>
            <Row title={t(ms.hold)} text={t(ms.holdText)}>
              <Select aria-label={t(ms.hold)} style={{ width: 140 }} value={String(d.counting.holdDays)} disabled={disabled} onChange={(e) => setCounting({ holdDays: Number(e.target.value) as Counting['holdDays'] })}>
                {[0, 3, 7, 14, 30].map((n) => (
                  <option key={n} value={n}>
                    {t(ms.holdDays, { count: n })}
                  </option>
                ))}
              </Select>
            </Row>
            {holdChanged ? (
              <div className="px-[18px] pb-3.5">
                <Note tone="info">
                  <span className="flex flex-col gap-1">
                    <span className="font-bold">{t(ms.impactTitle, { from: saved.counting.holdDays, to: d.counting.holdDays })}</span>
                    <span>
                      {!impact.data || impact.data.inHold === 0
                        ? t(ms.impactNone)
                        : d.counting.holdDays > saved.counting.holdDays
                          ? t(ms.impactLater, { count: impact.data.inHold, days: d.counting.holdDays - saved.counting.holdDays })
                          : t(ms.impactSooner, { count: impact.data.inHold, days: saved.counting.holdDays - d.counting.holdDays })}
                    </span>
                  </span>
                </Note>
              </div>
            ) : null}
            <Head title={t(ms.excludedTitle)} caption={t(ms.excludedCaption)} />
            <Row title={t(ms.self)} text={t(ms.selfText)}>
              <Badge tone="neutral">{t(ms.alwaysOn)}</Badge>
            </Row>
            <Row title={t(ms.already)} text={t(ms.alreadyText, { list: p.listName })}>
              <Badge tone="neutral">{t(ms.alwaysOn)}</Badge>
            </Row>
            <Row title={t(ms.never)} text={t(ms.neverText)}>
              <Badge tone="neutral">{t(ms.alwaysOn)}</Badge>
            </Row>
            <Row title={t(ms.sameIp)} text={t(ms.sameIpText)}>
              <Switch label={t(ms.sameIp)} checked={d.counting.sameIp} disabled={disabled} onChange={(v) => setCounting({ sameIp: v })} />
            </Row>
            <Row title={t(ms.disposable)} text={t(ms.disposableText)}>
              <Switch label={t(ms.disposable)} checked={d.counting.disposable} disabled={disabled} onChange={(v) => setCounting({ disposable: v })} />
            </Row>
            <Row title={t(ms.left)} text={t(ms.leftText)}>
              <Switch label={t(ms.left)} checked={d.counting.left} disabled={disabled} onChange={(v) => setCounting({ left: v })} />
            </Row>
            <Head title={t(ms.heldTitle)} caption={t(ms.heldCaption)} />
            <Row
              title={t(ms.burst)}
              text={
                <>
                  {t(ms.burstBefore)}{' '}
                  <Input
                    id="rs-burst"
                    className="gw-inl gw-num"
                    type="number"
                    min={2}
                    max={100}
                    aria-label={t(ms.burstThreshold)}
                    value={String(d.counting.burst.threshold)}
                    disabled={disabled || !d.counting.burst.on}
                    invalid={Boolean(err('rs-burst'))}
                    onChange={(e) => setCounting({ burst: { ...d.counting.burst, threshold: Number(e.target.value) } })}
                  />{' '}
                  {t(ms.burstAfter)}
                  {err('rs-burst') ? <span className="err mt-1 block">{err('rs-burst')}</span> : null}
                </>
              }
            >
              <Switch label={t(ms.burst)} checked={d.counting.burst.on} disabled={disabled} onChange={(v) => setCounting({ burst: { ...d.counting.burst, on: v } })} />
            </Row>
            <Row
              title={t(ms.shared)}
              text={
                <>
                  <Input
                    id="rs-shared"
                    className="gw-inl gw-num"
                    type="number"
                    min={2}
                    max={1000}
                    aria-label={t(ms.sharedThreshold)}
                    value={String(d.counting.sharedIp.threshold)}
                    disabled={disabled || !d.counting.sharedIp.on}
                    invalid={Boolean(err('rs-shared'))}
                    onChange={(e) => setCounting({ sharedIp: { ...d.counting.sharedIp, threshold: Number(e.target.value) } })}
                  />{' '}
                  {t(ms.sharedAfter)}
                  {err('rs-shared') ? <span className="err mt-1 block">{err('rs-shared')}</span> : null}
                </>
              }
            >
              <Switch label={t(ms.shared)} checked={d.counting.sharedIp.on} disabled={disabled} onChange={(v) => setCounting({ sharedIp: { ...d.counting.sharedIp, on: v } })} />
            </Row>
            <Row title={t(ms.numbered)} text={t(ms.numberedText)}>
              <Switch label={t(ms.numbered)} checked={d.counting.numbered} disabled={disabled} onChange={(v) => setCounting({ numbered: v })} />
            </Row>
          </section>
        </div>
        <div className="flex min-w-0 flex-col gap-4">
          <section className="card">
            <CardSection title={t(ms.pagesTitle)} caption={t(ms.pagesCaption)} />
            <div className="gw-pad flex flex-col gap-3.5">
              <div className="flex flex-col gap-1">
                <span className="lbl">{t(ms.linkFormat)}</span>
                <span className="mono">{p.linkExample}</span>
                <span className="k">
                  <T msg={ms.linkFormatText} values={{ code: p.linkExample.split('/').pop() ?? '', mono }} />
                </span>
              </div>
              <Field label={t(ms.headline)} hint={<T msg={ms.headlineHint} values={{ list: p.listName, mono }} />}>
                <Input value={d.pages.inviteHeadline} maxLength={200} disabled={disabled} onChange={(e) => setPages({ inviteHeadline: e.target.value })} />
              </Field>
              <Field label={t(ms.inviteText)}>
                <Textarea rows={2} value={d.pages.inviteText} maxLength={500} disabled={disabled} onChange={(e) => setPages({ inviteText: e.target.value })} />
              </Field>
              <Field label={t(ms.hubIntro)}>
                <Textarea rows={2} value={d.pages.hubIntro} maxLength={500} disabled={disabled} onChange={(e) => setPages({ hubIntro: e.target.value })} />
              </Field>
              <Field label={t(ms.shareMessage)} hint={<T msg={ms.shareMessageHint} values={{ mono }} />} error={err('rs-share')}>
                <Textarea id="rs-share" rows={2} value={d.pages.shareMessage} maxLength={500} disabled={disabled} onChange={(e) => setPages({ shareMessage: e.target.value })} />
              </Field>
              <fieldset className="m-0 flex flex-col gap-1.5 border-0 p-0">
                <legend className="lbl mb-1">{t(ms.shareButtons)}</legend>
                <div className="flex flex-wrap gap-x-4 gap-y-2">
                  {BUTTONS.map((b) => (
                    <label key={b} className="inline-flex items-center gap-1.5 text-[13px]">
                      <input
                        type="checkbox"
                        checked={d.pages.shareButtons.includes(b)}
                        disabled={disabled || b === 'copy'}
                        onChange={(e) => setPages({ shareButtons: e.target.checked ? BUTTONS.filter((x) => x === b || d.pages.shareButtons.includes(x)) : d.pages.shareButtons.filter((x) => x !== b) })}
                      />
                      {buttonName(t, b)}
                    </label>
                  ))}
                </div>
                <span className="k">{t(ms.shareNote)}</span>
              </fieldset>
            </div>
            <div className="gw-foot">
              <span className="k">
                <Link to="/w/$workspace/forms/branding" params={{ workspace: slug }}>
                  {t(ms.branding)}
                </Link>
              </span>
              <span className="flex flex-wrap gap-2">
                <PreviewButton page="invite" label={t(ms.previewInvite)} />
                <PreviewButton page="hub" label={t(ms.previewHub)} />
              </span>
            </div>
          </section>
          <BoardCard p={p} d={d} setD={setD} disabled={disabled} />
          <section className="card">
            <CardSection title={t(ms.emailsTitle)} caption={t(ms.emailsCaption, { count: p.emails.milestones })} />
            <Row title={t(ms.sender)} text={null}>
              <Select aria-label={t(ms.sender)} style={{ width: 300 }} value={d.senderId} disabled={disabled} onChange={(e) => setD((x) => ({ ...x, senderId: e.target.value }))}>
                <option value="">{t(ms.senderDefault)}</option>
                {p.senders.map((s) => (
                  <option key={s.id} value={s.id} disabled={!s.verified}>
                    {s.verified ? t(ms.senderOption, { name: s.name, email: s.email }) : t(ms.senderUnverified, { name: s.name, email: s.email })}
                  </option>
                ))}
              </Select>
            </Row>
            <Row title={t(ms.replyTo)} text={t(ms.replyToHint)}>
              <Input type="email" aria-label={t(ms.replyTo)} style={{ width: 300 }} value={d.replyTo} disabled={disabled} onChange={(e) => setD((x) => ({ ...x, replyTo: e.target.value }))} />
            </Row>
            <Row title={t(ms.stream)} text={t(ms.streamText)}>
              <Badge tone="neutral">{t(ms.streamValue)}</Badge>
            </Row>
            <div className="gw-foot">
              <span className="k">{t(ms.editEmails)}</span>
              <Link to="/w/$workspace/grow/referrals/milestones" params={{ workspace: slug }} search={programId ? { program: programId } : {}} className="gw-lnk">
                {t(ms.editEmailsLink)}
              </Link>
            </div>
          </section>
          {canEdit ? <DangerZone p={p} /> : null}
        </div>
      </div>
      {canEdit && dirty ? (
        <div className="gw-dbar" role="region" aria-label={t(ms.unsaved)}>
          <div className="flex flex-wrap items-center justify-between gap-3">
            <span className="flex flex-col gap-0.5">
              <span className="font-semibold">{t(ms.unsaved)}</span>
              <span className="k">
                {holdChanged
                  ? t(ms.unsavedHold, { from: saved.counting.holdDays, to: d.counting.holdDays, count: impact.data?.inHold ?? p.inHold })
                  : t(ms.unsavedSome, { count: changes(saved, d) })}
              </span>
            </span>
            <span className="flex gap-2">
              <Button
                onClick={() => {
                  setD(saved)
                  setErrors([])
                }}
              >
                {t(ms.discard)}
              </Button>
              <Button variant="primary" loading={saving} busyLabel={t(ms.saving)} onClick={() => void save()}>
                {t(ms.saveSettings)}
              </Button>
            </span>
          </div>
        </div>
      ) : null}
      <UnsavedChangesGuard dirty={dirty && !saving} description={t(ms.leaveText)} />
    </>
  )
}

/** How many top-level settings differ. */
function changes(a: Draft, b: Draft): number {
  let n = 0
  for (const k of Object.keys(a.counting) as (keyof Counting)[]) if (JSON.stringify(a.counting[k]) !== JSON.stringify(b.counting[k])) n++
  for (const k of Object.keys(a.pages) as (keyof Pages)[]) if (JSON.stringify(a.pages[k]) !== JSON.stringify(b.pages[k])) n++
  if (JSON.stringify(a.periods) !== JSON.stringify(b.periods)) n++
  if (a.rows !== b.rows) n++
  if (a.senderId !== b.senderId) n++
  if (a.replyTo !== b.replyTo) n++
  return Math.max(1, n)
}

function buttonName(t: ReturnType<typeof useT>, b: Pages['shareButtons'][number]): string {
  switch (b) {
    case 'copy':
      return t(ms.btnCopy)
    case 'email':
      return t(ms.btnEmail)
    case 'x':
      return t(ms.btnX)
    case 'linkedin':
      return t(ms.btnLinkedin)
    case 'whatsapp':
      return t(ms.btnWhatsapp)
  }
  return b
}

function Row({ title, text, children }: { title: string; text: ReactNode; children: ReactNode }) {
  return (
    <div className="gw-set">
      <div className="tx">
        <span className="font-semibold">{title}</span>
        {text ? <span className="k">{text}</span> : null}
      </div>
      <div className="ct">{children}</div>
    </div>
  )
}

function Head({ title, caption }: { title: string; caption: string }) {
  return (
    <div className="gw-sethd border-t border-line-2">
      <span className="gw-hd3">{title}</span>
      <span className="k">{caption}</span>
    </div>
  )
}

function Switch({ label, checked, disabled, onChange }: { label: string; checked: boolean; disabled?: boolean; onChange: (v: boolean) => void }) {
  return <input type="checkbox" className="tg" role="switch" aria-label={label} checked={checked} disabled={disabled} onChange={(e) => onChange(e.target.checked)} />
}

function ProgramCard({ p }: { p: Program }) {
  const t = useT()
  const { slug } = useWorkspace()
  const canEdit = useCan('grow:edit')
  const canAudit = useCan('audit:view')
  const sw = useProgramSwitch(p)
  const on = p.status === 'on'
  const statusLine = on
    ? p.startedAt
      ? p.startedBy
        ? t(ms.statusOn, { date: weekdayDate(p.startedAt), by: p.startedBy })
        : t(ms.statusOnNoBy, { date: weekdayDate(p.startedAt) })
      : t(ms.statusNever)
    : p.statusAt
      ? p.statusBy
        ? t(ms.statusOff, { date: dayMonthDate(p.statusAt), by: p.statusBy })
        : t(ms.statusOffNoBy, { date: dayMonthDate(p.statusAt) })
      : t(ms.statusNever)
  const started = p.startedAt ? (p.startedBy ? t(ms.startedValue, { date: weekdayDate(p.startedAt), by: p.startedBy }) : t(ms.startedValueNoBy, { date: weekdayDate(p.startedAt) })) : ''
  return (
    <section className="card">
      <CardSection title={t(ms.programTitle)} caption={t(ms.programCaption)} />
      <Row title={t(ms.list)} text={t(ms.listFixed)}>
        <span className="font-semibold">{p.listName}</span>
        <span className="k">{t(ms.listValue, { count: p.subscribers })}</span>
      </Row>
      <Row title={t(ms.status)} text={statusLine}>
        <Badge tone={on ? 'ok' : 'neutral'} dot>
          {on ? t(m.on) : t(m.off)}
        </Badge>
        {canEdit ? <Switch label={t(ms.switchLabel)} checked={on} onChange={(v) => sw.ask(v ? 'on' : 'off')} /> : null}
      </Row>
      {started ? (
        <Row title={t(ms.started)} text={p.announcedIn ? t(ms.startedAnnounced, { line: started, campaign: p.announcedIn }) : started}>
          {canAudit ? (
            <Link to="/w/$workspace/settings/audit" params={{ workspace: slug }} search={{ category: 'grow' }} className="gw-lnk">
              {t(ms.history)}
            </Link>
          ) : null}
        </Row>
      ) : null}
      {canEdit ? sw.dialog : null}
    </section>
  )
}

function BoardCard({ p, d, setD, disabled }: { p: Program; d: Draft; setD: (f: (x: Draft) => Draft) => void; disabled: boolean }) {
  const t = useT()
  const { slug } = useWorkspace()
  const refresh = useGrowRefresh()
  const lb = p.leaderboard
  const [busy, setBusy] = useState(false)
  const flip = async (on: boolean) => {
    setBusy(true)
    try {
      await http.PUT('/api/v1/workspaces/{workspace}/grow/referrals/{program}/leaderboard', { params: { path: { workspace: slug, program: p.id } }, body: { on } })
      toast({ kind: 'success', title: on ? t(ms.boardTurnedOn) : t(ms.boardTurnedOff), detail: on ? t(ms.boardTurnedOnDetail, { url: lb.url.replace(/^https?:\/\//, '') }) : undefined })
      await refresh()
    } catch (e) {
      toast({ kind: 'error', title: errorMessage(e) })
    } finally {
      setBusy(false)
    }
  }
  const togglePeriod = (k: 'month' | 'all', v: boolean) =>
    setD((x) => {
      const next = v ? (['month', 'all'] as const).filter((y) => y === k || x.periods.includes(y)) : x.periods.filter((y) => y !== k)
      return { ...x, periods: next.length ? [...next] : x.periods }
    })
  return (
    <section className="card">
      <CardSection title={t(ms.boardTitle)} caption={t(ms.boardCaption)} />
      <Row title={t(ms.board)} text={lb.on ? (lb.at ? (lb.by ? t(ms.boardOn, { date: dayMonthDate(lb.at), by: lb.by }) : t(ms.boardOnNoBy, { date: dayMonthDate(lb.at) })) : t(ms.boardOff)) : t(ms.boardOff)}>
        <input type="checkbox" className="tg" role="switch" aria-label={t(ms.boardSwitch)} checked={lb.on} disabled={disabled || busy} onChange={(e) => void flip(e.target.checked)} />
      </Row>
      <Row title={t(ms.periods)} text={null}>
        {(['month', 'all'] as const).map((k) => (
          <label key={k} className="inline-flex items-center gap-1.5 text-[13px]">
            <input type="checkbox" checked={d.periods.includes(k)} disabled={disabled} onChange={(e) => togglePeriod(k, e.target.checked)} />
            {k === 'month' ? t(ms.periodMonth) : t(ms.periodAll)}
          </label>
        ))}
      </Row>
      <Row title={t(ms.rows)} text={null}>
        <div className="seg" role="radiogroup" aria-label={t(ms.rows)}>
          {([5, 10, 25] as const).map((n) => (
            <button key={n} type="button" role="radio" aria-checked={d.rows === n} className={d.rows === n ? 'on' : undefined} disabled={disabled} onClick={() => setD((x) => ({ ...x, rows: n }))}>
              {t(ms.rowsN, { count: n })}
            </button>
          ))}
        </div>
      </Row>
      <div className="gw-foot">
        <span className="k">{t(ms.boardFoot)}</span>
        <PreviewButton page="leaderboard" label={t(ms.previewBoard)} />
      </div>
    </section>
  )
}

function DangerZone({ p }: { p: Program }) {
  const t = useT()
  const { slug } = useWorkspace()
  const sw = useProgramSwitch(p)
  const refresh = useGrowRefresh()
  const [deleting, setDeleting] = useState(false)
  const on = p.status === 'on'
  return (
    <section className="card" style={{ borderColor: 'var(--bad-line)' }}>
      <CardSection title={t(ms.dangerTitle)} caption={t(ms.dangerCaption)} />
      <Row title={on ? t(ms.offTitle) : t(ms.onTitle)} text={on ? t(ms.offText) : t(ms.onText)}>
        <Button variant={on ? 'danger' : 'primary'} onClick={() => sw.ask(on ? 'off' : 'on')}>
          {on ? t(mr.turnOff) : t(mr.turnOn)}
        </Button>
      </Row>
      <Row title={t(ms.deleteTitle)} text={on ? t(ms.deleteText, { count: p.links }) : t(ms.deleteTextOff, { count: p.links })}>
        <Button variant="danger" disabledReason={on ? t(ms.deleteNeedsOff) : undefined} onClick={() => setDeleting(true)}>
          {t(ms.deleteAction)}
        </Button>
      </Row>
      {sw.dialog}
      <ConfirmDialog
        open={deleting}
        onOpenChange={setDeleting}
        badge={t(ms.deleteAction)}
        title={t(ms.deleteConfirmTitle, { list: p.listName })}
        description={t(ms.deleteConfirmText, { count: p.links })}
        confirmLabel={t(ms.deleteAction)}
        typedConfirmation={{ label: t(ms.deleteConfirmLabel, { list: p.listName }), expected: p.listName }}
        onConfirm={async () => {
          await http.DELETE('/api/v1/workspaces/{workspace}/grow/referrals/{program}', { params: { path: { workspace: slug, program: p.id } } })
          toast({ title: t(ms.deleted) })
          await refresh()
        }}
      />
    </section>
  )
}

