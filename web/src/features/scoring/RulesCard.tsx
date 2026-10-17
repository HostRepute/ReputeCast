import type { ReactNode } from 'react'
import { Icon } from '@/components/icons/Icon'
import { CardSection } from '@/components/ui/Card'
import { Menu, MenuItem, MenuSeparator } from '@/components/ui/Menu'
import { TableScroll } from '@/components/ui/Table'
import { formatList, T, useT } from '@/i18n'
import { shortDate } from '@/lib/format'
import { m } from './messages'
import { capInline, capPhrase, capText, ruleDetail, ruleWhen, signed, type Options, type Rule } from './ruleText'

const mono = (s: string): ReactNode => <span className="mono">{s}</span>

/** What a rule counts, with the link text, event or field in code style. */
export function RuleWhen({ r, o }: { r: Rule; o?: Options }) {
  switch (r.kind) {
    case 'click':
      if (r.contains) return <T msg={m.whenClickContains} values={{ text: mono(r.contains) }} />
      break
    case 'api_event':
      return <T msg={m.whenApiEvent} values={{ event: mono(r.event ?? '') }} />
    case 'field': {
      const field = mono(r.field ?? '')
      if (r.op === 'is_set') return <T msg={m.whenFieldSet} values={{ field }} />
      const values = formatList(r.values ?? [], r.op === 'is_not' ? 'conjunction' : 'disjunction')
      return <T msg={r.op === 'is_not' ? m.whenFieldNot : m.whenField} values={{ field, values }} />
    }
  }
  return <>{ruleWhen(r, o)}</>
}

/** "+15 → +25" for an edited rule, the points alone otherwise. */
function Points({ r, was }: { r: Rule; was?: Rule }) {
  const t = useT()
  if (!was || was.points === r.points) return <span className="sc-pts">{signed(r.points)}</span>
  return (
    <span className="inline-flex items-center gap-1.5">
      <span className="sc-pts sc-was">{signed(was.points)}</span>
      <span className="sc-arrowr" aria-hidden="true">
        →
      </span>
      <span className="vh">{t(m.moveTo)}</span>
      <span className="sc-pts sc-now">{signed(r.points)}</span>
    </span>
  )
}

function differs(a: Rule | undefined, b: Rule): boolean {
  return !a || JSON.stringify({ ...a, values: a.values ?? [] }) !== JSON.stringify({ ...b, values: b.values ?? [] })
}

/**
 * The rules (ScoringRules): number, what counts, points, cap, a switch and
 * the row menu, or View without scoring:edit. Phones get a list.
 */
