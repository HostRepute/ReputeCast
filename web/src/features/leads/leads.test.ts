import { describe, expect, it } from 'vitest'
import { t } from '@/i18n'
import { refsFrom, triggerKindLabel, triggerRef, type TriggerConfig } from '@/features/automations/graph'
import { sourceLine, sourceName, type Source } from '@/features/growth/sourceText'
import { formsOf, type Connection } from './api'
import { leadTriggerRef } from './automation'
import { connectionStatus } from './LeadAdsPage'

/* Lead ads from Meta (M66): the words the page, the flow editor and Growth use. */

const status = (c: Pick<Connection, 'status' | 'receiving'>) => connectionStatus(t, c)
const receiving = (r: Partial<Connection['receiving']> = {}): Connection['receiving'] => ({ mode: 'webhook', url: 'https://news.acme.io', ...r })

describe('The connection page (IntegrationsLeadAds)', () => {
  it('says how leads arrive in the header badge', () => {
    expect(status({ status: 'connected', receiving: receiving() })).toEqual({ tone: 'ok', label: 'Receiving' })
    expect(status({ status: 'connected', receiving: receiving({ mode: 'polling', reason: 'quiet' }) })).toEqual({ tone: 'warn', label: 'Polling every 15 minutes' })
    expect(status({ status: 'attention', receiving: receiving() })).toEqual({ tone: 'warn', label: 'Needs attention' })
    expect(status({ status: 'disconnected', receiving: receiving({ mode: 'polling' }) })).toEqual({ tone: 'neutral', label: 'Disconnected' })
  })

  it('lists the chosen Pages’ forms in order', () => {
    const form = (id: string, page: string) => ({ id, page }) as Connection['pages'][number]['forms'][number]
    const pages = [
      { id: 'p1', name: 'Acme Inc', forms: [form('f1', 'Acme Inc'), form('f2', 'Acme Inc')] },
      { id: 'p2', name: 'Acme Labs', forms: [form('f3', 'Acme Labs')] },
    ] as Connection['pages']
    expect(formsOf({ pages }).map((f) => f.id)).toEqual(['f1', 'f2', 'f3'])
  })
})

describe('Submitted a lead ad form (AutomationsComplete › 8)', () => {
  const options = { leadForms: [{ id: 'lf1', name: 'Spring webinar signup', page: 'Acme Inc', paused: false }] }
  const refs = refsFrom({ lists: [], forms: [], emails: [], providers: [], fields: [], milestones: [], ...options } as unknown as Parameters<typeof refsFrom>[0])
  const trigger = (leadFormId?: string): TriggerConfig => ({ kind: 'lead_ad_submitted', leadFormId })

  it('names the trigger and what it listens to', () => {
    expect(triggerKindLabel('lead_ad_submitted')).toBe('Submitted a lead ad form')
    expect(triggerRef(trigger(), refs)).toBe('any lead form')
    expect(triggerRef(trigger('lf1'), refs)).toBe('Spring webinar signup · Acme Inc')
    expect(leadTriggerRef(trigger('gone'), refs.leadForms)).toBe('a deleted lead form')
  })
})

describe('Growth source (GrowthSources)', () => {
  const src = (s: Partial<Source>): Source => ({ key: 'k', kind: 'lead_ad', name: '', imported: false, hosts: [], ...s })
  it('reads “Lead ads · {form}” with the form’s kind under it', () => {
    expect(sourceName(src({ name: 'Field Guide download' }))).toBe('Lead ads · Field Guide download')
    expect(sourceLine(src({ name: 'Field Guide download' }))).toBe('Facebook and Instagram lead form')
  })
})
