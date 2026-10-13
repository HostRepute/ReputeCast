import { useId, useState, type FormEvent } from 'react'
import { ApiError, errorMessage, http, type Schemas } from '@/api/client'
import { Badge } from '@/components/ui/Badge'
import { Button } from '@/components/ui/Button'
import { CardSection } from '@/components/ui/Card'
import { Dialog } from '@/components/ui/Dialog'
import { KeyValues } from '@/components/ui/Data'
import { Note } from '@/components/ui/Note'
import { Table, Td, Th } from '@/components/ui/Table'
import { toast } from '@/components/ui/Toast'
import { passkeyErrorMessage, registerPasskey } from '@/features/auth/passkey'
import { RecoveryCodesPanel } from '@/features/auth/RecoveryCodes'
import { TotpSetupForm } from '@/features/auth/TotpSetup'
import { ago, usDate } from '@/lib/format'
import { useRefreshAccount } from './AccountPage'

type Account = Schemas['AccountOverview']
type Passkey = Schemas['AccountPasskey']

/** Password: hidden for people who only sign in through SSO. */
export function PasswordCard({ a }: { a: Account }) {
  const refresh = useRefreshAccount()
  const [current, setCurrent] = useState('')
  const [next, setNext] = useState('')
  const [confirm, setConfirm] = useState('')
  const [busy, setBusy] = useState(false)
  const [errors, setErrors] = useState<{ current?: string; password?: string; confirm?: string; form?: string }>({})
  const ids = { cur: useId(), pw: useId(), cf: useId() }
  const pw = a.password
  const submit = async (e: FormEvent) => {
    e.preventDefault()
    const errs: typeof errors = {}
    if (pw.set && !current) errs.current = 'Enter your current password.'
    if ([...next].length < pw.minLength) errs.password = `Use at least ${pw.minLength} characters. Passphrases are fine.`
    if (confirm !== next) errs.confirm = "The two passwords don't match."
    setErrors(errs)
    if (Object.keys(errs).length) return
    setBusy(true)
    try {
      await http.POST('/api/v1/me/password', { body: { current: pw.set ? current : undefined, password: next } })
      setCurrent('')
      setNext('')
      setConfirm('')
      await refresh()
      toast({ kind: 'success', title: 'Password changed', detail: 'Your other sessions were signed out.' })
    } catch (err) {
      const f = err instanceof ApiError ? err.problem.errors?.[0] : undefined
      if (f?.location === 'body.current') setErrors({ current: f.message })
      else if (f?.location === 'body.password') setErrors({ password: f.message })
      else setErrors({ form: errorMessage(err) })
    } finally {
      setBusy(false)
    }
  }
  return (
    <section className="card" aria-labelledby="h-password">
      <CardSection id="h-password" title="Password" caption="Hidden for members who sign in only through SSO" />
      <form className="pad col" onSubmit={submit} noValidate>
        <KeyValues
          items={[
            ['Last changed', pw.changedAt ? usDate(pw.changedAt) : pw.set ? 'Unknown' : 'No password yet'],
            ['Policy', `${pw.minLength} characters minimum · ${pw.breachedCheck ? 'checked against known breaches' : 'passphrases are fine'}`],
          ]}
        />
        <input type="email" name="email" autoComplete="username" value={a.profile.email} readOnly hidden />
        {errors.form ? (
          <Note tone="bad" label="Not changed">
            {errors.form}
          </Note>
        ) : null}
        {pw.set ? (
          <div className="lbl">
            <label htmlFor={ids.cur}>Current password</label>
            <input id={ids.cur} className={errors.current ? 'in err' : 'in'} type="password" value={current} onChange={(e) => setCurrent(e.target.value)} autoComplete="current-password" aria-invalid={errors.current ? true : undefined} />
            {errors.current ? (
              <span className="err" role="alert">
                {errors.current}
              </span>
            ) : null}
          </div>
        ) : null}
        <div className="grid grid-cols-2 gap-3.5 max-md:grid-cols-1">
          <div className="lbl">
            <label htmlFor={ids.pw}>New password</label>
            <input id={ids.pw} className={errors.password ? 'in err' : 'in'} type="password" value={next} onChange={(e) => setNext(e.target.value)} autoComplete="new-password" aria-invalid={errors.password ? true : undefined} />
            {errors.password ? (
              <span className="err" role="alert">
                {errors.password}
              </span>
            ) : null}
          </div>
          <div className="lbl">
            <label htmlFor={ids.cf}>Confirm new password</label>
            <input id={ids.cf} className={errors.confirm ? 'in err' : 'in'} type="password" value={confirm} onChange={(e) => setConfirm(e.target.value)} autoComplete="new-password" aria-invalid={errors.confirm ? true : undefined} />
            {errors.confirm ? (
              <span className="err" role="alert">
                {errors.confirm}
              </span>
            ) : null}
          </div>
        </div>
        <div className="row justify-between">
          <span className="k">Changing it signs out your other sessions.</span>
          <Button type="submit" variant="primary" size="sm" loading={busy}>
            Change password
          </Button>
        </div>
      </form>
    </section>
  )
}

