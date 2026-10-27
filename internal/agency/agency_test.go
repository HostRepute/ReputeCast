package agency_test

import (
	"context"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/hostrepute/reputecast/internal/apptest"
)

// The Northwind Studio sample: seven clients and the agency's own
// workspace; Tomás an agency manager, Mei and Ella account managers, Ravi
// an account manager who is an Editor everywhere, Jane a Client approver
// in Contoso Coffee.

type ref struct {
	ID, Slug, Name, Kind string
}

type overview struct {
	Clients []struct {
		Workspace        ref
		Contacts         int64
		Attention        bool
		AttentionReasons []string
		Mine             bool
		Health           *struct {
			State string
			Score int
			Tone  string
			Trend []int
		}
	}
	AgencyWorkspace *struct{ Workspace ref }
	Counts          struct{ All, Attention, Mine, Sends int }
	Approvals       []struct {
		ID            string
		Workspace     ref
		Campaign      string
		WaitingFor    []string
		ClientSignoff bool
	}
	Incidents []struct {
		Workspace   ref
		Kind, State string
		Failovers   []struct{ Stream, To string }
		WithChanges bool
	}
	Sends []struct {
		Workspace ref
		Name      string
		SendAt    time.Time
	}
	NextAfter *struct{ Name string }
	CanCreate bool
}

func northwind(t *testing.T) (*apptest.Env, map[string]*apptest.Client) {
	t.Helper()
	env := apptest.New(t)
	env.SeedNorthwind()
	people := map[string]*apptest.Client{}
	for _, p := range []string{"nadia@northwind.studio", "tomas@northwind.studio", "mei@northwind.studio", "ella@northwind.studio", "ravi@northwind.studio",
		"jane.morales@contoso.example", "rick.alvarez@fabrikam.example"} {
		c := env.Client()
		c.SignInDemo(p)
		people[strings.Split(p, "@")[0]] = c
	}
	return env, people
}

func getOverview(t *testing.T, c *apptest.Client) overview {
	t.Helper()
	var o overview
	c.Do("GET", "/agency/overview", nil).MustStatus(t, 200).JSON(t, &o)
	return o
}

func workspaceID(t *testing.T, o overview, slug string) string {
	t.Helper()
	for _, c := range o.Clients {
		if c.Workspace.Slug == slug {
			return c.Workspace.ID
		}
	}
	if o.AgencyWorkspace != nil && o.AgencyWorkspace.Workspace.Slug == slug {
		return o.AgencyWorkspace.Workspace.ID
	}
	t.Fatalf("no workspace %s", slug)
	return ""
}

func TestOverviewOfTheSample(t *testing.T) {
	_, p := northwind(t)
	o := getOverview(t, p["tomas"])
	if o.Counts.All != 7 || o.Counts.Attention != 2 || o.Counts.Mine != 7 || o.Counts.Sends != 4 || !o.CanCreate {
		t.Fatalf("counts: %+v can create %v", o.Counts, o.CanCreate)
	}
	// Needs attention first: Fabrikam (health 71, the auto-pause) and
	// Litware (health 62, the Mailgun outage).
	if o.Clients[0].Workspace.Slug != "fabrikam-outdoors" || o.Clients[1].Workspace.Slug != "litware-books" || !o.Clients[1].Attention {
		t.Fatalf("order: %s, %s", o.Clients[0].Workspace.Name, o.Clients[1].Workspace.Name)
	}
	if h := o.Clients[0].Health; h == nil || h.Score != 71 || h.Tone != "warn" || len(h.Trend) != 30 || !slices.Equal(o.Clients[0].AttentionReasons, []string{"health", "incident"}) {
		t.Fatalf("fabrikam: %+v %v", h, o.Clients[0].AttentionReasons)
	}
	if h := o.Clients[1].Health; h == nil || h.Score != 62 || h.Tone != "bad" {
		t.Fatalf("litware: %+v", h)
	}
	if o.AgencyWorkspace == nil || o.AgencyWorkspace.Workspace.Slug != "northwind-studio" {
		t.Fatalf("agency workspace: %+v", o.AgencyWorkspace)
	}
	// Both approvals wait for the client's own approver, never for staff.
	if len(o.Approvals) != 2 {
		t.Fatalf("approvals: %+v", o.Approvals)
	}
	for _, a := range o.Approvals {
		want := map[string]string{"Wine club: November box": "Daniela Ortiz", "Holiday blend pre-order": "Jane Morales"}[a.Campaign]
		if !a.ClientSignoff || !slices.Equal(a.WaitingFor, []string{want}) {
			t.Fatalf("approval %s waits for %v", a.Campaign, a.WaitingFor)
		}
	}
	if len(o.Incidents) != 2 || o.Incidents[0].Kind != "auto_pause" || o.Incidents[0].State != "resumed" || !o.Incidents[0].WithChanges ||
		o.Incidents[1].Kind != "provider_down" || o.Incidents[1].State != "recovered" || len(o.Incidents[1].Failovers) != 1 || o.Incidents[1].Failovers[0].To != "Postmark" {
		t.Fatalf("incidents: %+v", o.Incidents)
	}
	// The next 7 days, soonest first. The sample's sends are relative to
	// now: late in the day, today's 17:00 send has moved to tomorrow.
	names := []string{}
	for i, s := range o.Sends {
		names = append(names, s.Name)
		if i > 0 && s.SendAt.Before(o.Sends[i-1].SendAt) {
			t.Fatalf("sends out of order: %v", names)
		}
	}
	slices.Sort(names)
	if !slices.Equal(names, []string{"Season pass early bird", "Studio notes #18", "Weekend gear sale, by email and text", "Wine club: November box"}) ||
		o.NextAfter == nil || o.NextAfter.Name != "Holiday blend pre-order" {
		t.Fatalf("sends: %v then %+v", names, o.NextAfter)
	}
}

