package demo

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"html"
	"math"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/authn"
)

func init() { seedReports = seedReportsImpl }

// The reports seeder gives every campaign sent from ReputeCast a complete,
// consistent report: it keeps the headline counters seeded with the
// campaigns and adds what the report shows beyond them — results that vary
// by mailbox and sending provider (adding up to the same totals), 72 hours
// of hourly engagement, the email's links and their clicks, bounce events
// with each provider's original code, unsubscribe reasons, the median time
// to delivery per route and, for the October Product Update, the A/B test.
// The October Product Update reproduces the Report board.

// Counter metrics of one cell, in this order.
const (
	rmSent = iota
	rmDelivered
	rmBounced
	rmComplained
	rmHuman
	rmMachine
	rmClicked
	rmUnsub
	rmCount
)

// reportCell is one row of campaign_counters: sending provider × mailbox
// provider × A/B variant.
type reportCell struct {
	provider uuid.UUID
	adapter  string
	mailbox  string
	variant  string
	m        [rmCount]int64
	hard     int64
	soft     int64
	// classes are the bounce taxonomy classes of the cell's bounces.
	classes map[string]int64
}

type reportCampaignSeed struct {
	id                 uuid.UUID
	name               string
	started, completed time.Time
	html               string
	cells              []*reportCell
}

var reportMailboxes = []string{"gmail", "outlook", "yahoo", "icloud", "other"}

func seedReportsImpl(ctx context.Context, s *Seeder) error {
	ws := s.WS()
	adapters := map[uuid.UUID]string{}
	rows, err := s.Tx.Query(ctx, `SELECT id, adapter FROM providers WHERE workspace_id = $1`, ws.ID)
	if err != nil {
		return err
	}
	for rows.Next() {
		var id uuid.UUID
		var a string
		if err := rows.Scan(&id, &a); err != nil {
			rows.Close()
			return err
		}
		adapters[id] = a
	}
	rows.Close()
	var list []*reportCampaignSeed
	rows, err = s.Tx.Query(ctx, `SELECT id, name, started_at, completed_at, html FROM campaigns
		WHERE workspace_id = $1 AND status = 'sent' AND origin = '' AND started_at IS NOT NULL AND completed_at IS NOT NULL AND deleted_at IS NULL
		ORDER BY started_at DESC`, ws.ID)
	if err != nil {
		return err
	}
	for rows.Next() {
		c := &reportCampaignSeed{}
		if err := rows.Scan(&c.id, &c.name, &c.started, &c.completed, &c.html); err != nil {
			rows.Close()
			return err
		}
		list = append(list, c)
	}
	rows.Close()
	for i, c := range list {
		if err := reportLoadCells(ctx, s, c, adapters); err != nil {
			return err
		}
		if len(c.cells) == 0 {
			continue
		}
		oct := c.name == "October Product Update"
		if oct {
			if err := reportShapeOctober(c); err != nil {
				return fmt.Errorf("%s: %w", c.name, err)
			}
			if err := reportOctoberABTest(ctx, s, c); err != nil {
				return err
			}
			if err := reportShareLink(ctx, s, c); err != nil {
				return err
			}
		} else {
			reportShapeGeneric(c)
		}
		if err := reportWriteCounters(ctx, s, c); err != nil {
			return err
		}
		if err := reportBounceEvents(ctx, s, c, i); err != nil {
			return err
		}
		if err := reportHourly(ctx, s, c, oct); err != nil {
			return err
		}
		if err := reportLinks(ctx, s, c, oct); err != nil {
			return err
		}
		if err := reportUnsubscribeFeedback(ctx, s, c, oct); err != nil {
			return err
		}
		if err := reportRouteTimes(ctx, s, c, i); err != nil {
			return err
		}
	}
	return nil
}

func reportLoadCells(ctx context.Context, s *Seeder, c *reportCampaignSeed, adapters map[uuid.UUID]string) error {
	rows, err := s.Tx.Query(ctx, `SELECT provider_id, mailbox_provider, variant, sent, delivered, hard_bounced + soft_bounced, complained,
		opened_human, opened_machine, clicked, unsubscribed FROM campaign_counters WHERE campaign_id = $1 ORDER BY provider_id, mailbox_provider, variant`, c.id)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		cell := &reportCell{}
		if err := rows.Scan(&cell.provider, &cell.mailbox, &cell.variant, &cell.m[rmSent], &cell.m[rmDelivered], &cell.m[rmBounced], &cell.m[rmComplained],
			&cell.m[rmHuman], &cell.m[rmMachine], &cell.m[rmClicked], &cell.m[rmUnsub]); err != nil {
			return err
		}
		cell.adapter = adapters[cell.provider]
		c.cells = append(c.cells, cell)
	}
	return rows.Err()
}

func (c *reportCampaignSeed) total(k int) int64 {
	var t int64
	for _, cell := range c.cells {
		t += cell.m[k]
	}
	return t
}

// Relative results by mailbox provider and sending provider, from the Report
// board's October Product Update (each metric's average is 1).
var reportMailboxFactor = map[int]map[string]float64{
	rmDelivered:  {"gmail": 1.003, "outlook": 0.995, "yahoo": 0.999, "icloud": 1.004, "other": 0.997},
	rmBounced:    {"gmail": 0.67, "outlook": 1.67, "yahoo": 1.17, "icloud": 0.5, "other": 1.33},
	rmComplained: {"gmail": 0.75, "outlook": 1.5, "yahoo": 1.25, "icloud": 0.1, "other": 1.0},
	rmHuman:      {"gmail": 1.083, "outlook": 0.876, "yahoo": 0.966, "icloud": 0.757, "other": 0.983},
	rmMachine:    {"gmail": 0.495, "outlook": 1.15, "yahoo": 0.683, "icloud": 3.17, "other": 0.806},
	rmClicked:    {"gmail": 1.11, "outlook": 0.84, "yahoo": 0.94, "icloud": 0.81, "other": 1.02},
	rmUnsub:      {"gmail": 0.89, "outlook": 1.27, "yahoo": 1.1, "icloud": 0.76, "other": 1.02},
}

