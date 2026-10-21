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
import { ago, andList, count, money, monthDayTime, orList, pct } from '@/lib/format'
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
import { T, t as tt, useT } from '@/i18n'
import { common } from '@/i18n/common'
import { roleName } from '@/lib/roles'
import { m as rm } from '@/features/review/messages'
import { PlacementApprovalCard } from '@/features/placement/EntryPoints'
import { PageApprovalCard, PageBadge, PageDecisionDetail, PageDecisionNote } from '@/features/pages/approval'
import { ApprovalStatusBadge, approverNames, decidedText, deciderFirst, DiffLines, requestTone, statusWord, unchangedText, type Change } from './parts'
import { mr } from './review.messages'
import '@/features/review/review.css'
import { PostApprovalCard } from '@/features/website/elsewhere'
import { SmsApprovalCard } from '@/features/sms/ApprovalCard'
import { PushApprovalCard, WhatsAppApprovalCard } from '@/features/push/ApprovalCards'
import { ClientSignoffFacts, ClientSignoffIntro, ClientSignoffLine, ClientSignoffNote } from '@/features/agency/ClientApproval'

type Detail = Schemas['ApprovalDetail']
const detailPath = '/api/v1/workspaces/{workspace}/approvals/{approvalId}' as const

/** A5 §5.12 Approval review: what the request covers, the discussion and the decision. */
export function ApprovalPage({ approvalId }: { approvalId: string }) {
  const t = useT()
  const { slug } = useWorkspace()
  const q = $api.useQuery('get', detailPath, { params: { path: { workspace: slug, approvalId } } })
  const d = q.data
  return (
    <Page
      title={d?.name ?? t(mr.titleFallback)}
      documentTitle={d ? t(mr.documentTitle, { name: d.name }) : t(mr.documentTitleFallback)}
      breadcrumb={[
        { label: t(mr.crumbCampaigns), to: '/w/$workspace/campaigns', params: { workspace: slug } },
        { label: t(mr.crumbApprovals), to: '/w/$workspace/approvals', params: { workspace: slug } },
        { label: d?.name ?? t(mr.crumbRequest) },
      ]}
    >
      {q.isPending ? (
        <DetailSkeleton />
      ) : q.isError ? (
        q.error instanceof ApiError && q.error.status === 404 ? (
          <EmptyState
            title={t(mr.notFoundTitle)}
            action={
              <ButtonLink to="/w/$workspace/approvals" params={{ workspace: slug }}>
                {t(mr.backToApprovals)}
              </ButtonLink>
            }
          >
            {t(mr.notFoundBody)}
          </EmptyState>
        ) : (
          <ErrorPanel title={t(mr.loadFailed)} actions={<Button onClick={() => q.refetch()}>{t(common.retry)}</Button>}>
            {errorMessage(q.error)} {t(mr.nothingChanged)}
          </ErrorPanel>
        )
      ) : (
        <DetailBody detail={q.data} />
      )}
    </Page>
  )
}

