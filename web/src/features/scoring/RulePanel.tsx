import { Dialog as RDialog } from 'radix-ui'
import { useId, useState, type ReactNode } from 'react'
import { Icon } from '@/components/icons/Icon'
import { Button } from '@/components/ui/Button'
import { Input, Select } from '@/components/ui/Field'
import { T, useT } from '@/i18n'
import { count } from '@/lib/format'
import { m } from './messages'
import { capExplain, capKindLabel, capKinds, kindLabel, kindPick, ruleFor, ruleKinds, ruleProblems, signed, type CapKind, type Options, type Rule } from './ruleText'
import type { Impact } from './useScoringDraft'

export type PanelMode = 'edit' | 'new' | 'view'

/** "+25", "25", "-5" and "−5" are points; anything else isn't. */
function parsePoints(s: string): number {
  const v = s.trim().replace(/^\+/, '').replace(/^[−–]/, '-')
  return /^-?\d+$/.test(v) ? Number(v) : NaN
}

function Lbl({ children, htmlFor }: { children: ReactNode; htmlFor?: string }) {
  return htmlFor ? (
    <label className="sc-lbl" htmlFor={htmlFor}>
      {children}
    </label>
  ) : (
    <span className="sc-lbl">{children}</span>
  )
}

function FieldErr({ id, children }: { id?: string; children: ReactNode }) {
  return (
    <span className="sc-ferr" role="alert">
      <Icon name="alertCircle" size={14} />
      <span id={id}>{children}</span>
    </span>
  )
}

/**
 * The side panel that adds, edits or shows a rule (ScoringStates). Edits
 * apply to the page's draft as they're made, so the chart behind shows the
 * new spread; Cancel puts the rule back.
 */
