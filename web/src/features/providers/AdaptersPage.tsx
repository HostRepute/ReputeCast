import { useQueryClient } from '@tanstack/react-query'
import { useState } from 'react'
import { $api, errorMessage, type Schemas } from '@/api/client'
import { Page } from '@/components/shell/Page'
import { Badge } from '@/components/ui/Badge'
import { Button, ButtonAnchor } from '@/components/ui/Button'
import { Dialog } from '@/components/ui/Dialog'
import { ErrorPanel, Loading, Skeleton } from '@/components/ui/Feedback'
import { Field, Input, Select, Textarea } from '@/components/ui/Field'
import { Table, Td, Th } from '@/components/ui/Table'
import { toast } from '@/components/ui/Toast'
import { ScrollPre } from '@/components/ui/ScrollPre'
import { T, useT } from '@/i18n'
import { common } from '@/i18n/common'
import { count } from '@/lib/format'
import { useNow } from '@/lib/hooks'
import { useWorkspace } from '@/lib/session'
import { ma } from './adapters.messages'
import { ProvidersTabs, throwIfForbidden } from './common'
import { formProblems, parseHeaderLines, previewText, webhookAuths, webhookPathFields, type WebhookAuth, type WebhookPaths } from './http-adapter-model'
import { m } from './messages'

type Overview = Schemas['AdaptersOverview']
type Community = Schemas['AdapterCommunity']

const adaptersKey = ['get', '/api/v1/workspaces/{workspace}/adapters'] as const

const sampleTemplate = `{
  "from": "{{ sender }}",
  "to": "{{ recipient }}",
  "subject": "{{ subject }}",
  "html": "{{ html }}",
  "headers": {{ headers_json }}
}`

const sdkSample = `import { defineAdapter } from "@reputecast/adapter-sdk";

export default defineAdapter({
  id: "example-esp",
  capabilities: { webhooks: true, batch: 100, suppressionSync: "pull" },
  terms: { bulkMarketing: "allowed" },
  async send(msg, ctx) {
    const r = await ctx.http.post("/v1/messages", toPayload(msg));
    return { providerMessageId: r.data.id };
  },
  normalizeEvent(raw) {
    return { type: map[raw.event], code: raw.smtp_code, raw };
  },
});`

/** Code in a sentence. */
const mono = (code: string) => <span className="mono">{code}</span>

/** 7.3 Adapters: built-in and community adapters, the generic HTTP adapter and the SDK. */
export function AdaptersPage() {
  const t = useT()
  const { slug } = useWorkspace()
  const q = $api.useQuery('get', '/api/v1/workspaces/{workspace}/adapters', { params: { path: { workspace: slug } } })
  throwIfForbidden(q.error)
  const [installing, setInstalling] = useState(false)
  const data = q.data
  const reason = data && !data.canInstall ? t(ma.adminReason) : undefined
  return (
    <Page
      title={t(m.title)}
      filters={<ProvidersTabs />}
      actions={
        data ? (
          <>
            <ButtonAnchor href={data.sdkUrl}>{t(ma.sdkDocs)}</ButtonAnchor>
            <Button variant="primary" disabledReason={reason} onClick={() => setInstalling(true)}>
              {t(ma.installFromRegistry)}
            </Button>
          </>
        ) : null
      }
    >
      {q.isError ? (
        <ErrorPanel title={t(ma.loadFailed)} actions={<Button onClick={() => q.refetch()}>{t(common.retry)}</Button>}>
          {errorMessage(q.error)} {t(ma.loadFailedBody)}
        </ErrorPanel>
      ) : !data ? (
        <Loading label={t(ma.loading)}>
          <div className="card flex flex-col gap-3 px-5 py-[18px]">
            <Skeleton w={180} h={16} />
            {Array.from({ length: 6 }, (_, i) => (
              <Skeleton key={i} h={14} />
            ))}
          </div>
        </Loading>
      ) : (
        <>
          <BuiltinCard data={data} />
          {/* Community adapters take the full width so their table fits; the HTTP form and the SDK share the next row. */}
          <div className="flex flex-wrap items-start gap-5">
            <CommunityCard data={data} reason={reason} onInstall={() => setInstalling(true)} />
            <HTTPAdapterCard data={data} reason={reason} />
            <BuildYourOwn data={data} />
          </div>
        </>
      )}
      {installing ? <InstallDialog onClose={() => setInstalling(false)} /> : null}
    </Page>
  )
}

