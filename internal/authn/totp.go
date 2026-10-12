package authn

import (
	"crypto/rand"
	"strings"
	"time"

	"github.com/pquerna/otp"
	"github.com/pquerna/otp/totp"
)

// TOTPSecret holds a freshly generated authenticator secret.
type TOTPSecret struct {
	Secret string // base32, no padding
	URL    string // otpauth:// URL for the QR code
}

// NewTOTP generates a secret for the given issuer (instance name) and account.
func NewTOTP(issuer, account string) (TOTPSecret, error) {
	key, err := totp.Generate(totp.GenerateOpts{
		Issuer:      issuer,
		AccountName: account,
		Period:      30,
		Digits:      otp.DigitsSix,
		Algorithm:   otp.AlgorithmSHA1, // the only algorithm every authenticator app supports
	})
	if err != nil {
		return TOTPSecret{}, err
	}
	return TOTPSecret{Secret: key.Secret(), URL: key.URL()}, nil
}

// GroupSecret formats a secret for manual entry: "JBSW Y3DP EHPK 3PXP".
func GroupSecret(secret string) string {
	var b strings.Builder
	for i, r := range secret {
		if i > 0 && i%4 == 0 {
			b.WriteByte(' ')
		}
		b.WriteRune(r)
	}
	return b.String()
}

// VerifyTOTP checks a 6-digit code, allowing one 30-second step of clock skew.
// Spaces in pasted codes ("123 456") are ignored.
func VerifyTOTP(secret, code string, now time.Time) bool {
	code = strings.ReplaceAll(strings.TrimSpace(code), " ", "")
	ok, err := totp.ValidateCustom(code, secret, now, totp.ValidateOpts{
		Period:    30,
		Skew:      1,
		Digits:    otp.DigitsSix,
		Algorithm: otp.AlgorithmSHA1,
	})
	return err == nil && ok
}

// recoveryAlphabet avoids look-alike letters (no i, l, o, u).
const recoveryAlphabet = "0123456789abcdefghjkmnpqrstvwxyz"

// NewRecoveryCodes returns n one-time codes formatted "b4xk-9q2m-7hdn".
func NewRecoveryCodes(n int) []string {
	codes := make([]string, n)
	buf := make([]byte, 12)
	for i := range codes {
		if _, err := rand.Read(buf); err != nil {
			panic(err)
		}
		var b strings.Builder
		for j, c := range buf {
			if j > 0 && j%4 == 0 {
				b.WriteByte('-')
			}
			b.WriteByte(recoveryAlphabet[int(c)%len(recoveryAlphabet)])
		}
		codes[i] = b.String()
	}
	return codes
}

// NormalizeRecoveryCode lower-cases and strips spaces so pasted codes match.
func NormalizeRecoveryCode(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = strings.ReplaceAll(s, " ", "")
	if len(s) == 12 && !strings.Contains(s, "-") {
		s = s[0:4] + "-" + s[4:8] + "-" + s[8:12]
	}
	return s
}
