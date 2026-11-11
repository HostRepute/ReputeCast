import { useNavigate } from '@tanstack/react-router'
import { useEffect, useState, type ReactNode } from 'react'
import { $api, ApiError, errorMessage } from '@/api/client'
import { Icon } from '@/components/icons/Icon'
import { Button } from '@/components/ui/Button'
import { Dialog } from '@/components/ui/Dialog'
import { Skeleton } from '@/components/ui/Feedback'
import { StepTrail } from '@/components/ui/Stepper'
import { toast } from '@/components/ui/Toast'
import { Footer } from '@/features/integrations/ConnectDialog'
import { AttentionIcon } from '@/features/integrations/shared'
import '@/features/integrations/integrations.css'
import { T, useT } from '@/i18n'
import { useWorkspace } from '@/lib/session'
import { leadsBase, usePageChoices, useRefreshLeads, useSetup, type Connection } from './api'
import { mc, ml } from './messages'
import './leads.css'

/*
 * Connect Meta Lead Ads (LeadAdsConnect): create the app, paste the
 * credentials (checked with Meta before anything is kept), choose the
 * Pages to subscribe. Replace credentials and Choose Pages reuse the last
 * two steps on a connection's page.
 */

type Step = 'app' | 'credentials' | 'pages'
const steps: Step[] = ['app', 'credentials', 'pages']

const mono = (c: ReactNode[]) => <span className="mono text-[12.5px]">{c}</span>

/** Field errors by field ("token", "appSecret"…) from a 422, and the problem's sentence otherwise. */
function problemOf(e: unknown): { fields: Record<string, string>; form?: string } {
  const fields: Record<string, string> = {}
  if (e instanceof ApiError) {
    for (const x of e.problem.errors ?? []) if (x.location && x.message) fields[x.location.replace(/^body\./, '')] = x.message
    if (Object.keys(fields).length) return { fields }
  }
  return { fields, form: errorMessage(e) }
}

function FieldError({ id, children }: { id: string; children: ReactNode }) {
  return (
    <span className="k font-medium" id={id}>
      <span className="font-semibold text-bad-fg">{children}</span>
    </span>
  )
}

function FormError({ children }: { children: ReactNode }) {
  return (
    <div className="note b" role="alert">
      <AttentionIcon size={16} color="var(--bad-fg)" />
      <div className="min-w-0 flex-[1_1_320px]">{children}</div>
    </div>
  )
}

/** A line with an ok tick or a warning: Meta can reach this server, the webhook. */
function StatusLine({ ok, children }: { ok: boolean; children: ReactNode }) {
  return (
    <div className="la-line">
      {ok ? <Icon name="check" size={14} strokeWidth={2.2} className="text-ok-fg" /> : <AttentionIcon size={14} />}
      <span className="min-w-0 flex-auto">{children}</span>
    </div>
  )
}

export function ConnectLeadAdsDialog({ onOpenChange }: { onOpenChange: (open: boolean) => void }) {
  const t = useT()
  const { slug } = useWorkspace()
  const navigate = useNavigate()
  const refresh = useRefreshLeads()
  const [step, setStep] = useState<Step>('app')
  const [conn, setConn] = useState<Connection>()
  const setup = useSetup()
  const close = (o: boolean) => {
    if (!o && conn) void refresh()
    onOpenChange(o)
  }
  // One connection per workspace: when there is one, its page is where to go.
  const existing = setup.data?.connection
  useEffect(() => {
    if (!existing || conn) return
    onOpenChange(false)
    void navigate({ to: '/w/$workspace/settings/integrations/leads/$connectionId', params: { workspace: slug, connectionId: existing.id } })
  }, [existing, conn, onOpenChange, navigate, slug])
  return (
    <Dialog
      open
      onOpenChange={close}
      title={t(mc.title)}
      eyebrow={<span className="k">{t(ml.eyebrow)}</span>}
      size="md"
      className="max-w-[640px]"
      hideClose
      dismissible={step === 'app'}
    >
      <StepTrail className="-mt-1" label={t(mc.steps)} current={steps.indexOf(step) + 1} steps={steps.map((s) => ({ key: s, title: t({ app: mc.stepApp, credentials: mc.stepCredentials, pages: mc.stepPages }[s]) }))} />
      {step === 'app' ? (
        <AppStep setup={setup} onCancel={() => close(false)} onNext={() => setStep('credentials')} />
      ) : step === 'credentials' ? (
        <CredentialsStep
          onBack={() => setStep('app')}
          onDone={(c) => {
            setConn(c)
            setStep('pages')
          }}
        />
      ) : conn ? (
        <PagesStep
          connection={conn}
          webhook
          onDone={(c) => {
            toast({ kind: 'success', title: t(mc.connected) })
            void refresh()
            onOpenChange(false)
            void navigate({ to: '/w/$workspace/settings/integrations/leads/$connectionId', params: { workspace: slug, connectionId: c.id } })
          }}
        />
      ) : null}
    </Dialog>
  )
}

