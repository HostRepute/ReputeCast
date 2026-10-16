package i18n

import (
	"strconv"
	"strings"
	"time"
)

// Region is a region format: how dates, times and numbers are written.
// Standard is ReputeCast's own; the others are CLDR locales.
type Region string

// Standard keeps the formats ReputeCast has always used: Tue 14 Oct 2026 ·
// 09:00 · 48,213 · 99.1% · $2.84.
const Standard Region = "standard"

// HourCycle is a 24-hour (h23) or 12-hour (h12) clock.
type HourCycle string

const (
	H23 HourCycle = "h23"
	H12 HourCycle = "h12"
)

// regionData is a snapshot of CLDR for one region format, in the shapes
// ReputeCast writes. web/src/i18n/regions.ts holds the same data.
type regionData struct {
	name           string
	decimal, group string
	minGroup       int    // digits before grouping starts: 1, or 2 for Spanish (4821, 48.213)
	percent        string // "#%", "# %"
	currency       string // USD: "$#", "# US$"…
	months         [12]string
	monthsLong     [12]string
	days           [7]string // abbreviated, Sunday first
	daysLong       [7]string
	am, pm         string
	// Patterns: EEEE/EEE weekday, d day, MMMM/MMM month, y year, HH/H/h
	// hour, mm minute, ss second, a am/pm, 'quoted' literal text.
	date        string // Oct 14, 2026
	shortDate   string // Tue 14 Oct
	weekdayDate string // Tue 14 Oct 2026
	longDate    string // Tuesday 14 Oct 2026
	dayMonth    string // 14 Oct
	monthDay    string // Oct 14
	time24      string
	time12      string
	hour12      bool
	sundayFirst bool
}

const nbsp = " "

var (
	enMonths     = [12]string{"Jan", "Feb", "Mar", "Apr", "May", "Jun", "Jul", "Aug", "Sep", "Oct", "Nov", "Dec"}
	enMonthsLong = [12]string{"January", "February", "March", "April", "May", "June", "July", "August", "September", "October", "November", "December"}
	enDays       = [7]string{"Sun", "Mon", "Tue", "Wed", "Thu", "Fri", "Sat"}
	enDaysLong   = [7]string{"Sunday", "Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday"}
)

