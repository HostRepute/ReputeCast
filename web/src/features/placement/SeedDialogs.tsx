import { Link } from '@tanstack/react-router'
import { useId, useState, type ReactNode } from 'react'
import { errorMessage, http } from '@/api/client'
import { Icon } from '@/components/icons/Icon'
import { Button, ButtonAnchor } from '@/components/ui/Button'
import { Dialog } from '@/components/ui/Dialog'
import { toast } from '@/components/ui/Toast'
import { T, useT } from '@/i18n'
import { andList, clock, shortDate } from '@/lib/format'
import { useWorkspace } from '@/lib/session'
import { m } from './messages'
import { base, useRefreshPlacement, type SeedList, type SeedProvider, type SeedView } from './shared'

type Folder = SeedView['folders'][number]

const hint = (p: SeedProvider, t: ReturnType<typeof useT>) =>
  p.oauth ? t(m.hintOAuthOrApp) : p.password === 'App-specific password' ? t(m.hintAppSpecific) : p.password === 'Password' ? t(m.hintPasswordImap) : t(m.hintAppPassword)

const appName = (p: SeedProvider) => (p.oauth === 'microsoft' ? 'Microsoft' : 'Google')

/** "Inbox", "Focused, Other". */
export function inboxNames(folders: Folder[]): string[] {
  return folders.filter((f) => f.role === 'inbox').map((f) => f.label)
}

function signedIn(email: string, folders: Folder[]) {
  const inbox = andList(inboxNames(folders))
  const tabs = folders.filter((f) => f.role === 'tab').map((f) => f.label)
  const spam = folders.find((f) => f.role === 'spam')?.label ?? ''
  return tabs.length ? (
    <T msg={m.signedInAsTabs} values={{ email, inbox, tabs: andList(tabs), spam, mono: (c) => <span className="mono">{c}</span> }} />
  ) : (
    <T msg={m.signedInAs} values={{ email, inbox, spam, mono: (c) => <span className="mono">{c}</span> }} />
  )
}

function Field({ label, children, hint: help, id }: { label: string; children: ReactNode; hint?: ReactNode; id: string }) {
  return (
    <div className="flex min-w-0 flex-col gap-1.5">
      <label htmlFor={id} className="text-[12.5px] font-semibold">
        {label}
      </label>
      {children}
      {help ? <span className="k">{help}</span> : null}
    </div>
  )
}

function ExternalIcon() {
  return (
    <svg width="13" height="13" viewBox="0 0 16 16" fill="none" stroke="currentColor" strokeWidth="1.7" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
      <path d="M9 3h4v4M13 3 7 9M11 9.5V13H3V5h3.5" />
    </svg>
  )
}

function InstanceAuthLink({ children }: { children: ReactNode }) {
  const { slug } = useWorkspace()
  return (
    <Link to="/w/$workspace/admin/authentication" params={{ workspace: slug }} className="underline">
      {children}
    </Link>
  )
}

/** Starts Google's or Microsoft's consent; the browser comes back to Seed mailboxes. */
async function startOAuth(slug: string, provider: SeedProvider['key'], seedId?: string) {
  const res = await http.POST(`${base}/seeds/oauth`, { params: { path: { workspace: slug } }, body: { provider: provider as 'gmail', seedId } })
  if (!res.data) throw new Error(errorMessage(res.error))
  window.location.assign(res.data.url)
}

export type AddNote = { kind: 'cancelled' | 'failed'; reason?: string }

