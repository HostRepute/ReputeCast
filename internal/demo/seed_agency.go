package demo

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/hostrepute/reputecast/internal/authn"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/config"
	"github.com/hostrepute/reputecast/internal/secrets"
	"github.com/hostrepute/reputecast/internal/store"
	"github.com/hostrepute/reputecast/internal/workspaces"
)

// The agency sample (channels and agencies brief §4, "Agency: Northwind
// Studio"): an agency that runs seven client workspaces and its own, on an
// instance of its own. Acme stays without agency mode (seedAgency is a
// no-op there); this sample loads into an empty database with
// `reputecast seed --sample northwind`.
//
// The boards read as Wednesday 15 Oct 2026 around 10:00 in Chicago. As in
// the Acme sample, dates follow the real calendar: "today" is the Chicago
// date from 10:00 on, so the times the boards show for today (09:40) are
// always in the past, and the sends they show ahead stay ahead.

// NorthwindOwner signs in to the agency sample as its instance admin.
const NorthwindOwner = "nadia@northwind.studio"

// NorthwindStaff is the agency manager the boards show (QA signs in as him).
const NorthwindStaff = "tomas@northwind.studio"

// nwDesignToday is the boards' "today"; absolute dates on the boards keep
// their distance from it.
var nwDesignToday = time.Date(2026, 10, 15, 0, 0, 0, 0, time.UTC)

type nwPerson struct {
	key, email, name, color, tz string
	admin, totp, passkey        bool
	lastActive                  time.Duration
	title                       string // agency staff: what they do
	level                       string // agency staff level; "" for client people
}

var nwPeople = []nwPerson{
	{key: "nadia", email: NorthwindOwner, name: "Nadia Okafor", color: "info", tz: "America/Chicago", admin: true, totp: true, lastActive: 2 * time.Hour},
	{key: "tomas", email: NorthwindStaff, name: "Tomás Herrera", color: "ok", tz: "America/Chicago", totp: true, level: authz.AgencyManagerLevel},
	{key: "mei", email: "mei@northwind.studio", name: "Mei Lin", color: "pink", tz: "America/Chicago", totp: true, lastActive: 4 * time.Minute, level: authz.AccountManagerLevel},
	{key: "ella", email: "ella@northwind.studio", name: "Ella Fischer", color: "warn", tz: "America/Chicago", totp: true, lastActive: time.Hour, level: authz.AccountManagerLevel},
	{key: "ravi", email: "ravi@northwind.studio", name: "Ravi Menon", color: "purple", tz: "America/Chicago", totp: true, lastActive: 26 * time.Hour, level: authz.AccountManagerLevel,
		title: "designer"},
	// The clients' own people.
	{key: "jane", email: "jane.morales@contoso.example", name: "Jane Morales", color: "warn", tz: "America/Los_Angeles", passkey: true, totp: true, lastActive: 3 * time.Minute},
	{key: "rick", email: "rick.alvarez@fabrikam.example", name: "Rick Alvarez", color: "ok", tz: "America/Chicago", lastActive: 50 * time.Hour},
	{key: "hana", email: "hana.sato@fabrikam.example", name: "Hana Sato", color: "info", tz: "America/Chicago", lastActive: 6 * 24 * time.Hour},
	{key: "daniela", email: "daniela.ortiz@cohowinery.example", name: "Daniela Ortiz", color: "pink", tz: "America/Los_Angeles", lastActive: 4 * 24 * time.Hour},
	{key: "priya", email: "priya.shah@tailspintoys.example", name: "Priya Shah", color: "purple", tz: "America/New_York", lastActive: 9 * 24 * time.Hour},
	{key: "lars", email: "lars.berg@alpineskihouse.example", name: "Lars Berg", color: "ok", tz: "America/Denver", lastActive: 3 * 24 * time.Hour},
	{key: "grace", email: "grace.kim@litwarebooks.example", name: "Grace Kim", color: "warn", tz: "America/New_York", lastActive: 12 * 24 * time.Hour},
}