var regions = map[Region]*regionData{
	Standard: {
		name: "ReputeCast standard", decimal: ".", group: ",", minGroup: 1, percent: "#%", currency: "$#",
		months: enMonths, monthsLong: enMonthsLong, days: enDays, daysLong: enDaysLong, am: "AM", pm: "PM",
		date: "MMM d, y", shortDate: "EEE d MMM", weekdayDate: "EEE d MMM y", longDate: "EEEE d MMM y", dayMonth: "d MMM", monthDay: "MMM d",
		time24: "HH:mm", time12: "h:mm a",
	},
	"en-US": {
		name: "English (United States)", decimal: ".", group: ",", minGroup: 1, percent: "#%", currency: "$#",
		months: enMonths, monthsLong: enMonthsLong, days: enDays, daysLong: enDaysLong, am: "AM", pm: "PM",
		date: "MMM d, y", shortDate: "EEE, MMM d", weekdayDate: "EEE, MMM d, y", longDate: "EEEE, MMM d, y", dayMonth: "MMM d", monthDay: "MMM d",
		time24: "HH:mm", time12: "h:mm a", hour12: true, sundayFirst: true,
	},
	"en-GB": {
		name: "English (United Kingdom)", decimal: ".", group: ",", minGroup: 1, percent: "#%", currency: "US$#",
		months: [12]string{"Jan", "Feb", "Mar", "Apr", "May", "Jun", "Jul", "Aug", "Sept", "Oct", "Nov", "Dec"}, monthsLong: enMonthsLong,
		days: enDays, daysLong: enDaysLong, am: "am", pm: "pm",
		date: "d MMM y", shortDate: "EEE d MMM", weekdayDate: "EEE, d MMM y", longDate: "EEEE, d MMM y", dayMonth: "d MMM", monthDay: "d MMM",
		time24: "HH:mm", time12: "h:mm a",
	},
	"vi-VN": {
		name: "Tiếng Việt (Việt Nam)", decimal: ",", group: ".", minGroup: 1, percent: "#%", currency: "#" + nbsp + "US$",
		months:     [12]string{"thg 1", "thg 2", "thg 3", "thg 4", "thg 5", "thg 6", "thg 7", "thg 8", "thg 9", "thg 10", "thg 11", "thg 12"},
		monthsLong: [12]string{"tháng 1", "tháng 2", "tháng 3", "tháng 4", "tháng 5", "tháng 6", "tháng 7", "tháng 8", "tháng 9", "tháng 10", "tháng 11", "tháng 12"},
		// The abbreviations the design uses ("Th 3, 14 thg 10"); CLDR 48 moved to "Thứ 3".
		days:     [7]string{"CN", "Th 2", "Th 3", "Th 4", "Th 5", "Th 6", "Th 7"},
		daysLong: [7]string{"Chủ Nhật", "Thứ Hai", "Thứ Ba", "Thứ Tư", "Thứ Năm", "Thứ Sáu", "Thứ Bảy"},
		am:       "SA", pm: "CH",
		date: "d MMM, y", shortDate: "EEE, d MMM", weekdayDate: "EEE, d MMM, y", longDate: "EEEE, d MMMM, y", dayMonth: "d MMM", monthDay: "d MMM",
		time24: "HH:mm", time12: "h:mm a",
	},
	"de-DE": {
		name: "Deutsch (Deutschland)", decimal: ",", group: ".", minGroup: 1, percent: "#" + nbsp + "%", currency: "#" + nbsp + "$",
		months:     [12]string{"Jan.", "Feb.", "März", "Apr.", "Mai", "Juni", "Juli", "Aug.", "Sept.", "Okt.", "Nov.", "Dez."},
		monthsLong: [12]string{"Januar", "Februar", "März", "April", "Mai", "Juni", "Juli", "August", "September", "Oktober", "November", "Dezember"},
		days:       [7]string{"So.", "Mo.", "Di.", "Mi.", "Do.", "Fr.", "Sa."},
		daysLong:   [7]string{"Sonntag", "Montag", "Dienstag", "Mittwoch", "Donnerstag", "Freitag", "Samstag"},
		am:         "AM", pm: "PM",
		date: "d. MMM y", shortDate: "EEE, d. MMM", weekdayDate: "EEE, d. MMM y", longDate: "EEEE, d. MMM y", dayMonth: "d. MMM", monthDay: "d. MMM",
		time24: "HH:mm", time12: "h:mm a",
	},
	"es-ES": {
		name: "Español (España)", decimal: ",", group: ".", minGroup: 2, percent: "#" + nbsp + "%", currency: "#" + nbsp + "US$",
		months:     [12]string{"ene", "feb", "mar", "abr", "may", "jun", "jul", "ago", "sept", "oct", "nov", "dic"},
		monthsLong: [12]string{"enero", "febrero", "marzo", "abril", "mayo", "junio", "julio", "agosto", "septiembre", "octubre", "noviembre", "diciembre"},
		days:       [7]string{"dom", "lun", "mar", "mié", "jue", "vie", "sáb"},
		daysLong:   [7]string{"domingo", "lunes", "martes", "miércoles", "jueves", "viernes", "sábado"},
		am:         "a. m.", pm: "p. m.",
		date: "d MMM y", shortDate: "EEE, d MMM", weekdayDate: "EEE, d MMM y", longDate: "EEEE, d 'de' MMM 'de' y", dayMonth: "d MMM", monthDay: "d MMM",
		time24: "H:mm", time12: "h:mm a",
	},
	"fr-FR": {
		name: "Français (France)", decimal: ",", group: " ", minGroup: 1, percent: "#" + nbsp + "%", currency: "#" + nbsp + "$US",
		months:     [12]string{"janv.", "févr.", "mars", "avr.", "mai", "juin", "juil.", "août", "sept.", "oct.", "nov.", "déc."},
		monthsLong: [12]string{"janvier", "février", "mars", "avril", "mai", "juin", "juillet", "août", "septembre", "octobre", "novembre", "décembre"},
		days:       [7]string{"dim.", "lun.", "mar.", "mer.", "jeu.", "ven.", "sam."},
		daysLong:   [7]string{"dimanche", "lundi", "mardi", "mercredi", "jeudi", "vendredi", "samedi"},
		am:         "AM", pm: "PM",
		date: "d MMM y", shortDate: "EEE d MMM", weekdayDate: "EEE d MMM y", longDate: "EEEE d MMM y", dayMonth: "d MMM", monthDay: "d MMM",
		time24: "HH:mm", time12: "h:mm a",
	},
	"pt-BR": {
		name: "Português (Brasil)", decimal: ",", group: ".", minGroup: 1, percent: "#%", currency: "US$" + nbsp + "#",
		months:     [12]string{"jan.", "fev.", "mar.", "abr.", "mai.", "jun.", "jul.", "ago.", "set.", "out.", "nov.", "dez."},
		monthsLong: [12]string{"janeiro", "fevereiro", "março", "abril", "maio", "junho", "julho", "agosto", "setembro", "outubro", "novembro", "dezembro"},
		days:       [7]string{"dom.", "seg.", "ter.", "qua.", "qui.", "sex.", "sáb."},
		daysLong:   [7]string{"domingo", "segunda-feira", "terça-feira", "quarta-feira", "quinta-feira", "sexta-feira", "sábado"},
		am:         "AM", pm: "PM",
		date: "d 'de' MMM 'de' y", shortDate: "EEE, d 'de' MMM", weekdayDate: "EEE, d 'de' MMM 'de' y", longDate: "EEEE, d 'de' MMM 'de' y", dayMonth: "d 'de' MMM", monthDay: "d 'de' MMM",
		time24: "HH:mm", time12: "h:mm a", sundayFirst: true,
	},
}

