import type { ReactNode } from 'react'
import type { BlockKind } from './model'

/**
 * The page builder's icons, copied from the PageBuilder boards: the block
 * tiles and layers (24-unit grid) and the header, toolbar and note icons
 * (16-unit grid).
 */
const blocks: Record<BlockKind | 'columns3' | 'theme', ReactNode> = {
  section: <rect x="3" y="5" width="18" height="14" rx="2" />,
  theme: <rect x="3" y="5" width="18" height="14" rx="2" />,
  columns: (
    <>
      <rect x="3" y="5" width="8" height="14" rx="2" />
      <rect x="13" y="5" width="8" height="14" rx="2" />
    </>
  ),
  columns3: (
    <>
      <rect x="2.5" y="5" width="5.4" height="14" rx="1.5" />
      <rect x="9.3" y="5" width="5.4" height="14" rx="1.5" />
      <rect x="16.1" y="5" width="5.4" height="14" rx="1.5" />
    </>
  ),
  divider: <path d="M3 12h18" />,
  spacer: <path d="M4 6v12M20 6v12M8 12h8" />,
  hero: (
    <>
      <rect x="3" y="4" width="18" height="16" rx="2" />
      <path d="M7 9h10M8.5 12.2h7" />
      <rect x="9" y="15" width="6" height="2.4" rx="1.2" />
    </>
  ),
  text: <path d="M4 7h16M4 12h16M4 17h10" />,
  image: (
    <>
      <rect x="3" y="4" width="18" height="16" rx="2" />
      <path d="m3 16 5-5 4 4 3-3 6 6" />
      <circle cx="16" cy="9" r="1.5" />
    </>
  ),
  button: <rect x="3" y="8" width="18" height="8" rx="4" />,
  video: (
    <>
      <rect x="3" y="5" width="18" height="14" rx="2" />
      <path d="m10 9.2 5 2.8-5 2.8z" />
    </>
  ),
  features: <path d="m3.5 7 1.6 1.6L8 5.7M11 7h9.5M3.5 12.5l1.6 1.6L8 11.2M11 12.5h9.5M3.5 18l1.6 1.6L8 16.7M11 18h9.5" />,
  testimonials: <path d="M5 18v-4.5C5 10.4 6.6 8.3 9.5 7M5 13.5h4.5V18H5M13.5 18v-4.5c0-3.1 1.6-5.2 4.5-6.5M13.5 13.5H18V18h-4.5" />,
  faq: (
    <>
      <circle cx="12" cy="12" r="8.5" />
      <path d="M9.6 9.6a2.5 2.5 0 0 1 4.8.9c0 1.7-2.4 2.1-2.4 3.6M12 16.8v.1" />
    </>
  ),
  signup: (
    <>
      <rect x="3" y="5.5" width="18" height="5.5" rx="1.8" />
      <rect x="3" y="13.5" width="9" height="5" rx="1.8" />
    </>
  ),
  countdown: (
    <>
      <circle cx="12" cy="13" r="7.2" />
      <path d="M12 13V9.4M9.8 3h4.4M18.4 6.6l1.4-1.4" />
    </>
  ),
  event: (
    <>
      <rect x="3.5" y="5" width="17" height="15" rx="2" />
      <path d="M3.5 10h17M8 3v4M16 3v4M8 14h3" />
    </>
  ),
  product: (
    <>
      <path d="M3.5 12.6 11.4 4.7H19.3v7.9l-7.9 7.9z" />
      <circle cx="15.6" cy="8.4" r="1.4" />
    </>
  ),
  products: (
    <>
      <rect x="3.5" y="3.5" width="7.5" height="7.5" rx="1.5" />
      <rect x="13" y="3.5" width="7.5" height="7.5" rx="1.5" />
      <rect x="3.5" y="13" width="7.5" height="7.5" rx="1.5" />
      <rect x="13" y="13" width="7.5" height="7.5" rx="1.5" />
    </>
  ),
  store: (
    <>
      <path d="M5 8h14l-1.2 12H6.2z" />
      <path d="M9 8V6.6a3 3 0 0 1 6 0V8" />
    </>
  ),
  tiers: (
    <>
      <rect x="3" y="6.5" width="8" height="12.5" rx="1.6" />
      <rect x="13" y="4" width="8" height="15" rx="1.6" />
      <path d="M5.5 10.5h3M15.5 8h3M15.5 11h3" />
    </>
  ),
  posts: <path d="M4 6.5h2.5M9 6.5h11M4 12h2.5M9 12h11M4 17.5h2.5M9 17.5h11" />,
  bio: (
    <>
      <circle cx="8.5" cy="10" r="3" />
      <path d="M3.5 18.5a5 5 0 0 1 10 0M15.5 9h5M15.5 13h5" />
    </>
  ),
}

