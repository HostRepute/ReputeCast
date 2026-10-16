// Package growth implements Audience › Growth (audience-insights brief
// §3.4, M23): how contacts first joined and whether they left, by source
// (each form, the subscribe page, each import, API key and integration,
// people adding contacts; later referrals and recommendations), with the
// first touch's UTM parameters and referring page, monthly cohort
// retention, and bounce, complaint and human-open rates per source.
//
// Moving a list isn't growth: imports are counted apart unless asked for.
package growth

import (
	"context"
	"log/slog"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/store"
)

// Service is the growth domain.
type Service struct {
	Pool  *pgxpool.Pool
	Q     *store.Queries
	Audit *audit.Logger
	Log   *slog.Logger
	Now   func() time.Time
	// SubscribeURL is the address of a workspace's subscribe page
	// ("https://news.acme.io/subscribe"), for the source's second line.
	SubscribeURL func(ctx context.Context, ws *store.Workspace) string
}

func (s *Service) now() time.Time {
	if s.Now != nil {
		return s.Now()
	}
	return time.Now()
}

func (s *Service) log() *slog.Logger {
	if s.Log != nil {
		return s.Log
	}
	return slog.Default()
}

// Source kinds of the growth registry.
const (
	KindForm             = "form"              // a signup form (embedded, popup or hosted)
	KindSubscribePage    = "subscribe_page"    // the built-in subscribe page
	KindImport           = "import"            // an import that gave contacts the import day as their join date
	KindCarried          = "carried"           // the source an import carried over from the old platform
	KindAPIKey           = "api_key"           // an API key
	KindIntegration      = "integration"       // an integration, with the workflow that calls it
	KindPerson           = "person"            // added by someone in ReputeCast
	KindPreferenceCenter = "preference_center" // a list added in the preference center (one-list view)
	KindUnknown          = "unknown"
)

// Kind is a source kind the registry knows. Later phases (referrals,
// recommendations) register theirs.
type Kind struct {
	Key string
	// Imported kinds are counted apart from growth unless "Include
	// imports" is on.
	Imported bool
	// Order groups sources in the table (forms first, imports last).
	Order int
}

var (
	kindsMu sync.RWMutex
	kinds   = map[string]Kind{
		KindForm:             {Key: KindForm, Order: 10},
		KindSubscribePage:    {Key: KindSubscribePage, Order: 10},
		KindAPIKey:           {Key: KindAPIKey, Order: 20},
		KindIntegration:      {Key: KindIntegration, Order: 30},
		KindPerson:           {Key: KindPerson, Order: 40},
		KindPreferenceCenter: {Key: KindPreferenceCenter, Order: 45},
		KindImport:           {Key: KindImport, Imported: true, Order: 50},
		KindCarried:          {Key: KindCarried, Order: 60},
		KindUnknown:          {Key: KindUnknown, Order: 70},
	}
)

// RegisterKind adds a source kind (phase 5: referral, recommendation).
func RegisterKind(k Kind) {
	kindsMu.Lock()
	defer kindsMu.Unlock()
	kinds[k.Key] = k
}

func kindOf(key string) Kind {
	kindsMu.RLock()
	defer kindsMu.RUnlock()
	if k, ok := kinds[key]; ok {
		return k
	}
	return Kind{Key: key, Order: 65}
}

// SourceKey joins a kind and an id: "form:01a1…", "person".
func SourceKey(kind, id string) string {
	if id == "" {
		return kind
	}
	return kind + ":" + id
}

// splitKey reverses SourceKey.
func splitKey(key string) (kind, id string) {
	kind, id, _ = strings.Cut(key, ":")
	return kind, id
}

// GrowthSource names a source and says what the app needs to describe it.
// Names are workspace content (a form's or key's name); the app writes the
// words around them.
type GrowthSource struct {
	Key  string `json:"key"`
	Kind string `json:"kind"`
	// Name is the form, API key, integration or import platform.
	Name string `json:"name"`
	// Imported sources are counted apart from growth.
	Imported bool `json:"imported"`
	// FormKind is embedded, popup or hosted.
	FormKind string `json:"formKind,omitempty"`
	// Hosts are the sites a form's signups came from in the period.
	Hosts []string `json:"hosts"`
	// URL is the subscribe page's or a hosted form's address.
	URL string `json:"url,omitempty"`
	// KeyPrefix is an API key's masked prefix ("rc_live_8f2c…a91d").
	KeyPrefix string `json:"keyPrefix,omitempty"`
	// Integration and Workflow name an integration source ("Zapier",
	// "Beta application (Typeform)").
	Integration string `json:"integration,omitempty"`
	Workflow    string `json:"workflow,omitempty"`
	// ImportedAt and ImportCount describe an import; TookImportDay is how
	// many of its contacts got the import day as their join date (the
	// others kept the date and source they came with).
	ImportedAt    *time.Time `json:"importedAt,omitempty"`
	ImportCount   int64      `json:"importCount,omitempty"`
	TookImportDay int64      `json:"tookImportDay,omitempty"`
	// Carried is the source an import kept ("import:csv", "form:docs");
	// CarriedLabel says it in words.
	Carried      string `json:"carried,omitempty"`
	CarriedLabel string `json:"carriedLabel,omitempty"`
	// FirstJoin is the source's first join ever.
	FirstJoin *time.Time `json:"firstJoin,omitempty"`
	Order     int        `json:"-"`
}

// names resolves what sources point at, once per request.
type names struct {
	forms        map[string]store.GrowthFormsOfRow
	formsBySlug  map[string]store.GrowthFormsOfRow
	imports      map[string]store.GrowthImportsOfRow
	keys         map[string]store.GrowthKeysOfRow
	integrations map[string]store.GrowthIntegrationsOfRow
	workflows    map[string]string // integration id + ":" + key → name
	subscribeURL string
}

