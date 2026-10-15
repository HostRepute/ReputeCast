import { describe, expect, it } from 'vitest'
import { apiSample, messagesUrl, missingAnswer, type SampleInput } from './samples'

const input: SampleInput = {
  origin: 'https://reputecast.acme.io/',
  workspace: 'acme-newsletter',
  template: 'password-reset',
  to: { email: 'sam.rivera@outlook.com', name: 'Sam Rivera' },
  data: { first_name: "Sam O'Neil", reset_url: 'https://app.acme.io/reset/x', expires_in: '30 minutes', count: 2, admin: false, extra: null, items: [], meta: {} },
}

describe('API samples', () => {
  it('posts to the messages endpoint of the workspace', () => {
    expect(messagesUrl(input.origin, input.workspace)).toBe('https://reputecast.acme.io/api/v1/workspaces/acme-newsletter/transactional/messages')
  })
  it('writes a cURL call whose JSON survives the shell quotes', () => {
    const s = apiSample('curl', input)
    expect(s).toContain('curl https://reputecast.acme.io/api/v1/workspaces/acme-newsletter/transactional/messages \\')
    expect(s).toContain('-H "Authorization: Bearer $REPUTECAST_KEY"')
    expect(s).toContain('-H "Idempotency-Key: password-reset-7f3a91"')
    expect(s).toContain(`"first_name": "Sam O'\\''Neil"`)
    expect(s).toContain('"template": "password-reset"')
  })
  it('writes Python, Go and PHP literals', () => {
    const py = apiSample('python', input)
    expect(py).toContain('"admin": False')
    expect(py).toContain('"extra": None')
    const go = apiSample('go', input)
    expect(go).toContain('map[string]any{')
    expect(go).toContain('"items": []any{},')
    expect(go).toContain('"extra": nil,')
    const php = apiSample('php', input)
    expect(php).toContain("'first_name' => 'Sam O\\'Neil'")
    expect(php).toContain("'meta' => (object) []")
    expect(apiSample('node', input)).toContain('Authorization: `Bearer ${process.env.REPUTECAST_KEY}`')
  })
  it("words the missing_variables answer as the API does", () => {
    expect(JSON.parse(missingAnswer('password-reset', 3, ['reset_url', 'expires_in']))).toEqual({
      title: 'Missing variables',
      status: 422,
      code: 'missing_variables',
      detail: 'password-reset v3 needs reset_url and expires_in. Add them to data and call again.',
      missing: ['reset_url', 'expires_in'],
    })
  })
})