function useRefreshAdapters() {
  const qc = useQueryClient()
  return () => qc.invalidateQueries({ queryKey: adaptersKey })
}

/** Adapters that ship with ReputeCast. */
function BuiltinCard({ data }: { data: Overview }) {
  const t = useT()
  return (
    <section className="card flex flex-col" aria-labelledby="pv-builtin">
      <div className="flex flex-wrap items-center justify-between gap-2 border-b border-line-2 px-5 py-3.5">
        <div className="flex flex-col">
          <h2 className="h2" id="pv-builtin">
            {t(ma.builtinTitle, { count: count(data.builtin.length) })}
          </h2>
          <span className="k">{t(ma.builtinLine)}</span>
        </div>
        {data.upToDate ? (
          <Badge tone="ok">{t(ma.versionUpToDate, { version: data.version })}</Badge>
        ) : (
          <Badge tone="info">{t(ma.versionAvailable, { version: data.version, latest: data.latest })}</Badge>
        )}
      </div>
      <Table className="pv-10 pv-nw" phoneWrap>
        <thead>
          <tr>
            <Th>{t(ma.colAdapter)}</Th>
            <Th>{t(ma.colVersion)}</Th>
            <Th>{t(ma.colMaintainer)}</Th>
            <Th>{t(ma.colEvents)}</Th>
            <Th>{t(ma.colDomainSetup)}</Th>
            <Th>{t(ma.colSuppressionSync)}</Th>
            <Th>{t(ma.colStatus)}</Th>
          </tr>
        </thead>
        <tbody>
          {data.builtin.map((a) => (
            <tr key={a.adapter}>
              <Td className="font-semibold">{a.name}</Td>
              <Td className="mono">{a.version}</Td>
              <Td>{a.maintainer}</Td>
              <Td className="wrap">{a.events}</Td>
              <Td>{a.domainSetup}</Td>
              <Td>{a.suppressionSync}</Td>
              <Td>{a.connected ? <Badge tone="ok">{t(ma.connectedCount, { count: count(a.connected) })}</Badge> : <Badge>{t(ma.available)}</Badge>}</Td>
            </tr>
          ))}
        </tbody>
      </Table>
    </section>
  )
}

/** "Unmaintained · last release 14 months ago" */
function monthsAgo(at: string, now: Date): number {
  return Math.max(0, Math.floor((now.getTime() - new Date(at).getTime()) / (30.44 * 24 * 3600_000)))
}

