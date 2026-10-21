import { Link, useNavigate } from '@tanstack/react-router'
import { useEffect, useRef, useState, type ReactNode } from 'react'
import { $api, errorMessage, http, type Schemas } from '@/api/client'
import { Icon, LogoMark } from '@/components/icons/Icon'
import { SystemBanners } from '@/components/shell/SystemBanners'
import { Badge } from '@/components/ui/Badge'
import { Button, ButtonLink } from '@/components/ui/Button'
import { KeyValues } from '@/components/ui/Data'
import { ErrorPanel, Skeleton } from '@/components/ui/Feedback'
import { Select } from '@/components/ui/Field'
import { Note } from '@/components/ui/Note'
import { toast } from '@/components/ui/Toast'
import { T, formatList, useT } from '@/i18n'
import { cn } from '@/lib/cn'
import { clock, count, shortDate } from '@/lib/format'
import { useDocumentTitle, useOverflowX } from '@/lib/hooks'
import { useCan, useMe, useWorkspace } from '@/lib/session'
import type { Program } from './ProgramHeader'
import { PreviewButton } from './Previews'
import { RewardEditor } from './RewardEditor'
import { rewardLine } from './MilestonesPage'
import { m } from './messages'
import { mu } from './setup.messages'
import { GROW, fileSize, mono, orNames, useGrowRefresh } from './shared'
import '@/features/connect/connect.css'
import './grow.css'

type Checks = Schemas['ReferralChecks']

export interface SetupSearch {
  program?: string
  step?: number
}

/**
 * Set up a referral program (ReferralSetup): list, milestones and rewards,
 * counting rules, emails and pages, then the checks and Turn on. The step is
 * saved on the program, so a reload (or another day) resumes it.
 */
export function SetupGuide({ search, onSearch }: { search: SetupSearch; onSearch: (s: SetupSearch) => void }) {
  const t = useT()
  const { slug } = useWorkspace()
  const canEdit = useCan('grow:edit')
  const programs = $api.useQuery('get', `${GROW}/referrals`, { params: { path: { workspace: slug } } })
  // The program being set up: the one asked for, else one that was never on.
  const list = programs.data
  const draft = list?.items.find((p) => p.id === search.program) ?? list?.items.find((p) => !p.startedAt && p.status === 'off')
  const pq = $api.useQuery('get', `${GROW}/referrals/{program}`, { params: { path: { workspace: slug, program: draft?.id ?? '' } } }, { enabled: Boolean(draft) })
  const p = pq.data
  const [done, setDone] = useState<Schemas['ReferralTurnedOn']>()
  const saved = p ? Math.min(5, Math.max(1, p.setupStep || 1)) : 1
  const step = done ? 6 : (search.step ?? saved)
  useDocumentTitle(t(mu.docTitle))
  const error = programs.error ?? pq.error
  const loading = !list || (draft && !p)
  return (
    <Frame step={step} resumed={Boolean(p?.setupSavedAt) && !search.step && saved > 1} programId={p?.id} saveStep={step}>
      {error ? (
        <main className="cn-main" id="main">
          <ErrorPanel title={t(mu.loadError)} actions={<Button onClick={() => void (programs.isError ? programs.refetch() : pq.refetch())}>{t(m.retry)}</Button>}>
            {errorMessage(error)}
          </ErrorPanel>
        </main>
      ) : loading ? (
        <main className="cn-main" id="main" aria-busy="true">
          <span className="vh" role="status">
            {t(mu.loadingLabel)}
          </span>
          <Skeleton h={420} r={10} />
        </main>
      ) : !canEdit ? (
        <main className="cn-main" id="main">
          <Note tone="neutral">
            <T msg={list.helpers.edit.length ? mu.needsEdit : mu.needsEditNoHelpers} values={{ helpers: orNames(list.helpers.edit), mono }} />
          </Note>
          <div>
            <ButtonLink to="/w/$workspace/grow" params={{ workspace: slug }}>
              {t(mu.backToGrow)}
            </ButtonLink>
          </div>
        </main>
      ) : (
        <>
          <Rail step={step} p={p} list={list} />
          <main className="cn-main" id="main" tabIndex={-1}>
            {step === 1 || !p ? (
              <StepList list={list} p={p} onDone={(id) => onSearch({ program: id, step: 2 })} />
            ) : step === 2 ? (
              <StepMilestones p={p} onBack={() => onSearch({ program: p.id, step: 1 })} onNext={() => onSearch({ program: p.id, step: 3 })} />
            ) : step === 3 ? (
              <StepCounting p={p} resumed={Boolean(p.setupSavedAt) && !search.step && saved === 3} onBack={() => onSearch({ program: p.id, step: 2 })} onNext={() => onSearch({ program: p.id, step: 4 })} />
            ) : step === 4 ? (
              <StepEmails p={p} onBack={() => onSearch({ program: p.id, step: 3 })} onNext={() => onSearch({ program: p.id, step: 5 })} />
            ) : step === 5 ? (
              <StepTurnOn p={p} onBack={() => onSearch({ program: p.id, step: 4 })} onEditStep={(n) => onSearch({ program: p.id, step: n })} onOn={setDone} />
            ) : done ? (
              <Done r={done} />
            ) : null}
          </main>
        </>
      )}
    </Frame>
  )
}

