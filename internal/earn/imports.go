package earn

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"path"
	"slices"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/riverqueue/river"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/emailaddr"
	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/jobs"
	"github.com/hostrepute/reputecast/internal/notify"
	"github.com/hostrepute/reputecast/internal/payments"
	"github.com/hostrepute/reputecast/internal/store"
)

// Import limits.
const (
	MaxImportFile = 25 << 20
	MaxImportRows = 100000
)

// Import sources and where their export is (brief 5.10).
var importSources = []string{"substack", "ghost", "beehiiv", "kit", "csv"}

// platformName names an old platform in consent and labels.
func platformName(src string) string {
	switch src {
	case "substack":
		return "Substack"
	case "ghost":
		return "Ghost"
	case "beehiiv":
		return "beehiiv"
	case "kit":
		return "Kit"
	}
	return "a CSV file"
}

// columnUses are what a column can be used as, with the headers that mean it.
var columnUses = []struct {
	use     string
	headers []string
}{
	{"email", []string{"email", "email_address", "e-mail", "subscriber_email", "email address", "e-mail address", "member_email"}},
	{"customer", []string{"stripe_customer_id", "stripe customer id", "stripe_customer", "customer_id", "stripe_connected_customer_id", "stripe customer", "customer"}},
	{"plan", []string{"plan", "tier", "tiers", "subscription", "membership", "subscription_tier", "plan_name", "tier_name", "complimentary_plan"}},
	{"paid_through", []string{"paid_through", "expiry", "expires_at", "expiry date", "current_period_end", "paid until", "paid_until", "end date", "expiration", "subscription_expiry"}},
	{"name", []string{"name", "full_name", "full name", "subscriber_name"}},
}

// EarnImportColumn is a column of the export and what it's used as.
type EarnImportColumn struct {
	Name    string `json:"name"`
	UsedAs  string `json:"usedAs" enum:"email,customer,plan,paid_through,name,"`
	Example string `json:"example"`
}

// EarnImportGroup is one group of the match step.
type EarnImportGroup struct {
	Count      int64             `json:"count"`
	ByCustomer int64             `json:"byCustomer"`
	ByEmail    int64             `json:"byEmail"`
	Examples   []EarnImportMatch `json:"examples"`
}

// EarnImportMatch is a row of the export as matched.
type EarnImportMatch struct {
	Row         int        `json:"row"`
	Email       string     `json:"email"`
	Name        string     `json:"name,omitempty"`
	Customer    string     `json:"customer,omitempty"`
	Tier        string     `json:"tier,omitempty"`
	AmountCents int64      `json:"amountCents,omitempty"`
	Currency    string     `json:"currency,omitempty"`
	Period      string     `json:"period,omitempty"`
	EndedAt     *time.Time `json:"endedAt,omitempty"`
	PaidThrough *time.Time `json:"paidThrough,omitempty"`
}

// EarnImportCandidate is one of a conflict's customers.
type EarnImportCandidate struct {
	Customer    string     `json:"customer"`
	Created     time.Time  `json:"created"`
	Status      string     `json:"status,omitempty"`
	Since       *time.Time `json:"since,omitempty"`
	AmountCents int64      `json:"amountCents,omitempty"`
	Currency    string     `json:"currency,omitempty"`
	Period      string     `json:"period,omitempty"`
	Tier        string     `json:"tier,omitempty"`
}

// EarnImportConflict is one email with several customers.
type EarnImportConflict struct {
	Row        int                   `json:"row"`
	Email      string                `json:"email"`
	Candidates []EarnImportCandidate `json:"candidates"`
	Chosen     string                `json:"chosen,omitempty"`
}

// EarnImportPrice is a provider price the export's paid members pay.
type EarnImportPrice struct {
	PriceID       string `json:"priceId"`
	AmountCents   int64  `json:"amountCents"`
	Currency      string `json:"currency"`
	Period        string `json:"period" enum:"month,year,"`
	Subscriptions int64  `json:"subscriptions"`
	// TierID is the tier it maps to; Legacy keeps it as a legacy price.
	TierID string `json:"tierId,omitempty"`
	Legacy bool   `json:"legacy"`
	// Known: a tier's price already (nothing to choose).
	Known bool `json:"known"`
}

// EarnImportDecisions are the choices of steps 3 and 4.
type EarnImportDecisions struct {
	// NoSubscription is what rows without a subscription become.
	NoSubscription string `json:"noSubscription" enum:"complimentary,free,skip" default:"complimentary"`
	// Conflicts picks a customer per conflicting row ("" skips it).
	Conflicts map[string]string `json:"conflicts,omitempty"`
	// Prices maps each provider price to a tier.
	Prices map[string]EarnImportPriceChoice `json:"prices,omitempty"`
	// Lists overrides the tiers' lists (empty: the tiers' own).
	SendWelcome bool `json:"sendWelcome"`
}

// EarnImportPriceChoice maps a provider price.
type EarnImportPriceChoice struct {
	TierID string `json:"tierId"`
	Legacy bool   `json:"legacy"`
}

// EarnImportStats is the import's summary.
type EarnImportStats struct {
	Paid          int64      `json:"paid"`
	Free          int64      `json:"free"`
	Complimentary int64      `json:"complimentary"`
	Skipped       int64      `json:"skipped"`
	Merged        int64      `json:"merged"`
	New           int64      `json:"new"`
	PricesMapped  int64      `json:"pricesMapped"`
	EmailsSent    int64      `json:"emailsSent"`
	CompFirst     *time.Time `json:"compFirst,omitempty"`
	CompLast      *time.Time `json:"compLast,omitempty"`
}

// EarnImport is the import dialog's state.
type EarnImport struct {
	ID       string             `json:"id"`
	Mode     string             `json:"mode" enum:"live,test"`
	Source   string             `json:"source" enum:"substack,ghost,beehiiv,kit,csv"`
	Platform string             `json:"platform"`
	Status   string             `json:"status" enum:"uploaded,matched,importing,done,failed,canceled"`
	FileName string             `json:"fileName"`
	FileSize int64              `json:"fileSize"`
	Rows     int                `json:"rows"`
	Columns  []EarnImportColumn `json:"columns"`
	// Problem: no_email (pick the email column) or empty.
	Problem      string `json:"problem,omitempty" enum:"no_email,"`
	WithCustomer int64  `json:"withCustomer"`
	// Account names the account the rows are matched with.
	Account string `json:"account"`
	// The match step's groups.
	Active     *EarnImportGroup     `json:"active,omitempty"`
	Ended      *EarnImportGroup     `json:"ended,omitempty"`
	None       *EarnImportGroup     `json:"none,omitempty"`
	Conflicts  []EarnImportConflict `json:"conflicts"`
	PaidFirst  *time.Time           `json:"paidFirst,omitempty"`
	PaidLast   *time.Time           `json:"paidLast,omitempty"`
	Prices     []EarnImportPrice    `json:"prices"`
	Lists      []EarnListRef        `json:"lists"`
	Decisions  EarnImportDecisions  `json:"decisions"`
	Done       int                  `json:"done"`
	Stats      *EarnImportStats     `json:"stats,omitempty"`
	Error      string               `json:"error,omitempty"`
	CreatedBy  string               `json:"createdBy,omitempty"`
	CreatedAt  time.Time            `json:"createdAt"`
	StartedAt  *time.Time           `json:"startedAt,omitempty"`
	FinishedAt *time.Time           `json:"finishedAt,omitempty"`
	Tiers      []EarnListRef        `json:"tiers"`
}

