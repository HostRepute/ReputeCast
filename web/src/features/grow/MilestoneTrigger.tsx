import { Link } from '@tanstack/react-router'
import type { Schemas } from '@/api/client'
import { useT } from '@/i18n'
import { useWorkspace } from '@/lib/session'
import { m } from './trigger.messages'

type Trigger = Schemas['AutomationTrigger']
type Milestone = Schemas['AutomationMilestoneOption']

/**
 * The "Reached a milestone" trigger's settings (Grow): the program, and any
 * of its milestones or one. A contact enters once per milestone, so there's
 * no re-entry choice.
 */
export function MilestoneTriggerFields({ t: tr, set, disabled, milestones }: { t: Trigger; set: (p: Partial<Trigger>) => void; disabled: boolean; milestones: readonly Milestone[] }) {
  const t = useT()
  const { slug } = useWorkspace()
  const programs = [...new Map(milestones.map((x) => [x.programId, x.listName])).entries()]
  // With one program, "any program" is that program.
  const programId = tr.programId || (programs.length === 1 ? (programs[0]?.[0] ?? '') : '')
  const shown = milestones.filter((x) => !programId || x.programId === programId)
  if (milestones.length === 0) {
    return (
      <div className="flex flex-col gap-2">
        <span className="k">
          {t(m.noMilestones)}{' '}
          <Link to="/w/$workspace/grow" params={{ workspace: slug }}>
            {t(m.setUp)}
          </Link>
        </span>
        {disabled ? <span className="k">{t(m.needs)}</span> : null}
      </div>
    )
  }
  return (
    <div className="flex flex-col gap-2">
      <select className="in" aria-label={t(m.program)} disabled={disabled} value={programId} onChange={(e) => set({ programId: e.target.value || undefined, milestoneId: undefined })}>
        {programs.length > 1 ? <option value="">{t(m.anyProgram)}</option> : null}
        {programs.map(([id, name]) => (
          <option key={id} value={id}>
            {name}
          </option>
        ))}
      </select>
      <select className="in" aria-label={t(m.milestone)} disabled={disabled} value={tr.milestoneId ?? ''} onChange={(e) => set({ milestoneId: e.target.value || undefined })}>
        <option value="">{t(m.anyMilestoneOption)}</option>
        {shown.map((x) => (
          <option key={x.id} value={x.id}>
            {programs.length > 1 && !programId ? t(m.programMilestone, { list: x.listName, milestone: x.name }) : x.name}
          </option>
        ))}
      </select>
      <span className="k">{t(m.once)}</span>
      <span className="k">{t(m.explain)}</span>
      {disabled ? <span className="k">{t(m.needs)}</span> : null}
    </div>
  )
}
