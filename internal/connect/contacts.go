package connect

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/riverqueue/river"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/domains"
	"github.com/hostrepute/reputecast/internal/jobs"
	"github.com/hostrepute/reputecast/internal/providers"
	"github.com/hostrepute/reputecast/internal/providers/manage"
	"github.com/hostrepute/reputecast/internal/providers/state"
	"github.com/hostrepute/reputecast/internal/store"
)

// ConnectImportSource is one importer source of step 6.
type ConnectImportSource struct {
	Key    string `json:"key"`
	Name   string `json:"name"`
	Detail string `json:"detail"`
	Badge  string `json:"badge"`
	Tone   string `json:"tone" enum:"ok,neutral"`
}

// ConnectSuppressions is the provider's own suppression list.
type ConnectSuppressions struct {
	Available  bool   `json:"available"`
	Provider   string `json:"provider"`
	Short      string `json:"short"`
	Total      int    `json:"total"`
	Bounces    int    `json:"bounces"`
	Complaints int    `json:"complaints"`
	Error      string `json:"error,omitempty"`
}

// ConnectContacts is step 6.
type ConnectContacts struct {
	Sources      []ConnectImportSource `json:"sources"`
	Suppressions ConnectSuppressions   `json:"suppressions"`
	// Lists are the workspace's lists; Suggested the ones "Start empty"
	// creates when there are none.
	Lists     []string `json:"lists"`
	Suggested []string `json:"suggested"`
	Pull      bool     `json:"pull"`
}

var importSources = []ConnectImportSource{
	{Key: "brevo", Name: "Brevo", Detail: "API key · lists, attributes, blocklist", Badge: "Keeps consent", Tone: "ok"},
	{Key: "mailchimp", Name: "Mailchimp", Detail: "API key · audiences, tags, merge fields", Badge: "Keeps consent", Tone: "ok"},
	{Key: "mailerlite", Name: "MailerLite", Detail: "API key · groups, fields", Badge: "Keeps consent", Tone: "ok"},
	{Key: "listmonk", Name: "listmonk", Detail: "API or database · lists, attribs", Badge: "Keeps consent", Tone: "ok"},
	{Key: "keila", Name: "Keila", Detail: "API key · segments, data", Badge: "Keeps consent", Tone: "ok"},
	{Key: "csv", Name: "CSV or XLSX", Detail: "Map columns, choose a consent column", Badge: "You confirm consent", Tone: "neutral"},
}

// suggestedLists are the lists "Start empty" proposes from the workspace
// name: its newsletter, product updates and a beta program.
func suggestedLists(wsName string) []string {
	first := "Weekly Digest"
	if !strings.Contains(strings.ToLower(wsName), "newsletter") {
		first = strings.TrimSpace(wsName) + " Newsletter"
	}
	return []string{first, "Product Updates", "Beta Program"}
}

// Contacts returns step 6: importer sources, the provider's suppression
// list (read once and kept on the setup) and the workspace's lists.
func (s *Service) Contacts(ctx context.Context, ws *store.Workspace, rawID string) (ConnectContacts, error) {
	st, err := s.setup(ctx, ws, rawID)
	if err != nil {
		return ConnectContacts{}, err
	}
	p, err := s.provider(ctx, ws, st)
	if err != nil {
		return ConnectContacts{}, err
	}
	d, err := descriptor(st.Adapter)
	if err != nil {
		return ConnectContacts{}, err
	}
	c := copyFor(d)
	state := decodeState(st.State)
	out := ConnectContacts{Sources: importSources, Lists: []string{}, Suggested: suggestedLists(ws.Name), Pull: true}
	if lists, err := s.Q.ConnectListNames(ctx, ws.ID); err == nil {
		out.Lists = lists
	}
	if state.Contacts != nil && state.Contacts.At != nil {
		out.Pull = state.Contacts.Pull
	}
	prev := (*SuppressionPreview)(nil)
	if state.Contacts != nil {
		prev = state.Contacts.Suppressions
	}
	if prev == nil || s.now().Sub(prev.At) > 10*time.Minute && !manage.IsDemo(p) {
		preview := s.previewSuppressions(ctx, p)
		prev = &preview
		_, _ = s.update(ctx, ws, st.ID, "", nil, func(_ *store.ProviderSetup, state *State) error {
			if state.Contacts == nil {
				state.Contacts = &ContactsStep{Pull: true}
			}
			state.Contacts.Suppressions = prev
			return nil
		})
	}
	out.Suppressions = ConnectSuppressions{Available: prev.Available, Provider: p.Name, Short: c.Short, Total: prev.Total, Bounces: prev.Bounces, Complaints: prev.Complaints, Error: prev.Error}
	return out, nil
}

