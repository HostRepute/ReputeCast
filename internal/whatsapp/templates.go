package whatsapp

import (
	"context"
	"encoding/json"
	"errors"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/notify"
	"github.com/hostrepute/reputecast/internal/store"
	"github.com/hostrepute/reputecast/internal/webhooks"
	"github.com/hostrepute/reputecast/internal/whatsapp/meta"
)

// Templates (brief §3.9, board WhatsAppChannel › Templates): synced from
// Meta every hour and on demand, with their category, language, body and
// variables, buttons and status (Approved, Pending, Rejected with Meta's
// reason, Paused, Disabled). They're created and edited in Meta's
// WhatsApp Manager.

// WhatsAppTemplateVar is a template variable: {{1}}, its label and Meta's
// sample value.
type WhatsAppTemplateVar struct {
	Key     string `json:"key"`
	Label   string `json:"label"`
	Example string `json:"example,omitempty"`
}

// WhatsAppTemplateButton is a template's button.
type WhatsAppTemplateButton struct {
	Type string `json:"type" enum:"quick_reply,url,phone_number,other"`
	Text string `json:"text"`
	URL  string `json:"url,omitempty"`
}

// WhatsAppTemplate is a template as the Templates tab and the pickers show it.
type WhatsAppTemplate struct {
	ID            string                   `json:"id"`
	Name          string                   `json:"name"`
	Language      string                   `json:"language"`
	LanguageLabel string                   `json:"languageLabel"`
	Category      string                   `json:"category" enum:"marketing,utility,authentication"`
	Status        string                   `json:"status" enum:"approved,pending,rejected,paused,disabled,in_appeal,other"`
	StatusSince   *time.Time               `json:"statusSince,omitempty"`
	Reason        string                   `json:"reason,omitempty"`
	ReasonCode    string                   `json:"reasonCode,omitempty"`
	Body          string                   `json:"body"`
	Variables     []WhatsAppTemplateVar    `json:"variables"`
	Buttons       []WhatsAppTemplateButton `json:"buttons"`
	SyncedAt      time.Time                `json:"syncedAt"`
}

var languageLabels = map[string]string{"en_US": "English (US)", "en_GB": "English (UK)", "en": "English", "de": "German", "fr": "French", "es": "Spanish",
	"es_MX": "Spanish (MEX)", "es_ES": "Spanish (SPA)", "nl": "Dutch", "it": "Italian", "pt_BR": "Portuguese (BR)", "pt_PT": "Portuguese (POR)", "vi": "Vietnamese",
	"ja": "Japanese", "ko": "Korean", "zh_CN": "Chinese (CHN)", "ga": "Irish", "pl": "Polish", "sv": "Swedish", "da": "Danish", "nb": "Norwegian", "fi": "Finnish"}

// LanguageLabel names a template's language: "English (US)".
func LanguageLabel(code string) string {
	if l, ok := languageLabels[code]; ok {
		return l
	}
	return code
}

// reasonWords are Meta's rejection codes in words, used until Meta's own
// words arrive with a status webhook.
var reasonWords = map[string]string{
	"INVALID_FORMAT":       "The template's format isn't valid.",
	"ABUSIVE_CONTENT":      "Meta found content that breaks its policies.",
	"INCORRECT_CATEGORY":   "The category doesn't match the content.",
	"TAG_CONTENT_MISMATCH": "The content doesn't match the category.",
	"SCAM":                 "Meta flagged the content as a possible scam.",
	"PROMOTIONAL":          "The content is promotional: use the Marketing category.",
}

func templateView(t store.WhatsappTemplate) WhatsAppTemplate {
	out := WhatsAppTemplate{ID: t.ID.String(), Name: t.Name, Language: t.Language, LanguageLabel: LanguageLabel(t.Language), Category: t.Category,
		Status: t.Status, StatusSince: t.StatusSince, Reason: t.Reason, ReasonCode: t.ReasonCode, Body: t.Body, Variables: []WhatsAppTemplateVar{},
		Buttons: []WhatsAppTemplateButton{}, SyncedAt: t.SyncedAt}
	_ = json.Unmarshal(t.Variables, &out.Variables)
	_ = json.Unmarshal(t.Buttons, &out.Buttons)
	if out.Variables == nil {
		out.Variables = []WhatsAppTemplateVar{}
	}
	if out.Buttons == nil {
		out.Buttons = []WhatsAppTemplateButton{}
	}
	if out.Reason == "" && t.Status == "rejected" {
		out.Reason = reasonWords[strings.ToUpper(t.ReasonCode)]
	}
	return out
}

