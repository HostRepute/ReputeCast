package languages

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/contentcheck"
	"github.com/hostrepute/reputecast/internal/mjml"
	"github.com/hostrepute/reputecast/internal/store"
)

// ImportedVersion is a language version written elsewhere, such as the
// demo's sample translations: its content, who translated it and from which
// source version.
type ImportedVersion struct {
	Owner Owner
	// Source is the owner's source as it is now, written in SourceLanguage.
	Source         Source
	SourceLanguage string
	Language       string
	// Translated versions are sent; the others are Drafts.
	Translated                     bool
	Subject, Preheader, MJML, HTML string
	// AB is the version's A/B test, as campaigns store it.
	AB json.RawMessage
	By *uuid.UUID
	At time.Time
	// From is the source version it was translated from. FromMJML,
	// FromSubject and FromPreheader are the source as it was then (empty:
	// as it is now); blocks that changed since are Outdated.
	From                                 int
	FromMJML, FromSubject, FromPreheader string
	// Changes says who changed the source's blocks after From, by source
	// block index ("v6 → v9 · Sofia Rossi, 15 Oct 09:12").
	Changes map[int]BlockChange
	// Label names the save in the version's history.
	Label string
}

// ImportVersion stores a language version written elsewhere, linked to the
// source as it stood when it was translated. Call it in a transaction.
func (s *Service) ImportVersion(ctx context.Context, q *store.Queries, in ImportedVersion) error {
	st, err := s.syncSource(ctx, q, in.Owner, in.Source, in.SourceLanguage)
	if err != nil {
		return err
	}
	if len(in.Changes) > 0 {
		kind, id, ver := in.Owner.key()
		row, err := q.ContentSource(ctx, store.ContentSourceParams{OwnerKind: kind, OwnerID: id, OwnerVersion: ver})
		if err != nil {
			return err
		}
		for i, c := range in.Changes {
			if i >= 0 && i < len(st.Keys) {
				c := c
				st.Keys[i].Changed = &c
			}
		}
		if err := s.saveSource(ctx, q, in.Owner, st, int(row.NextKey)); err != nil {
			return err
		}
	}
	html := in.HTML
	if html == "" && in.MJML != "" {
		res := mjml.Compile(mjml.SetHeadMeta(in.MJML, in.Subject, in.Preheader))
		if len(res.Errors) > 0 {
			return fmt.Errorf("compile the %s version: %s", NameOf(in.Language), res.Errors[0].Message)
		}
		html = res.HTML
	}
	from := in.From
	if from <= 0 {
		from = st.Version
	}
	blocks := Blocks(in.MJML, html)
	links := FreshLinks(st.Blocks, st.Keys, blocks, in.Source.Subject, in.Source.Preheader, from)
	if in.FromMJML != "" {
		// Blocks link to the source's text of the version they were
		// translated from: those that changed since are Outdated.
		old := Blocks(in.FromMJML, "")
		m := Align(keysOf(old), keysOf(st.Blocks))
		byKey := map[string]int{}
		for i, k := range st.Keys {
			byKey[k.Key] = i
		}
		for i, l := range links.Blocks {
			if j, ok := byKey[l.Src]; ok && j < len(m) && m[j] >= 0 {
				links.Blocks[i].Hash, links.Blocks[i].Text = old[m[j]].Hash, old[m[j]].Text
			}
		}
	}
	if in.FromSubject != "" {
		links.Subject = &Link{Hash: HashText(in.FromSubject), Text: in.FromSubject, Version: from}
	}
	if in.FromPreheader != "" {
		links.Preheader = &Link{Hash: HashText(in.FromPreheader), Text: in.FromPreheader, Version: from}
	}
	lb, err := json.Marshal(links)
	if err != nil {
		return err
	}
	status := storedDraft
	var translatedBy *uuid.UUID
	var translatedAt *time.Time
	if in.Translated {
		status, translatedBy, translatedAt = storedTranslated, in.By, &in.At
	}
	kind, id, ver := in.Owner.key()
	if _, err := q.ContentTranslation(ctx, store.ContentTranslationParams{OwnerKind: kind, OwnerID: id, OwnerVersion: ver, Language: in.Language}); err == nil {
		return fmt.Errorf("the %s version already exists", NameOf(in.Language))
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return err
	}
	t := store.ContentTranslation{ID: uuid.Must(uuid.NewV7()), Subject: in.Subject, Preheader: in.Preheader, Mjml: in.MJML, Html: html, PlainText: contentcheck.PlainText(html),
		AbTest: in.AB, Links: lb}
	t.ContentHash = contentHash(t.Subject, t.Preheader, t.Mjml, t.Html, t.AbTest)
	if err := q.ContentTranslationInsert(ctx, store.ContentTranslationInsertParams{ID: t.ID, WorkspaceID: in.Owner.WorkspaceID, OwnerKind: kind, OwnerID: id,
		OwnerVersion: ver, Language: in.Language, Status: status, Subject: t.Subject, Preheader: t.Preheader, Mjml: t.Mjml, Html: t.Html, PlainText: t.PlainText,
		AbTest: t.AbTest, Links: t.Links, Version: 1, ContentHash: t.ContentHash, TranslatedFrom: int32(from), TranslatedBy: translatedBy, TranslatedAt: translatedAt,
		CreatedBy: in.By, CreatedAt: in.At, UpdatedBy: in.By, UpdatedAt: in.At, AutosavedAt: &in.At}); err != nil {
		return err
	}
	label := in.Label
	if label == "" {
		label = "translated from " + NameOf(st.Language)
	}
	return q.ContentTranslationVersionInsert(ctx, store.ContentTranslationVersionInsertParams{TranslationID: t.ID, Version: 1, Subject: t.Subject, Preheader: t.Preheader,
		Mjml: t.Mjml, Html: t.Html, PlainText: t.PlainText, AbTest: t.AbTest, Links: t.Links, SavedBy: in.By, SavedAt: in.At, Label: label, Source: "manual",
		SizeBytes: int32(len(t.Html))})
}
