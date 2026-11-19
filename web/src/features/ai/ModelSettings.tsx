import { Link } from '@tanstack/react-router'
import { useState } from 'react'
import { errorMessage, http, type Schemas } from '@/api/client'
import { Badge } from '@/components/ui/Badge'
import { Button } from '@/components/ui/Button'
import { ConfirmDialog } from '@/components/ui/ConfirmDialog'
import { Dialog } from '@/components/ui/Dialog'
import { Field, Input, Select, Toggle } from '@/components/ui/Field'
import { Table, Td, Th } from '@/components/ui/Table'
import { toast } from '@/components/ui/Toast'
import { T, formatList, useT, type MessageDescriptor } from '@/i18n'
import { clock, count, dayMonthDate, money, shortDate } from '@/lib/format'
import { useCan, useWorkspace } from '@/lib/session'
import { LimitFields } from './AgentSettings'
import { aiBase, AI_FEATURES, problemCode, useRefreshAI, type AISettings, type AITest } from './api'
import { AddModelDialog, ConnectDialog, PricesDialog, ReplaceKeyDialog, TestPanel, type AccountRef } from './ConnectDialog'
import { dollars, seconds, tokens } from './format'
import { m } from './messages'
import { mm } from './models.messages'
import { AINote, CardHead, EyeOffIcon, featureName, tags } from './parts'

type Account = Schemas['AIAccountView']
type Model = Schemas['AIModelView']
type Personal = Schemas['AIPersonalSettings']
type ModelFeature = Schemas['AIFeatureDefault']['feature']
type Translate = ReturnType<typeof useT>

/** Every feature with a model, in the board's order: the assistant and the agent first. */
const MODEL_FEATURES: ModelFeature[] = ['assistant', 'agent', ...AI_FEATURES]

function featureLabel(t: Translate, f: ModelFeature): string {
  return t(f === 'assistant' ? mm.featureAssistant : f === 'agent' ? mm.featureAgent : featureName[f])
}

const accountName = (a: Account) => a.label || a.providerName
const modelName = (x: Model) => x.modelName || x.model

function hostOf(url: string | undefined): string {
  if (!url) return ''
  try {
    return new URL(url).host
  } catch {
    return url
  }
}

/** A removed account's button is gone with it: focus goes to the card's Add button. */
function focusAdd(personal?: boolean) {
  window.setTimeout(() => document.getElementById(personal ? 'ai-yk-add' : 'ai-acc-add')?.focus())
}

/** Who set the key and when, or why there is none. */
function accountLine(t: Translate, a: Account, personal = false): string {
  if (a.status === 'disconnected') return t(mm.keyRemoved)
  if (a.key.set) {
    const date = dayMonthDate(a.key.addedAt ?? a.connectedAt)
    if (personal) return t(mm.yourKeySet, { date })
    return a.key.addedBy ? t(mm.keySetBy, { date, name: a.key.addedBy }) : t(mm.keySet, { date })
  }
  const host = hostOf(a.baseUrl)
  return a.privateNetwork ? t(mm.noKeyPrivate, { host }) : t(mm.noKeyHost, { host })
}

function AccountStatus({ a }: { a: Pick<Account, 'status'> & { models: Pick<Model, 'tools'>[] } }) {
  const t = useT()
  if (a.status === 'key_rejected')
    return (
      <Badge tone="bad" dot>
        {t(m.statusKeyRejected)}
      </Badge>
    )
  if (a.status === 'disconnected')
    return (
      <Badge tone="neutral" dot>
        {t(mm.statusNoKey)}
      </Badge>
    )
  if (a.models.some((x) => x.tools.state === 'unsupported'))
    return (
      <Badge tone="warn" dot>
        {t(mm.statusToolsFailed)}
      </Badge>
    )
  return (
    <Badge tone="ok" dot>
      {t(m.statusConnected)}
    </Badge>
  )
}

/** Why a model can't be a feature's: its key, images for alt text, the tool check for the assistant and the agent. */
function whyNot(f: ModelFeature, a: Account, x: Model): MessageDescriptor | undefined {
  if (a.status === 'key_rejected') return mm.whyKey
  if (a.status === 'disconnected') return mm.whyNoKey
  if (f === 'alt_text' && !x.imageInput) return mm.whyImages
  if (f === 'assistant' || f === 'agent') {
    if (x.tools.state === 'unsupported') return mm.whyTools
    if (x.tools.state !== 'supported') return mm.whyUntested
  }
  return undefined
}

/** The rejected key's notice, as the first connection's header said it. */
function KeyRejected({ a, onReplace }: { a: Account; onReplace?: () => void }) {
  const t = useT()
  const at = a.statusAt ?? a.connectedAt
  const today = dayMonthDate(at) === dayMonthDate(new Date())
  return (
    <AINote tone="bad">
      <div className="flex flex-col items-start gap-2">
        <span>
          <T
            msg={m.keyRejectedBody}
            values={{ ...tags, provider: accountName(a), when: today ? t(m.keyRejectedToday, { time: clock(at) }) : t(m.keyRejectedOn, { date: dayMonthDate(at) }), detail: a.statusDetail ?? '' }}
          />
        </span>
        {onReplace ? (
          <Button size="sm" onClick={onReplace}>
            {t(m.replaceKey)}
          </Button>
        ) : null}
      </div>
    </AINote>
  )
}

