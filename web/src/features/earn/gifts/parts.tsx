import { inZone } from '@/lib/format'
import { day } from '../format'

/** The gift mark of the boards (a box with a bow). */
export function GiftIcon({ size = 16, className }: { size?: number; className?: string }) {
  return (
    <svg className={className} width={size} height={size} viewBox="0 0 16 16" fill="none" stroke="currentColor" strokeWidth={1.6} strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
      <rect x="2.5" y="6.5" width="11" height="7.5" rx="1" />
      <path d="M1.8 4.2h12.4v2.3H1.8zM8 4.2V14M8 4.2c-1.2-2.3-4-2.6-4-.9 0 .9 1.6.9 4 .9zM8 4.2c1.2-2.3 4-2.6 4-.9 0 .9-1.6.9-4 .9z" />
    </svg>
  )
}

/** A moment's calendar day in a time zone ("2026-12-24"). */
function ymdIn(d: Date, tz: string | undefined): string {
  try {
    return new Intl.DateTimeFormat('en-CA', { timeZone: tz || undefined, year: 'numeric', month: '2-digit', day: '2-digit' }).format(d)
  } catch {
    return ''
  }
}

/** A moment in the buyer's time zone: "24 Dec" and "08:00". */
export function giftWhen(at: string, tz: string | undefined): { date: string; time: string } {
  const ymd = ymdIn(new Date(at), tz)
  const date = /^\d{4}-\d{2}-\d{2}$/.test(ymd) ? day(`${ymd}T12:00:00Z`) : day(at)
  return { date, time: inZone(at, tz).time }
}
