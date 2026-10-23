import { useRef, useState } from 'react'
import { $api, errorMessage, http } from '@/api/client'
import { Icon } from '@/components/icons/Icon'
import { Badge } from '@/components/ui/Badge'
import { Button } from '@/components/ui/Button'
import { Dialog } from '@/components/ui/Dialog'

import { Segmented } from '@/components/ui/Tabs'
import { toast } from '@/components/ui/Toast'
import { Ev } from '@/features/integrations/ConnectDialog'
import '@/features/integrations/integrations.css'
import { T, formatList, useT } from '@/i18n'
import { clock, clockSeconds, count } from '@/lib/format'
import { roleName } from '@/lib/roles'
import { useMe, useWorkspace } from '@/lib/session'
import { aiBase, AI_FEATURES, problemCode, problemMeta, useRefreshAI, type AIConnection, type AIFeature, type AIModelOption, type AISettings, type AITest } from './api'
import { AITile } from './catalog'
import { aiCapabilities, aiEntryName, AI_PROVIDERS, type AIProvider } from './entries'
import { calendarDay, seconds, tokens } from './format'
import { m } from './messages'
import { AINote, featureName, LeavesNote, tags } from './parts'
import './ai.css'

type CapKind = 'usd' | 'tokens' | 'none'
type Private = { host: string; ip?: string; message: string }
const TYPED = '__typed'
const DEFAULT_TOKENS = 20_000_000

/** Two decimals for a price or a cap in an input: 2 → "2.00". */
function fixed(v: number | undefined): string {
  return v === undefined || Number.isNaN(v) ? '' : v.toFixed(2)
}

function parse(v: string): number | undefined {
  const n = Number.parseFloat(v.replace(',', '.'))
  return Number.isFinite(n) ? n : undefined
}

/** The models a provider offers: ReputeCast's price list, then the provider's own list once a key is given. */
function useModels(provider: AIProvider, s: AISettings) {
  const { slug } = useWorkspace()
  const fromCatalog = s.catalog.find((c) => c.kind === provider)?.models ?? []
  // The price list's models (Azure's are OpenAI's): no key, nothing about the workspace.
  const base = $api.useQuery('post', `${aiBase}/models`, { params: { path: { workspace: slug } }, body: { provider } }, { staleTime: 10 * 60_000 })
  const [listed, setListed] = useState<{ provider: AIProvider; models: AIModelOption[]; fromProvider: boolean; error?: string }>()
  // Once a key or an address is typed, the provider's own list (asked on blur).
  const ask = async (extra: { apiKey?: string; baseUrl?: string; deployment?: string }) => {
    try {
      const res = await http.POST(`${aiBase}/models`, { params: { path: { workspace: slug } }, body: { provider, ...extra } })
      if (res.data) setListed({ provider, models: res.data.models, fromProvider: res.data.fromProvider, error: res.data.error })
    } catch {
      // The price list stands.
    }
  }
  const mine = listed?.provider === provider ? listed : undefined
  const models = mine?.models.length ? mine.models : (base.data?.models ?? fromCatalog)
  return { models, fromProvider: Boolean(mine?.fromProvider), error: mine?.error, ask }
}

/** The model a dialog starts on: the provider's default when it's listed, else the first. */
function defaultModel(models: AIModelOption[], preferred?: string): string {
  if (preferred && models.some((o) => o.id === preferred)) return preferred
  return models[0]?.id ?? ''
}

/**
 * Connect an AI model (AISettings · the connect dialog): the provider, the
 * key, the model, Test connection, the cap and prices, the features, what is
 * sent, and "When you connect" beside it.
 */
