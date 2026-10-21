import { describe, expect, it } from 'vitest'
import { cut, cutTitle, platform, views, withEllipsis } from './cuts'

// The board's notification (PushComposer): where each platform cuts it.
const title = 'Feature flags at scale'
const message = 'How we ship to 2% of accounts first, watch the error budget, and switch a flag off in seconds.'

describe('push previews', () => {
  it('cuts the message where each platform does', () => {
    const v = Object.fromEntries(views(title, message).map((x) => [x.platform.key, x]))
    expect(v.chrome_windows!.message.hidden).toBe('')
    expect(v.chrome_mac!.message.hidden).toBe('and switch a flag off in seconds.')
    expect(withEllipsis(v.chrome_mac!.message.shown, v.chrome_mac!.message.hidden)).toBe('How we ship to 2% of accounts first, watch the error budget…')
    expect(v.safari_mac!.message.hidden).toBe('seconds.')
    expect(withEllipsis(v.safari_mac!.message.shown, v.safari_mac!.message.hidden)).toBe(
      'How we ship to 2% of accounts first, watch the error budget, and switch a flag off in…',
    )
    expect(v.safari_ios!.message.hidden).toBe('')
    for (const x of Object.values(v)) expect(x.title.hidden).toBe('')
  })

  it('cuts a long title on Chrome on Windows', () => {
    const long = 'Feature flags at scale: how we roll out to 2% first'
    const w = cutTitle(long, platform('chrome_windows').titleChars)
    expect(w.hidden).toBe('to 2% first')
    expect(withEllipsis(w.shown, w.hidden)).toBe('Feature flags at scale: how we roll out…')
  })

  it('keeps words whole and handles empty text', () => {
    expect(cut('', 2, 10)).toEqual({ shown: '', hidden: '' })
    expect(cut('supercalifragilistic word', 1, 5)).toEqual({ shown: 'supercalifragilistic', hidden: 'word' })
  })
})
