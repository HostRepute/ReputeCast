/**
 * Region formats: how dates, times and numbers are written. "ReputeCast
 * standard" is the app's own (Tue 14 Oct 2026 · 09:00 · 48,213 · 99.1% ·
 * $2.84); the others are a snapshot of Unicode CLDR, kept here rather than
 * read from the browser's Intl so every browser (and the server, which has
 * the same tables in internal/i18n/region.go) writes a date the same way.
 * internal/i18n/testdata/regions.json holds the vectors both sides test.
 */
export type RegionFormat = 'standard' | 'en-US' | 'vi-VN' | 'de-DE' | 'en-GB' | 'es-ES' | 'fr-FR' | 'pt-BR'
export type HourCycle = 'h23' | 'h12'
export type WeekStart = 'monday' | 'sunday'

interface RegionData {
  /** The region format's name in its own language. */
  readonly name: string
  readonly decimal: string
  readonly group: string
  /** Digits before grouping starts: 1, or 2 for Spanish (4821, 48.213). */
  readonly minGroup: number
  readonly percent: string
  readonly currency: string
  readonly months: readonly string[]
  readonly monthsLong: readonly string[]
  /** Sunday first. */
  readonly days: readonly string[]
  readonly daysLong: readonly string[]
  readonly am: string
  readonly pm: string
  /** Patterns: EEEE/EEE weekday, d day, MMMM/MMM month, y year, HH/H/h hour, mm, ss, a, 'literal'. */
  readonly date: string
  readonly shortDate: string
  readonly weekdayDate: string
  readonly longDate: string
  readonly dayMonth: string
  readonly monthDay: string
  /** Month and year: Oct 2026 (CLDR yMMM). */
  readonly monthYear: string
  /** October 2026 (CLDR yMMMM). */
  readonly monthYearLong: string
  readonly time24: string
  readonly time12: string
  readonly hourCycle: HourCycle
  readonly weekStart: WeekStart
}

const NBSP = ' '
const enMonths = ['Jan', 'Feb', 'Mar', 'Apr', 'May', 'Jun', 'Jul', 'Aug', 'Sep', 'Oct', 'Nov', 'Dec']
const enMonthsLong = ['January', 'February', 'March', 'April', 'May', 'June', 'July', 'August', 'September', 'October', 'November', 'December']
const enDays = ['Sun', 'Mon', 'Tue', 'Wed', 'Thu', 'Fri', 'Sat']
const enDaysLong = ['Sunday', 'Monday', 'Tuesday', 'Wednesday', 'Thursday', 'Friday', 'Saturday']

