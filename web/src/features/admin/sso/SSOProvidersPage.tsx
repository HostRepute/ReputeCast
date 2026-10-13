import { useNavigate } from '@tanstack/react-router'
import { useEffect, useState } from 'react'
import { $api, errorMessage, type Schemas } from '@/api/client'
import { Button, ButtonLink } from '@/components/ui/Button'
import { ConfirmDialog } from '@/components/ui/ConfirmDialog'
import { EmptyState, ErrorPanel, Skeleton } from '@/components/ui/Feedback'
import { toast } from '@/components/ui/Toast'
import { ProviderMark } from '@/features/access/parts'
import { andList, usDate } from '@/lib/format'
import { useWorkspace } from '@/lib/session'
import { AdminPage } from '../AdminPage'
import { ssoStatusBadge } from '../auth/AuthenticationPage'
import { useInvalidateSSO } from '../queries'
import { GroupMappingNote } from './SSOSetupPage'
import { onTestDone, openTestPopup } from './testSignIn'

type Item = Schemas['SSOProviderItem']

const cards = 'grid items-start gap-5 [&>*]:min-w-0 grid-cols-[repeat(auto-fit,minmax(min(560px,100%),1fr))] max-md:grid-cols-1'

/** Instance admin › SSO providers: each provider at a glance (SSOStates). */
export function SSOProvidersPage() {
  const { slug } = useWorkspace()
  const q = $api.useQuery('get', '/api/v1/admin/sso')
  return (
    <AdminPage documentTitle="SSO providers">
      <div className="row justify-between gap-3">
        <span className="k">Configured at instance level · presets for Google Workspace, Microsoft Entra ID, Okta, Keycloak, Authentik and generic OIDC</span>
        <ButtonLink size="sm" variant="primary" to="/w/$workspace/admin/sso/$providerId" params={{ workspace: slug, providerId: 'new' }}>
          Add SSO provider
        </ButtonLink>
      </div>
      {q.isError ? (
        <ErrorPanel title="Couldn't load the SSO providers" actions={<Button onClick={() => q.refetch()}>Retry</Button>}>
          {errorMessage(q.error)} Sign-in keeps working with the saved providers.
        </ErrorPanel>
      ) : q.isPending ? (
        <div className={cards} aria-busy="true">
          <span className="vh" role="status">
            Loading SSO providers
          </span>
          {[0, 1].map((i) => (
            <section key={i} className="card">
              <div className="sec">
                <Skeleton w={200} h={16} />
              </div>
              <div className="pad col gap-2.5">
                <Skeleton w="90%" />
                <Skeleton w="80%" />
                <Skeleton w="70%" />
              </div>
            </section>
          ))}
        </div>
      ) : q.data.items.length === 0 ? (
        <EmptyState
          title="No SSO provider yet"
          action={
            <ButtonLink variant="primary" to="/w/$workspace/admin/sso/$providerId" params={{ workspace: slug, providerId: 'new' }}>
              Add SSO provider
            </ButtonLink>
          }
        >
          Connect your organization&apos;s identity provider over OpenID Connect: people sign in with it, accounts can be created automatically and groups can map to roles.
        </EmptyState>
      ) : (
        <div className={cards}>
          {q.data.items.map((p) => (
            <ProviderCard key={p.id} p={p} />
          ))}
        </div>
      )}
    </AdminPage>
  )
}

