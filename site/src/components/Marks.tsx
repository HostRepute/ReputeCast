/** The ReputeCast mark: an envelope whose flap is a check (brand/logo). */
export function LogoMark({ size }: { size: number }) {
  return (
    <svg width={size} height={size} viewBox="0 0 32 32" aria-hidden="true" focusable="false">
      <rect width="32" height="32" rx="8" fill="#0F766E" />
      <path
        d="M7 12h10.5M7 12v10.5A1.5 1.5 0 0 0 8.5 24h15a1.5 1.5 0 0 0 1.5-1.5V16.5"
        fill="none"
        stroke="#FFFFFF"
        strokeWidth="2.4"
        strokeLinecap="round"
        strokeLinejoin="round"
      />
      <path d="M7 12l9 7L27 8" fill="none" stroke="#FFFFFF" strokeWidth="2.4" strokeLinecap="round" strokeLinejoin="round" />
    </svg>
  )
}

export function GitHubMark() {
  return (
    <svg width="16" height="16" viewBox="0 0 16 16" fill="currentColor" aria-hidden="true" focusable="false">
      <path d="M8 .5a7.5 7.5 0 0 0-2.4 14.6c.4.1.5-.2.5-.4v-1.4c-2.1.5-2.5-.9-2.5-.9-.3-.9-.8-1.1-.8-1.1-.7-.5.1-.5.1-.5.8.1 1.2.8 1.2.8.7 1.2 1.8.8 2.2.6.1-.5.3-.8.5-1-1.7-.2-3.4-.8-3.4-3.7 0-.8.3-1.5.8-2-.1-.2-.3-1 .1-2 0 0 .6-.2 2.1.8a7 7 0 0 1 3.8 0c1.4-1 2.1-.8 2.1-.8.4 1 .2 1.8.1 2 .5.5.8 1.2.8 2 0 2.9-1.7 3.5-3.4 3.7.3.2.5.7.5 1.4v2.1c0 .2.1.5.5.4A7.5 7.5 0 0 0 8 .5z" />
    </svg>
  )
}
