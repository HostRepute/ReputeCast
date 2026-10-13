package instance

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/jobs"
	"github.com/hostrepute/reputecast/internal/notify"
	"github.com/hostrepute/reputecast/internal/store"
	"github.com/hostrepute/reputecast/internal/workspaces"
)

// RestoreHold is how long a deleted workspace can be restored before the
// purge job deletes it for good.
const RestoreHold = 7 * 24 * time.Hour

// InstanceWorkspace is one row of Instance admin › Workspaces.
type InstanceWorkspace struct {
	ID          string    `json:"id"`
	Slug        string    `json:"slug"`
	Name        string    `json:"name"`
	Owner       string    `json:"owner,omitempty"`
	Members     int       `json:"members"`
	Invited     int       `json:"invited"`
	Subscribers int64     `json:"subscribers"`
	Providers   []string  `json:"providers"`
	CreatedAt   time.Time `json:"createdAt"`
	// DeletedAt and PurgeAt are set during the restore hold.
	DeletedAt *time.Time `json:"deletedAt,omitempty"`
	PurgeAt   *time.Time `json:"purgeAt,omitempty"`
	// Member: the admin belongs to it, so its settings open.
	Member bool `json:"member"`
}

// InstanceWorkspaceList is GET /admin/workspaces.
type InstanceWorkspaceList struct {
	Items []InstanceWorkspace `json:"items"`
	// Live is the number of workspaces not deleted.
	Live int `json:"live"`
}

// Workspaces lists every workspace, deleted ones in their hold last.
func (s *Service) Workspaces(ctx context.Context) (InstanceWorkspaceList, error) {
	me, err := admin(ctx)
	if err != nil {
		return InstanceWorkspaceList{}, err
	}
	rows, err := s.Q.InstanceAdminWorkspaces(ctx)
	if err != nil {
		return InstanceWorkspaceList{}, err
	}
	mine := map[uuid.UUID]bool{}
	if wss, err := s.Q.ListUserWorkspaces(ctx, me.ID); err == nil {
		for _, w := range wss {
			mine[w.ID] = true
		}
	}
	out := InstanceWorkspaceList{Items: make([]InstanceWorkspace, 0, len(rows))}
	for _, r := range rows {
		w := InstanceWorkspace{ID: r.ID.String(), Slug: r.Slug, Name: r.Name, Owner: r.OwnerName, Members: int(r.Members), Invited: int(r.Invited),
			Subscribers: r.Subscribers, Providers: nonNil(r.Providers), CreatedAt: r.CreatedAt, DeletedAt: r.DeletedAt, Member: mine[r.ID]}
		if r.DeletedAt != nil {
			p := r.DeletedAt.Add(RestoreHold)
			w.PurgeAt = &p
		} else {
			out.Live++
		}
		out.Items = append(out.Items, w)
	}
	return out, nil
}

func nonNil(s []string) []string {
	if s == nil {
		return []string{}
	}
	return s
}

// InstanceWorkspaceInput creates a workspace.
type InstanceWorkspaceInput struct {
	Name string `json:"name" minLength:"1" maxLength:"80"`
	// OwnerID defaults to the admin creating it.
	OwnerID  string `json:"ownerId,omitempty" format:"uuid"`
	Timezone string `json:"timezone,omitempty" maxLength:"64"`
}

