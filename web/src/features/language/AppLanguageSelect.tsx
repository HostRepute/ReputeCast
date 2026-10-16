import { Select } from 'radix-ui'
import type { ReactNode } from 'react'
import { Badge } from '@/components/ui/Badge'

export interface LanguageOption {
  value: string
  label: string
  /** The option's language, when it's named in a language other than the page's. */
  lang?: string
  badge?: ReactNode
  description?: ReactNode
  /** Draw a line above the option (the pseudo-locale sits apart). */
  separated?: boolean
}

const Chevron = () => (
  <svg width="13" height="13" viewBox="0 0 16 16" fill="none" stroke="currentColor" strokeWidth="1.9" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
    <path d="m4 6 4 4 4-4" />
  </svg>
)

const Check = () => (
  <svg width="14" height="14" viewBox="0 0 16 16" fill="none" stroke="var(--accent)" strokeWidth="2.2" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
    <path d="m3 8 3.5 3.5L13 5" />
  </svg>
)

/**
 * The app-language picker. A plain select, like every other; when an option
 * carries a badge or a description (the pseudo-locale in development) it
 * opens as a listbox that can show them (LanguageRegion board).
 */
export function AppLanguageSelect({ id, value, options, onChange, label }: { id: string; value: string; options: LanguageOption[]; onChange: (v: string) => void; label: string }) {
  const rich = options.some((o) => o.badge || o.description)
  if (!rich) {
    return (
      <select id={id} className="in" value={value} onChange={(e) => onChange(e.target.value)}>
        {options.map((o) => (
          <option key={o.value} value={o.value} lang={o.lang}>
            {o.label}
          </option>
        ))}
      </select>
    )
  }
  return (
    <Select.Root value={value} onValueChange={onChange}>
      <Select.Trigger id={id} className="in lang-trigger" aria-label={label}>
        <Select.Value />
        <Select.Icon asChild>
          <span className="flex">
            <Chevron />
          </span>
        </Select.Icon>
      </Select.Trigger>
      <Select.Portal>
        <Select.Content className="menu lang-menu" position="popper" sideOffset={6} collisionPadding={12} aria-label={label}>
          <Select.Viewport>
            {options.map((o) => (
              <OptionRow key={o.value} o={o} />
            ))}
          </Select.Viewport>
        </Select.Content>
      </Select.Portal>
    </Select.Root>
  )
}

function OptionRow({ o }: { o: LanguageOption }) {
  return (
    <>
      {o.separated ? <hr /> : null}
      <Select.Item value={o.value} className="lopt">
        <span>
          <Select.ItemIndicator>
            <Check />
          </Select.ItemIndicator>
        </span>
        <Select.ItemText asChild>
          <span className="nm" lang={o.lang}>
            {o.label}
          </span>
        </Select.ItemText>
        <span>{o.badge ? <Badge tone="purple" size="sm">{o.badge}</Badge> : null}</span>
        {o.description ? <span className="sub">{o.description}</span> : null}
      </Select.Item>
    </>
  )
}
