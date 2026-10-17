import { useEffect, useMemo, useState } from 'react'
import { $api, type Schemas } from '@/api/client'
import { useWorkspace } from '@/lib/session'
import { t } from '@/i18n'
import { m } from './messages'
import { ruleProblems } from './ruleText'

export type Config = Schemas['ScoringConfig']
export type Impact = Schemas['ScoringImpact']
export type ScoringPageData = Schemas['ScoringPage']

/** Whether two configurations count and band the same. */
export function sameConfig(a: Config, b: Config): boolean {
  return JSON.stringify(normalize(a)) === JSON.stringify(normalize(b))
}

function normalize(c: Config): Config {
  return {
    rules: c.rules.map((r) => ({
      ...r,
      values: r.values ?? [],
      contains: r.contains || undefined,
      listId: r.listId || undefined,
      formId: r.formId || undefined,
      event: r.event || undefined,
      field: r.field || undefined,
      op: r.kind === 'field' ? r.op : undefined,
      cap: { kind: r.cap.kind, points: r.cap.kind.startsWith('points_') ? r.cap.points : undefined },
    })),
    decay: { ...c.decay },
    bands: { ...c.bands },
  }
}

/** What an edit changes: rule numbers (1-based) added, edited or deleted, decay, bands. */
export interface ConfigChanges {
  rules: number[]
  deleted: number
  decay: boolean
  bands: boolean
  onlyBands: boolean
  setup: boolean
}

export function changesOf(saved: Config, draft: Config, configured: boolean): ConfigChanges {
  const byID = new Map(saved.rules.map((r, i) => [r.id, { r, i }]))
  const rules: number[] = []
  draft.rules.forEach((r, i) => {
    const o = byID.get(r.id)
    if (!o || JSON.stringify(normalize({ ...saved, rules: [o.r] }).rules[0]) !== JSON.stringify(normalize({ ...draft, rules: [r] }).rules[0])) {
      rules.push(i + 1)
    }
  })
  const kept = new Set(draft.rules.map((r) => r.id))
  const deleted = saved.rules.filter((r) => !kept.has(r.id)).length
  const decay = JSON.stringify(saved.decay) !== JSON.stringify(draft.decay)
  const bands = JSON.stringify(saved.bands) !== JSON.stringify(draft.bands)
  return { rules, deleted, decay, bands, onlyBands: bands && !decay && rules.length === 0 && deleted === 0, setup: !configured }
}

/** What's wrong with the thresholds and decay, in words. */
export function configProblems(c: Config): { bands?: string; decay?: string } {
  const out: { bands?: string; decay?: string } = {}
  const b = c.bands
  if (!Number.isInteger(b.warmFrom) || b.warmFrom < 1 || b.warmFrom > 99) out.bands = t(m.errWarmFrom)
  else if (!Number.isInteger(b.hotFrom) || b.hotFrom <= b.warmFrom || b.hotFrom > 100) out.bands = t(m.errHotFrom, { warm: b.warmFrom })
  const d = c.decay
  if (d.enabled && (!Number.isInteger(d.afterWeeks) || d.afterWeeks < 1 || d.afterWeeks > 52 || !Number.isInteger(d.percent) || d.percent < 1 || d.percent > 100)) out.decay = t(m.errDecay)
  return out
}

/** Valid enough to ask the server for a preview. */
function previewable(c: Config): boolean {
  return (
    c.bands.warmFrom >= 1 &&
    c.bands.warmFrom <= 99 &&
    c.bands.hotFrom > c.bands.warmFrom &&
    c.bands.hotFrom <= 100 &&
    c.decay.afterWeeks >= 1 &&
    c.decay.afterWeeks <= 52 &&
    c.decay.percent >= 1 &&
    c.decay.percent <= 100 &&
    c.rules.every((r) => Object.keys(ruleProblems(r)).length === 0)
  )
}

/**
 * The impact of an edited set-up, asked for a moment after the last edit
 * (the preview replays only the contacts the edit can change). While the
 * next answer loads, the last one stays, so the chart doesn't flicker.
 */
export function useImpact(config: Config | null, enabled: boolean): { impact?: Impact; loading: boolean; error?: unknown } {
  const { slug } = useWorkspace()
  const preview = $api.useMutation('post', '/api/v1/workspaces/{workspace}/scoring/preview')
  const key = useMemo(() => (config ? JSON.stringify(config) : ''), [config])
  const active = enabled && config !== null && previewable(config)
  const [result, setResult] = useState<{ key: string; impact?: Impact; error?: unknown }>()
  useEffect(() => {
    if (!active || !config) return
    let live = true
    const timer = window.setTimeout(() => {
      preview
        .mutateAsync({ params: { path: { workspace: slug } }, body: { config } })
        .then((r) => live && setResult({ key, impact: r }))
        .catch((e: unknown) => live && setResult({ key, error: e }))
    }, 350)
    return () => {
      live = false
      window.clearTimeout(timer)
    }
    // The key stands for the config.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [key, active, slug])
  if (!active) return { loading: false }
  const fresh = result?.key === key
  return { impact: result?.impact, loading: !fresh, error: fresh ? result?.error : undefined }
}

/** A new rule's id (kept across versions once saved). */
export function newRuleID(): string {
  return crypto.randomUUID()
}
