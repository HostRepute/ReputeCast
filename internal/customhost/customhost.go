// Package customhost decides whose a custom host is. The instance serves
// hosts for workspaces: the public pages and website domain, the SMS
// short-link domain, a link tracking domain and the push site of a
// publication website. A CNAME to the instance, or addresses matching it,
// only show that a host points at this server, not whose host it is, so a
// host belongs to one workspace: the first to use it. Any other workspace,
// and any workspace for a host under the instance's own address (where one
// wildcard record can point every name at the server), shows that a host
// is its own only with the TXT record _reputecast.<host>, whose value is
// that workspace's alone. The instance's own hosts are never a workspace's.
//
// Every use of a host goes through here, so the pages, the short links and
// link tracking can't each draw the line in a different place.
package customhost

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/dnscheck"
	"github.com/hostrepute/reputecast/internal/store"
)

// PurposeInstance is the purpose of a store.CustomHostClaims row for one of
// the instance's own hosts (its SMTP relay's).
const PurposeInstance = "instance"

// Use is who else uses a host that a workspace saves.
type Use int

const (
	// Free: nobody else uses the host.
	Free Use = iota
	// Instance: the host is one of the instance's own.
	Instance
	// Taken: another workspace uses the host, whatever its status there.
	Taken
)

// Claim says who else uses host as workspace saves it; only a Free host
// may be saved. It locks the host for the rest of q's transaction first,
// so two workspaces saving it at once can't both find it free.
func Claim(ctx context.Context, q *store.Queries, workspace uuid.UUID, host string) (Use, error) {
	host = strings.ToLower(host)
	if err := q.CustomHostLock(ctx, host); err != nil {
		return Free, err
	}
	claims, err := q.CustomHostClaims(ctx, store.CustomHostClaimsParams{Host: host, WorkspaceID: workspace})
	if err != nil {
		return Free, err
	}
	use := Free
	for _, c := range claims {
		if c.Purpose == PurposeInstance {
			return Instance, nil
		}
		use = Taken
	}
	return use, nil
}

// Proof is what can show, at a check, that a host is a workspace's.
type Proof int

const (
	// Records: whatever the use checks shows it (a CNAME to the instance,
	// matching addresses, the TXT record).
	Records Proof = iota
	// InUse: another workspace used the host first, so only the TXT record
	// shows it.
	InUse
	// UnderInstance: the host is under the instance's own address, so only
	// the TXT record shows it.
	UnderInstance
	// Never: one of the instance's own hosts, which nothing makes a
	// workspace's.
	Never
)

// TXTOnly reports whether only the TXT record shows the host is the
// workspace's.
func (p Proof) TXTOnly() bool { return p == InUse || p == UnderInstance }

// ProofFor says what can show that host is workspace's. instance is the
// host of the instance's public URL. held is since when the workspace has
// had host verified for the use being checked (nil while it hasn't): a use
// another workspace made later takes nothing away.
func ProofFor(ctx context.Context, q *store.Queries, workspace uuid.UUID, host, instance string, held *time.Time) (Proof, error) {
	host, instance = strings.ToLower(host), strings.ToLower(instance)
	if host == instance {
		return Never, nil
	}
	claims, err := q.CustomHostClaims(ctx, store.CustomHostClaimsParams{Host: host, WorkspaceID: workspace})
	if err != nil {
		return Records, err
	}
	proof := Records
	for _, c := range claims {
		switch {
		case c.Purpose == PurposeInstance:
			return Never, nil
		case TakenBefore(c, held):
			proof = InUse
		}
	}
	if proof == Records && under(host, instance) {
		proof = UnderInstance
	}
	return proof, nil
}

// TakenBefore reports whether another workspace's use of a host keeps a
// workspace from it: any use while the workspace hasn't verified the host
// (held is nil), and afterwards one proven before the workspace's or at a
// time nobody recorded. A use proven later, or never, came second.
func TakenBefore(c store.CustomHostClaimsRow, held *time.Time) bool {
	if held == nil {
		return true
	}
	if !c.Verified {
		return false
	}
	return c.Since == nil || !c.Since.After(*held)
}

// under reports whether host is under the instance's own host.
func under(host, instance string) bool {
	return instance != "" && strings.HasSuffix(host, "."+instance)
}

// TXTName is where the TXT record that shows host is a workspace's lives.
func TXTName(host string) string { return "_reputecast." + host }

// Token is the value of that TXT record for workspace: a hash of the
// workspace and the host, so it proves nothing for any other workspace or
// any other host.
func Token(workspace uuid.UUID, host string) string {
	sum := sha256.Sum256([]byte("reputecast-domain:" + workspace.String() + ":" + host))
	return "reputecast-verify=" + hex.EncodeToString(sum[:12])
}

// HasToken looks up host's TXT record and reports whether it holds
// workspace's token. answered is false when DNS gave no answer (a timeout,
// a failing server), which shows nothing either way; a name with no such
// record is an answer.
func HasToken(ctx context.Context, dns dnscheck.Resolver, workspace uuid.UUID, host string) (found, answered bool) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	txts, err := dns.TXT(ctx, TXTName(host))
	switch {
	case err == nil:
		return slices.Contains(txts, Token(workspace, host)), true
	case dnscheck.IsNotFound(err):
		return false, true
	}
	return false, false
}
