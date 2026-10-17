import { Link } from '@tanstack/react-router'
import { useState, type ReactNode } from 'react'
import { Icon } from '@/components/icons/Icon'
import { Button } from '@/components/ui/Button'
import { Dialog } from '@/components/ui/Dialog'
import { OptionCard } from '@/components/ui/Field'
import { Note } from '@/components/ui/Note'
import { formatList, T, useT } from '@/i18n'
import { count, shortDate } from '@/lib/format'
import { useMe, useWorkspace } from '@/lib/session'
import { bandName, type Band } from './bands'
import { m } from './messages'
import { MoveChip, Swatch } from './ScoringCards'
import { signed, type Options, type Rule } from './ruleText'
import type { ConfigChanges, Impact } from './useScoringDraft'

/** The rule's kind for the save line's select. */
function lineKind(r: Rule): string {
  switch (r.kind) {
    case 'open':
      return r.listId ? 'openList' : 'open'
    case 'click':
      return r.contains ? 'clickContains' : 'click'
    case 'api_event':
      return 'apiEvent'
    case 'soft_bounce':
      return 'softBounce'
    case 'unsubscribe':
      return r.listId ? 'unsubscribeList' : 'unsubscribe'
  }
  return r.kind
}

/** "rule 3, decay and the thresholds". */
export function changesText(c: ConfigChanges, t: ReturnType<typeof useT>): string {
  const parts: string[] = []
  if (c.rules.length === 1) parts.push(t(m.changedRule, { n: c.rules[0]! }))
  else if (c.rules.length > 1) parts.push(t(m.changedRules, { list: formatList(c.rules.map(String)) }))
  if (c.deleted) parts.push(t(m.changedDeleted, { count: c.deleted }))
  if (c.decay) parts.push(t(m.changedDecay))
  if (c.bands) parts.push(t(m.changedBands))
  return formatList(parts)
}

/** Whether the only change is one rule's points. */
export function onlyPoints(c: ConfigChanges, rule: Rule | undefined, was: Rule | undefined): boolean {
  if (c.rules.length !== 1 || c.deleted || c.decay || c.bands || !rule || !was) return false
  return JSON.stringify({ ...was, points: 0, values: was.values ?? [] }) === JSON.stringify({ ...rule, points: 0, values: rule.values ?? [] }) && was.points !== rule.points
}

/** "about 3 minutes", from the preview's pace (a minute at least). */
function duration(impact: Impact | undefined, t: ReturnType<typeof useT>): string {
  const perContact = impact && impact.rescored > 0 ? impact.durationMs / impact.rescored : 0
  const minutes = Math.max(1, Math.round((perContact * (impact?.contacts ?? 0)) / 60000))
  return t(m.aboutMinutes, { minutes })
}

const mono = (s: string): ReactNode => <span className="mono">{s}</span>

/**
 * "Save the change to rule 3?" (ScoringStates): who changes band, and
 * whether automations listening for "Score crosses…" start for contacts who
 * cross only because of the change.
 */
