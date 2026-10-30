import { useQueryClient } from '@tanstack/react-query'
import { Link } from '@tanstack/react-router'
import type { Component } from 'grapesjs'
import { useId, useState, type ReactNode } from 'react'
import { $api, errorMessage, http, type Schemas } from '@/api/client'
import { Dialog } from '@/components/ui/Dialog'
import { toast } from '@/components/ui/Toast'
import { T, useT } from '@/i18n'
import { common } from '@/i18n/common'
import { useComposer } from './context'
import type { BlockDef } from './editor/blocks'
import { sectionLabel, sectionOf, sectionWord, tagOf } from './editor/labels'
import { isNamedSynced, replaceSection, SYNCED_ATTR, syncedSectionOf, unlinkSynced } from './editor/synced'
import { mbk } from './brandkit.messages'

/*
 * The brand kit in the composer (CO-11, ComposerEmpty · Composer · ComposerWrite):
 * the gallery's "Start from your brand kit" banner, the synced Header and
 * Footer and the named synced blocks (R4.A3) as blocks, synced sections kept
 * read-only on the canvas (they change in Settings › Brand kit, and every
 * draft holding them follows) or unlinked, and any section saved as a new
 * synced block.
 */

/** The workspace's brand kit (defaults until it's saved). */
export function useBrandKit() {
  const c = useComposer()
  return $api.useQuery('get', '/api/v1/workspaces/{workspace}/brand-kit', { params: { path: { workspace: c.slug } } }, { staleTime: 60_000 })
}

/** "Start from your brand kit" (ComposerEmpty), until the kit is saved. */
export function BrandKitBanner() {
  const t = useT()
  const c = useComposer()
  const kit = useBrandKit().data
  if (!kit || kit.saved) return null
  return (
    <div className="bkbanner">
      <span className="bki" aria-hidden="true">
        <svg width="18" height="18" viewBox="0 0 16 16" fill="none" stroke="currentColor" strokeWidth="1.6" strokeLinecap="round" strokeLinejoin="round">
          <path d="M8 1.8a6.2 6.2 0 1 0 0 12.4c.9 0 1.3-.6 1.3-1.2 0-.8-.6-1.1-.6-1.8 0-.7.6-1.1 1.3-1.1h1.3a2.9 2.9 0 0 0 2.9-2.9C14.2 4.4 11.4 1.8 8 1.8z" />
          <circle cx="5" cy="7" r=".8" />
          <circle cx="8" cy="4.8" r=".8" />
          <circle cx="11" cy="7" r=".8" />
        </svg>
      </span>
      <span className="bkt">
        <T msg={mbk.banner} values={{ b: (chunks: ReactNode) => <b>{chunks}</b> }} />
      </span>
      <Link to="/w/$workspace/settings/brand-kit" params={{ workspace: c.slug }} className="btn">
        {t(mbk.setUp)}
      </Link>
    </div>
  )
}

/** "Shown in default colours until you set up the brand kit", over the starters. */
export function useDefaultColoursNote(): string | null {
  const t = useT()
  const kit = useBrandKit().data
  return kit && !kit.saved ? t(mbk.defaultColours) : null
}

/** The synced Header and Footer, and the named synced blocks, as blocks (ReputeCast blocks, "Synced"). */
export function syncedBlockDefs(kit: Schemas['BrandKit'] | undefined, t: ReturnType<typeof useT>): BlockDef[] {
  return (kit?.synced ?? []).map((b) => {
    const named = isNamedSynced(b.kind)
    return {
      id: `rc-synced-${b.kind}`,
      label: named ? b.name : b.kind === 'header' ? t(mbk.syncedHeader) : t(mbk.syncedFooter),
      badge: t(mbk.synced),
      group: 'reputecast' as const,
      icon: named ? 'saved' : b.kind === 'header' ? 'rcHeader' : 'rcFooter',
      kind: 'section' as const,
      content: b.mjml,
      keywords: named ? `synced brand kit ${b.name.toLowerCase()}` : 'synced brand kit header footer',
    }
  })
}

/** A named synced block's name, from the kit; null once it's deleted (undefined while the kit loads). */
function useSyncedName(kind: string): string | null | undefined {
  const kit = useBrandKit().data
  if (!kit) return undefined
  return kit.synced.find((b) => b.kind === kind)?.name ?? null
}

/**
 * In the settings of a synced section's block: what it is, where it
 * changes (ComposerWrite · "Edit in Brand kit"), and unlinking it to edit
 * it in this email only (R4.A3).
 */
