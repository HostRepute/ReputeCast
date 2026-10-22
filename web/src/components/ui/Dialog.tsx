import { Dialog as RDialog } from 'radix-ui'
import { useLayoutEffect, useRef, type ReactNode } from 'react'
import { useT } from '@/i18n'
import { common } from '@/i18n/common'
import { cn } from '@/lib/cn'
import { Icon } from '../icons/Icon'
import { m } from './messages'

export interface DialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  title: ReactNode
  description?: ReactNode
  /** Small state badges above the title (e.g. "Dialog · step 2 of 2"). */
  eyebrow?: ReactNode
  children?: ReactNode
  footer?: ReactNode
  size?: 'sm' | 'md' | 'lg'
  /** Destructive and protective dialogs can't be dismissed by clicking outside. */
  dismissible?: boolean
  /** Hide the × button (boards that close with Cancel only); Escape still closes. */
  hideClose?: boolean
  className?: string
  /**
   * "standard" (header, body, footer with a divider and a close button) is
   * the team and access boards' dialog. "confirm" is the Dialogs board's one
   * padded column: badge, title, the consequences, rows, actions on the
   * right. "session" is the session-ended dialog. The last two have no close
   * button: their actions say what closing means.
   */
  layout?: 'standard' | 'confirm' | 'session'
  /** Element to focus when the dialog opens (Radix focuses the first one otherwise). */
  onOpenAutoFocus?: (e: Event) => void
}

/**
 * Modal dialog: focus is trapped inside and returned on close to whatever
 * had it when the dialog opened. Escape closes unless `dismissible` is false.
 */
export function Dialog({
  open,
  onOpenChange,
  title,
  description,
  eyebrow,
  children,
  footer,
  size = 'md',
  dismissible = true,
  hideClose,
  className,
  layout = 'standard',
  onOpenAutoFocus,
}: DialogProps) {
  const t = useT()
  // Radix gives focus back to a Dialog.Trigger, and most dialogs open from a
  // plain button instead: keep what had focus when the dialog opened (before
  // Radix moves it inside) and give it back on close.
  const opener = useRef<HTMLElement | null>(null)
  useLayoutEffect(() => {
    if (!open) return
    const el = document.activeElement
    opener.current = el instanceof HTMLElement && el !== document.body ? el : null
  }, [open])
  const onCloseAutoFocus = (e: Event) => {
    const el = opener.current
    opener.current = null
    if (el?.isConnected) {
      e.preventDefault()
      el.focus()
    }
  }
  if (layout !== 'standard') {
    return (
      <RDialog.Root open={open} onOpenChange={onOpenChange}>
        <RDialog.Portal>
          <RDialog.Overlay className="ovl">
            <RDialog.Content
              className={cn('dlg', layout === 'confirm' ? 'cdlg' : 'sdlg', className)}
              onPointerDownOutside={(e) => !dismissible && e.preventDefault()}
              onEscapeKeyDown={(e) => !dismissible && e.preventDefault()}
              onOpenAutoFocus={onOpenAutoFocus}
              onCloseAutoFocus={onCloseAutoFocus}
            >
              {eyebrow ? <div className="flex flex-wrap items-center gap-2">{eyebrow}</div> : null}
              <RDialog.Title className="ct">{title}</RDialog.Title>
              {description ? (
                <RDialog.Description className="cd">{description}</RDialog.Description>
              ) : (
                <RDialog.Description className="vh">{typeof title === 'string' ? title : t(m.dialog)}</RDialog.Description>
              )}
              {children}
              {footer ? <div className="dacts">{footer}</div> : null}
            </RDialog.Content>
          </RDialog.Overlay>
        </RDialog.Portal>
      </RDialog.Root>
    )
  }
  return (
    <RDialog.Root open={open} onOpenChange={onOpenChange}>
      <RDialog.Portal>
        <RDialog.Overlay className="ovl">
          <RDialog.Content
            className={cn('dlg', size === 'sm' && 'sm', size === 'lg' && 'lg', className)}
            onPointerDownOutside={(e) => !dismissible && e.preventDefault()}
            onEscapeKeyDown={(e) => !dismissible && e.preventDefault()}
            onOpenAutoFocus={onOpenAutoFocus}
            onCloseAutoFocus={onCloseAutoFocus}
            aria-describedby={description ? undefined : undefined}
          >
            <div className="dh">
              {eyebrow ? <div className="flex flex-wrap items-center gap-2">{eyebrow}</div> : null}
              <div className="flex items-start justify-between gap-3">
                <RDialog.Title className="m-0 text-[18px] font-bold tracking-[-0.01em]">{title}</RDialog.Title>
                {hideClose ? null : (
                  <RDialog.Close className="btn sm ic -mr-1.5 -mt-0.5 border-transparent" aria-label={t(common.close)}>
                    <Icon name="close" size={14} />
                  </RDialog.Close>
                )}
              </div>
              {description ? <RDialog.Description className="text-ink-2">{description}</RDialog.Description> : <RDialog.Description className="vh">{typeof title === 'string' ? title : t(m.dialog)}</RDialog.Description>}
            </div>
            {children ? <div className="db">{children}</div> : null}
            {footer ? <div className="df">{footer}</div> : null}
          </RDialog.Content>
        </RDialog.Overlay>
      </RDialog.Portal>
    </RDialog.Root>
  )
}

export const DialogClose = RDialog.Close
