import { useQueryClient } from '@tanstack/react-query'
import { useNavigate } from '@tanstack/react-router'
import { useEffect, useId, useMemo, useRef, useState, type ReactNode } from 'react'
import { $api, ApiError, errorMessage, http, type Schemas } from '@/api/client'
import { Page } from '@/components/shell/Page'
import { Avatar } from '@/components/ui/Avatar'
import { Badge } from '@/components/ui/Badge'
import { Button, ButtonLink } from '@/components/ui/Button'
import { ConfirmDialog } from '@/components/ui/ConfirmDialog'
import { Dialog } from '@/components/ui/Dialog'
import { EmptyState, ErrorPanel, Loading, Skeleton } from '@/components/ui/Feedback'
import { Td, Th } from '@/components/ui/Table'
import { toast } from '@/components/ui/Toast'
import { ago, count, monthDayTime, orList } from '@/lib/format'
import { useNow } from '@/lib/hooks'
import { useCan, useMe, usePermissionReason, useWorkspace } from '@/lib/session'
import { AccessTable } from '@/features/access/parts'
import { HeldCard, HeldActions } from '@/features/review/ApprovalCard'
import { agoWords, countsShort, groupTitles, groupWords, linkParts, minutes, providerColor, verdictTitle, zonedWhen } from '@/features/review/logic'
import { PreviewLanguagePicker } from '@/features/languages/approvals'
import { AppHref, CheckBadge, whenText } from '@/features/review/parts'
import { TransactionalApprovalCard, TransactionalDecisionDetail, TransactionalDecisionNote } from '@/features/transactional-templates/ApprovalCard'
import { IssueApprovalCard, RecurringDigestBadge } from '@/features/digests/IssueParts'
import { im } from '@/features/digests/issueMessages'
import { useIssueInfo, type IssueInfo } from '@/features/digests/shared'
import { useT } from '@/i18n'
import { PlacementApprovalCard } from '@/features/placement/EntryPoints'
import { ApprovalStatusBadge, approverNames, decidedText, deciderFirst, DiffLines, requestTone, unchangedText, type Change } from './parts'
import '@/features/review/review.css'
import { ClientSignoffFacts, ClientSignoffIntro, ClientSignoffLine, ClientSignoffNote } from '@/features/agency/ClientApproval'

type Detail = Schemas['ApprovalDetail']
const detailPath = '/api/v1/workspaces/{workspace}/approvals/{approvalId}' as const

/** A5 §5.12 Approval review: what the request covers, the discussion and the decision. */
export function ApprovalPage({ approvalId }: { approvalId: string }) {
  const { slug } = useWorkspace()
  const q = $api.useQuery('get', detailPath, { params: { path: { workspace: slug, approvalId } } })
  const d = q.data
  return (
    <Page
      title={d?.name ?? 'Approval request'}
      documentTitle={d ? `${d.name} · Approvals` : 'Approvals'}
      breadcrumb={[
        { label: 'Campaigns', to: '/w/$workspace/campaigns', params: { workspace: slug } },
        { label: 'Approvals', to: '/w/$workspace/approvals', params: { workspace: slug } },
        { label: d?.name ?? 'Request' },
      ]}
    >
      {q.isPending ? (
        <DetailSkeleton />
      ) : q.isError ? (
        q.error instanceof ApiError && q.error.status === 404 ? (
          <EmptyState
            title="This approval request doesn't exist"
            action={
              <ButtonLink to="/w/$workspace/approvals" params={{ workspace: slug }}>
                Back to approvals
              </ButtonLink>
            }
          >
            It may have been withdrawn, or the link is from another workspace.
          </EmptyState>
        ) : (
          <ErrorPanel title="Couldn't load the approval request" actions={<Button onClick={() => q.refetch()}>Retry</Button>}>
            {errorMessage(q.error)} Nothing was changed.
          </ErrorPanel>
        )
      ) : (
        <DetailBody detail={q.data} />
      )}
    </Page>
  )
}