// regionOrder is the order of the region picker.
var regionOrder = []Region{Standard, "en-US", "vi-VN", "de-DE", "en-GB", "es-ES", "fr-FR", "pt-BR"}

// Regions returns every region format, in picker order.
func Regions() []Region { return append([]Region{}, regionOrder...) }

func (r Region) known() bool { _, ok := regions[r]; return ok }

// Valid reports whether r is a region format ReputeCast has.
func (r Region) Valid() bool { return r.known() }

func (r Region) data() *regionData {
	if d, ok := regions[r]; ok {
		return d
	}
	return regions[Standard]
}

// Name is the region format's name in its own language: "Tiếng Việt (Việt Nam)".
func (r Region) Name() string { return r.data().name }

// DefaultRegion is the region format a language follows when the member
// hasn't chosen one: ReputeCast standard for English (and the
// pseudo-locale), Tiếng Việt (Việt Nam) for Vietnamese.
func (l Locale) DefaultRegion() Region {
	if l == Vietnamese {
		return "vi-VN"
	}
	return Standard
}

// HourCycle is the region's clock: 12-hour for English (United States).
func (r Region) HourCycle() HourCycle {
	if r.data().hour12 {
		return H12
	}
	return H23
}

// WeekStartsSunday reports whether the region's week starts on Sunday.
func (r Region) WeekStartsSunday() bool { return r.data().sundayFirst }

// Int writes an integer with the region's grouping: 48,213 · 48.213 · 48 213.
func (r Region) Int(n int64) string { return r.Decimal(float64(n), 0) }

// Decimal writes a number with exactly digits fraction digits.
func (r Region) Decimal(v float64, digits int) string {
	d := r.data()
	neg := v < 0
	if neg {
		v = -v
	}
	s := strconv.FormatFloat(v, 'f', digits, 64)
	intPart, frac, _ := strings.Cut(s, ".")
	out := group(intPart, d.group, d.minGroup)
	if frac != "" {
		out += d.decimal + frac
	}
	if neg && strings.Trim(s, "0.") != "" {
		out = "-" + out
	}
	return out
}