const REGIONS: Record<RegionFormat, RegionData> = {
  standard: {
    name: 'ReputeCast standard', decimal: '.', group: ',', minGroup: 1, percent: '#%', currency: '$#',
    months: enMonths, monthsLong: enMonthsLong, days: enDays, daysLong: enDaysLong, am: 'AM', pm: 'PM',
    date: 'MMM d, y', shortDate: 'EEE d MMM', weekdayDate: 'EEE d MMM y', longDate: 'EEEE d MMM y', dayMonth: 'd MMM', monthDay: 'MMM d', monthYear: 'MMM y', monthYearLong: 'MMMM y',
    time24: 'HH:mm', time12: 'h:mm a', hourCycle: 'h23', weekStart: 'monday',
  },
  'en-US': {
    name: 'English (United States)', decimal: '.', group: ',', minGroup: 1, percent: '#%', currency: '$#',
    months: enMonths, monthsLong: enMonthsLong, days: enDays, daysLong: enDaysLong, am: 'AM', pm: 'PM',
    date: 'MMM d, y', shortDate: 'EEE, MMM d', weekdayDate: 'EEE, MMM d, y', longDate: 'EEEE, MMM d, y', dayMonth: 'MMM d', monthDay: 'MMM d', monthYear: 'MMM y', monthYearLong: 'MMMM y',
    time24: 'HH:mm', time12: 'h:mm a', hourCycle: 'h12', weekStart: 'sunday',
  },
  'vi-VN': {
    name: 'Tiếng Việt (Việt Nam)', decimal: ',', group: '.', minGroup: 1, percent: '#%', currency: `#${NBSP}US$`,
    months: ['thg 1', 'thg 2', 'thg 3', 'thg 4', 'thg 5', 'thg 6', 'thg 7', 'thg 8', 'thg 9', 'thg 10', 'thg 11', 'thg 12'],
    monthsLong: ['tháng 1', 'tháng 2', 'tháng 3', 'tháng 4', 'tháng 5', 'tháng 6', 'tháng 7', 'tháng 8', 'tháng 9', 'tháng 10', 'tháng 11', 'tháng 12'],
    // The abbreviations the design uses ("Th 3, 14 thg 10"); CLDR 48 moved to "Thứ 3".
    days: ['CN', 'Th 2', 'Th 3', 'Th 4', 'Th 5', 'Th 6', 'Th 7'],
    daysLong: ['Chủ Nhật', 'Thứ Hai', 'Thứ Ba', 'Thứ Tư', 'Thứ Năm', 'Thứ Sáu', 'Thứ Bảy'],
    am: 'SA', pm: 'CH',
    date: 'd MMM, y', shortDate: 'EEE, d MMM', weekdayDate: 'EEE, d MMM, y', longDate: 'EEEE, d MMMM, y', dayMonth: 'd MMM', monthDay: 'd MMM', monthYear: 'MMM y', monthYearLong: "MMMM 'năm' y",
    time24: 'HH:mm', time12: 'h:mm a', hourCycle: 'h23', weekStart: 'monday',
  },
  'de-DE': {
    name: 'Deutsch (Deutschland)', decimal: ',', group: '.', minGroup: 1, percent: `#${NBSP}%`, currency: `#${NBSP}$`,
    months: ['Jan.', 'Feb.', 'März', 'Apr.', 'Mai', 'Juni', 'Juli', 'Aug.', 'Sept.', 'Okt.', 'Nov.', 'Dez.'],
    monthsLong: ['Januar', 'Februar', 'März', 'April', 'Mai', 'Juni', 'Juli', 'August', 'September', 'Oktober', 'November', 'Dezember'],
    days: ['So.', 'Mo.', 'Di.', 'Mi.', 'Do.', 'Fr.', 'Sa.'],
    daysLong: ['Sonntag', 'Montag', 'Dienstag', 'Mittwoch', 'Donnerstag', 'Freitag', 'Samstag'],
    am: 'AM', pm: 'PM',
    date: 'd. MMM y', shortDate: 'EEE, d. MMM', weekdayDate: 'EEE, d. MMM y', longDate: 'EEEE, d. MMM y', dayMonth: 'd. MMM', monthDay: 'd. MMM', monthYear: 'MMM y', monthYearLong: 'MMMM y',
    time24: 'HH:mm', time12: 'h:mm a', hourCycle: 'h23', weekStart: 'monday',
  },
  'en-GB': {
    name: 'English (United Kingdom)', decimal: '.', group: ',', minGroup: 1, percent: '#%', currency: 'US$#',
    months: ['Jan', 'Feb', 'Mar', 'Apr', 'May', 'Jun', 'Jul', 'Aug', 'Sept', 'Oct', 'Nov', 'Dec'], monthsLong: enMonthsLong,
    days: enDays, daysLong: enDaysLong, am: 'am', pm: 'pm',
    date: 'd MMM y', shortDate: 'EEE d MMM', weekdayDate: 'EEE, d MMM y', longDate: 'EEEE, d MMM y', dayMonth: 'd MMM', monthDay: 'd MMM', monthYear: 'MMM y', monthYearLong: 'MMMM y',
    time24: 'HH:mm', time12: 'h:mm a', hourCycle: 'h23', weekStart: 'monday',
  },
  'es-ES': {
    name: 'Español (España)', decimal: ',', group: '.', minGroup: 2, percent: `#${NBSP}%`, currency: `#${NBSP}US$`,
    months: ['ene', 'feb', 'mar', 'abr', 'may', 'jun', 'jul', 'ago', 'sept', 'oct', 'nov', 'dic'],
    monthsLong: ['enero', 'febrero', 'marzo', 'abril', 'mayo', 'junio', 'julio', 'agosto', 'septiembre', 'octubre', 'noviembre', 'diciembre'],
    days: ['dom', 'lun', 'mar', 'mié', 'jue', 'vie', 'sáb'],
    daysLong: ['domingo', 'lunes', 'martes', 'miércoles', 'jueves', 'viernes', 'sábado'],
    am: 'a. m.', pm: 'p. m.',
    date: 'd MMM y', shortDate: 'EEE, d MMM', weekdayDate: 'EEE, d MMM y', longDate: "EEEE, d 'de' MMM 'de' y", dayMonth: 'd MMM', monthDay: 'd MMM', monthYear: 'MMM y', monthYearLong: "MMMM 'de' y",
    time24: 'H:mm', time12: 'h:mm a', hourCycle: 'h23', weekStart: 'monday',
  },
  'fr-FR': {
    name: 'Français (France)', decimal: ',', group: ' ', minGroup: 1, percent: `#${NBSP}%`, currency: `#${NBSP}$US`,
    months: ['janv.', 'févr.', 'mars', 'avr.', 'mai', 'juin', 'juil.', 'août', 'sept.', 'oct.', 'nov.', 'déc.'],
    monthsLong: ['janvier', 'février', 'mars', 'avril', 'mai', 'juin', 'juillet', 'août', 'septembre', 'octobre', 'novembre', 'décembre'],
    days: ['dim.', 'lun.', 'mar.', 'mer.', 'jeu.', 'ven.', 'sam.'],
    daysLong: ['dimanche', 'lundi', 'mardi', 'mercredi', 'jeudi', 'vendredi', 'samedi'],
    am: 'AM', pm: 'PM',
    date: 'd MMM y', shortDate: 'EEE d MMM', weekdayDate: 'EEE d MMM y', longDate: 'EEEE d MMM y', dayMonth: 'd MMM', monthDay: 'd MMM', monthYear: 'MMM y', monthYearLong: 'MMMM y',
    time24: 'HH:mm', time12: 'h:mm a', hourCycle: 'h23', weekStart: 'monday',
  },
  'pt-BR': {
    name: 'Português (Brasil)', decimal: ',', group: '.', minGroup: 1, percent: '#%', currency: `US$${NBSP}#`,
    months: ['jan.', 'fev.', 'mar.', 'abr.', 'mai.', 'jun.', 'jul.', 'ago.', 'set.', 'out.', 'nov.', 'dez.'],
    monthsLong: ['janeiro', 'fevereiro', 'março', 'abril', 'maio', 'junho', 'julho', 'agosto', 'setembro', 'outubro', 'novembro', 'dezembro'],
    days: ['dom.', 'seg.', 'ter.', 'qua.', 'qui.', 'sex.', 'sáb.'],
    daysLong: ['domingo', 'segunda-feira', 'terça-feira', 'quarta-feira', 'quinta-feira', 'sexta-feira', 'sábado'],
    am: 'AM', pm: 'PM',
    date: "d 'de' MMM 'de' y", shortDate: "EEE, d 'de' MMM", weekdayDate: "EEE, d 'de' MMM 'de' y", longDate: "EEEE, d 'de' MMM 'de' y", dayMonth: "d 'de' MMM", monthDay: "d 'de' MMM", monthYear: "MMM 'de' y", monthYearLong: "MMMM 'de' y",
    time24: 'HH:mm', time12: 'h:mm a', hourCycle: 'h23', weekStart: 'sunday',
  },
}

