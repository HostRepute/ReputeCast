import type { ReactNode } from 'react'

/**
 * Composer-only icons copied from the Composer boards: block tiles (24-unit
 * grid), the selection toolbar and the layer tree (16-unit grid).
 */
const blocks: Record<string, ReactNode> = {
  col1: <rect x="3" y="5" width="18" height="14" rx="2" />,
  col2: (
    <>
      <rect x="3" y="5" width="8" height="14" rx="2" />
      <rect x="13" y="5" width="8" height="14" rx="2" />
    </>
  ),
  col3: (
    <>
      <rect x="2" y="5" width="6" height="14" rx="1.5" />
      <rect x="9" y="5" width="6" height="14" rx="1.5" />
      <rect x="16" y="5" width="6" height="14" rx="1.5" />
    </>
  ),
  wrapper: (
    <>
      <rect x="2" y="3" width="20" height="18" rx="2" />
      <rect x="6" y="7" width="12" height="10" rx="1" />
    </>
  ),
  hero: (
    <>
      <rect x="3" y="4" width="18" height="16" rx="2" />
      <path d="M7 15h10M7 11h6" />
    </>
  ),
  group: <path d="M3 12h18M12 3v18" />,
  text: <path d="M4 7h16M4 12h16M4 17h10" />,
  image: (
    <>
      <rect x="3" y="4" width="18" height="16" rx="2" />
      <path d="m3 16 5-5 4 4 3-3 6 6" />
      <circle cx="16" cy="9" r="1.5" />
    </>
  ),
  button: <rect x="3" y="8" width="18" height="8" rx="4" />,
  divider: <path d="M3 12h18" />,
  spacer: <path d="M4 6v12M20 6v12M8 12h8" />,
  social: (
    <>
      <circle cx="6" cy="12" r="2.5" />
      <circle cx="12" cy="12" r="2.5" />
      <circle cx="18" cy="12" r="2.5" />
    </>
  ),
  navbar: <path d="M3 6h18M3 12h18M3 18h18" />,
  raw: <path d="m8 7-4 5 4 5M16 7l4 5-4 5" />,
  table: (
    <>
      <rect x="4" y="4" width="16" height="16" rx="2" />
      <path d="M8 9h8M8 13h8" />
    </>
  ),
  rcHeader: (
    <>
      <rect x="3" y="5" width="18" height="14" rx="2" />
      <circle cx="7.5" cy="9.5" r="1.5" />
      <path d="M12 9h6" />
    </>
  ),
  rcArticle: (
    <>
      <rect x="3" y="4" width="18" height="16" rx="2" />
      <path d="M6 8h12M6 12h8M6 16h6" />
    </>
  ),
  rcProduct: (
    <>
      <rect x="3" y="3" width="18" height="18" rx="2" />
      <rect x="6" y="6" width="12" height="7" rx="1" />
      <path d="M6 17h6" />
    </>
  ),
  rcFooter: (
    <>
      <rect x="3" y="12" width="18" height="8" rx="2" />
      <path d="M6 16h8" />
    </>
  ),
  // The Poll block (design expansion, X11): a question over answer buttons.
  rcPoll: (
    <>
      <rect x="3" y="4" width="18" height="16" rx="2" />
      <path d="M7 8h10" />
      <rect x="7" y="11" width="10" height="3" rx="1.5" />
      <rect x="7" y="15.5" width="7" height="2" rx="1" />
    </>
  ),
  // The Sponsor block (Earn): a card with the sponsor's logo beside its lines.
  rcSponsor: (
    <>
      <rect x="3" y="4" width="18" height="16" rx="2" />
      <rect x="6" y="8" width="4" height="4" rx="1" />
      <path d="M12.5 9h5M12.5 12h3.5M6 16h12" />
    </>
  ),
  // The Upgrade block (Earn): a card with its price line and button.
  rcUpgrade: (
    <>
      <rect x="3" y="4" width="18" height="16" rx="2" />
      <path d="M8 9h8M9.5 12h5" />
      <rect x="8" y="15" width="8" height="2.5" rx="1.25" />
    </>
  ),
  // The Product grid (X20): three products side by side.
  rcGrid: (
    <>
      <rect x="2.5" y="5" width="5.5" height="7" rx="1" />
      <rect x="9.25" y="5" width="5.5" height="7" rx="1" />
      <rect x="16" y="5" width="5.5" height="7" rx="1" />
      <path d="M3 16h4.5M9.75 16h4.5M16.5 16h4.5" />
    </>
  ),
  saved: <path d="M12 3v18M3 12h18" />,
  // A recurring digest's Feed items block.
  rcFeed: <path d="M5 19a1 1 0 1 1 0 .01M5 13a6 6 0 0 1 6 6M5 7a12 12 0 0 1 12 12" />,
  rcReceipt: (
    <>
      <rect x="3" y="4" width="18" height="16" rx="2" />
      <path d="M7 9h10M7 13h6" />
    </>
  ),
  // Grow's blocks (design expansion, X15): Referral, Recommendations, Reward.
  growReferral: (
    <>
      <circle cx="9" cy="8" r="3.2" />
      <path d="M3 19.5a6 6 0 0 1 12 0" />
      <path d="M18.5 7v6M15.5 10h6" />
    </>
  ),
  growRecs: (
    <>
      <rect x="3" y="4" width="18" height="7" rx="1.6" />
      <rect x="3" y="13" width="18" height="7" rx="1.6" />
      <path d="M6.5 7.5h6M6.5 16.5h6" />
    </>
  ),
  growReward: (
    <>
      <rect x="3.5" y="9.5" width="17" height="11" rx="1.6" />
      <path d="M3 9.5h18M12 9.5v11" />
      <path d="M12 9.5c-1.2-3.6-5-4.4-5-2.3S10 9.5 12 9.5zm0 0c1.2-3.6 5-4.4 5-2.3S14 9.5 12 9.5z" />
    </>
  ),
}