export function RulePanel({
  mode,
  n,
  rule,
  saved,
  options,
  impact,
  previewing,
  onChange,
  onApply,
  onCancel,
}: {
  mode: PanelMode
  n: number
  rule: Rule
  saved?: Rule
  options: Options
  impact?: Impact
  previewing: boolean
  onChange: (r: Rule) => void
  onApply: () => void
  onCancel: () => void
}) {
  const t = useT()
  const id = useId()
  const view = mode === 'view'
  const [tried, setTried] = useState(false)
  const problems = ruleProblems(rule)
  const blocking = Boolean(problems.points || problems.cap)
  const edited = saved && JSON.stringify({ ...saved, values: saved.values ?? [] }) !== JSON.stringify({ ...rule, values: rule.values ?? [] })
  const eyebrow = mode === 'new' ? t(m.newRule) : view ? t(m.readOnly) : edited ? t(m.ruleNEdited, { n }) : t(m.ruleN, { n })
  const title = mode === 'new' ? t(m.addOne) : view ? t(m.ruleN, { n }) : t(m.editRule)
  const apply = () => {
    if (Object.keys(problems).length) {
      setTried(true)
      return
    }
    onApply()
  }
  const show = (k: keyof typeof problems) => (k === 'points' || k === 'cap' || tried ? problems[k] : undefined)
  const setKind = (kind: Rule['kind']) => onChange({ ...ruleFor(kind, rule.id, options), enabled: rule.enabled })

  return (
    <RDialog.Root open onOpenChange={(o) => !o && onCancel()}>
      <RDialog.Portal>
        <RDialog.Overlay className="sc-panel-scrim" />
        <RDialog.Content className="sc-panel" aria-describedby={undefined}>
          <div className="sc-ph">
            <div className="flex min-w-0 flex-1 flex-col gap-0.5">
              <span className="k">{eyebrow}</span>
              <RDialog.Title className="m-0 text-[17px] font-bold tracking-[-0.01em]">{title}</RDialog.Title>
            </div>
            <RDialog.Close className="ibtn" aria-label={t(m.closePanel)}>
              <Icon name="close" size={14} strokeWidth={1.8} />
            </RDialog.Close>
          </div>
          <div className="sc-pb">
            <fieldset disabled={view} className="sc-fs">
              <legend className="vh">{title}</legend>
              {/* When */}
              <div className="flex flex-col gap-1.5">
                {mode === 'new' ? (
                  <>
                    <Lbl>{t(m.when)}</Lbl>
                    <div role="radiogroup" aria-label={t(m.when)} className="flex flex-col gap-1.5">
                      {ruleKinds.map((k) => (
                        <label key={k} className="opt sc-opt">
                          <input type="radio" name={`${id}-kind`} checked={rule.kind === k} onChange={() => setKind(k)} />
                          <span className={rule.kind === k ? 'font-semibold' : 'font-medium'}>{kindPick(k)}</span>
                        </label>
                      ))}
                    </div>
                  </>
                ) : (
                  <>
                    <Lbl htmlFor={`${id}-kind`}>{t(m.when)}</Lbl>
                    <Select id={`${id}-kind`} className="w-full" value={rule.kind} onChange={(e) => setKind(e.target.value as Rule['kind'])}>
                      {ruleKinds.map((k) => (
                        <option key={k} value={k}>
                          {kindLabel(k)}
                        </option>
                      ))}
                    </Select>
                  </>
                )}
              </div>
              <KindFields rule={rule} options={options} impact={impact} onChange={onChange} error={show} />
              <PointsAndCap rule={rule} saved={saved} mode={mode} onChange={onChange} error={show} />
              <div className="sc-switchrow">
                <span className="flex flex-col gap-px">
                  <span className="text-[13px] font-semibold">{t(m.ruleSwitch)}</span>
                  <span className="k">{t(m.ruleSwitchLead)}</span>
                </span>
                <input type="checkbox" className="tg" role="switch" checked={rule.enabled} aria-label={t(m.ruleIsOn, { n })} onChange={(e) => onChange({ ...rule, enabled: e.target.checked })} />
              </div>
            </fieldset>
            {view ? null : previewing || impact ? (
              <div className="sc-box" aria-live="polite">
                <span className="text-[13px] font-semibold">{t(m.ifYouSave)}</span>
                <span className="k" style={{ color: 'var(--ink-nav)' }}>
                  {previewing || !impact ? t(m.previewing) : t(m.ifYouSaveText, { count: impact.changed })}
                </span>
              </div>
            ) : null}
          </div>
          <div className="sc-pf">
            {view ? (
              <Button onClick={onCancel}>{t(m.close)}</Button>
            ) : (
              <>
                <Button onClick={onCancel}>{t(m.cancel)}</Button>
                <Button variant="primary" disabled={blocking} onClick={apply}>
                  {mode === 'new' ? t(m.addRuleButton) : t(m.apply)}
                </Button>
              </>
            )}
          </div>
        </RDialog.Content>
      </RDialog.Portal>
    </RDialog.Root>
  )
}

type ErrFn = (k: 'points' | 'contains' | 'form' | 'event' | 'field' | 'values' | 'cap') => string | undefined

