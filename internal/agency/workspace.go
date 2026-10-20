package agency

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/store"
)

// WorkspaceAgency is what a workspace shows about the agency that manages
// it: the "Managed by" banner with the client's contact, the agency's
// staff among its members ("via Northwind Studio"), and, for staff, the
// client details only the agency keeps.
type WorkspaceAgency struct {
	// Managed is true in the agency's workspaces (its clients and its own).
	Managed bool   `json:"managed"`
	Kind    string `json:"kind,omitempty" enum:"client,agency"`
	Agency  string `json:"agency,omitempty"`
	// AgencyInitials and Host draw the agency's tile and address.
	AgencyInitials string `json:"agencyInitials,omitempty"`
	Host           string `json:"host,omitempty"`
	// Contact is the client's account manager.
	Contact *AgencyPersonRef `json:"contact,omitempty"`
	// Staff are the members of this workspace who are agency staff.
	Staff []WorkspaceAgencyStaff `json:"staff"`
	// ViewerIsStaff: you're on the agency's staff.
	ViewerIsStaff bool `json:"viewerIsStaff"`
	// Client details, for staff only.
	ClientSince *time.Time  `json:"clientSince,omitempty"`
	Template    string      `json:"template,omitempty"`
	Rate        *AgencyRate `json:"rate,omitempty"`
	RateSince   *time.Time  `json:"rateSince,omitempty"`
	// Billed is false when the client isn't billed (staff with agency:usage).
	Billed bool `json:"billed"`
	// CanSaveTemplate and CanAddToLibrary: you hold agency:library.
	CanSaveTemplate bool `json:"canSaveTemplate"`
}

// WorkspaceAgencyStaff is a staff member in this workspace.
type WorkspaceAgencyStaff struct {
	UserID string `json:"userId"`
	Level  string `json:"level" enum:"agency_manager,account_manager"`
	Title  string `json:"title,omitempty"`
	// Contact: they're the client's contact.
	Contact     bool `json:"contact"`
	EveryClient bool `json:"everyClient"`
}

// Workspace describes the agency for one of its workspaces.
func (s *Service) Workspace(ctx context.Context, ws *store.Workspace) (WorkspaceAgency, error) {
	out := WorkspaceAgency{Staff: []WorkspaceAgencyStaff{}}
	st, err := s.settings(ctx, s.Q)
	if err != nil || !st.Enabled {
		return out, err
	}
	w, err := s.Q.AgencyWorkspace(ctx, ws.ID)
	if isNoRows(err) {
		return out, nil
	} else if err != nil {
		return out, err
	}
	out.Managed, out.Kind, out.Agency, out.AgencyInitials, out.Host = true, w.Kind, st.Name, Initials(st.Name), host(st.PublicURL)
	if w.AccountManagerID != nil {
		out.Contact = &AgencyPersonRef{ID: w.AccountManagerID.String(), Name: displayName(w.AccountManagerName, w.AccountManagerEmail), Email: w.AccountManagerEmail, AvatarColor: w.AccountManagerColor}
	}
	people, err := s.people(ctx)
	if err != nil {
		return out, err
	}
	all, err := s.Q.AgencyWorkspaces(ctx)
	if err != nil {
		return out, err
	}
	clients := clientIDs(all)
	ids := []uuid.UUID{}
	for _, p := range people {
		if p.UserID != nil {
			ids = append(ids, *p.UserID)
		}
	}
	ms, err := s.Q.AgencyMemberships(ctx, ids)
	if err != nil {
		return out, err
	}
	inWS := map[uuid.UUID]bool{}
	count := map[uuid.UUID]int{}
	for _, m := range ms {
		if m.WorkspaceID == ws.ID {
			inWS[m.UserID] = true
		}
		for _, c := range clients {
			if c == m.WorkspaceID {
				count[m.UserID]++
			}
		}
	}
	for _, p := range people {
		if p.UserID == nil || !inWS[*p.UserID] {
			continue
		}
		out.Staff = append(out.Staff, WorkspaceAgencyStaff{UserID: p.UserID.String(), Level: p.Level, Title: p.Title,
			Contact: w.AccountManagerID != nil && *w.AccountManagerID == *p.UserID, EveryClient: len(clients) > 0 && count[*p.UserID] == len(clients)})
	}
	u := api.UserFrom(ctx)
	if u == nil {
		return out, nil
	}
	me, err := s.staffOf(ctx, u)
	if err != nil || me == nil {
		return out, err
	}
	out.ViewerIsStaff = true
	cs := w.ClientSince
	out.ClientSince, out.Template, out.CanSaveTemplate = &cs, w.TemplateName, me.Can(authz.AgencyLibrary)
	if me.Can(authz.AgencyUsage) {
		rates, err := s.Q.AgencyRates(ctx, []uuid.UUID{ws.ID})
		if err != nil {
			return out, err
		}
		out.Rate, out.RateSince = rateFor(rates, ws.ID, monthStart(s.now()))
		out.Billed = out.Rate != nil
	}
	return out, nil
}

