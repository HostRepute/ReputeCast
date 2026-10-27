import { Link } from '@tanstack/react-router'
import { DropdownMenu } from 'radix-ui'
import { useRef, useState, type ReactNode } from 'react'
import { $api } from '@/api/client'
import { Icon } from '@/components/icons/Icon'
import { Button, ButtonLink } from '@/components/ui/Button'
import { Note } from '@/components/ui/Note'
import { T, useT } from '@/i18n'
import { cn } from '@/lib/cn'
import { shortDate } from '@/lib/format'
import { useCan, useWorkspace } from '@/lib/session'
import { m } from './messages'
import { RunDialog, type RunPreset } from './RunDialog'
import { base, dayTime, PBar, Pv, resultLine, throughText, usePlacementAccess, type TestRow } from './shared'

/*
 * Where else a placement test starts or shows (brief §6): Review & send's
 * optional card, the approval's context, the composer's and the template
 * editor's Send test menu, the warm-up planner's card and the note in
 * Instance admin › Authentication. Each opens Run a placement test in place
 * with what it knows; a test is never part of a send's checks.
 */

const running = (r: TestRow) => r.status === 'sending' || r.status === 'waiting'

/**
 * The control that opens Run a placement test, and a close that gives it
 * the focus back (the dialog has no Radix trigger to return to).
 */
function useOpener<T extends HTMLElement>() {
  const ref = useRef<T>(null)
  const [open, setOpen] = useState(false)
  const close = () => {
    setOpen(false)
    window.setTimeout(() => ref.current?.focus(), 0)
  }
  return { ref, open, setOpen, close }
}

/** The workspace's tests, newest first, for people who can see Deliverability. */
function useTestList() {
  const { slug } = useWorkspace()
  const canView = useCan('deliverability:view')
  return $api.useQuery(
    'get',
    `${base}/tests`,
    { params: { path: { workspace: slug } } },
    { enabled: canView, refetchInterval: (q) => (q.state.data?.tests.some(running) ? 4000 : false) },
  )
}

/** A content's newest test. */
function useLatest(kind: TestRow['contentKind'], id: string) {
  const q = useTestList()
  return { q, row: q.data?.tests.find((x) => x.contentKind === kind && x.contentId === id) }
}

/** The person's reason when they can't run a test, from the list's helpers when it has them. */
function useRunReason(list?: { canRun: boolean; canManage: boolean; runHelpers: string[] }) {
  const canRun = useCan('placement:run')
  const access = usePlacementAccess(list ?? { canRun, canManage: false, runHelpers: [] })
  return canRun ? undefined : access.runReason
}

/** Inbox, tab, spam and not arrived per provider; found so far while it runs. */
function ProviderResults({ row }: { row: TestRow }) {
  const t = useT()
  const live = running(row)
  return (
    <div className="pl-eres">
      {row.providers.map((p) => (
        <div className={row.providers.length > 1 ? 'ln' : 'ln one'} key={p.id}>
          {row.providers.length > 1 ? <Pv adapter={p.adapter}>{p.short}</Pv> : null}
          <PBar counts={p.counts} waiting={live} />
          <span className="ct">{live ? t(m.entryFound, { found: p.counts.found, copies: p.counts.copies }) : resultLine(p.counts)}</span>
        </div>
      ))}
    </div>
  )
}

function ResultsLink({ row, children }: { row: TestRow; children: ReactNode }) {
  const { slug } = useWorkspace()
  return (
    <Link to="/w/$workspace/deliverability/placement/$testId" params={{ workspace: slug, testId: row.id }} className="pl-lnk">
      {children}
    </Link>
  )
}

/** "SES · Inbox 8 · Tab 2 · Spam 0 · Not arrived 1", or what's found so far while the test runs. */
function reviewLine(t: ReturnType<typeof useT>, p: TestRow['providers'][number], several: boolean, live: boolean): string {
  const c = p.counts
  const parts = live
    ? [t(m.entryFound, { found: c.found, copies: c.copies })]
    : [t(m.resInbox, { n: c.inbox }), t(m.resTab, { n: c.tab }), t(m.resSpam, { n: c.spam }), ...(c.notArrived > 0 ? [t(m.resNotArrived, { n: c.notArrived })] : [])]
  return (several ? [p.short, ...parts] : parts).join(' · ')
}