type nwClient struct {
	key, name, tz, color string
	agency               bool
	since                time.Time // on the boards' calendar
	manager              string    // the account manager (the agency workspace: its contact)
	template             string
	contacts             int
	// The client's people: person key → role key.
	people map[string]string
}

const (
	nwRetail      = "Retail starter (Shopify)"
	nwHospitality = "Hospitality starter"
	nwPublisher   = "Publisher starter"
)

var nwClients = []nwClient{
	{key: "contoso", name: "Contoso Coffee", tz: "America/Los_Angeles", color: "#92400E", since: date(2024, 3, 4), manager: "mei", template: nwRetail, contacts: 64210,
		people: map[string]string{"jane": authz.RoleClientApprover}},
	{key: "fabrikam", name: "Fabrikam Outdoors", tz: "America/Chicago", color: "#166534", since: date(2025, 2, 12), manager: "mei", template: nwRetail, contacts: 182406,
		people: map[string]string{"rick": authz.RoleClientApprover, "hana": authz.RoleClientViewer}},
	{key: "tailspin", name: "Tailspin Toys", tz: "America/New_York", color: "#1D4ED8", since: date(2025, 8, 20), manager: "ella", template: nwRetail, contacts: 41862,
		people: map[string]string{"priya": authz.RoleClientApprover}},
	{key: "alpine", name: "Alpine Ski House", tz: "America/Denver", color: "#0F766E", since: date(2025, 11, 3), manager: "ella", template: nwHospitality, contacts: 23115,
		people: map[string]string{"lars": authz.RoleClientApprover}},
	{key: "coho", name: "Coho Winery", tz: "America/Los_Angeles", color: "#9D174D", since: date(2026, 1, 26), manager: "mei", template: nwHospitality, contacts: 12940,
		people: map[string]string{"daniela": authz.RoleClientApprover}},
	{key: "litware", name: "Litware Books", tz: "America/New_York", color: "#5B21B6", since: date(2026, 4, 7), manager: "ella", template: nwPublisher, contacts: 8320,
		people: map[string]string{"grace": authz.RoleClientApprover}},
	{key: "wwi", name: "Wide World Importers", tz: "America/Chicago", color: "#475569", since: nwDesignToday, manager: "mei", template: nwRetail},
	{key: "northwind", name: "Northwind Studio", tz: "America/Chicago", agency: true, since: date(2026, 8, 1), manager: "nadia", contacts: 3410},
}

func date(y int, m time.Month, d int) time.Time { return time.Date(y, m, d, 0, 0, 0, 0, time.UTC) }

// nwSeed carries the agency sample's state between its steps.
type nwSeed struct {
	*Seeder
	loc   *time.Location
	today time.Time // midnight, Chicago
	// People and workspaces by key.
	people map[string]store.User
	ws     map[string]store.Workspace
	// Workspace templates by name; lists per workspace key and name.
	templates map[string]uuid.UUID
	lists     map[string]map[string]uuid.UUID
	// Providers by "<workspace>/<short>" ("contoso/ses").
	providers map[string]uuid.UUID
	senders   map[string]uuid.UUID // by workspace key
}

// at is hour:minute in Chicago, days after the sample's today (negative:
// before).
func (n *nwSeed) at(days, hour, minute int) time.Time {
	return time.Date(n.today.Year(), n.today.Month(), n.today.Day()+days, hour, minute, 0, 0, n.loc)
}

// ahead is at, moved a day later while it's less than an hour from now, so
// a send the boards show ahead never lands in the past.
func (n *nwSeed) ahead(days, hour, minute int) time.Time {
	t := n.at(days, hour, minute)
	for t.Before(n.Now.Add(time.Hour)) {
		t = t.AddDate(0, 0, 1)
	}
	return t
}

// onBoards places a date of the boards' calendar on the real one.
func (n *nwSeed) onBoards(d time.Time) time.Time {
	days := int(nwDesignToday.Sub(d).Hours() / 24)
	return n.at(-days, 10, 0)
}