// CreateWorkspace makes a workspace with its built-in roles and an Owner.
func (s *Service) CreateWorkspace(ctx context.Context, in InstanceWorkspaceInput) (InstanceWorkspace, error) {
	me, err := admin(ctx)
	if err != nil {
		return InstanceWorkspace{}, err
	}
	name := strings.TrimSpace(in.Name)
	if name == "" {
		return InstanceWorkspace{}, api.Invalid("body.name", "Name the workspace, like Acme Engineering Blog.")
	}
	owner := me.ID
	if in.OwnerID != "" {
		id, err := uuid.Parse(in.OwnerID)
		if err != nil {
			return InstanceWorkspace{}, api.Invalid("body.ownerId", "Pick an active account as the Owner.")
		}
		u, err := s.Q.GetUser(ctx, id)
		if err != nil || u.Status != "active" {
			return InstanceWorkspace{}, api.Invalid("body.ownerId", "Pick an active account as the Owner.")
		}
		owner = id
	}
	tz := strings.TrimSpace(in.Timezone)
	if tz == "" {
		if st, err := s.Q.GetInstanceSettings(ctx); err == nil {
			tz = st.DefaultTimezone
		}
	}
	if _, err := time.LoadLocation(tz); err != nil || tz == "" {
		return InstanceWorkspace{}, api.Invalid("body.timezone", "Use a timezone like America/New_York.")
	}
	var ws store.Workspace
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		var err error
		ws, err = workspaces.Create(ctx, tx, workspaces.CreateParams{Name: name, Timezone: tz, Owner: owner, Source: "admin"})
		if err != nil {
			return err
		}
		id := ws.ID
		return s.Audit.LogWith(ctx, store.New(tx), audit.Event{WorkspaceID: &id, Action: "workspace.create", TargetType: "workspace", TargetID: ws.ID.String(), TargetName: ws.Name,
			After: map[string]any{"name": ws.Name, "slug": ws.Slug, "timezone": tz, "owner": owner.String()}, Metadata: map[string]any{"by": "instance admin"}})
	})
	if err != nil {
		return InstanceWorkspace{}, err
	}
	return InstanceWorkspace{ID: ws.ID.String(), Slug: ws.Slug, Name: ws.Name, Members: 1, Providers: []string{}, CreatedAt: ws.CreatedAt, Member: owner == me.ID}, nil
}

// InstanceWorkspaceDeleteInput confirms a deletion by typing the name.
type InstanceWorkspaceDeleteInput struct {
	Confirm string `json:"confirm" maxLength:"80"`
}

// InstanceWorkspaceDeleted says until when the workspace can be restored.
type InstanceWorkspaceDeleted struct {
	PurgeAt   time.Time `json:"purgeAt"`
	Cancelled int64     `json:"cancelled" doc:"Scheduled sends that were stopped"`
}

// DeleteWorkspace deletes a workspace after a typed confirmation: sends
// stop now, members lose access, and a River job deletes it for good after
// the 7-day restore hold.
func (s *Service) DeleteWorkspace(ctx context.Context, id uuid.UUID, in InstanceWorkspaceDeleteInput) (InstanceWorkspaceDeleted, error) {
	ws, err := s.Q.InstanceAdminWorkspace(ctx, id)
	if err != nil {
		return InstanceWorkspaceDeleted{}, api.MapError(err, "Workspace")
	}
	if ws.DeletedAt != nil {
		return InstanceWorkspaceDeleted{}, api.Conflict(ws.Name + " is already deleted. Restore it or wait for the purge.")
	}
	if strings.TrimSpace(in.Confirm) != ws.Name {
		return InstanceWorkspaceDeleted{}, api.Invalid("body.confirm", "Type "+ws.Name+" exactly to confirm.")
	}
	var out InstanceWorkspaceDeleted
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		var err error
		if out.Cancelled, err = q.InstanceAdminCancelScheduled(ctx, id); err != nil {
			return err
		}
		deleted, err := q.InstanceAdminSoftDeleteWorkspace(ctx, id)
		if err != nil {
			return err
		}
		at := s.now()
		if deleted != nil {
			at = *deleted
		}
		out.PurgeAt = at.Add(RestoreHold)
		if err := notify.ToMembersWith(ctx, tx, id, "", notify.Notification{Kind: "workspace_deleted", Severity: "warn", Title: ws.Name + " was deleted",
			Summary: ws.Name + " was deleted by an instance admin",
			Body:    fmt.Sprintf("An instance admin deleted %s. It can be restored until %s; after that it's gone for good.", ws.Name, out.PurgeAt.Format("Jan 2"))}); err != nil {
			return err
		}
		if s.Jobs != nil {
			if c := s.Jobs(); c != nil {
				if _, err := c.InsertTx(ctx, tx, WorkspacePurgeArgs{WorkspaceID: id}, &river.InsertOpts{ScheduledAt: out.PurgeAt.Add(time.Minute)}); err != nil {
					return err
				}
			}
		}
		wsID := id
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &wsID, Action: "workspace.delete", TargetType: "workspace", TargetID: id.String(), TargetName: ws.Name,
			Before: map[string]any{"slug": ws.Slug}, After: map[string]any{"deleted": true, "purge_at": out.PurgeAt},
			Metadata: map[string]any{"by": "instance admin", "summary": fmt.Sprintf("restorable until %s · %d scheduled %s stopped", out.PurgeAt.Format("Jan 2"), out.Cancelled, plural(int(out.Cancelled), "send", "sends")),
				"permission": string(authz.WorkspaceDelete)}})
	})
	return out, err
}