var reportProviderFactor = map[int]map[string]float64{
	rmDelivered:  {"postal": 0.994},
	rmBounced:    {"postal": 1.6},
	rmComplained: {"postal": 1.5},
	rmHuman:      {"postal": 0.957},
	rmClicked:    {"postal": 0.91},
}

func reportFactor(k int, cell *reportCell) float64 {
	f := 1.0
	if m, ok := reportMailboxFactor[k]; ok {
		if v, ok := m[cell.mailbox]; ok {
			f *= v
		}
	}
	if p, ok := reportProviderFactor[k]; ok {
		if v, ok := p[cell.adapter]; ok {
			f *= v
		}
	}
	return f
}

// reportAllocate splits total by weights, never giving a cell more than its
// cap; the overflow goes to cells that still have room.
func reportAllocate(total int64, weights []float64, caps []int64) []int64 {
	out := make([]int64, len(weights))
	left := total
	open := make([]bool, len(weights))
	for i := range open {
		open[i] = caps[i] > 0
	}
	for left > 0 {
		w := make([]float64, len(weights))
		any := false
		for i := range w {
			if open[i] {
				w[i] = math.Max(weights[i], 1e-9)
				any = true
			}
		}
		if !any {
			break
		}
		part := Split(left, w)
		left = 0
		for i, v := range part {
			if room := caps[i] - out[i]; v > room {
				left += v - room
				v = room
				open[i] = false
			}
			out[i] += v
		}
	}
	return out
}

// reportShapeGeneric keeps each cell's sent count and every total, and lets
// results vary by mailbox and sending provider the way they do in practice.
func reportShapeGeneric(c *reportCampaignSeed) {
	totals := [rmCount]int64{}
	for k := 1; k < rmCount; k++ {
		totals[k] = c.total(k)
	}
	n := len(c.cells)
	reshape := func(k int, base func(*reportCell) int64, capOf func(*reportCell) int64) {
		w := make([]float64, n)
		caps := make([]int64, n)
		for i, cell := range c.cells {
			w[i] = float64(base(cell)) * reportFactor(k, cell)
			caps[i] = capOf(cell)
		}
		for i, v := range reportAllocate(totals[k], w, caps) {
			c.cells[i].m[k] = v
		}
	}
	sent := func(cell *reportCell) int64 { return cell.m[rmSent] }
	delivered := func(cell *reportCell) int64 { return cell.m[rmDelivered] }
	reshape(rmDelivered, sent, sent)
	reshape(rmBounced, sent, func(cell *reportCell) int64 { return cell.m[rmSent] - cell.m[rmDelivered] })
	for _, k := range []int{rmComplained, rmHuman, rmMachine, rmClicked, rmUnsub} {
		reshape(k, delivered, delivered)
	}
	mix := reportClassMix
	for _, cell := range c.cells {
		cell.classes = map[string]int64{}
		w := make([]float64, len(mix))
		for i, cm := range mix {
			w[i] = cm.share
		}
		for i, v := range Split(cell.m[rmBounced], w) {
			cell.classes[mix[i].class] += v
		}
		cell.splitHardSoft()
	}
}

// reportClassMix is the share of each bounce class (Report board).
var reportClassMix = []struct {
	class string
	share float64
}{
	{"mailbox_unknown", 171}, {"domain_invalid", 22}, {"mailbox_full", 41}, {"temporary", 38}, {"policy_block", 11}, {"other", 5},
}

// hardClasses bounce permanently; the others are soft.
var hardClasses = map[string]bool{"mailbox_unknown": true, "domain_invalid": true, "policy_block": true, "spam_block": true, "reputation": true}

func (cell *reportCell) splitHardSoft() {
	cell.hard, cell.soft = 0, 0
	for cls, n := range cell.classes {
		if hardClasses[cls] {
			cell.hard += n
		} else {
			cell.soft += n
		}
	}
}

// The October Product Update as the Report board shows it: rates per
// mailbox provider and per sending provider, bounce types, and the subject
// A/B test (10% sample each, the winner to the remaining 80% at 12:00).
var (
	octMailboxSent = map[string]int64{"gmail": 22071, "outlook": 10076, "yahoo": 4318, "icloud": 3838, "other": 7677}
	octTotals      = map[int]int64{rmDelivered: 47548, rmBounced: 288, rmComplained: 19, rmHuman: 19590, rmMachine: 8844, rmClicked: 2996, rmUnsub: 112}
	// Per mailbox provider, in reportMailboxes order (percent; decimals).
	octMailboxRates = map[int][]float64{
		rmDelivered:  {99.4, 98.6, 99.0, 99.5, 98.8},
		rmHuman:      {44.6, 36.1, 39.8, 31.2, 40.5},
		rmMachine:    {9.2, 21.4, 12.7, 58.9, 15.0},
		rmClicked:    {7.0, 5.3, 5.9, 5.1, 6.4},
		rmBounced:    {0.4, 1.0, 0.7, 0.3, 0.8},
		rmComplained: {0.03, 0.06, 0.05, 0.00, 0.04},
		rmUnsub:      {0.21, 0.30, 0.26, 0.18, 0.24},
	}
	octDigits = map[int]int{rmDelivered: 1, rmHuman: 1, rmMachine: 1, rmClicked: 1, rmBounced: 1, rmComplained: 2, rmUnsub: 2}
	// Per sending provider (SES, Postal).
	octProviderRates = map[int][]float64{
		rmDelivered:  {99.2, 98.6},
		rmHuman:      {41.4, 39.6},
		rmClicked:    {6.4, 5.8},
		rmBounced:    {0.5, 0.9},
		rmComplained: {0.04, 0.06},
	}
	// Bounce types per sending provider (SES, Postal); they add up to the
	// board's 171 / 22 / 41 / 38 / 11 / 5.
	octClasses = []struct {
		class       string
		ses, postal int64
	}{
		{"mailbox_unknown", 145, 26}, {"domain_invalid", 18, 4}, {"mailbox_full", 41, 0}, {"temporary", 30, 8}, {"policy_block", 4, 7}, {"other", 5, 0},
	}
	// A/B samples: 4,798 recipients each (SES 4,318 + Postal 480).
	octSample = map[string]int64{"ses": 4318, "postal": 480}
	// Variant results of the samples (percent of delivered).
	octVariantRates = map[int]map[string]float64{
		rmHuman:   {"A": 39.8, "B": 42.6},
		rmClicked: {"A": 5.9, "B": 6.4},
	}
	octVariantComplaints = map[string]int64{"A": 3, "B": 2}
)

