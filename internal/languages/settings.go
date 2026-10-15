package languages

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/hostrepute/reputecast/internal/access"
	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/store"
)

// Service holds the content-languages domain.
type Service struct {
	Pool  *pgxpool.Pool
	Q     *store.Queries
	Audit *audit.Logger
	Now   func() time.Time
	// AudienceLanguages counts the final recipients of a stored audience
	// definition per contact language code, as Send Readiness counts them,
	// and returns their total.
	AudienceLanguages func(ctx context.Context, ws *store.Workspace, audience json.RawMessage) (map[string]int64, int64, error)
	// Changed is told when a campaign's language versions changed (the
	// approvals domain re-checks open approvals).
	Changed func(ctx context.Context, ws *store.Workspace, campaignID uuid.UUID)
}

func (s *Service) now() time.Time {
	if s.Now != nil {
		return s.Now()
	}
	return time.Now()
}

// Settings are a workspace's languages: what it writes in, its default and
// its fallback.
type Settings struct {
	Default  string
	Fallback string
	// Languages are the active languages, the default first.
	Languages []string
	offered   map[string]bool
}

// Offers reports whether a contact language (a base or regional code) is
// one of the workspace's languages.
func (st Settings) Offers(code string) bool { return st.offered[Base(code)] }

// Settings returns a workspace's languages. A workspace that never set them
// writes in English.
func (s *Service) Settings(ctx context.Context, q *store.Queries, wsID uuid.UUID) (Settings, error) {
	if q == nil {
		q = s.Q
	}
	rows, err := q.LanguagesList(ctx, wsID)
	if err != nil {
		return Settings{}, err
	}
	return settingsOf(rows), nil
}

func settingsOf(rows []store.LanguagesListRow) Settings {
	st := Settings{offered: map[string]bool{}, Languages: []string{}}
	for _, r := range rows {
		if r.RemovedAt != nil {
			continue
		}
		st.offered[r.Code] = true
		st.Languages = append(st.Languages, r.Code)
		if r.IsDefault {
			st.Default = r.Code
		}
		if r.IsFallback {
			st.Fallback = r.Code
		}
	}
	if len(st.Languages) == 0 {
		st.Languages = []string{"en"}
		st.offered["en"] = true
	}
	if st.Default == "" {
		st.Default = st.Languages[0]
	}
	if st.Fallback == "" {
		st.Fallback = st.Default
	}
	return st
}

// LanguagePageCopy is the state of a language's public page copy.
type LanguagePageCopy struct {
	Status string `json:"status" enum:"live,incomplete,not_set_up"`
	// Missing counts the strings still to write when Incomplete.
	Missing int        `json:"missing"`
	By      string     `json:"by"`
	At      *time.Time `json:"at,omitempty"`
	// BuiltinLabels is true when ReputeCast's own labels (buttons, field
	// labels, messages) ship in this language.
	BuiltinLabels bool `json:"builtinLabels"`
}

// WorkspaceLanguage is one language of the workspace.
type WorkspaceLanguage struct {
	Code   string `json:"code" example:"es"`
	Name   string `json:"name" example:"Spanish"`
	Native string `json:"native" example:"Español"`
	// Codes are the codes its contacts have, most common first ("es",
	// "es-MX", "es-ES"); MoreCodes says there are others.
	Codes     []string `json:"codes"`
	MoreCodes bool     `json:"moreCodes"`
	// Contacts get this language: its own contacts (Own), plus, for the
	// fallback, contacts with no language or one the workspace doesn't offer.
	Contacts   int64            `json:"contacts"`
	Own        int64            `json:"own"`
	IsDefault  bool             `json:"isDefault"`
	IsFallback bool             `json:"isFallback"`
	PageCopy   LanguagePageCopy `json:"pageCopy"`
	// AddedBy is empty for the language the workspace started with.
	AddedBy string     `json:"addedBy"`
	AddedAt *time.Time `json:"addedAt,omitempty"`
}

// WorkspaceLanguages is Settings › General › Languages.
type WorkspaceLanguages struct {
	Languages     []WorkspaceLanguage `json:"languages"`
	Default       string              `json:"default"`
	Fallback      string              `json:"fallback"`
	TotalContacts int64               `json:"totalContacts"`
	NoLanguage    int64               `json:"noLanguage"`
	// NotOffered counts contacts with a language the workspace doesn't
	// offer; NotOfferedCodes lists those codes, most common first.
	NotOffered      int64    `json:"notOffered"`
	NotOfferedCodes []string `json:"notOfferedCodes"`
	// CanManage is workspace:manage; Helpers hold it.
	CanManage bool     `json:"canManage"`
	Helpers   []string `json:"helpers"`
}