/** Model accounts (AIModels): each account, its models and its status; Add account and Manage for managers. */
export function ModelAccounts({ s }: { s: AISettings }) {
  const t = useT()
  const [adding, setAdding] = useState(false)
  const [managing, setManaging] = useState<string>()
  const [reconnecting, setReconnecting] = useState(false)
  const c = s.connection
  // The workspace's first account after Disconnect: its models, cap and prices
  // are kept, and it connects again (not as another account) with a key.
  const kept = s.canManage && s.disconnected ? s.accounts.find((a) => a.primary && a.status === 'disconnected') : undefined
  const reconnect = () => {
    setManaging(undefined)
    setReconnecting(true)
  }
  const u = s.usage
  const total = !c
    ? undefined
    : c.cap.kind === 'tokens'
      ? t(mm.totalTokens, { spent: tokens(u.tokens), cap: tokens(c.cap.tokens) })
      : c.cap.kind === 'none'
        ? t(mm.totalNoCap, { spent: dollars(u.costMicros) })
        : t(mm.totalUsd, { spent: dollars(u.costMicros), cap: money(c.cap.usd) })
  const managed = s.accounts.find((a) => a.id === managing)
  const rejected = s.accounts.filter((a) => a.status === 'key_rejected')
  return (
    <section className="card ai-card" aria-labelledby="ai-acc-h">
      <CardHead id="ai-acc-h" title={t(mm.accountsTitle)} caption={t(mm.accountsCaption)}>
        {s.canManage ? (
          <Button id="ai-acc-add" variant="primary" icon="plus" onClick={() => setAdding(true)}>
            {t(mm.addAccount)}
          </Button>
        ) : null}
      </CardHead>
      {rejected.length ? (
        <div className="ai-cb">
          {rejected.map((a) => (
            <KeyRejected key={a.id} a={a} onReplace={s.canManage ? () => setManaging(a.id) : undefined} />
          ))}
        </div>
      ) : null}
      <Table fit caption={t(mm.accountsTitle)}>
        <thead>
          <tr>
            <Th>{t(mm.thAccount)}</Th>
            <Th>{t(mm.thModels)}</Th>
            <Th>{t(mm.thStatus)}</Th>
            {s.canManage ? (
              <Th className="ih-no-phone">
                <span className="vh">{t(mm.thActions)}</span>
              </Th>
            ) : null}
          </tr>
        </thead>
        <tbody>
          {s.accounts.map((a) => (
            <tr key={a.id}>
              <Td>
                <span className="flex min-w-0 flex-col gap-0.5">
                  <b>{accountName(a)}</b>
                  <span className="k">{accountLine(t, a)}</span>
                  {a.demo ? <span className="k">{t(mm.demoAccount)}</span> : null}
                  {/* On a phone the table scrolls: Manage goes with the account. */}
                  {s.canManage ? (
                    <span className="ih-only-phone mt-2 items-start gap-1.5">
                      {a === kept ? (
                        <Button size="sm" variant="primary" onClick={reconnect}>
                          {t(mm.reconnect)}
                        </Button>
                      ) : null}
                      <Button size="sm" aria-label={t(mm.manageLabel, { account: accountName(a) })} onClick={() => setManaging(a.id)}>
                        {t(mm.manage)}
                      </Button>
                    </span>
                  ) : null}
                </span>
              </Td>
              <Td>{a.models.map(modelName).join(', ')}</Td>
              <Td nowrap>
                <AccountStatus a={a} />
              </Td>
              {s.canManage ? (
                <Td align="right" className="ih-no-phone">
                  <span className="inline-flex gap-1.5">
                    {a === kept ? (
                      <Button size="sm" variant="primary" onClick={reconnect}>
                        {t(mm.reconnect)}
                      </Button>
                    ) : null}
                    <Button size="sm" aria-label={t(mm.manageLabel, { account: accountName(a) })} onClick={() => setManaging(a.id)}>
                      {t(mm.manage)}
                    </Button>
                  </span>
                </Td>
              ) : null}
            </tr>
          ))}
        </tbody>
      </Table>
      {total ? (
        <div className="ai-foot">
          <span className="k">{total}</span>
        </div>
      ) : null}
      {adding ? <ConnectDialog s={s} initial="anthropic" target="account" open onOpenChange={(o) => !o && setAdding(false)} /> : null}
      {/* Without a target the dialog connects the kept account again, with its settings (PUT /ai/connection). */}
      {reconnecting && kept ? <ConnectDialog s={s} initial={kept.provider} open onOpenChange={(o) => !o && setReconnecting(false)} /> : null}
      {managed ? <AccountDialog s={s} a={managed} owner={{ id: managed.id }} onClose={() => setManaging(undefined)} onReconnect={managed === kept ? reconnect : undefined} /> : null}
    </section>
  )
}

type Sub = { kind: 'key' } | { kind: 'model' } | { kind: 'prices'; model: Model } | { kind: 'removeModel'; model: Model } | { kind: 'removeAccount' } | null

/**
 * One account (Manage): its key, its models with their prices and tool
 * check, Add model and Remove. The workspace's for managers; a person's
 * own (owner.personal) for that person.
 */
