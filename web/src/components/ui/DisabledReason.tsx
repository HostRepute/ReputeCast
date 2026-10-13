import { Tooltip as RTooltip } from 'radix-ui'
import { cloneElement, useId, type MouseEvent, type ReactElement } from 'react'

/**
 * A control the current member can't use, kept visible with the reason in a
 * bubble on hover and focus (Feedback board: "Editors can't change routing ·
 * ask Linh or Marcus"). Hidden controls would make the product look smaller
 * than it is. The reason is also the control's description for screen
 * readers, and clicking does nothing.
 *
 *   <DisabledReason reason={usePermissionReason('providers:manage')}>
 *     <Button>Save routing</Button>
 *   </DisabledReason>
 *
 * Without a reason the child renders unchanged.
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
  if (!reason) return children
  const control = cloneElement(children, {
    'aria-disabled': true,
    'aria-describedby': id,
    onClick: (e: MouseEvent) => e.preventDefault(),
  })
  return (
    <RTooltip.Root delayDuration={150}>
      <RTooltip.Trigger asChild>{control}</RTooltip.Trigger>
      <span id={id} className="vh">
        {reason}
      </span>
      <RTooltip.Portal>
        <RTooltip.Content className="tip why" side={side} align="start" sideOffset={8} collisionPadding={12}>
          {reason}
          <RTooltip.Arrow className="arrow" width={10} height={5} />
        </RTooltip.Content>
      </RTooltip.Portal>
    </RTooltip.Root>
  )
}
