import { keepPreviousData, useInfiniteQuery } from '@tanstack/react-query'
import { Link, useNavigate } from '@tanstack/react-router'
import { useState, type ReactNode } from 'react'
import { ApiError, errorMessage, http, type Schemas } from '@/api/client'
import { Icon } from '@/components/icons/Icon'
import { Page } from '@/components/shell/Page'
import { Avatar } from '@/components/ui/Avatar'
import { Badge } from '@/components/ui/Badge'
import { Button, ButtonLink } from '@/components/ui/Button'
import { Skeleton } from '@/components/ui/Feedback'
import { Select, Toggle } from '@/components/ui/Field'

import { Table, Td, Th } from '@/components/ui/Table'
import { toast } from '@/components/ui/Toast'
import { T, formatList, useT } from '@/i18n'
import { clock, count, dayMonthDate, money, shortDate, weekdayDate } from '@/lib/format'
import { deniedReason, useMe, useWorkspace } from '@/lib/session'
import '@/features/integrations/integrations.css'
import { aiBase, AI_FEATURES, referenceOf, useAISettings, useRefreshAI, type AIConnection, type AIFeature, type AISettings, type AITest } from './api'
import { AIModelCard, AITile } from './catalog'
import { CapDialog, ChangeModelDialog, ConnectDialog, DisconnectDialog, PricesDialog, ReplaceKeyDialog, TestPanel } from './ConnectDialog'
import { AI_PROVIDERS, type AIProvider } from './entries'
import { calendarDay, dollars, seconds, tokens } from './format'
import { m } from './messages'
import { AINote, EyeOffIcon, featureName, featureSends, LeavesNote, OutcomeBadge, rowFeature, tags } from './parts'
import './ai.css'

type Dialog = { kind: 'connect'; provider: AIProvider } | { kind: 'change' | 'key' | 'cap' | 'prices' | 'disconnect' } | null
type TestState = { running: boolean; result?: AITest; error?: string }
type ActivityList = Schemas['AIActivityList']
type Outcome = 'accepted' | 'kept' | 'rejected' | 'refused' | 'error'

/** The log's first page is the latest few requests (AISettings shows six); Show more adds 25. */
const FIRST = 6
const MORE = 25

/** Settings › Integrations › AI assistant (AISettings, AIStates). */
export function AISettingsPage({ connect }: { connect?: AIProvider }) {
  const { slug } = useWorkspace()
  const t = useT()
  const navigate = useNavigate()
  const q = useAISettings()
  const s = q.data
  const [dialog, setDialog] = useState<Dialog>(connect ? { kind: 'connect', provider: connect } : null)
  const [test, setTest] = useState<TestState | null>(null)
  const refresh = useRefreshAI()

  const close = () => {
    setDialog(null)
    if (connect) void navigate({ to: '/w/$workspace/settings/integrations/ai', params: { workspace: slug }, search: {}, replace: true })
  }
  const onOpenChange = (o: boolean) => !o && close()

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

  const c = s?.connection
  const live = s?.connected && c ? c : undefined
  return (
    <Page
      title={t(m.title)}
      documentTitle={t(m.documentTitle)}
      breadcrumb={[
        { label: t(m.crumbSettings), to: '/w/$workspace/settings', params: { workspace: slug } },
        { label: t(m.crumbIntegrations), to: '/w/$workspace/settings/integrations', params: { workspace: slug } },
        { label: t(m.title) },
      ]}
    >
      {q.isError ? (
        <LoadError error={q.error} onRetry={() => void q.refetch()} />
      ) : !s ? (
        <SettingsSkeleton />
      ) : (
        <>
          <Header s={s} onDialog={(kind) => setDialog({ kind })} onTest={runTest} testing={test?.running ?? false} />
          {test && live ? (
            <TestPanel provider={live.providerName} running={test.running} result={test.result} error={test.error} onClose={() => setTest(null)} onPasteKey={s.canManage ? () => setDialog({ kind: 'key' }) : undefined} />
          ) : null}
          {!s.canManage ? <ReadOnlyNote s={s} /> : null}
          {live ? <Connected s={s} c={live} onDialog={(kind) => setDialog({ kind })} /> : <NotConnected s={s} onConnect={(provider) => setDialog({ kind: 'connect', provider })} />}
          {live ? <ActivityLog /> : null}
          {live && s.canManage ? (
            <section className="card ai-dz" aria-labelledby="ai-dz-h">
              <div>
                <h2 className="h2 m-0" id="ai-dz-h">
                  {t(m.disconnectTitle, { provider: live.providerName })}
                </h2>
                <span className="k">{t(m.disconnectLine)}</span>
              </div>
              <Button variant="danger" onClick={() => setDialog({ kind: 'disconnect' })}>
                {t(m.disconnectButton)}
              </Button>
            </section>
          ) : null}

          {dialog?.kind === 'connect' && s.canManage && !s.connected ? <ConnectDialog s={s} initial={dialog.provider} open onOpenChange={onOpenChange} /> : null}
          {live && s.canManage ? (
            <>
              {dialog?.kind === 'change' ? <ChangeModelDialog s={s} c={live} open onOpenChange={onOpenChange} /> : null}
              {dialog?.kind === 'key' ? (
                <ReplaceKeyDialog
                  c={live}
                  open
                  onOpenChange={onOpenChange}
                  onTested={(r) => {
                    setTest({ running: false, result: r })
                  }}
                />
              ) : null}
              {dialog?.kind === 'cap' ? <CapDialog c={live} open onOpenChange={onOpenChange} /> : null}
              {dialog?.kind === 'prices' ? <PricesDialog c={live} open onOpenChange={onOpenChange} /> : null}
              {dialog?.kind === 'disconnect' ? <DisconnectDialog c={live} open onOpenChange={onOpenChange} /> : null}
            </>
          ) : null}
        </>
      )}
    </Page>
  )
}