function AccountDialog({ s, a, owner, onClose, onReconnect }: { s: AISettings; a: Account; owner: AccountRef; onClose: () => void; onReconnect?: () => void }) {
  const { slug } = useWorkspace()
  const t = useT()
  const refresh = useRefreshAI()
  const [sub, setSub] = useState<Sub>(null)
  const [checking, setChecking] = useState<string>()
  const [test, setTest] = useState<{ running: boolean; result?: AITest; error?: string } | null>(null)
  const name = accountName(a)
  const compatible = a.provider === 'compatible'
  const usable = a.status === 'connected'
  const closeSub = (o: boolean) => !o && setSub(null)
  const path = { workspace: slug, account: a.id }

  async function checkTools(x: Model) {
    setChecking(x.id)
    try {
      const p = { params: { path: { ...path, model: x.id } } }
      const res = owner.personal ? await http.POST(`${aiBase}/personal/accounts/{account}/models/{model}/test-tools`, p) : await http.POST(`${aiBase}/accounts/{account}/models/{model}/test-tools`, p)
      await refresh()
      const ok = res.data?.state === 'supported'
      toast(ok ? { kind: 'success', title: t(mm.toolsPassedToast, { model: modelName(x) }) } : { kind: 'error', title: t(mm.toolsFailedToast, { model: modelName(x) }), detail: t(mm.toolsFailedDetail) })
    } catch (e) {
      toast({ kind: 'error', title: errorMessage(e) })
    } finally {
      setChecking(undefined)
    }
  }

  // The first account's saved connection has Test connection (the endpoint tests it alone).
  async function runTest() {
    setTest({ running: true })
    try {
      const res = await http.POST(`${aiBase}/test`, { params: { path: { workspace: slug } }, body: {} })
      setTest({ running: false, result: res.data })
    } catch (e) {
      setTest({ running: false, error: errorMessage(e) })
    } finally {
      void refresh()
    }
  }

  async function removeModel(x: Model) {
    const p = { params: { path: { ...path, model: x.id } } }
    if (owner.personal) await http.DELETE(`${aiBase}/personal/accounts/{account}/models/{model}`, p)
    else await http.DELETE(`${aiBase}/accounts/{account}/models/{model}`, p)
    await refresh()
    toast({ kind: 'deleted', title: t(mm.modelRemovedToast, { model: modelName(x) }) })
  }

  async function removeAccount() {
    if (owner.personal) await http.DELETE(`${aiBase}/personal/accounts/{account}`, { params: { path } })
    else await http.DELETE(`${aiBase}/accounts/{account}`, { params: { path } })
    await refresh()
    onClose()
    focusAdd(owner.personal)
    toast({ kind: 'deleted', title: owner.personal ? t(mm.yourKeyRemovedToast, { provider: a.providerName }) : t(mm.accountRemovedToast, { account: name }) })
  }

  const lt = a.lastTest
  const lastTest = !lt ? t(m.lastTestNever) : lt.ok ? t(m.lastTestAnswered, { date: shortDate(lt.at), time: clock(lt.at), seconds: seconds(lt.ms) }) : t(m.lastTestFailed, { date: shortDate(lt.at), time: clock(lt.at), seconds: seconds(lt.ms) })
  return (
    <Dialog
      open
      onOpenChange={(o) => !o && onClose()}
      eyebrow={<span className="k">{t(owner.personal ? mm.yourTitle : mm.accountEyebrow)}</span>}
      title={name}
      className="ai-dlg-md"
      footer={
        <>
          <Button variant="danger" className="mr-auto" onClick={() => setSub({ kind: 'removeAccount' })}>
            {t(owner.personal ? mm.removeKey : mm.removeAccount)}
          </Button>
          <Button onClick={onClose}>{t(m.close)}</Button>
        </>
      }
    >
      {a.status === 'key_rejected' ? <KeyRejected a={a} /> : null}
      <div className="ai-rows ai-facts">
        <div>
          <span>
            <span className="k">{t(m.kvKey)}</span>
            {a.key.set ? (
              <span className="inline-flex flex-wrap items-center gap-2">
                <span className="ai-mask">
                  <EyeOffIcon />
                  <span aria-label={t(m.keyHidden)}>••••••</span>
                  <Badge tone="neutral" className="h-5 font-mono text-[11px]">
                    {t(m.writeOnly)}
                  </Badge>
                </span>
                {a.key.addedAt ? (
                  <span className="k">{a.key.addedBy ? t(m.keyAdded, { date: dayMonthDate(a.key.addedAt), name: a.key.addedBy }) : t(m.keyAddedNoBy, { date: dayMonthDate(a.key.addedAt) })}</span>
                ) : null}
              </span>
            ) : (
              <span className="k">{t(m.keyNone)}</span>
            )}
          </span>
          {a.demo ? null : onReconnect ? (
            // The kept account's key comes with connecting it again: its own key endpoint refuses it.
            <Button size="sm" variant="primary" className="flex-none" onClick={onReconnect}>
              {t(mm.reconnect)}
            </Button>
          ) : (
            <Button size="sm" className="flex-none" onClick={() => setSub({ kind: 'key' })}>
              {a.key.set ? t(m.replace) : t(m.addKey)}
            </Button>
          )}
        </div>
        <div>
          <span>
            <span className="k">{compatible ? t(m.kvServer) : t(m.kvEndpoint)}</span>
            <span className="mono text-[12.5px]">{a.baseUrl || a.providerName}</span>
          </span>
        </div>
        <div>
          <span>
            <span className="k">{t(m.kvLastTest)}</span>
            <span className="text-[13px]">{lastTest}</span>
          </span>
          {a.primary && !owner.personal ? (
            <Button size="sm" icon="refresh" className="flex-none" onClick={runTest} loading={test?.running} busyLabel={t(m.testing)}>
              {t(m.testConnection)}
            </Button>
          ) : null}
        </div>
      </div>
      {test ? <TestPanel provider={a.providerName} running={test.running} result={test.result} error={test.error} embedded onPasteKey={() => setSub({ kind: 'key' })} /> : null}

      <h3 className="h2 m-0">{t(mm.modelsHeading)}</h3>
      <div className="ai-rows ai-facts">
        {a.models.map((x) => {
          const used = x.features.map((f) => featureLabel(t, f))
          const checked = x.tools.checkedAt ? dayMonthDate(x.tools.checkedAt) : ''
          return (
            <div key={x.id} className="flex-wrap">
              <span>
                <b>
                  {modelName(x)}
                  {x.modelName && x.modelName !== x.model ? <span className="mono k font-normal"> · {x.model}</span> : null}
                </b>
                <span className="k">{x.prices.known ? t(mm.modelPrices, { input: money(x.prices.input), output: money(x.prices.output) }) : t(mm.modelNoPrice)}</span>
                <span className="k">
                  {x.tools.state === 'supported' ? t(mm.toolsPassed, { date: checked }) : x.tools.state === 'unsupported' ? t(mm.toolsFailed, { date: checked }) : t(mm.toolsUntested)}
                  {' · '}
                  {used.length ? t(mm.usedBy, { features: formatList(used) }) : t(mm.usedByNone)}
                </span>
              </span>
              <div className="flex flex-wrap gap-1.5">
                <Button size="sm" aria-label={t(mm.editPricesLabel, { model: modelName(x) })} onClick={() => setSub({ kind: 'prices', model: x })}>
                  {t(mm.editPrices)}
                </Button>
                <Button
                  size="sm"
                  aria-label={t(mm.runToolCheckLabel, { model: modelName(x) })}
                  onClick={() => void checkTools(x)}
                  loading={checking === x.id}
                  busyLabel={t(m.testing)}
                  disabledReason={usable ? undefined : t(mm.toolsNeedKey)}
                >
                  {t(mm.runToolCheck)}
                </Button>
                <Button
                  size="sm"
                  variant="danger"
                  aria-label={t(mm.removeModelLabel, { model: modelName(x) })}
                  onClick={() => setSub({ kind: 'removeModel', model: x })}
                  disabledReason={a.models.length === 1 ? t(mm.onlyModel) : undefined}
                >
                  {t(mm.removeModel)}
                </Button>
              </div>
            </div>
          )
        })}
      </div>
      <Button className="self-start" icon="plus" onClick={() => setSub({ kind: 'model' })} disabledReason={usable ? undefined : t(mm.toolsNeedKey)}>
        {t(mm.addModel)}
      </Button>
      <p className="k m-0">{t(owner.personal ? mm.toolsCostPersonal : mm.toolsCost)}</p>

      {sub?.kind === 'key' ? <ReplaceKeyDialog c={a} account={owner} open onOpenChange={closeSub} onTested={(r) => setTest({ running: false, result: r })} /> : null}
      {sub?.kind === 'model' ? <AddModelDialog s={s} account={a} owner={owner} open onOpenChange={closeSub} /> : null}
      {sub?.kind === 'prices' ? <PricesDialog c={sub.model} model={{ id: sub.model.id, name: modelName(sub.model), account: owner }} open onOpenChange={closeSub} /> : null}
      {sub?.kind === 'removeModel' ? (
        <ConfirmDialog
          open
          onOpenChange={closeSub}
          badge={t(mm.removeBadge)}
          title={t(mm.removeModelTitle, { model: modelName(sub.model) })}
          description={t(mm.removeModelBody)}
          confirmLabel={t(mm.removeModel)}
          onConfirm={() => removeModel(sub.model)}
        />
      ) : null}
      {sub?.kind === 'removeAccount' ? (
        <ConfirmDialog
          open
          onOpenChange={closeSub}
          badge={t(owner.personal ? mm.removeKey : mm.removeAccount)}
          title={owner.personal ? t(mm.removeYourTitle, { provider: a.providerName }) : t(mm.removeAccountTitle, { account: name })}
          description={owner.personal ? t(mm.removeYourBody) : t(mm.removeAccountBody, { count: a.models.length })}
          confirmLabel={t(owner.personal ? mm.removeKey : mm.removeAccount)}
          onConfirm={removeAccount}
        />
      ) : null}
    </Dialog>
  )
}

