// Package phone reads phone numbers for the text channels (design expansion
// phase 8): E.164, the country the number belongs to, what kind of line it
// looks like, and the recipient's time zone when the contact has none.
//
// The country always comes from the number, never from the contact's
// address. Parsing and the area-code time zones come from libphonenumber's
// metadata (nyaruka/phonenumbers).
package phone

import (
	"errors"
	"slices"
	"strings"
	"time"

	"github.com/nyaruka/phonenumbers"
)

// LineType is what a number is known to be. Carriers and providers know
// better than the number plan, so a provider's error ("landline or
// unreachable carrier") overrides it.
type LineType string

// Line types.
const (
	Mobile   LineType = "mobile"
	Landline LineType = "landline"
	TollFree LineType = "toll_free"
	VoIP     LineType = "voip"
	Unknown  LineType = "unknown"
)

// Number is a parsed phone number.
type Number struct {
	// E164 is the number as providers take it: +13125550123.
	E164 string `json:"e164"`
	// Country is its ISO 3166-1 alpha-2 code ("US").
	Country string `json:"country"`
	// CallingCode is the country calling code (1, 44).
	CallingCode int `json:"callingCode"`
	// Display is the international form with spaces: +1 312 555 0123.
	Display string `json:"display"`
	// Type is what the number plan says about the line.
	Type LineType `json:"type" enum:"mobile,landline,toll_free,voip,unknown"`
}

// Errors.
var (
	ErrInvalid = errors.New("phone: not a phone number")
	// ErrNoCountry is a national number without a country to read it in.
	ErrNoCountry = errors.New("phone: the number needs its country code")
)

// Parse reads a number as a person typed it ("+1 (312) 555-0123",
// "07700 900456" with country "GB"). defaultCountry is used for national
// numbers; it may be empty when the number starts with + or 00.
func Parse(raw, defaultCountry string) (Number, error) {
	s := strings.TrimSpace(raw)
	if s == "" {
		return Number{}, ErrInvalid
	}
	if strings.HasPrefix(s, "00") {
		s = "+" + s[2:]
	}
	region := strings.ToUpper(strings.TrimSpace(defaultCountry))
	if !strings.HasPrefix(s, "+") && region == "" {
		return Number{}, ErrNoCountry
	}
	if region == "" {
		region = "ZZ"
	}
	n, err := phonenumbers.Parse(s, region)
	if err != nil {
		return Number{}, ErrInvalid
	}
	if !phonenumbers.IsPossibleNumber(n) {
		return Number{}, ErrInvalid
	}
	out := Number{
		E164:        phonenumbers.Format(n, phonenumbers.E164),
		Country:     phonenumbers.GetRegionCodeForNumber(n),
		CallingCode: int(n.GetCountryCode()),
		Display:     display(n),
		Type:        lineType(phonenumbers.GetNumberType(n)),
	}
	if out.Country == "" || out.Country == "ZZ" {
		// A possible number in a range the metadata doesn't assign yet:
		// take the main country of its calling code.
		out.Country = phonenumbers.GetRegionCodeForCountryCode(out.CallingCode)
		if out.Country == "ZZ" {
			out.Country = ""
		}
	}
	return out, nil
}

// MustParse parses an E.164 number known to be valid (tests, the demo).
func MustParse(e164 string) Number {
	n, err := Parse(e164, "")
	if err != nil {
		panic("phone: " + e164 + ": " + err.Error())
	}
	return n
}

func display(n *phonenumbers.PhoneNumber) string {
	s := phonenumbers.Format(n, phonenumbers.INTERNATIONAL)
	// The app writes groups with spaces everywhere: +1 312 555 0123.
	s = strings.NewReplacer("-", " ", "(", "", ")", "").Replace(s)
	return strings.Join(strings.Fields(s), " ")
}

func lineType(t phonenumbers.PhoneNumberType) LineType {
	switch t {
	case phonenumbers.MOBILE:
		return Mobile
	case phonenumbers.FIXED_LINE:
		return Landline
	case phonenumbers.TOLL_FREE:
		return TollFree
	case phonenumbers.VOIP:
		return VoIP
	}
	// FIXED_LINE_OR_MOBILE (all of North America) and the rest: only the
	// carriers know.
	return Unknown
}

// Display formats an E.164 number for people: +1 312 555 0123. Anything
// that doesn't parse is returned as it is.
func Display(e164 string) string {
	n, err := Parse(e164, "")
	if err != nil {
		return e164
	}
	return n.Display
}

