import { useState } from 'react'
import { errorMessage, http, type Schemas } from '@/api/client'
import { Button, ButtonAnchor } from '@/components/ui/Button'
import { CardSection } from '@/components/ui/Card'
import { Dialog } from '@/components/ui/Dialog'
import { Note } from '@/components/ui/Note'
import { Table, Td, Th } from '@/components/ui/Table'
import { toast } from '@/components/ui/Toast'
import { PasskeyGlyph, SocialMark } from '@/features/auth/AuthParts'
import { passkeyErrorMessage, registerPasskey } from '@/features/auth/passkey'
import { useRefreshAccount } from './AccountPage'

type Account = Schemas['AccountOverview']
type Identity = Schemas['AccountIdentity']

function Status({ i }: { i: Identity }) {
  if (!i.linked) return <span className="k">Not linked</span>
  const notes = i.notes.map((n) => ` · ${n}`).join('')
  const main = i.username ? (
    <span>
      Linked as <span className="mono">{i.username}</span>
      {notes}
    </span>
  ) : (
    <span>
      {i.email}
      {notes}
    </span>
  )
  if (!i.onlyMethod || i.kind !== 'sso') return main
  return (
    <div className="col" style={{ gap: 2 }}>
      {main}
      <span className="k">This is your only sign-in method. Add a passkey before unlinking.</span>
    </div>
  )
}

/** Google, Microsoft, GitHub and SSO identities; the last way in can't be unlinked. */
export function ConnectedSignInsCard({ a }: { a: Account }) {
  const refresh = useRefreshAccount()
  const [unlink, setUnlink] = useState<Identity>()
  const [adding, setAdding] = useState(false)
  const ssoOnly = a.password.ssoOnly
  const showPasskeyRow = !a.password.set && a.passkeys.length === 0
  const table = (
    <Table density="compact" className="[&_td]:whitespace-normal" caption="Connected sign-ins">
      <thead>
        <tr>
          <Th>Method</Th>
          <Th>Status</Th>
          <Th>
            <span className="vh">Actions</span>
          </Th>
        </tr>
      </thead>
      <tbody>
        {a.identities.map((i) => (
          <tr key={i.id ?? i.kind}>
            <Td>
              <div className="row flex-nowrap">
                <SocialMark provider={i.kind === 'sso' ? (i.preset === 'google' ? 'google' : 'sso') : i.kind} />
                <span className="font-semibold">{i.name}</span>
              </div>
            </Td>
            <Td wrap={i.linked}>
              <Status i={i} />
            </Td>
            <Td>
              {!i.linked ? (
                i.linkUrl ? (
                  <ButtonAnchor size="sm" href={i.linkUrl} newTab={false} aria-label={`Link ${i.name}`}>
                    Link
                  </ButtonAnchor>
                ) : null
              ) : i.managed ? (
                <Button size="sm" disabledReason={i.unlinkReason}>
                  Managed
                </Button>
              ) : (
                <Button size="sm" disabledReason={i.canUnlink ? undefined : i.unlinkReason} onClick={() => setUnlink(i)} aria-label={`Unlink ${i.name}`}>
                  Unlink
                </Button>
              )}
            </Td>
          </tr>
        ))}
        {showPasskeyRow ? (
          <tr>
            <Td>
              <div className="row flex-nowrap">
                <PasskeyGlyph />
                <span className="font-semibold">Passkeys</span>
              </div>
            </Td>
            <Td>
              <span className="k">None yet</span>
            </Td>
            <Td>
              <Button
                size="sm"
                variant="primary"
                loading={adding}
                onClick={async () => {
                  setAdding(true)
                  try {
                    const pk = await registerPasskey()
                    await refresh()
                    toast({ kind: 'success', title: 'Passkey added', detail: pk.name })
                  } catch (e) {
                    toast({ kind: 'error', title: 'No passkey added', detail: passkeyErrorMessage(e, 'add') })
                  } finally {
                    setAdding(false)
                  }
                }}
              >
                Add a passkey
              </Button>
            </Td>
          </tr>
        ) : null}
        {a.identities.length === 0 && !showPasskeyRow ? (
          <tr>
            <Td colSpan={3}>
              <span className="k">No social or single sign-on accounts are available on this instance.</span>
            </Td>
          </tr>
        ) : null}
      </tbody>
    </Table>
  )
  return (
    <section className="card" aria-labelledby="h-signins">
      <CardSection id="h-signins" title="Connected sign-ins" caption="Linking lets you sign in with these; your last remaining method can't be unlinked" />
      {ssoOnly ? (
        <div className="pad col" style={{ gap: 12 }}>
          <Note tone="info" label="SSO">
            {a.profile.email} signs in through {a.password.managedBy}. There is no ReputeCast password for this account, so the Password section is hidden.
          </Note>
          {table}
          <Note tone="neutral">
            Two-factor: <b>via IdP</b>. {a.password.managedBy} enforces it; ReputeCast trusts the <span className="mono">amr</span> claim and shows it in the team list.
          </Note>
        </div>
      ) : (
        table
      )}
      <UnlinkDialog identity={unlink} onClose={() => setUnlink(undefined)} onDone={refresh} />
    </section>
  )
}

function UnlinkDialog({ identity, onClose, onDone }: { identity?: Identity; onClose: () => void; onDone: () => Promise<void> }) {
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState<string>()
  return (
    <Dialog
      open={Boolean(identity)}
      onOpenChange={(o) => {
        if (o) return
        setError(undefined)
        onClose()
      }}
      title={`Unlink ${identity?.name ?? ''}?`}
      size="sm"
      footer={
        <>
          <Button onClick={onClose}>Keep it linked</Button>
          <Button
            variant="danger-primary"
            loading={busy}
            onClick={async () => {
              if (!identity?.id) return
              setBusy(true)
              setError(undefined)
              try {
                await http.DELETE('/api/v1/me/identities/{id}', { params: { path: { id: identity.id } } })
                await onDone()
                toast({ kind: 'deleted', title: `${identity.name} unlinked` })
                onClose()
              } catch (e) {
                setError(errorMessage(e))
              } finally {
                setBusy(false)
              }
            }}
          >
            Unlink
          </Button>
        </>
      }
    >
      <p className="text-ink-2">You won't be able to sign in with {identity?.name} until you link it again. Your other sign-in methods stay as they are.</p>
      {error ? (
        <Note tone="bad" label="Not unlinked">
          {error}
        </Note>
      ) : null}
    </Dialog>
  )
}
