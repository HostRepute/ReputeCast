import { ApiError, type Problem } from '@/api/client'
import { fieldErrors, formProblems, parseHeaderLines, previewText, webhookAuths } from './http-adapter-model'

describe('parseHeaderLines', () => {
  it('splits Name: value lines at the first colon and skips blank lines', () => {
    expect(parseHeaderLines('Authorization: Bearer {{ secrets.api_key }}\n\n  X-Callback : https://hooks.example.com/a  \n')).toEqual({
      headers: [
        { name: 'Authorization', value: 'Bearer {{ secrets.api_key }}' },
        { name: 'X-Callback', value: 'https://hooks.example.com/a' },
      ],
    })
    expect(parseHeaderLines('')).toEqual({ headers: [] })
    expect(parseHeaderLines('X-Empty:')).toEqual({ headers: [{ name: 'X-Empty', value: '' }] })
  })

  it('names the line that is not a header', () => {
    expect(parseHeaderLines('Accept: application/json\nBearer abc').error).toBe('Line 2: write each header as Name: value.')
    expect(parseHeaderLines(': no name').error).toBe('Line 1: write each header as Name: value.')
  })
})

describe('fieldErrors', () => {
  it('keeps the first message per location', () => {
    const problem = {
      status: 422,
      title: 'Unprocessable Entity',
      errors: [
        { location: 'body.headers', message: 'Header names are letters, digits and dashes.' },
        { location: 'body.headers', message: 'second' },
        { location: 'body.messageIdPath', message: 'The message id path must start with $.' },
        { message: 'no location' },
      ],
    } as Problem
    expect(fieldErrors(new ApiError(422, problem))).toEqual({
      'body.headers': 'Header names are letters, digits and dashes.',
      'body.messageIdPath': 'The message id path must start with $.',
    })
    expect(fieldErrors(new TypeError('offline'))).toEqual({})
  })

  it('counts a location inside a list for the list', () => {
    const problem = { status: 422, title: 'Unprocessable Entity', errors: [{ location: 'body.headers[1].name', message: 'expected length >= 1' }] } as Problem
    expect(fieldErrors(new ApiError(422, problem))).toEqual({ 'body.headers': 'expected length >= 1' })
  })
})

describe('formProblems', () => {
  const shown = new Set(['body.endpoint', 'body.headers'])

  it('keeps fields the form shows and says the rest under it', () => {
    const problem = {
      status: 422,
      title: 'Unprocessable Entity',
      detail: 'validation failed',
      errors: [
        { location: 'body.endpoint', message: 'Use https://.' },
        { location: 'body.webhookHeader', message: 'Header names are letters, digits and dashes.' },
        { location: 'body.name', message: 'Choose another name.' },
      ],
    } as Problem
    const p = formProblems(new ApiError(422, problem), shown)
    expect(p.fields['body.endpoint']).toBe('Use https://.')
    expect(p.general).toBe('Header names are letters, digits and dashes. Choose another name.')
    expect(formProblems(new ApiError(422, problem), new Set([...shown, 'body.webhookHeader', 'body.name'])).general).toBe('')
  })

  it('falls back to the problem when it names no field', () => {
    const problem = { status: 409, title: 'Conflict', detail: 'Example ESP is already installed.' } as Problem
    expect(formProblems(new ApiError(409, problem), shown)).toEqual({ fields: {}, general: 'Example ESP is already installed.' })
    expect(formProblems(new TypeError('Failed to fetch'), shown).general).toBe("Can't reach the server. Check your connection; nothing was lost.")
  })
})

describe('previewText', () => {
  it('prints the request as it goes on the wire', () => {
    expect(
      previewText({ method: 'POST', url: 'https://api.example-esp.com/v1/messages', headers: [{ name: 'Authorization', value: 'Bearer ••••' }], body: '{\n  "to": "ada@example.com"\n}' }),
    ).toBe('POST https://api.example-esp.com/v1/messages\nAuthorization: Bearer ••••\n\n{\n  "to": "ada@example.com"\n}')
    expect(previewText({ method: 'GET', url: 'https://api.example-esp.com/v1/ping', headers: [], body: '' })).toBe('GET https://api.example-esp.com/v1/ping')
  })
})

describe('webhookAuths', () => {
  it('asks for a header only where the mapping uses one', () => {
    expect(webhookAuths.filter((a) => a.header).map((a) => a.value)).toEqual(['hmac', 'header'])
  })
})
