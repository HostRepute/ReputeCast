import { useState, type ReactNode } from 'react'
import { Badge } from '@/components/ui/Badge'
import { Button } from '@/components/ui/Button'
import { Grid, Row } from '@/components/ui/Layout'
import { useT } from '@/i18n'
import { cn } from '@/lib/cn'
import { runAction, type LiveActions, type LivePermissions } from './actions'
import { MailboxTable, ProgressCard, ProviderTable, Tiles, useRatesControl } from './LiveParts'
import { m } from './messages'
import { ResumeDialog } from './ResumeDialog'
import {
  defaultOption,
  incidentTitle,
  logLines,
  moveFacts,
  optionAction,
  outageSummary,
  thresholdSummary,
  tiles,
  trippedFacts,
  type Incident,
  type IncidentOption,
  type MoveEstimate,
  type Snapshot,
} from './model'

/**
 * The decision view of an open incident: LiveMonitorPaused for an
 * auto-pause, LiveMonitorOutage for a provider outage.
 */
export function DecisionView({
  s,
  incident,
  actions,
  can,
  onCancelAll,
}: {
  s: Snapshot
  incident: Incident
  actions: LiveActions
  can: LivePermissions
  onCancelAll: () => void
}) {
  const outage = incident.kind === 'provider_outage'
  const [note, setNote] = useState<{ id: string; text: string }>()
  const rates = useRatesControl(s, actions, can, (text) => setNote({ id: incident.id, text }))
  return (
    <>
      <IncidentPanel s={s} incident={incident} actions={actions} can={can} onCancelAll={onCancelAll} />
      <ProgressCard s={s} incident={incident} note={note?.id === incident.id ? note.text : undefined} />
      <Tiles tiles={tiles(s, outage ? 'outage' : 'threshold', incident)} />
      {outage ? <ProviderTable s={s} outage={incident} /> : <MailboxTable s={s} decision rates={rates} />}
    </>
  )
}

const tones = {
  bad: { border: 'border-bad-line', head: 'bg-[#FEF2F2]', text: 'text-[#7F1D1D]' },
  warn: { border: 'border-[#FCD34D]', head: 'bg-[#FFFBEB]', text: 'text-[#78350F]' },
}

function IncidentPanel({ s, incident, actions, can, onCancelAll }: { s: Snapshot; incident: Incident; actions: LiveActions; can: LivePermissions; onCancelAll: () => void }) {
  const tr = useT()
  const outage = incident.kind === 'provider_outage'
  const t = tones[outage ? 'warn' : 'bad']
  const options = incident.detail.options ?? []
  const [choice, setChoice] = useState(() => defaultOption(incident))
  const picked = options.find((o) => o.key === choice) ?? options[0]
  const action = picked ? optionAction(incident, picked.key) : null
  const [busy, setBusy] = useState(false)
  const [asking, setAsking] = useState(false)
  const go = async () => {
    if (!action || busy) return
    if (!outage && picked?.key === 'resume') {
      // Resuming as is asks first: re-arm at the recommended rate, or
      // raise the limit for this campaign (Dialogs board, review GS-2).
      setAsking(true)
      return
    }
    setBusy(true)
    try {
      await runAction(actions, action, incident.detail.move?.targetShort)
    } catch {
      // The toast said what failed; the choice stays for another try.
    } finally {
      setBusy(false)
    }
  }
  return (
    <section role="alert" className={cn('card flex flex-col', t.border)} aria-labelledby={`incident-${incident.id}`}>
      <div className={cn('flex flex-wrap items-start gap-[18px] rounded-t-[10px] px-[22px] py-5', t.head)}>
        {outage ? <OutageIcon /> : <PauseIcon />}
        <div className="flex flex-col gap-1 max-md:min-w-[120px]" style={{ flex: '1 1 320px' }}>
          <span id={`incident-${incident.id}`} className={cn('text-[20px] font-bold tracking-[-0.01em]', t.text)}>
            {incidentTitle(incident)}
          </span>
          <span className={t.text}>{outage ? outageSummary(incident, s) : thresholdSummary(incident, s)}</span>
        </div>
      </div>
      <Grid min={300} gap={0}>
        {outage ? <ProviderLog s={s} incident={incident} /> : <Tripped s={s} incident={incident} />}
        <Causes incident={incident} border />
        {outage && incident.detail.move ? <MoveEstimate incident={incident} /> : <Recommended incident={incident} />}
      </Grid>
      {options.length ? (
        <div className="flex flex-col gap-3 border-t border-line-2 px-[22px] py-[18px]">
          <h2 className="h2">{tr(m.chooseTitle)}</h2>
          <Grid min={280} gap={12} role="radiogroup" aria-label={tr(m.chooseLabel)}>
            {options.map((o) => (
              <Choice
                key={o.key}
                name={`continue-${incident.id}`}
                option={o}
                checked={picked?.key === o.key}
                onChange={() => setChoice(o.key)}
                checks={o.key === 'move' ? incident.detail.move : undefined}
              />
            ))}
          </Grid>
          <div className="lv-dec-acts flex flex-wrap justify-end gap-2.5">
            {s.options.canCancel ? (
              <Button variant="danger" onClick={onCancelAll} disabledReason={can.cancel}>
                {tr(m.cancelWhole)}
              </Button>
            ) : null}
            <Button variant="primary" className="min-h-[40px] px-[18px]" onClick={() => void go()} loading={busy} disabledReason={can.decide ?? (action ? undefined : tr(m.pickFirst))}>
              {picked?.action || picked?.title || tr(m.continue)}
            </Button>
          </div>
        </div>
      ) : null}
      {outage ? null : <ResumeDialog open={asking} onOpenChange={setAsking} incident={incident} actions={actions} />}
    </section>
  )
}

