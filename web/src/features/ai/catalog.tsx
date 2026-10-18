import { Link } from '@tanstack/react-router'
import type { ReactNode } from 'react'
import type { Schemas } from '@/api/client'
import { Badge } from '@/components/ui/Badge'
import { Button, ButtonLink } from '@/components/ui/Button'
import { useT, type MessageDescriptor } from '@/i18n'
import { count, money } from '@/lib/format'
import { useWorkspace } from '@/lib/session'
import { aiCapabilities, aiEntryLine, aiEntryName, aiMaintainer, isAIProvider, type AIProvider } from './entries'
import { dollars } from './format'
import { m } from './messages'

type Entry = Schemas['IntegrationEntry']

/** The provider's logo tile: letters, or a server for compatible servers. */
export function AITile({ provider, size }: { provider: string; size?: 'sm' | 'lg' }) {
  const letters: Record<string, string> = { anthropic: 'A', openai: 'O', google: 'G', azure: 'Az' }
  const px = size === 'lg' ? 22 : size === 'sm' ? 14 : 18
  return (
    <span className={`ih-tile${size ? ` ${size}` : ''}`} aria-hidden="true">
      {letters[provider] ?? <ServerIcon size={px} />}
    </span>
  )
}

export function ServerIcon({ size = 18 }: { size?: number }) {
  return (
    <svg width={size} height={size} viewBox="0 0 16 16" fill="none" stroke="var(--ink-nav)" strokeWidth="1.7" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
      <rect x="2" y="2" width="12" height="5" rx="1.5" />
      <rect x="2" y="9" width="12" height="5" rx="1.5" />
      <path d="M5 4.5h.01M5 11.5h.01" strokeWidth="2" />
    </svg>
  )
}

/** "AI assistant settings", at the right of the group's heading. */
export function AIGroupLink() {
  const { slug } = useWorkspace()
  const t = useT()
  return (
    <Link to="/w/$workspace/settings/integrations/ai" params={{ workspace: slug }} className="ih-lnk">
      {t(m.groupSettings)}
    </Link>
  )
}

/** The group's line: an invitation until a model is connected. */
export function aiGroupLine(entries: Entry[], invite: MessageDescriptor): MessageDescriptor {
  return entries.some((e) => e.status === 'connected' || e.status === 'attention') ? m.groupLineConnected : invite
}

/** "One model per workspace: disconnect Anthropic first to switch provider." under the group. */
export function AIGroupNote({ entries }: { entries: Entry[] }) {
  const t = useT()
  const on = entries.find((e) => (e.status === 'connected' || e.status === 'attention') && isAIProvider(e.key))
  if (!on || !isAIProvider(on.key)) return null
  return <p className="k m-0">{t(m.oneModelNote, { provider: t(aiEntryName[on.key]) })}</p>
}

/**
 * One provider's card in the "AI models" group (IntegrationsCatalog,
 * AISettings): the connected one with this month's use and Manage; the
 * others with Connect, which waits while another model is connected.
 */
export function AIModelCard({
  provider,
  status,
  facts: ai,
  canManage,
  own,
  onConnect,
  denied,
}: {
  provider: AIProvider
  status: Entry['status']
  facts?: Schemas['IntegrationAIFacts']
  canManage: boolean
  own?: boolean
  onConnect?: (p: AIProvider) => void
  /** On the AI page, why someone without integrations:manage can't connect. */
  denied?: string
}) {
  const { slug } = useWorkspace()
  const t = useT()
  const connected = status === 'connected' || status === 'attention'
  const other = !connected && Boolean(ai?.connected)
  const name = t(aiEntryName[provider])
  const caps = aiCapabilities(own)[provider]
  const by = connected && ai?.modelName ? t(provider === 'azure' ? m.byDeploymentModel : m.byKeyModel, { model: ai.modelName }) : t(aiMaintainer[provider])
  let fact: ReactNode
  if (connected && ai) {
    fact =
      ai.capKind === 'tokens'
        ? t(m.factUsageTokens, { spent: count(ai.tokens), cap: count(ai.capTokens) })
        : ai.capKind === 'none'
          ? t(m.factUsageNoCap, { spent: dollars(ai.costMicros) })
          : t(m.factUsage, { spent: dollars(ai.costMicros), cap: money(ai.capUsd) })
  } else if (other) {
    fact = t(m.factOneModel)
  } else {
    fact = t(provider === 'compatible' ? m.factNoKey : provider === 'azure' && !own ? m.factAzure : m.factCap)
  }
  const settings = { workspace: slug }
  let action: ReactNode
  if (connected) {
    action = (
      <ButtonLink size="sm" to="/w/$workspace/settings/integrations/ai" params={settings}>
        {canManage ? t(m.manage) : t(m.view)}
      </ButtonLink>
    )
  } else if (other) {
    action = (
      <Button size="sm" disabledReason={t(m.oneModelNote, { provider: ai?.connected ?? '' })}>
        {t(m.connect)}
      </Button>
    )
  } else if (own && !canManage && denied) {
    action = (
      <Button size="sm" disabledReason={denied}>
        {t(m.connect)}
      </Button>
    )
  } else if (onConnect && canManage) {
    action = (
      <Button size="sm" variant={own && provider === 'anthropic' ? 'primary' : undefined} onClick={() => onConnect(provider)}>
        {t(m.connect)}
      </Button>
    )
  } else {
    action = (
      <ButtonLink size="sm" to="/w/$workspace/settings/integrations/ai" params={settings} search={canManage ? { connect: provider } : undefined}>
        {canManage ? t(m.connect) : t(m.view)}
      </ButtonLink>
    )
  }
  const share = Math.min(1, Math.max(0, ai?.share ?? 0))
  return (
    <article className={`card ih-card${connected ? ' inv' : ''}`} aria-label={name}>
      <div className="ih-head">
        <AITile provider={provider} />
        <div className="ih-name">
          <h3>{name}</h3>
          <span className="k">{by}</span>
        </div>
        {status === 'attention' ? (
          <Badge tone="bad" dot>
            {t(m.statusKeyRejected)}
          </Badge>
        ) : connected ? (
          <Badge tone="ok" dot>
            {t(m.statusConnected)}
          </Badge>
        ) : (
          <Badge tone="neutral">{t(m.statusNotConnected)}</Badge>
        )}
      </div>
      <p className="ih-desc">{t(aiEntryLine[provider])}</p>
      <div className="ih-caps">
        {caps.map((c) => (
          <span key={c.id} className="ih-cap">
            {t(c)}
          </span>
        ))}
      </div>
      {connected && ai && ai.capKind !== 'none' ? (
        <div
          role="meter"
          aria-label={t(m.usageBarLabel, { spent: dollars(ai.costMicros), cap: money(ai.capUsd) })}
          aria-valuemin={0}
          aria-valuemax={100}
          aria-valuenow={Math.round(share * 100)}
          className="ai-meter"
        >
          <span style={{ width: `${share * 100}%` }} />
        </div>
      ) : null}
      <div className="ih-ft">
        <span className="k">{fact}</span>
        {action}
      </div>
    </article>
  )
}