function DetailBody({ detail: d }: { detail: Detail }) {
  const { slug } = useWorkspace()
  const { data: me } = useMe()
  const qc = useQueryClient()
  const navigate = useNavigate()
  const [comment, setComment] = useState('')
  const [busy, setBusy] = useState<string>()
  const [ask, setAsk] = useState<'request_changes' | 'reject' | null>(null)
  const refresh = () =>
    Promise.all([
      qc.invalidateQueries({ queryKey: ['get', detailPath] }),
      qc.invalidateQueries({ queryKey: ['get', '/api/v1/workspaces/{workspace}/approvals'] }),
      qc.invalidateQueries({ queryKey: ['get', '/api/v1/workspaces/{workspace}/campaigns/{campaignId}/review'] }),
    ])

  const decide = async (decision: 'approve' | 'request_changes' | 'reject', text?: string) => {
    setBusy(decision)
    try {
      const { data } = await http.POST('/api/v1/workspaces/{workspace}/approvals/{approvalId}/decision', {
        params: { path: { workspace: slug, approvalId: d.id } },
        body: { decision, comment: (text ?? comment).trim() || undefined },
      })
      setComment('')
      const words = { approve: 'Approved', request_changes: 'Changes requested', reject: 'Rejected' }[decision]
      toast({
        kind: 'success',
        title: `${words} · ${d.name}`,
        detail: data && data.status === 'pending' ? `${data.given} of ${data.needed} approvals so far.` : `${d.author.name} was notified.`,
      })
      if (data && data.id !== d.id) {
        void navigate({ to: '/w/$workspace/approvals/$approvalId', params: { workspace: slug, approvalId: data.id }, replace: true })
      }
    } catch (e) {
      toast({ kind: 'error', title: "Couldn't record the decision", detail: errorMessage(e) })
      throw e
    } finally {
      setBusy(undefined)
      await refresh()
    }
  }

  const post = async () => {
    const body = comment.trim()
    if (!body) return
    setBusy('comment')
    try {
      await http.POST('/api/v1/workspaces/{workspace}/approvals/{approvalId}/comments', { params: { path: { workspace: slug, approvalId: d.id } }, body: { body } })
      setComment('')
    } catch (e) {
      toast({ kind: 'error', title: "Couldn't post the comment", detail: errorMessage(e) })
    } finally {
      setBusy(undefined)
      await refresh()
    }
  }

  const sendTest = async () => {
    if (!d.campaign || !me?.user.email) return
    setBusy('test')
    try {
      const { data } = await http.POST('/api/v1/workspaces/{workspace}/campaigns/{campaignId}/test', {
        params: { path: { workspace: slug, campaignId: d.campaign.id } },
        body: { to: [me.user.email] },
      })
      toast({ kind: 'success', title: `Test sent to ${data?.sent.join(', ') ?? me.user.email}`, detail: data?.provider ? `via ${data.provider}` : undefined })
    } catch (e) {
      toast({ kind: 'error', title: "Couldn't send the test", detail: errorMessage(e) })
    } finally {
      setBusy(undefined)
    }
  }

  const applyFix = async (check: Schemas['ReadinessCheck'], fix: Schemas['ReadinessFix']) => {
    if (!d.campaign) return
    setBusy(`${check.key}:${fix.action}`)
    try {
      const { data } = await http.POST('/api/v1/workspaces/{workspace}/campaigns/{campaignId}/readiness/fixes', {
        params: { path: { workspace: slug, campaignId: d.campaign.id } },
        body: { fixes: [{ action: fix.action!, params: fix.params }] },
      })
      toast({ kind: 'success', title: data?.applied[0] ?? 'Fix applied', detail: 'The campaign changed, so the approval restarts with the diff attached.' })
    } catch (e) {
      toast({ kind: 'error', title: "Couldn't apply the fix", detail: errorMessage(e) })
    } finally {
      setBusy(undefined)
      await refresh()
    }
  }

  const unschedule = async () => {
    if (!d.campaign) return
    try {
      await http.POST('/api/v1/workspaces/{workspace}/campaigns/{campaignId}/unschedule', { params: { path: { workspace: slug, campaignId: d.campaign.id } } })
      toast({ kind: 'success', title: `${d.name} won't go out`, detail: 'The send was cancelled; the campaign is a draft again.' })
    } catch (e) {
      toast({ kind: 'error', title: "Couldn't cancel the send", detail: errorMessage(e) })
    } finally {
      await refresh()
    }
  }

  const canEdit = useCan('campaigns:edit')
  // A recurring digest's issue: its posts, what changed since the last approved issue, "Approve issue #3".
  const issue = useIssueInfo(d.campaign?.id)
  const invalidated = d.status === 'invalidated'
  const others = [d.author, ...d.approvers].filter((p, i, all) => p.name !== me?.user.name && all.findIndex((x) => x.id === p.id) === i)
  return (
    <>
      <HeaderCard detail={d} me={me?.user.name} issue={issue} />
      <div className="ap-review">
        <div className="col gap-5!">
          {invalidated ? <InvalidatedCard detail={d} busy={busy} onApprove={() => void decide('approve').catch(() => undefined)} onChanges={() => setAsk('request_changes')} /> : null}
          {d.held ? (
            <HeldCard
              held={d.held}
              scheduledAt={d.campaign?.scheduledAt}
              approved={d.status === 'approved' && d.decidedAt ? `v${d.version} by ${d.decidedBy}, ${monthDayTime(d.decidedAt)} · still valid: nothing in the snapshot changed` : undefined}
              notified={`${d.author.name} (author), ${approverNames(deciderFirst(d.approvers, d.decidedBy), 'and')} (approvers) · in-app and email at ${whenText(d.held.since)}`}
            >
              <HeldActions held={d.held} busy={busy} onApply={(b, f) => void applyFix(b, f)} onCancel={canEdit ? () => void unschedule() : undefined} />
            </HeldCard>
          ) : null}
          {d.subjectType === 'automation' ? <AutomationCard detail={d} /> : null}
          {d.transactional ? <TransactionalApprovalCard snapshot={d.transactional} /> : null}
          {issue ? <IssueApprovalCard issue={issue} /> : null}
          {d.campaign ? <CampaignCard detail={d} busy={busy === 'test'} onTest={() => void sendTest()} /> : null}
          {d.readiness ? <ReadinessCard detail={d} busy={busy} onFix={(c, f) => void applyFix(c, f)} /> : null}
          {d.audience || d.route ? <AudienceRouteCard detail={d} /> : null}
          {d.campaign ? <PlacementApprovalCard campaignId={d.campaign.id} /> : null}
          <ScheduleNoteCard detail={d} />
          <DiscussionCard detail={d} comment={comment} setComment={setComment} busy={busy === 'comment'} onPost={() => void post()} others={others} />
        </div>
        <DecisionCard
          detail={d}
          issueNumber={issue?.number}
          busy={busy}
          onApprove={() => void decide('approve').catch(() => undefined)}
          onChanges={() => setAsk('request_changes')}
          onReject={() => setAsk('reject')}
        />
      </div>
      <ConfirmDialog
        open={ask !== null}
        onOpenChange={(v) => !v && setAsk(null)}
        tone={ask === 'reject' ? 'bad' : 'warn'}
        badge={ask === 'reject' ? 'Reject' : 'Request changes'}
        title={ask === 'reject' ? `Reject “${d.name}”?` : `Ask ${d.author.name} for changes?`}
        description={
          d.transactional ? (
            <TransactionalDecisionDetail kind={ask ?? 'request_changes'} author={d.author.name} />
          ) : ask === 'reject' ? (
            `${d.author.name} is told and the campaign goes back to a draft. Reject when it shouldn't go out at all; for fixes, request changes instead.`
          ) : (
            `The request goes back to ${d.author.name} with your comment. The campaign stays a draft until they ask again.`
          )
        }
        reason={{ label: ask === 'reject' ? 'Why reject it?' : 'What should change?', minLength: 3 }}
        confirmLabel={ask === 'reject' ? 'Reject' : 'Request changes'}
        confirmVariant={ask === 'reject' ? 'danger' : 'primary'}
        onConfirm={({ reason }) => decide(ask ?? 'request_changes', reason)}
      />
    </>
  )
}