/** Tile, title, status, the connection's facts and the actions. */
function Header({ s, onDialog, onTest, testing }: { s: AISettings; onDialog: (k: 'change' | 'key') => void; onTest: () => void; testing: boolean }) {
  const { slug } = useWorkspace()
  const t = useT()
  const c = s.connected ? s.connection : undefined
  const helpers = formatList(s.helpers, 'disjunction')
  if (!c) {
    return (
      <section className="card ai-head" aria-labelledby="ai-h">
        <div className="ai-head-main">
          <span className="ih-tile lg" aria-hidden="true">
            {t(m.tileAI)}
          </span>
          <div className="ai-head-text">
            <div className="row gap-2.5">
              <h2 className="ih-ttl" id="ai-h">
                {t(m.title)}
              </h2>
              <Badge tone="neutral">{t(m.statusNotConnected)}</Badge>
            </div>
            <span className="ai-head-sub">{s.helpers.length ? t(m.notConnectedLine, { helpers }) : t(m.notConnectedLineNobody)}</span>
            <p>{t(m.lead)}</p>
          </div>
        </div>
      </section>
    )
  }
  if (c.status === 'key_rejected') {
    const at = c.statusAt ?? c.connectedAt
    const today = dayMonthDate(at) === dayMonthDate(new Date())
    return (
      <section className="card ai-head ai-bad" aria-labelledby="ai-h">
        <div className="ai-head-main">
          <AITile provider={c.provider} size="lg" />
          <div className="ai-head-text">
            <div className="row flex-wrap gap-2.5">
              <h2 className="ih-ttl" id="ai-h">
                {t(m.title)}
              </h2>
              <Badge tone="bad" dot>
                {t(m.statusKeyRejected)}
              </Badge>
            </div>
            <span className="ai-head-sub">{`${c.providerName} · ${c.modelName}`}</span>
          </div>
        </div>
        <AINote tone="bad">
          <T
            msg={m.keyRejectedBody}
            values={{ ...tags, provider: c.providerName, when: today ? t(m.keyRejectedToday, { time: clock(at) }) : t(m.keyRejectedOn, { date: dayMonthDate(at) }), detail: c.statusDetail ?? '' }}
          />
        </AINote>
        {s.canManage ? (
          <div className="row flex-wrap gap-2">
            <Button variant="primary" onClick={() => onDialog('key')}>
              {t(m.replaceKey)}
            </Button>
            <Button icon="refresh" onClick={onTest} loading={testing}>
              {t(m.testConnection)}
            </Button>
          </div>
        ) : null}
      </section>
    )
  }
  const sub = c.connectedBy
    ? t(m.connectedLine, { provider: c.providerName, model: c.modelName, name: c.connectedBy, date: weekdayDate(c.connectedAt) })
    : t(m.connectedLineNoBy, { provider: c.providerName, model: c.modelName, date: weekdayDate(c.connectedAt) })
  return (
    <section className="card ai-head" aria-labelledby="ai-h">
      <div className="ai-head-row">
        <div className="ai-head-main">
          <AITile provider={c.provider} size="lg" />
          <div className="ai-head-text">
            <div className="row flex-wrap gap-2.5">
              <h2 className="ih-ttl" id="ai-h">
                {t(m.title)}
              </h2>
              <Badge tone="ok" dot>
                {t(m.statusConnected)}
              </Badge>
            </div>
            <span className="ai-head-sub">{sub}</span>
            <p>{t(m.lead)}</p>
          </div>
        </div>
        <div className="ai-kvp">{s.canManage ? <ManagerFacts c={c} onReplace={() => onDialog('key')} /> : <ReaderFacts c={c} />}</div>
      </div>
      {s.canManage ? (
        <div className="row flex-wrap gap-2">
          <Button icon="refresh" onClick={onTest} loading={testing}>
            {t(m.testConnection)}
          </Button>
          <Button onClick={() => onDialog('change')}>{t(m.changeModel)}</Button>
          <ButtonLink to="/w/$workspace/settings/integrations" params={{ workspace: slug }}>
            {t(m.allIntegrations)}
          </ButtonLink>
        </div>
      ) : null}
    </section>
  )
}