function AppStep({ setup, onCancel, onNext }: { setup: ReturnType<typeof useSetup>; onCancel: () => void; onNext: () => void }) {
  const t = useT()
  const s = setup.data
  return (
    <>
      <p className="m-0 text-ink-nav">{t(mc.appLead)}</p>
      <ol className="ih-ol">
        <li>
          <span>{t(mc.app1)}</span>
        </li>
        <li>
          <span>{t(mc.app2)}</span>
        </li>
        <li>
          <span className="flex min-w-0 flex-col gap-1.5">
            <span>{t(mc.app3)}</span>
            <span className="flex flex-wrap gap-1.5">
              {(s?.permissions ?? []).map((p) => (
                <span key={p} className="ih-mk">
                  {p}
                </span>
              ))}
            </span>
          </span>
        </li>
        <li>
          <span>{t(mc.app4)}</span>
        </li>
      </ol>
      <div className="note i" role="note">
        <Icon name="info" size={16} className="mt-px flex-none" />
        <div className="min-w-0 flex-[1_1_320px]">{t(mc.appReview)}</div>
      </div>
      {setup.isError ? (
        <FormError>{errorMessage(setup.error)}</FormError>
      ) : !s ? (
        <Skeleton w="70%" h={13} />
      ) : (
        <StatusLine ok={s.reachable}>{s.reachable ? <T msg={mc.reachable} values={{ url: s.url, mono }} /> : t(mc.unreachable)}</StatusLine>
      )}
      <Footer>
        <Button onClick={onCancel}>{t(mc.cancel)}</Button>
        <Button variant="primary" onClick={onNext}>
          {t(mc.next)}
        </Button>
      </Footer>
    </>
  )
}

/** Name, app ID, app secret and token. On a connection, blank secrets stay as they are. */
function CredentialFields({
  values,
  set,
  errors,
  replace,
  disabled,
}: {
  values: { name: string; appId: string; appSecret: string; token: string }
  set: (k: 'name' | 'appId' | 'appSecret' | 'token', v: string) => void
  errors: Record<string, string>
  replace?: boolean
  disabled: boolean
}) {
  const t = useT()
  const err = (k: string) => errors[k]
  // Meta refusing the app ID and secret together marks both fields; the message sits under the secret.
  const appBad = !!err('appSecret') || !!err('appId')
  const input = (k: 'appId' | 'appSecret' | 'token', label: string, type: 'text' | 'password', extra?: ReactNode) => {
    const bad = k === 'appId' ? appBad : !!err(k)
    const describedBy = [k === 'appId' && appBad ? 'la-c-appSecret-e' : undefined, err(k) ? `la-c-${k}-e` : undefined, replace && k !== 'appId' ? `la-c-${k}-h` : undefined]
      .filter(Boolean)
      .join(' ')
    return (
      <label className="lbl">
        {label}
        <input
          className={`in${k === 'appId' ? ' mono text-[12.5px]' : ''}`}
          type={type}
          value={values[k]}
          onChange={(e) => set(k, e.target.value)}
          autoComplete="off"
          spellCheck={false}
          readOnly={disabled}
          inputMode={k === 'appId' ? 'numeric' : undefined}
          aria-invalid={bad || undefined}
          aria-describedby={describedBy || undefined}
          style={bad ? { borderColor: 'var(--bad-strong)' } : undefined}
        />
        {err(k) ? <FieldError id={`la-c-${k}-e`}>{err(k)}</FieldError> : null}
        {extra}
      </label>
    )
  }
  const kept = (k: 'appSecret' | 'token') =>
    replace ? (
      <span className="k font-medium" id={`la-c-${k}-h`}>
        {t(mc.secretKept)}
      </span>
    ) : null
  return (
    <>
      <label className="lbl">
        {t(mc.name)}
        <input
          className="in"
          type="text"
          value={values.name}
          onChange={(e) => set('name', e.target.value)}
          maxLength={80}
          readOnly={disabled}
          aria-invalid={err('name') ? true : undefined}
          aria-describedby={err('name') ? 'la-c-name-e la-c-name-h' : 'la-c-name-h'}
          style={err('name') ? { borderColor: 'var(--bad-strong)' } : undefined}
        />
        {err('name') ? <FieldError id="la-c-name-e">{err('name')}</FieldError> : null}
        <span className="k font-medium" id="la-c-name-h">
          {t(mc.nameHelp)}
        </span>
      </label>
      {input('appId', t(mc.appId), 'text')}
      {input('appSecret', t(mc.appSecret), 'password', kept('appSecret'))}
      {input('token', t(mc.token), 'password', kept('token'))}
      <span className="k">{t(mc.secretsHelp)}</span>
    </>
  )
}

