import { useCallback, useEffect, useMemo, useRef, useState, type ReactNode } from 'react'
import { errorMessage, http, type Schemas } from '@/api/client'
import { Icon } from '@/components/icons/Icon'
import { Button, ButtonLink } from '@/components/ui/Button'
import { T, formatList, useT, type MessageDescriptor } from '@/i18n'
import { money } from '@/lib/format'
import { useCan, useWorkspace } from '@/lib/session'
import { aiBase, problemCode, problemMeta, useAIStatus, type AIFeature, type AIStatus } from './api'
import { costLine, dollars, tokens } from './format'
import { m, ms } from './messages'

import { AINote, tags } from './parts'
import './ai.css'

export type AISent = Schemas['AISent']
type Outcome = Schemas['AIOutcomeInput']['outcome']

/** The sparkle every AI entry point carries. */
export function Sparkle({ size = 14, color = 'var(--accent)' }: { size?: number; color?: string }) {
  return (
    <svg width={size} height={size} viewBox="0 0 16 16" fill="none" stroke={color} strokeWidth="1.6" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true" className="flex-none">
      <path d="M8 1.8 9.4 6.6 14.2 8 9.4 9.4 8 14.2 6.6 9.4 1.8 8 6.6 6.6z" />
    </svg>
  )
}

/**
 * The assistant as the signed-in person sees it: whether they hold ai:use,
 * the model, and whether a feature can be asked for now.
 */
export function useAssist() {
  const can = useCan('ai:use')
  const q = useAIStatus()
  const status = can ? q.data : undefined
  const loading = can && q.isPending
  const refetch = q.refetch
  return useMemo(
    () => ({
      can,
      status,
      loading,
      ready: status?.state === 'ready',
      configured: Boolean(status && status.state !== 'not_configured'),
      model: status?.modelName || status?.model || '',
      provider: status?.providerName ?? '',
      featureOn: (f: AIFeature) => Boolean(status?.features.find((x) => x.key === f && x.on && x.available)),
      refetch,
    }),
    [can, status, loading, refetch],
  )
}

export type Assist = ReturnType<typeof useAssist>

export type SuggestError = {
  kind: 'not_configured' | 'feature_off' | 'cap' | 'key' | 'rate' | 'provider' | 'no_image' | 'demo' | 'error'
  message: string
  /** Rate limited: when the provider takes requests again. */
  retryAt?: number
}

/** What went wrong, from the API's problem (its detail is already the sentence to show). */
export function suggestError(e: unknown): SuggestError {
  const message = errorMessage(e)
  switch (problemCode(e)) {
    case 'ai_not_configured':
      return { kind: 'not_configured', message }
    case 'ai_feature_off':
      return { kind: 'feature_off', message }
    case 'ai_cap_reached':
      return { kind: 'cap', message }
    case 'ai_key_rejected':
      return { kind: 'key', message }
    case 'ai_rate_limited': {
      const s = Number(problemMeta<number>(e, 'retryAfter') ?? 0)
      return { kind: 'rate', message, retryAt: Date.now() + (s > 0 ? s : 20) * 1000 }
    }
    case 'ai_provider_error':
      return { kind: 'provider', message }
    case 'ai_no_image_input':
      return { kind: 'no_image', message }
    case 'ai_demo_model':
      return { kind: 'demo', message }
  }
  return { kind: 'error', message }
}

/** Whether asking again can help: a provider error or limit, not the cap, the key or the settings. */
export function retryable(e: SuggestError): boolean {
  return e.kind === 'rate' || e.kind === 'provider' || e.kind === 'error'
}

export type SuggestState<T> = { phase: 'idle' } | { phase: 'working' } | { phase: 'done'; data: T } | { phase: 'error'; error: SuggestError }

/**
 * One suggestion's life: asking (cancellable), the answer, or what went
 * wrong. It starts from the person's click (never from an effect, so a
 * request is never sent twice), and nothing changes in the content until
 * the caller accepts.
 */