// contactCounts are the workspace's contacts per stored language code.
type contactCounts struct {
	total  int64
	none   int64
	byCode map[string]int64 // normalized code ("es-MX") → contacts
	byBase map[string]int64
}

func (s *Service) contactCounts(ctx context.Context, q *store.Queries, wsID uuid.UUID) (contactCounts, error) {
	rows, err := q.LanguagesContactCodes(ctx, wsID)
	if err != nil {
		return contactCounts{}, err
	}
	c := contactCounts{byCode: map[string]int64{}, byBase: map[string]int64{}}
	for _, r := range rows {
		c.total += r.N
		code, ok := Normalize(r.Code)
		if strings.TrimSpace(r.Code) == "" {
			c.none += r.N
			continue
		}
		if !ok {
			code = r.Code
		}
		c.byCode[code] += r.N
		c.byBase[Base(code)] += r.N
	}
	return c, nil
}

// codesOf lists the codes of one base language, most common first.
func (c contactCounts) codesOf(base string) []string {
	out := []string{}
	for code := range c.byCode {
		if Base(code) == base {
			out = append(out, code)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		a, b := c.byCode[out[i]], c.byCode[out[j]]
		if a != b {
			return a > b
		}
		return out[i] < out[j]
	})
	return out
}

// List returns the workspace's languages with their contacts and page copy.
func (s *Service) List(ctx context.Context, ws *store.Workspace, p *authz.Principal) (WorkspaceLanguages, error) {
	rows, err := s.Q.LanguagesList(ctx, ws.ID)
	if err != nil {
		return WorkspaceLanguages{}, err
	}
	if len(rows) == 0 {
		if err := s.Q.LanguagesEnsureDefault(ctx, ws.ID); err != nil {
			return WorkspaceLanguages{}, err
		}
		if rows, err = s.Q.LanguagesList(ctx, ws.ID); err != nil {
			return WorkspaceLanguages{}, err
		}
	}
	counts, err := s.contactCounts(ctx, s.Q, ws.ID)
	if err != nil {
		return WorkspaceLanguages{}, err
	}
	st := settingsOf(rows)
	out := WorkspaceLanguages{Languages: []WorkspaceLanguage{}, Default: st.Default, Fallback: st.Fallback, TotalContacts: counts.total, NoLanguage: counts.none,
		NotOfferedCodes: []string{}, Helpers: []string{}}
	for code, n := range counts.byCode {
		if !st.Offers(code) {
			out.NotOffered += n
			out.NotOfferedCodes = append(out.NotOfferedCodes, code)
		}
	}
	sort.Slice(out.NotOfferedCodes, func(i, j int) bool {
		a, b := counts.byCode[out.NotOfferedCodes[i]], counts.byCode[out.NotOfferedCodes[j]]
		if a != b {
			return a > b
		}
		return out.NotOfferedCodes[i] < out.NotOfferedCodes[j]
	})
	for _, r := range rows {
		if r.RemovedAt != nil {
			continue
		}
		l, _ := Lookup(r.Code)
		wl := WorkspaceLanguage{Code: r.Code, Name: l.Name, Native: l.Native, Own: counts.byBase[r.Code], IsDefault: r.IsDefault, IsFallback: r.IsFallback,
			PageCopy: LanguagePageCopy{Status: r.PageCopyStatus, Missing: int(r.PageCopyMissing), By: r.PageCopyByName, At: r.PageCopyAt, BuiltinLabels: BuiltinLabels[r.Code]},
			AddedBy: r.AddedByName}
		if r.AddedByName != "" {
			at := r.AddedAt
			wl.AddedAt = &at
		}
		codes := counts.codesOf(r.Code)
		if len(codes) == 0 {
			codes = []string{r.Code}
		}
		if len(codes) > 3 {
			codes, wl.MoreCodes = codes[:3], true
		}
		wl.Codes = codes
		wl.Contacts = wl.Own
		if r.IsFallback {
			wl.Contacts += out.NoLanguage + out.NotOffered
		}
		out.Languages = append(out.Languages, wl)
	}
	if p != nil {
		out.CanManage = p.Can(authz.WorkspaceManage)
		if !out.CanManage {
			out.Helpers = nonNil(access.Helpers(ctx, s.Q, ws.ID, authz.WorkspaceManage))
		}
	}
	return out, nil
}