/** The guide's own top bar (brand, title, step badge, Save and exit), outside the app shell. */
function Frame({ step, resumed, programId, saveStep, children }: { step: number; resumed: boolean; programId?: string; saveStep: number; children: ReactNode }) {
  const t = useT()
  const { slug } = useWorkspace()
  const navigate = useNavigate()
  const exit = async () => {
    if (programId && saveStep <= 5) {
      try {
        await http.PUT('/api/v1/workspaces/{workspace}/grow/referrals/{program}/setup', { params: { path: { workspace: slug, program: programId } }, body: { step: saveStep } })
      } catch {
        // Leaving keeps the last saved step.
      }
    }
    void navigate({ to: '/w/$workspace/grow', params: { workspace: slug } })
  }
  return (
    <div className="rc-app flex min-h-screen flex-col">
      <a href="#main" className="skip">
        {t(m.title)}
      </a>
      <header className="cn-head">
        <Link to="/w/$workspace" params={{ workspace: slug }} className="cn-home" aria-label="ReputeCast">
          <LogoMark size={26} />
          <span className="text-[15px] font-bold">ReputeCast</span>
        </Link>
        <span className="k" aria-hidden="true">
          /
        </span>
        <h1 className="m-0 text-[16px] font-semibold">{t(mu.title)}</h1>
        <Badge tone="neutral">{step > 5 ? t(mu.done) : resumed ? t(mu.stepOfResumed, { n: step }) : t(mu.stepOf, { n: step })}</Badge>
        <Button className="ml-auto" onClick={() => void exit()}>
          {t(mu.saveExit)}
        </Button>
      </header>
      <SystemBanners />
      <div className="cn-wiz">{children}</div>
    </div>
  )
}

function Rail({ step, p, list }: { step: number; p?: Program; list: Schemas['ReferralProgramList'] }) {
  const t = useT()
  const ref = useRef<HTMLElement>(null)
  const scrolls = useOverflowX(ref)
  const ms = $api.useQuery(
    'get',
    `${GROW}/referrals/{program}/milestones`,
    { params: { path: { workspace: useWorkspace().slug, program: p?.id ?? '' } } },
    { enabled: Boolean(p) },
  ).data
  const listOpt = list.lists.find((l) => l.id === p?.listId)
  const items: { title: string; hint: string; done: string }[] = [
    { title: t(mu.s1), hint: t(mu.s1Hint), done: p ? t(mu.s1Done, { list: p.listName, count: count(listOpt?.subscribers ?? p.subscribers) }) : '' },
    { title: t(mu.s2), hint: t(mu.s2Hint), done: ms?.items.length ? t(mu.s2Done, { counts: ms.items.map((x) => x.count).join(' · ') }) : '' },
    {
      title: t(mu.s3),
      hint: t(mu.s3Hint),
      done: p ? (p.counting.holdDays ? t(mu.s3Done, { days: p.counting.holdDays }) : t(mu.s3DoneNoHold)) : '',
    },
    { title: t(mu.s4), hint: t(mu.s4Hint), done: ms ? t(mu.s4Done, { count: ms.items.filter((x) => x.email.on).length }) : '' },
    { title: t(mu.s5), hint: t(mu.s5Hint), done: '' },
  ]
  return (
    // eslint-disable-next-line jsx-a11y/no-noninteractive-tabindex
    <nav ref={ref} className="card cn-steps" aria-label={t(mu.stepsLabel)} tabIndex={scrolls ? 0 : undefined}>
      {items.map((x, i) => {
        const n = i + 1
        const done = n < step
        const cur = n === step
        return (
          <div key={x.title} className={cn('step cn-step', cur && 'cur', done && 'done')} aria-current={cur ? 'step' : undefined}>
            <span className="num" aria-hidden="true">
              {done ? <Icon name="check" size={12} strokeWidth={2.6} /> : n}
            </span>
            <div>
              <div className="font-semibold">
                {done ? <span className="vh">{t(mu.doneLabel)} </span> : null}
                {x.title}
              </div>
              <div className="k">{done && x.done ? x.done : x.hint}</div>
            </div>
          </div>
        )
      })}
    </nav>
  )
}