func (s *Service) names(ctx context.Context, q *store.Queries, ws *store.Workspace) (*names, error) {
	n := &names{forms: map[string]store.GrowthFormsOfRow{}, formsBySlug: map[string]store.GrowthFormsOfRow{}, imports: map[string]store.GrowthImportsOfRow{},
		keys: map[string]store.GrowthKeysOfRow{}, integrations: map[string]store.GrowthIntegrationsOfRow{}, workflows: map[string]string{}}
	forms, err := q.GrowthFormsOf(ctx, ws.ID)
	if err != nil {
		return nil, err
	}
	for _, f := range forms {
		n.forms[f.ID.String()] = f
		n.formsBySlug[f.Slug] = f
	}
	imports, err := q.GrowthImportsOf(ctx, ws.ID)
	if err != nil {
		return nil, err
	}
	for _, i := range imports {
		n.imports[i.ID.String()] = i
	}
	keys, err := q.GrowthKeysOf(ctx, ws.ID)
	if err != nil {
		return nil, err
	}
	for _, k := range keys {
		n.keys[k.ID.String()] = k
	}
	ints, err := q.GrowthIntegrationsOf(ctx, ws.ID)
	if err != nil {
		return nil, err
	}
	for _, i := range ints {
		n.integrations[i.ID.String()] = i
	}
	wfs, err := q.GrowthWorkflowsOf(ctx, ws.ID)
	if err != nil {
		return nil, err
	}
	for _, w := range wfs {
		n.workflows[w.IntegrationID.String()+":"+w.Key] = w.Name
	}
	if s.SubscribeURL != nil {
		n.subscribeURL = s.SubscribeURL(ctx, ws)
	}
	return n, nil
}

// platformName is an importer's name as people write it.
func platformName(p string) string {
	switch p {
	case "brevo":
		return "Brevo"
	case "mailchimp":
		return "Mailchimp"
	case "mailerlite":
		return "MailerLite"
	case "listmonk":
		return "listmonk"
	case "keila":
		return "Keila"
	case "notifuse":
		return "Notifuse"
	case "csv":
		return "CSV"
	case "xlsx":
		return "XLSX"
	}
	if p == "" {
		return "File"
	}
	return strings.ToUpper(p[:1]) + p[1:]
}

var integrationNames = map[string]string{"zapier": "Zapier", "make": "Make", "n8n": "n8n", "wordpress": "WordPress"}

// workflowShort is a workflow's name without what it does in ReputeCast:
// "Beta application (Typeform) → ReputeCast contact" → "Beta application (Typeform)".
func workflowShort(name string) string {
	if i := strings.Index(name, " → "); i > 0 {
		return strings.TrimSpace(name[:i])
	}
	return name
}

// describe resolves a source key.
func (n *names) describe(key string) GrowthSource {
	kind, id := splitKey(key)
	k := kindOf(kind)
	out := GrowthSource{Key: key, Kind: kind, Imported: k.Imported, Order: k.Order, Hosts: []string{}}
	switch kind {
	case KindForm:
		if f, ok := n.forms[id]; ok {
			out.Name, out.FormKind = f.Name, f.Kind
		}
	case KindSubscribePage:
		out.URL = n.subscribeURL
	case KindAPIKey:
		if k, ok := n.keys[id]; ok {
			out.Name, out.KeyPrefix = k.Name, k.Prefix
		}
	case KindIntegration:
		intID, wf, _ := strings.Cut(id, ":")
		if i, ok := n.integrations[intID]; ok {
			out.Integration = integrationNames[i.Kind]
			if out.Integration == "" {
				out.Integration = i.Kind
			}
			if i.Kind == "wordpress" && i.Label != "" {
				out.Integration += " · " + i.Label
			}
		}
		if name := n.workflows[intID+":"+wf]; name != "" {
			out.Workflow = workflowShort(name)
		}
		out.Name = out.Integration
		if out.Workflow != "" {
			out.Name += " · " + out.Workflow
		}
	case KindImport:
		if i, ok := n.imports[id]; ok {
			out.Name = platformName(i.Source)
			at := i.CreatedAt
			if i.FinishedAt != nil {
				at = *i.FinishedAt
			}
			out.ImportedAt = &at
			out.ImportCount = i.Imported
		}
	case KindCarried:
		out.Carried = id
		ck, cid, _ := strings.Cut(id, ":")
		switch ck {
		case "import":
			out.Name = platformName(cid)
		case "form":
			if f, ok := n.formsBySlug[cid]; ok {
				out.Name = f.Name
			} else {
				out.Name = cid
			}
		default:
			out.Name = id
		}
	}
	return out
}

// sortSources orders sources as the table lists them: forms and the
// subscribe page, API keys, integrations, people, imports (newest first),
// then the sources imports carried; biggest first within each group.
func sortSources(rows []GrowthSourceRow) {
	slices.SortStableFunc(rows, func(a, b GrowthSourceRow) int {
		if a.Source.Order != b.Source.Order {
			return a.Source.Order - b.Source.Order
		}
		if a.Source.Kind == KindImport && b.Source.Kind == KindImport && a.Source.ImportedAt != nil && b.Source.ImportedAt != nil {
			return b.Source.ImportedAt.Compare(*a.Source.ImportedAt)
		}
		if a.sortJoined != b.sortJoined {
			return int(b.sortJoined - a.sortJoined)
		}
		return strings.Compare(a.Source.Name, b.Source.Name)
	})
}

func parseUUID(s string) (uuid.UUID, bool) {
	id, err := uuid.Parse(s)
	return id, err == nil
}
