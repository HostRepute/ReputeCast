import { useQueryClient } from '@tanstack/react-query'
import { useId, useState, type ReactNode } from 'react'
import { $api, errorMessage, type Schemas } from '@/api/client'
import { Button } from '@/components/ui/Button'
import { Dialog } from '@/components/ui/Dialog'
import { Note } from '@/components/ui/Note'
import { toast } from '@/components/ui/Toast'
import { copyText, ProviderMark } from '@/features/access/parts'
import { SeedAppNote } from '@/features/placement/EntryPoints'
import { T, useT } from '@/i18n'
import { common } from '@/i18n/common'
import { usDate } from '@/lib/format'
import { m as am } from './messages'

// Where to register the OAuth app: the consoles' own menu names, in English.
const help: Record<string, { where: string; console: string }> = {
  google: { where: 'Google Cloud console → APIs & Services → Credentials → OAuth client ID (Web application)', console: 'https://console.cloud.google.com/apis/credentials' },
  microsoft: { where: 'Microsoft Entra admin center → App registrations → New registration (Web platform)', console: 'https://entra.microsoft.com/#view/Microsoft_AAD_RegisteredApps' },
  github: { where: 'GitHub → Settings → Developer settings → OAuth Apps → New OAuth App', console: 'https://github.com/settings/developers' },
}

/**
 * Google, Microsoft or GitHub sign-in settings (the secret is write-only).
 * Turning the provider on happens with "Save policies".
 */
