import { useReactFlow, useStore, type ReactFlowState } from '@xyflow/react'
import { useRef } from 'react'
import type { StepNodeType } from './StepNode'

/** Zoom in, zoom out, fit view and lock (AutomationFlow › Canvas controls). */
export function CanvasControls({ locked, onLock }: { locked: boolean; onLock: (v: boolean) => void }) {
  const rf = useReactFlow()
  return (
    <div className="fe-ctl" role="group" aria-label="Canvas controls">
      <button type="button" className="btn ic" aria-label="Zoom in" onClick={() => void rf.zoomIn({ duration: 150 })}>
        +
      </button>
      <button type="button" className="btn ic" aria-label="Zoom out" onClick={() => void rf.zoomOut({ duration: 150 })}>
        −
      </button>
      <button type="button" className="btn ic" aria-label="Fit view" onClick={() => void rf.fitView({ padding: 0.08, duration: 200 })}>
        <svg width="14" height="14" viewBox="0 0 16 16" fill="none" stroke="currentColor" strokeWidth="1.6" aria-hidden="true">
          <path d="M2 6V2h4M14 6V2h-4M2 10v4h4M14 10v4h-4" />
        </svg>
      </button>
      <button type="button" className="btn ic" aria-label={locked ? 'Unlock canvas' : 'Lock canvas'} aria-pressed={locked} onClick={() => onLock(!locked)}>
        <svg width="14" height="14" viewBox="0 0 16 16" fill="none" stroke="currentColor" strokeWidth="1.6" aria-hidden="true">
          <rect x="3" y="7" width="10" height="7" rx="1.5" />
          <path d={locked ? 'M5 7V5a3 3 0 0 1 6 0v2' : 'M5 7V5a3 3 0 0 1 5.6-1.5'} />
        </svg>
      </button>
    </div>
  )
}

const selector = (s: ReactFlowState) => ({ transform: s.transform, width: s.width, height: s.height })

/**
 * The minimap: the steps fill the box as grey blocks (the selected one blue;
 * across and down are scaled separately, like the board's), and the visible
 * area is outlined, clipped to the box. Click or drag to move the view there.
 */
export function FlowMiniMap({ nodes, selected }: { nodes: StepNodeType[]; selected?: string }) {
  const { transform, width, height } = useStore(selector, (a, b) => a.transform === b.transform && a.width === b.width && a.height === b.height)
  const rf = useReactFlow()
  const box = useRef<HTMLDivElement>(null)
  const [tx, ty, zoom] = transform
  const view = { x: -tx / zoom, y: -ty / zoom, w: width / zoom, h: height / zoom }
  let minX = nodes.length ? Infinity : view.x
  let minY = nodes.length ? Infinity : view.y
  let maxX = nodes.length ? -Infinity : view.x + view.w
  let maxY = nodes.length ? -Infinity : view.y + view.h
  for (const n of nodes) {
    minX = Math.min(minX, n.position.x)
    minY = Math.min(minY, n.position.y)
    maxX = Math.max(maxX, n.position.x + (n.measured?.width ?? 224))
    maxY = Math.max(maxY, n.position.y + (n.measured?.height ?? 60))
  }
  const innerW = 142
  const innerH = 92
  const pad = 2
  const kx = (innerW - 2 * pad) / Math.max(1, maxX - minX)
  const ky = (innerH - 2 * pad) / Math.max(1, maxY - minY)
  const sx = (x: number) => pad + (x - minX) * kx
  const sy = (y: number) => pad + (y - minY) * ky
  const clip = (v: number, max: number) => Math.min(max, Math.max(0, v))
  const vx = clip(sx(view.x), innerW)
  const vy = clip(sy(view.y), innerH)
  const vw = clip(sx(view.x + view.w), innerW) - vx
  const vh = clip(sy(view.y + view.h), innerH) - vy
  const moveTo = (clientX: number, clientY: number) => {
    const r = box.current?.getBoundingClientRect()
    if (!r) return
    const fx = (clientX - r.left - pad) / kx + minX
    const fy = (clientY - r.top - pad) / ky + minY
    void rf.setCenter(fx, fy, { zoom, duration: 0 })
  }
  return (
    <div
      className="fe-minimap"
      role="img"
      aria-label="Minimap"
      onPointerDown={(e) => {
        e.currentTarget.setPointerCapture(e.pointerId)
        moveTo(e.clientX, e.clientY)
      }}
      onPointerMove={(e) => {
        if (e.buttons === 1) moveTo(e.clientX, e.clientY)
      }}
    >
      <div ref={box} className="relative h-full w-full">
        {nodes.map((n) => (
          <div
            key={n.id}
            className="absolute rounded-[2px]"
            style={{
              left: sx(n.position.x),
              top: sy(n.position.y),
              width: Math.max(2, (n.measured?.width ?? 224) * kx),
              height: Math.max(2, (n.measured?.height ?? 60) * ky),
              background: n.id === selected ? '#1D4ED8' : 'var(--line-hover)',
            }}
          />
        ))}
        {vw > 0 && vh > 0 ? <div className="absolute box-border rounded-[3px]" style={{ left: vx, top: vy, width: vw, height: vh, border: '1.5px solid #1D4ED8' }} /> : null}
      </div>
    </div>
  )
}
