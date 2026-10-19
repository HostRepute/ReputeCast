package verification

import (
	"context"
	"errors"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/imports"
	"github.com/hostrepute/reputecast/internal/store"
)

// ImportAnswers gives the import preview and the import the answers of the
// latest check of an import's new addresses (the import's optional "Verify
// the new addresses"): nil when none ran.
func (s *Service) ImportAnswers(ctx context.Context, ws *store.Workspace, importID uuid.UUID) (*imports.ImportChecks, error) {
	j, err := s.Q.VerifLatestImportJob(ctx, store.VerifLatestImportJobParams{WorkspaceID: ws.ID, ImportID: &importID})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	rows, err := s.Q.VerifJobResults(ctx, j.ID)
	if err != nil {
		return nil, err
	}
	out := &imports.ImportChecks{JobID: j.ID, Service: j.Service, ServiceName: ServiceName(j.Service), Status: j.Status, CheckedAt: j.FinishedAt,
		Checked: int64(j.Checked), Cost: cost(int64(j.Checked), j.PriceMicros), Answers: make(map[string]imports.ImportAnswer, len(rows))}
	for _, r := range rows {
		out.Answers[strings.ToLower(r.Email)] = imports.ImportAnswer{Result: r.Result, Reason: r.Reason, Code: r.Code}
	}
	return out, nil
}
