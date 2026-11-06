import type { ReactNode } from 'react'

/** A subset of the app's stroke icons (web/src/components/icons/Icon.tsx), on the same 16-unit grid. */
const icons = {
  edit: <path d="M11 2.5 13.5 5 6 12.5H3.5V10z" />,
  automations: (
    <>
      <circle cx="4" cy="3" r="2" />
      <circle cx="4" cy="13" r="2" />
      <circle cx="12" cy="8" r="2" />
      <path d="M4 5v6M6 8h4" />
    </>
  ),
  forms: (
    <>
      <rect x="2" y="2" width="12" height="12" rx="2" />
      <path d="M5 6h6M5 9h6M5 12h3" />
    </>
  ),
  desktop: (
    <>
      <rect x="1.5" y="2.5" width="13" height="8.5" rx="1.2" />
      <path d="M5.5 14h5M8 11v3" />
    </>
  ),
  transactional: (
    <>
      <rect x="1.5" y="3.5" width="13" height="9" rx="1.5" />
      <path d="m2 4.5 6 4.5 6-4.5" />
    </>
  ),
  mobile: (
    <>
      <rect x="4.5" y="1.5" width="7" height="13" rx="1.5" />
      <path d="M7 12.2h2" />
    </>
  ),
  language: (
    <>
      <circle cx="8" cy="8" r="6.2" />
      <path d="M1.8 8h12.4M8 1.8c1.8 1.7 2.7 3.8 2.7 6.2S9.8 12.5 8 14.2C6.2 12.5 5.3 10.4 5.3 8S6.2 3.5 8 1.8z" />
    </>
  ),
  providers: (
    <>
      <rect x="2" y="2" width="12" height="5" rx="1.5" />
      <rect x="2" y="9" width="12" height="5" rx="1.5" />
      <path d="M5 4.5h.01M5 11.5h.01" strokeWidth={2} />
    </>
  ),
  earn: (
    <>
      <circle cx="8" cy="8" r="6.2" />
      <path d="M10 5.9H7.1a1.3 1.3 0 0 0 0 2.6h1.8a1.3 1.3 0 0 1 0 2.6H6M8 4.4v1.5M8 11.1v1.5" />
    </>
  ),
  barChart: <path d="M2 14h12M4.2 11.5V8.2M8 11.5V3.8M11.8 11.5V6.2" />,
  filter: <path d="M2 3h12l-4.5 5.5V13l-3 1V8.5z" />,
  audience: (
    <>
      <circle cx="6" cy="5" r="2.5" />
      <path d="M1.5 14a4.5 4.5 0 0 1 9 0M11 3a2.5 2.5 0 0 1 0 5M14.5 14a4 4 0 0 0-3-3.8" />
    </>
  ),
  key: (
    <>
      <circle cx="5" cy="10" r="3" />
      <path d="M7.2 7.8 13 2M10.5 4.5 12 6M12 3l1.5 1.5" />
    </>
  ),
  code: <path d="M5.5 4 2 8l3.5 4M10.5 4 14 8l-3.5 4" />,
  shield: <path d="M8 1.5 13.5 3.5v4.5c0 3-2.3 5.3-5.5 6.5C4.8 13.3 2.5 11 2.5 8V3.5z" />,
  import: <path d="M8 11V3M4.5 6.5 8 3l3.5 3.5M2 13h12" />,
  arrowRight: <path d="M3 8h10M9 4l4 4-4 4" strokeWidth={1.8} />,
} satisfies Record<string, ReactNode>

export type IconName = keyof typeof icons

/** Decorative: the text next to it carries the meaning. */
export function Icon({ name, size = 16 }: { name: IconName; size?: number }) {
  return (
    <svg
      width={size}
      height={size}
      viewBox="0 0 16 16"
      fill="none"
      stroke="currentColor"
      strokeWidth={1.6}
      strokeLinecap="round"
      strokeLinejoin="round"
      aria-hidden="true"
      focusable="false"
    >
      {icons[name]}
    </svg>
  )
}