/** Adapters installed from the registry. */
function CommunityCard({ data, reason, onInstall }: { data: Overview; reason?: string; onInstall: () => void }) {
  const t = useT()
  const { slug } = useWorkspace()
  const refresh = useRefreshAdapters()
  const now = useNow(3600_000)
  const [removing, setRemoving] = useState<Community | null>(null)
  const update = $api.useMutation('post', '/api/v1/workspaces/{workspace}/adapters/{adapterId}/update', {
    onSuccess: (a) => {
      void refresh()
      toast({ kind: 'success', title: t(ma.updated, { name: a.name, version: a.version }) })
    },
    onError: (e) => toast({ kind: 'error', title: t(ma.updateFailed), detail: errorMessage(e) }),
  })
  const remove = $api.useMutation('delete', '/api/v1/workspaces/{workspace}/adapters/{adapterId}', {
    onSuccess: () => {
      void refresh()
      toast({ kind: 'deleted', title: removing?.name ? t(ma.removed, { name: removing.name }) : t(ma.removedNoName) })
      setRemoving(null)
    },
    onError: (e) => {
      setRemoving(null)
      toast({ kind: 'error', title: t(ma.removeFailed), detail: errorMessage(e) })
    },
  })
  const short = (name: string) => name.split(' ')[0]
  return (
    <section className="card flex min-w-0 flex-[1_1_100%] flex-col" aria-labelledby="pv-community">
      <div className="flex flex-wrap items-center justify-between gap-2 border-b border-line-2 px-5 py-3.5">
        <div className="flex flex-col">
          <h2 className="h2" id="pv-community">
            {t(ma.communityTitle)}
          </h2>
          <span className="k">{t(ma.communityLine)}</span>
        </div>
        <a href={data.registryUrl} className="k" target="_blank" rel="noopener noreferrer">
          {t(ma.browseRegistry)}
        </a>
      </div>
      <Table className="pv-10 pv-nw" phoneWrap>
        <thead>
          <tr>
            <Th>{t(ma.colAdapter)}</Th>
            <Th>{t(ma.colVersion)}</Th>
            <Th>{t(ma.colMaintainer)}</Th>
            <Th>{t(ma.colCapabilities)}</Th>
            <Th>{t(ma.colUpdate)}</Th>
            <Th>{t(ma.colStatus)}</Th>
          </tr>
        </thead>
        <tbody>
          {data.community.length ? (
            data.community.map((a) => (
              <tr key={a.id}>
                <Td className="font-semibold">{a.name}</Td>
                <Td className="mono">{a.version}</Td>
                <Td>{t(ma.communityMaintainer, { maintainer: a.maintainer })}</Td>
                <Td className="wrap">{a.capabilities.join(' · ') || '–'}</Td>
                <Td>
                  {a.update === 'update_available' ? (
                    <Badge tone="info">{t(ma.versionAvailableShort, { version: a.latest ?? '' })}</Badge>
                  ) : a.update === 'unmaintained' ? (
                    <Badge tone="warn">{a.lastRelease ? t(ma.unmaintainedSince, { months: monthsAgo(a.lastRelease, now) }) : t(ma.unmaintained)}</Badge>
                  ) : (
                    <Badge tone="ok">{t(ma.upToDate)}</Badge>
                  )}
                </Td>
                <Td>{a.connected ? <Badge tone="ok">{t(ma.connectedCount, { count: count(a.connected) })}</Badge> : <Badge>{t(ma.installedNotConnected)}</Badge>}</Td>
              </tr>
            ))
          ) : (
            <tr>
              <td colSpan={6} className="py-8 text-center text-ink-3">
                {t(ma.noCommunity)}
              </td>
            </tr>
          )}
        </tbody>
      </Table>
      <div className="flex flex-wrap gap-2.5 border-t border-line-2 px-5 py-3.5">
        {data.community
          .filter((a) => a.update === 'update_available')
          .map((a) => (
            <Button
              key={`u-${a.id}`}
              size="sm"
              disabledReason={reason}
              loading={update.isPending && update.variables?.params.path.adapterId === a.id}
              onClick={() => update.mutate({ params: { path: { workspace: slug, adapterId: a.id } } })}
            >
              {t(ma.updateTo, { name: short(a.name) ?? a.name, version: a.latest ?? '' })}
            </Button>
          ))}
        {data.community
          .filter((a) => a.update === 'unmaintained' && !a.connected)
          .map((a) => (
            <Button key={`r-${a.id}`} size="sm" disabledReason={reason} onClick={() => setRemoving(a)}>
              {t(ma.removeNamed, { name: a.name })}
            </Button>
          ))}
        <Button size="sm" disabledReason={reason} onClick={onInstall}>
          {t(ma.installFromRegistry)}
        </Button>
      </div>
      <Dialog
        open={Boolean(removing)}
        onOpenChange={(o) => !o && setRemoving(null)}
        size="sm"
        title={t(ma.removeTitle, { name: removing?.name ?? '' })}
        description={t(ma.removeBody)}
        footer={
          <>
            <Button onClick={() => setRemoving(null)}>{t(ma.keepIt)}</Button>
            <Button
              variant="danger-primary"
              loading={remove.isPending}
              onClick={() => removing && remove.mutate({ params: { path: { workspace: slug, adapterId: removing.id } } })}
            >
              {t(ma.remove)}
            </Button>
          </>
        }
      />
    </section>
  )
}

