import { Link, type LinkProps } from '@tanstack/react-router'
import { Tabs as RTabs } from 'radix-ui'
import type { ReactNode } from 'react'
import { cn } from '@/lib/cn'

export type NavTab = { label: ReactNode; to: LinkProps['to']; params?: LinkProps['params']; search?: LinkProps['search']; exact?: boolean; badge?: ReactNode }

/**
 * Section tabs under the screen title. They are links (each tab is its own
 * URL); the current one carries aria-current="page". On phones they scroll
 * horizontally instead of wrapping.
 */
export function NavTabs({ tabs, label, className }: { tabs: NavTab[]; label: string; className?: string }) {
  return (
    <nav className={cn('tabs scroll-sm', className)} aria-label={label}>
      {tabs.map((t, i) => (
        <Link key={i} to={t.to} params={t.params as never} search={t.search as never} activeOptions={{ exact: Boolean(t.exact), includeSearch: false }}>
          {t.label}
          {t.badge}
        </Link>
      ))}
    </nav>
  )
}

/**
 * Filter tabs that look like section tabs but change a search param of the
 * current screen (Campaigns: All · Drafts · …). They are toggle buttons
 * (aria-pressed), not navigation, so screen readers don't hear "current page".
 */
export function FilterTabs<T extends string>({
  value,
  items,
  onChange,
  label,
  className,
}: {
  value: T
  items: { value: T; label: ReactNode }[]
  onChange: (v: T) => void
  label: string
  className?: string
}) {
  return (
    <div className={cn('tabs scroll-sm', className)} role="group" aria-label={label}>
      {items.map((it) => (
        <button key={it.value} type="button" aria-pressed={value === it.value} onClick={() => onChange(it.value)}>
          {it.label}
        </button>
      ))}
    </div>
  )
}

/** In-page tabs (pill style) switching panels without changing the URL. */
export function PillTabs<T extends string>({
  value,
  onChange,
  items,
  label,
  className,
}: {
  value: T
  onChange: (v: T) => void
  items: { value: T; label: ReactNode }[]
  label: string
  className?: string
}) {
  return (
    <RTabs.Root value={value} onValueChange={(v) => onChange(v as T)}>
      <RTabs.List className={cn('tabs', className)} aria-label={label}>
        {items.map((it) => (
          <RTabs.Trigger key={it.value} value={it.value}>
            {it.label}
          </RTabs.Trigger>
        ))}
      </RTabs.List>
    </RTabs.Root>
  )
}

/** Side-panel tabs (Blocks · Layers · Templates) with underline. */
export function PanelTabs<T extends string>({
  value,
  onChange,
  items,
  label,
  children,
  className,
}: {
  value: T
  onChange: (v: T) => void
  items: { value: T; label: ReactNode }[]
  label: string
  children?: ReactNode
  className?: string
}) {
  return (
    <RTabs.Root value={value} onValueChange={(v) => onChange(v as T)} className={className}>
      <RTabs.List className="ptab" aria-label={label}>
        {items.map((it) => (
          <RTabs.Trigger key={it.value} value={it.value} asChild>
            <button type="button">{it.label}</button>
          </RTabs.Trigger>
        ))}
      </RTabs.List>
      {children}
    </RTabs.Root>
  )
}

export const TabPanel = RTabs.Content

/** Segmented control: a small group of mutually exclusive buttons. */
export function Segmented<T extends string>({
  value,
  onChange,
  items,
  label,
  className,
}: {
  value: T
  onChange: (v: T) => void
  items: { value: T; label: ReactNode; title?: string }[]
  label: string
  className?: string
}) {
  return (
    <div className={cn('seg', className)} role="group" aria-label={label}>
      {items.map((it) => (
        <button key={it.value} type="button" aria-pressed={value === it.value} title={it.title} onClick={() => onChange(it.value)}>
          {it.label}
        </button>
      ))}
    </div>
  )
}
