import { useState } from 'react'
import { $api, errorMessage, type Schemas } from '@/api/client'
import { Badge } from '@/components/ui/Badge'
import { Button } from '@/components/ui/Button'
import { ConfirmDialog } from '@/components/ui/ConfirmDialog'
import { Dialog } from '@/components/ui/Dialog'
import { Field, Input, Select } from '@/components/ui/Field'
import { Note } from '@/components/ui/Note'
import { toast } from '@/components/ui/Toast'
import { useT } from '@/i18n'
import { dayMonthYear } from '@/lib/format'
import { usePermissionReason, useWorkspace } from '@/lib/session'
import { Capabilities, CopyButton, useRefreshSms } from './common'
import { m } from './dialogs.messages'
import { m as sm } from './messages'

type Card = Schemas['SmsProviderCard']

const statusWord = { healthy: sm.statusHealthy, degraded: sm.statusDegraded, down: sm.statusDown, credentials_rejected: sm.statusRejected } as const

/**
 * An SMS provider's settings: replace the write-only credentials, register
 * the webhooks again, check it now, remove it.
 */
export function ProviderDialog({ card, focus, open, onOpenChange }: { card: Card; focus?: 'credentials'; open: boolean; onOpenChange: (o: boolean) => void }) {
  const t = useT()
  const { slug } = useWorkspace()
  const refresh = useRefreshSms()
  const reason = usePermissionReason('providers:manage')
  const catalog = $api.useQuery('get', '/api/v1/workspaces/{workspace}/sms/catalog', { params: { path: { workspace: slug } } }, { staleTime: 300_000 })
  const fields = catalog.data?.items.find((i) => i.adapter === card.adapter)?.fields ?? []
  const [values, setValues] = useState<Record<string, string>>({})
  const [error, setError] = useState<string>()
  const [removing, setRemoving] = useState(false)
  const path = { workspace: slug, providerId: card.id }

  const creds = $api.useMutation('put', '/api/v1/workspaces/{workspace}/sms/providers/{providerId}/credentials', {
    onSuccess: () => {
      void refresh()
      setValues({})
      setError(undefined)
      toast({ kind: 'success', title: t(m.replaced, { name: card.name }), detail: t(m.replacedDetail) })
    },
    onError: (e) => setError(errorMessage(e)),
  })
  const hooks = $api.useMutation('post', '/api/v1/workspaces/{workspace}/sms/providers/{providerId}/webhooks', {
    onSuccess: () => {
      void refresh()
      toast({ kind: 'success', title: t(m.registered, { name: card.name }) })
    },
    onError: (e) => toast({ kind: 'error', title: errorMessage(e) }),
  })
  const check = $api.useMutation('post', '/api/v1/workspaces/{workspace}/sms/providers/{providerId}/check', {
    onSuccess: (d) => {
      void refresh()
      const now = d.providers.find((p) => p.id === card.id)
      toast({ kind: 'info', title: t(m.checked, { name: card.name, status: now ? t(statusWord[now.status]) : '' }) })
    },
    onError: (e) => toast({ kind: 'error', title: errorMessage(e) }),
  })
  const remove = $api.useMutation('delete', '/api/v1/workspaces/{workspace}/sms/providers/{providerId}')

  const submit = () => {
    const filled = Object.fromEntries(Object.entries(values).filter(([, v]) => v.trim() !== ''))
    creds.mutate({ params: { path }, body: { fields: filled } })
  }

  return (
    <>
      <Dialog
        open={open && !removing}
        onOpenChange={onOpenChange}
        title={t(m.settingsTitle, { name: card.name })}
        description={t(m.settingsSub, { account: card.accountLabel, date: dayMonthYear(card.connectedAt) })}
        size="lg"
        footer={
          <>
            <Button variant="danger" className="mr-auto max-md:mr-0" disabledReason={reason} onClick={() => setRemoving(true)}>
              {t(m.removeProvider)}
            </Button>
            <Button onClick={() => onOpenChange(false)}>{t(m.close)}</Button>
          </>
        }
      >
        <div className="flex flex-col gap-5">
          <div className="flex flex-wrap items-center gap-2">
            <Badge tone={card.status === 'healthy' ? 'ok' : card.status === 'degraded' ? 'warn' : 'bad'} dot>
              {t(statusWord[card.status])}
            </Badge>
            <Button size="sm" loading={check.isPending} onClick={() => check.mutate({ params: { path } })}>
              {t(m.checkNow)}
            </Button>
          </div>
          <Capabilities badges={card.badges} />
          <section className="flex flex-col gap-3">
            <h3 className="h2">{t(m.credentials)}</h3>
            <span className="k">{t(m.credentialsHint)}</span>
            {reason ? <Note tone="neutral">{t(m.readOnly)}</Note> : null}
            {error ? <Note tone="bad">{error}</Note> : null}
            {fields.map((f, i) =>
              f.options?.length ? (
                <Field key={f.key} label={f.label} hint={f.help}>
                  <Select value={values[f.key] ?? ''} disabled={Boolean(reason)} onChange={(e) => setValues({ ...values, [f.key]: e.target.value })}>
                    <option value="">{t(m.keepPlaceholder)}</option>
                    {f.options.map((o) => (
                      <option key={o} value={o}>
                        {o}
                      </option>
                    ))}
                  </Select>
                </Field>
              ) : (
                <Field key={f.key} label={f.label} hint={f.help}>
                  <Input
                    type={f.secret ? 'password' : 'text'}
                    autoComplete="off"
                    autoFocus={focus === 'credentials' && i === 0}
                    disabled={Boolean(reason)}
                    placeholder={f.secret ? t(m.secretPlaceholder) : t(m.keepPlaceholder)}
                    value={values[f.key] ?? ''}
                    onChange={(e) => setValues({ ...values, [f.key]: e.target.value })}
                  />
                </Field>
              ),
            )}
            <div>
              <Button variant="primary" size="sm" disabledReason={reason} loading={creds.isPending} onClick={submit}>
                {t(m.replace)}
              </Button>
            </div>
          </section>
          <section className="flex flex-col gap-2">
            <h3 className="h2">{t(m.webhooks)}</h3>
            <span className="k">{card.webhooks === 'registered' ? t(m.webhooksRegistered, { name: card.name }) : t(m.webhooksManual, { name: card.name })}</span>
            <div className="flex flex-wrap items-center gap-2">
              <span className="mono min-w-0 flex-1 text-[12px] [overflow-wrap:anywhere]">{card.webhookUrl}</span>
              <CopyButton value={card.webhookUrl} label={t(m.webhooks)} />
              <Button size="sm" disabledReason={reason} loading={hooks.isPending} onClick={() => hooks.mutate({ params: { path } })}>
                {t(m.registerAgain)}
              </Button>
            </div>
          </section>
        </div>
      </Dialog>
      <ConfirmDialog
        open={removing}
        onOpenChange={(o) => {
          setRemoving(o)
        }}
        badge={t(m.removeProvider)}
        title={t(m.removeTitle, { name: card.name })}
        description={t(m.removeText)}
        confirmLabel={t(m.removeProvider)}
        onConfirm={async () => {
          await remove.mutateAsync({ params: { path } })
          void refresh()
          toast({ kind: 'deleted', title: t(m.removed, { name: card.name }) })
          setRemoving(false)
          onOpenChange(false)
        }}
      />
    </>
  )
}