/** Maps any HTTP sending API without code. */
function HTTPAdapterCard({ data, reason }: { data: Overview; reason?: string }) {
  const t = useT()
  const { slug } = useWorkspace()
  const refresh = useRefreshAdapters()
  const [endpoint, setEndpoint] = useState('POST https://api.example-esp.com/v1/messages')
  const [headerText, setHeaderText] = useState('Authorization: Bearer {{ secrets.api_key }}')
  const [template, setTemplate] = useState(sampleTemplate)
  const [idPath, setIdPath] = useState('$.data.id')
  const [mapping, setMapping] = useState<'status' | 'jsonpath'>('status')
  const [paths, setPaths] = useState<WebhookPaths>({ type: '$.event', recipient: '$.email', messageId: '$.message_id', events: '' })
  const [auth, setAuth] = useState<WebhookAuth>('hmac')
  const [authHeader, setAuthHeader] = useState('X-Signature')
  const [naming, setNaming] = useState(false)
  const [name, setName] = useState('')
  const [error, setError] = useState('')
  const [saveError, setSaveError] = useState('')
  // Problems per field, by location ("body.headers"), from the last test or save.
  const [errors, setErrors] = useState<Record<string, string>>({})
  const [preview, setPreview] = useState<Schemas['AdapterHTTPPreview'] | null>(null)
  const authOption = webhookAuths.find((a) => a.value === auth)
  const parsed = parseHeaderLines(headerText)
  const body = {
    endpoint,
    headers: parsed.headers,
    bodyTemplate: template,
    messageIdPath: idPath,
    webhookMapping: mapping,
    webhookAuth: auth,
    ...(authOption?.header ? { webhookHeader: authHeader.trim() } : {}),
    ...(mapping === 'jsonpath' ? { webhookPaths: paths } : {}),
  }
  // The fields the form shows a problem next to, by location; any other
  // problem is shown under the form.
  const shown = new Set([
    'body.endpoint',
    'body.headers',
    'body.bodyTemplate',
    'body.messageIdPath',
    'body.webhookMapping',
    'body.webhookAuth',
    ...(authOption?.header ? ['body.webhookHeader'] : []),
    ...(mapping === 'jsonpath' ? webhookPathFields.map((f) => `body.webhookPaths.${f.key}`) : []),
  ])
  const edited = () => {
    setPreview(null)
    setErrors({})
    setError('')
  }
  const save = $api.useMutation('post', '/api/v1/workspaces/{workspace}/adapters/http', {
    onSuccess: (a) => {
      void refresh()
      setNaming(false)
      toast({ kind: 'success', title: t(ma.saved, { name: a.name }), detail: t(ma.savedDetail) })
    },
    onError: (e) => {
      const inDialog = formProblems(e, new Set([...shown, 'body.name']))
      setErrors(inDialog.fields)
      if (Object.keys(inDialog.fields).some((k) => shown.has(k))) {
        // A mapping problem: close the dialog and show it on its field, and
        // anything else (the name included) under the form.
        setNaming(false)
        setError(formProblems(e, shown).general)
        return
      }
      setSaveError(inDialog.general)
    },
  })
  const test = $api.useMutation('post', '/api/v1/workspaces/{workspace}/adapters/http/preview', {
    onSuccess: (p) => setPreview(p),
    onError: (e) => {
      const p = formProblems(e, shown)
      setErrors(p.fields)
      setError(p.general)
    },
  })
  const testReason = data.httpAdapter ? reason : t(ma.testReason)
  return (
    <section className="card flex min-w-0 flex-[1_1_380px] flex-col" aria-labelledby="pv-http">
      <div className="flex flex-col border-b border-line-2 px-5 py-3.5">
        <h2 className="h2" id="pv-http">
          {t(ma.httpTitle)}
        </h2>
        <span className="k">{t(ma.httpLine)}</span>
      </div>
      <div className="flex flex-col gap-3 px-5 py-4">
        <Field label={t(ma.sendEndpoint)} error={errors['body.endpoint']}>
          <Input
            className="mono"
            value={endpoint}
            onChange={(e) => {
              setEndpoint(e.target.value)
              edited()
            }}
            disabled={Boolean(reason)}
          />
        </Field>
        <Field label={t(ma.requestHeaders)} hint={t(ma.requestHeadersHint, { token: '{{ secrets.api_key }}' })} error={parsed.error ?? errors['body.headers']}>
          <Textarea
            className="mono resize-y"
            value={headerText}
            spellCheck={false}
            disabled={Boolean(reason)}
            rows={Math.max(1, headerText.split('\n').length)}
            onChange={(e) => {
              setHeaderText(e.target.value)
              edited()
            }}
          />
        </Field>
        <Field label={t(ma.bodyTemplate)} error={errors['body.bodyTemplate']}>
          <textarea
            className="pv-pre resize-none border-0 outline-offset-2"
            value={template}
            spellCheck={false}
            disabled={Boolean(reason)}
            // As tall as its lines, like the board's <pre> (at least the sample's 7).
            rows={Math.max(7, template.split('\n').length)}
            onChange={(e) => {
              setTemplate(e.target.value)
              edited()
            }}
          />
        </Field>
        <Field label={t(ma.messageIdPath)} error={errors['body.messageIdPath']}>
          <Input
            className="mono"
            value={idPath}
            onChange={(e) => {
              setIdPath(e.target.value)
              edited()
            }}
            disabled={Boolean(reason)}
          />
        </Field>
        <Field
          label={t(ma.webhookMapping)}
          hint={mapping === 'jsonpath' ? t(ma.jsonpathHint, { types: 'delivered, bounce, spamreport, dropped, open, click…' }) : undefined}
          error={errors['body.webhookMapping']}
        >
          <select
            className="in"
            value={mapping}
            disabled={Boolean(reason)}
            onChange={(e) => {
              setMapping(e.target.value as 'status' | 'jsonpath')
              edited()
            }}
          >
            <option value="status">{'status → delivered | bounced | complained'}</option>
            <option value="jsonpath">{t(ma.mappingJsonpath)}</option>
          </select>
        </Field>
        {mapping === 'jsonpath' ? (
          <div className="grid grid-cols-1 gap-3 sm:grid-cols-2">
            {webhookPathFields.map((f) => (
              <Field key={f.key} label={f.label} optional={f.optional} error={errors[`body.webhookPaths.${f.key}`]}>
                <Input
                  className="mono"
                  value={paths[f.key]}
                  placeholder={f.placeholder}
                  disabled={Boolean(reason)}
                  onChange={(e) => {
                    setPaths({ ...paths, [f.key]: e.target.value })
                    edited()
                  }}
                />
              </Field>
            ))}
          </div>
        ) : null}
        <Field label={t(ma.webhookAuth)} error={errors['body.webhookAuth']}>
          <Select
            value={auth}
            disabled={Boolean(reason)}
            onChange={(e) => {
              const next = e.target.value as WebhookAuth
              setAuth(next)
              setAuthHeader(webhookAuths.find((a) => a.value === next)?.placeholder ?? '')
              edited()
            }}
          >
            {webhookAuths.map((a) => (
              <option key={a.value} value={a.value}>
                {a.label}
              </option>
            ))}
          </Select>
        </Field>
        {authOption?.header ? (
          <Field label={authOption.header} error={errors['body.webhookHeader']}>
            <Input
              className="mono"
              value={authHeader}
              placeholder={authOption.placeholder}
              onChange={(e) => {
                setAuthHeader(e.target.value)
                edited()
              }}
              disabled={Boolean(reason)}
            />
          </Field>
        ) : null}
        {error ? (
          <span className="err" role="alert">
            {error}
          </span>
        ) : null}
        <div className="flex flex-wrap gap-2">
          <Button
            size="sm"
            variant="primary"
            disabledReason={testReason ?? parsed.error}
            loading={test.isPending}
            onClick={() => {
              edited()
              test.mutate({ params: { path: { workspace: slug } }, body })
            }}
          >
            {t(ma.testSample)}
          </Button>
          <Button
            size="sm"
            disabledReason={reason ?? parsed.error}
            onClick={() => {
              setSaveError('')
              setNaming(true)
            }}
          >
            {t(ma.saveAsAdapter)}
          </Button>
        </div>
        {preview ? (
          <div className="flex flex-col gap-1.5" role="status" aria-label={t(ma.sampleRequest)}>
            <span className="k">{t(ma.sampleNote)}</span>
            <pre className="pv-pre min-w-0 whitespace-pre-wrap [overflow-wrap:anywhere]">{previewText(preview)}</pre>
          </div>
        ) : null}
      </div>
      <Dialog
        open={naming}
        onOpenChange={setNaming}
        size="sm"
        title={t(ma.saveAsAdapter)}
        description={t(ma.saveDialogBody)}
        footer={
          <>
            <Button onClick={() => setNaming(false)}>{t(common.cancel)}</Button>
            <Button
              variant="primary"
              loading={save.isPending}
              onClick={() => {
                setErrors({})
                setSaveError('')
                save.mutate({ params: { path: { workspace: slug } }, body: { name: name.trim(), ...body } })
              }}
            >
              {t(ma.saveAdapter)}
            </Button>
          </>
        }
      >
        <Field label={t(ma.name)} hint={t(ma.nameHint)} error={errors['body.name']}>
          <Input value={name} onChange={(e) => setName(e.target.value)} maxLength={60} autoFocus />
        </Field>
        {saveError ? (
          <span className="err" role="alert">
            {saveError}
          </span>
        ) : null}
      </Dialog>
    </section>
  )
}