function lastTest(t: ReturnType<typeof useT>, c: AIConnection, reader: boolean): string {
  const lt = c.lastTest
  if (!lt) return t(m.lastTestNever)
  const values = { date: shortDate(lt.at), time: clock(lt.at), seconds: seconds(lt.ms) }
  if (!lt.ok) return t(m.lastTestFailed, values)
  return reader || !lt.ms ? t(m.lastTestPassed, values) : t(m.lastTestAnswered, values)
}

/** Model, API key, endpoint, last test (people with integrations:manage). */
function ManagerFacts({ c, onReplace }: { c: AIConnection; onReplace: () => void }) {
  const t = useT()
  const compatible = c.provider === 'compatible'
  const azure = c.provider === 'azure'
  return (
    <dl className="kv">
      <dt>{t(m.kvModel)}</dt>
      <dd>
        {c.modelName && c.modelName !== c.model ? (
          <>
            {c.modelName} · <span className="mono text-[12.5px]">{c.model}</span>
          </>
        ) : (
          <span className="mono text-[12.5px]">{c.model}</span>
        )}
      </dd>
      {azure ? (
        <>
          <dt>{t(m.kvDeployment)}</dt>
          <dd className="mono text-[12.5px]">{c.deployment}</dd>
        </>
      ) : null}
      <dt>{t(m.kvKey)}</dt>
      <dd>
        {c.key.set ? (
          <span className="inline-flex flex-wrap items-center gap-2">
            <span className="ai-mask">
              <EyeOffIcon />
              <span aria-label={t(m.keyHidden)}>••••••</span>
              <Badge tone="neutral" className="h-5 font-mono text-[11px]">
                {t(m.writeOnly)}
              </Badge>
            </span>
            {c.key.addedAt ? (
              <span className="k">{c.key.addedBy ? t(m.keyAdded, { date: dayMonthDate(c.key.addedAt), name: c.key.addedBy }) : t(m.keyAddedNoBy, { date: dayMonthDate(c.key.addedAt) })}</span>
            ) : null}
            <Button size="sm" onClick={onReplace}>
              {t(m.replace)}
            </Button>
          </span>
        ) : (
          <span className="inline-flex flex-wrap items-center gap-2">
            <span className="k">{t(m.keyNone)}</span>
            {compatible ? (
              <Button size="sm" onClick={onReplace}>
                {t(m.addKey)}
              </Button>
            ) : null}
          </span>
        )}
      </dd>
      <dt>{compatible ? t(m.kvServer) : t(m.kvEndpoint)}</dt>
      <dd>
        {c.baseUrl ? (
          <span className="mono text-[12.5px]">{c.baseUrl}</span>
        ) : (
          <T msg={m.endpointDefault} values={{ url: <span className="mono text-[12.5px]">{defaultEndpoint(c.provider)}</span>, provider: c.providerName }} />
        )}
      </dd>
      <dt>{t(m.kvLastTest)}</dt>
      <dd>{lastTest(t, c, false)}</dd>
    </dl>
  )
}