// SeedNorthwind loads the agency sample into an empty database.
func SeedNorthwind(ctx context.Context, pool *pgxpool.Pool, cfg *config.Config, box *secrets.Box, log *slog.Logger) (*Result, error) {
	res := &Result{OwnerEmail: NorthwindStaff, Password: Password, TOTPSecret: TOTPSecret}
	var exists, others bool
	if err := pool.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM users WHERE email = $1), EXISTS (SELECT 1 FROM users WHERE email <> $1)`, NorthwindOwner).Scan(&exists, &others); err != nil {
		return nil, err
	}
	if exists {
		log.Info("the Northwind Studio sample is already present")
		return res, nil
	}
	if others {
		return nil, fmt.Errorf("the Northwind Studio sample needs an empty database: it sets up the instance as an agency (create one with createdb and point REPUTECAST_DATABASE__URL at it)")
	}
	tx, err := pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx) //nolint:errcheck
	loc, err := time.LoadLocation("America/Chicago")
	if err != nil {
		return nil, err
	}
	now := seedClock()
	local := now.In(loc).Add(-10 * time.Hour)
	n := &nwSeed{
		Seeder: &Seeder{Tx: tx, Q: store.New(tx), Box: box, Cfg: cfg, Now: now, Log: log,
			Users: map[string]store.User{}, Workspaces: map[string]store.Workspace{}, Roles: map[string]store.Role{},
			Lists: map[string]uuid.UUID{}, Providers: map[string]uuid.UUID{}, Campaigns: map[string]uuid.UUID{},
			Senders: map[string]uuid.UUID{}, Domains: map[string]uuid.UUID{}, Extra: map[string]any{}},
		loc: loc, today: time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, loc),
		people: map[string]store.User{}, ws: map[string]store.Workspace{}, templates: map[string]uuid.UUID{},
		lists: map[string]map[string]uuid.UUID{}, providers: map[string]uuid.UUID{}, senders: map[string]uuid.UUID{},
	}
	steps := []struct {
		name string
		fn   func(context.Context) error
	}{
		{"instance and people", n.instance},
		{"workspace templates", n.workspaceTemplates},
		{"workspaces", n.workspaces},
		{"staff and assignments", n.staff},
		{"client people", n.clientPeople},
		{"providers and senders", n.sending},
		{"contacts", n.contacts},
		{"campaigns", n.campaigns},
		{"approvals", n.approvals},
		{"incidents", n.incidents},
		{"library", n.library},
		{"usage, rates and health", n.usage},
		{"notifications and audit", n.history},
	}
	for _, st := range steps {
		if err := st.fn(ctx); err != nil {
			return nil, fmt.Errorf("seed northwind %s: %w", st.name, err)
		}
	}
	for i, fn := range n.after {
		if err := fn(ctx, n.Seeder); err != nil {
			return nil, fmt.Errorf("seed northwind deferred step %d: %w", i+1, err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return res, nil
}

// instance names the instance after the agency and creates its people.
func (n *nwSeed) instance(ctx context.Context) error {
	if _, err := n.Tx.Exec(ctx, `UPDATE instance_settings SET name = 'Northwind Studio', public_url = $1, default_timezone = 'America/Chicago', logo_svg = $2,
		setup_completed_at = $3, basics_confirmed_at = $3, twofa_enforcement = 'sensitive', twofa_grace_days = 7, session_lifetime = '14 days', session_idle = '12 hours',
		lockout_attempts = 10, lockout_minutes = 15`, n.Cfg.HTTP.PublicURL, nwLogoSVG, n.at(-420, 9, 0)); err != nil {
		return err
	}
	hash, err := authn.HashPassword(Password)
	if err != nil {
		return err
	}
	for _, p := range nwPeople {
		set := n.at(-400, 9, 0)
		u, err := n.Q.CreateUser(ctx, store.CreateUserParams{ID: ID(), Email: p.email, Name: p.name, Timezone: p.tz, PasswordHash: &hash, PasswordSetAt: &set,
			IsInstanceAdmin: p.admin, BreakGlass: p.admin, AvatarColor: p.color})
		if err != nil {
			return err
		}
		if p.totp {
			enc, err := n.Box.SealString(TOTPSecret, "user:"+u.ID.String()+":totp")
			if err != nil {
				return err
			}
			t := n.at(-380, 9, 30)
			if err := n.Q.SetUserTOTP(ctx, store.SetUserTOTPParams{ID: u.ID, TotpSecretEnc: enc, TotpEnabledAt: &t}); err != nil {
				return err
			}
			if p.admin {
				for _, c := range DemoRecoveryCodes {
					if err := n.Q.CreateRecoveryCode(ctx, store.CreateRecoveryCodeParams{ID: ID(), UserID: u.ID, CodeHash: authn.HashToken(authn.NormalizeRecoveryCode(c))}); err != nil {
						return err
					}
				}
			}
		}
		if p.passkey {
			// Placeholder key material: the list shows it; signing in with it
			// needs a real passkey, so the sample keeps a password too.
			if _, err := n.Tx.Exec(ctx, `INSERT INTO user_passkeys (id, user_id, name, credential_id, public_key, attestation_type, sign_count, transports,
				backup_eligible, backup_state, attachment, provider, created_at, last_used_at) VALUES ($1,$2,'iPhone',$3,$4,'none',0,$5,true,true,'platform','iCloud Keychain',$6,$7)`,
				ID(), u.ID, randomBytes(32), randomBytes(77), transportsFor("platform"), n.onBoards(date(2024, 3, 4)), n.Ago(p.lastActive)); err != nil {
				return err
			}
		}
		created := n.at(-400, 9, 0)
		if p.level == "" && !p.admin {
			created = n.at(-300, 11, 0)
		}
		if _, err := n.Tx.Exec(ctx, `UPDATE users SET last_active_at = $2, last_sign_in_at = $2, created_at = $3, twofa_grace_until = NULL WHERE id = $1`,
			u.ID, n.Ago(p.lastActive), created); err != nil {
			return err
		}
		u, _ = n.Q.GetUser(ctx, u.ID)
		n.people[p.key] = u
		n.Users[p.email] = u
	}
	return nil
}

// nwLogoSVG is Northwind Studio's mark: a white N on a near-black tile.
const nwLogoSVG = `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 28 28"><rect width="28" height="28" rx="6" fill="#0B1220"/>` +
	`<path d="M8.5 20V8h2.4l6.4 8.1V8h2.2v12h-2.3l-6.5-8.2V20z" fill="#fff"/></svg>`

