import { useState } from 'react'
import { $api, errorMessage, type Schemas } from '@/api/client'
import { Button } from '@/components/ui/Button'
import { Dialog } from '@/components/ui/Dialog'
import { Loading } from '@/components/ui/Feedback'
import { Field, Input, Select } from '@/components/ui/Field'
import { Note } from '@/components/ui/Note'
import { toast } from '@/components/ui/Toast'
import { useT } from '@/i18n'
import { useWorkspace } from '@/lib/session'
import { CountryChips, RegistrationBadges, useKindName, useRefreshSms } from './common'
import { m } from './dialogs.messages'

type Card = Schemas['SmsProviderCard']

/** Parses "GB, de nl" into ["GB", "DE", "NL"]; undefined when a code isn't two letters. */
export function parseCountries(s: string): string[] | undefined {
  const codes = s
    .split(/[\s,;]+/)
    .map((c) => c.trim().toUpperCase())
    .filter(Boolean)
  if (codes.some((c) => !/^[A-Z]{2}$/.test(c))) return undefined
  return [...new Set(codes)]
}

/** Add an alphanumeric sender ID with its countries (SmsStates, step 3's "Or add"). */
export function AddSenderIdDialog({ providers, open, onOpenChange }: { providers: Card[]; open: boolean; onOpenChange: (o: boolean) => void }) {
  const t = useT()
  const { slug } = useWorkspace()
  const refresh = useRefreshSms()
  const capable = providers.filter((p) => p.badges.some((b) => b.capability === 'sender_ids' && b.support !== 'no'))
  const [providerId, setProviderId] = useState(capable[0]?.id ?? '')
  const [value, setValue] = useState('')
  const [countries, setCountries] = useState('')
  const [errors, setErrors] = useState<Record<string, string>>({})
  const add = $api.useMutation('post', '/api/v1/workspaces/{workspace}/sms/senders', {
    onSuccess: () => {
      void refresh()
      toast({ kind: 'success', title: t(m.idAdded, { value: value.trim().toUpperCase() }) })
      onOpenChange(false)
    },
    onError: (e) => setErrors({ form: errorMessage(e) }),
  })
  const submit = () => {
    const next: Record<string, string> = {}
    const v = value.trim()
    if (!/^(?=.*[A-Za-z])[A-Za-z0-9]{1,11}$/.test(v)) next.value = t(m.idInvalid)
    const cs = parseCountries(countries)
    if (!cs || cs.length === 0) next.countries = t(m.idCountriesInvalid)
    setErrors(next)
    if (Object.keys(next).length) return
    add.mutate({ params: { path: { workspace: slug } }, body: { providerId, value: v.toUpperCase(), countries: cs! } })
  }
  return (
    <Dialog
      open={open}
      onOpenChange={onOpenChange}
      title={t(m.addIdTitle)}
      description={t(m.addIdText)}
      footer={
        <>
          <Button onClick={() => onOpenChange(false)}>{t(m.cancel)}</Button>
          <Button variant="primary" loading={add.isPending} onClick={submit}>
            {t(m.add)}
          </Button>
        </>
      }
    >
      <div className="flex flex-col gap-3.5">
        {errors.form ? <Note tone="bad">{errors.form}</Note> : null}
        <Field label={t(m.idValue)} error={errors.value}>
          <Input value={value} maxLength={11} className="mono" onChange={(e) => setValue(e.target.value)} autoComplete="off" />
        </Field>
        <Field label={t(m.idProvider)}>
          <Select value={providerId} onChange={(e) => setProviderId(e.target.value)}>
            {capable.map((p) => (
              <option key={p.id} value={p.id}>
                {p.name}
              </option>
            ))}
          </Select>
        </Field>
        <Field label={t(m.idCountries)} hint={t(m.idCountriesHint)} error={errors.countries}>
          <Input value={countries} onChange={(e) => setCountries(e.target.value)} autoComplete="off" />
        </Field>
      </div>
    </Dialog>
  )
}

