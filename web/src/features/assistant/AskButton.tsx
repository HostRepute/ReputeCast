import { useContext } from 'react'
import { Button } from '@/components/ui/Button'
import { useT } from '@/i18n'
import { useCan } from '@/lib/session'
import { AssistantPlace } from './AssistantShell'
import { ml } from './launcher.messages'
import { loadPanel, openAssistant, type AssistantSuggestion } from './store'

/**
 * "Ask the assistant" on a report card, a Send Readiness row or a composer
 * warning (AssistantEntry): opens the panel with the resource suggested as
 * context; nothing is shared until the person attaches it. Shown where the
 * panel has a place, to people who can use AI features.
 */
export function AskAssistant({ resource, about, className }: { resource: Omit<AssistantSuggestion, 'path'>; about?: string; className?: string }) {
  const t = useT()
  if (!useCanAsk()) return null
  const prefetch = () => void loadPanel().catch(() => {})
  return (
    <Button
      size="sm"
      icon="chat"
      className={className}
      aria-label={about ? t(ml.askAbout, { about }) : undefined}
      onPointerEnter={prefetch}
      onFocus={prefetch}
      onClick={() => openAssistant({ suggest: resource })}
    >
      {t(ml.ask)}
    </Button>
  )
}

/** Whether Ask the assistant shows here: the panel has a place, and the person can use AI features. */
export function useCanAsk(): boolean {
  const place = useContext(AssistantPlace)
  const can = useCan('ai:use')
  return place && can
}
