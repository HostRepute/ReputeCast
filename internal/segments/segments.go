// Package segments implements audience segments (spec 5.10, 7.9): a
// condition builder with AND/OR groups compiled to parameterized SQL, a live
// count with the tier mix, the generated SQL shown to power users, and SQL
// mode (segments:sql) where people write read-only SQL over the segment
// views. Static segments keep a snapshot of their members.
package segments

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/countcache"
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/dependents"
	"github.com/hostrepute/reputecast/internal/store"
)

type Service struct {
	Pool  *pgxpool.Pool
	Q     *store.Queries
	Audit *audit.Logger
	SQL   *SQLRunner
	// Counts keeps previews' counts for a minute (review 2026-11 OP-12);
	// nil counts every time.
	Counts *countcache.Cache
	// CountTimeout bounds builder counts (SQL mode uses SQL.Timeout).
	CountTimeout time.Duration
	Now          func() time.Time
}

func (s *Service) now() time.Time {
	if s.Now != nil {
		return s.Now()
	}
	return time.Now()
}

func (s *Service) countTimeout() time.Duration {
	if s.CountTimeout > 0 {
		return s.CountTimeout
	}
	return 10 * time.Second
}

// SegmentTierMix splits a count by engagement tier.
type SegmentTierMix struct {
	Active  int64 `json:"active"`
	Cooling int64 `json:"cooling"`
	AtRisk  int64 `json:"atRisk"`
	Sunset  int64 `json:"sunset"`
}

// SegmentListItem is one saved segment.
type SegmentListItem struct {
	ID        string     `json:"id"`
	Name      string     `json:"name"`
	Mode      string     `json:"mode" enum:"builder,sql"`
	Refresh   string     `json:"refresh" enum:"dynamic,static"`
	Count     *int64     `json:"count,omitempty" doc:"Last known count"`
	CountedAt *time.Time `json:"countedAt,omitempty"`
	UpdatedAt time.Time  `json:"updatedAt"`
}

// SegmentList is the saved segments of a workspace.
type SegmentList struct {
	Items []SegmentListItem `json:"items"`
}

// List returns the saved segments with their last counts.
func (s *Service) List(ctx context.Context, ws *store.Workspace) (SegmentList, error) {
	rows, err := s.Q.ListSegments(ctx, ws.ID)
	if err != nil {
		return SegmentList{}, err
	}
	out := SegmentList{Items: make([]SegmentListItem, 0, len(rows))}
	for _, r := range rows {
		out.Items = append(out.Items, SegmentListItem{ID: r.ID.String(), Name: r.Name, Mode: r.Mode, Refresh: r.Refresh, Count: r.CachedCount, CountedAt: r.CountedAt, UpdatedAt: r.UpdatedAt})
	}
	return out, nil
}

// SegmentDetail is one segment for the builder.
type SegmentDetail struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Mode        string `json:"mode" enum:"builder,sql"`
	Refresh     string `json:"refresh" enum:"dynamic,static"`
	// Definition is the builder's groups (builder mode).
	Definition SegmentDefinition `json:"definition,omitempty" required:"false"`
	// SQL is the person's own SQL (SQL mode).
	SQL string `json:"sql"`
	// GeneratedSQL is the builder's SQL with values inlined.
	GeneratedSQL string         `json:"generatedSql"`
	Count        *int64         `json:"count,omitempty"`
	CountedAt    *time.Time     `json:"countedAt,omitempty"`
	CountMs      *int32         `json:"countMs,omitempty"`
	Tiers        SegmentTierMix `json:"tiers"`
	UpdatedAt    time.Time      `json:"updatedAt"`
	UpdatedBy    string         `json:"updatedBy"`
}

func (s *Service) Get(ctx context.Context, ws *store.Workspace, id uuid.UUID) (SegmentDetail, error) {
	seg, err := s.Q.GetSegment(ctx, store.GetSegmentParams{WorkspaceID: ws.ID, ID: id})
	if err != nil {
		return SegmentDetail{}, err
	}
	return s.detail(ctx, ws, seg)
}