var deletedSuffix = regexp.MustCompile(`-deleted-\d+$`)

// RestoreWorkspace brings a deleted workspace back during its hold, under
// its old address when that is still free.
func (s *Service) RestoreWorkspace(ctx context.Context, id uuid.UUID) (InstanceWorkspace, error) {
	ws, err := s.Q.InstanceAdminWorkspace(ctx, id)
	if err != nil {
		return InstanceWorkspace{}, api.MapError(err, "Workspace")
	}
	if ws.DeletedAt == nil {
		return InstanceWorkspace{}, api.Conflict(ws.Name + " isn't deleted.")
	}
	if s.now().Sub(*ws.DeletedAt) > RestoreHold {
		return InstanceWorkspace{}, api.Err(http.StatusGone, "restore_expired", ws.Name+" was deleted more than 7 days ago and can't be restored.")
	}
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		slug := deletedSuffix.ReplaceAllString(ws.Slug, "")
		if taken, err := q.SlugTaken(ctx, slug); err != nil {
			return err
		} else if taken || len(slug) < 2 {
			if slug, err = workspaces.UniqueSlug(ctx, q, ws.Name); err != nil {
				return err
			}
		}
		if err := q.InstanceAdminRestoreWorkspace(ctx, store.InstanceAdminRestoreWorkspaceParams{ID: id, Slug: slug}); err != nil {
			return err
		}
		ws.Slug = slug
		wsID := id
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &wsID, Action: "workspace.restore", TargetType: "workspace", TargetID: id.String(), TargetName: ws.Name,
			Before: map[string]any{"deleted": true}, After: map[string]any{"deleted": false, "slug": slug}, Metadata: map[string]any{"by": "instance admin"}})
	})
	if err != nil {
		return InstanceWorkspace{}, err
	}
	return InstanceWorkspace{ID: ws.ID.String(), Slug: ws.Slug, Name: ws.Name, Providers: []string{}, CreatedAt: ws.CreatedAt}, nil
}

// WorkspacePurgeArgs deletes one workspace for good after its restore hold.
type WorkspacePurgeArgs struct {
	WorkspaceID uuid.UUID `json:"workspace_id"`
}

// Kind implements river.JobArgs.
func (WorkspacePurgeArgs) Kind() string { return "instance.workspace_purge" }

// InsertOpts runs purges on the maintenance queue, one per workspace.
func (WorkspacePurgeArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{Queue: jobs.QueueMaintenance, MaxAttempts: 5, UniqueOpts: river.UniqueOpts{ByArgs: true}}
}

// WorkspacePurgeWorker deletes a workspace whose hold is over. A restored
// workspace, or one still in its hold, is left alone.
type WorkspacePurgeWorker struct {
	river.WorkerDefaults[WorkspacePurgeArgs]
	Pool  *pgxpool.Pool
	Audit *audit.Logger
	Log   *slog.Logger
}

// Timeout allows large workspaces time to cascade.
func (w *WorkspacePurgeWorker) Timeout(*river.Job[WorkspacePurgeArgs]) time.Duration {
	return 30 * time.Minute
}

// Work implements river.Worker.
func (w *WorkspacePurgeWorker) Work(ctx context.Context, job *river.Job[WorkspacePurgeArgs]) error {
	return db.Tx(ctx, w.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		row, err := q.InstanceAdminPurgeWorkspace(ctx, job.Args.WorkspaceID)
		if err != nil {
			if isNoRows(err) {
				return nil // restored, still in its hold, or already gone
			}
			return err
		}
		w.Log.InfoContext(ctx, "purged deleted workspace", "id", row.ID, "name", row.Name)
		return w.Audit.LogWith(ctx, q, audit.Event{Action: "workspace.purge", TargetType: "workspace", TargetID: row.ID.String(), TargetName: row.Name,
			Actor: &audit.Actor{Type: authz.ActorSystem, Name: "System"}, Metadata: map[string]any{"summary": "deleted for good after the 7-day restore hold"}})
	})
}
