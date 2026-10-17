package audience

import (
	"context"
	"strconv"
	"strings"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/scoring"
)

// Contact scoring and growth sources in the contacts table
// (audience-insights brief §6): the Score column and its sort, the band
// filter, the score range, and the source, UTM and referring-domain
// filters under More filters.

// ScoreGrowthFilter narrows contacts by their score and by how they first
// joined. Saved views keep it with the rest of the filter.
type ScoreGrowthFilter struct {
	// Band is a score band, or "none" for contacts without a score.
	Band string `json:"band,omitempty" enum:"cold,warm,hot,none"`
	// ScoreFrom and ScoreTo are a range of scores (0–100), both included.
	ScoreFrom string `json:"scoreFrom,omitempty" maxLength:"3"`
	ScoreTo   string `json:"scoreTo,omitempty" maxLength:"3"`
	// GrowthSource is a source key from Audience › Growth ("form:<id>",
	// "api_key:<id>", "integration:<id>:<workflow>", "person"…).
	GrowthSource    string `json:"growthSource,omitempty" maxLength:"200"`
	UTMSource       string `json:"utmSource,omitempty" maxLength:"200"`
	UTMMedium       string `json:"utmMedium,omitempty" maxLength:"200"`
	UTMCampaign     string `json:"utmCampaign,omitempty" maxLength:"200"`
	ReferringDomain string `json:"referringDomain,omitempty" maxLength:"255"`
	// Sort orders the table: by last activity (default) or score, highest first.
	Sort string `json:"sort,omitempty" enum:"activity,score"`
}

// ScoreGrowthQuery is the same filter as query parameters.
type ScoreGrowthQuery struct {
	Band            string `query:"band" enum:"cold,warm,hot,none,"`
	ScoreFrom       string `query:"scoreFrom" maxLength:"3"`
	ScoreTo         string `query:"scoreTo" maxLength:"3"`
	GrowthSource    string `query:"growthSource" maxLength:"200"`
	UTMSource       string `query:"utmSource" maxLength:"200"`
	UTMMedium       string `query:"utmMedium" maxLength:"200"`
	UTMCampaign     string `query:"utmCampaign" maxLength:"200"`
	ReferringDomain string `query:"referringDomain" maxLength:"255"`
	Sort            string `query:"sort" enum:"activity,score,"`
}

func (q ScoreGrowthQuery) filter() ScoreGrowthFilter {
	return ScoreGrowthFilter(q)
}

func scoreArg(s, loc string) (int, error) {
	n, err := strconv.Atoi(strings.TrimSpace(s))
	if err != nil || n < 0 || n > 100 {
		return 0, api.Invalid(loc, "Scores go from 0 to 100.")
	}
	return n, nil
}

// scoreGrowthFilter appends the score and growth conditions of f.
func (s *Service) scoreGrowthFilter(ctx context.Context, wsID uuid.UUID, out *filterSQL, f ScoreGrowthFilter) error {
	score := "coalesce((SELECT cs.score FROM contact_scores cs WHERE cs.contact_id = c.id), 0)"
	if f.Band != "" || f.ScoreFrom != "" || f.ScoreTo != "" {
		bands, ok, err := scoring.AppliedBands(ctx, s.Q, wsID)
		if err != nil {
			return err
		}
		if f.Band == "none" && f.ScoreFrom == "" && f.ScoreTo == "" {
			// Without scoring nobody has a score; with it, contacts added
			// since the last update don't have one yet.
			if ok {
				out.add("NOT EXISTS (SELECT 1 FROM contact_scores cs WHERE cs.contact_id = c.id)")
			}
			return s.growthFilter(out, f)
		}
		if !ok || f.Band == "none" {
			// No scoring: nobody has a score or a band.
			out.add("false")
			return nil
		}
		switch f.Band {
		case scoring.BandHot:
			out.add(score+" >= ?", bands.HotFrom)
		case scoring.BandWarm:
			out.add(score+" >= ? AND "+score+" < ?", bands.WarmFrom, bands.HotFrom)
		case scoring.BandCold:
			out.add(score+" < ?", bands.WarmFrom)
		}
		if f.ScoreFrom != "" {
			n, err := scoreArg(f.ScoreFrom, "scoreFrom")
			if err != nil {
				return err
			}
			out.add(score+" >= ?", n)
		}
		if f.ScoreTo != "" {
			n, err := scoreArg(f.ScoreTo, "scoreTo")
			if err != nil {
				return err
			}
			out.add(score+" <= ?", n)
		}
	}
	return s.growthFilter(out, f)
}

// growthFilter appends the source, UTM and referring-domain conditions of f.
func (s *Service) growthFilter(out *filterSQL, f ScoreGrowthFilter) error {
	growth := func(cond string, vals ...any) {
		out.add("EXISTS (SELECT 1 FROM contact_growth g WHERE g.contact_id = c.id AND "+cond+")", vals...)
	}
	if v := strings.TrimSpace(f.GrowthSource); v != "" {
		kind, id, _ := strings.Cut(v, ":")
		growth("g.source_kind = ? AND g.source_id = ?", kind, id)
	}
	for _, x := range []struct{ col, v string }{{"utm_source", f.UTMSource}, {"utm_medium", f.UTMMedium}, {"utm_campaign", f.UTMCampaign}} {
		if v := strings.TrimSpace(x.v); v != "" {
			growth("lower(g."+x.col+") = ?", strings.ToLower(v))
		}
	}
	if v := strings.ToLower(strings.TrimSpace(f.ReferringDomain)); v != "" {
		growth("(g.referrer_domain = ? OR g.referrer_domain LIKE ?)", v, "%."+escapeLike(v))
	}
	return nil
}

// contactOrder is the ORDER BY of the contacts table.
func contactOrder(f ScoreGrowthFilter) string {
	if f.Sort == "score" {
		return "cs.score DESC NULLS LAST, c.last_activity_at DESC NULLS LAST, c.id DESC"
	}
	return "c.last_activity_at DESC NULLS LAST, c.id DESC"
}