var varRe = regexp.MustCompile(`\{\{\s*(\d{1,2})\s*\}\}`)

// variablesOf lists a body's variables in order: {{1}}, {{2}}…
func variablesOf(body string) []string {
	seen := map[int]bool{}
	nums := []int{}
	for _, m := range varRe.FindAllStringSubmatch(body, -1) {
		n, _ := strconv.Atoi(m[1])
		if !seen[n] {
			seen[n] = true
			nums = append(nums, n)
		}
	}
	sort.Ints(nums)
	out := make([]string, 0, len(nums))
	for _, n := range nums {
		out = append(out, strconv.Itoa(n))
	}
	return out
}

// render fills a body's variables with params, in order ({{1}} is
// params[0]); a missing value leaves the variable as it is.
func render(body string, params []string) string {
	return varRe.ReplaceAllStringFunc(body, func(m string) string {
		n, _ := strconv.Atoi(varRe.FindStringSubmatch(m)[1])
		if n >= 1 && n <= len(params) {
			return params[n-1]
		}
		return m
	})
}

// exampleVars are a template's sample values, for tests.
func exampleVars(t store.WhatsappTemplate) []string {
	var vars []WhatsAppTemplateVar
	_ = json.Unmarshal(t.Variables, &vars)
	keys := variablesOf(t.Body)
	out := make([]string, 0, len(keys))
	for i := range keys {
		v := ""
		if i < len(vars) {
			v = vars[i].Example
			if v == "" {
				v = vars[i].Label
			}
		}
		if v == "" {
			v = "test"
		}
		out = append(out, v)
	}
	return out
}

func statusOf(s string) string {
	switch strings.ToUpper(s) {
	case "APPROVED", "REINSTATED":
		return "approved"
	case "PENDING", "PENDING_DELETION", "IN_REVIEW":
		return "pending"
	case "REJECTED":
		return "rejected"
	case "PAUSED", "FLAGGED":
		return "paused"
	case "DISABLED":
		return "disabled"
	case "IN_APPEAL":
		return "in_appeal"
	}
	return "other"
}

func categoryOf(s string) string {
	switch strings.ToUpper(s) {
	case "MARKETING":
		return "marketing"
	case "AUTHENTICATION":
		return "authentication"
	}
	return "utility"
}

func buttonsOf(t meta.Template) []WhatsAppTemplateButton {
	out := []WhatsAppTemplateButton{}
	for _, b := range t.Buttons() {
		typ := "other"
		switch strings.ToUpper(b.Type) {
		case "QUICK_REPLY":
			typ = "quick_reply"
		case "URL":
			typ = "url"
		case "PHONE_NUMBER":
			typ = "phone_number"
		}
		out = append(out, WhatsAppTemplateButton{Type: typ, Text: b.Text, URL: b.URL})
	}
	return out
}

// WhatsAppSyncResult is what a sync found.
type WhatsAppSyncResult struct {
	Templates int       `json:"templates"`
	Added     []string  `json:"added"`
	Changed   []string  `json:"changed"`
	Removed   []string  `json:"removed"`
	SyncedAt  time.Time `json:"syncedAt"`
}

