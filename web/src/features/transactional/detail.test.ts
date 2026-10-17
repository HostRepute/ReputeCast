import { codeSample, highlight } from './codeSamples'
import { markValues, requestJson } from './MessagePage'

describe('the request of a message', () => {
  it('prints small objects and lists on one line, data open', () => {
    const body = {
      template: 'password-reset',
      to: { email: 'sam.rivera@outlook.com', name: 'Sam Rivera' },
      data: { first_name: 'Sam', reset_url: '••••••', expires_in: '30 minutes' },
      tags: ['auth'],
      metadata: { user_id: 'u_48213' },
    }
    expect(requestJson(body)).toBe(
      [
        '{',
        '  "template": "password-reset",',
        '  "to": { "email": "sam.rivera@outlook.com", "name": "Sam Rivera" },',
        '  "data": {',
        '    "first_name": "Sam",',
        '    "reset_url": "••••••",',
        '    "expires_in": "30 minutes"',
        '  },',
        '  "tags": ["auth"],',
        '  "metadata": { "user_id": "u_48213" }',
        '}',
      ].join('\n'),
    )
  })

  it('opens objects too long for one line', () => {
    const out = requestJson({ attachments: [{ filename: 'invoice-INV-2026-0418.pdf', content_type: 'application/pdf', content: '(not stored)' }] })
    expect(out).toContain('"attachments": [\n    {\n      "filename": "invoice-INV-2026-0418.pdf",\n')
  })
})

describe('the rendered content', () => {
  it('marks the values the call filled in, as whole words outside links', () => {
    const html = '<html><head></head><body><p>Hi Sam, Samantha</p><a href="https://x.example/Sam">Sam</a><p>Works for 30 minutes.</p></body></html>'
    const out = markValues(html, { first_name: 'Sam', expires_in: '30 minutes', reset_url: '••••••' })
    expect(out).toContain('Hi <span class="rc-tok" title="first_name">Sam</span>, Samantha')
    expect(out).toContain('<a href="https://x.example/Sam">Sam</a>')
    expect(out).toContain('<span class="rc-tok" title="expires_in">30 minutes</span>')
    expect(out).toContain('.rc-tok{')
  })

  it('leaves the email as it was when nothing matches', () => {
    const html = '<p>Hello</p>'
    expect(markValues(html, { first_name: 'Sam' })).toBe(html)
    expect(markValues(html, undefined)).toBe(html)
  })
})

describe('the cURL sample', () => {
  const vars = [
    { name: 'first_name', type: 'text', required: false, sample: 'Sam', default: 'there', sensitive: false, description: '' },
    { name: 'reset_url', type: 'url', required: true, sample: 'https://app.acme.io/reset/sample-token', default: '', sensitive: true, description: '' },
    { name: 'expires_in', type: 'text', required: true, sample: '30 minutes', default: '', sensitive: false, description: '' },
  ]
  it('reads like the guide: literal values, sensitive links cut, tags and metadata', () => {
    const code = codeSample('curl', {
      endpoint: 'https://reputecast.acme.io/api/v1/workspaces/acme-newsletter/transactional/messages',
      template: 'password-reset',
      variables: vars as never,
      literal: { data: { first_name: 'Sam' }, tags: ['auth'], metadata: { user_id: 'u_48213' } },
    })
    expect(code).toBe(
      [
        '$ curl https://reputecast.acme.io/api/v1/workspaces/acme-newsletter/transactional/messages \\',
        '  -H "Authorization: Bearer $REPUTECAST_KEY" \\',
        '  -H "Idempotency-Key: pwreset-7f3a91" \\',
        '  -H "Content-Type: application/json" \\',
        "  -d '{",
        '    "template": "password-reset",',
        '    "to": { "email": "sam.rivera@outlook.com", "name": "Sam Rivera" },',
        '    "data": {',
        '      "first_name": "Sam",',
        '      "reset_url": "https://app.acme.io/reset/…",',
        '      "expires_in": "30 minutes"',
        '    },',
        '    "tags": ["auth"],',
        '    "metadata": { "user_id": "u_48213" }',
        "  }'",
      ].join('\n'),
    )
    // Highlighting keeps every character.
    expect(highlight(code, 'curl')).toBeTruthy()
  })

  it("escapes a quote in the shell's single-quoted body", () => {
    const code = codeSample('curl', { endpoint: 'https://x.example/messages', template: 'welcome', variables: [], literal: { data: {} } })
    expect(code).toContain('"data": {}')
    const quoted = codeSample('curl', {
      endpoint: 'https://x.example/messages',
      template: 'welcome',
      variables: [{ name: 'company', type: 'text', required: false, sample: "O'Brien & Co", default: '', sensitive: false, description: '' }] as never,
    })
    expect(quoted).toContain(`"company": "O'\\''Brien & Co"`)
  })
})
