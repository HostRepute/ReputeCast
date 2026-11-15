import { Link } from '@tanstack/react-router'
import { useRef, useState, type ReactNode } from 'react'
import { useForm, useWatch } from 'react-hook-form'
import { $api, ApiError, type Schemas } from '@/api/client'
import { AuthLanguageMenu } from '@/features/language/AuthLanguageMenu'
import { InstanceBrand } from '@/components/shell/AuthLayout'
import { Button } from '@/components/ui/Button'
import { Card } from '@/components/ui/Card'
import { Checkbox, Field, Select, Textarea } from '@/components/ui/Field'
import { ErrorPanel, Loading, Skeleton } from '@/components/ui/Feedback'
import { Note } from '@/components/ui/Note'
import { useT } from '@/i18n'
import { common } from '@/i18n/common'
import { logTime } from '@/lib/format'
import { useDocumentTitle, useNow } from '@/lib/hooks'
import { useMe } from '@/lib/session'
import { mc } from './messages'
import { capabilities, ConnectionDisclosure, ConnectionIdentity } from './parts'

type Consent = Schemas['MCPConsent']
type Options = Schemas['MCPConsentOptions']

export function ConsentPage({ request }: { request: string }) {
  const t = useT()
  const me = useMe()
  const q = $api.useQuery('get', '/api/v1/mcp-connect/consent', { params: { query: { request } } }, {
    enabled: Boolean(me.data && request), retry: false, gcTime: 0, refetchOnWindowFocus: false,
  })
  const [slug, setSlug] = useState('')
  const [uncertain, setUncertain] = useState(false)
  const [invalid, setInvalid] = useState(false)
  const decisionStarted = useRef(false)
  const now = useNow(1000)
  const deny = $api.useMutation('post', '/api/v1/mcp-connect/consent/deny', { retry: false })
  const allow = $api.useMutation('post', '/api/v1/workspaces/{workspace}/mcp-connect/consent', { retry: false })
  const decisionFailed = (e: unknown) => {
    if (e instanceof ApiError && e.status === 422) { decisionStarted.current = false; setInvalid(true) }
    else setUncertain(true)
  }
  const denyButton = <Button loading={deny.isPending} disabled={uncertain || allow.isPending || !q.data} onClick={() => {
    if (decisionStarted.current) return
    decisionStarted.current = true
    deny.mutate({ body: { request } }, { onSuccess: (out) => window.location.replace(out.redirect), onError: decisionFailed })
  }}>{t(mc.deny)}</Button>
  const expired = Boolean(q.data && new Date(q.data.expiresAt).getTime() <= now.getTime())
  let content: ReactNode
  if (!request || expired || (q.error instanceof ApiError && q.error.status === 409)) {
    content = <ErrorPanel title={t(mc.invalidTitle)}>{t(mc.invalidBody)}</ErrorPanel>
  } else if (me.error) {
    content = <ErrorPanel title={t(mc.loadFailed)} actions={<Button onClick={() => void me.refetch()}>{t(mc.retry)}</Button>} />
  } else if (me.isPending) {
    content = <ConsentLoading />
  } else if (!me.data) {
    content = <Card className="flex flex-col gap-4 p-5"><h2 className="h2">{t(mc.signinTitle)}</h2><p className="m-0">{t(mc.signinBody)}</p>
      <div><Link className="btn pri" to="/sign-in" search={{ next: `/mcp-consent?request=${encodeURIComponent(request)}` }}>{t(mc.signin)}</Link></div>
    </Card>
  } else if (q.error && !q.data) {
    content = <ErrorPanel title={t(mc.loadFailed)} actions={<Button onClick={() => void q.refetch()}>{t(mc.retry)}</Button>}>{t(mc.invalidBody)}</ErrorPanel>
  } else if (!q.data) {
    content = <ConsentLoading />
  } else {
    content = <Card>
      <div className="flex flex-col gap-3 border-b border-line p-5 max-md:p-4">
        <ConnectionIdentity name={q.data.clientName} verified={q.data.verified} />
        {!q.data.verified ? <p className="k m-0">{t(mc.unverifiedHint)}</p> : null}
        <p className="m-0">{t(mc.callback)}: <code>{q.data.callbackHost}</code></p>
      </div>
      <div className="mcp-connect-form">
        <p className="k m-0">{t(mc.signedIn, { name: me.data.user.name, date: logTime(q.data.expiresAt) })}</p>
        {uncertain ? <ErrorPanel title={t(mc.uncertainTitle)} actions={<><Button loading={q.isFetching} onClick={async () => {
          const result = await q.refetch()
          if (result.isSuccess) { decisionStarted.current = false; setUncertain(false); setInvalid(false) }
        }}>{t(mc.retry)}</Button>{slug ? <Link className="btn" to="/w/$workspace/settings/connections" params={{ workspace: slug }}>{t(mc.managing)}</Link> : null}</>}>
          {t(mc.uncertainBody)}
        </ErrorPanel> : null}
        {invalid ? <Note tone="bad">{t(mc.validation)}</Note> : null}
        {me.data.pending ? <><Note tone="warn" label={t(mc.setupTitle)}>{t(mc.setupBody)}</Note><div className="flex flex-wrap gap-2">{denyButton}<Link className="btn" to="/setup">{t(mc.setup)}</Link></div></> : <>
          <Field label={t(mc.workspace)}><Select value={slug} disabled={deny.isPending || allow.isPending || uncertain} onChange={(e) => { setSlug(e.target.value); setInvalid(false) }}>
            <option value="">{t(mc.chooseWorkspace)}</option>{me.data.workspaces.map((w) => <option key={w.id} value={w.slug}>{w.name}</option>)}
          </Select></Field>
          {slug ? <WorkspaceConsent key={slug} slug={slug} request={request} consent={q.data} locked={uncertain || deny.isPending || allow.isPending} busy={allow.isPending} deny={denyButton} onAllow={(body) => {
            if (decisionStarted.current) return
            decisionStarted.current = true
            allow.mutate({ params: { path: { workspace: slug } }, body }, { onSuccess: (out) => window.location.replace(out.redirect), onError: decisionFailed })
          }} /> : <div>{denyButton}</div>}
        </>}
      </div>
    </Card>
  }
  useDocumentTitle(`${t(mc.consentTitle)} · ReputeCast`)
  return <div className="rc-app"><main id="main" className="mcp-consent"><InstanceBrand />
    <header><h1 className="m-0 text-[28px] font-semibold max-md:text-2xl">{t(mc.consentTitle)}</h1><p className="mb-0 text-ink-2">{t(mc.consentLead)}</p></header>
    <div className="mcp-connect-grid">{content}<ConnectionDisclosure /></div>
    <AuthLanguageMenu languages={me.data?.instance.languages} />
  </main></div>
}

