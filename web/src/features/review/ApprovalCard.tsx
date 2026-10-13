import { useState, type ReactNode } from 'react'
import type { Schemas } from '@/api/client'
import { Badge } from '@/components/ui/Badge'
import { Button, ButtonLink } from '@/components/ui/Button'
import { Dialog } from '@/components/ui/Dialog'
import { Field, Textarea } from '@/components/ui/Field'
import { count } from '@/lib/format'
import { useNow } from '@/lib/hooks'
import { useMe, useWorkspace } from '@/lib/session'
import { approverNames, DiffLines, MiniBanner, PersonLine, unchangedSentence } from '@/features/approvals/parts'
import { acknowledgements, countsShort, plural, scheduleShort, verdictTitle, waitWords, type Review, type Run } from './logic'
import { AppHref, whenText } from './parts'

export type ApprovalMode = 'none' | 'required' | 'waiting' | 'approved' | 'invalidated'

/**
 * Which approval card Review & send shows (SendReadinessApproval): none when
 * no rule matches or a valid approval lets the viewer send.
 */
export function approvalMode(r: Review): ApprovalMode {
  const a = r.approval
  if (!['draft', 'approval', 'approved'].includes(r.campaign.status)) return 'none'
  if (a.request?.status === 'pending') return 'waiting'
  // Blocks come first: nobody can ask for approval of a blocked send.
  if (!a.required || r.run?.verdict === 'blocked') return 'none'
  if (a.valid) return r.access.canSend ? 'none' : 'approved'
  if (a.request?.status === 'invalidated') return 'invalidated'
  return 'required'
}

type Card = { review: Review; run: Run; children: ReactNode; title: string; id: string }

function Shell({ review, run, title, id, children }: Card) {
  return (
    <section className="card" aria-labelledby={id}>
      <div className="sec">
        <div className="col gap-0.5!">
          <h2 className="h2" id={id}>
            {title}
          </h2>
          <span className="k">
            Review &amp; send · {review.campaign.name} · {count(run.estimates.audience.final)} recipients
          </span>
        </div>
      </div>
      <div className="pad col gap-3.5!">{children}</div>
    </section>
  )
}

function Foot({ children }: { children: ReactNode }) {
  return <div className="row ap-foot">{children}</div>
}

/** "You don't hold campaigns:send · Content Writer" when the reader is the author. */
function ruleText(rule: Schemas['ApprovalRuleMatch'], me?: string): string {
  if (rule.key === 'author_lacks_send' && rule.author && rule.author === me) return `You don't hold campaigns:send${rule.role ? ` · ${rule.role}` : ''}`
  return rule.label
}

/** Approval required: the rules that matched, a note, Request approval. */
export function RequiredCard({
  review,
  run,
  busy,
  onRequest,
  onBypass,
  onSaveDraft,
}: {
  review: Review
  run: Run
  busy: boolean
  onRequest: (note: string) => void
  onBypass?: () => void
  onSaveDraft: (note: string) => void
}) {
  const { data: me } = useMe()
  const a = review.approval
  const storeKey = `rc-approval-note:${review.campaign.id}`
  const [note, setNote] = useState(() => {
    try {
      return window.sessionStorage.getItem(storeKey) ?? ''
    } catch {
      return ''
    }
  })
  const keep = (v: string) => {
    setNote(v)
    try {
      window.sessionStorage.setItem(storeKey, v)
    } catch {
      // A private window may refuse storage; the note stays in the field.
    }
  }
  const myName = me?.user.name
  const isAuthor = !a.request || a.request.requestedBy.name === myName || ['changes_requested', 'rejected', 'cancelled'].includes(a.request.status)
  const approvers = a.approvers.filter((p) => p.name !== myName)
  const prev = a.request && ['changes_requested', 'rejected'].includes(a.request.status) ? a.request : undefined
  return (
    <Shell review={review} run={run} title="Approval required" id="rv-approval">
      <MiniBanner tone="warn" title={verdictTitle(run.verdict)}>
        {countsShort(run.counts)}
      </MiniBanner>
      {prev ? (
        <div className="note i" role="status">
          <Badge tone="info">{prev.status === 'rejected' ? 'Rejected' : 'Changes requested'}</Badge>
          <span>
            {prev.decidedBy || 'An approver'} {prev.status === 'rejected' ? 'rejected' : 'asked for changes to'} v{prev.version}
            {prev.comment ? `: “${prev.comment}”` : '.'} Make the changes, then request approval again.
          </span>
        </div>
      ) : null}
      <div className="card ap-inner i-warn">
        <div className="row gap-2!">
          <span className="font-semibold">Approval required before this send</span>
          <Badge tone="warn">
            {a.rules.length} {plural(a.rules.length, 'rule', 'rules')} matched
          </Badge>
        </div>
        <ul className="ap-list">
          {a.rules.map((r) => (
            <li key={r.key}>{ruleText(r, myName)}</li>
          ))}
        </ul>
        <Field label="Note for approvers" optional>
          <Textarea rows={2} className="min-h-0" value={note} maxLength={2000} onChange={(e) => keep(e.target.value)} />
        </Field>
        <span className="k">
          Approvers: {approverNames(approvers, 'comma') || 'none yet'} · {a.needed} {plural(a.needed, 'approval', 'approvals')} needed
          {isAuthor ? " · you can't approve your own campaign" : ''}
        </span>
      </div>
      <Foot>
        <Button onClick={() => onSaveDraft(note)}>Save draft</Button>
        {onBypass ? (
          <Button variant="danger" onClick={onBypass}>
            Send without approval
          </Button>
        ) : null}
        <Button variant="primary" disabledReason={a.canRequest ? undefined : a.requestReason || 'Approval can’t be requested right now.'} loading={busy} onClick={() => onRequest(note.trim())}>
          Request approval
        </Button>
      </Foot>
    </Shell>
  )
}