function HeaderCard({ detail: d, me, issue }: { detail: Detail; me?: string; issue?: IssueInfo }) {
  const now = useNow()
  const tone = requestTone(d.status)
  const edge = { ok: 'h-ok', bad: 'h-bad', info: 'h-info', neutral: 'h-mut', purple: 'h-pur', warn: '' }[tone]
  const role = d.can.decide ? `you are reviewing as ${me ?? 'an approver'}` : d.can.isAuthor ? 'you are the author' : `you are viewing as ${me ?? 'a member'}`
  return (
    <section className={`card pad ap-head ${edge}`} aria-label="Request">
      <div className="col flex-[1_1_360px] gap-1!">
        <div className="row gap-2!">
          <span className="text-[18px] font-bold">{d.name}</span>
          {issue ? <RecurringDigestBadge issue={issue} /> : null}
          <ApprovalStatusBadge status={d.status} long />
        </div>
        {d.clientSignoff ? (
          // An agency's client signs off its own sends (agency accounts).
          <ClientSignoffLine detail={d} me={me} />
        ) : (
          <span className="k">
            Requested by {d.requestedBy.name} · {agoWords(d.requestedAt, now)} · {d.needed} {d.needed === 1 ? 'approval' : 'approvals'} needed from {orList(d.approvers.filter((p) => p.id !== d.requestedBy.id).map((p) => p.name))} · {role}
          </span>
        )}
      </div>
      {d.rules.length ? (
        <div className="row gap-2!">
          <span className="k">Rules matched</span>
          {d.rules.map((r) => (
            <Badge key={r.key} tone="neutral">
              {r.badge}
            </Badge>
          ))}
        </div>
      ) : null}
    </section>
  )
}