// workspaces creates the seven clients and the agency's own workspace,
// turns agency mode on and sets each client up from its template.
func (n *nwSeed) workspaces(ctx context.Context) error {
	nadia, tomas := n.people["nadia"], n.people["tomas"]
	for _, c := range nwClients {
		created := n.onBoards(c.since)
		if c.key == "wwi" {
			created = n.at(0, 9, 40)
		}
		ws, err := workspaces.Create(ctx, n.Tx, workspaces.CreateParams{Name: c.name, Timezone: c.tz, Owner: nadia.ID, Source: "agency"})
		if err != nil {
			return err
		}
		if _, err := n.Tx.Exec(ctx, `UPDATE workspaces SET created_at = $2, updated_at = $2 WHERE id = $1`, ws.ID, created); err != nil {
			return err
		}
		if _, err := n.Tx.Exec(ctx, `UPDATE workspace_members SET joined_at = $2 WHERE workspace_id = $1`, ws.ID, created); err != nil {
			return err
		}
		ws.CreatedAt, ws.UpdatedAt = created, created
		n.ws[c.key] = ws
		n.Workspaces[ws.Slug] = ws
		roles, err := n.Q.ListRoles(ctx, ws.ID)
		if err != nil {
			return err
		}
		for _, r := range roles {
			n.Roles[c.key+"/"+r.Key] = store.Role{ID: r.ID, WorkspaceID: r.WorkspaceID, Key: r.Key, Name: r.Name, Builtin: r.Builtin}
		}
		kind, steps := "client", []string{}
		if c.agency {
			kind = "agency"
		}
		var tpl *uuid.UUID
		if id, ok := n.templates[c.template]; ok {
			tpl = &id
		}
		if c.key == "wwi" {
			steps = []string{"provider", "domain", "invite", "shopify"}
		}
		manager := n.people[c.manager].ID
		addedBy := nadia.ID
		if c.key == "wwi" {
			addedBy = tomas.ID
		}
		if err := n.Q.AgencyWorkspaceInsert(ctx, store.AgencyWorkspaceInsertParams{WorkspaceID: ws.ID, Kind: kind, AccountManagerID: &manager,
			ClientSince: time.Date(created.Year(), created.Month(), created.Day(), 0, 0, 0, 0, time.UTC), TemplateID: tpl, SetupSteps: steps, Color: c.color, AddedBy: &addedBy}); err != nil {
			return err
		}
		if _, err := n.Tx.Exec(ctx, `UPDATE agency_workspaces SET added_at = $2 WHERE workspace_id = $1`, ws.ID, created); err != nil {
			return err
		}
		if c.agency {
			continue
		}
		// Client workspaces get the client roles.
		for _, r := range authz.ClientRoles {
			role, err := n.Q.CreateRole(ctx, store.CreateRoleParams{ID: ID(), WorkspaceID: ws.ID, Key: r.Key, Name: r.Name, Description: r.Description,
				Builtin: true, Permissions: []string{}})
			if err != nil {
				return err
			}
			n.Roles[c.key+"/"+r.Key] = role
		}
	}
	return n.Q.AgencySaveSettings(ctx, store.AgencySaveSettingsParams{Enabled: true, Name: "Northwind Studio", DefaultTemplateID: ptr(n.templates[nwRetail]),
		EnabledBy: &nadia.ID, EnabledAt: ptr(n.onBoards(date(2026, 8, 1))), UpdatedBy: &tomas.ID})
}