/** Every region format, in picker order. */
export const REGION_FORMATS: readonly RegionFormat[] = ['standard', 'en-US', 'vi-VN', 'de-DE', 'en-GB', 'es-ES', 'fr-FR', 'pt-BR']

export function isRegionFormat(v: unknown): v is RegionFormat {
  return typeof v === 'string' && Object.hasOwn(REGIONS, v)
}

function data(r: RegionFormat): RegionData {
  return REGIONS[r] ?? REGIONS.standard
}

/** The region format's name in its own language: "Tiếng Việt (Việt Nam)". */
export function regionName(r: RegionFormat): string {
  return data(r).name
}

/** The region's own clock and first day of the week. */
export function regionHourCycle(r: RegionFormat): HourCycle {
  return data(r).hourCycle
}
export function regionWeekStart(r: RegionFormat): WeekStart {
  return data(r).weekStart
}

/** The BCP 47 locale Intl uses for this region's numbers inside messages. */
export function regionLocale(r: RegionFormat): string {
  return r === 'standard' ? 'en-US' : r
}

/** A moment split into its fields in the display timezone. Weekday: 0 is Sunday. */
export interface DateParts {
  year: number
  month: number
  day: number
  weekday: number
  hour: number
  minute: number
  second: number
}

function pad(v: number, n: number): string {
  return String(v).padStart(n, '0')
}