/**
 * A feature's model as a select: the models grouped by account, each one
 * that can't be the feature's saying why (the board's "· failed the tool
 * check").
 */
function ModelSelect({
  id,
  feature,
  accounts,
  value,
  disabled,
  first,
  onChange,
}: {
  id: string
  feature: ModelFeature
  accounts: Account[]
  value: string
  disabled: boolean
  /** The option before the models: none chosen yet, or "the workspace's model". */
  first?: { value: string; label: string; disabled?: boolean }
  onChange: (modelId: string) => void
}) {
  const t = useT()
  return (
    <Select id={id} className="w-full min-w-0 max-w-[320px]" value={value} disabled={disabled} onChange={(e) => onChange(e.target.value)}>
      {first ? (
        <option value={first.value} disabled={first.disabled}>
          {first.label}
        </option>
      ) : null}
      {accounts.map((a) => (
        <optgroup key={a.id} label={accountName(a)}>
          {a.models.map((x) => {
            const why = whyNot(feature, a, x)
            return (
              <option key={x.id} value={x.id} disabled={Boolean(why) && x.id !== value}>
                {why ? t(why, { model: modelName(x) }) : modelName(x)}
              </option>
            )
          })}
        </optgroup>
      ))}
    </Select>
  )
}

/** Default model for each feature (AIModels): the assistant and the agent only on a model that passed the tool check. */
export function FeatureModels({ s }: { s: AISettings }) {
  const { slug } = useWorkspace()
  const t = useT()
  const refresh = useRefreshAI()
  const [busy, setBusy] = useState<ModelFeature>()
  const [refused, setRefused] = useState<{ feature: ModelFeature; message: string }>()
  const models = s.accounts.flatMap((a) => a.models)
  async function pick(f: ModelFeature, modelId: string) {
    setBusy(f)
    setRefused(undefined)
    try {
      await http.PUT(`${aiBase}/features/{feature}/model`, { params: { path: { workspace: slug, feature: f } }, body: { modelId } })
      await refresh()
      const x = models.find((y) => y.id === modelId)
      toast({ kind: 'success', title: t(mm.modelPickedToast, { feature: featureLabel(t, f), model: x ? modelName(x) : '' }) })
    } catch (e) {
      setRefused({ feature: f, message: errorMessage(e) })
      void refresh()
    } finally {
      setBusy(undefined)
    }
  }
  return (
    <section className="card ai-card" aria-labelledby="ai-fm-h">
      <CardHead id="ai-fm-h" title={t(mm.defaultsTitle)} />
      <Table fit caption={t(mm.defaultsTitle)}>
        <thead>
          <tr>
            <Th>{t(mm.thFeature)}</Th>
            <Th>{t(mm.thModel)}</Th>
          </tr>
        </thead>
        <tbody>
          {MODEL_FEATURES.map((f) => {
            const id = `ai-fm-${f}`
            const value = s.defaults.find((d) => d.feature === f)?.modelId ?? ''
            return (
              <tr key={f}>
                <Td>
                  <label htmlFor={id}>{featureLabel(t, f)}</label>
                </Td>
                <Td>
                  <ModelSelect
                    id={id}
                    feature={f}
                    accounts={s.accounts}
                    value={value}
                    disabled={!s.canManage || busy === f}
                    first={value ? undefined : { value: '', label: t(mm.chooseModel), disabled: true }}
                    onChange={(v) => void pick(f, v)}
                  />
                </Td>
              </tr>
            )
          })}
        </tbody>
      </Table>
      {refused ? (
        <div className="ai-cb">
          <AINote tone="bad">
            <b>{t(mm.pickRefused, { feature: featureLabel(t, refused.feature) })}</b> {refused.message}
          </AINote>
        </div>
      ) : null}
      <div className="ai-foot">
        <span className="k">{t(mm.defaultsFoot)}</span>
      </div>
    </section>
  )
}

