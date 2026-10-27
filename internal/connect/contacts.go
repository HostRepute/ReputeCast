package connect

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/riverqueue/river"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/domains"
	"github.com/hostrepute/reputecast/internal/i18n"
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

// importSource is an importer source of step 6: its name (empty: a
// message), what an import brings, and whether it keeps consent.
type importSource struct {
	Key, Name string
	Detail    i18n.Msg
	Consent   bool
}

var importSources = []importSource{
	{Key: "brevo", Name: "Brevo", Detail: msgs.M("source.brevo"), Consent: true},
	{Key: "mailchimp", Name: "Mailchimp", Detail: msgs.M("source.mailchimp"), Consent: true},
	{Key: "mailerlite", Name: "MailerLite", Detail: msgs.M("source.mailerlite"), Consent: true},
	{Key: "listmonk", Name: "listmonk", Detail: msgs.M("source.listmonk"), Consent: true},
	{Key: "keila", Name: "Keila", Detail: msgs.M("source.keila"), Consent: true},
	{Key: "csv", Detail: msgs.M("source.csv")},
}

// importSourcesIn lists the importer sources for a reader.
func importSourcesIn(l i18n.Lang) []ConnectImportSource {
	out := make([]ConnectImportSource, 0, len(importSources))
	for _, src := range importSources {
		s := ConnectImportSource{Key: src.Key, Name: sourceName(l, src.Key), Detail: src.Detail.Text(l, nil), Badge: mSourceYouConfirm.Text(l, nil), Tone: "neutral"}
		if src.Consent {
			s.Badge, s.Tone = mSourceKeepsConsent.Text(l, nil), "ok"
		}
		out = append(out, s)
	}
	return out
}