/** Model, cap, last test (read-only). */
function ReaderFacts({ c }: { c: AIConnection }) {
  const t = useT()
  return (
    <dl className="kv">
      <dt>{t(m.kvModel)}</dt>
      <dd>{c.modelName || c.model}</dd>
      <dt>{t(m.kvCap)}</dt>
      <dd>{capShort(t, c)}</dd>
      <dt>{t(m.kvLastTest)}</dt>
      <dd>{lastTest(t, c, true)}</dd>
    </dl>
  )
}

function defaultEndpoint(p: string): string {
  switch (p) {
    case 'anthropic':
      return 'https://api.anthropic.com'
    case 'openai':
      return 'https://api.openai.com'
    case 'google':
      return 'https://generativelanguage.googleapis.com'
  }
  return ''
}

function capAmount(c: AIConnection): string {
  return c.cap.kind === 'tokens' ? count(c.cap.tokens) : money(c.cap.usd)
}

function capShort(t: ReturnType<typeof useT>, c: AIConnection): string {
  if (c.cap.kind === 'none') return t(m.capShortNone)
  const cap = c.cap.kind === 'tokens' ? t(m.capTokens, { amount: tokens(c.cap.tokens) }) : money(c.cap.usd)
  return c.cap.setBy ? t(m.capShort, { cap, name: c.cap.setBy }) : cap
}

/** "Read-only · Sofia Rossi, Editor." */
function ReadOnlyNote({ s }: { s: AISettings }) {
  const { me } = useWorkspace()
  const user = useMe().data?.user
  const values = { ...tags, name: user?.name ?? '', role: me.workspace.roleName, helpers: formatList(s.helpers, 'disjunction') }
  return (
    <AINote tone="neutral">
      <T msg={s.helpers.length ? m.readOnly : m.readOnlyAlone} values={values} />
    </AINote>
  )
}

/** Not connected: the invitation and the five providers (AISettings · not connected). */
function NotConnected({ s, onConnect }: { s: AISettings; onConnect: (p: AIProvider) => void }) {
  const t = useT()
  const denied = s.canManage ? undefined : deniedReason('integrations:manage', s.helpers)
  return (
    <>
      <AINote tone="info">{t(m.notConnectedNote)}</AINote>
      <div className="ih-grid">
        {AI_PROVIDERS.map((p) => (
          <AIModelCard key={p} provider={p} status="not_connected" canManage={s.canManage} own onConnect={onConnect} denied={denied} />
        ))}
      </div>
    </>
  )
}

/** Usage, by feature, by member, the features, for a connected model. */
function Connected({ s, c, onDialog }: { s: AISettings; c: AIConnection; onDialog: (k: 'cap' | 'prices') => void }) {
  const usage = <UsageCard s={s} c={c} onDialog={onDialog} />
  const byFeature = <ByFeature s={s} c={c} />
  const byMember = <ByMember s={s} c={c} />
  const features = <Features s={s} />
  if (!s.canManage) {
    return (
      <div className="ai-two">
        <div className="ai-col">
          {usage}
          {byFeature}
          {byMember}
        </div>
        <div className="ai-col">{features}</div>
      </div>
    )
  }
  return (
    <>
      <div className="ai-grid3">
        {usage}
        {byFeature}
        {byMember}
      </div>
      {features}
    </>
  )
}

function CardHead({ id, title, caption, children }: { id: string; title: string; caption?: ReactNode; children?: ReactNode }) {
  return (
    <div className="ai-ch">
      <div>
        <h2 className="h2" id={id}>
          {title}
        </h2>
        {caption ? <span className="k">{caption}</span> : null}
      </div>
      {children}
    </div>
  )
}