func (s *Service) importConn(ctx context.Context, wsID uuid.UUID) (connRow, bool, error) {
	conns, err := s.connections(ctx, wsID)
	if err != nil {
		return connRow{}, false, err
	}
	c, ok := active(conns)
	return c, ok, nil
}

func (s *Service) needImport(ctx context.Context, p *authz.Principal) error {
	if !p.Can(authz.SubscribersImport) {
		return api.Forbidden(authz.SubscribersImport, nil)
	}
	return nil
}

// readExport reads the CSV of an export: the file itself, or the CSV a
// zip holds (Substack's export is a zip).
func readExport(name string, data []byte) ([][]string, error) {
	if len(data) >= 4 && bytes.HasPrefix(data, []byte("PK\x03\x04")) {
		zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
		if err != nil {
			return nil, err
		}
		var best *zip.File
		for _, f := range zr.File {
			if strings.HasSuffix(strings.ToLower(f.Name), ".csv") && !strings.HasPrefix(path.Base(f.Name), ".") {
				// Substack's paid list is the one named after the email list.
				if best == nil || strings.Contains(strings.ToLower(f.Name), "email_list") || f.UncompressedSize64 > best.UncompressedSize64 {
					best = f
				}
			}
		}
		if best == nil {
			return nil, errors.New("no csv in the zip")
		}
		if best.UncompressedSize64 > MaxImportFile*4 {
			return nil, errors.New("too large")
		}
		rc, err := best.Open()
		if err != nil {
			return nil, err
		}
		defer func() { _ = rc.Close() }()
		data, err = io.ReadAll(io.LimitReader(rc, MaxImportFile*4))
		if err != nil {
			return nil, err
		}
	}
	data = bytes.TrimPrefix(data, []byte("\xef\xbb\xbf"))
	if !utf8.Valid(data) {
		return nil, errors.New("not UTF-8")
	}
	r := csv.NewReader(bytes.NewReader(data))
	r.FieldsPerRecord = -1
	r.LazyQuotes = true
	if first, _, _ := bytes.Cut(data, []byte("\n")); bytes.Count(first, []byte(";")) > bytes.Count(first, []byte(",")) {
		r.Comma = ';'
	}
	rows, err := r.ReadAll()
	if err != nil {
		return nil, err
	}
	return rows, nil
}

// detect picks each use's column from the headers.
func detect(header []string) map[string]string {
	out := map[string]string{}
	norm := func(h string) string { return strings.ToLower(strings.TrimSpace(strings.ReplaceAll(h, "-", "_"))) }
	for _, u := range columnUses {
		for _, h := range header {
			n := norm(h)
			if slices.ContainsFunc(u.headers, func(c string) bool { return norm(c) == n }) {
				if _, taken := out[u.use]; !taken {
					out[u.use] = h
				}
			}
		}
	}
	return out
}

func columnsOf(rows [][]string, mapping map[string]string) []EarnImportColumn {
	if len(rows) == 0 {
		return []EarnImportColumn{}
	}
	used := map[string]string{}
	for use, col := range mapping {
		used[col] = use
	}
	out := []EarnImportColumn{}
	for i, h := range rows[0] {
		ex := ""
		for _, r := range rows[1:] {
			if i < len(r) && strings.TrimSpace(r[i]) != "" {
				ex = strings.TrimSpace(r[i])
				break
			}
		}
		if len([]rune(ex)) > 40 {
			ex = string([]rune(ex)[:40]) + "…"
		}
		out = append(out, EarnImportColumn{Name: h, UsedAs: used[h], Example: ex})
	}
	return out
}

func parseDay(v string) *time.Time {
	v = strings.TrimSpace(v)
	if v == "" {
		return nil
	}
	for _, layout := range []string{"2006-01-02", time.RFC3339, "2006-01-02 15:04:05", "2006-01-02T15:04:05", "01/02/2006", "1/2/2006", "2 Jan 2006", "Jan 2, 2006"} {
		if t, err := time.Parse(layout, v); err == nil {
			d := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
			return &d
		}
	}
	return nil
}

// UploadImport reads an export and finds its columns. Nothing is written
// to contacts until the import is started.
func (s *Service) UploadImport(ctx context.Context, ws *store.Workspace, p *authz.Principal, source, name string, data []byte) (EarnImport, error) {
	if err := s.needImport(ctx, p); err != nil {
		return EarnImport{}, err
	}
	if !slices.Contains(importSources, source) {
		return EarnImport{}, invalid(ctx, "query.source", "import.source", nil)
	}
	conn, ok, err := s.importConn(ctx, ws.ID)
	if err != nil {
		return EarnImport{}, err
	}
	if !ok {
		return EarnImport{}, problem(ctx, http.StatusConflict, CodeNotConnected, "import.notConnected", nil)
	}
	if len(data) == 0 || len(data) > MaxImportFile {
		return EarnImport{}, invalid(ctx, "body", "import.size", i18n.Args{"mb": MaxImportFile >> 20})
	}
	rows, err := readExport(name, data)
	if err != nil || len(rows) < 2 {
		return EarnImport{}, invalid(ctx, "body", "import.unreadable", nil)
	}
	if len(rows)-1 > MaxImportRows {
		return EarnImport{}, invalid(ctx, "body", "import.tooMany", i18n.Args{"count": MaxImportRows})
	}
	mapping := detect(rows[0])
	cols := columnsOf(rows, mapping)
	mb, _ := json.Marshal(mapping)
	cb, _ := json.Marshal(cols)
	id := newID()
	now := s.now()
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if err := q.EarnImportInsert(ctx, store.EarnImportInsertParams{ID: id, WorkspaceID: ws.ID, Mode: conn.Mode, Source: source, FileName: clip(path.Base(name), 200),
			FileSize: int64(len(data)), Columns: cb, Mapping: mb, Total: int32(len(rows) - 1), CreatedBy: userID(p), CreatedAt: now}); err != nil {
			return err
		}
		if err := q.EarnImportFileInsert(ctx, store.EarnImportFileInsertParams{ImportID: id, Data: data}); err != nil {
			return err
		}
		return s.stageRows(ctx, q, id, rows, mapping)
	})
	if err != nil {
		return EarnImport{}, err
	}
	return s.Import(ctx, ws, p, id.String())
}