/** The inputs for what the rule counts. */
function KindFields({ rule, options, impact, onChange, error }: { rule: Rule; options: Options; impact?: Impact; onChange: (r: Rule) => void; error: ErrFn }) {
  const t = useT()
  const id = useId()
  switch (rule.kind) {
    case 'open':
      return (
        <div className="flex flex-col gap-1.5">
          <Lbl>{t(m.whichEmails)}</Lbl>
          <div role="radiogroup" aria-label={t(m.whichEmails)} className="flex flex-col gap-2">
            <label className="opt sc-opt">
              <input type="radio" name={`${id}-em`} checked={!rule.listId} onChange={() => onChange({ ...rule, listId: undefined })} />
              <span className={!rule.listId ? 'font-semibold' : 'font-medium'}>{t(m.anyEmail)}</span>
            </label>
            <label className="opt sc-opt">
              <input type="radio" name={`${id}-em`} checked={Boolean(rule.listId)} onChange={() => onChange({ ...rule, listId: options.lists[0]?.id })} disabled={!options.lists.length} />
              <span className="flex min-w-0 flex-1 flex-col gap-1.5">
                <span className={rule.listId ? 'font-semibold' : 'font-medium'}>{t(m.listEmails)}</span>
                {rule.listId ? (
                  <Select className="w-full" aria-label={t(m.whichList)} value={rule.listId} onChange={(e) => onChange({ ...rule, listId: e.target.value })}>
                    {options.lists.map((l) => (
                      <option key={l.id} value={l.id}>
                        {l.name}
                      </option>
                    ))}
                  </Select>
                ) : null}
              </span>
            </label>
          </div>
        </div>
      )
    case 'click': {
      const err = error('contains')
      return (
        <div className="flex flex-col gap-1.5">
          <Lbl>{t(m.whichLink)}</Lbl>
          <div role="radiogroup" aria-label={t(m.whichLink)} className="flex flex-col gap-2">
            <label className="opt sc-opt">
              <input type="radio" name={`${id}-lk`} checked={rule.contains === undefined} onChange={() => onChange({ ...rule, contains: undefined })} />
              <span className={rule.contains === undefined ? 'font-semibold' : 'font-medium'}>{t(m.anyLink)}</span>
            </label>
            <label className="opt sc-opt">
              <input type="radio" name={`${id}-lk`} checked={rule.contains !== undefined} onChange={() => onChange({ ...rule, contains: rule.contains ?? '' })} />
              <span className="flex min-w-0 flex-1 flex-col gap-1.5">
                <span className={rule.contains !== undefined ? 'font-semibold' : 'font-medium'}>{t(m.linkContains)}</span>
                {rule.contains !== undefined ? (
                  <Input
                    className="mono w-full"
                    value={rule.contains}
                    aria-label={t(m.linkText)}
                    invalid={Boolean(err)}
                    aria-describedby={err ? `${id}-cerr` : undefined}
                    onChange={(e) => onChange({ ...rule, contains: e.target.value })}
                  />
                ) : null}
              </span>
            </label>
          </div>
          {err ? <FieldErr id={`${id}-cerr`}>{err}</FieldErr> : null}
        </div>
      )
    }
    case 'form': {
      const err = error('form')
      return (
        <div className="flex flex-col gap-1.5">
          <Lbl htmlFor={`${id}-form`}>{t(m.whichForm)}</Lbl>
          <Select id={`${id}-form`} className="w-full" value={rule.formId ?? ''} invalid={Boolean(err)} onChange={(e) => onChange({ ...rule, formId: e.target.value || undefined })}>
            {rule.formId ? null : <option value="">{t(m.pickAForm)}</option>}
            {options.forms.map((f) => (
              <option key={f.id} value={f.id}>
                {f.name}
              </option>
            ))}
          </Select>
          {err ? <FieldErr>{err}</FieldErr> : null}
        </div>
      )
    }
    case 'api_event': {
      const err = error('event')
      const events = options.apiEvents
      return (
        <div className="flex flex-col gap-1.5">
          <Lbl htmlFor={`${id}-ev`}>{t(m.whichEvent)}</Lbl>
          <Input
            id={`${id}-ev`}
            className="mono w-full"
            value={rule.event ?? ''}
            aria-label={t(m.eventName)}
            invalid={Boolean(err)}
            onChange={(e) => onChange({ ...rule, event: e.target.value })}
          />
          {err ? <FieldErr>{err}</FieldErr> : null}
          <span className="k font-medium">
            {events.length ? (
              <T
                msg={m.eventsLately}
                values={{
                  events: (
                    <>
                      {events.map((ev, i) => (
                        <span key={ev.name}>
                          {i > 0 ? ' · ' : ''}
                          <span className="mono">{ev.name}</span> {count(ev.count)}
                        </span>
                      ))}
                    </>
                  ),
                }}
              />
            ) : (
              t(m.noEventsLately)
            )}
          </span>
        </div>
      )
    }
    case 'unsubscribe':
      return (
        <div className="flex flex-col gap-1.5">
          <Lbl htmlFor={`${id}-list`}>{t(m.whichList)}</Lbl>
          <Select id={`${id}-list`} className="w-full" value={rule.listId ?? ''} onChange={(e) => onChange({ ...rule, listId: e.target.value || undefined })}>
            <option value="">{t(m.anyList)}</option>
            {options.lists.map((l) => (
              <option key={l.id} value={l.id}>
                {l.name}
              </option>
            ))}
          </Select>
        </div>
      )
    case 'field':
      return <FieldCondition rule={rule} options={options} impact={impact} onChange={onChange} error={error} />
  }
  return null
}

