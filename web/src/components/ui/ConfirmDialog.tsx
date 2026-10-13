import { useBlocker } from '@tanstack/react-router'
import { useId, useState, type ReactNode } from 'react'
import { errorMessage } from '@/api/client'
import { cn } from '@/lib/cn'
import { count } from '@/lib/format'
import { Badge, type Tone } from './Badge'
import { Button } from './Button'
import { Dialog } from './Dialog'
import { Note } from './Note'

/**
 * Confirmation dialogs (Dialogs board). The dialog states the blast radius
 * first, offers the safer alternative, and asks for a deliberate step where
 * the action can't be undone: a typed name or a required reason.
 *
 *   destructive  tone "bad": delete list, cancel send, remove provider, delete workspace
 *   protective   tone "warn" or "ok": unsaved changes, revoke key, resume sending
 *   bulk result  BulkResultDialog: what was done, what was skipped and why, Undo
 */

/** One "badge + sentence" line inside a dialog (Affects, Grace period, Keeps…). */
export function DialogRow({ tone = 'warn', label, children, aside, first }: { tone?: Tone; label: ReactNode; children: ReactNode; aside?: ReactNode; first?: boolean }) {
  return (
    <div className={cn('drow', first && 'first')}>
      <Badge tone={tone}>{label}</Badge>
      <span className="min-w-0 flex-1">{children}</span>
      {aside}
    </div>
  )
}

/** A radio or checkbox choice inside a dialog ("Cancel the remaining 12,010" / "Pause instead"). */
export function DialogOption({
  type = 'radio',
  name,
  checked,
  onChange,
  title,
  description,
}: {
  type?: 'radio' | 'checkbox'
  name?: string
  checked: boolean
  onChange: (checked: boolean) => void
  title: ReactNode
  description?: ReactNode
}) {
  return (
    <label className="dopt">
      <input type={type} name={name} checked={checked} onChange={(e) => onChange(e.target.checked)} />
      <span className="min-w-0">
        {type === 'radio' ? <b>{title}</b> : title}
        {description ? (
          <>
            <br />
            <span className="k">{description}</span>
          </>
        ) : null}
      </span>
    </label>
  )
}

export interface ConfirmDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  /** bad: destructive; warn: protective; ok: resuming something; info: neutral. */
  tone?: Tone
  /** The action in two or three words: "Delete list", "Revoke API key". */
  badge: ReactNode
  title: ReactNode
  /** The consequences, in numbers. */
  description: ReactNode
  /** Rows, options and fields between the consequences and the actions. */
  children?: ReactNode
  confirmLabel: ReactNode
  /** Shown while onConfirm runs ("Deleting…"). */
  busyLabel?: ReactNode
  /** Destructive actions use the red button; everything else the teal one. */
  confirmVariant?: 'danger' | 'primary'
  /** Resolves when done; a rejection keeps the dialog open with the error. */
  onConfirm: (input: { reason?: string }) => void | Promise<unknown>
  cancelLabel?: ReactNode
  /** The safer alternative, before Cancel ("Export everything first"). */
  secondary?: ReactNode
  /** Asks the person to type a name before the button works. */
  typedConfirmation?: { label: ReactNode; expected: string; hint?: ReactNode }
  /** Asks for a reason, recorded with the action (emergency bypass, 2FA reset). */
  reason?: { label: ReactNode; hint?: ReactNode; minLength?: number }
}

export function ConfirmDialog({
  open,
  onOpenChange,
  tone = 'bad',
  badge,
  title,
  description,
  children,
  confirmLabel,
  busyLabel,
  confirmVariant = tone === 'bad' ? 'danger' : 'primary',
  onConfirm,
  cancelLabel = 'Cancel',
  secondary,
  typedConfirmation,
  reason,
}: ConfirmDialogProps) {
  const id = useId()
  const [typed, setTyped] = useState('')
  const [why, setWhy] = useState('')
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState<string>()

  const minReason = reason?.minLength ?? 10
  const blocked =
    typedConfirmation && typed.trim() !== typedConfirmation.expected
      ? `Type ${typedConfirmation.expected} to confirm.`
      : reason && why.trim().length < minReason
        ? `Give a reason of at least ${minReason} characters.`
        : undefined

  const close = (next: boolean) => {
    if (busy) return
    if (!next) {
      setTyped('')
      setWhy('')
      setError(undefined)
    }
    onOpenChange(next)
  }

  const confirm = async () => {
    if (blocked || busy) return
    setError(undefined)
    setBusy(true)
    try {
      await onConfirm({ reason: reason ? why.trim() : undefined })
      setBusy(false)
      close(false)
    } catch (e) {
      setBusy(false)
      setError(errorMessage(e))
    }
  }

  return (
    <Dialog
      open={open}
      onOpenChange={close}
      layout="confirm"
      // A destructive step is never taken by a stray click outside.
      dismissible={tone !== 'bad' && !busy}
      eyebrow={<Badge tone={tone}>{badge}</Badge>}
      title={title}
      description={description}
      footer={
        <>
          {secondary}
          <Button onClick={() => close(false)} disabled={busy}>
            {cancelLabel}
          </Button>
          <Button variant={confirmVariant === 'danger' ? 'danger-primary' : 'primary'} onClick={confirm} loading={busy} busyLabel={busyLabel} disabledReason={blocked}>
            {confirmLabel}
          </Button>
        </>
      }
    >
      {children}
      {typedConfirmation ? (
        <div className="lblc">
          <label htmlFor={`${id}-typed`}>{typedConfirmation.label}</label>
          <input
            id={`${id}-typed`}
            className="in"
            value={typed}
            onChange={(e) => setTyped(e.target.value)}
            autoComplete="off"
            spellCheck={false}
            aria-describedby={typedConfirmation.hint ? `${id}-typed-h` : undefined}
          />
          {typedConfirmation.hint ? (
            <span className="k" id={`${id}-typed-h`}>
              {typedConfirmation.hint}
            </span>
          ) : null}
        </div>
      ) : null}
      {reason ? (
        <div className="lblc">
          <label htmlFor={`${id}-reason`}>{reason.label}</label>
          <textarea
            id={`${id}-reason`}
            className="in"
            value={why}
            onChange={(e) => setWhy(e.target.value)}
            aria-describedby={reason.hint ? `${id}-reason-h` : undefined}
            required
          />
          {reason.hint ? (
            <span className="k" id={`${id}-reason-h`}>
              {reason.hint}
            </span>
          ) : null}
        </div>
      ) : null}
      {error ? <Note tone="bad">{error}</Note> : null}
    </Dialog>
  )
}