function passkeyKind(p: Passkey) {
  if (p.kind === 'security_key') return 'Security key'
  const base = p.kind === 'platform' ? 'Platform passkey' : 'Passkey'
  return p.provider ? `${base} · ${p.provider}` : base
}

function lastUsed(p: Passkey) {
  if (p.thisSession) return 'used now · this session'
  if (!p.lastUsedAt) return 'never used'
  const minutes = (Date.now() - new Date(p.lastUsedAt).getTime()) / 60000
  return `used ${minutes < 24 * 60 ? ago(p.lastUsedAt) : usDate(p.lastUsedAt)}`
}

/** Passkeys: add, rename, remove; created and last used. */
export function PasskeysCard({ a }: { a: Account }) {
  const refresh = useRefreshAccount()
  const [adding, setAdding] = useState(false)
  const [rename, setRename] = useState<Passkey>()
  const [remove, setRemove] = useState<Passkey>()
  const add = async () => {
    setAdding(true)
    try {
      const pk = await registerPasskey()
      await refresh()
      toast({ kind: 'success', title: 'Passkey added', detail: `${pk.name}. Rename it so you recognise it later.` })
    } catch (e) {
      toast({ kind: 'error', title: 'No passkey added', detail: passkeyErrorMessage(e, 'add') })
    } finally {
      setAdding(false)
    }
  }
  return (
    <section className="card" aria-labelledby="h-passkeys">
      <CardSection
        id="h-passkeys"
        title="Passkeys"
        caption="Passkeys sign you in on their own and count as a second factor"
        actions={
          <Button size="sm" variant="primary" loading={adding} onClick={add}>
            Add a passkey
          </Button>
        }
      />
      <Table density="compact" className="[&_td]:whitespace-normal" caption="Passkeys">
        <thead>
          <tr>
            <Th>Passkey</Th>
            <Th>Created · last used</Th>
            <Th>
              <span className="vh">Actions</span>
            </Th>
          </tr>
        </thead>
        <tbody>
          {a.passkeys.length === 0 ? (
            <tr>
              <Td colSpan={3}>
                <span className="k">No passkeys yet. Add one to sign in with Touch ID, Face ID, Windows Hello or a security key.</span>
              </Td>
            </tr>
          ) : (
            a.passkeys.map((p) => (
              <tr key={p.id}>
                <Td>
                  <div className="col" style={{ gap: 0 }}>
                    <span className="font-semibold">{p.name}</span>
                    <span className="k">{passkeyKind(p)}</span>
                  </div>
                </Td>
                <Td>
                  <div className="col" style={{ gap: 0 }}>
                    <span>{usDate(p.createdAt)}</span>
                    <span className="k">{lastUsed(p)}</span>
                  </div>
                </Td>
                <Td>
                  <div className="row flex-nowrap" style={{ gap: 6 }}>
                    <Button size="sm" onClick={() => setRename(p)} aria-label={`Rename ${p.name}`}>
                      Rename
                    </Button>
                    <Button size="sm" onClick={() => setRemove(p)} aria-label={`Remove ${p.name}`}>
                      Remove
                    </Button>
                  </div>
                </Td>
              </tr>
            ))
          )}
        </tbody>
      </Table>
      <RenamePasskeyDialog passkey={rename} onClose={() => setRename(undefined)} onDone={refresh} />
      <RemovePasskeyDialog passkey={remove} onClose={() => setRemove(undefined)} onDone={refresh} />
    </section>
  )
}