func reportShapeOctober(c *reportCampaignSeed) error {
	// Collapse to sending provider × mailbox provider first.
	type key struct {
		p uuid.UUID
		m string
	}
	byKey := map[key]*reportCell{}
	var provs []uuid.UUID
	provAdapter := map[uuid.UUID]string{}
	for _, cell := range c.cells {
		k := key{cell.provider, cell.mailbox}
		if byKey[k] == nil {
			byKey[k] = &reportCell{provider: cell.provider, adapter: cell.adapter, mailbox: cell.mailbox}
			if _, ok := provAdapter[cell.provider]; !ok {
				provs = append(provs, cell.provider)
				provAdapter[cell.provider] = cell.adapter
			}
		}
		byKey[k].m[rmSent] += cell.m[rmSent]
	}
	// SES first, then Postal.
	sort.SliceStable(provs, func(i, j int) bool { return provAdapter[provs[i]] == "ses" && provAdapter[provs[j]] != "ses" })
	if len(provs) != 2 || provAdapter[provs[0]] != "ses" || provAdapter[provs[1]] != "postal" {
		return fmt.Errorf("expected the SES and Postal split, got %v", provAdapter)
	}
	np, nm := len(provs), len(reportMailboxes)
	grid := make([][]*reportCell, nm)
	for i, mb := range reportMailboxes {
		grid[i] = make([]*reportCell, np)
		for j, p := range provs {
			cell := byKey[key{p, mb}]
			if cell == nil {
				cell = &reportCell{provider: p, adapter: provAdapter[p], mailbox: mb}
			}
			grid[i][j] = cell
		}
	}
	// Sent per mailbox provider as on the board (the provider split rounds
	// Outlook and iCloud one apart); provider totals stay as sent.
	provSent := make([]int64, np)
	for j := range provs {
		for i := range reportMailboxes {
			provSent[j] += grid[i][j].m[rmSent]
		}
	}
	rowSent := make([]int64, nm)
	init := make([][]float64, nm)
	for i, mb := range reportMailboxes {
		rowSent[i] = octMailboxSent[mb]
		init[i] = make([]float64, np)
		for j := range provs {
			init[i][j] = float64(grid[i][j].m[rmSent])
		}
	}
	for i, row := range reportFit2D(rowSent, provSent, init) {
		for j, v := range row {
			grid[i][j].m[rmSent] = v
		}
	}
	// Each metric: margins per mailbox and per provider from the board's
	// rates, then the table between them.
	fit := func(k int, base func(*reportCell) int64) {
		rowBase := make([]int64, nm)
		colBase := make([]int64, np)
		for i := range reportMailboxes {
			for j := range provs {
				b := base(grid[i][j])
				rowBase[i] += b
				colBase[j] += b
			}
		}
		total := octTotals[k]
		rows := reportMargins(octMailboxRates[k], rowBase, total, octDigits[k])
		var cols []int64
		if pr, ok := octProviderRates[k]; ok {
			cols = reportMargins(pr, colBase, total, octDigits[k])
		} else {
			// No provider rates on the board: what the mailbox mix implies.
			exp := make([]float64, np)
			for i := range reportMailboxes {
				for j := range provs {
					if rowBase[i] > 0 {
						exp[j] += float64(rows[i]) * float64(base(grid[i][j])) / float64(rowBase[i])
					}
				}
			}
			cols = Split(total, exp)
		}
		in := make([][]float64, nm)
		for i := range reportMailboxes {
			in[i] = make([]float64, np)
			for j := range provs {
				if rowBase[i] > 0 {
					in[i][j] = float64(rows[i]) * float64(base(grid[i][j])) / float64(rowBase[i])
				}
			}
		}
		for i, row := range reportFit2D(rows, cols, in) {
			for j, v := range row {
				grid[i][j].m[k] = v
			}
		}
	}
	sent := func(cell *reportCell) int64 { return cell.m[rmSent] }
	delivered := func(cell *reportCell) int64 { return cell.m[rmDelivered] }
	fit(rmDelivered, sent)
	fit(rmBounced, sent)
	for _, k := range []int{rmHuman, rmMachine, rmClicked, rmComplained, rmUnsub} {
		fit(k, delivered)
	}
	// Bounce types: per provider, the board's type totals spread over the
	// mailbox providers' bounces.
	for j := range provs {
		rows := make([]int64, nm)
		var colsT []int64
		for i := range reportMailboxes {
			rows[i] = grid[i][j].m[rmBounced]
		}
		in := make([][]float64, nm)
		for _, oc := range octClasses {
			if j == 0 {
				colsT = append(colsT, oc.ses)
			} else {
				colsT = append(colsT, oc.postal)
			}
		}
		var sumRows, sumCols int64
		for _, v := range rows {
			sumRows += v
		}
		for _, v := range colsT {
			sumCols += v
		}
		if sumRows != sumCols {
			return fmt.Errorf("bounces of %s: %d by mailbox, %d by type", provAdapter[provs[j]], sumRows, sumCols)
		}
		for i := range reportMailboxes {
			in[i] = make([]float64, len(octClasses))
			for t := range octClasses {
				if sumCols > 0 {
					in[i][t] = float64(rows[i]) * float64(colsT[t]) / float64(sumCols)
				}
			}
		}
		for i, row := range reportFit2D(rows, colsT, in) {
			grid[i][j].classes = map[string]int64{}
			for t, v := range row {
				if v > 0 {
					grid[i][j].classes[octClasses[t].class] = v
				}
			}
			grid[i][j].splitHardSoft()
		}
	}
	// Variants: A and B samples, the rest got the winner ('').
	variants := []string{"A", "B", ""}
	flat := make([]*reportCell, 0, nm*np)
	for i := range reportMailboxes {
		flat = append(flat, grid[i]...)
	}
	out := make([]*reportCell, 0, len(flat)*len(variants))
	split := make([][]*reportCell, len(flat))
	for r, cell := range flat {
		split[r] = make([]*reportCell, len(variants))
		for v, name := range variants {
			split[r][v] = &reportCell{provider: cell.provider, adapter: cell.adapter, mailbox: cell.mailbox, variant: name, classes: map[string]int64{}}
		}
	}
	// Sent per variant and provider.
	for j, p := range provs {
		var rows []int64
		var idx []int
		for r, cell := range flat {
			if cell.provider == p {
				rows = append(rows, cell.m[rmSent])
				idx = append(idx, r)
			}
		}
		sample := octSample[provAdapter[p]]
		cols := []int64{sample, sample, provSent[j] - 2*sample}
		in := make([][]float64, len(rows))
		for a, rv := range rows {
			in[a] = make([]float64, 3)
			for v := range cols {
				in[a][v] = float64(rv) * float64(cols[v]) / float64(provSent[j])
			}
		}
		for a, row := range reportFit2D(rows, cols, in) {
			for v, val := range row {
				split[idx[a]][v].m[rmSent] = val
			}
		}
	}
	// Other metrics: each cell split across the variants, with the board's
	// sample results for opens, clicks and complaints.
	variantTotals := func(k int) []int64 {
		var vsent, vdel [3]int64
		for r := range flat {
			for v := range variants {
				vsent[v] += split[r][v].m[rmSent]
				vdel[v] += split[r][v].m[rmDelivered]
			}
		}
		total := c.totalOf(flat, k)
		switch k {
		case rmHuman, rmClicked:
			a := int64(math.Round(octVariantRates[k]["A"] / 100 * float64(vdel[0])))
			b := int64(math.Round(octVariantRates[k]["B"] / 100 * float64(vdel[1])))
			return []int64{a, b, total - a - b}
		case rmComplained:
			a, b := octVariantComplaints["A"], octVariantComplaints["B"]
			return []int64{a, b, total - a - b}
		case rmDelivered, rmBounced:
			return Split(total, []float64{float64(vsent[0]), float64(vsent[1]), float64(vsent[2])})
		}
		return Split(total, []float64{float64(vdel[0]), float64(vdel[1]), float64(vdel[2])})
	}
	splitMetric := func(k int, value func(*reportCell) int64, set func(*reportCell, int64), weight func(r, v int) float64) {
		rows := make([]int64, len(flat))
		for r, cell := range flat {
			rows[r] = value(cell)
		}
		cols := variantTotals(k)
		in := make([][]float64, len(flat))
		for r := range flat {
			in[r] = make([]float64, 3)
			var wsum float64
			for v := range variants {
				wsum += weight(r, v)
			}
			for v := range variants {
				if wsum > 0 {
					in[r][v] = float64(rows[r]) * weight(r, v) / wsum
				}
			}
		}
		for r, row := range reportFit2D(rows, cols, in) {
			for v, val := range row {
				set(split[r][v], val)
			}
		}
	}
	bySent := func(r, v int) float64 { return float64(split[r][v].m[rmSent]) }
	byDelivered := func(r, v int) float64 { return float64(split[r][v].m[rmDelivered]) }
	splitMetric(rmDelivered, func(c *reportCell) int64 { return c.m[rmDelivered] }, func(c *reportCell, x int64) { c.m[rmDelivered] = x }, bySent)
	splitMetric(rmBounced, func(c *reportCell) int64 { return c.m[rmBounced] }, func(c *reportCell, x int64) { c.m[rmBounced] = x }, bySent)
	for _, k := range []int{rmHuman, rmMachine, rmClicked, rmComplained, rmUnsub} {
		k := k
		splitMetric(k, func(c *reportCell) int64 { return c.m[k] }, func(c *reportCell, x int64) { c.m[k] = x }, byDelivered)
	}
	// Bounce types follow each variant's share of the cell's bounces.
	for r, cell := range flat {
		for _, oc := range octClasses {
			n := cell.classes[oc.class]
			if n == 0 {
				continue
			}
			w := make([]float64, 3)
			for v := range variants {
				w[v] = float64(split[r][v].m[rmBounced])
			}
			parts := reportAllocate(n, w, []int64{split[r][0].m[rmBounced], split[r][1].m[rmBounced], split[r][2].m[rmBounced]})
			for v, x := range parts {
				if x > 0 {
					split[r][v].classes[oc.class] += x
				}
			}
		}
		for v := range variants {
			// Keep the cell's bounce count equal to its typed bounces.
			var typed int64
			for _, x := range split[r][v].classes {
				typed += x
			}
			split[r][v].m[rmBounced] = typed
			split[r][v].splitHardSoft()
			out = append(out, split[r][v])
		}
	}
	c.cells = out
	return nil
}