/** "Approved, then invalidated" (ApprovalStates): what happened and the diff. */
function InvalidatedCard({ detail: d, busy, onApprove, onChanges }: { detail: Detail; busy?: string; onApprove: () => void; onChanges: () => void }) {
  const now = useNow()
  const approved = d.decisions.filter((x) => x.decision === 'approve')
  const last = approved[approved.length - 1]
  const next = d.campaign?.currentVersion || d.version
  const notified = deciderFirst(d.approvers, last?.approver.name)
    .map((p) => p.name.split(' ')[0])
    .join(' and ')
  const scheduled = d.campaign?.scheduledAt ? 'the send stays scheduled but won’t start without a new approval' : 'the send won’t start without a new approval'
  return (
    <section className="card" aria-labelledby="ap-inval">
      <div className="sec">
        <div className="col gap-0.5!">
          <h2 className="h2" id="ap-inval">
            {last ? 'Approved, then invalidated' : 'Invalidated before a decision'}
          </h2>
          <span className="k">
            {d.name} · {d.invalidatedReason || 'the campaign changed'}
            {last ? ' after approval' : ''}
          </span>
        </div>
        <Badge tone="bad">Invalidated</Badge>
      </div>
      <div className="pad col gap-3!">
        <ol className="ap-steps">
          {last ? (
            <li>
              <b>
                Approved by {last.approver.name}, {decidedText(last.at, now)}
              </b>{' '}
              · snapshot v{d.version}
              {last.comment ? ` · comment: “${last.comment}”` : ''}
            </li>
          ) : (
            <li>
              <b>Requested by {d.requestedBy.name}</b> · snapshot v{d.version}
            </li>
          )}
          {d.changes.some((c) => c.field !== 'audience_count') ? (
            <li>
              {d.invalidatedBy || d.requestedBy.name} {editWords(d.changes)}
              {d.invalidatedAt ? `, ${decidedText(d.invalidatedAt, now)}` : ''} · v{next}
            </li>
          ) : (
            <li>The audience moved more than 10% since the request · {d.audience ? count(d.audience.current) : ''} contacts now</li>
          )}
          <li>
            <b>Approval invalidated{d.invalidatedAt ? `, ${decidedText(d.invalidatedAt, now)}` : ''}</b> · {notified} notified · {scheduled}
          </li>
        </ol>
        <span className="ap-label">
          What changed · v{d.version} → v{next}
        </span>
        <DiffLines changes={d.changes} labelled />
        <dl className="kv m-0 text-[12.5px]">
          <dt>Unchanged</dt>
          <dd>{unchangedText(d.changes, d.snapshot, d.audience ? { current: d.audience.current, changePct: d.audience.changePct } : undefined) || 'nothing else'}</dd>
        </dl>
        {d.can.decide ? (
          <div className="row gap-2!">
            <Button variant="primary" loading={busy === 'approve'} onClick={onApprove}>
              Approve v{next}
            </Button>
            <Button loading={busy === 'request_changes'} onClick={onChanges}>
              Request changes
            </Button>
          </div>
        ) : null}
      </div>
    </section>
  )
}

/** "edited the subject and replaced the link". */
function editWords(changes: Change[]): string {
  const words: string[] = []
  const links = changes.filter((c) => c.field === 'link').length
  for (const f of ['subject', 'preheader', 'content', 'sender', 'audience', 'route', 'schedule'] as const) {
    if (!changes.some((c) => c.field === f)) continue
    words.push(
      {
        subject: 'edited the subject',
        preheader: 'edited the preheader',
        content: 'edited the content',
        sender: 'changed the sender',
        audience: 'changed the audience',
        route: 'changed the route',
        schedule: 'moved the schedule',
      }[f],
    )
    if (f === 'subject' && links) words.push(links === 1 ? 'replaced the link' : `replaced ${links} links`)
  }
  if (links && !changes.some((c) => c.field === 'subject')) words.unshift(links === 1 ? 'replaced a link' : `replaced ${links} links`)
  return words.length <= 1 ? (words[0] ?? 'changed the campaign') : `${words.slice(0, -1).join(', ')} and ${words[words.length - 1]}`
}

function Sec({ id, title, caption, children, actions }: { id: string; title: string; caption?: string; children: ReactNode; actions?: ReactNode }) {
  return (
    <section className="card" aria-labelledby={id}>
      <div className="sec">
        <div className="flex min-w-0 flex-col gap-0.5">
          <h2 className="h2" id={id}>
            {title}
          </h2>
          {caption ? <span className="k">{caption}</span> : null}
        </div>
        {actions}
      </div>
      {children}
    </section>
  )
}

function CampaignCard({ detail: d, busy, onTest }: { detail: Detail; busy: boolean; onTest: () => void }) {
  const { slug } = useWorkspace()
  const now = useNow()
  const [open, setOpen] = useState(false)
  const [language, setLanguage] = useState<string>()
  const preview = $api.useQuery(
    'get',
    '/api/v1/workspaces/{workspace}/approvals/{approvalId}/preview',
    { params: { path: { workspace: slug, approvalId: d.id }, query: language ? { language } : {} } },
    { placeholderData: (prev) => prev },
  )
  const c = d.campaign!
  return (
    <Sec id="ap-campaign" title="Campaign" caption="Snapshot the approval covers">
      <div className="pad col gap-3!">
        <dl className="kv">
          <dt>Subject</dt>
          <dd className="font-semibold">{d.snapshot.subject || c.subject}</dd>
          <dt>Preheader</dt>
          <dd>{c.preheader || '—'}</dd>
          <dt>From</dt>
          <dd>{c.from || '—'}</dd>
          <dt>Version</dt>
          <dd>
            v{d.version}
            {c.editedBy && c.editedAt && c.version === d.version ? ` · edited by ${c.editedBy} ${ago(c.editedAt, now)}` : ''}
          </dd>
        </dl>
        <div className="ap-preview">
          {preview.data ? (
            <EmailThumb html={preview.data.html} title={`Preview of “${d.name}”, v${preview.data.version}`} />
          ) : preview.isError ? (
            <span className="k block py-6 text-center">The preview isn't available: {errorMessage(preview.error)}</span>
          ) : (
            <div className="ap-preview-mail flex flex-col gap-2 p-3.5" aria-hidden="true">
              <Skeleton w="40%" h={10} />
              <Skeleton w="100%" h={90} r={4} />
              <Skeleton w="80%" h={12} />
              <Skeleton w="90%" h={10} />
            </div>
          )}
        </div>
        <div className="row gap-2!">
          <PreviewLanguagePicker preview={preview.data} onChange={setLanguage} />
          <Button size="sm" onClick={() => setOpen(true)} disabledReason={preview.data ? undefined : 'The preview is loading.'}>
            Open full preview
          </Button>
          <Button size="sm" loading={busy} onClick={onTest}>
            Send me a test
          </Button>
        </div>
      </div>
      {open && preview.data ? (
        <Dialog open onOpenChange={setOpen} size="lg" title={preview.data.subject || d.name} description={preview.data.preheader || `Version ${preview.data.version}`}>
          <iframe title={`Full preview of “${d.name}”`} sandbox="allow-same-origin" srcDoc={withAppFonts(preview.data.html)} className="h-[70vh] w-full rounded-lg border border-line bg-white" />
        </Dialog>
      ) : null}
    </Sec>
  )
}