/** One provider: the summary of SSOStates' Google Workspace card. */
function ProviderCard({ p }: { p: Item }) {
  const { slug } = useWorkspace()
  const navigate = useNavigate()
  const invalidate = useInvalidateSSO()
  const test = $api.useMutation('post', '/api/v1/admin/sso/{providerId}/test')
  const [disabling, setDisabling] = useState(false)
  const [deleting, setDeleting] = useState(false)
  const disable = $api.useMutation('post', '/api/v1/admin/sso/{providerId}/disable')
  const enable = $api.useMutation('post', '/api/v1/admin/sso/{providerId}/enable')
  const del = $api.useMutation('delete', '/api/v1/admin/sso/{providerId}')
  // A finished test opens the provider's page, where the claims are.
  useEffect(
    () => onTestDone(p.id, () => void navigate({ to: '/w/$workspace/admin/sso/$providerId', params: { workspace: slug, providerId: p.id } })),
    [navigate, p.id, slug],
  )
  const caption = [p.domains.join(', ') || 'any domain', p.status === 'enabled' ? `enabled ${p.enabledAt ? usDate(p.enabledAt) : ''}`.trim() : p.status === 'draft' ? 'setup in progress' : 'disabled']
    .filter(Boolean)
    .join(' · ')
  const enforced = p.enforced.length
    ? `Yes · password off for ${p.enforced.join(', ')}${p.breakGlass.length ? `, except break-glass ${p.breakGlass.length === 1 ? 'Owner' : 'Owners'} ${andList(p.breakGlass)}` : ''}`
    : 'No · password sign-in stays on'
  return (
    <section className="card" aria-labelledby={`sso-${p.id}`}>
      <div className="sec">
        <div className="row flex-nowrap gap-2.5">
          <ProviderMark preset={p.preset} size={16} />
          <div className="col gap-0.5">
            <h2 className="h2" id={`sso-${p.id}`}>
              {p.name}
            </h2>
            <span className="k">{caption}</span>
          </div>
        </div>
        {ssoStatusBadge(p.status)}
      </div>
      <div className="pad col gap-3">
        <dl className="kv">
          <dt>Issuer</dt>
          <dd className="mono">{p.issuer || '–'}</dd>
          <dt>Allowed domains</dt>
          <dd>{p.domains.join(', ') || 'Any domain'}</dd>
          <dt>JIT</dt>
          <dd>{p.jit.enabled ? `On · ${p.jit.workspaceName} · ${p.jit.roleName}` : 'Off · only existing accounts sign in'}</dd>
          {p.groupsClaim ? (
            <>
              <dt>Group mapping</dt>
              <dd>{p.mappings ? `${p.mappings} ${p.mappings === 1 ? 'mapping' : 'mappings'}` : 'None yet'}</dd>
            </>
          ) : null}
          <dt>Enforced</dt>
          <dd>{enforced}</dd>
          <dt>Two-factor</dt>
          <dd>{p.twoFactor}</dd>
          {p.status !== 'enabled' && p.discoveryOk === false ? (
            <>
              <dt>Discovery</dt>
              <dd className="text-bad-fg">Failed · open the setup to fix the issuer</dd>
            </>
          ) : null}
        </dl>
        {!p.groupsClaim ? <GroupMappingNote preset={p.preset} name={p.name} /> : null}
        <div className="row gap-2">
          {p.status === 'enabled' ? (
            <>
              <Button
                size="sm"
                loading={test.isPending}
                onClick={async () => {
                  try {
                    const r = await test.mutateAsync({ params: { path: { providerId: p.id } }, body: { workspace: slug } })
                    openTestPopup(r.authorizeUrl)
                  } catch (e) {
                    toast({ kind: 'error', title: "Couldn't start the test sign-in", detail: errorMessage(e) })
                  }
                }}
              >
                Test sign-in
              </Button>
              <ButtonLink size="sm" to="/w/$workspace/admin/sso/$providerId" params={{ workspace: slug, providerId: p.id }}>
                Edit
              </ButtonLink>
              <Button size="sm" variant="danger" onClick={() => setDisabling(true)}>
                Disable
              </Button>
            </>
          ) : (
            <>
              <ButtonLink size="sm" variant={p.status === 'draft' ? 'primary' : 'default'} to="/w/$workspace/admin/sso/$providerId" params={{ workspace: slug, providerId: p.id }}>
                {p.status === 'draft' ? 'Continue setup' : 'Edit'}
              </ButtonLink>
              {p.status === 'disabled' ? (
                <Button
                  size="sm"
                  loading={enable.isPending}
                  onClick={async () => {
                    try {
                      await enable.mutateAsync({ params: { path: { providerId: p.id } } })
                      await invalidate()
                      toast({ kind: 'success', title: `${p.name} enabled`, detail: 'its button is on the sign-in page' })
                    } catch (e) {
                      toast({ kind: 'error', title: `Couldn't enable ${p.name}`, detail: errorMessage(e) })
                    }
                  }}
                >
                  Enable
                </Button>
              ) : null}
              <Button size="sm" variant="danger" onClick={() => setDeleting(true)}>
                Delete
              </Button>
            </>
          )}
        </div>
      </div>
      {disabling ? (
        <ConfirmDialog
          open
          onOpenChange={setDisabling}
          badge="Disable SSO provider"
          title={`Disable ${p.name}?`}
          description={`Its button leaves the sign-in page and ${p.members} ${p.members === 1 ? 'person' : 'people'} who sign in with it can't until it's enabled again${p.enforced.length ? `; enforcement for ${p.enforced.join(', ')} ends and password sign-in comes back there` : ''}. Settings, mappings and linked identities are kept.`}
          confirmLabel="Disable"
          onConfirm={async () => {
            await disable.mutateAsync({ params: { path: { providerId: p.id } } })
            await invalidate()
            toast({ kind: 'success', title: `${p.name} disabled` })
          }}
        />
      ) : null}
      {deleting ? (
        <ConfirmDialog
          open
          onOpenChange={setDeleting}
          badge="Delete SSO provider"
          title={`Delete ${p.name}?`}
          description={`Deletes the settings, the client secret and ${p.members} linked ${p.members === 1 ? 'identity' : 'identities'}. Accounts stay; people sign in another way.`}
          confirmLabel="Delete provider"
          typedConfirmation={{ label: `Type ${p.name} to confirm`, expected: p.name }}
          onConfirm={async () => {
            await del.mutateAsync({ params: { path: { providerId: p.id } } })
            await invalidate()
            toast({ kind: 'deleted', title: `${p.name} deleted` })
          }}
        />
      ) : null}
    </section>
  )
}