/**
 * Review & send's placement card (SR-9, SendReadiness): the seed mailboxes a
 * test reaches, the campaign's last result per sending provider (Inbox,
 * Tab, Spam, Not arrived) with View results, and Run placement
 * test, which starts one for this campaign as scheduled. Optional, and
 * never part of the verdict; a route over several providers compares them.
 */
export function PlacementReviewCard({ campaignId, campaignName, providers, className }: { campaignId: string; campaignName: string; providers: string[]; className?: string }) {
  const t = useT()
  const { slug } = useWorkspace()
  const canView = useCan('deliverability:view')
  const { q, row } = useLatest('campaign', campaignId)
  const seeds = $api.useQuery('get', `${base}/seeds`, { params: { path: { workspace: slug } } }, { enabled: canView })
  const reason = useRunReason(q.data)
  const { ref: runRef, open: runOpen, setOpen: setRunOpen, close: closeRun } = useOpener<HTMLButtonElement>()
  if (!q.data) return null
  const used = [...new Set(providers)]
  const preset: RunPreset = {
    origin: 'review',
    content: { kind: 'campaign', id: campaignId },
    through: used.length > 1 ? 'compare' : 'route',
    providers: used.length > 1 ? used : undefined,
    from: campaignName,
  }
  const connected = seeds.data?.seeds.filter((s) => s.status === 'connected') ?? []
  const names = [...new Set(connected.map((s) => s.providerName))]
  // A test skips the seeds whose sign-in failed or expired: the caption says so.
  const attention = seeds.data?.seeds.filter((s) => s.status === 'failed' || s.status === 'expired') ?? []
  const live = row ? running(row) : false
  return (
    <section className={cn('card pl-rv', className)} aria-labelledby="pl-review-card">
      <div className="pl-rv-h">
        <div className="flex min-w-0 flex-col gap-0.5">
          <h2 className="h2" id="pl-review-card">
            {t(m.entryTitle)}
          </h2>
          {seeds.data ? (
            <span className="k">
              {connected.length ? t(m.entrySeeds, { count: connected.length, names: names.join(', '), attention: attention.length }) : t(m.entryNoSeeds)}
            </span>
          ) : null}
        </div>
        <span className="badge mut">{t(m.entryOptional)}</span>
      </div>
      <div className="pl-rv-b">
        {row ? (
          <div className="pl-rv-last">
            <span className="k">{t(m.entryLast, { day: shortDate(row.startedAt), name: row.contentName })}</span>
            {row.providers.map((p) => (
              <div key={p.id} className="flex flex-col gap-1.5">
                <span className="font-medium">{reviewLine(t, p, row.providers.length > 1, live)}</span>
                <PBar counts={p.counts} waiting={live} />
              </div>
            ))}
          </div>
        ) : (
          <span className="k pl-rv-last">{t(m.entryNotRun)}</span>
        )}
        {row ? (
          <ButtonLink to="/w/$workspace/deliverability/placement/$testId" params={{ workspace: slug, testId: row.id }}>
            {t(m.entryViewResults)}
          </ButtonLink>
        ) : null}
        <Button ref={runRef} disabledReason={reason} onClick={() => setRunOpen(true)}>
          {t(m.runTest)}
        </Button>
      </div>
      {runOpen ? <RunDialog open onOpenChange={(v) => !v && closeRun()} preset={preset} /> : null}
    </section>
  )
}

/** A send approval's context: the campaign's latest test, when there is one. */
export function PlacementApprovalCard({ campaignId }: { campaignId: string }) {
  const t = useT()
  const { slug } = useWorkspace()
  const { row } = useLatest('campaign', campaignId)
  if (!row) return null
  return (
    <section className="card" aria-labelledby="pl-approval-card">
      <div className="sec">
        <div className="flex min-w-0 flex-col gap-0.5">
          <h2 className="h2" id="pl-approval-card">
            {t(m.entryTitle)}
          </h2>
          <span className="k">{t(m.approvalCaption, { time: dayTime(row.startedAt), name: row.startedBy, through: throughText(row) })}</span>
        </div>
        <ButtonLink size="sm" to="/w/$workspace/deliverability/placement/$testId" params={{ workspace: slug, testId: row.id }}>
          {t(m.entryViewResults)}
        </ButtonLink>
      </div>
      <div className="pad">
        <ProviderResults row={row} />
      </div>
    </section>
  )
}

