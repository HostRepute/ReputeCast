import { describe, expect, it } from 'vitest'
import type { Schemas } from '@/api/client'
import { t } from '@/i18n'
import { statusText, targetText } from './CallLog'
import { keyPermissions } from './ConnectDialog'
import type { ItemLock } from './shared'

const lock = (key: string, kind: 'trigger' | 'action', permissions: string[], sensitive = false): ItemLock => ({
  key,
  kind,
  permissions,
  sensitive,
  locked: false,
  new: false,
  on: !sensitive,
})

const info = {
  order: ['subscribers:create', 'subscribers:edit', 'subscribers:view', 'campaigns:edit', 'campaigns:view', 'transactional:view', 'transactional:send', 'webhooks:manage'],
  includes: { 'subscribers:edit': ['subscribers:view'], 'campaigns:edit': ['campaigns:view'] } as Record<string, string[]>,
}

const items = [
  lock('contact.subscribed', 'trigger', ['subscribers:view']),
  lock('message.bounced', 'trigger', ['transactional:view']),
  lock('upsert_contact', 'action', ['subscribers:create', 'subscribers:edit']),
  lock('send_transactional', 'action', ['transactional:send'], true),
]

describe('keyPermissions', () => {
  it('lists what the switches that are on need, shortest first, with webhooks:manage for triggers', () => {
    const on = new Set(['contact.subscribed', 'upsert_contact'])
    const p = keyPermissions(info, items, on)
    expect(p.short).toEqual(['subscribers:create', 'subscribers:edit', 'webhooks:manage'])
    // subscribers:edit includes subscribers:view, so the key's list leaves it out and says why.
    expect([...p.included]).toEqual([['subscribers:view', 'subscribers:edit']])
    expect(p.sensitive).toEqual([])
  })

  it('flags a sensitive permission once its action is on', () => {
    const p = keyPermissions(info, items, new Set(['send_transactional']))
    expect(p.short).toEqual(['transactional:send'])
    expect(p.sensitive).toEqual(['transactional:send'])
  })

  it('has nothing when nothing is on', () => {
    expect(keyPermissions(info, items, new Set()).short).toEqual([])
  })
})

const call = (target: Schemas['CallTarget'], who: Schemas['IntegrationCallView']['who'] = 'email') =>
  ({ target, who, path: '/api/v1/x', status: 200 }) as Schemas['IntegrationCallView']

describe('targetText', () => {
  it('words a call as the boards do', () => {
    expect(targetText(t, call({ kind: 'send_transactional', template: 'payment-failed', email: 'finance@initech.example' }))).toBe('payment-failed to finance@initech.example')
    expect(targetText(t, call({ kind: 'add_tag', tag: 'customer', email: 'dmitri@fabrikam.example' }))).toBe('customer on dmitri@fabrikam.example')
    expect(targetText(t, call({ kind: 'upsert_contact', email: 'rob@example.net', list: 'Beta Program' }))).toBe('rob@example.net · Beta Program')
    expect(targetText(t, call({ kind: 'test' }))).toBe('Test connection')
  })

  it('words n8n checking its webhook like turning a trigger on', () => {
    const endpoint = 'https://n8n.acme.io/webhook/7c1e…/reputecast'
    expect(targetText(t, call({ kind: 'hook', event: 'contact.unsubscribed', endpoint }))).toBe(`contact.unsubscribed → ${endpoint}`)
  })

  it('hides addresses from members who can not see contacts, and erased ones', () => {
    expect(targetText(t, call({ kind: 'add_tag', tag: 'customer' }, 'contact'))).toBe('customer on a contact')
    expect(targetText(t, call({ kind: 'upsert_contact', list: 'Beta Program' }, 'erased'))).toBe('Erased contact · Beta Program')
  })
})

describe('statusText', () => {
  it('gives HTTP reason phrases', () => {
    expect(statusText(422)).toBe('Unprocessable Content')
    expect(statusText(502)).toBe('Bad Gateway')
    expect(statusText(299)).toBe('')
  })
})
