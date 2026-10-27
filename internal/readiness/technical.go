package readiness

import (
	"net/netip"
	"strconv"
	"strings"

	"github.com/hostrepute/reputecast/internal/dnscheck"
)

// The technical details of a listed IP (SR-13): the list and its zone,
// what the listing covers, the other lists, where the result comes from,
// and the list's DNS answers when they are known.

// listZones are the DNS zones of lists HostRepute reports beside
// ReputeCast's own (dnscheck.Builtin).
var listZones = map[string]string{
	"UCEPROTECT Level 1": "dnsbl-1.uceprotect.net",
	"UCEPROTECT Level 2": "dnsbl-2.uceprotect.net",
	"UCEPROTECT Level 3": "dnsbl-3.uceprotect.net",
	"Barracuda":          "b.barracudacentral.org",
	"Spamhaus SBL":       "sbl.spamhaus.org",
	"Spamhaus XBL":       "xbl.spamhaus.org",
	"Spamhaus PBL":       "pbl.spamhaus.org",
}

// listZone is a list's DNS zone, "" when it isn't known.
func listZone(source string) string {
	for _, l := range dnscheck.Builtin {
		if strings.EqualFold(l.Name, source) {
			return l.Zone
		}
	}
	return listZones[source]
}

// reversedIPv4 is the name a DNS blocklist looks an IPv4 address up under
// ("25.113.0.203"), "" for anything else.
func reversedIPv4(ip string) string {
	a, err := netip.ParseAddr(ip)
	if err != nil || !a.Is4() {
		return ""
	}
	b := a.As4()
	return strings.Join([]string{strconv.Itoa(int(b[3])), strconv.Itoa(int(b[2])), strconv.Itoa(int(b[1])), strconv.Itoa(int(b[0]))}, ".")
}

// listingDetails are a listed IP's technical rows and raw DNS.
func (b *builder) listingDetails(ip string, listed []ListingFacts) ([]techRow, []string) {
	r := b.f.Reputation
	first := listed[0]
	zone := listZone(first.Source)
	rows := []techRow{{Label: tx("tech.blocklist", nil), Value: tx("tech.blocklistValue", A{"source": first.Source, "zone": orNone(zone)})}}
	if first.Detail != "" {
		rows = append(rows, techRow{Label: tx("tech.scope", nil), Value: RawText(first.Detail)})
	}
	if other := r.Sources - len(listed); other > 0 {
		rows = append(rows, techRow{Label: tx("tech.otherLists", nil), Value: tx("tech.otherListsValue", A{"asset": ip, "count": other})})
	}
	since := "none"
	if first.Since != nil {
		since = "day"
	}
	args := A{"since": since}
	if first.Since != nil {
		args["day"] = ArgDay(*first.Since)
	}
	if r.CheckedAt != nil {
		args["time"] = b.when(*r.CheckedAt)
		args["checked"] = "yes"
	} else {
		args["checked"] = "no"
	}
	key := "tech.sourceHostRepute"
	if r.Via == "builtin" {
		key = "tech.sourceBuiltin"
	}
	rows = append(rows, techRow{Label: tx("tech.source", nil), Value: tx(key, args)})
	raw := []string{}
	if q := reversedIPv4(ip); q != "" && zone != "" {
		name := q + "." + zone
		if first.Answer != "" {
			raw = append(raw, "$ dig +short "+name+" A", first.Answer)
		}
		if first.TXT != "" {
			raw = append(raw, "$ dig +short "+name+" TXT", `"`+first.TXT+`"`)
		}
	}
	if len(raw) == 0 {
		raw = nil
	}
	return rows, raw
}

func orNone(s string) string {
	if s == "" {
		return "none"
	}
	return s
}