// previewSuppressions counts the provider's own suppression list.
func (s *Service) previewSuppressions(ctx context.Context, p store.Provider) SuppressionPreview {
	out := SuppressionPreview{At: s.now()}
	if manage.IsDemo(p) {
		// The demo's providers have no account: count what their sync pulled.
		row := s.Pool.QueryRow(ctx, `SELECT count(*) FILTER (WHERE reason = 'hard_bounce'), count(*) FILTER (WHERE reason = 'complaint')
			FROM suppressions WHERE source_provider_id = $1`, p.ID)
		_ = row.Scan(&out.Bounces, &out.Complaints)
		out.Total = out.Bounces + out.Complaints
		out.Available = state.Caps(p).SuppressionSync
		return out
	}
	adapter, err := s.Factory.Open(ctx, p)
	if err != nil {
		out.Error = err.Error()
		return out
	}
	syncer, ok := adapter.(providers.SuppressionSyncer)
	if !ok || !state.Caps(p).SuppressionSync {
		return out
	}
	out.Available = true
	cctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	list, err := syncer.ListSuppressions(cctx, time.Time{})
	cancel()
	if err != nil {
		out.Error = callMessage(err)
		return out
	}
	for _, sp := range list {
		switch sp.Reason {
		case "complaint":
			out.Complaints++
		default:
			out.Bounces++
		}
	}
	out.Total = len(list)
	return out
}

// ConnectFinishInput ends the setup.
type ConnectFinishInput struct {
	// PullSuppressions imports the provider's own suppression list into the
	// shared one.
	PullSuppressions bool `json:"pullSuppressions"`
	// StartEmpty creates the suggested lists when the workspace has none.
	StartEmpty bool `json:"startEmpty"`
	// Source is the importer source the person opened, if any.
	Source string `json:"source,omitempty" enum:"brevo,mailchimp,mailerlite,listmonk,keila,csv"`
}

// Finish ends the setup: the suppression pull is queued, the suggested
// lists are created when asked and the summary becomes available.
func (s *Service) Finish(ctx context.Context, ws *store.Workspace, rawID string, in ConnectFinishInput) (ConnectSetup, error) {
	cur, err := s.setup(ctx, ws, rawID)
	if err != nil {
		return ConnectSetup{}, err
	}
	p, err := s.provider(ctx, ws, cur)
	if err != nil {
		return ConnectSetup{}, err
	}
	if cur.Status == "abandoned" {
		return ConnectSetup{}, api.Conflict("This setup was cancelled.")
	}
	lists, err := s.Q.ConnectListNames(ctx, ws.ID)
	if err != nil {
		return ConnectSetup{}, err
	}
	now := s.now()
	created := []string{}
	var st store.ProviderSetup
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if in.StartEmpty && len(lists) == 0 {
			for _, name := range suggestedLists(ws.Name) {
				id := newID()
				if err := q.ConnectCreateList(ctx, store.ConnectCreateListParams{ID: id, WorkspaceID: ws.ID, Name: name, CreatedAt: now}); err != nil {
					return err
				}
				if err := s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "list.create", TargetType: "list", TargetID: id.String(), TargetName: name,
					Metadata: map[string]any{"source": "provider_setup"}}); err != nil {
					return err
				}
				created = append(created, name)
			}
		}
		mode := "push"
		if in.PullSuppressions {
			mode = "two_way"
		}
		if state.Caps(p).SuppressionSync {
			if err := q.ConnectSuppressionSyncMode(ctx, store.ConnectSuppressionSyncModeParams{ProviderID: p.ID, Mode: mode}); err != nil {
				return err
			}
		}
		locked, err := q.SetupLock(ctx, cur.ID)
		if err != nil {
			return err
		}
		stt := decodeState(locked.State)
		if stt.Contacts == nil {
			stt.Contacts = &ContactsStep{}
		}
		stt.Contacts.Pull, stt.Contacts.At, stt.Contacts.ListsCreated, stt.Contacts.Source = in.PullSuppressions, &now, created, in.Source
		stt.Contacts.Choice = "empty"
		if in.Source != "" {
			stt.Contacts.Choice = "import"
		}
		locked.Step, locked.Status = "done", "finished"
		if locked.FinishedAt == nil {
			locked.FinishedAt = &now
		}
		if st, err = s.save(ctx, q, locked, stt); err != nil {
			return err
		}
		if in.PullSuppressions && state.Caps(p).SuppressionSync {
			if c := s.jobs(); c != nil {
				if _, err := c.InsertTx(ctx, tx, PullArgs{SetupID: cur.ID}, nil); err != nil {
					return err
				}
			}
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "provider.setup_finish", TargetType: "provider", TargetID: p.ID.String(), TargetName: p.Name,
			After: map[string]any{"pull_suppressions": in.PullSuppressions, "lists_created": created, "source": in.Source}})
	})
	if err != nil {
		return ConnectSetup{}, err
	}
	if s.OnRoutingChange != nil {
		s.OnRoutingChange()
	}
	return s.View(ctx, ws, st)
}

