import { useState } from 'react'
import { $api, ApiError, errorMessage } from '@/api/client'
import { Badge } from '@/components/ui/Badge'
import { Button } from '@/components/ui/Button'
import { Dialog } from '@/components/ui/Dialog'
import { Field, Input, Select } from '@/components/ui/Field'
import { Grid } from '@/components/ui/Layout'
import { Note } from '@/components/ui/Note'
import { toast } from '@/components/ui/Toast'
import { shortDate } from '@/lib/format'
import { usePermissionReason, useWorkspace } from '@/lib/session'
import type { ProviderCard } from './common'
import { useRefreshProviders } from './ProviderIncidents'

type Model = ProviderCard['cost']['basis']

/** Parses "2.0 %", "0.10", "$24" into a number; undefined when empty or invalid. */
export function parseNumber(v: string): number | undefined {
  const n = Number(v.replace(/[%$,\s]/g, ''))
  return v.trim() === '' || Number.isNaN(n) ? undefined : n
}

/**
 * A provider's settings: its label, pricing and rate limits, write-only
 * credentials (replaced, never shown) and disconnecting it.
 */
export function ProviderSettingsDialog({
  card,
  open,
  onOpenChange,
  focus,
}: {
  card: ProviderCard
  open: boolean
  onOpenChange: (open: boolean) => void
  focus?: 'credentials'
}) {
  const { slug } = useWorkspace()
  const refresh = useRefreshProviders()
  const reason = usePermissionReason('providers:manage')
  const readOnly = Boolean(reason)
  const [label, setLabel] = useState(card.label)
  const [model, setModel] = useState<Model>(card.cost.basis)
  const [per1000, setPer1000] = useState(card.cost.basis === 'metered' && card.cost.per1000 !== undefined ? String(card.cost.per1000) : '')
  const [monthly, setMonthly] = useState(card.cost.monthly !== undefined ? String(card.cost.monthly) : '')
  const [bounce, setBounce] = useState(String(card.bounce.limit))
  const [complaint, setComplaint] = useState(String(card.complaint.limit))
  // The dialog mounts each time it opens, so the form starts from the card.
  const [replacing, setReplacing] = useState<Record<string, string>>(() =>
    focus === 'credentials' && card.credentials[0] ? { [card.credentials[0].key]: '' } : {},
  )
  const [errors, setErrors] = useState<Record<string, string>>({})
  const [confirmDisconnect, setConfirmDisconnect] = useState(false)

  const path = { workspace: slug, providerId: card.id }
  const save = $api.useMutation('patch', '/api/v1/workspaces/{workspace}/providers/{providerId}', {
    onSuccess: () => {
      void refresh()
      onOpenChange(false)
      toast({ kind: 'success', title: `${card.name} settings saved` })
    },
    onError: (e) => setErrors({ form: errorMessage(e) }),
  })
  const creds = $api.useMutation('put', '/api/v1/workspaces/{workspace}/providers/{providerId}/credentials', {
    onSuccess: () => {
      void refresh()
      setReplacing({})
      toast({ kind: 'success', title: `${card.name} accepted the new credentials`, detail: 'They are stored encrypted and never shown again.' })
    },
    onError: (e) => setErrors({ credentials: errorMessage(e) }),
  })
  const disconnect = $api.useMutation('delete', '/api/v1/workspaces/{workspace}/providers/{providerId}', {
    onSuccess: () => {
      void refresh()
      setConfirmDisconnect(false)
      onOpenChange(false)
      toast({ kind: 'deleted', title: `${card.name} disconnected`, detail: 'Its credentials were deleted.' })
    },
    onError: (e) => {
      setConfirmDisconnect(false)
      setErrors({ form: e instanceof ApiError && e.status === 409 ? e.message : errorMessage(e) })
    },
  })

  const submit = () => {
    const next: Record<string, string> = {}
    const b = parseNumber(bounce)
    const c = parseNumber(complaint)
    const p = parseNumber(per1000)
    const m = parseNumber(monthly)
    if (b === undefined || b <= 0 || b > 100) next.bounce = 'Enter a bounce limit between 0 and 100%.'
    if (c === undefined || c <= 0 || c > 100) next.complaint = 'Enter a complaint limit between 0 and 100%.'
    if (model === 'metered' && (p === undefined || p < 0)) next.per1000 = 'Enter the price per 1,000 emails, like 0.10.'
    if (model !== 'metered' && (m === undefined || m < 0)) next.monthly = 'Enter the price per month, like 15.'
    if (!label.trim()) next.label = 'Give the provider a label, like the region or server name.'
    setErrors(next)
    if (Object.keys(next).length) return
    save.mutate({
      params: { path },
      body: {
        label: label.trim(),
        pricing: model === 'metered' ? { model, per1000: p } : { model, monthly: m },
        limits: { bounce: b!, complaint: c! },
      },
    })
  }

  const replace = () => {
    const values = Object.fromEntries(Object.entries(replacing).filter(([, v]) => v.trim() !== ''))
    if (!Object.keys(values).length) {
      setErrors({ credentials: 'Enter the new value for the credential you are replacing.' })
      return
    }
    creds.mutate({ params: { path }, body: { values } })
  }

  return (
    <>
      <Dialog
        open={open}
        onOpenChange={onOpenChange}
        title={`${card.name} settings`}
        description={`${card.label ? `${card.label} · ` : ''}connected ${shortDate(card.connectedAt)}`}
        size="lg"
        footer={
          <>
            <Button variant="danger" className="mr-auto max-md:mr-0" disabledReason={reason} onClick={() => setConfirmDisconnect(true)}>
              Disconnect
            </Button>
            <Button onClick={() => onOpenChange(false)}>Cancel</Button>
            <Button variant="primary" disabledReason={reason} loading={save.isPending} onClick={submit}>
              Save changes
            </Button>
          </>
        }
      >
        <div className="flex flex-col gap-4">
          {errors.form ? <Note tone="bad">{errors.form}</Note> : null}
          <Field label="Label" hint="Shown beside the name, like the region or server." error={errors.label}>
            <Input value={label} onChange={(e) => setLabel(e.target.value)} maxLength={120} disabled={readOnly} />
          </Field>

          <section className="flex flex-col gap-2.5" aria-labelledby="pv-creds">
            <h3 id="pv-creds" className="m-0 text-[14px] font-semibold">
              Credentials
            </h3>
            <span className="k">Stored encrypted and never shown again. New values are checked with {card.name} before they replace the old ones.</span>
            {card.credentials.map((f) => {
              const open = f.key in replacing
              return (
                <div key={f.key} className="flex flex-col gap-1.5">
                  <div className="flex flex-wrap items-center justify-between gap-2 text-[13px]">
                    <span className="font-semibold">{f.label}</span>
                    <span className="flex items-center gap-2">
                      <Badge tone={f.set ? 'ok' : 'neutral'}>{f.set ? 'Set · write-only' : 'Not set'}</Badge>
                      {!open ? (
                        <Button size="sm" disabledReason={reason} onClick={() => setReplacing((r) => ({ ...r, [f.key]: '' }))}>
                          Replace
                        </Button>
                      ) : null}
                    </span>
                  </div>
                  {open ? (
                    <Input
                      type="password"
                      autoComplete="new-password"
                      aria-label={`New ${f.label}`}
                      placeholder={`New ${f.label.toLowerCase()}`}
                      value={replacing[f.key]}
                      autoFocus={focus === 'credentials' && f.key === card.credentials[0]?.key}
                      onChange={(e) => setReplacing((r) => ({ ...r, [f.key]: e.target.value }))}
                    />
                  ) : null}
                </div>
              )
            })}
            {errors.credentials ? <span className="err" role="alert">{errors.credentials}</span> : null}
            {Object.keys(replacing).length ? (
              <div className="flex flex-wrap gap-2">
                <Button size="sm" variant="primary" loading={creds.isPending} onClick={replace}>
                  Check and save credentials
                </Button>
                <Button size="sm" onClick={() => setReplacing({})}>
                  Keep current
                </Button>
              </div>
            ) : null}
          </section>

          <section className="flex flex-col gap-2.5" aria-labelledby="pv-pricing">
            <h3 id="pv-pricing" className="m-0 text-[14px] font-semibold">
              Pricing
            </h3>
            <Grid cols={2} gap={10}>
              <Field label="How you pay">
                <Select value={model} onChange={(e) => setModel(e.target.value as Model)} disabled={readOnly}>
                  <option value="metered">Per email sent</option>
                  <option value="plan">Monthly plan</option>
                  <option value="self_hosted">Self-hosted server</option>
                </Select>
              </Field>
              {model === 'metered' ? (
                <Field label="Price per 1,000 emails" error={errors.per1000}>
                  <Input inputMode="decimal" value={per1000} onChange={(e) => setPer1000(e.target.value)} disabled={readOnly} />
                </Field>
              ) : (
                <Field label={model === 'plan' ? 'Plan price per month' : 'Server cost per month'} error={errors.monthly}>
                  <Input inputMode="decimal" value={monthly} onChange={(e) => setMonthly(e.target.value)} disabled={readOnly} />
                </Field>
              )}
            </Grid>
          </section>

          <section className="flex flex-col gap-2.5" aria-labelledby="pv-limits">
            <h3 id="pv-limits" className="m-0 text-[14px] font-semibold">
              Limits shown on the card
            </h3>
            <Grid cols={2} gap={10}>
              <Field label="Bounce rate limit (%)" error={errors.bounce}>
                <Input inputMode="decimal" value={bounce} onChange={(e) => setBounce(e.target.value)} disabled={readOnly} />
              </Field>
              <Field label="Complaint rate limit (%)" error={errors.complaint}>
                <Input inputMode="decimal" value={complaint} onChange={(e) => setComplaint(e.target.value)} disabled={readOnly} />
              </Field>
            </Grid>
          </section>
        </div>
      </Dialog>
      <Dialog
        open={confirmDisconnect}
        onOpenChange={setConfirmDisconnect}
        dismissible={false}
        size="sm"
        title={`Disconnect ${card.name}?`}
        description={`ReputeCast deletes its credentials and stops checking it. Routing must not use ${card.name} any more: move its streams to another provider first. Past events and reports stay.`}
        footer={
          <>
            <Button onClick={() => setConfirmDisconnect(false)}>Keep connected</Button>
            <Button variant="danger-primary" loading={disconnect.isPending} onClick={() => disconnect.mutate({ params: { path } })}>
              Disconnect
            </Button>
          </>
        }
      />
    </>
  )
}