func nonNil(s []string) []string {
	if s == nil {
		return []string{}
	}
	return s
}

// CatalogLanguage is a language that can be added, with the contacts who
// already have it.
type CatalogLanguage struct {
	Code    string   `json:"code"`
	Name    string   `json:"name"`
	Native  string   `json:"native"`
	RTL     bool     `json:"rtl"`
	Regions []string `json:"regions"`
	// Contacts already have it; ContactCodes are their codes ("pt-BR").
	Contacts      int64    `json:"contacts"`
	ContactCodes  []string `json:"contactCodes"`
	BuiltinLabels bool     `json:"builtinLabels"`
	// Added is true for the workspace's languages.
	Added bool `json:"added"`
}

// LanguageCatalog lists the languages to add: those the workspace's
// contacts already have first, then the rest by name. A query matches the
// English name, the language's own name or its code.
func (s *Service) LanguageCatalog(ctx context.Context, ws *store.Workspace, query string) ([]CatalogLanguage, error) {
	st, err := s.Settings(ctx, s.Q, ws.ID)
	if err != nil {
		return nil, err
	}
	counts, err := s.contactCounts(ctx, s.Q, ws.ID)
	if err != nil {
		return nil, err
	}
	q := fold(query)
	out := []CatalogLanguage{}
	for _, l := range Catalog() {
		if q != "" && !strings.HasPrefix(fold(l.Name), q) && !strings.HasPrefix(fold(l.Native), q) && !strings.HasPrefix(l.Code, q) &&
			!strings.Contains(fold(l.Name), q) {
			continue
		}
		codes := counts.codesOf(l.Code)
		out = append(out, CatalogLanguage{Code: l.Code, Name: l.Name, Native: l.Native, RTL: l.RTL, Regions: nonNil(l.Regions),
			Contacts: counts.byBase[l.Code], ContactCodes: codes, BuiltinLabels: BuiltinLabels[l.Code], Added: st.Offers(l.Code)})
	}
	sort.SliceStable(out, func(i, j int) bool {
		a, b := out[i], out[j]
		if a.Added != b.Added {
			return !a.Added
		}
		if (a.Contacts > 0) != (b.Contacts > 0) {
			return a.Contacts > 0
		}
		if a.Contacts != b.Contacts {
			return a.Contacts > b.Contacts
		}
		return false
	})
	return out, nil
}

// ErrRTL is the board's sentence for right-to-left languages.
func rtlRefusal(l Language, fallback string) error {
	return api.Invalid("body.code", fmt.Sprintf("Right-to-left languages aren't supported yet. Contacts with %s get %s.", l.Name, NameOf(fallback)))
}

// AddLanguage adds a language to the workspace.
func (s *Service) AddLanguage(ctx context.Context, ws *store.Workspace, p *authz.Principal, code string) (WorkspaceLanguages, error) {
	l, ok := Lookup(code)
	if !ok || Base(code) != strings.ToLower(strings.TrimSpace(code)) {
		return WorkspaceLanguages{}, api.Invalid("body.code", "Pick a language from the list. A workspace language is a base language, such as es; es-MX matches it.")
	}
	st, err := s.Settings(ctx, s.Q, ws.ID)
	if err != nil {
		return WorkspaceLanguages{}, err
	}
	if l.RTL {
		return WorkspaceLanguages{}, rtlRefusal(l, st.Fallback)
	}
	if st.Offers(l.Code) {
		return WorkspaceLanguages{}, api.Conflict(l.Name + " is already one of your languages.")
	}
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if err := q.LanguagesEnsureDefault(ctx, ws.ID); err != nil {
			return err
		}
		if _, err := q.LanguagesLock(ctx, ws.ID); err != nil {
			return err
		}
		pageCopy := "not_set_up"
		if err := q.LanguagesAdd(ctx, store.LanguagesAddParams{WorkspaceID: ws.ID, Code: l.Code, AddedBy: actorID(p), At: s.now(), PageCopyStatus: pageCopy}); err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "language.add", TargetType: "language", TargetID: l.Code, TargetName: l.Name,
			After: map[string]any{"code": l.Code}, Metadata: map[string]any{"permission": string(authz.WorkspaceManage)}})
	})
	if err != nil {
		return WorkspaceLanguages{}, err
	}
	return s.List(ctx, ws, p)
}

func actorID(p *authz.Principal) *uuid.UUID {
	if p == nil || p.UserID == uuid.Nil {
		return nil
	}
	id := p.UserID
	return &id
}

