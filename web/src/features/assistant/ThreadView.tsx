import { useCallback, useEffect, useId, useLayoutEffect, useMemo, useRef, useState, useSyncExternalStore, type ReactNode } from 'react'
import { useQueryClient } from '@tanstack/react-query'
import { ApiError, errorMessage } from '@/api/client'
import { Button, ButtonLink } from '@/components/ui/Button'
import { Skeleton } from '@/components/ui/Feedback'
import { LoadFailed } from '@/components/ui/LoadFailed'
import { toast } from '@/components/ui/Toast'
import { useAIStatus } from '@/features/ai/api'
import { useSharingReview } from '@/features/ai/SharingReview'
import { useT } from '@/i18n'
import { count, money } from '@/lib/format'
import { deniedReason, useCan, useWorkspace, workspaceMeKey } from '@/lib/session'
import type { NotificationItem } from '@/components/shell/NotificationsPopover'
import { startSignIn, useUsableMCPServers, type MCPServer } from '@/features/mcpservers/api'
import { attachContext, createThread, decideConfirmation, detachContext, requestKey, sendMessage, setThreadServers, stopTurn, testModelTools, updateThread, useAssistantModels, useRefreshThreads, useThread, type AssistantModel } from './api'
import { Composer, type ComposerHandle } from './Composer'
import { CONFIRM_ID, ConfirmCard, MCPStates, pendingConfirmation, WaitingLine } from './Confirmation'
import { buildItems, Conversation, LiveContext } from './Conversation'
import { placeholderOf, suggestions, useResourceTitle, useScreenContext } from './context'
import { AsstIcon } from './icons'
import { mx } from './mcp.messages'
import { m } from './messages'
import { ModelPicker } from './ModelPicker'
import { turnActive, type Confirmation, type ThreadContext } from './reducer'
import { ServerPicker } from './ServerPicker'
import { modelProblems, problemCopy, problemOf, StateCard, useProblemTitle, type TurnProblem } from './states'
import { closeAssistant, getAssistant, markDraftTaken, setAssistantThread, untakenDraft, useAssistant, waitingThread } from './store'
import { MAX_ATTEMPTS } from '@/features/live/useLiveSend'
import { useThreadState, useThreadStream } from './useThreadStream'

/** A state that keeps the assistant from answering here at all. */
export type Gate = 'noAccess' | 'off' | 'noModel' | 'noUsable' | 'expired' | 'lost'

const clip = (s: string, n: number) => (s.length > n ? `${s.slice(0, n - 1).trimEnd()}…` : s)

/** A text that renders only where it's read (the composer's draft). */
function textStore(initial: string) {
  let value = initial
  const listeners = new Set<() => void>()
  return {
    get: () => value,
    set: (next: string | ((now: string) => string)) => {
      const v = typeof next === 'function' ? next(value) : next
      if (v === value) return
      value = v
      for (const l of listeners) l()
    },
    subscribe: (l: () => void) => {
      listeners.add(l)
      return () => {
        listeners.delete(l)
      }
    },
  }
}

/** An answer's first sentence, without Markdown marks, for the live region. */
export function firstSentence(text: string): string {
  const plain = text
    .replace(/```[\s\S]*?```/g, ' ')
    .replace(/[*_#>|`]/g, '')
    .replace(/\[([^\]]*)\]\([^)]*\)/g, '$1')
    .replace(/\s+/g, ' ')
    .trim()
  const end = plain.search(/[.!?。](\s|$)/)
  return clip(end >= 0 ? plain.slice(0, end + 1) : plain, 160)
}

/**
 * One thread's data and actions, for the panel and the full page: the
 * thread from the API and its stream, the models, the screen's suggested
 * context, the composer's draft and what the live region says.
 */
