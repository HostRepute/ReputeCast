package audience

import (
	"context"
	"fmt"
	"net/http"
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
	JoinedAfter string `query:"joinedAfter" maxLength:"10"`
	Page        int    `query:"page" minimum:"1" default:"1"`
	PageSize    int    `query:"pageSize" minimum:"1" maximum:"100" default:"10"`
}

func (q contactsQuery) filter() ContactFilter {
	return ContactFilter{Q: q.Q, List: q.List, Tier: q.Tier, Mailbox: q.Mailbox, Consent: q.Consent, Source: q.Source, Tag: q.Tag, Segment: q.Segment, JoinedAfter: q.JoinedAfter}
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
	Q        string `query:"q" maxLength:"200" doc:"Address or domain"`
	Reason   string `query:"reason" enum:"hard_bounce,complaint,unsubscribe,manual,"`
	Source   string `query:"source" maxLength:"60"`
	Since    string `query:"since" enum:"7d,30d,"`
	Import   string `query:"import" doc:"Show the rows of one import group"`
	Page     int    `query:"page" minimum:"1" default:"1"`
	PageSize int    `query:"pageSize" minimum:"1" maximum:"100" default:"10"`
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
	api.Put(h, api.Op{ID: "save-sunset-policy", Path: ws + "/audience/sunset-policy", Summary: "Save the sunset policy (applies at the next nightly run)", Tags: tags, Access: manage},
		func(ctx context.Context, in *body[SunsetPolicy]) (*api.Body[SunsetPolicy], error) {
			out, err := s.SavePolicy(ctx, api.WorkspaceFrom(ctx), in.Body)
			return api.OK(out), err
		})

	// Contacts
	api.Get(h, api.Op{ID: "list-contacts", Path: ws + "/contacts", Summary: "List contacts, most recent activity first", Tags: tags, Access: view},
		func(ctx context.Context, in *contactsQuery) (*api.Body[ContactList], error) {
			out, err := s.ListContacts(ctx, api.WorkspaceFrom(ctx), in.filter(), in.Page, in.PageSize)
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
		func(ctx context.Context, in *ContactPathInput) (*File, error) {
			b, name, err := s.ExportContact(ctx, api.WorkspaceFrom(ctx), uuid.MustParse(in.Contact))
			if err != nil {
				return nil, err
			}
			return download(b, "application/json", name), nil
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
	api.Delete(h, api.Op{ID: "delete-list", Path: ws + "/lists/{id}", Summary: "Delete a list (contacts stay)", Tags: tags, Access: manage},
		func(ctx context.Context, in *AudienceIDInput) (*api.Empty, error) {
			return api.NoContent, s.DeleteList(ctx, api.WorkspaceFrom(ctx), uuid.MustParse(in.ID))
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
	api.Delete(h, api.Op{ID: "delete-field", Path: ws + "/fields/{id}", Summary: "Delete a custom field and its values", Tags: tags, Access: manage},
		func(ctx context.Context, in *AudienceIDInput) (*api.Empty, error) {
			return api.NoContent, s.DeleteField(ctx, api.WorkspaceFrom(ctx), uuid.MustParse(in.ID))
		})

	// Suppression list
	stags := []string{"Suppressions"}
	api.Get(h, api.Op{ID: "list-suppressions", Path: ws + "/suppressions", Summary: "The shared suppression list with reasons, sources and provider sync", Tags: stags, Access: view},
		func(ctx context.Context, in *suppressionsQuery) (*api.Body[SuppressionList], error) {
			out, err := s.ListSuppressions(ctx, api.WorkspaceFrom(ctx), SuppressionFilter{Q: in.Q, Reason: in.Reason, Source: in.Source, Since: in.Since, Import: in.Import}, in.Page, in.PageSize)
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