func (c *reportCampaignSeed) totalOf(cells []*reportCell, k int) int64 {
	var t int64
	for _, cell := range cells {
		t += cell.m[k]
	}
	return t
}

func reportWriteCounters(ctx context.Context, s *Seeder, c *reportCampaignSeed) error {
	if _, err := s.Tx.Exec(ctx, `DELETE FROM campaign_counters WHERE campaign_id = $1`, c.id); err != nil {
		return err
	}
	rows := make([][]any, 0, len(c.cells))
	for _, cell := range c.cells {
		if cell.m[rmSent] == 0 && cell.m[rmDelivered] == 0 {
			continue
		}
		rows = append(rows, []any{c.id, cell.provider, cell.mailbox, cell.variant, cell.m[rmSent], cell.m[rmDelivered], cell.hard, cell.soft,
			cell.m[rmComplained], cell.m[rmHuman], cell.m[rmMachine], cell.m[rmClicked], cell.m[rmUnsub]})
	}
	_, err := s.Tx.CopyFrom(ctx, pgx.Identifier{"campaign_counters"}, []string{"campaign_id", "provider_id", "mailbox_provider", "variant", "sent", "delivered",
		"hard_bounced", "soft_bounced", "complained", "opened_human", "opened_machine", "clicked", "unsubscribed"}, pgx.CopyFromRows(rows))
	return err
}

