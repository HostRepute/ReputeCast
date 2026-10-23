package deliverability

import (
	"bytes"
	"context"
	"encoding/csv"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/store"
)

// assetOrder ranks assets as the workspace lists them (sending domains,
// organization domains, IPs, hostnames).
func (st *state) assetOrder() map[string]int {
	out := map[string]int{}
	for i, c := range st.cands {
		out[strings.ToLower(c.Asset)] = i
	}
	return out
}

func sortAssets(list []string, order map[string]int) {
	sort.SliceStable(list, func(i, j int) bool {
		a, aok := order[list[i]]
		b, bok := order[list[j]]
		switch {
		case aok && bok:
			return a < b
		case aok != bok:
			return aok
		}
		return list[i] < list[j]
	})
}

// blocklistStatus groups the current source's results by blocklist.
func (st *state) blocklistStatus(builtin int) DeliverabilityBlocklistStatus {
	via := st.blocklistVia()
	out := DeliverabilityBlocklistStatus{Via: via, Sources: st.sourcesFor(via, builtin), Interval: "daily", Assets: []string{}, Results: []DeliverabilityBlocklistSource{}}
	if via == "hostrepute" {
		out.Interval = "hourly"
		out.Stale = st.hostReputeStale()
	}
	if via == "none" {
		return out
	}
	type group struct {
		res  DeliverabilityBlocklistSource
		rank int32
	}
	groups := map[string]*group{}
	var names []string
	assets := map[string]bool{}
	for _, b := range st.blocklists {
		if b.Via != via {
			continue
		}
		g, ok := groups[b.Source]
		if !ok {
			g = &group{res: DeliverabilityBlocklistSource{Source: b.Source, Impact: b.Impact, Assets: []string{}, ListedAssets: []string{}, DelistURL: b.DelistUrl}, rank: b.Rank}
			groups[b.Source] = g
			names = append(names, b.Source)
		}
		g.res.Assets = append(g.res.Assets, b.Asset)
		assets[b.Asset] = true
		if out.CheckedAt == nil || b.CheckedAt.After(*out.CheckedAt) {
			t := b.CheckedAt
			out.CheckedAt = &t
		}
		if b.Listed {
			g.res.Listed = true
			g.res.ListedAssets = append(g.res.ListedAssets, b.Asset)
			if b.ListedSince != nil && (g.res.Since == nil || b.ListedSince.Before(*g.res.Since)) {
				g.res.Since = b.ListedSince
			}
			if b.Detail != "" {
				g.res.Detail = b.Detail
			}
			if b.Delisting != "" {
				g.res.Delisting, g.res.DelistingAt = b.Delisting, b.DelistingRequestedAt
			}
			if b.DelistUrl != "" {
				g.res.DelistURL = b.DelistUrl
			}
		}
	}
	order := st.assetOrder()
	for a := range assets {
		out.Assets = append(out.Assets, a)
	}
	sortAssets(out.Assets, order)
	sort.SliceStable(names, func(i, j int) bool {
		a, b := groups[names[i]], groups[names[j]]
		if a.res.Listed != b.res.Listed {
			return a.res.Listed
		}
		if a.rank != b.rank {
			return a.rank < b.rank
		}
		return names[i] < names[j]
	})
	for _, n := range names {
		g := groups[n]
		sortAssets(g.res.Assets, order)
		sortAssets(g.res.ListedAssets, order)
		if g.res.Listed {
			out.Listed++
		} else {
			out.Clean++
			g.res.DelistURL = ""
		}
		out.Results = append(out.Results, g.res)
	}
	return out
}

// Export writes the findings as CSV: one row per check, for spreadsheets
// and tickets. Its rows are data: they stay English in every language.
func (s *Service) Export(ctx context.Context, ws *store.Workspace) ([]byte, error) {
	o, err := s.Overview(i18n.WithLang(ctx, i18n.Default), ws)
	if err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	w := csv.NewWriter(&buf)
	_ = w.Write([]string{"section", "asset", "check", "status", "detail", "since", "checked_at"})
	ts := func(t *time.Time) string {
		if t == nil {
			return ""
		}
		return t.UTC().Format(time.RFC3339)
	}
	pct := func(v *float64) string {
		if v == nil {
			return ""
		}
		return fmt.Sprintf("%.2f%%", *v)
	}
	for _, a := range o.Assets {
		if a.Auth != nil {
			_ = w.Write([]string{"authentication", a.Asset, "SPF · DKIM · DMARC", a.Auth.SPF + " · " + a.Auth.DKIM + " · " + a.Auth.DMARC, "return path " + a.Auth.ReturnPath + ", BIMI " + a.Auth.BIMI, "", ts(a.Auth.CheckedAt)})
		}
		if a.ReverseDNS != nil {
			_ = w.Write([]string{"reverse_dns", a.Asset, "PTR", a.ReverseDNS.Status, a.ReverseDNS.PTR, "", ""})
		}
		if a.DMARCAligned != nil {
			_ = w.Write([]string{"dmarc", a.Asset, "alignment, 7 days", pct(a.DMARCAligned), "", "", ""})
		}
	}
	for _, r := range o.Blocklists.Results {
		status := "clean"
		if r.Listed {
			status = "listed"
		}
		_ = w.Write([]string{"blocklist", strings.Join(r.Assets, " "), r.Source, status, strings.TrimSpace(r.Impact + " impact " + r.Detail + " " + r.DelistURL), ts(r.Since), ts(o.Blocklists.CheckedAt)})
	}
	for _, p := range o.Providers {
		_ = w.Write([]string{"provider", p.Name, p.Bounce.Label, pct(p.Bounce.Value), p.Bounce.Thresholds, "", ""})
		_ = w.Write([]string{"provider", p.Name, p.Complaint.Label, pct(p.Complaint.Value), p.Complaint.Thresholds, "", ""})
	}
	if pm := o.Postmaster; pm != nil {
		_ = w.Write([]string{"postmaster", pm.Domain, "domain reputation", pm.DomainReputation, "", "", ""})
		_ = w.Write([]string{"postmaster", pm.Domain, "spam rate, 30 days", pct(pm.SpamRate), "", "", ""})
	}
	if sn := o.SNDS; sn != nil {
		_ = w.Write([]string{"snds", sn.IP, "filter result", sn.FilterResult, fmt.Sprintf("%d messages, %d trap hits in 7 days", sn.Messages7d, sn.TrapHits), "", ""})
	}
	for _, src := range o.DMARC.Sources {
		_ = w.Write([]string{"dmarc", src.Label, "messages, 7 days", src.Result, fmt.Sprintf("%d messages, SPF %s, DKIM %s", src.Messages, src.SPF, src.DKIM), "", ""})
	}
	for _, it := range o.Incidents.Items {
		_ = w.Write([]string{"incident", "", it.Title, it.Severity, it.Detail, it.OccurredAt.UTC().Format(time.RFC3339), ""})
	}
	w.Flush()
	return buf.Bytes(), w.Error()
}