/** The app's IBM Plex @font-face rules, so the email renders in the fonts it names (as in the composer). */
function plexFontFaces(): string {
  const out: string[] = []
  for (const sheet of Array.from(document.styleSheets)) {
    let rules: CSSRuleList
    try {
      rules = sheet.cssRules
    } catch {
      continue
    }
    for (const r of Array.from(rules)) if (r instanceof CSSFontFaceRule && r.cssText.includes('IBM Plex')) out.push(r.cssText)
  }
  return out.join('\n')
}

/** The email's HTML with the app's Plex fonts available to it. */
function withAppFonts(html: string): string {
  const style = `<style>${plexFontFaces()}</style>`
  return html.includes('</head>') ? html.replace('</head>', `${style}</head>`) : style + html
}

/**
 * The email at 70%, as the board's 420 px frame shows a 600 px email. In a
 * narrower frame (tablets, phones) the email renders narrower at the same
 * scale, so it reflows (its own mobile layout) instead of shrinking, as the
 * board's preview does. Long emails are cut at 340 px.
 */
const THUMB_SCALE = 0.7

function EmailThumb({ html, title }: { html: string; title: string }) {
  const box = useRef<HTMLDivElement>(null)
  const frame = useRef<HTMLIFrameElement>(null)
  const [width, setWidth] = useState(600)
  const [height, setHeight] = useState(430)
  const doc = useMemo(() => withAppFonts(html), [html])
  useEffect(() => {
    const el = box.current
    if (!el) return
    const ro = new ResizeObserver(() => setWidth(Math.min(600, Math.round(el.clientWidth / THUMB_SCALE))))
    ro.observe(el)
    return () => ro.disconnect()
  }, [])
  // The body's own height: the document's is at least the frame's.
  const measure = () => {
    const body = frame.current?.contentDocument?.body
    if (body) setHeight(Math.max(120, body.scrollHeight))
  }
  // Measured again whenever the email's body changes size: it reflows when
  // the frame's width changes, and web fonts change its text's height.
  const watch = () => {
    const win = frame.current?.contentWindow as (Window & typeof globalThis) | null | undefined
    const body = frame.current?.contentDocument?.body
    if (!win || !body) return
    measure()
    new win.ResizeObserver(measure).observe(body)
  }
  const visible = Math.min(340, height * THUMB_SCALE)
  return (
    <div ref={box} className="ap-preview-mail" style={{ height: visible }}>
      <iframe
        ref={frame}
        title={title}
        sandbox="allow-same-origin"
        srcDoc={doc}
        tabIndex={-1}
        onLoad={watch}
        style={{ width, height, transform: `scale(${THUMB_SCALE})` }}
      />
    </div>
  )
}

