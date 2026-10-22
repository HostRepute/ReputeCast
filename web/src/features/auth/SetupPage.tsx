import { useQuery, useQueryClient } from '@tanstack/react-query'
import { Link, useNavigate } from '@tanstack/react-router'
import { useEffect, useId, useRef, useState, type FormEvent, type ReactNode } from 'react'
import { ApiError, errorMessage, http, type Schemas } from '@/api/client'
import { LogoMark } from '@/components/icons/Icon'
import { useInstanceMeta } from '@/components/shell/AuthLayout'
import { Badge } from '@/components/ui/Badge'
import { Button, ButtonLink } from '@/components/ui/Button'
import { Skeleton } from '@/components/ui/Feedback'
import { Note } from '@/components/ui/Note'
import { toast } from '@/components/ui/Toast'
import { InstanceLanguageFields } from '@/features/language/InstanceLanguageFields'
import { useMemberLanguage } from '@/features/language/useLanguage'
import { T, t as tt, useT } from '@/i18n'
import { common } from '@/i18n/common'
import { cn } from '@/lib/cn'
import { useDocumentTitle } from '@/lib/hooks'
import { meQueryKey } from '@/lib/session'
import { PasskeyGlyph } from './AuthParts'
import { passkeyErrorMessage, registerPasskey } from './passkey'
import { RecoveryCodesPanel } from './RecoveryCodes'
import { m } from './setup.messages'
import { browserTimezone, timezoneLabel, timezoneOptions } from './timezones'
import { TotpSetupForm } from './TotpSetup'
import './auth.css'

type State = Schemas['SetupState']
type StepKey = Schemas['SetupStep']['key']

const setupKey = ['setup'] as const

/** The command that creates the install Owner; never translated. */
const INSTALL_COMMAND = 'reputecast install'

const mono = (c: ReactNode) => <span className="mono">{c}</span>

function hostOf(u?: string) {
  try {
    return u ? new URL(u).host : ''
  } catch {
    return ''
  }
}

/**
 * A5 §5.5: the install Owner's first sign-in (InstanceSetup board), from the
 * temporary password and a second factor (can't be skipped) to the first
 * sending provider. People who must finish personal steps (temporary
 * password, required two-factor) get the same screen with only those steps.
 */
