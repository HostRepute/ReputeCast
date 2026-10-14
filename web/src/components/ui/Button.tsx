import { Link, type LinkProps } from '@tanstack/react-router'
import { forwardRef, type ButtonHTMLAttributes, type ReactNode } from 'react'
import { cn } from '@/lib/cn'
import { Icon, type IconName } from '../icons/Icon'
import { DisabledReason } from './DisabledReason'

export type ButtonVariant = 'default' | 'primary' | 'danger' | 'danger-primary' | 'ghost'
export type ButtonSize = 'xs' | 'sm' | 'md' | 'lg'

export interface ButtonStyleProps {
  variant?: ButtonVariant
  size?: ButtonSize
  icon?: IconName
  iconRight?: IconName
  /** Square icon-only button; pass aria-label. */
  iconOnly?: boolean
  block?: boolean
  loading?: boolean
}

export function buttonClass({ variant = 'default', size = 'md', iconOnly, block }: ButtonStyleProps, className?: string) {
  return cn(
    'btn',
    variant === 'primary' && 'pri',
    variant === 'danger' && 'danger',
    variant === 'danger-primary' && 'danger pri',
    variant === 'ghost' && 'ghost',
    size === 'sm' && 'sm',
    size === 'xs' && 'xs',
    size === 'lg' && 'lg',
    iconOnly && 'ic',
    block && 'block',
    className,
  )
}

function content(
  icon: IconName | undefined,
  iconRight: IconName | undefined,
  loading: boolean | undefined,
  children: ReactNode,
  size: ButtonSize,
  onDark = false,
) {
  const s = size === 'lg' || size === 'md' ? 16 : 14
  return (
    <>
      {loading ? <span className={cn('spin sm', onDark && 'on-dark')} aria-hidden="true" /> : icon ? <Icon name={icon} size={s} /> : null}
      {children}
      {iconRight ? <Icon name={iconRight} size={s} /> : null}
    </>
  )
}

export interface ButtonProps extends ButtonHTMLAttributes<HTMLButtonElement>, ButtonStyleProps {
  /**
   * Disables the button and says why (principle 7), in a DisabledReason
   * bubble. The button stays focusable with aria-disabled.
   */
  disabledReason?: string
  /**
   * The label while `loading` ("Saving…"). The button keeps the width of
   * the wider of its two states, so nothing moves when it changes.
   */
  busyLabel?: ReactNode
}

export const Button = forwardRef<HTMLButtonElement, ButtonProps>(function Button(props, ref) {
  const { variant, size = 'md', icon, iconRight, iconOnly, block, loading, disabledReason, busyLabel, className, children, type = 'button', onClick, title, ...rest } = props
  const blocked = Boolean(disabledReason)
  const onDark = variant === 'primary' || variant === 'danger-primary'
  const button = (
    <button
      ref={ref}
      type={type}
      className={buttonClass({ variant, size, iconOnly, block }, className)}
      aria-disabled={blocked || undefined}
      aria-busy={loading || undefined}
      title={blocked ? undefined : title}
      onClick={blocked || loading ? (e) => e.preventDefault() : onClick}
      {...rest}
    >
      {busyLabel === undefined ? (
        content(icon, iconRight, loading, children, size, onDark)
      ) : (
        <span className="bl">
          <span className={cn('inline-flex items-center justify-center gap-2', loading && 'reserve')} aria-hidden={loading || undefined}>
            {content(icon, iconRight, false, children, size, onDark)}
          </span>
          <span className={cn('inline-flex items-center justify-center gap-2', !loading && 'reserve')} aria-hidden={!loading || undefined}>
            {content(undefined, undefined, true, busyLabel, size, onDark)}
          </span>
        </span>
      )}
    </button>
  )
  // A button that can be disabled with a reason (the prop is passed, even as
  // undefined) always goes through the bubble, so it keeps its DOM node and
  // focus when the reason comes and goes. Other buttons skip it.
  return 'disabledReason' in props ? <DisabledReason reason={disabledReason}>{button}</DisabledReason> : button
})

export type ButtonLinkProps = LinkProps & ButtonStyleProps & { className?: string; children?: ReactNode; 'aria-label'?: string; title?: string }

/** A router link styled as a button (the prototype's `<a class="btn">`). */
export function ButtonLink({ variant, size = 'md', icon, iconRight, iconOnly, block, className, children, ...rest }: ButtonLinkProps) {
  return (
    <Link className={buttonClass({ variant, size, iconOnly, block }, className)} {...rest}>
      {content(icon, iconRight, false, children as ReactNode, size)}
    </Link>
  )
}

/** An external link styled as a button. Opens in a new tab with rel=noopener. */
export function ButtonAnchor({
  href,
  variant,
  size = 'md',
  icon,
  iconRight,
  iconOnly,
  block,
  className,
  children,
  newTab = true,
  ...rest
}: ButtonStyleProps & { href: string; className?: string; children?: ReactNode; newTab?: boolean; 'aria-label'?: string; download?: string }) {
  return (
    <a
      href={href}
      className={buttonClass({ variant, size, iconOnly, block }, className)}
      target={newTab ? '_blank' : undefined}
      rel={newTab ? 'noopener noreferrer' : undefined}
      {...rest}
    >
      {content(icon, iconRight, false, children, size)}
    </a>
  )
}