// stageRows keeps the rows' useful columns.
func (s *Service) stageRows(ctx context.Context, q *store.Queries, id uuid.UUID, rows [][]string, mapping map[string]string) error {
	if err := q.EarnImportRowsClear(ctx, id); err != nil {
		return err
	}
	idx := map[string]int{}
	for i, h := range rows[0] {
		for use, col := range mapping {
			if col == h {
				idx[use] = i
			}
		}
	}
	cell := func(r []string, use string) string {
		i, ok := idx[use]
		if !ok || i >= len(r) {
			return ""
		}
		return strings.TrimSpace(r[i])
	}
	params := make([]store.EarnImportRowInsertParams, 0, len(rows)-1)
	for n, r := range rows[1:] {
		email := strings.ToLower(cell(r, "email"))
		params = append(params, store.EarnImportRowInsertParams{ImportID: id, RowNo: int32(n + 1), Email: clip(email, 320), Name: clip(cell(r, "name"), 200),
			CustomerID: clip(cell(r, "customer"), 100), Plan: clip(cell(r, "plan"), 120), PaidThrough: parseDay(cell(r, "paid_through"))})
	}
	_, err := q.EarnImportRowInsert(ctx, params)
	return err
}

func (s *Service) loadImport(ctx context.Context, ws *store.Workspace, id string) (store.EarnImportGetRow, error) {
	iid, err := uuid.Parse(id)
	if err != nil {
		return store.EarnImportGetRow{}, api.NotFound("Import")
	}
	r, err := s.Q.EarnImportGet(ctx, store.EarnImportGetParams{ID: iid, WorkspaceID: ws.ID})
	if err != nil {
		return store.EarnImportGetRow{}, api.MapError(err, "Import")
	}
	return r, nil
}

// EarnImportColumnsInput picks the columns by hand.
type EarnImportColumnsInput struct {
	Email       string `json:"email" maxLength:"200"`
	Customer    string `json:"customer,omitempty" maxLength:"200"`
	Plan        string `json:"plan,omitempty" maxLength:"200"`
	PaidThrough string `json:"paidThrough,omitempty" maxLength:"200"`
	Name        string `json:"name,omitempty" maxLength:"200"`
}

// SetImportColumns picks the columns of an export by hand (another CSV,
// or a file without an email column).
func (s *Service) SetImportColumns(ctx context.Context, ws *store.Workspace, p *authz.Principal, id string, in EarnImportColumnsInput) (EarnImport, error) {
	imp, err := s.loadImport(ctx, ws, id)
	if err != nil {
		return EarnImport{}, err
	}
	if imp.Status != "uploaded" && imp.Status != "matched" {
		return EarnImport{}, problem(ctx, http.StatusConflict, "import_started", "import.started", nil)
	}
	data, err := s.Q.EarnImportFile(ctx, imp.ID)
	if err != nil {
		return EarnImport{}, err
	}
	rows, err := readExport(imp.FileName, data)
	if err != nil {
		return EarnImport{}, invalid(ctx, "body", "import.unreadable", nil)
	}
	mapping := map[string]string{}
	for use, col := range map[string]string{"email": in.Email, "customer": in.Customer, "plan": in.Plan, "paid_through": in.PaidThrough, "name": in.Name} {
		if col == "" {
			continue
		}
		if !slices.Contains(rows[0], col) {
			return EarnImport{}, invalid(ctx, "body."+use, "import.column", i18n.Args{"column": col})
		}
		mapping[use] = col
	}
	if mapping["email"] == "" {
		return EarnImport{}, invalid(ctx, "body.email", "import.noEmail", nil)
	}
	mb, _ := json.Marshal(mapping)
	cb, _ := json.Marshal(columnsOf(rows, mapping))
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if err := q.EarnImportSetColumns(ctx, store.EarnImportSetColumnsParams{ID: imp.ID, Columns: cb, Mapping: mb, Total: int32(len(rows) - 1)}); err != nil {
			return err
		}
		return s.stageRows(ctx, q, imp.ID, rows, mapping)
	})
	if err != nil {
		return EarnImport{}, err
	}
	return s.Import(ctx, ws, p, id)
}

// subSummary is a matched subscription as the rows keep it.
type subSummary struct {
	ID       string     `json:"id"`
	Status   string     `json:"status"`
	Price    string     `json:"price"`
	Amount   int64      `json:"amount"`
	Currency string     `json:"currency"`
	Interval string     `json:"interval"`
	Since    *time.Time `json:"since,omitempty"`
	Ended    *time.Time `json:"ended,omitempty"`
}

func liveStatus(st string) bool { return st == "active" || st == "trialing" || st == "past_due" }