function RenamePasskeyDialog({ passkey, onClose, onDone }: { passkey?: Passkey; onClose: () => void; onDone: () => Promise<void> }) {
  // The draft belongs to one passkey; opening another starts from its name.
  const [draft, setDraft] = useState<{ id: string; name: string }>()
  const name = draft && draft.id === passkey?.id ? draft.name : (passkey?.name ?? '')
  const setName = (v: string) => passkey && setDraft({ id: passkey.id, name: v })
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState<string>()
  const id = useId()
  const submit = async (e: FormEvent) => {
    e.preventDefault()
    if (!passkey) return
    if (!name.trim()) {
      setError('Give the passkey a name, like “MacBook Pro · Touch ID”.')
      return
    }
    setBusy(true)
    try {
      await http.PATCH('/api/v1/me/passkeys/{id}', { params: { path: { id: passkey.id } }, body: { name: name.trim() } })
      await onDone()
      onClose()
    } catch (err) {
      setError(errorMessage(err))
    } finally {
      setBusy(false)
    }
  }
  return (
    <Dialog
      open={Boolean(passkey)}
      onOpenChange={(o) => {
        if (o) return
        setError(undefined)
        onClose()
      }}
      title="Rename passkey"
      description="Name it after the device or app that holds it."
      size="sm"
      footer={
        <>
          <Button onClick={onClose}>Cancel</Button>
          <Button type="submit" form="rename-passkey" variant="primary" loading={busy}>
            Save name
          </Button>
        </>
      }
    >
      <form id="rename-passkey" onSubmit={submit} noValidate>
        <div className="lbl">
          <label htmlFor={id}>Name</label>
          <input
            id={id}
            className={error ? 'in err' : 'in'}
            value={name}
            onChange={(e) => setName(e.target.value)}
            maxLength={80}
            autoFocus
            aria-invalid={error ? true : undefined}
          />
          {error ? (
            <span className="err" role="alert">
              {error}
            </span>
          ) : null}
        </div>
      </form>
    </Dialog>
  )
}

function RemovePasskeyDialog({ passkey, onClose, onDone }: { passkey?: Passkey; onClose: () => void; onDone: () => Promise<void> }) {
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState<string>()
  return (
    <Dialog
      open={Boolean(passkey)}
      onOpenChange={(o) => {
        if (o) return
        setError(undefined)
        onClose()
      }}
      title={`Remove “${passkey?.name ?? ''}”?`}
      size="sm"
      footer={
        <>
          <Button onClick={onClose}>Keep it</Button>
          <Button
            variant="danger-primary"
            loading={busy}
            onClick={async () => {
              if (!passkey) return
              setBusy(true)
              setError(undefined)
              try {
                await http.DELETE('/api/v1/me/passkeys/{id}', { params: { path: { id: passkey.id } } })
                await onDone()
                toast({ kind: 'deleted', title: 'Passkey removed', detail: passkey.name })
                onClose()
              } catch (err) {
                setError(errorMessage(err))
              } finally {
                setBusy(false)
              }
            }}
          >
            Remove passkey
          </Button>
        </>
      }
    >
      <p className="text-ink-2">You won't be able to sign in or confirm sign-ins with this passkey. Sessions it already signed in stay signed in until they end.</p>
      {error ? (
        <Note tone="bad" label="Not removed">
          {error}
        </Note>
      ) : null}
    </Dialog>
  )
}

/** Authenticator app (TOTP) and recovery codes. */
export function AuthenticatorCard({ a }: { a: Account }) {
  const refresh = useRefreshAccount()
  const [setup, setSetup] = useState(false)
  const [regen, setRegen] = useState(false)
  const t = a.twoFactor
  const caption =
    t.requiredBy === 'everyone'
      ? 'Required for everyone on this instance'
      : t.requiredBy === 'sensitive'
        ? 'Required for your role: you hold sensitive permissions'
        : 'Optional: asks for a code after your password'
  const hasSecond = t.totp || a.passkeys.length > 0
  return (
    <section className="card" aria-labelledby="h-totp">
      <CardSection id="h-totp" title="Authenticator app" caption={caption} />
      <div className="pad col" style={{ gap: 14 }}>
        <div className="row justify-between">
          <div className="row">
            {t.totp ? (
              <Badge tone="ok" dot>
                On
              </Badge>
            ) : (
              <Badge tone="neutral" dot>
                Off
              </Badge>
            )}
            <div className="col" style={{ gap: 0 }}>
              <span className="font-semibold">Time-based codes (TOTP)</span>
              <span className="k">
                {t.totp && t.totpSetAt
                  ? `Set up ${usDate(t.totpSetAt)}${t.totpLabel ? ` · ${t.totpLabel}` : ''}`
                  : t.graceUntil
                    ? `Not set up · required by ${usDate(t.graceUntil)}`
                    : 'Not set up'}
              </span>
            </div>
          </div>
          <Button size="sm" variant={t.totp ? 'default' : 'primary'} onClick={() => setSetup(true)}>
            {t.totp ? 'Replace app' : 'Set up'}
          </Button>
        </div>
        {hasSecond ? (
          <div className="row justify-between rounded-lg border border-line px-3.5 py-3">
            <div className="col" style={{ gap: 0 }}>
              <span className="font-semibold">Recovery codes</span>
              <span className="k">
                {t.recoveryTotal > 0 ? `${t.recoveryRemaining} of ${t.recoveryTotal} remain` : 'None yet'}
                {t.recoveryLastUsedAt ? ` · last used ${usDate(t.recoveryLastUsedAt)}` : ''}
              </span>
            </div>
            <Button size="sm" onClick={() => setRegen(true)}>
              {t.recoveryTotal > 0 ? 'Regenerate codes' : 'Generate codes'}
            </Button>
          </div>
        ) : null}
        <span className="k">Regenerating replaces all unused codes. New codes are shown once, with download and print.</span>
      </div>
      <TotpDialog open={setup} replacing={t.totp} email={a.profile.email} onClose={() => setSetup(false)} onDone={refresh} />
      <RegenerateDialog open={regen} remaining={t.recoveryRemaining} email={a.profile.email} onClose={() => setRegen(false)} onDone={refresh} />
    </section>
  )
}

