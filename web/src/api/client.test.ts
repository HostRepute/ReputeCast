import { setCatalogLoader, setLocale } from '@/i18n'
import { ApiError, problemSentence } from './client'
import vi from './messages.vi'

describe('problem sentences', () => {
  it('keeps ReputeCast details as they are', () => {
    expect(problemSentence(404, { detail: 'Campaign not found.' })).toBe('Campaign not found.')
    expect(problemSentence(409, { detail: 'Postmark still carries System. Change the routing first, then disconnect it.' })).toBe(
      'Postmark still carries System. Change the routing first, then disconnect it.',
    )
  })

  it("words the router's unhandled-error fallback in the product voice", () => {
    expect(new ApiError(500, { type: 'about:blank', status: 500, title: 'Internal Server Error', detail: 'unexpected error occurred' }).message).toBe(
      'ReputeCast hit an unexpected error; the details are in the server log.',
    )
    const withRef = { type: 'about:blank', status: 500, title: 'Internal Server Error', detail: 'unexpected error occurred', code: 'internal_error', meta: { reference: 'err_7KQ2M4XA' } }
    expect(new ApiError(500, withRef).message).toBe('ReputeCast hit an unexpected error; the server log has the details under err_7KQ2M4XA.')
  })

  it('words it in the member’s language', async () => {
    setCatalogLoader('vi', async () => vi)
    await setLocale('vi')
    try {
      expect(problemSentence(500, { detail: 'unexpected error occurred', meta: { reference: 'err_7KQ2M4XA' } })).toBe(
        'ReputeCast gặp lỗi không mong muốn; chi tiết đã được ghi vào nhật ký máy chủ với mã err_7KQ2M4XA.',
      )
    } finally {
      await setLocale('en')
    }
  })

  it('makes a bare phrase a sentence', () => {
    expect(problemSentence(502, { title: 'Bad Gateway' })).toBe('Bad Gateway.')
    expect(problemSentence(400, { detail: 'request body is required' })).toBe('Request body is required.')
  })
})