// MatchImport matches every row with the connected account, read-only: by
// customer ID first, then by email. One email with several customers is a
// conflict to settle by hand.
func (s *Service) MatchImport(ctx context.Context, ws *store.Workspace, p *authz.Principal, id string) (EarnImport, error) {
	imp, err := s.loadImport(ctx, ws, id)
	if err != nil {
		return EarnImport{}, err
	}
	if imp.Status != "uploaded" && imp.Status != "matched" {
		return EarnImport{}, problem(ctx, http.StatusConflict, "import_started", "import.started", nil)
	}
	var mapping map[string]string
	_ = json.Unmarshal(imp.Mapping, &mapping)
	if mapping["email"] == "" {
		return EarnImport{}, problem(ctx, http.StatusUnprocessableEntity, "no_email_column", "import.noEmail", nil)
	}
	conn, ok, err := s.importConn(ctx, ws.ID)
	if err != nil {
		return EarnImport{}, err
	}
	if !ok {
		return EarnImport{}, problem(ctx, http.StatusConflict, CodeNotConnected, "import.notConnected", nil)
	}
	client, err := s.open(conn.PaymentConnection)
	if err != nil {
		return EarnImport{}, err
	}
	// Read every customer and subscription once (pages of 100).
	customers := map[string]payments.Customer{}
	byEmail := map[string][]string{}
	for cursor := ""; ; {
		pg, err := client.ListCustomers(ctx, cursor)
		if err != nil {
			return EarnImport{}, keyProblem(ctx, err)
		}
		for _, c := range pg.Items {
			customers[c.ID] = c
			if c.Email != "" {
				byEmail[c.Email] = append(byEmail[c.Email], c.ID)
			}
		}
		if pg.Next == "" {
			break
		}
		cursor = pg.Next
	}
	subs := map[string][]payments.Subscription{}
	for cursor := ""; ; {
		pg, err := client.ListSubscriptions(ctx, cursor)
		if err != nil {
			return EarnImport{}, keyProblem(ctx, err)
		}
		for _, sub := range pg.Items {
			subs[sub.CustomerID] = append(subs[sub.CustomerID], sub)
		}
		if pg.Next == "" {
			break
		}
		cursor = pg.Next
	}
	best := func(customer string) (payments.Subscription, bool) {
		var out payments.Subscription
		found := false
		for _, s := range subs[customer] {
			switch {
			case !found:
				out, found = s, true
			case liveStatus(s.Status) && !liveStatus(out.Status):
				out = s
			case liveStatus(s.Status) == liveStatus(out.Status) && s.StartedAt.After(out.StartedAt):
				out = s
			}
		}
		return out, found
	}
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		after := int32(0)
		for {
			rows, err := q.EarnImportRows(ctx, store.EarnImportRowsParams{ImportID: imp.ID, After: after, Lim: 1000})
			if err != nil {
				return err
			}
			if len(rows) == 0 {
				break
			}
			for _, r := range rows {
				after = r.RowNo
				m := store.EarnImportRowMatchParams{ImportID: imp.ID, RowNo: r.RowNo, Subscription: []byte(`{}`), Candidates: []byte(`[]`)}
				email, problemText := emailaddr.Check(r.Email)
				switch {
				case problemText != "":
					m.Match = "invalid"
				default:
					customer := ""
					if r.CustomerID != "" {
						if _, ok := customers[r.CustomerID]; ok {
							customer, m.MatchedBy = r.CustomerID, "customer"
						}
					}
					if customer == "" {
						ids := byEmail[email]
						switch {
						case len(ids) == 1:
							customer, m.MatchedBy = ids[0], "email"
						case len(ids) > 1:
							m.Match = "conflict"
							cands := []EarnImportCandidate{}
							for _, cid := range ids {
								c := EarnImportCandidate{Customer: cid, Created: customers[cid].Created}
								if sub, ok := best(cid); ok {
									c.Status, c.AmountCents, c.Currency, c.Period = sub.Status, sub.AmountCents, sub.Currency, sub.Interval
									st := sub.StartedAt
									c.Since = &st
								}
								cands = append(cands, c)
							}
							m.Candidates, _ = json.Marshal(cands)
						}
					}
					if m.Match == "conflict" {
						break
					}
					m.StripeCustomerID = customer
					sub, ok := best(customer)
					switch {
					case customer == "" || !ok:
						m.Match = "none"
					case liveStatus(sub.Status):
						m.Match = "active"
					default:
						m.Match = "ended"
					}
					if ok {
						m.StripeSubscriptionID, m.StripePriceID = sub.ID, sub.PriceID
						st := sub.StartedAt
						m.Subscription, _ = json.Marshal(subSummary{ID: sub.ID, Status: sub.Status, Price: sub.PriceID, Amount: sub.AmountCents, Currency: sub.Currency,
							Interval: sub.Interval, Since: &st, Ended: firstTime(sub.EndedAt, sub.CanceledAt)})
					}
				}
				if err := q.EarnImportRowMatch(ctx, m); err != nil {
					return err
				}
			}
		}
		return q.EarnImportMatched(ctx, store.EarnImportMatchedParams{ID: imp.ID, At: ptr(s.now()), Stats: []byte(`{}`)})
	})
	if err != nil {
		return EarnImport{}, err
	}
	return s.Import(ctx, ws, p, id)
}

// SetImportDecisions saves the choices of steps 3 and 4.
func (s *Service) SetImportDecisions(ctx context.Context, ws *store.Workspace, p *authz.Principal, id string, in EarnImportDecisions) (EarnImport, error) {
	imp, err := s.loadImport(ctx, ws, id)
	if err != nil {
		return EarnImport{}, err
	}
	if imp.Status != "matched" {
		return EarnImport{}, problem(ctx, http.StatusConflict, "import_not_matched", "import.notMatched", nil)
	}
	if in.NoSubscription == "" {
		in.NoSubscription = "complimentary"
	}
	tiers, err := s.Q.EarnTiers(ctx, ws.ID)
	if err != nil {
		return EarnImport{}, err
	}
	for pr, c := range in.Prices {
		if c.TierID == "" {
			delete(in.Prices, pr)
			continue
		}
		if !slices.ContainsFunc(tiers, func(t store.EarnTier) bool { return t.ID.String() == c.TierID }) {
			return EarnImport{}, invalid(ctx, "body.prices", "grant.tier", nil)
		}
	}
	b, _ := json.Marshal(in)
	if err := s.Q.EarnImportSetDecisions(ctx, store.EarnImportSetDecisionsParams{ID: imp.ID, Decisions: b, SendWelcome: in.SendWelcome}); err != nil {
		return EarnImport{}, err
	}
	return s.Import(ctx, ws, p, id)
}