func (s *Service) detail(ctx context.Context, ws *store.Workspace, seg store.Segment) (SegmentDetail, error) {
	out := SegmentDetail{
		ID: seg.ID.String(), Name: seg.Name, Description: seg.Description, Mode: seg.Mode, Refresh: seg.Refresh,
		SQL: seg.Sql, Count: seg.CachedCount, CountedAt: seg.CountedAt, CountMs: seg.CountMs, UpdatedAt: seg.UpdatedAt,
		Definition: SegmentDefinition{Groups: []SegmentGroup{}},
	}
	_ = json.Unmarshal(seg.Definition, &out.Definition)
	if out.Definition.Groups == nil {
		out.Definition.Groups = []SegmentGroup{}
	}
	for i := range out.Definition.Groups {
		if out.Definition.Groups[i].Conditions == nil {
			out.Definition.Groups[i].Conditions = []SegmentCondition{}
		}
	}
	_ = json.Unmarshal(seg.TierCounts, &out.Tiers)
	if seg.Mode == "builder" {
		custom, err := s.customFields(ctx, ws.ID)
		if err != nil {
			return SegmentDetail{}, err
		}
		out.GeneratedSQL = Compile(out.Definition, ws.ID, custom).Display()
	}
	if seg.UpdatedBy != nil {
		if u, err := s.Q.GetUser(ctx, *seg.UpdatedBy); err == nil {
			out.UpdatedBy = u.Name
		}
	}
	return out, nil
}

func (s *Service) customFields(ctx context.Context, ws uuid.UUID) (map[string]CustomField, error) {
	rows, err := s.Q.ListCustomFields(ctx, ws)
	if err != nil {
		return nil, err
	}
	out := make(map[string]CustomField, len(rows))
	for _, f := range rows {
		out[f.Key] = CustomField{Key: f.Key, Label: f.Label, Type: f.Type}
	}
	return out, nil
}

func (s *Service) lists(ctx context.Context, ws uuid.UUID) (map[uuid.UUID]string, error) {
	rows, err := s.Q.ListWorkspaceLists(ctx, ws)
	if err != nil {
		return nil, err
	}
	out := make(map[uuid.UUID]string, len(rows))
	for _, l := range rows {
		out[l.ID] = l.Name
	}
	return out, nil
}

// SegmentInput is what the builder saves or counts.
type SegmentInput struct {
	// Name is required to save; a preview doesn't need one.
	Name        string `json:"name,omitempty" maxLength:"120"`
	Description string `json:"description,omitempty" maxLength:"500"`
	Mode        string `json:"mode" enum:"builder,sql"`
	Refresh     string `json:"refresh,omitempty" enum:"dynamic,static"`
	// Definition is the builder's groups; SQL mode leaves it out.
	Definition SegmentDefinition `json:"definition,omitempty" required:"false"`
	SQL        string            `json:"sql,omitempty" maxLength:"20000"`
}

// validate checks the input and returns the compiled builder SQL or the
// checked user SQL.
func (s *Service) validate(ctx context.Context, ws *store.Workspace, in *SegmentInput, needName bool) (*Compiled, string, error) {
	in.Name = strings.TrimSpace(in.Name)
	if needName && in.Name == "" {
		return nil, "", invalid("body.name", says(mName))
	}
	if in.Refresh == "" {
		in.Refresh = "dynamic"
	}
	if in.Mode == "sql" {
		q, err := CheckSQL(in.SQL)
		if err != nil {
			return nil, "", err
		}
		in.SQL = q
		return nil, q, nil
	}
	custom, err := s.CheckDefinition(ctx, ws.ID, &in.Definition)
	if err != nil {
		return nil, "", err
	}
	c := Compile(in.Definition, ws.ID, custom)
	return &c, "", nil
}

