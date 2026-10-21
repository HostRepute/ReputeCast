import { $api, type Schemas } from '@/api/client'
import type { useT } from '@/i18n'
import { useWorkspace } from '@/lib/session'
import { m } from './automation.messages'
import { measure } from './measure'

/** The rules' locked parts of a step's text: the brand prefix, the opt-out text and the short-link domain. */
export interface SmsTextParts {
  prefix: string
  optOut: string
  shortDomain: string
}

/** The rules' parts for the canvas's counters: one request for every Send SMS node. */
export function useSmsTextParts(enabled: boolean): SmsTextParts | undefined {
  const { slug } = useWorkspace()
  const q = $api.useQuery('get', '/api/v1/workspaces/{workspace}/sms/rules', { params: { path: { workspace: slug } } }, { enabled, staleTime: 60_000, retry: false })
  const r = q.data
  return r ? { prefix: r.brandPrefix, optOut: r.optOutInSteps ? r.optOutText : '', shortDomain: r.shortDomain.domain } : undefined
}

/** The canvas node's lines for a Send SMS step: its length as sent, and what happens outside quiet hours. */
export function smsNodeLines(s: Schemas['AutomationSMSStep'] | undefined, t: ReturnType<typeof useT>, parts?: SmsTextParts): string[] {
  if (!s?.body) return []
  const c = measure(parts?.prefix ?? '', s.body, s.noOptOut ? '' : (parts?.optOut ?? ''), parts?.shortDomain ?? '').count
  return [t(m.nodeLength, { characters: c.characters, segments: Math.max(1, c.segments) }), s.outside === 'skip' ? t(m.nodeSkip) : t(m.nodeWait)]
}