export function RulesCard({
  rules,
  saved,
  options,
  canEdit,
  setUpBy,
  setUpAt,
  onToggle,
  onEdit,
  onDuplicate,
  onDelete,
}: {
  rules: Rule[]
  saved: Rule[]
  options: Options
  canEdit: boolean
  setUpBy?: string
  setUpAt?: string
  onToggle: (i: number, on: boolean) => void
  onEdit: (i: number) => void
  onDuplicate: (i: number) => void
  onDelete: (i: number) => void
}) {
  const t = useT()
  const before = new Map(saved.map((r) => [r.id, r]))
  return (
    <section className="card" aria-labelledby="sc-rules">
      <CardSection id="sc-rules" title={t(m.rulesTitle, { count: rules.length })} caption={t(m.rulesLead)} />
      <div className="sc-no-phone">
        <TableScroll label={t(m.rulesTableLabel)}>
          <table className="tbl sc-rules">
            <thead>
              <tr>
                <th scope="col">{t(m.colNumber)}</th>
                <th scope="col">{t(m.colWhen)}</th>
                <th scope="col">{t(m.colPoints)}</th>
                <th scope="col">{t(m.colCap)}</th>
                <th scope="col" className="c">
                  {t(m.colOn)}
                </th>
                <th scope="col">
                  <span className="vh">{t(m.colActions)}</span>
                </th>
              </tr>
            </thead>
            <tbody>
              {rules.length === 0 ? (
                <tr>
                  <td colSpan={6} className="k">
                    {t(m.noRulesRow)}
                  </td>
                </tr>
              ) : null}
              {rules.map((r, i) => {
                const was = before.get(r.id)
                const n = i + 1
                return (
                  <tr key={r.id} className={differs(was, r) ? 'sel' : undefined}>
                    <td className="nw">
                      <span className="k num">{n}</span>
                    </td>
                    <td>
                      <div className="flex min-w-0 flex-col gap-0.5 whitespace-normal">
                        <span className="font-semibold">
                          <RuleWhen r={r} o={options} />
                        </span>
                        <span className="k">{ruleDetail(r)}</span>
                      </div>
                    </td>
                    <td className="nw">
                      <Points r={r} was={was} />
                    </td>
                    <td className="nw">{capText(r)}</td>
                    <td className="c">
                      <input type="checkbox" className="tg" role="switch" checked={r.enabled} disabled={!canEdit} aria-label={t(m.ruleOn, { n })} onChange={(e) => onToggle(i, e.target.checked)} />
                    </td>
                    <td className="r">
                      <RowActions n={n} canEdit={canEdit} onEdit={() => onEdit(i)} onDuplicate={() => onDuplicate(i)} onDelete={() => onDelete(i)} />
                    </td>
                  </tr>
                )
              })}
            </tbody>
          </table>
        </TableScroll>
      </div>
      <div className="sc-only-phone">
        {rules.map((r, i) => {
          const n = i + 1
          const was = before.get(r.id)
          return (
            <div key={r.id} className={differs(was, r) ? 'sc-prow sel' : 'sc-prow'}>
              <Points r={r} was={was} />
              <button type="button" className="sc-prow-main" onClick={() => onEdit(i)} aria-label={t(canEdit ? m.editRuleN : m.viewRuleN, { n })}>
                <span className="font-semibold">
                  <RuleWhen r={r} o={options} />
                </span>
                <span className="k">{t(m.capWith, { detail: ruleDetail(r), cap: capInline(r) })}</span>
              </button>
              <input type="checkbox" className="tg" role="switch" checked={r.enabled} disabled={!canEdit} aria-label={t(m.ruleOn, { n })} onChange={(e) => onToggle(i, e.target.checked)} />
            </div>
          )
        })}
      </div>
      <div className="sc-foot">
        <span className="k" style={{ flex: '1 1 420px', minWidth: 0 }}>
          {t(m.rulesFoot)}
        </span>
        {setUpBy && setUpAt ? <span className="k">{t(m.setUpBy, { name: setUpBy, day: shortDate(setUpAt) })}</span> : null}
      </div>
    </section>
  )
}

function RowActions({ n, canEdit, onEdit, onDuplicate, onDelete }: { n: number; canEdit: boolean; onEdit: () => void; onDuplicate: () => void; onDelete: () => void }) {
  const t = useT()
  if (!canEdit) {
    return (
      <button type="button" className="btn sm" onClick={onEdit} aria-label={t(m.viewRuleN, { n })}>
        {t(m.view)}
      </button>
    )
  }
  return (
    <Menu
      label={t(m.ruleN, { n })}
      trigger={
        <button type="button" className="ibtn" style={{ width: 30, height: 30 }} aria-label={t(m.ruleMenu, { n })}>
          <svg width="15" height="15" viewBox="0 0 16 16" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" aria-hidden="true">
            <circle cx="3.5" cy="8" r=".9" />
            <circle cx="8" cy="8" r=".9" />
            <circle cx="12.5" cy="8" r=".9" />
          </svg>
        </button>
      }
    >
      <MenuItem onSelect={onEdit}>
        <Icon name="edit" size={14} />
        {t(m.edit)}
      </MenuItem>
      <MenuItem onSelect={onDuplicate}>
        <Icon name="duplicate" size={14} />
        {t(m.duplicate)}
      </MenuItem>
      <MenuSeparator />
      <MenuItem className="text-bad-fg" onSelect={onDelete}>
        <Icon name="trash" size={14} />
        {t(m.delete)}
      </MenuItem>
    </Menu>
  )
}

/** The suggested rules on the empty state: points, what counts, the cap. */
export function SuggestedRules({ rules, options, bands, weeks }: { rules: Rule[]; options: Options; bands: { warmFrom: number; hotFrom: number }; weeks: number }) {
  const t = useT()
  return (
    <div className="sc-suggest">
      <div className="flex flex-wrap items-center justify-between gap-2 pt-1.5 pb-1">
        <span className="text-[13px] font-semibold">{t(m.suggestedTitle, { count: rules.length })}</span>
        <span className="k">{t(m.suggestedLead, { warm: bands.warmFrom, hot: bands.hotFrom, weeks })}</span>
      </div>
      {rules.map((r) => (
        <div key={r.id} className="sc-srow">
          <span className="sc-pts">{signed(r.points)}</span>
          <span className="min-w-0 flex-1 text-[13px]">
            <RuleWhen r={r} o={options} />
          </span>
          <span className="k whitespace-nowrap">{capPhrase(r)}</span>
        </div>
      ))}
    </div>
  )
}
