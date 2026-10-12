import { DropdownMenu, Popover as RPopover, Tooltip as RTooltip } from 'radix-ui'
import type { ReactNode } from 'react'
import { cn } from '@/lib/cn'

/** Dropdown menu with keyboard navigation and typeahead (Radix). */
export function Menu({
  trigger,
  children,
  align = 'end',
  className,
  label,
}: {
  trigger: ReactNode
  children: ReactNode
  align?: 'start' | 'end' | 'center'
  className?: string
  label?: string
}) {
  return (
    <DropdownMenu.Root modal={false}>
      <DropdownMenu.Trigger asChild>{trigger}</DropdownMenu.Trigger>
      <DropdownMenu.Portal>
        <DropdownMenu.Content className={cn('menu', className)} align={align} sideOffset={6} collisionPadding={12} aria-label={label}>
          {children}
        </DropdownMenu.Content>
      </DropdownMenu.Portal>
    </DropdownMenu.Root>
  )
}

export function MenuItem({ children, onSelect, disabled, className, asChild }: { children: ReactNode; onSelect?: () => void; disabled?: boolean; className?: string; asChild?: boolean }) {
  return (
    <DropdownMenu.Item className={className} onSelect={onSelect} disabled={disabled} asChild={asChild}>
      {children}
    </DropdownMenu.Item>
  )
}

export function MenuLabel({ children }: { children: ReactNode }) {
  return <DropdownMenu.Label className="mh">{children}</DropdownMenu.Label>
}

export function MenuSeparator() {
  return <DropdownMenu.Separator />
}

/** Non-modal popover (notification center, filters). */
export function Popover({
  trigger,
  children,
  align = 'end',
  className,
  open,
  onOpenChange,
  label,
}: {
  trigger: ReactNode
  children: ReactNode
  align?: 'start' | 'end' | 'center'
  className?: string
  open?: boolean
  onOpenChange?: (o: boolean) => void
  label?: string
}) {
  return (
    <RPopover.Root open={open} onOpenChange={onOpenChange}>
      <RPopover.Trigger asChild>{trigger}</RPopover.Trigger>
      <RPopover.Portal>
        <RPopover.Content className={cn('menu', className)} align={align} sideOffset={6} collisionPadding={12} aria-label={label}>
          {children}
        </RPopover.Content>
      </RPopover.Portal>
    </RPopover.Root>
  )
}

/** Tooltip for icon-only controls in the rail and toolbars. */
export function Tooltip({ content, children, side = 'right' }: { content: ReactNode; children: ReactNode; side?: 'top' | 'right' | 'bottom' | 'left' }) {
  return (
    <RTooltip.Root delayDuration={300}>
      <RTooltip.Trigger asChild>{children}</RTooltip.Trigger>
      <RTooltip.Portal>
        <RTooltip.Content className="tip" side={side} sideOffset={6}>
          {content}
        </RTooltip.Content>
      </RTooltip.Portal>
    </RTooltip.Root>
  )
}

export const TooltipProvider = RTooltip.Provider