export function ConnectDialog({ s, initial, open, onOpenChange }: { s: AISettings; initial: AIProvider; open: boolean; onOpenChange: (o: boolean) => void }) {
  const { slug } = useWorkspace()
  const t = useT()
  const me = useMe().data?.user
  const refresh = useRefreshAI()
  const kept = s.disconnected ? s.connection : undefined
  const [provider, setProvider] = useState<AIProvider>(initial)
  const entry = s.catalog.find((c) => c.kind === provider)
  const { models, fromProvider, error: listError, ask } = useModels(provider, s)
  const keepFor = kept && kept.provider === provider ? kept : undefined
  const [key, setKey] = useState('')
  const [label, setLabel] = useState(keepFor?.label ?? '')
  const [baseUrl, setBaseUrl] = useState(keepFor?.baseUrl ?? '')
  const [deployment, setDeployment] = useState(keepFor?.deployment ?? '')
  const [apiVersion, setApiVersion] = useState(keepFor?.apiVersion ?? '')
  // '' follows the provider's default model.
  const [model, setModel] = useState<string>(keepFor?.model ?? '')
  const [typed, setTyped] = useState('')
  const compatible = provider === 'compatible'
  const azure = provider === 'azure'
  const [capKind, setCapKind] = useState<CapKind>(keepFor?.cap.kind ?? (compatible ? 'tokens' : 'usd'))
  const [capUsd, setCapUsd] = useState(fixed(keepFor?.cap.usd ?? 25))
  const [capTokens, setCapTokens] = useState(String(keepFor?.cap.tokens || DEFAULT_TOKENS))
  // null follows the model's price-list prices until edited.
  const [inputPrice, setInputPrice] = useState<string | null>(keepFor ? fixed(keepFor.prices.input) : null)
  const [outputPrice, setOutputPrice] = useState<string | null>(keepFor ? fixed(keepFor.prices.output) : null)
  const [features, setFeatures] = useState<Set<AIFeature>>(new Set(AI_FEATURES))
  const [test, setTest] = useState<{ running: boolean; result?: AITest } | null>(null)
  const [priv, setPriv] = useState<Private | null>(null)
  const [problem, setProblem] = useState<string>()
  const [saving, setSaving] = useState(false)
  const keyRef = useRef<HTMLInputElement>(null)
  const name = t(aiEntryName[provider])

  // A new provider starts from its own defaults (or the kept connection's).
  function pick(p: AIProvider) {
    if (p === provider) return
    const keep = kept && kept.provider === p ? kept : undefined
    setProvider(p)
    setKey('')
    setLabel(keep?.label ?? '')
    setBaseUrl(keep?.baseUrl ?? '')
    setDeployment(keep?.deployment ?? '')
    setApiVersion(keep?.apiVersion ?? '')
    setModel(keep?.model ?? '')
    setTyped('')
    setCapKind(keep?.cap.kind ?? (p === 'compatible' ? 'tokens' : 'usd'))
    setInputPrice(keep ? fixed(keep.prices.input) : null)
    setOutputPrice(keep ? fixed(keep.prices.output) : null)
    setTest(null)
    setPriv(null)
    setProblem(undefined)
  }

  const effective = model || defaultModel(models, entry?.defaultModel)
  const option = models.find((o) => o.id === effective)
  const typing = effective === TYPED || (compatible && !option)
  const modelId = typing ? typed.trim() || (effective !== TYPED ? effective : '') : effective
  const shownInput = inputPrice ?? (option?.known ? fixed(option.input) : '')
  const shownOutput = outputPrice ?? (option?.known ? fixed(option.output) : '')
  function chooseModel(v: string) {
    setModel(v)
    setInputPrice(null)
    setOutputPrice(null)
    stale()
  }

  const stale = () => {
    setTest(null)
    setPriv(null)
    setProblem(undefined)
  }
  const candidate = { provider, baseUrl: baseUrl.trim() || undefined, deployment: deployment.trim() || undefined, apiVersion: apiVersion.trim() || undefined, apiKey: key.trim() || undefined, model: modelId || undefined }

  function failed(e: unknown) {
    const code = problemCode(e)
    if (code === 'ai_private_address') {
      const host = problemMeta<string>(e, 'host') ?? ''
      const ip = problemMeta<string>(e, 'ip')
      setPriv({ host, ip, message: host ? (ip && ip !== host ? t(m.privateRefusedBody, { host, ip }) : t(m.privateRefusedHost, { host })) : errorMessage(e) })
      return
    }
    if (code === 'ai_test_failed') {
      const r = problemMeta<AITest>(e, 'test')
      if (r) {
        setTest({ running: false, result: r })
        return
      }
    }
    setProblem(errorMessage(e))
  }

  async function runTest() {
    setTest({ running: true })
    setPriv(null)
    setProblem(undefined)
    try {
      const res = await http.POST(`${aiBase}/test`, { params: { path: { workspace: slug } }, body: candidate })
      setTest({ running: false, result: res.data })
    } catch (e) {
      setTest(null)
      failed(e)
    }
  }

  async function connect() {
    setSaving(true)
    setProblem(undefined)
    setPriv(null)
    try {
      await http.PUT(`${aiBase}/connection`, {
        params: { path: { workspace: slug } },
        body: {
          ...candidate,
          model: modelId,
          label: compatible ? label.trim() || undefined : undefined,
          capKind,
          capUsd: capKind === 'usd' ? (parse(capUsd) ?? 0) : undefined,
          capTokens: capKind === 'tokens' ? Math.round(parse(capTokens) ?? 0) : undefined,
          inputPrice: parse(shownInput),
          outputPrice: parse(shownOutput),
          features: AI_FEATURES.filter((f) => features.has(f)),
        },
      })
      await refresh()
      onOpenChange(false)
      toast({ kind: 'success', title: t(m.connectedToast, { provider: name }), detail: t(m.connectedDetail, { model: option?.name ?? modelId }) })
    } catch (e) {
      failed(e)
    } finally {
      setSaving(false)
    }
  }

  const imageless = Boolean(option && !option.imageInput)
  const on = AI_FEATURES.filter((f) => features.has(f) && !(f === 'alt_text' && imageless)).length
  const passed = test?.result?.ok ? test.result : undefined
  const admins = s.instanceAdmins
  return (
    <Dialog
      open={open}
      onOpenChange={onOpenChange}
      eyebrow={<span className="k">{t(m.connectEyebrow)}</span>}
      title={t(m.connectTitle)}
      className="ai-dlg"
      hideClose
      dismissible={!saving}
      footer={
        <>
          <span className="k mr-auto">{t(m.connectRecorded)}</span>
          <Button onClick={() => onOpenChange(false)}>{t(m.cancel)}</Button>
          <Button variant="primary" onClick={connect} loading={saving} busyLabel={t(m.connecting)} disabledReason={modelId ? undefined : t(m.modelNeeded)}>
            {t(m.connectButton, { provider: name })}
          </Button>
        </>
      }
    >
      <div className="ai-cgrid">
        <div className="ai-cmain">
          <div className="flex flex-col gap-1.5">
            <span className="lbl">{t(m.providerLabel)}</span>
            <Segmented
              label={t(m.providerLabel)}
              value={provider}
              onChange={pick}
              className="max-w-full flex-wrap self-start"
              items={AI_PROVIDERS.map((p) => ({ value: p, label: t(aiEntryName[p]) }))}
            />
          </div>

          {compatible ? (
            <div className="ai-pair">
              <label className="lbl">
                {t(m.serverAddress)}
                <input
                  className={`in mono text-[12.5px]${priv ? ' err' : ''}`}
                  value={baseUrl}
                  onChange={(e) => {
                    setBaseUrl(e.target.value)
                    stale()
                  }}
                  onBlur={() => baseUrl.trim() && void ask({ baseUrl: baseUrl.trim(), apiKey: key.trim() || undefined })}
                  placeholder={t(m.serverPlaceholder)}
                  spellCheck={false}
                  autoComplete="url"
                />
              </label>
              <label className="lbl">
                {t(m.serverName)}
                <input className="in" value={label} onChange={(e) => setLabel(e.target.value)} placeholder={t(m.serverNamePlaceholder)} maxLength={80} />
              </label>
            </div>
          ) : azure ? (
            <div className="ai-pair">
              <label className="lbl">
                {t(m.endpointLabel)}
                <input
                  className="in mono text-[12.5px]"
                  value={baseUrl}
                  onChange={(e) => {
                    setBaseUrl(e.target.value)
                    stale()
                  }}
                  placeholder={t(m.azurePlaceholder)}
                  spellCheck={false}
                  autoComplete="url"
                />
              </label>
              <label className="lbl">
                {t(m.deploymentLabel)}
                <input
                  className="in mono text-[12.5px]"
                  value={deployment}
                  onChange={(e) => {
                    setDeployment(e.target.value)
                    stale()
                  }}
                  spellCheck={false}
                  autoComplete="off"
                />
                <span className="k font-medium">{t(m.azureHint)}</span>
              </label>
            </div>
          ) : null}

          <div className="ai-pair">
            <label className="lbl">
              {compatible ? t(m.apiKeyOptional) : t(m.apiKey)}
              <input
                ref={keyRef}
                type="password"
                className="in mono text-[12.5px]"
                value={key}
                onChange={(e) => {
                  setKey(e.target.value)
                  stale()
                }}
                onBlur={() => key.trim() && void ask({ apiKey: key.trim(), baseUrl: baseUrl.trim() || undefined, deployment: deployment.trim() || undefined })}
                autoComplete="off"
                spellCheck={false}
              />
              <span className="k font-medium">{compatible ? t(m.apiKeyLocal) : t(m.apiKeyHint)}</span>
            </label>
            {azure ? (
              <label className="lbl">
                {t(m.apiVersionLabel)}
                <input className="in mono text-[12.5px]" value={apiVersion} onChange={(e) => setApiVersion(e.target.value)} placeholder="2024-10-21" spellCheck={false} autoComplete="off" />
              </label>
            ) : (
              <ModelField
                provider={name}
                models={models}
                value={effective}
                typed={typed}
                compatible={compatible}
                fromProvider={fromProvider}
                listError={listError}
                onChange={chooseModel}
                onTyped={(v) => {
                  setTyped(v)
                  stale()
                }}
              />
            )}
          </div>
          {azure ? (
            <div className="ai-pair">
              <ModelField
                provider={name}
                models={models}
                value={effective}
                typed={typed}
                compatible={false}
                fromProvider={false}
                hint={t(m.azureModelHint)}
                onChange={chooseModel}
                onTyped={(v) => {
                  setTyped(v)
                  stale()
                }}
              />
            </div>
          ) : null}

          {priv ? (
            <PrivateRefused priv={priv} admins={admins} isAdmin={Boolean(me?.isInstanceAdmin)} />
          ) : passed ? (
            <>
              <div className="ai-pass" role="status">
                <span>
                  <Icon name="check" size={16} strokeWidth={2.2} />
                  <span>
                    <T msg={m.testPassed} values={{ ...tags, seconds: seconds(passed.ms), model: option?.name ?? passed.model ?? modelId, answer: passed.answer ?? '' }} />
                  </span>
                </span>
                <Button size="sm" icon="refresh" onClick={runTest}>
                  {t(m.testAgain)}
                </Button>
              </div>
              {passed.privateHost ? (
                <AINote tone="ok">
                  <span className="font-semibold">{t(m.privateAllowedTitle)}</span>
                  <div>{t(key.trim() ? m.privateAllowedBody : m.privateAllowedNoKey, { host: passed.privateHost, ip: passed.privateIp ?? passed.privateHost })}</div>
                </AINote>
              ) : null}
            </>
          ) : test?.result ? (
            <TestPanel provider={name} running={false} result={test.result} embedded onPasteKey={() => keyRef.current?.focus()} onRetry={runTest} />
          ) : (
            <div className="row flex-wrap gap-3">
              <Button icon="refresh" onClick={runTest} loading={test?.running} busyLabel={t(m.testing)} disabledReason={modelId ? undefined : t(m.modelNeeded)}>
                {t(m.testConnection)}
              </Button>
              <span className="k">{t(m.testFirst)}</span>
            </div>
          )}

          <CapFields
            compatible={compatible}
            kind={capKind}
            onKind={setCapKind}
            usd={capUsd}
            onUsd={setCapUsd}
            tokensValue={capTokens}
            onTokens={setCapTokens}
            input={shownInput}
            onInput={setInputPrice}
            output={shownOutput}
            onOutput={setOutputPrice}
          />
          {capKind !== 'none' && !(compatible && capKind === 'tokens') ? (
            <span className="k">{option?.known ? t(m.pricesPrefilled, { date: calendarDay(s.priceListDate) }) : t(m.pricesUnknown)}</span>
          ) : null}

          <fieldset className="ai-checks">
            <legend>{on === AI_FEATURES.length ? t(m.featuresAllOn) : t(m.featuresSomeOn, { on, total: AI_FEATURES.length })}</legend>
            {AI_FEATURES.map((f) => {
              const off = f === 'alt_text' && imageless
              return (
                <label key={f} title={off ? t(m.altNeedsImages) : undefined}>
                  <input
                    type="checkbox"
                    checked={features.has(f) && !off}
                    disabled={off}
                    onChange={(e) => {
                      const next = new Set(features)
                      if (e.target.checked) next.add(f)
                      else next.delete(f)
                      setFeatures(next)
                    }}
                  />
                  {t(featureName[f])}
                </label>
              )
            })}
          </fieldset>
          {imageless ? <span className="k -mt-2">{t(m.altNeedsImages)}</span> : null}

          <LeavesNote provider={name} />
          {problem ? <AINote tone="bad">{problem}</AINote> : null}
        </div>
        <aside className="ai-cside" aria-labelledby="ai-when-h">
          <h3 id="ai-when-h">{t(m.whenTitle)}</h3>
          <ol>
            <li>{t(m.whenKey)}</li>
            <li>{t(m.whenFeatures, { roles: formatList(s.aiUseRoles.map((name) => roleName({ name }))) })}</li>
            <li>{t(m.whenLog)}</li>
            <li>{t(m.whenNothing, { provider: name })}</li>
          </ol>
          <span className="k">{t(m.whenAgents)}</span>
          {compatible ? (
            <div className="ih-caps">
              {aiCapabilities(true).compatible.map((c) => (
                <span key={c.id} className="ih-cap">
                  {t(c)}
                </span>
              ))}
            </div>
          ) : null}
        </aside>
      </div>
    </Dialog>
  )
}

