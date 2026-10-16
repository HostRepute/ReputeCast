import { useState, type ReactNode } from 'react'
import { errorMessage, http, type Schemas } from '@/api/client'
import { Button, ButtonAnchor } from '@/components/ui/Button'
import { CardSection } from '@/components/ui/Card'
import { Dialog } from '@/components/ui/Dialog'
import { Note } from '@/components/ui/Note'
import { Table, Td, Th } from '@/components/ui/Table'
import { toast } from '@/components/ui/Toast'
import { PasskeyGlyph, SocialMark } from '@/features/auth/AuthParts'
import { passkeyErrorMessage, registerPasskey } from '@/features/auth/passkey'
import { T, useT } from '@/i18n'
import { useRefreshAccount } from './AccountPage'
import { m } from './messages'

type Account = Schemas['AccountOverview']
type Identity = Schemas['AccountIdentity']

const mono = (c: ReactNode) => <span className="mono">{c}</span>

function Status({ i }: { i: Identity }) {
  const t = useT()
  if (!i.linked) return <span className="k">{t(m.notLinked)}</span>
  // The notes come from the server, in the member's language: "org acme-eng".
  const notes = i.notes.map((n) => ` · ${n}`).join('')
  const main = i.username ? (
    <span>
      <T msg={m.linkedAs} values={{ username: i.username, mono }} />
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
      <span className="k">{t(m.onlyMethod)}</span>
    </div>
  )
}

/** Google, Microsoft, GitHub and SSO identities; the last way in can't be unlinked. */
export function ConnectedSignInsCard({ a }: { a: Account }) {
  const t = useT()
  const refresh = useRefreshAccount()
  const [unlink, setUnlink] = useState<Identity>()
  const [adding, setAdding] = useState(false)
  const ssoOnly = a.password.ssoOnly
  const showPasskeyRow = !a.password.set && a.passkeys.length === 0
  const table = (
    <Table density="compact" className="[&_td]:whitespace-normal" caption={t(m.connectedSignIns)}>
      <thead>
        <tr>
          <Th>{t(m.method)}</Th>
          <Th>{t(m.status)}</Th>
          <Th>
            <span className="vh">{t(m.actions)}</span>
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
                  <ButtonAnchor size="sm" href={i.linkUrl} newTab={false} aria-label={t(m.linkNamed, { name: i.name })}>
                    {t(m.link)}
                  </ButtonAnchor>
                ) : null
              ) : i.managed ? (
                <Button size="sm" disabledReason={i.unlinkReason}>
                  {t(m.managed)}
                </Button>
              ) : (
                <Button size="sm" disabledReason={i.canUnlink ? undefined : i.unlinkReason} onClick={() => setUnlink(i)} aria-label={t(m.unlinkNamed, { name: i.name })}>
                  {t(m.unlink)}
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
                <span className="font-semibold">{t(m.passkeys)}</span>
              </div>
            </Td>
            <Td>
              <span className="k">{t(m.noneYet)}</span>
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
                    toast({ kind: 'success', title: t(m.passkeyAdded), detail: pk.name })
                  } catch (e) {
                    toast({ kind: 'error', title: t(m.noPasskeyAdded), detail: passkeyErrorMessage(e, 'add') })
                  } finally {
                    setAdding(false)
                  }
                }}
              >
                {t(m.addPasskey)}
              </Button>
            </Td>
          </tr>
        ) : null}
        {a.identities.length === 0 && !showPasskeyRow ? (
          <tr>
            <Td colSpan={3}>
              <span className="k">{t(m.noIdentities)}</span>
            </Td>
          </tr>
        ) : null}
      </tbody>
    </Table>
  )
  return (
    <section className="card" aria-labelledby="h-signins">
      <CardSection id="h-signins" title={t(m.connectedSignIns)} caption={t(m.signInsCaption)} />
      {ssoOnly ? (
        <div className="pad col" style={{ gap: 12 }}>
          <Note tone="info" label={t(m.sso)}>
            {t(m.ssoOnlyNote, { email: a.profile.email, provider: a.password.managedBy ?? '' })}
          </Note>
          {table}
          <Note tone="neutral">
            <T msg={m.viaIdpNote} values={{ provider: a.password.managedBy ?? '', b: (c) => <b>{c}</b>, mono }} />
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
  const t = useT()
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
      title={t(m.unlinkTitle, { name: identity?.name ?? '' })}
      size="sm"
      footer={
        <>
          <Button onClick={onClose}>{t(m.keepLinked)}</Button>
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
                toast({ kind: 'deleted', title: t(m.unlinked, { name: identity.name }) })
                onClose()
              } catch (e) {
                setError(errorMessage(e))
              } finally {
                setBusy(false)
              }
            }}
          >
            {t(m.unlink)}
          </Button>
        </>
      }
    >
      <p className="text-ink-2">{t(m.unlinkBody, { name: identity?.name ?? '' })}</p>
      {error ? (
        <Note tone="bad" label={t(m.notUnlinked)}>
          {error}
        </Note>
      ) : null}
    </Dialog>
  )
}
