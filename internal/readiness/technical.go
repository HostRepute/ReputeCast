package readiness

import (
	"fmt"
	"net/netip"
	"regexp"
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

// txtAnswer is a TXT record as dig shows it, for a check's technical
// details (R1.U2): the lookup and the record, never inside a sentence.
func txtAnswer(name, record string) []string {
	if strings.TrimSpace(name) == "" || strings.TrimSpace(record) == "" {
		return nil
	}
	return []string{"$ dig +short " + name + " TXT", `"` + record + `"`}
}

// dnsAnswerRow names what DNS answered a lookup that failed (SERVFAIL,
// REFUSED, a timeout), nil when there is nothing to name.
func dnsAnswerRow(trouble string) []techRow {
	switch trouble {
	case TroubleTimeout, TroubleServFail, TroubleRefused:
		return []techRow{{Label: tx("tech.dnsAnswer", nil), Value: tx("tech.dnsAnswerValue", A{"trouble": trouble})}}
	}
	return nil
}

// unsubscribeHeaders are the one-click unsubscribe headers as the send
// writes them; {token} stands for each recipient's.
func unsubscribeHeaders(cf ComplianceFacts) []string {
	lu := "<" + cf.UnsubscribeURL + ">"
	if cf.Mailto != "" {
		lu += ", <mailto:" + cf.Mailto + "?subject=unsubscribe%20{token}>"
	}
	return []string{"List-Unsubscribe: " + lu, "List-Unsubscribe-Post: List-Unsubscribe=One-Click"}
}

// maxLinkRows is how many links' or images' responses technical details
// list.
const maxLinkRows = 10

// responseRow is what a link or an image answered, in a check's technical
// details: "HTTP 404", "No response".
func responseRow(url string, status int) techRow {
	v := tx("tech.noResponse", nil)
	if status != 0 {
		v = tx("tech.httpStatus", A{"status": fmt.Sprint(status)})
	}
	return techRow{Label: RawText(displayURL(url)), Value: v}
}

// codeSuffix splits an outage's English sentence from the error it ends
// with in brackets: "mta.acme.io has refused connections since 08:52
// (ECONNREFUSED on port 25 and 443)".
var codeSuffix = regexp.MustCompile(`^(.+?)\s*\(([^()]+)\)\.?$`)

func orNone(s string) string {
	if s == "" {
		return "none"
	}
	return s
}
