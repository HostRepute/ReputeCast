import { useQueryClient } from '@tanstack/react-query'
import { useId, useState } from 'react'
import { $api, errorMessage, type Schemas } from '@/api/client'
import { Button } from '@/components/ui/Button'
import { Dialog } from '@/components/ui/Dialog'
import { Note } from '@/components/ui/Note'
import { toast } from '@/components/ui/Toast'
import { copyText, ProviderMark } from '@/features/access/parts'
import { usDate } from '@/lib/format'

const help: Record<string, { where: string; console: string }> = {
  google: { where: 'Google Cloud console → APIs & Services → Credentials → OAuth client ID (Web application)', console: 'https://console.cloud.google.com/apis/credentials' },
  microsoft: { where: 'Microsoft Entra admin center → App registrations → New registration (Web platform)', console: 'https://entra.microsoft.com/#view/Microsoft_AAD_RegisteredApps' },
  github: { where: 'GitHub → Settings → Developer settings → OAuth Apps → New OAuth App', console: 'https://github.com/settings/developers' },
}

/**
 * Google, Microsoft or GitHub sign-in settings: the redirect URI to register,
 * the client ID and the write-only secret, GitHub's organization and
 * Microsoft's tenant. Turning the provider on happens with "Save policies".
 */
export function SocialSettingsDialog({ provider, onClose }: { provider: Schemas['SSOSocialProvider']; onClose: () => void }) {
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
  return (
    <Dialog
      open
      onOpenChange={(o) => !o && !save.isPending && onClose()}
      hideClose
      eyebrow={<span className="k">Dialog · instance admin</span>}
      title={
        <span className="row flex-nowrap gap-2">
          <ProviderMark preset={provider.kind} size={18} /> {provider.name} sign-in
        </span>
      }
      description={
        <span className="text-[13.5px]">
          Signs in people who already have an account (invited); it never creates one. Register ReputeCast in {h.where}.
        </span>
      }
      footer={
        <>
          <Button onClick={onClose} disabled={save.isPending}>
            Cancel
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
                toast({ kind: 'success', title: `${provider.name} sign-in saved`, detail: provider.enabled ? undefined : 'turn it on, then save the policies' })
                onClose()
              } catch (e) {
                setError(errorMessage(e))
              }
            }}
          >
            Save
          </Button>
        </>
      }
    >
      <div className="lbl">
        <span>
          Redirect URI <span className="k font-medium">· add this to the OAuth app</span>
        </span>
        <div className="row flex-nowrap gap-2">
          <input className="in mono flex-1 bg-ground" value={provider.redirectUri} readOnly aria-label="Redirect URI" />
          <Button
            onClick={async () => {
              const ok = await copyText(provider.redirectUri)
              toast(ok ? { kind: 'success', title: 'Redirect URI copied' } : { kind: 'error', title: "Couldn't copy", detail: 'select it and copy' })
            }}
          >
            Copy
          </Button>
        </div>
      </div>
      <div className="grid grid-cols-2 gap-3.5 max-md:grid-cols-1">
        <label className="lbl" htmlFor={`${id}-client`}>
          Client ID
          <input id={`${id}-client`} className="in mono" value={clientId} onChange={(e) => setClientId(e.target.value)} autoComplete="off" spellCheck={false} />
        </label>
        <div className="lbl">
          <label htmlFor={`${id}-secret`}>
            Client secret <span className="k font-medium">· write-only</span>
          </label>
          {replacing ? (
            <input
              id={`${id}-secret`}
              className="in mono"
              type="password"
              value={secret}
              onChange={(e) => setSecret(e.target.value)}
              autoComplete="new-password"
              placeholder={provider.secretSet ? 'New secret' : 'Paste the client secret'}
            />
          ) : (
            <div className="row flex-nowrap gap-2">
              <input id={`${id}-secret`} className="in flex-1 bg-ground" type="password" value="••••••••••••••••••••" readOnly aria-describedby={`${id}-sec-hint`} />
              <Button onClick={() => setReplacing(true)}>Replace</Button>
            </div>
          )}
          <span className="k font-medium" id={`${id}-sec-hint`}>
            {provider.secretSet && provider.secretSetAt ? `Saved ${usDate(provider.secretSetAt)}. ` : ''}Secrets can be replaced but never read back.
          </span>
        </div>
      </div>
      {provider.kind === 'github' ? (
        <label className="lbl" htmlFor={`${id}-org`}>
          <span>
            Organization <span className="k font-medium">· optional</span>
          </span>
          <input id={`${id}-org`} className="in mono max-w-[260px]" value={org} placeholder="acme-eng" onChange={(e) => setOrg(e.target.value)} aria-describedby={`${id}-org-hint`} />
          <span className="k font-medium" id={`${id}-org-hint`}>
            Only active members of this GitHub organization can sign in (ReputeCast asks for read:org to check).
          </span>
        </label>
      ) : null}
      {provider.kind === 'microsoft' ? (
        <label className="lbl" htmlFor={`${id}-tenant`}>
          Accounts
          <select id={`${id}-tenant`} className="in max-w-[320px]" value={['common', 'organizations', 'consumers'].includes(tenant) ? tenant : 'tenant'} onChange={(e) => setTenant(e.target.value === 'tenant' ? '' : e.target.value)}>
            <option value="common">Personal and work accounts</option>
            <option value="organizations">Work and school accounts only</option>
            <option value="consumers">Personal Microsoft accounts only</option>
            <option value="tenant">One tenant (enter its ID)</option>
          </select>
          {!['common', 'organizations', 'consumers'].includes(tenant) ? (
            <input className="in mono max-w-[320px]" value={tenant} placeholder="7c1a2b3d-0e4f-4a5b-9c6d-1e2f3a4b5c6d" onChange={(e) => setTenant(e.target.value)} aria-label="Tenant ID" />
          ) : null}
        </label>
      ) : null}
      {provider.kind === 'microsoft' ? (
        <Note tone="info">
          Microsoft doesn&apos;t verify the email it sends, so a Microsoft sign-in is only linked through an invitation link or Account &amp; security › Link.
        </Note>
      ) : null}
      {error ? <Note tone="bad">{error}</Note> : null}
    </Dialog>
  )
}
