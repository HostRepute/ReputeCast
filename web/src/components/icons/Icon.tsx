import type { ReactNode, SVGProps } from 'react'

/**
 * Stroke icons copied from the prototype (16-unit grid, round joins). Icons
 * are decorative by default (aria-hidden); pass `label` when an icon is the
 * only content of a control and no aria-label is set on the control itself.
 */
type IconDef = { body: ReactNode; strokeWidth?: number; viewBox?: string; fill?: boolean }

const icons = {
  // Main navigation
  overview: { body: <path d="M2 2h5v5H2zM9 2h5v5H9zM2 9h5v5H2zM9 9h5v5H9z" /> },
  campaigns: { body: <path d="M14 2 2 7l5 2 2 5zM7 9l7-7" strokeLinejoin="round" /> },
  automations: {
    body: (
      <>
        <circle cx="4" cy="3" r="2" />
        <circle cx="4" cy="13" r="2" />
        <circle cx="12" cy="8" r="2" />
        <path d="M4 5v6M6 8h4" />
      </>
    ),
  },
  audience: {
    body: (
      <>
        <circle cx="6" cy="5" r="2.5" />
        <path d="M1.5 14a4.5 4.5 0 0 1 9 0M11 3a2.5 2.5 0 0 1 0 5M14.5 14a4 4 0 0 0-3-3.8" />
      </>
    ),
  },
  forms: {
    body: (
      <>
        <rect x="2" y="2" width="12" height="12" rx="2" />
        <path d="M5 6h6M5 9h6M5 12h3" />
      </>
    ),
  },
  deliverability: {
    body: (
      <>
        <path d="M8 1.5 13.5 3.5v4.5c0 3-2.3 5.3-5.5 6.5C4.8 13.3 2.5 11 2.5 8V3.5z" strokeLinejoin="round" />
        <path d="m5.5 8 1.8 1.8L10.8 6" />
      </>
    ),
  },
  providers: {
    body: (
      <>
        <rect x="2" y="2" width="12" height="5" rx="1.5" />
        <rect x="2" y="9" width="12" height="5" rx="1.5" />
        <path d="M5 4.5h.01M5 11.5h.01" strokeWidth={2} strokeLinecap="round" />
      </>
    ),
  },
  settings: {
    body: (
      <>
        <path d="M2 4h7M13 4h1M2 8h2M8 8h6M2 12h7M13 12h1" />
        <circle cx="11" cy="4" r="1.8" />
        <circle cx="6" cy="8" r="1.8" />
        <circle cx="11" cy="12" r="1.8" />
      </>
    ),
  },
  sliders: { body: <path d="M2 4h7M13 4h1M2 8h2M8 8h6M2 12h7M13 12h1" /> },
  import: { body: <path d="M8 11V3M4.5 6.5 8 3l3.5 3.5M2 13h12" /> },
  export: { body: <path d="M8 3v8M4.5 7.5 8 11l3.5-3.5M2 13h12" /> },
  bell: { body: <path d="M4 11V7a4 4 0 0 1 8 0v4l1.5 1.5h-11zM6.5 14a1.5 1.5 0 0 0 3 0" /> },

  // Arrows and chevrons
  chevronDown: { body: <path d="m4 6 4 4 4-4" />, strokeWidth: 1.8 },
  chevronUp: { body: <path d="m4 10 4-4 4 4" />, strokeWidth: 1.8 },
  chevronLeft: { body: <path d="m10 3-5 5 5 5" />, strokeWidth: 1.8 },
  chevronRight: { body: <path d="m6 4 4 4-4 4" />, strokeWidth: 1.8 },
  arrowRight: { body: <path d="M3 8h10M9 4l4 4-4 4" />, strokeWidth: 1.8 },
  external: { body: <path d="M9 2h5v5M14 2 7.5 8.5M12 9.5V13a1 1 0 0 1-1 1H3a1 1 0 0 1-1-1V5a1 1 0 0 1 1-1h3.5" /> },
  undo: { body: <path d="M6 4 2 8l4 4M2 8h8a4 4 0 0 1 0 8" />, strokeWidth: 1.7 },
  redo: { body: <path d="m10 4 4 4-4 4M14 8H6a4 4 0 0 0 0 8" />, strokeWidth: 1.7 },
  move: { body: <path d="M8 2v12M2 8h12M5 5 8 2l3 3M5 11l3 3 3-3" /> },
  fullscreen: { body: <path d="M2 6V2h4M14 6V2h-4M2 10v4h4M14 10v4h-4" /> },

  // Actions
  plus: { body: <path d="M8 3v10M3 8h10" />, strokeWidth: 2 },
  check: { body: <path d="m3 8 3.5 3.5L13 5" />, strokeWidth: 2 },
  close: { body: <path d="M4 4l8 8M12 4l-8 8" />, strokeWidth: 1.8 },
  trash: { body: <path d="M3 4h10M6 4V2.5h4V4M5 4l.6 9h4.8L11 4" /> },
  duplicate: {
    body: (
      <>
        <rect x="5" y="5" width="9" height="9" rx="1.5" />
        <path d="M11 5V3.5A1.5 1.5 0 0 0 9.5 2h-6A1.5 1.5 0 0 0 2 3.5v6A1.5 1.5 0 0 0 3.5 11H5" />
      </>
    ),
  },
  copy: {
    body: (
      <>
        <rect x="5" y="5" width="9" height="9" rx="1.5" />
        <path d="M11 5V3.5A1.5 1.5 0 0 0 9.5 2h-6A1.5 1.5 0 0 0 2 3.5v6A1.5 1.5 0 0 0 3.5 11H5" />
      </>
    ),
  },
  search: {
    body: (
      <>
        <circle cx="7" cy="7" r="4.5" />
        <path d="m13.5 13.5-3.2-3.2" />
      </>
    ),
    strokeWidth: 1.8,
  },
  filter: { body: <path d="M2 3h12l-4.5 5.5V13l-3 1V8.5z" strokeLinejoin="round" /> },
  more: {
    body: (
      <>
        <circle cx="3.5" cy="8" r="1" fill="currentColor" stroke="none" />
        <circle cx="8" cy="8" r="1" fill="currentColor" stroke="none" />
        <circle cx="12.5" cy="8" r="1" fill="currentColor" stroke="none" />
      </>
    ),
  },
  edit: { body: <path d="M11 2.5 13.5 5 6 12.5H3.5V10z" strokeLinejoin="round" /> },
  refresh: { body: <path d="M13.5 8a5.5 5.5 0 1 1-1.6-3.9M13.5 2.5V5H11" /> },
  signOut: { body: <path d="M6 2H3v12h3M10 11l3-3-3-3M13 8H6" /> },
  pause: { body: <path d="M5.5 3v10M10.5 3v10" strokeWidth={2} /> },
  play: { body: <path d="M5 3l8 5-8 5z" strokeLinejoin="round" /> },
  stop: { body: <rect x="4" y="4" width="8" height="8" rx="1" /> },
  send: { body: <path d="M14 2 2 7l5 2 2 5zM7 9l7-7" strokeLinejoin="round" /> },
  calendar: {
    body: (
      <>
        <rect x="2" y="3" width="12" height="11" rx="1.5" />
        <path d="M2 6.5h12M5.5 1.5v3M10.5 1.5v3" />
      </>
    ),
  },
  clock: {
    body: (
      <>
        <circle cx="8" cy="8" r="6" />
        <path d="M8 4.5V8l2.5 1.5" />
      </>
    ),
  },
  link: { body: <path d="M6.5 9.5a3 3 0 0 0 4.2 0l2.1-2.1a3 3 0 0 0-4.2-4.2l-.8.8M9.5 6.5a3 3 0 0 0-4.2 0L3.2 8.6a3 3 0 0 0 4.2 4.2l.8-.8" /> },
  code: { body: <path d="M5.5 4 2 8l3.5 4M10.5 4 14 8l-3.5 4" /> },
  eye: {
    body: (
      <>
        <circle cx="8" cy="8" r="2.5" />
        <path d="M1.5 8s2.5-4.5 6.5-4.5S14.5 8 14.5 8 12 12.5 8 12.5 1.5 8 1.5 8z" />
      </>
    ),
    strokeWidth: 1.8,
  },
  cursor: { body: <path d="M3 2.5 13 7l-5 1-2.5 5z" strokeLinejoin="round" />, strokeWidth: 1.8 },
  mail: {
    body: (
      <>
        <rect x="2" y="3" width="12" height="10" rx="2" />
        <path d="m2 5 6 4 6-4" />
      </>
    ),
  },
  star: { body: <path d="M8 2l1.8 3.7 4 .6-2.9 2.8.7 4L8 11.2 4.4 13l.7-4L2.2 6.3l4-.6z" /> },
  user: {
    body: (
      <>
        <circle cx="8" cy="5.5" r="3" />
        <path d="M2.5 14a5.5 5.5 0 0 1 11 0" />
      </>
    ),
  },
  lock: {
    body: (
      <>
        <rect x="2" y="7" width="12" height="7" rx="1.5" />
        <path d="M5 7V5a3 3 0 0 1 6 0v2" />
      </>
    ),
    strokeWidth: 1.8,
  },
  key: {
    body: (
      <>
        <circle cx="5" cy="10" r="3" />
        <path d="M7.2 7.8 13 2M10.5 4.5 12 6M12 3l1.5 1.5" />
      </>
    ),
  },
  shield: { body: <path d="M8 1.5 13.5 3.5v4.5c0 3-2.3 5.3-5.5 6.5C4.8 13.3 2.5 11 2.5 8V3.5z" strokeLinejoin="round" /> },
  rss: { body: <path d="M3 11a2 2 0 1 1 0 .01M3 7a6 6 0 0 1 6 6M3 3a10 10 0 0 1 10 10" />, strokeWidth: 1.8 },
  globe: {
    body: (
      <>
        <circle cx="8" cy="8" r="6" />
        <path d="M2 8h12M8 2c1.8 1.9 2.6 3.9 2.6 6S9.8 12.1 8 14C6.2 12.1 5.4 10.1 5.4 8S6.2 3.9 8 2z" />
      </>
    ),
  },
  bolt: { body: <path d="M9 1.5 3.5 9H8l-1 5.5L12.5 7H8z" strokeLinejoin="round" /> },
  webhook: {
    body: (
      <>
        <circle cx="4" cy="11.5" r="2" />
        <circle cx="12" cy="11.5" r="2" />
        <circle cx="8" cy="4" r="2" />
        <path d="M6 11.5h4M7 5.8 5 9.7M9 5.8l2 3.9" />
      </>
    ),
  },
  sparkle: { body: <path d="M8 2v3M8 11v3M2 8h3M11 8h3M4 4l2 2M10 10l2 2M12 4l-2 2M6 10l-2 2" /> },
  grid: { body: <path d="M2 2h5v5H2zM9 2h5v5H9zM2 9h5v5H2zM9 9h5v5H9z" /> },
  list: { body: <path d="M3 5h10M3 8h10M3 11h6" /> },
  menu: { body: <path d="M2 4h12M2 8h12M2 12h12" /> },
  desktop: {
    body: (
      <>
        <rect x="1.5" y="2.5" width="13" height="8.5" rx="1.2" />
        <path d="M5.5 14h5M8 11v3" />
      </>
    ),
  },
  mobile: {
    body: (
      <>
        <rect x="4.5" y="1.5" width="7" height="13" rx="1.5" />
        <path d="M7 12.2h2" />
      </>
    ),
  },
  moon: { body: <path d="M13 9.5A5.5 5.5 0 0 1 6.5 3a5.5 5.5 0 1 0 6.5 6.5z" strokeLinejoin="round" /> },
  image: {
    body: (
      <>
        <rect x="2" y="3" width="12" height="10" rx="1.5" />
        <circle cx="5.5" cy="6.5" r="1.2" />
        <path d="m2.5 12 3.5-3.5 2.5 2.5 2-2 3 3" />
      </>
    ),
  },
  upload: { body: <path d="M8 11V3M4.5 6.5 8 3l3.5 3.5M2 13h12" /> },
  download: { body: <path d="M8 3v8M4.5 7.5 8 11l3.5-3.5M2 13h12" /> },
  print: {
    body: (
      <>
        <path d="M4 6V2h8v4M4 12H2.5V7.5a1.5 1.5 0 0 1 1.5-1.5h8a1.5 1.5 0 0 1 1.5 1.5V12H12" />
        <rect x="4" y="9.5" width="8" height="4.5" rx=".5" />
      </>
    ),
  },
  history: { body: <path d="M2.5 8a5.5 5.5 0 1 0 1.6-3.9M2.5 2.5V5H5M8 5v3l2 1.5" /> },
  tag: {
    body: (
      <>
        <path d="M2 2h5.5L14 8.5 8.5 14 2 7.5z" strokeLinejoin="round" />
        <circle cx="5" cy="5" r="1" />
      </>
    ),
  },
  passkey: {
    body: (
      <>
        <rect x="2" y="7" width="12" height="7" rx="1.5" />
        <path d="M5 7V5a3 3 0 0 1 6 0v2" />
      </>
    ),
    strokeWidth: 1.8,
  },

  table: {
    body: (
      <>
        <rect x="2" y="2" width="12" height="12" rx="2" />
        <path d="M2 6h12M6 6v8" />
      </>
    ),
  },
  instance: {
    body: (
      <>
        <rect x="2" y="3" width="12" height="10" rx="2" />
        <path d="M5 7h6M5 10h3" />
      </>
    ),
  },

  // Status
  alert: { body: <path d="M8 3v6M8 12v.5" />, strokeWidth: 2 },
  alertCircle: {
    body: (
      <>
        <circle cx="10" cy="10" r="8" />
        <path d="M10 6v5M10 13.5v.5" strokeLinecap="round" />
      </>
    ),
    viewBox: '0 0 20 20',
    strokeWidth: 1.8,
  },
  info: {
    body: (
      <>
        <circle cx="8" cy="8" r="6" />
        <path d="M8 7.5V11M8 5v.5" />
      </>
    ),
    strokeWidth: 1.8,
  },
  bounce: { body: <path d="M8 2v7M8 12v1" />, strokeWidth: 1.8 },
  warningTriangle: {
    body: (
      <>
        <path d="M12 9v4M12 17h.01" />
        <path d="M10.3 3.9 2.4 18a2 2 0 0 0 1.7 3h15.8a2 2 0 0 0 1.7-3L13.7 3.9a2 2 0 0 0-3.4 0z" />
      </>
    ),
    viewBox: '0 0 24 24',
    strokeWidth: 2,
  },
  searchLarge: {
    body: (
      <>
        <circle cx="11" cy="11" r="7" />
        <path d="m20 20-4-4" />
      </>
    ),
    viewBox: '0 0 24 24',
    strokeWidth: 2,
  },
  lockLarge: {
    body: (
      <>
        <rect x="4" y="10" width="16" height="11" rx="2" />
        <path d="M8 10V7a4 4 0 0 1 8 0v3" />
      </>
    ),
    viewBox: '0 0 24 24',
    strokeWidth: 2,
  },
  // Instance admin (user actions menu).
  terminal: { body: <path d="M3 4l4 4-4 4M8 12h5" /> },
  ban: {
    body: (
      <>
        <circle cx="8" cy="8" r="6" />
        <path d="M4 4l8 8" />
      </>
    ),
  },
} satisfies Record<string, IconDef>