// applyTemplates sets each client up from its template, as creating it
// did, once the staff are members (the template's drafts name them).
func (n *nwSeed) applyTemplates(ctx context.Context) error {
	for _, c := range nwClients {
		id, ok := n.templates[c.template]
		if !ok {
			continue
		}
		by, at := n.people["nadia"].ID, n.onBoards(c.since)
		if c.key == "wwi" {
			by, at = n.people["tomas"].ID, n.at(0, 9, 40)
		}
		if _, err := agencySeedTemplate(ctx, n, id, n.ws[c.key].ID, by, at); err != nil {
			return fmt.Errorf("%s: %w", c.name, err)
		}
		// Each client's own lists, by name.
		rows, err := n.Q.AgencyCaptureLists(ctx, n.ws[c.key].ID)
		if err != nil {
			return err
		}
		n.lists[c.key] = map[string]uuid.UUID{}
		for _, l := range rows {
			n.lists[c.key][l.Name] = l.ID
		}
	}
	// Alpine and Coho ask for client sign-off too.
	for _, key := range []string{"alpine", "coho"} {
		if err := n.signoff(ctx, key); err != nil {
			return err
		}
	}
	return nil
}

// signoff turns client sign-off on in a client's approval policy.
func (n *nwSeed) signoff(ctx context.Context, key string) error {
	var raw json.RawMessage
	if err := n.Tx.QueryRow(ctx, `SELECT approval_policy FROM workspaces WHERE id = $1`, n.ws[key].ID).Scan(&raw); err != nil {
		return err
	}
	out, err := agencyEnableSignoff(raw)
	if err != nil {
		return err
	}
	_, err = n.Tx.Exec(ctx, `UPDATE workspaces SET approval_policy = $2 WHERE id = $1`, n.ws[key].ID, out)
	return err
}

