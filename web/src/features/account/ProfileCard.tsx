import { useQueryClient } from '@tanstack/react-query'
import { Link } from '@tanstack/react-router'
import { useId, useRef, useState, type FormEvent } from 'react'
import { ApiError, errorMessage, http, type Schemas } from '@/api/client'
import { Avatar } from '@/components/ui/Avatar'
import { Button } from '@/components/ui/Button'
import { CardSection } from '@/components/ui/Card'
import { Dialog } from '@/components/ui/Dialog'
import { Note } from '@/components/ui/Note'
import { toast } from '@/components/ui/Toast'
import { InlineAction } from '@/features/auth/AuthParts'
import { timezoneLabel, timezoneOptions } from '@/features/auth/timezones'
import { clock, usDate } from '@/lib/format'
import { meQueryKey, useWorkspace } from '@/lib/session'
import { useRefreshAccount } from './AccountPage'

type Account = Schemas['AccountOverview']

const MAX_PHOTO = 2 * 1024 * 1024

function toBase64(file: File): Promise<string> {
  return new Promise((resolve, reject) => {
    const r = new FileReader()
    r.onload = () => resolve(String(r.result).replace(/^data:[^,]*,/, ''))
    r.onerror = () => reject(r.error ?? new Error("The file couldn't be read."))
    r.readAsDataURL(file)
  })
}