// CheckDefinition validates a builder definition and returns the custom fields Compile needs.
func (s *Service) CheckDefinition(ctx context.Context, workspaceID uuid.UUID, d *SegmentDefinition) (map[string]CustomField, error) {
	custom, err := s.customFields(ctx, workspaceID)
	if err != nil {
		return nil, err
	}
	lists, err := s.lists(ctx, workspaceID)
	if err != nil {
		return nil, err
	}
	if d.Groups == nil {
		d.Groups = []SegmentGroup{}
	}
	if err := d.Validate(custom, lists); err != nil {
		return nil, err
	}
	return custom, nil
}

// SegmentCountResult is a live count.
type SegmentCountResult struct {
	Count int64          `json:"count"`
	Total int64          `json:"total" doc:"Contacts in the workspace"`
	Tiers SegmentTierMix `json:"tiers"`
	Ms    int64          `json:"ms" doc:"How long the count took"`
	// GeneratedSQL is the builder's SQL with values inlined (builder mode).
	GeneratedSQL string `json:"generatedSql"`
	// Cached: the same definition was counted within the last minute and
	// nothing this instance did has changed the audience since (review
	// 2026-11 OP-12). Ms is that count's time.
	Cached bool `json:"cached,omitempty"`
}

// Preview counts an unsaved definition or SQL.
func (s *Service) Preview(ctx context.Context, ws *store.Workspace, in SegmentInput) (SegmentCountResult, error) {
	compiled, sql, err := s.validate(ctx, ws, &in, false)
	if err != nil {
		return SegmentCountResult{}, err
	}
	asked := time.Now()
	out, at, err := countcache.Get(ctx, s.Counts, ws.ID, previewKey(compiled, sql), func(ctx context.Context) (SegmentCountResult, error) {
		return s.count(ctx, ws, compiled, sql)
	})
	out.Cached = err == nil && at.Before(asked)
	return out, err
}

// previewKey names a definition's count in the count cache: its query and
// arguments, or its SQL.
func previewKey(compiled *Compiled, userSQL string) string {
	h := sha256.New()
	if compiled != nil {
		q, args := compiled.Query(2)
		b, _ := json.Marshal(args)
		h.Write([]byte(q))
		h.Write([]byte{0})
		h.Write(b)
	} else {
		h.Write([]byte("sql\x00" + userSQL))
	}
	return "segment.preview:" + hex.EncodeToString(h.Sum(nil))
}

func (s *Service) count(ctx context.Context, ws *store.Workspace, compiled *Compiled, userSQL string) (SegmentCountResult, error) {
	var out SegmentCountResult
	total, _, err := countcache.Get(ctx, s.Counts, ws.ID, "contacts.total", func(ctx context.Context) (store.AudienceCountsRow, error) {
		return s.Q.AudienceCounts(ctx, ws.ID)
	})
	if err != nil {
		return out, err
	}
	out.Total = total.Contacts
	start := s.now()
	var n int64
	scan := func(row pgx.Row) error {
		return row.Scan(&n, &out.Tiers.Active, &out.Tiers.Cooling, &out.Tiers.AtRisk, &out.Tiers.Sunset)
	}
	if compiled != nil {
		out.GeneratedSQL = compiled.Display()
		q, args := compiled.Query(2)
		args = append([]any{ws.ID}, args...)
		err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
			if _, err := tx.Exec(ctx, fmt.Sprintf("SET LOCAL statement_timeout = %d", s.countTimeout().Milliseconds())); err != nil {
				return err
			}
			return scan(tx.QueryRow(ctx, withViews()+"SELECT "+tierCounts+"\nFROM contacts x WHERE x.id IN (SELECT seg.id FROM (\n"+q+"\n) seg)", args...))
		})
		err = mapSQLError(err)
	} else {
		err = s.SQL.Run(ctx, ws.ID, func(tx pgx.Tx) error {
			return scan(tx.QueryRow(ctx, "SELECT "+tierCounts+"\nFROM public.contacts x WHERE x.workspace_id = $1 AND x.erased_at IS NULL AND x.id IN (SELECT reputecast_sql.segment_ids($2))", ws.ID, userSQL))
		})
	}
	if err != nil {
		return out, err
	}
	out.Count = n
	out.Ms = s.now().Sub(start).Milliseconds()
	return out, nil
}