// Sync reads the account's templates from Meta (Sync now, the hourly job).
// A sync that changed something is logged: by the person who asked, or by
// the system. A newly rejected template tells the providers' managers.
func (s *Service) Sync(ctx context.Context, ws *store.Workspace, p *authz.Principal, system bool) (WhatsAppSyncResult, error) {
	out := WhatsAppSyncResult{Added: []string{}, Changed: []string{}, Removed: []string{}}
	a, err := s.account(ctx, s.Q, ws)
	if errors.Is(err, errNoAccount) {
		return out, api.Conflict("Connect WhatsApp first.")
	}
	if err != nil {
		return out, err
	}
	c, err := s.client(a.ID, a.CredentialsEnc, a.Demo)
	if err != nil {
		return out, err
	}
	list, err := c.Templates(ctx)
	if err != nil {
		var me *meta.Error
		if errors.As(err, &me) && me.TokenRejected() {
			_ = s.tokenRejected(ctx, ws, accountOf(a), me)
			if !system {
				return out, api.Err(502, "token_rejected", "Meta rejected the access token. Replace it to sync the templates.")
			}
		}
		if !system {
			return out, api.Err(502, "meta_unavailable", "Meta's Graph API didn't answer. Try again in a moment.")
		}
		return out, err
	}
	before, err := s.Q.WhatsAppTemplates(ctx, a.ID)
	if err != nil {
		return out, err
	}
	prev := map[string]store.WhatsappTemplate{}
	for _, t := range before {
		prev[t.Name+"/"+t.Language] = t
	}
	now := s.now()
	rejected := []string{}
	err = s.tx(ctx, func(q *store.Queries, tx pgx.Tx) error {
		keep := []uuid.UUID{}
		for _, t := range list {
			body := t.Body()
			keys := variablesOf(body)
			examples := t.Examples()
			vars := make([]WhatsAppTemplateVar, 0, len(keys))
			for i, k := range keys {
				v := WhatsAppTemplateVar{Key: k}
				if i < len(examples) {
					v.Example = examples[i]
				}
				vars = append(vars, v)
			}
			status := statusOf(t.Status)
			old, had := prev[t.Name+"/"+t.Language]
			id := uuid.Must(uuid.NewV7())
			if had {
				id = old.ID
			}
			reason := ""
			if status == "rejected" {
				reason = reasonWords[strings.ToUpper(t.RejectedReason)]
			}
			code := ""
			if status == "rejected" && !strings.EqualFold(t.RejectedReason, "NONE") {
				code = strings.ToUpper(t.RejectedReason)
			}
			if err := q.WhatsAppTemplateUpsert(ctx, store.WhatsAppTemplateUpsertParams{ID: id, WorkspaceID: ws.ID, AccountID: a.ID, MetaID: t.ID, Name: t.Name,
				Language: t.Language, Category: categoryOf(t.Category), Status: status, At: &now, ReasonCode: code, Reason: reason, Body: body,
				Variables: jsonOf(vars), Buttons: jsonOf(buttonsOf(t))}); err != nil {
				return err
			}
			cur, err := q.WhatsAppTemplateByName(ctx, store.WhatsAppTemplateByNameParams{AccountID: a.ID, Name: t.Name, Language: t.Language})
			if err != nil {
				return err
			}
			keep = append(keep, cur.ID)
			switch {
			case !had:
				out.Added = append(out.Added, t.Name)
			case old.Status != status:
				out.Changed = append(out.Changed, t.Name+": "+old.Status+" → "+status)
			}
			if status == "rejected" && (!had || old.Status != "rejected") {
				rejected = append(rejected, t.Name)
			}
		}
		for _, t := range before {
			if !containsID(keep, t.ID) {
				out.Removed = append(out.Removed, t.Name)
			}
		}
		if err := q.WhatsAppTemplatesRemoveMissing(ctx, store.WhatsAppTemplatesRemoveMissingParams{At: &now, AccountID: a.ID, Keep: keep}); err != nil {
			return err
		}
		if err := q.WhatsAppAccountSynced(ctx, store.WhatsAppAccountSyncedParams{At: &now, ID: a.ID}); err != nil {
			return err
		}
		out.Templates, out.SyncedAt = len(list), now
		if len(out.Added)+len(out.Changed)+len(out.Removed) == 0 && system {
			return nil
		}
		ev := audit.Event{WorkspaceID: &ws.ID, Action: "whatsapp_templates.synced", TargetType: "whatsapp_account", TargetID: a.ID.String(), TargetName: a.BusinessName,
			Metadata: map[string]any{"templates": len(list), "added": out.Added, "changed": out.Changed, "removed": out.Removed}}
		if system || p == nil {
			ev.Actor = systemActor
		}
		for _, name := range rejected {
			if _, err := webhooks.Emit(ctx, tx, ws.ID, ws.Slug, "whatsapp.template_status", map[string]any{"template": name, "status": "rejected"}); err != nil {
				return err
			}
		}
		return s.Audit.LogWith(ctx, q, ev)
	})
	if err != nil {
		return out, err
	}
	for _, name := range rejected {
		s.notifyRejected(ctx, ws, a.ID, name)
	}
	return out, nil
}

func containsID(ids []uuid.UUID, id uuid.UUID) bool {
	for _, x := range ids {
		if x == id {
			return true
		}
	}
	return false
}