/**
 * A radio option card with the boards' inline layout (title, line break,
 * caption). A move lists the checks ReputeCast ran on its target (review
 * GS-5): passed ones with a tick, what it couldn't tell with a dash.
 */
function Choice({
  name,
  option,
  checked,
  onChange,
  checks,
}: {
  name: string
  option: IncidentOption
  checked: boolean
  onChange: () => void
  checks?: MoveEstimate
}) {
  const t = useT()
  const title = <span className="font-semibold">{option.title}</span>
  const body = <span className="k">{option.body}</span>
  const list = checks?.checks ?? []
  return (
    <label className="opt">
      {/* On the boards the radio is a flex item that shrinks beside long text (to 13 px). */}
      <input type="radio" className="shrink" name={name} value={option.key} checked={checked} onChange={onChange} />
      <span>
        {title}
        <br />
        {body}
        {list.length && checks ? (
          <>
            <span className="k mt-2 block font-semibold text-ink">{t(m.checksTitle, { provider: checks.targetShort })}</span>
            <span className="lv-chks" role="list">
              {list.map((c) => (
                <span role="listitem" key={c.key}>
                  {c.result === 'pass' ? <CheckTick /> : <CheckDash />}
                  <span>{c.text}</span>
                </span>
              ))}
            </span>
          </>
        ) : null}
      </span>
    </label>
  )
}

function CheckTick() {
  return (
    <svg width="12" height="12" viewBox="0 0 16 16" fill="none" stroke="#15803D" strokeWidth="2.4" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
      <path d="m3 8.5 3.2 3L13 4.5" />
    </svg>
  )
}

function CheckDash() {
  return (
    <svg width="12" height="12" viewBox="0 0 16 16" fill="none" stroke="#B45309" strokeWidth="2.4" strokeLinecap="round" aria-hidden="true">
      <path d="M4 8h8" />
    </svg>
  )
}

function Column({ title, children, border = true }: { title: string; children: ReactNode; border?: boolean }) {
  return (
    <div className={cn('flex flex-col gap-2.5 px-[22px] py-[18px]', border && 'border-r border-line-2')}>
      <h2 className="h2">{title}</h2>
      {children}
    </div>
  )
}

/** "What tripped": the scope's rate against the limit, then the context. */
function Tripped({ s, incident }: { s: Snapshot; incident: Incident }) {
  const t = useT()
  const f = trippedFacts(incident, s)
  return (
    <Column title={t(m.whatTripped)}>
      <Row justify="space-between" gap={12} align="normal" className="text-[13px]">
        <span>{f.head.label}</span>
        <span className="n text-bad-fg">{f.head.value}</span>
      </Row>
      <div className="bar" style={{ height: 8 }} aria-hidden="true">
        <i style={{ width: `${f.bar.width}%`, background: 'var(--bad-strong)' }} />
      </div>
      {f.evidence ? (
        <Row gap={8} align="flex-start" className="text-[12.5px] text-ink-3">
          <Badge tone="bad" dot className="flex-none">
            {f.evidence.state}
          </Badge>
          <span>{f.evidence.text}</span>
        </Row>
      ) : null}
      {f.rows.map((r) => (
        <Row key={r.label} justify="space-between" gap={12} align="normal" className="text-[13px]">
          <span>{r.label}</span>
          <span className="n">{r.value}</span>
        </Row>
      ))}
      {incident.detail.note ? <span className="k">{incident.detail.note}</span> : null}
      {f.watch ? <p className="lv-fn rounded-lg border border-line bg-bg-2 px-2.5 py-2">{f.watch}</p> : null}
    </Column>
  )
}