/** This month against the cap: the bar, the 80% notice, the cap, prices and tokens. */
export function UsageCard({ s, c, onDialog, caption }: { s: AISettings; c: AIConnection; onDialog?: (k: 'cap' | 'prices') => void; caption?: string }) {
  const t = useT()
  const u = s.usage
  const kind = c.cap.kind
  const reset = dayMonthDate(u.resetsAt)
  const big = kind === 'tokens' ? tokens(u.tokens) : dollars(u.costMicros)
  const of =
    kind === 'tokens'
      ? t(m.usageOfTokens, { cap: tokens(c.cap.tokens), date: reset })
      : kind === 'none'
        ? t(m.usageNoCap, { date: reset })
        : t(m.usageOf, { cap: money(c.cap.usd), date: reset })
  const share = Math.min(1, Math.max(0, u.share))
  const notice = kind === 'tokens' ? tokens(Math.round(c.cap.tokens * 0.8)) : money(c.cap.usd * 0.8)
  const manage = s.canManage && onDialog
  return (
    <section className="card ai-card" aria-labelledby="ai-usage-h">
      <CardHead id="ai-usage-h" title={t(m.usageTitle)} caption={caption ?? t(m.usageCaption)} />
      <div className="ai-cb">
        <div className="flex min-w-0 flex-col gap-2">
          <div className="flex flex-wrap items-baseline gap-2">
            <span className="ai-big">{big}</span>
            <span className="k text-[13px]">{of}</span>
          </div>
          {kind === 'none' ? null : (
            <>
              <div
                role="meter"
                aria-label={t(m.usageBarLabel, { spent: big, cap: capAmount(c) })}
                aria-valuemin={0}
                aria-valuemax={100}
                aria-valuenow={Math.round(share * 100)}
                className={`ai-track${u.state === 'ok' ? '' : ` ai-${u.state}`}`}
              >
                <span className="ai-fill" style={{ width: `${share * 100}%` }} />
                <span className="ai-mark" aria-hidden="true" />
              </div>
              <div className="ai-scale k" aria-hidden="true">
                <span style={{ left: 0 }}>{kind === 'tokens' ? '0' : money(0, 0)}</span>
                <span style={{ right: '20%', paddingRight: 6 }}>{t(m.noticeAt, { amount: notice })}</span>
                <span style={{ right: 0 }}>{kind === 'tokens' ? tokens(c.cap.tokens) : money(c.cap.usd)}</span>
              </div>
            </>
          )}
        </div>
        {u.state === 'notice' || u.state === 'reached' ? (
          <AINote tone={u.state === 'reached' ? 'bad' : 'warn'}>
            <div className="flex flex-col items-start gap-2">
              <span>{u.state === 'reached' ? t(m.noticeReached, { date: reset }) : t(m.noticeEighty, { cap: capAmount(c), date: reset })}</span>
              {manage ? (
                <Button size="sm" onClick={() => onDialog('cap')}>
                  {t(m.raiseCap)}
                </Button>
              ) : null}
            </div>
          </AINote>
        ) : null}
        {onDialog ? (
          <>
            <div className="ai-rows">
              <div>
                <span>
                  <span className="k">{t(m.monthlyCap)}</span>
                  <span className="text-[13px]">
                    <T msg={capRowMsg(c)} values={{ ...tags, cap: c.cap.kind === 'tokens' ? count(c.cap.tokens) : money(c.cap.usd), name: c.cap.setBy ?? '' }} />
                  </span>
                </span>
                {manage ? (
                  <Button size="sm" className="flex-none" onClick={() => onDialog('cap')} aria-label={`${t(m.edit)} · ${t(m.monthlyCap)}`}>
                    {t(m.edit)}
                  </Button>
                ) : null}
              </div>
              <div>
                <span>
                  <span className="k">{t(m.pricesTitle)}</span>
                  <span className="text-[13px]">
                    <T msg={m.pricesValue} values={{ ...tags, input: money(c.prices.input), output: money(c.prices.output) }} />
                  </span>
                </span>
                {manage ? (
                  <Button size="sm" className="flex-none" onClick={() => onDialog('prices')} aria-label={`${t(m.edit)} · ${t(m.pricesTitle)}`}>
                    {t(m.edit)}
                  </Button>
                ) : null}
              </div>
              <div>
                <span>
                  <span className="k">{t(m.tokensTitle)}</span>
                  <span className="text-[13px]">
                    <T msg={m.tokensValue} values={{ ...tags, input: tokens(u.inputTokens), output: tokens(u.outputTokens) }} />
                  </span>
                </span>
              </div>
            </div>
            <span className="k leading-normal">{c.prices.from ? t(m.pricesNote, { date: calendarDay(c.prices.from) }) : t(m.pricesNoteTyped)}</span>
          </>
        ) : null}
      </div>
    </section>
  )
}