/**
 * The composer's Send test as a split button: its own test, and "Run a
 * placement test…" through the route as configured. `before` saves the
 * draft first, as test emails do.
 */
export function TestSplit({
  children,
  campaignId,
  toMe,
  onSendTest,
  before,
}: {
  /** The main Send test control. */
  children: ReactNode
  campaignId: string
  /** The person can only send tests to themselves. */
  toMe?: boolean
  onSendTest: () => void
  before?: () => Promise<unknown>
}) {
  const t = useT()
  return (
    <div className="pl-split">
      {children}
      <TestMenu
        trigger={
          <button type="button" className="btn ic" aria-label={t(m.testMore)}>
            <Icon name="chevronDown" size={14} />
          </button>
        }
        sendLabel={toMe ? t(m.testMenuMe) : t(m.testMenuEmail)}
        onSendTest={onSendTest}
        preset={{ origin: 'composer', content: { kind: 'campaign', id: campaignId } }}
        before={before}
      />
    </div>
  )
}

/**
 * The transactional template editor's Send test menu: "Run a placement
 * test…" with the published version and a test data set, through the
 * Transactional route. (The editor has no test email of a draft; see
 * DESIGN_INVENTORY #151.)
 */
export function TemplateTestMenu({ templateId, published }: { templateId: string; published: boolean }) {
  const t = useT()
  return (
    <TestMenu
      trigger={
        <button type="button" className="btn">
          {t(m.sendTest)}
          <Icon name="chevronDown" size={14} />
        </button>
      }
      preset={{ origin: 'template', content: { kind: 'transactional', id: templateId } }}
      unpublished={!published}
    />
  )
}

function TestMenu({
  trigger,
  sendLabel,
  onSendTest,
  preset,
  before,
  unpublished,
}: {
  trigger: ReactNode
  sendLabel?: string
  onSendTest?: () => void
  preset: RunPreset
  before?: () => Promise<unknown>
  unpublished?: boolean
}) {
  const t = useT()
  const canRun = useCan('placement:run')
  const { ref: runRef, open: runOpen, setOpen: setRunOpen, close: closeRun } = useOpener<HTMLButtonElement>()
  // What to open once the menu has closed and given the focus back to its
  // trigger, so the popover or dialog returns it there too.
  const next = useRef<(() => void) | null>(null)
  const openRun = async () => {
    if (before) await before()
    setRunOpen(true)
  }
  const note = !canRun ? t(m.testMenuNeeds) : unpublished ? t(m.testMenuPublish) : undefined
  return (
    <>
      <DropdownMenu.Root modal={false}>
        <DropdownMenu.Trigger asChild ref={runRef}>
          {trigger}
        </DropdownMenu.Trigger>
        <DropdownMenu.Portal>
          <DropdownMenu.Content
            className="menu"
            align="end"
            sideOffset={6}
            collisionPadding={12}
            aria-label={t(m.sendTest)}
            onCloseAutoFocus={() => {
              const go = next.current
              next.current = null
              if (go) window.setTimeout(go, 0)
            }}
          >
            {onSendTest ? (
              <DropdownMenu.Item
                onSelect={() => {
                  next.current = onSendTest
                }}
              >
                {sendLabel}
              </DropdownMenu.Item>
            ) : null}
            <DropdownMenu.Item
              disabled={Boolean(note)}
              onSelect={() => {
                next.current = () => void openRun()
              }}
            >
              {t(m.testMenuPlacement)}
              {note ? <span className="k ml-auto pl-4">{note}</span> : null}
            </DropdownMenu.Item>
          </DropdownMenu.Content>
        </DropdownMenu.Portal>
      </DropdownMenu.Root>
      {runOpen ? <RunDialog open onOpenChange={(v) => !v && closeRun()} preset={preset} /> : null}
    </>
  )
}