export type IconName = keyof typeof icons

export interface IconProps extends Omit<SVGProps<SVGSVGElement>, 'name'> {
  name: IconName
  size?: number
  label?: string
}

export function Icon({ name, size = 16, label, strokeWidth, ...rest }: IconProps) {
  const def: IconDef = icons[name]
  return (
    <svg
      width={size}
      height={size}
      viewBox={def.viewBox ?? '0 0 16 16'}
      fill="none"
      stroke="currentColor"
      strokeWidth={strokeWidth ?? def.strokeWidth ?? 1.6}
      strokeLinecap="round"
      strokeLinejoin="round"
      aria-hidden={label ? undefined : true}
      role={label ? 'img' : undefined}
      aria-label={label}
      focusable="false"
      {...rest}
    >
      {def.body}
    </svg>
  )
}

/** Brand marks used on sign-in buttons. They keep their own colours. */
export function GoogleMark({ size = 16 }: { size?: number }) {
  return (
    <svg width={size} height={size} viewBox="0 0 16 16" aria-hidden="true" focusable="false">
      <path d="M14.5 8.2c0-.5 0-.9-.1-1.3H8v2.5h3.7c-.2.9-.7 1.6-1.4 2.1v1.7h2.2c1.3-1.2 2-3 2-5z" fill="#4285F4" />
      <path d="M8 14.7c1.8 0 3.3-.6 4.4-1.6l-2.2-1.7c-.6.4-1.4.7-2.2.7-1.7 0-3.2-1.2-3.7-2.7H2.1v1.7A6.7 6.7 0 0 0 8 14.7z" fill="#34A853" />
      <path d="M4.3 9.4a4 4 0 0 1 0-2.6V5.1H2.1a6.7 6.7 0 0 0 0 6l2.2-1.7z" fill="#FBBC05" />
      <path d="M8 4c1 0 1.9.3 2.6 1l1.9-1.9A6.6 6.6 0 0 0 8 1.3a6.7 6.7 0 0 0-5.9 3.8l2.2 1.7C4.8 5.2 6.3 4 8 4z" fill="#EA4335" />
    </svg>
  )
}