export function SaveDialog({
  changes,
  rules,
  saved,
  options,
  impact,
  saving,
  error,
  onCancel,
  onSave,
}: {
  changes: ConfigChanges
  rules: Rule[]
  saved: Rule[]
  options: Options
  impact?: Impact
  saving: boolean
  error?: ReactNode
  onCancel: () => void
  onSave: (startAutomations: boolean) => void
}) {
  const t = useT()
  const me = useMe()
  const { slug } = useWorkspace()
  const [start, setStart] = useState(false)
  const n = changes.rules[0]
  const rule = n ? rules[n - 1] : undefined
  const was = rule ? saved.find((r) => r.id === rule.id) : undefined
  const single = onlyPoints(changes, rule, was)
  const title = changes.setup ? t(m.saveTitleSetup) : single || (changes.rules.length === 1 && !changes.deleted && !changes.decay && !changes.bands) ? t(m.saveTitleRule, { n: n! }) : t(m.saveTitle)
  const flows = (impact?.automations ?? []).filter((a) => a.crossed > 0)
  const crossing = flows.reduce((s, a) => Math.max(s, a.crossed), 0)
  const button = changes.setup ? t(m.saveSetupButton) : changes.rules.length === 1 && !changes.deleted && !changes.decay && !changes.bands ? t(m.saveRule, { n: n! }) : t(m.saveAll)
  const order: Band[] = ['hot', 'warm', 'cold']

  return (
    <Dialog
      open
      onOpenChange={(o) => !o && !saving && onCancel()}
      size="lg"
      className="sc-savedlg"
      hideClose
      eyebrow={<span className="k">{t(m.saveEyebrow, { name: me.data?.user.name ?? '', day: shortDate(new Date()) })}</span>}
      title={title}
      footer={
        <>
          <Button onClick={onCancel} disabled={saving}>
            {t(m.cancel)}
          </Button>
          <Button variant="primary" loading={saving} onClick={() => onSave(start)}>
            {button}
          </Button>
        </>
      }
    >
      <div className="flex flex-col gap-2">
        <span style={{ color: 'var(--ink-nav)' }}>
          {changes.setup ? (
            t(m.saveSetup)
          ) : single && rule && was ? (
            <T
              msg={m.saveRuleLine}
              values={{
                n: n!,
                kind: lineKind(rule),
                list: options.lists.find((l) => l.id === rule.listId)?.name ?? '',
                text: mono(rule.contains ?? ''),
                form: options.forms.find((f) => f.id === rule.formId)?.name ?? '',
                event: mono(rule.event ?? ''),
                field: mono(rule.field ?? ''),
                from: signed(was.points),
                to: signed(rule.points),
              }}
            />
          ) : (
            t(m.saveChanges, { changes: changesText(changes, t) })
          )}
        </span>
        {impact ? (
          <div className="sc-mrow">
            <span className="font-bold">{impact.changed ? t(m.moves, { count: impact.changed }) : t(m.noMoves)}</span>
            {impact.moves.map((mv) => (
              <MoveChip key={mv.from + mv.to} from={mv.from} to={mv.to} n={mv.count} />
            ))}
          </div>
        ) : (
          <span className="k">{t(m.previewing)}</span>
        )}
      </div>
      {impact ? (
        <div className="sc-bandbox-row">
          {order.map((b) => (
            <div key={b} className="sc-bandbox">
              <span className="k">
                <span className="sc-band">
                  <Swatch band={b} />
                  {bandName(b)}
                </span>
              </span>
              <span className="num whitespace-nowrap text-[13.5px] font-bold">
                {count(impact.bandsBefore[b])}{' '}
                <span className="sc-arrowr" aria-hidden="true">
                  →
                </span>
                <span className="vh">{t(m.moveTo)}</span> {count(impact.bandsAfter[b])}
              </span>
            </div>
          ))}
        </div>
      ) : null}
      {flows.length ? (
        <fieldset className="sc-fs" style={{ gap: 8 }}>
          <legend className="mb-1.5 text-[14px] font-bold">{t(m.automationsQuestion)}</legend>
          {flows.map((a) => (
            <span key={a.id} style={{ color: 'var(--ink-nav)', fontSize: 13.5 }}>
              <T
                msg={m.automationsLead}
                values={{
                  count: a.crossed,
                  threshold: a.threshold,
                  direction: a.direction,
                  name: a.name,
                  link: (chunks) => (
                    <Link to="/w/$workspace/automations/$automationId" params={{ workspace: slug, automationId: a.id }}>
                      {chunks}
                    </Link>
                  ),
                }}
              />
            </span>
          ))}
          <OptionCard
            name="sc-start"
            value="no"
            checked={!start}
            onChange={() => setStart(false)}
            title={t(m.dontStart)}
            description={t(flows.every((a) => a.direction === 'falls') ? m.dontStartLeadFalls : m.dontStartLead)}
          />
          <OptionCard
            name="sc-start"
            value="yes"
            checked={start}
            onChange={() => setStart(true)}
            title={t(m.start, { count: crossing })}
            description={flows.map((a) => t(m.startLead, { name: a.name })).join(' ')}
          />
        </fieldset>
      ) : null}
      {error ? (
        <Note tone="bad" label={t(m.saveFailed)}>
          {error}
        </Note>
      ) : null}
      <div className="note n" role="note">
        <Icon name="info" size={16} />
        <div>{t(m.recalcNote, { duration: duration(impact, t) })}</div>
      </div>
    </Dialog>
  )
}