// PullArgs imports a provider's suppression list after the setup.
type PullArgs struct {
	SetupID uuid.UUID `json:"setup_id"`
}

// Kind implements river.JobArgs.
func (PullArgs) Kind() string { return "connect.pull_suppressions" }

// InsertOpts implements river.JobArgsWithInsertOpts.
func (PullArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{Queue: jobs.QueueDefault, MaxAttempts: 5}
}

// PullWorker runs PullArgs through the suppression sync (internal/suppressions).
type PullWorker struct {
	river.WorkerDefaults[PullArgs]
	S *Service
}

// Timeout bounds one pull.
func (w *PullWorker) Timeout(*river.Job[PullArgs]) time.Duration { return 10 * time.Minute }

// Work implements river.Worker.
func (w *PullWorker) Work(ctx context.Context, job *river.Job[PullArgs]) error {
	s := w.S
	st, err := s.Q.SetupGetAny(ctx, job.Args.SetupID)
	if err != nil || st.ProviderID == nil {
		return err
	}
	p, err := s.Q.ProvGetAny(ctx, *st.ProviderID)
	if err != nil {
		return err
	}
	var before int64
	_ = s.Pool.QueryRow(ctx, `SELECT count(*) FROM suppressions WHERE workspace_id = $1`, p.WorkspaceID).Scan(&before)
	if !manage.IsDemo(p) && s.Suppressions != nil {
		if err := s.Suppressions.SyncProvider(ctx, p); err != nil {
			return err
		}
	}
	var after int64
	_ = s.Pool.QueryRow(ctx, `SELECT count(*) FROM suppressions WHERE workspace_id = $1`, p.WorkspaceID).Scan(&after)
	imported := after - before
	return db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		locked, err := q.SetupLock(ctx, st.ID)
		if err != nil {
			return err
		}
		stt := decodeState(locked.State)
		if stt.Contacts == nil {
			stt.Contacts = &ContactsStep{}
		}
		stt.Contacts.Imported = &imported
		_, err = s.save(ctx, q, locked, stt)
		return err
	})
}

// ConnectDoneRow is one line of "What was configured".
type ConnectDoneRow struct {
	Status string `json:"status" enum:"done,skipped,failed,pending"`
	Lead   string `json:"lead"`
	// Text follows the bold lead; `backticks` mark code.
	Text string `json:"text"`
	// ImportLink ends the line with "Import now".
	ImportLink bool `json:"importLink"`
}

