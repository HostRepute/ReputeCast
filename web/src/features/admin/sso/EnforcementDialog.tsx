import { Fragment, useState, type ReactNode } from 'react'
import { $api, errorMessage, type Schemas } from '@/api/client'
import { Button } from '@/components/ui/Button'
import { Dialog } from '@/components/ui/Dialog'
import { Skeleton } from '@/components/ui/Feedback'
import { Note } from '@/components/ui/Note'
import { Td, Th } from '@/components/ui/Table'
import { toast } from '@/components/ui/Toast'
import { AccessTable, PersonCell } from '@/features/access/parts'
import { T, useT } from '@/i18n'
import { common } from '@/i18n/common'
import { andList } from '@/lib/format'
import { useInvalidateSSO } from '../queries'
import { m as sm } from './messages'

type Detail = Schemas['SSOProviderDetail']

/** "A, B and C" with nodes, in the app language. */
function andNodes(nodes: ReactNode[]): ReactNode {
  const words = andList(nodes.map((_, i) => `§${i}§`))
  return words.split(/§(\d+)§/).map((part, i) => (i % 2 === 1 ? <Fragment key={i}>{nodes[Number(part)]}</Fragment> : part))
}

/**
 * Turning on domain enforcement (SSOStates): who signs in how today and
 * after, the break-glass Owners who keep their password, and who will be
 * sent to the IdP at their next sign-in.
 */
export function EnforcementDialog({ d, onClose, onDone }: { d: Detail; onClose: () => void; onDone: (d: Detail) => void }) {
  const t = useT()
  const domains = d.domains
  const preview = $api.useQuery('get', '/api/v1/admin/sso/{providerId}/enforcement', { params: { path: { providerId: d.id }, query: { domain: domains } } })
  const set = $api.useMutation('put', '/api/v1/admin/sso/{providerId}/enforcement')
  const invalidate = useInvalidateSSO()
  const [error, setError] = useState<string>()
  const p = preview.data
  const list = domains.join(', ')
  const unlinked = p?.members.filter((m) => m.unlinked && !m.breakGlass) ?? []
  return (
    <Dialog
      open
      onOpenChange={(o) => !o && !set.isPending && onClose()}
      hideClose
      dismissible={false}
      size="lg"
      eyebrow={<span className="k">{t(sm.enforceEyebrow)}</span>}
      title={t(sm.enforceTitle, { provider: d.name, domains: list })}
      description={<span className="text-[13.5px]">{t(sm.enforceBody, { domains: list })}</span>}
      footer={
        <>
          <Button onClick={onClose} disabled={set.isPending}>
            {t(common.cancel)}
          </Button>
          <Button
            variant="primary"
            loading={set.isPending}
            disabledReason={!p ? t(sm.enforceLoading) : p.conflict ? t(sm.enforceConflictReason, { provider: p.conflict }) : undefined}
            onClick={async () => {
              setError(undefined)
              try {
                const r = await set.mutateAsync({ params: { path: { providerId: d.id } }, body: { domains, confirm: true } })
                onDone(r)
                await invalidate()
                toast({ kind: 'success', title: t(sm.enforcedToast, { provider: d.name, domains: list }), detail: t(sm.enforcedDetail) })
                onClose()
              } catch (e) {
                setError(errorMessage(e))
              }
            }}
          >
            {t(sm.enforceConfirm)}
          </Button>
        </>
      }
    >
      {preview.isError ? (
        <Note tone="bad">{errorMessage(preview.error)}</Note>
      ) : !p ? (
        <div className="col gap-2" aria-busy="true">
          <Skeleton w="100%" h={40} />
          <Skeleton w="100%" h={40} />
          <Skeleton w="100%" h={40} />
        </div>
      ) : (
        <>
          {p.members.length > 0 ? (
            <AccessTable caption={t(sm.enforceTable)}>
              <thead>
                <tr>
                  <Th>{t(sm.colMember)}</Th>
                  <Th>{t(sm.colToday)}</Th>
                  <Th>{t(sm.colAfter)}</Th>
                </tr>
              </thead>
              <tbody>
                {p.members.map((m) => (
                  <tr key={m.id}>
                    <Td>
                      <PersonCell name={m.name} sub={m.email} color={m.avatarColor} />
                    </Td>
                    <Td>{m.today}</Td>
                    <Td>{m.after}</Td>
                  </tr>
                ))}
              </tbody>
            </AccessTable>
          ) : (
            <Note tone="neutral">{t(sm.enforceNobody, { domains: list, provider: d.name })}</Note>
          )}
          {p.breakGlass.length > 0 ? (
            <Note tone="ok" label={t(sm.breakGlassLabel)}>
              <T
                msg={sm.breakGlassKeeps}
                values={{
                  owners: <>{andNodes(p.breakGlass.map((o) => <T key={o.email} msg={sm.breakGlassOwner} values={{ name: o.name, workspace: o.workspace, b: (c) => <b>{c}</b> }} />))}</>,
                }}
              />
            </Note>
          ) : (
            <Note tone="warn" label={t(sm.noBreakGlassLabel)}>
              {t(sm.noBreakGlass, { domains: list, provider: d.name })}
            </Note>
          )}
          {unlinked.length > 0 ? (
            <Note tone="warn" label={t(sm.unlinkedLabel, { count: unlinked.length })}>
              <T
                msg={sm.unlinkedBody}
                values={{
                  names: <>{andNodes(unlinked.map((m) => <b key={m.id}>{m.name}</b>))}</>,
                  count: unlinked.length,
                  provider: d.name,
                  short: d.name.split(' ')[0] ?? d.name,
                }}
              />
            </Note>
          ) : null}
          {p.conflict ? <Note tone="bad">{t(sm.enforceConflict, { provider: p.conflict })}</Note> : null}
        </>
      )}
      {error ? <Note tone="bad">{error}</Note> : null}
    </Dialog>
  )
}
