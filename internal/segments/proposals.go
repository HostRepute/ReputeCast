package segments

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/store"
)

// ProjectDefinition leaves every comparison literal and custom field key on
// the instance. The returned bindings are local, not model context.
func ProjectDefinition(d SegmentDefinition) (SegmentDefinition, map[string]string) {
	out := SegmentDefinition{Groups: make([]SegmentGroup, 0, len(d.Groups))}
	bindings := map[string]string{}
	bind := func(value string) string {
		if value == "" {
			return ""
		}
		if symbol, ok := bindings[value]; ok {
			return symbol
		}
		symbol := fmt.Sprintf("bound_%d", len(bindings)+1)
		bindings[value] = symbol
		return symbol
	}
	for _, group := range d.Groups {
		g := SegmentGroup{Match: group.Match, Conditions: []SegmentCondition{}}
		for _, c := range group.Conditions {
			if strings.HasPrefix(c.Field, "custom:") {
				c.Field = "custom:" + bind(strings.TrimPrefix(c.Field, "custom:"))
			}
			c.Value, c.To = bind(c.Value), bind(c.To)
			g.Conditions = append(g.Conditions, c)
		}
		out.Groups = append(out.Groups, g)
	}
	return out, bindings
}

// RestoreDefinition accepts only previously selected literals; callers still
// validate field/operator combinations and current access before staging.
func RestoreDefinition(d SegmentDefinition, bindings map[string]string) (SegmentDefinition, error) {
	if len(d.Groups) == 0 || len(d.Groups) > 5 {
		return d, errors.New("invalid audience")
	}
	values := map[string]string{"": ""}
	for value, symbol := range bindings {
		values[symbol] = value
	}
	out := SegmentDefinition{Groups: make([]SegmentGroup, 0, len(d.Groups))}
	for _, group := range d.Groups {
		if len(group.Conditions) == 0 || len(group.Conditions) > 10 {
			return out, errors.New("invalid audience")
		}
		g := SegmentGroup{Match: group.Match, Conditions: []SegmentCondition{}}
		for _, c := range group.Conditions {
			if strings.HasPrefix(c.Field, "custom:") {
				field, ok := values[strings.TrimPrefix(c.Field, "custom:")]
				if !ok || field == "" {
					return out, errors.New("unknown field binding")
				}
				c.Field = "custom:" + field
			}
			value, ok := values[c.Value]
			to, toOK := values[c.To]
			if !ok || !toOK {
				return out, errors.New("unknown value binding")
			}
			c.Value, c.To = value, to
			g.Conditions = append(g.Conditions, c)
		}
		out.Groups = append(out.Groups, g)
	}
	return out, nil
}

// CheckProposal applies the normal builder validator without reading contact
// values or building a catalog from them. Segment audiences, like segment
// triggers, require unrestricted list scope; a list filter is not a scope grant.
func (s *Service) CheckProposal(ctx context.Context, q *store.Queries, wsID uuid.UUID, p *authz.Principal, d SegmentDefinition) (map[string]CustomField, error) {
	if p.WorkspaceID != wsID || !p.Can(authz.SubscribersView) || len(p.Scope.Lists) != 0 {
		return nil, api.Forbidden(authz.SubscribersView, nil)
	}
	local := *s
	local.Q = q
	return local.CheckDefinition(ctx, wsID, &d)
}

// PreviewProposal evaluates one human-selected builder audience. Both counts
// share a statement snapshot in a bounded READ ONLY transaction. It returns
// no members, field samples, dimensions or generated SQL.
func (s *Service) PreviewProposal(ctx context.Context, ws *store.Workspace, p *authz.Principal, d SegmentDefinition) (count, total int64, err error) {
	tx, err := s.Pool.BeginTx(ctx, pgx.TxOptions{AccessMode: pgx.ReadOnly})
	if err != nil {
		return 0, 0, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	custom, err := s.CheckProposal(ctx, store.New(tx), ws.ID, p, d)
	if err != nil {
		return 0, 0, err
	}
	if _, err := tx.Exec(ctx, fmt.Sprintf("SET LOCAL statement_timeout = %d", s.countTimeout().Milliseconds())); err != nil {
		return 0, 0, err
	}
	compiled := Compile(d, ws.ID, custom)
	query, args := compiled.Query(2)
	args = append([]any{ws.ID}, args...)
	err = tx.QueryRow(ctx, withViews()+"SELECT (SELECT count(*) FROM ("+query+") audience), (SELECT count(*) FROM contacts)", args...).Scan(&count, &total)
	if err != nil {
		return 0, 0, mapSQLError(err)
	}
	return count, total, tx.Commit(ctx)
}

// KeepProposalIn creates a new dynamic segment as part of a human bundle
// decision. It never changes an existing segment, materializes membership,
// emits entry events or enrolls contacts. The ordinary refresh/publish paths
// own those operations after this transaction.
func (s *Service) KeepProposalIn(ctx context.Context, tx pgx.Tx, ws *store.Workspace, p *authz.Principal, name string, d SegmentDefinition, origin audit.ProposalOrigin) (uuid.UUID, error) {
	if !p.IsHuman() || !p.Can(authz.SegmentsEdit) {
		return uuid.Nil, api.Forbidden(authz.SegmentsEdit, nil)
	}
	if strings.TrimSpace(name) == "" || len([]rune(name)) > 120 || len(d.Groups) == 0 {
		return uuid.Nil, invalid("body.name", says(mName))
	}
	q := store.New(tx)
	if _, err := s.CheckProposal(ctx, q, ws.ID, p, d); err != nil {
		return uuid.Nil, err
	}
	raw, err := json.Marshal(d)
	if err != nil {
		return uuid.Nil, err
	}
	id := uuid.Must(uuid.NewV7())
	if _, err := q.CreateSegment(ctx, store.CreateSegmentParams{ID: id, WorkspaceID: ws.ID, Name: name, Mode: "builder", Refresh: "dynamic", Definition: raw, CreatedBy: &p.UserID}); err != nil {
		return uuid.Nil, err
	}
	err = s.Audit.LogWith(audit.WithPrincipal(ctx, p), q, audit.Event{WorkspaceID: &ws.ID, Action: "segment.create", TargetType: "segment", TargetID: id.String(),
		Metadata: origin.Metadata()})
	return id, err
}
