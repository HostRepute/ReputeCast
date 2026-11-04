import { translate } from '@/i18n/define'
import { m } from './format.messages'

/** French (docs/i18n/glossary-fr.md). */
export default translate(m, {
  never: 'jamais',
  neverTitle: 'Jamais',
  now: 'à l’instant',
  minutesAgo: 'il y a {n} min',
  hoursAgo: 'il y a {n} h',
  yesterday: 'hier',
  daysAgo: '{n, plural, one {il y a # jour} other {il y a # jours}}',
  inMinutes: 'dans {n} min',
  inHoursMinutes: 'dans {h} h {m} min',
  inDaysHours: '{days, plural, one {dans # jour et {h} h} other {dans # jours et {h} h}}',
  inHours: 'dans {n} h',
  inDays: '{n, plural, one {dans # jour} other {dans # jours}}',
  expired: 'expiré',
  minutes: '{n} min',
  hours: '{h} h',
  hoursMinutes: '{h} h {m} min',
  perSecond: '{n}/s',
  spanHours: '{n, plural, one {# heure} other {# heures}}',
  spanMinutes: '{n, plural, one {# minute} other {# minutes}}',
  countdownMinutes: 'dans {n} minutes',
  countdownOneMinute: 'dans 1 minute',
  countdownSeconds: '{n, plural, one {dans # seconde} other {dans # secondes}}',
  todayAt: 'Aujourd’hui à {time}',
  yesterdayAt: 'hier à {time}',
  thisMorning: 'ce matin',
  thisAfternoon: 'cet après-midi',
  thisEvening: 'ce soir',
  onDate: 'le {date}',
})
