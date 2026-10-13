import { useState, type ReactNode } from 'react'
import { Badge } from '@/components/ui/Badge'
import { Button } from '@/components/ui/Button'
import { Grid, Row } from '@/components/ui/Layout'
import { cn } from '@/lib/cn'
import { runAction, type LiveActions, type LivePermissions } from './actions'
import { MailboxTable, ProgressCard, ProviderTable, Tiles } from './LiveParts'
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
  type Snapshot,
} from './model'

/**
 * The decision view of an open incident (LiveMonitorPaused for an
 * auto-pause, LiveMonitorOutage for a provider outage): what happened, the
 * evidence, likely causes, what to do, and the choices; then progress,
 * counters and the breakdown that matters.
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
  return (
    <>
      <IncidentPanel s={s} incident={incident} actions={actions} can={can} onCancelAll={onCancelAll} />
      <ProgressCard s={s} incident={incident} />
      <Tiles tiles={tiles(s, outage ? 'outage' : 'threshold', incident)} />
      {outage ? <ProviderTable s={s} outage={incident} /> : <MailboxTable s={s} decision />}
    </>
  )
}

const tones = {
  bad: { border: 'border-bad-line', head: 'bg-[#FEF2F2]', text: 'text-[#7F1D1D]' },
  warn: { border: 'border-[#FCD34D]', head: 'bg-[#FFFBEB]', text: 'text-[#78350F]' },
}

function IncidentPanel({ s, incident, actions, can, onCancelAll }: { s: Snapshot; incident: Incident; actions: LiveActions; can: LivePermissions; onCancelAll: () => void }) {
  const outage = incident.kind === 'provider_outage'
  const t = tones[outage ? 'warn' : 'bad']
  const options = incident.detail.options ?? []
  const [choice, setChoice] = useState(() => defaultOption(incident))
  const picked = options.find((o) => o.key === choice) ?? options[0]
  const action = picked ? optionAction(incident, picked.key) : null
  const [busy, setBusy] = useState(false)
  const go = async () => {
    if (!action || busy) return
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
        {outage ? <ProviderLog incident={incident} /> : <Tripped s={s} incident={incident} />}
        <Causes incident={incident} border />
        {outage && incident.detail.move ? <MoveEstimate incident={incident} /> : <Recommended incident={incident} />}
      </Grid>
      {options.length ? (
        <div className="flex flex-col gap-3 border-t border-line-2 px-[22px] py-[18px]">
          <h2 className="h2">Choose how to continue</h2>
          <Grid min={280} gap={12} role="radiogroup" aria-label="Choose how to continue">
            {options.map((o) => (
              <Choice key={o.key} name={`continue-${incident.id}`} option={o} checked={picked?.key === o.key} onChange={() => setChoice(o.key)} />
            ))}
          </Grid>
          <div className="flex flex-wrap justify-end gap-2.5">
            {s.options.canCancel ? (
              <Button variant="danger" onClick={onCancelAll} disabledReason={can.cancel}>
                Cancel whole campaign
              </Button>
            ) : null}
            <Button variant="primary" className="min-h-[40px] px-[18px]" onClick={() => void go()} loading={busy} disabledReason={can.decide ?? (action ? undefined : 'Pick how to continue first.')}>
              {picked?.action || picked?.title || 'Continue'}
            </Button>
          </div>
        </div>
      ) : null}
    </section>
  )
}

/** A radio option card with the boards' inline layout (title, line break, caption). */
function Choice({ name, option, checked, onChange }: { name: string; option: IncidentOption; checked: boolean; onChange: () => void }) {
  const title = <span className="font-semibold">{option.title}</span>
  const body = <span className="k">{option.body}</span>
  return (
    <label className="opt">
      <input type="radio" name={name} value={option.key} checked={checked} onChange={onChange} />
      <span>
        {title}
        <br />
        {body}
      </span>
    </label>
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
  const f = trippedFacts(incident, s)
  return (
    <Column title="What tripped">
      <Row justify="space-between" gap={12} align="normal" className="text-[13px]">
        <span>{f.head.label}</span>
        <span className="n text-bad-fg">{f.head.value}</span>
      </Row>
      <div className="bar" style={{ height: 8 }}>
        <i style={{ width: `${f.bar.width}%`, background: 'var(--bad-strong)' }} />
      </div>
      {f.rows.map((r) => (
        <Row key={r.label} justify="space-between" gap={12} align="normal" className="text-[13px]">
          <span>{r.label}</span>
          <span className="n">{r.value}</span>
        </Row>
      ))}
      {incident.detail.note ? <span className="k">{incident.detail.note}</span> : null}
    </Column>
  )
}

/** "What Postal returned": the failed calls, then how retries worked. */
function ProviderLog({ incident }: { incident: Incident }) {
  const lines = logLines(incident)
  return (
    <Column title={`What ${incident.providerShort || incident.provider} returned`}>
      {lines.length ? (
        <div className="mono flex flex-col gap-1.5 text-[12.5px]">
          {lines.map((l) => (
            <span key={l.key} className={l.bad ? 'text-bad-fg' : 'text-ink-3'}>
              {l.text}
            </span>
          ))}
        </div>
      ) : (
        <span className="text-[13px]">ReputeCast&apos;s health checks marked the provider down; no send call to it has succeeded since.</span>
      )}
      <span className="k">Retries used exponential back-off (5 s to 60 s). No message was sent twice: each attempt carries an idempotency key.</span>
    </Column>
  )
}

function Causes({ incident, border }: { incident: Incident; border?: boolean }) {
  const causes = incident.detail.causes ?? []
  return (
    <Column title="Likely causes" border={border}>
      {causes.length ? (
        <ol className="m-0 flex flex-col gap-2 pl-[18px] text-[13px]">
          {causes.map((c, i) => (
            <li key={i}>
              {c.lead ? <strong>{c.lead}</strong> : null}
              {c.lead && c.body ? ' ' : null}
              {c.body}
            </li>
          ))}
        </ol>
      ) : (
        <span className="text-[13px]">Nothing in this send explains it yet; the evidence is on the left.</span>
      )}
    </Column>
  )
}

function Recommended({ incident }: { incident: Incident }) {
  const recs = incident.detail.recommendations ?? []
  return (
    <Column title="Recommended" border={false}>
      {recs.length ? (
        <ul className="m-0 flex flex-col gap-2 pl-[18px] text-[13px]">
          {recs.map((r, i) => (
            <li key={i}>{r}</li>
          ))}
        </ul>
      ) : (
        <span className="text-[13px]">Resume when you&apos;re confident, or cancel this share; the rest of the campaign keeps going.</span>
      )}
    </Column>
  )
}

/** "If you move the share to SES": cost, time, quota and reputation risk. */
function MoveEstimate({ incident }: { incident: Incident }) {
  const m = moveFacts(incident)
  if (!m) return null
  return (
    <Column title={m.title} border={false}>
      {m.rows.map((r) => (
        <Row key={r.label} justify="space-between" gap={12} align="normal" className="text-[13px]">
          <span>{r.label}</span>
          <span className="n">{r.value}</span>
        </Row>
      ))}
      {m.risk ? (
        <Row justify="space-between" gap={12} align="normal" className="text-[13px]">
          <span>Reputation risk</span>
          <Badge tone={m.risk.tone}>{m.risk.text}</Badge>
        </Row>
      ) : null}
      {m.note ? <span className="k">{m.note}</span> : null}
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
