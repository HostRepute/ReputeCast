import { Tooltip as RTooltip } from 'radix-ui'
import { cloneElement, useId, useState, type MouseEvent, type ReactElement } from 'react'
import { createPortal } from 'react-dom'

/**
 * A control the current member can't use, kept visible with the reason in a
 * bubble on hover and focus (Feedback board: "Editors can't change routing ·
 * ask Linh or Marcus"). Hidden controls would make the product look smaller
 * than it is. The reason is also the control's description for screen
 * readers, and clicking does nothing. Button's `disabledReason` uses it.
 *
 *   <DisabledReason reason={usePermissionReason('providers:manage')}>
 *     <a className="btn" href="…">Save routing</a>
 *   </DisabledReason>
 *
 * Without a reason the child works as usual. The component tree is the same
 * either way, so the control keeps its DOM node (and focus) when the reason
 * comes and goes. It adds no element next to the control and brings its own
 * tooltip provider, so it works anywhere.
 */
export function DisabledReason({
  reason,
  children,
  side = 'bottom',
}: {
  reason: string | undefined
  children: ReactElement<{ 'aria-disabled'?: boolean; 'aria-describedby'?: string; onClick?: (e: MouseEvent) => void }>
  side?: 'top' | 'bottom'
}) {
  const id = useId()
  const [open, setOpen] = useState(false)
  const control = reason
    ? cloneElement(children, {
        'aria-disabled': true,
        'aria-describedby': [children.props['aria-describedby'], id].filter(Boolean).join(' '),
        onClick: (e: MouseEvent) => e.preventDefault(),
      })
    : children
  return (
    <RTooltip.Provider delayDuration={150} skipDelayDuration={300}>
      <RTooltip.Root open={Boolean(reason) && open} onOpenChange={setOpen}>
        <RTooltip.Trigger asChild>{control}</RTooltip.Trigger>
        {reason ? (
          <>
            {/* The description lives outside the layout, so flex gaps and sibling selectors don't change. */}
            {createPortal(
              <span id={id} className="vh">
                {reason}
              </span>,
              document.body,
            )}
            <RTooltip.Portal>
              {/* The arrow is the board's: 14 px in from the bubble's start (app.css). */}
              <RTooltip.Content className="tip why" side={side} align="start" sideOffset={8} collisionPadding={12} aria-hidden="true">
                {reason}
              </RTooltip.Content>
            </RTooltip.Portal>
          </>
        ) : null}
      </RTooltip.Root>
    </RTooltip.Provider>
  )
}
