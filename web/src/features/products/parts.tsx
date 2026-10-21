import type { ReactNode } from 'react'
import type { Schemas } from '@/api/client'
import { Badge } from '@/components/ui/Badge'
import { useT } from '@/i18n'
import { cn } from '@/lib/cn'
import { downloadChip, orderStatus, panelChip, type ChipTone } from './format'
import { pm } from './messages'

/** The boards' 16 px circle icons in notes: "i" (info) and "!" (a warning). */
export function CircleIcon({ kind = 'info', color }: { kind?: 'info' | 'alert'; color: string }) {
  return (
    <svg width="16" height="16" viewBox="0 0 16 16" fill="none" stroke={color} strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
      <circle cx="8" cy="8" r="6.2" />
      <path d={kind === 'info' ? 'M8 7.2v3.8M8 5v.1' : 'M8 4.8v3.6M8 10.9v.1'} />
    </svg>
  )
}

/** The download chip's arrow. */
function DownloadIcon() {
  return (
    <svg width="13" height="13" viewBox="0 0 16 16" fill="none" stroke="currentColor" strokeWidth="1.9" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
      <path d="M8 2.5v8M4.5 7 8 10.5 11.5 7M2.5 13.5h11" />
    </svg>
  )
}

/** A file's icon in the Files list: a page, or a grid for spreadsheets. */
export function FileIcon({ kind, size = 17 }: { kind?: string; size?: number }) {
  const sheet = kind === 'xlsx' || kind === 'csv' || /\.(xlsx|xls|csv|numbers)$/i.test(kind ?? '')
  return (
    <svg width={size} height={size} viewBox="0 0 16 16" fill="none" stroke="currentColor" strokeWidth="1.6" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
      {sheet ? (
        <>
          <rect x="2" y="2" width="12" height="12" rx="1.5" />
          <path d="M2 6h12M2 10h12M6 2v12" />
        </>
      ) : (
        <>
          <path d="M4 1.5h5.5L12.5 4.5V14.5h-8.5z" />
          <path d="M9.5 1.5v3h3" />
        </>
      )}
    </svg>
  )
}

/**
 * A product's cover: the uploaded image, or a lavender tile with a page
 * when there's none yet. `alt` is the cover's alt text; without it the
 * cover is named after the product.
 */
export function Cover({ url, alt, name, w, h, decorative }: { url?: string; alt?: string; name: string; w: number; h: number; decorative?: boolean }) {
  const t = useT()
  const label = alt || t(pm.coverOf, { name })
  if (url) {
    return (
      <span className="pr-cov" style={{ width: w, height: h }}>
        <img src={url} alt={decorative ? '' : label} loading="lazy" />
      </span>
    )
  }
  return (
    <span className="pr-cov" style={{ width: w, height: h }} role={decorative ? undefined : 'img'} aria-label={decorative ? undefined : label} aria-hidden={decorative || undefined}>
      <FileIcon size={Math.max(14, Math.round(w / 2.6))} />
    </span>
  )
}

/** The design system's download chip. */
export function Chip({ tone, children, className }: { tone: ChipTone; children: ReactNode; className?: string }) {
  return (
    <span className={cn('pr-dlc', tone, className)}>
      <DownloadIcon />
      {children}
    </span>
  )
}

/** The Download column: "Delivered via Postmark · 1 of 5 downloads". */
export function DownloadCell({ d }: { d: Schemas['ProductDownloadState'] }) {
  const t = useT()
  const c = downloadChip(t, d)
  return <Chip tone={c.tone}>{c.text}</Chip>
}

/** The order panel's link: "Link works until 22 Oct · 4 of 5 downloads left". */
export function PanelChip({ d }: { d: Schemas['ProductDownloadState'] }) {
  const t = useT()
  const c = panelChip(t, d)
  return <Chip tone={c.tone}>{c.text}</Chip>
}

/** An order's status badge, with the line under it in tables. */
export function OrderStatus({ o, sub = true, dot = true }: { o: Schemas['ProductOrderItem']; sub?: boolean; dot?: boolean }) {
  const t = useT()
  const s = orderStatus(t, o)
  return (
    <span className="pr-stk">
      <Badge tone={s.tone} dot={dot && s.dot}>
        {s.label}
      </Badge>
      {sub && s.sub ? <span className="k">{s.sub}</span> : null}
    </span>
  )
}

/** A note with the boards' icon and a bold title (ProductsStates). */
export function IconNote({
  tone,
  title,
  children,
  actions,
  role = 'note',
  className,
}: {
  tone: 'n' | 'w' | 'pur' | 'b' | 'i'
  title?: ReactNode
  children: ReactNode
  actions?: ReactNode
  role?: 'note' | 'status' | 'alert'
  className?: string
}) {
  const color = { n: '#4B5563', w: '#B45309', pur: '#5B21B6', b: '#B91C1C', i: '#1E40AF' }[tone]
  return (
    <div className={cn('note pr-note', tone === 'pur' ? 'pur' : tone, className)} role={role}>
      <CircleIcon kind={tone === 'w' || tone === 'b' ? 'alert' : 'info'} color={color} />
      <div className="tx">
        {title ? <span className="ttl">{title}</span> : null}
        <div>{children}</div>
        {actions ? <div className="acts">{actions}</div> : null}
      </div>
    </div>
  )
}

/** `<mono>` in messages. */
export const mono = (c: ReactNode[]) => <span className="mono">{c}</span>
/** `<b>` in messages. */
export const bold = (c: ReactNode[]) => <b>{c}</b>