/** The adapter SDK in one card. */
function BuildYourOwn({ data }: { data: Overview }) {
  const t = useT()
  return (
    <section className="card flex min-w-0 flex-[1.4_1_560px] flex-wrap items-start gap-5 px-[22px] py-5" aria-labelledby="pv-sdk">
      <div className="flex flex-[1_1_320px] flex-col gap-2">
        <h2 className="h2" id="pv-sdk">
          {t(ma.buildTitle)}
        </h2>
        <span className="text-[13px] text-ink-2">
          <T
            msg={ma.buildBody}
            values={{
              iface: mono('SendingProvider'),
              send: mono('send()'),
              normalize: mono('normalizeEvent()'),
              setup: mono('setupDomain()'),
              sync: mono('syncSuppressions()'),
              quota: mono('discoverQuota()'),
              manifest: mono('manifest.json'),
            }}
          />
        </span>
        <div className="flex flex-wrap gap-2">
          <ButtonAnchor size="sm" href={data.sdkUrl}>
            {t(ma.readSdk)}
          </ButtonAnchor>
          <ButtonAnchor size="sm" href={data.templateUrl}>
            {t(ma.templateRepo)}
          </ButtonAnchor>
        </div>
      </div>
      <ScrollPre className="pv-pre min-w-0 flex-[1_1_420px]" label={t(ma.exampleAdapter)}>
        {sdkSample}
      </ScrollPre>
    </section>
  )
}