/** Waiting for approval: who was asked, what restarts the request. */
export function WaitingCard({
  review,
  run,
  busy,
  onRemind,
  onCancel,
  onBypass,
}: {
  review: Review
  run: Run
  busy?: 'remind' | 'cancel'
  onRemind: () => void
  onCancel: () => void
  onBypass?: () => void
}) {
  const { slug } = useWorkspace()
  const { data: me } = useMe()
  const now = useNow()
  const a = review.approval
  const req = a.request!
  const mine = req.requestedBy.name === me?.user.name
  const wait = a.medianWaitMinutes != null ? `; median wait is ${waitWords(a.medianWaitMinutes)}` : ''
  return (
    <Shell review={review} run={run} title="Waiting for approval" id="rv-approval">
      <MiniBanner tone="warn" title={verdictTitle(run.verdict)}>
        Snapshot v{req.version} · locked while the request is open
      </MiniBanner>
      <div className="card ap-inner">
        <div className="row gap-2.5!">
          <Badge tone="warn">Pending</Badge>
          <span className="font-semibold">
            Requested {ago(req.requestedAt, now)} · {req.given} of {req.needed} {plural(req.needed, 'approval', 'approvals')}
            {mine ? '' : ` · by ${req.requestedBy.name}`}
          </span>
        </div>
        <div className="row gap-3!">
          {a.approvers
            .filter((p) => p.id !== req.requestedBy.id)
            .map((p) => (
              <PersonLine key={p.id} person={p} line={`approver · ${p.presence ?? 'away'}`} />
            ))}
        </div>
        <div className="note n text-[12.5px]">
          <span>
            Editing the content, audience, route or schedule restarts the request. Approvers were notified in-app and by email{wait}.
          </span>
        </div>
      </div>
      <Foot>
        {mine ? (
          <>
            <ButtonLink to="/w/$workspace/campaigns/$campaignId/edit" params={{ workspace: slug, campaignId: review.campaign.id }}>
              Edit campaign (restarts approval)
            </ButtonLink>
            <Button loading={busy === 'remind'} onClick={onRemind}>
              Remind approvers
            </Button>
            <Button variant="danger" loading={busy === 'cancel'} onClick={onCancel}>
              Cancel request
            </Button>
          </>
        ) : (
          <>
            {onBypass ? (
              <Button variant="danger" onClick={onBypass}>
                Send without approval
              </Button>
            ) : null}
            {review.access.isOwner ? (
              <Button variant="danger" loading={busy === 'cancel'} onClick={onCancel}>
                Cancel request
              </Button>
            ) : null}
            <ButtonLink variant="primary" to="/w/$workspace/approvals/$approvalId" params={{ workspace: slug, approvalId: req.id }}>
              {review.access.canApprove ? 'Review request' : 'Open request'}
            </ButtonLink>
          </>
        )}
      </Foot>
    </Shell>
  )
}