function StepCard({ n, title, lead, resumed, children, footer, failing }: { n: number; title: string; lead: string; resumed?: boolean; children: ReactNode; footer: ReactNode; failing?: boolean }) {
  const t = useT()
  return (
    <section className="card flex flex-col gap-4 px-6 py-[22px]" style={failing ? { borderColor: 'var(--bad-line)' } : undefined}>
      <div className="flex flex-col gap-1">
        <span className="self-start">
          <Badge tone="neutral">{resumed ? t(mu.stepOfResumed, { n }) : t(mu.stepOf, { n })}</Badge>
        </span>
        <h2 className="m-0 text-[21px] font-bold tracking-[-0.02em]">{title}</h2>
        <span className="max-w-[760px] text-ink-2">{lead}</span>
      </div>
      {children}
      <div className="flex flex-wrap items-center justify-between gap-2.5 border-t border-line-2 pt-3.5">{footer}</div>
    </section>
  )
}

async function saveStep(slug: string, program: string, step: number, listId?: string) {
  await http.PUT('/api/v1/workspaces/{workspace}/grow/referrals/{program}/setup', { params: { path: { workspace: slug, program } }, body: { step, listId } })
}

function StepList({ list, p, onDone }: { list: Schemas['ReferralProgramList']; p?: Program; onDone: (id: string) => void }) {
  const t = useT()
  const { slug } = useWorkspace()
  const refresh = useGrowRefresh()
  const free = list.lists.filter((l) => !l.taken || l.id === p?.listId)
  const [pick, setPick] = useState(p?.listId ?? free[0]?.id ?? '')
  const [busy, setBusy] = useState(false)
  const chosen = list.lists.find((l) => l.id === pick)
  const next = async () => {
    if (!pick) return
    setBusy(true)
    try {
      let id = p?.id
      if (!id) {
        const { data } = await http.POST('/api/v1/workspaces/{workspace}/grow/referrals', { params: { path: { workspace: slug } }, body: { listId: pick } })
        id = data!.id
      }
      await saveStep(slug, id, 2, p && p.listId !== pick ? pick : undefined)
      await refresh()
      onDone(id)
    } catch (e) {
      toast({ kind: 'error', title: t(mu.saveFailed), detail: errorMessage(e) })
    } finally {
      setBusy(false)
    }
  }
  return (
    <StepCard
      n={1}
      title={t(mu.t1)}
      lead={t(mu.l1)}
      footer={
        <>
          <ButtonLink to="/w/$workspace/grow" params={{ workspace: slug }}>
            {t(mu.cancel)}
          </ButtonLink>
          <Button variant="primary" className="min-h-10 px-4.5" loading={busy} disabled={!pick} onClick={() => void next()}>
            {t(mu.continue2)}
          </Button>
        </>
      }
    >
      {list.lists.length ? (
        <div className="grid gap-2.5" style={{ gridTemplateColumns: 'repeat(auto-fit, minmax(min(220px, 100%), 1fr))' }}>
          {list.lists.map((l) => {
            const taken = l.taken && l.id !== p?.listId
            return (
              <label key={l.id} className={cn('gw-radio', taken && 'off')}>
                <input type="radio" name="setup-list" checked={pick === l.id} disabled={taken} onChange={() => setPick(l.id)} />
                <span className="flex flex-col gap-0.5">
                  <span className="font-semibold">{l.name}</span>
                  <span className="k">{l.description ? t(mu.listLineDesc, { count: l.subscribers, description: l.description }) : t(mu.listLine, { count: l.subscribers })}</span>
                  {taken ? <span className="k">{t(mu.listTaken)}</span> : null}
                </span>
              </label>
            )
          })}
        </div>
      ) : (
        <Note tone="warn">{t(mu.noLists)}</Note>
      )}
      {chosen ? <Note tone="neutral">{t(mu.linksNote, { count: chosen.subscribers, list: chosen.name })}</Note> : null}
    </StepCard>
  )
}

function useMilestones(p: Program) {
  const { slug } = useWorkspace()
  return $api.useQuery('get', `${GROW}/referrals/{program}/milestones`, { params: { path: { workspace: slug, program: p.id } } })
}