function CredentialsStep({ onBack, onDone }: { onBack: () => void; onDone: (c: Connection) => void }) {
  const t = useT()
  const { slug } = useWorkspace()
  const [values, setValues] = useState({ name: '', appId: '', appSecret: '', token: '' })
  const [problem, setProblem] = useState<{ fields: Record<string, string>; form?: string }>({ fields: {} })
  const m = $api.useMutation('post', `${leadsBase}/connections`)
  const ready = values.name.trim() && values.appId.trim() && values.appSecret.trim() && values.token.trim()
  return (
    <form
      className="contents"
      onSubmit={(e) => {
        e.preventDefault()
        if (!ready || m.isPending) return
        setProblem({ fields: {} })
        m.mutate(
          { params: { path: { workspace: slug } }, body: { name: values.name.trim(), appId: values.appId.trim(), appSecret: values.appSecret.trim(), token: values.token.trim() } },
          { onSuccess: onDone, onError: (err) => setProblem(problemOf(err)) },
        )
      }}
    >
      <CredentialFields values={values} set={(k, v) => setValues((x) => ({ ...x, [k]: v }))} errors={problem.fields} disabled={m.isPending} />
      {problem.form ? <FormError>{problem.form}</FormError> : null}
      <Footer>
        <Button className="mr-auto" onClick={onBack} disabled={m.isPending}>
          {t(mc.back)}
        </Button>
        {/* The board draws "Checking with Meta…" on a wider button, and Connect at its own width. */}
        <Button type="submit" variant="primary" loading={m.isPending} disabled={!ready}>
          {m.isPending ? t(mc.checking) : t(mc.connect)}
        </Button>
      </Footer>
    </form>
  )
}

/** The Pages the token reaches: the chosen ones subscribed, the others not. */
function PagesStep({ connection: c, webhook, onDone }: { connection: Connection; webhook?: boolean; onDone: (c: Connection) => void }) {
  const t = useT()
  const { slug } = useWorkspace()
  const q = usePageChoices(c.id)
  const [chosen, setChosen] = useState<Set<string> | null>(null)
  const [error, setError] = useState<string>()
  const m = $api.useMutation('put', `${leadsBase}/connections/{id}/pages`)
  const items = q.data?.items
  // A first connection proposes every Page with lead forms; later, the ones already chosen.
  const picked = chosen ?? new Set((items ?? []).filter((p) => (c.pages.length ? p.chosen : p.chosen || (p.forms ?? 0) > 0)).map((p) => p.pageId))
  const toggle = (id: string, on: boolean) => {
    const next = new Set(picked)
    if (on) next.add(id)
    else next.delete(id)
    setChosen(next)
  }
  const subscribed = !!c.receiving.subscribedAt
  return (
    <form
      className="contents"
      onSubmit={(e) => {
        e.preventDefault()
        setError(undefined)
        m.mutate({ params: { path: { workspace: slug, id: c.id } }, body: { pageIds: [...picked] } }, { onSuccess: onDone, onError: (err) => setError(errorMessage(err)) })
      }}
    >
      <p className="m-0 text-ink-nav">{t(mc.pagesLead)}</p>
      {q.isError ? (
        <FormError>{errorMessage(q.error)}</FormError>
      ) : !items ? (
        <div className="flex flex-col gap-2" aria-busy="true">
          <Skeleton w="100%" h={42} r={8} />
          <Skeleton w="100%" h={42} r={8} />
        </div>
      ) : items.length === 0 ? (
        <FormError>{t(mc.pagesNone)}</FormError>
      ) : (
        <fieldset className="m-0 flex flex-col gap-2 border-0 p-0">
          <legend className="vh">{t(mc.stepPages)}</legend>
          {items.map((p) => {
            const on = picked.has(p.pageId)
            return (
              <label key={p.pageId} className={`la-page${on ? ' on' : ''}`}>
                <input type="checkbox" checked={on} onChange={(e) => toggle(p.pageId, e.target.checked)} disabled={m.isPending} />
                <span>
                  <b className="font-semibold">{p.name}</b> · {p.forms === undefined ? t(mc.pageFormsUnknown) : t(mc.pageForms, { count: p.forms })}
                </span>
              </label>
            )
          })}
        </fieldset>
      )}
      {webhook ? <StatusLine ok={subscribed}>{t(subscribed ? mc.webhookOk : mc.webhookNo)}</StatusLine> : null}
      {error ? <FormError>{error}</FormError> : null}
      <Footer>
        <Button type="submit" variant="primary" loading={m.isPending} disabled={!items || items.length === 0}>
          {t(mc.subscribe, { count: picked.size })}
        </Button>
      </Footer>
    </form>
  )
}