export function SetupPage() {
  const t = useT()
  const navigate = useNavigate()
  const qc = useQueryClient()
  const { data: meta } = useInstanceMeta()
  const state = useQuery({ queryKey: setupKey, queryFn: async () => (await http.GET('/api/v1/setup')).data! })
  // Codes from the authenticator step, shown once in the next step.
  const [codes, setCodes] = useState<string[]>()
  const [created, setCreated] = useState<Schemas['CreatedWorkspace']>()
  const [back, setBack] = useState<StepKey>()
  useDocumentTitle(t(m.documentTitle, { name: meta?.instanceName ?? 'ReputeCast' }))

  const refresh = async () => {
    setBack(undefined)
    await qc.invalidateQueries({ queryKey: setupKey })
    await qc.invalidateQueries({ queryKey: meQueryKey })
  }

  const done = state.data?.mode === 'done' && !created
  useEffect(() => {
    if (done) void navigate({ to: state.data?.next ?? '/' })
  }, [done, navigate, state.data?.next])

  if (state.isError) {
    const signedOut = state.error instanceof ApiError && state.error.status === 401
    return (
      <Frame>
        <section className="card" style={{ gridColumn: '1 / -1' }}>
          <h1 className="text-[22px] font-bold tracking-[-.02em]">{signedOut ? t(m.signInToContinue) : t(m.loadFailed)}</h1>
          <Note tone={signedOut ? 'info' : 'bad'} label={signedOut ? t(m.signedOut) : t(m.error)}>
            {signedOut ? t(m.sessionEnded) : `${errorMessage(state.error)} ${t(m.nothingChanged)}`}
          </Note>
          <div>
            {signedOut ? (
              <ButtonLink variant="primary" to="/sign-in" search={{ next: '/setup' }}>
                {t(m.signIn)}
              </ButtonLink>
            ) : (
              <Button variant="primary" onClick={() => state.refetch()}>
                {t(common.tryAgain)}
              </Button>
            )}
          </div>
        </section>
      </Frame>
    )
  }
  if (state.isPending || done) return <SetupSkeleton />

  const s = state.data
  const steps = s.steps
  const current: StepKey = created ? 'provider' : (back ?? (s.current as StepKey) ?? steps[0]?.key ?? 'provider')
  const index = Math.max(0, steps.findIndex((x) => x.key === current))
  const instanceMode = s.mode === 'instance'
  const name = instanceMode ? s.instance.name : meta?.instanceName
  const stepHeader = (title: string, sub: ReactNode, id = 'h-step') => (
    <div className="flex flex-col gap-1">
      <span className="k">{t(m.stepOf, { n: index + 1, total: steps.length })}</span>
      <h2 id={id} className="text-[22px] font-bold tracking-[-.02em]">
        {title}
      </h2>
      <span className="k">{sub}</span>
    </div>
  )

  let body: ReactNode
  switch (current) {
    case 'password':
      body = <PasswordStep s={s} header={stepHeader} onDone={refresh} />
      break
    case 'second_factor':
      body = (
        <SecondFactorStep
          s={s}
          header={stepHeader}
          onTotp={(c) => {
            setCodes(c)
            void refresh()
          }}
          onPasskey={refresh}
        />
      )
      break
    case 'recovery_codes':
      body = (
        <RecoveryStep
          s={s}
          header={stepHeader}
          codes={codes}
          onCodes={setCodes}
          onDone={async () => {
            setCodes(undefined)
            if (instanceMode) await refresh()
            else {
              await qc.invalidateQueries({ queryKey: meQueryKey })
              toast({ kind: 'success', title: t(m.twoFactorOn), detail: t(m.accountProtected) })
              void navigate({ to: '/' })
            }
          }}
        />
      )
      break
    case 'instance':
      body = <InstanceStep s={s} header={stepHeader} onBack={() => setBack('recovery_codes')} onDone={refresh} />
      break
    case 'workspace':
      body = (
        <WorkspaceStep
          s={s}
          header={stepHeader}
          onBack={() => setBack('instance')}
          onDone={async (ws) => {
            setCreated(ws)
            await qc.invalidateQueries({ queryKey: meQueryKey })
          }}
        />
      )
      break
    case 'provider':
      body = <ProviderStep header={stepHeader} workspace={created ?? s.workspace} instance={name ?? 'ReputeCast'} />
      break
  }

  const doneItems: string[] = []
  if (instanceMode) {
    if (!s.password.mustChange) doneItems.push(t(m.doneTempPassword))
    if (s.secondFactor.passkeys.length) doneItems.push(t(m.donePasskey, { name: s.secondFactor.passkeys[0]! }))
    if (s.secondFactor.totp) doneItems.push(t(m.doneAuthenticator))
    if (steps.find((x) => x.key === 'recovery_codes')?.done) doneItems.push(t(m.doneRecovery, { count: s.recoveryCodes.remaining }))
    if (s.instance.confirmed) doneItems.push(t(m.doneInstance))
    if (created ?? s.workspace) doneItems.push(t(m.doneWorkspace, { name: (created ?? s.workspace)!.name }))
  }

  return (
    <Frame>
      <aside className="flex flex-col gap-4" aria-label={t(m.progressLabel)}>
        <section className="card" style={{ gap: 12 }}>
          <div className="brand">
            <LogoMark size={28} />
            <span>
              {name ?? 'ReputeCast'} <small>{hostOf(s.instance.publicUrl)}</small>
            </span>
          </div>
          <div className="flex flex-col gap-0.5">
            <h1 className="text-[17px] font-bold tracking-[-.01em]">{instanceMode ? t(m.setUpName, { name }) : t(m.finishAccount)}</h1>
            <span className="k">
              {instanceMode ? (
                <T msg={m.signedInOwner} values={{ email: s.email, command: INSTALL_COMMAND, mono }} />
              ) : (
                t(m.signedInReason, { email: s.email, reason: s.secondFactor.reason ?? '' })
              )}
            </span>
          </div>
          <nav className="steps" aria-label={t(m.stepsLabel)}>
            {steps.map((st, i) => {
              const isDone = st.done || (created !== undefined && st.key !== 'provider')
              const on = st.key === current
              return (
                <span key={st.key} className={cn('st', isDone && 'done', on && 'on')} aria-current={on ? 'step' : undefined}>
                  <span className="n" aria-hidden="true">
                    {isDone ? '✓' : i + 1}
                  </span>
                  <span className="t">
                    {t(m.stepLabel, { step: st.key, label: st.label })}{' '}
                    {isDone ? <span className="vh">{t(m.stepCompleted)}</span> : on ? <span className="vh">{t(m.stepCurrent)}</span> : null}
                  </span>
                </span>
              )
            })}
          </nav>
        </section>
        {instanceMode && doneItems.length > 0 ? (
          <section className="card" style={{ gap: 8 }}>
            <h2 className="text-[13px] font-semibold text-ink-2">{t(m.doneSoFar)}</h2>
            {doneItems.map((d) => (
              <div className="chk-row" key={d}>
                <Badge tone="ok">✓</Badge>
                {d}
              </div>
            ))}
            <span className="k">{t(m.firstOwnerRequired)}</span>
          </section>
        ) : null}
      </aside>
      <section className="card" aria-labelledby="h-step" style={{ gap: 18 }}>
        {body}
      </section>
    </Frame>
  )
}