function StepMilestones({ p, onBack, onNext }: { p: Program; onBack: () => void; onNext: () => void }) {
  const t = useT()
  const { slug } = useWorkspace()
  const q = useMilestones(p)
  const d = q.data
  const [edit, setEdit] = useState<string>()
  const [busy, setBusy] = useState(false)
  const next = async () => {
    setBusy(true)
    try {
      await saveStep(slug, p.id, 3)
      onNext()
    } catch (e) {
      toast({ kind: 'error', title: t(mu.saveFailed), detail: errorMessage(e) })
    } finally {
      setBusy(false)
    }
  }
  const have = new Set(d?.items.map((x) => x.count))
  return (
    <StepCard
      n={2}
      title={t(mu.t2)}
      lead={t(mu.l2)}
      footer={
        <>
          <Button onClick={onBack}>{t(mu.back)}</Button>
          <Button variant="primary" className="min-h-10 px-4.5" loading={busy} disabled={!d?.items.length} onClick={() => void next()}>
            {t(mu.continue3)}
          </Button>
        </>
      }
    >
      <div className="flex flex-wrap items-center gap-2">
        <span className="k">{t(mu.suggested)}</span>
        {[1, 3, 10].map((n) => (
          <span key={n} className={cn('chip', have.has(n) && 'opacity-50')}>
            {t(mu.suggestedChip, { count: n })}
          </span>
        ))}
      </div>
      {!d ? (
        <Skeleton h={160} r={10} />
      ) : d.items.length ? (
        <div className="flex flex-col gap-2">
          {d.items.map((x) => (
            <div key={x.id} className="gw-lrow rounded-[10px] border border-line px-3.5 py-3" style={{ borderTop: '1px solid var(--line)' }}>
              <span className="gw-num w-[30px] text-center text-[18px] font-bold">{x.count}</span>
              <span className="tx">
                <span className="gw-tt2">{x.rewardName}</span>
                <span className="k">{rewardLine(t, x)}</span>
              </span>
              <Button size="sm" onClick={() => setEdit(x.id)}>
                {t(mu.edit)}
              </Button>
            </div>
          ))}
        </div>
      ) : (
        <span className="k">{t(mu.noMilestones)}</span>
      )}
      <div>
        <Button size="sm" icon="plus" onClick={() => setEdit('new')}>
          {t(mu.addMilestone)}
        </Button>
      </div>
      {d && edit ? <RewardEditor d={d} milestone={edit === 'new' ? undefined : d.items.find((x) => x.id === edit)} onClose={() => setEdit(undefined)} /> : null}
    </StepCard>
  )
}

function StepCounting({ p, resumed, onBack, onNext }: { p: Program; resumed: boolean; onBack: () => void; onNext: () => void }) {
  const t = useT()
  const { slug } = useWorkspace()
  const user = useMe().data?.user
  const refresh = useGrowRefresh()
  const ms = useMilestones(p).data
  const [c, setC] = useState(p.counting)
  const [busy, setBusy] = useState(false)
  const next = async () => {
    setBusy(true)
    try {
      if (JSON.stringify(c) !== JSON.stringify(p.counting)) {
        await http.PUT('/api/v1/workspaces/{workspace}/grow/referrals/{program}/settings', {
          params: { path: { workspace: slug, program: p.id } },
          body: { baseUpdatedAt: p.updatedAt, counting: c, pages: p.pages, leaderboardPeriods: p.leaderboard.periods, leaderboardRows: p.leaderboard.rows, replyTo: p.emails.replyTo, senderId: p.emails.senderId || undefined },
        })
      }
      await saveStep(slug, p.id, 4)
      await refresh()
      onNext()
    } catch (e) {
      toast({ kind: 'error', title: t(mu.saveFailed), detail: errorMessage(e) })
    } finally {
      setBusy(false)
    }
  }
  const row = (title: string, text: string, control: ReactNode) => (
    <div className="flex flex-nowrap items-center justify-between gap-2.5">
      <span className="flex flex-col gap-0.5">
        <span className="font-semibold">{title}</span>
        <span className="k">{text}</span>
      </span>
      {control}
    </div>
  )
  const sw = (label: string, on: boolean, set: (v: boolean) => void) => <input type="checkbox" className="tg" role="switch" aria-label={label} checked={on} onChange={(e) => set(e.target.checked)} />
  return (
    <StepCard
      n={3}
      title={t(mu.t3)}
      lead={t(mu.l3)}
      resumed={resumed}
      footer={
        <>
          <Button onClick={onBack}>{t(mu.back)}</Button>
          <Button variant="primary" className="min-h-10 px-4.5" loading={busy} onClick={() => void next()}>
            {t(mu.continue4)}
          </Button>
        </>
      }
    >
      {resumed && p.setupSavedAt ? (
        <Note tone="ok">
          <span className="flex flex-col gap-1">
            <span className="font-bold">{t(mu.welcomeBack, { name: (user?.name ?? '').split(' ')[0] ?? '' })}</span>
            <span>{t(mu.welcomeBackText, { step: p.setupStep, date: `${shortDate(p.setupSavedAt)}, ${clock(p.setupSavedAt)}`, count: ms?.items.length ?? 0 })}</span>
          </span>
        </Note>
      ) : null}
      <div className="grid items-start gap-4" style={{ gridTemplateColumns: 'repeat(auto-fit, minmax(min(380px, 100%), 1fr))' }}>
        <div className="gw-sbox">
          <span className="gw-hd3">{t(mu.counting)}</span>
          {row(t(mu.confirmedOnly), t(mu.confirmedOnlyText), <Badge tone="neutral">{t(mu.alwaysOn)}</Badge>)}
          {row(
            t(mu.hold),
            t(mu.holdText),
            <Select aria-label={t(mu.hold)} style={{ width: 120 }} value={String(c.holdDays)} onChange={(e) => setC({ ...c, holdDays: Number(e.target.value) as Program['counting']['holdDays'] })}>
              {[0, 3, 7, 14, 30].map((n) => (
                <option key={n} value={n}>
                  {t(mu.holdDays, { count: n })}
                </option>
              ))}
            </Select>,
          )}
          <span className="gw-hd3 mt-1.5">{t(mu.autoTitle)}</span>
          <span className="k text-ink-nav">{t(mu.autoText)}</span>
        </div>
        <div className="gw-sbox">
          <span className="gw-hd3">{t(mu.heldTitle)}</span>
          {row(t(mu.burst), t(mu.burstText, { n: c.burst.threshold }), sw(t(mu.burst), c.burst.on, (v) => setC({ ...c, burst: { ...c.burst, on: v } })))}
          {row(t(mu.shared), t(mu.sharedText, { n: c.sharedIp.threshold }), sw(t(mu.shared), c.sharedIp.on, (v) => setC({ ...c, sharedIp: { ...c.sharedIp, on: v } })))}
          {row(t(mu.numbered), t(mu.numberedText), sw(t(mu.numbered), c.numbered, (v) => setC({ ...c, numbered: v })))}
          <span className="k">{t(mu.heldNote)}</span>
        </div>
      </div>
    </StepCard>
  )
}