function DetailBody({ detail: d }: { detail: Detail }) {
  const t = useT()
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
      toast({
        kind: 'success',
        title: t(mr.decided, { decision, name: d.name }),
        detail: data && data.status === 'pending' ? t(mr.decidedProgress, { given: data.given, needed: data.needed }) : t(mr.decidedNotified, { name: d.author.name }),
      })
      if (data && data.id !== d.id) {
        void navigate({ to: '/w/$workspace/approvals/$approvalId', params: { workspace: slug, approvalId: data.id }, replace: true })
      }
    } catch (e) {
      toast({ kind: 'error', title: t(mr.decideFailed), detail: errorMessage(e) })
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
      toast({ kind: 'error', title: t(mr.commentFailed), detail: errorMessage(e) })
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
      toast({
        kind: 'success',
        title: t(mr.testSent, { to: data ? andList(data.sent) : me.user.email }),
        detail: data?.provider ? t(mr.testVia, { provider: data.provider }) : undefined,
      })
    } catch (e) {
      toast({ kind: 'error', title: t(mr.testFailed), detail: errorMessage(e) })
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
      toast({ kind: 'success', title: data?.applied[0] ?? t(mr.fixApplied), detail: t(mr.fixAppliedDetail) })
    } catch (e) {
      toast({ kind: 'error', title: t(mr.fixFailed), detail: errorMessage(e) })
    } finally {
      setBusy(undefined)
      await refresh()
    }
  }

  const unschedule = async () => {
    if (!d.campaign) return
    try {
      await http.POST('/api/v1/workspaces/{workspace}/campaigns/{campaignId}/unschedule', { params: { path: { workspace: slug, campaignId: d.campaign.id } } })
      toast({ kind: 'success', title: t(mr.unscheduled, { name: d.name }), detail: t(mr.unscheduledDetail) })
    } catch (e) {
      toast({ kind: 'error', title: t(mr.unscheduleFailed), detail: errorMessage(e) })
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
              approved={
                d.status === 'approved' && d.decidedAt
                  ? t(rm.heldApproved, { version: d.version, name: d.decidedBy ?? '', when: monthDayTime(d.decidedAt), valid: 'yes' })
                  : undefined
              }
              notified={t(rm.heldNotified, {
                people: [t(rm.heldAuthor, { name: d.author.name }), t(rm.heldApprovers, { names: approverNames(deciderFirst(d.approvers, d.decidedBy), 'and') })].join(', '),
                time: whenText(d.held.since),
              })}
            >
              <HeldActions held={d.held} busy={busy} onApply={(b, f) => void applyFix(b, f)} onCancel={canEdit ? () => void unschedule() : undefined} />
            </HeldCard>
          ) : null}
          {d.subjectType === 'automation' ? <AutomationCard detail={d} /> : null}
          {/* A website post (features/website): its snapshot and its page on the web. */}
          {d.subjectType === 'website_post' && d.website ? <PostApprovalCard approvalId={d.id} name={d.name} snapshot={d.website} /> : null}
          {d.transactional ? <TransactionalApprovalCard snapshot={d.transactional} /> : null}
          {d.page ? <PageApprovalCard detail={d} /> : null}
          {issue ? <IssueApprovalCard issue={issue} /> : null}
          {/* Texts (SMS, X21): the text, its length and the cost by country. */}
          {d.channel === 'sms' && d.sms ? (
            <SmsApprovalCard sms={d.sms} />
          ) : d.channel === 'push' && d.push ? (
            // Push and WhatsApp (X22): the notification, or the template as it reads.
            <PushApprovalCard push={d.push} />
          ) : d.channel === 'whatsapp' && d.whatsapp ? (
            <WhatsAppApprovalCard wa={d.whatsapp} />
          ) : d.campaign ? (
            <CampaignCard detail={d} busy={busy === 'test'} onTest={() => void sendTest()} />
          ) : null}
          {d.readiness ? <ReadinessCard detail={d} busy={busy} onFix={(c, f) => void applyFix(c, f)} /> : null}
          {d.audience || d.route ? <AudienceRouteCard detail={d} /> : null}
          {d.campaign && d.channel !== 'sms' && d.channel !== 'push' && d.channel !== 'whatsapp' ? <PlacementApprovalCard campaignId={d.campaign.id} /> : null}
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
        badge={ask === 'reject' ? t(mr.rejectBadge) : t(mr.changesBadge)}
        title={ask === 'reject' ? t(mr.rejectTitle, { name: d.name }) : t(mr.changesTitle, { name: d.author.name })}
        description={
          d.page ? (
            <PageDecisionDetail kind={ask ?? 'request_changes'} author={d.author.name} />
          ) : d.transactional ? (
            <TransactionalDecisionDetail kind={ask ?? 'request_changes'} author={d.author.name} />
          ) : ask === 'reject' ? (
            t(mr.rejectDescription, { name: d.author.name })
          ) : (
            t(mr.changesDescription, { name: d.author.name })
          )
        }
        reason={{ label: ask === 'reject' ? t(mr.rejectReason) : t(mr.changesReason), minLength: 3 }}
        confirmLabel={ask === 'reject' ? t(mr.rejectConfirm) : t(mr.changesConfirm)}
        confirmVariant={ask === 'reject' ? 'danger' : 'primary'}
        onConfirm={({ reason }) => decide(ask ?? 'request_changes', reason)}
      />
    </>
  )
}

