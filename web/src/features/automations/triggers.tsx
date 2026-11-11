import type { ReactNode } from 'react'
import type { Schemas } from '@/api/client'
import { Button } from '@/components/ui/Button'
import { Note } from '@/components/ui/Note'
import { T, useT } from '@/i18n'
import { m } from './complete.messages'
import { mp } from './panels.messages'
import { isLeadTrigger, LeadFormSelect } from '@/features/leads/automation'
import { ALSO_KINDS, triggerKindLabel, type TriggerConfig } from './graph'

type Options = Schemas['AutomationOptions']

function Check({ checked, onChange, disabled, children }: { checked: boolean; onChange: (v: boolean) => void; disabled?: boolean; children: ReactNode }) {
  return (
    <label className="fe-chk">
      <input type="checkbox" checked={checked} disabled={disabled} onChange={(e) => onChange(e.target.checked)} />
      {children}
    </label>
  )
}

/**
 * "Enters a segment", "Leaves a segment" and "Clicked a link" (review
 * 2026-11 AT-10): the segment, or the campaign or email and what the link
 * contains.
 */
export function NewKindFields({ t, set, options, disabled }: { t: TriggerConfig; set: (p: Partial<TriggerConfig>) => void; options?: Options; disabled: boolean }) {
  const tr = useT()
  if (t.kind === 'segment_entered' || t.kind === 'segment_left') {
    return (
      <>
        <select className="in" aria-label={tr(m.segment)} disabled={disabled} value={t.segmentId ?? ''} onChange={(e) => set({ segmentId: e.target.value })}>
          <option value="">{tr(m.pickSegment)}</option>
          {options?.segments.map((s) => (
            <option key={s.id} value={s.id}>
              {s.name}
            </option>
          ))}
        </select>
        <span className="k">{tr(m.segmentHint)}</span>
        {(() => {
          // A slow segment is checked less often (review 2026-11 AT-15).
          const seg = options?.segments.find((s) => s.id === t.segmentId)
          return seg && seg.checkMinutes > 10 ? (
            <Note tone="warn">{tr(m.segmentSlow, { seconds: seg.evalSeconds ?? 0, minutes: seg.checkMinutes })}</Note>
          ) : null
        })()}
      </>
    )
  }
  if (t.kind === 'link_clicked') {
    return (
      <>
        <select className="in" aria-label={tr(m.clickedIn)} disabled={disabled} value={t.source ?? ''} onChange={(e) => set({ source: e.target.value || undefined })}>
          <option value="">{tr(m.anyCampaignOrEmail)}</option>
          {options?.campaigns.length ? (
            <optgroup label={tr(m.campaignsGroup)}>
              {options.campaigns.map((c) => (
                <option key={c.id} value={`campaign:${c.id}`}>
                  {c.name}
                </option>
              ))}
            </optgroup>
          ) : null}
          {options?.emails.length ? (
            <optgroup label={tr(m.emailsGroup)}>
              {options.emails.map((e) => (
                <option key={e.id} value={`email:${e.id}`}>
                  {e.name}
                </option>
              ))}
            </optgroup>
          ) : null}
        </select>
        <input
          className="in"
          aria-label={tr(m.linkContains)}
          placeholder={tr(m.linkContains)}
          maxLength={200}
          disabled={disabled}
          value={t.urlContains ?? ''}
          onChange={(e) => set({ urlContains: e.target.value || undefined })}
        />
        <span className="k">{tr(m.linkHint)}</span>
      </>
    )
  }
  return null
}

