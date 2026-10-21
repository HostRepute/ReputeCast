import { useMemo } from 'react'
import { Dialog } from '@/components/ui/Dialog'
import { useT } from '@/i18n'
import { modKeyLabel } from '@/lib/shortcuts'
import { useBuilder } from './context'
import { MediaDialog } from './MediaDialog'
import { diffPages } from './model'
import { m } from './messages'
import { PreviewDialog } from './PreviewDialog'
import { ApprovalDialog, BlockDiffRows, PublishDialog, UnpublishDialog } from './PublishDialog'
import { VersionsDialog } from './VersionsDialog'

/** A conflict's Compare: their saved version and the edits on the canvas, block by block. */
function CompareDialog({ open, onClose }: { open: boolean; onClose: () => void }) {
  const t = useT()
  const b = useBuilder()
  const s = b.status
  const theirs = b.conflict?.html
  const blocks = useMemo(() => (open && theirs !== undefined && b.pe ? diffPages(theirs, b.pe.getHtml()) : null), [open, theirs, b.pe])
  const who = s.kind === 'conflict' ? s.theirs.by || t(m.someone) : t(m.someone)
  const version = s.kind === 'conflict' ? s.theirs.version : 0
  return (
    <Dialog open={open} onOpenChange={(o) => !o && onClose()} title={t(m.compareTitle, { name: who, version })} description={t(m.compareLead, { name: who })} size="lg">
      {!blocks ? (
        <p className="k">{t(m.compareLoading)}</p>
      ) : !blocks.length ? (
        <p className="k">{t(m.compareNone)}</p>
      ) : (
        <div className="diff pb-diff">
          <BlockDiffRows blocks={blocks} />
        </div>
      )}
    </Dialog>
  )
}

function ShortcutsDialog({ open, onClose }: { open: boolean; onClose: () => void }) {
  const t = useT()
  const mod = modKeyLabel()
  const rows: [string, string][] = [
    [t(m.scSave), `${mod} S`],
    [t(m.scUndo), `${mod} Z`],
    [t(m.scRedo), `⇧ ${mod} Z`],
    [t(m.scDelete), '⌫'],
    [t(m.scEdit), t(m.scEditKeys)],
    [t(m.scList), `${mod} /`],
  ]
  return (
    <Dialog open={open} onOpenChange={(o) => !o && onClose()} title={t(m.shortcutsTitle)} size="sm">
      <dl className="kv">
        {rows.map(([what, keys]) => (
          <div key={what} className="contents">
            <dt>
              <span className="mk">{keys}</span>
            </dt>
            <dd>{what}</dd>
          </div>
        ))}
      </dl>
    </Dialog>
  )
}

/** The builder's dialogs: one at a time. */
export function Dialogs() {
  const b = useBuilder()
  const d = b.dialog
  const close = () => b.openDialog(null)
  return (
    <>
      <PublishDialog open={d?.kind === 'publish'} onClose={close} key={d?.kind === 'publish' ? 'publish-open' : 'publish'} />
      <ApprovalDialog open={d?.kind === 'approval'} onClose={close} />
      <UnpublishDialog open={d?.kind === 'unpublish'} onClose={close} />
      <VersionsDialog open={d?.kind === 'versions'} onClose={close} />
      {d?.kind === 'preview' ? <PreviewDialog open onClose={close} version={d.version} /> : null}
      {d?.kind === 'media' ? <MediaDialog open onClose={close} pick={d.pick} /> : null}
      <CompareDialog open={d?.kind === 'compare'} onClose={close} />
      <ShortcutsDialog open={d?.kind === 'shortcuts'} onClose={close} />
    </>
  )
}
