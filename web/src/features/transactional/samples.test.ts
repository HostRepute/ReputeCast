import { renderToStaticMarkup } from 'react-dom/server'
import { describe, expect, it } from 'vitest'
import type { Schemas } from '@/api/client'
import { codeSample, highlight } from './codeSamples'

type Variable = Schemas['TransactionalVariable']

const v = (name: string, sample = '', type = 'string'): Variable => ({ name, sample, type, required: true, sensitive: false, description: '' }) as Variable
const vars = [v('first_name', 'Sam'), v('reset_url', 'https://app.acme.io/reset/abc'), v('expires_in', '30 minutes')]
const endpoint = 'https://reputecast.acme.io/api/v1/workspaces/acme-newsletter/transactional/messages'
const input = { endpoint, template: 'password-reset', variables: vars }

describe('SDK samples (M64 G-34)', () => {
  it('are the default for Node.js, Python and Go', () => {
    expect(codeSample('node', input)).toContain('import { createClient, unwrap } from "@reputecast/client";')
    expect(codeSample('python', input)).toContain('from reputecast import ReputeCast, unwrap')
    expect(codeSample('go', input)).toContain('reputecast.New("https://reputecast.acme.io", os.Getenv("REPUTECAST_KEY"))')
    // cURL and PHP have no SDK: the style changes nothing.
    expect(codeSample('curl', { ...input, style: 'sdk' })).toBe(codeSample('curl', { ...input, style: 'http' }))
    expect(codeSample('php', { ...input, style: 'sdk' })).toBe(codeSample('php', { ...input, style: 'http' }))
  })

  it('call the instance with the workspace as a path parameter', () => {
    const node = codeSample('node', input)
    expect(node).toContain('createClient({ baseUrl: "https://reputecast.acme.io", apiKey: process.env.REPUTECAST_KEY })')
    expect(node).toContain('rc.POST("/api/v1/workspaces/{workspace}/transactional/messages", {')
    expect(node).toContain('path: { workspace: "acme-newsletter" },')
    expect(node).toContain('header: { "Idempotency-Key": `pwreset-${user.id}-${token.id}` },')
    expect(node).toContain('        first_name: user.firstName,\n        reset_url: resetLink,\n        expires_in: "30 minutes",')
    const py = codeSample('python', input)
    expect(py).toContain('rc = ReputeCast(os.environ["REPUTECAST_KEY"], base_url="https://reputecast.acme.io")')
    expect(py).toContain('    workspace="acme-newsletter",\n    idempotency_key=f"pwreset-{user.id}-{token.id}",')
    expect(py).toContain('            "first_name": user.first_name,\n            "reset_url": reset_link,')
    const go = codeSample('go', input)
    expect(go).toContain('key := "pwreset-" + user.ID + "-" + token.ID')
    expect(go).toContain('rc.SendTransactionalMessageWithResponse(ctx, "acme-newsletter",')
    expect(go).toContain('\t\t\t"first_name": user.FirstName,\n\t\t\t"reset_url":  ResetLink,')
  })

  it('keep a sub-path install and an event key for other mail', () => {
    const s = codeSample('node', { endpoint: 'https://example.com/mail/api/v1/workspaces/north-wind/transactional/messages', template: 'invoice-paid', variables: [] })
    expect(s).toContain('baseUrl: "https://example.com/mail"')
    expect(s).toContain('workspace: "north-wind"')
    expect(s).toContain('`invoicepaid-${user.id}-${event.id}`')
    expect(s).toContain('      data: {},')
    expect(codeSample('go', { ...input, variables: [] })).not.toContain('Data:')
  })

  it('leave the install line out of the short sample', () => {
    expect(codeSample('node', input)).toMatch(/^\/\/ npm install @reputecast\/client\n/)
    expect(codeSample('python', input)).toMatch(/^# pip install reputecast\n/)
    expect(codeSample('go', input)).toMatch(/^\/\/ go get github\.com\/hostrepute\/reputecast\/sdk\/client-go\n/)
    for (const lang of ['node', 'python', 'go'] as const) {
      expect(codeSample(lang, { ...input, compact: true })).not.toMatch(/install|go get/)
    }
  })

  it('switch back to the plain request', () => {
    const http = codeSample('node', { ...input, style: 'http' })
    expect(http).toMatch(/^const res = await fetch\("https:\/\/reputecast\.acme\.io\/api\/v1\/workspaces\/acme-newsletter\/transactional\/messages", \{/)
    expect(http).not.toContain('@reputecast/client')
    expect(codeSample('python', { ...input, style: 'http' })).toMatch(/^import os, requests/)
    expect(codeSample('go', { ...input, style: 'http' })).toMatch(/^body, _ := json\.Marshal/)
  })

  it('colour imports as keywords and comments grey', () => {
    const html = renderToStaticMarkup(highlight(codeSample('node', input), 'node'))
    expect(html).toContain('<span class="cm">// npm install @reputecast/client</span>\n<span class="c5">import</span> { createClient, unwrap } <span class="c5">from</span> <span class="c2">&quot;@reputecast/client&quot;</span>;')
    const py = renderToStaticMarkup(highlight(codeSample('python', input), 'python'))
    expect(py).toContain('<span class="cm"># msg.id → &quot;0192f1c4-…&quot;, msg.status → &quot;queued&quot;</span>')
  })
})