// staff adds the agency's staff and their assignments: Nadia owns every
// workspace (she created them), Tomás is Admin everywhere, Mei and Ella
// are Admin in their clients and Editors in the agency's workspace, and
// Ravi, the designer, is an Editor everywhere.
func (n *nwSeed) staff(ctx context.Context) error {
	nadia := n.people["nadia"]
	for _, p := range nwPeople {
		if p.level == "" {
			continue
		}
		u := n.people[p.key]
		if err := n.Q.AgencyStaffInsert(ctx, store.AgencyStaffInsertParams{ID: ID(), UserID: &u.ID, Email: p.email, Level: p.level, Title: p.title, AddedBy: &nadia.ID}); err != nil {
			return err
		}
		if _, err := n.Tx.Exec(ctx, `UPDATE agency_staff SET added_at = $2 WHERE user_id = $1`, u.ID, n.onBoards(date(2026, 8, 1)).Add(time.Duration(len(p.key))*time.Minute)); err != nil {
			return err
		}
	}
	assign := map[string]map[string]string{
		"tomas": {"*": authz.RoleAdmin},
		"mei":   {"contoso": authz.RoleAdmin, "fabrikam": authz.RoleAdmin, "coho": authz.RoleAdmin, "wwi": authz.RoleAdmin, "northwind": authz.RoleEditor},
		"ella":  {"tailspin": authz.RoleAdmin, "alpine": authz.RoleAdmin, "litware": authz.RoleAdmin, "northwind": authz.RoleEditor},
		"ravi":  {"*": authz.RoleEditor},
	}
	for _, who := range []string{"tomas", "mei", "ella", "ravi"} {
		u := n.people[who]
		for _, c := range nwClients {
			role, ok := assign[who][c.key]
			if !ok {
				role, ok = assign[who]["*"]
			}
			if !ok {
				continue
			}
			joined := n.onBoards(c.since).Add(20 * time.Minute)
			if c.key == "wwi" {
				joined = n.at(0, 9, 40)
			}
			if err := n.member(ctx, c.key, u, role, "agency", &nadia.ID, joined); err != nil {
				return err
			}
		}
	}
	return n.applyTemplates(ctx)
}

// member adds a person to a workspace.
func (n *nwSeed) member(ctx context.Context, wsKey string, u store.User, role, source string, by *uuid.UUID, joined time.Time) error {
	r, ok := n.Roles[wsKey+"/"+role]
	if !ok {
		return fmt.Errorf("no role %s in %s", role, wsKey)
	}
	m, err := n.Q.CreateMember(ctx, store.CreateMemberParams{ID: ID(), WorkspaceID: n.ws[wsKey].ID, UserID: u.ID, RoleID: r.ID, Scope: json.RawMessage(`{}`), Source: source, InvitedBy: by})
	if err != nil {
		return err
	}
	_, err = n.Tx.Exec(ctx, `UPDATE workspace_members SET joined_at = $2, last_active_at = $3 WHERE id = $1`, m.ID, joined, u.LastActiveAt)
	return err
}

// clientPeople adds the clients' own people with the client roles, and
// Owen Price's invitation to Contoso Coffee (Mei Lin, two days ago).
func (n *nwSeed) clientPeople(ctx context.Context) error {
	for _, c := range nwClients {
		for who, role := range c.people {
			u := n.people[who]
			joined := n.onBoards(c.since).Add(26 * time.Hour)
			if err := n.member(ctx, c.key, u, role, "invite", ptr(n.people[c.manager].ID), joined); err != nil {
				return err
			}
		}
	}
	token, hash := authn.NewToken()
	id := ID()
	enc, err := n.Box.SealString(token, "invitation:"+id.String()+":token")
	if err != nil {
		return err
	}
	mei := n.people["mei"]
	sent := n.at(-2, 15, 12)
	via := "Amazon SES"
	if _, err := n.Q.TeamCreateInvitation(ctx, store.TeamCreateInvitationParams{ID: id, WorkspaceID: n.ws["contoso"].ID, Email: "owen.price@contoso.example",
		RoleID: n.Roles["contoso/"+authz.RoleClientViewer].ID, Scope: json.RawMessage(`{}`), TokenHash: hash, TokenEnc: enc, InvitedBy: &mei.ID, SentVia: &via,
		LastSentAt: &sent, ExpiresAt: sent.Add(7 * 24 * time.Hour), ExpiresDays: 7}); err != nil {
		return err
	}
	_, err = n.Tx.Exec(ctx, `UPDATE invitations SET created_at = $2 WHERE id = $1`, id, sent)
	return err
}