export function SyncedNotice({ component }: { component: Component | null }) {
  const t = useT()
  const c = useComposer()
  const synced = syncedSectionOf(component)
  const named = synced ? isNamedSynced(synced.kind) : false
  const name = useSyncedName(synced?.kind ?? '')
  if (!synced) return null
  // A named block deleted from the kit: nothing follows it any more.
  const gone = named && name === null
  const unlink = () => {
    if (!c.editor) return
    unlinkSynced(c.editor, synced.section)
    toast({ kind: 'success', title: t(mbk.unlinked), detail: t(mbk.unlinkedDetail) })
  }
  return (
    <div className="note i" role="note" style={{ margin: '12px 16px 0' }}>
      <div className="flex flex-col gap-1.5">
        {gone ? (
          <span>
            <b>{t(mbk.syncedGone)}</b> {t(mbk.syncedGoneBody)}
          </span>
        ) : (
          <span>
            <b>{named ? t(mbk.syncedNamedNotice, { name: name ?? t(mbk.syncedBlock) }) : synced.kind === 'header' ? t(mbk.syncedHeaderNotice) : t(mbk.syncedFooterNotice)}</b> {t(mbk.syncedNoticeBody)}
          </span>
        )}
        <span className="flex flex-wrap items-center gap-x-4 gap-y-1">
          {gone ? null : (
            <Link to="/w/$workspace/settings/brand-kit" params={{ workspace: c.slug }} className="font-semibold underline">
              {t(mbk.editInBrandKit)}
            </Link>
          )}
          {c.canEdit && c.editor ? (
            <button type="button" className="btn sm" onClick={unlink}>
              {t(mbk.unlink)}
            </button>
          ) : null}
        </span>
      </div>
    </div>
  )
}

/**
 * "Save as synced block" for the selected section (R4.A3): it goes to the
 * brand kit under a name, and the section here becomes that block. Other
 * emails insert it from Blocks › Your blocks; one edit in the brand kit
 * reaches every draft and scheduled email holding it.
 */
export function SaveAsSynced({ component }: { component: Component | null }) {
  const t = useT()
  const c = useComposer()
  const section = sectionOf(component) ?? (component && tagOf(component.parent()) === 'mj-body' ? component : undefined)
  const [open, setOpen] = useState(false)
  if (!section || c.kind === 'transactional' || !c.canEdit || !c.editor) return null
  // A synced section, or one holding a synced block, stays what it is.
  let holdsSynced = false
  section.onAll((x: Component) => {
    if ((x.getAttributes() as Record<string, unknown>)[SYNCED_ATTR]) holdsSynced = true
  })
  if (holdsSynced || syncedSectionOf(section)) return null
  const tag = tagOf(section)
  if (tag !== 'mj-section' && tag !== 'mj-wrapper') return null
  return (
    <div className="flex flex-col gap-1.5" style={{ padding: '12px 16px 0' }}>
      <span className="font-semibold">{t(mbk.saveSyncedTitle)}</span>
      <span className="k">{t(mbk.saveSyncedLead)}</span>
      <button type="button" className="btn sm self-start" onClick={() => setOpen(true)}>
        {t(mbk.saveSynced)}
      </button>
      {open ? <SaveSyncedDialog section={section} onClose={() => setOpen(false)} /> : null}
    </div>
  )
}

function SaveSyncedDialog({ section, onClose }: { section: Component; onClose: () => void }) {
  const t = useT()
  const c = useComposer()
  const qc = useQueryClient()
  const id = useId()
  const word = sectionWord(sectionLabel(section))
  const [name, setName] = useState(() => word.charAt(0).toLocaleUpperCase() + word.slice(1))
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState('')
  const save = async () => {
    setBusy(true)
    setError('')
    try {
      const { data } = await http.POST('/api/v1/workspaces/{workspace}/brand-kit/synced', {
        params: { path: { workspace: c.slug } },
        body: { name: name.trim(), mjml: section.toHTML() },
      })
      if (data && c.editor) replaceSection(c.editor, section, data.mjml)
      void qc.invalidateQueries({ queryKey: ['get', '/api/v1/workspaces/{workspace}/brand-kit'] })
      toast({ kind: 'success', title: t(mbk.savedSynced, { name: name.trim() }), detail: t(mbk.savedSyncedDetail) })
      onClose()
    } catch (e) {
      setError(errorMessage(e))
    } finally {
      setBusy(false)
    }
  }
  return (
    <Dialog
      open
      onOpenChange={(o) => !o && !busy && onClose()}
      title={t(mbk.saveSyncedDialog)}
      description={t(mbk.saveSyncedDialogLead)}
      size="sm"
      footer={
        <>
          <button type="button" className="btn" onClick={onClose} disabled={busy}>
            {t(common.cancel)}
          </button>
          <button type="button" className="btn pri" disabled={!name.trim() || busy} aria-busy={busy || undefined} onClick={() => void save()}>
            {t(mbk.saveSyncedConfirm)}
          </button>
        </>
      }
    >
      <label className="lbl" htmlFor={`${id}-name`}>
        {t(mbk.syncedName)}
        <input id={`${id}-name`} className="in" value={name} onChange={(e) => setName(e.target.value)} maxLength={80} aria-invalid={error ? true : undefined} />
      </label>
      {error ? (
        <p className="m-0 mt-2 text-bad-fg" role="alert">
          {t(mbk.saveSyncedFailed, { error })}
        </p>
      ) : null}
    </Dialog>
  )
}