/** The fields of a second or third trigger (the event kinds only). */
function AlsoFields({ t, set, options, disabled }: { t: TriggerConfig; set: (p: Partial<TriggerConfig>) => void; options?: Options; disabled: boolean }) {
  const tr = useT()
  switch (t.kind) {
    case 'joined_list':
      return (
        <select className="in" aria-label={tr(mp.list)} disabled={disabled} value={t.listId ?? ''} onChange={(e) => set({ listId: e.target.value })}>
          <option value="">{tr(mp.pickList)}</option>
          {options?.lists.map((l) => (
            <option key={l.id} value={l.id}>
              {l.name}
            </option>
          ))}
        </select>
      )
    case 'form_submitted':
      return (
        <select className="in" aria-label={tr(mp.form)} disabled={disabled} value={t.formId ?? ''} onChange={(e) => set({ formId: e.target.value })}>
          <option value="">{tr(mp.pickForm)}</option>
          {options?.forms.map((f) => (
            <option key={f.id} value={f.id}>
              {f.name} ({f.slug})
            </option>
          ))}
        </select>
      )
    case 'tag_added':
      return <input className="in" aria-label={tr(mp.tag)} list="fe-tags" placeholder={tr(mp.tagPlaceholder)} disabled={disabled} value={t.tag ?? ''} onChange={(e) => set({ tag: e.target.value })} />
    case 'api_event':
      return <input className="in mono" aria-label={tr(mp.eventName)} list="fe-events" placeholder={'workspace.created'} disabled={disabled} value={t.event ?? ''} onChange={(e) => set({ event: e.target.value })} />
    case 'lead_ad_submitted':
      return <LeadFormSelect t={t} set={set} disabled={disabled} forms={options?.leadForms} />
  }
  return <NewKindFields t={t} set={set} options={options} disabled={disabled} />
}

/**
 * The second and third triggers of a flow, any of which enrols (review
 * 2026-11 AT-11, the trigger stack).
 */
export function AlsoTriggers({ t, set, options, disabled }: { t: TriggerConfig; set: (p: Partial<TriggerConfig>) => void; options?: Options; disabled: boolean }) {
  const tr = useT()
  const also = t.also ?? []
  const update = (i: number, patch: Partial<TriggerConfig>) => set({ also: also.map((a, j) => (j === i ? { ...a, ...patch } : a)) })
  const remove = (i: number) => {
    const next = also.filter((_, j) => j !== i)
    set({ also: next.length ? next : undefined })
  }
  return (
    <div className="flex flex-col gap-2">
      {also.map((a, i) => (
        <div key={i} className="fe-also" role="group" aria-label={tr(m.triggerN, { n: i + 2, kind: triggerKindLabel(a.kind) })}>
          <span className="fe-or" aria-hidden="true">
            {tr(m.or)}
          </span>
          <select
            className="in"
            aria-label={tr(m.triggerN, { n: i + 2, kind: triggerKindLabel(a.kind) })}
            disabled={disabled}
            value={a.kind}
            onChange={(e) =>
              update(i, { kind: e.target.value as TriggerConfig['kind'], listId: undefined, formId: undefined, tag: undefined, event: undefined, segmentId: undefined, source: undefined, urlContains: undefined, leadFormId: undefined })
            }
          >
            {/* "Submitted a lead ad form" once there is a lead form (M66). */}
            {ALSO_KINDS.filter((k) => !isLeadTrigger(k) || (options?.leadForms.length ?? 0) > 0 || k === a.kind).map((k) => (
              <option key={k} value={k}>
                {triggerKindLabel(k)}
              </option>
            ))}
          </select>
          <AlsoFields t={a} set={(p) => update(i, p)} options={options} disabled={disabled} />
          {disabled ? null : (
            <button type="button" className="self-start text-[13px] font-semibold text-accent underline underline-offset-2" onClick={() => remove(i)}>
              {tr(m.removeTrigger)}
            </button>
          )}
        </div>
      ))}
      {also.length < 2 && !disabled ? (
        <Button size="sm" className="self-start" onClick={() => set({ also: [...also, { kind: 'tag_added' }] })}>
          {tr(m.addTrigger)}
        </Button>
      ) : null}
      {also.length > 0 ? <span className="k">{tr(m.alsoNote)}</span> : null}
    </div>
  )
}

/** "Once every [30] days" (review 2026-11 AT-6). */
export function ReentryDays({ value, onChange, disabled }: { value?: number; onChange: (n: number) => void; disabled: boolean }) {
  const tr = useT()
  return (
    <span className="flex flex-wrap items-center gap-1.5 text-[13px]">
      <T
        msg={m.onceEvery}
        values={{
          n: () => (
            <input
              type="number"
              inputMode="numeric"
              min={1}
              max={3650}
              className="in w-20"
              aria-label={tr(m.daysBetween)}
              disabled={disabled}
              value={value ?? 30}
              onChange={(e) => onChange(Math.max(1, Math.min(3650, Number(e.target.value) || 1)))}
            />
          ),
        }}
      />
    </span>
  )
}

export { Check as TriggerCheck }