// suggestedLists are the lists "Start empty" proposes from the workspace
// name: its newsletter, product updates and a beta program, named in the
// language of whoever creates them.
func suggestedLists(l i18n.Lang, wsName string) []string {
	first := mListsWeeklyDigest.Text(l, nil)
	if !strings.Contains(strings.ToLower(wsName), "newsletter") {
		first = mListsNewsletter.Text(l, i18n.Args{"workspace": strings.TrimSpace(wsName)})
	}
	return []string{first, mListsProductUpdates.Text(l, nil), mListsBeta.Text(l, nil)}
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
	d, err := descriptor(ctx, st.Adapter)
	if err != nil {
		return ConnectContacts{}, err
	}
	l := i18n.LangFrom(ctx)
	c := copyFor(l, d)
	state := decodeState(st.State)
	out := ConnectContacts{Sources: importSourcesIn(l), Lists: []string{}, Suggested: suggestedLists(l, ws.Name), Pull: true}
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
	out.Suppressions = ConnectSuppressions{Available: prev.Available, Provider: p.Name, Short: c.Short, Total: prev.Total, Bounces: prev.Bounces, Complaints: prev.Complaints,
		Error: textIn(prev.Error, prev.ErrorText, l)}
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
		out.Error, out.ErrorText = callMessage(err), callText(err)
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
		return ConnectSetup{}, conflict(ctx, mProblemCancelledShort, nil)
	}
	// An SES setup finishes only once a real event came back (PR-3).
	if check, _ := eventsCheckOf(decodeState(cur.State).Connection.Provisioning); check != "verified" && check != "none" {
		return ConnectSetup{}, conflict(ctx, mProblemEventsUnverified, i18n.Args{"name": p.Name})
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
			for _, name := range suggestedLists(i18n.LangFrom(ctx), ws.Name) {
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
	// Status verified: the events arrived (PR-3).
	Status string `json:"status" enum:"done,verified,skipped,failed,pending"`
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
func (s *Service) doneView(ctx context.Context, l i18n.Lang, ws *store.Workspace, st store.ProviderSetup, d providers.Descriptor, c adapterCopy, stt State,
	prov *store.Provider) ConnectDone {
	out := ConnectDone{Title: mDoneTitle.Text(l, i18n.Args{"name": c.Display}), Rows: []ConnectDoneRow{}}
	// A row is its bold lead, then its text after a separator (": " or " ").
	row := func(status string, lead i18n.Msg, leadArgs i18n.Args, sep string, text i18n.Msg, args i18n.Args) ConnectDoneRow {
		return ConnectDoneRow{Status: status, Lead: lead.Text(l, leadArgs), Text: sep + text.Text(l, args)}
	}
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
		args := i18n.Args{"via": "keys", "region": cn.Region, "hasAccount": yesNo(cn.AccountID != ""), "account": cn.AccountID, "quota": "none",
			"hasKey": yesNo(cn.Settings["access_key_hint"] != ""), "key": cn.Settings["access_key_hint"], "hasUser": yesNo(cn.Settings["iam_user"] != ""),
			"user": cn.Settings["iam_user"]}
		if cn.Access == "role" {
			role := cn.RoleARN
			if i := strings.LastIndex(role, "/"); i >= 0 {
				role = role[i+1:]
			}
			args["via"], args["role"] = "role", role
		}
		for _, r := range cn.Regions {
			if r.Region == cn.Region && r.Production {
				args["quota"], args["max"], args["rate"] = "production", r.Max24h, r.MaxRate
			}
		}
		if args["quota"] == "none" && strings.Contains(cn.Summary, "sandbox") {
			args["quota"] = "sandbox"
		}
		out.Rows = append(out.Rows, row("done", mDoneAWSAccess, nil, " ", mDoneAWSAccessText, args))
		events := ConnectDoneRow{Status: "pending", Lead: mDoneEvents.Text(l, nil)}
		topic := stringOf(cfg["topic_arn"])
		if arns, ok := cfg["topic_arns"].([]any); ok && topic == "" && len(arns) > 0 {
			topic = stringOf(arns[len(arns)-1])
		}
		if i := strings.LastIndex(topic, ":"); i >= 0 {
			topic = topic[i+1:]
		}
		if topic != "" && prov != nil {
			events.Status = "done"
			queue := cn.Events == providers.EventsQueue
			args := i18n.Args{"topic": topic, "url": s.webhookURL(*prov, ""), "hasFirst": "no", "via": viaKey(queue), "queue": queueName(cfg)}
			if prov.LastEventAt != nil {
				args["hasFirst"], args["at"] = "yes", prov.LastEventAt.In(loc)
			}
			if t := s.firstDelivery(ctx, stt); t != nil {
				args["hasFirst"], args["at"] = "yes", t.In(loc)
			}
			// The event check's delivery from the mailbox simulator verified
			// the events (PR-3).
			if check, at := eventsCheckOf(cn.Provisioning); check == "verified" && at != nil {
				events.Status = "verified"
				args["at"] = at.In(loc)
				events.Text = ": " + mDoneEventsVerified.Text(l, args)
			} else {
				events.Text = ": " + mDoneEventsSNS.Text(l, args)
			}
		} else {
			events.Text = ": " + mDoneEventsUnfinished.Text(l, nil)
		}
		out.Rows = append(out.Rows, events)
		// The workspace's own SES tenant (PR-11).
		if tenant := stringOf(cfg["tenant"]); tenant != "" {
			policy := tenantPolicyName(stringOf(cfg["tenant_reputation_policy"]))
			out.Rows = append(out.Rows, row("done", mDoneTenant, nil, " ", mDoneTenantText, i18n.Args{"tenant": tenant, "hasPolicy": yesNo(policy != ""), "policy": policy}))
		}
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
			out.Rows = append(out.Rows, row("done", mDoneConfigSets, nil, " ", mDoneConfigSetsText, i18n.Args{"sets": strings.Join(sets, ", ")}))
		}
	case "smtp":
		args := i18n.Args{"host": stringOf(cfg["host"]), "hasPort": "no", "tls": stringOf(cfg["tls"]), "hasRate": "no", "hasCap": "no"}
		if port := stringOf(cfg["port"]); port != "" {
			args["hasPort"], args["port"] = "yes", port
		}
		if r := stringOf(cfg["max_rate"]); r != "" {
			args["hasRate"], args["rate"] = "yes", r
		}
		if dc := stringOf(cfg["daily_cap"]); dc != "" {
			args["hasCap"], args["cap"] = "yes", dc
		}
		out.Rows = append(out.Rows, row("done", mDoneSMTP, nil, " ", mDoneSMTPText, args))
		mb := row("pending", mDoneBounces, nil, ": ", mDoneNoBounceMailbox, nil)
		if m, ok := cfg["bounce_mailbox"].(map[string]any); ok && cn.Mailbox != nil && cn.Mailbox.OK {
			mb = row("done", mDoneBounces, nil, ": ", mDoneBouncesText, i18n.Args{"address": stringOf(m["address"]), "protocol": strings.ToUpper(stringOf(m["protocol"]))})
		}
		out.Rows = append(out.Rows, mb)
	default:
		args := i18n.Args{"hasAccount": "no"}
		if a := accountWords(l, cn); a != "" {
			args["hasAccount"], args["account"] = "yes", a
		}
		out.Rows = append(out.Rows, row("done", mDoneAccess, i18n.Args{"name": d.Name}, " ", mDoneAccessText, args))
		ev := ConnectDoneRow{Status: "pending", Lead: mDoneEvents.Text(l, nil)}
		if w := cn.Webhook; w != nil {
			switch {
			case w.Mode == "polling":
				ev.Status, ev.Text = "done", ": "+mDoneEventsPolling.Text(l, i18n.Args{"name": d.Name})
			case w.Registration != nil && w.Registration.Status == "done":
				ev.Status, ev.Text = "done", ": "+mDoneEventsRegistered.Text(l, i18n.Args{"url": w.URL})
			case w.Registration != nil && w.Registration.Status == "manual":
				ev.Text = ": " + mDoneEventsManual.Text(l, i18n.Args{"url": w.URL, "name": d.Name})
			default:
				ev.Text = ": " + mDoneEventsUnreachable.Text(l, nil)
			}
		}
		out.Rows = append(out.Rows, ev)
	}
	dom := row("skipped", mDoneDomain, nil, ": ", mDoneDomainSkipped, nil)
	if stt.Domain != nil && !stt.Domain.Skipped && stt.Domain.DomainID != "" && prov != nil {
		dom = s.domainRow(ctx, l, stt.Domain, *prov)
	}
	out.Rows = append(out.Rows, dom)
	str := row("skipped", mDoneStreams, nil, ": ", mDoneStreamsNone, i18n.Args{"short": c.Short})
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
			groups[role] = append(groups[role], streamName(l, k))
		}
		if len(order) > 0 {
			parts := []string{}
			for _, r := range order {
				parts = append(parts, mDoneStreamsRole.Text(l, i18n.Args{"role": r, "streams": i18n.AndList(l.Locale, groups[r])}))
			}
			str = row("done", mDoneStreams, nil, ": ", mDoneStreamsText, i18n.Args{"short": c.Short, "roles": strings.Join(parts, "; ")})
		}
	}
	out.Rows = append(out.Rows, str)
	if cs := stt.Contacts; cs != nil && cs.Suppressions != nil && cs.Suppressions.Available {
		if cs.Pull {
			n := cs.Suppressions.Total
			if cs.Imported != nil && int(*cs.Imported) > 0 {
				n = int(*cs.Imported)
			}
			// With an SES tenant, what bounces and complaints add from now on
			// lands on the workspace's own list (PR-1, PR-11).
			out.Rows = append(out.Rows, row("done", mDoneSuppressions, nil, ": ", mDoneSuppressionsImported, i18n.Args{"count": n, "short": c.Short,
				"tenant": yesNo(stringOf(cfg["tenant"]) != "")}))
		} else {
			out.Rows = append(out.Rows, row("skipped", mDoneSuppressions, nil, ": ", mDoneSuppressionsPushed, i18n.Args{"short": c.Short}))
		}
	}
	if cs := stt.Contacts; cs != nil {
		var r ConnectDoneRow
		switch {
		case cs.Choice == "import":
			r = row("done", mDoneContacts, nil, ": ", mDoneContactsImport, i18n.Args{"source": sourceName(l, cs.Source)})
		case len(cs.ListsCreated) > 0:
			r = row("skipped", mDoneContacts, nil, ": ", mDoneContactsEmpty, i18n.Args{"count": len(cs.ListsCreated)})
			r.ImportLink = true
		default:
			r = row("skipped", mDoneContacts, nil, ": ", mDoneContactsKept, nil)
			r.ImportLink = true
		}
		out.Rows = append(out.Rows, r)
	}
	var connected bool
	_ = s.Pool.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM hostrepute_connections WHERE workspace_id = $1 AND status = 'connected')`, ws.ID).Scan(&connected)
	out.HostRepute = connected
	return out
}

// accountWords is the account's standing inside a sentence ("production ·
// no enforcement").
func accountWords(l i18n.Lang, cn Connection) string {
	a := textIn(cn.Account, cn.AccountText, l)
	switch {
	case a == "":
		return ""
	case english(l):
		return strings.ToLower(a[:1]) + a[1:]
	}
	return lowerFirst(a)
}

func sourceName(l i18n.Lang, key string) string {
	for _, s := range importSources {
		if s.Key == key {
			if s.Name == "" {
				return mSourceCSVName.Text(l, nil)
			}
			return s.Name
		}
	}
	return key
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
func (s *Service) domainRow(ctx context.Context, l i18n.Lang, ds *DomainStep, p store.Provider) ConnectDoneRow {
	lead := mDoneDomain.Text(l, nil)
	id, err := uuid.Parse(ds.DomainID)
	if err != nil {
		return ConnectDoneRow{Status: "skipped", Lead: lead, Text: " " + ds.Name}
	}
	d, err := s.Q.DomGetAny(ctx, id)
	if err != nil {
		return ConnectDoneRow{Status: "skipped", Lead: lead, Text: " " + ds.Name}
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
	// "news.acme.io: DKIM (3 CNAMEs), MAIL FROM bounce.news.acme.io, SPF and
	// DMARC p=quarantine all verified"
	args := i18n.Args{"domain": d.Domain, "hasDKIM": yesNo(dkimN > 0), "dkim": dkimN, "type": dkimType, "hasMailFrom": yesNo(mailFrom != ""), "mailFrom": mailFrom,
		"hasPolicy": yesNo(det.DMARCPolicy != ""), "policy": det.DMARCPolicy, "verifiedAll": yesNo(verified == total), "verified": verified, "total": total}
	status := "done"
	if verified != total {
		status = "pending"
	}
	return ConnectDoneRow{Status: status, Lead: lead, Text: " " + mDoneDomainText.Text(l, args)}
}

// queueName is the SQS queue of an SES connection whose events come
// through one ("reputecast-events").
func queueName(cfg map[string]any) string {
	u := stringOf(cfg["queue_url"])
	if i := strings.LastIndex(u, "/"); i >= 0 {
		return u[i+1:]
	}
	return u
}

// tenantPolicyName is a reputation policy's name as people read it
// ("standard" → "Standard").
func tenantPolicyName(p string) string {
	if p == "" {
		return ""
	}
	if i := strings.LastIndexAny(p, "/:"); i >= 0 {
		p = p[i+1:]
	}
	return strings.ToUpper(p[:1]) + p[1:]
}
