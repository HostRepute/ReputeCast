import { useState, type ReactNode } from 'react'
import type { Schemas } from '@/api/client'
import { Badge } from '@/components/ui/Badge'
import { Button, ButtonLink } from '@/components/ui/Button'
import { Dialog } from '@/components/ui/Dialog'
import { Field, Textarea } from '@/components/ui/Field'
import { formatList, T, t as tr, useT } from '@/i18n'
import { common } from '@/i18n/common'
import { monthDayTime } from '@/lib/format'
import { useNow } from '@/lib/hooks'
import { useMe, useWorkspace } from '@/lib/session'
import { approverNames, DiffLines, MiniBanner, PersonLine, unchangedSentence } from '@/features/approvals/parts'
import { acknowledgedKeys, acknowledgements, agoWords, countsShort, linkParts, scheduleShort, statusWord, verdictTitle, waitWords, type Review, type Run } from './logic'
import { m } from './messages'
import { AppHref, whenText } from './parts'

export type ApprovalMode = 'none' | 'required' | 'waiting' | 'approved' | 'invalidated'

/**
 * Which approval card Review & send shows (SendReadinessApproval): none when
 * no rule matches or a valid approval lets the viewer send.
 */
export function approvalMode(r: Review): ApprovalMode {
  const a = r.approval
  if (r.campaign.status === 'scheduled') {
    // A scheduled send keeps its time. It needs the approval card only when
    // a request made for it doesn't cover it: the send won't start then.
    const req = a.request
    if (req?.status === 'pending') return 'waiting'
    if (req?.status === 'invalidated') return 'invalidated'
    const settled = req && ['changes_requested', 'rejected', 'cancelled'].includes(req.status)
    return settled && r.scheduled && new Date(req.requestedAt) > new Date(r.scheduled.at) ? 'required' : 'none'
  }
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
  const t = useT()
  return (
    <section className="card" aria-labelledby={id}>
      <div className="sec">
        <div className="col gap-0.5!">
          <h2 className="h2" id={id}>
            {title}
          </h2>
          <span className="k">{t(m.shellCaption, { name: review.campaign.name, recipients: run.estimates.audience.final })}</span>
        </div>
      </div>
      <div className="pad col gap-3.5!">{children}</div>
    </section>
  )
}

function Foot({ children }: { children: ReactNode }) {
  return <div className="row ap-foot">{children}</div>
}

/**
 * A matched rule as Review & send words it: the author's role and who can
 * approve, never a permission key ("Your role, Content Writer, needs an
 * approval to send: Marcus Chen or Linh Nguyen can approve").
 */
function ruleText(rule: Schemas['ApprovalRuleMatch'], approvers: string[], me?: string): string {
  if (rule.key === 'author_lacks_send') {
    const names = approvers.length ? formatList(approvers, 'disjunction') : 'none'
    const mine = !rule.author || rule.author === me
    return tr(m.ruleNeedsApproval, { mine: mine ? 'yes' : 'no', author: rule.author ?? '', hasRole: rule.role ? 'yes' : 'no', role: rule.role ?? '', names })
  }
  // The other rules come from the server, as written there.
  return rule.label
}