export function SocialSettingsDialog({ provider, onClose }: { provider: Schemas['SSOSocialProvider']; onClose: () => void }) {
  const t = useT()
  const id = useId()
  const qc = useQueryClient()
  const save = $api.useMutation('put', '/api/v1/admin/social/{kind}')
  const [clientId, setClientId] = useState(provider.clientId)
  const [secret, setSecret] = useState('')
  const [replacing, setReplacing] = useState(!provider.secretSet)
  const [org, setOrg] = useState(provider.org ?? '')
  const [tenant, setTenant] = useState(provider.tenant || 'common')
  const [error, setError] = useState<string>()
  const h = help[provider.kind]!
  const k = (c: ReactNode) => <span className="k font-medium">{c}</span>
  return (
    <Dialog
      open
      onOpenChange={(o) => !o && !save.isPending && onClose()}
      hideClose
      eyebrow={<span className="k">{t(am.socialEyebrow)}</span>}
      title={
        <span className="row flex-nowrap gap-2">
          <ProviderMark preset={provider.kind} size={18} /> {t(am.socialTitle, { name: provider.name })}
        </span>
      }
      description={<span className="text-[13.5px]">{t(am.socialBody, { where: h.where })}</span>}
      footer={
        <>
          <Button onClick={onClose} disabled={save.isPending}>
            {t(common.cancel)}
          </Button>
          <Button
            variant="primary"
            loading={save.isPending}
            onClick={async () => {
              setError(undefined)
              try {
                await save.mutateAsync({
                  params: { path: { kind: provider.kind } },
                  body: {
                    enabled: provider.enabled && Boolean(clientId.trim()),
                    clientId: clientId.trim(),
                    clientSecret: replacing && secret.trim() ? secret.trim() : undefined,
                    org: provider.kind === 'github' ? org.trim() : undefined,
                    tenant: provider.kind === 'microsoft' ? tenant.trim() : undefined,
                  },
                })
                await Promise.all([qc.invalidateQueries({ queryKey: ['get', '/api/v1/admin/policy'] }), qc.invalidateQueries({ queryKey: ['meta'] })])
                toast({ kind: 'success', title: t(am.socialSaved, { name: provider.name }), detail: provider.enabled ? undefined : t(am.socialSavedDetail) })
                onClose()
              } catch (e) {
                setError(errorMessage(e))
              }
            }}
          >
            {t(common.save)}
          </Button>
        </>
      }
    >
      <div className="lbl">
        <span>
          <T msg={am.redirectLabel} values={{ k }} />
        </span>
        <div className="row flex-nowrap gap-2">
          <input className="in mono flex-1 bg-ground" value={provider.redirectUri} readOnly aria-label={t(am.redirectUri)} />
          <Button
            onClick={async () => {
              const ok = await copyText(provider.redirectUri)
              toast(ok ? { kind: 'success', title: t(am.redirectCopied) } : { kind: 'error', title: t(am.copyFailed), detail: t(am.copyFailedDetail) })
            }}
          >
            {t(common.copy)}
          </Button>
        </div>
      </div>
      <div className="grid grid-cols-2 gap-3.5 max-md:grid-cols-1">
        <label className="lbl" htmlFor={`${id}-client`}>
          {t(am.clientId)}
          <input id={`${id}-client`} className="in mono" value={clientId} onChange={(e) => setClientId(e.target.value)} autoComplete="off" spellCheck={false} />
        </label>
        <div className="lbl">
          <label htmlFor={`${id}-secret`}>
            <T msg={am.clientSecretLabel} values={{ k }} />
          </label>
          {replacing ? (
            <input
              id={`${id}-secret`}
              className="in mono"
              type="password"
              value={secret}
              onChange={(e) => setSecret(e.target.value)}
              autoComplete="new-password"
              placeholder={provider.secretSet ? t(am.newSecret) : t(am.pasteSecret)}
            />
          ) : (
            <div className="row flex-nowrap gap-2">
              <input id={`${id}-secret`} className="in flex-1 bg-ground" type="password" value="••••••••••••••••••••" readOnly aria-describedby={`${id}-sec-hint`} />
              <Button onClick={() => setReplacing(true)}>{t(am.replaceSecret)}</Button>
            </div>
          )}
          <span className="k font-medium" id={`${id}-sec-hint`}>
            {t(am.secretHint, { saved: provider.secretSet && provider.secretSetAt ? 'yes' : 'no', date: provider.secretSetAt ? usDate(provider.secretSetAt) : '' })}
          </span>
        </div>
      </div>
      {provider.kind === 'github' ? (
        <label className="lbl" htmlFor={`${id}-org`}>
          <span>
            <T msg={am.orgLabel} values={{ k }} />
          </span>
          <input id={`${id}-org`} className="in mono max-w-[260px]" value={org} placeholder={'acme-eng'} onChange={(e) => setOrg(e.target.value)} aria-describedby={`${id}-org-hint`} />
          <span className="k font-medium" id={`${id}-org-hint`}>
            {t(am.orgHint)}
          </span>
        </label>
      ) : null}
      {provider.kind === 'microsoft' ? (
        <label className="lbl" htmlFor={`${id}-tenant`}>
          {t(am.accounts)}
          <select id={`${id}-tenant`} className="in max-w-[320px]" value={['common', 'organizations', 'consumers'].includes(tenant) ? tenant : 'tenant'} onChange={(e) => setTenant(e.target.value === 'tenant' ? '' : e.target.value)}>
            <option value="common">{t(am.tenantCommon)}</option>
            <option value="organizations">{t(am.tenantOrganizations)}</option>
            <option value="consumers">{t(am.tenantConsumers)}</option>
            <option value="tenant">{t(am.tenantOne)}</option>
          </select>
          {!['common', 'organizations', 'consumers'].includes(tenant) ? (
            <input className="in mono max-w-[320px]" value={tenant} placeholder={'7c1a2b3d-0e4f-4a5b-9c6d-1e2f3a4b5c6d'} onChange={(e) => setTenant(e.target.value)} aria-label={t(am.tenantId)} />
          ) : null}
        </label>
      ) : null}
      {provider.kind === 'microsoft' ? <Note tone="info">{t(am.microsoftNote)}</Note> : null}
      <SeedAppNote kind={provider.kind} signInRedirect={provider.redirectUri} />
      {error ? <Note tone="bad">{error}</Note> : null}
    </Dialog>
  )
}