/** Add a seed mailbox: OAuth for Google and Microsoft, an IMAP password for the rest. */
export function AddSeedDialog({ d, open, onOpenChange, preset, note }: { d: SeedList; open: boolean; onOpenChange: (o: boolean) => void; preset?: string; note?: AddNote }) {
  const t = useT()
  const { slug } = useWorkspace()
  const refresh = useRefreshPlacement()
  const ids = { host: useId(), port: useId(), email: useId(), password: useId(), provider: useId() }
  const initial = d.providers.find((p) => p.key === preset)
  const [provider, setProvider] = useState<SeedProvider | undefined>(initial)
  const [mode, setMode] = useState<'oauth' | 'imap'>(initial?.oauth ? 'oauth' : 'imap')
  const [host, setHost] = useState(initial?.host ?? '')
  const [port, setPort] = useState(String(initial?.port ?? 993))
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState<string>()
  const [done, setDone] = useState<{ email: string; folders: Folder[] }>()
  const [oauthNote, setOauthNote] = useState<AddNote | undefined>(note)

  const pick = (p: SeedProvider) => {
    setProvider(p)
    setMode(p.oauth ? 'oauth' : 'imap')
    setHost(p.host)
    setPort(String(p.port))
    setError(undefined)
    setOauthNote(undefined)
  }
  const reset = () => {
    setProvider(undefined)
    setEmail('')
    setPassword('')
    setDone(undefined)
    setError(undefined)
    setOauthNote(undefined)
  }
  const ready = provider && (mode === 'oauth' ? provider.oauth && (provider.oauth === 'google' ? d.oauthGoogle : d.oauthMicrosoft) : email.trim() && password && host.trim())
  const submit = async () => {
    if (!provider || busy) return
    setBusy(true)
    setError(undefined)
    try {
      if (mode === 'oauth') {
        await startOAuth(slug, provider.key)
        return
      }
      const res = await http.POST(`${base}/seeds`, {
        params: { path: { workspace: slug } },
        body: { provider: provider.key as SeedView['provider'], email: email.trim(), password, host: host.trim(), port: Number(port) || 993 },
      })
      if (!res.data) throw new Error(errorMessage(res.error))
      setDone({ email: res.data.seed.email, folders: res.data.folders })
      setPassword('')
      void refresh()
    } catch (e) {
      setError(errorMessage(e))
    } finally {
      setBusy(false)
    }
  }
  const oauthReady = provider?.oauth ? (provider.oauth === 'google' ? d.oauthGoogle : d.oauthMicrosoft) : false
  const compact = Boolean(preset) || Boolean(done) || (provider && !provider.oauth)
  const gmailCom = provider?.key === 'gmail'

  const footer = done ? (
    <>
      <Button className="mr-auto" onClick={reset}>
        {t(m.addAnother)}
      </Button>
      <Button variant="primary" onClick={() => onOpenChange(false)}>
        {t(m.done)}
      </Button>
    </>
  ) : (
    <>
      <Button className="mr-auto" onClick={() => onOpenChange(false)}>
        {t(m.cancel)}
      </Button>
      {provider ? (
        mode === 'oauth' ? (
          <Button variant="primary" onClick={() => void submit()} loading={busy} disabledReason={busy || oauthReady ? undefined : t(m.oauthNotSetUp, { app: appName(provider) })}>
            <ExternalIcon />
            {t(m.connectWithApp, { app: appName(provider) })}
          </Button>
        ) : (
          <Button variant="primary" onClick={() => void submit()} loading={busy} busyLabel={t(m.checking)} disabledReason={busy || ready ? undefined : t(m.fillFields)}>
            {t(m.checkAndAdd)}
          </Button>
        )
      ) : null}
    </>
  )

  return (
    <Dialog
      open={open}
      onOpenChange={(o) => !busy && onOpenChange(o)}
      eyebrow={<span className="k">{t(m.dialogCrumb)}</span>}
      title={t(m.addTitle)}
      footer={footer}
      dismissible={!busy}
    >
      {compact ? (
        <Field label={t(m.providerLabel)} id={ids.provider}>
          <select
            id={ids.provider}
            className="in"
            value={provider?.key ?? ''}
            disabled={Boolean(done)}
            onChange={(e) => {
              const p = d.providers.find((x) => x.key === e.target.value)
              if (p) pick(p)
            }}
          >
            {d.providers.map((p) => (
              <option key={p.key} value={p.key}>
                {p.name}
              </option>
            ))}
          </select>
        </Field>
      ) : (
        <div className="grid gap-2" style={{ gridTemplateColumns: 'repeat(auto-fit,minmax(min(160px,100%),1fr))' }} role="radiogroup" aria-label={t(m.providerLabel)}>
          {d.providers.map((p) => {
            const on = provider?.key === p.key
            return (
              <button
                type="button"
                role="radio"
                aria-checked={on}
                key={p.key}
                onClick={() => pick(p)}
                className={on ? 'pl-opt on flex-col! gap-0.5! text-left' : 'pl-opt flex-col! gap-0.5! text-left'}
              >
                <b className="font-semibold">{p.name}</b>
                <span className="k">{hint(p, t)}</span>
              </button>
            )
          })}
        </div>
      )}
      {provider?.oauth && !done ? (
        <div className="flex flex-col gap-1.5">
          <span className="text-[12.5px] font-semibold">{t(m.connectWith)}</span>
          <div className="seg w-fit" role="group" aria-label={t(m.connectWith)}>
            <button type="button" className={mode === 'oauth' ? 'on' : undefined} aria-pressed={mode === 'oauth'} onClick={() => setMode('oauth')}>
              {t(m.viaOAuth)}
            </button>
            <button type="button" className={mode === 'imap' ? 'on' : undefined} aria-pressed={mode === 'imap'} onClick={() => setMode('imap')}>
              {t(m.viaImap)}
            </button>
          </div>
        </div>
      ) : null}
      {provider && mode === 'oauth' && !done ? (
        <>
          {oauthNote?.kind === 'cancelled' ? (
            <div className="note n" role="status">
              <Icon name="info" size={16} className="mt-px flex-none text-ink-3" />
              <span>{t(m.oauthCancelled, { app: appName(provider) })}</span>
            </div>
          ) : oauthNote?.kind === 'failed' ? (
            <div className="note b" role="alert">
              <Icon name="alertCircle" size={16} className="mt-px flex-none" />
              <span>{t(m.oauthFailed, { app: appName(provider), reason: oauthNote.reason ?? '' })}</span>
            </div>
          ) : gmailCom ? (
            <div className="note w" role="note">
              <Icon name="alertCircle" size={16} className="mt-px flex-none text-warn-bar" />
              <div className="flex flex-col gap-1">
                <span className="font-bold">{t(m.gmailOAuthTitle)}</span>
                <span>{t(m.gmailOAuthBody)}</span>
                <Button size="sm" className="mt-1.5 w-fit" onClick={() => setMode('imap')}>
                  {t(m.useAppPassword)}
                </Button>
              </div>
            </div>
          ) : oauthReady ? (
            <div className="rounded-lg border border-line bg-ground-3 px-3.5 py-3 text-[13px] text-ink-nav">
              <T msg={provider.oauth === 'microsoft' ? m.oauthMicrosoft : m.oauthGoogle} values={{ link: (c) => <InstanceAuthLink>{c}</InstanceAuthLink> }} />
            </div>
          ) : (
            <div className="note w" role="note">
              <Icon name="alertCircle" size={16} className="mt-px flex-none text-warn-bar" />
              <span>
                <T msg={m.oauthNotSetUp} values={{ app: appName(provider), link: (c) => <InstanceAuthLink>{c}</InstanceAuthLink> }} />
              </span>
            </div>
          )}
          {oauthNote || gmailCom ? null : (
            <div className="note i" role="note">
              <Icon name="info" size={16} className="mt-px flex-none" />
              <span>{t(m.testOnly)}</span>
            </div>
          )}
        </>
      ) : null}
      {provider && mode === 'imap' ? (
        <>
          {provider.oauth === 'microsoft' && !done ? (
            <div className="note w" role="note">
              <Icon name="alertCircle" size={16} className="mt-px flex-none text-warn-bar" />
              <div className="flex flex-col gap-1">
                <span>{t(m.microsoftImap)}</span>
                <Button size="sm" className="mt-1.5 w-fit" onClick={() => setMode('oauth')}>
                  {t(m.useOAuth)}
                </Button>
              </div>
            </div>
          ) : null}
          <div className="grid gap-2.5" style={{ gridTemplateColumns: 'minmax(0,1fr) 128px' }}>
            <Field label={t(m.imapServer)} id={ids.host}>
              <input id={ids.host} className="in" value={host} onChange={(e) => setHost(e.target.value)} disabled={Boolean(done)} autoComplete="off" spellCheck={false} />
            </Field>
            <Field label={t(m.port)} id={ids.port}>
              <input id={ids.port} className="in" inputMode="numeric" value={port} onChange={(e) => setPort(e.target.value.replace(/\D/g, ''))} disabled={Boolean(done)} />
            </Field>
          </div>
          <Field label={provider.usernameLabel === 'Apple ID' ? t(m.appleId) : t(m.emailAddress)} id={ids.email}>
            <input id={ids.email} className="in" type="email" value={done ? done.email : email} onChange={(e) => setEmail(e.target.value)} disabled={Boolean(done)} autoComplete="off" spellCheck={false} />
          </Field>
          {done ? null : (
            <Field label={provider.password} id={ids.password} hint={t(m.passwordHint)}>
              <input
                id={ids.password}
                className={error ? 'in border-bad-strong!' : 'in'}
                type="password"
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                autoComplete="new-password"
                aria-invalid={error ? true : undefined}
              />
            </Field>
          )}
          {provider.passwordHelp && !done && !error ? (
            <a href={provider.passwordHelp} target="_blank" rel="noopener noreferrer" className="pl-lnk w-fit">
              {t(m.passwordHelp, { password: provider.password.toLowerCase(), company: provider.company })}
              <ExternalIcon />
            </a>
          ) : null}
          {error ? (
            <div className="note b" role="alert">
              <Icon name="alertCircle" size={16} className="mt-px flex-none" />
              <div className="flex flex-col gap-2">
                <span>{error}</span>
                {provider.imapHelp ? (
                  <ButtonAnchor size="sm" href={provider.imapHelp} className="w-fit">
                    {t(m.imapHelp, { provider: provider.name })}
                  </ButtonAnchor>
                ) : null}
              </div>
            </div>
          ) : null}
          {done ? (
            <div className="note g" role="status">
              <Icon name="check" size={16} className="mt-px flex-none" />
              <div className="flex flex-col gap-1">
                <span className="font-bold">{t(m.checkPassed)}</span>
                <span>{signedIn(done.email, done.folders)}</span>
              </div>
            </div>
          ) : null}
        </>
      ) : null}
    </Dialog>
  )
}