// Import is the import dialog's state.
func (s *Service) Import(ctx context.Context, ws *store.Workspace, p *authz.Principal, id string) (EarnImport, error) {
	imp, err := s.loadImport(ctx, ws, id)
	if err != nil {
		return EarnImport{}, err
	}
	out := EarnImport{ID: imp.ID.String(), Mode: imp.Mode, Source: imp.Source, Platform: platformName(imp.Source), Status: imp.Status, FileName: imp.FileName,
		FileSize: imp.FileSize, Rows: int(imp.Total), Columns: []EarnImportColumn{}, Conflicts: []EarnImportConflict{}, Prices: []EarnImportPrice{}, Lists: []EarnListRef{},
		Done: int(imp.DoneRows), Error: imp.Error, CreatedBy: imp.CreatedByName, CreatedAt: imp.CreatedAt, StartedAt: imp.StartedAt, FinishedAt: imp.FinishedAt, Tiers: []EarnListRef{}}
	_ = json.Unmarshal(imp.Columns, &out.Columns)
	_ = json.Unmarshal(imp.Decisions, &out.Decisions)
	if out.Decisions.NoSubscription == "" {
		out.Decisions.NoSubscription = "complimentary"
	}
	var mapping map[string]string
	_ = json.Unmarshal(imp.Mapping, &mapping)
	if mapping["email"] == "" {
		out.Problem = "no_email"
	}
	if conn, ok, _ := s.importConn(ctx, ws.ID); ok {
		out.Account = conn.AccountName
	}
	_ = s.Pool.QueryRow(ctx, `SELECT count(*) FROM earn_import_rows WHERE import_id = $1 AND customer_id <> ''`, imp.ID).Scan(&out.WithCustomer)
	tiers, err := s.Q.EarnTiers(ctx, ws.ID)
	if err != nil {
		return out, err
	}
	names := s.listNamer(ctx, ws.ID)
	for _, t := range tiers {
		out.Tiers = append(out.Tiers, EarnListRef{ID: t.ID.String(), Name: shortTier(t.Name, names(&t.ID))})
	}
	if imp.Status == "uploaded" {
		return out, nil
	}
	counts, err := s.Q.EarnImportGroupCounts(ctx, imp.ID)
	if err != nil {
		return out, err
	}
	groups := map[string]*EarnImportGroup{}
	for _, c := range counts {
		g := groups[c.Match]
		if g == nil {
			g = &EarnImportGroup{Examples: []EarnImportMatch{}}
			groups[c.Match] = g
		}
		g.Count += c.N
		switch c.MatchedBy {
		case "customer":
			g.ByCustomer += c.N
		case "email":
			g.ByEmail += c.N
		}
	}
	priceTier := s.priceTiers(ctx, ws, imp.Mode)
	for _, k := range []string{"active", "ended", "none"} {
		g := groups[k]
		if g == nil {
			g = &EarnImportGroup{Examples: []EarnImportMatch{}}
		}
		ex, err := s.Q.EarnImportExamples(ctx, store.EarnImportExamplesParams{ImportID: imp.ID, Match: k, Lim: 3})
		if err != nil {
			return out, err
		}
		for _, r := range ex {
			m := EarnImportMatch{Row: int(r.RowNo), Email: r.Email, Name: r.Name, Customer: r.StripeCustomerID, PaidThrough: r.PaidThrough}
			var sum subSummary
			if json.Unmarshal(r.Subscription, &sum) == nil && sum.ID != "" {
				m.AmountCents, m.Currency, m.Period, m.EndedAt = sum.Amount, sum.Currency, sum.Interval, sum.Ended
				m.Tier = priceTier[sum.Price]
			}
			g.Examples = append(g.Examples, m)
		}
		switch k {
		case "active":
			out.Active = g
		case "ended":
			out.Ended = g
		case "none":
			out.None = g
		}
	}
	_ = s.Pool.QueryRow(ctx, `SELECT min(paid_through), max(paid_through) FROM earn_import_rows WHERE import_id = $1 AND match = 'none'`, imp.ID).
		Scan(&out.PaidFirst, &out.PaidLast)
	conflicts, err := s.Q.EarnImportConflicts(ctx, imp.ID)
	if err != nil {
		return out, err
	}
	for _, r := range conflicts {
		c := EarnImportConflict{Row: int(r.RowNo), Email: r.Email, Candidates: []EarnImportCandidate{}, Chosen: out.Decisions.Conflicts[fmt.Sprint(r.RowNo)]}
		_ = json.Unmarshal(r.Candidates, &c.Candidates)
		out.Conflicts = append(out.Conflicts, c)
	}
	prices, err := s.Q.EarnImportPriceCounts(ctx, imp.ID)
	if err != nil {
		return out, err
	}
	known := s.knownPrices(ctx, ws, imp.Mode)
	for _, pc := range prices {
		ip := EarnImportPrice{PriceID: pc.StripePriceID, AmountCents: pc.Amount, Currency: pc.Currency, Period: periodOf(pc.Interval), Subscriptions: pc.N}
		if k, ok := known[pc.StripePriceID]; ok {
			ip.TierID, ip.Legacy, ip.Known = k.TierID.String(), k.Status == "legacy", true
		} else if ch, ok := out.Decisions.Prices[pc.StripePriceID]; ok {
			ip.TierID, ip.Legacy = ch.TierID, ch.Legacy
		} else if t := s.guessTier(ctx, ws, pc.Amount, periodOf(pc.Interval)); t != "" {
			ip.TierID = t
		}
		out.Prices = append(out.Prices, ip)
	}
	lists := map[uuid.UUID]bool{}
	for _, ip := range out.Prices {
		if tid, err := uuid.Parse(ip.TierID); err == nil {
			for _, t := range tiers {
				if t.ID == tid {
					for _, l := range t.ListIds {
						lists[l] = true
					}
				}
			}
		}
	}
	all, _ := s.Q.EarnLists(ctx, ws.ID)
	for _, l := range all {
		if lists[l.ID] {
			out.Lists = append(out.Lists, EarnListRef{ID: l.ID.String(), Name: l.Name})
		}
	}
	if len(imp.Stats) > 2 && (imp.Status == "done" || imp.Status == "importing" || imp.Status == "failed") {
		var st EarnImportStats
		if json.Unmarshal(imp.Stats, &st) == nil {
			out.Stats = &st
		}
	}
	return out, nil
}

func (s *Service) knownPrices(ctx context.Context, ws *store.Workspace, mode string) map[string]store.EarnPrice {
	out := map[string]store.EarnPrice{}
	prices, err := s.Q.EarnPrices(ctx, ws.ID)
	if err != nil {
		return out
	}
	for _, p := range prices {
		id := p.StripeLiveID
		if mode == "test" {
			id = p.StripeTestID
		}
		if id != "" {
			out[id] = p
		}
	}
	return out
}

// priceTiers names the tier a provider price maps to.
func (s *Service) priceTiers(ctx context.Context, ws *store.Workspace, mode string) map[string]string {
	out := map[string]string{}
	names := s.listNamer(ctx, ws.ID)
	tiers, _ := s.Q.EarnTiers(ctx, ws.ID)
	tn := map[uuid.UUID]string{}
	for _, t := range tiers {
		tn[t.ID] = shortTier(t.Name, names(&t.ID))
	}
	for id, p := range s.knownPrices(ctx, ws, mode) {
		out[id] = tn[p.TierID]
	}
	return out
}

// guessTier suggests the tier whose current price has the same amount and period.
func (s *Service) guessTier(ctx context.Context, ws *store.Workspace, amount int64, period string) string {
	prices, err := s.Q.EarnPrices(ctx, ws.ID)
	if err != nil {
		return ""
	}
	for _, p := range prices {
		if p.AmountCents == amount && p.Period == period {
			return p.TierID.String()
		}
	}
	return ""
}

// ImportArgs runs a paid-members import.
type ImportArgs struct {
	ImportID string `json:"import_id"`
}

// Kind implements river.JobArgs.
func (ImportArgs) Kind() string { return "earn.import" }

// InsertOpts runs on the default queue.
func (ImportArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{Queue: jobs.QueueDefault, MaxAttempts: 5}
}

// ImportWorker runs ImportArgs.
type ImportWorker struct {
	river.WorkerDefaults[ImportArgs]
	S *Service
}

// Timeout bounds one run.
func (w *ImportWorker) Timeout(*river.Job[ImportArgs]) time.Duration { return 30 * time.Minute }

// Work implements river.Worker.
func (w *ImportWorker) Work(ctx context.Context, job *river.Job[ImportArgs]) error {
	id, err := uuid.Parse(job.Args.ImportID)
	if err != nil {
		return river.JobCancel(err)
	}
	return w.S.RunImport(ctx, id)
}

