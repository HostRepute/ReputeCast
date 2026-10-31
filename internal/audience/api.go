package audience

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"slices"
	"strings"

	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/authz"
)

// File is a download (CSV or JSON).
type File struct {
	ContentType        string `header:"Content-Type"`
	ContentDisposition string `header:"Content-Disposition"`
	CacheControl       string `header:"Cache-Control"`
	Body               []byte
}

func download(body []byte, contentType, name string) *File {
	return &File{ContentType: contentType, ContentDisposition: fmt.Sprintf(`attachment; filename="%s"`, name), CacheControl: "no-store", Body: body}
}

type contactsQuery struct {
	api.WorkspaceInput
	Q           string `query:"q" maxLength:"200" doc:"Email, name or tag"`
	List        string `query:"list" doc:"List id"`
	Tier        string `query:"tier" enum:"active,cooling,at_risk,sunset,"`
	Mailbox     string `query:"mailbox" enum:"gmail,outlook,yahoo,icloud,other,"`
	Consent     string `query:"consent" enum:"double,single,pending,"`
	Source      string `query:"source" maxLength:"80"`
	Tag         string `query:"tag" maxLength:"80"`
	Segment     string `query:"segment" doc:"Segment id"`
	JoinedAfter string `query:"joinedAfter" maxLength:"10" doc:"Joined on or after this day (2026-09-01), in the workspace's time zone"`
	// JoinedBefore and Import: review 2026-11 AU-10.
	JoinedBefore string `query:"joinedBefore" maxLength:"10" doc:"Joined on or before this day, in the workspace's time zone"`
	Import       string `query:"import" maxLength:"40" doc:"An import's id: the contacts it brought in"`
	Check        string `query:"check" enum:"valid,invalid,risky,unknown,never," doc:"The latest address check's result"`
	Language     string `query:"language" maxLength:"16" doc:"A workspace language's code (es also matches es-MX), none or not_offered"`
	ScoreGrowthQuery
	WebsiteQuery
	PaidQuery
	PushWhatsAppQuery
	ProductsQuery
	StoreQuery
	ChannelQuery
	Cursor   string `query:"cursor" maxLength:"600" doc:"nextCursor or prevCursor of the page before; pages in constant time"`
	Page     int    `query:"page" minimum:"1" default:"1" doc:"A page number from the start, when there is no cursor"`
	PageSize int    `query:"pageSize" minimum:"1" maximum:"100" default:"10"`
	Exact    bool   `query:"exact" doc:"Count past 10,000 (slower on large audiences)"`
	Fields   string `query:"fields" maxLength:"600" doc:"Custom field keys, comma-separated (at most 10), whose values each row carries"`
}

// fieldKeys reads the fields parameter: up to ten keys.
func (q contactsQuery) fieldKeys() []string {
	var out []string
	for _, k := range strings.Split(q.Fields, ",") {
		if k = strings.TrimSpace(k); k != "" && len(out) < 10 && !slices.Contains(out, k) {
			out = append(out, k)
		}
	}
	return out
}

func (q contactsQuery) filter() ContactFilter {
	return ContactFilter{Q: q.Q, List: q.List, Tier: q.Tier, Mailbox: q.Mailbox, Consent: q.Consent, Source: q.Source, Tag: q.Tag, Segment: q.Segment, JoinedAfter: q.JoinedAfter, JoinedBefore: q.JoinedBefore, Import: q.Import,
		Language: q.Language, ScoreGrowthFilter: q.ScoreGrowthQuery.filter(), Check: q.Check, WebsiteFilter: q.WebsiteQuery.filter(), PaidFilter: q.PaidQuery.filter(),
		PushWhatsAppFilter: q.PushWhatsAppQuery.filter(), ProductsFilter: q.ProductsQuery.filter(), StoreFilter: q.StoreQuery.filter(), ChannelFilter: q.ChannelQuery.filter()}
}

type uploadRemoveInput struct {
	api.WorkspaceInput
	ID   string `path:"id" format:"uuid"`
	Body RemoveUploadInput
}

type dryRunInput struct {
	api.WorkspaceInput
	Months int `query:"months" minimum:"1" maximum:"120" default:"12"`
}

type jobPathInput struct {
	api.WorkspaceInput
	ID string `path:"id" format:"uuid"`
}

type contactListPath struct {
	ContactPathInput
	List string `path:"list" format:"uuid"`
}

