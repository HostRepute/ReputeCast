package coupons

import (
	"bytes"
	"context"
	"errors"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/csvexport"
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/store"
)

var codeRe = regexp.MustCompile(`^[A-Za-z0-9_-]{3,64}$`)

// insertChunk is how many codes one insert statement takes.
const insertChunk = 5000

// CouponCodesResult is what an upload added.
type CouponCodesResult struct {
	Added int64 `json:"added"`
	// Duplicates were in the upload twice or in the pool already.
	Duplicates int64 `json:"duplicates"`
	// Invalid aren't 3–64 letters, digits, - or _; InvalidLines are the
	// first three of them.
	Invalid      int64               `json:"invalid"`
	InvalidLines []CouponInvalidLine `json:"invalidLines"`
	Free         int64               `json:"free"`
}

// CouponInvalidLine is a line of an upload that isn't a code.
type CouponInvalidLine struct {
	Line int    `json:"line" doc:"1 is the first line"`
	Text string `json:"text" doc:"The line, up to 40 characters"`
}

// CouponRemoveResult is how many unused codes were removed.
type CouponRemoveResult struct {
	Removed int64 `json:"removed"`
}

// parseCodes reads codes from text or CSV: one per line, the first column
// of a CSV, a header row named "code" skipped, quotes taken off, empty
// lines skipped. lines are the first three that aren't codes.
func parseCodes(body []byte) (codes []string, repeated, bad int64, lines []CouponInvalidLine) {
	lines = []CouponInvalidLine{}
	seen := map[string]bool{}
	for i, line := range strings.Split(strings.ReplaceAll(string(body), "\r\n", "\n"), "\n") {
		line = strings.TrimSpace(strings.TrimPrefix(line, "\ufeff"))
		if c := strings.IndexAny(line, ",;\t"); c >= 0 {
			line = strings.TrimSpace(line[:c])
		}
		line = strings.TrimSpace(strings.Trim(line, `"'`))
		if line == "" || (i == 0 && strings.EqualFold(line, "code")) {
			continue
		}
		if !codeRe.MatchString(line) {
			bad++
			if len(lines) < 3 {
				lines = append(lines, CouponInvalidLine{Line: i + 1, Text: clipRunes(line, 40)})
			}
			continue
		}
		key := strings.ToUpper(line)
		if seen[key] {
			repeated++
			continue
		}
		seen[key] = true
		codes = append(codes, line)
	}
	return codes, repeated, bad, lines
}

// insertCodes seals and stores codes in a batch, skipping those the pool
// has; it returns how many were added. externalIDs, when given, are the
// store's ids of the codes.
func (s *Service) insertCodes(ctx context.Context, q *store.Queries, p store.CouponPoolsRow, batch uuid.UUID, codes []string, externalIDs []string) (int64, error) {
	var added int64
	now := s.now()
	for start := 0; start < len(codes); start += insertChunk {
		end := min(start+insertChunk, len(codes))
		n := end - start
		params := store.CouponCodesInsertParams{WorkspaceID: p.WorkspaceID, PoolID: p.ID, BatchID: batch, Now: now,
			Ids: make([]uuid.UUID, n), CodesSealed: make([][]byte, n), CodeHashes: make([][]byte, n), ExternalIds: make([]string, n)}
		for i, c := range codes[start:end] {
			sealed, err := s.Box.SealString(c, codeAD(p.ID))
			if err != nil {
				return added, err
			}
			params.Ids[i], params.CodesSealed[i], params.CodeHashes[i] = newID(), sealed, CodeHash(p.WorkspaceID, c)
			if externalIDs != nil {
				params.ExternalIds[i] = externalIDs[start+i]
			}
		}
		ids, err := q.CouponCodesInsert(ctx, params)
		if err != nil {
			return added, err
		}
		added += int64(len(ids))
	}
	return added, nil
}