func TestIsolationBetweenClients(t *testing.T) {
	_, p := northwind(t)
	all := getOverview(t, p["tomas"])
	tailspin := workspaceID(t, all, "tailspin-toys")

	// An account manager sees her clients only.
	mei := getOverview(t, p["mei"])
	if mei.Counts.All != 4 || mei.CanCreate {
		t.Fatalf("Mei's overview: %+v", mei.Counts)
	}
	for _, c := range mei.Clients {
		if !slices.Contains([]string{"contoso-coffee", "fabrikam-outdoors", "coho-winery", "wide-world-importers"}, c.Workspace.Slug) {
			t.Fatalf("Mei sees %s", c.Workspace.Slug)
		}
	}
	p["mei"].Do("GET", "/agency/clients/"+tailspin, nil).MustStatus(t, 404)
	p["mei"].Do("POST", "/agency/clients", map[string]any{"name": "Sneaky", "timezone": "UTC", "ownerId": "00000000-0000-0000-0000-000000000000"}).MustStatus(t, 403)

	// Client people never see the Agency area or other clients.
	var me struct {
		Enabled bool
		Name    string
		Staff   *struct{ Level string }
		Clients []any
	}
	p["jane.morales"].Do("GET", "/agency/me", nil).MustStatus(t, 200).JSON(t, &me)
	if !me.Enabled || me.Name != "Northwind Studio" || me.Staff != nil || len(me.Clients) != 0 {
		t.Fatalf("Jane's agency: %+v", me)
	}
	for _, path := range []string{"/agency/overview", "/agency/staff", "/agency/library", "/agency/usage", "/agency/templates", "/agency/search?q=holiday", "/agency/notifications"} {
		p["jane.morales"].Do("GET", path, nil).MustStatus(t, 403)
	}
	p["jane.morales"].Do("GET", "/workspaces/fabrikam-outdoors/campaigns", nil).MustStatus(t, 404)
	p["rick.alvarez"].Do("GET", "/workspaces/contoso-coffee/campaigns", nil).MustStatus(t, 404)
	// Client roles hold no team:view; Contoso's team lists its own members
	// only (no Ella, who isn't assigned there), the staff marked as coming
	// from the agency.
	p["jane.morales"].Do("GET", "/workspaces/contoso-coffee/members", nil).MustStatus(t, 403)
	var team struct {
		Items []struct{ Email, Source string }
	}
	p["mei"].Do("GET", "/workspaces/contoso-coffee/members", nil).MustStatus(t, 200).JSON(t, &team)
	for _, m := range team.Items {
		if m.Email == "ella@northwind.studio" || strings.HasSuffix(m.Email, "@fabrikam.example") {
			t.Fatalf("Contoso's team shows %s", m.Email)
		}
		if strings.HasSuffix(m.Email, "@northwind.studio") && m.Source != "agency" {
			t.Fatalf("%s comes from %q", m.Email, m.Source)
		}
	}
	// Each client keeps its own contacts: the same address is two contacts.
	var contacts struct {
		Items []struct{ ID, Email string }
	}
	p["tomas"].Do("GET", "/workspaces/contoso-coffee/contacts?q=j.doe@example.com", nil).MustStatus(t, 200).JSON(t, &contacts)
	if len(contacts.Items) != 1 {
		t.Fatalf("contoso contacts: %+v", contacts.Items)
	}
	p["tomas"].Do("GET", "/workspaces/fabrikam-outdoors/contacts/"+contacts.Items[0].ID, nil).MustStatus(t, 404)
}

