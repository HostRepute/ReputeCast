package placement

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/riverqueue/river"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/placement/seedbox"
	"github.com/hostrepute/reputecast/internal/store"
)

// Why a copy wasn't looked for (placement_copies.not_checked).
const (
	ReasonSignInFailed  = "signin_failed"
	ReasonSignInExpired = "signin_expired"
	ReasonSeedRemoved   = "seed_removed"
	ReasonStopped       = "stopped"
	ReasonUnsent        = "unsent"
)

// CheckArgs looks for the copies of every test that is sending or waiting,
// and for late copies of tests that finished within the hour.
type CheckArgs struct{}

// Kind implements river.JobArgs.
func (CheckArgs) Kind() string { return "placement.check" }

// CheckWorker runs CheckArgs.
type CheckWorker struct {
	river.WorkerDefaults[CheckArgs]
	Svc *Service
}

// Timeout leaves room for slow mailboxes (each sign-in has its own timeout).
func (w *CheckWorker) Timeout(*river.Job[CheckArgs]) time.Duration { return 10 * time.Minute }

// Work implements river.Worker.
func (w *CheckWorker) Work(ctx context.Context, _ *river.Job[CheckArgs]) error {
	return w.Svc.CheckOpen(ctx)
}

// DeleteArgs deletes the copies that are due to go from the seeds.
type DeleteArgs struct{}

// Kind implements river.JobArgs.
func (DeleteArgs) Kind() string { return "placement.delete_copies" }

// DeleteWorker runs DeleteArgs.
type DeleteWorker struct {
	river.WorkerDefaults[DeleteArgs]
	Svc *Service
}

// Timeout leaves room for slow mailboxes.
func (w *DeleteWorker) Timeout(*river.Job[DeleteArgs]) time.Duration { return 10 * time.Minute }

// Work implements river.Worker.
func (w *DeleteWorker) Work(ctx context.Context, _ *river.Job[DeleteArgs]) error {
	return w.Svc.DeleteDue(ctx)
}

// CheckOpen looks for the copies of the open tests, one test at a time.
func (s *Service) CheckOpen(ctx context.Context) error {
	tests, err := s.Q.PlacementTestsOpen(ctx, ptrTime(s.now()))
	if err != nil {
		return err
	}
	for _, t := range tests {
		if err := s.Check(ctx, t.ID); err != nil {
			s.log().WarnContext(ctx, "placement: check a test", "test", t.ID, "err", err)
		}
	}
	return nil
}

// Check signs in to the seeds of a test, finds its copies by their test
// token and records where each landed. A test waiting for its copies
// finishes when every copy is found or 15 minutes after the last hand-off;
// afterwards it keeps looking for late copies for an hour.
func (s *Service) Check(ctx context.Context, id uuid.UUID) error {
	t, err := s.Q.PlacementTestByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}
		return err
	}
	now := s.now()
	if t.Status == "stopped" || (t.Status == "finished" && (t.LateUntil == nil || !now.Before(*t.LateUntil))) {
		return nil
	}
	ws, err := s.Q.GetWorkspace(ctx, t.WorkspaceID)
	if err != nil {
		return err
	}
	copies, err := s.Q.PlacementCopies(ctx, id)
	if err != nil {
		return err
	}
	bySeed := map[uuid.UUID][]store.PlacementCopy{}
	order := []uuid.UUID{}
	for _, c := range copies {
		if c.SeedID == nil || c.HandedAt == nil {
			continue
		}
		look := c.Status == "sent" || (c.Status == "not_arrived" && c.NotChecked == "" && now.Before(c.HandedAt.Add(LateWithin)))
		if !look {
			continue
		}
		if bySeed[*c.SeedID] == nil {
			order = append(order, *c.SeedID)
		}
		bySeed[*c.SeedID] = append(bySeed[*c.SeedID], c)
	}
	for _, sid := range order {
		s.checkSeed(ctx, &ws, t, sid, bySeed[sid])
	}
	if t.Status == "finished" {
		return nil
	}
	return s.maybeFinish(ctx, &ws, id)
}