// reportBounceCode is what a provider reports for a bounce class.
type reportBounceCode struct{ typ, code, diag string }

var reportBounceCodes = map[string]map[string]reportBounceCode{
	"ses": {
		"mailbox_unknown": {"Permanent/General", "550 5.1.1", "smtp; 550 5.1.1 The email account that you tried to reach does not exist"},
		"domain_invalid":  {"Permanent/NoEmail", "", "Amazon SES couldn't find a mail server for the recipient's domain"},
		"mailbox_full":    {"Transient/MailboxFull", "452 4.2.2", "smtp; 452 4.2.2 The recipient's mailbox is full"},
		"temporary":       {"DeliveryDelay/TransientCommunicationFailure", "", "Delivery attempts stopped after 14 hours: the receiving server kept timing out"},
		"policy_block":    {"Permanent/Suppressed", "", "The address is on the Amazon SES account-level suppression list"},
		"other":           {"Undetermined", "", "Amazon SES couldn't determine why the message bounced"},
	},
	"postal": {
		"mailbox_unknown": {"HardFail", "550 5.1.1", "Recipient address rejected: User unknown in virtual mailbox table"},
		"domain_invalid":  {"HardFail", "5.1.2", "Host or domain name not found. Name service error for the recipient's domain"},
		"mailbox_full":    {"SoftFail", "452 4.2.2", "Mailbox full"},
		"temporary":       {"", "421 4.7.0 TSS04", "Messages from 203.0.113.25 temporarily deferred due to user complaints (TSS04); delivery attempts expired"},
		"policy_block":    {"", "550 5.7.1 (Outlook S3150)", "Unfortunately, messages from [203.0.113.25] weren't sent. Please contact your Internet service provider (S3150)"},
		"other":           {"SoftFail", "", "Delivery failed for an unknown reason"},
	},
}

var reportMailboxDomain = map[string]string{"gmail": "gmail.com", "outlook": "outlook.com", "yahoo": "yahoo.com", "icloud": "icloud.com", "other": "example.org"}