export function useSuggest<V, T>(run: (vars: V, signal: AbortSignal) => Promise<T>) {
  const [state, setState] = useState<SuggestState<T>>({ phase: 'idle' })
  const runRef = useRef(run)
  const ctl = useRef<AbortController | null>(null)
  const last = useRef<{ vars: V } | null>(null)
  useEffect(() => {
    runRef.current = run
  })
  useEffect(() => () => ctl.current?.abort(), [])
  // Resolves with the answer (undefined when it failed or was cancelled).
  const start = useCallback(async (vars: V): Promise<T | undefined> => {
    last.current = { vars }
    ctl.current?.abort()
    const c = new AbortController()
    ctl.current = c
    setState({ phase: 'working' })
    try {
      const data = await runRef.current(vars, c.signal)
      if (c.signal.aborted) return undefined
      setState({ phase: 'done', data })
      return data
    } catch (e) {
      if (!c.signal.aborted) setState({ phase: 'error', error: suggestError(e) })
      return undefined
    }
  }, [])
  const retry = useCallback(() => {
    if (last.current) void start(last.current.vars)
  }, [start])
  const cancel = useCallback(() => {
    ctl.current?.abort()
    setState({ phase: 'idle' })
  }, [])
  return useMemo(() => ({ state, start, retry, cancel }), [state, start, retry, cancel])
}

export type Suggest<V, T> = ReturnType<typeof useSuggest<V, T>>

/** Records what the person did with a suggestion (the AI activity log); never fails the screen. */
export function useOutcome() {
  const { slug } = useWorkspace()
  return useCallback(
    (requestId: string | undefined, outcome: Outcome, detail?: string) => {
      if (!requestId) return
      void http.POST(`${aiBase}/requests/{request}/outcome`, { params: { path: { workspace: slug, request: requestId } }, body: { outcome, detail } }).catch(() => undefined)
    },
    [slug],
  )
}

/** "Sent to Anthropic (Claude Sonnet 5.5): …. No contact data. About $0.01." */
export function sentText(t: ReturnType<typeof useT>, msg: MessageDescriptor, sent: Pick<AISent, 'providerName' | 'modelName' | 'costMicros'>, values: Record<string, string | number> = {}): string {
  return `${t(msg, { provider: sent.providerName, model: sent.modelName, ...values })} ${costLine(sent.costMicros)}`
}

/** The same line before the answer, with the estimate for what is about to be sent. */
export function sendingText(t: ReturnType<typeof useT>, msg: MessageDescriptor, status: AIStatus | undefined, micros: number, values: Record<string, string | number> = {}): string {
  return `${t(msg, { provider: status?.providerName ?? '', model: status?.modelName || status?.model || '', ...values })} ${costLine(micros)}`
}

/** A check under a suggestion: "1 token kept: first_name", "Says what the chart shows". */
export type Check = { ok: boolean; text: ReactNode; key?: string }

/**
 * Every suggestion card (AIComposer, AISegmentReport, AIStates): the
 * feature and the model, the suggestion, its checks, Accept · Edit ·
 * Reject · Try again, and what was sent and what it cost.
 */
export function SuggestionCard({
  title,
  model,
  onClose,
  tone,
  children,
  checks,
  actions,
  sent,
  floating,
  className,
}: {
  title: string
  model?: string
  onClose?: () => void
  tone?: 'bad' | 'warn'
  children?: ReactNode
  checks?: Check[]
  actions?: ReactNode
  sent?: string
  floating?: boolean
  className?: string
}) {
  const t = useT()
  return (
    <section
      className={`ai-sug${tone ? ` ai-sug-${tone}` : ''}${floating ? ' ai-sug-float' : ''}${className ? ` ${className}` : ''}`}
      aria-label={t(ms.cardLabel, { title })}
    >
      <div className="ai-sug-h">
        <Sparkle />
        <span className="ai-sug-t">{title}</span>
        {model ? <span className="ai-sug-m">{t(m.cardModel, { model })}</span> : null}
        {onClose ? (
          <button type="button" className="ai-sug-x" aria-label={t(ms.closeSuggestion)} onClick={onClose}>
            <Icon name="close" size={14} />
          </button>
        ) : null}
      </div>
      {children ? <div className="ai-sug-b">{children}</div> : null}
      {checks?.length ? (
        <div className="ai-sug-c">
          {checks.map((c, i) => (
            <span key={c.key ?? i} className={c.ok ? undefined : 'ai-no'}>
              {c.ok ? <Icon name="check" size={13} strokeWidth={2.2} /> : <Icon name="alertCircle" size={13} />}
              <span>{c.text}</span>
            </span>
          ))}
        </div>
      ) : null}
      {actions ? <div className="ai-sug-a">{actions}</div> : null}
      {sent ? <div className="ai-sug-f">{sent}</div> : null}
    </section>
  )
}

