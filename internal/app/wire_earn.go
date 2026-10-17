package app

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"sync"

	"github.com/danielgtaylor/huma/v2"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/audience"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/demo"
	"github.com/hostrepute/reputecast/internal/earn"
	"github.com/hostrepute/reputecast/internal/jobs"
	"github.com/hostrepute/reputecast/internal/routing"
	"github.com/hostrepute/reputecast/internal/store"
	"github.com/hostrepute/reputecast/internal/transactional"

	// Stripe, the payment provider available so far. Earn reaches it
	// through internal/payments only.
	_ "github.com/hostrepute/reputecast/internal/payments/stripe"
	"github.com/hostrepute/reputecast/internal/payments/stripe/stripetest"
)

// Payments and paid subscriptions (design expansion phase 6, X16): the
// payment connection and its webhook, tiers and the upgrade page, paid
// members, imports and member emails.

// The service is shared by the API, the public pages, the webhook and the
// jobs of one App (tests point its Stripe at a fake and set its clock).
var earnServices sync.Map // *App → *earn.Service

func earnService(a *App) *earn.Service {
	if v, ok := earnServices.Load(a); ok {
		return v.(*earn.Service)
	}
	s := &earn.Service{
		Pool: a.Pool, Q: a.Q, Audit: a.Audit, Box: a.Secrets, Links: a.Links, URLs: a.URLs, Log: a.Log,
		Jobs:          func() *jobs.Client { return a.Jobs },
		Mailer:        earnMailer{a},
		Signup:        earnSignup{a},
		Confirmations: earnConfirmations{a},
		PageBase:      func(ctx context.Context, ws *store.Workspace) string { return earnPageBase(ctx, a, ws) },
		// The demo's sample connections talk to an in-process Stripe.
		Demo: func() *http.Client { return demoStripe().HTTPClient() },
	}
	if a.Cfg != nil {
		s.AllowPrivate = a.Cfg.Providers.AllowPrivateNetworks
	}
	v, _ := earnServices.LoadOrStore(a, s)
	return v.(*earn.Service)
}

// Earn is the Earn domain (tests set its clock and its Stripe).
func (a *App) Earn() *earn.Service { return earnService(a) }

// demoStripe is the Stripe the demo's sample connections use: in process,
// answering for the sample objects it doesn't hold, with the sample keys.
var demoStripe = sync.OnceValue(func() *stripetest.Fake {
	f := stripetest.New()
	f.Permissive = true
	f.AddKey(demo.EarnLiveKey, stripetest.KeySpec{})
	f.AddKey(demo.EarnTestKey, stripetest.KeySpec{})
	return f
})

// earnPageBase is the workspace's verified custom domain, where the Earn
// pages answer at the root (news.acme.io/upgrade).
func earnPageBase(ctx context.Context, a *App, ws *store.Workspace) string {
	fs := a.formsService()
	set, err := fs.LoadSettings(ctx, ws)
	if err != nil || set.CustomDomain() == "" {
		return ""
	}
	return fs.Links(ws, set).Origin()
}

// earnMailer sends member emails through transactional email: each is a
// template in the Essential "Memberships" category, sent only once
// published.
type earnMailer struct{ a *App }

// systemPrincipal acts for ReputeCast itself where the transactional domain
// wants a permission (creating the Memberships category when Stripe is
// connected): the person's own action is in the audit log through the
// request's context.
func systemPrincipal(ws *store.Workspace, perms ...authz.Permission) *authz.Principal {
	set := authz.Set{}
	for _, p := range perms {
		set[p] = struct{}{}
	}
	return &authz.Principal{Type: authz.ActorSystem, Name: "ReputeCast", WorkspaceID: ws.ID, Permissions: set}
}