/** The assistant (AIModels): its switch, its limits per thread and at once, and what it keeps. */
export function AssistantCard({ s }: { s: AISettings }) {
  const { slug } = useWorkspace()
  const t = useT()
  const refresh = useRefreshAI()
  const [saving, setSaving] = useState(false)
  const [editing, setEditing] = useState(false)
  const a = s.assistant
  async function toggle(enabled: boolean) {
    setSaving(true)
    try {
      await http.PUT(`${aiBase}/assistant/settings`, { params: { path: { workspace: slug } }, body: { ...a, enabled } })
      await refresh()
      toast(enabled ? { kind: 'success', title: t(mm.assistantOnToast) } : { kind: 'success', title: t(mm.assistantOffToast), detail: t(mm.assistantOffDetail) })
    } catch (e) {
      toast({ kind: 'error', title: problemCode(e) === 'ai_assistant_changed' ? t(mm.assistantChanged) : errorMessage(e) })
      void refresh()
    } finally {
      setSaving(false)
    }
  }
  const limits = t(mm.limitValue, { calls: a.limits.toolCalls, tokens: count(a.limits.tokens), cost: dollars(a.limits.costMicros) })
  return (
    <section className="card ai-card" aria-labelledby="ai-asst-h">
      <CardHead id="ai-asst-h" title={t(mm.assistantTitle)}>
        <Toggle label={t(mm.assistantToggle)} hideLabel checked={a.enabled} disabled={!s.canManage || saving} onChange={(e) => void toggle(e.target.checked)} />
      </CardHead>
      <div className="ai-cb">
        <div className="ai-rows ai-facts">
          <div>
            <span>{t(mm.limitPerThread)}</span>
            <b>{limits}</b>
          </div>
          <div>
            <span>{t(mm.atOnce)}</span>
            <b>{t(mm.atOnceValue, { person: a.perPerson, workspace: a.perWorkspace })}</b>
          </div>
          <div>
            <span>{t(mm.contentKept)}</span>
            <b>{t(mm.contentKeptValue)}</b>
          </div>
        </div>
        {s.canManage ? (
          <Button size="sm" className="self-start" onClick={() => setEditing(true)}>
            {t(mm.editLimits)}
          </Button>
        ) : null}
        <AINote tone="neutral" role="note">
          <b>{t(mm.leavesTitle)}</b>
          <div>{t(mm.leavesBody)}</div>
          <Link to="/w/$workspace/settings/integrations/mcp-servers" params={{ workspace: slug }}>
            {t(mm.mcpLink)}
          </Link>
        </AINote>
      </div>
      <div className="ai-foot">
        <span className="k">{t(mm.limitsFoot)}</span>
      </div>
      {editing ? <AssistantLimitsDialog s={s} onClose={() => setEditing(false)} /> : null}
    </section>
  )
}

