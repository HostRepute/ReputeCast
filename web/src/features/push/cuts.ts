/**
 * How each platform shows a notification, and where it cuts the text. The
 * same table and rules as the server's pre-send check (internal/push
 * review.go: Platforms, Cut and cutTitle), so a preview and its check agree.
 */

export type PlatformKey = 'chrome_windows' | 'chrome_mac' | 'safari_mac' | 'chrome_android' | 'safari_ios'

export interface Platform {
  key: PlatformKey
  /** About how many characters of the title fit on one line. */
  titleChars: number
  /** How many lines of the message it shows, of about lineChars each. */
  messageLines: number
  lineChars: number
  image: boolean
  buttons: boolean
  desktop: boolean
}

export const PLATFORMS: readonly Platform[] = [
  { key: 'chrome_windows', titleChars: 40, messageLines: 3, lineChars: 47, image: true, buttons: true, desktop: true },
  { key: 'chrome_mac', titleChars: 34, messageLines: 2, lineChars: 31, image: false, buttons: true, desktop: true },
  { key: 'safari_mac', titleChars: 40, messageLines: 2, lineChars: 43, image: false, buttons: false, desktop: true },
  { key: 'chrome_android', titleChars: 38, messageLines: 1, lineChars: 40, image: true, buttons: true, desktop: false },
  { key: 'safari_ios', titleChars: 40, messageLines: 4, lineChars: 44, image: false, buttons: false, desktop: false },
]

export function platform(key: PlatformKey): Platform {
  return PLATFORMS.find((p) => p.key === key)!
}

/** What a platform shows of a text: lines of whole words up to its width, and the rest hidden. */
export function cut(text: string, lines: number, width: number): { shown: string; hidden: string } {
  const words = text.trim().split(/\s+/).filter(Boolean)
  if (!words.length) return { shown: '', hidden: '' }
  let line = 1
  let used = 0
  for (let i = 0; i < words.length; i++) {
    const n = [...words[i]!].length
    let need = used > 0 ? n + 1 : n
    if (used + need > width && used > 0) {
      line++
      used = 0
      need = n
    }
    if (line > lines) return { shown: words.slice(0, i).join(' '), hidden: words.slice(i).join(' ') }
    used += need
  }
  return { shown: words.join(' '), hidden: '' }
}

/** What a platform shows of a title (one line of about `chars` characters). */
export function cutTitle(title: string, chars: number): { shown: string; hidden: string } {
  const r = [...title]
  if (r.length <= chars) return { shown: title, hidden: '' }
  return { shown: r.slice(0, chars).join('').trim(), hidden: r.slice(chars).join('').trim() }
}

/** A cut text as the platform draws it: the shown part and an ellipsis, without a dangling comma. */
export function withEllipsis(shown: string, hidden: string): string {
  if (!hidden) return shown
  return `${shown.replace(/[\s,;:–—-]+$/, '')}…`
}

/** Each platform's view of a notification: its title and message, cut. */
export interface PlatformView {
  platform: Platform
  title: { shown: string; hidden: string }
  message: { shown: string; hidden: string }
}

export function views(title: string, message: string): PlatformView[] {
  return PLATFORMS.map((p) => ({
    platform: p,
    title: cutTitle(title, p.titleChars),
    message: cut(message, p.messageLines, p.lineChars),
  }))
}

/** "+1 hour", the expiries the composer offers, in minutes. */
export const EXPIRIES = [60, 240, 720, 1440, 40320] as const