func (m earnMailer) Category(ctx context.Context, ws *store.Workspace) (uuid.UUID, error) {
	cats, err := m.a.Q.TxnCategories(ctx, ws.ID)
	if err != nil {
		return uuid.Nil, err
	}
	for _, c := range cats {
		if c.Name == "Memberships" {
			return c.ID, nil
		}
	}
	c, err := transactionalService(m.a).CreateCategory(ctx, ws, systemPrincipal(ws, authz.TransactionalPublish), transactional.TransactionalCategoryInput{
		Name: "Memberships", Kind: transactional.KindEssential, Description: "Paid members' emails: welcome, trials, payments, renewals and gifts"})
	if err != nil {
		return uuid.Nil, err
	}
	return uuid.Parse(c.ID)
}

func (m earnMailer) CreateDraft(ctx context.Context, ws *store.Workspace, p *authz.Principal, d earn.TemplateDraft) (uuid.UUID, error) {
	// A template already holding the key (made by hand, or kept from an
	// earlier connection) is used as it is.
	var existing uuid.UUID
	err := m.a.Pool.QueryRow(ctx, `SELECT id FROM transactional_templates WHERE workspace_id = $1 AND key = $2 AND deleted_at IS NULL`, ws.ID, d.Key).Scan(&existing)
	if err == nil {
		return existing, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return uuid.Nil, err
	}
	vars := make([]transactional.TransactionalVariable, 0, len(d.Variables))
	for _, v := range d.Variables {
		vars = append(vars, transactional.TransactionalVariable{Name: v.Name, Type: v.Type, Required: v.Required, Sample: v.Sample, Default: v.Default, Sensitive: v.Sensitive})
	}
	if p == nil {
		p = systemPrincipal(ws)
	}
	t, err := transactionalService(m.a).CreateTemplate(ctx, ws, p, transactional.TransactionalTemplateCreate{
		Name: d.Name, Key: d.Key, Description: d.Description, CategoryID: d.Category.String(), Subject: d.Subject, Preheader: d.Preheader,
		EditorMode: "visual", MJML: d.MJML, Variables: vars, TestData: []transactional.TransactionalTestData{{Name: "Sample paid member", Data: d.TestData}},
		Origin: map[string]any{"source": "earn"},
	})
	if err != nil {
		return uuid.Nil, err
	}
	return uuid.Parse(t.ID)
}

func (m earnMailer) Templates(ctx context.Context, ws *store.Workspace, ids []uuid.UUID) (map[uuid.UUID]earn.TemplateState, error) {
	out := map[uuid.UUID]earn.TemplateState{}
	rows, err := m.a.Pool.Query(ctx, `SELECT t.id, t.key, t.name, coalesce(t.live_version, 0), t.versions_count, t.draft_hash, coalesce(v.content_hash, ''),
		t.deleted_at IS NOT NULL, coalesce(sn.name, ''), coalesce(sn.email, ''), t.reply_to, coalesce(c.name, ''), coalesce(c.kind, '')
		FROM transactional_templates t
		LEFT JOIN transactional_template_versions v ON v.template_id = t.id AND v.version = t.live_version
		LEFT JOIN transactional_categories c ON c.id = t.category_id
		LEFT JOIN senders sn ON sn.id = t.sender_id
		WHERE t.workspace_id = $1 AND t.id = ANY($2)`, ws.ID, ids)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var id uuid.UUID
		var st earn.TemplateState
		var count int
		var draft, live string
		if err := rows.Scan(&id, &st.Key, &st.Name, &st.LiveVersion, &count, &draft, &live, &st.Deleted, &st.FromName, &st.FromEmail, &st.ReplyTo,
			&st.CategoryName, &st.CategoryKind); err != nil {
			return nil, err
		}
		if st.LiveVersion == 0 || (draft != "" && draft != live) {
			st.DraftVersion = count + 1
		}
		out[id] = st
	}
	return out, rows.Err()
}

