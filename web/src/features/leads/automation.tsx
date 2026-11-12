import type { Schemas } from '@/api/client'
import { Note } from '@/components/ui/Note'
import { t as tt, useT } from '@/i18n'
import { ma } from './messages'

/*
 * "Submitted a lead ad form" (M66, G-41): a flow starts when someone sends
 * one of the workspace's Facebook or Instagram lead forms, any form or
 * one. internal/leads queues the event when the lead is recorded, or, with
 * double opt-in, once the lead confirms. The step panel
 * (features/automations/panels.tsx) shows these settings, and the canvas
 * names them with the text functions below, as
 * internal/automations/leads.go does.
 */

type Trigger = Schemas['AutomationTrigger']
type LeadFormOption = Schemas['AutomationLeadFormOption']

export const LEAD_TRIGGER = 'lead_ad_submitted' as const satisfies Trigger['kind']

export function isLeadTrigger(kind: string | undefined): boolean {
  return kind === LEAD_TRIGGER
}

/** The palette's and the select's name. */
export function leadTriggerName(): string {
  return tt(ma.kind)
}

/** The palette's line under it. */
export function leadTriggerLine(): string {
  return tt(ma.kindLine)
}

/** The palette's heading over it. */
export function leadGroupName(): string {
  return tt(ma.group)
}

/** "Spring webinar signup · Acme Inc", as the server names a lead form. */
export function leadFormLabel(f: Pick<LeadFormOption, 'name' | 'page'>): string {
  return f.page ? tt(ma.formOption, { name: f.name, page: f.page }) : f.name
}

/** What the trigger listens to: "any lead form" or the form and its Page. */
export function leadTriggerRef(trig: Trigger, forms?: ReadonlyMap<string, string>): string {
  if (!trig.leadFormId) return tt(ma.refAny)
  return forms?.get(trig.leadFormId) ?? tt(ma.refDeleted)
}

/** The lead form select alone: a second or third trigger's field (AT-11). */
export function LeadFormSelect({ t: trig, set, disabled, forms }: { t: Trigger; set: (patch: Partial<Trigger>) => void; disabled: boolean; forms?: LeadFormOption[] }) {
  const t = useT()
  const value = trig.leadFormId ?? ''
  const known = !value || (forms ?? []).some((f) => f.id === value)
  return (
    <select className="in" aria-label={t(ma.leadForm)} disabled={disabled} value={value} onChange={(e) => set({ leadFormId: e.target.value || undefined })}>
      <option value="">{t(ma.anyForm)}</option>
      {(forms ?? []).map((f) => (
        <option key={f.id} value={f.id}>
          {leadFormLabel(f)}
        </option>
      ))}
      {known ? null : <option value={value}>{t(ma.deletedForm)}</option>}
    </select>
  )
}

/** The lead form (any when none is picked) and when the flow starts. */
export function LeadTriggerFields({ t: trig, set, disabled, forms }: { t: Trigger; set: (patch: Partial<Trigger>) => void; disabled: boolean; forms?: LeadFormOption[] }) {
  const t = useT()
  const value = trig.leadFormId ?? ''
  const known = !value || (forms ?? []).some((f) => f.id === value)
  return (
    <div className="flex flex-col gap-2.5">
      <label className="lbl flex flex-col gap-1">
        {t(ma.leadForm)}
        <select className="in" disabled={disabled} value={value} onChange={(e) => set({ leadFormId: e.target.value || undefined })}>
          <option value="">{t(ma.anyForm)}</option>
          {(forms ?? []).map((f) => (
            <option key={f.id} value={f.id}>
              {leadFormLabel(f)}
            </option>
          ))}
          {known ? null : <option value={value}>{t(ma.deletedForm)}</option>}
        </select>
      </label>
      <Note tone="info" role="note">
        {t(ma.note)}
      </Note>
    </div>
  )
}