/** Profile: photo, name, timezone and email (changed with a verification link). */
export function ProfileCard({ a }: { a: Account }) {
  const { slug } = useWorkspace()
  const refresh = useRefreshAccount()
  const qc = useQueryClient()
  const p = a.profile
  const [name, setName] = useState(p.name)
  const [tz, setTz] = useState(p.timezone)
  const [busy, setBusy] = useState(false)
  const [photoBusy, setPhotoBusy] = useState(false)
  const [saved, setSaved] = useState<string>()
  const [errors, setErrors] = useState<{ name?: string; timezone?: string; photo?: string; form?: string }>({})
  const [emailOpen, setEmailOpen] = useState(false)
  const [emailError, setEmailError] = useState<{ email: string; message: string }>()
  const file = useRef<HTMLInputElement>(null)
  const ids = { name: useId(), tz: useId(), email: useId(), emailHint: useId(), photoHint: useId() }

  const save = async (e: FormEvent) => {
    e.preventDefault()
    if (!name.trim()) {
      setErrors({ name: 'Enter your name as the team should see it.' })
      return
    }
    setBusy(true)
    setErrors({})
    try {
      const { data } = await http.PATCH('/api/v1/me/profile', { body: { name: name.trim(), timezone: tz } })
      setSaved(data?.name)
      await refresh()
    } catch (err) {
      const f = err instanceof ApiError ? err.problem.errors?.[0] : undefined
      if (f?.location === 'body.name') setErrors({ name: f.message })
      else if (f?.location === 'body.timezone') setErrors({ timezone: f.message })
      else setErrors({ form: errorMessage(err) })
    } finally {
      setBusy(false)
    }
  }

  const upload = async (f: File) => {
    if (!/^image\/(png|jpeg)$/.test(f.type)) {
      setErrors({ photo: 'Use a PNG or JPG photo.' })
      return
    }
    if (f.size > MAX_PHOTO) {
      setErrors({ photo: 'The photo is larger than 2 MB. Choose a smaller one.' })
      return
    }
    setPhotoBusy(true)
    setErrors({})
    try {
      await http.PUT('/api/v1/me/avatar', { body: { data: await toBase64(f) } })
      await refresh()
      toast({ kind: 'success', title: 'Photo updated' })
    } catch (err) {
      setErrors({ photo: errorMessage(err) })
    } finally {
      setPhotoBusy(false)
    }
  }

  const cancelChange = async () => {
    try {
      await http.DELETE('/api/v1/me/email')
      await refresh()
      toast({ kind: 'deleted', title: 'Email change cancelled', detail: `${p.email} stays your email.` })
    } catch (err) {
      toast({ kind: 'error', title: "Couldn't cancel", detail: errorMessage(err) })
    }
  }

  return (
    <section className="card" aria-labelledby="h-profile">
      <CardSection id="h-profile" title="Profile" caption="How you appear to the team" />
      <form className="pad col" style={{ gap: 14 }} onSubmit={save} noValidate>
        <div className="row" style={{ gap: 16 }}>
          <Avatar name={p.name || p.email} color={p.avatarColor} size={56} fontSize={13} src={p.avatarUrl} />
          <div className="col" style={{ gap: 6 }}>
            <div className="row" style={{ gap: 8 }}>
              <Button size="sm" loading={photoBusy} onClick={() => file.current?.click()} aria-describedby={ids.photoHint}>
                Upload photo
              </Button>
              <Button
                size="sm"
                disabledReason={p.avatarUrl ? undefined : 'No photo to remove: your initials show instead'}
                onClick={async () => {
                  try {
                    await http.DELETE('/api/v1/me/avatar')
                    await refresh()
                    toast({ kind: 'deleted', title: 'Photo removed', detail: 'Your initials show instead.' })
                  } catch (err) {
                    setErrors({ photo: errorMessage(err) })
                  }
                }}
              >
                Remove
              </Button>
              <input
                ref={file}
                type="file"
                accept="image/png,image/jpeg"
                className="vh"
                tabIndex={-1}
                aria-hidden="true"
                onChange={(e) => {
                  const f = e.target.files?.[0]
                  e.target.value = ''
                  if (f) void upload(f)
                }}
              />
            </div>
            {errors.photo ? (
              <span className="err" role="alert">
                {errors.photo}
              </span>
            ) : (
              <span className="k" id={ids.photoHint}>
                PNG or JPG, square, up to 2 MB
              </span>
            )}
          </div>
        </div>
        <div className="grid grid-cols-2 gap-3.5 max-md:grid-cols-1">
          <div className="lbl">
            <label htmlFor={ids.name}>Full name</label>
            <input id={ids.name} className={errors.name ? 'in err' : 'in'} value={name} onChange={(e) => setName(e.target.value)} autoComplete="name" aria-invalid={errors.name ? true : undefined} />
            {errors.name ? (
              <span className="err" role="alert">
                {errors.name}
              </span>
            ) : null}
          </div>
          <div className="lbl">
            <label htmlFor={ids.tz}>Timezone</label>
            <select id={ids.tz} className={errors.timezone ? 'in err' : 'in'} value={tz} onChange={(e) => setTz(e.target.value)}>
              {timezoneOptions(tz).map((z) => (
                <option key={z} value={z}>
                  {timezoneLabel(z)}
                </option>
              ))}
            </select>
            {errors.timezone ? (
              <span className="err" role="alert">
                {errors.timezone}
              </span>
            ) : null}
          </div>
        </div>
        <div className="lbl">
          <label htmlFor={ids.email}>Email</label>
          <div className="flex gap-2 max-md:flex-wrap">
            <input id={ids.email} className="in flex-1 max-md:min-w-full" value={p.email} readOnly aria-describedby={ids.emailHint} />
            <Button onClick={() => setEmailOpen(true)}>Change email</Button>
          </div>
          <span className="k font-medium" id={ids.emailHint}>
            A verification link goes to the new address; the change applies after you click it. SSO and invites keep working during the switch.
          </span>
        </div>
        {p.pendingEmail ? (
          <Note tone="info" label="Pending">
            Waiting for you to confirm <b>{p.pendingEmail.email}</b>: the link works until {usDate(p.pendingEmail.expiresAt)}, {clock(p.pendingEmail.expiresAt)}.{' '}
            <InlineAction onClick={cancelChange}>Cancel the change</InlineAction>
          </Note>
        ) : null}
        {emailError ? (
          <Note tone="bad" label="Not sent">
            The verification email for <b>{emailError.email}</b> couldn't be sent: {emailError.message} Your current email stays active.{' '}
            <InlineAction onClick={() => setEmailOpen(true)}>Try again</InlineAction> or check{' '}
            <Link to="/w/$workspace/providers" params={{ workspace: slug }}>
              Sending providers
            </Link>
            .
          </Note>
        ) : null}
        {saved ? (
          <Note tone="ok" label="Saved">
            Profile updated. Your name shows as “{saved}” in the audit log from now on; older entries keep the name they were recorded with.
          </Note>
        ) : null}
        {errors.form ? (
          <Note tone="bad" label="Not saved">
            {errors.form}
          </Note>
        ) : null}
        <div className="row justify-end">
          <Button type="submit" variant="primary" size="sm" loading={busy}>
            Save profile
          </Button>
        </div>
      </form>
      <ChangeEmailDialog
        open={emailOpen}
        onOpenChange={setEmailOpen}
        current={p.email}
        hasPassword={a.password.set}
        initial={emailError?.email}
        onSent={async (email) => {
          setEmailError(undefined)
          await refresh()
          await qc.invalidateQueries({ queryKey: meQueryKey })
          toast({ kind: 'info', title: 'Verification link sent', detail: `Open it from ${email} to finish.` })
        }}
        onUnsent={(email, message) => setEmailError({ email, message })}
      />
    </section>
  )
}