function AssistantLimitsDialog({ s, onClose }: { s: AISettings; onClose: () => void }) {
  const { slug } = useWorkspace()
  const t = useT()
  const refresh = useRefreshAI()
  const a = s.assistant
  const [limits, setLimits] = useState(a.limits)
  const [perPerson, setPerPerson] = useState(a.perPerson)
  const [perWorkspace, setPerWorkspace] = useState(a.perWorkspace)
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState<string>()
  async function save() {
    setBusy(true)
    setError(undefined)
    try {
      await http.PUT(`${aiBase}/assistant/settings`, { params: { path: { workspace: slug } }, body: { enabled: a.enabled, limits, perPerson, perWorkspace, revision: a.revision } })
      await refresh()
      onClose()
      toast({ kind: 'success', title: t(mm.limitsSavedToast) })
    } catch (e) {
      setError(problemCode(e) === 'ai_assistant_changed' ? t(mm.assistantChanged) : errorMessage(e))
      if (problemCode(e) === 'ai_assistant_changed') void refresh()
    } finally {
      setBusy(false)
    }
  }
  return (
    <Dialog
      open
      onOpenChange={(o) => !o && !busy && onClose()}
      dismissible={!busy}
      eyebrow={<span className="k">{t(mm.assistantTitle)}</span>}
      title={t(mm.limitsTitle)}
      description={t(mm.limitsLead)}
      size="lg"
      footer={
        <>
          <span className="k mr-auto">{t(m.changeRecorded)}</span>
          <Button onClick={onClose} disabled={busy}>
            {t(m.cancel)}
          </Button>
          <Button type="submit" form="ai-asst-limits" variant="primary" loading={busy}>
            {t(m.saveButton)}
          </Button>
        </>
      }
    >
      <form
        id="ai-asst-limits"
        className="flex flex-col gap-4"
        onSubmit={(e) => {
          e.preventDefault()
          if (!busy) void save()
        }}
      >
        <fieldset disabled={busy} className="min-w-0">
          <legend className="mb-3 text-sm font-semibold">{t(mm.limitPerThread)}</legend>
          <LimitFields limits={limits} onChange={setLimits} />
        </fieldset>
        <fieldset disabled={busy} className="min-w-0">
          <legend className="mb-3 text-sm font-semibold">{t(mm.atOnce)}</legend>
          <div className="ai-pair">
            <Field label={t(mm.perPerson)}>
              <Input type="number" required min={1} max={10} step={1} value={perPerson || ''} onChange={(e) => setPerPerson(Number(e.target.value))} />
            </Field>
            <Field label={t(mm.perWorkspace)}>
              <Input type="number" required min={1} max={100} step={1} value={perWorkspace || ''} onChange={(e) => setPerWorkspace(Number(e.target.value))} />
            </Field>
          </div>
          <p className="k m-0 mt-2">{t(mm.perThreadFixed)}</p>
        </fieldset>
        {error ? <AINote tone="bad">{error}</AINote> : null}
      </form>
    </Dialog>
  )
}

/** Personal keys (AIModels): the switch and each person's cap, for managers to change. */
export function PersonalKeys({ s }: { s: AISettings }) {
  const { slug } = useWorkspace()
  const t = useT()
  const refresh = useRefreshAI()
  const [saving, setSaving] = useState(false)
  const [editing, setEditing] = useState(false)
  const p = s.personal
  async function toggle(enabled: boolean) {
    setSaving(true)
    try {
      await http.PUT(`${aiBase}/personal-keys`, { params: { path: { workspace: slug } }, body: { enabled, capUsd: p.capUsd } })
      await refresh()
      toast(enabled ? { kind: 'success', title: t(mm.personalOnToast) } : { kind: 'success', title: t(mm.personalOffToast), detail: t(mm.personalOffDetail) })
    } catch (e) {
      toast({ kind: 'error', title: errorMessage(e) })
    } finally {
      setSaving(false)
    }
  }
  const cap = money(p.capUsd)
  return (
    <section className="card ai-card" aria-labelledby="ai-pk-h">
      <CardHead id="ai-pk-h" title={t(mm.personalTitle)} caption={t(mm.personalCaption)}>
        <Toggle label={t(mm.personalToggle)} hideLabel checked={p.enabled} disabled={!s.canManage || saving} onChange={(e) => void toggle(e.target.checked)} />
      </CardHead>
      <div className="ai-cb">
        <div className="ai-rows ai-facts">
          <div>
            <span>{t(mm.personalCap)}</span>
            <b>{p.capSetBy ? t(mm.personalCapValueBy, { cap, name: p.capSetBy }) : t(mm.personalCapValue, { cap })}</b>
          </div>
          <div>
            <span>{t(mm.managersSee)}</span>
            <b>{t(mm.managersSeeValue)}</b>
          </div>
        </div>
        {s.canManage ? (
          <Button size="sm" className="self-start" onClick={() => setEditing(true)}>
            {t(mm.editCap)}
          </Button>
        ) : null}
      </div>
      {editing ? <PersonalCapDialog p={p} onClose={() => setEditing(false)} /> : null}
    </section>
  )
}