function capRowMsg(c: AIConnection) {
  if (c.cap.kind === 'none') return m.capRowNone
  if (c.cap.kind === 'tokens') return m.capRowTokens
  return c.cap.setBy ? m.capRow : m.capRowNoBy
}

function ShareRows({ rows }: { rows: { key: string; label: string; value: string; amount: number }[] }) {
  const max = Math.max(0, ...rows.map((r) => r.amount))
  return (
    <div className="ai-share">
      {rows.map((r) => (
        <div key={r.key}>
          <div>
            <span>{r.label}</span>
            <span className="num">{r.value}</span>
          </div>
          <div className="bar" aria-hidden="true">
            <i style={{ width: `${max > 0 ? (r.amount / max) * 100 : 0}%` }} />
          </div>
        </div>
      ))}
    </div>
  )
}

function ByFeature({ s, c }: { s: AISettings; c: AIConnection }) {
  const t = useT()
  const byTokens = c.cap.kind === 'tokens'
  const rows = AI_FEATURES.map((f) => {
    const u = s.byFeature.find((x) => x.feature === f)
    const amount = byTokens ? (u?.tokens ?? 0) : (u?.costMicros ?? 0)
    return { key: f, label: t(featureName[f]), amount, value: byTokens ? tokens(amount) : dollars(amount) }
  })
  return (
    <section className="card ai-card" aria-labelledby="ai-feat-h">
      <CardHead id="ai-feat-h" title={t(m.byFeature)} caption={byTokens ? t(m.byFeatureCaptionTokens, { amount: tokens(s.usage.tokens) }) : t(m.byFeatureCaption, { amount: dollars(s.usage.costMicros) })} />
      <div className="ai-cb">
        <ShareRows rows={rows} />
      </div>
    </section>
  )
}

function ByMember({ s, c }: { s: AISettings; c: AIConnection }) {
  const t = useT()
  const byTokens = c.cap.kind === 'tokens'
  const rows = s.byMember.map((u, i) => {
    const amount = byTokens ? u.tokens : u.costMicros
    return { key: u.userId || `${u.name}-${i}`, label: u.name, amount, value: byTokens ? tokens(amount) : dollars(amount) }
  })
  return (
    <section className="card ai-card" aria-labelledby="ai-mem-h">
      <CardHead id="ai-mem-h" title={t(m.byMember)} caption={t(m.byMemberCaption)} />
      <div className="ai-cb">{rows.length ? <ShareRows rows={rows} /> : <span className="k">{t(m.nobodyYet)}</span>}</div>
      <div className="ai-foot">
        <span className="k leading-normal">
          {s.aiUseRoles.length ? `${t(m.aiUseRoles, { roles: formatList(s.aiUseRoles) })} ` : ''}
          {s.withoutAiUse.slice(0, 3).map((w) => `${t(m.aiUseWithout, { name: w.name, role: w.role })} `)}
          {t(m.aiUseAgents)}
        </span>
      </div>
    </section>
  )
}

/** Each feature, what it sends, and its switch (On/Off for readers). */
function Features({ s }: { s: AISettings }) {
  const { slug } = useWorkspace()
  const t = useT()
  const refresh = useRefreshAI()
  const [busy, setBusy] = useState<AIFeature>()
  async function toggle(f: AIFeature, on: boolean) {
    setBusy(f)
    try {
      await http.PUT(`${aiBase}/features/{feature}`, { params: { path: { workspace: slug, feature: f } }, body: { on } })
      await refresh()
      toast({ kind: 'success', title: t(on ? m.featureOnToast : m.featureOffToast, { feature: t(featureName[f]) }), detail: t(m.featureToastDetail) })
    } catch (e) {
      toast({ kind: 'error', title: errorMessage(e) })
    } finally {
      setBusy(undefined)
    }
  }
  return (
    <section className="card ai-card" aria-labelledby="ai-features-h">
      <CardHead id="ai-features-h" title={t(m.featuresTitle)} caption={s.canManage ? t(m.featuresCaption) : t(m.featuresCaptionReadOnly)} />
      <div>
        {AI_FEATURES.map((f) => {
          const st = s.features.find((x) => x.key === f)
          const available = st?.available ?? true
          const on = Boolean(st?.on) && available
          const name = t(featureName[f])
          return (
            <div className="ai-frow" key={f}>
              <div className="ai-fl">
                <span>{name}</span>
                <span className="k">
                  <T msg={featureSends[f]} values={tags} />
                </span>
                {!available ? (
                  <span className="ai-warn">
                    <Icon name="alertCircle" size={14} />
                    {t(m.altNeedsImages)}
                  </span>
                ) : null}
              </div>
              <div className="flex-none pt-0.5">
                {s.canManage ? (
                  <Toggle label={name} hideLabel checked={on} disabled={!available || busy === f} onChange={(e) => void toggle(f, e.target.checked)} />
                ) : (
                  <Badge tone={on ? 'ok' : 'neutral'}>{on ? t(m.on) : t(m.off)}</Badge>
                )}
              </div>
            </div>
          )
        })}
      </div>
      <div className="ai-leaves">
        <LeavesNote />
      </div>
    </section>
  )
}

