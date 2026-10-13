import { BaseEdge, EdgeLabelRenderer, type Edge, type EdgeProps } from '@xyflow/react'
import { memo, type CSSProperties } from 'react'
import { cn } from '@/lib/cn'
import { edgeGeometry, edgeKind, nodeById, pathLabel, type FlowGraph } from '../graph'
import { useFlowView } from '../view'

export type StepEdgeType = Edge<{ path?: string }, 'step'>

/** "Yes · 61%" from the branch's stats, "A · 50%" from the split's weights. */
function labelText(g: FlowGraph, source: string, path: string, paths?: { path: string; count: number }[]): string {
  const src = nodeById(g, source)
  if (!src) return pathLabel(path)
  if (src.type === 'split') {
    const v = src.data.variants?.find((x) => x.key === path)
    return v ? `${path} · ${v.weight}%` : path
  }
  const total = paths?.reduce((s, p) => s + p.count, 0) ?? 0
  const mine = paths?.find((p) => p.path === path)?.count ?? 0
  return total > 0 ? `${pathLabel(path)} · ${Math.round((mine / total) * 100)}%` : pathLabel(path)
}

function StepEdgeImpl({ id, source, sourceX, sourceY, targetX, targetY, data, selected }: EdgeProps<StepEdgeType>) {
  const v = useFlowView()
  const edge = v.graph.edges.find((e) => e.id === id)
  const kind = edge ? edgeKind(v.graph, edge) : 'plain'
  const geo = edgeGeometry(sourceX, sourceY, targetX, targetY, kind)
  const delay = v.entrance?.edges.get(id)
  const drawn = delay !== undefined
  const style: CSSProperties & Record<string, string | number> = drawn
    ? { strokeDasharray: geo.length, strokeDashoffset: geo.length, '--fl-d': `${delay}s`, '--fl-t': kind === 'plain' ? '0.45s' : '0.6s' }
    : {}
  const path = data?.path
  const showLabel = path && geo.label && (kind === 'branch' || kind === 'split')
  const labelDelay = v.entrance?.labels.get(id)
  return (
    <>
      <BaseEdge id={id} path={geo.path} className={cn('fe-edge', drawn && 'draw', selected && 'selected')} style={style} interactionWidth={14} />
      {showLabel ? (
        <EdgeLabelRenderer>
          <div
            className={cn('fe-elab', kind === 'branch' && path === 'yes' && 'yes', kind === 'branch' && path === 'no' && 'no')}
            style={{ transform: `translate(${geo.label!.x}px, ${geo.label!.y}px)`, ['--fl-d' as string]: `${labelDelay ?? 0}s` }}
          >
            {labelText(v.graph, source, path, v.showStats ? v.stats.get(source)?.paths : undefined)}
          </div>
        </EdgeLabelRenderer>
      ) : null}
    </>
  )
}

export const StepEdge = memo(StepEdgeImpl)
export const edgeTypes = { step: StepEdge }