export function useThreadController(threadId: string | undefined, opts: { onNewThread?: () => void } = {}) {
  const t = useT()
  const { slug, me } = useWorkspace()
  const canUse = useCan('ai:use')
  const canManage = useCan('integrations:manage')
  const ui = useAssistant()
  const status = useAIStatus()
  const helpers = status.data?.helpers ?? []
  const models = useAssistantModels()
  const query = useThread(threadId)
  const refreshThreads = useRefreshThreads()
  const [state, dispatch, live] = useThreadState()
  // The composer's text lives outside this state: a keystroke renders the composer alone.
  const [draft] = useState(() => textStore(untakenDraft()))
  const setDraft = draft.set
  const [picked, setPicked] = useState<string>()
  const [pickerOpen, setPickerOpen] = useState(false)
  const [announce, setAnnounce] = useState('')
  // A message refused before its turn started (the monthly cap, a rejected key…), with the server's words.
  const [sendProblem, setSendProblem] = useState<{ problem: TurnProblem; detail?: string }>()
  // The composer registers its handle; the controller only focuses it.
  const composer = useRef<ComposerHandle | null>(null)
  const attachComposer = useCallback((h: ComposerHandle | null) => {
    composer.current = h
  }, [])
  /** Focus to the composer: at once when it is there (the panel opening), else once it is. */
  const focusComposer = useCallback((now = false) => {
    if (now && composer.current) composer.current.focus()
    else window.setTimeout(() => composer.current?.focus())
  }, [])
  const screen = useScreenContext()
  const resourceTitle = useResourceTitle(screen.resource)
  // MCP servers: those the person may pick, the thread's, and a call waiting on them.
  const usable = useUsableMCPServers()
  const servers = usable.data?.servers ?? []
  const [pickedServers, setPickedServers] = useState<string[]>([])
  const [deciding, setDeciding] = useState<'allow' | 'deny'>()
  const [closedConf, setClosedConf] = useState<Confirmation>()
  const [confProblem, setConfProblem] = useState<{ id: string; code: 'mcp_tool_changed' | 'mcp_tool_blocked' }>()

  // A new thread starts from nothing; the API's thread replaces what was kept.
  useEffect(() => {
    dispatch({ type: 'reset' })
  }, [threadId, dispatch])
  useEffect(() => {
    if (query.data && query.data.id === threadId) dispatch({ type: 'thread', thread: query.data })
  }, [query.data, threadId, dispatch])
  // A thread that is gone (deleted, not this person's) leaves a new one.
  useEffect(() => {
    if (query.error instanceof ApiError && query.error.status === 404) setAssistantThread(slug, undefined)
  }, [query.error, slug])

  // The server removes a waiting call's notification once the call is
  // decided, stopped or expired: a thread that shows here without its call
  // waiting, while a notification of it is still kept, has the bell and the
  // launcher's dot asked again.
  const qc = useQueryClient()
  const shown = state.thread && state.thread.id === threadId ? (state.turn?.state ?? 'none') : undefined
  useEffect(() => {
    if (!threadId || !shown || shown === 'waiting') return
    const notes = qc.getQueriesData<{ items?: NotificationItem[] }>({ queryKey: ['get', '/api/v1/workspaces/{workspace}/notifications'] })
    if (!notes.some(([, d]) => d?.items?.some((n) => waitingThread(n) === threadId))) return
    void qc.invalidateQueries({ queryKey: ['get', '/api/v1/workspaces/{workspace}/notifications'] })
    void qc.invalidateQueries({ queryKey: workspaceMeKey(slug) })
  }, [threadId, shown, qc, slug])

  // A stopped turn's request is accounted after it stops, with no usage
  // event: read the thread again for its cost.
  const { refetch } = query
  const turnKey = state.turn?.state === 'stopped' ? state.turn.id : undefined
  useEffect(() => {
    if (!turnKey) return
    const id = window.setTimeout(() => void refetch(), 2500)
    return () => window.clearTimeout(id)
  }, [turnKey, refetch])

  const stream = useThreadStream(slug, threadId, state.lastId, Boolean(state.thread && state.thread.id === threadId && state.thread.state === 'open'), dispatch)

  // ⌘K › Ask the assistant: the question waits in the composer, unsent.
  const nonce = useRef(ui.draftNonce)
  useEffect(() => {
    if (nonce.current !== ui.draftNonce) {
      nonce.current = ui.draftNonce
      setDraft(getAssistant().draft)
    }
    markDraftTaken(ui.draftNonce)
    if (ui.draftNonce) focusComposer()
  }, [ui.draftNonce, focusComposer, setDraft])

  const list = models.data?.models ?? []
  const modelId = state.thread?.model?.id ?? picked ?? models.data?.defaultId
  const model: AssistantModel | undefined = list.find((x) => x.id === modelId) ?? state.thread?.model
  const active = turnActive(state.turn) || Boolean(state.pending)

  // The live region: the start, the end with its first sentence, a stop.
  // Worked out as the turn changes state (render-time state, not an effect).
  const [seen, setSeen] = useState<{ turn?: string; state?: string }>({})
  const turnNow = state.turn
  if (turnNow && (seen.turn !== turnNow.id || seen.state !== turnNow.state)) {
    setSeen({ turn: turnNow.id, state: turnNow.state })
    if (seen.turn === turnNow.id) {
      if (turnNow.state === 'done') {
        const last = [...state.messages].reverse().find((x) => x.role === 'assistant' && x.turnId === turnNow.id)
        if (last) setAnnounce(t(m.announceDone, { sentence: firstSentence(last.text) }))
      } else if (turnNow.state === 'stopped') setAnnounce(t(m.announceStopped))
      else if (turnNow.state === 'failed')
        setAnnounce(t(m.announceFailed, { title: t(problemCopy[problemOf(turnNow.reason)].title, { provider: model?.providerName ?? '', model: model?.modelName ?? '' }) }))
    }
  }

  // A call that waits for the person is announced once.
  const asking = state.confirmation?.state === 'pending' ? state.confirmation : undefined
  const [seenConf, setSeenConf] = useState<string>()
  if (asking && seenConf !== asking.id) {
    setSeenConf(asking.id)
    setAnnounce(t(mx.announce, { server: asking.serverName, tool: asking.tool }))
  }

  // What keeps the assistant from answering here.
  const offCode = 'ai_assistant_off'
  let gate: Gate | undefined
  if (!canUse) gate = 'noAccess'
  else if ((models.error instanceof ApiError && models.error.code === offCode) || state.problem?.code === offCode || (query.error instanceof ApiError && query.error.code === offCode)) gate = 'off'
  else if (state.problem && state.problem.status === 403) gate = 'lost'
  else if (models.isSuccess && list.length === 0) gate = 'noModel'
  else if (models.isSuccess && !list.some((x) => x.usable) && !state.thread) gate = 'noUsable'
  else if (state.thread?.state === 'expired' || (query.error instanceof ApiError && query.error.code === 'ai_assistant_thread_expired')) gate = 'expired'

  /** The thread to write to: this one, or a new one on the screen it opened on. */
  const ensureThread = useCallback(
    async (title?: string): Promise<string> => {
      if (threadId) return threadId
      let th = await createThread(slug, { requestKey: requestKey(), screen: screen.screen, modelId: picked, title: title ? clip(title, 80) : undefined })
      // Servers picked before the thread existed come with it.
      if (pickedServers.length) th = await setThreadServers(slug, th.id, pickedServers)
      dispatch({ type: 'thread', thread: th })
      setAssistantThread(slug, th.id)
      void refreshThreads()
      return th.id
    },
    [threadId, slug, screen.screen, picked, pickedServers, refreshThreads, dispatch],
  )

  const send = useCallback(
    async (text: string) => {
      const body = text.trim()
      if (!body || active || gate) return
      dispatch({ type: 'pending', text: body })
      setDraft('')
      try {
        const id = await ensureThread(body)
        if (state.thread && !state.thread.title) void updateThread(slug, id, { title: clip(body, 80) }).then(() => refreshThreads())
        const th = await sendMessage(slug, id, body, requestKey())
        setSendProblem(undefined)
        dispatch({ type: 'thread', thread: th })
        setAnnounce(t(m.announceStart))
      } catch (e) {
        dispatch({ type: 'pending' })
        setDraft((d) => d || body)
        // A refusal the panel has a state for shows as that state; anything else is a toast.
        const known = e instanceof ApiError && e.code ? problemOf(e.code.replace(/^ai_/, '')) : 'failed'
        if (known !== 'failed') {
          setSendProblem({ problem: known, detail: e instanceof ApiError ? e.problem.detail : undefined })
          setAnnounce(t(m.announceFailed, { title: t(problemCopy[known].title, { provider: model?.providerName ?? '', model: model?.modelName ?? '' }) }))
        } else toast({ kind: 'error', title: t(m.sendFailed), detail: errorMessage(e) })
      }
    },
    [active, gate, ensureThread, state.thread, slug, t, refreshThreads, model, dispatch, setDraft],
  )

  const stop = useCallback(async () => {
    const turn = state.turn
    if (!threadId || !turn || !turnActive(turn)) return
    try {
      dispatch({ type: 'thread', thread: await stopTurn(slug, threadId, turn.id) })
    } catch (e) {
      toast({ kind: 'error', title: t(m.stopFailed), detail: errorMessage(e) })
    }
  }, [state.turn, threadId, slug, t, dispatch])

  const lastQuestion = [...state.messages].reverse().find((x) => x.role === 'user')?.text ?? ''
  // A refused message waits in the composer; a failed turn's question is the thread's last.
  const retry = () => void send(sendProblem ? draft.get() || lastQuestion : lastQuestion)
  const edit = () => {
    setDraft(lastQuestion)
    focusComposer()
  }

  const pickModel = async (id: string) => {
    const chosen = list.find((x) => x.id === id)
    if (!chosen || !chosen.usable) return
    if (!threadId || !state.thread) setPicked(id)
    else {
      try {
        dispatch({ type: 'thread', thread: await updateThread(slug, threadId, { modelId: id }) })
      } catch (e) {
        toast({ kind: 'error', title: errorMessage(e) })
        return
      }
    }
    setAnnounce(t(m.modelChanged, { model: chosen.modelName || chosen.model }))
  }

  const [review, reviewDialog] = useSharingReview(model?.providerName ?? t(m.title))
  const suggestedTitle =
    screen.resource && resourceTitle && !state.thread?.contexts.some((c) => c.targetId === screen.resource!.id)
      ? screen.resource.kind === 'campaign'
        ? t(m.draftTitle, { title: resourceTitle })
        : resourceTitle
      : undefined
  const attach = () => {
    const res = screen.resource
    if (!res || !resourceTitle) return
    review(async () => {
      try {
        const id = await ensureThread()
        dispatch({ type: 'thread', thread: await attachContext(slug, id, { id: res.id, kind: res.kind, sharingReviewed: true }) })
        setAnnounce(t(m.attached, { title: resourceTitle }))
        // The Attach button is gone: the composer has focus.
        focusComposer()
      } catch (e) {
        toast({ kind: 'error', title: t(m.attachFailed, { title: resourceTitle }), detail: errorMessage(e) })
      }
    })
  }
  const detach = async (c: ThreadContext) => {
    if (!threadId) return
    const title = c.title || t(m.unavailable)
    try {
      dispatch({ type: 'thread', thread: await detachContext(slug, threadId, c.id) })
      setAnnounce(t(m.removed, { title }))
      focusComposer()
    } catch (e) {
      toast({ kind: 'error', title: t(m.removeFailed, { title }), detail: errorMessage(e) })
    }
  }

  const serverIds = state.thread?.mcpServerIds ?? pickedServers
  const toggleServer = async (id: string, on: boolean) => {
    const server = servers.find((x) => x.id === id)
    const next = on ? [...new Set([...serverIds, id])] : serverIds.filter((x) => x !== id)
    if (!threadId || !state.thread) setPickedServers(next)
    else {
      try {
        dispatch({ type: 'thread', thread: await setThreadServers(slug, threadId, next) })
      } catch (e) {
        toast({ kind: 'error', title: t(mx.failed), detail: errorMessage(e) })
        return
      }
    }
    if (server) setAnnounce(t(on ? mx.on : mx.off, { server: server.name }))
  }
  /** The person's own sign-in to an OAuth server: the browser goes to its authorization page and comes back to Settings › MCP servers. */
  const signIn = async (server: MCPServer) => {
    try {
      await startSignIn(slug, server.id)
    } catch (e) {
      toast({ kind: 'error', title: t(mx.signInFailed, { server: server.name }), detail: errorMessage(e) })
    }
  }
  /** Allow once or Deny: sent once; a call closed meanwhile (409) keeps its card, saying so. */
  const decide = async (conf: Confirmation, decision: 'allow' | 'deny') => {
    if (!threadId || deciding) return
    setDeciding(decision)
    try {
      dispatch({ type: 'thread', thread: await decideConfirmation(slug, threadId, conf.id, decision) })
      setAnnounce(t(decision === 'allow' ? mx.announceAllowed : mx.announceDenied, { server: conf.serverName, tool: conf.tool }))
      focusComposer()
    } catch (e) {
      if (e instanceof ApiError && e.status === 409 && e.code === 'ai_assistant_confirmation_closed') {
        setClosedConf(conf)
        setAnnounce(t(mx.closed))
        void refetch()
      } else if (e instanceof ApiError && (e.code === 'mcp_tool_changed' || e.code === 'mcp_tool_blocked')) {
        // The tool moved since the model asked: it can't run, only be denied.
        setConfProblem({ id: conf.id, code: e.code })
        setAnnounce(t(e.code === 'mcp_tool_blocked' ? mx.confirmBlocked : mx.confirmChanged, { server: conf.serverName, tool: conf.tool }))
      } else toast({ kind: 'error', title: t(mx.decideFailed), detail: errorMessage(e) })
    } finally {
      setDeciding(undefined)
    }
  }

  const newThread = () => {
    setAssistantThread(slug, undefined)
    setPicked(undefined)
    setPickedServers([])
    setSendProblem(undefined)
    if (opts.onNewThread) opts.onNewThread()
    else focusComposer()
  }

  return {
    slug,
    workspace: me.workspace.name,
    threadId,
    query,
    state,
    live,
    stream,
    models,
    list,
    model,
    gate,
    active,
    canManage,
    helpers,
    screen,
    resourceTitle,
    suggestedTitle,
    draft,
    setDraft,
    attachComposer,
    focusComposer,
    pickerOpen,
    setPickerOpen,
    announce,
    sendProblem,
    reviewDialog,
    send,
    stop,
    retry,
    edit,
    pickModel,
    attach,
    detach,
    newThread,
    servers,
    serverIds,
    toggleServer,
    signIn,
    decide,
    deciding,
    closedConf,
    confirmClosed: closedConf?.id,
    confProblem,
  }
}