// RemovalItem is something with a version in a language being removed.
type RemovalItem struct {
	Kind string `json:"kind" enum:"campaign,automation_email,transactional_template"`
	ID   string `json:"id"`
	Name string `json:"name"`
	// Detail says what it is and where its version stands ("Campaign draft · German is outdated").
	Detail string `json:"detail"`
}

// LanguageRemoval is what removing a language affects.
type LanguageRemoval struct {
	Code string        `json:"code"`
	Name string        `json:"name"`
	Has  []RemovalItem `json:"has"`
	// Readers are the contacts with this language: they get the fallback.
	Readers  int64  `json:"readers"`
	Fallback string `json:"fallback"`
	// Refusal is set when it can't be removed (the default or the fallback).
	Refusal string `json:"refusal,omitempty"`
}

// Removal says what removing a language affects.
func (s *Service) Removal(ctx context.Context, ws *store.Workspace, code string) (LanguageRemoval, error) {
	st, err := s.Settings(ctx, s.Q, ws.ID)
	if err != nil {
		return LanguageRemoval{}, err
	}
	code = Base(code)
	if !st.Offers(code) {
		return LanguageRemoval{}, api.NotFound("Language")
	}
	out := LanguageRemoval{Code: code, Name: NameOf(code), Has: []RemovalItem{}, Fallback: st.Fallback, Refusal: refusal(st, code)}
	counts, err := s.contactCounts(ctx, s.Q, ws.ID)
	if err != nil {
		return out, err
	}
	out.Readers = counts.byBase[code]
	rows, err := s.Q.LanguagesVersionsIn(ctx, store.LanguagesVersionsInParams{WorkspaceID: ws.ID, Language: code})
	if err != nil {
		return out, err
	}
	seen := map[string]bool{}
	for _, r := range rows {
		item := RemovalItem{Kind: r.OwnerKind, ID: r.OwnerID.String(), Name: r.Name}
		switch r.OwnerKind {
		case "campaign":
			item.Detail = campaignStatusWord(r.CampaignStatus)
		case "automation_email":
			// One row per automation: its emails' versions are listed together.
			if r.AutomationName != "" {
				item.Name, item.ID = r.AutomationName, r.AutomationID.String()
			}
			item.Detail = "Automation email"
		case "transactional_template":
			item.Detail = "Transactional template"
		}
		key := r.OwnerKind + ":" + item.ID
		if seen[key] {
			continue
		}
		seen[key] = true
		out.Has = append(out.Has, item)
	}
	return out, nil
}

func campaignStatusWord(status string) string {
	switch status {
	case "scheduled":
		return "Scheduled campaign"
	case "approval", "approved":
		return "Campaign waiting to send"
	case "sending", "paused":
		return "Campaign sending"
	}
	return "Campaign draft"
}

func refusal(st Settings, code string) string {
	name := NameOf(code)
	switch {
	case st.Default == code:
		return fmt.Sprintf("%s is the default language. Choose another default before removing it.", name)
	case st.Fallback == code:
		return fmt.Sprintf("%s is the fallback language. Choose another fallback before removing it.", name)
	}
	return ""
}

// RemoveLanguage takes a language out of the workspace. Its versions and
// page copy are kept, but no longer used; its contacts get the fallback.
func (s *Service) RemoveLanguage(ctx context.Context, ws *store.Workspace, p *authz.Principal, code string) (WorkspaceLanguages, error) {
	code = Base(code)
	err := db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if _, err := q.LanguagesLock(ctx, ws.ID); err != nil {
			return err
		}
		st, err := s.Settings(ctx, q, ws.ID)
		if err != nil {
			return err
		}
		if !st.Offers(code) {
			return api.NotFound("Language")
		}
		if why := refusal(st, code); why != "" {
			return api.Conflict(why)
		}
		at := s.now()
		n, err := q.LanguagesRemove(ctx, store.LanguagesRemoveParams{WorkspaceID: ws.ID, Code: code, At: &at, RemovedBy: actorID(p)})
		if err != nil {
			return err
		}
		if n == 0 {
			return api.NotFound("Language")
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "language.remove", TargetType: "language", TargetID: code, TargetName: NameOf(code),
			Before: map[string]any{"code": code}, Metadata: map[string]any{"permission": string(authz.WorkspaceManage)}})
	})
	if err != nil {
		return WorkspaceLanguages{}, err
	}
	return s.List(ctx, ws, p)
}