export type BlockIconName = keyof typeof blocks

/** A block's icon: 22 px on the palette's tiles, 14 px in the layers, 18 px in the Block tab. */
export function BlockIcon({ name, size = 22, color = '#374151' }: { name: BlockIconName; size?: number; color?: string }) {
  return (
    <svg width={size} height={size} viewBox="0 0 24 24" fill="none" stroke={color} strokeWidth={1.5} strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
      {blocks[name]}
    </svg>
  )
}

const ui = {
  back: <path d="m10 3-5 5 5 5" />,
  desktop: (
    <>
      <rect x="1.5" y="2.5" width="13" height="9" rx="1.3" />
      <path d="M5.5 14h5M8 11.5V14" />
    </>
  ),
  tablet: (
    <>
      <rect x="3" y="1.5" width="10" height="13" rx="1.6" />
      <path d="M7 12.3h2" />
    </>
  ),
  phone: (
    <>
      <rect x="4.5" y="1.5" width="7" height="13" rx="1.6" />
      <path d="M7.3 12.3h1.4" />
    </>
  ),
  eye: (
    <>
      <path d="M1.5 8C2.6 5.1 5.1 3.5 8 3.5s5.4 1.6 6.5 4.5c-1.1 2.9-3.6 4.5-6.5 4.5S2.6 10.9 1.5 8z" />
      <circle cx="8" cy="8" r="2" />
    </>
  ),
  eyeOff: <path d="M2 2l12 12M6.6 6.6a2 2 0 0 0 2.8 2.8M4.2 4.3C2.9 5.2 2 6.5 1.5 8c1.1 2.9 3.6 4.5 6.5 4.5 1.3 0 2.5-.3 3.6-.9M8 3.5c2.9 0 5.4 1.6 6.5 4.5-.3.9-.8 1.7-1.4 2.4" />,
  undo: (
    <>
      <path d="M5.5 4 2.5 7l3 3" />
      <path d="M2.5 7h7a4 4 0 0 1 0 8h-2" />
    </>
  ),
  redo: (
    <>
      <path d="M10.5 4l3 3-3 3" />
      <path d="M13.5 7h-7a4 4 0 0 0 0 8h2" />
    </>
  ),
  up: <path d="m4 10 4-4 4 4" />,
  down: <path d="m4 6 4 4 4-4" />,
  duplicate: (
    <>
      <rect x="5" y="5" width="9" height="9" rx="1.5" />
      <path d="M11 5V3.5A1.5 1.5 0 0 0 9.5 2h-6A1.5 1.5 0 0 0 2 3.5v6A1.5 1.5 0 0 0 3.5 11H5" />
    </>
  ),
  trash: <path d="M3 4h10M6 4V2.5h4V4M5 4l.6 9h4.8L11 4" />,
  info: (
    <>
      <circle cx="8" cy="8" r="6.2" />
      <path d="M8 7.2v3.8M8 5v.1" />
    </>
  ),
  alert: (
    <>
      <circle cx="8" cy="8" r="6.2" />
      <path d="M8 4.8v3.6M8 10.9v.1" />
    </>
  ),
  check: <path d="m3 8 3.5 3.5L13 5" />,
  cross: <path d="M4 4l8 8M12 4l-8 8" />,
  link: (
    <>
      <path d="M6.8 9.2a2.6 2.6 0 0 0 3.7 0l2.2-2.2a2.6 2.6 0 0 0-3.7-3.7l-.9.9" />
      <path d="M9.2 6.8a2.6 2.6 0 0 0-3.7 0L3.3 9a2.6 2.6 0 0 0 3.7 3.7l.9-.9" />
    </>
  ),
  external: <path d="M9 3h4v4M13 3 7.5 8.5M12 9.5V13H3V4h3.5" />,
}

export type UiIconName = keyof typeof ui

const weights: Partial<Record<UiIconName, number>> = { back: 1.8, undo: 1.8, redo: 1.8, up: 1.8, down: 1.8, duplicate: 1.6, trash: 1.7, info: 1.8, alert: 1.9, check: 2.2, cross: 2.2 }

/** A 16-unit icon of the builder's chrome, in the text's colour unless `color` says otherwise. */
export function UiIcon({ name, size = 14, color = 'currentColor', strokeWidth }: { name: UiIconName; size?: number; color?: string; strokeWidth?: number }) {
  return (
    <svg width={size} height={size} viewBox="0 0 16 16" fill="none" stroke={color} strokeWidth={strokeWidth ?? weights[name] ?? 1.7} strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
      {ui[name]}
    </svg>
  )
}