/** Reconnect a seed whose sign-in failed: a new password, or OAuth again. */
export function ReconnectDialog({ d, seed, open, onOpenChange }: { d: SeedList; seed: SeedView; open: boolean; onOpenChange: (o: boolean) => void }) {
  const t = useT()
  const { slug } = useWorkspace()
  const refresh = useRefreshPlacement()
  const p = d.providers.find((x) => x.key === seed.provider)
  const ids = { host: useId(), port: useId(), user: useId(), password: useId() }
  const [host, setHost] = useState(seed.host ?? p?.host ?? '')
  const [port, setPort] = useState(String(seed.port ?? p?.port ?? 993))
  const [password, setPassword] = useState('')
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState<string>()
  const oauth = seed.auth === 'oauth'
  const at = seed.statusAt ?? seed.checkedAt ?? seed.addedAt
  const submit = async () => {
    setBusy(true)
    setError(undefined)
    try {
      if (oauth && p) {
        await startOAuth(slug, p.key, seed.id)
        return
      }
      const res = await http.POST(`${base}/seeds/{seed}/reconnect`, { params: { path: { workspace: slug, seed: seed.id } }, body: { password, host: host.trim(), port: Number(port) || 993 } })
      if (!res.data) throw new Error(errorMessage(res.error))
      void refresh()
      onOpenChange(false)
      toast({ kind: 'success', title: t(m.reconnected, { email: seed.email }) })
    } catch (e) {
      setError(errorMessage(e))
    } finally {
      setBusy(false)
    }
  }
  const label = p?.usernameLabel === 'Apple ID' ? t(m.appleId) : t(m.emailAddress)
  const who = p?.usernameLabel === 'Apple ID' ? t(m.appleId) : t(m.emailAddressLower)
  const app = p?.oauth === 'microsoft' ? 'Microsoft' : 'Google'
  return (
    <Dialog
      open={open}
      onOpenChange={(o) => !busy && onOpenChange(o)}
      eyebrow={<span className="k">{t(m.dialogCrumb)}</span>}
      title={t(m.reconnectTitle, { email: seed.email })}
      footer={
        <>
          <Button className="mr-auto" onClick={() => onOpenChange(false)}>
            {t(m.cancel)}
          </Button>
          <Button variant="primary" onClick={() => void submit()} loading={busy} busyLabel={t(m.checking)} disabledReason={busy || oauth || password ? undefined : t(m.fillFields)}>
            {oauth ? (
              <>
                <ExternalIcon />
                {t(m.reconnectWith, { app })}
              </>
            ) : (
              t(m.checkAndSave)
            )}
          </Button>
        </>
      }
    >
      {seed.status === 'failed' || seed.status === 'expired' ? (
        <div className="note b" role="note">
          <Icon name="alertCircle" size={16} className="mt-px flex-none" />
          <span>
            {seed.status === 'expired'
              ? t(m.reconnectExpiredBody, { company: seed.company, date: shortDate(at), time: clock(at) })
              : t(m.reconnectFailedBody, { company: seed.company, password: seed.passwordLabel.toLowerCase(), date: shortDate(at), time: clock(at), usernameLabel: who })}
          </span>
        </div>
      ) : (
        <span className="text-ink-2">{t(m.reconnectBody, { password: seed.passwordLabel.toLowerCase(), usernameLabel: who })}</span>
      )}
      {oauth ? null : (
        <>
          <div className="grid gap-2.5" style={{ gridTemplateColumns: 'minmax(0,1fr) 128px' }}>
            <Field label={t(m.imapServer)} id={ids.host}>
              <input id={ids.host} className="in" value={host} onChange={(e) => setHost(e.target.value)} autoComplete="off" spellCheck={false} />
            </Field>
            <Field label={t(m.port)} id={ids.port}>
              <input id={ids.port} className="in" inputMode="numeric" value={port} onChange={(e) => setPort(e.target.value.replace(/\D/g, ''))} />
            </Field>
          </div>
          <Field label={label} id={ids.user}>
            <input id={ids.user} className="in" value={seed.username || seed.email} readOnly />
          </Field>
          <Field label={t(m.newPassword, { password: seed.passwordLabel.toLowerCase() })} id={ids.password}>
            <input
              id={ids.password}
              className={error ? 'in border-bad-strong!' : 'in'}
              type="password"
              placeholder={'xxxx-xxxx-xxxx-xxxx'}
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              autoComplete="new-password"
              aria-invalid={error ? true : undefined}
            />
          </Field>
          {p?.passwordHelp ? (
            <a href={p.passwordHelp} target="_blank" rel="noopener noreferrer" className="pl-lnk w-fit">
              {t(m.passwordHelp, { password: seed.passwordLabel.toLowerCase(), company: seed.company })}
              <ExternalIcon />
            </a>
          ) : null}
        </>
      )}
      {error ? (
        <div className="note b" role="alert">
          <Icon name="alertCircle" size={16} className="mt-px flex-none" />
          <span>{error}</span>
        </div>
      ) : null}
    </Dialog>
  )
}