// LanguageDefaults are the default and fallback languages.
type LanguageDefaults struct {
	Default  string `json:"default" pattern:"^[a-z]{2,3}$"`
	Fallback string `json:"fallback" pattern:"^[a-z]{2,3}$"`
}

// FallbackImpact is what changing the fallback does.
type FallbackImpact struct {
	Fallback string `json:"fallback"`
	// Moved counts the contacts who'd get the new fallback where an email has
	// a version in it: those with no language and those whose language
	// isn't offered.
	Moved      int64 `json:"moved"`
	NoLanguage int64 `json:"noLanguage"`
	NotOffered int64 `json:"notOffered"`
}

// FallbackPreview says how many contacts a new fallback moves.
func (s *Service) FallbackPreview(ctx context.Context, ws *store.Workspace, code string) (FallbackImpact, error) {
	st, err := s.Settings(ctx, s.Q, ws.ID)
	if err != nil {
		return FallbackImpact{}, err
	}
	code = Base(code)
	if !st.Offers(code) {
		return FallbackImpact{}, api.Invalid("code", "Pick one of your languages.")
	}
	counts, err := s.contactCounts(ctx, s.Q, ws.ID)
	if err != nil {
		return FallbackImpact{}, err
	}
	out := FallbackImpact{Fallback: code, NoLanguage: counts.none}
	for c, n := range counts.byCode {
		if !st.Offers(c) {
			out.NotOffered += n
		}
	}
	if code != st.Fallback {
		out.Moved = out.NoLanguage + out.NotOffered
	}
	return out, nil
}

// SetDefaults changes the default and fallback languages.
func (s *Service) SetDefaults(ctx context.Context, ws *store.Workspace, p *authz.Principal, in LanguageDefaults) (WorkspaceLanguages, error) {
	err := db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if err := q.LanguagesEnsureDefault(ctx, ws.ID); err != nil {
			return err
		}
		if _, err := q.LanguagesLock(ctx, ws.ID); err != nil {
			return err
		}
		st, err := s.Settings(ctx, q, ws.ID)
		if err != nil {
			return err
		}
		if !st.Offers(in.Default) {
			return api.Invalid("body.default", "Pick one of your languages as the default.")
		}
		if !st.Offers(in.Fallback) {
			return api.Invalid("body.fallback", "Pick one of your languages as the fallback.")
		}
		meta := map[string]any{"permission": string(authz.WorkspaceManage)}
		if in.Default != st.Default {
			if err := q.LanguagesClearDefault(ctx, ws.ID); err != nil {
				return err
			}
			if _, err := q.LanguagesSetDefault(ctx, store.LanguagesSetDefaultParams{WorkspaceID: ws.ID, Code: in.Default}); err != nil {
				return err
			}
			if err := s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "language.default_change", TargetType: "language", TargetID: in.Default,
				TargetName: NameOf(in.Default), Before: map[string]any{"default": st.Default}, After: map[string]any{"default": in.Default}, Metadata: meta}); err != nil {
				return err
			}
		}
		if in.Fallback != st.Fallback {
			if err := q.LanguagesClearFallback(ctx, ws.ID); err != nil {
				return err
			}
			if _, err := q.LanguagesSetFallback(ctx, store.LanguagesSetFallbackParams{WorkspaceID: ws.ID, Code: in.Fallback}); err != nil {
				return err
			}
			if err := s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "language.fallback_change", TargetType: "language", TargetID: in.Fallback,
				TargetName: NameOf(in.Fallback), Before: map[string]any{"fallback": st.Fallback}, After: map[string]any{"fallback": in.Fallback}, Metadata: meta}); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return WorkspaceLanguages{}, err
	}
	return s.List(ctx, ws, p)
}

// SetPageCopy records the status of a language's page copy. The forms and
// public pages domain calls it when page copy changes; it doesn't audit
// (the page copy change is audited where it happens).
func (s *Service) SetPageCopy(ctx context.Context, q *store.Queries, wsID uuid.UUID, code, status string, missing int, by *uuid.UUID, at time.Time) error {
	if q == nil {
		q = s.Q
	}
	switch status {
	case "live", "incomplete", "not_set_up":
	default:
		return fmt.Errorf("languages: unknown page copy status %q", status)
	}
	n, err := q.LanguagesSetPageCopy(ctx, store.LanguagesSetPageCopyParams{WorkspaceID: wsID, Code: Base(code), Status: status, Missing: int32(missing), By: by, At: &at})
	if err != nil {
		return err
	}
	if n == 0 {
		return errors.New("languages: not a workspace language")
	}
	return nil
}