/** The model: the list (the price list's, or the provider's), or a typed name. */
function ModelField({
  provider,
  models,
  value,
  typed,
  compatible,
  fromProvider,
  listError,
  hint,
  onChange,
  onTyped,
}: {
  provider: string
  models: AIModelOption[]
  value: string
  typed: string
  compatible: boolean
  fromProvider: boolean
  listError?: string
  hint?: string
  onChange: (v: string) => void
  onTyped: (v: string) => void
}) {
  const t = useT()
  // A compatible server's model is typed until the server lists its own.
  const typing = value === TYPED || (compatible && !models.some((o) => o.id === value))
  return (
    <div className="flex min-w-0 flex-col gap-1.5">
      <label className="lbl">
        {t(m.modelLabel)}
        {models.length ? (
          <select className="in" value={typing ? TYPED : value} onChange={(e) => onChange(e.target.value)}>
            {models.map((o) => (
              <option key={o.id} value={o.id}>
                {o.name && o.name !== o.id ? `${o.name} · ${o.id}` : o.id}
              </option>
            ))}
            <option value={TYPED}>{t(m.modelTypeOption)}</option>
          </select>
        ) : null}
        {typing || !models.length ? (
          <input
            className="in mono text-[12.5px]"
            value={typed || (value !== TYPED && !models.some((o) => o.id === value) ? value : '')}
            onChange={(e) => onTyped(e.target.value)}
            placeholder={compatible ? 'llama3.3:70b' : t(m.modelTypePlaceholder)}
            aria-label={models.length ? t(m.modelTypePlaceholder) : undefined}
            spellCheck={false}
            autoComplete="off"
          />
        ) : null}
      </label>
      <span className="k font-medium">{hint ?? (compatible ? (fromProvider ? t(m.modelHintServer) : t(m.modelTyped)) : listError && !fromProvider ? t(m.modelsListError, { provider }) : t(m.modelHint, { provider }))}</span>
    </div>
  )
}