function StepEmails({ p, onBack, onNext }: { p: Program; onBack: () => void; onNext: () => void }) {
  const t = useT()
  const { slug } = useWorkspace()
  const user = useMe().data?.user
  const refresh = useGrowRefresh()
  const ms = useMilestones(p).data
  const checks = $api.useQuery('get', `${GROW}/referrals/{program}/checks`, { params: { path: { workspace: slug, program: p.id } } }).data
  const [busy, setBusy] = useState(false)
  const [boardBusy, setBoardBusy] = useState(false)
  const next = async () => {
    setBusy(true)
    try {
      await saveStep(slug, p.id, 5)
      onNext()
    } catch (e) {
      toast({ kind: 'error', title: t(mu.saveFailed), detail: errorMessage(e) })
    } finally {
      setBusy(false)
    }
  }
  const test = async (id: string) => {
    if (!user) return
    try {
      await http.POST('/api/v1/workspaces/{workspace}/grow/milestones/{milestone}/email/test', { params: { path: { workspace: slug, milestone: id } }, body: { to: [user.email] } })
      toast({ kind: 'success', title: t(mu.testSent, { email: user.email }) })
    } catch (e) {
      toast({ kind: 'error', title: errorMessage(e) })
    }
  }
  const board = async (on: boolean) => {
    setBoardBusy(true)
    try {
      await http.PUT('/api/v1/workspaces/{workspace}/grow/referrals/{program}/leaderboard', { params: { path: { workspace: slug, program: p.id } }, body: { on } })
      await refresh()
    } catch (e) {
      toast({ kind: 'error', title: errorMessage(e) })
    } finally {
      setBoardBusy(false)
    }
  }
  const issue = checks?.nextIssue
  return (
    <StepCard
      n={4}
      title={t(mu.t4)}
      lead={t(mu.l4)}
      footer={
        <>
          <Button onClick={onBack}>{t(mu.back)}</Button>
          <Button variant="primary" className="min-h-10 px-4.5" loading={busy} onClick={() => void next()}>
            {t(mu.continue5)}
          </Button>
        </>
      }
    >
      <div className="gw-sbox">
        <span className="gw-hd3">{t(mu.emailsHd, { count: ms?.items.length ?? 0 })}</span>
        <div>
          {ms?.items.map((x) => (
            <div key={x.id} className="gw-lrow flex-wrap" style={{ gap: '8px 12px' }}>
              <span className="tx" style={{ flex: '1 1 260px' }}>
                <span className="gw-tt2">{x.email.on ? x.email.subject : x.rewardName}</span>
                <span className="k">{x.email.on ? t(mu.emailLine, { count: x.count }) : t(mu.emailNone)}</span>
              </span>
              <Badge tone={x.email.on ? 'ok' : 'neutral'}>{x.email.on ? t(mu.ready) : t(mu.noEmail)}</Badge>
              {x.email.on ? (
                <span className="flex gap-1.5">
                  <Link to="/w/$workspace/grow/referrals/emails/$milestoneId" params={{ workspace: slug, milestoneId: x.id }} className="btn sm">
                    {t(mu.editInComposer)}
                  </Link>
                  <Button size="sm" onClick={() => void test(x.id)}>
                    {t(mu.sendTest)}
                  </Button>
                </span>
              ) : null}
            </div>
          ))}
        </div>
      </div>
      {issue ? (
        <div className="gw-callout">
          <span className="flex min-w-0 flex-[1_1_300px] flex-col gap-0.5">
            <span className="font-semibold">{t(mu.addBlock)}</span>
            <span className="k">
              {issue.scheduledAt ? t(mu.addBlockText, { campaign: issue.name, date: shortDate(issue.scheduledAt) }) : t(mu.addBlockTextNoDate, { campaign: issue.name })}
            </span>
          </span>
          <Link to="/w/$workspace/campaigns/$campaignId/edit" params={{ workspace: slug, campaignId: issue.id }} className="btn sm pri">
            {t(mu.openDraft)}
          </Link>
        </div>
      ) : (
        <Note tone="neutral">{t(mu.addBlockNone, { list: p.listName })}</Note>
      )}
      <div className="grid gap-3" style={{ gridTemplateColumns: 'repeat(auto-fit, minmax(min(200px, 100%), 1fr))' }}>
        <PreviewCard title={t(mu.invitePage)} text={p.linkExample || t(mu.invitePageText)} page="invite" />
        <PreviewCard title={t(mu.hub)} text={t(mu.hubText)} page="hub" />
        <div className="gw-sbox justify-between">
          <span className="flex flex-col gap-0.5">
            <span className="font-semibold">{t(mu.board)}</span>
            <span className="k">{t(mu.boardText)}</span>
          </span>
          <span className="flex items-center justify-between">
            <span className="k">{p.leaderboard.on ? t(mu.boardOn) : t(mu.boardOff)}</span>
            <input type="checkbox" className="tg" role="switch" aria-label={t(mu.board)} checked={p.leaderboard.on} disabled={boardBusy} onChange={(e) => void board(e.target.checked)} />
          </span>
        </div>
      </div>
      {!p.links ? <span className="k">{t(mu.previewsLater)}</span> : null}
    </StepCard>
  )
}