/** Installs a community adapter from its manifest URL. */
function InstallDialog({ onClose }: { onClose: () => void }) {
  const t = useT()
  const { slug } = useWorkspace()
  const refresh = useRefreshAdapters()
  const [url, setUrl] = useState('')
  const [error, setError] = useState('')
  const install = $api.useMutation('post', '/api/v1/workspaces/{workspace}/adapters', {
    onSuccess: (a) => {
      void refresh()
      onClose()
      toast({ kind: 'success', title: t(ma.installed, { name: a.name, version: a.version }), detail: t(ma.installedDetail) })
    },
    onError: (e) => setError(errorMessage(e)),
  })
  return (
    <Dialog
      open
      onOpenChange={(o) => !o && onClose()}
      title={t(ma.installFromRegistry)}
      description={t(ma.installBody)}
      footer={
        <>
          <Button onClick={onClose}>{t(common.cancel)}</Button>
          <Button variant="primary" loading={install.isPending} onClick={() => install.mutate({ params: { path: { workspace: slug } }, body: { manifestUrl: url.trim() } })}>
            {t(ma.install)}
          </Button>
        </>
      }
    >
      <Field label={t(ma.manifestUrl)} error={error}>
        <Input type="url" value={url} onChange={(e) => setUrl(e.target.value)} placeholder={'https://github.com/…/manifest.json'} autoFocus />
      </Field>
    </Dialog>
  )
}