// checkSeed finds the copies of one seed.
func (s *Service) checkSeed(ctx context.Context, ws *store.Workspace, t store.PlacementTest, seedID uuid.UUID, cs []store.PlacementCopy) {
	seed, err := s.Q.PlacementSeed(ctx, store.PlacementSeedParams{ID: seedID, WorkspaceID: ws.ID})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			_ = s.Q.PlacementCopiesNotChecked(ctx, store.PlacementCopiesNotCheckedParams{TestID: t.ID, SeedID: &seedID, Reason: ReasonSeedRemoved})
		}
		return
	}
	box, err := s.openSeed(ctx, seed)
	if err != nil {
		s.seedTrouble(ctx, ws, t, seed, err)
		return
	}
	defer func() { _ = box.Close() }()
	tokens := make([]string, 0, 2*len(cs))
	since := t.StartedAt
	for _, c := range cs {
		tokens = append(tokens, c.Token, c.ID.String())
		if c.HandedAt != nil && c.HandedAt.Before(since) {
			since = *c.HandedAt
		}
	}
	found, err := box.Find(ctx, tokens, since.Add(-10*time.Minute))
	if err != nil {
		s.seedTrouble(ctx, ws, t, seed, err)
		return
	}
	var folders []PlacementFolder
	_ = json.Unmarshal(seed.Folders, &folders)
	now := s.now()
	late := []string{}
	for _, f := range found {
		var c *store.PlacementCopy
		for i := range cs {
			if strings.EqualFold(cs[i].Token, f.Token) || strings.EqualFold(cs[i].ID.String(), f.Token) {
				c = &cs[i]
				break
			}
		}
		if c == nil {
			continue
		}
		if err := s.recordFound(ctx, *c, f, folders, now); err != nil {
			s.log().WarnContext(ctx, "placement: record a copy", "copy", c.ID, "err", err)
			continue
		}
		if c.Status == "not_arrived" {
			late = append(late, c.Token)
		}
	}
	// Copies found after the test's copies were deleted go at once.
	if len(late) > 0 && t.CopiesDeletedAt != nil {
		if _, err := box.Delete(ctx, late, since.Add(-10*time.Minute)); err == nil {
			_ = s.Q.PlacementCopiesDeleted(ctx, store.PlacementCopiesDeletedParams{TestID: t.ID, SeedID: &seed.ID, At: &now})
		}
	}
}

// recordFound saves where a copy landed and what its headers say.
func (s *Service) recordFound(ctx context.Context, c store.PlacementCopy, f seedbox.Found, folders []PlacementFolder, now time.Time) error {
	folder := f.Role
	label := f.Folder
	for _, fl := range folders {
		if fl.Name == f.Folder {
			label = fl.Label
		}
	}
	received := f.ReceivedAt
	if received.IsZero() || (c.HandedAt != nil && received.Before(c.HandedAt.Add(-time.Minute))) {
		received = now
	}
	ab, _ := json.Marshal(f.Header.AuthResults())
	hb := []byte("{}")
	if hop := f.Header.HandOver(); hop != nil {
		hb, _ = json.Marshal(hop)
	}
	rb, _ := json.Marshal(f.Ref)
	return s.Q.PlacementCopyFound(ctx, store.PlacementCopyFoundParams{ID: c.ID, Folder: folder, FolderName: label, Tab: f.Tab, Late: c.Status == "not_arrived",
		ReceivedAt: &received, FoundAt: &now, Auth: ab, Hop: hb, Headers: clip(string(f.Header.Raw), 64<<10), Ref: rb})
}

// seedTrouble handles a seed that couldn't be read: a refused or ended
// sign-in marks its copies Not checked and the seed failed (and tells
// managers once); anything else is tried again on the next round.
func (s *Service) seedTrouble(ctx context.Context, ws *store.Workspace, t store.PlacementTest, seed store.PlacementSeed, err error) {
	reason := ""
	switch {
	case errors.Is(err, seedbox.ErrAuthExpired):
		reason = ReasonSignInExpired
	case errors.Is(err, seedbox.ErrAuthFailed):
		reason = ReasonSignInFailed
	}
	if reason == "" {
		s.log().WarnContext(ctx, "placement: read a seed", "seed", seed.Email, "test", t.ID, "err", err)
		return
	}
	s.seedFailed(ctx, ws, seed, err)
	_ = s.Q.PlacementCopiesNotChecked(ctx, store.PlacementCopiesNotCheckedParams{TestID: t.ID, SeedID: &seed.ID, Reason: reason})
}