function HeaderCard({ detail: d, me, issue }: { detail: Detail; me?: string; issue?: IssueInfo }) {
  const t = useT()
  const now = useNow()
  const tone = requestTone(d.status)
  const edge = { ok: 'h-ok', bad: 'h-bad', info: 'h-info', neutral: 'h-mut', purple: 'h-pur', warn: '' }[tone]
  const role = d.can.decide ? (me ? 'reviewing' : 'reviewingAnon') : d.can.isAuthor ? 'author' : me ? 'viewing' : 'viewingAnon'
  return (
    <section className={`card pad ap-head ${edge}`} aria-label={t(mr.headLabel)}>
      <div className="col flex-[1_1_360px] gap-1!">
        <div className="row gap-2!">
          <span className="text-[18px] font-bold">{d.name}</span>
          {issue ? <RecurringDigestBadge issue={issue} /> : null}
          {d.page ? <PageBadge page={d.page} /> : null}
          <ApprovalStatusBadge status={d.status} long />
        </div>
        {d.clientSignoff ? (
          // An agency's client signs off its own sends (agency accounts).
          <ClientSignoffLine detail={d} me={me} />
        ) : (
          <span className="k">
            {t(mr.requestedLine, {
              name: d.requestedBy.name,
              ago: agoWords(d.requestedAt, now),
              needed: d.needed,
              approvers: orList(d.approvers.filter((p) => p.id !== d.requestedBy.id).map((p) => p.name)),
              role,
              me: me ?? '',
            })}
          </span>
        )}
      </div>
      {d.rules.length ? (
        <div className="row gap-2!">
          <span className="k">{t(mr.rulesMatched)}</span>
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
  const t = useT()
  const now = useNow()
  const approved = d.decisions.filter((x) => x.decision === 'approve')
  const last = approved[approved.length - 1]
  const next = d.campaign?.currentVersion || d.version
  const notified = andList(deciderFirst(d.approvers, last?.approver.name).map((p) => p.name.split(' ')[0]!))
  const b = (chunks: ReactNode[]) => <b>{chunks}</b>
  return (
    <section className="card" aria-labelledby="ap-inval">
      <div className="sec">
        <div className="col gap-0.5!">
          <h2 className="h2" id="ap-inval">
            {last ? t(mr.invalidatedAfterTitle) : t(mr.invalidatedBeforeTitle)}
          </h2>
          <span className="k">
            {t(mr.invalidatedCaption, { name: d.name, reason: d.invalidatedReason || t(mr.campaignChanged), after: last ? 'yes' : 'no' })}
          </span>
        </div>
        <Badge tone="bad">{statusWord('invalidated')}</Badge>
      </div>
      <div className="pad col gap-3!">
        <ol className="ap-steps">
          {last ? (
            <li>
              <T
                msg={mr.stepApproved}
                values={{
                  name: last.approver.name,
                  when: decidedText(last.at, now),
                  version: d.version,
                  hasComment: last.comment ? 'yes' : 'no',
                  comment: last.comment ?? '',
                  b,
                }}
              />
            </li>
          ) : (
            <li>
              <T msg={mr.stepRequested} values={{ name: d.requestedBy.name, version: d.version, b }} />
            </li>
          )}
          {d.changes.some((c) => c.field !== 'audience_count') ? (
            <li>
              {t(mr.stepEdited, {
                name: d.invalidatedBy || d.requestedBy.name,
                edits: editWords(d.changes),
                hasWhen: d.invalidatedAt ? 'yes' : 'no',
                when: d.invalidatedAt ? decidedText(d.invalidatedAt, now) : '',
                version: next,
              })}
            </li>
          ) : (
            <li>{t(mr.stepAudienceMoved, { count: d.audience ? count(d.audience.current) : '' })}</li>
          )}
          <li>
            <T
              msg={mr.stepInvalidated}
              values={{
                hasWhen: d.invalidatedAt ? 'yes' : 'no',
                when: d.invalidatedAt ? decidedText(d.invalidatedAt, now) : '',
                names: notified,
                scheduled: d.campaign?.scheduledAt ? 'yes' : 'no',
                b,
              }}
            />
          </li>
        </ol>
        <span className="ap-label">{t(mr.whatChangedVersions, { from: d.version, to: next })}</span>
        <DiffLines changes={d.changes} labelled />
        <dl className="kv m-0 text-[12.5px]">
          <dt>{t(mr.unchanged)}</dt>
          <dd>{unchangedText(d.changes, d.snapshot, d.audience ? { current: d.audience.current, changePct: d.audience.changePct } : undefined) || t(mr.nothingElse)}</dd>
        </dl>
        {d.can.decide ? (
          <div className="row gap-2!">
            <Button variant="primary" loading={busy === 'approve'} onClick={onApprove}>
              {t(mr.approveVersion, { version: next })}
            </Button>
            <Button loading={busy === 'request_changes'} onClick={onChanges}>
              {t(mr.requestChanges)}
            </Button>
          </div>
        ) : null}
      </div>
    </section>
  )
}

const editPhrases = {
  subject: mr.editSubject,
  preheader: mr.editPreheader,
  content: mr.editContent,
  sender: mr.editSender,
  audience: mr.editAudience,
  route: mr.editRoute,
  schedule: mr.editSchedule,
} as const

/** "edited the subject and replaced the link". */
function editWords(changes: Change[]): string {
  const words: string[] = []
  const links = changes.filter((c) => c.field === 'link').length
  for (const f of ['subject', 'preheader', 'content', 'sender', 'audience', 'route', 'schedule'] as const) {
    if (!changes.some((c) => c.field === f)) continue
    words.push(tt(editPhrases[f]))
    if (f === 'subject' && links) words.push(tt(mr.editLinkAfterSubject, { count: links }))
  }
  if (links && !changes.some((c) => c.field === 'subject')) words.unshift(tt(mr.editLink, { count: links }))
  return words.length ? andList(words) : tt(mr.editCampaign)
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
  const t = useT()
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
    <Sec id="ap-campaign" title={t(mr.campaignTitle)} caption={t(mr.campaignCaption)}>
      <div className="pad col gap-3!">
        <dl className="kv">
          <dt>{t(mr.subject)}</dt>
          <dd className="font-semibold">{d.snapshot.subject || c.subject}</dd>
          <dt>{t(mr.preheader)}</dt>
          <dd>{c.preheader || '—'}</dd>
          <dt>{t(mr.from)}</dt>
          <dd>{c.from || '—'}</dd>
          <dt>{t(mr.version)}</dt>
          <dd>{c.editedBy && c.editedAt && c.version === d.version ? t(mr.versionEdited, { version: d.version, name: c.editedBy, ago: ago(c.editedAt, now) }) : `v${d.version}`}</dd>
        </dl>
        <div className="ap-preview">
          {preview.data ? (
            <EmailThumb html={preview.data.html} title={t(mr.previewTitle, { name: d.name, version: preview.data.version })} />
          ) : preview.isError ? (
            <span className="k block py-6 text-center">{t(mr.previewUnavailable, { error: errorMessage(preview.error) })}</span>
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
          <Button size="sm" onClick={() => setOpen(true)} disabledReason={preview.data ? undefined : t(mr.previewLoading)}>
            {t(mr.openFullPreview)}
          </Button>
          <Button size="sm" loading={busy} onClick={onTest}>
            {t(mr.sendMeTest)}
          </Button>
        </div>
      </div>
      {open && preview.data ? (
        <Dialog open onOpenChange={setOpen} size="lg" title={preview.data.subject || d.name} description={preview.data.preheader || t(mr.versionNumber, { version: preview.data.version })}>
          <iframe title={t(mr.fullPreviewTitle, { name: d.name })} sandbox="allow-same-origin" srcDoc={withAppFonts(preview.data.html)} className="h-[70vh] w-full rounded-lg border border-line bg-white" />
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
  const t = useT()
  const { slug } = useWorkspace()
  const now = useNow()
  const fixReason = usePermissionReason('campaigns:edit')
  const r = d.readiness!
  const issueGroups = new Set<string>(r.issues.map((x) => x.group))
  const clean = Object.keys(groupWords).filter((g) => !issueGroups.has(g))
  const cleanText = clean.length ? capital(t(mr.groupsPassed, { groups: andList(clean.map((g) => groupWords[g]!)) })) : t(mr.everyOtherPassed)
  const tone = r.verdict === 'blocked' ? 'bad' : r.verdict === 'warnings' ? 'warn' : 'ok'
  return (
    <Sec id="ap-readiness" title={t(mr.readinessTitle)} caption={t(mr.readinessCaption)}>
      <div className="pad col gap-2.5!">
        <div className="row gap-2.5!">
          <Badge tone={tone}>{verdictTitle(r.verdict)}</Badge>
          <span className="k">{t(mr.checkedLine, { counts: countsShort(r), ago: ago(r.checkedAt, now) })}</span>
        </div>
        <AccessTable caption={t(mr.issuesCaption)}>
          <thead>
            <tr>
              <Th>{t(mr.colCheck)}</Th>
              <Th>{t(mr.colEvidence)}</Th>
              <Th>{t(mr.colFix)}</Th>
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
                        {f.action === 'exclude_sunset' ? t(mr.excludeThem) : f.label}
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
                  <span>{r.issues.length ? t(mr.everythingElse) : t(mr.everyCheck)}</span>
                </span>
              </Td>
              <Td>
                <span className="wrap">{r.issues.length ? cleanText : t(mr.allPassed)}</span>
              </Td>
              <Td>
                {d.campaign ? (
                  <ButtonLink size="sm" to="/w/$workspace/campaigns/$campaignId/review" params={{ workspace: slug, campaignId: d.campaign.id }}>
                    {t(mr.allChecks, { count: r.total })}
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

/** The sentence with its first letter in capitals, whatever word the language starts it with. */
function capital(s: string): string {
  return s ? s[0]!.toUpperCase() + s.slice(1) : s
}

function AudienceRouteCard({ detail: d }: { detail: Detail }) {
  const t = useT()
  const a = d.audience
  const r = d.route
  const change = a
    ? Math.abs(a.changePct) < 0.05
      ? t(mr.changeNone)
      : t(mr.changeSince, { change: `${a.changePct > 0 ? '+' : '−'}${pct(Math.abs(a.changePct), 1)}` })
    : ''
  return (
    <Sec id="ap-audience" title={t(mr.audienceRouteTitle)}>
      <div className="pad ap-ar">
        {a ? (
          <div className="col gap-1.5!">
            <span className="ap-label">{t(mr.audienceLabel, { count: count(a.count) })}</span>
            <dl className="kv">
              <dt>{t(mr.lists)}</dt>
              <dd>{a.lists.join(', ') || '—'}</dd>
              <dt>{t(mr.excluded)}</dt>
              <dd>{a.excluded || '—'}</dd>
              <dt>{t(mr.change)}</dt>
              <dd>{t(mr.changeLimit, { change })}</dd>
            </dl>
          </div>
        ) : null}
        {r ? (
          <div className="col min-w-0 gap-1.5!">
            <span className="ap-label">{t(mr.routeLabel, { split: r.splitLabel })}</span>
            <AccessTable caption={t(mr.routeCaption)}>
              <thead>
                <tr>
                  <Th>{t(mr.colProvider)}</Th>
                  <Th>{t(mr.colRecipients)}</Th>
                  <Th>{t(mr.colCost)}</Th>
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
                      <Td>{s.cost === 0 && s.priceLabel === 'self-hosted' ? t(mr.costSelfHosted, { cost: money(0, 0) }) : money(s.cost)}</Td>
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
  const t = useT()
  const s = d.schedule
  const when = !s
    ? d.snapshot.scheduleText
    : s.mode === 'now'
      ? t(mr.sendNow)
      : s.at
        ? s.mode === 'best_time'
          ? t(mr.bestTime, { when: zonedWhen(s.at, s.timezone) })
          : zonedWhen(s.at, s.timezone)
        : t(mr.notScheduled)
  const scheduled = s ? t(mr.estimated, { when, duration: s.minutes >= 60 ? minutes(s.minutes) : t(mr.estimatedMinutes, { n: s.minutes }) }) : when
  return (
    <Sec id="ap-schedule" title={d.subjectType === 'campaign' ? t(mr.scheduleNoteTitle) : t(mr.noteTitle)}>
      <div className="pad col gap-3!">
        <dl className="kv">
          {d.subjectType === 'campaign' ? (
            <>
              <dt>{t(mr.scheduled)}</dt>
              <dd>{scheduled}</dd>
              <dt>{t(mr.autoPause)}</dt>
              <dd>{s?.autoPause ?? '—'}</dd>
            </>
          ) : null}
          <dt>{t(mr.changesSinceLast)}</dt>
          <dd>
            {d.changesSince.length
              ? t(mr.changedSince, { changes: andList(d.changesSince.map((c) => c.label.toLowerCase())) })
              : d.followsId
                ? t(mr.nothingChangedSince)
                : t(mr.firstRequest)}
          </dd>
        </dl>
        {d.changesSince.length ? <DiffLines changes={d.changesSince} labelled /> : null}
        {d.note ? (
          <div className="note n items-start!">
            <Avatar name={d.requestedBy.name} color={d.requestedBy.color} size={32} fontSize={13} />
            <span className="col gap-0.5!">
              <span className="font-semibold">{t(mr.noteForApprovers, { name: d.requestedBy.name })}</span>
              <span>{d.note}</span>
            </span>
          </div>
        ) : null}
      </div>
    </Sec>
  )
}

function AutomationCard({ detail: d }: { detail: Detail }) {
  const t = useT()
  const a = d.automation
  return (
    <Sec id="ap-automation" title={t(mr.automationTitle)} caption={t(mr.automationCaption)}>
      <div className="pad col gap-3!">
        <dl className="kv">
          <dt>{t(mr.version)}</dt>
          <dd>v{a?.version ?? d.version}</dd>
          <dt>{t(mr.contactsInFlight)}</dt>
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

type Entry = { key: string; who: Schemas['ApprovalPerson']; at: string; decision?: Schemas['ApprovalDecisionView']['decision']; body?: string }

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
  const t = useT()
  const { data: me } = useMe()
  const now = useNow()
  const id = useId()
  const entries: Entry[] = [
    ...d.comments.map((c) => ({ key: c.id, who: c.author, at: c.at, body: c.body })),
    ...d.decisions.map((x, i) => ({ key: `d${i}`, who: x.approver, at: x.at, decision: x.decision, body: x.comment || undefined })),
  ].sort((a, b) => new Date(a.at).getTime() - new Date(b.at).getTime())
  const user = me?.user
  const b = (chunks: ReactNode[]) => <b className="text-ink">{chunks}</b>
  return (
    <Sec id="ap-discussion" title={t(mr.discussionTitle)} caption={t(mr.discussionCaption)}>
      <div className="pad col gap-3!">
        {entries.length === 0 && !d.can.comment ? <span className="k">{t(mr.noComments)}</span> : null}
        {entries.map((e) => (
          <div key={e.key} className="row items-start! gap-2.5!">
            <Avatar name={e.who.name} color={e.who.color} size={32} fontSize={13} />
            <div className="col gap-0.5!">
              <span className="k">
                {e.decision ? (
                  <T msg={mr.entryDecision} values={{ name: e.who.name, decision: e.decision, ago: ago(e.at, now), b }} />
                ) : (
                  <T msg={mr.entryComment} values={{ name: e.who.name, ago: ago(e.at, now), b }} />
                )}
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
                {t(mr.commentLabel)}
              </label>
              <textarea
                id={id}
                className="in min-h-0"
                rows={2}
                maxLength={4000}
                placeholder={others.length ? t(mr.replyTo, { names: andList(others.map((p) => p.name.split(' ')[0]!)) }) : t(mr.addComment)}
                value={comment}
                onChange={(e) => setComment(e.target.value)}
              />
              <div className="row justify-end">
                <Button size="sm" loading={busy} disabledReason={comment.trim() ? undefined : t(mr.writeFirst)} onClick={onPost}>
                  {t(mr.postComment)}
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
  const reason = d.can.decide ? undefined : d.can.decideReason || t(mr.onlyApprovers)
  const last = d.decisions[d.decisions.length - 1]
  return (
    <section className="card ap-sticky" aria-labelledby="ap-decision">
      <div className="sec">
        <h2 className="h2" id="ap-decision">
          {open || d.status === 'invalidated' ? t(mr.yourDecision) : t(mr.decision)}
        </h2>
        {d.clientSignoff ? <ClientSignoffIntro detail={d} /> : null}
      </div>
      <div className="pad col gap-2.5!">
        {open ? (
          <>
            <Button variant="primary" className="justify-center" loading={busy === 'approve'} disabledReason={reason} onClick={onApprove}>
              {issueNumber ? t(im.approveIssue, { number: issueNumber }) : t(mr.approve)}
            </Button>
            <Button className="justify-center" loading={busy === 'request_changes'} disabledReason={reason} onClick={onChanges}>
              {t(mr.requestChanges)}
            </Button>
            <Button variant="danger" className="justify-center" loading={busy === 'reject'} disabledReason={reason} onClick={onReject}>
              {t(mr.reject)}
            </Button>
            {d.page ? (
              <PageDecisionNote author={d.author.name} page={d.page} />
            ) : d.transactional ? (
              <TransactionalDecisionNote author={d.author.name} />
            ) : d.clientSignoff ? (
              <ClientSignoffNote detail={d} />
            ) : (
              <div className="note n text-[12.5px]">
                <span>{t(mr.approveNote, { name: d.author.name.split(' ')[0]! })}</span>
              </div>
            )}
          </>
        ) : d.status === 'invalidated' ? (
          <div className="note n text-[12.5px]">
            <span>{t(mr.invalidatedNote, { decide: d.can.decide ? 'yes' : 'no', version: d.version, next: d.campaign?.currentVersion || d.version })}</span>
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
            {d.bypassReason ? <span className="text-[13px]">{t(mr.bypassedReason, { reason: d.bypassReason })}</span> : null}
            {d.campaign && d.can.isAuthor ? (
              <ButtonLink size="sm" className="self-start" to="/w/$workspace/campaigns/$campaignId/review" params={{ workspace: slug, campaignId: d.campaign.id }}>
                {t(mr.openReview)}
              </ButtonLink>
            ) : null}
          </div>
        )}
        {d.clientSignoff ? (
          <ClientSignoffFacts detail={d} />
        ) : (
          <dl className="kv m-0 text-[12.5px]">
            <dt>{t(mr.author)}</dt>
            <dd>{d.author.role ? t(mr.authorRole, { name: d.author.name, role: roleName({ name: d.author.role }) }) : d.author.name}</dd>
            <dt>{t(mr.approvers)}</dt>
            <dd>{approverNames(d.approvers, 'comma') || '—'}</dd>
            <dt>{t(mr.needed)}</dt>
            <dd>{t(mr.neededValue, { left: Math.max(0, d.needed - d.given), needed: d.needed })}</dd>
            <dt>{t(mr.requested)}</dt>
            <dd>{monthDayTime(d.requestedAt)}</dd>
          </dl>
        )}
      </div>
    </section>
  )
}

function DetailSkeleton() {
  const t = useT()
  return (
    <Loading label={t(mr.loading)} className="flex flex-col gap-5">
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
