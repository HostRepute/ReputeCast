import { Dialog as RDialog } from 'radix-ui'
import { useId, useState } from 'react'
import { ApiError } from '@/api/client'
import { Badge } from '@/components/ui/Badge'
import { Button } from '@/components/ui/Button'
import { count } from '@/lib/format'
import { acknowledgements, minutes, plural, scheduleSentence, thresholdsLine, verdictTitle, type Review, type Run } from './logic'

export type SendIntent = { acknowledged: string[]; confirm?: string }

/**
 * The deliberate last step before a send (SendReadinessConfirm): what goes
 * out, to whom and how; each warning confirmed; for large sends the
 * recipient count typed out.
 */
export function ConfirmSend({
  open,
  onOpenChange,
  review,
  run,
  action,
  onConfirm,
  onExclude,
}: {
  open: boolean
  onOpenChange: (open: boolean) => void
  review: Review
  run: Run
  /** "Schedule send", "Send now", "Schedule as approved". */
  action: string
  onConfirm: (intent: SendIntent) => Promise<void>
  /** "Exclude them instead": applies the unengaged fix and closes. */
  onExclude?: () => void
}) {
  const id = useId()
  const c = review.campaign
  const recipients = run.estimates.audience.final
  const acks = acknowledgements(run)
  const large = recipients >= review.largeSend
  const [checked, setChecked] = useState<string[]>([])
  const [typed, setTyped] = useState('')
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState<string>()
  const schedule = review.schedule
  const route = run.estimates.route
  const shares = route.shares.filter((s) => s.recipients > 0)
  const a = run.estimates.audience
  const missing = acks.filter((x) => !checked.includes(x.key)).length
  const countOK = !large || typed.replace(/\D/g, '') === String(recipients)
  const reason = missing ? `Confirm ${missing === 1 ? 'the warning' : `the ${missing} warnings`} first.` : !countOK ? `Type ${recipients} to confirm.` : undefined
  const close = (next: boolean) => {
    if (busy) return
    if (!next) {
      setChecked([])
      setTyped('')
      setError(undefined)
    }
    onOpenChange(next)
  }
  const confirm = async () => {
    if (reason || busy) return
    setBusy(true)
    setError(undefined)
    try {
      await onConfirm({ acknowledged: checked, confirm: large ? typed : undefined })
      setBusy(false)
      close(false)
    } catch (e) {
      setBusy(false)
      setError(e instanceof ApiError ? e.message : "Couldn't reach the server. Nothing was sent; try again.")
    }
  }
  const title =
    schedule.mode === 'now' ? `Send "${c.name}" to ${count(recipients)} ${plural(recipients, 'person', 'people')} now?` : `Schedule "${c.name}" for ${count(recipients)} ${plural(recipients, 'person', 'people')}?`
  const when =
    schedule.mode === 'now'
      ? 'Sending starts within a minute of confirming. Send Readiness runs again as it starts, and a Block stops the send.'
      : schedule.mode === 'best_time'
        ? `Each contact gets it at their usual open time, within 24 h of ${scheduleSentence(schedule)}. Checks run again 15 minutes before the send, and the campaign pauses itself if any check turns to Block.`
        : `${scheduleSentence(schedule)}. Checks run again 15 minutes before the send, and the campaign pauses itself if any check turns to Block.`
  const excluded = [a.suppressed ? `${count(a.suppressed)} suppressed` : '', a.role ? `${count(a.role)} role` : '', a.duplicates ? `${count(a.duplicates)} duplicates` : '']
    .filter(Boolean)
    .join(' · ')
  const rows: [string, string][] = [
    ['From', c.from || '—'],
    ['Reply-to', c.replyTo || 'same as From'],
    ['Subject', c.subject],
    ['A/B', c.ab ? c.ab.label : 'No test'],
    ['Route', shares.map((s) => `${s.short} ${count(s.recipients)}`).join(' · ')],
    ['Cost · duration', `$${route.totalCost.toFixed(2)} · about ${minutes(route.totalMinutes)}`],
    ['Auto-pause', thresholdsLine(review.thresholds)],
    ['Excluded', excluded || 'nobody'],
  ]
  return (
    <RDialog.Root open={open} onOpenChange={close}>
      <RDialog.Portal>
        <RDialog.Overlay className="ovl">
          <RDialog.Content className="rv-confirm" aria-describedby={`${id}-d`} onPointerDownOutside={(e) => e.preventDefault()}>
            <section className="c-head">
              <Badge tone={run.verdict === 'warnings' ? 'warn' : 'ok'} dot className="self-start">
                {verdictTitle(run.verdict)}
              </Badge>
              <RDialog.Title className="c-title">{title}</RDialog.Title>
              <RDialog.Description id={`${id}-d`} className="text-ink-2">
                {when}
              </RDialog.Description>
            </section>
            <section>
              {/* Two columns as on the board; one on small screens, where two would break the labels. */}
              <div className="grid grid-cols-2 gap-x-6 gap-y-2.5 max-sm:grid-cols-1 [&>*]:min-w-0">
                {rows.map(([k, v]) => (
                  <div key={k} className="rv-sum">
                    <span className="k shrink-0">{k}</span>
                    <span className={k === 'Subject' ? 'n text-right' : 'n'}>{v}</span>
                  </div>
                ))}
              </div>
            </section>
            {acks.length ? (
              <section className="c-acks" aria-label="Warnings to confirm">
                {acks.map((x) => (
                  <label key={x.key} className="rv-chk">
                    <input
                      type="checkbox"
                      checked={checked.includes(x.key)}
                      onChange={(e) => setChecked((v) => (e.target.checked ? [...v, x.key] : v.filter((k) => k !== x.key)))}
                    />
                    <span>
                      {x.text}
                      {x.fix && onExclude ? (
                        <>
                          {' '}
                          <a
                            href="#exclude"
                            onClick={(e) => {
                              e.preventDefault()
                              close(false)
                              onExclude()
                            }}
                          >
                            Exclude them instead
                          </a>
                        </>
                      ) : null}
                    </span>
                  </label>
                ))}
              </section>
            ) : null}
            <section className="c-last">
              {large ? (
                <label className="rv-typed">
                  Type the recipient count to confirm
                  <input
                    className="in mono rv-count"
                    inputMode="numeric"
                    autoComplete="off"
                    value={typed}
                    placeholder={String(recipients)}
                    onChange={(e) => setTyped(e.target.value)}
                    onKeyDown={(e) => {
                      if (e.key === 'Enter') void confirm()
                    }}
                  />
                  <span className="k font-medium">This is deliberate: large sends can't be recalled once they leave the queue.</span>
                </label>
              ) : null}
              {error ? (
                <div className="note b" role="alert">
                  <span>{error}</span>
                </div>
              ) : null}
              <div className="flex flex-wrap justify-end gap-2.5">
                <RDialog.Close className="btn min-h-11">Back</RDialog.Close>
                <Button variant="primary" className="min-h-11 px-5 text-[14px]" disabledReason={reason} loading={busy} onClick={() => void confirm()}>
                  {action}
                </Button>
              </div>
            </section>
          </RDialog.Content>
        </RDialog.Overlay>
      </RDialog.Portal>
    </RDialog.Root>
  )
}