func group(digits, sep string, minGroup int) string {
	if len(digits) < 3+minGroup {
		return digits
	}
	var b strings.Builder
	head := len(digits) % 3
	if head > 0 {
		b.WriteString(digits[:head])
	}
	for i := head; i < len(digits); i += 3 {
		if b.Len() > 0 {
			b.WriteString(sep)
		}
		b.WriteString(digits[i : i+3])
	}
	return b.String()
}

// Percent writes a percentage given in percent units (99.1 → 99.1%).
func (r Region) Percent(v float64, digits int) string {
	return strings.Replace(r.data().percent, "#", r.Decimal(v, digits), 1)
}

// Money writes US dollars, the currency provider costs are in: $2.84 · 2,84 US$.
func (r Region) Money(v float64, digits int) string {
	return strings.Replace(r.data().currency, "#", r.Decimal(v, digits), 1)
}

// Date writes a day with its year: Oct 14, 2026 · 14 thg 10, 2026.
func (r Region) Date(t time.Time) string { return r.pattern(r.data().date, t) }

// ShortDate writes a day with its weekday: Tue 14 Oct · Th 3, 14 thg 10.
func (r Region) ShortDate(t time.Time) string { return r.pattern(r.data().shortDate, t) }

// WeekdayDate writes a day with its weekday and year: Tue 14 Oct 2026.
func (r Region) WeekdayDate(t time.Time) string { return r.pattern(r.data().weekdayDate, t) }

// LongDate writes a day with the weekday in full: Tuesday 14 Oct 2026.
func (r Region) LongDate(t time.Time) string { return r.pattern(r.data().longDate, t) }

// DayMonth writes a day without the weekday or year: 14 Oct · 14 thg 10.
func (r Region) DayMonth(t time.Time) string { return r.pattern(r.data().dayMonth, t) }

// Time writes a time of day on the given clock, or the region's own when hc is empty.
func (r Region) Time(t time.Time, hc HourCycle) string {
	d := r.data()
	p := d.time24
	if hc == H12 || (hc == "" && d.hour12) {
		p = d.time12
	}
	return r.pattern(p, t)
}

// pattern writes t with a CLDR-style pattern.
func (r Region) pattern(p string, t time.Time) string {
	d := r.data()
	var b strings.Builder
	rs := []rune(p)
	for i := 0; i < len(rs); {
		c := rs[i]
		if c == '\'' {
			j := i + 1
			for j < len(rs) && rs[j] != '\'' {
				j++
			}
			b.WriteString(string(rs[i+1 : j]))
			i = j + 1
			continue
		}
		if (c < 'a' || c > 'z') && (c < 'A' || c > 'Z') {
			b.WriteRune(c)
			i++
			continue
		}
		n := 1
		for i+n < len(rs) && rs[i+n] == c {
			n++
		}
		i += n
		switch c {
		case 'E':
			if n >= 4 {
				b.WriteString(d.daysLong[t.Weekday()])
			} else {
				b.WriteString(d.days[t.Weekday()])
			}
		case 'd':
			b.WriteString(pad(t.Day(), n))
		case 'M':
			switch {
			case n >= 4:
				b.WriteString(d.monthsLong[t.Month()-1])
			case n == 3:
				b.WriteString(d.months[t.Month()-1])
			default:
				b.WriteString(pad(int(t.Month()), n))
			}
		case 'y':
			b.WriteString(strconv.Itoa(t.Year()))
		case 'H':
			b.WriteString(pad(t.Hour(), n))
		case 'h':
			h := t.Hour() % 12
			if h == 0 {
				h = 12
			}
			b.WriteString(pad(h, n))
		case 'm':
			b.WriteString(pad(t.Minute(), n))
		case 's':
			b.WriteString(pad(t.Second(), n))
		case 'a':
			if t.Hour() < 12 {
				b.WriteString(d.am)
			} else {
				b.WriteString(d.pm)
			}
		default:
			b.WriteString(strings.Repeat(string(c), n))
		}
	}
	return b.String()
}

func pad(v, n int) string {
	s := strconv.Itoa(v)
	for len(s) < n {
		s = "0" + s
	}
	return s
}
