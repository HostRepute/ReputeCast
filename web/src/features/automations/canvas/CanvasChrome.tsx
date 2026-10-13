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
 * The minimap: steps as grey blocks (the selected one blue) and the visible
 * area outlined. Click or drag to move the view there.
 */
export function FlowMiniMap({ nodes, selected }: { nodes: StepNodeType[]; selected?: string }) {
  const { transform, width, height } = useStore(selector, (a, b) => a.transform === b.transform && a.width === b.width && a.height === b.height)
  const rf = useReactFlow()
  const box = useRef<HTMLDivElement>(null)
  const [tx, ty, zoom] = transform
  const view = { x: -tx / zoom, y: -ty / zoom, w: width / zoom, h: height / zoom }
  let minX = view.x
  let minY = view.y
  let maxX = view.x + view.w
  let maxY = view.y + view.h
  for (const n of nodes) {
    const w = n.measured?.width ?? 224
    const h = n.measured?.height ?? 60
    minX = Math.min(minX, n.position.x)
    minY = Math.min(minY, n.position.y)
    maxX = Math.max(maxX, n.position.x + w)
    maxY = Math.max(maxY, n.position.y + h)
  }
  const innerW = 142
  const innerH = 92
  const scale = Math.min(innerW / Math.max(1, maxX - minX), innerH / Math.max(1, maxY - minY))
  const offX = (innerW - (maxX - minX) * scale) / 2
  const offY = (innerH - (maxY - minY) * scale) / 2
  const sx = (x: number) => offX + (x - minX) * scale
  const sy = (y: number) => offY + (y - minY) * scale
  const moveTo = (clientX: number, clientY: number) => {
    const r = box.current?.getBoundingClientRect()
    if (!r) return
    const fx = (clientX - r.left - offX) / scale + minX
    const fy = (clientY - r.top - offY) / scale + minY
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
              width: Math.max(2, (n.measured?.width ?? 224) * scale),
              height: Math.max(2, (n.measured?.height ?? 60) * scale),
              background: n.id === selected ? '#1D4ED8' : 'var(--line-hover)',
            }}
          />
        ))}
        <div
          className="absolute rounded-[3px] box-border"
          style={{ left: sx(view.x), top: sy(view.y), width: view.w * scale, height: view.h * scale, border: '1.5px solid #1D4ED8' }}
        />
      </div>
    </div>
  )
}
