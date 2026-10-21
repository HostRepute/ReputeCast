import { useState } from 'react'
import { $api, errorMessage } from '@/api/client'
import { Badge } from '@/components/ui/Badge'
import { Button } from '@/components/ui/Button'
import { Dialog } from '@/components/ui/Dialog'
import { Field, Input, Select } from '@/components/ui/Field'
import { Grid } from '@/components/ui/Layout'
import { Note } from '@/components/ui/Note'
import { toast } from '@/components/ui/Toast'
import { useT } from '@/i18n'
import { common } from '@/i18n/common'
import { count, decimal, shortDate } from '@/lib/format'
import { usePermissionReason, useWorkspace } from '@/lib/session'
import type { ProviderCard } from './common'
import { mo } from './overview.messages'
import { useRefreshProviders } from './ProviderIncidents'
import { RemoveProviderDialog } from './RemoveProviderDialog'

type Model = ProviderCard['cost']['basis']

/**
 * Parses "2.0 %", "0.10", "$24" into a number; undefined when empty or
 * invalid. In a region format with a decimal comma (Vietnamese) "0,10" and
 * "1.200" read as written there, and a lone dot is still a decimal point
 * unless three digits follow it ("0.1" → 0.1, "1.200" → 1200).
 */
export function parseNumber(v: string): number | undefined {
  let s = v.replace(/[%$\s]/g, '')
  if (!decimal(1.5, 1).includes(',')) s = s.replace(/,/g, '')
  else if (s.includes(',')) s = s.replace(/\./g, '').replace(',', '.')
  else if (/^\d+\.\d{3}$/.test(s) && !s.startsWith('0.')) s = s.replace('.', '')
  const n = Number(s)
  return v.trim() === '' || Number.isNaN(n) ? undefined : n
}

