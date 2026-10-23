package mcpserver

import (
	"context"
	"errors"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/languages"
	"github.com/hostrepute/reputecast/internal/store"
)

// Content languages: the language versions of campaigns.

// addLanguageTools registers get_campaign_languages.
func (s *Service) addLanguageTools(srv *mcp.Server) {
	tool(s, srv, "get_campaign_languages", authz.CampaignsView, func(ctx context.Context, c *agentCall, in CampaignRef) (any, entry, error) {
		e := entry{action: "campaign.languages_read", targetType: "campaign", targetName: in.Campaign}
		cp, err := s.findCampaign(ctx, c.Workspace.ID, in.Campaign)
		if err != nil {
			return nil, e, err
		}
		e.targetID, e.targetName = cp.ID.String(), cp.Name
		if s.Languages == nil {
			return nil, e, errors.New("content languages aren't available in this process")
		}
		ws := c.Workspace
		ls, err := s.Languages.CampaignLanguages(ctx, &ws, nil, cp.ID)
		if err != nil {
			return nil, e, err
		}
		versions := make([]map[string]any, 0, len(ls.Versions))
		for _, v := range ls.Versions {
			m := map[string]any{"language": v.Language, "name": v.Name, "status": v.Status, "gets": v.Gets}
			if v.Readers != nil {
				m["readers"] = *v.Readers
			}
			if v.Status == languages.StatusOutdated {
				m["outdated_blocks"] = v.Attention
				m["changed_in_source"] = v.ChangedBlocks
				m["translated_from_version"] = v.TranslatedFrom
			}
			if v.Status == languages.StatusDraft {
				m["blocks_translated"] = v.Translated
				m["blocks"] = v.Blocks
			}
			if v.Held {
				m["held"] = true
			}
			if !v.Offered {
				m["note"] = "Not one of the workspace's languages any more: kept, not sent."
			}
			if v.Subject != "" {
				m["subject"] = v.Subject
			}
			versions = append(versions, m)
		}
		e.detail = ls.Source
		return map[string]any{"campaign": cp.Name, "source": ls.Source, "source_version": ls.SourceVersion, "fallback": ls.Fallback, "versions": versions,
			"note": "Readers get the version in their language when it's up to date or outdated, otherwise the fallback's, otherwise the source. Drafts are never sent."}, e, nil
	}, s.campaignTarget("campaign.languages_read"))
}

// updateLanguage changes a draft's language version for update_campaign_content.
func (s *Service) updateLanguage(ctx context.Context, c *agentCall, cp store.McpFindCampaignRow, in updateContentIn, e entry) (any, entry, error) {
	ws := c.Workspace
	saved, err := s.Languages.AgentSaveCampaign(ctx, &ws, c.Principal, cp.ID, in.Language, languages.AgentEdit{Subject: in.Subject, Preheader: in.Preheader, MJML: in.MJML})
	if err != nil {
		return nil, e, err
	}
	e.action = "campaign.language.updated"
	// The language by its English name: a name, as the audit log keeps it.
	if fields := changedFields(in); len(fields) > 0 {
		e.detailText = keep("detail.languageChanged", i18n.Args{"language": languages.NameOf(saved.Language), "fields": fields})
	} else {
		e.detailText = keep("detail.languageUnchanged", i18n.Args{"language": languages.NameOf(saved.Language)})
	}
	e.before = map[string]any{"language": saved.Language, "subject": saved.BeforeSubject, "preheader": saved.BeforePreheader, "version": saved.BeforeVersion}
	e.after = map[string]any{"language": saved.Language, "subject": saved.Subject, "version": saved.Version}
	return map[string]any{"id": cp.ID, "name": cp.Name, "language": saved.Language, "version": saved.Version, "subject": saved.Subject}, e, nil
}