function Frame({ children }: { children: ReactNode }) {
  return (
    <div className="rc-app min-h-dvh">
      <main className="setup" id="main">
        {children}
      </main>
    </div>
  )
}

function SetupSkeleton() {
  const t = useT()
  return (
    <Frame>
      <aside className="flex flex-col gap-4" aria-busy="true">
        <section className="card" style={{ gap: 12 }}>
          <Skeleton w={180} h={28} />
          <Skeleton w="80%" h={17} />
          {[0, 1, 2, 3, 4, 5].map((i) => (
            <Skeleton key={i} w="90%" h={22} />
          ))}
        </section>
      </aside>
      <section className="card" style={{ gap: 18 }} aria-busy="true">
        <span className="vh" role="status">
          {t(m.loading)}
        </span>
        <Skeleton w={80} h={12} />
        <Skeleton w="40%" h={24} />
        <Skeleton w="100%" h={40} r={8} />
        <Skeleton w="100%" h={40} r={8} />
      </section>
    </Frame>
  )
}

type Header = (title: string, sub: ReactNode, id?: string) => ReactNode

function Footer({ children }: { children: ReactNode }) {
  return <div className="flex flex-wrap justify-between gap-2.5 border-t border-line-2 pt-1.5">{children}</div>
}

function PasswordStep({ s, header, onDone }: { s: State; header: Header; onDone: () => Promise<void> }) {
  const t = useT()
  const [current, setCurrent] = useState('')
  const [next, setNext] = useState('')
  const [confirm, setConfirm] = useState('')
  const [busy, setBusy] = useState(false)
  const [errors, setErrors] = useState<{ current?: string; password?: string; confirm?: string; form?: string }>({})
  const ids = { cur: useId(), pw: useId(), cf: useId(), hint: useId() }
  const submit = async (e: FormEvent) => {
    e.preventDefault()
    const errs: typeof errors = {}
    if (!current) errs.current = t(m.tempPasswordRequired)
    if ([...next].length < s.password.minLength) errs.password = t(m.passwordTooShort, { min: s.password.minLength })
    if (confirm !== next) errs.confirm = t(m.passwordsDiffer)
    setErrors(errs)
    if (Object.keys(errs).length) return
    setBusy(true)
    try {
      await http.POST('/api/v1/me/password', { body: { current, password: next } })
      await onDone()
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
    <form className="flex flex-col gap-[18px]" onSubmit={submit} noValidate>
      {header(t(m.passwordTitle), <T msg={m.passwordSub} values={{ command: INSTALL_COMMAND, mono }} />)}
      {errors.form ? (
        <Note tone="bad" label={t(m.notSaved)}>
          {errors.form}
        </Note>
      ) : null}
      <input type="email" name="email" autoComplete="username" value={s.email} readOnly hidden />
      <div className="lbl">
        <label htmlFor={ids.cur}>{t(m.tempPassword)}</label>
        <input id={ids.cur} className={errors.current ? 'in err' : 'in'} type="password" autoComplete="current-password" value={current} onChange={(e) => setCurrent(e.target.value)} autoFocus aria-invalid={errors.current ? true : undefined} />
        {errors.current ? (
          <span className="err" role="alert">
            {errors.current}
          </span>
        ) : null}
      </div>
      <div className="two">
        <div className="lbl">
          <label htmlFor={ids.pw}>{t(m.newPassword)}</label>
          <input
            id={ids.pw}
            className={errors.password ? 'in err' : 'in'}
            type="password"
            autoComplete="new-password"
            value={next}
            onChange={(e) => setNext(e.target.value)}
            aria-invalid={errors.password ? true : undefined}
            aria-describedby={ids.hint}
          />
          {errors.password ? (
            <span className="err" role="alert">
              {errors.password}
            </span>
          ) : null}
          <span className="k font-medium" id={ids.hint}>
            {s.password.breachedCheck ? t(m.passwordHintBreached, { min: s.password.minLength }) : t(m.passwordHint, { min: s.password.minLength })}
          </span>
        </div>
        <div className="lbl">
          <label htmlFor={ids.cf}>{t(m.confirmPassword)}</label>
          <input id={ids.cf} className={errors.confirm ? 'in err' : 'in'} type="password" autoComplete="new-password" value={confirm} onChange={(e) => setConfirm(e.target.value)} aria-invalid={errors.confirm ? true : undefined} />
          {errors.confirm ? (
            <span className="err" role="alert">
              {errors.confirm}
            </span>
          ) : null}
        </div>
      </div>
      <Footer>
        <span />
        <Button type="submit" variant="primary" loading={busy}>
          {t(m.saveContinue)}
        </Button>
      </Footer>
    </form>
  )
}

function SecondFactorStep({ s, header, onTotp, onPasskey }: { s: State; header: Header; onTotp: (codes: string[]) => void; onPasskey: () => Promise<void> }) {
  const t = useT()
  const [mode, setMode] = useState<'choose' | 'totp'>('choose')
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState<string>()
  const required = s.mode === 'instance' ? t(m.firstOwnerRequired) : s.secondFactor.reason
  if (mode === 'totp') {
    return (
      <div className="flex flex-col gap-[18px]">
        {header(t(m.totpTitle), t(m.totpSub))}
        <TotpSetupForm onConfirmed={onTotp} onCancel={() => setMode('choose')} cancelLabel={t(common.back)} />
      </div>
    )
  }
  return (
    <div className="flex flex-col gap-[18px]">
      {header(
        t(m.secondFactorTitle),
        <>
          {required} {t(m.passkeyQuickest)}
        </>,
      )}
      {error ? (
        <span className="err" role="alert">
          {error}
        </span>
      ) : null}
      <div className="flex flex-col gap-2">
        <Button
          variant="primary"
          loading={busy}
          onClick={async () => {
            setBusy(true)
            setError(undefined)
            try {
              const pk = await registerPasskey()
              toast({ kind: 'success', title: t(m.passkeyAdded), detail: pk.name })
              await onPasskey()
            } catch (e) {
              setError(passkeyErrorMessage(e, 'add'))
            } finally {
              setBusy(false)
            }
          }}
        >
          {busy ? null : <PasskeyGlyph />}
          {t(m.addPasskey)}
        </Button>
        <Button onClick={() => setMode('totp')}>{t(m.setUpAuthenticator)}</Button>
      </div>
      <span className="k">{t(m.changeLater)}</span>
    </div>
  )
}

function RecoveryStep({ s, header, codes, onCodes, onDone }: { s: State; header: Header; codes?: string[]; onCodes: (c: string[]) => void; onDone: () => Promise<void> }) {
  const t = useT()
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState<string>()
  const sub = t(m.recoverySub)
  if (!codes) {
    return (
      <div className="flex flex-col gap-[18px]">
        {header(t(m.recoveryTitle), sub)}
        {s.recoveryCodes.remaining > 0 ? (
          <Note tone="info" label={t(m.shownOnce)}>
            {t(m.earlierCodes)}
          </Note>
        ) : null}
        {error ? (
          <span className="err" role="alert">
            {error}
          </span>
        ) : null}
        <div>
          <Button
            variant="primary"
            loading={busy}
            onClick={async () => {
              setBusy(true)
              setError(undefined)
              try {
                const { data } = await http.POST('/api/v1/me/recovery-codes')
                onCodes(data?.codes ?? [])
              } catch (e) {
                setError(errorMessage(e))
              } finally {
                setBusy(false)
              }
            }}
          >
            {t(m.generateCodes)}
          </Button>
        </div>
      </div>
    )
  }
  return (
    <div className="flex flex-col gap-[18px]">
      {header(t(m.recoveryTitle), sub)}
      <RecoveryCodesPanel
        codes={codes}
        email={s.email}
        busy={busy}
        finishLabel={t(m.saveContinue)}
        onFinish={async () => {
          setBusy(true)
          setError(undefined)
          try {
            await http.POST('/api/v1/me/recovery-codes/saved')
            await onDone()
          } catch (e) {
            setError(errorMessage(e))
          } finally {
            setBusy(false)
          }
        }}
      />
      {error ? (
        <span className="err" role="alert">
          {error}
        </span>
      ) : null}
    </div>
  )
}

const checkBadge: Record<string, 'ok' | 'warn' | 'bad' | 'neutral'> = { ok: 'ok', warn: 'warn', bad: 'bad', skipped: 'neutral' }

function secondsAgo(at: string, now: number) {
  const s = Math.max(1, Math.round((now - new Date(at).getTime()) / 1000))
  if (s < 60) return tt(m.secondsAgo, { n: s })
  return tt(m.minutesAgo, { n: Math.round(s / 60) })
}

/** The IP and host parts of a check sentence in mono, as on the board. */
function CheckDetail({ text }: { text: string }) {
  const parts = text.split(/(\b\d{1,3}(?:\.\d{1,3}){3}\b|\b[0-9a-f:]{6,}\b)/i)
  return (
    <span>
      {parts.map((p, i) =>
        i % 2 === 1 ? (
          <span className="mono" key={i}>
            {p}
          </span>
        ) : (
          p
        ),
      )}
    </span>
  )
}

function InstanceStep({ s, header, onBack, onDone }: { s: State; header: Header; onBack: () => void; onDone: () => Promise<void> }) {
  const t = useT()
  const [name, setName] = useState(s.instance.name === 'ReputeCast' ? '' : s.instance.name)
  // Until this step is saved the instance has the install default (UTC);
  // the Owner's own timezone is the likelier choice.
  const saved = s.steps.find((x) => x.key === 'instance')?.done
  const [tz, setTz] = useState(() => (!saved && (!s.instance.timezone || s.instance.timezone === 'UTC') ? browserTimezone() : s.instance.timezone) || 'UTC')
  const [url, setUrl] = useState(s.instance.publicUrl)
  const [updateCheck, setUpdateCheck] = useState(s.instance.updateCheck)
  const { inst } = useMemberLanguage()
  const [language, setLanguage] = useState(inst.defaultLanguage)
  const [check, setCheck] = useState<Schemas['URLCheck']>()
  const [checking, setChecking] = useState(false)
  const [busy, setBusy] = useState(false)
  const [errors, setErrors] = useState<{ name?: string; url?: string; form?: string }>({})
  const [now, setNow] = useState(() => Date.now())
  const ids = { name: useId(), tz: useId(), url: useId(), upd: useId() }
  const ran = useRef(false)

  const runCheck = async (value: string) => {
    setChecking(true)
    try {
      const { data } = await http.POST('/api/v1/setup/check-url', { body: { url: value } })
      setCheck(data)
      setErrors((e) => ({ ...e, url: undefined }))
    } catch (err) {
      setErrors((e) => ({ ...e, url: errorMessage(err) }))
    } finally {
      setChecking(false)
      setNow(Date.now())
    }
  }
  useEffect(() => {
    if (!ran.current && url) {
      ran.current = true
      void runCheck(url)
    }
    const h = window.setInterval(() => setNow(Date.now()), 5000)
    return () => window.clearInterval(h)
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [])

  const submit = async (e: FormEvent) => {
    e.preventDefault()
    const errs: typeof errors = {}
    if (!name.trim()) errs.name = t(m.instanceNameRequired)
    if (!/^https?:\/\/[^\s/]+/.test(url.trim())) errs.url = t(m.urlInvalid)
    setErrors(errs)
    if (errs.name || errs.url) return
    setBusy(true)
    try {
      await http.PUT('/api/v1/setup/instance', {
        body: { name: name.trim(), publicUrl: url.trim(), timezone: tz, updateCheck, defaultLanguage: language as Schemas['InstanceBasics']['defaultLanguage'] },
      })
      await onDone()
    } catch (err) {
      const f = err instanceof ApiError ? err.problem.errors?.[0] : undefined
      if (f?.location === 'body.publicUrl') setErrors({ url: f.message })
      else if (f?.location === 'body.name') setErrors({ name: f.message })
      else setErrors({ form: errorMessage(err) })
    } finally {
      setBusy(false)
    }
  }

  const rows: [string, Schemas['CheckLine']][] = check
    ? [
        ['DNS', check.dns],
        ['TLS', check.tls],
        [t(m.checkReachable), check.reachable],
      ]
    : []
  return (
    <form className="flex flex-col gap-[18px]" onSubmit={submit} noValidate>
      {header(t(m.instanceTitle), t(m.instanceSub), 'h-basics')}
      {errors.form ? (
        <Note tone="bad" label={t(m.notSaved)}>
          {errors.form}
        </Note>
      ) : null}
      <div className="two">
        <div className="lbl">
          <label htmlFor={ids.name}>{t(m.instanceName)}</label>
          <input id={ids.name} className={errors.name ? 'in err' : 'in'} value={name} placeholder={'Acme ReputeCast'} onChange={(e) => setName(e.target.value)} aria-invalid={errors.name ? true : undefined} />
          {errors.name ? (
            <span className="err" role="alert">
              {errors.name}
            </span>
          ) : null}
          <span className="k font-medium">{t(m.instanceNameHint)}</span>
        </div>
        <div className="lbl">
          <label htmlFor={ids.tz}>{t(m.defaultTimezone)}</label>
          <select id={ids.tz} className="in" value={tz} onChange={(e) => setTz(e.target.value)}>
            {timezoneOptions(tz).map((z) => (
              <option key={z} value={z}>
                {timezoneLabel(z)}
              </option>
            ))}
          </select>
          <span className="k font-medium">{t(m.defaultTimezoneHint)}</span>
        </div>
      </div>
      {/* The first sign-in asks for the default language (connect-and-reach brief §6). */}
      <InstanceLanguageFields language={language} region="" onLanguage={setLanguage} onRegion={() => {}} showRegion={false} />
      <div className="lbl">
        <label htmlFor={ids.url}>{t(m.publicUrl)}</label>
        <div className="flex gap-2">
          <input id={ids.url} className={cn('in mono flex-1', errors.url && 'err')} value={url} onChange={(e) => setUrl(e.target.value)} inputMode="url" aria-invalid={errors.url ? true : undefined} />
          <Button loading={checking} onClick={() => runCheck(url.trim())}>
            {t(m.checkAgain)}
          </Button>
        </div>
        {errors.url ? (
          <span className="err" role="alert">
            {errors.url}
          </span>
        ) : null}
        <span className="k font-medium">{t(m.publicUrlHint)}</span>
      </div>
      {check ? (
        <div className="card flex flex-col gap-2 bg-ground-3 px-3.5 py-3" role="status" aria-label={t(m.urlCheckLabel)}>
          <span className="text-[12px] font-semibold text-ink-2">{checking ? t(m.checking) : t(m.checked, { ago: secondsAgo(check.checkedAt, now) })}</span>
          {rows.map(([label, line]) => (
            <div className="chk-row" key={label}>
              <Badge tone={checkBadge[line.status] ?? 'neutral'}>{label}</Badge>
              <CheckDetail text={line.detail} />
            </div>
          ))}
        </div>
      ) : checking ? (
        <div className="card flex flex-col gap-2 bg-ground-3 px-3.5 py-3" aria-busy="true">
          <span className="text-[12px] font-semibold text-ink-2">{t(m.checkingHost, { host: hostOf(url) })}</span>
          <Skeleton w="70%" h={14} />
          <Skeleton w="60%" h={14} />
          <Skeleton w="75%" h={14} />
        </div>
      ) : null}
      <div className="chk-row items-start">
        <input type="checkbox" id={ids.upd} className="m-[2px_3px_3px_4px] cursor-pointer" checked={updateCheck} onChange={(e) => setUpdateCheck(e.target.checked)} aria-describedby={`${ids.upd}-hint`} />
        <span className="flex flex-col gap-0.5">
          <label htmlFor={ids.upd} className="cursor-pointer font-semibold">
            {t(m.updateCheck)}
          </label>
          <span className="k" id={`${ids.upd}-hint`}>
            {t(m.updateCheckHint)}
          </span>
        </span>
      </div>
      <Footer>
        <Button onClick={onBack}>{t(common.back)}</Button>
        <Button type="submit" variant="primary" loading={busy}>
          {t(m.saveCreateWorkspace)}
        </Button>
      </Footer>
    </form>
  )
}

function WorkspaceStep({ s, header, onBack, onDone }: { s: State; header: Header; onBack: () => void; onDone: (ws: Schemas['CreatedWorkspace']) => Promise<void> }) {
  const t = useT()
  const [name, setName] = useState('')
  const [tz, setTz] = useState(s.instance.timezone || 'UTC')
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState<string>()
  const [formError, setFormError] = useState<string>()
  const ids = { name: useId(), tz: useId() }
  const submit = async (e: FormEvent) => {
    e.preventDefault()
    if (!name.trim()) {
      setError(t(m.workspaceNameRequired))
      return
    }
    setBusy(true)
    setError(undefined)
    setFormError(undefined)
    try {
      const { data } = await http.POST('/api/v1/workspaces', { body: { name: name.trim(), timezone: tz } })
      if (data) await onDone(data)
    } catch (err) {
      const f = err instanceof ApiError ? err.problem.errors?.[0] : undefined
      if (f?.location === 'body.name') setError(f.message)
      else setFormError(errorMessage(err))
    } finally {
      setBusy(false)
    }
  }
  return (
    <form className="flex flex-col gap-[18px]" onSubmit={submit} noValidate>
      {header(t(m.workspaceTitle), t(m.workspaceSub))}
      {formError ? (
        <Note tone="bad" label={t(m.notCreated)}>
          {formError}
        </Note>
      ) : null}
      <div className="two">
        <div className="lbl">
          <label htmlFor={ids.name}>{t(m.workspaceName)}</label>
          <input id={ids.name} className={error ? 'in err' : 'in'} value={name} placeholder={t(m.workspaceNamePlaceholder)} onChange={(e) => setName(e.target.value)} autoFocus aria-invalid={error ? true : undefined} />
          {error ? (
            <span className="err" role="alert">
              {error}
            </span>
          ) : null}
          <span className="k font-medium">{t(m.workspaceNameHint)}</span>
        </div>
        <div className="lbl">
          <label htmlFor={ids.tz}>{t(m.timezone)}</label>
          <select id={ids.tz} className="in" value={tz} onChange={(e) => setTz(e.target.value)}>
            {timezoneOptions(tz).map((z) => (
              <option key={z} value={z}>
                {timezoneLabel(z)}
              </option>
            ))}
          </select>
          <span className="k font-medium">{t(m.timezoneHint)}</span>
        </div>
      </div>
      <Footer>
        <Button onClick={onBack}>{t(common.back)}</Button>
        <Button type="submit" variant="primary" loading={busy}>
          {t(m.createWorkspace)}
        </Button>
      </Footer>
    </form>
  )
}

function ProviderStep({ header, workspace, instance }: { header: Header; workspace?: Schemas['CreatedWorkspace']; instance: string }) {
  const t = useT()
  if (!workspace) {
    return (
      <div className="flex flex-col gap-[18px]">
        {header(t(m.providerTitle), t(m.providerNeedsWorkspace))}
        <Link className="btn" to="/">
          {t(m.continue)}
        </Link>
      </div>
    )
  }
  return (
    <div className="flex flex-col gap-[18px]">
      {header(t(m.providerTitle), t(m.providerSub))}
      <Note tone="ok" label={t(m.ready)}>
        {t(m.readyBody, { instance, workspace: workspace.name })}
      </Note>
      <Footer>
        <ButtonLink to="/w/$workspace" params={{ workspace: workspace.slug }}>
          {t(m.skipForNow)}
        </ButtonLink>
        <ButtonLink variant="primary" to="/w/$workspace/providers/connect" params={{ workspace: workspace.slug }}>
          {t(m.connectProvider)}
        </ButtonLink>
      </Footer>
    </div>
  )
}
