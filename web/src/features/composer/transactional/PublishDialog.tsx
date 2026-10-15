import { useCallback, useEffect, useRef, useState, type ReactNode } from 'react'
import { ApiError, errorMessage, http } from '@/api/client'
import { Avatar } from '@/components/ui/Avatar'
import { Badge } from '@/components/ui/Badge'
import { Button } from '@/components/ui/Button'
import { Dialog } from '@/components/ui/Dialog'
import { OptionCard } from '@/components/ui/Field'
import { Icon } from '@/components/icons/Icon'
import { toast } from '@/components/ui/Toast'
import { formatList, T, useT } from '@/i18n'
import { ago, clock, dayMonthDate } from '@/lib/format'
import { useComposer } from '../context'
import { useCategoryLabel } from './Header'
import { mp } from './messages'
import { useTx } from './state'
import { PREVIEW_TO, type ApprovalState, type PublishPreview } from './types'

type Choice = 'optional' | 'later' | 'anyway'

function Diff({ preview, full }: { preview: PublishPreview; full: boolean }) {
  const t = useT()
  const d = preview.diff
  return (
    <div className="diff">
      {d.subject ? (
        <>
          <div className="ctx txe-dh">{t(mp.subject)}</div>
          <div className="del">− {d.subject.before}</div>
          <div className="add">+ {d.subject.after}</div>
        </>
      ) : null}
      {d.preheader ? (
        <>
          <div className="ctx txe-dh">{t(mp.preheader)}</div>
          <div className="del">− {d.preheader.before}</div>
          <div className="add">+ {d.preheader.after}</div>
        </>
      ) : null}
      {d.variables.length ? (
        <>
          <div className="ctx txe-dh">{t(mp.variables)}</div>
          {d.variables.map((v) => (
            <div key={v.name} className={v.change === 'removed' ? 'del' : v.change === 'added' ? 'add' : 'ctx'}>
              {v.change === 'removed' ? '− ' : v.change === 'added' ? '+ ' : '~ '}
              {v.name} · {v.summary}
            </div>
          ))}
        </>
      ) : null}
      {d.settings.length ? (
        <>
          <div className="ctx txe-dh">{t(mp.settings)}</div>
          <div className="ctx">{formatList(d.settings)}</div>
        </>
      ) : null}
      <div className="ctx txe-dh">{t(mp.content)}</div>
      {full ? (
        d.content.length ? (
          d.content.map((l, i) => (
            <div key={i} className={l.kind}>
              {l.kind === 'add' ? '+ ' : l.kind === 'del' ? '− ' : '  '}
              {l.text}
            </div>
          ))
        ) : (
          <div className="ctx">{t(mp.noContentChange)}</div>
        )
      ) : (
        <div className="ctx">{d.contentSummary || t(mp.noContentChange)}</div>
      )}
    </div>
  )
}

/**
 * Publishing the draft (TransactionalPublish, TransactionalPublishStates):
 * the diff since the live version, the breaking-change check against the
 * calls of the last 7 days with its fixes, the test renders including the
 * last real call, the checks, and the approval the workspace rule asks for.
 * Published: a toast with Undo, which publishes the previous version again.
 */
export function PublishDialog({ open, onClose }: { open: boolean; onClose: () => void }) {
  // Each opening starts afresh: the draft is saved and compared again.
  return open ? <PublishFlow onClose={onClose} /> : null
}