function ConsentLoading() {
  const t = useT()
  return <Loading label={t(common.loading)} className="card flex flex-col gap-4 p-5"><Skeleton h={28} w="65%" /><Skeleton h={18} /><Skeleton h={80} /><Skeleton h={140} /><Skeleton h={44} /></Loading>
}

type ConsentFormProps = { request: string; consent: Consent; locked: boolean; busy: boolean; onAllow: (body: Schemas['MCPConsentDecision']) => void; deny: ReactNode }

function WorkspaceConsent({ slug, ...props }: ConsentFormProps & { slug: string }) {
  const t = useT()
  const q = $api.useQuery('get', '/api/v1/workspaces/{workspace}/mcp-connect/options', { params: { path: { workspace: slug } } }, { retry: false })
  if (q.error) return <><Note tone="warn" label={t(mc.optionsFailed)}>{t(mc.optionsHint)}</Note><div className="flex flex-wrap gap-2">{props.deny}<Button onClick={() => void q.refetch()}>{t(mc.retry)}</Button></div></>
  if (!q.data) return <><ConsentLoading /><div>{props.deny}</div></>
  return <ConsentForm {...props} options={q.data} />
}

function ConsentForm({ request, consent, options, locked, busy, onAllow, deny }: ConsentFormProps & { options: Options }) {
  const t = useT()
  const { register, control, handleSubmit } = useForm({ defaultValues: {
    scopes: [] as string[], lists: options.defaultLists, domains: options.defaultDomains,
    restrictLists: options.defaultLists.length > 0, restrictDomains: options.defaultDomains.length > 0, ips: '',
  } })
  const values = useWatch({ control })
  const listInvalid = Boolean(values.restrictLists && !values.lists?.length)
  const domainInvalid = Boolean(values.restrictDomains && !values.domains?.length)
  return <form className="flex flex-col gap-5" onSubmit={handleSubmit((data) => {
    if (locked || listInvalid || domainInvalid || !data.scopes.length) return
    onAllow({
      request, scopes: data.scopes, lists: data.restrictLists ? data.lists : options.defaultLists, senderDomains: data.restrictDomains ? data.domains : options.defaultDomains,
      allowedIps: data.ips.split(/\r?\n/).map((s) => s.trim()).filter(Boolean),
    })
  })}>
    <fieldset disabled={locked}><legend>{t(mc.capabilities)}</legend><div className="flex flex-col gap-4">
      {capabilities.filter((c) => consent.scopes.includes(c.key)).map((c) => <Checkbox key={c.key} value={c.key} {...register('scopes')} disabled={!options.scopes.includes(c.key)}
        label={<><strong>{t(c.title)}</strong><br /><code>{c.key}</code></>}
        hint={<>{t(c.hint)}{!options.scopes.includes(c.key) ? <span className="block text-warn">{t(mc.permission)}</span> : null}</>} />)}
    </div></fieldset>
    <fieldset disabled={locked}><legend>{t(mc.resourceScope)}</legend><div className="flex flex-col gap-3">
      <Checkbox {...register('restrictLists')} label={t(mc.restrictLists)} hint={!values.restrictLists ? t(mc.allLists) : undefined} />
      {values.restrictLists ? <div className="mcp-connect-options">{options.lists.map((l) => <Checkbox key={l.id} value={l.id} {...register('lists')} label={l.name} />)}{listInvalid ? <p role="alert" className="err m-0">{t(mc.selectList)}</p> : null}</div> : null}
      <Checkbox {...register('restrictDomains')} label={t(mc.restrictDomains)} hint={!values.restrictDomains ? t(mc.allDomains) : undefined} />
      {values.restrictDomains ? <div className="mcp-connect-options">{options.senderDomains.map((d) => <Checkbox key={d} value={d} {...register('domains')} label={d} />)}{domainInvalid ? <p role="alert" className="err m-0">{t(mc.selectDomain)}</p> : null}</div> : null}
      <Field label={t(mc.ips)} hint={t(mc.ipsHint)}><Textarea rows={2} maxLength={900} {...register('ips')} /></Field>
    </div></fieldset>
    <Note tone="info" label={t(mc.expiryTitle)}>{t(mc.expiryHint)}</Note>
    <div className="flex flex-wrap gap-2">{deny}<Button type="submit" variant="primary" loading={busy} disabled={locked || listInvalid || domainInvalid || !values.scopes?.length}>{t(mc.allow)}</Button></div>
  </form>
}