function ReadinessCard({ detail: d, busy, onFix }: { detail: Detail; busy?: string; onFix: (c: Schemas['ReadinessCheck'], f: Schemas['ReadinessFix']) => void }) {
  const { slug } = useWorkspace()
  const now = useNow()
  const fixReason = usePermissionReason('campaigns:edit')
  const r = d.readiness!
  const issueGroups = new Set<string>(r.issues.map((x) => x.group))
  const clean = Object.keys(groupWords).filter((g) => !issueGroups.has(g))
  const cleanText = clean.length ? `${capital(listWords(clean.map((g) => groupWords[g]!)))} checks passed.` : 'Every other check passed.'
  const tone = r.verdict === 'blocked' ? 'bad' : r.verdict === 'warnings' ? 'warn' : 'ok'
  return (
    <Sec id="ap-readiness" title="Send Readiness" caption="With evidence; fixes change the campaign and restart the approval">
      <div className="pad col gap-2.5!">
        <div className="row gap-2.5!">
          <Badge tone={tone}>{verdictTitle(r.verdict)}</Badge>
          <span className="k">
            {countsShort(r)} · checked {ago(r.checkedAt, now)}, rechecked at send time
          </span>
        </div>
        <AccessTable caption="Send Readiness issues">
          <thead>
            <tr>
              <Th>Check</Th>
              <Th>Evidence</Th>
              <Th>Fix</Th>
            </tr>
          </thead>
          <tbody>
            {r.issues.map((x) => {
              const f = x.fixes[0]
              return (
                <tr key={x.key}>
                  <Td>
                    <span className="row gap-2!">
                      <CheckBadge status={x.status} />
                      <span>
                        {groupTitles[x.group]}
                        {x.label ? ` · ${x.label}` : ''}
                      </span>
                    </span>
                  </Td>
                  <Td>
                    <span className="wrap">
                      {linkParts(x.evidence || x.title).map((p, i) =>
                        p.link ? (
                          <span key={i} className="mono">
                            {p.text}
                          </span>
                        ) : (
                          p.text
                        ),
                      )}
                    </span>
                  </Td>
                  <Td>
                    {!f ? null : f.kind === 'link' ? (
                      <AppHref href={f.href ?? '#'} className="btn sm">
                        {f.label}
                      </AppHref>
                    ) : (
                      <Button size="sm" loading={busy === `${x.key}:${f.action}`} disabledReason={fixReason} onClick={() => onFix(x, f)}>
                        {f.action === 'exclude_sunset' ? 'Exclude them' : f.label}
                      </Button>
                    )}
                  </Td>
                </tr>
              )
            })}
            <tr>
              <Td>
                <span className="row gap-2!">
                  <CheckBadge status="pass" />
                  <span>{r.issues.length ? 'Everything else' : 'Every check'}</span>
                </span>
              </Td>
              <Td>
                <span className="wrap">{r.issues.length ? cleanText : 'Reputation, authentication, content, links, compliance, audience quality and routing checks passed.'}</span>
              </Td>
              <Td>
                {d.campaign ? (
                  <ButtonLink size="sm" to="/w/$workspace/campaigns/$campaignId/review" params={{ workspace: slug, campaignId: d.campaign.id }}>
                    All {r.total} checks
                  </ButtonLink>
                ) : null}
              </Td>
            </tr>
          </tbody>
        </AccessTable>
      </div>
    </Sec>
  )
}

function capital(s: string): string {
  return s ? s[0]!.toUpperCase() + s.slice(1) : s
}

function listWords(words: string[]): string {
  return words.length <= 1 ? (words[0] ?? '') : `${words.slice(0, -1).join(', ')} and ${words[words.length - 1]}`
}

function AudienceRouteCard({ detail: d }: { detail: Detail }) {
  const a = d.audience
  const r = d.route
  const change = a ? (Math.abs(a.changePct) < 0.05 ? 'No change since the request' : `${a.changePct > 0 ? '+' : '−'}${Math.abs(a.changePct).toFixed(1)}% since request`) : ''
  return (
    <Sec id="ap-audience" title="Audience and route">
      <div className="pad ap-ar">
        {a ? (
          <div className="col gap-1.5!">
            <span className="ap-label">Audience · {count(a.count)} after exclusions</span>
            <dl className="kv">
              <dt>Lists</dt>
              <dd>{a.lists.join(', ') || '—'}</dd>
              <dt>Excluded</dt>
              <dd>{a.excluded || '—'}</dd>
              <dt>Change</dt>
              <dd>{change} (limit for invalidation: 10%)</dd>
            </dl>
          </div>
        ) : null}
        {r ? (
          <div className="col min-w-0 gap-1.5!">
            <span className="ap-label">Route · {r.splitLabel}</span>
            <AccessTable caption="Route">
              <thead>
                <tr>
                  <Th>Provider</Th>
                  <Th>Recipients</Th>
                  <Th>Cost</Th>
                </tr>
              </thead>
              <tbody>
                {r.shares
                  .filter((s) => s.recipients > 0)
                  .map((s) => (
                    <tr key={s.providerId}>
                      <Td>
                        <span className="row gap-1.5!">
                          <span className="dot" style={{ color: providerColor[s.color] ?? providerColor.other }} aria-hidden="true" />
                          {providerName(s)}
                          {s.status === 'warming_up' ? <Badge tone="warn">{s.statusLabel}</Badge> : s.status === 'down' ? <Badge tone="bad">{s.statusLabel}</Badge> : null}
                        </span>
                      </Td>
                      <Td>{count(s.recipients)}</Td>
                      <Td>{s.cost === 0 && s.priceLabel === 'self-hosted' ? '$0 (self-hosted)' : `$${s.cost.toFixed(2)}`}</Td>
                    </tr>
                  ))}
              </tbody>
            </AccessTable>
          </div>
        ) : null}
      </div>
    </Sec>
  )
}

/** "Amazon SES us-east-1", "Postal · mta.acme.io". */
function providerName(s: Schemas['ApprovalShare']): string {
  const label = (s.label ?? '').replace(/^self-hosted · /, '')
  if (!label) return s.name
  return label.includes('.') ? `${s.name} · ${label}` : `${s.name} ${label}`
}