func (m earnMailer) Send(ctx context.Context, ws *store.Workspace, msg earn.MemberMessage) (earn.Sent, error) {
	in := transactional.TransactionalSend{Template: msg.Template, To: transactional.TransactionalAddress{Email: msg.To, Name: msg.ToName}, Data: msg.Data,
		Tags: []string{"earn"}, Metadata: map[string]string{"source": "earn"}}
	idem := ""
	if msg.Key != "" {
		idem = msg.Key
		if len(idem) > 255 {
			idem = idem[:255]
		}
	}
	status, body, err := transactionalService(m.a).Send(ctx, ws, in, idem, "api")
	if err != nil {
		var p *api.Problem
		if errors.As(err, &p) {
			switch p.Code {
			case transactional.CodeTemplateNotPublished, transactional.CodeTemplateNotFound:
				return earn.Sent{}, earn.ErrNotPublished
			case transactional.CodeNoHealthyRoute:
				return earn.Sent{}, earn.ErrNoRoute
			case transactional.CodeRecipientSuppressed:
				return earn.Sent{}, earn.ErrSuppressed
			}
		}
		return earn.Sent{}, err
	}
	var res struct {
		ID     string `json:"id"`
		Status string `json:"status"`
		Code   string `json:"code"`
	}
	_ = json.Unmarshal(body, &res)
	if status >= 400 {
		if res.Code == transactional.CodeRecipientSuppressed {
			return earn.Sent{}, earn.ErrSuppressed
		}
		return earn.Sent{}, errors.New("member email refused: " + res.Code)
	}
	out := earn.Sent{Status: res.Status}
	if id, err := uuid.Parse(res.ID); err == nil {
		out.MessageID = id
	}
	return out, nil
}

func (m earnMailer) Route(ctx context.Context, ws *store.Workspace) (earn.Route, error) {
	set, err := routing.Load(ctx, m.a.Q, ws.ID)
	if err != nil {
		return earn.Route{}, err
	}
	def, ok := set.Default(routing.StreamTransactional)
	if !ok || len(def.Splits) == 0 {
		return earn.Route{}, nil
	}
	provs, err := m.a.Q.ProvListActive(ctx, ws.ID)
	if err != nil {
		return earn.Route{}, err
	}
	name := func(id uuid.UUID) string {
		for _, p := range provs {
			if p.ID == id {
				return p.Name
			}
		}
		return ""
	}
	top := def.Splits[0]
	for _, sp := range def.Splits[1:] {
		if sp.Weight > top.Weight {
			top = sp
		}
	}
	out := earn.Route{Primary: name(top.ProviderID)}
	if def.Fallback != nil {
		out.Fallback = name(*def.Fallback)
	}
	return out, nil
}

// earnSignup joins a paid member's lists with consent, through the
// audience domain (the evidence: "paid_subscription").
type earnSignup struct{ a *App }

func (s earnSignup) Join(ctx context.Context, tx pgx.Tx, ws *store.Workspace, j earn.Join) (uuid.UUID, bool, error) {
	res, err := s.a.audience().Signup(ctx, tx, ws, audience.SignupInput{Email: j.Email, FirstName: j.FirstName, LastName: j.LastName, Lists: j.Lists,
		DoubleOptIn: j.Double, Evidence: audience.PublicEvidence{Source: j.Source, Note: j.Note, At: j.At}})
	if err != nil {
		return uuid.Nil, false, err
	}
	return res.ContactID, res.Created, nil
}

// earnConfirmations sends the double opt-in email to a new address given
// complimentary access.
type earnConfirmations struct{ a *App }

func (c earnConfirmations) Send(ctx context.Context, ws *store.Workspace, contactID uuid.UUID) error {
	_, err := c.a.audience().SendConfirmations(ctx, ws, []uuid.UUID{contactID})
	return err
}

func init() {
	apiRegistrars = append(apiRegistrars, func(a *App, h huma.API) {
		earn.Register(h, earnService(a))
	})
	ingestRegistrars = append(ingestRegistrars, func(a *App, r chi.Router) {
		earnService(a).Routes(r)
	})
	jobRegistrars = append(jobRegistrars, func(a *App) {
		earn.RegisterJobs(a.JobReg, earnService(a))
	})
}