// maybeFinish ends a waiting test when nothing is left to wait for.
func (s *Service) maybeFinish(ctx context.Context, ws *store.Workspace, id uuid.UUID) error {
	t, err := s.Q.PlacementTestByID(ctx, id)
	if err != nil || t.Status != "waiting" {
		return err
	}
	copies, err := s.Q.PlacementCopies(ctx, id)
	if err != nil {
		return err
	}
	pending := 0
	var lastHanded time.Time
	for _, c := range copies {
		if c.Status == "queued" || c.Status == "sent" {
			pending++
		}
		if c.HandedAt != nil && c.HandedAt.After(lastHanded) {
			lastHanded = *c.HandedAt
		}
	}
	now := s.now()
	if pending > 0 && (t.WaitUntil == nil || now.Before(*t.WaitUntil)) {
		return nil
	}
	finished := now
	if pending > 0 && t.WaitUntil != nil {
		finished = *t.WaitUntil
	}
	lateUntil := finished
	if !lastHanded.IsZero() && pending > 0 {
		lateUntil = lastHanded.Add(LateWithin)
	}
	deleteAt := deleteTime(t.DeleteAfter, finished)
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if err := q.PlacementCopiesNotArrived(ctx, store.PlacementCopiesNotArrivedParams{TestID: id, Reason: ""}); err != nil {
			return err
		}
		return q.PlacementTestSetStatus(ctx, store.PlacementTestSetStatusParams{ID: id, Status: "finished", FinishedAt: &finished, LateUntil: &lateUntil, DeleteAt: deleteAt})
	})
	if err != nil {
		return err
	}
	t.Status, t.FinishedAt = "finished", &finished
	if s.Notify != nil && t.StartedBy != nil {
		s.Notify.TestFinished(ctx, ws, t, s.summaryFor(ctx, t))
	}
	if deleteAt != nil && !deleteAt.After(now) {
		return s.deleteDueTest(ctx, ws, t)
	}
	return nil
}

// deleteTime is when copies go from the seeds after a test ends.
func deleteTime(setting string, finished time.Time) *time.Time {
	switch setting {
	case "never":
		return nil
	case "finished":
		return &finished
	}
	at := finished.Add(24 * time.Hour)
	return &at
}

// summaryFor is a finished test's result per provider, in the starter's
// language ("SES: Inbox 8 · Tab 2 · Postal: Inbox 5 · Spam 2").
func (s *Service) summaryFor(ctx context.Context, t store.PlacementTest) string {
	l := i18n.Default
	if s.LangOf != nil && t.StartedBy != nil {
		l = s.LangOf(ctx, *t.StartedBy)
	}
	copies, err := s.Q.PlacementCopies(ctx, t.ID)
	if err != nil {
		return ""
	}
	snap := snapshots(t)
	parts := []string{}
	for _, pid := range t.ProviderIds {
		cnt := countCopies(copies, &pid)
		name := snap[pid.String()].Short
		if name == "" {
			name = snap[pid.String()].Name
		}
		parts = append(parts, mSummaryProvider.Text(l, i18n.Args{"provider": name, "result": resultLine(l, cnt)}))
	}
	return strings.Join(parts, " · ")
}

// resultLine is "Inbox 8 · Tab 2 · Spam 0" in l, leaving out what's zero.
func resultLine(l i18n.Lang, c PlacementCounts) string {
	parts := []string{}
	add := func(m i18n.Msg, n int) {
		if n > 0 {
			parts = append(parts, m.Text(l, i18n.Args{"count": n}))
		}
	}
	add(mResultInbox, c.Inbox)
	add(mResultTab, c.Tab)
	add(mResultSpam, c.Spam)
	add(mResultNotArrived, c.NotArrived)
	add(mResultNotChecked, c.NotChecked)
	add(mResultFailed, c.Failed)
	return strings.Join(parts, " · ")
}

// Stop ends a test now: copies not found yet are Not arrived and aren't
// looked for any more (audited).
func (s *Service) Stop(ctx context.Context, ws *store.Workspace, id uuid.UUID) (PlacementTestDetail, error) {
	l := i18n.LangFrom(ctx)
	t, err := s.Q.PlacementTestByID(ctx, id)
	if err != nil || t.WorkspaceID != ws.ID {
		return PlacementTestDetail{}, api.NotFound("Placement test")
	}
	if t.Status != "sending" && t.Status != "waiting" {
		return PlacementTestDetail{}, api.Err(http.StatusConflict, "placement_not_waiting", mTestNotWaiting.Text(l, nil))
	}
	now := s.now()
	deleteAt := deleteTime(t.DeleteAfter, now)
	copies, _ := s.Q.PlacementCopies(ctx, id)
	before := countCopies(copies, nil)
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		n, err := q.PlacementTestStop(ctx, store.PlacementTestStopParams{ID: id, WorkspaceID: ws.ID, StoppedBy: actor(ctx), At: &now, DeleteAt: deleteAt})
		if err != nil {
			return err
		}
		if n == 0 {
			return api.Err(http.StatusConflict, "placement_not_waiting", mTestNotWaiting.Text(l, nil))
		}
		if err := q.PlacementCopiesNotArrived(ctx, store.PlacementCopiesNotArrivedParams{TestID: id, Reason: ReasonStopped}); err != nil {
			return err
		}
		if err := q.PlacementCopiesUnsent(ctx, id); err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "placement.test_stop", TargetType: "placement_test", TargetID: id.String(),
			TargetName: t.Name, Metadata: map[string]any{"found": before.Found, "copies": before.Copies, "minutesIn": int(now.Sub(t.StartedAt).Minutes()),
				"summary": fmt.Sprintf("stopped waiting with %d of %d copies found", before.Found, before.Copies)}})
	})
	if err != nil {
		return PlacementTestDetail{}, err
	}
	if deleteAt != nil && !deleteAt.After(now) {
		if t, err := s.Q.PlacementTestByID(ctx, id); err == nil {
			_ = s.deleteDueTest(context.WithoutCancel(ctx), ws, t)
		}
	}
	return s.Test(ctx, ws, id)
}