/** Field, comparison and values, with how many match today. */
function FieldCondition({ rule, options, impact, onChange, error }: { rule: Rule; options: Options; impact?: Impact; onChange: (r: Rule) => void; error: ErrFn }) {
  const t = useT()
  const [text, setText] = useState((rule.values ?? []).join(', '))
  const [last, setLast] = useState(rule.values ?? [])
  if (rule.values !== last && (rule.values ?? []).join(',') !== last.join(',')) {
    setLast(rule.values ?? [])
    setText((rule.values ?? []).join(', '))
  }
  const matches = impact?.fieldMatches?.[rule.id] as number | undefined
  const err = error('field') ?? error('values')
  return (
    <div className="flex flex-col gap-1.5">
      <Lbl>{t(m.fieldAndValue)}</Lbl>
      <div className="sc-fieldrow">
        <Select aria-label={t(m.field)} value={rule.field ?? ''} onChange={(e) => onChange({ ...rule, field: e.target.value })}>
          {rule.field ? null : <option value="">{t(m.pickAField)}</option>}
          {options.fields.map((f) => (
            <option key={f.key} value={f.key}>
              {f.key}
            </option>
          ))}
        </Select>
        <Select aria-label={t(m.op)} value={rule.op ?? 'is_one_of'} onChange={(e) => onChange({ ...rule, op: e.target.value as Rule['op'], values: e.target.value === 'is_set' ? [] : rule.values })}>
          <option value="is_one_of">{t(m.opOneOf)}</option>
          <option value="is_not">{t(m.opNot)}</option>
          <option value="is_set">{t(m.opSet)}</option>
        </Select>
        {rule.op === 'is_set' ? (
          <span />
        ) : (
          <Input
            className="w-full"
            value={text}
            aria-label={t(m.values)}
            title={t(m.valuesHint)}
            invalid={Boolean(error('values'))}
            onChange={(e) => {
              setText(e.target.value)
              const values = e.target.value
                .split(',')
                .map((v) => v.trim())
                .filter(Boolean)
              setLast(values)
              onChange({ ...rule, values })
            }}
          />
        )}
      </div>
      {err ? <FieldErr>{err}</FieldErr> : null}
      {matches !== undefined ? <span className="k font-medium">{t(m.matchToday, { count: matches })}</span> : null}
    </div>
  )
}

const simpleCaps: CapKind[] = ['none', 'once', 'once_day', 'once_week', 'once_month']