export type ThreadController = ReturnType<typeof useThreadController>

/** The thread's cost against its ceiling, a warning from 80%. */
function Cost({ c }: { c: ThreadController }) {
  const t = useT()
  const thread = c.state.thread
  if (!thread) return null
  const usage = c.state.usage ?? thread.usage
  const used = money(usage.costMicros / 1e6)
  const limit = money(thread.limits.costMicros / 1e6)
  const warn = thread.limits.costMicros > 0 && usage.costMicros >= thread.limits.costMicros * 0.8
  return (
    <span className={warn ? 'asst-cost warn' : 'asst-cost'} title={t(m.costTitle, { calls: usage.toolCalls, limit: thread.limits.toolCalls })}>
      {warn ? <AsstIcon name="warn" size={14} /> : null}
      {usage.uncertainRequests > 0
        ? t(m.costUnknown, { used, limit, count: usage.uncertainRequests, held: money(usage.uncertainCostMicros / 1e6) })
        : t(m.cost, { used, limit })}
    </span>
  )
}

/** The model picker and the cost, then the attached and suggested context. */
export function ThreadSub({ c }: { c: ThreadController }) {
  const t = useT()
  const contexts = c.state.thread?.contexts ?? []
  const showChips = !c.gate && (contexts.length > 0 || c.suggestedTitle)
  return (
    <div className="asst-subwrap">
      {c.list.length || c.state.thread ? (
        <div className="asst-sub">
          {c.list.length ? <ModelPicker models={c.list} value={c.model?.id} onChange={(id) => void c.pickModel(id)} open={c.pickerOpen} onOpenChange={c.setPickerOpen} /> : null}
          {!c.gate && (c.servers.length || c.serverIds.length) ? (
            <ServerPicker servers={c.servers} value={c.serverIds} onToggle={(id, on) => void c.toggleServer(id, on)} onSignIn={(srv) => void c.signIn(srv)} canManage={c.canManage} slug={c.slug} />
          ) : null}
          <Cost c={c} />
        </div>
      ) : null}
      {showChips ? (
        <div className="asst-sub asst-chips" role="group" aria-label={t(m.contextLabel)} data-scroll-row="">
          {contexts.map((ctx) => {
            const title = ctx.available ? ctx.title : t(m.unavailable)
            return (
              <span key={ctx.id} className={ctx.available ? 'asst-chip' : 'asst-chip off'}>
                <AsstIcon name="link" />
                <span>{title}</span>
                <button type="button" onClick={() => void c.detach(ctx)} aria-label={t(m.remove, { title })}>
                  <AsstIcon name="x" />
                </button>
              </span>
            )
          })}
          {c.suggestedTitle ? (
            <span className="asst-chip suggested">
              <span>{t(m.suggested, { title: c.suggestedTitle })}</span>
              <button type="button" onClick={c.attach}>
                {t(m.attach)}
              </button>
            </span>
          ) : null}
        </div>
      ) : null}
      {c.reviewDialog}
    </div>
  )
}