/** Approved · ready to schedule (the author schedules it as approved). */
export function ApprovedCard({
  review,
  run,
  busy,
  onTest,
  onSchedule,
}: {
  review: Review
  run: Run
  busy?: 'test'
  onTest: () => void
  onSchedule: () => void
}) {
  const now = useNow()
  const req = review.approval.request!
  const snap = req.snapshot
  const decided = req.decidedAt ? whenText(req.decidedAt, now) : ''
  const route = snap.routeLabel ? `, route ${snap.routeLabel}` : ''
  const sched = snap.scheduleMode === 'now' ? ', send now' : `, schedule ${snap.scheduleText}`
  const reason = review.access.canEdit ? undefined : 'Scheduling an approved campaign needs campaigns:edit.'
  return (
    <Shell review={review} run={run} title="Approved" id="rv-approval">
      <MiniBanner tone="ok" title="Approved · ready to schedule">
        Approved by {req.decidedBy || 'an approver'}
        {decided ? `, ${decided}` : ''}
        {req.comment ? ` · comment: “${req.comment}”` : ''}
      </MiniBanner>
      <div className="card ap-inner">
        <dl className="kv m-0 text-[13px]">
          <dt>Approved snapshot</dt>
          <dd>
            v{req.version} · subject, content, audience {count(snap.audienceCount)}
            {route}
            {sched}
          </dd>
          <dt>Valid until</dt>
          <dd>Any change to the snapshot, or the audience moving more than 10%</dd>
          <dt>At send time</dt>
          <dd>Send Readiness runs again; a Block stops the send and notifies everyone</dd>
        </dl>
        {req.audit ? (
          <div className="note g text-[12.5px]">
            <Badge tone="ok">Logged</Badge>
            <span>
              {req.audit.action} · {req.audit.actor} · {whenText(req.audit.at, now)}
              {req.audit.ip ? ` · from ${req.audit.ip}` : ''}
            </span>
          </div>
        ) : null}
      </div>
      <Foot>
        <Button loading={busy === 'test'} onClick={onTest}>
          Send a test
        </Button>
        <Button variant="primary" disabledReason={reason} onClick={onSchedule}>
          {review.schedule.mode === 'now' ? 'Send now' : `Schedule for ${scheduleShort(review.schedule)}`}
        </Button>
      </Foot>
    </Shell>
  )
}

/** Approval invalidated: the diff of what changed since the approval. */
export function InvalidatedCard({
  review,
  run,
  busy,
  onRestore,
  onRequest,
}: {
  review: Review
  run: Run
  busy?: 'restore' | 'request'
  onRestore?: () => void
  onRequest: () => void
}) {
  const now = useNow()
  const { data: me } = useMe()
  const req = review.approval.request!
  const who = req.invalidatedBy && req.invalidatedBy === me?.user.name ? 'You' : req.invalidatedBy || 'Someone'
  const fields = req.invalidatedReason?.replace(/ changed$/, '') ?? 'the campaign'
  const moved = req.changes.some((c) => c.field === 'audience_count')
  const approvedBy = req.decidedBy ? `${req.decidedBy}'s approval covered v${req.version}.` : `The request covered v${req.version}.`
  const edit = moved
    ? ` The audience moved more than 10% since the request.`
    : ` ${who} edited the ${fields}${req.invalidatedAt ? ` at ${whenText(req.invalidatedAt, now)}` : ''} (v${review.campaign.version}).`
  const scheduled = review.campaign.scheduledAt ? 'The scheduled time stays reserved but the send' : 'The send'
  return (
    <Shell review={review} run={run} title="Approval invalidated" id="rv-approval">
      <MiniBanner tone="bad" title={`Approval invalidated: ${req.invalidatedReason || 'the campaign changed'}`}>
        {approvedBy}
        {edit}
      </MiniBanner>
      <div className="card ap-inner">
        {req.changes.length ? <DiffLines changes={req.changes} /> : null}
        <span className="k">
          {unchangedSentence(req.changes)} {scheduled} won't start without a new approval; approvers see this diff.
        </span>
      </div>
      <Foot>
        {onRestore ? (
          <Button loading={busy === 'restore'} onClick={onRestore}>
            Restore v{req.version} {restoreWhat(req.changes)}(keeps approval)
          </Button>
        ) : null}
        <Button variant="primary" loading={busy === 'request'} disabledReason={review.approval.canRequest ? undefined : review.approval.requestReason} onClick={onRequest}>
          Request approval again
        </Button>
      </Foot>
    </Shell>
  )
}

