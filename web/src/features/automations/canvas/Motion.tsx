import { EdgeLabelRenderer, useNodes, useNodesInitialized, useStoreApi } from '@xyflow/react'
import { useEffect, useLayoutEffect, useMemo, useRef, useState } from 'react'
import { cn } from '@/lib/cn'
import { FLASH, HIT, PULSE_SPEED, timeline, type Box, type Timeline } from '../graph'
import { useFlowView } from '../view'
import type { StepNodeType } from './StepNode'

/** First wave after the entrance has settled, later waves after a short pause. */
const FIRST_DELAY_MS = 3200
const RESTART_DELAY_MS = 400

/**
 * The flow board's "pulse through steps" motion (see timeline in graph.ts):
 * it loops every period, pauses with "Animate flow" and is off with reduced
 * motion.
 */
export function Motion({ playing, reduced }: { playing: boolean; reduced: boolean }) {
  const nodes = useNodes<StepNodeType>()
  const ready = useNodesInitialized()
  const v = useFlowView()
  const store = useStoreApi()
  const [tl, setTl] = useState<Timeline | null>(null)
  const layer = useRef<HTMLDivElement>(null)
  const anims = useRef<Animation[]>([])
  const first = useRef(true)
  const playingRef = useRef(playing)

  const key = useMemo(
    () =>
      nodes.map((n) => `${n.id}:${Math.round(n.position.x)},${Math.round(n.position.y)},${n.measured?.width ?? 0}x${n.measured?.height ?? 0}`).join('|') +
      '#' +
      v.graph.edges.map((e) => `${e.source}>${e.target}:${e.data.path ?? ''}`).join('|'),
    [nodes, v.graph.edges],
  )

  useEffect(() => {
    if (reduced || !ready) return
    const h = window.setTimeout(
      () => {
        const boxes = new Map<string, Box>()
        for (const n of nodes) boxes.set(n.id, { x: n.position.x, y: n.position.y, w: n.measured?.width ?? 224, h: n.measured?.height ?? 60 })
        const shares = new Map<string, Map<string, number>>()
        for (const [id, st] of v.stats) if (st.paths.length > 1) shares.set(id, new Map(st.paths.map((p) => [p.path, p.count])))
        setTl(timeline(v.graph, boxes, shares))
      },
      first.current ? 0 : 300,
    )
    return () => window.clearTimeout(h)
    // The geometry key covers nodes and edges.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [key, reduced, ready, v.stats])

  useLayoutEffect(() => {
    const stop = () => {
      anims.current.forEach((a) => a.cancel())
      anims.current = []
    }
    stop()
    const root = store.getState().domNode
    if (!tl || reduced || !layer.current || !root || tl.period <= 0) return stop
    const period = tl.period
    const at = (t: number) => Math.min(1, Math.max(0, t / period))
    const opts: KeyframeAnimationOptions = {
      duration: period * 1000,
      delay: first.current ? FIRST_DELAY_MS : RESTART_DELAY_MS,
      iterations: Infinity,
      easing: 'linear',
    }
    first.current = false
    const pulses = Array.from(layer.current.children) as HTMLElement[]
    tl.pulses.forEach((p, i) => {
      const el = pulses[i]
      if (!el) return
      const w = p.horizontal ? 18 : 4
      const h = p.horizontal ? 4 : 18
      const dur = p.end - p.start
      const fade = Math.min(4 / PULSE_SPEED, dur / 4)
      const point = (f: number) => `translate(${p.from.x + (p.to.x - p.from.x) * f - w / 2}px, ${p.from.y + (p.to.y - p.from.y) * f - h / 2}px)`
      anims.current.push(
        el.animate(
          [
            { offset: 0, opacity: 0, transform: point(0) },
            { offset: at(p.start), opacity: 0, transform: point(0) },
            { offset: at(p.start + fade), opacity: p.opacity, transform: point(fade / dur) },
            { offset: at(p.end - fade), opacity: p.opacity, transform: point(1 - fade / dur) },
            { offset: at(p.end), opacity: 0, transform: point(1) },
            { offset: 1, opacity: 0, transform: point(1) },
          ],
          opts,
        ),
      )
    })
    const sel = (id: string) => `.react-flow__node[data-id="${CSS.escape(id)}"]`
    for (const hit of tl.hits) {
      const el = root.querySelector(`${sel(hit.node)} .hit`)
      if (!el) continue
      const s = hit.start
      anims.current.push(
        el.animate(
          [
            { offset: 0, opacity: 0 },
            { offset: at(s), opacity: 0 },
            { offset: at(s + HIT.ramp), opacity: hit.opacity },
            { offset: at(s + HIT.ramp + HIT.hold), opacity: hit.opacity },
            { offset: at(s + HIT.ramp + HIT.hold + HIT.fade), opacity: 0 },
            { offset: 1, opacity: 0 },
          ],
          opts,
        ),
      )
    }
    const rest = { transform: 'translateX(-50%) scale(1)', background: '#fff', borderColor: '#6B7280' }
    const lit = { transform: 'translateX(-50%) scale(1.7)', background: '#0F766E', borderColor: '#0F766E' }
    const flash = (selector: string, s: number) => {
      const el = root.querySelector(selector)
      if (!el) return
      anims.current.push(
        el.animate(
          [
            { offset: 0, ...rest },
            { offset: at(s), ...rest },
            { offset: at(s + FLASH.grow), ...lit },
            { offset: at(s + FLASH.grow + FLASH.hold), ...lit },
            { offset: at(s + FLASH.grow + FLASH.hold + FLASH.shrink), ...rest },
            { offset: 1, ...rest },
          ],
          opts,
        ),
      )
    }
    tl.flashIn.forEach((f) => flash(`${sel(f.node)} .fn-hb.i`, f.start))
    tl.flashOut.forEach((f) => flash(`${sel(f.node)} .fn-hb.o`, f.start))
    if (!playingRef.current) anims.current.forEach((a) => a.pause())
    return stop
  }, [tl, reduced, store])

  useEffect(() => {
    playingRef.current = playing
    anims.current.forEach((a) => (playing ? a.play() : a.pause()))
  }, [playing])

  if (!tl || reduced) return null
  return (
    <EdgeLabelRenderer>
      <div ref={layer} className="absolute left-0 top-0" aria-hidden="true">
        {tl.pulses.map((p) => (
          <span key={p.key} className={cn('fe-pu', p.horizontal && 'h')} />
        ))}
      </div>
    </EdgeLabelRenderer>
  )
}