export function MicrosoftMark({ size = 16 }: { size?: number }) {
  return (
    <svg width={size} height={size} viewBox="0 0 16 16" aria-hidden="true" focusable="false">
      <rect x="1" y="1" width="6.5" height="6.5" fill="#F25022" />
      <rect x="8.5" y="1" width="6.5" height="6.5" fill="#7FBA00" />
      <rect x="1" y="8.5" width="6.5" height="6.5" fill="#00A4EF" />
      <rect x="8.5" y="8.5" width="6.5" height="6.5" fill="#FFB900" />
    </svg>
  )
}

export function GitHubMark({ size = 16 }: { size?: number }) {
  return (
    <svg width={size} height={size} viewBox="0 0 16 16" aria-hidden="true" focusable="false">
      <path
        fill="currentColor"
        d="M8 .8a7.2 7.2 0 0 0-2.3 14c.4.1.5-.2.5-.4v-1.3c-2 .4-2.4-1-2.4-1-.3-.8-.8-1-.8-1-.7-.5 0-.5 0-.5.7.1 1.1.8 1.1.8.6 1.1 1.7.8 2.1.6.1-.5.3-.8.5-1-1.6-.2-3.3-.8-3.3-3.6 0-.8.3-1.4.7-1.9-.1-.2-.3-.9.1-1.9 0 0 .6-.2 2 .7a6.9 6.9 0 0 1 3.6 0c1.4-.9 2-.7 2-.7.4 1 .2 1.7.1 1.9.5.5.7 1.1.7 1.9 0 2.8-1.7 3.4-3.3 3.6.3.2.5.7.5 1.3v2c0 .2.1.5.5.4A7.2 7.2 0 0 0 8 .8z"
      />
    </svg>
  )
}

/** The ReputeCast mark: an envelope whose flap is a check. */
export function LogoMark({ size = 26, title }: { size?: number; title?: string }) {
  return (
    <svg width={size} height={size} viewBox="0 0 32 32" role={title ? 'img' : undefined} aria-label={title} aria-hidden={title ? undefined : true} focusable="false">
      <rect width="32" height="32" rx="8" fill="#0F766E" />
      <path d="M7 12h10.5M7 12v10.5A1.5 1.5 0 0 0 8.5 24h15a1.5 1.5 0 0 0 1.5-1.5V16.5" fill="none" stroke="#FFFFFF" strokeWidth="2.4" strokeLinecap="round" strokeLinejoin="round" />
      <path d="M7 12l9 7L27 8" fill="none" stroke="#FFFFFF" strokeWidth="2.4" strokeLinecap="round" strokeLinejoin="round" />
    </svg>
  )
}