// DeleteDue deletes the copies whose time has come.
func (s *Service) DeleteDue(ctx context.Context) error {
	tests, err := s.Q.PlacementTestsToDelete(ctx, ptrTime(s.now()))
	if err != nil {
		return err
	}
	for _, t := range tests {
		ws, err := s.Q.GetWorkspace(ctx, t.WorkspaceID)
		if err != nil {
			continue
		}
		if err := s.deleteDueTest(ctx, &ws, t); err != nil {
			s.log().WarnContext(ctx, "placement: delete copies", "test", t.ID, "err", err)
		}
	}
	return nil
}

func (s *Service) deleteDueTest(ctx context.Context, ws *store.Workspace, t store.PlacementTest) error {
	_, err := s.deleteCopies(ctx, ws, t, nil)
	return err
}

// DeleteCopiesNow deletes a test's copies from the seeds (audited).
func (s *Service) DeleteCopiesNow(ctx context.Context, ws *store.Workspace, id uuid.UUID) (PlacementTestDetail, error) {
	l := i18n.LangFrom(ctx)
	t, err := s.Q.PlacementTestByID(ctx, id)
	if err != nil || t.WorkspaceID != ws.ID {
		return PlacementTestDetail{}, api.NotFound("Placement test")
	}
	if t.CopiesDeletedAt != nil {
		return PlacementTestDetail{}, api.Err(http.StatusConflict, "placement_copies_gone", mTestCopiesGone.Text(l, nil))
	}
	if _, err := s.deleteCopies(ctx, ws, t, actor(ctx)); err != nil {
		return PlacementTestDetail{}, err
	}
	return s.Test(ctx, ws, id)
}

// deleteCopies removes a test's copies from every seed that got one,
// found again by test token so only test copies are touched. It records
// the deletion (audited, by a person or by the schedule) and returns how
// many messages went.
func (s *Service) deleteCopies(ctx context.Context, ws *store.Workspace, t store.PlacementTest, by *uuid.UUID) (int, error) {
	copies, err := s.Q.PlacementCopies(ctx, t.ID)
	if err != nil {
		return 0, err
	}
	bySeed := map[uuid.UUID][]string{}
	order := []uuid.UUID{}
	for _, c := range copies {
		if c.SeedID == nil || c.HandedAt == nil || c.DeletedAt != nil {
			continue
		}
		if bySeed[*c.SeedID] == nil {
			order = append(order, *c.SeedID)
		}
		bySeed[*c.SeedID] = append(bySeed[*c.SeedID], c.Token)
	}
	now := s.now()
	deleted, seeds, skipped := 0, 0, []string{}
	for _, sid := range order {
		seed, err := s.Q.PlacementSeed(ctx, store.PlacementSeedParams{ID: sid, WorkspaceID: ws.ID})
		if err != nil {
			continue // removed: its copies stay in the mailbox
		}
		box, err := s.openSeed(ctx, seed)
		if err != nil {
			skipped = append(skipped, seed.Email)
			if errors.Is(err, seedbox.ErrAuthFailed) || errors.Is(err, seedbox.ErrAuthExpired) {
				s.seedFailed(ctx, ws, seed, err)
			}
			continue
		}
		n, err := box.Delete(ctx, bySeed[sid], t.StartedAt.Add(-10*time.Minute))
		_ = box.Close()
		if err != nil {
			skipped = append(skipped, seed.Email)
			continue
		}
		deleted += n
		seeds++
		_ = s.Q.PlacementCopiesDeleted(ctx, store.PlacementCopiesDeletedParams{TestID: t.ID, SeedID: &sid, At: &now})
	}
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if err := q.PlacementTestCopiesDeleted(ctx, store.PlacementTestCopiesDeletedParams{ID: t.ID, At: &now, By: by}); err != nil {
			return err
		}
		meta := map[string]any{"deleted": deleted, "seeds": seeds, "summary": fmt.Sprintf("%d copies deleted from %d seeds", deleted, seeds)}
		if by == nil {
			meta["schedule"] = t.DeleteAfter
		}
		if len(skipped) > 0 {
			meta["skipped"] = skipped
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "placement.copies_delete", TargetType: "placement_test",
			TargetID: t.ID.String(), TargetName: t.Name, Metadata: meta})
	})
	return deleted, err
}
