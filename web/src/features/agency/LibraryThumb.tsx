import { useEffect, useRef, useState } from 'react'
import type { Schemas } from '@/api/client'

type Item = Pick<Schemas['AgencyLibraryItem'], 'kind' | 'thumb' | 'background' | 'accent' | 'steps' | 'name' | 'columns'>
type ThumbItem = { h?: number; w?: number; accent?: boolean; row?: ThumbItem[] }

/** How dark a colour is (0 white … 1 black), to choose light or dark marks on it. */
function darkness(hex: string | undefined) {
  const m = /^#?([0-9a-f]{6})$/i.exec(hex ?? '')
  if (!m) return 0
  const n = parseInt(m[1]!, 16)
  const r = (n >> 16) & 255
  const g = (n >> 8) & 255
  const b = n & 255
  return 1 - (0.299 * r + 0.587 * g + 0.114 * b) / 255
}

/** A light tint of a colour for blocks ("#92400E" → "#FEF3C7"-like). */
function tint(hex: string | undefined) {
  return darkness(hex) > 0 ? '#FEF3C7' : '#E3F1EE'
}

/**
 * A library item's card picture (AgencyLibrary): a wireframe in the item's
 * colours, drawn by kind: a hero or a row of products for sections, a page
 * for campaign and transactional templates, a flow for automations, a
 * message for SMS texts.
 */
export function LibraryThumb({ item, height = 104 }: { item: Item; height?: number }) {
  const thumb = (Array.isArray(item.thumb) ? item.thumb : []) as ThumbItem[]
  const bg = item.background
  const accent = item.accent ?? '#0F766E'
  const box = { width: '100%', height, viewBox: '0 0 220 104', preserveAspectRatio: 'xMidYMid meet', 'aria-hidden': true } as const
  if (item.kind === 'automation') {
    const n = Math.min(3, Math.max(2, (item.steps ?? 3) - 1))
    const ys = n === 3 ? [10, 40, 72] : [20, 60]
    return (
      <svg {...box} style={{ display: 'block', background: '#F3F4F6' }}>
        <rect width="220" height="104" fill="#F3F4F6" />
        {ys.slice(1).map((y, i) => (
          <path key={y} d={`M110 ${ys[i]! + 20}v${y - ys[i]! - 20}`} stroke="#9AA3B2" strokeWidth="1.5" />
        ))}
        {ys.map((y, i) => {
          const mid = i === Math.floor(ys.length / 2) && ys.length === 3
          return (
            <g key={y}>
              <rect x="74" y={y} width="72" height={mid ? 22 : 20} rx="6" fill={mid ? '#E3F1EE' : '#FFFFFF'} stroke={mid ? '#0F766E' : '#C9CED6'} />
              <rect x="86" y={y + (mid ? 9 : 8)} width="48" height="4" rx="2" fill={mid ? '#0F766E' : '#9AA3B2'} />
            </g>
          )
        })}
      </svg>
    )
  }
  if (item.kind === 'sms') {
    return (
      <svg {...box} style={{ display: 'block', background: '#F3F4F6' }}>
        <rect width="220" height="104" fill="#F3F4F6" />
        <path d="M44 18h132a8 8 0 0 1 8 8v40a8 8 0 0 1-8 8H76l-14 12V74H44a8 8 0 0 1-8-8V26a8 8 0 0 1 8-8z" fill="#FFFFFF" stroke="#E3E6EB" />
        <rect x="48" y="30" width="96" height="5" rx="2.5" fill="#9AA3B2" />
        <rect x="48" y="41" width="122" height="5" rx="2.5" fill="#9AA3B2" />
        <rect x="48" y="52" width="70" height="5" rx="2.5" fill="#1D4ED8" opacity=".7" />
        <rect x="48" y="63" width="84" height="4" rx="2" fill="#C9CED6" />
      </svg>
    )
  }
  if (item.kind === 'transactional') {
    return (
      <svg {...box} style={{ display: 'block', background: '#F3F4F6' }}>
        <rect width="220" height="104" fill="#F3F4F6" />
        <rect x="52" y="8" width="116" height="96" rx="3" fill="#FFFFFF" stroke="#E3E6EB" />
        <rect x="60" y="16" width="34" height="5" rx="2.5" fill="#0B1220" />
        <rect x="60" y="30" width="80" height="6" rx="3" fill="#9AA3B2" />
        <rect x="60" y="42" width="100" height="4" rx="2" fill="#C9CED6" />
        <rect x="60" y="50" width="92" height="4" rx="2" fill="#C9CED6" />
        <rect x="60" y="62" width="100" height="18" rx="2" fill="#F9FAFB" stroke="#EEF0F3" />
        <rect x="60" y="86" width="40" height="10" rx="3" fill={accent} />
      </svg>
    )
  }
  if (item.kind === 'template') {
    const block = bg ?? tint(accent)
    return (
      <svg {...box} style={{ display: 'block', background: '#F3F4F6' }}>
        <rect width="220" height="104" fill="#F3F4F6" />
        <rect x="40" y="8" width="140" height="96" rx="3" fill="#FFFFFF" stroke="#E3E6EB" />
        <rect x="48" y="16" width="40" height="5" rx="2.5" fill="#0B1220" />
        <rect x="48" y="28" width="124" height="34" rx="2" fill={block} />
        <rect x="48" y="68" width="100" height="5" rx="2.5" fill="#9AA3B2" />
        <rect x="48" y="78" width="84" height="5" rx="2.5" fill="#C9CED6" />
        <rect x="48" y="88" width="44" height="10" rx="3" fill={accent} />
      </svg>
    )
  }
  // Sections: a row of products, or a hero on its colour.
  const row = thumb.find((t) => t.row && t.row.length > 1)?.row
  const cols = Math.max(row?.length ?? 0, item.columns ?? 0)
  if (cols >= 3 || (row && cols >= 2)) {
    const n = Math.min(3, cols)
    const w = 56
    const gap = (220 - n * w) / (n + 1)
    return (
      <svg {...box} style={{ display: 'block', background: '#F3F4F6' }}>
        <rect width="220" height="104" fill="#F3F4F6" />
        {Array.from({ length: n }, (_, i) => {
          const x = gap + i * (w + gap)
          return (
            <g key={i}>
              <rect x={x} y="14" width={w} height="44" rx="5" fill="#FFFFFF" stroke="#E3E6EB" />
              <rect x={x} y="64" width="44" height="5" rx="2.5" fill="#9AA3B2" />
              <rect x={x} y="74" width="30" height="5" rx="2.5" fill="#C9CED6" />
              <rect x={x} y="84" width="38" height="10" rx="5" fill={accent} />
            </g>
          )
        })}
      </svg>
    )
  }
  const dark = darkness(bg) > 0.5
  const fill = bg ?? '#F3F4F6'
  const mark = dark ? '#FFFFFF' : '#9AA3B2'
  const dots = dark
    ? [
        [18, 14, 1.6],
        [44, 30, 1.2],
        [70, 10, 1.8],
        [96, 26, 1.2],
        [128, 12, 1.6],
        [150, 34, 1.3],
        [176, 16, 1.8],
        [204, 28, 1.2],
        [30, 52, 1.2],
        [190, 56, 1.5],
        [12, 84, 1.4],
        [206, 88, 1.2],
      ]
    : []
  const hasButton = thumb.some((t) => t.accent)
  return (
    <svg {...box} style={{ display: 'block', background: fill }}>
      <rect width="220" height="104" fill={fill} />
      {dots.map(([x, y, r]) => (
        <circle key={`${x}-${y}`} cx={x} cy={y} r={r} fill="#FFFFFF" opacity=".75" />
      ))}
      <rect x="52" y={hasButton ? 40 : 44} width="116" height="9" rx="4.5" fill={mark} />
      <rect x="70" y={hasButton ? 55 : 59} width="80" height="5" rx="2.5" fill={mark} opacity=".6" />
      {hasButton ? <rect x="84" y="70" width="52" height="14" rx="7" fill={dark ? tint(accent) : accent} /> : null}
    </svg>
  )
}