/** "subject ", "subject and link ", "" — what Restore puts back. */
function restoreWhat(changes: Schemas['ApprovalChange'][]): string {
  const f = [...new Set(changes.map((c) => (c.field === 'link' ? 'link' : c.field)))]
  if (f.length === 1 && ['subject', 'preheader', 'schedule', 'audience'].includes(f[0]!)) return `${f[0]} `
  if (f.length === 1 && f[0] === 'link') return 'link '
  return ''
}

/** Changes Restore can undo: content, audience and schedule (not route or drift). */
export function canRestore(changes: Schemas['ApprovalChange'][]): boolean {
  return changes.length > 0 && changes.every((c) => ['subject', 'preheader', 'link', 'content', 'sender', 'audience', 'schedule'].includes(c.field))
}

/** The send-time recheck turned Blocked: the send didn't start. */
export function HeldCard({
  held,
  scheduledAt,
  approved,
  notified,
  children,
}: {
  held: Schemas['ReviewHeld']
  scheduledAt?: string | null
  approved?: ReactNode
  notified?: ReactNode
  children?: ReactNode
}) {
  const now = useNow()
  return (
    <section className="card border-bad-line" aria-labelledby="rv-held">
      <div className="sec">
        <div className="col gap-0.5!">
          <h2 className="h2" id="rv-held">
            Send-time recheck turned Blocked
          </h2>
          <span className="k">{scheduledAt ? `${whenText(scheduledAt, now)} · the send did not start` : 'The send did not start'}</span>
        </div>
        <Badge tone="bad">Blocked</Badge>
      </div>
      <div className="pad col gap-3!">
        {held.blocks.length ? (
          held.blocks.map((b) => {
            const [group, ...rest] = (b.flatTitle || b.title).split(': ')
            return (
              <div key={b.key} className="note b" role="alert">
                <Badge tone="bad">Block</Badge>
                <span>
                  {rest.length ? (
                    <>
                      <b>{group}:</b> {rest.join(': ')}.
                    </>
                  ) : (
                    `${b.title}.`
                  )}{' '}
                  {b.detail ? `${b.detail} ` : ''}
                  The approved send was held.
                </span>
              </div>
            )
          })
        ) : (
          <div className="note b" role="alert">
            <Badge tone="bad">Block</Badge>
            <span>{held.reason}. The approved send was held.</span>
          </div>
        )}
        <dl className="kv m-0 text-[12.5px]">
          {approved ? (
            <>
              <dt>Approved</dt>
              <dd>{approved}</dd>
            </>
          ) : null}
          {notified ? (
            <>
              <dt>Notified</dt>
              <dd>{notified}</dd>
            </>
          ) : null}
          <dt>Schedule</dt>
          <dd>Held · resumes only after a passing recheck</dd>
        </dl>
        <div className="note n">
          <span>
            Fixing a block changes the campaign, so the approval will be invalidated and re-requested with the diff attached. Fixing it outside the campaign (delisting the
            domain, bringing a provider back) keeps the approval intact.
          </span>
        </div>
        {children ? <div className="row gap-2!">{children}</div> : null}
      </div>
    </section>
  )
}

/** Fix links and buttons for the held card. */
export function HeldActions({ held, onApply, onCancel, busy }: { held: Schemas['ReviewHeld']; onApply: (b: Schemas['ReadinessCheck'], f: Schemas['ReadinessFix']) => void; onCancel?: () => void; busy?: string }) {
  return (
    <>
      {held.blocks.flatMap((b) =>
        b.fixes.map((f, i) =>
          f.kind === 'link' ? (
            <AppHref key={`${b.key}-${i}`} href={f.href ?? '#'} className="btn sm">
              {f.label}
            </AppHref>
          ) : (
            <Button key={`${b.key}-${i}`} size="sm" loading={busy === `${b.key}:${f.action}`} onClick={() => onApply(b, f)}>
              {f.label} (re-request approval)
            </Button>
          ),
        ),
      )}
      {onCancel ? (
        <Button size="sm" variant="danger" onClick={onCancel}>
          Cancel send
        </Button>
      ) : null}
    </>
  )
}

