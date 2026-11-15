import { useRef, useState, type ComponentProps } from 'react'
import { $api, errorMessage, type Schemas } from '@/api/client'
import { Button } from '@/components/ui/Button'
import { Field, Select } from '@/components/ui/Field'
import { ErrorPanel, Skeleton } from '@/components/ui/Feedback'
import { Condition } from '@/features/audience/SegmentBuilder'
import { m } from '@/features/audience/segments.messages'
import { useT } from '@/i18n'
import { common } from '@/i18n/common'
import { useWorkspace } from '@/lib/session'

type Definition = Schemas['SegmentDefinition']
type Catalog = Schemas['SegmentCatalog']
type Cond = ComponentProps<typeof Condition>['c']

/** Local selection only: this editor neither saves a segment nor calls a model. */
export function AgentAudience({ value, onChange, disabled }: { value: Definition; onChange: (value: Definition) => void; disabled: boolean }) {
  const t = useT()
  const { slug } = useWorkspace()
  const q = $api.useQuery('get', '/api/v1/workspaces/{workspace}/segments/catalog', { params: { path: { workspace: slug } } })
  if (q.error) return <ErrorPanel title={errorMessage(q.error)} actions={<Button onClick={() => void q.refetch()}>{t(common.retry)}</Button>} />
  if (!q.data) return <Skeleton h={100} />
  return <AudienceFields value={value} onChange={onChange} disabled={disabled} catalog={q.data} />
}

function AudienceFields({ value, onChange, disabled, catalog }: { value: Definition; onChange: (value: Definition) => void; disabled: boolean; catalog: Catalog }) {
  const t = useT()
  const next = useRef(100)
  const blank = (field: Catalog['fields'][number] | undefined): Cond => ({ key: ++next.current, field: field?.key ?? '', op: field?.ops[0]?.key ?? '', value: field?.key === 'signed_in' ? '7d' : field?.options[0]?.value ?? '', to: '', amount: 30, unit: 'days' })
  const [groups, setGroups] = useState(() => value.groups.map((group, gi) => ({ key: gi, match: group.match, conditions: group.conditions.map((c, ci) => ({ key: ci, field: c.field, op: c.op, value: c.value ?? '', to: c.to ?? '', amount: c.amount ?? 30, unit: c.unit === 'weeks' ? 'weeks' as const : 'days' as const })) })))
  const update = (nextGroups: typeof groups) => {
    setGroups(nextGroups)
    onChange({ groups: nextGroups.map((g) => ({ match: g.match, conditions: g.conditions.map((c) => catalog.fields.find((f) => f.key === c.field)?.kind === 'period'
      ? { field: c.field, op: c.op, amount: c.amount, unit: c.unit, value: c.value || undefined }
      : { field: c.field, op: c.op, value: c.value, to: c.op === 'between' ? c.to : undefined }) })) })
  }
  const condition = (gi: number, ci: number, patch: Partial<Cond>) => update(groups.map((g, i) => i === gi ? { ...g, conditions: g.conditions.map((c, j) => j === ci ? { ...c, ...patch } : c) } : g))
  return <fieldset disabled={disabled} className="agent-audience m-0 flex min-w-0 flex-col gap-3 border-0 p-0"><p className="k m-0">{t(m.combine)}</p>
    {groups.map((group, gi) => <fieldset key={group.key} className="m-0 flex min-w-0 flex-col gap-3 rounded-lg border border-line p-3">
      <legend className="px-1 text-sm font-semibold">{t(m.group, { n: gi + 1 })}</legend>
      <div className="flex flex-wrap items-end gap-2"><Field label={t(m.matchRule, { n: gi + 1 })}><Select value={group.match} onChange={(e) => update(groups.map((g, i) => i === gi ? { ...g, match: e.target.value as typeof g.match } : g))}><option value="all">{t(m.matchAll)}</option><option value="any">{t(m.matchAny)}</option></Select></Field><Button className="ml-auto" onClick={() => update(groups.filter((_, i) => i !== gi))}>{t(m.removeGroup)}</Button></div>
      {group.conditions.map((c, ci) => <Condition key={c.key} c={c} catalog={catalog} invalid={false} onChange={(patch) => condition(gi, ci, patch)} onField={(key) => condition(gi, ci, { ...blank(catalog.fields.find((f) => f.key === key)), key: c.key })} onRemove={() => update(groups.map((g, i) => i === gi ? { ...g, conditions: g.conditions.filter((_, j) => j !== ci) } : g))} canRemove={group.conditions.length > 1} />)}
      <div><Button disabled={group.conditions.length >= 10 || !catalog.fields.length} onClick={() => update(groups.map((g, i) => i === gi ? { ...g, conditions: [...g.conditions, blank(catalog.fields[0])] } : g))}>{t(m.addCondition)}</Button></div>
    </fieldset>)}
    <div><Button disabled={groups.length >= 5 || !catalog.fields.length} onClick={() => update([...groups, { key: ++next.current, match: 'all', conditions: [blank(catalog.fields[0])] }])}>{t(m.addGroup)}</Button></div>
  </fieldset>
}