// StartImport checks the choices and starts the import in the background.
func (s *Service) StartImport(ctx context.Context, ws *store.Workspace, p *authz.Principal, id string) (EarnImport, error) {
	if err := s.needImport(ctx, p); err != nil {
		return EarnImport{}, err
	}
	cur, err := s.Import(ctx, ws, p, id)
	if err != nil {
		return EarnImport{}, err
	}
	if cur.Status != "matched" {
		return EarnImport{}, problem(ctx, http.StatusConflict, "import_not_matched", "import.notMatched", nil)
	}
	for _, pr := range cur.Prices {
		if pr.TierID == "" {
			return EarnImport{}, problem(ctx, http.StatusUnprocessableEntity, "unmapped_price", "import.unmappedPrice",
				i18n.Args{"price": priceText(pr.AmountCents, pr.Currency, pr.Period), "id": shortID(pr.PriceID)})
		}
	}
	// Any guessed mapping becomes the decision.
	if cur.Decisions.Prices == nil {
		cur.Decisions.Prices = map[string]EarnImportPriceChoice{}
	}
	for _, pr := range cur.Prices {
		if !pr.Known {
			cur.Decisions.Prices[pr.PriceID] = EarnImportPriceChoice{TierID: pr.TierID, Legacy: pr.Legacy}
		}
	}
	for _, c := range cur.Conflicts {
		if _, ok := cur.Decisions.Conflicts[fmt.Sprint(c.Row)]; !ok {
			return EarnImport{}, problem(ctx, http.StatusUnprocessableEntity, "unsettled_conflict", "import.conflict", i18n.Args{"email": c.Email})
		}
	}
	iid, _ := uuid.Parse(id)
	b, _ := json.Marshal(cur.Decisions)
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if err := q.EarnImportSetDecisions(ctx, store.EarnImportSetDecisionsParams{ID: iid, Decisions: b, SendWelcome: cur.Decisions.SendWelcome}); err != nil {
			return err
		}
		n, err := q.EarnImportStart(ctx, store.EarnImportStartParams{ID: iid, At: ptr(s.now())})
		if err != nil {
			return err
		}
		if n == 0 {
			return problem(ctx, http.StatusConflict, "import_not_matched", "import.notMatched", nil)
		}
		if s.Jobs != nil {
			if jc := s.Jobs(); jc != nil {
				_, err := jc.InsertTx(ctx, tx, ImportArgs{ImportID: id}, nil)
				return err
			}
		}
		return nil
	})
	if err != nil {
		return EarnImport{}, err
	}
	return s.Import(ctx, ws, p, id)
}

// CancelImport drops an import that hasn't started, with its file.
func (s *Service) CancelImport(ctx context.Context, ws *store.Workspace, id string) error {
	iid, err := uuid.Parse(id)
	if err != nil {
		return api.NotFound("Import")
	}
	n, err := s.Q.EarnImportCancel(ctx, store.EarnImportCancelParams{ID: iid, WorkspaceID: ws.ID, At: ptr(s.now())})
	if err != nil {
		return err
	}
	if n == 0 {
		return api.NotFound("Import")
	}
	return s.Q.EarnImportFileDelete(ctx, iid)
}

