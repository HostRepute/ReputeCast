import { ApiError, type Problem } from '@/api/client'
import { fieldErrors, parseHeaderLines, previewText, webhookAuths } from './http-adapter-model'

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