function PreviewCard({ title, text, page }: { title: string; text: string; page: 'invite' | 'hub' }) {
  return (
    <div className="card gw-pv">
      <span className="mock" aria-hidden="true">
        <span className="flex items-center gap-1.5">
          <i style={{ width: 16, height: 16, background: 'var(--ink)' }} />
          <i style={{ width: 60, height: 8, background: 'var(--line-hover)' }} />
        </span>
        <i style={{ width: '80%', height: 10, background: 'var(--dark-surface)' }} />
        <i style={{ width: '95%', height: 7, background: 'var(--line)' }} />
        <i style={{ width: '40%', height: 18, background: 'var(--publisher-accent)', marginTop: 2 }} />
      </span>
      <span className="text-[13px] font-semibold">{title}</span>
      <span className="k">{text}</span>
      <span>
        <PreviewButton page={page} label={title} icon="eye" />
      </span>
    </div>
  )
}

function checkText(t: ReturnType<typeof useT>, c: Schemas['ReferralCheck']): { title: string; line: string } {
  switch (c.key) {
    case 'milestones':
      return { title: t(mu.ckMilestones), line: c.status === 'pass' ? t(mu.ckMilestonesPass, { count: c.count, counts: formatList(c.counts.map(String)) }) : t(mu.ckMilestonesFail) }
    case 'emails':
      return {
        title: t(mu.ckEmails),
        line: c.status === 'pass' ? t(mu.ckEmailsPass, { count: c.emails }) : t(mu.ckEmailsFail, { rewards: formatList(c.missing), count: c.missing.length }),
      }
    case 'files':
      return {
        title: t(mu.ckFiles),
        line:
          c.status === 'pass'
            ? c.files.length
              ? t(mu.ckFilesPass, { files: formatList(c.files.map((f) => (f.size ? t(mu.ckFileItem, { name: f.name, size: fileSize(f.size) }) : f.name))) })
              : t(mu.ckFilesNone)
            : t(mu.ckFilesFail, { rewards: formatList(c.missing), count: c.missing.length }),
      }
    case 'stripe':
      return {
        title: t(mu.ckStripe),
        line:
          c.status === 'pass'
            ? c.rewards.length
              ? t(mu.ckStripePass, { account: c.account, offer: c.offer })
              : t(mu.ckStripeNone)
            : t(mu.ckStripeFail, { rewards: formatList(c.rewards), count: c.rewards.length }),
      }
    case 'sender':
      return { title: t(mu.ckSender), line: c.status === 'pass' ? c.sender : t(mu.ckSenderFail, { sender: c.sender }) }
  }
  return { title: c.key, line: '' }
}

