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
import { t as tr, useT } from '@/i18n'
import { ago, usDate } from '@/lib/format'
import { useRefreshAccount } from './AccountPage'
import { m } from './messages'

type Account = Schemas['AccountOverview']
type Passkey = Schemas['AccountPasskey']

/** Password: hidden for people who only sign in through SSO. */
export function PasswordCard({ a }: { a: Account }) {
  const t = useT()
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
    if (pw.set && !current) errs.current = t(m.currentRequired)
    if ([...next].length < pw.minLength) errs.password = t(m.tooShort, { min: pw.minLength })
    if (confirm !== next) errs.confirm = t(m.mismatch)
    setErrors(errs)
    if (Object.keys(errs).length) return
    setBusy(true)
    try {
      await http.POST('/api/v1/me/password', { body: { current: pw.set ? current : undefined, password: next } })
      setCurrent('')
      setNext('')
      setConfirm('')
      await refresh()
      toast({ kind: 'success', title: t(m.passwordChanged), detail: t(m.othersSignedOut) })
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
      <CardSection id="h-password" title={t(m.password)} caption={t(m.passwordCaption)} />
      <form className="pad col" onSubmit={submit} noValidate>
        <KeyValues
          items={[
            [t(m.lastChanged), pw.changedAt ? usDate(pw.changedAt) : pw.set ? t(m.unknown) : t(m.noPasswordYet)],
            [t(m.policy), t(m.policyValue, { min: pw.minLength, breached: pw.breachedCheck ? 'yes' : 'no' })],
          ]}
        />
        <input type="email" name="email" autoComplete="username" value={a.profile.email} readOnly hidden />
        {errors.form ? (
          <Note tone="bad" label={t(m.notChanged)}>
            {errors.form}
          </Note>
        ) : null}
        {pw.set ? (
          <div className="lbl">
            <label htmlFor={ids.cur}>{t(m.currentPassword)}</label>
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
            <label htmlFor={ids.pw}>{t(m.newPassword)}</label>
            <input id={ids.pw} className={errors.password ? 'in err' : 'in'} type="password" value={next} onChange={(e) => setNext(e.target.value)} autoComplete="new-password" aria-invalid={errors.password ? true : undefined} />
            {errors.password ? (
              <span className="err" role="alert">
                {errors.password}
              </span>
            ) : null}
          </div>
          <div className="lbl">
            <label htmlFor={ids.cf}>{t(m.confirmPassword)}</label>
            <input id={ids.cf} className={errors.confirm ? 'in err' : 'in'} type="password" value={confirm} onChange={(e) => setConfirm(e.target.value)} autoComplete="new-password" aria-invalid={errors.confirm ? true : undefined} />
            {errors.confirm ? (
              <span className="err" role="alert">
                {errors.confirm}
              </span>
            ) : null}
          </div>
        </div>
        <div className="row justify-between">
          <span className="k">{t(m.signsOutOthers)}</span>
          <Button type="submit" variant="primary" size="sm" loading={busy}>
            {t(m.changePassword)}
          </Button>
        </div>
      </form>
    </section>
  )
}

function passkeyKind(p: Passkey) {
  if (p.kind === 'security_key') return tr(m.securityKey)
  const base = p.kind === 'platform' ? tr(m.platformPasskey) : tr(m.passkey)
  return p.provider ? `${base} · ${p.provider}` : base
}

function lastUsed(p: Passkey) {
  if (p.thisSession) return tr(m.usedNow)
  if (!p.lastUsedAt) return tr(m.neverUsed)
  const minutes = (Date.now() - new Date(p.lastUsedAt).getTime()) / 60000
  return minutes < 24 * 60 ? tr(m.usedAgo, { ago: ago(p.lastUsedAt) }) : tr(m.usedOn, { date: usDate(p.lastUsedAt) })
}

/** Passkeys: add, rename, remove; created and last used. */
export function PasskeysCard({ a }: { a: Account }) {
  const t = useT()
  const refresh = useRefreshAccount()
  const [adding, setAdding] = useState(false)
  const [rename, setRename] = useState<Passkey>()
  const [remove, setRemove] = useState<Passkey>()
  const add = async () => {
    setAdding(true)
    try {
      const pk = await registerPasskey()
      await refresh()
      toast({ kind: 'success', title: t(m.passkeyAdded), detail: t(m.renameHint, { name: pk.name }) })
    } catch (e) {
      toast({ kind: 'error', title: t(m.noPasskeyAdded), detail: passkeyErrorMessage(e, 'add') })
    } finally {
      setAdding(false)
    }
  }
  return (
    <section className="card" aria-labelledby="h-passkeys">
      <CardSection
        id="h-passkeys"
        title={t(m.passkeys)}
        caption={t(m.passkeysCaption)}
        actions={
          <Button size="sm" variant="primary" loading={adding} onClick={add}>
            {t(m.addPasskey)}
          </Button>
        }
      />
      <Table density="compact" className="[&_td]:whitespace-normal" caption={t(m.passkeys)}>
        <thead>
          <tr>
            <Th>{t(m.passkey)}</Th>
            <Th>{t(m.colCreated)}</Th>
            <Th>
              <span className="vh">{t(m.actions)}</span>
            </Th>
          </tr>
        </thead>
        <tbody>
          {a.passkeys.length === 0 ? (
            <tr>
              <Td colSpan={3}>
                <span className="k">{t(m.noPasskeys)}</span>
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
                    <Button size="sm" onClick={() => setRename(p)} aria-label={t(m.renameNamed, { name: p.name })}>
                      {t(m.rename)}
                    </Button>
                    <Button size="sm" onClick={() => setRemove(p)} aria-label={t(m.removeNamed, { name: p.name })}>
                      {t(m.remove)}
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
  const t = useT()
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
      setError(t(m.passkeyNameRequired))
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
      title={t(m.renamePasskey)}
      description={t(m.renameDescription)}
      size="sm"
      footer={
        <>
          <Button onClick={onClose}>{t(m.cancel)}</Button>
          <Button type="submit" form="rename-passkey" variant="primary" loading={busy}>
            {t(m.saveName)}
          </Button>
        </>
      }
    >
      <form id="rename-passkey" onSubmit={submit} noValidate>
        <div className="lbl">
          <label htmlFor={id}>{t(m.name)}</label>
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
  const t = useT()
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
      title={t(m.removeTitle, { name: passkey?.name ?? '' })}
      size="sm"
      footer={
        <>
          <Button onClick={onClose}>{t(m.keepIt)}</Button>
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
                toast({ kind: 'deleted', title: t(m.passkeyRemoved), detail: passkey.name })
                onClose()
              } catch (err) {
                setError(errorMessage(err))
              } finally {
                setBusy(false)
              }
            }}
          >
            {t(m.removePasskey)}
          </Button>
        </>
      }
    >
      <p className="text-ink-2">{t(m.removeBody)}</p>
      {error ? (
        <Note tone="bad" label={t(m.notRemoved)}>
          {error}
        </Note>
      ) : null}
    </Dialog>
  )
}

/** Authenticator app (TOTP) and recovery codes. */
export function AuthenticatorCard({ a }: { a: Account }) {
  const t = useT()
  const refresh = useRefreshAccount()
  const [setup, setSetup] = useState(false)
  const [regen, setRegen] = useState(false)
  const tf = a.twoFactor
  const caption = tf.requiredBy === 'everyone' ? t(m.requiredEveryone) : tf.requiredBy === 'sensitive' ? t(m.requiredSensitive) : t(m.optional)
  const hasSecond = tf.totp || a.passkeys.length > 0
  const recovery = [tf.recoveryTotal > 0 ? t(m.codesRemain, { remaining: tf.recoveryRemaining, total: tf.recoveryTotal }) : t(m.noneYet)]
  if (tf.recoveryLastUsedAt) recovery.push(t(m.lastUsedOn, { date: usDate(tf.recoveryLastUsedAt) }))
  return (
    <section className="card" aria-labelledby="h-totp">
      <CardSection id="h-totp" title={t(m.authenticatorApp)} caption={caption} />
      <div className="pad col" style={{ gap: 14 }}>
        <div className="row justify-between">
          <div className="row">
            {tf.totp ? (
              <Badge tone="ok" dot>
                {t(m.on)}
              </Badge>
            ) : (
              <Badge tone="neutral" dot>
                {t(m.off)}
              </Badge>
            )}
            <div className="col" style={{ gap: 0 }}>
              <span className="font-semibold">{t(m.totp)}</span>
              <span className="k">
                {tf.totp && tf.totpSetAt
                  ? [t(m.setUpOn, { date: usDate(tf.totpSetAt) }), tf.totpLabel].filter(Boolean).join(' · ')
                  : tf.graceUntil
                    ? t(m.notSetUpBy, { date: usDate(tf.graceUntil) })
                    : t(m.notSetUp)}
              </span>
            </div>
          </div>
          <Button size="sm" variant={tf.totp ? 'default' : 'primary'} onClick={() => setSetup(true)}>
            {tf.totp ? t(m.replaceApp) : t(m.setUp)}
          </Button>
        </div>
        {hasSecond ? (
          <div className="row justify-between rounded-lg border border-line px-3.5 py-3">
            <div className="col" style={{ gap: 0 }}>
              <span className="font-semibold">{t(m.recoveryCodes)}</span>
              <span className="k">{recovery.join(' · ')}</span>
            </div>
            <Button size="sm" onClick={() => setRegen(true)}>
              {tf.recoveryTotal > 0 ? t(m.regenerateCodes) : t(m.generateCodes)}
            </Button>
          </div>
        ) : null}
        <span className="k">{t(m.regenerateNote)}</span>
      </div>
      <TotpDialog open={setup} replacing={tf.totp} email={a.profile.email} onClose={() => setSetup(false)} onDone={refresh} />
      <RegenerateDialog open={regen} remaining={tf.recoveryRemaining} email={a.profile.email} onClose={() => setRegen(false)} onDone={refresh} />
    </section>
  )
}

function TotpDialog({ open, replacing, email, onClose, onDone }: { open: boolean; replacing: boolean; email: string; onClose: () => void; onDone: () => Promise<void> }) {
  const t = useT()
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
      title={codes ? t(m.saveCodesTitle) : replacing ? t(m.replaceTitle) : t(m.setUpTitle)}
      description={codes ? t(m.stepTwo) : replacing ? t(m.stepOneReplacing) : t(m.stepOne)}
      eyebrow={
        codes ? (
          <Badge tone="ok" dot>
            {t(m.authenticatorOn)}
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
            toast({ kind: 'success', title: t(m.authenticatorAppOn), detail: t(m.codeAsked) })
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
  const t = useT()
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
      title={codes ? t(m.saveCodesTitle) : t(m.regenerateTitle)}
      description={codes ? t(m.shownOnce) : undefined}
      size={codes ? 'md' : 'sm'}
      footer={
        codes ? undefined : (
          <>
            <Button onClick={close}>{t(m.cancel)}</Button>
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
              {t(m.generateNew)}
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
            toast({ kind: 'success', title: t(m.newCodesSaved), detail: t(m.oldCodesGone) })
            close()
          }}
        />
      ) : (
        <>
          <p className="text-ink-2">{t(m.regenerateBody, { remaining })}</p>
          {error ? (
            <Note tone="bad" label={t(m.notGenerated)}>
              {error}
            </Note>
          ) : null}
        </>
      )}
    </Dialog>
  )
}