/**
 * An email's HTML at its own width (600 px), scaled down to the space it
 * gets. Sandboxed: no scripts run.
 */
export function HtmlPreview({ html, label, maxHeight = 360 }: { html: string; label: string; maxHeight?: number }) {
  const box = useRef<HTMLDivElement>(null)
  const frame = useRef<HTMLIFrameElement>(null)
  const [scale, setScale] = useState(0.66)
  const [height, setHeight] = useState(240)
  useEffect(() => {
    const el = box.current
    if (!el) return
    const ro = new ResizeObserver(() => setScale(Math.min(1, el.clientWidth / 600)))
    ro.observe(el)
    return () => ro.disconnect()
  }, [])
  const measure = () => {
    const doc = frame.current?.contentDocument
    if (doc?.body) setHeight(doc.body.scrollHeight)
  }
  const shown = Math.min(maxHeight, Math.round(height * scale))
  return (
    <div ref={box} className="overflow-hidden rounded-md border border-line-2 bg-white" style={{ height: shown }}>
      <iframe
        ref={frame}
        title={label}
        srcDoc={html}
        sandbox="allow-same-origin"
        onLoad={measure}
        style={{ width: 600, height, border: 0, transform: `scale(${scale})`, transformOrigin: '0 0', pointerEvents: 'none' }}
        tabIndex={-1}
      />
    </div>
  )
}