// Mask hides the middle of a number: +1 773 ••• 0148. Numbers are shown
// this way in lists (suppressions, event logs, reports).
func Mask(e164 string) string {
	d := Display(e164)
	groups := strings.Fields(d)
	digits := strings.Map(func(r rune) rune {
		if r >= '0' && r <= '9' {
			return r
		}
		return -1
	}, d)
	if len(groups) < 3 || len(digits) < 8 {
		return d
	}
	return groups[0] + " " + groups[1] + " ••• " + digits[len(digits)-4:]
}

// mainZones is the main time zone of countries that span several, for
// numbers whose area code doesn't settle it (toll-free numbers, mobile
// ranges, a Canadian area code across two zones).
var mainZones = map[string]string{
	"US": "America/New_York",
	"CA": "America/Toronto",
	"MX": "America/Mexico_City",
	"BR": "America/Sao_Paulo",
	"AR": "America/Argentina/Buenos_Aires",
	"CL": "America/Santiago",
	"AU": "Australia/Sydney",
	"NZ": "Pacific/Auckland",
	"RU": "Europe/Moscow",
	"KZ": "Asia/Almaty",
	"ID": "Asia/Jakarta",
	"CD": "Africa/Kinshasa",
	"ES": "Europe/Madrid",
	"PT": "Europe/Lisbon",
	"FR": "Europe/Paris",
	"GB": "Europe/London",
	"DE": "Europe/Berlin",
	"NL": "Europe/Amsterdam",
	"IE": "Europe/Dublin",
	"DK": "Europe/Copenhagen",
	"EC": "America/Guayaquil",
	"MN": "Asia/Ulaanbaatar",
	"MY": "Asia/Kuala_Lumpur",
	"UA": "Europe/Kyiv",
	"KI": "Pacific/Tarawa",
	"FM": "Pacific/Pohnpei",
	"PF": "Pacific/Tahiti",
	"UM": "Pacific/Wake",
	"GL": "America/Nuuk",
}

// CallingCode is a country's calling code ("GB" → 44); 0 when unknown.
func CallingCode(country string) int {
	return phonenumbers.GetCountryCodeForRegion(strings.ToUpper(country))
}

// Countries are the countries numbers can be from, as ISO codes.
func Countries() []string {
	out := []string{}
	for c := range phonenumbers.GetSupportedRegions() {
		out = append(out, c)
	}
	slices.Sort(out)
	return out
}

// ZoneSource says where a recipient's time zone came from.
type ZoneSource string

// Zone sources. Both phone sources are "assumed": the pre-send check counts
// them.
const (
	ZoneContact  ZoneSource = "contact"   // the contact's time zone field
	ZoneAreaCode ZoneSource = "area_code" // the number's area code maps to one zone
	ZoneCountry  ZoneSource = "country"   // the country's main zone
	ZoneUnknown  ZoneSource = "unknown"
)

// TimeZone is the zone the number points to: its area code's when that maps
// to one zone, else the country's main zone. It returns "" with
// ZoneUnknown when neither is known.
func TimeZone(e164 string) (string, ZoneSource) {
	n, err := phonenumbers.Parse(e164, "ZZ")
	if err != nil {
		return "", ZoneUnknown
	}
	zones, _ := phonenumbers.GetTimezonesForNumber(n)
	valid := zones[:0:0]
	for _, z := range zones {
		if z == phonenumbers.UNKNOWN_TIMEZONE {
			continue
		}
		if _, err := time.LoadLocation(z); err == nil {
			valid = append(valid, z)
		}
	}
	if len(valid) == 1 {
		return valid[0], ZoneAreaCode
	}
	country := phonenumbers.GetRegionCodeForNumber(n)
	if country == "" || country == "ZZ" {
		country = phonenumbers.GetRegionCodeForCountryCode(int(n.GetCountryCode()))
	}
	if z, ok := mainZones[country]; ok {
		return z, ZoneCountry
	}
	if len(valid) > 0 {
		return valid[0], ZoneCountry
	}
	return "", ZoneUnknown
}

// RecipientZone is the zone quiet hours use for a recipient: the contact's
// own time zone field when it has a valid one, otherwise the number's.
func RecipientZone(contactZone, e164 string) (string, ZoneSource) {
	if contactZone != "" {
		if _, err := time.LoadLocation(contactZone); err == nil {
			return contactZone, ZoneContact
		}
	}
	return TimeZone(e164)
}