// RunImport imports the rows in pages: paid members with their
// subscription, ended ones as free subscribers, rows without a
// subscription as the choice says. Contacts are merged; everyone joins the
// tiers' lists with consent "Imported from Substack (paid member)". Nothing
// changes in the provider.
func (s *Service) RunImport(ctx context.Context, id uuid.UUID) error {
	imp, err := s.Q.EarnImportByID(ctx, id)
	if err != nil {
		return err
	}
	if imp.Status != "importing" {
		return nil
	}
	ws, err := s.Q.GetWorkspace(ctx, imp.WorkspaceID)
	if err != nil {
		return err
	}
	var dec EarnImportDecisions
	_ = json.Unmarshal(imp.Decisions, &dec)
	now := s.now()
	stats := EarnImportStats{}
	_ = json.Unmarshal(imp.Stats, &stats)
	platform := platformName(imp.Source)
	consent := msgs.M("import.consent").Text(defaultLang, i18n.Args{"platform": platform})
	label := msgs.M("import.source").Text(defaultLang, i18n.Args{"platform": platform})
	// Prices first: each provider price found maps to its tier.
	priceIDs := map[string]uuid.UUID{}
	tierOf := map[string]uuid.UUID{}
	known := s.knownPrices(ctx, &ws, imp.Mode)
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		rows, err := q.EarnImportPriceCounts(ctx, id)
		if err != nil {
			return err
		}
		for _, pc := range rows {
			if k, ok := known[pc.StripePriceID]; ok {
				priceIDs[pc.StripePriceID], tierOf[pc.StripePriceID] = k.ID, k.TierID
				continue
			}
			ch, ok := dec.Prices[pc.StripePriceID]
			if !ok {
				continue
			}
			tid, err := uuid.Parse(ch.TierID)
			if err != nil {
				continue
			}
			period := periodOf(pc.Interval)
			if period == "" {
				period = "month"
			}
			status := "legacy"
			if !ch.Legacy {
				// Mapped as the tier's price: it is that price in the provider.
				prices, err := q.EarnTierPrices(ctx, tid)
				if err != nil {
					return err
				}
				status = "current"
				for _, p := range prices {
					if p.Period == period && p.Status == "current" {
						if p.AmountCents == pc.Amount && ((imp.Mode == "live" && p.StripeLiveID == "") || (imp.Mode == "test" && p.StripeTestID == "")) {
							// The tier's price, not in the provider yet: it is this one.
							if err := q.EarnPriceSetStripe(ctx, store.EarnPriceSetStripeParams{ID: p.ID, Mode: imp.Mode, StripeID: pc.StripePriceID}); err != nil {
								return err
							}
							priceIDs[pc.StripePriceID], tierOf[pc.StripePriceID] = p.ID, tid
							status = ""
							break
						}
						status = "legacy"
					}
				}
				if status == "" {
					stats.PricesMapped++
					continue
				}
			}
			pid := newID()
			live, test := "", ""
			if imp.Mode == "live" {
				live = pc.StripePriceID
			} else {
				test = pc.StripePriceID
			}
			if err := q.EarnPriceInsert(ctx, store.EarnPriceInsertParams{ID: pid, WorkspaceID: ws.ID, TierID: tid, Period: period, AmountCents: pc.Amount, Currency: pc.Currency,
				Status: status, StripeLiveID: live, StripeTestID: test, Origin: "mapped", At: now}); err != nil {
				return err
			}
			if status == "legacy" {
				if _, err := tx.Exec(ctx, `UPDATE earn_prices SET retired_at = $2 WHERE id = $1`, pid, now); err != nil {
					return err
				}
			}
			priceIDs[pc.StripePriceID], tierOf[pc.StripePriceID] = pid, tid
			stats.PricesMapped++
		}
		return nil
	})
	if err != nil {
		return s.failImport(ctx, imp, stats, err)
	}
	tiers, err := s.Q.EarnTiers(ctx, ws.ID)
	if err != nil {
		return s.failImport(ctx, imp, stats, err)
	}
	listsOf := map[uuid.UUID][]uuid.UUID{}
	var allLists []uuid.UUID
	for _, t := range tiers {
		listsOf[t.ID] = t.ListIds
		for _, l := range t.ListIds {
			if !slices.Contains(allLists, l) {
				allLists = append(allLists, l)
			}
		}
	}
	// The export's first tier's lists for rows without a known price.
	defaultTier := uuid.Nil
	if len(tiers) > 0 {
		defaultTier = tiers[0].ID
	}
	conns, err := s.connections(ctx, ws.ID)
	if err != nil {
		return s.failImport(ctx, imp, stats, err)
	}
	conn, _ := connected(conns, imp.Mode)
	var client payments.Client
	if conn.ID != uuid.Nil {
		client, _ = s.open(conn.PaymentConnection)
	}
	var welcome []memberEmail
	after := imp.DoneRows
	for {
		rows, err := s.Q.EarnImportRows(ctx, store.EarnImportRowsParams{ImportID: id, After: after, Lim: 200})
		if err != nil {
			return s.failImport(ctx, imp, stats, err)
		}
		if len(rows) == 0 {
			break
		}
		err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
			q := store.New(tx)
			for _, r := range rows {
				res, contact, wm, err := s.importRow(ctx, q, &ws, imp, dec, r, conn, client, priceIDs, tierOf, listsOf, defaultTier, consent, label, now, &stats)
				if err != nil {
					return err
				}
				if wm != nil {
					welcome = append(welcome, *wm)
				}
				if err := q.EarnImportRowResult(ctx, store.EarnImportRowResultParams{ImportID: id, RowNo: r.RowNo, Result: res, ContactID: contact}); err != nil {
					return err
				}
			}
			after = rows[len(rows)-1].RowNo
			return q.EarnImportProgress(ctx, store.EarnImportProgressParams{ID: id, DoneRows: after})
		})
		if err != nil {
			return s.failImport(ctx, imp, stats, err)
		}
	}
	if dec.SendWelcome {
		for _, m := range welcome {
			if s.sendMemberEmail(ctx, &ws, imp.Mode, m, nil) == "sent" {
				stats.EmailsSent++
			}
		}
	}
	sb, _ := json.Marshal(stats)
	finished := s.now()
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if err := q.EarnImportFinish(ctx, store.EarnImportFinishParams{ID: id, Status: "done", At: &finished, Stats: sb, Error: "", DoneRows: after}); err != nil {
			return err
		}
		if err := q.EarnImportFileDelete(ctx, id); err != nil {
			return err
		}
		actor := (*uuid.UUID)(nil)
		if imp.CreatedBy != nil {
			actor = imp.CreatedBy
		}
		if err := s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "paid_member.imported", TargetType: "earn_import", TargetID: id.String(),
			TargetName: imp.FileName, Actor: importActor(ctx, s, actor),
			Metadata: map[string]any{"source": imp.Source, "mode": imp.Mode, "rows": imp.Total, "paid": stats.Paid, "free": stats.Free, "complimentary": stats.Complimentary,
				"skipped": stats.Skipped, "merged": stats.Merged, "new": stats.New, "prices_mapped": stats.PricesMapped, "emails_sent": stats.EmailsSent}}); err != nil {
			return err
		}
		if imp.CreatedBy != nil {
			n := notify.Notification{WorkspaceID: &ws.ID, Kind: KindPaidImportDone, Severity: "ok",
				Title: msgs.M("notify.importTitle").Text(defaultLang, i18n.Args{"platform": platform}),
				Body:  msgs.M("notify.importBody").Text(defaultLang, i18n.Args{"count": imp.Total, "paid": stats.Paid, "free": stats.Free, "comp": stats.Complimentary}),
				Link:  "/w/" + ws.Slug + "/earn/members"}
			return notify.ToUser(ctx, tx, *imp.CreatedBy, n)
		}
		return nil
	})
	return err
}

func importActor(ctx context.Context, s *Service, id *uuid.UUID) *audit.Actor {
	if id == nil {
		return nil
	}
	name, _ := s.Q.EarnUserName(ctx, *id)
	return &audit.Actor{Type: authz.ActorUser, ID: id, Name: name}
}

func (s *Service) failImport(ctx context.Context, imp store.EarnImport, stats EarnImportStats, cause error) error {
	sb, _ := json.Marshal(stats)
	_ = s.Q.EarnImportFinish(ctx, store.EarnImportFinishParams{ID: imp.ID, Status: "failed", At: ptr(s.now()), Stats: sb, Error: clip(cause.Error(), 500), DoneRows: imp.DoneRows})
	return cause
}