type viewUpdateInput struct {
	api.WorkspaceInput
	ID   string `path:"id" format:"uuid"`
	Body ContactViewUpdate
}

type ContactPathInput struct {
	api.WorkspaceInput
	Contact string `path:"contact" format:"uuid"`
}

type activityQuery struct {
	ContactPathInput
	Filter string `query:"filter" enum:"all,deliveries,engagement,consent" default:"all"`
	Cursor string `query:"cursor" maxLength:"200"`
	Limit  int    `query:"limit" minimum:"1" maximum:"100" default:"8"`
}

type suppressionsQuery struct {
	api.WorkspaceInput
	Q      string `query:"q" maxLength:"200" doc:"Address or domain"`
	Reason string `query:"reason" enum:"hard_bounce,complaint,unsubscribe,manual,invalid,spam_trap,"`
	Source string `query:"source" maxLength:"60"`
	Since  string `query:"since" enum:"7d,30d,"`
	Import string `query:"import" doc:"Show the rows of one import group"`
	// AppliesTo: all mail, or marketing only (transactional mail still
	// reaches the address).
	AppliesTo string `query:"appliesTo" enum:"all,marketing," doc:"all: every stream; marketing: Broadcasts and Automations only (unsubscribes, role addresses)"`
	Cursor    string `query:"cursor" maxLength:"600" doc:"nextCursor or prevCursor of the page before"`
	Page      int    `query:"page" minimum:"1" default:"1" doc:"A page number from the start, when there is no cursor"`
	PageSize  int    `query:"pageSize" minimum:"1" maximum:"100" default:"10"`
	Exact     bool   `query:"exact" doc:"Count past 10,000"`
}

type AudienceIDInput struct {
	api.WorkspaceInput
	ID string `path:"id" format:"uuid"`
}

type body[T any] struct {
	api.WorkspaceInput
	Body T
}

type idBody[T any] struct {
	AudienceIDInput
	Body T
}

type contactBody[T any] struct {
	ContactPathInput
	Body T
}

type uploadInput struct {
	api.WorkspaceInput
	FileName string `query:"fileName" maxLength:"200"`
	Reason   string `query:"reason" enum:"manual,hard_bounce,complaint,unsubscribe," default:"manual"`
	Note     string `query:"note" maxLength:"300"`
	RawBody  []byte `contentType:"text/csv"`
}

// Items wraps a list response.
type Items[T any] struct {
	Items []T `json:"items"`
}

// PreferenceLinkResult is a link to open.
type PreferenceLinkResult struct {
	URL string `json:"url"`
}