/**
 * The warm-up planner: "Placement during this warm-up", the tests through
 * the warming provider on the days they ran, and "Run a test through
 * Postal" with the last campaign sent.
 */
export function WarmupPlacementCard({ providerId, provider, day, days, className }: { providerId: string; provider: string; day: number; days: number; className?: string }) {
  const t = useT()
  const { slug } = useWorkspace()
  const q = useTestList()
  const reason = useRunReason(q.data)
  const { ref: runRef, open: runOpen, setOpen: setRunOpen, close: closeRun } = useOpener<HTMLButtonElement>()
  const canRun = useCan('placement:run')
  // The last campaign sent, for the dialog (the search lists the newest first).
  const sent = $api.useQuery('get', `${base}/content`, { params: { path: { workspace: slug }, query: { q: 'sent' } } }, { enabled: runOpen && canRun })
  if (!q.data) return null
  const rows = q.data.tests
    .map((row) => ({ row, p: row.providers.find((p) => p.id === providerId && (p.warmDay ?? 0) > 0) }))
    .filter((x): x is { row: TestRow; p: TestRow['providers'][number] } => Boolean(x.p))
    .sort((a, b) => (a.p.warmDay ?? 0) - (b.p.warmDay ?? 0) || a.row.startedAt.localeCompare(b.row.startedAt))
  const last = sent.data?.items.find((c) => c.kind === 'campaign' && c.note === 'sent')
  return (
    <section className={cn('card flex flex-col gap-2.5 px-5 py-[18px]', className)} aria-labelledby="pl-warmup-card">
      <h2 className="h2" id="pl-warmup-card">
        {t(m.warmTitle)}
      </h2>
      {rows.length ? (
        <div className="flex flex-col gap-2.5">
          {rows.map(({ row, p }) => (
            <div key={row.id} className="flex flex-col gap-1">
              <span className="flex justify-between gap-2 text-[13px]">
                <span className="min-w-0">
                  <ResultsLink row={row}>{t(m.warmDay, { day: p.warmDay ?? 0 })}</ResultsLink>
                  <span className="k"> · {row.contentName}</span>
                </span>
                <span className="k flex-none">{shortDate(row.startedAt)}</span>
              </span>
              <PBar counts={p.counts} waiting={running(row)} />
              <span className="k">{running(row) ? t(m.entryFound, { found: p.counts.found, copies: p.counts.copies }) : resultLine(p.counts)}</span>
            </div>
          ))}
        </div>
      ) : (
        <span className="k">{t(m.warmNone, { provider })}</span>
      )}
      <Button ref={runRef} className="self-start" disabledReason={reason} onClick={() => setRunOpen(true)}>
        {t(m.warmRun, { provider })}
      </Button>
      {runOpen && !sent.isPending ? (
        <RunDialog
          open
          onOpenChange={(v) => !v && closeRun()}
          preset={{
            origin: 'warmup',
            content: last ? { kind: 'campaign', id: last.id } : undefined,
            through: 'provider',
            providers: [providerId],
            from: t(m.warmFrom, { provider, day, days }),
          }}
        />
      ) : null}
    </section>
  )
}

/**
 * Instance admin › Authentication: the Google and Microsoft sign-in apps
 * also give seed mailboxes their mail access, at their own redirect URI.
 */
export function SeedAppNote({ kind, signInRedirect }: { kind: string; signInRedirect: string }) {
  if (kind !== 'google' && kind !== 'microsoft') return null
  // The seeds' callback sits beside the sign-in one, under the same base URL.
  const at = signInRedirect.indexOf('/auth/social/')
  const uri = (at >= 0 ? signInRedirect.slice(0, at) : signInRedirect.replace(/\/+$/, '')) + '/api/v1/placement/oauth/callback'
  return (
    <Note tone="info" role="note">
      <T msg={kind === 'google' ? m.seedAppGoogle : m.seedAppMicrosoft} values={{ uri, mono: (c) => <span className="mono break-all">{c}</span> }} />
    </Note>
  )
}