/** Choose Pages on a connection's page. */
export function ChoosePagesDialog({ connection: c, onOpenChange }: { connection: Connection; onOpenChange: (open: boolean) => void }) {
  const t = useT()
  const refresh = useRefreshLeads()
  return (
    <Dialog open onOpenChange={onOpenChange} title={t(mc.stepPages)} eyebrow={<span className="k">{t(ml.eyebrow)}</span>} size="md" className="max-w-[640px]" hideClose>
      <PagesStep
        connection={c}
        onDone={(next) => {
          toast({ kind: 'success', title: t(mc.pagesSaved, { count: next.pages.length }) })
          void refresh()
          onOpenChange(false)
        }}
      />
    </Dialog>
  )
}

/** Replace credentials: the name, app ID, app secret or token; blank secrets stay. */
export function ReplaceCredentialsDialog({ connection: c, onOpenChange }: { connection: Connection; onOpenChange: (open: boolean) => void }) {
  const t = useT()
  const { slug } = useWorkspace()
  const refresh = useRefreshLeads()
  const [values, setValues] = useState({ name: c.name, appId: c.appId, appSecret: '', token: '' })
  const [problem, setProblem] = useState<{ fields: Record<string, string>; form?: string }>({ fields: {} })
  const m = $api.useMutation('put', `${leadsBase}/connections/{id}/credentials`)
  return (
    <Dialog
      open
      onOpenChange={onOpenChange}
      title={t(mc.replaceTitle)}
      eyebrow={<span className="k">{t(ml.eyebrow)}</span>}
      size="md"
      className="max-w-[640px]"
      hideClose
      dismissible={!m.isPending}
    >
      <form
        className="contents"
        onSubmit={(e) => {
          e.preventDefault()
          if (m.isPending) return
          setProblem({ fields: {} })
          const body = {
            name: values.name.trim() !== c.name ? values.name.trim() : undefined,
            appId: values.appId.trim() !== c.appId ? values.appId.trim() : undefined,
            appSecret: values.appSecret.trim() || undefined,
            token: values.token.trim() || undefined,
          }
          m.mutate(
            { params: { path: { workspace: slug, id: c.id } }, body },
            {
              onSuccess: (next) => {
                toast({ kind: 'success', title: t(mc.replaced, { name: next.name }) })
                void refresh()
                onOpenChange(false)
              },
              onError: (err) => setProblem(problemOf(err)),
            },
          )
        }}
      >
        <p className="m-0 text-ink-nav">{t(mc.replaceLead)}</p>
        <CredentialFields values={values} set={(k, v) => setValues((x) => ({ ...x, [k]: v }))} errors={problem.fields} replace disabled={m.isPending} />
        {problem.form ? <FormError>{problem.form}</FormError> : null}
        <Footer>
          <Button onClick={() => onOpenChange(false)} disabled={m.isPending}>
            {t(mc.cancel)}
          </Button>
          <Button type="submit" variant="primary" loading={m.isPending} disabled={!values.name.trim() || !values.appId.trim()}>
            {m.isPending ? t(mc.checking) : t(mc.save)}
          </Button>
        </Footer>
      </form>
    </Dialog>
  )
}
