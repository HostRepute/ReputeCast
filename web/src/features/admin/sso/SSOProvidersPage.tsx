import { useNavigate } from '@tanstack/react-router'
import { useEffect, useState } from 'react'
import { $api, errorMessage, type Schemas } from '@/api/client'
import { Button, ButtonLink } from '@/components/ui/Button'
import { ConfirmDialog } from '@/components/ui/ConfirmDialog'
import { EmptyState, ErrorPanel, Skeleton } from '@/components/ui/Feedback'
import { toast } from '@/components/ui/Toast'
import { ProviderMark } from '@/features/access/parts'
import { useT } from '@/i18n'
import { common } from '@/i18n/common'
import { andList, usDate } from '@/lib/format'
import { roleName } from '@/lib/roles'
import { useWorkspace } from '@/lib/session'
import { AdminPage } from '../AdminPage'
import { ssoStatusBadge } from '../auth/AuthenticationPage'
import { useInvalidateSSO } from '../queries'
import { m as sm } from './messages'
import { GroupMappingNote } from './SSOSetupPage'
import { onTestDone, openTestPopup } from './testSignIn'

type Item = Schemas['SSOProviderItem']

const cards = 'grid items-start gap-5 [&>*]:min-w-0 grid-cols-[repeat(auto-fit,minmax(min(560px,100%),1fr))] max-md:grid-cols-1'

/** Instance admin › SSO providers: each provider at a glance (SSOStates). */
export function SSOProvidersPage() {
  const t = useT()
  const { slug } = useWorkspace()
  const q = $api.useQuery('get', '/api/v1/admin/sso')
  return (
    <AdminPage documentTitle={t(sm.listTitle)}>
      <div className="row justify-between gap-3">
        <span className="k">{t(sm.listCaption)}</span>
        <ButtonLink size="sm" variant="primary" to="/w/$workspace/admin/sso/$providerId" params={{ workspace: slug, providerId: 'new' }}>
          {t(sm.addProvider)}
        </ButtonLink>
      </div>
      {q.isError ? (
        <ErrorPanel title={t(sm.listLoadFailed)} actions={<Button onClick={() => q.refetch()}>{t(common.retry)}</Button>}>
          {errorMessage(q.error)} {t(sm.listLoadFailedBody)}
        </ErrorPanel>
      ) : q.isPending ? (
        <div className={cards} aria-busy="true">
          <span className="vh" role="status">
            {t(sm.listLoading)}
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
          title={t(sm.emptyTitle)}
          action={
            <ButtonLink variant="primary" to="/w/$workspace/admin/sso/$providerId" params={{ workspace: slug, providerId: 'new' }}>
              {t(sm.addProvider)}
            </ButtonLink>
          }
        >
          {t(sm.emptyBody)}
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
  const t = useT()
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
  const caption = [
    p.domains.join(', ') || t(sm.cardAnyDomain),
    t(sm.cardState, { state: p.status === 'enabled' ? (p.enabledAt ? 'enabledOn' : 'enabled') : p.status, date: p.enabledAt ? usDate(p.enabledAt) : '' }),
  ].join(' · ')
  const enforced = p.enforced.length
    ? t(sm.cardEnforcedYes, { domains: p.enforced.join(', '), count: p.breakGlass.length, names: andList(p.breakGlass) })
    : t(sm.cardEnforcedNo)
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
          <dt>{t(sm.cardIssuer)}</dt>
          <dd className="mono">{p.issuer || '–'}</dd>
          <dt>{t(sm.cardDomains)}</dt>
          <dd>{p.domains.join(', ') || t(sm.cardAnyDomainValue)}</dd>
          <dt>{t(sm.cardJit)}</dt>
          <dd>
            {p.jit.enabled
              ? t(sm.cardJitOn, { workspace: p.jit.workspaceName ?? '', role: roleName({ key: p.jit.roleKey, name: p.jit.roleName ?? '' }) })
              : t(sm.cardJitOff)}
          </dd>
          {p.groupsClaim ? (
            <>
              <dt>{t(sm.cardGroupMapping)}</dt>
              <dd>{t(sm.cardMappings, { count: p.mappings })}</dd>
            </>
          ) : null}
          <dt>{t(sm.cardEnforced)}</dt>
          <dd>{enforced}</dd>
          <dt>{t(sm.cardTwoFactor)}</dt>
          <dd>{p.twoFactor}</dd>
          {p.status !== 'enabled' && p.discoveryOk === false ? (
            <>
              <dt>{t(sm.cardDiscovery)}</dt>
              <dd className="text-bad-fg">{t(sm.cardDiscoveryFailed)}</dd>
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
                    toast({ kind: 'error', title: t(sm.testStartFailed), detail: errorMessage(e) })
                  }
                }}
              >
                {t(sm.testSignIn)}
              </Button>
              <ButtonLink size="sm" to="/w/$workspace/admin/sso/$providerId" params={{ workspace: slug, providerId: p.id }}>
                {t(common.edit)}
              </ButtonLink>
              <Button size="sm" variant="danger" onClick={() => setDisabling(true)}>
                {t(sm.disable)}
              </Button>
            </>
          ) : (
            <>
              <ButtonLink size="sm" variant={p.status === 'draft' ? 'primary' : 'default'} to="/w/$workspace/admin/sso/$providerId" params={{ workspace: slug, providerId: p.id }}>
                {p.status === 'draft' ? t(sm.continueSetup) : t(common.edit)}
              </ButtonLink>
              {p.status === 'disabled' ? (
                <Button
                  size="sm"
                  loading={enable.isPending}
                  onClick={async () => {
                    try {
                      await enable.mutateAsync({ params: { path: { providerId: p.id } } })
                      await invalidate()
                      toast({ kind: 'success', title: t(sm.enabledToast, { name: p.name }), detail: t(sm.enabledDetail) })
                    } catch (e) {
                      toast({ kind: 'error', title: t(sm.enableFailed, { name: p.name }), detail: errorMessage(e) })
                    }
                  }}
                >
                  {t(sm.enable)}
                </Button>
              ) : null}
              <Button size="sm" variant="danger" onClick={() => setDeleting(true)}>
                {t(common.delete)}
              </Button>
            </>
          )}
        </div>
      </div>
      {disabling ? (
        <ConfirmDialog
          open
          onOpenChange={setDisabling}
          badge={t(sm.disableBadge)}
          title={t(sm.disableTitle, { name: p.name })}
          description={t(sm.disableBody, { count: p.members, enforced: p.enforced.length ? 'yes' : 'no', domains: p.enforced.join(', ') })}
          confirmLabel={t(sm.disable)}
          onConfirm={async () => {
            await disable.mutateAsync({ params: { path: { providerId: p.id } } })
            await invalidate()
            toast({ kind: 'success', title: t(sm.disabledToast, { name: p.name }) })
          }}
        />
      ) : null}
      {deleting ? (
        <ConfirmDialog
          open
          onOpenChange={setDeleting}
          badge={t(sm.deleteBadge)}
          title={t(sm.deleteTitle, { name: p.name })}
          description={t(sm.deleteBody, { count: p.members })}
          confirmLabel={t(sm.deleteConfirm)}
          typedConfirmation={{ label: t(sm.deleteTyped, { name: p.name }), expected: p.name }}
          onConfirm={async () => {
            await del.mutateAsync({ params: { path: { providerId: p.id } } })
            await invalidate()
            toast({ kind: 'deleted', title: t(sm.deletedToast, { name: p.name }) })
          }}
        />
      ) : null}
    </section>
  )
}