/** Writes a date with a CLDR-style pattern. */
export function formatPattern(r: RegionFormat, pattern: string, p: DateParts): string {
  const d = data(r)
  let out = ''
  for (let i = 0; i < pattern.length; ) {
    const c = pattern[i]!
    if (c === "'") {
      const end = pattern.indexOf("'", i + 1)
      out += pattern.slice(i + 1, end < 0 ? undefined : end)
      i = end < 0 ? pattern.length : end + 1
      continue
    }
    if (!/[A-Za-z]/.test(c)) {
      out += c
      i++
      continue
    }
    let n = 1
    while (pattern[i + n] === c) n++
    i += n
    switch (c) {
      case 'E':
        out += n >= 4 ? d.daysLong[p.weekday] : d.days[p.weekday]
        break
      case 'd':
        out += pad(p.day, n)
        break
      case 'M':
        out += n >= 4 ? d.monthsLong[p.month - 1] : n === 3 ? d.months[p.month - 1] : pad(p.month, n)
        break
      case 'y':
        out += String(p.year)
        break
      case 'H':
        out += pad(p.hour, n)
        break
      case 'h':
        out += pad(p.hour % 12 || 12, n)
        break
      case 'm':
        out += pad(p.minute, n)
        break
      case 's':
        out += pad(p.second, n)
        break
      case 'a':
        out += p.hour < 12 ? d.am : d.pm
        break
      default:
        out += c.repeat(n)
    }
  }
  return out
}

type DatePattern = 'date' | 'shortDate' | 'weekdayDate' | 'longDate' | 'dayMonth' | 'monthDay' | 'monthYear' | 'monthYearLong'

export function formatDate(r: RegionFormat, kind: DatePattern, p: DateParts): string {
  return formatPattern(r, data(r)[kind], p)
}

/** A time of day on the given clock (the region's own when omitted). */
export function formatTime(r: RegionFormat, p: DateParts, hourCycle?: HourCycle, seconds = false): string {
  const d = data(r)
  let pattern = (hourCycle ?? d.hourCycle) === 'h12' ? d.time12 : d.time24
  if (seconds) pattern = pattern.replace('mm', 'mm:ss')
  return formatPattern(r, pattern, p)
}

/** The month's name: "Oct", "thg 10"; "October", "tháng 10". */
export function monthLabel(r: RegionFormat, month: number, style: 'short' | 'long'): string {
  const d = data(r)
  return (style === 'long' ? d.monthsLong : d.months)[month - 1] ?? ''
}

/** The weekday's full name. 0 is Sunday. */
export function weekdayLabel(r: RegionFormat, weekday: number, style: 'short' | 'long' = 'long'): string {
  const d = data(r)
  return (style === 'long' ? d.daysLong : d.days)[weekday] ?? ''
}

function group(digits: string, sep: string, minGroup: number): string {
  if (digits.length < 3 + minGroup) return digits
  const head = digits.length % 3
  const parts = head ? [digits.slice(0, head)] : []
  for (let i = head; i < digits.length; i += 3) parts.push(digits.slice(i, i + 3))
  return parts.join(sep)
}

/** A number with exactly `digits` fraction digits. */
export function formatDecimal(r: RegionFormat, v: number, digits = 0): string {
  const d = data(r)
  const neg = v < 0
  const fixed = Math.abs(v).toFixed(digits)
  const [int = '0', frac] = fixed.split('.')
  let out = group(int, d.group, d.minGroup)
  if (frac) out += d.decimal + frac
  return neg && /[1-9]/.test(fixed) ? `-${out}` : out
}

export function formatInt(r: RegionFormat, n: number): string {
  return formatDecimal(r, Math.round(n), 0)
}

/** A percentage given in percent units: 99.1 → 99.1% · 99,1 %. */
export function formatPercent(r: RegionFormat, v: number, digits = 1): string {
  return data(r).percent.replace('#', formatDecimal(r, v, digits))
}

/** US dollars, the currency provider costs are in: $2.84 · 2,84 US$. */
export function formatMoney(r: RegionFormat, v: number, digits = 2): string {
  return data(r).currency.replace('#', formatDecimal(r, v, digits))
}