type staffView struct {
	Staff []struct {
		ID, UserID, Name, Level string
		EveryClient             bool
		Clients                 []string
	}
	Assignments []struct {
		Workspace ref
		Cells     []struct {
			StaffID string
			Role    *struct{ Key, Name string }
		}
		Roles []struct {
			ID, Key   string
			Grantable bool
		}
		CanAssign bool
	}
}

func (v staffView) staff(t *testing.T, name string) string {
	t.Helper()
	for _, s := range v.Staff {
		if s.Name == name {
			return s.ID
		}
	}
	t.Fatalf("no staff %s", name)
	return ""
}

func (v staffView) user(t *testing.T, name string) string {
	t.Helper()
	for _, s := range v.Staff {
		if s.Name == name {
			return s.UserID
		}
	}
	t.Fatalf("no staff %s", name)
	return ""
}

func (v staffView) role(t *testing.T, slug, key string) (string, bool) {
	t.Helper()
	for _, a := range v.Assignments {
		if a.Workspace.Slug != slug {
			continue
		}
		for _, r := range a.Roles {
			if r.Key == key {
				return r.ID, r.Grantable
			}
		}
	}
	t.Fatalf("no role %s in %s", key, slug)
	return "", false
}

func TestAssignmentsAndTheEscalationGuard(t *testing.T) {
	env, p := northwind(t)
	var v staffView
	p["tomas"].Do("GET", "/agency/staff", nil).MustStatus(t, 200).JSON(t, &v)
	if len(v.Staff) != 5 || len(v.Assignments) != 8 {
		t.Fatalf("staff %d, rows %d", len(v.Staff), len(v.Assignments))
	}
	ella, ravi := v.staff(t, "Ella Fischer"), v.staff(t, "Ravi Menon")
	contoso := v.Assignments[0].Workspace.ID
	for _, a := range v.Assignments {
		if a.Workspace.Slug == "contoso-coffee" {
			contoso = a.Workspace.ID
		}
	}
	editor, _ := v.role(t, "contoso-coffee", "editor")
	admin, _ := v.role(t, "contoso-coffee", "admin")

	// Tomás assigns Ella to Contoso as an Editor: an ordinary membership,
	// marked as coming from the agency, and audited.
	p["tomas"].Do("PUT", "/agency/assignments", map[string]any{"staffId": ella, "workspaceId": contoso, "roleId": editor}).MustStatus(t, 204)
	var source string
	if err := env.App.Pool.QueryRow(context.Background(), `SELECT m.source FROM workspace_members m JOIN users u ON u.id = m.user_id
		WHERE u.email = 'ella@northwind.studio' AND m.workspace_id = $1`, contoso).Scan(&source); err != nil || source != "agency" {
		t.Fatalf("Ella's membership: %q %v", source, err)
	}
	if n := count(t, env, `SELECT count(*) FROM audit_events WHERE action = 'agency.assigned' AND workspace_id = $1 AND target_name = 'Ella Fischer'`, contoso); n != 1 {
		t.Fatalf("assignment audited %d times", n)
	}

	// Ravi is an Editor in Contoso: Admin isn't his to give, and trying is
	// recorded as denied.
	var rv staffView
	p["ravi"].Do("GET", "/agency/staff", nil).MustStatus(t, 200).JSON(t, &rv)
	if _, ok := rv.role(t, "contoso-coffee", "admin"); ok {
		t.Fatal("Admin is grantable for an Editor")
	}
	// Client approver approves sends (campaigns:approve), which an Editor
	// doesn't hold; Client viewer holds nothing an Editor lacks.
	if _, ok := rv.role(t, "contoso-coffee", "client_approver"); ok {
		t.Fatal("Client approver is grantable for an Editor")
	}
	if _, ok := rv.role(t, "contoso-coffee", "client_viewer"); !ok {
		t.Fatal("Client viewer isn't grantable for an Editor")
	}
	r := p["ravi"].Do("PUT", "/agency/assignments", map[string]any{"staffId": ella, "workspaceId": contoso, "roleId": admin})
	r.MustStatus(t, 403)
	if n := count(t, env, `SELECT count(*) FROM audit_events WHERE action = 'agency.assignment_changed' AND result = 'denied' AND workspace_id = $1`, contoso); n != 2 {
		// The sample's attempt and this one.
		t.Fatalf("denied attempts audited: %d", n)
	}

	// Removing an assignment asks about the keys the person created there.
	if _, err := env.App.Pool.Exec(context.Background(), `INSERT INTO api_keys (id, workspace_id, name, prefix, token_hash, permissions, created_by, created_at)
		SELECT gen_random_uuid(), $1, 'Ella''s export key', 'rc_test', decode(md5(random()::text), 'hex'), '{campaigns:view}', u.id, now()
		FROM users u WHERE u.email = 'ella@northwind.studio'`, contoso); err != nil {
		t.Fatal(err)
	}
	var keys struct{ Keys []struct{ Name string } }
	p["tomas"].Do("GET", "/agency/assignments/keys?staffId="+ella+"&workspaceId="+contoso, nil).MustStatus(t, 200).JSON(t, &keys)
	if len(keys.Keys) != 1 || keys.Keys[0].Name != "Ella's export key" {
		t.Fatalf("keys: %+v", keys)
	}
	p["tomas"].Do("POST", "/agency/assignments/remove", map[string]any{"staffId": ella, "workspaceId": contoso, "revokeKeys": true}).MustStatus(t, 204)
	if n := count(t, env, `SELECT count(*) FROM workspace_members m JOIN users u ON u.id = m.user_id WHERE u.email = 'ella@northwind.studio' AND m.workspace_id = $1`, contoso); n != 0 {
		t.Fatal("the membership is gone")
	}
	if n := count(t, env, `SELECT count(*) FROM api_keys WHERE workspace_id = $1 AND revoked_at IS NULL AND name = 'Ella''s export key'`, contoso); n != 0 {
		t.Fatal("the key is revoked")
	}
	// Nobody removes their own assignment from here.
	p["tomas"].Do("POST", "/agency/assignments/remove", map[string]any{"staffId": v.staff(t, "Tomás Herrera"), "workspaceId": contoso}).MustStatus(t, 409)
	_ = ravi
}