// reportBounceEvents writes one delivery event per bounce, with the
// provider's original code. Addresses of hard bounces (unknown user,
// domain not found) were suppressed when they bounced, so they come from
// the shared suppression list; the others aren't suppressed. Events other
// seeders already wrote for the campaign count towards the totals.
func reportBounceEvents(ctx context.Context, s *Seeder, c *reportCampaignSeed, seq int) error {
	ws := s.WS()
	existing := map[string]int64{}
	rows, err := s.Tx.Query(ctx, `SELECT coalesce(provider_id::text, ''), mailbox_provider, bounce_class, count(*) FROM delivery_events
		WHERE campaign_id = $1 AND type IN ('hard_bounce', 'soft_bounce') GROUP BY 1, 2, 3`, c.id)
	if err != nil {
		return err
	}
	for rows.Next() {
		var p, mb, cls string
		var n int64
		if err := rows.Scan(&p, &mb, &cls, &n); err != nil {
			rows.Close()
			return err
		}
		existing[p+"|"+mb+"|"+cls] += n
	}
	rows.Close()
	// Suppressed hard-bounce addresses by provider and mailbox domain.
	pool := map[string][]string{}
	rows, err = s.Tx.Query(ctx, `SELECT email, coalesce(source_provider_id::text, '') FROM suppressions
		WHERE workspace_id = $1 AND reason = 'hard_bounce' ORDER BY created_at DESC, email`, ws.ID)
	if err != nil {
		return err
	}
	for rows.Next() {
		var email, p string
		if err := rows.Scan(&email, &p); err != nil {
			rows.Close()
			return err
		}
		mb := "other"
		if at := strings.LastIndex(email, "@"); at > 0 {
			for k, d := range reportMailboxDomain {
				if strings.HasSuffix(email, "@"+d) {
					mb = k
				}
			}
		}
		pool[p+"|"+mb] = append(pool[p+"|"+mb], email)
		pool["|"+mb] = append(pool["|"+mb], email)
	}
	rows.Close()
	used := map[string]bool{}
	take := func(provider uuid.UUID, mb string) string {
		for _, k := range []string{provider.String() + "|" + mb, "|" + mb} {
			for len(pool[k]) > 0 {
				e := pool[k][0]
				pool[k] = pool[k][1:]
				if !used[e] {
					used[e] = true
					return e
				}
			}
		}
		return ""
	}
	type ev struct {
		provider uuid.UUID
		adapter  string
		mailbox  string
		class    string
	}
	var events []ev
	// Group the cells' typed bounces by provider × mailbox (variants share
	// one event stream).
	type pm struct {
		p  uuid.UUID
		mb string
	}
	agg := map[pm]map[string]int64{}
	adapterOf := map[uuid.UUID]string{}
	var keys []pm
	for _, cell := range c.cells {
		k := pm{cell.provider, cell.mailbox}
		if agg[k] == nil {
			agg[k] = map[string]int64{}
			keys = append(keys, k)
			adapterOf[cell.provider] = cell.adapter
		}
		for cls, n := range cell.classes {
			agg[k][cls] += n
		}
	}
	for _, k := range keys {
		classes := make([]string, 0, len(agg[k]))
		for cls := range agg[k] {
			classes = append(classes, cls)
		}
		sort.Strings(classes)
		for _, cls := range classes {
			n := agg[k][cls] - existing[k.p.String()+"|"+k.mb+"|"+cls]
			for i := int64(0); i < n; i++ {
				events = append(events, ev{k.p, adapterOf[k.p], k.mb, cls})
			}
		}
	}
	short := strings.ToLower(nonAlnum.ReplaceAllString(c.name, ""))
	if len(short) > 10 {
		short = short[:10]
	}
	window := c.completed.Sub(c.started)
	out := make([][]any, 0, len(events))
	var newSupp [][]any
	for i, e := range events {
		codes := reportBounceCodes[e.adapter]
		if codes == nil {
			codes = reportBounceCodes["ses"]
		}
		code := codes[e.class]
		typ := "soft_bounce"
		if hardClasses[e.class] {
			typ = "hard_bounce"
		}
		at := c.started.Add(time.Duration(float64(window) * float64(i%97) / 97))
		if e.class == "temporary" {
			at = at.Add(14 * time.Hour)
		}
		recipient := ""
		if e.class == "mailbox_unknown" || e.class == "domain_invalid" {
			recipient = take(e.provider, e.mailbox)
			if recipient == "" {
				recipient = fmt.Sprintf("gone.%s%d.%04d@%s", short, seq, i, reportMailboxDomain[e.mailbox])
				newSupp = append(newSupp, []any{ID(), ws.ID, recipient, "hard_bounce", "webhook", e.provider, code.code, at})
			}
		} else {
			recipient = fmt.Sprintf("reader.%s%d.%04d@%s", short, seq, i, reportMailboxDomain[e.mailbox])
		}
		dedupe := fmt.Sprintf("seed-report-%s-%d", c.id, i)
		out = append(out, []any{ID(), ws.ID, c.id, e.provider, typ, at, e.mailbox, code.typ, code.code, code.diag, e.class, recipient, at.Add(4 * time.Second), dedupe, c.name})
	}
	if len(newSupp) > 0 {
		if _, err := s.Tx.CopyFrom(ctx, pgx.Identifier{"suppressions"}, []string{"id", "workspace_id", "email", "reason", "source", "source_provider_id", "original_code", "created_at"},
			pgx.CopyFromRows(newSupp)); err != nil {
			return err
		}
	}
	_, err = s.Tx.CopyFrom(ctx, pgx.Identifier{"delivery_events"}, []string{"id", "workspace_id", "campaign_id", "provider_id", "type", "occurred_at", "mailbox_provider",
		"original_type", "original_code", "diagnostic", "bounce_class", "recipient", "received_at", "dedupe_key", "source"}, pgx.CopyFromRows(out))
	return err
}

var nonAlnum = regexp.MustCompile(`[^A-Za-z0-9]+`)

// reportCurve is the cumulative share of a metric's 72-hour total that has
// come in by hour h: a fast and a slow wave.
func reportCurve(h, fast, tauFast, tauSlow float64) float64 {
	if h <= 0 {
		return 0
	}
	f := func(x float64) float64 { return fast*(1-math.Exp(-x/tauFast)) + (1-fast)*(1-math.Exp(-x/tauSlow)) }
	return f(h) / f(72)
}

// reportHourly spreads opens and clicks over the first 72 hours. With an
// A/B test, the samples (20%) start at the send and everyone else when the
// winner goes out two hours later: the second rise of the chart.
func reportHourly(ctx context.Context, s *Seeder, c *reportCampaignSeed, ab bool) error {
	type curve struct{ fast, tauFast, tauSlow float64 }
	curves := map[int]curve{rmHuman: {0.578, 1.7, 20}, rmMachine: {0.78, 1.0, 14}, rmClicked: {0.72, 1.2, 12}}
	cum := func(k int, h float64) float64 {
		cv := curves[k]
		if !ab {
			return reportCurve(h, cv.fast, cv.tauFast, cv.tauSlow)
		}
		return 0.2*reportCurve(h, cv.fast, cv.tauFast, cv.tauSlow) + 0.8*reportCurve(h-2, cv.fast, cv.tauFast, cv.tauSlow)/reportCurve(70, cv.fast, cv.tauFast, cv.tauSlow)
	}
	counts := map[int][]int64{}
	for _, k := range []int{rmHuman, rmMachine, rmClicked} {
		w := make([]float64, 72)
		for h := 0; h < 72; h++ {
			w[h] = math.Max(cum(k, float64(h+1))-cum(k, float64(h)), 0)
		}
		counts[k] = Split(c.total(k), w)
	}
	rows := make([][]any, 0, 72)
	for h := 0; h < 72; h++ {
		rows = append(rows, []any{c.id, int32(h), counts[rmHuman][h], counts[rmMachine][h], counts[rmClicked][h]})
	}
	if _, err := s.Tx.Exec(ctx, `DELETE FROM campaign_hourly WHERE campaign_id = $1`, c.id); err != nil {
		return err
	}
	_, err := s.Tx.CopyFrom(ctx, pgx.Identifier{"campaign_hourly"}, []string{"campaign_id", "hour", "opened_human", "opened_machine", "clicked"}, pgx.CopyFromRows(rows))
	return err
}