/** The AI activity log: who asked, for what, the tokens and the outcome. Never the text. */
function ActivityLog() {
  const { slug } = useWorkspace()
  const t = useT()
  const [member, setMember] = useState('')
  const [feature, setFeature] = useState<AIFeature | ''>('')
  const [outcome, setOutcome] = useState<Outcome | ''>('')
  const query = { member: member || undefined, feature: feature || undefined, outcome: outcome || undefined }
  const q = useInfiniteQuery({
    queryKey: ['get', `${aiBase}/activity`, slug, query],
    queryFn: async ({ pageParam }) => (await http.GET(`${aiBase}/activity`, { params: { path: { workspace: slug }, query: { ...query, cursor: pageParam || undefined, limit: pageParam ? MORE : FIRST } } })).data!,
    initialPageParam: '',
    getNextPageParam: (last: ActivityList) => last.next || undefined,
    placeholderData: keepPreviousData,
  })
  const first = q.data?.pages[0]
  const items = q.data?.pages.flatMap((p) => p.items) ?? []
  const filtered = Boolean(member || feature || outcome)
  const audit = (c: ReactNode[]) => (
    <Link to="/w/$workspace/settings/audit" params={{ workspace: slug }} search={{ category: 'ai' }}>
      {c}
    </Link>
  )
  return (
    <section className="card ai-card" aria-labelledby="ai-log-h">
      <CardHead id="ai-log-h" title={t(m.logTitle)} caption={t(m.logCaption)} />
      <div className="ai-filters">
        <div>
          <Select aria-label={t(m.filterMember)} value={member} onChange={(e) => setMember(e.target.value)}>
            <option value="">{t(m.allMembers)}</option>
            {(first?.members ?? []).map((mb) => (
              <option key={mb.id} value={mb.id}>
                {mb.name}
              </option>
            ))}
          </Select>
          <Select aria-label={t(m.filterFeature)} value={feature} onChange={(e) => setFeature(e.target.value as AIFeature | '')}>
            <option value="">{t(m.allFeatures)}</option>
            {AI_FEATURES.map((f) => (
              <option key={f} value={f}>
                {t(featureName[f])}
              </option>
            ))}
          </Select>
          <Select aria-label={t(m.filterOutcome)} value={outcome} onChange={(e) => setOutcome(e.target.value as Outcome | '')}>
            <option value="">{t(m.anyOutcome)}</option>
            <option value="accepted">{t(m.outcomeAccepted)}</option>
            <option value="kept">{t(m.outcomeKept)}</option>
            <option value="rejected">{t(m.outcomeRejected)}</option>
            <option value="refused">{t(m.outcomeRefused)}</option>
            <option value="error">{t(m.outcomeError)}</option>
          </Select>
        </div>
        {first ? <span className="k">{t(m.logCount, { shown: count(items.length), count: first.monthRequests })}</span> : null}
      </div>
      {q.isError ? (
        <div className="ai-cb">
          <AINote tone="bad">{errorMessage(q.error)}</AINote>
        </div>
      ) : !first ? (
        <div className="ai-cb" aria-busy="true">
          <Skeleton w="100%" h={18} />
          <Skeleton w="100%" h={18} />
          <Skeleton w="100%" h={18} />
        </div>
      ) : !items.length ? (
        <div className="ai-cb">
          <span className="k">{filtered ? t(m.logEmptyFiltered) : t(m.logEmpty)}</span>
        </div>
      ) : (
        <>
          <div className="ih-no-narrow">
            <Table aria-labelledby="ai-log-h">
              <thead>
                <tr>
                  <Th>{t(m.thTime)}</Th>
                  <Th>{t(m.thMember)}</Th>
                  <Th>{t(m.thFeature)}</Th>
                  <Th>{t(m.thOn)}</Th>
                  <Th align="right">{t(m.thTokens)}</Th>
                  <Th>{t(m.thOutcome)}</Th>
                </tr>
              </thead>
              <tbody>
                {items.map((r) => (
                  <tr key={r.id}>
                    <Td nowrap>{`${shortDate(r.at)} ${clock(r.at)}`}</Td>
                    <Td nowrap>
                      <span className="ai-who">
                        <Avatar name={r.member} color={r.memberColor || undefined} size={22} fontSize={10} />
                        {r.member}
                      </span>
                    </Td>
                    <Td>{rowFeature(t, r)}</Td>
                    <Td>{r.on}</Td>
                    <Td num align="right">
                      {count(r.tokens)}
                    </Td>
                    <Td nowrap>
                      <OutcomeBadge outcome={r.outcome} detail={r.detail} />
                    </Td>
                  </tr>
                ))}
              </tbody>
            </Table>
          </div>
          <div className="ih-only-narrow">
            <div className="ai-loglist">
            {items.map((r) => (
              <div key={r.id}>
                <div className="ai-lt">
                  <span className="ai-who font-semibold">
                    <Avatar name={r.member} color={r.memberColor || undefined} size={22} fontSize={10} />
                    {r.member}
                  </span>
                  <OutcomeBadge outcome={r.outcome} detail={r.detail} />
                </div>
                <span className="text-[13px]">{r.on ? `${rowFeature(t, r)} · ${r.on}` : rowFeature(t, r)}</span>
                <span className="k">{`${shortDate(r.at)} ${clock(r.at)} · ${t(m.tokensCount, { count: r.tokens })}`}</span>
              </div>
            ))}
            </div>
          </div>
          {q.hasNextPage ? (
            <div className="ai-foot justify-center">
              <Button size="sm" onClick={() => void q.fetchNextPage()} loading={q.isFetchingNextPage}>
                {t(m.showMore)}
              </Button>
            </div>
          ) : null}
        </>
      )}
      <div className="ai-foot">
        <span className="k">
          <T msg={m.logFoot} values={{ a: audit }} />
        </span>
      </div>
    </section>
  )
}