/** A sheet or the full screen would cover the page a card's link opens. */
const uncover = () => {
  if (!document.querySelector('.asst-docked')) closeAssistant({ refocus: false })
}

/** A state card's manager action: a link for those who can, a disabled button naming who can for the rest. */
function ManagerAction({ c, label }: { c: ThreadController; label: string }) {
  if (c.canManage)
    return (
      <ButtonLink to="/w/$workspace/settings/integrations/ai" params={{ workspace: c.slug }} variant="primary" onClick={uncover}>
        {label}
      </ButtonLink>
    )
  return (
    <Button variant="primary" disabledReason={deniedReason('integrations:manage', c.helpers)}>
      {label}
    </Button>
  )
}

/** A state card's action on the person's own key: Settings › Integrations › AI models, at Your personal key. */
function OwnKeyAction({ c, label }: { c: ThreadController; label: string }) {
  return (
    <ButtonLink to="/w/$workspace/settings/integrations/ai" params={{ workspace: c.slug }} hash="ai-yk-h" variant="primary" onClick={uncover}>
      {label}
    </ButtonLink>
  )
}

/**
 * Runs the model's tool check again, from its card: the person's own
 * model, or a workspace model for those who manage integrations (the rest
 * see who can). Once it passes, Retry sends the message.
 */
