import { Link } from '@tanstack/react-router'
import { Badge } from '@/components/ui/Badge'
import { Button } from '@/components/ui/Button'
import { Note } from '@/components/ui/Note'
import { useT } from '@/i18n'
import { clock } from '@/lib/format'
import { useMe, useOptionalWorkspace } from '@/lib/session'
import { useComposer } from '../context'
import { ToolIcon } from '../icons'
import { m } from './messages'
import { useTx } from './state'

/** "Account · Essential" */
export function useCategoryLabel(): string {
  const t = useT()
  const { template } = useTx()
  const c = template.category
  if (!c) return t(m.noCategory)
  return c.kind === 'optional' ? t(m.categoryOptional, { name: c.name }) : t(m.categoryEssential, { name: c.name })
}

/** The live version and the draft: "v3 live · draft matches v3", "v1 live · draft v2 unpublished". */
function VersionBadge() {
  const t = useT()
  const { template: tpl } = useTx()
  if (tpl.archived) return <Badge tone="neutral">{t(m.archived)}</Badge>
  if (!tpl.liveVersion)
    return (
      <Badge tone="warn" dot>
        {t(m.neverPublished)}
      </Badge>
    )
  if (!tpl.draftChanges)
    return (
      <Badge tone="ok" dot>
        {t(m.liveMatches, { version: tpl.liveVersion })}
      </Badge>
    )
  return (
    <Badge tone="warn" dot>
      {t(m.liveDraft, { live: tpl.liveVersion, next: tpl.versionsCount + 1 })}
    </Badge>
  )
}

/** Autosave state, as the composer's header says it. */
function SaveState() {
  const t = useT()
  const { status } = useComposer()
  const { canEdit } = useTx()
  if (!canEdit) return null
  let text: string
  let tone = ''
  switch (status.kind) {
    case 'saving':
      text = t(m.saving)
      break
    case 'dirty':
      text = status.at ? t(m.editingSince, { time: clock(status.at) }) : t(m.editing)
      break
    case 'offline':
      text = t(m.offline)
      tone = 'txe-bad'
      break
    case 'error':
      text = t(m.notSaved)
      tone = 'txe-bad'
      break
    case 'conflict':
      text = t(m.conflict)
      tone = 'txe-warn'
      break
    case 'saved':
      text = t(m.autosaved, { time: clock(status.at) })
      break
    default:
      text = t(m.notSavedYet)
  }
  return (
    <span className={`k ${tone}`} role="status" aria-live="polite">
      {text}
    </span>
  )
}

/** Who is looking, on the read-only header: "Aisha Bello · Viewer". */
function ViewerLine() {
  const t = useT()
  const ws = useOptionalWorkspace()
  const { data } = useMe()
  const user = data?.user
  if (!user || !ws) return null
  return <span className="k">{t(m.viewerLine, { name: user.name || user.email, role: ws.me.workspace.roleName })}</span>
}

/** Publish, with why it can't happen yet. */
function PublishButton() {
  const t = useT()
  const c = useComposer()
  const tx = useTx()
  const tpl = tx.template
  const next = tpl.versionsCount + 1
  const blocked = (tx.checks?.checks ?? []).some((x) => x.severity === 'block' && (x.id === 'liquid.syntax' || x.id === 'content.compile'))
  const unchanged = !!tpl.liveVersion && !tpl.draftChanges && c.status.kind !== 'dirty' && c.status.kind !== 'conflict'
  const reason = !tx.canPublish ? tpl.access.publishReason || t(m.publishNeedsPermission) : tpl.archived ? t(m.publishArchived) : undefined
  // A Viewer can neither edit nor publish: the board shows no Publish button.
  if (!tx.canEdit && !tx.canPublish) return null
  if (unchanged)
    return (
      <span className="inline-flex items-center gap-2">
        <span className="k">{t(m.noChanges, { version: tpl.liveVersion ?? 0 })}</span>
        <Button variant="primary" disabled>
          {t(m.publish)}
        </Button>
      </span>
    )
  if (blocked && tx.canPublish)
    return (
      <span className="inline-flex items-center gap-2">
        <span className="k">{t(m.needsRender)}</span>
        <Button variant="primary" disabled>
          {t(m.publish)}
        </Button>
      </span>
    )
  return (
    <Button variant="primary" disabledReason={reason} disabled={c.isEmpty && !tx.htmlMode} onClick={tx.openPublish}>
      {t(m.publishVersion, { version: next })}
    </Button>
  )
}

/**
 * The editor's top bar (TransactionalTemplate): back, the name, ID,
 * category, live version and draft, then Versions and Publish. The badges
 * row leaves room for X05's language switcher.
 */
export function TxHeader() {
  const t = useT()
  const c = useComposer()
  const tx = useTx()
  const tpl = tx.template
  const category = useCategoryLabel()
  return (
    <header className="cmp-head">
      <Link to="/w/$workspace/transactional" params={{ workspace: c.slug }} className="btn ic" aria-label={t(m.back)}>
        <ToolIcon name="back" size={16} />
      </Link>
      <div className="flex min-w-0 flex-col">
        {tx.canEdit ? (
          <>
            <label className="k" htmlFor="tname" style={{ fontSize: 11 }}>
              {t(m.kindLabel)}
            </label>
            <input
              id="tname"
              className="cmp-name"
              value={c.name}
              placeholder={t(m.untitled)}
              onChange={(e) => c.setName(e.target.value)}
              onBlur={() => c.commitName()}
              onKeyDown={(e) => {
                if (e.key === 'Enter') (e.target as HTMLInputElement).blur()
                if (e.key === 'Escape') {
                  c.setName(tpl.name)
                  ;(e.target as HTMLInputElement).blur()
                }
              }}
              maxLength={120}
            />
          </>
        ) : (
          <>
            <span className="k" style={{ fontSize: 11 }}>
              {t(m.kindLabel)}
            </span>
            <span className="text-[15px] font-bold">{c.name || t(m.untitled)}</span>
          </>
        )}
      </div>
      <span className="txe-id" title={tpl.keyLocked ? t(m.idLocked) : t(m.idEditable)}>
        {t(m.idChip, { key: tpl.key })}
      </span>
      <Badge tone={tpl.category?.kind === 'optional' ? 'info' : 'neutral'}>{category}</Badge>
      <VersionBadge />
      {tx.canEdit ? null : <Badge tone="neutral">{t(m.readOnly)}</Badge>}
      {tx.canEdit ? null : <ViewerLine />}
      <SaveState />
      <div className="flex flex-wrap items-center gap-2" style={{ marginLeft: 'auto' }}>
        <Button onClick={tx.openVersions}>{t(m.versions)}</Button>
        <PublishButton />
      </div>
    </header>
  )
}

/** A Viewer's note (TransactionalTemplateStates · Aisha Bello): every tab opens, nothing edits. */
export function ReadOnlyBanner() {
  const t = useT()
  const { template } = useTx()
  return (
    <div className="txe-banner">
      <Note tone="neutral">{template.access.editReason || t(m.readOnlyNote)}</Note>
    </div>
  )
}