// notifyRejected tells the providers' managers Meta rejected a template.
func (s *Service) notifyRejected(ctx context.Context, ws *store.Workspace, accountID uuid.UUID, name string) {
	t, err := s.Q.WhatsAppTemplateByName(ctx, store.WhatsAppTemplateByNameParams{AccountID: accountID, Name: name})
	reason := ""
	if err == nil {
		reason = templateView(t).Reason
	}
	summary := "Meta rejected " + name
	if reason != "" {
		summary += ": “" + reason + "”"
	}
	if err := notify.ToMembersWith(ctx, s.Pool, ws.ID, authz.ProvidersView, notify.Notification{Kind: notify.KindWhatsAppAttention, Severity: "warn",
		Title: "WhatsApp template " + name + " rejected", Summary: summary,
		Body: "Fix the template in WhatsApp Manager and submit it again; ReputeCast picks up the new status at the next sync.",
		Link: "/w/" + ws.Slug + "/providers/whatsapp/templates", Data: map[string]any{"template": name}}); err != nil {
		s.log().WarnContext(ctx, "whatsapp template notification", "err", err)
	}
}

// accountOf turns the account row into the table's row.
func accountOf(a store.WhatsAppAccountRow) store.WhatsappAccount {
	return store.WhatsappAccount{ID: a.ID, WorkspaceID: a.WorkspaceID, CredentialsEnc: a.CredentialsEnc, BusinessHint: a.BusinessHint, BusinessName: a.BusinessName,
		PhoneNumber: a.PhoneNumber, DisplayName: a.DisplayName, NameStatus: a.NameStatus, PhoneStatus: a.PhoneStatus, Quality: a.Quality,
		PreviousQuality: a.PreviousQuality, QualityChangedAt: a.QualityChangedAt, MessagingLimit: a.MessagingLimit, PreviousLimit: a.PreviousLimit,
		LimitChangedAt: a.LimitChangedAt, VerifyToken: a.VerifyToken, VerifiedAt: a.VerifiedAt, LastEventAt: a.LastEventAt, Status: a.Status,
		StatusDetail: a.StatusDetail, StatusSince: a.StatusSince, TemplatesSyncedAt: a.TemplatesSyncedAt, CheckedAt: a.CheckedAt, Demo: a.Demo,
		ConnectedBy: a.ConnectedBy, ConnectedAt: a.ConnectedAt, RemovedAt: a.RemovedAt, RemovedBy: a.RemovedBy}
}

// Templates lists the account's templates.
func (s *Service) Templates(ctx context.Context, ws *store.Workspace) ([]WhatsAppTemplate, error) {
	out := []WhatsAppTemplate{}
	a, err := s.account(ctx, s.Q, ws)
	if errors.Is(err, errNoAccount) {
		return out, nil
	}
	if err != nil {
		return out, err
	}
	ts, err := s.Q.WhatsAppTemplates(ctx, a.ID)
	if err != nil {
		return out, err
	}
	for _, t := range ts {
		out = append(out, templateView(t))
	}
	return out, nil
}

// applyTemplateStatus records a template status from Meta's webhook.
func (s *Service) applyTemplateStatus(ctx context.Context, ws *store.Workspace, a store.WhatsappAccount, e meta.Event) error {
	status := statusOf(e.TemplateEvent)
	t, err := s.Q.WhatsAppTemplateByName(ctx, store.WhatsAppTemplateByNameParams{AccountID: a.ID, Name: e.TemplateName, Language: e.TemplateLanguage})
	if err != nil {
		if notFound(err) {
			// A template the last sync didn't see: the next one brings it.
			return nil
		}
		return err
	}
	now := s.now()
	code := ""
	if status == "rejected" && !strings.EqualFold(e.Reason, "NONE") {
		code = strings.ToUpper(e.Reason)
	}
	words := strings.TrimSpace(e.ReasonText)
	if words == "" && status == "rejected" {
		words = reasonWords[code]
	}
	err = s.tx(ctx, func(q *store.Queries, tx pgx.Tx) error {
		if err := q.WhatsAppTemplateSetStatus(ctx, store.WhatsAppTemplateSetStatusParams{Status: status, At: now, ReasonCode: code, Reason: words, AccountID: a.ID,
			Name: t.Name, Language: t.Language}); err != nil {
			return err
		}
		if t.Status == status {
			return nil
		}
		if _, err := webhooks.Emit(ctx, tx, ws.ID, ws.Slug, "whatsapp.template_status", map[string]any{"template": t.Name, "language": t.Language,
			"status": status, "reason": words}); err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "whatsapp_templates.synced", TargetType: "whatsapp_account", TargetID: a.ID.String(),
			TargetName: a.BusinessName, Actor: systemActor, Metadata: map[string]any{"changed": []string{t.Name + ": " + t.Status + " → " + status}, "source": "webhook"}})
	})
	if err != nil {
		return err
	}
	if status == "rejected" && t.Status != "rejected" {
		s.notifyRejected(ctx, ws, a.ID, t.Name)
	}
	return nil
}
