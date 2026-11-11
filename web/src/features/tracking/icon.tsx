/** The browser window with a pulse: Website tracking's tile (the boards' icon). */
export function TrackingIcon({ size = 18 }: { size?: number }) {
  return (
    <svg width={size} height={size} viewBox="0 0 16 16" fill="none" stroke="currentColor" strokeWidth="1.6" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
      <rect x="1.5" y="2.5" width="13" height="11" rx="1.5" />
      <path d="M1.5 5.5h13" />
      <path d="M3.8 10.2h1.9l1.1-2 1.6 3 1.2-2.1h2.6" />
    </svg>
  )
}