/** The cap and prices: dollars with prices, tokens, or none. */
function CapFields({
  compatible,
  kind,
  onKind,
  usd,
  onUsd,
  tokensValue,
  onTokens,
  input,
  onInput,
  output,
  onOutput,
}: {
  compatible: boolean
  kind: CapKind
  onKind: (k: CapKind) => void
  usd: string
  onUsd: (v: string) => void
  tokensValue: string
  onTokens: (v: string) => void
  input: string
  onInput: (v: string) => void
  output: string
  onOutput: (v: string) => void
}) {
  const t = useT()
  const prices = (
    <>
      <label className="lbl">
        {t(m.inputPrice)}
        <span className="ai-dollar">
          <span aria-hidden="true">$</span>
          <input className="in" inputMode="decimal" value={input} onChange={(e) => onInput(e.target.value)} />
        </span>
        <span className="k font-medium">{t(m.perMillion)}</span>
      </label>
      <label className="lbl">
        {t(m.outputPrice)}
        <span className="ai-dollar">
          <span aria-hidden="true">$</span>
          <input className="in" inputMode="decimal" value={output} onChange={(e) => onOutput(e.target.value)} />
        </span>
        <span className="k font-medium">{t(m.perMillion)}</span>
      </label>
    </>
  )
  if (compatible) {
    const n = Math.round(parse(tokensValue) ?? DEFAULT_TOKENS)
    return (
      <div className="flex flex-col gap-3">
        <div className="ai-pair">
          <label className="lbl">
            {t(m.capLabel)}
            <select className="in" value={kind} onChange={(e) => onKind(e.target.value as CapKind)}>
              <option value="tokens">{t(m.capTokens, { amount: tokens(n) })}</option>
              <option value="usd">{t(m.capDollars)}</option>
              <option value="none">{t(m.capNone)}</option>
            </select>
            <span className="k font-medium">{t(m.capLocalHint)}</span>
          </label>
          {kind === 'tokens' ? (
            <label className="lbl">
              {t(m.capTokensPlain)}
              <input className="in" inputMode="numeric" value={tokensValue} onChange={(e) => onTokens(e.target.value.replace(/[^\d]/g, ''))} />
              <span className="k font-medium">{count(n)}</span>
            </label>
          ) : null}
        </div>
        {kind === 'usd' ? (
          <div className="ai-trio">
            <label className="lbl">
              {t(m.capLabel)}
              <span className="ai-dollar">
                <span aria-hidden="true">$</span>
                <input className="in" inputMode="decimal" value={usd} onChange={(e) => onUsd(e.target.value)} />
              </span>
              <span className="k font-medium">{t(m.capHint)}</span>
            </label>
            {prices}
          </div>
        ) : null}
      </div>
    )
  }
  return (
    <div className="ai-trio">
      <label className="lbl">
        {t(m.capLabel)}
        <span className="ai-dollar">
          <span aria-hidden="true">$</span>
          <input className="in" inputMode="decimal" value={usd} onChange={(e) => onUsd(e.target.value)} />
        </span>
        <span className="k font-medium">{t(m.capHint)}</span>
      </label>
      {prices}
    </div>
  )
}

