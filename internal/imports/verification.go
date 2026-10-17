package imports

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/store"
)

// The import preview's optional check with a verification service (design
// expansion, X14): the new addresses go to the service, its answers join
// the preview's groups (invalid ones are skipped, risky ones imported and
// tagged), and the import never needs it.

// NewAddresses returns the addresses the import would add as new contacts:
// the preview's rows to import and its risky rows (with typo fixes), each
// once; and the import's name as the use reads it (Mailchimp “Acme
// Customers”).
func (s *Service) NewAddresses(ctx context.Context, ws *store.Workspace, id uuid.UUID) (string, []string, error) {
	imp, err := s.get(ctx, ws, id)
	if err != nil {
		return "", nil, err
	}
	if imp.StagedAt == nil {
		return "", nil, api.Conflict("The rows aren't read from the source yet. Wait for the field mapping to load, then try again.")
	}
	custom, err := s.customFields(ctx, ws.ID)
	if err != nil {
		return "", nil, err
	}
	fm := s.effectiveMap(imp, custom)
	opts := parseOptions(imp.Options)
	cm := parseConsentMap(imp.ConsentMap, opts)
	now := s.now()
	seen := map[string]bool{}
	out := []string{}
	var after int32
	for {
		b, err := s.loadBatch(ctx, ws.ID, imp, fm, after, 5000)
		if err != nil {
			return "", nil, err
		}
		if len(b.rows) == 0 {
			break
		}
		for _, r := range b.rows {
			after = r.RowNo
			v := classify(r, b, cm, now)
			if v.Class != classImport && !isRisky(v.Class) {
				continue
			}
			e := strings.ToLower(strings.TrimSpace(r.Email))
			if v.Fixed != "" {
				e = strings.ToLower(v.Fixed)
			}
			if e == "" || seen[e] {
				continue
			}
			seen[e] = true
			out = append(out, e)
		}
	}
	return importLabel(imp), out, nil
}

// importLabel names an import by its platform and lists: Mailchimp “Acme
// Customers”.
func importLabel(imp store.Import) string {
	name := SourceName(imp.Source)
	conn := parseConnection(imp.Connection)
	opts := parseOptions(imp.Options)
	names := []string{}
	for _, c := range opts.Lists {
		for _, l := range conn.Lists {
			if l.ID == c.SourceID && l.Name != "" {
				names = append(names, "“"+l.Name+"”")
			}
		}
	}
	if len(names) == 0 {
		return name
	}
	return name + " " + strings.Join(names, ", ")
}

// AddressChecks gives a verification service's answers for an import's new
// addresses (internal/verification): the preview folds them into its
// groups and the import follows them. Nil leaves the import as it was.
type AddressChecks interface {
	ImportAnswers(ctx context.Context, ws *store.Workspace, importID uuid.UUID) (*ImportChecks, error)
}

// ImportChecks are a service's answers for one import.
type ImportChecks struct {
	JobID       uuid.UUID
	Service     string
	ServiceName string
	// Status is the job's: running while it checks, finished after.
	Status    string
	CheckedAt *time.Time
	Checked   int64
	Cost      float64
	// Answers by lower-cased address.
	Answers map[string]ImportAnswer
}

// ImportAnswer is one address's result: valid, invalid, risky or unknown,
// with the reason (catch_all, role, disposable, abuse, invalid_address,
// spam_trap) and the service's own code.
type ImportAnswer struct {
	Result, Reason, Code string
}

// ProblemService is the invalid reason of rows a verification service
// rejected.
const ProblemService = "rejected by the verification service"

// Risky classes only a verification service finds.
const (
	classCatchAll = "catch_all"
	classAbuse    = "abuse"
	classUnknown  = "unknown"
)

// serviceTag is the tag a service-risky row is imported with.
func serviceTag(class string) string {
	switch class {
	case classCatchAll:
		return "verify:catch-all"
	case classRole:
		return "verify:role"
	case classDisposable:
		return "verify:disposable"
	case classAbuse:
		return "verify:abuse"
	case classUnknown:
		return "verify:unknown"
	}
	return ""
}

// serviceClass is the risky class of a service's reason.
func serviceClass(reason string) string {
	switch reason {
	case "role":
		return classRole
	case "disposable":
		return classDisposable
	case "abuse":
		return classAbuse
	}
	return classCatchAll
}

// withCheck folds a service's answer into a row's verdict: an address it
// rejects is invalid (skipped), a risky or unknown one joins its risky
// group, flagged as the service's; where ReputeCast found the same, they
// agree.
func withCheck(v verdict, a *ImportAnswer) verdict {
	if a == nil {
		return v
	}
	switch {
	case v.Class == classImport:
		switch a.Result {
		case "invalid":
			v.Class, v.Problem, v.ByService = classInvalid, ProblemService, true
		case "risky":
			v.Class, v.ByService = serviceClass(a.Reason), true
		case "unknown":
			v.Class, v.ByService = classUnknown, true
		}
	case isRisky(v.Class):
		switch {
		case a.Result == "invalid":
			v.Class, v.Problem, v.ByService = classInvalid, ProblemService, true
		case a.Result == "risky" && serviceClass(a.Reason) == v.Class:
			v.Agree = true
		}
	}
	return v
}

func (c *ImportChecks) answer(email string) *ImportAnswer {
	if c == nil {
		return nil
	}
	if a, ok := c.Answers[strings.ToLower(strings.TrimSpace(email))]; ok {
		return &a
	}
	return nil
}

// checks reads the service's answers for an import; a failure leaves the
// preview and the import as they were without them.
func (s *Service) checks(ctx context.Context, ws *store.Workspace, id uuid.UUID) *ImportChecks {
	if s.Checks == nil {
		return nil
	}
	c, err := s.Checks.ImportAnswers(ctx, ws, id)
	if err != nil || c == nil || len(c.Answers) == 0 {
		return nil
	}
	return c
}

// ImportVerification is the preview's line about the service's check.
type ImportVerification struct {
	JobID       string     `json:"jobId"`
	Service     string     `json:"service"`
	ServiceName string     `json:"serviceName"`
	Status      string     `json:"status"`
	CheckedAt   *time.Time `json:"checkedAt,omitempty"`
	Checked     int64      `json:"checked"`
	Cost        float64    `json:"cost"`
	// Invalid rows the service rejected (skipped); Risky the rows it alone
	// found risky or couldn't answer; Fewer how many fewer new contacts.
	Invalid int64 `json:"invalid"`
	Risky   int64 `json:"risky"`
	Fewer   int64 `json:"fewer"`
}