// AgencyGalleryView is the "Northwind library" tab of a client's composer
// gallery and automation starters: the items that can be used there.
type AgencyGalleryView struct {
	Agency string              `json:"agency"`
	Items  []AgencyLibraryItem `json:"items"`
	// Copies are this workspace's copies, with the library's latest version.
	Copies []AgencyGalleryCopy `json:"copies"`
	// CanUse: you can edit that kind of content here.
	CanUse bool   `json:"canUse"`
	Reason string `json:"reason,omitempty"`
	// You is how you work here ("Mei Lin · Admin in Contoso Coffee, via Northwind Studio").
	Role string `json:"role,omitempty"`
}

// AgencyGalleryCopy is a copy of a library item in this workspace.
type AgencyGalleryCopy struct {
	ID       string `json:"id"`
	ItemID   string `json:"itemId"`
	Kind     string `json:"kind"`
	TargetID string `json:"targetId"`
	Name     string `json:"name"`
	Version  int    `json:"version"`
	Latest   int    `json:"latest"`
}

// Gallery returns the library items usable in a workspace, for its staff.
func (s *Service) Gallery(ctx context.Context, ws *store.Workspace, kind string) (AgencyGalleryView, error) {
	me, st, err := s.require(ctx, authz.AgencyView)
	if err != nil {
		return AgencyGalleryView{}, err
	}
	l := i18n.LangFrom(ctx)
	if _, err := s.Q.AgencyWorkspace(ctx, ws.ID); err != nil {
		return AgencyGalleryView{}, api.MapError(err, "Agency workspace")
	}
	lib, err := s.Library(ctx)
	if err != nil {
		return AgencyGalleryView{}, err
	}
	out := AgencyGalleryView{Agency: st.Name, Items: []AgencyLibraryItem{}, Copies: []AgencyGalleryCopy{}}
	kinds := map[string][]string{"composer": {KindSection, KindTemplate}, "automation": {KindAutomation}, "transactional": {KindTransactional}, "sms": {KindSMS}}[kind]
	if kinds == nil {
		kinds = []string{KindSection, KindTemplate}
	}
	for _, it := range lib.Items {
		for _, k := range kinds {
			if it.Kind == k {
				out.Items = append(out.Items, it)
			}
		}
	}
	p := api.PrincipalFrom(ctx)
	perm := editPermission(kinds[0])
	out.CanUse = p.Can(perm)
	if !out.CanUse {
		out.Reason = mCopyNeedsHere.Text(l, i18n.Args{"permission": string(perm)})
	}
	out.Role = api.RoleName(l, p.RoleKey, p.RoleName)
	copies, err := s.Q.AgencyCopiesOfWorkspace(ctx, ws.ID)
	if err != nil {
		return out, err
	}
	for _, c := range copies {
		out.Copies = append(out.Copies, AgencyGalleryCopy{ID: c.ID.String(), ItemID: c.ItemID.String(), Kind: c.Kind, TargetID: c.TargetID.String(), Name: c.Name,
			Version: int(c.Version), Latest: int(c.Latest)})
	}
	_ = me
	return out, nil
}

// UseItem copies a library item into this workspace (the gallery's "Use a
// copy"). It needs the edit permission of that kind of content here.
func (s *Service) UseItem(ctx context.Context, ws *store.Workspace, itemID string) (AgencyCopyResult, error) {
	me, _, err := s.require(ctx, authz.AgencyView)
	if err != nil {
		return AgencyCopyResult{}, err
	}
	set, err := s.workspacesOf(ctx, s.Q, me)
	if err != nil {
		return AgencyCopyResult{}, err
	}
	w, ok := set.find(ws.ID)
	if !ok {
		return AgencyCopyResult{}, api.NotFound("Agency workspace")
	}
	return s.copyItem(ctx, me, set, w, itemID)
}