/** "Can't connect to a private address", with who can allow it. */
function PrivateRefused({ priv, admins, isAdmin }: { priv: Private; admins: string[]; isAdmin: boolean }) {
  const t = useT()
  return (
    <div className="flex flex-col gap-2">
      <AINote tone="bad">
        <span className="font-semibold">{t(m.privateRefusedTitle)}</span>
        <div>{priv.message}</div>
        {isAdmin ? (
          <div className="mt-1">
            <T msg={m.privateHowTo} values={tags} />
          </div>
        ) : null}
      </AINote>
      <span className="k">{t(m.privateSame, { count: admins.length, admins: formatList(admins) })}</span>
    </div>
  )
}

/**
 * Test connection's result (AISettings · TEST CONNECTION): the fixed prompt,
 * the answer and its time, or the provider's refusal with what to do.
 */
export function TestPanel({
  provider,
  running,
  result,
  error,
  embedded,
  onClose,
  onPasteKey,
  onRetry,
}: {
  provider: string
  running: boolean
  result?: AITest
  error?: string
  embedded?: boolean
  onClose?: () => void
  onPasteKey?: () => void
  onRetry?: () => void
}) {
  const t = useT()
  const auth = result?.errorKind === 'auth'
  const badge = running ? null : result?.ok ? (
    <Badge tone="ok" dot>
      {t(m.passed)}
    </Badge>
  ) : auth ? (
    <Badge tone="bad" dot>
      {t(m.keyRejectedBadge)}
    </Badge>
  ) : (
    <Badge tone="bad" dot>
      {t(m.failed)}
    </Badge>
  )
  const body = (
    <>
      <header>
        <h2 className="h2">{t(m.testPanelTitle)}</h2>
        <div className="row gap-2">
          {badge}
          {onClose ? (
            <Button size="sm" iconOnly icon="close" className="border-transparent" aria-label={t(m.close)} onClick={onClose} />
          ) : null}
        </div>
      </header>
      {running ? (
        <div className="flex flex-col" aria-busy="true">
          <Ev icon="live" title={t(m.sentFixed)} sub={t(m.testing)} />
        </div>
      ) : error ? (
        <AINote tone="bad">{error}</AINote>
      ) : result ? (
        <div className="flex flex-col">
          <Ev icon="ok" title={t(m.sentFixed)} sub={t(m.sentFixedDetail, { prompt: result.prompt })} time={clockSeconds(result.sentAt)} />
          {result.ok ? (
            <Ev
              icon="ok"
              title={t(m.answered, { provider })}
              sub={t(m.answeredDetail, { answer: result.answer ?? '', model: result.model ?? '', seconds: seconds(result.ms), input: count(result.inputTokens), output: count(result.outputTokens) })}
              time={clockSeconds(result.answeredAt)}
            />
          ) : (
            <Ev
              icon="bad"
              title={auth ? t(m.refusedKey, { provider }) : t(m.noAnswer, { provider })}
              sub={auth && result.errorDetail ? <span className="mono">{result.errorDetail}</span> : (result.errorMessage ?? result.errorDetail)}
              time={clockSeconds(result.answeredAt)}
            />
          )}
        </div>
      ) : null}
      {result && !result.ok ? (
        <AINote tone="bad">
          <div className="flex flex-col items-start gap-2">
            <span>{auth ? t(m.checkKey, { provider }) : (result.errorMessage ?? t(m.failed))}</span>
            {auth && onPasteKey ? (
              <Button size="sm" onClick={onPasteKey}>
                {t(m.pasteNewKey)}
              </Button>
            ) : !auth && onRetry ? (
              <Button size="sm" icon="refresh" onClick={onRetry}>
                {t(m.testAgain)}
              </Button>
            ) : null}
          </div>
        </AINote>
      ) : null}
    </>
  )
  if (embedded) return <div className="ai-test card">{body}</div>
  return (
    <section className={`card ai-test${result && !result.ok ? ' ai-bad' : ''}`} aria-live="polite">
      {body}
    </section>
  )
}