/** Remove a seed: its password or tokens go, past results keep their row. */
export function RemoveSeedDialog({ seed, open, onOpenChange }: { seed: SeedView; open: boolean; onOpenChange: (o: boolean) => void }) {
  const t = useT()
  const { slug } = useWorkspace()
  const refresh = useRefreshPlacement()
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState<string>()
  const remove = async () => {
    setBusy(true)
    setError(undefined)
    const res = await http.DELETE(`${base}/seeds/{seed}`, { params: { path: { workspace: slug, seed: seed.id } } })
    setBusy(false)
    if (res.error) {
      setError(errorMessage(res.error))
      return
    }
    void refresh()
    onOpenChange(false)
    toast({ kind: 'deleted', title: t(m.removed, { email: seed.email }) })
  }
  const auth = seed.auth === 'oauth' ? t(m.connOAuth) : t(m.connImap, { password: seed.passwordLabel.toLowerCase() })
  return (
    <Dialog
      open={open}
      onOpenChange={(o) => !busy && onOpenChange(o)}
      layout="confirm"
      dismissible={false}
      eyebrow={<span className="k">{t(m.removeEyebrow, { provider: seed.providerName, auth })}</span>}
      title={t(m.removeTitle, { email: seed.email })}
      description={seed.auth === 'oauth' ? t(m.removeBodyOAuth) : t(m.removeBodyPassword, { password: seed.passwordLabel.toLowerCase() })}
      footer={
        <>
          <Button onClick={() => onOpenChange(false)} disabled={busy}>
            {t(m.cancel)}
          </Button>
          <Button variant="danger-primary" onClick={() => void remove()} loading={busy}>
            {t(m.removeConfirm)}
          </Button>
        </>
      }
    >
      <span className="k">{t(m.removeKeepRow, { provider: seed.providerName })}</span>
      {error ? (
        <div className="note b" role="alert">
          <span>{error}</span>
        </div>
      ) : null}
    </Dialog>
  )
}