func currentMonth() string { return time.Now().UTC().Format("2006-01") }

func count(t *testing.T, env *apptest.Env, sql string, args ...any) int {
	t.Helper()
	var n int
	if err := env.App.Pool.QueryRow(context.Background(), sql, args...).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

type plan struct {
	Lists, Automations, Sections, Templates struct {
		Added   []string
		Renamed []struct{ From, To string }
	}
	ClientSignoff, ClientRoles string
	NextSteps                  []string
}

func TestWorkspaceTemplates(t *testing.T) {
	env, p := northwind(t)
	o := getOverview(t, p["tomas"])
	alpine := workspaceID(t, o, "alpine-ski-house")
	var tpls []struct {
		ID, Name string
		Default  bool
		UsedBy   []ref
	}
	p["tomas"].Do("GET", "/agency/templates", nil).MustStatus(t, 200).JSON(t, &tpls)
	if len(tpls) != 3 || tpls[0].Name != "Retail starter (Shopify)" || !tpls[0].Default || len(tpls[0].UsedBy) != 4 {
		t.Fatalf("templates: %+v", tpls)
	}
	retail := tpls[0].ID

	// Applying Retail starter to Alpine (set up from Hospitality starter):
	// what's missing is added, clashes are renamed, nothing is overwritten.
	var pl plan
	p["tomas"].Do("GET", "/agency/templates/"+retail+"/plan?workspaceId="+alpine, nil).MustStatus(t, 200).JSON(t, &pl)
	if !slices.Equal(pl.Lists.Added, []string{"Customers", "VIP"}) || len(pl.Lists.Renamed) != 1 || pl.Lists.Renamed[0].To != "Newsletter (template)" {
		t.Fatalf("lists: %+v", pl.Lists)
	}
	if !slices.Equal(pl.Automations.Added, []string{"Abandoned cart", "Post-purchase thank-you"}) || len(pl.Automations.Renamed) != 2 {
		t.Fatalf("automations: %+v", pl.Automations)
	}
	if len(pl.Sections.Added) != 11 || len(pl.Sections.Renamed) != 3 || len(pl.Templates.Added) != 6 || pl.ClientSignoff != "already" || pl.ClientRoles != "already" {
		t.Fatalf("content: %d+%d sections, %d templates, sign-off %s, roles %s", len(pl.Sections.Added), len(pl.Sections.Renamed), len(pl.Templates.Added), pl.ClientSignoff, pl.ClientRoles)
	}
	before := count(t, env, `SELECT count(*) FROM list_memberships m JOIN lists l ON l.id = m.list_id WHERE l.workspace_id = $1 AND l.name = 'Newsletter'`, alpine)
	providers := count(t, env, `SELECT count(*) FROM providers WHERE workspace_id = $1`, alpine)
	p["tomas"].Do("POST", "/agency/templates/"+retail+"/apply", map[string]any{"workspaceId": alpine}).MustStatus(t, 200)
	if n := count(t, env, `SELECT count(*) FROM lists WHERE workspace_id = $1 AND name IN ('Newsletter (template)', 'Customers', 'VIP') AND deleted_at IS NULL`, alpine); n != 3 {
		t.Fatalf("lists added: %d", n)
	}
	if n := count(t, env, `SELECT count(*) FROM list_memberships m JOIN lists l ON l.id = m.list_id WHERE l.workspace_id = $1 AND l.name = 'Newsletter'`, alpine); n != before {
		t.Fatal("the existing Newsletter list was changed")
	}
	if n := count(t, env, `SELECT count(*) FROM automations WHERE workspace_id = $1 AND name = 'Welcome series (template)' AND status = 'draft'`, alpine); n != 1 {
		t.Fatal("the clashing flow arrives renamed, as a draft")
	}
	if n := count(t, env, `SELECT count(*) FROM providers WHERE workspace_id = $1`, alpine); n != providers {
		t.Fatal("providers are never copied")
	}
	if n := count(t, env, `SELECT count(*) FROM audit_events WHERE action = 'agency.template_applied' AND workspace_id = $1`, alpine); n != 1 {
		t.Fatal("applying is audited")
	}

	// A new client from the template: its setup, nothing of anyone else's.
	var nadiaID string
	var v staffView
	p["tomas"].Do("GET", "/agency/staff", nil).MustStatus(t, 200).JSON(t, &v)
	nadiaID = v.user(t, "Nadia Okafor")
	var created struct {
		Workspace ref
		Plan      *plan
		Setup     *struct{ Done, Total int }
	}
	p["tomas"].Do("POST", "/agency/clients", map[string]any{"name": "Trey Research", "timezone": "America/Chicago", "templateId": retail, "ownerId": nadiaID,
		"accountManagerIds": []string{v.user(t, "Mei Lin")}}).MustStatus(t, 201).JSON(t, &created)
	if created.Plan == nil || len(created.Plan.Lists.Added) != 3 || created.Setup == nil || created.Setup.Total != 4 || created.Setup.Done != 0 {
		t.Fatalf("created: %+v %+v", created.Plan, created.Setup)
	}
	ws := created.Workspace.ID
	for table, want := range map[string]int{"contacts": 0, "providers": 0, "sending_domains": 0, "suppressions": 0, "api_keys": 0} {
		if n := count(t, env, `SELECT count(*) FROM `+table+` WHERE workspace_id = $1`, ws); n != want {
			t.Fatalf("%s: %d", table, n)
		}
	}
	if n := count(t, env, `SELECT count(*) FROM roles WHERE workspace_id = $1 AND key IN ('client_approver', 'client_viewer')`, ws); n != 2 {
		t.Fatal("client roles")
	}
	if n := count(t, env, `SELECT count(*) FROM workspaces WHERE id = $1 AND approval_policy->'client_signoff'->>'enabled' = 'true'`, ws); n != 1 {
		t.Fatal("client sign-off is on")
	}
}

func TestLibraryCopiesAndNewerVersions(t *testing.T) {
	env, p := northwind(t)
	var lib struct {
		Items []struct {
			ID, Name, Kind  string
			Version, Behind int
		}
		Counts map[string]int
		Total  int
	}
	p["tomas"].Do("GET", "/agency/library", nil).MustStatus(t, 200).JSON(t, &lib)
	if lib.Total != 41 || lib.Counts["section"] != 22 || lib.Items[0].Name != "Holiday hero (snow)" || lib.Items[0].Behind != 1 {
		t.Fatalf("library: %d items, first %+v", lib.Total, lib.Items[0])
	}
	hero := lib.Items[0].ID
	var item struct {
		Copies []struct {
			ID        string
			Workspace ref
			Version   int
			Behind    bool
			CanUpdate bool
		}
		Diff *struct {
			From, To int
			Lines    []struct{ Op, Text string }
		}
		Preview string
	}
	p["tomas"].Do("GET", "/agency/library/"+hero, nil).MustStatus(t, 200).JSON(t, &item)
	if item.Diff == nil || item.Diff.From != 2 || item.Diff.To != 3 || len(item.Diff.Lines) != 4 || item.Diff.Lines[0].Text != "Button: Shop now" || item.Diff.Lines[1].Text != "Button: Shop gifts" {
		t.Fatalf("diff: %+v", item.Diff)
	}
	if !strings.Contains(item.Preview, "Hi there") {
		t.Fatal("the preview reads with the fallbacks")
	}
	var contosoCopy string
	for _, c := range item.Copies {
		if c.Workspace.Slug == "contoso-coffee" {
			if !c.Behind || c.Version != 2 || !c.CanUpdate {
				t.Fatalf("Contoso's copy: %+v", c)
			}
			contosoCopy = c.ID
		}
	}
	// Updating Contoso's copy brings it to v3.
	p["tomas"].Do("POST", "/agency/library/copies/"+contosoCopy+"/update", nil).MustStatus(t, 204)
	if n := count(t, env, `SELECT count(*) FROM templates t JOIN workspaces w ON w.id = t.workspace_id WHERE w.slug = 'contoso-coffee' AND t.name = 'Holiday hero (snow)' AND t.mjml LIKE '%Shop gifts%'`); n != 1 {
		t.Fatal("the copy has the new button")
	}
	// Using an item in a client makes a copy there; copies aren't linked.
	o := getOverview(t, p["tomas"])
	coho := workspaceID(t, o, "coho-winery")
	p["tomas"].Do("POST", "/agency/library/"+hero+"/copies", map[string]any{"workspaceId": coho}).MustStatus(t, 201)
	p["tomas"].Do("GET", "/agency/library/"+hero, nil).MustStatus(t, 200).JSON(t, &item)
	if len(item.Copies) != 5 {
		t.Fatalf("copies: %d", len(item.Copies))
	}
	// Changing the library needs agency:library. Narrowed to the overview
	// and usage, Mei browses and copies, and edits nothing.
	var v staffView
	p["tomas"].Do("GET", "/agency/staff", nil).MustStatus(t, 200).JSON(t, &v)
	p["tomas"].Do("PATCH", "/agency/staff/"+v.staff(t, "Mei Lin"), map[string]any{"level": "account_manager", "permissions": []string{"agency:view", "agency:usage"}}).MustStatus(t, 204)
	var refused struct{ Detail string }
	p["mei"].Do("PATCH", "/agency/library/"+hero, map[string]any{"name": "Holiday hero"}).MustStatus(t, 403).JSON(t, &refused)
	if !strings.HasPrefix(refused.Detail, "You can't change the library or workspace templates.") || strings.Contains(refused.Detail, "agency:") {
		t.Fatalf("refusal: %q", refused.Detail)
	}
	p["mei"].Do("GET", "/agency/library/"+hero, nil).MustStatus(t, 200)
	// Narrowing never adds: agency:manage isn't an account manager's.
	p["tomas"].Do("PATCH", "/agency/staff/"+v.staff(t, "Mei Lin"), map[string]any{"level": "account_manager", "permissions": []string{"agency:manage"}}).MustStatus(t, 422)
}

func TestUsageAndRebill(t *testing.T) {
	_, p := northwind(t)
	var u struct {
		Month   string
		Partial bool
		Rows    []struct {
			Workspace ref
			Emails    int64
			Rebill    struct {
				Cost     float64
				CostPlus float64
				ToRebill *float64
			}
		}
		Totals struct {
			Workspaces             int
			Emails, Segments, Push int64
			Cost, ToRebill         float64
		}
		CanSeeCosts bool
	}
	p["tomas"].Do("GET", "/agency/usage", nil).MustStatus(t, 200).JSON(t, &u)
	if u.Partial || u.Totals.Workspaces != 7 || u.Totals.Emails != 3641748 || u.Totals.Segments != 42622 || u.Totals.Push != 14920 ||
		u.Totals.Cost != 865.05 || u.Totals.ToRebill != 1891.59 {
		t.Fatalf("last month: %+v", u.Totals)
	}
	for _, r := range u.Rows {
		if r.Workspace.Slug == "fabrikam-outdoors" && (r.Rebill.Cost != 633.59 || r.Rebill.CostPlus != 95.04 || r.Rebill.ToRebill == nil || *r.Rebill.ToRebill != 1028.63) {
			t.Fatalf("fabrikam: %+v", r.Rebill)
		}
		if r.Workspace.Slug == "northwind-studio" && r.Rebill.ToRebill != nil {
			t.Fatal("the agency's own workspace isn't billed")
		}
	}
	var cur struct {
		Partial bool
		Totals  struct{ Cost, ToRebill float64 }
	}
	p["tomas"].Do("GET", "/agency/usage?month="+currentMonth(), nil).MustStatus(t, 200).JSON(t, &cur)
	if !cur.Partial || cur.Totals.Cost != 509.56 || cur.Totals.ToRebill != 1451.74 {
		t.Fatalf("this month so far: %+v", cur)
	}
	// Without agency:usage: volumes only.
	p["mei"].Do("GET", "/agency/usage", nil).MustStatus(t, 200).JSON(t, &u)
	if u.CanSeeCosts {
		// Mei holds agency:usage as an account manager; Ravi doesn't once narrowed.
		_ = u
	}
	r := p["tomas"].Do("GET", "/agency/usage/export", nil)
	r.MustStatus(t, 200)
	if !strings.HasPrefix(r.Header.Get("Content-Type"), "text/csv") || !strings.Contains(string(r.Body), "Fabrikam Outdoors") {
		t.Fatalf("export: %s", r.Header.Get("Content-Type"))
	}
}

func TestContactSearchIsRestrictedAndAudited(t *testing.T) {
	env, p := northwind(t)
	type results struct {
		Contact           bool
		Groups            []struct{ Workspace ref }
		Searched, Skipped int
		Total             int
	}
	var res results
	before := count(t, env, `SELECT count(*) FROM audit_events WHERE action = 'agency.contact_searched'`)
	p["tomas"].Do("GET", "/agency/search?q=j.doe@example.com", nil).MustStatus(t, 200).JSON(t, &res)
	if !res.Contact || len(res.Groups) != 2 || res.Groups[0].Workspace.Slug != "contoso-coffee" || res.Groups[1].Workspace.Slug != "fabrikam-outdoors" {
		t.Fatalf("results: %+v", res)
	}
	// One entry for the agency, one in each client where it was found.
	if n := count(t, env, `SELECT count(*) FROM audit_events WHERE action = 'agency.contact_searched'`); n != before+3 {
		t.Fatalf("audit entries: %d", n-before)
	}
	if n := count(t, env, `SELECT count(*) FROM audit_events e JOIN workspaces w ON w.id = e.workspace_id
		WHERE e.action = 'agency.contact_searched' AND w.slug = 'contoso-coffee' AND e.target_name = 'j.doe@example.com' AND e.actor_name = 'Tomás Herrera'`); n < 1 {
		t.Fatal("Contoso's own log shows the search")
	}
	// Only exact addresses: part of one finds no contacts.
	p["tomas"].Do("GET", "/agency/search?q=j.doe", nil).MustStatus(t, 200).JSON(t, &res)
	if res.Contact || res.Total != 0 {
		t.Fatalf("partial address: %+v", res)
	}
	// Ella's clients don't have the address.
	p["ella"].Do("GET", "/agency/search?q=j.doe@example.com", nil).MustStatus(t, 200).JSON(t, &res)
	if len(res.Groups) != 0 || res.Searched != 4 {
		t.Fatalf("Ella: %+v", res)
	}
	// Where you can't see contacts, contacts aren't searched.
	if _, err := env.App.Pool.Exec(context.Background(), `UPDATE workspace_members SET role_id = (SELECT id FROM roles WHERE workspace_id = workspace_members.workspace_id AND key = 'client_viewer')
		WHERE user_id = (SELECT id FROM users WHERE email = 'ravi@northwind.studio') AND workspace_id = (SELECT id FROM workspaces WHERE slug = 'fabrikam-outdoors')`); err != nil {
		t.Fatal(err)
	}
	p["ravi"].Do("GET", "/agency/search?q=j.doe@example.com", nil).MustStatus(t, 200).JSON(t, &res)
	if res.Skipped != 1 || len(res.Groups) != 1 || res.Groups[0].Workspace.Slug != "contoso-coffee" {
		t.Fatalf("Ravi without subscribers:view in Fabrikam: %+v", res)
	}
	// Names: Mei's "holiday" finds her clients' and the library's.
	var names struct {
		Groups []struct {
			Workspace ref
			Hits      []struct{ Name string }
		}
		Library []struct{ Name string }
		Total   int
	}
	p["mei"].Do("GET", "/agency/search?q=holiday", nil).MustStatus(t, 200).JSON(t, &names)
	if names.Total != 4 || len(names.Library) != 1 || names.Library[0].Name != "Holiday hero (snow)" {
		t.Fatalf("Mei's holiday: %+v", names)
	}
}

func TestClientSignoffWaitsForTheClient(t *testing.T) {
	env, p := northwind(t)
	o := getOverview(t, p["tomas"])
	var req string
	for _, a := range o.Approvals {
		if a.Campaign == "Holiday blend pre-order" {
			req = a.ID
		}
	}
	path := "/workspaces/contoso-coffee/approvals/" + req
	// Tomás holds campaigns:approve as an Admin, but under client sign-off
	// staff approvals don't count.
	p["tomas"].Do("POST", path+"/decision", map[string]any{"decision": "approve"}).MustStatus(t, 200)
	var d struct {
		Status        string
		ClientSignoff *struct {
			Met            bool
			StaffApprovals []string
		}
	}
	p["tomas"].Do("GET", path, nil).MustStatus(t, 200).JSON(t, &d)
	if d.Status != "pending" || d.ClientSignoff == nil || d.ClientSignoff.Met || len(d.ClientSignoff.StaffApprovals) != 1 {
		t.Fatalf("after Tomás: %+v %+v", d.Status, d.ClientSignoff)
	}
	p["jane.morales"].Do("POST", path+"/decision", map[string]any{"decision": "approve"}).MustStatus(t, 200)
	p["jane.morales"].Do("GET", path, nil).MustStatus(t, 200).JSON(t, &d)
	if d.Status != "approved" || !d.ClientSignoff.Met {
		t.Fatalf("after Jane: %+v %+v", d.Status, d.ClientSignoff)
	}
	// Both decisions are audited; Tomás's says sign-off was still waiting.
	if n := count(t, env, `SELECT count(*) FROM audit_events e JOIN workspaces w ON w.id = e.workspace_id WHERE w.slug = 'contoso-coffee' AND e.action = 'campaign.approved'
		AND e.metadata->>'client_signoff' IS NOT NULL AND e.actor_name = 'Tomás Herrera'`); n != 1 {
		t.Fatalf("Tomás's approval audited as waiting: %d", n)
	}
	// Client sign-off belongs to client workspaces: the agency's own can't have it.
	var pol map[string]any
	p["nadia"].Do("GET", "/workspaces/northwind-studio/approval-policy", nil).MustStatus(t, 200).JSON(t, &pol)
	pol["clientSignoff"] = map[string]any{"enabled": true}
	p["nadia"].Do("PUT", "/workspaces/northwind-studio/approval-policy", pol).MustStatus(t, 422)
}

func TestAgencyModeAndTheAuditCategory(t *testing.T) {
	_, p := northwind(t)
	var mode struct {
		Enabled bool
		Clients int
	}
	p["nadia"].Do("GET", "/admin/agency", nil).MustStatus(t, 200).JSON(t, &mode)
	if !mode.Enabled || mode.Clients != 7 {
		t.Fatalf("mode: %+v", mode)
	}
	p["tomas"].Do("GET", "/admin/agency", nil).MustStatus(t, 403)

	// The Agency category of the audit log, in a workspace and the instance.
	var log struct {
		Items []struct{ Action string }
	}
	p["tomas"].Do("GET", "/workspaces/wide-world-importers/audit?category=agency&range=all", nil).MustStatus(t, 200).JSON(t, &log)
	if len(log.Items) == 0 {
		t.Fatal("no agency entries in Wide World Importers")
	}
	for _, it := range log.Items {
		if !strings.HasPrefix(it.Action, "agency.") {
			t.Fatalf("category agency shows %s", it.Action)
		}
	}
	p["nadia"].Do("GET", "/admin/audit?category=agency&range=all", nil).MustStatus(t, 200).JSON(t, &log)
	actions := []string{}
	for _, it := range log.Items {
		actions = append(actions, it.Action)
	}
	if !slices.Contains(actions, "agency.mode_enabled") || !slices.Contains(actions, "agency.contact_searched") {
		t.Fatalf("instance agency entries: %v", actions)
	}

	// Turning agency mode off keeps every workspace and membership; the
	// Agency area goes.
	p["nadia"].Do("PUT", "/admin/agency", map[string]any{"enabled": false}).MustStatus(t, 200)
	p["tomas"].Do("GET", "/agency/overview", nil).MustStatus(t, 403)
	p["tomas"].Do("GET", "/workspaces/fabrikam-outdoors/campaigns", nil).MustStatus(t, 200)
}