// Create saves a new segment and counts it.
func (s *Service) Create(ctx context.Context, ws *store.Workspace, actor *uuid.UUID, in SegmentInput) (SegmentDetail, error) {
	compiled, userSQL, err := s.validate(ctx, ws, &in, true)
	if err != nil {
		return SegmentDetail{}, err
	}
	res, countErr := s.count(ctx, ws, compiled, userSQL)
	if countErr != nil && !errors.Is(countErr, ErrTimeout) {
		return SegmentDetail{}, countErr
	}
	ids, err := s.snapshot(ctx, ws, in, compiled, userSQL)
	if err != nil {
		return SegmentDetail{}, err
	}
	def, _ := json.Marshal(in.Definition)
	var seg store.Segment
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		seg, err = q.CreateSegment(ctx, store.CreateSegmentParams{
			ID: uuid.Must(uuid.NewV7()), WorkspaceID: ws.ID, Name: in.Name, Description: in.Description, Mode: in.Mode,
			Refresh: in.Refresh, Definition: def, Sql: in.SQL, CreatedBy: actor,
		})
		if err != nil {
			return err
		}
		if err := s.storeCount(ctx, q, seg.ID, res, countErr == nil, ids); err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{
			WorkspaceID: &ws.ID, Action: "segment.create", TargetType: "segment", TargetID: seg.ID.String(), TargetName: seg.Name,
			After: map[string]any{"name": seg.Name, "mode": seg.Mode, "refresh": seg.Refresh, "definition": in.Definition, "sql": in.SQL},
		})
	})
	if err != nil {
		return SegmentDetail{}, err
	}
	return s.Get(ctx, ws, seg.ID)
}

// Update changes a segment and recounts it.
func (s *Service) Update(ctx context.Context, ws *store.Workspace, actor *uuid.UUID, id uuid.UUID, in SegmentInput) (SegmentDetail, error) {
	before, err := s.Q.GetSegment(ctx, store.GetSegmentParams{WorkspaceID: ws.ID, ID: id})
	if err != nil {
		return SegmentDetail{}, err
	}
	compiled, userSQL, err := s.validate(ctx, ws, &in, true)
	if err != nil {
		return SegmentDetail{}, err
	}
	res, countErr := s.count(ctx, ws, compiled, userSQL)
	if countErr != nil && !errors.Is(countErr, ErrTimeout) {
		return SegmentDetail{}, countErr
	}
	ids, err := s.snapshot(ctx, ws, in, compiled, userSQL)
	if err != nil {
		return SegmentDetail{}, err
	}
	def, _ := json.Marshal(in.Definition)
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		seg, err := q.UpdateSegment(ctx, store.UpdateSegmentParams{
			WorkspaceID: ws.ID, ID: id, Name: in.Name, Description: in.Description, Mode: in.Mode, Refresh: in.Refresh,
			Definition: def, Sql: in.SQL, UpdatedBy: actor,
		})
		if err != nil {
			return err
		}
		if err := s.storeCount(ctx, q, seg.ID, res, countErr == nil, ids); err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{
			WorkspaceID: &ws.ID, Action: "segment.update", TargetType: "segment", TargetID: seg.ID.String(), TargetName: seg.Name,
			Before: map[string]any{"name": before.Name, "mode": before.Mode, "refresh": before.Refresh, "definition": before.Definition, "sql": before.Sql},
			After:  map[string]any{"name": seg.Name, "mode": seg.Mode, "refresh": seg.Refresh, "definition": in.Definition, "sql": in.SQL},
		})
	})
	if err != nil {
		return SegmentDetail{}, err
	}
	return s.Get(ctx, ws, id)
}

// snapshot returns the matching contact ids of a static segment.
func (s *Service) snapshot(ctx context.Context, ws *store.Workspace, in SegmentInput, compiled *Compiled, userSQL string) ([]uuid.UUID, error) {
	if in.Refresh != "static" {
		return nil, nil
	}
	return s.ids(ctx, ws.ID, compiled, userSQL)
}

