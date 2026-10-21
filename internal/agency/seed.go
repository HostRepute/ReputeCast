package agency

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/store"
)

// The Northwind Studio sample (internal/demo) sets its clients up through
// these, so its rows read exactly like the ones the app writes.

// SeedTemplate applies a workspace template to a workspace in tx, with
// every permission, as creating a client does, and records the
// application. It writes no audit entry: the sample writes its own history.
func SeedTemplate(ctx context.Context, tx pgx.Tx, templateID, workspaceID, by uuid.UUID, at time.Time) (AgencyTemplatePlan, error) {
	q := store.New(tx)
	s := &Service{Q: q, Audit: audit.New(q, slog.New(slog.NewTextHandler(io.Discard, nil))), Now: func() time.Time { return at }}
	tpl, err := q.AgencyTemplate(ctx, templateID)
	if err != nil {
		return AgencyTemplatePlan{}, err
	}
	rows, err := q.AgencyWorkspaces(ctx)
	if err != nil {
		return AgencyTemplatePlan{}, err
	}
	for _, w := range rows {
		if w.ID != workspaceID {
			continue
		}
		plan, err := s.plan(ctx, q, tx, tpl, w, authz.All(), &by, true)
		if err != nil {
			return plan, err
		}
		var name string
		if u, err := q.GetUser(ctx, by); err == nil {
			name = u.Name
		}
		plan.AppliedAt, plan.AppliedBy = &at, name
		result, _ := json.Marshal(plan)
		return plan, q.AgencyTemplateApplicationInsert(ctx, store.AgencyTemplateApplicationInsertParams{ID: uuid.Must(uuid.NewV7()), TemplateID: tpl.ID,
			WorkspaceID: w.ID, AppliedBy: &by, AppliedAt: at, Result: result})
	}
	return AgencyTemplatePlan{}, api.NotFound("Client")
}

// SeedLibraryCopy makes a copy of one version of a library item in a
// workspace, as "Copy to a client" does, and records it. Sections,
// campaign templates and automation templates only.
func SeedLibraryCopy(ctx context.Context, tx pgx.Tx, itemID uuid.UUID, version int32, workspaceID, by uuid.UUID, at time.Time) (uuid.UUID, error) {
	q := store.New(tx)
	s := &Service{Q: q, Now: func() time.Time { return at }}
	item, err := q.AgencyLibraryItem(ctx, itemID)
	if err != nil {
		return uuid.Nil, err
	}
	v, err := q.AgencyLibraryVersion(ctx, store.AgencyLibraryVersionParams{ItemID: itemID, Version: version})
	if err != nil {
		return uuid.Nil, err
	}
	c := parseLibraryContent(v.Content)
	var id uuid.UUID
	target := "template"
	switch item.Kind {
	case KindSection, KindTemplate:
		id, err = insertTemplate(ctx, q, workspaceID, item.Kind, item.Name, TplItem{Name: item.Name, MJML: c.MJML, HTML: c.HTML, Thumb: c.Thumb}, &by, at)
	case KindAutomation:
		if c.Automation == nil {
			return uuid.Nil, api.Conflict("the automation template is empty")
		}
		lists, lerr := q.AgencyCaptureLists(ctx, workspaceID)
		if lerr != nil {
			return uuid.Nil, lerr
		}
		ids := map[string]string{}
		for _, l := range lists {
			ids[l.Name] = l.ID.String()
		}
		name := item.Name
		if taken, terr := q.AgencyAutomationNameTaken(ctx, store.AgencyAutomationNameTakenParams{WorkspaceID: workspaceID, Name: name}); terr != nil {
			return uuid.Nil, terr
		} else if taken {
			name += " (library)"
		}
		target = "automation"
		id, err = s.insertAutomation(ctx, q, workspaceID, name, *c.Automation, ids, map[string]string{}, &by, "copied from the agency library")
	default:
		return uuid.Nil, api.Conflict("only sections, campaign templates and automation templates")
	}
	if err != nil {
		return uuid.Nil, err
	}
	return id, q.AgencyLibraryInsertCopy(ctx, store.AgencyLibraryInsertCopyParams{ID: uuid.Must(uuid.NewV7()), ItemID: itemID, Version: version, WorkspaceID: workspaceID,
		TargetKind: target, TargetID: id, CopiedBy: &by, At: at})
}

// SeedLibraryVersion returns what a library item's version stores besides
// its content: the lines its diff compares and the Liquid fields it needs.
func SeedLibraryVersion(kind string, c LibraryContent) (summary []string, fields json.RawMessage) {
	srcs := []string{c.MJML, c.Subject, c.Body}
	if c.Automation != nil {
		for _, e := range c.Automation.Emails {
			srcs = append(srcs, e.MJML, e.Subject)
		}
	}
	fields, _ = json.Marshal(libraryFields(srcs...))
	return summaryLines(kind, c), fields
}