function SettingsSkeleton() {
  const t = useT()
  return (
    <div className="ai-two" aria-busy="true">
      <span className="vh" role="status">
        {t(m.loading)}
      </span>
      <section className="card ai-cb">
        <div className="ai-grid3">
          {[0, 1, 2].map((i) => (
            <div key={i} className="card flex flex-col gap-2 p-4">
              <Skeleton w="50%" h={12} />
              <Skeleton w="40%" h={20} />
              <Skeleton w="70%" h={10} />
            </div>
          ))}
        </div>
        {[0, 1, 2].map((i) => (
          <div key={i} className="flex items-center gap-3 border-t border-line-2 pt-3">
            <Skeleton w="30%" h={16} />
            <Skeleton w="15%" h={16} />
            <Skeleton w="30%" h={12} />
            <Skeleton w="10%" h={28} />
          </div>
        ))}
      </section>
    </div>
  )
}

function LoadError({ error, onRetry }: { error: unknown; onRetry: () => void }) {
  const t = useT()
  const status = error instanceof ApiError ? error.status : 0
  const id = referenceOf(error)
  return (
    <section className="card flex flex-col items-center gap-2 px-5 py-6 text-center" role="alert">
      <span className="text-bad-strong">
        <Icon name="alertCircle" size={24} />
      </span>
      <h2 className="h2 m-0">{t(m.loadError)}</h2>
      <span className="k">{status ? t(m.loadErrorBody, { status }) : t(m.loadErrorPlain, { detail: errorMessage(error) })}</span>
      <Button variant="primary" onClick={onRetry}>
        {t(m.retry)}
      </Button>
      {id ? <span className="mono text-[12px] text-ink-3">{t(m.requestRef, { id })}</span> : null}
    </section>
  )
}