function ScheduleNoteCard({ detail: d }: { detail: Detail }) {
  const s = d.schedule
  const est = s ? ` · estimated ${s.minutes >= 60 ? minutes(s.minutes) : `${s.minutes} minutes`} at current rate limits` : ''
  const when = !s
    ? d.snapshot.scheduleText
    : s.mode === 'now'
      ? 'Send now'
      : s.at
        ? `${s.mode === 'best_time' ? 'Best time per recipient within 24 h of ' : ''}${zonedWhen(s.at, s.timezone)}`
        : 'Not scheduled yet'
  return (
    <Sec id="ap-schedule" title={d.subjectType === 'campaign' ? 'Schedule and note' : 'Note'}>
      <div className="pad col gap-3!">
        <dl className="kv">
          {d.subjectType === 'campaign' ? (
            <>
              <dt>Scheduled</dt>
              <dd>
                {when}
                {est}
              </dd>
              <dt>Auto-pause</dt>
              <dd>{s?.autoPause ?? '—'}</dd>
            </>
          ) : null}
          <dt>Changes since last approval</dt>
          <dd>{d.changesSince.length ? `${listWords(d.changesSince.map((c) => c.label.toLowerCase()))} changed since the previous request` : d.followsId ? 'Nothing changed since the previous request' : 'First request · nothing to compare'}</dd>
        </dl>
        {d.changesSince.length ? <DiffLines changes={d.changesSince} labelled /> : null}
        {d.note ? (
          <div className="note n items-start!">
            <Avatar name={d.requestedBy.name} color={d.requestedBy.color} size={32} fontSize={13} />
            <span className="col gap-0.5!">
              <span className="font-semibold">{d.requestedBy.name} · note for approvers</span>
              <span>{d.note}</span>
            </span>
          </div>
        ) : null}
      </div>
    </Sec>
  )
}

function AutomationCard({ detail: d }: { detail: Detail }) {
  const a = d.automation
  return (
    <Sec id="ap-automation" title="Automation" caption="The version the approval publishes">
      <div className="pad col gap-3!">
        <dl className="kv">
          <dt>Version</dt>
          <dd>v{a?.version ?? d.version}</dd>
          <dt>Contacts in flight</dt>
          <dd>{a?.migration || '—'}</dd>
        </dl>
        {a?.changes.length ? (
          <ul className="ap-list">
            {a.changes.map((c, i) => (
              <li key={i}>{c}</li>
            ))}
          </ul>
        ) : null}
      </div>
    </Sec>
  )
}

type Entry = { key: string; who: Schemas['ApprovalPerson']; at: string; text: ReactNode; body?: string }

function DiscussionCard({
  detail: d,
  comment,
  setComment,
  busy,
  onPost,
  others,
}: {
  detail: Detail
  comment: string
  setComment: (v: string) => void
  busy: boolean
  onPost: () => void
  others: Schemas['ApprovalPerson'][]
}) {
  const { data: me } = useMe()
  const now = useNow()
  const id = useId()
  const verb = { approve: 'approved', request_changes: 'requested changes', reject: 'rejected' } as const
  const entries: Entry[] = [
    ...d.comments.map((c) => ({ key: c.id, who: c.author, at: c.at, text: null, body: c.body })),
    ...d.decisions.map((x, i) => ({ key: `d${i}`, who: x.approver, at: x.at, text: ` · ${verb[x.decision]}`, body: x.comment || undefined })),
  ].sort((a, b) => new Date(a.at).getTime() - new Date(b.at).getTime())
  const user = me?.user
  return (
    <Sec id="ap-discussion" title="Discussion" caption="Visible to the author and approvers">
      <div className="pad col gap-3!">
        {entries.length === 0 && !d.can.comment ? <span className="k">No comments yet.</span> : null}
        {entries.map((e) => (
          <div key={e.key} className="row items-start! gap-2.5!">
            <Avatar name={e.who.name} color={e.who.color} size={32} fontSize={13} />
            <div className="col gap-0.5!">
              <span className="k">
                <b className="text-ink">{e.who.name}</b>
                {e.text} · {ago(e.at, now)}
              </span>
              {e.body ? <span>{e.body}</span> : null}
            </div>
          </div>
        ))}
        {d.can.comment && user ? (
          <div className="row items-start! gap-2.5!">
            <Avatar name={user.name || user.email} color={user.avatarColor} size={32} fontSize={13} />
            <div className="col flex-1 gap-1!">
              <label className="vh" htmlFor={id}>
                Comment
              </label>
              <textarea
                id={id}
                className="in min-h-0"
                rows={2}
                maxLength={4000}
                placeholder={others.length ? `Reply to ${listWords(others.map((p) => p.name.split(' ')[0]!))}…` : 'Add a comment…'}
                value={comment}
                onChange={(e) => setComment(e.target.value)}
              />
              <div className="row justify-end">
                <Button size="sm" loading={busy} disabledReason={comment.trim() ? undefined : 'Write a comment first.'} onClick={onPost}>
                  Post comment
                </Button>
              </div>
            </div>
          </div>
        ) : null}
      </div>
    </Sec>
  )
}

