package referrals

import (
	"crypto/rand"
	"math/big"
	"net/netip"
	"strings"

	"github.com/hostrepute/reputecast/internal/emailaddr"
)

// CodeAlphabet is what referral codes are made of: digits and capital
// letters without look-alikes (no 0, O, 1, I or L). A code says nothing
// about the person.
const CodeAlphabet = "23456789ABCDEFGHJKMNPQRSTUVWXYZ"

// CodeLength is the length of a referral code ("K7Q2MX").
const CodeLength = 6

// NewCode returns a random referral code.
func NewCode() string {
	b := make([]byte, CodeLength)
	n := big.NewInt(int64(len(CodeAlphabet)))
	for i := range b {
		v, err := rand.Int(rand.Reader, n)
		if err != nil {
			panic(err)
		}
		b[i] = CodeAlphabet[v.Int64()]
	}
	return string(b)
}

// NormalizeCode reads a code as people type or paste it ("k7q2mx",
// " K7Q2MX/") and reports whether it can be a referral code at all.
func NormalizeCode(s string) (string, bool) {
	s = strings.ToUpper(strings.Trim(strings.TrimSpace(s), "/"))
	if len(s) != CodeLength {
		return "", false
	}
	for i := 0; i < len(s); i++ {
		if strings.IndexByte(CodeAlphabet, s[i]) < 0 {
			return "", false
		}
	}
	return s, true
}

// SameMailbox reports whether two addresses reach the same mailbox: plus
// tags and dots in Gmail addresses are ignored (the self-referral rule).
func SameMailbox(a, b string) bool {
	ka, kb := emailaddr.MailboxKey(a), emailaddr.MailboxKey(b)
	return ka != "" && ka == kb
}

// NumberedKey is an address with every run of digits in its local part
// replaced by "#", after folding plus tags and Gmail dots:
// ben.t01@outlook.com and ben.t02@outlook.com both give
// "ben.t#@outlook.com". It is empty for an address without a digit in its
// local part, so such addresses never match.
func NumberedKey(email string) string {
	key := emailaddr.MailboxKey(email)
	at := strings.LastIndexByte(key, '@')
	if at <= 0 {
		return ""
	}
	local, domain := key[:at], key[at:]
	var b strings.Builder
	digits, inRun := false, false
	for _, r := range local {
		if r >= '0' && r <= '9' {
			digits = true
			if !inRun {
				b.WriteByte('#')
				inRun = true
			}
			continue
		}
		inRun = false
		b.WriteRune(r)
	}
	if !digits {
		return ""
	}
	return b.String() + domain
}

// NetworkOf is the network an address belongs to, what an address is
// reduced to after 90 days: its /24 for IPv4, its /48 for IPv6.
func NetworkOf(ip netip.Addr) string {
	if !ip.IsValid() {
		return ""
	}
	ip = ip.Unmap()
	bits := 48
	if ip.Is4() {
		bits = 24
	}
	p, err := ip.Prefix(bits)
	if err != nil {
		return ""
	}
	return p.String()
}

// Channels where links are shared, as share buttons add them (?via=).
const (
	ViaCopy     = "copy"
	ViaEmail    = "email"
	ViaX        = "x"
	ViaLinkedIn = "linkedin"
	ViaWhatsApp = "whatsapp"
	ViaOther    = "other"
)

// Channels in the order results list them.
var Channels = []string{ViaCopy, ViaX, ViaLinkedIn, ViaWhatsApp, ViaEmail, ViaOther}

// ShareButtons a program can show, in the order pages show them.
var ShareButtons = []string{ViaCopy, ViaEmail, ViaX, ViaLinkedIn, ViaWhatsApp}

// ChannelOf reads ?via=: no value is a copied link, an unknown one "other".
func ChannelOf(via string) string {
	switch v := strings.ToLower(strings.TrimSpace(via)); v {
	case "":
		return ViaCopy
	case ViaCopy, ViaEmail, ViaX, ViaLinkedIn, ViaWhatsApp:
		return v
	case "twitter":
		return ViaX
	}
	return ViaOther
}