// ConnectDone is the summary.
type ConnectDone struct {
	Title string `json:"title"`
	// Minutes the setup took.
	Minutes int              `json:"minutes"`
	Domain  string           `json:"domain"`
	Rows    []ConnectDoneRow `json:"rows"`
	// HostRepute is connected already (the recommendation becomes "Open").
	HostRepute bool `json:"hostRepute"`
}

// doneView builds the summary from the setup's state.
func (s *Service) doneView(ctx context.Context, ws *store.Workspace, st store.ProviderSetup, d providers.Descriptor, c adapterCopy, stt State, prov *store.Provider) ConnectDone {
	out := ConnectDone{Title: c.Display + " is connected", Rows: []ConnectDoneRow{}}
	if st.FinishedAt != nil {
		out.Minutes = max(1, int(st.FinishedAt.Sub(st.CreatedAt).Round(time.Minute).Minutes()))
	}
	if stt.Domain != nil && !stt.Domain.Skipped {
		out.Domain = stt.Domain.Name
	}
	var cfg map[string]any
	if prov != nil {
		_ = json.Unmarshal(prov.Config, &cfg)
	}
	cn := stt.Connection
	loc := state.Location(ws.Timezone)
	switch flowOf(d) {
	case "ses":
		via := "IAM access keys"
		if cn.Access == "role" {
			role := cn.RoleARN
			if i := strings.LastIndex(role, "/"); i >= 0 {
				role = role[i+1:]
			}
			via = "CloudFormation role `" + role + "`"
		}
		region := cn.Region
		quota := ""
		for _, r := range cn.Regions {
			if r.Region == region && r.Production {
				quota = fmt.Sprintf(" (production, %s per day, %s per second)", thousands(int(r.Max24h)), trimFloat(r.MaxRate))
			}
		}
		if quota == "" && strings.Contains(cn.Summary, "sandbox") {
			quota = " (sandbox)"
		}
		acct := ""
		if cn.AccountID != "" {
			acct = ", account " + cn.AccountID
		}
		out.Rows = append(out.Rows, ConnectDoneRow{Status: "done", Lead: "AWS access", Text: " via " + via + acct + ", region " + region + quota})
		events := ConnectDoneRow{Status: "pending", Lead: "Events"}
		topic := stringOf(cfg["topic_arn"])
		if arns, ok := cfg["topic_arns"].([]any); ok && topic == "" && len(arns) > 0 {
			topic = stringOf(arns[len(arns)-1])
		}
		if i := strings.LastIndex(topic, ":"); i >= 0 {
			topic = topic[i+1:]
		}
		if topic != "" && prov != nil {
			events.Status = "done"
			events.Text = ": SNS topic `" + topic + "` subscribed to `" + s.webhookURL(*prov, "") + "`"
			if prov.LastEventAt != nil {
				events.Text += "; first delivery event received at " + prov.LastEventAt.In(loc).Format("15:04")
			}
			if t := s.firstDelivery(ctx, stt); t != nil {
				events.Text = ": SNS topic `" + topic + "` subscribed to `" + s.webhookURL(*prov, "") + "`; first delivery event received at " + t.In(loc).Format("15:04")
			}
		} else {
			events.Text = ": provisioning didn't finish; retry it from the connection step"
		}
		out.Rows = append(out.Rows, events)
		sets := []string{}
		if m, ok := cfg["configuration_sets"].(map[string]any); ok {
			for _, k := range []string{"broadcast", "automation", "system"} {
				if v := stringOf(m[k]); v != "" {
					sets = append(sets, "`"+v+"`")
				}
			}
		} else if v := stringOf(cfg["configuration_set"]); v != "" {
			sets = append(sets, "`"+v+"`")
		}
		if len(sets) > 0 {
			out.Rows = append(out.Rows, ConnectDoneRow{Status: "done", Lead: "Configuration sets", Text: " " + strings.Join(sets, ", ") + ", provider-side tracking off"})
		}
	case "smtp":
		text := "`" + stringOf(cfg["host"]) + "`"
		if port := stringOf(cfg["port"]); port != "" {
			text += ":" + port
		}
		tls := map[string]string{"starttls": "STARTTLS", "tls": "implicit TLS", "none": "no TLS"}[stringOf(cfg["tls"])]
		if tls != "" {
			text += " with " + tls
		}
		if r := stringOf(cfg["max_rate"]); r != "" {
			text += ", " + r
		}
		if dc := stringOf(cfg["daily_cap"]); dc != "" {
			text += ", " + dc + " per day"
		}
		out.Rows = append(out.Rows, ConnectDoneRow{Status: "done", Lead: "SMTP server", Text: " " + text})
		mb := ConnectDoneRow{Status: "pending", Lead: "Bounces"}
		if m, ok := cfg["bounce_mailbox"].(map[string]any); ok && cn.Mailbox != nil && cn.Mailbox.OK {
			mb.Status = "done"
			mb.Text = ": bounce mailbox `" + stringOf(m["address"]) + "` over " + strings.ToUpper(stringOf(m["protocol"]))
		} else {
			mb.Text = ": no bounce mailbox yet"
		}
		out.Rows = append(out.Rows, mb)
	default:
		out.Rows = append(out.Rows, ConnectDoneRow{Status: "done", Lead: d.Name + " access", Text: " checked with a read-only call" + accountSuffix(cn)})
		ev := ConnectDoneRow{Status: "pending", Lead: "Events"}
		if w := cn.Webhook; w != nil {
			switch {
			case w.Mode == "polling":
				ev.Status, ev.Text = "done", ": polling "+d.Name+"'s events API every minute"
			case w.Registration != nil && w.Registration.Status == "done":
				ev.Status, ev.Text = "done", ": webhooks registered at `"+w.URL+"`"
			case w.Registration != nil && w.Registration.Status == "manual":
				ev.Text = ": add `" + w.URL + "` as a webhook in " + d.Name
			default:
				ev.Text = ": the webhook endpoint wasn't reachable yet"
			}
		}
		out.Rows = append(out.Rows, ev)
	}
	dom := ConnectDoneRow{Status: "skipped", Lead: "Domain", Text: ": skipped; verify it later in Settings › Senders"}
	if stt.Domain != nil && !stt.Domain.Skipped && stt.Domain.DomainID != "" && prov != nil {
		dom = s.domainRow(ctx, stt.Domain, *prov)
	}
	out.Rows = append(out.Rows, dom)
	str := ConnectDoneRow{Status: "skipped", Lead: "Streams", Text: ": not routed yet; " + c.Short + " carries nothing until a route uses it"}
	if stt.Streams != nil && stt.Streams.Saved {
		groups := map[string][]string{}
		order := []string{}
		for _, k := range []string{"broadcast", "automation", "system"} {
			ch := stt.Streams.Streams[k]
			if !ch.Enabled {
				continue
			}
			role := ch.Role
			if role == "" {
				role = "primary"
			}
			if _, ok := groups[role]; !ok {
				order = append(order, role)
			}
			groups[role] = append(groups[role], streamName(k))
		}
		if len(order) > 0 {
			parts := []string{}
			for _, r := range order {
				word := map[string]string{"primary": "primary for", "split": "a weighted share of", "fallback": "the fallback for"}[r]
				parts = append(parts, word+" "+joinAnd(groups[r]))
			}
			str = ConnectDoneRow{Status: "done", Lead: "Streams", Text: ": " + c.Short + " is " + strings.Join(parts, "; ")}
		}
	}
	out.Rows = append(out.Rows, str)
	if cs := stt.Contacts; cs != nil && cs.Suppressions != nil && cs.Suppressions.Available {
		if cs.Pull {
			n := cs.Suppressions.Total
			if cs.Imported != nil && int(*cs.Imported) > 0 {
				n = int(*cs.Imported)
			}
			out.Rows = append(out.Rows, ConnectDoneRow{Status: "done", Lead: "Suppressions",
				Text: fmt.Sprintf(": %s addresses imported from the %s account-level list into the shared suppression list", thousands(n), c.Short)})
		} else {
			out.Rows = append(out.Rows, ConnectDoneRow{Status: "skipped", Lead: "Suppressions", Text: ": not imported; the shared list is pushed to " + c.Short})
		}
	}
	if cs := stt.Contacts; cs != nil {
		switch {
		case cs.Choice == "import":
			out.Rows = append(out.Rows, ConnectDoneRow{Status: "done", Lead: "Contacts", Text: ": import from " + sourceName(cs.Source) + " started."})
		case len(cs.ListsCreated) > 0:
			out.Rows = append(out.Rows, ConnectDoneRow{Status: "skipped", Lead: "Contacts", Text: fmt.Sprintf(": started empty with %s lists.", numberWord(len(cs.ListsCreated))), ImportLink: true})
		default:
			out.Rows = append(out.Rows, ConnectDoneRow{Status: "skipped", Lead: "Contacts", Text: ": kept the workspace's lists; nothing imported.", ImportLink: true})
		}
	}
	var connected bool
	_ = s.Pool.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM hostrepute_connections WHERE workspace_id = $1 AND status = 'connected')`, ws.ID).Scan(&connected)
	out.HostRepute = connected
	return out
}

func accountSuffix(cn Connection) string {
	if cn.Account == "" {
		return ""
	}
	return "; " + strings.ToLower(cn.Account[:1]) + cn.Account[1:]
}

func sourceName(key string) string {
	for _, s := range importSources {
		if s.Key == key {
			return s.Name
		}
	}
	return key
}

func numberWord(n int) string {
	words := []string{"no", "one", "two", "three", "four", "five", "six"}
	if n < len(words) {
		return words[n]
	}
	return fmt.Sprint(n)
}

// firstDelivery is when the test message's delivery event arrived.
func (s *Service) firstDelivery(ctx context.Context, stt State) *time.Time {
	if stt.Test == nil {
		return nil
	}
	id, err := uuid.Parse(stt.Test.MessageID)
	if err != nil {
		return nil
	}
	evs, err := s.Q.ConnectMessageEvents(ctx, &id)
	if err != nil {
		return nil
	}
	for _, e := range evs {
		if e.Type == "delivered" {
			t := e.ReceivedAt
			return &t
		}
	}
	return nil
}

// domainRow summarizes the domain records of the provider.
func (s *Service) domainRow(ctx context.Context, ds *DomainStep, p store.Provider) ConnectDoneRow {
	id, err := uuid.Parse(ds.DomainID)
	if err != nil {
		return ConnectDoneRow{Status: "skipped", Lead: "Domain", Text: " " + ds.Name}
	}
	d, err := s.Q.DomGetAny(ctx, id)
	if err != nil {
		return ConnectDoneRow{Status: "skipped", Lead: "Domain", Text: " " + ds.Name}
	}
	recs := domains.DecodeRecords(d.DnsRecords)
	var det domains.AuthDetail
	_ = json.Unmarshal(d.AuthDetail, &det)
	pid := p.ID.String()
	dkimN, dkimType, mailFrom := 0, "", ""
	verified, total := 0, 0
	for _, r := range recs {
		if r.ProviderID != "" && r.ProviderID != pid {
			continue
		}
		total++
		if r.Status == domains.StatusVerified {
			verified++
		}
		switch r.Purpose {
		case domains.PurposeDKIM:
			dkimN++
			dkimType = r.Type
		case domains.PurposeReturnPath:
			mailFrom = r.Name
		}
	}
	parts := []string{}
	if dkimN > 0 {
		word := map[string]string{"CNAME": "CNAMEs", "TXT": "TXT records"}[dkimType]
		if dkimN == 1 {
			word = map[string]string{"CNAME": "CNAME", "TXT": "TXT record"}[dkimType]
		}
		parts = append(parts, fmt.Sprintf("DKIM (%d %s)", dkimN, word))
	}
	if mailFrom != "" {
		parts = append(parts, "MAIL FROM "+mailFrom)
	}
	policy := det.DMARCPolicy
	tail := "SPF and DMARC"
	if policy != "" {
		tail += " p=" + policy
	}
	parts = append(parts, tail)
	text := " " + d.Domain + ": " + strings.Join(parts, ", ")
	status := "done"
	if verified == total {
		text += " all verified"
	} else {
		status = "pending"
		text += fmt.Sprintf(", %d of %d records verified so far", verified, total)
	}
	return ConnectDoneRow{Status: status, Lead: "Domain", Text: text}
}