function PersonalCapDialog({ p, onClose }: { p: Personal; onClose: () => void }) {
  const { slug } = useWorkspace()
  const t = useT()
  const refresh = useRefreshAI()
  const [usd, setUsd] = useState(p.capUsd.toFixed(2))
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState<string>()
  async function save() {
    setBusy(true)
    setError(undefined)
    try {
      const n = Number.parseFloat(usd.replace(',', '.'))
      await http.PUT(`${aiBase}/personal-keys`, { params: { path: { workspace: slug } }, body: { enabled: p.enabled, capUsd: Number.isFinite(n) ? n : -1 } })
      await refresh()
      onClose()
      toast({ kind: 'success', title: t(m.capSavedToast) })
    } catch (e) {
      setError(errorMessage(e))
    } finally {
      setBusy(false)
    }
  }
  return (
    <Dialog
      open
      onOpenChange={(o) => !o && !busy && onClose()}
      dismissible={!busy}
      eyebrow={<span className="k">{t(mm.personalTitle)}</span>}
      title={t(mm.personalCap)}
      description={t(mm.capDialogLead)}
      size="sm"
      footer={
        <>
          <span className="k mr-auto">{t(m.changeRecorded)}</span>
          <Button onClick={onClose} disabled={busy}>
            {t(m.cancel)}
          </Button>
          <Button variant="primary" onClick={() => void save()} loading={busy}>
            {t(m.saveButton)}
          </Button>
        </>
      }
    >
      <label className="lbl">
        {t(mm.capPerPerson)}
        <span className="ai-dollar">
          <span aria-hidden="true">$</span>
          <input className="in" inputMode="decimal" value={usd} onChange={(e) => setUsd(e.target.value)} autoFocus />
        </span>
      </label>
      {error ? <AINote tone="bad">{error}</AINote> : null}
    </Dialog>
  )
}

/** People with personal keys, for managers: their keys and this month's use, with Remove for offboarding. */
export function PersonalPeople({ s }: { s: AISettings }) {
  const { slug } = useWorkspace()
  const t = useT()
  const refresh = useRefreshAI()
  const [removing, setRemoving] = useState<{ name: string; id: string; provider: string }>()
  const people = s.personal.people
  if (!s.canManage || (!people.length && !s.personal.enabled)) return null
  return (
    <section className="card ai-card" aria-labelledby="ai-pp-h">
      <CardHead id="ai-pp-h" title={t(mm.peopleTitle)} caption={t(mm.peopleCaption)} />
      {people.length ? (
        <Table fit caption={t(mm.peopleTitle)}>
          <thead>
            <tr>
              <Th>{t(mm.thPerson)}</Th>
              <Th>{t(mm.thKeys)}</Th>
              <Th align="right">{t(mm.thThisMonth)}</Th>
            </tr>
          </thead>
          <tbody>
            {people.map((person) => (
              <tr key={person.userId}>
                <Td nowrap>{person.name}</Td>
                <Td>
                  {person.accounts.length ? (
                    <span className="flex flex-col gap-1">
                      {person.accounts.map((a) => (
                        <span key={a.id} className="flex flex-wrap items-center gap-x-2 gap-y-1">
                          <span>{a.models.length ? `${a.providerName} · ${a.models.join(', ')}` : a.providerName}</span>
                          {a.status === 'connected' ? null : <AccountStatus a={{ status: a.status, models: [] }} />}
                          <Button size="sm" variant="danger" aria-label={t(mm.removeMemberLabel, { name: person.name, provider: a.providerName })} onClick={() => setRemoving({ name: person.name, id: a.id, provider: a.providerName })}>
                            {t(mm.removeBadge)}
                          </Button>
                        </span>
                      ))}
                    </span>
                  ) : (
                    <span className="k">{t(mm.noKeyNow)}</span>
                  )}
                </Td>
                <Td num align="right">
                  {t(mm.personUsage, { cost: dollars(person.costMicros), requests: person.requests })}
                </Td>
              </tr>
            ))}
          </tbody>
        </Table>
      ) : (
        <div className="ai-cb">
          <span className="k">{t(mm.nobodyPersonal)}</span>
        </div>
      )}
      {removing ? (
        <ConfirmDialog
          open
          onOpenChange={(o) => !o && setRemoving(undefined)}
          badge={t(mm.removeKey)}
          title={t(mm.removeMemberTitle, { name: removing.name, provider: removing.provider })}
          description={t(mm.removeMemberBody, { name: removing.name })}
          confirmLabel={t(mm.removeKey)}
          onConfirm={async () => {
            await http.DELETE(`${aiBase}/personal-keys/accounts/{account}`, { params: { path: { workspace: slug, account: removing.id } } })
            await refresh()
            toast({ kind: 'deleted', title: t(mm.memberKeyRemovedToast, { name: removing.name, provider: removing.provider }) })
          }}
        />
      ) : null}
    </section>
  )
}

/**
 * Your personal key (AIModels): the person's own accounts and this month's
 * use against their cap, Add and Remove, and the model they pick for their
 * own requests of each feature. Any member who can use AI features.
 */