/** Change model: another model of the same provider; the key stays. */
export function ChangeModelDialog({ s, c, open, onOpenChange }: { s: AISettings; c: AIConnection; open: boolean; onOpenChange: (o: boolean) => void }) {
  const { slug } = useWorkspace()
  const t = useT()
  const refresh = useRefreshAI()
  const provider = c.provider as AIProvider
  const { models, fromProvider, error: listError } = useModels(provider, s)
  const [model, setModel] = useState(c.model)
  const [typed, setTyped] = useState('')
  // null follows the chosen model's price-list prices until edited.
  const [input, setInput] = useState<string | null>(fixed(c.prices.input))
  const [output, setOutput] = useState<string | null>(fixed(c.prices.output))
  const [saving, setSaving] = useState(false)
  const [problem, setProblem] = useState<string>()
  const option = models.find((o) => o.id === model)
  const modelId = model === TYPED || (!option && provider === 'compatible') ? typed.trim() : model
  const shownInput = input ?? (option?.known ? fixed(option.input) : '')
  const shownOutput = output ?? (option?.known ? fixed(option.output) : '')
  async function save() {
    setSaving(true)
    setProblem(undefined)
    try {
      await http.PUT(`${aiBase}/model`, { params: { path: { workspace: slug } }, body: { model: modelId, inputPrice: parse(shownInput), outputPrice: parse(shownOutput) } })
      await refresh()
      onOpenChange(false)
      toast({ kind: 'success', title: t(m.modelChangedToast, { model: option?.name ?? modelId }) })
    } catch (e) {
      setProblem(errorMessage(e))
    } finally {
      setSaving(false)
    }
  }
  return (
    <Dialog
      open={open}
      onOpenChange={onOpenChange}
      eyebrow={<span className="k">{t(m.connectEyebrow)}</span>}
      title={t(m.changeTitle)}
      description={t(m.changeLead, { provider: c.providerName })}
      className="ai-dlg-md"
      dismissible={!saving}
      footer={
        <>
          <span className="k mr-auto">{t(m.changeRecorded)}</span>
          <Button onClick={() => onOpenChange(false)}>{t(m.cancel)}</Button>
          <Button variant="primary" onClick={save} loading={saving} disabledReason={modelId && modelId !== c.model ? undefined : t(m.modelSame)}>
            {t(m.saveButton)}
          </Button>
        </>
      }
    >
      <div className="row gap-3">
        <AITile provider={provider} size="sm" />
        <span className="font-semibold">{c.providerName}</span>
      </div>
      <ModelField
        provider={c.providerName}
        models={models}
        value={model}
        typed={typed}
        compatible={provider === 'compatible'}
        fromProvider={fromProvider}
        listError={listError}
        onChange={(v) => {
          setModel(v)
          setInput(null)
          setOutput(null)
        }}
        onTyped={setTyped}
      />
      <div className="ai-pair">
        <label className="lbl">
          {t(m.inputPrice)}
          <span className="ai-dollar">
            <span aria-hidden="true">$</span>
            <input className="in" inputMode="decimal" value={shownInput} onChange={(e) => setInput(e.target.value)} />
          </span>
          <span className="k font-medium">{t(m.perMillion)}</span>
        </label>
        <label className="lbl">
          {t(m.outputPrice)}
          <span className="ai-dollar">
            <span aria-hidden="true">$</span>
            <input className="in" inputMode="decimal" value={shownOutput} onChange={(e) => setOutput(e.target.value)} />
          </span>
          <span className="k font-medium">{t(m.perMillion)}</span>
        </label>
      </div>
      {option && !option.imageInput ? <AINote tone="warn">{t(m.altNeedsImages)}</AINote> : null}
      {problem ? <AINote tone="bad">{problem}</AINote> : null}
    </Dialog>
  )
}