/** "Writing…" with skeleton lines (AIComposer · working). */
export function Writing() {
  const t = useT()
  return (
    <>
      <div className="ai-skel" aria-hidden="true">
        <span style={{ width: '100%' }} />
        <span style={{ width: '94%' }} />
        <span style={{ width: '58%' }} />
      </div>
      <div role="status" className="flex flex-wrap items-center gap-2.5">
        <span className="font-semibold">{t(m.writing)}</span>
        <span className="k">{t(m.writingHint)}</span>
      </div>
    </>
  )
}

/** "Try again in 20 s", counting down, then Try again (AIStates · rate limit). */
export function RetryButton({ error, onRetry }: { error: SuggestError; onRetry: () => void }) {
  const t = useT()
  const [now, setNow] = useState(() => Date.now())
  const left = error.retryAt ? Math.max(0, Math.ceil((error.retryAt - now) / 1000)) : 0
  useEffect(() => {
    if (!left) return
    const id = window.setInterval(() => setNow(Date.now()), 1000)
    return () => window.clearInterval(id)
  }, [left])
  if (left > 0) {
    return (
      <Button size="sm" icon="clock" disabled aria-live="polite">
        {t(m.tryAgainIn, { seconds: left })}
      </Button>
    )
  }
  return (
    <Button size="sm" icon="refresh" onClick={onRetry}>
      {t(m.tryAgain)}
    </Button>
  )
}

/**
 * A card for any state but the answer: working (with Cancel), an error, the
 * cap, a rejected key, not configured. `children` renders the answer.
 */
export function SuggestFlow<T>({
  title,
  assist,
  s,
  sending,
  onClose,
  onRetry,
  floating,
  className,
  done,
}: {
  title: string
  assist: Assist
  s: SuggestState<T>
  /** The "Sent to …" line while working (an estimate). */
  sending?: string
  onClose: () => void
  onRetry: () => void
  floating?: boolean
  className?: string
  done: (data: T) => { body: ReactNode; checks?: Check[]; actions?: ReactNode; sent?: string; tone?: 'bad' | 'warn' }
}) {
  const t = useT()
  if (s.phase === 'idle') return null
  if (s.phase === 'working') {
    return (
      <SuggestionCard title={title} model={assist.model} onClose={onClose} sent={sending} floating={floating} className={className} actions={<Button size="sm" onClick={onClose}>{t(m.cancel)}</Button>}>
        <Writing />
      </SuggestionCard>
    )
  }
  if (s.phase === 'error') {
    const e = s.error
    const retry = retryable(e)
    return (
      <SuggestionCard
        title={title}
        model={assist.model}
        onClose={onClose}
        tone={e.kind === 'rate' ? 'warn' : 'bad'}
        floating={floating}
        className={className}
        actions={
          <>
            {retry ? <RetryButton error={e} onRetry={onRetry} /> : null}
            {e.kind === 'not_configured' ? <ConnectButton assist={assist} /> : null}
            <Button size="sm" onClick={onClose}>
              {t(m.close)}
            </Button>
          </>
        }
      >
        <AINote tone={e.kind === 'rate' ? 'warn' : 'bad'}>{e.message}</AINote>
      </SuggestionCard>
    )
  }
  const d = done(s.data)
  return (
    <SuggestionCard title={title} model={assist.model} onClose={onClose} tone={d.tone} checks={d.checks} actions={d.actions} sent={d.sent} floating={floating} className={className}>
      {d.body}
    </SuggestionCard>
  )
}

/** The model's refusal, in its own words (AIStates · a refusal). Nothing changed. */
export function Refusal({ model, text, instruction }: { model: string; text: string; instruction?: string }) {
  const t = useT()
  return (
    <>
      {instruction ? (
        <div className="flex flex-col gap-0.5">
          <span className="k">{t(ms.yourInstruction)}</span>
          <span className="font-semibold">{t(ms.quoted, { text: instruction })}</span>
        </div>
      ) : null}
      <AINote tone="neutral">
        <span className="k">{t(m.modelAnswered, { model })}</span>
        <div>{t(ms.quoted, { text })}</div>
      </AINote>
      <span className="font-semibold">{t(m.nothingChanged)}</span>
    </>
  )
}

