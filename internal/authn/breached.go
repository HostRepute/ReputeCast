package authn

import (
	"bufio"
	"context"
	"crypto/sha1" //nolint:gosec // the HIBP range API is keyed by SHA-1 prefixes; nothing is stored.
	"encoding/hex"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// BreachChecker asks the Have I Been Pwned range API whether a password is
// known from breaches. Only the first five hex characters of the SHA-1 leave
// the server (k-anonymity). Instance admins can turn it off for offline installs.
type BreachChecker struct {
	Client  *http.Client
	BaseURL string
}

// NewBreachChecker returns a checker with a short timeout.
func NewBreachChecker() *BreachChecker {
	return &BreachChecker{Client: &http.Client{Timeout: 4 * time.Second}, BaseURL: "https://api.pwnedpasswords.com/range/"}
}

// Breached reports whether the password appears in the breach corpus.
func (c *BreachChecker) Breached(ctx context.Context, password string) (bool, error) {
	sum := sha1.Sum([]byte(password)) //nolint:gosec // see above
	hexSum := strings.ToUpper(hex.EncodeToString(sum[:]))
	prefix, suffix := hexSum[:5], hexSum[5:]
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.BaseURL+prefix, nil)
	if err != nil {
		return false, err
	}
	req.Header.Set("Add-Padding", "true")
	req.Header.Set("User-Agent", "ReputeCast")
	resp, err := c.Client.Do(req)
	if err != nil {
		return false, err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return false, fmt.Errorf("breach check: HTTP %d", resp.StatusCode)
	}
	sc := bufio.NewScanner(resp.Body)
	for sc.Scan() {
		line := sc.Text()
		hashPart, count, ok := strings.Cut(line, ":")
		if ok && hashPart == suffix && strings.TrimSpace(count) != "0" {
			return true, nil
		}
	}
	return false, sc.Err()
}
