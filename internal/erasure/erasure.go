// Package erasure is the registry of where ReputeCast keeps a contact's
// personal data, so erasing a contact and exporting what is stored about one
// reach every table (review 2026-11 DP-1, DP-2).
//
// Each package that stores personal data about contacts declares a Source:
// the tables and columns it holds, how to erase them, and what to export.
// internal/app collects the sources (wire_audience.go) and a test there
// fails when a table with personal-data columns is in no source and not
// exempted with a reason. The package imports nothing from ReputeCast but
// authz and i18n, so any package can register without an import cycle.
package erasure

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/i18n"
)

// Reason says why contacts are erased; it travels with contact.erased.
type Reason string

const (
	// ReasonTeam: a person (or an API key) erased them.
	ReasonTeam Reason = "team"
	// ReasonPolicy: the sunset policy's "delete after N months".
	ReasonPolicy Reason = "policy"
	// ReasonUnconfirmed: a double opt-in signup never confirmed in time.
	ReasonUnconfirmed Reason = "unconfirmed"
)

// Subject is one contact being erased, read before anything is changed so
// sources can still match rows by the address or phone number.
type Subject struct {
	ID    uuid.UUID
	Email string
	Phone string
}

// Batch is the contacts of one workspace erased together.
type Batch struct {
	WorkspaceID uuid.UUID
	Subjects    []Subject
	Reason      Reason
	At          time.Time
}

// IDs are the batch's contact ids.
func (b *Batch) IDs() []uuid.UUID {
	out := make([]uuid.UUID, len(b.Subjects))
	for i, s := range b.Subjects {
		out[i] = s.ID
	}
	return out
}

// Emails are the batch's addresses, lower-cased, without blanks.
func (b *Batch) Emails() []string {
	out := make([]string, 0, len(b.Subjects))
	for _, s := range b.Subjects {
		if e := strings.ToLower(strings.TrimSpace(s.Email)); e != "" {
			out = append(out, e)
		}
	}
	return out
}

// Phones are the batch's phone numbers (E.164), without blanks.
func (b *Batch) Phones() []string {
	out := make([]string, 0, len(b.Subjects))
	for _, s := range b.Subjects {
		if s.Phone != "" {
			out = append(out, s.Phone)
		}
	}
	return out
}

// Table is one table a source covers: the personal-data columns it clears
// (or the rows it deletes) and the ones it keeps on purpose, with why.
type Table struct {
	Name string
	// Action: "delete" (the rows go), "clear" (the columns are emptied),
	// "anonymize" (identifying columns go, the rest stays for statistics).
	Action string
	// Columns are the personal-data columns the action removes.
	Columns []string
	// Kept are personal-data-looking columns left as they are, with why
	// ("contact_id": "counts distinct clickers; the contact is scrubbed").
	Kept map[string]string
}

// Refusal is contacts a source won't let go: a live paid subscription in
// Stripe, say. Message takes {n}, the number refused.
type Refusal struct {
	IDs     []uuid.UUID
	Code    string
	Message i18n.Msg
}

// Export is one section of a contact's data export: a query whose rows are
// written as JSON objects, column by column. It gets $1 the contact id,
// $2 the workspace id, $3 the contact's address (lower-cased) and $4 the
// phone number. Needs, when set, is the permission a section also needs
// (paid membership needs earn:view); without it the section says so.
type Export struct {
	Section string
	SQL     string
	Needs   authz.Permission
}

// Source is one package's personal data about contacts.
type Source struct {
	Name   string
	Tables []Table
	// Guard, optional, names contacts that can't be erased now.
	Guard func(ctx context.Context, tx pgx.Tx, b *Batch) (*Refusal, error)
	// Erase removes the source's personal data for the batch. It runs before
	// the contacts themselves are scrubbed, so the address still matches.
	Erase   func(ctx context.Context, tx pgx.Tx, b *Batch) error
	Exports []Export
}

// Registry is every source, in a fixed order: sources by name, then the
// core source (the contacts table itself) last.
type Registry struct {
	sources []Source
	core    Source
}

// New builds the registry. Duplicate names are a programming error.
func New(core Source, sources ...Source) *Registry {
	seen := map[string]bool{core.Name: true}
	for _, s := range sources {
		if seen[s.Name] {
			panic(fmt.Sprintf("erasure: two sources named %q", s.Name))
		}
		seen[s.Name] = true
	}
	sorted := slices.Clone(sources)
	slices.SortFunc(sorted, func(a, b Source) int { return strings.Compare(a.Name, b.Name) })
	return &Registry{sources: sorted, core: core}
}

// Sources are all sources in the order they run, the core source last.
func (r *Registry) Sources() []Source {
	if r == nil {
		return nil
	}
	return append(slices.Clone(r.sources), r.core)
}

// Guard asks every source's guard; the refusals of all sources come back
// together.
func (r *Registry) Guard(ctx context.Context, tx pgx.Tx, b *Batch) ([]Refusal, error) {
	var out []Refusal
	for _, s := range r.Sources() {
		if s.Guard == nil {
			continue
		}
		ref, err := s.Guard(ctx, tx, b)
		if err != nil {
			return nil, fmt.Errorf("erasure guard %s: %w", s.Name, err)
		}
		if ref != nil && len(ref.IDs) > 0 {
			out = append(out, *ref)
		}
	}
	return out, nil
}

// Erase runs every source's Erase, the core source last.
func (r *Registry) Erase(ctx context.Context, tx pgx.Tx, b *Batch) error {
	if len(b.Subjects) == 0 {
		return nil
	}
	for _, s := range r.Sources() {
		if s.Erase == nil {
			continue
		}
		if err := s.Erase(ctx, tx, b); err != nil {
			return fmt.Errorf("erase %s: %w", s.Name, err)
		}
	}
	return nil
}

// Exec runs each statement with $1 the batch's ids, $2 its addresses, $3 its
// phone numbers and $4 the workspace: the shape most sources need. A CTE
// names the four parameters' types, so a statement needn't use them all.
func Exec(ctx context.Context, tx pgx.Tx, b *Batch, stmts ...string) error {
	ids, emails, phones := b.IDs(), b.Emails(), b.Phones()
	for _, q := range stmts {
		if _, err := tx.Exec(ctx, paramTypes+q, ids, emails, phones, b.WorkspaceID); err != nil {
			return fmt.Errorf("%w: %.80s", err, q)
		}
	}
	return nil
}

// paramTypes declares Exec's parameters for statements that don't use all.
const paramTypes = "WITH erasure_params AS (SELECT $1::uuid[], $2::text[], $3::text[], $4::uuid) "