/**
 * The result of a bulk action that partly failed: what was done, what was
 * skipped and why, and Undo while it is safe.
 */
export function BulkResultDialog({
  open,
  onOpenChange,
  title,
  description,
  done,
  skipped,
  onUndo,
  undoLabel,
}: {
  open: boolean
  onOpenChange: (open: boolean) => void
  title: ReactNode
  description?: ReactNode
  done: { count: number; text: ReactNode }
  skipped?: { count: number; text: ReactNode }
  onUndo?: () => void | Promise<unknown>
  undoLabel?: ReactNode
}) {
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState<string>()
  const partial = Boolean(skipped && skipped.count > 0)
  const undo = async () => {
    if (!onUndo) return
    setBusy(true)
    setError(undefined)
    try {
      await onUndo()
      setBusy(false)
      onOpenChange(false)
    } catch (e) {
      setBusy(false)
      setError(errorMessage(e))
    }
  }
  return (
    <Dialog
      open={open}
      onOpenChange={onOpenChange}
      layout="confirm"
      eyebrow={<Badge tone={partial ? 'info' : 'ok'}>{partial ? 'Bulk action · partial result' : 'Bulk action · done'}</Badge>}
      title={title}
      description={description}
      footer={
        <>
          {onUndo ? (
            <Button onClick={undo} loading={busy}>
              {undoLabel ?? `Undo for the ${count(done.count)}`}
            </Button>
          ) : null}
          <Button variant="primary" onClick={() => onOpenChange(false)} disabled={busy}>
            Done
          </Button>
        </>
      }
    >
      <DialogRow tone="ok" label={count(done.count)} first>
        {done.text}
      </DialogRow>
      {partial && skipped ? (
        <DialogRow tone="bad" label={count(skipped.count)}>
          {skipped.text}
        </DialogRow>
      ) : null}
      {error ? <Note tone="bad">{error}</Note> : null}
    </Dialog>
  )
}

/**
 * "Leave the composer?" when navigating away with unsaved changes: Stay,
 * Leave without saving, or Save and leave. Pass `dirty` and how to save;
 * the browser's own prompt covers closing the tab.
 */
export function UnsavedChangesGuard({
  dirty,
  title = 'Leave without saving?',
  description,
  saveLabel = 'Save and leave',
  onSave,
}: {
  dirty: boolean
  title?: ReactNode
  /** What would be lost: "You changed the subject line and two blocks since the last autosave 40 seconds ago." */
  description: ReactNode
  saveLabel?: ReactNode
  onSave?: () => Promise<unknown>
}) {
  const blocker = useBlocker({ shouldBlockFn: () => dirty, enableBeforeUnload: dirty, withResolver: true })
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState<string>()
  if (blocker.status !== 'blocked') return null
  const save = async () => {
    if (!onSave) return
    setBusy(true)
    setError(undefined)
    try {
      await onSave()
      blocker.proceed()
    } catch (e) {
      setError(errorMessage(e))
    } finally {
      setBusy(false)
    }
  }
  return (
    <Dialog
      open
      onOpenChange={(o) => !o && !busy && blocker.reset()}
      layout="confirm"
      eyebrow={<Badge tone="warn">Unsaved changes</Badge>}
      title={title}
      description={description}
      footer={
        <>
          <Button onClick={() => blocker.reset()} disabled={busy}>
            Stay
          </Button>
          <Button onClick={() => blocker.proceed()} disabled={busy}>
            Leave without saving
          </Button>
          {onSave ? (
            <Button variant="primary" onClick={save} loading={busy}>
              {saveLabel}
            </Button>
          ) : null}
        </>
      }
    >
      {error ? <Note tone="bad">{error}</Note> : null}
    </Dialog>
  )
}
