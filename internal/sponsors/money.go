package sponsors

import (
	"fmt"
	"strconv"
	"strings"
)

var symbols = map[string]string{"usd": "$", "eur": "€", "gbp": "£", "cad": "CA$", "aud": "A$", "jpy": "¥", "inr": "₹", "brl": "R$"}

// money writes an amount the design's way for server text: the symbol
// first, thousands separators, whole when it has no cents ("$750",
// "$750.50").
func money(cents int64, currency string) string {
	cur := strings.ToLower(currency)
	sign := ""
	if cents < 0 {
		sign, cents = "−", -cents
	}
	sym, ok := symbols[cur]
	if !ok {
		sym = strings.ToUpper(cur) + " "
	}
	whole, frac := cents/100, cents%100
	s := strconv.FormatInt(whole, 10)
	var b strings.Builder
	for i, r := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			b.WriteByte(',')
		}
		b.WriteRune(r)
	}
	if frac == 0 {
		return sign + sym + b.String()
	}
	return sign + sym + b.String() + "." + fmt.Sprintf("%02d", frac)
}
