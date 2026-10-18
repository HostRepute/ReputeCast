import type { MessageDescriptor } from '@/i18n'
import { m } from './messages'

/** The five providers of the "AI models" group, in the catalog's order. */
export type AIProvider = 'anthropic' | 'openai' | 'google' | 'azure' | 'compatible'

export const AI_PROVIDERS: AIProvider[] = ['anthropic', 'openai', 'google', 'azure', 'compatible']

export function isAIProvider(k: string): k is AIProvider {
  return (AI_PROVIDERS as string[]).includes(k)
}

/** The integrations catalog's words for the AI providers (shared.tsx spreads them). */
export const aiEntryName: Record<AIProvider, MessageDescriptor> = {
  anthropic: m.nameAnthropic,
  openai: m.nameOpenai,
  google: m.nameGoogle,
  azure: m.nameAzure,
  compatible: m.nameCompatible,
}

export const aiMaintainer: Record<AIProvider, MessageDescriptor> = {
  anthropic: m.byKey,
  openai: m.byKey,
  google: m.byKey,
  azure: m.byDeployment,
  compatible: m.byCompatible,
}

export const aiEntryLine: Record<AIProvider, MessageDescriptor> = {
  anthropic: m.lineAnthropic,
  openai: m.lineOpenai,
  google: m.lineGoogle,
  azure: m.lineAzure,
  compatible: m.lineCompatible,
}

/** Capability badges; `own` words the network badge as the AI page does. */
export function aiCapabilities(own = false): Record<AIProvider, MessageDescriptor[]> {
  return {
    anthropic: [m.capText, m.capImage, m.capStructured],
    openai: [m.capText, m.capImage, m.capStructured],
    google: [m.capText, m.capImage, m.capStructured],
    azure: [m.capText, m.capImage, m.capRegion],
    compatible: [m.capText, own ? m.capOwnNetwork : m.capNetwork],
  }
}

export const aiCatalogCapabilities = aiCapabilities(false)