function ChangeEmailDialog({
  open,
  onOpenChange,
  current,
  hasPassword,
  initial,
  onSent,
  onUnsent,
}: {
  open: boolean
  onOpenChange: (o: boolean) => void
  current: string
  hasPassword: boolean
  initial?: string
  onSent: (email: string) => Promise<void>
  onUnsent: (email: string, message: string) => void
}) {
  const [email, setEmail] = useState(initial ?? '')
  const [password, setPassword] = useState('')
  const [busy, setBusy] = useState(false)
  const [errors, setErrors] = useState<{ email?: string; password?: string; form?: string }>({})
  const ids = { email: useId(), pw: useId() }
  const submit = async (e: FormEvent) => {
    e.preventDefault()
    const next = email.trim().toLowerCase()
    if (!/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(next)) {
      setErrors({ email: 'Enter the new address, like you@company.com.' })
      return
    }
    if (hasPassword && !password) {
      setErrors({ password: 'Enter your current password to confirm.' })
      return
    }
    setBusy(true)
    setErrors({})
    try {
      await http.POST('/api/v1/me/email', { body: { email: next, password: hasPassword ? password : undefined } })
      setPassword('')
      onOpenChange(false)
      await onSent(next)
    } catch (err) {
      if (err instanceof ApiError && err.code === 'provider_unavailable') {
        setPassword('')
        onOpenChange(false)
        onUnsent(next, 'the System stream has no healthy provider right now.')
        return
      }
      const f = err instanceof ApiError ? err.problem.errors?.[0] : undefined
      if (f?.location === 'body.email') setErrors({ email: f.message })
      else if (f?.location === 'body.password') setErrors({ password: f.message })
      else setErrors({ form: errorMessage(err) })
    } finally {
      setBusy(false)
    }
  }
  return (
    <Dialog
      open={open}
      onOpenChange={(o) => {
        if (o) setEmail(initial ?? '')
        setErrors({})
        onOpenChange(o)
      }}
      title="Change email"
      description={`You sign in with ${current} until you open the link sent to the new address.`}
      size="sm"
      footer={
        <>
          <Button onClick={() => onOpenChange(false)}>Cancel</Button>
          <Button type="submit" form="change-email" variant="primary" loading={busy}>
            Send verification link
          </Button>
        </>
      }
    >
      <form id="change-email" className="flex flex-col gap-3" onSubmit={submit} noValidate>
        {errors.form ? (
          <Note tone="bad" label="Not sent">
            {errors.form}
          </Note>
        ) : null}
        <div className="lbl">
          <label htmlFor={ids.email}>New email</label>
          <input id={ids.email} className={errors.email ? 'in err' : 'in'} type="email" value={email} onChange={(e) => setEmail(e.target.value)} autoComplete="email" autoFocus aria-invalid={errors.email ? true : undefined} />
          {errors.email ? (
            <span className="err" role="alert">
              {errors.email}
            </span>
          ) : null}
        </div>
        {hasPassword ? (
          <div className="lbl">
            <label htmlFor={ids.pw}>Current password</label>
            <input id={ids.pw} className={errors.password ? 'in err' : 'in'} type="password" value={password} onChange={(e) => setPassword(e.target.value)} autoComplete="current-password" aria-invalid={errors.password ? true : undefined} />
            {errors.password ? (
              <span className="err" role="alert">
                {errors.password}
              </span>
            ) : null}
          </div>
        ) : null}
      </form>
    </Dialog>
  )
}