// Register adds the audience operations.
func Register(h huma.API, s *Service, box Box) {
	tags := []string{"Audience"}
	ws := "/workspaces/{workspace}"
	view := api.Workspace(authz.SubscribersView)
	edit := api.Workspace(authz.SubscribersEdit)
	manage := api.Workspace(authz.ListsManage)

	api.Get(h, api.Op{ID: "audience-summary", Path: ws + "/audience", Summary: "Counts for the Audience header and filters", Tags: tags, Access: view},
		func(ctx context.Context, in *api.WorkspaceInput) (*api.Body[AudienceSummary], error) {
			out, err := s.Summary(ctx, api.WorkspaceFrom(ctx))
			return api.OK(out), err
		})
	api.Get(h, api.Op{ID: "list-health", Path: ws + "/audience/health", Summary: "Tiers, tier trend, sunset policy and health per list", Tags: tags, Access: view},
		func(ctx context.Context, in *api.WorkspaceInput) (*api.Body[ListHealth], error) {
			out, err := s.Health(ctx, api.WorkspaceFrom(ctx))
			return api.OK(out), err
		})
	api.Get(h, api.Op{ID: "sunset-erase-dry-run", Path: ws + "/audience/sunset-policy/dry-run", Summary: "How many contacts erasing after N months in Sunset would erase first (nothing changes)",
		Tags: tags, Access: manage},
		func(ctx context.Context, in *dryRunInput) (*api.Body[SunsetDryRun], error) {
			out, err := s.SunsetDryRun(ctx, api.WorkspaceFrom(ctx), in.Months)
			return api.OK(out), err
		})
	api.Put(h, api.Op{ID: "save-sunset-policy", Path: ws + "/audience/sunset-policy", Summary: "Save the sunset policy (applies at the next nightly run)", Tags: tags, Access: manage},
		func(ctx context.Context, in *body[SunsetPolicy]) (*api.Body[SunsetPolicy], error) {
			out, err := s.SavePolicy(ctx, api.WorkspaceFrom(ctx), in.Body)
			return api.OK(out), err
		})

	// Contacts
	api.Get(h, api.Op{ID: "list-contacts", Path: ws + "/contacts", Summary: "List contacts, most recent activity first (or the order asked for)", Tags: tags, Access: view},
		func(ctx context.Context, in *contactsQuery) (*api.Body[ContactList], error) {
			out, err := s.ListContacts(ctx, api.WorkspaceFrom(ctx), in.filter(), ContactPage{Cursor: in.Cursor, Page: in.Page, PageSize: in.PageSize, Exact: in.Exact, Fields: in.fieldKeys()})
			return api.OK(out), err
		})
	api.Post(h, api.Op{ID: "create-contact", Path: ws + "/contacts", Summary: "Add a contact", Tags: tags, Access: api.Workspace(authz.SubscribersCreate), Status: http.StatusCreated},
		func(ctx context.Context, in *body[NewContact]) (*api.Body[NewContactResult], error) {
			out, err := s.CreateContact(ctx, api.WorkspaceFrom(ctx), in.Body)
			return api.OK(out), err
		})
	api.Get(h, api.Op{ID: "list-contact-views", Path: ws + "/contacts/views", Summary: "Saved views with their counts", Tags: tags, Access: view},
		func(ctx context.Context, in *api.WorkspaceInput) (*api.Body[ContactSavedViews], error) {
			out, err := s.Views(ctx, api.WorkspaceFrom(ctx))
			return api.OK(out), err
		})
	api.Post(h, api.Op{ID: "create-contact-view", Path: ws + "/contacts/views", Summary: "Save the current filters as a view", Tags: tags, Access: edit, Status: http.StatusCreated},
		func(ctx context.Context, in *body[ContactViewInput]) (*api.Body[ContactSavedView], error) {
			out, err := s.CreateView(ctx, api.WorkspaceFrom(ctx), in.Body)
			return api.OK(out), err
		})
	api.Patch(h, api.Op{ID: "update-contact-view", Path: ws + "/contacts/views/{id}", Summary: "Rename a saved view, or save new filters or columns in it",
		Description: "A personal view is its creator's only. Audited as contact_view.update.", Tags: tags, Access: edit},
		func(ctx context.Context, in *viewUpdateInput) (*api.Body[ContactSavedView], error) {
			id, err := uuid.Parse(in.ID)
			if err != nil {
				return nil, api.NotFound("Saved view")
			}
			out, err := s.UpdateView(ctx, api.WorkspaceFrom(ctx), id, in.Body)
			return api.OK(out), err
		})
	api.Delete(h, api.Op{ID: "delete-contact-view", Path: ws + "/contacts/views/{id}", Summary: "Delete a saved view", Tags: tags, Access: edit},
		func(ctx context.Context, in *AudienceIDInput) (*api.Empty, error) {
			return api.NoContent, s.DeleteView(ctx, api.WorkspaceFrom(ctx), uuid.MustParse(in.ID))
		})
	api.Post(h, api.Op{ID: "bulk-contacts", Path: ws + "/contacts/bulk", Summary: "Add to a list, add a tag, resend opt-in or suppress many contacts", Tags: tags, Access: edit},
		func(ctx context.Context, in *body[ContactBulkInput]) (*api.Body[ContactBulkResult], error) {
			out, err := s.Bulk(ctx, api.WorkspaceFrom(ctx), in.Body)
			return api.OK(out), err
		})
	api.Post(h, api.Op{ID: "erase-contacts", Path: ws + "/contacts/erase", Summary: "Erase contacts' personal data", Tags: tags, Access: api.Workspace(authz.SubscribersDelete)},
		func(ctx context.Context, in *body[ContactEraseInput]) (*api.Body[ContactBulkResult], error) {
			out, err := s.Erase(ctx, api.WorkspaceFrom(ctx), in.Body)
			return api.OK(out), err
		})
	api.Post(h, api.Op{ID: "start-contacts-export", Path: ws + "/contacts/jobs/export", Summary: "Export contacts in the background (logged)",
		Description: "For any number of contacts: the file is written in the background (the jobs tray follows it) and kept 7 days for whoever started it. " +
			"Needs subscribers:export; starting, finishing and each download are audited.", Tags: tags, Access: api.Workspace(authz.SubscribersExport), Status: http.StatusAccepted},
		func(ctx context.Context, in *body[ContactExportInput]) (*api.Body[AudienceJob], error) {
			out, err := s.StartJob(ctx, api.WorkspaceFrom(ctx), JobExport, in.Body.Selection)
			return api.OK(out), err
		})
	api.Get(h, api.Op{ID: "get-contacts-job", Path: ws + "/contacts/jobs/{id}", Summary: "A background export or resend: its progress and what it did", Tags: tags, Access: view},
		func(ctx context.Context, in *jobPathInput) (*api.Body[AudienceJob], error) {
			id, err := uuid.Parse(in.ID)
			if err != nil {
				return nil, api.NotFound("Job")
			}
			out, err := s.Job(ctx, api.WorkspaceFrom(ctx), id)
			return api.OK(out), err
		})
	api.Get(h, api.Op{ID: "download-contacts-export", Path: ws + "/contacts/jobs/{id}/download", Summary: "Download a finished export (its starter only, logged)",
		Tags: tags, Access: api.Workspace(authz.SubscribersExport)},
		func(ctx context.Context, in *jobPathInput) (*huma.StreamResponse, error) {
			id, err := uuid.Parse(in.ID)
			if err != nil {
				return nil, api.NotFound("Export")
			}
			wsp := api.WorkspaceFrom(ctx)
			f, size, err := s.OpenExport(ctx, wsp, id)
			if err != nil {
				return nil, err
			}
			name := fmt.Sprintf("contacts-%s-%s.csv", wsp.Slug, s.now().In(workspaceLocation(wsp)).Format("2006-01-02"))
			return &huma.StreamResponse{Body: func(hctx huma.Context) {
				defer func() { _ = f.Close() }()
				hctx.SetHeader("Content-Type", "text/csv; charset=utf-8")
				hctx.SetHeader("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, name))
				hctx.SetHeader("Content-Length", fmt.Sprint(size))
				hctx.SetHeader("Cache-Control", "no-store")
				hctx.SetStatus(200)
				_, _ = io.Copy(hctx.BodyWriter(), f)
			}}, nil
		})
	api.Delete(h, api.Op{ID: "remove-contact-from-list", Path: ws + "/contacts/{contact}/lists/{list}", Summary: "Take a contact off a list (the team's change, not an unsubscribe)",
		Description: "Recorded in the contact's history as removed by the team; audited.", Tags: tags, Access: edit},
		func(ctx context.Context, in *contactListPath) (*api.Empty, error) {
			list, err := uuid.Parse(in.List)
			if err != nil {
				return nil, api.NotFound("List")
			}
			return api.NoContent, s.RemoveFromList(ctx, api.WorkspaceFrom(ctx), uuid.MustParse(in.Contact), list)
		})
	api.Post(h, api.Op{ID: "export-contacts", Path: ws + "/contacts/export", Summary: "Export contacts as CSV (logged)",
		Description: "Needs subscribers:export. Every export is written to the audit log with its size and filter.", Tags: tags, Access: api.Workspace(authz.SubscribersExport)},
		func(ctx context.Context, in *body[ContactExportInput]) (*File, error) {
			wsp := api.WorkspaceFrom(ctx)
			b, _, err := s.Export(ctx, wsp, in.Body)
			if err != nil {
				return nil, err
			}
			return download(b, "text/csv; charset=utf-8", fmt.Sprintf("contacts-%s-%s.csv", wsp.Slug, s.now().In(workspaceLocation(wsp)).Format("2006-01-02"))), nil
		})
	api.Get(h, api.Op{ID: "get-contact", Path: ws + "/contacts/{contact}", Summary: "Contact profile: fields, consent per list, stats", Tags: tags, Access: view},
		func(ctx context.Context, in *ContactPathInput) (*api.Body[ContactProfile], error) {
			out, err := s.Profile(ctx, api.WorkspaceFrom(ctx), uuid.MustParse(in.Contact))
			return api.OK(out), err
		})
	api.Patch(h, api.Op{ID: "update-contact", Path: ws + "/contacts/{contact}", Summary: "Change a contact's fields and tags", Tags: tags, Access: edit},
		func(ctx context.Context, in *contactBody[ContactUpdate]) (*api.Body[ContactProfile], error) {
			out, err := s.UpdateContact(ctx, api.WorkspaceFrom(ctx), uuid.MustParse(in.Contact), in.Body)
			return api.OK(out), err
		})
	api.Get(h, api.Op{ID: "contact-activity", Path: ws + "/contacts/{contact}/activity", Summary: "Activity timeline, including which provider delivered each email", Tags: tags, Access: view},
		func(ctx context.Context, in *activityQuery) (*api.Body[ContactActivityPage], error) {
			out, err := s.Activity(ctx, api.WorkspaceFrom(ctx), uuid.MustParse(in.Contact), in.Filter, in.Cursor, in.Limit)
			return api.OK(out), err
		})
	api.Post(h, api.Op{ID: "resend-contact-optin", Path: ws + "/contacts/{contact}/resend-optin", Summary: "Send the opt-in confirmation email again", Tags: tags, Access: edit},
		func(ctx context.Context, in *ContactPathInput) (*api.Body[OptInResendResult], error) {
			out, err := s.ResendOptIn(ctx, api.WorkspaceFrom(ctx), uuid.MustParse(in.Contact))
			return api.OK(out), err
		})
	api.Post(h, api.Op{ID: "suppress-contact", Path: ws + "/contacts/{contact}/suppress", Summary: "Put a contact's address on the suppression list", Tags: tags, Access: edit},
		func(ctx context.Context, in *contactBody[ContactSuppressInput]) (*api.Empty, error) {
			return api.NoContent, s.SuppressContact(ctx, api.WorkspaceFrom(ctx), uuid.MustParse(in.Contact), in.Body)
		})
	api.Post(h, api.Op{ID: "contact-preference-link", Path: ws + "/contacts/{contact}/preference-link", Summary: "Get the contact's preference center link (logged)", Tags: tags, Access: edit},
		func(ctx context.Context, in *ContactPathInput) (*api.Body[PreferenceLinkResult], error) {
			url, err := s.PreferenceLink(ctx, api.WorkspaceFrom(ctx), uuid.MustParse(in.Contact))
			return api.OK(PreferenceLinkResult{URL: url}), err
		})
	api.Get(h, api.Op{ID: "export-contact", Path: ws + "/contacts/{contact}/export", Summary: "Everything stored about one contact, as JSON (GDPR, logged)", Tags: tags, Access: api.Workspace(authz.SubscribersExport)},
		func(ctx context.Context, in *ContactPathInput) (*huma.StreamResponse, error) {
			ex, err := s.ExportContact(ctx, api.WorkspaceFrom(ctx), uuid.MustParse(in.Contact))
			if err != nil {
				return nil, err
			}
			return &huma.StreamResponse{Body: func(hctx huma.Context) {
				hctx.SetHeader("Content-Type", "application/json")
				hctx.SetHeader("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, ex.Name))
				hctx.SetHeader("Cache-Control", "no-store")
				hctx.SetStatus(200)
				_ = ex.Write(hctx.BodyWriter())
			}}, nil
		})

	// Lists and custom fields
	api.Get(h, api.Op{ID: "list-lists", Path: ws + "/lists", Summary: "Lists with their sizes", Tags: tags, Access: view},
		func(ctx context.Context, in *api.WorkspaceInput) (*api.Body[Items[AudienceListSummary]], error) {
			sum, err := s.Summary(ctx, api.WorkspaceFrom(ctx))
			return api.OK(Items[AudienceListSummary]{Items: sum.Lists}), err
		})
	api.Post(h, api.Op{ID: "create-list", Path: ws + "/lists", Summary: "Create a list", Tags: tags, Access: manage, Status: http.StatusCreated},
		func(ctx context.Context, in *body[AudienceListInput]) (*api.Body[AudienceListSummary], error) {
			out, err := s.CreateList(ctx, api.WorkspaceFrom(ctx), in.Body)
			return api.OK(out), err
		})
	api.Patch(h, api.Op{ID: "update-list", Path: ws + "/lists/{id}", Summary: "Rename a list or change its opt-in setting", Tags: tags, Access: manage},
		func(ctx context.Context, in *idBody[AudienceListInput]) (*api.Body[AudienceListSummary], error) {
			out, err := s.UpdateList(ctx, api.WorkspaceFrom(ctx), uuid.MustParse(in.ID), in.Body)
			return api.OK(out), err
		})
	api.Get(h, api.Op{ID: "get-list-impact", Path: ws + "/lists/{id}/impact", Summary: "What deleting a list would affect", Tags: tags, Access: manage},
		func(ctx context.Context, in *AudienceIDInput) (*api.Body[ListImpact], error) {
			out, err := s.ListImpact(ctx, api.WorkspaceFrom(ctx), uuid.MustParse(in.ID))
			return api.OK(out), err
		})
	api.Delete(h, api.Op{ID: "delete-list", Path: ws + "/lists/{id}", Summary: "Delete a list (contacts stay; restorable for 30 days)", Tags: tags, Access: manage},
		func(ctx context.Context, in *AudienceIDInput) (*api.Empty, error) {
			return api.NoContent, s.DeleteList(ctx, api.WorkspaceFrom(ctx), uuid.MustParse(in.ID))
		})
	api.Post(h, api.Op{ID: "restore-list", Path: ws + "/lists/{id}/restore", Summary: "Restore a list deleted in the last 30 days", Tags: tags, Access: manage},
		func(ctx context.Context, in *AudienceIDInput) (*api.Body[ListRestore], error) {
			out, err := s.RestoreList(ctx, api.WorkspaceFrom(ctx), uuid.MustParse(in.ID))
			return api.OK(out), err
		})
	api.Get(h, api.Op{ID: "list-fields", Path: ws + "/fields", Summary: "Custom fields", Tags: tags, Access: view},
		func(ctx context.Context, in *api.WorkspaceInput) (*api.Body[CustomFieldList], error) {
			out, err := s.Fields(ctx, api.WorkspaceFrom(ctx))
			return api.OK(out), err
		})
	api.Post(h, api.Op{ID: "create-field", Path: ws + "/fields", Summary: "Create a custom field", Tags: tags, Access: manage, Status: http.StatusCreated},
		func(ctx context.Context, in *body[CustomFieldInput]) (*api.Body[CustomFieldDef], error) {
			out, err := s.CreateField(ctx, api.WorkspaceFrom(ctx), in.Body)
			return api.OK(out), err
		})
	api.Patch(h, api.Op{ID: "update-field", Path: ws + "/fields/{id}", Summary: "Change a custom field's label, options or display", Tags: tags, Access: manage},
		func(ctx context.Context, in *idBody[CustomFieldInput]) (*api.Body[CustomFieldDef], error) {
			out, err := s.UpdateField(ctx, api.WorkspaceFrom(ctx), uuid.MustParse(in.ID), in.Body)
			return api.OK(out), err
		})
	api.Delete(h, api.Op{ID: "delete-field", Path: ws + "/fields/{id}", Summary: "Delete a custom field (restorable for 30 days, then its values go)",
		Description: "The field leaves forms, segments, views and exports at once; its values stay on contacts for 30 days, then the maintenance pass removes them. Audited.",
		Tags:        tags, Access: manage},
		func(ctx context.Context, in *AudienceIDInput) (*api.Empty, error) {
			return api.NoContent, s.DeleteField(ctx, api.WorkspaceFrom(ctx), uuid.MustParse(in.ID))
		})
	// Lists & fields (review 2026-11 AU-11).
	api.Get(h, api.Op{ID: "fields-overview", Path: ws + "/fields/overview", Summary: "What uses each custom field, how many contacts have a value, and deleted fields",
		Tags: tags, Access: view},
		func(ctx context.Context, in *api.WorkspaceInput) (*api.Body[FieldsOverview], error) {
			out, err := s.FieldsOverview(ctx, api.WorkspaceFrom(ctx))
			return api.OK(out), err
		})
	api.Get(h, api.Op{ID: "get-field-impact", Path: ws + "/fields/{id}/impact", Summary: "What deleting a custom field would affect", Tags: tags, Access: manage},
		func(ctx context.Context, in *AudienceIDInput) (*api.Body[FieldImpact], error) {
			out, err := s.FieldImpact(ctx, api.WorkspaceFrom(ctx), uuid.MustParse(in.ID))
			return api.OK(out), err
		})
	api.Post(h, api.Op{ID: "restore-field", Path: ws + "/fields/{id}/restore", Summary: "Restore a custom field deleted in the last 30 days, with its values",
		Tags: tags, Access: manage},
		func(ctx context.Context, in *AudienceIDInput) (*api.Body[CustomFieldDef], error) {
			out, err := s.RestoreField(ctx, api.WorkspaceFrom(ctx), uuid.MustParse(in.ID))
			return api.OK(out), err
		})

	// Suppression list
	stags := []string{"Suppressions"}
	api.Get(h, api.Op{ID: "list-suppressions", Path: ws + "/suppressions", Summary: "The shared suppression list with reasons, sources and provider sync", Tags: stags, Access: view},
		func(ctx context.Context, in *suppressionsQuery) (*api.Body[SuppressionList], error) {
			out, err := s.ListSuppressions(ctx, api.WorkspaceFrom(ctx), SuppressionFilter{Q: in.Q, Reason: in.Reason, Source: in.Source, Since: in.Since, Import: in.Import, AppliesTo: in.AppliesTo}, ContactPage{Cursor: in.Cursor, Page: in.Page, PageSize: in.PageSize, Exact: in.Exact})
			return api.OK(out), err
		})
	api.Post(h, api.Op{ID: "add-suppressions", Path: ws + "/suppressions", Summary: "Add addresses to the suppression list", Tags: stags, Access: edit},
		func(ctx context.Context, in *body[AddSuppressionsInput]) (*api.Body[SuppressionAddResult], error) {
			out, err := s.AddSuppressions(ctx, api.WorkspaceFrom(ctx), in.Body)
			return api.OK(out), err
		})
	api.Post(h, api.Op{ID: "remove-suppression", Path: ws + "/suppressions/{id}/remove", Summary: "Take an address off the suppression list (needs a reason, logged)", Tags: stags, Access: edit},
		func(ctx context.Context, in *idBody[RemoveSuppressionInput]) (*api.Empty, error) {
			return api.NoContent, s.RemoveSuppression(ctx, api.WorkspaceFrom(ctx), uuid.MustParse(in.ID), in.Body)
		})
	api.Post(h, api.Op{ID: "remove-suppression-upload", Path: ws + "/suppressions/uploads/{id}/remove", Summary: "Take a whole CSV upload off the suppression list (needs a reason, logged)",
		Tags: stags, Access: edit},
		func(ctx context.Context, in *uploadRemoveInput) (*api.Body[RemoveUploadResult], error) {
			id, err := uuid.Parse(in.ID)
			if err != nil {
				return nil, api.NotFound("Upload")
			}
			out, err := s.RemoveUpload(ctx, api.WorkspaceFrom(ctx), id, in.Body)
			return api.OK(out), err
		})
	api.Get(h, api.Op{ID: "export-suppressions", Path: ws + "/suppressions/export", Summary: "Export the suppression list as CSV (logged)", Tags: stags, Access: api.Workspace(authz.SubscribersExport)},
		func(ctx context.Context, in *api.WorkspaceInput) (*File, error) {
			wsp := api.WorkspaceFrom(ctx)
			b, _, err := s.ExportSuppressions(ctx, wsp)
			if err != nil {
				return nil, err
			}
			return download(b, "text/csv; charset=utf-8", fmt.Sprintf("suppressions-%s-%s.csv", wsp.Slug, s.now().In(workspaceLocation(wsp)).Format("2006-01-02"))), nil
		})
	api.Post(h, api.Op{ID: "upload-suppressions", Path: ws + "/suppressions/upload", Summary: "Add addresses from a CSV file", Tags: stags, Access: edit, MaxBodyBytes: 16 << 20},
		func(ctx context.Context, in *uploadInput) (*api.Body[SuppressionUploadResult], error) {
			name := strings.TrimSpace(in.FileName)
			if name == "" {
				name = "upload.csv"
			}
			body := append([]byte(nil), in.RawBody...)
			out, err := s.UploadSuppressions(ctx, api.WorkspaceFrom(ctx), body, name, in.Reason, in.Note)
			return api.OK(out), err
		})
	api.Post(h, api.Op{ID: "pull-suppressions", Path: ws + "/suppressions/pull", Summary: "Pull existing suppressions from connected sending providers", Tags: stags, Access: edit},
		func(ctx context.Context, in *api.WorkspaceInput) (*api.Body[SuppressionPullResult], error) {
			out, err := s.PullFromProviders(ctx, api.WorkspaceFrom(ctx), box)
			return api.OK(out), err
		})
}
