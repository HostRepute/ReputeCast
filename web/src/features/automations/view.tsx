import { createContext, useContext, useEffect, useSyncExternalStore } from 'react'
import type { Schemas } from '@/api/client'
import type { FlowGraph, Refs } from './graph'

export type StepStats = Schemas['AutomationStepStats']

/**
 * What the canvas nodes and edges need besides their own data: names of the
 * lists and emails they point at, live stats, which steps are new, and the
 * entrance timing. Provided once by the editor.
 */
export interface FlowView {
  /** The automation, for the channel steps' estimates on the canvas (X22). */
  automationId?: string
  graph: FlowGraph
  refs: Refs
  stats: Map<string, StepStats>
  /** Contacts who entered through the trigger in the stats window. */
  entered: number
  hasStats: boolean
  showStats: boolean
  liveIds: Set<string>
  publishedIds: Set<string>
  /** The automation has a live version (new steps are "new in vN"). */
  published: boolean
  live: boolean
  draftVersion: number
  issues: Set<string>
  /** Fallback provider of the Automations route (its sends say "failover"). */
  fallbackId?: string
  entrance?: { nodes: Map<string, number>; edges: Map<string, number>; labels: Map<string, number> }
}

const Ctx = createContext<FlowView | null>(null)

export const FlowViewProvider = Ctx.Provider

export function useFlowView(): FlowView {
  const v = useContext(Ctx)
  if (!v) throw new Error('useFlowView outside the flow editor')
  return v
}

/*
 * Counters on the canvas count up once when the editor opens (0.9 s delay,
 * 1.5 s, ease-out), like the board; with reduced motion they show the final
 * numbers at once.
 */
let progress = 1
const listeners = new Set<() => void>()
let raf = 0
let timer = 0

function emit() {
  listeners.forEach((l) => l())
}

export function startCountUp(reduced: boolean) {
  window.clearTimeout(timer)
  window.cancelAnimationFrame(raf)
  if (reduced) {
    progress = 1
    emit()
    return
  }
  progress = 0
  emit()
  timer = window.setTimeout(() => {
    let start: number | null = null
    const step = (ts: number) => {
      if (start === null) start = ts
      const p = Math.min(1, (ts - start) / 1500)
      progress = 1 - Math.pow(1 - p, 3)
      emit()
      if (p < 1) raf = window.requestAnimationFrame(step)
    }
    raf = window.requestAnimationFrame(step)
  }, 900)
}

export function useCountUp(): number {
  return useSyncExternalStore(
    (l) => {
      listeners.add(l)
      return () => listeners.delete(l)
    },
    () => progress,
    () => 1,
  )
}

export function useCountUpOnMount(reduced: boolean) {
  useEffect(() => {
    startCountUp(reduced)
    return () => {
      window.clearTimeout(timer)
      window.cancelAnimationFrame(raf)
      progress = 1
    }
  }, [reduced])
}