/** Approval required: the rules that matched, a note, Request approval. */
export function RequiredCard({
  review,
  run,
  busy,
  onRequest,
  more,
  onSaveDraft,
}: {
  review: Review
  run: Run
  busy: boolean
  onRequest: (note: string) => void
  /** More: the request's link, the policy and the emergency send (MoreMenu). */
  more?: ReactNode
  onSaveDraft: (note: string) => void
}) {
  const t = useT()
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
    <Shell review={review} run={run} title={t(m.requiredTitle)} id="rv-approval">
      <MiniBanner tone="warn" title={verdictTitle(run.verdict)}>
        {countsShort(run.counts)}
      </MiniBanner>
      {prev ? (
        <div className="note i" role="status">
          <Badge tone="info">{prev.status === 'rejected' ? t(m.badgeRejected) : t(m.badgeChanges)}</Badge>
          <span>
            {t(
              prev.status === 'rejected' ? (prev.comment ? m.prevRejectedComment : m.prevRejected) : prev.comment ? m.prevChangesComment : m.prevChanges,
              { name: prev.decidedBy || t(m.anApprover), version: prev.version, comment: prev.comment ?? '' },
            )}
          </span>
        </div>
      ) : null}
      <div className="card ap-inner i-warn">
        <div className="row gap-2!">
          <span className="font-semibold">{t(m.requiredBefore)}</span>
          <Badge tone="warn">{t(m.rulesMatched, { count: a.rules.length })}</Badge>
        </div>
        <ul className="ap-list">
          {a.rules.map((r) => (
            <li key={r.key}>{ruleText(r, approvers.map((p) => p.name), myName)}</li>
          ))}
        </ul>
        <Field label={t(m.noteForApprovers)} optional>
          <Textarea rows={2} className="min-h-0" value={note} maxLength={2000} onChange={(e) => keep(e.target.value)} />
        </Field>
        <span className="k">
          {t(m.approversLine, { names: approverNames(approvers, 'comma') || t(m.noneYet), needed: a.needed, author: isAuthor ? 'yes' : 'no' })}
        </span>
      </div>
      <Foot>
        <Button onClick={() => onSaveDraft(note)}>{t(m.saveDraft)}</Button>
        {more}
        <Button variant="primary" disabledReason={a.canRequest ? undefined : a.requestReason || t(m.cantRequest)} loading={busy} onClick={() => onRequest(note.trim())}>
          {t(m.requestApproval)}
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
  more,
}: {
  review: Review
  run: Run
  busy?: 'remind' | 'cancel'
  onRemind: () => void
  onCancel: () => void
  /** More: the request's link, the policy and the emergency send (MoreMenu). */
  more?: ReactNode
}) {
  const t = useT()
  const { slug } = useWorkspace()
  const { data: me } = useMe()
  const now = useNow()
  const a = review.approval
  const req = a.request!
  const mine = req.requestedBy.name === me?.user.name
  return (
    <Shell review={review} run={run} title={t(m.waitingTitle)} id="rv-approval">
      <MiniBanner tone="warn" title={verdictTitle(run.verdict)}>
        {t(m.snapshotLocked, { version: req.version })}
      </MiniBanner>
      <div className="card ap-inner">
        <div className="row gap-2.5!">
          <Badge tone="warn">{t(m.pending)}</Badge>
          <span className="font-semibold">
            {t(m.requestedLine, { when: agoWords(req.requestedAt, now), given: req.given, needed: req.needed, mine: mine ? 'yes' : 'no', name: req.requestedBy.name })}
          </span>
        </div>
        <div className="row gap-3!">
          {a.approvers
            .filter((p) => p.id !== req.requestedBy.id)
            .map((p) => (
              <PersonLine key={p.id} person={p} line={t(m.approverLine, { presence: p.presence ?? t(m.away) })} />
            ))}
        </div>
        <div className="note n text-[12.5px]">
          <span>
            {t(m.restartsNote, { hasWait: a.medianWaitMinutes != null ? 'yes' : 'no', wait: a.medianWaitMinutes != null ? waitWords(a.medianWaitMinutes) : '' })}
          </span>
        </div>
      </div>
      <Foot>
        {mine ? (
          <>
            <ButtonLink to="/w/$workspace/campaigns/$campaignId/edit" params={{ workspace: slug, campaignId: review.campaign.id }}>
              {t(m.editRestarts)}
            </ButtonLink>
            <Button loading={busy === 'remind'} onClick={onRemind}>
              {t(m.remind)}
            </Button>
            <Button variant="danger" loading={busy === 'cancel'} onClick={onCancel}>
              {t(m.cancelRequest)}
            </Button>
            {more}
          </>
        ) : (
          <>
            {more}
            {review.access.isOwner ? (
              <Button variant="danger" loading={busy === 'cancel'} onClick={onCancel}>
                {t(m.cancelRequest)}
              </Button>
            ) : null}
            <ButtonLink variant="primary" to="/w/$workspace/approvals/$approvalId" params={{ workspace: slug, approvalId: req.id }}>
              {review.access.canApprove ? t(m.reviewRequest) : t(m.openRequest)}
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
  const t = useT()
  const now = useNow()
  const req = review.approval.request!
  const snap = req.snapshot
  const decided = req.decidedAt ? whenText(req.decidedAt, now) : ''
  // Who can, by name, never a permission key.
  const helpers = review.access.editHelpers
  const reason = review.access.canEdit ? undefined : t(m.needsEdit, { names: helpers.length ? formatList(helpers, 'disjunction') : 'none' })
  return (
    <Shell review={review} run={run} title={t(m.approvedTitle)} id="rv-approval">
      <MiniBanner tone="ok" title={t(m.approvedBanner)}>
        {t(m.approvedBy, {
          name: req.decidedBy || t(m.anApproverLower),
          hasWhen: decided ? 'yes' : 'no',
          when: decided,
          hasComment: req.comment ? 'yes' : 'no',
          comment: req.comment ?? '',
        })}
      </MiniBanner>
      <div className="card ap-inner">
        <dl className="kv m-0 text-[13px]">
          <dt>{t(m.approvedSnapshot)}</dt>
          <dd>
            {t(m.snapshotValue, {
              version: req.version,
              audience: snap.audienceCount,
              hasRoute: snap.routeLabel ? 'yes' : 'no',
              route: snap.routeLabel ?? '',
              mode: snap.scheduleMode === 'now' ? 'now' : 'at',
              schedule: snap.scheduleText ?? '',
            })}
          </dd>
          <dt>{t(m.validUntil)}</dt>
          <dd>{t(m.validUntilValue)}</dd>
          <dt>{t(m.atSendTime)}</dt>
          <dd>{t(m.atSendTimeValue)}</dd>
        </dl>
        {req.audit ? (
          <div className="note g text-[12.5px]">
            <Badge tone="ok">{t(m.logged)}</Badge>
            <span>
              {t(m.auditLine, { action: req.audit.action, actor: req.audit.actor, when: whenText(req.audit.at, now), hasIp: req.audit.ip ? 'yes' : 'no', ip: req.audit.ip ?? '' })}
            </span>
          </div>
        ) : null}
      </div>
      <Foot>
        <Button loading={busy === 'test'} onClick={onTest}>
          {t(m.sendTest)}
        </Button>
        <Button variant="primary" disabledReason={reason} onClick={onSchedule}>
          {review.schedule.mode === 'now' ? t(m.sendNowButton) : t(m.scheduleFor, { when: scheduleShort(review.schedule) })}
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
  const t = useT()
  const now = useNow()
  const { data: me } = useMe()
  const req = review.approval.request!
  const who = req.invalidatedBy && req.invalidatedBy === me?.user.name ? t(m.you) : req.invalidatedBy || t(m.someone)
  // The server names the edited fields in the reader's language; empty when the reason isn't about fields.
  const fields = req.invalidatedFields || t(m.theCampaign)
  const moved = req.changes.some((c) => c.field === 'audience_count')
  const approvedBy = req.decidedBy ? t(m.coveredBy, { name: req.decidedBy, version: req.version }) : t(m.coveredRequest, { version: req.version })
  const edit = moved
    ? t(m.audienceMoved)
    : t(m.editedFields, {
        who,
        fields,
        hasAt: req.invalidatedAt ? 'yes' : 'no',
        at: req.invalidatedAt ? whenText(req.invalidatedAt, now) : '',
        version: review.campaign.version,
      })
  return (
    <Shell review={review} run={run} title={t(m.invalidatedTitle)} id="rv-approval">
      <MiniBanner tone="bad" title={t(m.invalidatedBanner, { reason: req.invalidatedReason || t(m.campaignChanged) })}>
        {approvedBy} {edit}
      </MiniBanner>
      <div className="card ap-inner">
        {req.changes.length ? <DiffLines changes={req.changes} /> : null}
        <span className="k">
          {[unchangedSentence(req.changes), t(m.invalidatedNote, { scheduled: review.campaign.scheduledAt ? 'yes' : 'no' })].filter(Boolean).join(' ')}
        </span>
      </div>
      <Foot>
        {onRestore ? (
          <Button loading={busy === 'restore'} onClick={onRestore}>
            {t(m.restore, { version: req.version, what: restoreWhat(req.changes) })}
          </Button>
        ) : null}
        <Button variant="primary" loading={busy === 'request'} disabledReason={review.approval.canRequest ? undefined : review.approval.requestReason} onClick={onRequest}>
          {t(m.requestAgain)}
        </Button>
      </Foot>
    </Shell>
  )
}

/** What Restore puts back when it's one thing: "subject", "link"; "all" otherwise. */
function restoreWhat(changes: Schemas['ApprovalChange'][]): string {
  const f = [...new Set(changes.map((c) => (c.field === 'link' ? 'link' : c.field)))]
  if (f.length === 1 && ['subject', 'preheader', 'schedule', 'audience', 'link'].includes(f[0]!)) return f[0]!
  return 'all'
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
  const t = useT()
  return (
    <section className="card border-bad-line" aria-labelledby="rv-held">
      <div className="sec">
        <div className="col gap-0.5!">
          <h2 className="h2" id="rv-held">
            {t(m.heldTitle)}
          </h2>
          <span className="k">{scheduledAt ? t(m.heldCaption, { when: monthDayTime(scheduledAt) }) : t(m.heldCaptionNoTime)}</span>
        </div>
        <Badge tone="bad">{verdictTitle('blocked')}</Badge>
      </div>
      <div className="pad col gap-3!">
        {held.blocks.length ? (
          held.blocks.map((b) => {
            const [group, ...rest] = (b.flatTitle || b.title).split(': ')
            return (
              <div key={b.key} className="note b" role="alert">
                <Badge tone="bad">{statusWord('block')}</Badge>
                <span>
                  {rest.length ? (
                    <>
                      <b>{group}:</b> <Mono text={`${rest.join(': ')}.`} />
                    </>
                  ) : (
                    <Mono text={`${b.title}.`} />
                  )}{' '}
                  {heldSentence(b.detail)}
                </span>
              </div>
            )
          })
        ) : (
          <div className="note b" role="alert">
            <Badge tone="bad">{statusWord('block')}</Badge>
            <span>{t(m.heldReason, { reason: held.reason })}</span>
          </div>
        )}
        <dl className="kv m-0 text-[12.5px]">
          {approved ? (
            <>
              <dt>{t(m.approved)}</dt>
              <dd>{approved}</dd>
            </>
          ) : null}
          {notified ? (
            <>
              <dt>{t(m.notified)}</dt>
              <dd>{notified}</dd>
            </>
          ) : null}
          <dt>{t(m.schedule)}</dt>
          <dd>{t(m.heldSchedule)}</dd>
        </dl>
        <div className="note n">
          <span>{heldNote(held.blocks)}</span>
        </div>
        {children ? <div className="row gap-2!">{children}</div> : null}
      </div>
    </section>
  )
}

/** "Sending would hurt inbox placement for every recipient, so the approved send was held." */
export function heldSentence(detail?: string): string {
  const d = detail?.trim().replace(/[.\s]+$/, '')
  return d ? tr(m.heldSentence, { detail: d }) : tr(m.heldPlain)
}

/** What fixing the blocks does to the approval, for the held card. */
export function heldNote(blocks: Schemas['ReadinessCheck'][]): string {
  const groups = new Set(blocks.map((b) => b.group))
  if (groups.size === 1 && groups.has('links')) return tr(m.heldLinks)
  if (groups.size === 1 && groups.has('routing')) return tr(m.heldRouting)
  return tr(m.heldOther)
}

/** Text with its web addresses set in mono. */
function Mono({ text }: { text: string }) {
  return (
    <>
      {linkParts(text).map((p, i) =>
        p.link ? (
          <span key={i} className="mono">
            {p.text}
          </span>
        ) : (
          p.text
        ),
      )}
    </>
  )
}

/** Fix links and buttons for the held card. */
export function HeldActions({ held, onApply, onCancel, busy }: { held: Schemas['ReviewHeld']; onApply: (b: Schemas['ReadinessCheck'], f: Schemas['ReadinessFix']) => void; onCancel?: () => void; busy?: string }) {
  const t = useT()
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
              {t(m.reRequest, { label: f.label })}
            </Button>
          ),
        ),
      )}
      {onCancel ? (
        <Button size="sm" variant="danger" onClick={onCancel}>
          {t(m.cancelSend)}
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
  const t = useT()
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
    ? t(m.reasonRequired)
    : acks.some((x) => !checked.includes(x.key))
      ? t(m.confirmEach)
      : needsPassword && !password
        ? t(m.enterPassword)
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
    const err = await onConfirm({ reason: reason.trim(), password: needsPassword ? password : undefined, acknowledged: acknowledgedKeys(acks, checked) })
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
      eyebrow={<span className="k">{t(m.bypassEyebrow)}</span>}
      title={t(m.bypassTitle)}
      description={
        <span className="text-[13.5px]">
          {t(m.bypassDescription, {
            campaign: review.campaign.name,
            recipients: run.estimates.audience.final,
            rules,
            hasName: me?.user.name ? 'yes' : 'no',
            name: me?.user.name ?? '',
          })}
        </span>
      }
      footer={
        <>
          <Button onClick={() => close(false)}>{t(common.cancel)}</Button>
          <Button variant="primary" disabledReason={blocked} loading={busy} onClick={() => void submit()}>
            {review.schedule.mode === 'now' ? t(m.sendNowWithout) : t(m.scheduleWithout)}
          </Button>
        </>
      }
    >
      <Field
        label={
          <>
            {t(m.reason)} <span className="k font-medium">· {t(m.requiredLower)}</span>
          </>
        }
      >
        <Textarea
          rows={3}
          className="min-h-0"
          value={reason}
          maxLength={1000}
          placeholder={t(m.reasonPlaceholder)}
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
        <Badge tone="warn">{t(m.logged)}</Badge>
        <span>
          <T
            msg={m.bypassLogged}
            values={{ hasOthers: others.length ? 'yes' : 'no', others: approverNames(others, 'and'), mono: (c) => <span className="mono">{c}</span> }}
          />
        </span>
      </div>
      {needsPassword ? (
        <Field label={t(m.confirmPassword)}>
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


