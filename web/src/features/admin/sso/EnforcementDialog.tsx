import { useState } from 'react'
import { $api, errorMessage, type Schemas } from '@/api/client'
import { Button } from '@/components/ui/Button'
import { Dialog } from '@/components/ui/Dialog'
import { Skeleton } from '@/components/ui/Feedback'
import { Note } from '@/components/ui/Note'
import { Td, Th } from '@/components/ui/Table'
import { toast } from '@/components/ui/Toast'
import { AccessTable, PersonCell } from '@/features/access/parts'
import { useInvalidateSSO } from '../queries'

type Detail = Schemas['SSOProviderDetail']

/**
 * Turning on domain enforcement (SSOStates): who signs in how today and
 * after, the break-glass Owners who keep their password, and who will be
 * sent to the IdP at their next sign-in.
 */
export function EnforcementDialog({ d, onClose, onDone }: { d: Detail; onClose: () => void; onDone: (d: Detail) => void }) {
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
      eyebrow={<span className="k">Dialog · turning on domain enforcement</span>}
      title={`Enforce ${d.name} sign-in for ${list}?`}
      description={
        <span className="text-[13.5px]">
          Password sign-in stops working for {list} accounts at their next sign-in. Passkeys and social sign-ins for this domain are disabled too.
        </span>
      }
      footer={
        <>
          <Button onClick={onClose} disabled={set.isPending}>
            Cancel
          </Button>
          <Button
            variant="primary"
            loading={set.isPending}
            disabledReason={!p ? 'Loading who is affected…' : p.conflict ? `${p.conflict} already enforces this domain` : undefined}
            onClick={async () => {
              setError(undefined)
              try {
                const r = await set.mutateAsync({ params: { path: { providerId: d.id } }, body: { domains, confirm: true } })
                onDone(r)
                await invalidate()
                toast({ kind: 'success', title: `${d.name} is enforced for ${list}`, detail: 'break-glass Owners keep their password' })
                onClose()
              } catch (e) {
                setError(errorMessage(e))
              }
            }}
          >
            Turn on enforcement
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
            <AccessTable caption="Who signs in how">
              <thead>
                <tr>
                  <Th>Member</Th>
                  <Th>Today</Th>
                  <Th>After</Th>
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
            <Note tone="neutral">Nobody has an account at {list} yet. People who sign in later go straight to {d.name}.</Note>
          )}
          {p.breakGlass.length > 0 ? (
            <Note tone="ok" label="Break-glass">
              Keeps password + two-factor access, even with enforcement on:{' '}
              {p.breakGlass.map((o, i) => (
                <span key={o.email}>
                  {i > 0 ? (i === p.breakGlass.length - 1 ? ' and ' : ', ') : null}
                  <b>{o.name}</b> (Owner, {o.workspace})
                </span>
              ))}
              . Their password sign-ins are logged as break-glass.
            </Note>
          ) : (
            <Note tone="warn" label="No break-glass">
              No Owner at {list} keeps a password sign-in. If {d.name} is down, nobody at {list} can sign in; ask an instance admin at another domain to keep access.
            </Note>
          )}
          {unlinked.length > 0 ? (
            <Note tone="warn" label={`${unlinked.length} ${unlinked.length === 1 ? 'member' : 'members'}`}>
              {unlinked.map((m, i) => (
                <span key={m.id}>
                  {i > 0 ? (i === unlinked.length - 1 ? ' and ' : ', ') : null}
                  <b>{m.name}</b>
                </span>
              ))}{' '}
              {unlinked.length === 1 ? 'signs in with a password today and has' : 'sign in with a password today and have'} no {d.name} account linked yet.{' '}
              {unlinked.length === 1 ? 'They will be asked' : 'They will each be asked'} to continue with {d.name.split(' ')[0]} at their next sign-in; their role and history stay.
            </Note>
          ) : null}
          {p.conflict ? <Note tone="bad">{p.conflict} already enforces one of these domains. Turn it off there first.</Note> : null}
        </>
      )}
      {error ? <Note tone="bad">{error}</Note> : null}
    </Dialog>
  )
}
