/**
 * The assistant panel's icons, copied from the AssistantPanel board (a
 * 24-unit grid, 1.7 stroke). Decorative: the control carries the label.
 */
const paths = {
  threads: 'M8 6h12M8 12h12M8 18h12M4 6h.01M4 12h.01M4 18h.01',
  expand: 'M14 4h6v6M10 20H4v-6M20 4l-7 7M4 20l7-7',
  x: 'M6 6l12 12M18 6 6 18',
  send: 'M4 12 20 4l-6 16-3-7Z',
  plus: 'M12 5v14M5 12h14',
  check: 'm5 12 4 4L19 6',
  tool: 'M14.5 6.5a4 4 0 0 0-5.4 5.2L4 16.8V20h3.2l5.1-5.1a4 4 0 0 0 5.2-5.4l-2.5 2.5-2.4-.6-.6-2.4Z',
  link: 'M10 14a4 4 0 0 0 5.7 0l3-3a4 4 0 0 0-5.7-5.7l-1 1M14 10a4 4 0 0 0-5.7 0l-3 3a4 4 0 0 0 5.7 5.7l1-1',
  warn: 'M12 4 2.5 20h19ZM12 10v4M12 17h.01',
  shield: 'M12 3 4 6v6c0 5 8 9 8 9s8-4 8-9V6Z',
  mail: 'M3 7a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2v10a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2ZM3 6l9 7 9-7',
  chevron: 'm8 10 4 4 4-4',
  back: 'M15 6 9 12l6 6',
  clock: 'M12 3a9 9 0 1 0 0 18 9 9 0 0 0 0-18ZM12 7v5l3 2',
  more: 'M5 12h.01M12 12h.01M19 12h.01',
  copy: 'M9 9h10v10H9ZM5 15V5h10',
} as const

export type AsstIconName = keyof typeof paths | 'stop'

export function AsstIcon({ name, size = 18, className }: { name: AsstIconName; size?: number; className?: string }) {
  return (
    <svg className={className} width={size} height={size} viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth={name === 'more' ? 2.4 : 1.7} strokeLinecap="round" strokeLinejoin="round" aria-hidden="true" focusable="false">
      {name === 'stop' ? <rect x="7" y="7" width="10" height="10" rx="1.5" /> : <path d={paths[name]} />}
    </svg>
  )
}