// Upload adds codes to an upload pool from text or a CSV. Codes are sealed
// and only the counts go in the audit log.
func (s *Service) Upload(ctx context.Context, ws *store.Workspace, id uuid.UUID, body []byte) (CouponCodesResult, error) {
	p, err := s.pool(ctx, ws.ID, id)
	if err != nil {
		return CouponCodesResult{}, err
	}
	switch {
	case p.Status != StatusActive:
		return CouponCodesResult{}, conflict(ctx, "pool_archived", mPoolArchived, i18n.Args{"name": p.Name})
	case p.Source != SourceUpload:
		return CouponCodesResult{}, conflict(ctx, "store_pool", mCodesStorePool, i18n.Args{"name": p.Name, "store": p.StoreName})
	}
	if len(body) > MaxUploadBytes {
		return CouponCodesResult{}, invalid(ctx, "body", mCodesTooMany, nil)
	}
	codes, repeated, bad, lines := parseCodes(body)
	if len(codes) == 0 {
		return CouponCodesResult{}, invalid(ctx, "body", mCodesNone, nil)
	}
	if len(codes) > MaxUploadCodes {
		return CouponCodesResult{}, invalid(ctx, "body", mCodesTooMany, nil)
	}
	out := CouponCodesResult{Invalid: bad, InvalidLines: lines}
	by := actorUser(api.PrincipalFrom(ctx))
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		now := s.now()
		batch, err := q.CouponBatchInsert(ctx, store.CouponBatchInsertParams{ID: newID(), WorkspaceID: ws.ID, PoolID: p.ID, Kind: "upload",
			Requested: int32(len(codes)), Status: "running", CreatedBy: by, Now: now})
		if err != nil {
			return err
		}
		if out.Added, err = s.insertCodes(ctx, q, p, batch, codes, nil); err != nil {
			return err
		}
		out.Duplicates = repeated + int64(len(codes)) - out.Added
		if err := q.CouponBatchFinish(ctx, store.CouponBatchFinishParams{ID: batch, Status: "done", Created: int32(out.Added), Failed: int32(out.Duplicates), At: now}); err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "coupon_pool.codes_uploaded", TargetType: "coupon_pool", TargetID: p.ID.String(),
			TargetName: p.Name, Metadata: map[string]any{"summary": strconv.FormatInt(out.Added, 10) + " codes uploaded", "added": out.Added,
				"duplicates": out.Duplicates, "invalid": out.Invalid}})
	})
	if err != nil {
		return CouponCodesResult{}, err
	}
	out.Free, _ = s.Q.CouponPoolFree(ctx, p.ID)
	s.recheckLow(ctx, p, out.Free)
	return out, nil
}

// RemoveUnused removes every code nobody got yet; a store pool's codes are
// then removed in the store too (RemoveArgs).
func (s *Service) RemoveUnused(ctx context.Context, ws *store.Workspace, id uuid.UUID) (CouponRemoveResult, error) {
	p, err := s.pool(ctx, ws.ID, id)
	if err != nil {
		return CouponRemoveResult{}, err
	}
	var out CouponRemoveResult
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if out.Removed, err = q.CouponCodesRemoveFree(ctx, store.CouponCodesRemoveFreeParams{PoolID: p.ID, Now: s.now()}); err != nil {
			return err
		}
		if out.Removed > 0 && p.Source != SourceUpload {
			if c := s.jobs(); c != nil {
				if _, err := c.InsertTx(ctx, tx, RemoveArgs{PoolID: p.ID.String()}, nil); err != nil {
					return err
				}
			}
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "coupon_pool.codes_removed", TargetType: "coupon_pool", TargetID: p.ID.String(),
			TargetName: p.Name, Metadata: map[string]any{"summary": strconv.FormatInt(out.Removed, 10) + " unused codes removed", "removed": out.Removed}})
	})
	return out, err
}

// Export is the pool's codes as CSV (code, status, assigned_at,
// redeemed_at, order), audited with its row count.
func (s *Service) Export(ctx context.Context, ws *store.Workspace, id uuid.UUID) ([]byte, string, error) {
	p, err := s.pool(ctx, ws.ID, id)
	if err != nil {
		return nil, "", err
	}
	rows, err := s.Q.CouponCodesExport(ctx, p.ID)
	if err != nil {
		return nil, "", err
	}
	var buf bytes.Buffer
	w := csvexport.NewWriter(&buf)
	_ = w.Write([]string{"code", "status", "assigned_at", "redeemed_at", "order"})
	at := func(t *time.Time) string {
		if t == nil {
			return ""
		}
		return t.UTC().Format(time.RFC3339)
	}
	for _, r := range rows {
		code, err := s.Box.OpenString(r.CodeSealed, codeAD(p.ID))
		if err != nil {
			return nil, "", err
		}
		status := "free"
		switch {
		case r.RedeemedAt != nil:
			status = "redeemed"
		case r.AssignedAt != nil:
			status = "sent"
		case r.RemovedAt != nil:
			status = "removed"
		}
		if err := w.Write([]string{code, status, at(r.AssignedAt), at(r.RedeemedAt), r.OrderName}); err != nil {
			return nil, "", err
		}
	}
	w.Flush()
	if err := w.Error(); err != nil {
		return nil, "", err
	}
	if err := s.Audit.Log(ctx, audit.Event{WorkspaceID: &ws.ID, Action: "coupon_pool.codes_exported", TargetType: "coupon_pool", TargetID: p.ID.String(),
		TargetName: p.Name, Metadata: map[string]any{"rows": len(rows)}}); err != nil {
		return nil, "", err
	}
	return buf.Bytes(), "discount-codes-" + p.Slug + "-" + s.now().UTC().Format("2006-01-02") + ".csv", nil
}

// recheckLow clears a pool's low alert once it has its low mark again.
func (s *Service) recheckLow(ctx context.Context, p store.CouponPoolsRow, free int64) {
	if free >= int64(p.LowAt) {
		if err := s.Q.CouponPoolClearLow(ctx, p.ID); err != nil && !errors.Is(err, context.Canceled) {
			s.log().WarnContext(ctx, "coupons: clear the low alert", "pool", p.ID, "err", err)
		}
	}
}