var (
	anchorRe = regexp.MustCompile(`(?is)<a\s[^>]*?href\s*=\s*["']([^"']+)["'][^>]*>(.*?)</a>`)
	tagRe    = regexp.MustCompile(`(?s)<[^>]+>`)
	attrRe   = regexp.MustCompile(`(?i)\b(aria-label|title|alt)\s*=\s*["']([^"']+)["']`)
	spaceRe  = regexp.MustCompile(`\s+`)
)

type reportAnchor struct{ url, label string }

func reportAnchors(doc string) []reportAnchor {
	var out []reportAnchor
	for _, m := range anchorRe.FindAllStringSubmatch(doc, -1) {
		u := strings.TrimSpace(html.UnescapeString(m[1]))
		if u == "" || strings.HasPrefix(u, "#") || strings.HasPrefix(strings.ToLower(u), "mailto:") {
			continue
		}
		label := strings.TrimSpace(spaceRe.ReplaceAllString(html.UnescapeString(tagRe.ReplaceAllString(m[2], " ")), " "))
		if label == "" {
			// An image link: its aria-label, title or the image's alt text.
			for _, src := range []string{m[0], m[2]} {
				if a := attrRe.FindStringSubmatch(src); a != nil {
					label = strings.TrimSpace(html.UnescapeString(a[2]))
					break
				}
			}
		}
		out = append(out, reportAnchor{u, label})
	}
	return out
}

// Shares of clicks (percent) for the October Product Update's links, as on
// the Report board's click map; other links share what is left.
// The images are labelled by their alt text (the composer's email) or as
// placeholders (reportEmailHTML).
var octLinkShares = map[string]float64{
	"read the docs": 38, "see changelog": 24, "read the guide": 19, "docs": 6,
	"email preferences": 2.0, "view in browser": 1.8, "unsubscribe": 1.6, "blog": 1.5, "changelog": 1.4, "acme": 1.3,
	"october product update": 1.2, "hero image": 1.2,
	"feature flags dashboard": 1.1, "feature flags ga image": 1.1, "audit log export settings": 1.1, "audit log exports image": 1.1,
}

var footerLabel = regexp.MustCompile(`(?i)unsubscribe|preferences|view (in|online)|browser|privacy|github|^x$|twitter|linkedin|image$`)

// reportLinks stores the email (when no other seeder wrote its content)
// and the clicks per link: Σ clicks equals the campaign's clicks.
func reportLinks(ctx context.Context, s *Seeder, c *reportCampaignSeed, oct bool) error {
	if strings.TrimSpace(c.html) == "" {
		c.html = reportEmailHTML(c.name, oct)
		if _, err := s.Tx.Exec(ctx, `UPDATE campaigns SET html = $2 WHERE id = $1 AND html = ''`, c.id, c.html); err != nil {
			return err
		}
	}
	anchors := reportAnchors(c.html)
	if len(anchors) == 0 {
		return nil
	}
	weights := make([]float64, len(anchors))
	body := []float64{38, 24, 19, 12, 8, 6, 5, 4, 3, 3}
	b := 0
	for i, a := range anchors {
		l := strings.ToLower(a.label)
		if v, ok := octLinkShares[l]; ok && oct {
			weights[i] = v
			continue
		}
		switch {
		case footerLabel.MatchString(a.label):
			weights[i] = 1.5
		case len(a.label) <= 10 && i < 4:
			weights[i] = 2 // header navigation
		default:
			if b < len(body) {
				weights[i] = body[b]
			} else {
				weights[i] = 2
			}
			b++
		}
	}
	clicks := Split(c.total(rmClicked), weights)
	// Links other seeders created for the campaign (matched by address) are
	// updated, so a link appears once.
	existing := map[string][]uuid.UUID{}
	rows, err := s.Tx.Query(ctx, `SELECT id, url FROM campaign_links WHERE campaign_id = $1 ORDER BY position, id`, c.id)
	if err != nil {
		return err
	}
	for rows.Next() {
		var id uuid.UUID
		var u string
		if err := rows.Scan(&id, &u); err != nil {
			rows.Close()
			return err
		}
		existing[u] = append(existing[u], id)
	}
	rows.Close()
	for i, a := range anchors {
		unique := int64(math.Round(float64(clicks[i]) * 0.86))
		if ids := existing[a.url]; len(ids) > 0 {
			existing[a.url] = ids[1:]
			if _, err := s.Tx.Exec(ctx, `UPDATE campaign_links SET label = $2, position = $3, clicks = $4, unique_clicks = $5 WHERE id = $1`,
				ids[0], a.label, i, clicks[i], unique); err != nil {
				return err
			}
			continue
		}
		if _, err := s.Tx.Exec(ctx, `INSERT INTO campaign_links (id, campaign_id, url, label, position, clicks, unique_clicks) VALUES ($1,$2,$3,$4,$5,$6,$7)`,
			ID(), c.id, a.url, a.label, i, clicks[i], unique); err != nil {
			return err
		}
	}
	return nil
}

