import { translate } from '@/i18n/define'
import { m } from './format.messages'

/** German (docs/i18n/glossary-de.md). */
export default translate(m, {
  never: 'nie',
  neverTitle: 'Nie',
  now: 'jetzt',
  minutesAgo: 'vor {n} Min.',
  hoursAgo: 'vor {n} Std.',
  yesterday: 'gestern',
  daysAgo: '{n, plural, one {vor # Tag} other {vor # Tagen}}',
  inMinutes: 'in {n} Min.',
  inHoursMinutes: 'in {h} Std. {m} Min.',
  inDaysHours: '{days, plural, one {in # Tag {h} Std.} other {in # Tagen {h} Std.}}',
  inHours: 'in {n} Std.',
  inDays: '{n, plural, one {in # Tag} other {in # Tagen}}',
  expired: 'abgelaufen',
  minutes: '{n} Min.',
  hours: '{h} Std.',
  hoursMinutes: '{h} Std. {m} Min.',
  perSecond: '{n}/s',
  spanHours: '{n, plural, one {# Stunde} other {# Stunden}}',
  spanMinutes: '{n, plural, one {# Minute} other {# Minuten}}',
  countdownMinutes: 'in {n} Minuten',
  countdownOneMinute: 'in 1 Minute',
  countdownSeconds: '{n, plural, one {in # Sekunde} other {in # Sekunden}}',
  todayAt: 'Heute um {time}',
  yesterdayAt: 'gestern um {time}',
  thisMorning: 'heute Morgen',
  thisAfternoon: 'heute Nachmittag',
  thisEvening: 'heute Abend',
  onDate: 'am {date}',
})