/** Import from providers: the account's numbers that aren't senders yet. */
export function ImportSendersDialog({ providers, open, onOpenChange }: { providers: Card[]; open: boolean; onOpenChange: (o: boolean) => void }) {
  const t = useT()
  const { slug } = useWorkspace()
  const refresh = useRefreshSms()
  const kindName = useKindName()
  const listing = providers.filter((p) => p.badges.some((b) => b.capability === 'toll_free' || b.capability === '10dlc'))
  const [providerId, setProviderId] = useState(listing[0]?.id ?? '')
  const provider = providers.find((p) => p.id === providerId)
  const q = $api.useQuery(
    'get',
    '/api/v1/workspaces/{workspace}/sms/providers/{providerId}/numbers',
    { params: { path: { workspace: slug, providerId } } },
    { enabled: Boolean(providerId) },
  )
  const [picked, setPicked] = useState<string[]>([])
  const found = (q.data?.items ?? []).filter((s) => !s.imported)
  const imp = $api.useMutation('post', '/api/v1/workspaces/{workspace}/sms/providers/{providerId}/senders', {
    onSuccess: (d) => {
      void refresh()
      toast({ kind: 'success', title: t(m.imported, { count: d.items.length }) })
      onOpenChange(false)
    },
    onError: (e) => toast({ kind: 'error', title: errorMessage(e) }),
  })
  return (
    <Dialog
      open={open}
      onOpenChange={onOpenChange}
      title={t(m.importTitle)}
      description={t(m.importText)}
      size="lg"
      footer={
        <>
          <Button onClick={() => onOpenChange(false)}>{t(m.cancel)}</Button>
          <Button
            variant="primary"
            disabled={!picked.length}
            loading={imp.isPending}
            onClick={() =>
              imp.mutate({
                params: { path: { workspace: slug, providerId } },
                body: { senders: picked.map((value) => ({ value })) },
              })
            }
          >
            {t(m.importDo, { count: picked.length })}
          </Button>
        </>
      }
    >
      <div className="flex flex-col gap-3.5">
        <Field label={t(m.importProvider)}>
          <Select
            value={providerId}
            onChange={(e) => {
              setProviderId(e.target.value)
              setPicked([])
            }}
          >
            {listing.map((p) => (
              <option key={p.id} value={p.id}>
                {p.name}
              </option>
            ))}
          </Select>
        </Field>
        {q.isError ? <Note tone="bad">{errorMessage(q.error)}</Note> : null}
        {q.isPending ? (
          <Loading label={t(m.importLoading, { name: provider?.name ?? '' })}>
            <span className="k">{t(m.importLoading, { name: provider?.name ?? '' })}</span>
          </Loading>
        ) : found.length === 0 ? (
          <span className="k">{t(m.importNone, { name: provider?.name ?? '' })}</span>
        ) : (
          <div className="flex flex-col gap-2">
            {found.map((s) => (
              <label key={s.value} className={`sms-rad${picked.includes(s.value) ? ' on' : ''}`}>
                <input
                  type="checkbox"
                  checked={picked.includes(s.value)}
                  onChange={(e) => setPicked(e.target.checked ? [...picked, s.value] : picked.filter((v) => v !== s.value))}
                />
                <span className="flex min-w-0 flex-1 flex-col gap-1">
                  <span className="flex flex-wrap items-center gap-2">
                    <b className="mono text-[13px] font-semibold">{s.label}</b>
                    <CountryChips codes={s.countries} />
                  </span>
                  <span className="k">{kindName(s.kind)}</span>
                </span>
                <span className="flex flex-col items-end gap-1">
                  <RegistrationBadges
                    row={{
                      id: s.value,
                      value: s.value,
                      label: s.label,
                      kind: s.kind,
                      providerId,
                      provider: provider?.name ?? '',
                      adapter: provider?.adapter ?? '',
                      countries: s.countries,
                      registration: s.registration,
                      throughput: s.throughput,
                      routes: [],
                      held: false,
                      manual: false,
                      canSend: true,
                    }}
                  />
                </span>
              </label>
            ))}
          </div>
        )}
      </div>
    </Dialog>
  )
}