// reportUnsubscribeFeedback writes the answers people gave on the
// unsubscribe page: about four in five answer; one in four of those who
// came to leave chose fewer emails instead.
func reportUnsubscribeFeedback(ctx context.Context, s *Seeder, c *reportCampaignSeed, oct bool) error {
	ws := s.WS()
	total := c.total(rmUnsub)
	var counts map[string]int64
	var fewer int64
	if oct {
		counts = map[string]int64{"too_frequent": 41, "not_relevant": 33, "never_signed_up": 9, "other": 7}
		fewer = 28
	} else {
		answered := int64(math.Round(float64(total) * 0.804))
		parts := Split(answered, []float64{41, 33, 9, 7})
		counts = map[string]int64{"too_frequent": parts[0], "not_relevant": parts[1], "never_signed_up": parts[2], "other": parts[3]}
		fewer = int64(math.Round(float64(total) * 0.25))
	}
	comments := map[string]string{"other": "Switching to the RSS feed"}
	var rows [][]any
	i := 0
	add := func(reason string, chose bool) {
		at := c.started.Add(time.Duration(30+(i*523)%(70*60)) * time.Minute)
		rows = append(rows, []any{ID(), ws.ID, c.id, reason, comments[reason], chose, at})
		i++
	}
	for _, r := range []string{"too_frequent", "not_relevant", "never_signed_up", "other"} {
		for n := int64(0); n < counts[r]; n++ {
			add(r, false)
		}
	}
	for n := int64(0); n < fewer; n++ {
		add("too_frequent", true)
	}
	if len(rows) == 0 {
		return nil
	}
	_, err := s.Tx.CopyFrom(ctx, pgx.Identifier{"unsubscribe_feedback"}, []string{"id", "workspace_id", "campaign_id", "reason", "comment", "chose_fewer", "created_at"},
		pgx.CopyFromRows(rows))
	return err
}

// reportRouteTimes stores the median time to delivery of each route.
func reportRouteTimes(ctx context.Context, s *Seeder, c *reportCampaignSeed, seq int) error {
	jitter := int32((seq * 137) % 300)
	_, err := s.Tx.Exec(ctx, `UPDATE campaign_routes cr SET delivery_p50_ms = CASE p.adapter WHEN 'ses' THEN 1800 + $2 WHEN 'postal' THEN 3400 + $2 ELSE 2500 + $2 END
		FROM providers p WHERE p.id = cr.provider_id AND cr.campaign_id = $1`, c.id, func() int32 {
		if c.name == "October Product Update" {
			return 0
		}
		return jitter - 150
	}())
	return err
}

// reportOctoberABTest records how the October Product Update's subject test
// was decided: human opens two hours after the 10% + 10% samples went out,
// machine opens excluded, then B to the remaining 80% at 12:00. The send
// ends when the remaining recipients are through.
func reportOctoberABTest(ctx context.Context, s *Seeder, c *reportCampaignSeed) error {
	var raw json.RawMessage
	if err := s.Tx.QueryRow(ctx, `SELECT coalesce(ab_test, '{}'::jsonb) FROM campaigns WHERE id = $1`, c.id).Scan(&raw); err != nil {
		return err
	}
	ab := map[string]any{}
	_ = json.Unmarshal(raw, &ab)
	if ab["kind"] == nil {
		ab["kind"] = "subject"
	}
	if ab["variants"] == nil {
		ab["variants"] = octoberVariants
	}
	decided := c.started.Add(2 * time.Hour)
	ab["winner"] = "B"
	ab["metric"] = "human_opens"
	ab["duration_hours"] = 2
	ab["test_percent"] = 20
	ab["decided_at"] = decided.UTC().Format(time.RFC3339)
	// Recipients whose opens could be measured by then (Apple Mail
	// prefetches excluded) and how many of them opened.
	ab["decision"] = map[string]any{"variants": []map[string]any{
		{"key": "A", "eligible": 2690, "conversions": 710},
		{"key": "B", "eligible": 2690, "conversions": 785},
	}}
	b, err := json.Marshal(ab)
	if err != nil {
		return err
	}
	// The remaining 38,384 go out at 12:00: SES at 14 /s, Postal at its
	// warm-up rate of 2 /s, in parallel.
	var sesRest, postalRest int64
	for _, cell := range c.cells {
		if cell.variant != "" {
			continue
		}
		switch cell.adapter {
		case "ses":
			sesRest += cell.m[rmSent]
		case "postal":
			postalRest += cell.m[rmSent]
		}
	}
	secs := math.Max(float64(sesRest)/14, float64(postalRest)/2)
	done := decided.Add(time.Duration(secs) * time.Second).Truncate(time.Minute)
	_, err = s.Tx.Exec(ctx, `UPDATE campaigns SET ab_test = $2, completed_at = $3, updated_at = $3 WHERE id = $1`, c.id, b, done)
	c.completed = done
	return err
}

// reportShareLink is the read-only link the Report board shows at the
// bottom of the October Product Update: made by Linh two days ago, opened
// a few times since. The token is sealed like the app seals it.
func reportShareLink(ctx context.Context, s *Seeder, c *reportCampaignSeed) error {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return err
	}
	token := base64.RawURLEncoding.EncodeToString(b)
	id := ID()
	sealed, err := s.Box.SealString(token, "report_share:"+id.String()+":token")
	if err != nil {
		return err
	}
	linh := s.Users["linh@acme.io"]
	_, err = s.Tx.Exec(ctx, `INSERT INTO report_shares (id, campaign_id, token_hash, token_sealed, created_by, created_at, views, last_viewed_at)
		VALUES ($1,$2,$3,$4,$5,$6,14,$7)`, id, c.id, authn.HashToken(token), sealed, linh.ID, s.Ago(46*time.Hour), s.Ago(3*time.Hour))
	return err
}