/** Replace the key: the new key is tested before it's saved. */
export function ReplaceKeyDialog({ c, open, onOpenChange, onTested }: { c: AIConnection; open: boolean; onOpenChange: (o: boolean) => void; onTested: (r: AITest) => void }) {
  const { slug } = useWorkspace()
  const t = useT()
  const refresh = useRefreshAI()
  const [key, setKey] = useState('')
  const [saving, setSaving] = useState(false)
  const [result, setResult] = useState<AITest>()
  const [problem, setProblem] = useState<string>()
  const keyRef = useRef<HTMLInputElement>(null)
  async function save() {
    setSaving(true)
    setProblem(undefined)
    setResult(undefined)
    try {
      const res = await http.PUT(`${aiBase}/key`, { params: { path: { workspace: slug } }, body: { apiKey: key.trim() } })
      await refresh()
      onOpenChange(false)
      if (res.data) onTested(res.data.test)
      toast({ kind: 'success', title: t(m.keyReplacedToast), detail: t(m.keyReplacedDetail) })
    } catch (e) {
      const r = problemCode(e) === 'ai_test_failed' ? problemMeta<AITest>(e, 'test') : undefined
      if (r) setResult(r)
      else setProblem(errorMessage(e))
    } finally {
      setSaving(false)
    }
  }
  return (
    <Dialog
      open={open}
      onOpenChange={onOpenChange}
      eyebrow={<span className="k">{t(m.connectEyebrow)}</span>}
      title={c.key.set ? t(m.replaceTitle) : t(m.addKey)}
      description={t(m.replaceLead, { provider: c.providerName })}
      className="ai-dlg-md"
      dismissible={!saving}
      footer={
        <>
          <span className="k mr-auto">{t(m.changeRecorded)}</span>
          <Button onClick={() => onOpenChange(false)}>{t(m.cancel)}</Button>
          <Button variant="primary" onClick={save} loading={saving} busyLabel={t(m.testing)} disabledReason={key.trim() ? undefined : t(m.keyNeeded, { provider: c.providerName })}>
            {c.key.set ? t(m.replaceKey) : t(m.addKey)}
          </Button>
        </>
      }
    >
      <label className="lbl">
        {t(m.apiKey)}
        <input
          ref={keyRef}
          type="password"
          className="in mono text-[12.5px]"
          value={key}
          onChange={(e) => {
            setKey(e.target.value)
            setResult(undefined)
          }}
          autoComplete="off"
          spellCheck={false}
          autoFocus
        />
        <span className="k font-medium">{t(m.apiKeyHint)}</span>
      </label>
      {result ? <TestPanel provider={c.providerName} running={false} result={result} embedded onPasteKey={() => keyRef.current?.focus()} /> : null}
      {problem ? <AINote tone="bad">{problem}</AINote> : null}
    </Dialog>
  )
}

/** The monthly cap: dollars, tokens, or none. */
export function CapDialog({ c, open, onOpenChange }: { c: AIConnection; open: boolean; onOpenChange: (o: boolean) => void }) {
  const { slug } = useWorkspace()
  const t = useT()
  const refresh = useRefreshAI()
  const [kind, setKind] = useState<CapKind>(c.cap.kind)
  const [usd, setUsd] = useState(fixed(c.cap.usd || 25))
  const [tok, setTok] = useState(String(c.cap.tokens || DEFAULT_TOKENS))
  const [saving, setSaving] = useState(false)
  const [problem, setProblem] = useState<string>()
  async function save() {
    setSaving(true)
    setProblem(undefined)
    try {
      await http.PUT(`${aiBase}/cap`, {
        params: { path: { workspace: slug } },
        body: { kind, usd: kind === 'usd' ? (parse(usd) ?? 0) : undefined, tokens: kind === 'tokens' ? Math.round(parse(tok) ?? 0) : undefined },
      })
      await refresh()
      onOpenChange(false)
      toast({ kind: 'success', title: t(m.capSavedToast) })
    } catch (e) {
      setProblem(errorMessage(e))
    } finally {
      setSaving(false)
    }
  }
  const n = Math.round(parse(tok) ?? 0)
  return (
    <Dialog
      open={open}
      onOpenChange={onOpenChange}
      eyebrow={<span className="k">{t(m.connectEyebrow)}</span>}
      title={t(m.capDialogTitle)}
      description={t(m.capHint)}
      size="sm"
      dismissible={!saving}
      footer={
        <>
          <span className="k mr-auto">{t(m.changeRecorded)}</span>
          <Button onClick={() => onOpenChange(false)}>{t(m.cancel)}</Button>
          <Button variant="primary" onClick={save} loading={saving}>
            {t(m.saveButton)}
          </Button>
        </>
      }
    >
      <Segmented
        className="self-start"
        label={t(m.capKindLabel)}
        value={kind}
        onChange={setKind}
        items={[
          { value: 'usd', label: t(m.capKindUsd) },
          { value: 'tokens', label: t(m.capKindTokens) },
          { value: 'none', label: t(m.capNone) },
        ]}
      />
      {kind === 'usd' ? (
        <label className="lbl">
          {t(m.capLabel)}
          <span className="ai-dollar">
            <span aria-hidden="true">$</span>
            <input className="in" inputMode="decimal" value={usd} onChange={(e) => setUsd(e.target.value)} autoFocus />
          </span>
        </label>
      ) : kind === 'tokens' ? (
        <label className="lbl">
          {t(m.capTokensPlain)}
          <input className="in" inputMode="numeric" value={tok} onChange={(e) => setTok(e.target.value.replace(/[^\d]/g, ''))} autoFocus />
          <span className="k font-medium">{count(n)}</span>
        </label>
      ) : (
        <AINote tone="warn">{t(m.capNoneNote)}</AINote>
      )}
      {problem ? <AINote tone="bad">{problem}</AINote> : null}
    </Dialog>
  )
}

