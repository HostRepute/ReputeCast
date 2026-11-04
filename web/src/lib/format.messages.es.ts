import { translate } from '@/i18n/define'
import { m } from './format.messages'

/** Spanish (docs/i18n/glossary-es.md). */
export default translate(m, {
  never: 'nunca',
  neverTitle: 'Nunca',
  now: 'ahora',
  minutesAgo: 'hace {n} min',
  hoursAgo: 'hace {n} h',
  yesterday: 'ayer',
  daysAgo: '{n, plural, one {hace # día} other {hace # días}}',
  inMinutes: 'dentro de {n} min',
  inHoursMinutes: 'dentro de {h} h {m} min',
  inDaysHours: '{days, plural, one {dentro de # día y {h} h} other {dentro de # días y {h} h}}',
  inHours: 'dentro de {n} h',
  inDays: '{n, plural, one {dentro de # día} other {dentro de # días}}',
  expired: 'caducada',
  minutes: '{n} min',
  hours: '{h} h',
  hoursMinutes: '{h} h {m} min',
  perSecond: '{n}/s',
  spanHours: '{n, plural, one {# hora} other {# horas}}',
  spanMinutes: '{n, plural, one {# minuto} other {# minutos}}',
  countdownMinutes: 'dentro de {n} minutos',
  countdownOneMinute: 'dentro de 1 minuto',
  countdownSeconds: '{n, plural, one {dentro de # segundo} other {dentro de # segundos}}',
  todayAt: 'Hoy a las {time}',
  yesterdayAt: 'ayer a las {time}',
  thisMorning: 'esta mañana',
  thisAfternoon: 'esta tarde',
  thisEvening: 'esta noche',
  onDate: 'el {date}',
})