// storeCount keeps a static segment's members and, when counted is true,
// the count (a count that timed out keeps the last one).
func (s *Service) storeCount(ctx context.Context, q *store.Queries, id uuid.UUID, res SegmentCountResult, counted bool, ids []uuid.UUID) error {
	if ids != nil {
		// A diff, so members keep when they entered and the leavers are
		// recorded (review 2026-11 AU-16).
		if _, err := q.ReplaceSegmentMembers(ctx, store.ReplaceSegmentMembersParams{SegmentID: id, ContactIds: ids, At: s.now()}); err != nil {
			return err
		}
	}
	if !counted {
		return nil
	}
	tiers, _ := json.Marshal(res.Tiers)
	now := s.now()
	ms := int32(res.Ms)
	n := res.Count
	return q.SetSegmentCount(ctx, store.SetSegmentCountParams{ID: id, CachedCount: &n, CountedAt: &now, CountMs: &ms, TierCounts: tiers})
}

// Recount counts a saved segment and stores the result.
func (s *Service) Recount(ctx context.Context, ws *store.Workspace, id uuid.UUID) (SegmentCountResult, error) {
	seg, err := s.Q.GetSegment(ctx, store.GetSegmentParams{WorkspaceID: ws.ID, ID: id})
	if err != nil {
		return SegmentCountResult{}, err
	}
	in := SegmentInput{Name: seg.Name, Mode: seg.Mode, Refresh: seg.Refresh, SQL: seg.Sql}
	_ = json.Unmarshal(seg.Definition, &in.Definition)
	compiled, userSQL, err := s.validate(ctx, ws, &in, false)
	if err != nil {
		return SegmentCountResult{}, err
	}
	res, err := s.count(ctx, ws, compiled, userSQL)
	if err != nil {
		return res, err
	}
	return res, s.storeCount(ctx, s.Q, seg.ID, res, true, nil)
}

func (s *Service) Delete(ctx context.Context, ws *store.Workspace, id uuid.UUID) error {
	return db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		seg, err := q.GetSegment(ctx, store.GetSegmentParams{WorkspaceID: ws.ID, ID: id})
		if err != nil {
			return err
		}
		if _, err := q.DeleteSegment(ctx, store.DeleteSegmentParams{WorkspaceID: ws.ID, ID: id}); err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{
			WorkspaceID: &ws.ID, Action: "segment.delete", TargetType: "segment", TargetID: id.String(), TargetName: seg.Name,
			Before: map[string]any{"name": seg.Name, "mode": seg.Mode, "definition": seg.Definition, "sql": seg.Sql},
		})
	})
}

// SegmentImpact is what deleting a segment would affect: campaigns and
// digests sending to it, automations using it, saved views and other
// segments built on it.
type SegmentImpact struct {
	ID   string           `json:"id"`
	Name string           `json:"name"`
	Uses []dependents.Use `json:"uses"`
}

// Impact lists what uses a segment.
func (s *Service) Impact(ctx context.Context, ws *store.Workspace, id uuid.UUID) (SegmentImpact, error) {
	seg, err := s.Q.GetSegment(ctx, store.GetSegmentParams{WorkspaceID: ws.ID, ID: id})
	if err != nil {
		return SegmentImpact{}, err
	}
	idx, err := dependents.Load(ctx, s.Pool, ws.ID)
	if err != nil {
		return SegmentImpact{}, err
	}
	return SegmentImpact{ID: seg.ID.String(), Name: seg.Name, Uses: idx.Segment(seg.ID)}, nil
}