function ToolCheck({ c }: { c: ThreadController }) {
  const t = useT()
  const [state, setState] = useState<'idle' | 'running' | 'passed'>('idle')
  const model = c.model
  const name = model?.modelName || model?.model || ''
  if (state === 'passed')
    return (
      <Button variant="primary" onClick={c.retry}>
        {t(m.retry)}
      </Button>
    )
  if (!model?.personal && !c.canManage)
    return (
      <Button variant="primary" disabledReason={deniedReason('integrations:manage', c.helpers)}>
        {t(m.runCheck)}
      </Button>
    )
  const run = async () => {
    if (!model) return
    setState('running')
    try {
      const passed = await testModelTools(c.slug, model)
      await c.models.refetch()
      setState(passed ? 'passed' : 'idle')
      toast(passed ? { kind: 'success', title: t(m.checkPassed, { model: name }) } : { kind: 'error', title: t(m.checkFailed, { model: name }) })
    } catch (e) {
      setState('idle')
      toast({ kind: 'error', title: t(m.checkFailed, { model: name }), detail: errorMessage(e) })
    }
  }
  return (
    <Button variant="primary" loading={state === 'running'} onClick={() => void run()}>
      {t(m.runCheck)}
    </Button>
  )
}

/** A gate's card: the assistant can't answer here at all. */
function GateCard({ c, gate }: { c: ThreadController; gate: Gate }) {
  const t = useT()
  switch (gate) {
    case 'noAccess':
      return (
        <StateCard tone="bad" badge={t(m.noAccessBadge)} title={t(m.noAccessTitle)}>
          <p>{t(m.noAccessBody, { workspace: c.workspace })}</p>
          <p className="k">{deniedReason('ai:use', c.helpers)}</p>
        </StateCard>
      )
    case 'off':
      return (
        <StateCard tone="neutral" badge={t(m.offBadge)} title={t(m.offTitle)} actions={<ManagerAction c={c} label={t(m.offAction)} />}>
          <p>{t(m.offBody, { workspace: c.workspace })}</p>
        </StateCard>
      )
    case 'noModel':
      return (
        <StateCard tone="neutral" badge={t(m.noModelBadge)} title={t(m.noModelTitle)} actions={<ManagerAction c={c} label={t(m.noModelAction)} />}>
          <p>{t(m.noModelBody)}</p>
        </StateCard>
      )
    case 'noUsable':
      return (
        <StateCard
          tone="warn"
          badge={t(m.noModelBadge)}
          title={t(m.noUsableTitle)}
          actions={
            <>
              <Button onClick={() => c.setPickerOpen(true)}>{t(m.switchModel)}</Button>
              <ManagerAction c={c} label={t(m.offAction)} />
            </>
          }
        >
          <p>{t(m.noUsableBody)}</p>
        </StateCard>
      )
    case 'expired':
      return (
        <StateCard tone="neutral" badge={t(m.expiredBadge)} title={t(m.expiredTitle)} actions={<Button variant="primary" onClick={c.newThread}>{t(m.newThreadAction)}</Button>}>
          <p>{t(m.expiredBody)}</p>
        </StateCard>
      )
    case 'lost':
      return (
        <StateCard tone="bad" badge={t(m.lostBadge)} title={t(m.lostTitle)}>
          <p>{t(m.lostBody)}</p>
        </StateCard>
      )
  }
}