// importRow imports one row.
func (s *Service) importRow(ctx context.Context, q *store.Queries, ws *store.Workspace, imp store.EarnImport, dec EarnImportDecisions, r store.EarnImportRow,
	conn connRow, client payments.Client, priceIDs map[string]uuid.UUID, tierOf map[string]uuid.UUID, listsOf map[uuid.UUID][]uuid.UUID, defaultTier uuid.UUID,
	consent, label string, now time.Time, stats *EarnImportStats) (string, *uuid.UUID, *memberEmail, error) {
	match := r.Match
	customer, subID := r.StripeCustomerID, r.StripeSubscriptionID
	var sum subSummary
	_ = json.Unmarshal(r.Subscription, &sum)
	if match == "conflict" {
		chosen := dec.Conflicts[fmt.Sprint(r.RowNo)]
		if chosen == "" {
			stats.Skipped++
			return "skipped", nil, nil, nil
		}
		customer = chosen
		match = "none"
		if client != nil {
			if pg, err := client.ListSubscriptions(ctx, ""); err == nil {
				for _, sub := range pg.Items {
					if sub.CustomerID == chosen && liveStatus(sub.Status) {
						match, subID = "active", sub.ID
						st := sub.StartedAt
						sum = subSummary{ID: sub.ID, Status: sub.Status, Price: sub.PriceID, Amount: sub.AmountCents, Currency: sub.Currency, Interval: sub.Interval, Since: &st}
					}
				}
			}
		}
	}
	if match == "invalid" || match == "" {
		stats.Skipped++
		return "skipped", nil, nil, nil
	}
	if match == "none" && dec.NoSubscription == "skip" {
		stats.Skipped++
		return "skipped", nil, nil, nil
	}
	tier := defaultTier
	if t, ok := tierOf[sum.Price]; ok {
		tier = t
	}
	first, last := splitName(r.Name)
	c, err := q.EarnContactUpsertImport(ctx, store.EarnContactUpsertImportParams{ID: newID(), WorkspaceID: ws.ID, Email: r.Email, FirstName: first, LastName: last,
		MailboxProvider: emailaddr.MailboxProvider(emailaddr.Domain(r.Email)), Source: "import:" + imp.Source, RoleAddress: emailaddr.IsRole(r.Email), At: now})
	if err != nil {
		return "", nil, nil, err
	}
	if c.Created {
		stats.New++
	} else {
		stats.Merged++
	}
	cid := c.ID
	// Everyone joins the tier's lists (test imports never touch real lists).
	if imp.Mode == "live" && tier != uuid.Nil {
		for _, l := range listsOf[tier] {
			lid := l
			if err := q.EarnImportMembership(ctx, store.EarnImportMembershipParams{ListID: l, ContactID: cid, Source: "import:" + imp.Source, At: &now}); err != nil {
				return "", nil, nil, err
			}
			if err := q.EarnImportConsent(ctx, store.EarnImportConsentParams{ID: newID(), ContactID: cid, ListID: &lid, Source: consent, Detail: imp.FileName, Note: consent, At: now}); err != nil {
				return "", nil, nil, err
			}
		}
	}
	detail, _ := json.Marshal(map[string]any{"platform": imp.Source, "import_id": imp.ID.String(), "matched_by": r.MatchedBy, "imported_at": now.UTC().Format(time.RFC3339)})
	switch match {
	case "active":
		if existing, err := q.EarnSubscriptionByStripeID(ctx, store.EarnSubscriptionByStripeIDParams{WorkspaceID: ws.ID, Mode: imp.Mode, StripeSubscriptionID: subID}); err == nil {
			stats.Paid++
			return "paid", &cid, nil, s.timeline(ctx, q, ws.ID, cid, &existing.ID, imp.Mode, "imported", now, nil, "", nil, "", imp.CreatedBy,
				map[string]any{"platform": imp.Source, "matchedBy": r.MatchedBy, "existing": true})
		}
		var sub payments.Subscription
		if client != nil && subID != "" {
			if full, err := client.GetSubscription(ctx, subID); err == nil {
				sub = full
			}
		}
		if sub.ID == "" {
			since := now
			if sum.Since != nil {
				since = *sum.Since
			}
			sub = payments.Subscription{ID: subID, CustomerID: customer, Status: sum.Status, PriceID: sum.Price, AmountCents: sum.Amount, Currency: sum.Currency,
				Interval: sum.Interval, StartedAt: since}
		}
		sid := newID()
		pid := priceIDs[sub.PriceID]
		var pidp, tidp *uuid.UUID
		if pid != uuid.Nil {
			pidp = &pid
		}
		if t, ok := tierOf[sub.PriceID]; ok {
			tidp = &t
		}
		status := paidStatus(sub, now)
		var ended *time.Time
		if status == StatusCanceled {
			ended = &now
		}
		if err := q.EarnSubscriptionInsert(ctx, store.EarnSubscriptionInsertParams{ID: sid, WorkspaceID: ws.ID, ContactID: cid, Mode: imp.Mode, Kind: "stripe", TierID: tidp,
			PriceID: pidp, StripePriceID: sub.PriceID, Status: status, Period: periodOf(sub.Interval), AmountCents: sub.AmountCents, Currency: sub.Currency,
			Discount: discountJSON(sub), StartedAt: nonZero(sub.StartedAt, now), TrialEnd: sub.TrialEnd, CurrentPeriodEnd: sub.CurrentPeriodEnd, CancelAt: sub.CancelAt,
			CanceledAt: sub.CanceledAt, EndedAt: ended, StripeCustomerID: customer, StripeSubscriptionID: sub.ID, BillingEmail: r.Email, SourceKind: "import",
			SourceLabel: label, SourceDetail: detail, ImportID: &imp.ID, StripeSyncedAt: &now, At: now}); err != nil {
			return "", nil, nil, err
		}
		if err := s.timeline(ctx, q, ws.ID, cid, &sid, imp.Mode, "imported", now, nil, "", nil, "", imp.CreatedBy,
			map[string]any{"platform": imp.Source, "matchedBy": r.MatchedBy, "consent": consent}); err != nil {
			return "", nil, nil, err
		}
		stats.Paid++
		row, err := q.EarnSubscriptionForUpdate(ctx, sid)
		if err != nil {
			return "", nil, nil, err
		}
		short := s.tierShortName(ctx, q, ws.ID, tidp)
		m := s.memberEmailFor(EmailWelcome, row, ws, short, now)
		return "paid", &cid, &m, nil
	case "ended":
		stats.Free++
		return "free", &cid, nil, nil
	}
	// No subscription in the provider: complimentary until the paid-through
	// date (the default), or a free subscriber.
	if dec.NoSubscription == "free" || r.PaidThrough == nil {
		stats.Free++
		return "free", &cid, nil, nil
	}
	until := r.PaidThrough.Add(24*time.Hour - time.Second)
	if until.Before(now) {
		stats.Free++
		return "free", &cid, nil, nil
	}
	sid := newID()
	tid := tier
	var tidp *uuid.UUID
	if tid != uuid.Nil {
		tidp = &tid
	}
	if err := q.EarnSubscriptionInsert(ctx, store.EarnSubscriptionInsertParams{ID: sid, WorkspaceID: ws.ID, ContactID: cid, Mode: imp.Mode, Kind: "complimentary",
		TierID: tidp, Status: StatusComplimentary, StartedAt: now, SourceKind: "import", SourceLabel: label, SourceDetail: detail, GrantReason: "other",
		GrantNote: msgs.M("import.compNote").Text(defaultLang, i18n.Args{"platform": platformName(imp.Source)}), GrantedBy: imp.CreatedBy, GrantUntil: &until,
		ImportID: &imp.ID, At: now}); err != nil {
		return "", nil, nil, err
	}
	if err := s.timeline(ctx, q, ws.ID, cid, &sid, imp.Mode, "imported", now, nil, "", nil, "", imp.CreatedBy,
		map[string]any{"platform": imp.Source, "complimentary": true, "until": until.UTC().Format("2006-01-02")}); err != nil {
		return "", nil, nil, err
	}
	stats.Complimentary++
	if stats.CompFirst == nil || until.Before(*stats.CompFirst) {
		t := until
		stats.CompFirst = &t
	}
	if stats.CompLast == nil || until.After(*stats.CompLast) {
		t := until
		stats.CompLast = &t
	}
	return "complimentary", &cid, nil, nil
}