function TotpDialog({ open, replacing, email, onClose, onDone }: { open: boolean; replacing: boolean; email: string; onClose: () => void; onDone: () => Promise<void> }) {
  const [codes, setCodes] = useState<string[]>()
  const close = () => {
    setCodes(undefined)
    onClose()
  }
  return (
    <Dialog
      open={open}
      onOpenChange={(o) => {
        if (!o) {
          if (codes) void onDone()
          close()
        }
      }}
      dismissible={!codes}
      title={codes ? 'Save your recovery codes' : replacing ? 'Replace your authenticator app' : 'Set up an authenticator app'}
      description={codes ? 'Step 2 of 2 · shown once' : `Step 1 of 2${replacing ? ' · the old app stops working once you confirm' : ''}`}
      eyebrow={
        codes ? (
          <Badge tone="ok" dot>
            Authenticator on
          </Badge>
        ) : undefined
      }
    >
      {codes ? (
        <RecoveryCodesPanel
          codes={codes}
          email={email}
          onFinish={async () => {
            await onDone()
            toast({ kind: 'success', title: 'Authenticator app on', detail: 'Sign-ins with your password now ask for a code.' })
            close()
          }}
        />
      ) : open ? (
        <TotpSetupForm
          onConfirmed={async (c) => {
            setCodes(c)
            await onDone()
          }}
          onCancel={close}
        />
      ) : null}
    </Dialog>
  )
}

function RegenerateDialog({ open, remaining, email, onClose, onDone }: { open: boolean; remaining: number; email: string; onClose: () => void; onDone: () => Promise<void> }) {
  const [codes, setCodes] = useState<string[]>()
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState<string>()
  const close = () => {
    setCodes(undefined)
    setError(undefined)
    onClose()
  }
  return (
    <Dialog
      open={open}
      onOpenChange={(o) => {
        if (!o) close()
      }}
      dismissible={!codes}
      title={codes ? 'Save your recovery codes' : 'Regenerate recovery codes?'}
      description={codes ? 'Shown once' : undefined}
      size={codes ? 'md' : 'sm'}
      footer={
        codes ? undefined : (
          <>
            <Button onClick={close}>Cancel</Button>
            <Button
              variant="primary"
              loading={busy}
              onClick={async () => {
                setBusy(true)
                setError(undefined)
                try {
                  const { data } = await http.POST('/api/v1/me/recovery-codes')
                  setCodes(data?.codes ?? [])
                  await onDone()
                } catch (e) {
                  setError(errorMessage(e))
                } finally {
                  setBusy(false)
                }
              }}
            >
              Generate new codes
            </Button>
          </>
        )
      }
    >
      {codes ? (
        <RecoveryCodesPanel
          codes={codes}
          email={email}
          onFinish={() => {
            toast({ kind: 'success', title: 'New recovery codes saved', detail: 'The old codes no longer work.' })
            close()
          }}
        />
      ) : (
        <>
          <p className="text-ink-2">
            {remaining > 0 ? `Your ${remaining} unused ${remaining === 1 ? 'code stops' : 'codes stop'} working right away.` : 'You get 10 new codes.'} The new codes are shown once, with
            download and print.
          </p>
          {error ? (
            <Note tone="bad" label="Not generated">
              {error}
            </Note>
          ) : null}
        </>
      )}
    </Dialog>
  )
}