/** The latest turn's failure, with what can be done. */
function TurnProblemCard({ c, problem }: { c: ThreadController; problem: TurnProblem }) {
  const t = useT()
  const title = useProblemTitle()
  const thread = c.state.thread
  const usage = c.state.usage ?? thread?.usage
  const values = { provider: c.model?.providerName, model: c.model?.modelName || c.model?.model }
  // The thread's model on the person's own key: they fix it themselves.
  const own = c.model?.personal ? 'yes' : 'no'
  const copy = problemCopy[problem]
  const personal = c.list.find((x) => x.personal && x.usable && x.id !== c.model?.id)
  const retry = <Button variant="primary" onClick={c.retry}>{t(m.retry)}</Button>
  const switchModel = <Button onClick={() => c.setPickerOpen(true)}>{t(m.switchModel)}</Button>
  const newThread = <Button variant="primary" onClick={c.newThread}>{t(m.newThreadAction)}</Button>
  let body: string
  let actions: ReactNode = retry
  switch (problem) {
    case 'threadCap':
      body = t(m.threadCapBody, {
        calls: count(usage?.toolCalls ?? 0),
        callLimit: count(thread?.limits.toolCalls ?? 0),
        tokens: count(usage?.tokens ?? 0),
        tokenLimit: count(thread?.limits.tokens ?? 0),
        cost: money((usage?.costMicros ?? 0) / 1e6),
        costLimit: money((thread?.limits.costMicros ?? 0) / 1e6),
      })
      actions = newThread
      break
    case 'monthly':
      body = t(m.monthlyBody, { workspace: c.workspace }) + (personal ? ` ${t(m.monthlyPersonal, { model: personal.modelName || personal.model })}` : '')
      actions = personal ? (
        <Button
          variant="primary"
          onClick={() => {
            void c.pickModel(personal.id).then(c.retry)
          }}
        >
          {t(m.usePersonal, { model: personal.modelName || personal.model })}
        </Button>
      ) : (
        <ManagerAction c={c} label={t(m.offAction)} />
      )
      break
    case 'personalCap':
      body = t(m.personalCapBody)
      actions = switchModel
      break
    case 'key':
      body = t(m.keyBody, { ...values, own })
      actions = (
        <>
          {own === 'yes' ? <OwnKeyAction c={c} label={t(m.replaceKey)} /> : <ManagerAction c={c} label={t(m.replaceKey)} />}
          {switchModel}
        </>
      )
      break
    case 'disconnected':
      body = t(m.disconnectedBody, { ...values, own })
      actions = (
        <>
          {own === 'yes' ? <OwnKeyAction c={c} label={t(m.openModels)} /> : <ManagerAction c={c} label={t(m.openModels)} />}
          {switchModel}
        </>
      )
      break
    case 'toolsUnchecked':
      body = t(m.toolsBody, { own })
      actions = (
        <>
          <ToolCheck c={c} />
          {switchModel}
        </>
      )
      break
    case 'personalKeysOff':
      body = t(m.personalOffBody, { workspace: c.workspace })
      actions = (
        <Button variant="primary" onClick={() => c.setPickerOpen(true)}>
          {t(m.switchModel)}
        </Button>
      )
      break
    case 'notConfigured':
      body = t(m.notConfiguredBody, { ...values, own })
      actions =
        own === 'yes' ? (
          <>
            <OwnKeyAction c={c} label={t(m.addKey)} />
            {switchModel}
          </>
        ) : (
          <ManagerAction c={c} label={t(m.noModelAction)} />
        )
      break
    case 'confirmationExpired':
      body = t(m.confirmExpiredBody)
      break
    case 'rate':
      body = t(m.rateBody)
      break
    case 'provider':
      body = t(m.providerBody, values)
      actions = (
        <>
          {retry}
          {switchModel}
        </>
      )
      break
    case 'refusal':
      body = t(m.refusalBody, values)
      actions = <Button variant="primary" onClick={c.edit}>{t(m.editMessage)}</Button>
      break
    case 'lost':
      body = t(m.lostBody)
      actions = undefined
      break
    case 'off':
      body = t(m.offBody, { workspace: c.workspace })
      actions = <ManagerAction c={c} label={t(m.offAction)} />
      break
    case 'modelGone':
      body = t(m.modelGoneBody)
      actions = switchModel
      break
    case 'interrupted':
      body = t(m.interruptedBody)
      break
    case 'expired':
      body = t(m.expiredBody)
      actions = newThread
      break
    default:
      body = t(m.failedBody, { reason: c.state.turn?.reason || 'error' })
  }
  // A refused message's detail already says what happened and who can fix it, in the server's words.
  const detail = c.sendProblem?.problem === problem && modelProblems.has(problem) ? c.sendProblem.detail : undefined
  if (detail) body = detail
  return (
    <StateCard tone={copy.tone} badge={t(copy.badge)} title={title(problem, values)} actions={actions}>
      <p>{body}</p>
    </StateCard>
  )
}

