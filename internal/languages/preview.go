package languages

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/store"
)

// PreviewContact is a contact to preview an email as ("Preview as" on the
// composer's canvas), with the version they get.
type PreviewContact struct {
	ID    string `json:"id" format:"uuid"`
	Name  string `json:"name"`
	Email string `json:"email"`
	// Language is the contact's language code as stored; Gets the version
	// they get, their own or the fallback.
	Language string `json:"language"`
	Gets     string `json:"gets"`
	// Offered is false for a language the workspace doesn't offer.
	Offered bool `json:"offered"`
}

// PreviewContactList is one contact per workspace language found in the
// audience, in the workspace's order, then one whose language the workspace
// doesn't offer.
type PreviewContactList struct {
	Items []PreviewContact `json:"items"`
}

// CampaignPreviewContacts picks the contacts to preview a campaign as: one
// per workspace language among its audience's lists (people with a first
// name first), then one whose language isn't offered.
func (s *Service) CampaignPreviewContacts(ctx context.Context, ws *store.Workspace, id uuid.UUID) (PreviewContactList, error) {
	out := PreviewContactList{Items: []PreviewContact{}}
	var audience json.RawMessage
	var settings Settings
	var ch Chooser
	err := s.campaignTx(ctx, ws, id, false, func(q *store.Queries, _ Owner, _ Source, c store.LanguagesCampaignSourceRow, _ sourceState, set Settings) error {
		var err error
		audience, settings = c.Audience, set
		ch, err = s.campaignChooser(ctx, q, ws, id)
		return err
	})
	if err != nil {
		return out, err
	}
	rows, err := s.Q.LanguagesPreviewContacts(ctx, store.LanguagesPreviewContactsParams{WorkspaceID: ws.ID, Lists: audienceLists(audience)})
	if err != nil {
		return out, err
	}
	byBase := map[string]store.LanguagesPreviewContactsRow{}
	var other *store.LanguagesPreviewContactsRow
	for i, r := range rows {
		switch b := Base(r.Language); {
		case settings.Offers(b):
			byBase[b] = r
		case other == nil:
			other = &rows[i]
		}
	}
	add := func(r store.LanguagesPreviewContactsRow, offered bool) {
		out.Items = append(out.Items, PreviewContact{
			ID: r.ID.String(), Name: strings.TrimSpace(r.FirstName + " " + r.LastName), Email: r.Email,
			Language: r.Language, Gets: ch.Pick(r.Language).Language, Offered: offered,
		})
	}
	for _, lang := range settings.Languages {
		if r, ok := byBase[lang]; ok {
			add(r, true)
		}
	}
	if other != nil {
		add(*other, false)
	}
	return out, nil
}

// audienceLists are the list ids of a stored audience definition (its
// segments aren't resolved: the contacts are examples, not recipients).
func audienceLists(raw json.RawMessage) []uuid.UUID {
	var a struct {
		Lists []string `json:"lists"`
	}
	_ = json.Unmarshal(raw, &a)
	out := []uuid.UUID{}
	for _, id := range a.Lists {
		if u, err := uuid.Parse(id); err == nil {
			out = append(out, u)
		}
	}
	return out
}