function DecisionCard({
  detail: d,
  issueNumber,
  busy,
  onApprove,
  onChanges,
  onReject,
}: {
  detail: Detail
  /** A recurring digest's issue: the button reads "Approve issue #3". */
  issueNumber?: number
  busy?: string
  onApprove: () => void
  onChanges: () => void
  onReject: () => void
}) {
  const { slug } = useWorkspace()
  const now = useNow()
  const t = useT()
  const open = d.status === 'pending'
  const reason = d.can.decide ? undefined : d.can.decideReason || 'Only approvers can decide.'
  const last = d.decisions[d.decisions.length - 1]
  return (
    <section className="card ap-sticky" aria-labelledby="ap-decision">
      <div className="sec">
        <h2 className="h2" id="ap-decision">
          {open || d.status === 'invalidated' ? 'Your decision' : 'Decision'}
        </h2>
        {d.clientSignoff ? <ClientSignoffIntro detail={d} /> : null}
      </div>
      <div className="pad col gap-2.5!">
        {open ? (
          <>
            <Button variant="primary" className="justify-center" loading={busy === 'approve'} disabledReason={reason} onClick={onApprove}>
              {issueNumber ? t(im.approveIssue, { number: issueNumber }) : 'Approve'}
            </Button>
            <Button className="justify-center" loading={busy === 'request_changes'} disabledReason={reason} onClick={onChanges}>
              Request changes
            </Button>
            <Button variant="danger" className="justify-center" loading={busy === 'reject'} disabledReason={reason} onClick={onReject}>
              Reject
            </Button>
            {d.transactional ? (
              <TransactionalDecisionNote author={d.author.name} />
            ) : d.clientSignoff ? (
              <ClientSignoffNote detail={d} />
            ) : (
              <div className="note n text-[12.5px]">
                <span>
                  Approving covers this exact snapshot. If {d.author.name.split(' ')[0]} edits the content, audience, route or schedule, or the audience moves more than 10%, the
                  approval is invalidated and you see the diff.
                </span>
              </div>
            )}
          </>
        ) : d.status === 'invalidated' ? (
          <div className="note n text-[12.5px]">
            <span>
              This request covered v{d.version}. {d.can.decide ? `Deciding on v${d.campaign?.currentVersion || d.version} opens a new request with the diff attached.` : 'A new request carries the diff to the approvers.'}
            </span>
          </div>
        ) : (
          <div className="col gap-1.5!">
            <span className="row gap-2!">
              <ApprovalStatusBadge status={d.status} />
              {d.decidedBy && d.decidedAt ? (
                <span className="k">
                  {d.decidedBy} · {decidedText(d.decidedAt, now)}
                </span>
              ) : null}
            </span>
            {last?.comment ? <span className="text-[13px]">“{last.comment}”</span> : null}
            {d.bypassReason ? <span className="text-[13px]">Sent without approval: “{d.bypassReason}”</span> : null}
            {d.campaign && d.can.isAuthor ? (
              <ButtonLink size="sm" className="self-start" to="/w/$workspace/campaigns/$campaignId/review" params={{ workspace: slug, campaignId: d.campaign.id }}>
                Open Review &amp; send
              </ButtonLink>
            ) : null}
          </div>
        )}
        {d.clientSignoff ? (
          <ClientSignoffFacts detail={d} />
        ) : (
          <dl className="kv m-0 text-[12.5px]">
            <dt>Author</dt>
            <dd>
              {d.author.name}
              {d.author.role ? ` · ${d.author.role}` : ''}
            </dd>
            <dt>Approvers</dt>
            <dd>{approverNames(d.approvers, 'comma') || '—'}</dd>
            <dt>Needed</dt>
            <dd>
              {Math.max(0, d.needed - d.given)} of {d.needed}
            </dd>
            <dt>Requested</dt>
            <dd>{monthDayTime(d.requestedAt)}</dd>
          </dl>
        )}
      </div>
    </section>
  )
}

function DetailSkeleton() {
  return (
    <Loading label="Loading the approval request" className="flex flex-col gap-5">
      <div className="card pad ap-head">
        <div className="col flex-[1_1_360px] gap-2!">
          <Skeleton w={260} h={18} />
          <Skeleton w="60%" h={10} />
        </div>
        <Skeleton w={300} h={22} r={999} />
      </div>
      <div className="ap-review">
        <div className="col gap-5!">
          {[4, 3, 3].map((n, i) => (
            <div key={i} className="card">
              <div className="sec">
                <Skeleton w={140} h={14} />
              </div>
              <div className="pad col gap-3!">
                {Array.from({ length: n }, (_, j) => (
                  <Skeleton key={j} w={`${70 - j * 10}%`} h={12} />
                ))}
              </div>
            </div>
          ))}
        </div>
        <div className="card pad col gap-2.5!">
          <Skeleton w="100%" h={36} r={8} />
          <Skeleton w="100%" h={36} r={8} />
          <Skeleton w="100%" h={36} r={8} />
        </div>
      </div>
    </Loading>
  )
}