/** Emergency bypass (Owners): a reason, the warnings, the password. */
export function BypassDialog({
  open,
  onOpenChange,
  review,
  run,
  onConfirm,
}: {
  open: boolean
  onOpenChange: (v: boolean) => void
  review: Review
  run: Run
  onConfirm: (input: { reason: string; password?: string; acknowledged: string[] }) => Promise<string | undefined>
}) {
  const { data: me } = useMe()
  const [reason, setReason] = useState('')
  const [password, setPassword] = useState('')
  const [checked, setChecked] = useState<string[]>([])
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState<string>()
  const acks = acknowledgements(run)
  const others = review.approval.approvers.filter((p) => p.name !== me?.user.name)
  const rules = review.approval.rules.map((r) => r.short).join(' · ')
  const needsPassword = review.access.hasPassword
  const blocked = !reason.trim()
    ? 'A reason is required'
    : acks.some((x) => !checked.includes(x.key))
      ? 'Confirm each warning first'
      : needsPassword && !password
        ? 'Enter your password'
        : undefined
  const close = (v: boolean) => {
    if (busy) return
    if (!v) {
      setReason('')
      setPassword('')
      setChecked([])
      setError(undefined)
    }
    onOpenChange(v)
  }
  const submit = async () => {
    if (blocked || busy) return
    setBusy(true)
    setError(undefined)
    const err = await onConfirm({ reason: reason.trim(), password: needsPassword ? password : undefined, acknowledged: checked })
    setBusy(false)
    if (err) setError(err)
    else close(false)
  }
  return (
    <Dialog
      open={open}
      onOpenChange={close}
      dismissible={false}
      hideClose
      eyebrow={<span className="k">Dialog · Owner only · emergency bypass</span>}
      title="Send without approval?"
      description={
        <span className="text-[13.5px]">
          “{review.campaign.name}” · {count(run.estimates.audience.final)} recipients · rules matched: {rules}. {me?.user.name ? `${me.user.name}, you` : 'You'} can bypass the
          approval policy as Owner.
        </span>
      }
      footer={
        <>
          <Button onClick={() => close(false)}>Cancel</Button>
          <Button variant="primary" disabledReason={blocked} loading={busy} onClick={() => void submit()}>
            {review.schedule.mode === 'now' ? 'Send now without approval' : 'Schedule without approval'}
          </Button>
        </>
      }
    >
      <Field
        label={
          <>
            Reason <span className="k font-medium">· required</span>
          </>
        }
      >
        <Textarea
          rows={3}
          className="min-h-0"
          value={reason}
          maxLength={1000}
          placeholder="e.g. Customer-facing outage notice, approved verbally by Marcus at 09:12; he is unavailable to click"
          onChange={(e) => setReason(e.target.value)}
        />
      </Field>
      {acks.map((x) => (
        <label key={x.key} className="rv-chk">
          <input type="checkbox" checked={checked.includes(x.key)} onChange={(e) => setChecked((v) => (e.target.checked ? [...v, x.key] : v.filter((k) => k !== x.key)))} />
          <span>{x.text}</span>
        </label>
      ))}
      <div className="note w">
        <Badge tone="warn">Logged</Badge>
        <span>
          Recorded as <span className="mono">campaign.approval_bypassed</span> with your reason, IP and the campaign snapshot. Every approver is notified now
          {others.length ? `: ${approverNames(others, 'and')}` : ''}. Send Readiness still runs and a Blocked result still stops the send.
        </span>
      </div>
      {needsPassword ? (
        <Field label="Confirm with your password">
          <input className="in" type="password" autoComplete="current-password" value={password} onChange={(e) => setPassword(e.target.value)} />
        </Field>
      ) : null}
      {error ? (
        <div className="note b" role="alert">
          <span>{error}</span>
        </div>
      ) : null}
    </Dialog>
  )
}

function ago(d: string, now: Date): string {
  const m = Math.round((now.getTime() - new Date(d).getTime()) / 60_000)
  if (m < 1) return 'just now'
  if (m < 60) return `${m} ${plural(m, 'minute', 'minutes')} ago`
  const h = Math.round(m / 60)
  if (h < 24) return `${h} ${plural(h, 'hour', 'hours')} ago`
  const days = Math.round(h / 24)
  return days === 1 ? 'yesterday' : `${days} days ago`
}

