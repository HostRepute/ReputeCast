import { Link } from '@tanstack/react-router'
import { Dialog as RDialog } from 'radix-ui'
import { useRef, type ReactElement } from 'react'
import { useT } from '@/i18n'
import { cn } from '@/lib/cn'
import { useWorkspace } from '@/lib/session'
import { Icon } from '../icons/Icon'
import { useHostReputeLine } from './HostReputeStatus'
import { m } from './messages'
import type { NavItem, SectionKey } from './nav'
import { WorkspaceSwitcher } from './WorkspaceSwitcher'

/**
 * The phone tab bar's More (ShellMore): a sheet with the sections the tab
 * bar leaves out, the workspace switcher and HostRepute's status, which the
 * sidebar shows on larger screens.
 *
 * A modal dialog: focus moves into the sheet and stays there, Escape and the
 * scrim close it, and focus goes back to More (or to the new screen's title
 * after following a link). It rises in as the design system's entrances do,
 * and appears at once with reduced motion.
 */
export function MoreSheet({
  open,
  onOpenChange,
  items,
  current,
  children,
}: {
  open: boolean
  onOpenChange: (open: boolean) => void
  /** The sections in the sheet, in sidebar order. */
  items: NavItem[]
  /** The section of the current screen. */
  current: SectionKey | undefined
  /** The More tab: the sheet's trigger. */
  children: ReactElement
}) {
  const { slug } = useWorkspace()
  const t = useT()
  const hr = useHostReputeLine()
  const trigger = useRef<HTMLButtonElement>(null)
  // Following a link moves focus to the new screen's title instead of back to More.
  const followed = useRef(false)
  const follow = () => {
    followed.current = true
    onOpenChange(false)
  }
  return (
    <RDialog.Root open={open} onOpenChange={onOpenChange}>
      <RDialog.Trigger asChild ref={trigger}>
        {children}
      </RDialog.Trigger>
      <RDialog.Portal>
        <RDialog.Overlay className="sheet-scrim" />
        <RDialog.Content
          className="sheet"
          aria-modal="true"
          aria-describedby={undefined}
          onOpenAutoFocus={() => {
            followed.current = false
          }}
          onCloseAutoFocus={(e) => {
            if (!followed.current) return
            e.preventDefault()
            document.querySelector<HTMLElement>('.top h1')?.focus({ preventScroll: true })
          }}
          // The tab bar stays above the scrim: More closes the sheet (its own
          // click does, so the press outside mustn't), the other tabs close it
          // and open their section.
          onPointerDownOutside={(e) => {
            const target = e.target as Node | null
            if (target && trigger.current?.contains(target)) e.preventDefault()
            else if (target instanceof Element && target.closest('.side')) followed.current = true
          }}
        >
          <span className="sheet-grab" aria-hidden="true" />
          <div className="sheet-head">
            <RDialog.Title className="sheet-title">{t(m.moreSheetTitle)}</RDialog.Title>
            <RDialog.Close className="ibtn" aria-label={t(m.moreSheetClose)}>
              <Icon name="close" size={16} />
            </RDialog.Close>
          </div>
          <WorkspaceSwitcher variant="sheet" />
          <nav className="sheet-nav" aria-label={t(m.moreSheetNavLabel)}>
            {items.map((item) => (
              <Link
                key={item.key}
                to={item.to}
                params={{ workspace: slug }}
                activeOptions={{ exact: Boolean(item.exact), includeSearch: false }}
                className={cn('sheet-item', current === item.key && 'on')}
                aria-current={current === item.key ? 'page' : undefined}
                onClick={follow}
              >
                <span className="sheet-ic">
                  <Icon name={item.icon} size={18} />
                </span>
                <span className="sheet-lbl">{t(item.label)}</span>
                <Icon name="chevronRight" size={14} className="sheet-chev" />
              </Link>
            ))}
          </nav>
          <div className="sheet-sep" />
          <Link to="/w/$workspace/settings/integrations/hostrepute" params={{ workspace: slug }} className="sheet-row" onClick={follow}>
            <span className="dot" style={{ color: hr.color, margin: '0 14px' }} aria-hidden="true" />
            <span className="sheet-row-t">
              <span className="sheet-row-l">{hr.title}</span>
              <span className="k">{hr.line}</span>
            </span>
          </Link>
          <Link to="/w/$workspace/import" params={{ workspace: slug }} className="sheet-row" onClick={follow}>
            <span className="sheet-row-ic">
              <Icon name="import" size={18} />
            </span>
            <span className="sheet-row-l">{t(m.importContacts)}</span>
          </Link>
        </RDialog.Content>
      </RDialog.Portal>
    </RDialog.Root>
  )
}
