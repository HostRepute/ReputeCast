import type { Component } from 'grapesjs'
import { useEffect, useState, type ReactNode } from 'react'
import { Dialog } from '@/components/ui/Dialog'
import { T, useT } from '@/i18n'
import { useComposer } from './context'
import { CommentsTab, useComments } from './Comments'
import { useReadiness } from './Header'
import { ToolIcon } from './icons'
import { InboxStrip } from './InboxStrip'
import { ChecksContent } from './RightPanel'
import { SendTestButton } from './SendTest'
import { tagOf } from './editor/labels'
import { mh } from './chrome.messages'

/**
 * Quick review on phones (CO-14, ComposerPhone board): the email reads as
 * it is, texts can be edited in place, and Inbox, Checks, Comments and Send
 * test open as sheets. Layout editing waits for a larger screen.
 */

type Sheet = 'edit' | 'inbox' | 'checks' | 'comments' | 'test' | null

/** "Quick review on this screen…": above the email on phones. */
export function QuickReviewNote() {
  return (
    <div className="qnote" role="note">
      <ToolIcon name="info" size={15} strokeWidth={1.6} />
      <span>
        <T msg={mh.quickNote} values={{ b: (chunks: ReactNode[]) => <b>{chunks}</b> }} />
      </span>
    </div>
  )
}

/** While Edit text is on, a tap on a text starts editing it (texts are outlined). */
function useTapToEdit(on: boolean) {
  const c = useComposer()
  const ed = c.editor
  useEffect(() => {
    if (!ed || !on) return
    const body = ed.Canvas.getBody()
    body?.classList.add('rc-qedit')
    const onSelect = (comp: Component) => {
      if (tagOf(comp) !== 'mj-text' || !c.canEdit) return
      ;(comp.view as unknown as { onActive?: (e?: Event) => void } | undefined)?.onActive?.()
    }
    ed.on('component:selected', onSelect)
    return () => {
      body?.classList.remove('rc-qedit')
      ed.off('component:selected', onSelect)
    }
  }, [ed, on, c.canEdit])
}

export function QuickReviewBar() {
  const c = useComposer()
  const t = useT()
  const [sheet, setSheet] = useState<Sheet>(null)
  const ready = useReadiness()
  const comments = useComments().data?.open ?? 0
  const [editing, setEditing] = useState(false)
  useTapToEdit(editing)
  const close = () => setSheet(null)
  const item = (key: Exclude<Sheet, null>, icon: string, label: string, badge?: number, aria?: string) => (
    <button type="button" aria-label={aria} aria-pressed={key === 'edit' ? editing : undefined} onClick={() => (key === 'edit' ? setEditing((e) => !e) : setSheet(key))}>
      <span className="qi" aria-hidden="true">
        <ToolIcon name={icon} size={18} strokeWidth={1.6} />
        {/* Drawn from data-n, so the button's name (its aria-label) contains all of its text (WCAG 2.5.3). */}
        {badge ? <span className="qn" data-n={badge} /> : null}
      </span>
      {label}
    </button>
  )
  return (
    <>
      <div className="qbar" role="group" aria-label={t(mh.quickGroup)}>
        {item('edit', 'pencil', t(mh.editText))}
        {item('inbox', 'mail', t(mh.inbox))}
        {item('checks', 'shield', t(mh.checks), ready.blocks + ready.warnings, t(mh.sheetChecks, { blocks: ready.blocks, warnings: ready.warnings }))}
        {item('comments', 'comment', t(mh.comments), comments, t(mh.sheetComments, { count: comments }))}
        {item('test', 'send', t(mh.sendTest))}
      </div>
      {editing ? (
        <div className="qedit" role="status">
          <span>{t(mh.editTextLead)}</span>
          <button type="button" className="btn sm" onClick={() => setEditing(false)}>
            {t(mh.inboxDone)}
          </button>
        </div>
      ) : null}
      <Dialog open={sheet === 'inbox'} onOpenChange={(o) => !o && close()} title={t(mh.inbox)} className="qsheet">
        <InboxStrip phoneSheet />
      </Dialog>
      <Dialog open={sheet === 'checks'} onOpenChange={(o) => !o && close()} title={t(mh.sheetChecks, { blocks: ready.blocks, warnings: ready.warnings })} className="qsheet">
        <ChecksContent />
      </Dialog>
      <Dialog open={sheet === 'comments'} onOpenChange={(o) => !o && close()} title={t(mh.sheetComments, { count: comments })} className="qsheet">
        <CommentsTab />
      </Dialog>
      <Dialog open={sheet === 'test'} onOpenChange={(o) => !o && close()} title={t(mh.sendTest)} className="qsheet">
        <div className="flex flex-col gap-2 p-3">
          <SendTestButton disabled={c.isEmpty} />
        </div>
      </Dialog>
    </>
  )
}