function StepTurnOn({ p, onBack, onEditStep, onOn }: { p: Program; onBack: () => void; onEditStep: (n: number) => void; onOn: (r: Schemas['ReferralTurnedOn']) => void }) {
  const t = useT()
  const { slug } = useWorkspace()
  const refresh = useGrowRefresh()
  const ms = useMilestones(p).data
  const cq = $api.useQuery('get', `${GROW}/referrals/{program}/checks`, { params: { path: { workspace: slug, program: p.id } } })
  const checks: Checks | undefined = cq.data
  const [busy, setBusy] = useState(false)
  useEffect(() => {
    void cq.refetch()
  }, []) // eslint-disable-line react-hooks/exhaustive-deps
  const failing = checks?.checks.filter((c) => c.status === 'fail') ?? []
  const rules = [p.counting.burst.on, p.counting.sharedIp.on, p.counting.numbered].filter(Boolean).length
  const turnOn = async () => {
    setBusy(true)
    try {
      const { data } = await http.POST('/api/v1/workspaces/{workspace}/grow/referrals/{program}/on', { params: { path: { workspace: slug, program: p.id } } })
      if (data) onOn(data)
      await refresh()
    } catch (e) {
      toast({ kind: 'error', title: errorMessage(e) })
      void cq.refetch()
    } finally {
      setBusy(false)
    }
  }
  const sum: [number, string, string][] = [
    [1, t(mu.sumList), t(mu.sumListText, { list: p.listName, count: checks?.links ?? p.subscribers })],
    [2, t(mu.sumMilestones), (ms?.items ?? []).map((x) => `${x.count} · ${x.rewardName}`).join(' · ')],
    [3, t(mu.sumCounting), p.counting.holdDays ? t(mu.sumCountingText, { days: p.counting.holdDays, rules }) : t(mu.sumCountingTextNoHold, { rules })],
    [4, t(mu.sumEmails), t(mu.sumEmailsText, { count: ms?.items.filter((x) => x.email.on).length ?? 0 })],
    [4, t(mu.sumBoard), p.leaderboard.on ? t(mu.sumBoardOn, { rows: p.leaderboard.rows }) : t(mu.sumBoardOff)],
  ]
  return (
    <StepCard
      n={5}
      title={t(mu.t5)}
      lead={t(mu.l5)}
      failing={failing.length > 0}
      footer={
        <>
          <Button onClick={onBack}>{t(mu.back)}</Button>
          <span className="flex items-center gap-2.5">
            {failing.length ? (
              <span className="k" id="ton-why">
                {failing.length === 1 ? t(mu.fixFirst) : t(mu.fixFirstMany)}
              </span>
            ) : null}
            <Button
              variant="primary"
              className="min-h-10 px-4.5"
              loading={busy}
              busyLabel={t(mu.turningOn)}
              disabledReason={!checks ? t(m.loading) : failing.length ? (failing.length === 1 ? t(mu.fixFirst) : t(mu.fixFirstMany)) : undefined}
              onClick={() => void turnOn()}
            >
              {t(mu.turnOn)}
            </Button>
          </span>
        </>
      }
    >
      <div className="grid items-stretch gap-4" style={{ gridTemplateColumns: 'repeat(auto-fit, minmax(min(380px, 100%), 1fr))' }}>
        <div className="gw-sbox">
          <div className="flex items-center justify-between">
            <span className="gw-hd3">{t(mu.summary)}</span>
            <span className="k">{t(mu.summarySteps)}</span>
          </div>
          <div className="flex flex-col">
            {sum.map(([n, title, line], i) => (
              <div key={i} className="gw-chk">
                <span className="gw-num k" style={{ width: 14, textAlign: 'right', paddingTop: 1 }}>
                  {i + 1}
                </span>
                <span>
                  <span className="font-semibold">{title}</span>
                  <span className="k">{line}</span>
                </span>
                <button type="button" className="gw-lnk border-0 bg-transparent p-0 text-[12.5px] text-accent" onClick={() => onEditStep(n)}>
                  {t(mu.edit)}
                </button>
              </div>
            ))}
          </div>
        </div>
        <div className="gw-sbox">
          <div className="flex items-center justify-between">
            <span className="gw-hd3">{t(mu.checks)}</span>
            {checks ? (
              failing.length ? (
                <Badge tone="bad">{t(mu.checksFail, { count: failing.length })}</Badge>
              ) : (
                <Badge tone="ok">{t(mu.checksPass, { count: checks.checks.length, total: checks.checks.length })}</Badge>
              )
            ) : null}
          </div>
          {!checks ? (
            <Skeleton h={180} r={8} />
          ) : (
            <div className="flex flex-col">
              {checks.checks.map((c) => {
                const x = checkText(t, c)
                const ok = c.status === 'pass'
                return (
                  <div key={c.key} className="gw-chk">
                    <span style={{ color: ok ? 'var(--ok-fg)' : 'var(--bad-fg)', paddingTop: 1 }} aria-hidden="true">
                      <Icon name={ok ? 'check' : 'alertCircle'} size={16} />
                    </span>
                    <span>
                      <span className="font-semibold">{x.title}</span>
                      <span className="k">{x.line}</span>
                    </span>
                    <Badge tone={ok ? 'ok' : 'bad'}>{ok ? t(mu.pass) : t(mu.fix)}</Badge>
                  </div>
                )
              })}
            </div>
          )}
          {failing.some((c) => c.key === 'stripe') ? (
            <div className="flex flex-wrap gap-2">
              <ButtonLink to="/w/$workspace/earn/settings" params={{ workspace: slug }} search={{ tab: 'payments' }} size="sm" variant="primary">
                {t(mu.connectStripe)}
              </ButtonLink>
              <Button size="sm" onClick={() => onEditStep(2)}>
                {t(mu.pickReward)}
              </Button>
            </div>
          ) : null}
        </div>
      </div>
      <Note tone="info">
        <span className="flex flex-col gap-1">
          <span className="font-bold">{t(mu.whenOn)}</span>
          <span>{t(mu.whenOnText, { count: checks?.links ?? p.subscribers })}</span>
        </span>
      </Note>
    </StepCard>
  )
}

