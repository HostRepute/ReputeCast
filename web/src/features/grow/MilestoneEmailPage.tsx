import { Link } from '@tanstack/react-router'
import type { Component } from 'grapesjs'
import { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import { $api, ApiError, errorMessage, http, type Schemas } from '@/api/client'
import { LogoMark } from '@/components/icons/Icon'
import { Badge } from '@/components/ui/Badge'
import { Button } from '@/components/ui/Button'
import { UnsavedChangesGuard } from '@/components/ui/ConfirmDialog'
import { Dialog } from '@/components/ui/Dialog'
import { ErrorPanel, Skeleton } from '@/components/ui/Feedback'
import { Field, Input, Select, Textarea } from '@/components/ui/Field'
import { Note } from '@/components/ui/Note'
import { Segmented } from '@/components/ui/Tabs'
import { toast } from '@/components/ui/Toast'
import { T, useT } from '@/i18n'
import { clock, dayMonthDate } from '@/lib/format'
import { useDocumentTitle } from '@/lib/hooks'
import { useCan, useMe, useWorkspace } from '@/lib/session'
import { BlockIcon } from '@/features/composer/icons'
import { createComposerEditor, type ComposerEditor } from '@/features/composer/editor/grapes'
import { canvasHead, joinMjml, splitMjml, withHeadMeta } from '@/features/composer/editor/mjmlDoc'
import { growBlockDefs, isGrowBlock, setGrowPreview } from './composer/blocks'
import { GrowBlockInspector, GrowBlocksNote } from './composer/Panel'
import { typeLabel } from './MilestonesPage'
import { me } from './email.messages'
import { GROW, fileKind, fileSize, mono } from './shared'
import '@/features/composer/composer.css'
import '@/features/connect/connect.css'
import './grow.css'

type Detail = Schemas['ReferralMilestoneEmailDetail']
type Status = { kind: 'saved' | 'autosaved' | 'dirty' | 'saving' | 'conflict' | 'new'; at?: string }

/**
 * A milestone email (ComposerGrowBlocks, frame 2): the canvas and its MJML,
 * the subject and preheader, "Send it at milestone 3", the sender and
 * stream, a preview as a referrer, tests, autosave and Save (the saved
 * version is the one that goes out).
 */
export function MilestoneEmailPage({ milestoneId }: { milestoneId: string }) {
  const t = useT()
  const { slug } = useWorkspace()
  const q = $api.useQuery('get', `${GROW}/milestones/{milestone}/email`, { params: { path: { workspace: slug, milestone: milestoneId } } }, { refetchOnWindowFocus: false })
  const d = q.data
  useDocumentTitle(d ? t(me.docTitle, { count: d.count }) : t(me.loading))
  if (!d) {
    return (
      <div className="cmp rc-app" style={{ padding: '56px 16px', alignItems: 'center' }}>
        <main id="main" className="w-full max-w-[640px]">
          {q.isError ? (
            <ErrorPanel title={t(me.loadError)} actions={<Button onClick={() => void q.refetch()}>{t(me.reload)}</Button>}>
              {errorMessage(q.error)}
            </ErrorPanel>
          ) : (
            <div aria-busy="true" className="flex flex-col gap-3">
              <span className="vh" role="status">
                {t(me.loading)}
              </span>
              <Skeleton h={48} />
              <Skeleton h={420} />
            </div>
          )}
        </main>
      </div>
    )
  }
  return <Editor key={`${d.milestoneId}:${d.revision}`} d={d} />
}

function Editor({ d }: { d: Detail }) {
  const t = useT()
  const { slug } = useWorkspace()
  const canEdit = useCan('grow:edit')
  const parts = useMemo(() => splitMjml(d.mjml), [d.mjml])
  const headRef = useRef(parts.head)
  const bodyRef = useRef(parts.body)
  const [subject, setSubject] = useState(d.subject)
  const [preheader, setPreheader] = useState(d.preheader)
  const [view, setView] = useState<'visual' | 'code'>('visual')
  const [code, setCode] = useState('')
  const [on, setOn] = useState(d.on)
  const [status, setStatus] = useState<Status>(d.savedAt ? { kind: 'saved', at: d.savedAt } : { kind: 'new' })
  const [errors, setErrors] = useState<{ line: number; message: string }[]>([])
  const [testing, setTesting] = useState(false)
  const [contact, setContact] = useState('')
  const [rev, setRev] = useState(0)
  const revision = useRef(d.revision)
  const container = useRef<HTMLDivElement>(null)
  const geRef = useRef<ComposerEditor | null>(null)
  const [ge, setGe] = useState<ComposerEditor | null>(null)
  const [selected, setSelected] = useState<Component | null>(null)
  const dirty = status.kind === 'dirty'

  // The canvas draws Grow's blocks for this milestone and the "Preview as" reader.
  useEffect(() => {
    setGrowPreview({ slug, milestoneId: d.milestoneId, contactId: contact })
  }, [slug, d.milestoneId, contact])
  useEffect(() => () => setGrowPreview({ milestoneId: '', contactId: null }), [])

  useEffect(() => {
    if (!container.current) return
    let cancelled = false
    let created: ComposerEditor | null = null
    const mount = document.createElement('div')
    container.current.appendChild(mount)
    void createComposerEditor(mount, { body: bodyRef.current, head: () => canvasHead(headRef.current), device: 'desktop' })
      .then((inst) => {
        if (cancelled) {
          inst.editor.destroy()
          mount.remove()
          return
        }
        created = inst
        geRef.current = inst
        const mark = () => {
          setStatus((s) => (s.kind === 'conflict' ? s : { kind: 'dirty', at: s.at }))
          setRev((r) => r + 1)
        }
        inst.editor.on('update', mark)
        inst.editor.on('component:input', mark)
        inst.editor.on('component:selected component:deselected', () => setSelected(inst.editor.getSelected() ?? null))
        if (!canEdit) inst.editor.getWrapper()?.onAll((c) => c.set({ draggable: false, droppable: false, editable: false, removable: false, copyable: false }))
        // GrapesJS's frame has no title of its own: screen readers announce the canvas by it.
        const title = () => inst.editor.Canvas.getFrameEl()?.setAttribute('title', t(me.canvasLabel))
        title()
        inst.editor.on('canvas:frame:load', title)
        setGe(inst)
      })
      .catch((e: unknown) => toast({ kind: 'error', title: t(me.editorFailed), detail: e instanceof Error ? e.message : String(e) }))
    return () => {
      cancelled = true
      created?.editor.destroy()
      mount.remove()
      geRef.current = null
    }
    // The editor is created once per load.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [])

  /** The whole document as MJML, with the subject and preheader in its head. */
  const currentMjml = useCallback(() => {
    const body = view === 'code' ? (splitMjml(code).body ?? bodyRef.current) : geRef.current ? geRef.current.getBody() : bodyRef.current
    const head = view === 'code' ? splitMjml(code).head : headRef.current
    return joinMjml(withHeadMeta(head, { title: subject, preview: preheader }), body)
  }, [view, code, subject, preheader])

  const compile = useCallback(() => {
    const inst = geRef.current
    const mjml = currentMjml()
    if (!inst) return { mjml, html: d.html, errors: [] as { line: number; message: string }[] }
    const out = inst.compile(mjml)
    return { mjml, html: out.html, errors: out.errors.map((e) => ({ line: e.line, message: e.message })) }
  }, [currentMjml, d.html])

  const save = useCallback(
    async (final: boolean) => {
      if (!canEdit) return
      const c = compile()
      setErrors(c.errors)
      setStatus((s) => ({ kind: 'saving', at: s.at }))
      const body = { baseRevision: revision.current, html: c.html, mjml: c.mjml, subject, preheader }
      const path = { params: { path: { workspace: slug, milestone: d.milestoneId } }, body }
      try {
        const { data } = final
          ? await http.POST('/api/v1/workspaces/{workspace}/grow/milestones/{milestone}/email/save', path)
          : await http.PUT('/api/v1/workspaces/{workspace}/grow/milestones/{milestone}/email', path)
        if (data) {
          revision.current = data.revision
          setStatus({ kind: final ? 'saved' : 'autosaved', at: data.savedAt })
          if (final) toast({ kind: 'success', title: t(me.saved, { version: data.version ?? 1 }), detail: t(me.savedDetail) })
        }
      } catch (e) {
        if (e instanceof ApiError && e.status === 409) setStatus({ kind: 'conflict' })
        else {
          setStatus({ kind: 'dirty' })
          toast({ kind: 'error', title: errorMessage(e) })
        }
      }
    },
    [canEdit, compile, subject, preheader, slug, d.milestoneId, t],
  )

  // Autosave a few seconds after the last edit.
  useEffect(() => {
    if (!dirty) return
    const h = window.setTimeout(() => void save(false), 3000)
    return () => window.clearTimeout(h)
  }, [dirty, rev, subject, preheader, code, save])

  const switchView = (v: 'visual' | 'code') => {
    if (v === view) return
    if (v === 'code') {
      setCode(currentMjml())
    } else {
      const parts = splitMjml(code)
      headRef.current = parts.head
      bodyRef.current = parts.body
      geRef.current?.setBody(parts.body)
    }
    setView(v)
  }

  const flip = async (next: boolean) => {
    setOn(next)
    try {
      await http.PUT('/api/v1/workspaces/{workspace}/grow/milestones/{milestone}/email/on', { params: { path: { workspace: slug, milestone: d.milestoneId } }, body: { on: next } })
      toast({ title: next ? t(me.turnedOn, { count: d.count }) : t(me.turnedOff, { count: d.count }) })
    } catch (e) {
      setOn(!next)
      toast({ kind: 'error', title: errorMessage(e) })
    }
  }

  const statusText =
    status.kind === 'saving'
      ? t(me.saving)
      : status.kind === 'dirty'
        ? t(me.unsaved)
        : status.kind === 'conflict'
          ? t(me.conflict)
          : status.at
            ? status.kind === 'autosaved'
              ? t(me.autosaved, { time: clock(status.at) })
              : t(me.savedAt, { time: clock(status.at) })
            : t(me.notSaved)

  return (
    <div className="cmp rc-app" style={{ minHeight: '100vh' }}>
      <a href="#main" className="skip">
        {t(me.canvasLabel)}
      </a>
      <header className="cn-head" style={{ gap: 12 }}>
        <Link to="/w/$workspace" params={{ workspace: slug }} aria-label="ReputeCast" className="flex items-center">
          <LogoMark size={26} />
        </Link>
        <nav aria-label={t(me.crumbs)} className="flex flex-wrap items-center gap-1.5 text-[13px]">
          <Link to="/w/$workspace/grow" params={{ workspace: slug }} className="text-ink-3">
            {t(me.crumbGrow)}
          </Link>
          <span className="k">›</span>
          <Link to="/w/$workspace/grow/referrals" params={{ workspace: slug }} className="text-ink-3">
            {t(me.crumbReferrals)}
          </Link>
          <span className="k">›</span>
          <Link to="/w/$workspace/grow/referrals/milestones" params={{ workspace: slug }} search={{ program: d.programId }} className="text-ink-3">
            {t(me.crumbMilestones)}
          </Link>
          <span className="k">›</span>
          <span className="font-semibold">{t(me.crumbMilestone, { count: d.count })}</span>
        </nav>
        <span className="flex min-w-0 flex-col">
          <h1 className="m-0 text-[15px] font-semibold">{t(me.title, { count: d.count, reward: d.rewardName })}</h1>
          <span className="k">{statusText}</span>
        </span>
        <span className="ml-auto flex flex-wrap items-center gap-2">
          <span className="k">
            {t(me.from, { sender: d.sender })}
            {d.senderDefault ? <Badge tone="neutral">{t(me.fromDefault)}</Badge> : null}
          </span>
          <Badge tone="neutral">{t(me.stream)}</Badge>
          <Button onClick={() => setTesting(true)}>{t(me.sendTest)}</Button>
          {canEdit ? (
            <Button variant="primary" loading={status.kind === 'saving'} onClick={() => void save(true)}>
              {t(me.save)}
            </Button>
          ) : null}
        </span>
      </header>
      {status.kind === 'conflict' ? (
        <div className="px-7 pt-3">
          <Note tone="warn">
            <span className="flex flex-wrap items-center gap-2.5">
              {t(me.conflictText)}
              <Button size="sm" onClick={() => window.location.reload()}>
                {t(me.reload)}
              </Button>
            </span>
          </Note>
        </div>
      ) : null}
      {!canEdit ? (
        <div className="px-7 pt-3">
          <Note tone="neutral">
            <T msg={me.readOnly} values={{ mono }} />
          </Note>
        </div>
      ) : null}
      <main id="main" className="flex flex-wrap items-start gap-5 px-7 py-5">
        <div className="flex min-w-0 flex-[999_1_560px] flex-col gap-3">
          <div className="card grid gap-3 p-4" style={{ gridTemplateColumns: 'repeat(auto-fit, minmax(min(280px, 100%), 1fr))' }}>
            <Field label={t(me.subject)}>
              <Input
                value={subject}
                disabled={!canEdit}
                onChange={(e) => {
                  setSubject(e.target.value)
                  setStatus((s) => ({ kind: 'dirty', at: s.at }))
                }}
              />
            </Field>
            <Field label={t(me.preheader)} optional>
              <Input
                value={preheader}
                disabled={!canEdit}
                onChange={(e) => {
                  setPreheader(e.target.value)
                  setStatus((s) => ({ kind: 'dirty', at: s.at }))
                }}
              />
            </Field>
          </div>
          <div className="flex flex-wrap items-center justify-between gap-2">
            <Segmented
              label={t(me.view)}
              value={view}
              onChange={switchView}
              items={[
                { value: 'visual', label: t(me.visual) },
                { value: 'code', label: t(me.code) },
              ]}
            />
            {errors.length ? <Badge tone="warn">{t(me.compileErrors, { count: errors.length })}</Badge> : null}
          </div>
          {errors.length ? (
            <Note tone="warn">
              <ul className="m-0 pl-4">
                {errors.slice(0, 5).map((e, i) => (
                  <li key={i}>{t(me.compileLine, { line: e.line, message: e.message })}</li>
                ))}
              </ul>
            </Note>
          ) : null}
          <div className="card overflow-hidden" style={{ display: view === 'visual' ? undefined : 'none' }}>
            <div ref={container} className="cmp-canvas" role="region" aria-label={t(me.canvasLabel)} style={{ minHeight: 520 }} />
            {!ge ? <Skeleton h={520} /> : null}
          </div>
          {view === 'code' ? (
            <Textarea
              className="mono min-h-[520px] text-[12.5px]"
              spellCheck={false}
              aria-label={t(me.codeLabel)}
              value={code}
              disabled={!canEdit}
              onChange={(e) => {
                setCode(e.target.value)
                setStatus((s) => ({ kind: 'dirty', at: s.at }))
              }}
            />
          ) : null}
        </div>
        <aside className="flex min-w-0 flex-[1_1_320px] flex-col gap-3">
          <section className="card flex flex-col gap-3 p-4">
            <h2 className="gw-hd3">{t(me.thisEmail)}</h2>
            <label className="flex items-start justify-between gap-3">
              <span className="flex flex-col gap-0.5">
                <span className="font-semibold">{t(me.sendAt, { count: d.count })}</span>
                <span className="k">{t(me.sendAtText)}</span>
              </span>
              <input type="checkbox" className="tg" role="switch" aria-label={t(me.sendAtLabel, { count: d.count })} checked={on} disabled={!canEdit} onChange={(e) => void flip(e.target.checked)} />
            </label>
            <div className="flex flex-col gap-0.5">
              <span className="lbl">{t(me.sender)}</span>
              <span>{d.senderDefault ? t(me.senderValue, { sender: d.sender }) : d.sender}</span>
            </div>
            <div className="flex flex-col gap-0.5">
              <span className="lbl">{t(me.rules)}</span>
              <span>{t(me.rulesValue)}</span>
            </div>
          </section>
          <section className="card flex flex-col gap-2 p-4">
            <h2 className="gw-hd3">{t(me.rewardHd)}</h2>
            <span className="font-semibold">{d.rewardName}</span>
            <span className="k">
              {d.fileName
                ? t(me.rewardLine, { type: typeLabel(t, d.rewardType as Schemas['ReferralMilestoneView']['type']), file: `${fileKind(d.fileType)}, ${fileSize(d.fileSize)}` })
                : typeLabel(t, d.rewardType as Schemas['ReferralMilestoneView']['type'])}
            </span>
            <span className="k">{t(me.rewardText)}</span>
            <span>
              <Link to="/w/$workspace/grow/referrals/milestones" params={{ workspace: slug }} search={{ program: d.programId, edit: d.milestoneId }} className="btn sm">
                {t(me.editReward)}
              </Link>
            </span>
            <span className="k">
              {t(me.tokens)}: <span className="mono">reward.name · reward.description · reward.download_url · reward.code · reward.claim_url · reward.expires_at</span>
            </span>
          </section>
          {view === 'visual' && ge ? <BlocksCard ge={ge} selected={selected} canEdit={canEdit} /> : null}
          <PreviewCard d={d} compile={compile} contact={contact} setContact={setContact} rev={rev} subject={subject} preheader={preheader} />
          {d.versions.length ? (
            <section className="card flex flex-col gap-1.5 p-4">
              <h2 className="gw-hd3">{t(me.versions, { count: d.versions.length })}</h2>
              {d.versions.slice(0, 5).map((v) => (
                <span key={v.version} className="k">
                  {t(me.versionLine, { version: v.version, date: dayMonthDate(v.savedAt), by: v.savedBy })}
                </span>
              ))}
            </section>
          ) : null}
        </aside>
      </main>
      {testing ? <TestDialog d={d} compile={compile} subject={subject} preheader={preheader} contact={contact} onClose={() => setTesting(false)} /> : null}
      <UnsavedChangesGuard dirty={dirty} description={t(me.leave)} />
    </div>
  )
}

/**
 * Grow's blocks for this email (ComposerGrowBlocks frame 2): Reward,
 * Referral and Recommendations to add, and the selected block's settings.
 */
function BlocksCard({ ge, selected, canEdit }: { ge: ComposerEditor; selected: Component | null; canEdit: boolean }) {
  const t = useT()
  const add = (content: string) => {
    const body = ge.editor.getWrapper()?.findType('mj-body')[0]
    if (!body) return
    // Before the footer, the last section.
    const at = Math.max(0, body.components().length - 1)
    const added = body.append(content, { at })
    const first = (Array.isArray(added) ? added : [added])[0]
    if (first) ge.editor.select(first as Component)
  }
  return (
    <section className="card flex flex-col p-0">
      <h2 className="gw-hd3" style={{ padding: '14px 16px 4px' }}>
        {t(me.blocks)}
      </h2>
      <div className="blk-grid gw-cmp-grid">
        {growBlockDefs(true).map((b) => (
          <button key={b.id} type="button" className="blk rc" disabled={!canEdit} onClick={() => add(b.content)}>
            <BlockIcon name={b.icon} />
            {b.label}
          </button>
        ))}
      </div>
      <GrowBlocksNote milestone />
      {selected && isGrowBlock(selected) ? (
        <fieldset disabled={!canEdit} className="flex flex-col border-t border-line-2">
          <GrowBlockInspector key={selected.cid} c={selected} />
        </fieldset>
      ) : (
        <p className="k border-t border-line-2" style={{ margin: 0, padding: '12px 16px' }}>
          {t(me.selectBlock)}
        </p>
      )}
    </section>
  )
}

/** The email as a referrer gets it, rendered by the server. */
function PreviewCard({
  d,
  compile,
  contact,
  setContact,
  rev,
  subject,
  preheader,
}: {
  d: Detail
  compile: () => { html: string }
  contact: string
  setContact: (v: string) => void
  rev: number
  subject: string
  preheader: string
}) {
  const t = useT()
  const { slug } = useWorkspace()
  const people = $api.useQuery('get', `${GROW}/blocks/preview-contacts`, { params: { path: { workspace: slug }, query: { program: d.programId } } }).data
  const [html, setHtml] = useState('')
  // The canvas settles before the server renders it again.
  useEffect(() => {
    const h = window.setTimeout(() => setHtml(compile().html), 600)
    return () => window.clearTimeout(h)
  }, [rev, subject, preheader, compile])
  const q = $api.useQuery(
    'post',
    `${GROW}/milestones/{milestone}/email/preview`,
    { params: { path: { workspace: slug, milestone: d.milestoneId } }, body: { html: html || d.html, subject, preheader, contactId: contact || undefined } },
    { placeholderData: (prev) => prev },
  )
  return (
    <section className="card flex flex-col gap-2 p-4">
      <Field label={t(me.previewAs)}>
        <Select value={contact} onChange={(e) => setContact(e.target.value)}>
          <option value="">{t(me.previewSample, { count: d.count })}</option>
          {people?.items.map((p) => (
            <option key={p.id} value={p.id}>
              {p.name || p.email}
            </option>
          ))}
        </Select>
      </Field>
      {q.data ? (
        <>
          <span className="k">{q.data.for ? t(me.inboxPreview, { name: q.data.for }) : t(me.inboxPreviewSample, { count: d.count })}</span>
          <span className="font-semibold">{q.data.subject}</span>
          <div className="overflow-hidden rounded-[10px] border border-line">
            <iframe title={t(me.previewFrame)} srcDoc={q.data.html} sandbox="allow-same-origin" className="block w-full border-0" style={{ height: 520 }} />
          </div>
        </>
      ) : (
        <Skeleton h={520} r={10} />
      )}
    </section>
  )
}

function TestDialog({
  d,
  compile,
  subject,
  preheader,
  contact,
  onClose,
}: {
  d: Detail
  compile: () => { html: string }
  subject: string
  preheader: string
  contact: string
  onClose: () => void
}) {
  const t = useT()
  const { slug } = useWorkspace()
  const user = useMe().data?.user
  const [to, setTo] = useState(user?.email ?? '')
  const [busy, setBusy] = useState(false)
  const send = async () => {
    setBusy(true)
    try {
      const list = to
        .split(',')
        .map((x) => x.trim())
        .filter(Boolean)
        .slice(0, 5)
      const { data } = await http.POST('/api/v1/workspaces/{workspace}/grow/milestones/{milestone}/email/test', {
        params: { path: { workspace: slug, milestone: d.milestoneId } },
        body: { to: list, html: compile().html, subject, preheader, contactId: contact || undefined },
      })
      toast({ kind: 'success', title: t(me.testSent, { to: list.join(', ') }), detail: data?.provider ? t(me.testSentDetail, { provider: data.provider }) : undefined })
      onClose()
    } catch (e) {
      toast({ kind: 'error', title: errorMessage(e) })
    } finally {
      setBusy(false)
    }
  }
  return (
    <Dialog
      open
      onOpenChange={(o) => !o && onClose()}
      title={t(me.testTitle)}
      footer={
        <>
          <Button onClick={onClose}>{t(me.cancel)}</Button>
          <Button variant="primary" loading={busy} disabled={!to.trim()} onClick={() => void send()}>
            {t(me.testSend)}
          </Button>
        </>
      }
    >
      <Field label={t(me.testTo)} hint={t(me.testToHint)}>
        <Input value={to} onChange={(e) => setTo(e.target.value)} />
      </Field>
    </Dialog>
  )
}
