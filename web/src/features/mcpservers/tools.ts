import type { MCPTool } from './api'

/** A schema or annotations as JSON for reading: indented, keys as the server sent them. */
export function pretty(v: unknown): string {
  if (v === undefined || v === null) return '{}'
  try {
    return JSON.stringify(v, null, 2)
  } catch {
    return String(v)
  }
}

/** Annotations as the board writes them: "readOnlyHint true, destructiveHint false"; empty: none. */
export function hints(v: unknown): string {
  if (!v || typeof v !== 'object' || Array.isArray(v)) return v === undefined || v === null ? '' : pretty(v)
  return Object.entries(v as Record<string, unknown>)
    .map(([k, x]) => `${k} ${typeof x === 'string' ? x : JSON.stringify(x)}`)
    .join(', ')
}

/** The first line of a description, for the tools table. */
export function firstLine(text: string, max = 140): string {
  const line = text.trim().split('\n')[0] ?? ''
  return line.length > max ? `${line.slice(0, max - 1).trimEnd()}…` : line
}

/** The version a manager reviews: the hash's first 12 characters. */
export function shortHash(hash: string): string {
  return hash.slice(0, 12)
}

export type ChangedPart = { part: 'description' | 'schema' | 'annotations'; before: string; after: string }

/**
 * What changed since the approval, part by part (the board's Before and
 * After). Undefined when the approved version wasn't kept.
 */
export function changedParts(tool: Pick<MCPTool, 'approved' | 'description' | 'inputSchema' | 'annotations'>): ChangedPart[] | undefined {
  const a = tool.approved
  if (!a) return undefined
  const parts: ChangedPart[] = []
  if (a.description !== tool.description) parts.push({ part: 'description', before: a.description, after: tool.description })
  const [sb, sa] = [pretty(a.inputSchema), pretty(tool.inputSchema)]
  if (canonical(a.inputSchema) !== canonical(tool.inputSchema)) parts.push({ part: 'schema', before: sb, after: sa })
  if (canonical(a.annotations) !== canonical(tool.annotations)) parts.push({ part: 'annotations', before: hints(a.annotations) || '{}', after: hints(tool.annotations) || '{}' })
  return parts
}

/** JSON with sorted keys, so the same schema in another key order isn't a change. */
function canonical(v: unknown): string {
  const sort = (x: unknown): unknown => {
    if (Array.isArray(x)) return x.map(sort)
    if (x && typeof x === 'object') return Object.fromEntries(Object.keys(x as object).sort().map((k) => [k, sort((x as Record<string, unknown>)[k])]))
    return x
  }
  return JSON.stringify(sort(v ?? {}))
}

/** A tool that changed after a manager allowed it: off until reviewed again (a blocked one stays blocked). */
export function isChanged(tool: Pick<MCPTool, 'changed' | 'decision' | 'removed'>): boolean {
  return tool.changed && !tool.removed && tool.decision !== 'blocked'
}