/** Points, and the cap (While true for field conditions). */
function PointsAndCap({ rule, saved, mode, onChange, error }: { rule: Rule; saved?: Rule; mode: PanelMode; onChange: (r: Rule) => void; error: ErrFn }) {
  const t = useT()
  const id = useId()
  const [text, setText] = useState(Number.isFinite(rule.points) ? signed(rule.points).replace('−', '-') : '')
  const [last, setLast] = useState(rule.points)
  if (rule.points !== last && !(Number.isNaN(rule.points) && Number.isNaN(last))) {
    setLast(rule.points)
    if (parsePoints(text) !== rule.points) setText(Number.isFinite(rule.points) ? signed(rule.points).replace('−', '-') : '')
  }
  const pErr = error('points')
  const cErr = error('cap')
  const hint = mode === 'new' ? t(m.pointsRangeShort) : saved && saved.points !== rule.points ? t(m.pointsWas, { points: signed(saved.points) }) : t(m.pointsRange)
  const points = (
    <div className="flex flex-col gap-1.5">
      <Lbl htmlFor={`${id}-pts`}>{t(m.points)}</Lbl>
      <div className="flex items-center gap-2">
        <Input
          id={`${id}-pts`}
          className="num"
          style={{ width: 90, fontWeight: 700 }}
          type="text"
          inputMode="numeric"
          value={text}
          invalid={Boolean(pErr)}
          aria-describedby={`${id}-ph${pErr ? ` ${id}-perr` : ''}`}
          onChange={(e) => {
            setText(e.target.value)
            const v = parsePoints(e.target.value)
            setLast(v)
            onChange({ ...rule, points: v })
          }}
        />
        {mode === 'new' ? null : (
          <span className="k" id={`${id}-ph`}>
            {hint}
          </span>
        )}
      </div>
      {mode === 'new' ? (
        <span className="k font-medium" id={`${id}-ph`}>
          {hint}
        </span>
      ) : null}
      {pErr ? <FieldErr id={`${id}-perr`}>{pErr}</FieldErr> : null}
    </div>
  )
  if (rule.kind === 'field') {
    return (
      <>
        <div className="sc-pcgrid">
          {points}
          <div className="flex flex-col gap-1.5">
            <Lbl>{t(m.cap)}</Lbl>
            <div className="in sc-fixed" aria-label={t(m.cap)} role="note">
              {t(m.capWhileTrueLabel)}
            </div>
          </div>
        </div>
        <span className="k">{t(m.whileTrueExplain)}</span>
      </>
    )
  }
  const pointsCap = rule.cap.kind.startsWith('points_')
  const period = pointsCap ? rule.cap.kind.slice(7) : 'week'
  const defaultCap = Math.max(1, Math.abs(Number.isFinite(rule.points) ? rule.points : 5) * 4)
  const setCap = (kind: CapKind) => onChange({ ...rule, cap: kind.startsWith('points_') ? { kind, points: rule.cap.points ?? defaultCap } : { kind } })
  const cap = (
    <div className="flex flex-col gap-1.5">
      <Lbl htmlFor={`${id}-cap`}>{t(m.cap)}</Lbl>
      {pointsCap ? (
        <div className="flex items-center gap-2">
          <Select
            id={`${id}-cap`}
            style={{ width: 104 }}
            aria-label={t(m.capKindLabel)}
            value="points"
            onChange={(e) => setCap(e.target.value === 'points' ? rule.cap.kind : (e.target.value as CapKind))}
          >
            {simpleCaps.map((k) => (
              <option key={k} value={k}>
                {capKindLabel(k)}
              </option>
            ))}
            <option value="points">{t(m.capAtMost)}</option>
          </Select>
          <Input
            className="num"
            type="number"
            min={1}
            style={{ width: 70 }}
            value={rule.cap.points ?? ''}
            aria-label={t(m.capPointsLabel)}
            invalid={Boolean(cErr)}
            aria-describedby={cErr ? `${id}-caperr` : undefined}
            onChange={(e) => onChange({ ...rule, cap: { kind: rule.cap.kind, points: e.target.value === '' ? undefined : Number(e.target.value) } })}
          />
          <Select className="min-w-0 flex-1" aria-label={t(m.capPeriodLabel)} value={period} onChange={(e) => onChange({ ...rule, cap: { kind: `points_${e.target.value}` as CapKind, points: rule.cap.points } })}>
            <option value="day">{t(m.capPerDay)}</option>
            <option value="week">{t(m.capPerWeek)}</option>
            <option value="month">{t(m.capPerMonth)}</option>
          </Select>
        </div>
      ) : (
        <Select id={`${id}-cap`} className="w-full" aria-label={t(m.cap)} value={rule.cap.kind} onChange={(e) => setCap(e.target.value as CapKind)}>
          {capKinds.map((k) => (
            <option key={k} value={k}>
              {capKindLabel(k)}
            </option>
          ))}
        </Select>
      )}
      {cErr ? <FieldErr id={`${id}-caperr`}>{cErr}</FieldErr> : <span className="k font-medium">{capExplain(rule)}</span>}
    </div>
  )
  if (mode === 'new') {
    return (
      <div className="sc-pcgrid">
        {points}
        {cap}
      </div>
    )
  }
  return (
    <>
      {points}
      {cap}
    </>
  )
}