const contactTool = /contact|recipient|subscriber|export/i

/** The conversation's log: the empty thread's suggestions, the messages, the latest turn's state and the connection. */
export function ThreadLog({ c, label }: { c: ThreadController; label?: string }) {
  const t = useT()
  const log = useRef<HTMLDivElement>(null)
  const items = useMemo(() => buildItems(c.state), [c.state])
  const turn = c.state.turn
  const liveLength = c.state.live?.text.length ?? 0
  // The conversation sticks to its end while the person is there.
  const near = useRef(true)
  const content = `${items.length}:${c.state.messages.length}:${c.state.pending ? 1 : 0}:${liveLength}:${c.state.tools.length}:${turn?.id}:${turn?.state}:${c.sendProblem?.problem ?? ''}:${c.state.confirmation?.id ?? ''}`
  const sending = Boolean(c.state.pending)
  useLayoutEffect(() => {
    const el = log.current
    // A message the person just sent brings them to the end.
    if (sending) near.current = true
    if (el && near.current) el.scrollTop = el.scrollHeight
  }, [content, sending])
  const onScroll = () => {
    const el = log.current
    if (el) near.current = el.scrollHeight - el.scrollTop - el.clientHeight < 80
  }
  // The streaming answer grows without the log rendering again: it keeps the log at its end itself.
  const liveText = c.live
  const live = useMemo(
    () => ({
      text: liveText,
      stick: () => {
        const el = log.current
        if (el && near.current) el.scrollTop = el.scrollHeight
      },
    }),
    [liveText],
  )
  const loading = Boolean(c.threadId) && !c.state.thread && c.query.isPending
  const empty = !c.gate && !loading && !c.query.isError && items.length === 0 && !c.sendProblem
  const latest = items[items.length - 1]
  const denied = latest && latest.kind === 'answer' && latest.tools.some((r) => !r.ok && (r.code === 'not_available' || r.code === 'forbidden') && contactTool.test(r.name))
  const problem = c.sendProblem?.problem ?? (turn && turn.state === 'failed' ? problemOf(turn.reason) : undefined)
  const [lostFor, setLostFor] = useState(0)
  const reconnecting = c.stream.status === 'retrying' || c.stream.status === 'stopped'
  useEffect(() => {
    if (!reconnecting || !c.stream.lostAt) return
    const tick = () => setLostFor(Math.round((Date.now() - c.stream.lostAt!.getTime()) / 1000))
    tick()
    const id = window.setInterval(tick, 1000)
    return () => window.clearInterval(id)
  }, [reconnecting, c.stream.lostAt])
  const kind = c.screen.kind
  const emptyId = useId()
  // The call the turn waits on, or one closed as it was decided (its card says so).
  const asking = pendingConfirmation(c)
  // An expired call keeps its card, saying so, beside the turn it stopped.
  const expired = c.state.confirmation?.state === 'expired' && c.state.confirmation.turnId === turn?.id ? c.state.confirmation : undefined
  const conf = asking ?? (c.closedConf && c.closedConf.turnId === turn?.id && turnActive(turn) ? c.closedConf : undefined) ?? expired
  const latestRows = latest && latest.kind === 'answer' ? latest.tools : []
  // A new request takes focus, unless the person is typing.
  const askingId = asking?.id
  useEffect(() => {
    if (!askingId) return
    const card = document.getElementById(CONFIRM_ID)
    const el = document.activeElement
    const typing = (el instanceof HTMLTextAreaElement || el instanceof HTMLInputElement) && el.value.trim() !== ''
    if (card && !typing && (el === document.body || (el && log.current?.closest('.asst')?.contains(el)))) card.focus({ preventScroll: true })
  }, [askingId])
  return (
    // The log scrolls: it takes focus, as a scrolling region must.
    // eslint-disable-next-line jsx-a11y/no-noninteractive-tabindex
    <div className="asst-msgs" ref={log} onScroll={onScroll} role="log" aria-live="off" aria-label={label ?? t(m.log)} tabIndex={0}>
      {c.gate && c.gate !== 'lost' ? <GateCard c={c} gate={c.gate} /> : null}
      {loading ? (
        <div className="asst-loading" aria-hidden="true">
          <Skeleton w="60%" h={34} r={12} className="ml-auto" />
          <Skeleton w="90%" h={12} />
          <Skeleton w="80%" h={12} />
          <Skeleton w="50%" h={12} />
        </div>
      ) : null}
      {c.query.isError && !(c.query.error instanceof ApiError && [404, 409].includes(c.query.error.status)) ? <LoadFailed title={t(m.loadFailed)} onRetry={() => void c.query.refetch()} /> : null}
      {empty ? (
        <section className="asst-empty" aria-labelledby={emptyId}>
          <h3 id={emptyId}>{c.resourceTitle ? t(m.emptyAbout, { name: c.resourceTitle }) : t(m.emptyTitle)}</h3>
          <p className="sub">{t(m.emptyBody)}</p>
          <div className="asst-suggestions">
            {suggestions[kind].map((s) => (
              <button
                key={s.id}
                type="button"
                className="btn"
                onClick={() => {
                  c.setDraft(t(s))
                  c.focusComposer()
                }}
              >
                {t(s)}
              </button>
            ))}
          </div>
          <small>{t(m.emptyNote)}</small>
        </section>
      ) : null}
      <LiveContext.Provider value={live}>
        <Conversation items={items}>
          {turn?.state === 'stopped' ? <p className="asst-status">{t(m.stoppedNote)}</p> : null}
          {turn?.state === 'done' && turn.reason === 'length' ? <p className="asst-status">{t(m.lengthNote)}</p> : null}
          {denied ? (
            <StateCard
              tone="bad"
              badge={t(m.contactsBadge)}
              title={t(m.contactsTitle)}
              actions={
                <Button
                  variant="primary"
                  onClick={() => {
                    c.setDraft(t(m.contactsQuestion))
                    c.focusComposer()
                  }}
                >
                  {t(m.contactsAction)}
                </Button>
              }
            >
              <p>{t(m.contactsBody)}</p>
            </StateCard>
          ) : null}
          <MCPStates c={c} rows={latestRows} />
          {conf ? <ConfirmCard c={c} conf={conf} /> : null}
          {problem ? <TurnProblemCard c={c} problem={problem} /> : null}
          {c.gate === 'lost' ? <GateCard c={c} gate="lost" /> : null}
        </Conversation>
      </LiveContext.Provider>
      {reconnecting ? (
        <StateCard
                    tone="warn"
          badge={t(m.reconnectBadge)}
          title={t(m.reconnectTitle)}
          actions={<Button onClick={c.stream.reconnect}>{t(m.reconnectNow)}</Button>}
        >
          <p>{c.stream.status === 'stopped' ? t(m.reconnectStopped, { seconds: lostFor }) : t(m.reconnectBody, { seconds: lostFor, attempt: c.stream.attempt, max: MAX_ATTEMPTS })}</p>
        </StateCard>
      ) : null}
      {asking ? <WaitingLine conf={asking} /> : null}
    </div>
  )
}

/** The composer, with its placeholder for the screen. */
export function ThreadComposer({ c, bound }: { c: ThreadController; bound?: () => HTMLElement | null }) {
  const t = useT()
  const answering = turnActive(c.state.turn)
  const value = useSyncExternalStore(c.draft.subscribe, c.draft.get)
  return (
    <Composer
      handle={c.attachComposer}
      value={value}
      onChange={c.setDraft}
      onSend={() => void c.send(value)}
      onStop={() => void c.stop()}
      answering={answering}
      blocked={c.gate ? t(problemCopyOfGate(c.gate)) : undefined}
      placeholder={t(placeholderOf(c.screen.kind))}
      bound={bound}
    />
  )
}

function problemCopyOfGate(g: Gate) {
  return { noAccess: m.noAccessTitle, off: m.offTitle, noModel: m.noModelTitle, noUsable: m.noUsableTitle, expired: m.expiredTitle, lost: m.lostTitle }[g]
}