/** Prices per million tokens: what the cap is estimated with. */
export function PricesDialog({ c, open, onOpenChange }: { c: AIConnection; open: boolean; onOpenChange: (o: boolean) => void }) {
  const { slug } = useWorkspace()
  const t = useT()
  const refresh = useRefreshAI()
  const [input, setInput] = useState(fixed(c.prices.input))
  const [output, setOutput] = useState(fixed(c.prices.output))
  const [saving, setSaving] = useState(false)
  const [problem, setProblem] = useState<string>()
  async function save() {
    setSaving(true)
    setProblem(undefined)
    try {
      await http.PUT(`${aiBase}/prices`, { params: { path: { workspace: slug } }, body: { input: parse(input) ?? -1, output: parse(output) ?? -1 } })
      await refresh()
      onOpenChange(false)
      toast({ kind: 'success', title: t(m.pricesSavedToast) })
    } catch (e) {
      setProblem(errorMessage(e))
    } finally {
      setSaving(false)
    }
  }
  return (
    <Dialog
      open={open}
      onOpenChange={onOpenChange}
      eyebrow={<span className="k">{t(m.connectEyebrow)}</span>}
      title={t(m.pricesDialogTitle)}
      description={t(m.pricesLead)}
      size="sm"
      dismissible={!saving}
      footer={
        <>
          <span className="k mr-auto">{t(m.changeRecorded)}</span>
          <Button onClick={() => onOpenChange(false)}>{t(m.cancel)}</Button>
          <Button variant="primary" onClick={save} loading={saving}>
            {t(m.saveButton)}
          </Button>
        </>
      }
    >
      <div className="ai-pair">
        <label className="lbl">
          {t(m.inputPrice)}
          <span className="ai-dollar">
            <span aria-hidden="true">$</span>
            <input className="in" inputMode="decimal" value={input} onChange={(e) => setInput(e.target.value)} autoFocus />
          </span>
          <span className="k font-medium">{t(m.perMillion)}</span>
        </label>
        <label className="lbl">
          {t(m.outputPrice)}
          <span className="ai-dollar">
            <span aria-hidden="true">$</span>
            <input className="in" inputMode="decimal" value={output} onChange={(e) => setOutput(e.target.value)} />
          </span>
          <span className="k font-medium">{t(m.perMillion)}</span>
        </label>
      </div>
      {c.prices.from ? <span className="k">{t(m.pricesPrefilled, { date: calendarDay(c.prices.from) })}</span> : null}
      {problem ? <AINote tone="bad">{problem}</AINote> : null}
    </Dialog>
  )
}

/** Disconnect {provider}? (AISettings · DISCONNECT CONFIRMATION). */
export function DisconnectDialog({ c, open, onOpenChange }: { c: AIConnection; open: boolean; onOpenChange: (o: boolean) => void }) {
  const { slug } = useWorkspace()
  const t = useT()
  const refresh = useRefreshAI()
  const [busy, setBusy] = useState(false)
  const [problem, setProblem] = useState<string>()
  async function disconnect() {
    setBusy(true)
    setProblem(undefined)
    try {
      await http.DELETE(`${aiBase}/connection`, { params: { path: { workspace: slug } } })
      await refresh()
      onOpenChange(false)
      toast({ kind: 'deleted', title: t(m.disconnectedToast, { provider: c.providerName }), detail: t(m.disconnectedDetail, { time: clock(new Date()) }) })
    } catch (e) {
      setProblem(errorMessage(e))
    } finally {
      setBusy(false)
    }
  }
  return (
    <Dialog
      open={open}
      onOpenChange={onOpenChange}
      eyebrow={<span className="k">{t(m.dlgEyebrow)}</span>}
      title={t(m.dlgTitle, { provider: c.providerName })}
      className="ai-dlg-md"
      hideClose
      dismissible={!busy}
      footer={
        <>
          <span className="k mr-auto">{t(m.recorded)}</span>
          <Button onClick={() => onOpenChange(false)}>{t(m.cancel)}</Button>
          <Button variant="danger-primary" onClick={disconnect} loading={busy} busyLabel={t(m.disconnecting)}>
            {t(m.disconnectConfirm, { provider: c.providerName })}
          </Button>
        </>
      }
    >
      <p className="m-0 text-[14px]">{t(m.dlgLead)}</p>
      <div className="ai-chips">
        {AI_FEATURES.map((f) => (
          <span key={f}>{t(featureName[f])}</span>
        ))}
      </div>
      <div className="ai-boxes">
        <div>
          <b>{t(m.removed)}</b>
          <span className="text-ink-2">{t(m.removedBody)}</span>
        </div>
        <div>
          <b>{t(m.kept)}</b>
          <span className="text-ink-2">{t(m.keptBody)}</span>
        </div>
      </div>
      {problem ? <AINote tone="bad">{problem}</AINote> : null}
    </Dialog>
  )
}
