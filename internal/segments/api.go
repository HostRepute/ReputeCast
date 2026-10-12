package segments

import (
	"context"
	"errors"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/permcheck"
)

type segmentPath struct {
	api.WorkspaceInput
	ID string `path:"segment" format:"uuid" doc:"Segment id"`
}

type segmentBody struct {
	api.WorkspaceInput
	Body Input
}

type segmentUpdate struct {
	segmentPath
	Body Input
}

// Problem codes of this package.
const (
	CodeSQLError    = "sql_error"
	CodeSQLDisabled = "sql_disabled"
	CodeTimeout     = "count_timeout"
)

// Problem turns segment errors into API problems.
func Problem(err error) error {
	var ve *ValidationError
	var se *SQLError
	switch {
	case err == nil:
		return nil
	case errors.As(err, &ve):
		return api.Invalid(ve.Location, ve.Message)
	case errors.As(err, &se):
		p := api.Err(http.StatusUnprocessableEntity, CodeSQLError, se.Message)
		if se.Position > 0 {
			p = p.WithMeta("position", se.Position)
		}
		return p
	case errors.Is(err, ErrTimeout):
		return api.Err(http.StatusGatewayTimeout, CodeTimeout, "The count took longer than 10 s. Segments with event conditions over many contacts can be slow on default Postgres settings; the segment still works at send time.")
	case errors.Is(err, ErrSQLDisabled):
		return api.Err(http.StatusConflict, CodeSQLDisabled, ErrSQLDisabled.Error())
	}
	return api.MapError(err, "Segment")
}

// Register adds the segments operations.
func Register(h huma.API, s *Service) {
	tags := []string{"Segments"}
	base := "/workspaces/{workspace}/segments"
	view := api.Workspace(authz.SubscribersView)
	edit := api.Workspace(authz.SegmentsEdit)

	api.Get(h, api.Op{ID: "list-segments", Path: base, Summary: "List saved segments with their last counts", Tags: tags, Access: view},
		func(ctx context.Context, in *api.WorkspaceInput) (*api.Body[SegmentList], error) {
			out, err := s.List(ctx, api.WorkspaceFrom(ctx))
			return api.OK(out), Problem(err)
		})
	api.Get(h, api.Op{ID: "segment-catalog", Path: base + "/catalog", Summary: "Fields, operators and values the builder can use", Tags: tags, Access: view},
		func(ctx context.Context, in *api.WorkspaceInput) (*api.Body[Catalog], error) {
			out, err := s.Catalog(ctx, api.WorkspaceFrom(ctx))
			return api.OK(out), Problem(err)
		})
	api.Post(h, api.Op{ID: "preview-segment", Path: base + "/preview", Summary: "Count an unsaved segment (live count while editing)",
		Description: "Counts contacts matching a builder definition or, with segments:sql, a SQL segment. Nothing is saved.", Tags: tags, Access: view},
		func(ctx context.Context, in *segmentBody) (*api.Body[CountResult], error) {
			ws := api.WorkspaceFrom(ctx)
			if in.Body.Mode == "sql" {
				if err := permcheck.Require(ctx, s.Q, s.Audit, ws.ID, authz.SegmentsSQL, "segment.sql_preview"); err != nil {
					return nil, err
				}
			}
			out, err := s.Preview(ctx, ws, in.Body)
			return api.OK(out), Problem(err)
		})
	api.Post(h, api.Op{ID: "create-segment", Path: base, Summary: "Save a new segment", Tags: tags, Access: edit, Status: http.StatusCreated},
		func(ctx context.Context, in *segmentBody) (*api.Body[SegmentDetail], error) {
			ws := api.WorkspaceFrom(ctx)
			if in.Body.Mode == "sql" {
				if err := permcheck.Require(ctx, s.Q, s.Audit, ws.ID, authz.SegmentsSQL, "segment.create"); err != nil {
					return nil, err
				}
			}
			out, err := s.Create(ctx, ws, actorID(ctx), in.Body)
			return api.OK(out), Problem(err)
		})
	api.Get(h, api.Op{ID: "get-segment", Path: base + "/{segment}", Summary: "Get a segment with its definition and generated SQL", Tags: tags, Access: view},
		func(ctx context.Context, in *segmentPath) (*api.Body[SegmentDetail], error) {
			out, err := s.Get(ctx, api.WorkspaceFrom(ctx), uuid.MustParse(in.ID))
			return api.OK(out), Problem(err)
		})
	api.Put(h, api.Op{ID: "update-segment", Path: base + "/{segment}", Summary: "Change a segment", Tags: tags, Access: edit},
		func(ctx context.Context, in *segmentUpdate) (*api.Body[SegmentDetail], error) {
			ws := api.WorkspaceFrom(ctx)
			id := uuid.MustParse(in.ID)
			cur, err := s.Get(ctx, ws, id)
			if err != nil {
				return nil, Problem(err)
			}
			// Writing or replacing SQL needs segments:sql.
			if in.Body.Mode == "sql" || cur.Mode == "sql" {
				if err := permcheck.Require(ctx, s.Q, s.Audit, ws.ID, authz.SegmentsSQL, "segment.update"); err != nil {
					return nil, err
				}
			}
			out, err := s.Update(ctx, ws, actorID(ctx), id, in.Body)
			return api.OK(out), Problem(err)
		})
	api.Delete(h, api.Op{ID: "delete-segment", Path: base + "/{segment}", Summary: "Delete a segment", Tags: tags, Access: edit},
		func(ctx context.Context, in *segmentPath) (*api.Empty, error) {
			return api.NoContent, Problem(s.Delete(ctx, api.WorkspaceFrom(ctx), uuid.MustParse(in.ID)))
		})
	api.Post(h, api.Op{ID: "count-segment", Path: base + "/{segment}/count", Summary: "Count a saved segment again and store the result", Tags: tags, Access: view},
		func(ctx context.Context, in *segmentPath) (*api.Body[CountResult], error) {
			ws := api.WorkspaceFrom(ctx)
			id := uuid.MustParse(in.ID)
			out, err := s.Recount(ctx, ws, id)
			if errors.Is(err, ErrTimeout) {
				p := Problem(err).(*api.Problem)
				if cur, gerr := s.Get(ctx, ws, id); gerr == nil && cur.Count != nil {
					p = p.WithMeta("lastCount", *cur.Count, "countedAt", cur.CountedAt)
				}
				return nil, p
			}
			return api.OK(out), Problem(err)
		})
}

func actorID(ctx context.Context) *uuid.UUID {
	if u := api.UserFrom(ctx); u != nil {
		id := u.ID
		return &id
	}
	return nil
}