// Condition is a saved segment as a condition on contacts aliased c, for
// a query whose $1 is the workspace, its own placeholders numbered from
// first: the snapshot for a static segment, the rules as a subquery for a
// dynamic one, so its members never travel through Go (review 2026-11
// AU-3). A SQL segment, which runs as its own restricted role, is
// evaluated first and comes back as ids.
func (s *Service) Condition(ctx context.Context, workspaceID, segmentID uuid.UUID, first int) (string, []any, error) {
	seg, err := s.Q.GetSegment(ctx, store.GetSegmentParams{WorkspaceID: workspaceID, ID: segmentID})
	if err != nil {
		return "", nil, err
	}
	switch {
	case seg.Refresh == "static":
		return fmt.Sprintf("c.id IN (SELECT sm.contact_id FROM segment_members sm WHERE sm.segment_id = $%d)", first), []any{seg.ID}, nil
	case seg.Mode == "sql":
		ids, err := s.ids(ctx, workspaceID, nil, seg.Sql)
		if err != nil {
			return "", nil, err
		}
		return fmt.Sprintf("c.id = ANY ($%d)", first), []any{ids}, nil
	}
	var def SegmentDefinition
	_ = json.Unmarshal(seg.Definition, &def)
	custom, err := s.customFields(ctx, workspaceID)
	if err != nil {
		return "", nil, err
	}
	q, args := Compile(def, workspaceID, custom).Query(first)
	return "c.id IN (" + withViews() + "SELECT seg.id FROM (\n" + q + "\n) seg)", args, nil
}

// ContactIDs resolves a saved segment to contact ids: the snapshot for
// static segments, a fresh evaluation for dynamic ones. The send pipeline
// still excludes suppressed and role addresses on its own.
func (s *Service) ContactIDs(ctx context.Context, workspaceID, segmentID uuid.UUID) ([]uuid.UUID, error) {
	seg, err := s.Q.GetSegment(ctx, store.GetSegmentParams{WorkspaceID: workspaceID, ID: segmentID})
	if err != nil {
		return nil, err
	}
	if seg.Refresh == "static" {
		return s.Q.SegmentMemberIDs(ctx, seg.ID)
	}
	if seg.Mode == "sql" {
		return s.ids(ctx, workspaceID, nil, seg.Sql)
	}
	var def SegmentDefinition
	_ = json.Unmarshal(seg.Definition, &def)
	custom, err := s.customFields(ctx, workspaceID)
	if err != nil {
		return nil, err
	}
	c := Compile(def, workspaceID, custom)
	return s.ids(ctx, workspaceID, &c, "")
}

func (s *Service) ids(ctx context.Context, ws uuid.UUID, compiled *Compiled, userSQL string) ([]uuid.UUID, error) {
	out := []uuid.UUID{}
	collect := func(rows pgx.Rows, err error) error {
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var id uuid.UUID
			if err := rows.Scan(&id); err != nil {
				return err
			}
			out = append(out, id)
		}
		return rows.Err()
	}
	if compiled != nil {
		q, args := compiled.Query(2)
		args = append([]any{ws}, args...)
		err := db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
			if _, err := tx.Exec(ctx, fmt.Sprintf("SET LOCAL statement_timeout = %d", (3*s.countTimeout()).Milliseconds())); err != nil {
				return err
			}
			return collect(tx.Query(ctx, withViews()+"SELECT seg.id FROM (\n"+q+"\n) seg", args...))
		})
		return out, mapSQLError(err)
	}
	err := s.SQL.Run(ctx, ws, func(tx pgx.Tx) error {
		return collect(tx.Query(ctx, "SELECT DISTINCT x.id FROM public.contacts x WHERE x.workspace_id = $1 AND x.erased_at IS NULL AND x.id IN (SELECT reputecast_sql.segment_ids($2))", ws, userSQL))
	})
	return out, err
}

// RecountAll refreshes the stored count of every dynamic segment of a
// workspace (the nightly job). Counts that time out keep their last value.
func (s *Service) RecountAll(ctx context.Context, ws *store.Workspace) error {
	rows, err := s.Q.ListSegments(ctx, ws.ID)
	if err != nil {
		return err
	}
	for _, r := range rows {
		if r.Refresh == "static" {
			continue
		}
		if _, err := s.Recount(ctx, ws, r.ID); err != nil && !errors.Is(err, ErrTimeout) && !errors.Is(err, ErrSQLDisabled) {
			var se *SQLError
			if !errors.As(err, &se) {
				return err
			}
		}
	}
	return nil
}