/** "What Postal returned": the failed calls, then how retries worked (what the provider's idempotency really guarantees, review GS-1). */
function ProviderLog({ s, incident }: { s: Snapshot; incident: Incident }) {
  const t = useT()
  const lines = logLines(incident)
  const provider = incident.providerShort || incident.provider
  const keyed = s.providers.find((p) => p.id === incident.providerId)?.idempotent ?? false
  return (
    <Column title={t(m.whatReturned, { provider: incident.providerShort || incident.provider })}>
      {lines.length ? (
        <div className="mono flex flex-col gap-1.5 text-[12.5px]">
          {lines.map((l) => (
            <span key={l.key} className={l.bad ? 'text-bad-fg' : 'text-ink-3'}>
              {l.text}
            </span>
          ))}
        </div>
      ) : (
        <span className="text-[13px]">{t(m.healthDown)}</span>
      )}
      <span className="k">{t(keyed ? m.retriesKeyed : m.retriesUnkeyed, { provider })}</span>
    </Column>
  )
}

function Causes({ incident, border }: { incident: Incident; border?: boolean }) {
  const t = useT()
  const causes = incident.detail.causes ?? []
  return (
    <Column title={t(m.likelyCauses)} border={border}>
      {causes.length ? (
        <ol className="m-0 flex list-decimal flex-col gap-2 pl-[18px] text-[13px]">
          {causes.map((c, i) => (
            <li key={i}>
              {c.lead ? <strong>{c.lead}</strong> : null}
              {c.lead && c.body ? ' ' : null}
              {c.body}
            </li>
          ))}
        </ol>
      ) : (
        <span className="text-[13px]">{t(m.noCauses)}</span>
      )}
    </Column>
  )
}

function Recommended({ incident }: { incident: Incident }) {
  const t = useT()
  const recs = incident.detail.recommendations ?? []
  return (
    <Column title={t(m.recommended)} border={false}>
      {recs.length ? (
        <ul className="m-0 flex list-disc flex-col gap-2 pl-[18px] text-[13px]">
          {recs.map((r, i) => (
            <li key={i}>{r}</li>
          ))}
        </ul>
      ) : (
        <span className="text-[13px]">{t(m.recommendedDefault)}</span>
      )}
    </Column>
  )
}

/** "If you move the share to SES": cost, time, quota and reputation risk. */
function MoveEstimate({ incident }: { incident: Incident }) {
  const t = useT()
  const mv = moveFacts(incident)
  if (!mv) return null
  return (
    <Column title={mv.title} border={false}>
      {mv.rows.map((r) => (
        <Row key={r.label} justify="space-between" gap={12} align="normal" className="text-[13px]">
          <span>{r.label}</span>
          <span className="n">{r.value}</span>
        </Row>
      ))}
      {mv.risk ? (
        <Row justify="space-between" gap={12} align="normal" className="text-[13px]">
          <span>{t(m.reputationRisk)}</span>
          <Badge tone={mv.risk.tone}>{mv.risk.text}</Badge>
        </Row>
      ) : null}
      {mv.note ? <span className="k">{mv.note}</span> : null}
    </Column>
  )
}

function PauseIcon() {
  return (
    <svg width="40" height="40" viewBox="0 0 40 40" aria-hidden="true" className="flex-none">
      <circle cx="20" cy="20" r="19" fill="#FECACA" />
      <path d="M15 13v14M25 13v14" stroke="#991B1B" strokeWidth="3.5" strokeLinecap="round" />
    </svg>
  )
}

function OutageIcon() {
  return (
    <svg width="40" height="40" viewBox="0 0 40 40" aria-hidden="true" className="flex-none">
      <circle cx="20" cy="20" r="19" fill="#FDE68A" />
      <path d="M20 11v11M20 27v1" stroke="#92400E" strokeWidth="3" strokeLinecap="round" />
    </svg>
  )
}
