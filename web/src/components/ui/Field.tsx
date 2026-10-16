import {
  cloneElement,
  forwardRef,
  isValidElement,
  useId,
  type InputHTMLAttributes,
  type ReactElement,
  type ReactNode,
  type SelectHTMLAttributes,
  type TextareaHTMLAttributes,
} from 'react'
import { useT } from '@/i18n'
import { cn } from '@/lib/cn'
import { m } from './messages'

/**
 * A labelled form control: label above, helper or error below, wired with
 * aria-describedby and aria-invalid. The child control gets the id.
 */
export function Field({
  label,
  hint,
  error,
  success,
  children,
  className,
  size = 'md',
  optional,
}: {
  label: ReactNode
  hint?: ReactNode
  error?: ReactNode
  /** Validation passed and there's something worth saying ("Within the SES quota…"): green border and text. */
  success?: ReactNode
  children: ReactElement<{ id?: string; className?: string; 'aria-describedby'?: string; 'aria-invalid'?: boolean }>
  className?: string
  size?: 'md' | 'lg'
  optional?: boolean
}) {
  const t = useT()
  const id = useId()
  const ok = success && !error ? success : undefined
  const hintId = hint || ok ? `${id}-hint` : undefined
  const errId = error ? `${id}-err` : undefined
  const describedBy = [errId, hintId].filter(Boolean).join(' ') || undefined
  const control = isValidElement(children)
    ? cloneElement(children, {
        id: children.props.id ?? id,
        'aria-describedby': describedBy,
        'aria-invalid': error ? true : undefined,
        ...(ok ? { className: cn(children.props.className, 'okb') } : {}),
      })
    : children
  return (
    <div className={cn('lbl', size === 'lg' && 'lg', className)}>
      <label htmlFor={children.props.id ?? id}>
        {label}
        {optional ? <span className="k font-medium"> · {t(m.optional)}</span> : null}
      </label>
      {control}
      {error ? (
        <span className="err" id={errId} role="alert">
          {error}
        </span>
      ) : null}
      {ok ? (
        <span className="k okmsg" id={hintId}>
          {ok}
        </span>
      ) : hint ? (
        <span className="k" id={hintId}>
          {hint}
        </span>
      ) : null}
    </div>
  )
}

export interface InputProps extends Omit<InputHTMLAttributes<HTMLInputElement>, 'size'> {
  size?: 'md' | 'lg'
  invalid?: boolean
}

export const Input = forwardRef<HTMLInputElement, InputProps>(function Input({ className, size, invalid, ...rest }, ref) {
  return <input ref={ref} className={cn('in', size === 'lg' && 'lg', invalid && 'err', className)} aria-invalid={invalid || rest['aria-invalid']} {...rest} />
})

export const Select = forwardRef<HTMLSelectElement, SelectHTMLAttributes<HTMLSelectElement> & { invalid?: boolean }>(function Select(
  { className, invalid, children, ...rest },
  ref,
) {
  return (
    <select ref={ref} className={cn('in', invalid && 'err', className)} {...rest}>
      {children}
    </select>
  )
})

export const Textarea = forwardRef<HTMLTextAreaElement, TextareaHTMLAttributes<HTMLTextAreaElement> & { invalid?: boolean }>(function Textarea(
  { className, invalid, ...rest },
  ref,
) {
  return <textarea ref={ref} className={cn('in', invalid && 'err', className)} {...rest} />
})

/** A checkbox with its label (the prototype's `.chk`). */
export const Checkbox = forwardRef<HTMLInputElement, Omit<InputHTMLAttributes<HTMLInputElement>, 'type'> & { label: ReactNode; hint?: ReactNode }>(
  function Checkbox({ label, hint, className, id, ...rest }, ref) {
    const auto = useId()
    const cid = id ?? auto
    return (
      <div className={cn('chk', className)}>
        <input ref={ref} type="checkbox" id={cid} {...rest} />
        <label htmlFor={cid} className="flex flex-col gap-0.5">
          <span>{label}</span>
          {hint ? <span className="k">{hint}</span> : null}
        </label>
      </div>
    )
  },
)

/** A toggle switch with a visible or visually hidden label. */
export const Toggle = forwardRef<
  HTMLInputElement,
  Omit<InputHTMLAttributes<HTMLInputElement>, 'type'> & { label: ReactNode; hideLabel?: boolean; description?: ReactNode }
>(function Toggle({ label, hideLabel, description, className, id, ...rest }, ref) {
  const auto = useId()
  const tid = id ?? auto
  return (
    <div className={cn('flex items-center gap-3', className)}>
      <input ref={ref} type="checkbox" role="switch" className="tg" id={tid} {...rest} />
      <label htmlFor={tid} className={cn(hideLabel ? 'vh' : 'flex min-w-0 flex-col gap-0.5')}>
        <span className={hideLabel ? undefined : 'font-semibold'}>{label}</span>
        {description && !hideLabel ? <span className="k">{description}</span> : null}
      </label>
    </div>
  )
})

/** A radio option card (`label.opt`): title, description and extra content. */
export function OptionCard({
  name,
  value,
  checked,
  onChange,
  title,
  description,
  children,
  className,
  disabled,
}: {
  name: string
  value: string
  checked: boolean
  onChange: (value: string) => void
  title: ReactNode
  description?: ReactNode
  children?: ReactNode
  className?: string
  disabled?: boolean
}) {
  return (
    <label className={cn('opt', className)}>
      <input type="radio" name={name} value={value} checked={checked} disabled={disabled} onChange={() => onChange(value)} />
      <span className="flex min-w-0 flex-col gap-0.5">
        <span className="font-semibold">{title}</span>
        {description ? <span className="k">{description}</span> : null}
        {children}
      </span>
    </label>
  )
}