export function YourPersonalKey({ s }: { s: AISettings }) {
  const { slug } = useWorkspace()
  const t = useT()
  const refresh = useRefreshAI()
  const canUse = useCan('ai:use')
  const [adding, setAdding] = useState(false)
  const [managing, setManaging] = useState<string>()
  const [removing, setRemoving] = useState<Account>()
  const p = s.personal
  if (!canUse && !p.accounts.length) return null
  const u = p.usage
  const reset = dayMonthDate(u.resetsAt)
  const usage = t(mm.yourUsageValue, { spent: dollars(u.costMicros), cap: money(p.capUsd) })
  const managed = p.accounts.find((a) => a.id === managing)
  return (
    <section className="card ai-card" aria-labelledby="ai-yk-h">
      <CardHead id="ai-yk-h" title={t(mm.yourTitle)}>
        <Badge tone="neutral" dot>
          {p.enabled ? t(mm.onlyYou) : t(mm.offHere)}
        </Badge>
      </CardHead>
      <div className="ai-cb">
        {!p.enabled ? <AINote tone="neutral">{t(mm.offNote)}</AINote> : null}
        {u.state === 'reached' ? <AINote tone="bad">{t(mm.yourCapReached, { date: reset })}</AINote> : null}
        {p.accounts.length ? (
          <div className="ai-rows ai-facts">
            {p.accounts.map((a) => (
              <div key={a.id} className="flex-wrap">
                <span>
                  <b>{a.models.length ? `${accountName(a)} · ${a.models.map(modelName).join(', ')}` : accountName(a)}</b>
                  <span className="text-ink-2">{p.accounts.length === 1 ? `${accountLine(t, a, true)} · ${usage}` : accountLine(t, a, true)}</span>
                </span>
                <div className="flex flex-wrap gap-1.5">
                  <Button size="sm" aria-label={t(mm.manageLabel, { account: accountName(a) })} onClick={() => setManaging(a.id)}>
                    {t(mm.manage)}
                  </Button>
                  <Button size="sm" variant="danger" aria-label={t(mm.removeKeyLabel, { provider: a.providerName })} onClick={() => setRemoving(a)}>
                    {t(mm.removeKey)}
                  </Button>
                </div>
              </div>
            ))}
            {p.accounts.length > 1 ? (
              <div>
                <span>{t(mm.yourUsage)}</span>
                <b>{usage}</b>
              </div>
            ) : null}
          </div>
        ) : p.enabled ? (
          <p className="k m-0">{t(mm.yourNone, { cap: money(p.capUsd) })}</p>
        ) : null}
        {p.canAdd ? (
          <Button id="ai-yk-add" className="self-start" icon="plus" onClick={() => setAdding(true)}>
            {t(mm.addPersonal)}
          </Button>
        ) : null}
      </div>
      {p.accounts.some((a) => a.models.length) ? <YourPicks s={s} /> : null}
      {adding ? <ConnectDialog s={s} initial="anthropic" target="personal" open onOpenChange={(o) => !o && setAdding(false)} /> : null}
      {managed ? <AccountDialog s={s} a={managed} owner={{ id: managed.id, personal: true }} onClose={() => setManaging(undefined)} /> : null}
      {removing ? (
        <ConfirmDialog
          open
          onOpenChange={(o) => !o && setRemoving(undefined)}
          badge={t(mm.removeKey)}
          title={t(mm.removeYourTitle, { provider: removing.providerName })}
          description={t(mm.removeYourBody)}
          confirmLabel={t(mm.removeKey)}
          onConfirm={async () => {
            await http.DELETE(`${aiBase}/personal/accounts/{account}`, { params: { path: { workspace: slug, account: removing.id } } })
            await refresh()
            focusAdd(true)
            toast({ kind: 'deleted', title: t(mm.yourKeyRemovedToast, { provider: removing.providerName }) })
          }}
        />
      ) : null}
    </section>
  )
}

/** The person's own model for each feature, or the workspace's. */
function YourPicks({ s }: { s: AISettings }) {
  const { slug } = useWorkspace()
  const t = useT()
  const refresh = useRefreshAI()
  const [busy, setBusy] = useState<ModelFeature>()
  const [refused, setRefused] = useState<{ feature: ModelFeature; message: string }>()
  const p = s.personal
  const own = p.accounts.flatMap((a) => a.models)
  const workspaceModels = s.accounts.flatMap((a) => a.models)
  async function pick(f: ModelFeature, modelId: string) {
    setBusy(f)
    setRefused(undefined)
    const path = { workspace: slug, feature: f }
    try {
      if (modelId) await http.PUT(`${aiBase}/personal/features/{feature}/model`, { params: { path }, body: { modelId } })
      else await http.DELETE(`${aiBase}/personal/features/{feature}/model`, { params: { path } })
      await refresh()
      const x = own.find((y) => y.id === modelId)
      toast({ kind: 'success', title: x ? t(mm.pickedOwnToast, { feature: featureLabel(t, f), model: modelName(x) }) : t(mm.pickedWorkspaceToast, { feature: featureLabel(t, f) }) })
    } catch (e) {
      setRefused({ feature: f, message: errorMessage(e) })
      void refresh()
    } finally {
      setBusy(undefined)
    }
  }
  return (
    <>
      <div className="ai-ch border-t border-line-2">
        <div>
          <h3 className="h2 m-0" id="ai-yp-h">
            {t(mm.yourPicksTitle)}
          </h3>
          <span className="k">{t(mm.yourPicksCaption)}</span>
        </div>
      </div>
      <Table fit caption={t(mm.yourPicksTitle)}>
        <thead>
          <tr>
            <Th>{t(mm.thFeature)}</Th>
            <Th>{t(mm.thModel)}</Th>
          </tr>
        </thead>
        <tbody>
          {MODEL_FEATURES.map((f) => {
            const id = `ai-yp-${f}`
            const ws = workspaceModels.find((x) => x.id === s.defaults.find((d) => d.feature === f)?.modelId)
            return (
              <tr key={f}>
                <Td>
                  <label htmlFor={id}>{featureLabel(t, f)}</label>
                </Td>
                <Td>
                  <ModelSelect
                    id={id}
                    feature={f}
                    accounts={p.accounts}
                    value={p.choices.find((c) => c.feature === f)?.modelId ?? ''}
                    disabled={busy === f}
                    first={{ value: '', label: ws ? t(mm.workspaceModel, { model: modelName(ws) }) : t(mm.workspaceModelNone) }}
                    onChange={(v) => void pick(f, v)}
                  />
                </Td>
              </tr>
            )
          })}
        </tbody>
      </Table>
      {refused ? (
        <div className="ai-cb">
          <AINote tone="bad">
            <b>{t(mm.pickRefused, { feature: featureLabel(t, refused.feature) })}</b> {refused.message}
          </AINote>
        </div>
      ) : null}
    </>
  )
}