/** A number as a field shows it, with its own decimals, in the region format: "0.1", "24.95" · "0,1". */
function fieldNumber(v: number): string {
  return decimal(v, (String(v).split('.')[1] ?? '').length)
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
  const t = useT()
  const { slug } = useWorkspace()
  const refresh = useRefreshProviders()
  const reason = usePermissionReason('providers:manage')
  const readOnly = Boolean(reason)
  const [label, setLabel] = useState(card.label)
  const [model, setModel] = useState<Model>(card.cost.basis)
  const [per1000, setPer1000] = useState(card.cost.basis === 'metered' && card.cost.per1000 !== undefined ? fieldNumber(card.cost.per1000) : '')
  const [monthly, setMonthly] = useState(card.cost.monthly !== undefined ? fieldNumber(card.cost.monthly) : '')
  const [bounce, setBounce] = useState(fieldNumber(card.bounce.limit))
  const [complaint, setComplaint] = useState(fieldNumber(card.complaint.limit))
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
      toast({ kind: 'success', title: t(mo.settingsSaved, { name: card.name }) })
    },
    onError: (e) => setErrors({ form: errorMessage(e) }),
  })
  const creds = $api.useMutation('put', '/api/v1/workspaces/{workspace}/providers/{providerId}/credentials', {
    onSuccess: () => {
      void refresh()
      setReplacing({})
      toast({ kind: 'success', title: t(mo.credentialsAccepted, { name: card.name }), detail: t(mo.credentialsAcceptedDetail) })
    },
    onError: (e) => setErrors({ credentials: errorMessage(e) }),
  })

  const submit = () => {
    const next: Record<string, string> = {}
    const b = parseNumber(bounce)
    const c = parseNumber(complaint)
    const p = parseNumber(per1000)
    const m = parseNumber(monthly)
    if (b === undefined || b <= 0 || b > 100) next.bounce = t(mo.errBounce)
    if (c === undefined || c <= 0 || c > 100) next.complaint = t(mo.errComplaint)
    if (model === 'metered' && (p === undefined || p < 0)) next.per1000 = t(mo.errPer1000, { thousand: count(1000), example: decimal(0.1, 2) })
    if (model !== 'metered' && (m === undefined || m < 0)) next.monthly = t(mo.errMonthly)
    if (!label.trim()) next.label = t(mo.errLabel)
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
      setErrors({ credentials: t(mo.errCredential) })
      return
    }
    creds.mutate({ params: { path }, body: { values } })
  }

  return (
    <>
      <Dialog
        open={open}
        onOpenChange={onOpenChange}
        title={t(mo.dialogTitle, { name: card.name })}
        description={
          card.label
            ? t(mo.dialogDescriptionLabel, { label: card.label, date: shortDate(card.connectedAt) })
            : t(mo.dialogDescription, { date: shortDate(card.connectedAt) })
        }
        size="lg"
        footer={
          <>
            <Button variant="danger" className="mr-auto max-md:mr-0" disabledReason={reason} onClick={() => setConfirmDisconnect(true)}>
              {t(mo.removeProvider)}
            </Button>
            <Button onClick={() => onOpenChange(false)}>{t(common.cancel)}</Button>
            <Button variant="primary" disabledReason={reason} loading={save.isPending} onClick={submit}>
              {t(common.saveChanges)}
            </Button>
          </>
        }
      >
        <div className="flex flex-col gap-4">
          {errors.form ? <Note tone="bad">{errors.form}</Note> : null}
          <Field label={t(mo.label)} hint={t(mo.labelHint)} error={errors.label}>
            <Input value={label} onChange={(e) => setLabel(e.target.value)} maxLength={120} disabled={readOnly} />
          </Field>

          <section className="flex flex-col gap-2.5" aria-labelledby="pv-creds">
            <h3 id="pv-creds" className="m-0 text-[14px] font-semibold">
              {t(mo.credentials)}
            </h3>
            <span className="k">{t(mo.credentialsLine, { name: card.name })}</span>
            {card.credentials.map((f) => {
              const open = f.key in replacing
              return (
                <div key={f.key} className="flex flex-col gap-1.5">
                  <div className="flex flex-wrap items-center justify-between gap-2 text-[13px]">
                    <span className="font-semibold">{f.label}</span>
                    <span className="flex items-center gap-2">
                      <Badge tone={f.set ? 'ok' : 'neutral'}>{f.set ? t(mo.credentialSet) : t(mo.credentialNotSet)}</Badge>
                      {!open ? (
                        <Button size="sm" disabledReason={reason} onClick={() => setReplacing((r) => ({ ...r, [f.key]: '' }))}>
                          {t(mo.replace)}
                        </Button>
                      ) : null}
                    </span>
                  </div>
                  {open ? (
                    <Input
                      type="password"
                      autoComplete="new-password"
                      aria-label={t(mo.newCredential, { credential: f.label })}
                      placeholder={t(mo.newCredentialPlaceholder, { credential: f.label.toLowerCase() })}
                      value={replacing[f.key]}
                      autoFocus={focus === 'credentials' && f.key === card.credentials[0]?.key}
                      onChange={(e) => setReplacing((r) => ({ ...r, [f.key]: e.target.value }))}
                    />
                  ) : null}
                </div>
              )
            })}
            {errors.credentials ? (
              <span className="err" role="alert">
                {errors.credentials}
              </span>
            ) : null}
            {Object.keys(replacing).length ? (
              <div className="flex flex-wrap gap-2">
                <Button size="sm" variant="primary" loading={creds.isPending} onClick={replace}>
                  {t(mo.checkAndSave)}
                </Button>
                <Button size="sm" onClick={() => setReplacing({})}>
                  {t(mo.keepCurrent)}
                </Button>
              </div>
            ) : null}
          </section>

          <section className="flex flex-col gap-2.5" aria-labelledby="pv-pricing">
            <h3 id="pv-pricing" className="m-0 text-[14px] font-semibold">
              {t(mo.pricing)}
            </h3>
            <Grid cols={2} gap={10}>
              <Field label={t(mo.howYouPay)}>
                <Select value={model} onChange={(e) => setModel(e.target.value as Model)} disabled={readOnly}>
                  <option value="metered">{t(mo.perEmail)}</option>
                  <option value="plan">{t(mo.monthlyPlan)}</option>
                  <option value="self_hosted">{t(mo.selfHostedServer)}</option>
                </Select>
              </Field>
              {model === 'metered' ? (
                <Field label={t(mo.pricePer1000, { thousand: count(1000) })} error={errors.per1000}>
                  <Input inputMode="decimal" value={per1000} onChange={(e) => setPer1000(e.target.value)} disabled={readOnly} />
                </Field>
              ) : (
                <Field label={model === 'plan' ? t(mo.planPrice) : t(mo.serverCost)} error={errors.monthly}>
                  <Input inputMode="decimal" value={monthly} onChange={(e) => setMonthly(e.target.value)} disabled={readOnly} />
                </Field>
              )}
            </Grid>
          </section>

          <section className="flex flex-col gap-2.5" aria-labelledby="pv-limits">
            <h3 id="pv-limits" className="m-0 text-[14px] font-semibold">
              {t(mo.limitsTitle)}
            </h3>
            <Grid cols={2} gap={10}>
              <Field label={t(mo.bounceLimit)} error={errors.bounce}>
                <Input inputMode="decimal" value={bounce} onChange={(e) => setBounce(e.target.value)} disabled={readOnly} />
              </Field>
              <Field label={t(mo.complaintLimit)} error={errors.complaint}>
                <Input inputMode="decimal" value={complaint} onChange={(e) => setComplaint(e.target.value)} disabled={readOnly} />
              </Field>
            </Grid>
          </section>
        </div>
      </Dialog>
      {confirmDisconnect ? (
        <RemoveProviderDialog
          providerId={card.id}
          name={card.name}
          open
          onOpenChange={setConfirmDisconnect}
          onRemoved={() => {
            setConfirmDisconnect(false)
            onOpenChange(false)
          }}
        />
      ) : null}
    </>
  )
}