/** HostRepute's own assistant, for questions about sender reputation. */
export const HOSTREPUTE_ASSISTANT = 'https://app.hostrepute.com/assistant'

/** "Sender reputation isn't something this assistant looks at." (AIStates). */
export function Reputation({ hostRepute, instruction }: { hostRepute: boolean; instruction?: string }) {
  const t = useT()
  return (
    <>
      {instruction ? (
        <div className="flex flex-col gap-0.5">
          <span className="k">{t(ms.yourInstruction)}</span>
          <span className="font-semibold">{t(ms.quoted, { text: instruction })}</span>
        </div>
      ) : null}
      <AINote tone="info">
        <span className="font-semibold">{t(m.reputationTitle)}</span>
        <div>{hostRepute ? t(m.reputationHostRepute) : t(m.reputationNoHostRepute)}</div>
      </AINote>
      <span className="k">{t(m.reputationNothingSent)}</span>
    </>
  )
}

/** Ask HostRepute's assistant, or open Deliverability without HostRepute. */
export function ReputationAction({ hostRepute }: { hostRepute: boolean }) {
  const { slug } = useWorkspace()
  const t = useT()
  if (hostRepute) {
    return (
      <a className="btn sm pri" href={HOSTREPUTE_ASSISTANT} target="_blank" rel="noreferrer">
        <Icon name="shield" size={14} />
        {t(m.askHostRepute)}
        <span className="vh">{t(ms.newTab)}</span>
      </a>
    )
  }
  return (
    <ButtonLink size="sm" variant="primary" to="/w/$workspace/deliverability" params={{ workspace: slug }}>
      {t(m.openDeliverability)}
    </ButtonLink>
  )
}

/** "Connect a model": to Settings › Integrations · AI assistant (people with integrations:manage). */
export function ConnectButton({ assist, size = 'sm' }: { assist: Assist; size?: 'sm' | 'md' }) {
  const { slug } = useWorkspace()
  const t = useT()
  if (!assist.status?.canManage) return null
  return (
    <ButtonLink size={size} variant="primary" to="/w/$workspace/settings/integrations/ai" params={{ workspace: slug }}>
      {t(m.connectModel)}
    </ButtonLink>
  )
}

/**
 * Not configured (AIStates): an invitation to connect a model, never an
 * upgrade prompt. The words depend on whether the reader can connect one.
 */
export function NotConfigured({ assist, workspace }: { assist: Assist; workspace: string }) {
  const t = useT()
  const s = assist.status
  const helpers = s?.helpers ?? []
  return (
    <div className="flex flex-col gap-2.5">
      <p className="m-0 text-[13px] leading-normal">{t(ms.notConfiguredLead, { workspace })}</p>
      {s?.canManage ? (
        <>
          <div>
            <ConnectButton assist={assist} />
          </div>
          <span className="k">{t(m.connectWhere)}</span>
        </>
      ) : (
        <span className="text-[12.5px]">{helpers.length ? t(ms.notConfiguredAsk, { helpers: formatList(helpers, 'disjunction') }) : t(ms.notConfiguredNobody)}</span>
      )}
    </div>
  )
}

/** "Claude Sonnet 5.5 · $7.84 of $25.00 this month." */
export function usageLine(t: ReturnType<typeof useT>, s: AIStatus | undefined): string {
  if (!s) return ''
  const model = s.modelName || s.model || ''
  if (s.cap.kind === 'none') return t(ms.usageNoCap, { model, spent: dollars(s.usage.costMicros) })
  if (s.cap.kind === 'tokens') return t(ms.usageTokens, { model, spent: tokens(s.usage.tokens), cap: tokens(s.cap.tokens) })
  return t(ms.usageDollars, { model, spent: dollars(s.usage.costMicros), cap: money(s.cap.usd) })
}

/** A sentence with <b>, <mono> and <a> from the assistant's messages. */
export function Rich({ msg, values }: { msg: MessageDescriptor; values?: Record<string, ReactNode> }) {
  return <T msg={msg} values={{ ...tags, ...values }} />
}