export function BlockIcon({ name }: { name: string }) {
  return (
    <svg aria-hidden="true" viewBox="0 0 24 24" focusable="false">
      {blocks[name] ?? blocks.col1}
    </svg>
  )
}

const tool: Record<string, ReactNode> = {
  parent: <path d="m4 10 4-4 4 4" />,
  move: <path d="M8 2v12M2 8h12M5 5 8 2l3 3M5 11l3 3 3-3" />,
  duplicate: (
    <>
      <rect x="5" y="5" width="9" height="9" rx="1.5" />
      <path d="M11 5V3.5A1.5 1.5 0 0 0 9.5 2h-6A1.5 1.5 0 0 0 2 3.5v6A1.5 1.5 0 0 0 3.5 11H5" />
    </>
  ),
  delete: <path d="M3 4h10M6 4V2.5h4V4M5 4l.6 9h4.8L11 4" />,
  chevronRight: <path d="m6 4 4 4-4 4" />,
  chevronDown: <path d="m4 6 4 4 4-4" />,
  layImage: <rect x="2" y="3" width="12" height="10" rx="1.5" />,
  layText: <path d="M3 5h10M3 8h10M3 11h6" />,
  layButton: <rect x="2" y="5" width="12" height="6" rx="3" />,
  layDivider: <path d="M2 8h12" />,
  layDot: <circle cx="8" cy="8" r="1.5" />,
  warn: <path d="M8 3v6M8 12v.5" />,
  audience: (
    <>
      <circle cx="6" cy="5" r="2.5" />
      <path d="M1.5 14a4.5 4.5 0 0 1 9 0M11 3a2.5 2.5 0 0 1 0 5M14.5 14a4 4 0 0 0-3-3.8" />
    </>
  ),
  back: <path d="m10 3-5 5 5 5" />,
  undo: <path d="M6 4 2 8l4 4M2 8h8a4 4 0 0 1 0 8" />,
  redo: <path d="m10 4 4 4-4 4M14 8H6a4 4 0 0 0 0 8" />,
  // The top bar, the Inbox strip and quick review (CO-14).
  desktop: (
    <>
      <rect x="1.5" y="2.5" width="13" height="8.5" rx="1.2" />
      <path d="M6 14h4M8 11v3" />
    </>
  ),
  mobile: (
    <>
      <rect x="4.5" y="1.5" width="7" height="13" rx="1.5" />
      <path d="M7.25 12.25h1.5" />
    </>
  ),
  eye: (
    <>
      <path d="M1.5 8S3.9 3.5 8 3.5 14.5 8 14.5 8 12.1 12.5 8 12.5 1.5 8 1.5 8z" />
      <circle cx="8" cy="8" r="2" />
    </>
  ),
  globe: (
    <>
      <circle cx="8" cy="8" r="6.2" />
      <path d="M1.8 8h12.4M8 1.8c1.8 1.7 2.7 3.8 2.7 6.2S9.8 12.5 8 14.2C6.2 12.5 5.3 10.4 5.3 8S6.2 3.5 8 1.8z" />
    </>
  ),
  check: <path d="m3 8.5 3.2 3L13 4.5" />,
  mail: (
    <>
      <rect x="1.5" y="3" width="13" height="10" rx="1.5" />
      <path d="m2 4.2 6 4.8 6-4.8" />
    </>
  ),
  pencil: <path d="M10.6 2.6a1.5 1.5 0 0 1 2.1 0l.7.7a1.5 1.5 0 0 1 0 2.1L6 12.8l-3.3.6.6-3.3z" />,
  shield: (
    <>
      <path d="M8 1.5 13 3.4v4.2c0 3-2.1 5.4-5 6.5-2.9-1.1-5-3.5-5-6.5V3.4z" />
      <path d="m5.6 8 1.7 1.7 3.2-3.2" />
    </>
  ),
  comment: <path d="M2.5 3.5h11v7.5H7l-3 2.5V11H2.5z" />,
  send: <path d="M14.2 1.8 7 9M14.2 1.8l-4.4 12.4-2.8-5.2-5.2-2.8z" />,
  alignLeft: <path d="M2 4h12M2 8h8M2 12h11" />,
  alignCenter: <path d="M2 4h12M4 8h8M3 12h10" />,
  alignRight: <path d="M2 4h12M6 8h8M3 12h11" />,
  info: (
    <>
      <circle cx="8" cy="8" r="6.3" />
      <path d="M8 7.2v4M8 4.8v.2" />
    </>
  ),
}

export function ToolIcon({ name, size = 14, strokeWidth = 1.8 }: { name: string; size?: number; strokeWidth?: number }) {
  return (
    <svg width={size} height={size} viewBox="0 0 16 16" fill="none" stroke="currentColor" strokeWidth={strokeWidth} aria-hidden="true" focusable="false">
      {tool[name]}
    </svg>
  )
}