function PublishFlow({ onClose }: { onClose: () => void }) {
  const open = true
  const t = useT()
  const c = useComposer()
  const tx = useTx()
  const tpl = tx.template
  const category = useCategoryLabel()
  const [dirty] = useState(() => c.status.kind !== 'saved' && c.status.kind !== 'new')
  const [phase, setPhase] = useState<'saving' | 'loading' | 'ready' | 'publishing' | 'requesting'>(dirty ? 'saving' : 'loading')
  const [preview, setPreview] = useState<PublishPreview | null>(null)
  const [approval, setApproval] = useState<ApprovalState | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [choice, setChoice] = useState<Choice>('optional')
  const [defaults, setDefaults] = useState<Record<string, string>>({})
  const [note, setNote] = useState('')
  const [full, setFull] = useState(false)
  const seq = useRef(0)
  const path = { workspace: c.slug, template: tpl.id }

  const makeOptional = useCallback(
    (p: PublishPreview | null, d: Record<string, string>) => (p?.breaking.breaking ?? []).map((b) => ({ name: b.name, default: d[b.name] ?? b.suggestedDefault })),
    [],
  )

  const load = useCallback(
    async (opts: { optional?: { name: string; default: string }[] } = {}) => {
      const n = ++seq.current
      try {
        const [p, a] = await Promise.all([
          http.POST('/api/v1/workspaces/{workspace}/transactional/templates/{template}/publish-preview', { params: { path }, body: { makeOptional: opts.optional?.length ? opts.optional : undefined } }),
          http.GET('/api/v1/workspaces/{workspace}/transactional/templates/{template}/approval', { params: { path } }),
        ])
        if (n !== seq.current) return
        setPreview(p.data ?? null)
        setApproval(a.data ?? null)
        if (!opts.optional && p.data) setDefaults(Object.fromEntries(p.data.breaking.breaking.map((b) => [b.name, b.suggestedDefault])))
        setPhase('ready')
      } catch (e) {
        if (n !== seq.current) return
        setError(errorMessage(e))
        setPhase('ready')
      }
    },
    // The path only changes with the template.
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [c.slug, tpl.id],
  )

  // Opening saves the draft first, so the dialog compares what is saved.
  useEffect(() => {
    let cancelled = false
    const run = async () => {
      if (dirty) {
        const ok = await c.save({ quiet: true })
        if (cancelled) return
        if (!ok) {
          setError(t(mp.saveFirst))
          setPhase('ready')
          return
        }
        setPhase('loading')
      }
      await load()
    }
    void run()
    return () => {
      cancelled = true
    }
    // Runs once, when the dialog opens.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [])

  // Making the variables optional changes the test renders: preview that too.
  const breaking = preview?.breaking.breaking ?? []
  useEffect(() => {
    if (phase !== 'ready' || !breaking.length || choice !== 'optional') return
    const h = window.setTimeout(() => void load({ optional: makeOptional(preview, defaults) }), 350)
    return () => window.clearTimeout(h)
    // Re-run when the defaults or the choice change.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [choice, JSON.stringify(defaults)])

  const needsApproval = !!preview?.approval.required && !(approval?.status === 'approved' && approval.coversDraft)
  const next = preview?.nextVersion ?? tpl.versionsCount + 1

  const undo = async (previous: number) => {
    try {
      const { data } = await http.POST('/api/v1/workspaces/{workspace}/transactional/templates/{template}/rollback', { params: { path }, body: { version: previous, undo: true } })
      if (data) tx.mergeTemplate(data.template)
      toast({ kind: 'success', title: t(mp.undone, { key: tpl.key, version: previous }) })
    } catch (e) {
      toast({ kind: 'error', title: t(mp.undoFailed), detail: errorMessage(e) })
    }
  }

  const publish = async () => {
    if (!preview) return
    setPhase('publishing')
    setError(null)
    const optional = choice === 'optional' && breaking.length ? makeOptional(preview, defaults) : undefined
    try {
      const { data } = await http.POST('/api/v1/workspaces/{workspace}/transactional/templates/{template}/publish', {
        params: { path },
        body: { baseRevision: preview.draftRevision, makeOptional: optional, publishAnyway: choice === 'anyway' && breaking.length ? true : undefined },
      })
      if (!data) return
      tx.afterPublish(data.template)
      onClose()
      const at = clock(new Date())
      const previous = data.previous
      toast({
        kind: 'success',
        title: t(mp.published, { key: data.template.key, version: data.version }),
        detail: optional ? t(mp.publishedDefaults, { time: at }) : at,
        undo: previous ? () => void undo(previous) : undefined,
      })
    } catch (e) {
      setPhase('ready')
      if (e instanceof ApiError && e.status === 409) {
        // Breaking change, approval, a newer draft or a check that blocks: show where things stand now.
        setError(e.problem.detail ?? errorMessage(e))
        await load(optional ? { optional } : {})
        return
      }
      setError(errorMessage(e))
    }
  }

  const requestApproval = async () => {
    setPhase('requesting')
    setError(null)
    try {
      const { data } = await http.POST('/api/v1/workspaces/{workspace}/transactional/templates/{template}/approval', { params: { path }, body: { note: note.trim() || undefined } })
      if (data) setApproval(data)
      toast({ kind: 'success', title: t(mp.requested), detail: t(mp.requestedDetail, { approvers: formatList(data?.approvers ?? [], 'disjunction') }) })
      setPhase('ready')
    } catch (e) {
      setError(errorMessage(e))
      setPhase('ready')
    }
  }

  const copyPayload = () => {
    const names = breaking.map((b) => b.name)
    const data: Record<string, unknown> = { ...tx.previewData }
    for (const b of breaking) if (!(b.name in data)) data[b.name] = tx.variables.find((v) => v.name === b.name)?.sample ?? ''
    const payload = JSON.stringify({ template: tpl.key, to: PREVIEW_TO, data }, null, 2)
    void navigator.clipboard.writeText(payload).then(
      () => {
        toast({ kind: 'success', title: t(mp.payloadCopied), detail: t(mp.payloadCopiedDetail, { names: formatList(names) }) })
        onClose()
      },
      () => toast({ kind: 'error', title: t(mp.copyFailed), detail: t(mp.copyFailedDetail) }),
    )
  }

  const busy = phase === 'publishing' || phase === 'requesting'
  const loading = phase === 'saving' || phase === 'loading' || !preview

  // The approval variant (TransactionalPublishStates · Approval required).
  if (open && preview && needsApproval) {
    const pending = approval?.status === 'pending' && approval.coversDraft
    const decided = approval && approval.coversDraft && (approval.status === 'changes_requested' || approval.status === 'rejected') ? approval : null
    return (
      <Dialog
        open={open}
        onOpenChange={(o) => !o && !busy && onClose()}
        size="md"
        eyebrow={<span className="k">{t(mp.approvalEyebrow, { key: tpl.key, version: next })}</span>}
        title={pending ? t(mp.waitingTitle) : t(mp.approvalTitle)}
        footer={
          <>
            <Button onClick={onClose}>{pending ? t(mp.close) : t(mp.cancel)}</Button>
            {pending ? null : (
              <Button variant="primary" loading={phase === 'requesting'} disabledReason={approval && !approval.canRequest ? approval.requestReason || undefined : undefined} onClick={() => void requestApproval()}>
                {decided ? t(mp.requestAgain) : t(mp.requestApproval)}
              </Button>
            )}
          </>
        }
      >
        {pending ? (
          <p className="m-0 text-ink-nav">
            {t(mp.waitingBody, { when: ago(approval.requestedAt), approvers: formatList(approval.approvers, 'disjunction') })}
          </p>
        ) : (
          <>
            <p className="m-0 text-ink-nav">{preview.approval.reason || approval?.requestReason}</p>
            {decided ? (
              <div className="note w" role="note">
                <Icon name="alertCircle" size={16} />
                <div>
                  {decided.status === 'rejected'
                    ? t(mp.rejectedBy, { name: decided.decidedBy ?? '', comment: decided.comment ?? '' })
                    : t(mp.changesBy, { name: decided.decidedBy ?? '', comment: decided.comment ?? '' })}
                </div>
              </div>
            ) : null}
            <label className="lbl">
              {t(mp.noteForApprover)}
              <textarea className="in" rows={3} value={note} maxLength={1000} onChange={(e) => setNote(e.target.value)} />
            </label>
          </>
        )}
        {error ? <ErrorNote>{error}</ErrorNote> : null}
      </Dialog>
    )
  }

  const first = !tpl.liveVersion
  const blocking = preview?.blocking ?? []
  const renders = preview?.testRenders ?? []
  const passed = (preview?.checks.checks ?? []).filter((x) => x.severity === 'pass')
  const allRender = renders.length > 0 && renders.every((r) => r.ok)
  const primary = (() => {
    if (breaking.length && choice === 'later')
      return (
        <Button variant="primary" onClick={copyPayload}>
          {t(mp.copyPayload)}
        </Button>
      )
    const label = breaking.length && choice === 'optional' ? t(mp.optionalAndPublish, { count: breaking.length, version: next }) : breaking.length && choice === 'anyway' ? t(mp.publishAnyway) : t(mp.publishVersion, { version: next })
    return (
      <Button
        variant={breaking.length && choice === 'anyway' ? 'danger-primary' : 'primary'}
        loading={phase === 'publishing'}
        busyLabel={t(mp.publishing)}
        disabledReason={loading ? t(mp.preparing) : blocking.length ? t(mp.blockedReason) : !preview?.canPublish ? tpl.access.publishReason || t(mp.cantPublish) : undefined}
        onClick={() => void publish()}
      >
        {label}
      </Button>
    )
  })()

  return (
    <Dialog
      open={open}
      onOpenChange={(o) => !o && !busy && onClose()}
      size="lg"
      dismissible={!busy}
      eyebrow={<span className="k">{t(mp.eyebrow, { key: tpl.key, category })}</span>}
      title={t(mp.title, { version: next, name: tpl.name })}
      description={first ? t(mp.leadFirst, { key: tpl.key }) : t(mp.lead, { version: next, live: tpl.liveVersion ?? 0 })}
      footer={
        <>
          <span className="k" style={{ marginRight: 'auto' }}>
            {t(mp.audit)}
          </span>
          <Button onClick={onClose} disabled={busy}>
            {t(mp.cancel)}
          </Button>
          {primary}
        </>
      }
    >
      {phase === 'publishing' ? (
        <div className="txe-busy" role="status">
          <span className="spin" aria-hidden="true" />
          <span className="flex flex-col">
            <span className="font-semibold">{t(mp.publishingTitle, { version: next })}</span>
            <span className="k">{t(mp.publishingDetail, { count: renders.filter((r) => r.source === 'test_data').length })}</span>
          </span>
        </div>
      ) : null}
      {loading ? (
        <div className="txe-busy" role="status">
          <span className="spin" aria-hidden="true" />
          <span className="k">{phase === 'saving' ? t(mp.savingDraft) : first ? t(mp.loadingFirst) : t(mp.loading, { live: tpl.liveVersion ?? 0 })}</span>
        </div>
      ) : null}
      {error ? <ErrorNote>{error}</ErrorNote> : null}
      {preview ? (
        <>
          {preview.draftSavedBy ? (
            <div className="flex flex-wrap items-center gap-2.5">
              <span className="inline-flex items-center gap-2 whitespace-nowrap">
                <Avatar name={preview.draftSavedBy} color="warn" size={22} />
                <span>{preview.draftSavedBy}</span>
              </span>
              <span className="k">
                {preview.liveSince && preview.liveVersion
                  ? t(mp.editedLive, { time: clock(preview.draftSavedAt), live: preview.liveVersion, date: dayMonthDate(preview.liveSince), by: preview.liveBy ?? '' })
                  : t(mp.edited, { time: clock(preview.draftSavedAt) })}
              </span>
            </div>
          ) : null}
          <div className="flex flex-col gap-2">
            <div className="flex items-center justify-between gap-2">
              <span className="font-semibold">{t(mp.whatChanges)}</span>
              {preview.diff.content.length ? (
                <button type="button" className="txe-linklike" aria-expanded={full} onClick={() => setFull((v) => !v)}>
                  {full ? t(mp.hideDiff) : t(mp.showDiff)}
                </button>
              ) : null}
            </div>
            {first ? <span className="k">{t(mp.firstVersion)}</span> : preview.diff.changed ? <Diff preview={preview} full={full} /> : <span className="k">{t(mp.noChanges, { version: preview.liveVersion ?? 0 })}</span>}
          </div>
          {blocking.length ? (
            <div className="note b" role="alert">
              <Icon name="alertCircle" size={16} />
              <div className="flex flex-col gap-1">
                <span className="font-bold">{t(mp.blockedTitle)}</span>
                {blocking.map((b) => (
                  <span key={b}>{b}</span>
                ))}
              </div>
            </div>
          ) : null}
          {breaking.length ? (
            <>
              <div className="note b" role="alert">
                <Icon name="alertCircle" size={16} />
                <div className="flex flex-col gap-1">
                  <span className="font-bold">{t(mp.breakingTitle, { version: next })}</span>
                  <span>{preview.breaking.summary}</span>
                </div>
              </div>
              <fieldset className="txe-fs">
                <legend className="font-semibold">{t(mp.howPublish)}</legend>
                <OptionCard
                  name="tx-breaking"
                  value="optional"
                  checked={choice === 'optional'}
                  onChange={() => setChoice('optional')}
                  title={
                    <>
                      {breaking.length === 1 ? (
                        <T msg={mp.makeOptional} values={{ name: breaking[0]!.name, value: defaults[breaking[0]!.name] ?? '', mono: (ch) => <span className="mono">{ch}</span> }} />
                      ) : (
                        t(mp.makeOptionalMany, { names: formatList(breaking.map((b) => b.name)) })
                      )}{' '}
                      <Badge tone="ok" className="ml-1">
                        {t(mp.recommended)}
                      </Badge>
                    </>
                  }
                  description={t(mp.makeOptionalDetail, { count: breaking.length })}
                />
                {choice === 'optional' ? (
                  <div className="txe-defaults">
                    {breaking.map((b) => (
                      <label key={b.name} className="lbl">
                        {t(mp.defaultFor, { name: b.name })}
                        <input className="in" value={defaults[b.name] ?? ''} maxLength={4000} onChange={(e) => setDefaults((d) => ({ ...d, [b.name]: e.target.value }))} />
                      </label>
                    ))}
                  </div>
                ) : null}
                <OptionCard
                  name="tx-breaking"
                  value="later"
                  checked={choice === 'later'}
                  onChange={() => setChoice('later')}
                  title={t(mp.updateApp, { count: breaking.length })}
                  description={t(mp.updateAppDetail)}
                />
                <OptionCard
                  name="tx-breaking"
                  value="anyway"
                  checked={choice === 'anyway'}
                  onChange={() => setChoice('anyway')}
                  title={<span className="txe-bad">{t(mp.publishAnyway)}</span>}
                  description={t(mp.publishAnywayDetail, { names: formatList(breaking.map((b) => b.name)) })}
                />
              </fieldset>
            </>
          ) : null}
          {breaking.length || !allRender ? (
            <div className="flex flex-col gap-2">
              <span className="font-semibold">{renders.some((r) => r.source === 'last_call') ? t(mp.testRendersCall) : t(mp.testRenders)}</span>
              <div className="txe-renders">
                {renders.map((r) => (
                  <div key={`${r.source}-${r.name}`} className="chk">
                    <Icon name={r.ok ? 'check' : 'alertCircle'} size={14} />
                    <span style={{ flex: 1 }}>
                      <b className="font-semibold">{r.source === 'last_call' && r.at ? t(mp.realCall, { when: clock(r.at) }) : r.name}</b>{' '}
                      <span className="k">
                        {r.source === 'last_call'
                          ? t(mp.realCallDetail, { recipient: r.recipient ?? '' })
                          : r.defaults.length
                            ? t(mp.testDataDefaults, { names: formatList(r.defaults) })
                            : t(mp.testData)}
                      </span>
                      {!r.ok && r.problem ? <span className="k txe-bad"> · {r.problem}</span> : null}
                    </span>
                    <Badge tone={r.ok ? 'ok' : 'bad'}>{r.ok ? (r.defaults.length ? t(mp.rendersDefault) : t(mp.renders)) : t(mp.fails)}</Badge>
                  </div>
                ))}
                {renders.length === 0 ? <span className="k">{t(mp.noTestData)}</span> : null}
              </div>
            </div>
          ) : null}
          {breaking.length ? (
            <div className="flex flex-col gap-2">
              <span className="font-semibold">{t(mp.checksPassed, { count: passed.length })}</span>
              <div className="flex flex-wrap gap-1.5">
                {passed.map((x) => (
                  <Badge key={x.id} tone="ok">
                    {x.title}
                  </Badge>
                ))}
              </div>
            </div>
          ) : (
            <div className="flex flex-wrap gap-1.5">
              <Badge tone="ok">{preview.breaking.calls ? t(mp.noBreakingCalls, { count: preview.breaking.calls }) : t(mp.noBreaking)}</Badge>
              {allRender ? <Badge tone="ok">{t(mp.rendersAll, { count: renders.length })}</Badge> : null}
              <Badge tone={preview.checks.blocks ? 'bad' : 'ok'}>{t(mp.checksPassedChip, { count: preview.checks.passed })}</Badge>
              {preview.checks.warnings ? <Badge tone="warn">{t(mp.checksWarnings, { count: preview.checks.warnings })}</Badge> : null}
            </div>
          )}
          {preview.approval.required && approval?.status === 'approved' && approval.coversDraft ? (
            <div className="note g" role="status">
              <Icon name="check" size={14} />
              <div>{t(mp.approvedBy, { name: approval.decidedBy ?? '', when: approval.decidedAt ? ago(approval.decidedAt) : '' })}</div>
            </div>
          ) : null}
        </>
      ) : null}
    </Dialog>
  )
}

function ErrorNote({ children }: { children: ReactNode }) {
  return (
    <div className="note b" role="alert">
      <Icon name="alertCircle" size={16} />
      <div>{children}</div>
    </div>
  )
}