function Done({ r }: { r: Schemas['ReferralTurnedOn'] }) {
  const t = useT()
  const { slug } = useWorkspace()
  const p = r.program
  const ms = useMilestones(p).data
  return (
    <section className="card flex flex-col gap-3.5 px-6 py-[22px]">
      <div className="flex items-center gap-2.5">
        <span className="gw-ti" style={{ width: 36, height: 36, background: 'var(--ok-bg)', color: 'var(--ok-fg)' }} aria-hidden="true">
          <Icon name="check" size={18} strokeWidth={2.4} />
        </span>
        <h2 className="m-0 text-[20px] font-bold tracking-[-0.02em]">{t(mu.doneTitle)}</h2>
      </div>
      <KeyValues
        items={[
          [t(mu.doneList), t(mu.doneListValue, { list: p.listName, count: r.linksCreated })],
          [t(mu.doneMilestones), t(mu.doneMilestonesValue, { counts: (ms?.items ?? []).map((x) => x.count).join(' · '), emails: ms?.items.filter((x) => x.email.on).length ?? 0 })],
          [t(mu.doneCounting), p.counting.holdDays ? t(mu.s3Done, { days: p.counting.holdDays }) : t(mu.s3DoneNoHold)],
          [t(mu.donePages), t(mu.donePagesValue, { board: p.leaderboard.on ? t(mu.boardOn).toLowerCase() : t(mu.boardOff).toLowerCase() })],
          [t(mu.doneNext), r.nextIssue ? t(mu.doneNextValue, { campaign: r.nextIssue.name }) : t(mu.doneNextNone)],
        ]}
      />
      <div className="flex flex-wrap gap-2">
        <ButtonLink to="/w/$workspace/grow/referrals" params={{ workspace: slug }} variant="primary">
          {t(mu.openResults)}
        </ButtonLink>
        {r.nextIssue ? (
          <ButtonLink to="/w/$workspace/campaigns/$campaignId/edit" params={{ workspace: slug, campaignId: r.nextIssue.id }}>
            {t(mu.openCampaign, { campaign: r.nextIssue.name })}
          </ButtonLink>
        ) : null}
        <ButtonLink to="/w/$workspace/grow/referrals/milestones" params={{ workspace: slug }}>
          {t(mu.milestonesLink)}
        </ButtonLink>
      </div>
    </section>
  )
}

