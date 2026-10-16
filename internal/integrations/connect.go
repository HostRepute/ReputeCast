package integrations

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/access"
	"github.com/hostrepute/reputecast/internal/accounts"
	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/authn"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/store"
)

// RotationGrace is how long a rotated integration key keeps working.
const RotationGrace = 24 * time.Hour

// IntegrationConnectInput connects an integration: what it may do, its key's
// name, list scope and expiry.
type IntegrationConnectInput struct {
	Items     []string   `json:"items,omitempty" maxItems:"20" doc:"Catalog keys of the triggers and actions turned on"`
	Name      string     `json:"name,omitempty" maxLength:"80" doc:"The key's name, \"Make · Acme ops\""`
	Lists     []string   `json:"lists,omitempty" maxItems:"100" doc:"Limit the key to these lists; empty for all lists"`
	ExpiresAt *time.Time `json:"expiresAt,omitempty"`
	// Address is n8n's address (to open it and check that ReputeCast can
	// reach it), or the WordPress site's.
	Address string `json:"address,omitempty" maxLength:"300"`
}

// IntegrationSecret carries a new key, shown once.
type IntegrationSecret struct {
	IntegrationID string             `json:"integrationId"`
	Kind          string             `json:"kind"`
	Key           IntegrationKeyView `json:"key"`
	Secret        string             `json:"secret" doc:"The full key. Shown only in this response."`
	InstanceURL   string             `json:"instanceUrl" doc:"Paste it into the other tool with the key"`
	// OldKeyUntil is when a rotated key stops working.
	OldKeyUntil *time.Time `json:"oldKeyUntil,omitempty"`
}

// IntegrationKeyView is an integration's key, without its secret.
type IntegrationKeyView struct {
	ID          string              `json:"id"`
	Name        string              `json:"name"`
	Masked      string              `json:"masked" example:"rc_live_5d1e…7b20"`
	Permissions []string            `json:"permissions" doc:"As held: implied view permissions left out"`
	Sensitive   []string            `json:"sensitive"`
	Scope       access.ScopeSummary `json:"scope"`
	Owner       IntegrationPerson   `json:"owner"`
	// OwnerStatus is active, deactivated or removed.
	OwnerStatus string     `json:"ownerStatus" enum:"active,deactivated,removed"`
	OwnerLeftAt *time.Time `json:"ownerLeftAt,omitempty"`
	CreatedAt   time.Time  `json:"createdAt"`
	ExpiresAt   *time.Time `json:"expiresAt,omitempty"`
	LastUsedAt  *time.Time `json:"lastUsedAt,omitempty"`
	LastUsedIP  string     `json:"lastUsedIp,omitempty"`
	// LastUsedClient is the app ("Zapier, ReputeCast app 1.2.0").
	LastUsedClient string     `json:"lastUsedClient,omitempty"`
	RevokedAt      *time.Time `json:"revokedAt,omitempty"`
	GraceUntil     *time.Time `json:"graceUntil,omitempty" doc:"A rotated key keeps working until then"`
	// Status: active, grace (rotated, still working), revoked or expired.
	Status string `json:"status" enum:"active,grace,revoked,expired"`
}

// IntegrationPerson is a member named on an integration.
type IntegrationPerson struct {
	ID          string `json:"id,omitempty"`
	Name        string `json:"name"`
	AvatarColor string `json:"avatarColor,omitempty"`
}

func keyView(r store.IntegrationKeysRow, lists map[uuid.UUID]string, now time.Time) IntegrationKeyView {
	v := IntegrationKeyView{
		ID: r.ID.String(), Name: r.Name, Masked: r.Prefix, Permissions: shortPermissions(r.Permissions), Sensitive: []string{},
		Scope: access.View(access.ParseScope(r.Scope), lists, false), Owner: IntegrationPerson{Name: r.OwnerName, AvatarColor: r.OwnerColor},
		OwnerStatus: "active", CreatedAt: r.CreatedAt, ExpiresAt: r.ExpiresAt, LastUsedAt: r.LastUsedAt, LastUsedClient: r.LastUsedClient,
		RevokedAt: r.RevokedAt, GraceUntil: r.GraceUntil, Status: "active",
	}
	if r.CreatedBy != nil {
		v.Owner.ID = r.CreatedBy.String()
	}
	if v.Owner.Name == "" {
		v.Owner.Name = "Former member"
		v.Owner.AvatarColor = "neutral"
	}
	switch r.OwnerStatus {
	case "active":
	case "":
		v.OwnerStatus = "removed"
	default:
		v.OwnerStatus = "deactivated"
		v.OwnerLeftAt = r.OwnerLeftAt
	}
	if r.LastUsedIp != nil {
		v.LastUsedIP = r.LastUsedIp.String()
	}
	for _, p := range v.Permissions {
		if authz.IsSensitive(authz.Permission(p)) {
			v.Sensitive = append(v.Sensitive, p)
		}
	}
	switch {
	case r.RevokedAt != nil && r.GraceUntil != nil && r.GraceUntil.After(now):
		v.Status = "grace"
	case r.RevokedAt != nil:
		v.Status = "revoked"
	case r.ExpiresAt != nil && !r.ExpiresAt.After(now):
		v.Status = "expired"
	}
	return v
}

// shortPermissions leaves out permissions another one in the list includes
// (subscribers:edit includes subscribers:view).
func shortPermissions(keys []string) []string {
	out := []string{}
	for _, k := range keys {
		p := authz.Permission(k)
		if !authz.Valid(p) || slices.Contains(out, k) {
			continue
		}
		implied := false
		for _, o := range keys {
			if o != k && authz.Valid(authz.Permission(o)) && authz.NewSet(authz.Permission(o)).Has(p) {
				implied = true
			}
		}
		if !implied {
			out = append(out, k)
		}
	}
	return out
}

// storedPermissions is what goes on the key: the items' permissions, then
// what they include, as the Keys page expects.
func storedPermissions(kind string, items []string) []string {
	short := Permissions(kind, items)
	out := []string{}
	for _, p := range short {
		out = append(out, string(p))
	}
	for _, p := range authz.NewSet(short...).Strings() {
		if !slices.Contains(out, p) {
			out = append(out, p)
		}
	}
	return out
}

// checkItems validates the items turned on and applies the escalation guard
// to the ones being added: nobody can give a key what they don't hold.
func (s *Service) checkItems(ctx context.Context, ws *store.Workspace, p *authz.Principal, kind string, items, before []string) ([]string, error) {
	out := []string{}
	for _, key := range items {
		it, ok := ItemByKey(kind, key)
		if !ok {
			return nil, api.Invalid("body.items", fmt.Sprintf("%s can't do %q.", Name(kind), key))
		}
		if slices.Contains(out, key) {
			continue
		}
		if !slices.Contains(before, key) {
			for _, need := range Needs(it) {
				if !p.Can(need) {
					pr := api.Err(http.StatusForbidden, api.CodeEscalation, access.CantGrant(need, access.Helpers(ctx, s.Q, ws.ID, need)))
					pr.Permission = string(need)
					return nil, pr
				}
			}
		}
		out = append(out, key)
	}
	if len(out) == 0 {
		return nil, api.Invalid("body.items", fmt.Sprintf("Turn on at least one thing %s can do.", Name(kind)))
	}
	// Keep the catalog's order.
	ordered := []string{}
	for _, it := range Items(kind) {
		if slices.Contains(out, it.Key) {
			ordered = append(ordered, it.Key)
		}
	}
	return ordered, nil
}

// offItems are the catalog items not turned on.
func offItems(kind string, items []string) []string {
	out := []string{}
	for _, it := range Items(kind) {
		if !slices.Contains(items, it.Key) {
			out = append(out, it.Key)
		}
	}
	return out
}

// keyName returns "{Integration} · {label}" and the label.
func keyName(kind, name, fallback string) (string, string) {
	prefix := Name(kind) + " · "
	name = strings.TrimSpace(name)
	label := strings.TrimSpace(strings.TrimPrefix(name, prefix))
	if label == "" {
		label = fallback
	}
	return prefix + label, label
}

// DefaultLabel suggests the label of a new key: the one the last automation
// platform was connected with ("Acme ops"), else the workspace's name.
func (s *Service) DefaultLabel(ctx context.Context, ws *store.Workspace) string {
	if l, err := s.Q.IntegrationLastLabel(ctx, ws.ID); err == nil && l != "" {
		return l
	}
	return ws.Name
}

// siteHost normalizes a WordPress site's address to its host, port and path
// ("blog.acme.io", "acme.io/news"). https is assumed; a site typed with
// http:// keeps it.
func siteHost(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", errors.New("enter the site's address, like blog.acme.io")
	}
	if !strings.Contains(raw, "://") {
		raw = "https://" + raw
	}
	u, err := url.Parse(raw)
	if err != nil || u.Hostname() == "" || strings.Contains(u.Hostname(), " ") || (u.Scheme != "https" && u.Scheme != "http") {
		return "", errors.New("enter the site's address, like blog.acme.io")
	}
	host := strings.ToLower(u.Host)
	if p := strings.Trim(u.EscapedPath(), "/"); p != "" {
		host += "/" + p
	}
	if u.Scheme == "http" {
		host = "http://" + host
	}
	return host, nil
}

// siteBase is the address "Check now" calls: https unless typed with http.
func siteBase(address string) string {
	if strings.HasPrefix(address, "http://") || strings.HasPrefix(address, "https://") {
		return strings.TrimRight(address, "/")
	}
	return "https://" + strings.TrimRight(address, "/")
}

// Connect creates an integration's key with the triggers and actions turned
// on. The key is shown once; the integration waits for its first call.
func (s *Service) Connect(ctx context.Context, ws *store.Workspace, p *authz.Principal, kind string, in IntegrationConnectInput) (IntegrationSecret, error) {
	if !Valid(kind) {
		return IntegrationSecret{}, api.NotFound("Integration")
	}
	if kind == KindZapier || kind == KindMake {
		if r := s.Reachability(ctx); !r.Reachable {
			return IntegrationSecret{}, api.Err(http.StatusConflict, "instance_unreachable", fmt.Sprintf(
				"%s can't reach %s, your public URL. Give the instance a public address (Instance admin › Instance & updates), or use n8n on your own network.", Name(kind), r.URL))
		}
	}
	items, err := s.checkItems(ctx, ws, p, kind, in.Items, nil)
	if err != nil {
		return IntegrationSecret{}, err
	}
	settings := map[string]any{}
	var label, address string
	if kind == KindWordPress {
		if address, err = siteHost(in.Address); err != nil {
			return IntegrationSecret{}, api.Invalid("body.address", capitalize(err.Error())+".")
		}
		label = address
		in.Name = Name(kind) + " · " + address
	} else {
		if existing, err := s.Q.IntegrationActiveByKind(ctx, store.IntegrationActiveByKindParams{WorkspaceID: ws.ID, Kind: kind}); err == nil {
			return IntegrationSecret{}, api.Err(http.StatusConflict, api.CodeConflict, fmt.Sprintf(
				"%s is already connected (%s). Change what it can do, rotate its key, or disconnect it first.", Name(kind), existing.Label))
		}
		if kind == KindN8n && strings.TrimSpace(in.Address) != "" {
			a, err := normalizeAddress(in.Address)
			if err != nil {
				return IntegrationSecret{}, api.Invalid("body.address", "Enter n8n's address, like https://n8n.acme.io.")
			}
			settings["address"] = a
		}
	}
	name, l := keyName(kind, in.Name, s.DefaultLabel(ctx, ws))
	if label == "" {
		label = l
	}
	scope, err := s.scope(ctx, ws, p, in.Lists)
	if err != nil {
		return IntegrationSecret{}, err
	}
	if in.ExpiresAt != nil && !in.ExpiresAt.After(s.now()) {
		return IntegrationSecret{}, api.Invalid("body.expiresAt", "Pick an expiry date in the future.")
	}
	id := uuid.Must(uuid.NewV7())
	setJSON, _ := json.Marshal(settings)
	off := offItems(kind, items)
	var out IntegrationSecret
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if err := q.IntegrationInsert(ctx, store.IntegrationInsertParams{ID: id, WorkspaceID: ws.ID, Kind: kind, Label: label, Items: items, OffItems: off,
			Settings: setJSON, ConnectedBy: &p.UserID, ConnectedAt: s.now()}); err != nil {
			if isUnique(err) {
				return api.Conflict(fmt.Sprintf("%s is already connected. Open it to change what it can do.", label))
			}
			return err
		}
		if kind == KindWordPress {
			if err := q.WordPressSiteInsert(ctx, store.WordPressSiteInsertParams{IntegrationID: id, WorkspaceID: ws.ID, Address: address}); err != nil {
				return err
			}
		}
		var err error
		out, err = s.newKey(ctx, q, ws, p.UserID, id, kind, name, items, scope, in.ExpiresAt, nil)
		if err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "integration.connected", TargetType: "integration", TargetID: id.String(),
			TargetName: fmt.Sprintf("%s · key “%s”", Name(kind), name),
			After:      map[string]any{"kind": kind, "key": name, "masked": out.Key.Masked, "items": items, "permissions": out.Key.Permissions, "scope": scopeWords(out.Key.Scope), "expires_at": in.ExpiresAt},
			Metadata:   map[string]any{"summary": connectSummary(kind, items, out.Key.Scope, in.ExpiresAt), "permission": string(authz.IntegrationsManage)}})
	})
	if err != nil {
		return IntegrationSecret{}, err
	}
	return out, nil
}

// connectSummary is the audit line: "10 triggers and actions · Send a
// transactional email left off · all lists · no expiry".
func connectSummary(kind string, items []string, scope access.ScopeSummary, expires *time.Time) string {
	parts := []string{fmt.Sprintf("%d %s", len(items), map[bool]string{true: "triggers and actions", false: "things it can do"}[IsPlatform(kind)])}
	for _, off := range offItems(kind, items) {
		parts = append(parts, itemLabel(off)+" left off")
	}
	parts = append(parts, scopeWords(scope))
	if expires != nil {
		parts = append(parts, "expires "+expires.Format("Jan 2, 2006"))
	} else {
		parts = append(parts, "no expiry")
	}
	return strings.Join(parts, " · ")
}

func scopeWords(v access.ScopeSummary) string {
	if v.All {
		return "all lists"
	}
	return v.Label
}

// itemLabel names an item in audit summaries (English, like every audit line).
func itemLabel(key string) string {
	switch key {
	case "contact.subscribed":
		return "Contact subscribed"
	case "contact.unsubscribed":
		return "Contact unsubscribed"
	case "contact.updated":
		return "Contact updated"
	case "form.submitted":
		return "Form submitted"
	case "campaign.completed":
		return "Campaign sent"
	case "message.bounced":
		return "Transactional message bounced"
	case "upsert_contact":
		return "Create or update a contact"
	case "add_to_list":
		return "Add to a list"
	case "add_tag":
		return "Add a tag"
	case "record_event":
		return "Record an event"
	case "send_transactional":
		return "Send a transactional email"
	case "list_forms":
		return "List your signup forms"
	case "create_drafts":
		return "Create drafts from posts"
	}
	return key
}

func (s *Service) scope(ctx context.Context, ws *store.Workspace, p *authz.Principal, in []string) (authz.Scope, error) {
	lists, err := access.ListNames(ctx, s.Q, ws.ID)
	if err != nil {
		return authz.Scope{}, err
	}
	scope, err := (&access.ScopeRequest{Lists: in}).Resolve(lists, nil)
	if err != nil {
		return authz.Scope{}, api.Invalid("body.lists", capitalize(err.Error())+".")
	}
	if !scope.Within(p.Scope) {
		return authz.Scope{}, api.Err(http.StatusForbidden, api.CodeEscalation, "A key can't reach lists outside your own scope.")
	}
	return scope, nil
}

// newKey inserts a key for an integration and returns its secret once.
func (s *Service) newKey(ctx context.Context, q *store.Queries, ws *store.Workspace, owner, integrationID uuid.UUID, kind, name string, items []string,
	scope authz.Scope, expires *time.Time, rotatedFrom *uuid.UUID) (IntegrationSecret, error) {
	secret, hash, masked := authn.NewAPIKey(authn.APIKeyPrefix)
	keyID := uuid.Must(uuid.NewV7())
	perms := storedPermissions(kind, items)
	if err := q.IntegrationKeyInsert(ctx, store.IntegrationKeyInsertParams{ID: keyID, WorkspaceID: ws.ID, Name: name, Prefix: masked, TokenHash: hash,
		Permissions: perms, Scope: access.ScopeJSON(scope), CreatedBy: &owner, CreatedAt: s.now(), ExpiresAt: expires, RotatedFrom: rotatedFrom,
		IntegrationID: &integrationID}); err != nil {
		return IntegrationSecret{}, err
	}
	rows, err := q.IntegrationKeys(ctx, &integrationID)
	if err != nil {
		return IntegrationSecret{}, err
	}
	lists, err := access.ListNames(ctx, q, ws.ID)
	if err != nil {
		return IntegrationSecret{}, err
	}
	out := IntegrationSecret{IntegrationID: integrationID.String(), Kind: kind, Secret: secret, InstanceURL: s.base()}
	for _, r := range rows {
		if r.ID == keyID {
			out.Key = keyView(r, lists, s.now())
		}
	}
	return out, nil
}

// connection loads an active connection of the workspace with its keys.
func (s *Service) connection(ctx context.Context, ws *store.Workspace, id uuid.UUID) (store.IntegrationGetRow, []store.IntegrationKeysRow, error) {
	in, err := s.Q.IntegrationGet(ctx, store.IntegrationGetParams{ID: id, WorkspaceID: ws.ID})
	if err != nil {
		return in, nil, api.MapError(err, "Integration")
	}
	if in.DisconnectedAt != nil {
		return in, nil, api.NotFound("Integration")
	}
	keys, err := s.Q.IntegrationKeys(ctx, &in.ID)
	return in, keys, err
}

// current returns the newest key of a connection, and the rotated one
// still in its grace period.
func current(keys []store.IntegrationKeysRow, now time.Time) (*store.IntegrationKeysRow, *store.IntegrationKeysRow) {
	var cur, old *store.IntegrationKeysRow
	for i := range keys {
		k := &keys[i]
		if cur == nil {
			cur = k
			continue
		}
		if old == nil && k.RevokedAt != nil && k.GraceUntil != nil && k.GraceUntil.After(now) {
			old = k
		}
	}
	return cur, old
}

// ConnectionID finds the active connection of a platform, or a WordPress
// site by its id.
func (s *Service) ConnectionID(ctx context.Context, ws *store.Workspace, kind string) (uuid.UUID, error) {
	in, err := s.Q.IntegrationActiveByKind(ctx, store.IntegrationActiveByKindParams{WorkspaceID: ws.ID, Kind: kind})
	if err != nil {
		return uuid.Nil, api.MapError(err, Name(kind))
	}
	return in.ID, nil
}

// IntegrationChangeInput changes what an integration can do.
type IntegrationChangeInput struct {
	Items []string `json:"items" maxItems:"20"`
	Lists []string `json:"lists,omitempty" maxItems:"100"`
}

// IntegrationChanged reports a change of what an integration can do.
type IntegrationChanged struct {
	Items       []string `json:"items"`
	Permissions []string `json:"permissions"`
	// WebhooksRemoved counts managed webhooks of triggers turned off.
	WebhooksRemoved int `json:"webhooksRemoved"`
}

// Change updates the key's permissions in place: nothing to paste again.
// Triggers turned off lose their managed webhooks.
func (s *Service) Change(ctx context.Context, ws *store.Workspace, p *authz.Principal, id uuid.UUID, in IntegrationChangeInput) (IntegrationChanged, error) {
	conn, keys, err := s.connection(ctx, ws, id)
	if err != nil {
		return IntegrationChanged{}, err
	}
	items, err := s.checkItems(ctx, ws, p, conn.Kind, in.Items, conn.Items)
	if err != nil {
		return IntegrationChanged{}, err
	}
	var scope *authz.Scope
	if in.Lists != nil {
		sc, err := s.scope(ctx, ws, p, in.Lists)
		if err != nil {
			return IntegrationChanged{}, err
		}
		scope = &sc
	}
	perms := storedPermissions(conn.Kind, items)
	out := IntegrationChanged{Items: items, Permissions: shortPermissions(perms)}
	var beforePerms []string
	if cur, _ := current(keys, s.now()); cur != nil {
		beforePerms = shortPermissions(cur.Permissions)
	}
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if err := q.IntegrationSetItems(ctx, store.IntegrationSetItemsParams{ID: conn.ID, Items: items, OffItems: offItems(conn.Kind, items)}); err != nil {
			return err
		}
		if err := q.IntegrationKeySetPermissions(ctx, store.IntegrationKeySetPermissionsParams{IntegrationID: &conn.ID, Permissions: perms}); err != nil {
			return err
		}
		if scope != nil {
			if err := q.IntegrationKeySetScope(ctx, store.IntegrationKeySetScopeParams{IntegrationID: &conn.ID, Scope: access.ScopeJSON(*scope)}); err != nil {
				return err
			}
		}
		hooks, err := q.IntegrationWebhooks(ctx, conn.ID)
		if err != nil {
			return err
		}
		for _, h := range hooks {
			if slices.Contains(items, h.Event) {
				continue
			}
			if _, err := q.IntegrationWebhookDelete(ctx, h.ID); err != nil {
				return err
			}
			out.WebhooksRemoved++
			if err := s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "webhook.deleted", TargetType: "webhook", TargetID: h.ID.String(), TargetName: h.Url,
				Before:   map[string]any{"url": h.Url, "events": h.Events},
				Metadata: map[string]any{"summary": fmt.Sprintf("managed by %s · %s turned off", Name(conn.Kind), itemLabel(h.Event)), "permission": string(authz.IntegrationsManage)}}); err != nil {
				return err
			}
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "integration.permissions_changed", TargetType: "integration", TargetID: conn.ID.String(),
			TargetName: Name(conn.Kind) + " · " + conn.Label,
			Before:     map[string]any{"items": conn.Items, "permissions": beforePerms},
			After:      map[string]any{"items": items, "permissions": out.Permissions},
			Metadata:   map[string]any{"summary": changeSummary(conn.Items, items), "permission": string(authz.IntegrationsManage)}})
	})
	return out, err
}

func changeSummary(before, after []string) string {
	var on, off []string
	for _, k := range after {
		if !slices.Contains(before, k) {
			on = append(on, itemLabel(k))
		}
	}
	for _, k := range before {
		if !slices.Contains(after, k) {
			off = append(off, itemLabel(k))
		}
	}
	var parts []string
	if len(on) > 0 {
		parts = append(parts, "turned on: "+strings.Join(on, ", "))
	}
	if len(off) > 0 {
		parts = append(parts, "turned off: "+strings.Join(off, ", "))
	}
	if len(parts) == 0 {
		return "no change to what it can do"
	}
	return strings.Join(parts, " · ")
}

// Rotate gives the integration a new key, shown once. The old one keeps
// working for 24 hours, so runs don't fail while the new one is pasted.
func (s *Service) Rotate(ctx context.Context, ws *store.Workspace, p *authz.Principal, id uuid.UUID) (IntegrationSecret, error) {
	conn, keys, err := s.connection(ctx, ws, id)
	if err != nil {
		return IntegrationSecret{}, err
	}
	cur, _ := current(keys, s.now())
	if cur == nil || cur.RevokedAt != nil {
		return IntegrationSecret{}, api.Conflict(fmt.Sprintf("%s's key was revoked. Reconnect to get a new one.", Name(conn.Kind)))
	}
	if err := s.canHandOut(ctx, ws, p, cur.Permissions, "rotate"); err != nil {
		return IntegrationSecret{}, err
	}
	scope := access.ParseScope(cur.Scope)
	var expires *time.Time
	if cur.ExpiresAt != nil {
		life := cur.ExpiresAt.Sub(cur.CreatedAt)
		if life < 24*time.Hour {
			life = 24 * time.Hour
		}
		e := s.now().Add(life)
		expires = &e
	}
	owner := p.UserID
	if cur.CreatedBy != nil && cur.OwnerStatus == "active" {
		owner = *cur.CreatedBy
	}
	grace := s.now().Add(RotationGrace)
	var out IntegrationSecret
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		// One overlap at a time: a key already in its grace period stops now.
		if _, err := q.IntegrationKeyStopGrace(ctx, &conn.ID); err != nil {
			return err
		}
		if _, err := q.IntegrationKeyRevoke(ctx, store.IntegrationKeyRevokeParams{ID: cur.ID, IntegrationID: &conn.ID, At: s.now(), GraceUntil: &grace}); err != nil {
			return err
		}
		var err error
		out, err = s.newKey(ctx, q, ws, owner, conn.ID, conn.Kind, cur.Name, conn.Items, scope, expires, &cur.ID)
		if err != nil {
			return err
		}
		out.OldKeyUntil = &grace
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "integration.key_rotated", TargetType: "integration", TargetID: conn.ID.String(),
			TargetName: Name(conn.Kind) + " · " + conn.Label,
			Before:     map[string]any{"masked": cur.Prefix}, After: map[string]any{"masked": out.Key.Masked, "old_key_until": grace},
			Metadata: map[string]any{"summary": fmt.Sprintf("new key %s · old %s works until %s", out.Key.Masked, cur.Prefix, grace.Format("Jan 2 15:04")),
				"permission": string(authz.IntegrationsManage)}})
	})
	return out, err
}

// canHandOut refuses to show a key's secret to someone who doesn't hold
// every permission on it (rotating or reconnecting hands out a new secret).
func (s *Service) canHandOut(ctx context.Context, ws *store.Workspace, p *authz.Principal, perms []string, verb string) error {
	for _, k := range shortPermissions(perms) {
		perm := authz.Permission(k)
		if !p.Can(perm) {
			pr := api.Err(http.StatusForbidden, api.CodeEscalation, fmt.Sprintf("You can't %s this key: it holds %s, which you don't hold. Ask %s.",
				verb, perm, access.OrList(access.Helpers(ctx, s.Q, ws.ID, perm))))
			pr.Permission = k
			return pr
		}
	}
	return nil
}

// StopOldKey ends a rotated key's grace period at once.
func (s *Service) StopOldKey(ctx context.Context, ws *store.Workspace, id uuid.UUID) error {
	conn, keys, err := s.connection(ctx, ws, id)
	if err != nil {
		return err
	}
	_, old := current(keys, s.now())
	if old == nil {
		return api.Conflict("No old key is still working.")
	}
	return db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if _, err := q.IntegrationKeyStopGrace(ctx, &conn.ID); err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "integration.key_revoked", TargetType: "integration", TargetID: conn.ID.String(),
			TargetName: Name(conn.Kind) + " · " + conn.Label, Before: map[string]any{"masked": old.Prefix, "grace_until": old.GraceUntil},
			Metadata: map[string]any{"summary": "old key " + old.Prefix + " stopped before its 24 hours ran out", "permission": string(authz.IntegrationsManage)}})
	})
}

// Reconnect gives a connection whose key was revoked or expired a new key,
// with what it could do before. It waits for its first call again.
func (s *Service) Reconnect(ctx context.Context, ws *store.Workspace, p *authz.Principal, id uuid.UUID, in IntegrationConnectInput) (IntegrationSecret, error) {
	conn, keys, err := s.connection(ctx, ws, id)
	if err != nil {
		return IntegrationSecret{}, err
	}
	cur, _ := current(keys, s.now())
	if cur != nil && cur.RevokedAt == nil && (cur.ExpiresAt == nil || cur.ExpiresAt.After(s.now())) {
		return IntegrationSecret{}, api.Conflict(fmt.Sprintf("%s's key still works. Rotate it to get a new one.", Name(conn.Kind)))
	}
	items := conn.Items
	if len(in.Items) > 0 {
		if items, err = s.checkItems(ctx, ws, p, conn.Kind, in.Items, conn.Items); err != nil {
			return IntegrationSecret{}, err
		}
	}
	if err := s.canHandOut(ctx, ws, p, storedPermissions(conn.Kind, items), "reconnect"); err != nil {
		return IntegrationSecret{}, err
	}
	name := Name(conn.Kind) + " · " + conn.Label
	scope := authz.Scope{}
	if cur != nil {
		name = cur.Name
		scope = access.ParseScope(cur.Scope)
	}
	if in.Lists != nil {
		if scope, err = s.scope(ctx, ws, p, in.Lists); err != nil {
			return IntegrationSecret{}, err
		}
	}
	if in.ExpiresAt != nil && !in.ExpiresAt.After(s.now()) {
		return IntegrationSecret{}, api.Invalid("body.expiresAt", "Pick an expiry date in the future.")
	}
	var out IntegrationSecret
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if err := q.IntegrationSetItems(ctx, store.IntegrationSetItemsParams{ID: conn.ID, Items: items, OffItems: offItems(conn.Kind, items)}); err != nil {
			return err
		}
		if err := q.IntegrationReconnected(ctx, conn.ID); err != nil {
			return err
		}
		var from *uuid.UUID
		if cur != nil {
			from = &cur.ID
		}
		var err error
		out, err = s.newKey(ctx, q, ws, p.UserID, conn.ID, conn.Kind, name, items, scope, in.ExpiresAt, from)
		if err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "integration.reconnected", TargetType: "integration", TargetID: conn.ID.String(),
			TargetName: Name(conn.Kind) + " · " + conn.Label, After: map[string]any{"masked": out.Key.Masked, "items": items},
			Metadata: map[string]any{"summary": "new key " + out.Key.Masked + " · " + connectSummary(conn.Kind, items, out.Key.Scope, in.ExpiresAt), "permission": string(authz.IntegrationsManage)}})
	})
	return out, err
}

// IntegrationTakeOverPreview is what taking over a key would change.
type IntegrationTakeOverPreview struct {
	Key      IntegrationKeyView `json:"key"`
	NewOwner IntegrationPerson  `json:"newOwner"`
	RoleName string             `json:"roleName"`
	// Held and Missing split the key's permissions by whether the new owner
	// holds them; Stops lists the triggers and actions that stop.
	Held    []string `json:"held"`
	Missing []string `json:"missing"`
	Stops   []string `json:"stops"`
}

// TakeOverPreview lists what the caller holds of the key's permissions.
func (s *Service) TakeOverPreview(ctx context.Context, ws *store.Workspace, p *authz.Principal, id uuid.UUID) (IntegrationTakeOverPreview, error) {
	conn, keys, err := s.connection(ctx, ws, id)
	if err != nil {
		return IntegrationTakeOverPreview{}, err
	}
	cur, _ := current(keys, s.now())
	if cur == nil || cur.RevokedAt != nil {
		return IntegrationTakeOverPreview{}, api.Conflict(fmt.Sprintf("%s has no key to take over. Reconnect it instead.", Name(conn.Kind)))
	}
	lists, err := access.ListNames(ctx, s.Q, ws.ID)
	if err != nil {
		return IntegrationTakeOverPreview{}, err
	}
	out := IntegrationTakeOverPreview{Key: keyView(*cur, lists, s.now()), NewOwner: IntegrationPerson{ID: p.UserID.String(), Name: p.Name}, RoleName: p.RoleName,
		Held: []string{}, Missing: []string{}, Stops: []string{}}
	if u, err := s.Q.GetUser(ctx, p.UserID); err == nil {
		out.NewOwner.AvatarColor = u.AvatarColor
	}
	for _, k := range shortPermissions(cur.Permissions) {
		if p.Can(authz.Permission(k)) {
			out.Held = append(out.Held, k)
		} else {
			out.Missing = append(out.Missing, k)
		}
	}
	for _, key := range conn.Items {
		it, ok := ItemByKey(conn.Kind, key)
		if !ok {
			continue
		}
		for _, need := range it.Permissions {
			if !p.Can(need) {
				out.Stops = append(out.Stops, key)
				break
			}
		}
	}
	return out, nil
}

// TakeOver makes the caller the owner of the integration's key. The key
// keeps its value; from now on it's limited by the new owner's permissions.
func (s *Service) TakeOver(ctx context.Context, ws *store.Workspace, p *authz.Principal, id uuid.UUID) (IntegrationTakeOverPreview, error) {
	prev, err := s.TakeOverPreview(ctx, ws, p, id)
	if err != nil {
		return prev, err
	}
	conn, _, err := s.connection(ctx, ws, id)
	if err != nil {
		return prev, err
	}
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if err := q.IntegrationKeyTakeOver(ctx, store.IntegrationKeyTakeOverParams{IntegrationID: &conn.ID, Owner: &p.UserID}); err != nil {
			return err
		}
		if err := q.IntegrationSetAttention(ctx, store.IntegrationSetAttentionParams{ID: conn.ID, Attention: []byte(`[]`), Notified: []string{}}); err != nil {
			return err
		}
		summary := fmt.Sprintf("%s now owns %s; the key keeps its value", p.Name, prev.Key.Masked)
		if len(prev.Missing) > 0 {
			summary += " · lost " + strings.Join(prev.Missing, ", ")
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "integration.taken_over", TargetType: "integration", TargetID: conn.ID.String(),
			TargetName: Name(conn.Kind) + " · " + conn.Label,
			Before:     map[string]any{"owner": prev.Key.Owner.Name}, After: map[string]any{"owner": p.Name, "missing": prev.Missing},
			Metadata: map[string]any{"summary": summary, "permission": string(authz.IntegrationsManage)}})
	})
	return prev, err
}

// IntegrationDisconnectInput says why, for the audit log.
type IntegrationDisconnectInput struct {
	Reason string `json:"reason,omitempty" maxLength:"300"`
}

// IntegrationDisconnectPreview is what disconnecting stops, removes and keeps.
type IntegrationDisconnectPreview struct {
	Workflows []IntegrationWorkflowRef `json:"workflows" doc:"Workflows seen in the last 30 days: they stop"`
	Key       *IntegrationKeyView      `json:"key,omitempty"`
	Webhooks  int64                    `json:"webhooks"`
	Queued    int64                    `json:"queued"`
	Dead      int64                    `json:"dead"`
}

// IntegrationWorkflowRef is a workflow named in a dialog.
type IntegrationWorkflowRef struct {
	Name      string    `json:"name"`
	Unnamed   bool      `json:"unnamed"`
	LastRunAt time.Time `json:"lastRunAt"`
}

// DisconnectPreview lists what stops, what's removed and what stays.
func (s *Service) DisconnectPreview(ctx context.Context, ws *store.Workspace, id uuid.UUID) (IntegrationDisconnectPreview, error) {
	conn, keys, err := s.connection(ctx, ws, id)
	if err != nil {
		return IntegrationDisconnectPreview{}, err
	}
	out := IntegrationDisconnectPreview{Workflows: []IntegrationWorkflowRef{}}
	wfs, err := s.Q.IntegrationWorkflows(ctx, store.IntegrationWorkflowsParams{IntegrationID: conn.ID, Since: s.now().Add(-WorkflowRetention)})
	if err != nil {
		return out, err
	}
	for _, w := range wfs {
		out.Workflows = append(out.Workflows, IntegrationWorkflowRef{Name: w.Name, Unnamed: w.Name == "", LastRunAt: w.LastSeenAt})
	}
	if cur, _ := current(keys, s.now()); cur != nil {
		lists, _ := access.ListNames(ctx, s.Q, ws.ID)
		v := keyView(*cur, lists, s.now())
		out.Key = &v
	}
	hooks, err := s.Q.IntegrationWebhooks(ctx, conn.ID)
	if err != nil {
		return out, err
	}
	out.Webhooks = int64(len(hooks))
	if qd, err := s.Q.IntegrationWebhookQueued(ctx, conn.ID); err == nil {
		out.Queued, out.Dead = qd.Queued, qd.Dead
	}
	return out, nil
}

// Disconnect revokes the key at once and deletes the managed webhooks with
// their queued and dead-letter deliveries. Contacts, events and messages
// created through the integration stay.
func (s *Service) Disconnect(ctx context.Context, ws *store.Workspace, p *authz.Principal, id uuid.UUID, in IntegrationDisconnectInput) error {
	prev, err := s.DisconnectPreview(ctx, ws, id)
	if err != nil {
		return err
	}
	conn, _, err := s.connection(ctx, ws, id)
	if err != nil {
		return err
	}
	return db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		revoked, err := q.IntegrationKeysRevokeAll(ctx, store.IntegrationKeysRevokeAllParams{IntegrationID: &conn.ID, At: s.now()})
		if err != nil {
			return err
		}
		hooks, err := q.IntegrationWebhooksDeleteAll(ctx, conn.ID)
		if err != nil {
			return err
		}
		if _, err := q.IntegrationDisconnect(ctx, store.IntegrationDisconnectParams{ID: conn.ID, WorkspaceID: ws.ID, At: s.now(), By: &p.UserID}); err != nil {
			return err
		}
		names := []string{}
		for _, w := range prev.Workflows {
			if !w.Unnamed {
				names = append(names, w.Name)
			}
		}
		masked := []string{}
		for _, k := range revoked {
			masked = append(masked, k.Prefix)
		}
		urls := []string{}
		for _, h := range hooks {
			urls = append(urls, h.Url)
		}
		summary := fmt.Sprintf("key %s revoked · %d %s removed with %d queued and %d dead-letter deliveries", strings.Join(masked, ", "),
			len(hooks), access.Plural(len(hooks), "webhook", "webhooks"), prev.Queued, prev.Dead)
		if len(names) > 0 {
			summary += " · stops " + strings.Join(names, ", ")
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "integration.disconnected", TargetType: "integration", TargetID: conn.ID.String(),
			TargetName: Name(conn.Kind) + " · " + conn.Label, Reason: strings.TrimSpace(in.Reason),
			Before:   map[string]any{"kind": conn.Kind, "keys": masked, "webhooks": urls, "items": conn.Items},
			Metadata: map[string]any{"summary": summary, "workflows": names, "permission": string(authz.IntegrationsManage)}})
	})
}

// IntegrationReachability is the public URL check as Zapier and Make see it.
type IntegrationReachability struct {
	URL string `json:"url"`
	// Private: the public URL is a private, loopback or local address.
	Private   bool               `json:"private"`
	Reachable bool               `json:"reachable"`
	CheckedAt *time.Time         `json:"checkedAt,omitempty"`
	DNS       accounts.CheckLine `json:"dns"`
	TLS       accounts.CheckLine `json:"tls"`
	Reach     accounts.CheckLine `json:"reach"`
	// Admins are the instance admins, who can give the instance a public
	// address or allow webhooks to private addresses.
	Admins []string `json:"admins"`
	// PrivateWebhooks: webhooks may go to private addresses on this server.
	PrivateWebhooks bool `json:"privateWebhooks"`
}

// Reachability says whether apps on the internet can reach the instance,
// from the public URL and the last check of it.
func (s *Service) Reachability(ctx context.Context) IntegrationReachability {
	out := IntegrationReachability{URL: s.base(), Admins: []string{}, PrivateWebhooks: s.allowPrivate()}
	if admins, err := s.Q.IntegrationInstanceAdmins(ctx); err == nil {
		for _, a := range admins {
			out.Admins = append(out.Admins, firstNonEmpty(a.Name, a.Email))
		}
	}
	u, err := url.Parse(out.URL)
	if err != nil || u.Hostname() == "" {
		out.DNS = accounts.CheckLine{Status: "bad", Detail: "No public URL is set"}
		return out
	}
	if st, err := s.Q.GetInstanceSettings(ctx); err == nil && len(st.PublicUrlCheck) > 0 && string(st.PublicUrlCheck) != "null" &&
		strings.TrimRight(st.PublicUrl, "/") == strings.TrimRight(out.URL, "/") {
		var c accounts.URLCheck
		if json.Unmarshal(st.PublicUrlCheck, &c) == nil {
			out.DNS, out.TLS, out.Reach = c.DNS, c.TLS, c.Reachable
			out.CheckedAt = st.PublicUrlCheckedAt
		}
	}
	host := u.Hostname()
	if addr, private := privateHost(ctx, host); private {
		out.Private = true
		out.DNS = accounts.CheckLine{Status: "bad", Detail: fmt.Sprintf("%s is a private address", addr)}
		if u.Scheme == "http" {
			out.TLS = accounts.CheckLine{Status: "bad", Detail: "no certificate on http"}
		}
		out.Reach = accounts.CheckLine{Status: "bad", Detail: "Not reachable from the internet"}
		return out
	}
	out.Reachable = out.Reach.Status != "bad" && out.DNS.Status != "bad"
	return out
}

// CheckReachability runs the public URL check again and keeps the result,
// as Instance admin's "Check" does.
func (s *Service) CheckReachability(ctx context.Context) (IntegrationReachability, error) {
	if s.URLCheck == nil || s.base() == "" {
		return s.Reachability(ctx), nil
	}
	res, err := s.URLCheck.CheckPublicURL(ctx, s.base())
	if err != nil {
		return IntegrationReachability{}, err
	}
	if st, err := s.Q.GetInstanceSettings(ctx); err == nil && (strings.TrimRight(st.PublicUrl, "/") == s.base() || st.PublicUrl == "") {
		b, _ := json.Marshal(res)
		_ = s.Q.InstanceAdminSetURLCheck(ctx, b)
	}
	return s.Reachability(ctx), nil
}

// privateHost reports whether a host is (or resolves only to) a private,
// loopback or local address, and which.
func privateHost(ctx context.Context, host string) (string, bool) {
	lower := strings.ToLower(host)
	if lower == "localhost" || strings.HasSuffix(lower, ".localhost") || strings.HasSuffix(lower, ".internal") || strings.HasSuffix(lower, ".local") {
		return host, true
	}
	if ip, err := netip.ParseAddr(host); err == nil {
		return ip.String(), isPrivate(ip)
	}
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	addrs, err := net.DefaultResolver.LookupHost(ctx, host)
	if err != nil || len(addrs) == 0 {
		return "", false
	}
	for _, a := range addrs {
		ip, err := netip.ParseAddr(a)
		if err != nil || !isPrivate(ip) {
			return "", false
		}
	}
	return addrs[0], true
}

func isPrivate(ip netip.Addr) bool {
	ip = ip.Unmap()
	return ip.IsPrivate() || ip.IsLoopback() || ip.IsLinkLocalUnicast() || ip.IsUnspecified() ||
		netip.MustParsePrefix("100.64.0.0/10").Contains(ip)
}

// IntegrationAddressCheck is what ReputeCast sees of a workflow tool's address.
type IntegrationAddressCheck struct {
	Address string `json:"address"`
	Host    string `json:"host"`
	// Resolves is the first address the host resolves to.
	Resolves string `json:"resolves,omitempty"`
	Private  bool   `json:"private"`
	// Allowed: webhooks may reach it (public, or private ones are allowed).
	Allowed bool     `json:"allowed"`
	Admins  []string `json:"admins"`
}

// CheckAddress tells whether triggers can reach a workflow tool's address
// (n8n on a private network needs an instance admin to allow it).
func (s *Service) CheckAddress(ctx context.Context, raw string) (IntegrationAddressCheck, error) {
	a, err := normalizeAddress(raw)
	if err != nil {
		return IntegrationAddressCheck{}, api.Invalid("body.address", "Enter the address, like https://n8n.acme.io.")
	}
	u, _ := url.Parse(a)
	out := IntegrationAddressCheck{Address: a, Host: u.Hostname(), Admins: s.Reachability(ctx).Admins}
	addr, private := privateHost(ctx, u.Hostname())
	out.Resolves, out.Private = addr, private
	if !private {
		ctx2, cancel := context.WithTimeout(ctx, 2*time.Second)
		defer cancel()
		if addrs, err := net.DefaultResolver.LookupHost(ctx2, u.Hostname()); err == nil && len(addrs) > 0 {
			out.Resolves = addrs[0]
		}
	}
	out.Allowed = !private || s.allowPrivate()
	return out, nil
}

// IntegrationAddressInput is n8n's address, set while connecting.
type IntegrationAddressInput struct {
	Address string `json:"address" maxLength:"300" example:"https://n8n.acme.io" doc:"Empty to forget it"`
}

// SetAddress keeps n8n's address, to open it from its page and to check
// that triggers can reach it. It answers what ReputeCast sees of it.
func (s *Service) SetAddress(ctx context.Context, ws *store.Workspace, id uuid.UUID, raw string) (IntegrationAddressCheck, error) {
	conn, _, err := s.connection(ctx, ws, id)
	if err != nil {
		return IntegrationAddressCheck{}, err
	}
	if conn.Kind != KindN8n {
		return IntegrationAddressCheck{}, api.Invalid("body.address", Name(conn.Kind)+" has no address to set.")
	}
	var settings map[string]any
	_ = json.Unmarshal(conn.Settings, &settings)
	if settings == nil {
		settings = map[string]any{}
	}
	before, _ := settings["address"].(string)
	var out IntegrationAddressCheck
	if strings.TrimSpace(raw) == "" {
		delete(settings, "address")
	} else {
		if out, err = s.CheckAddress(ctx, raw); err != nil {
			return out, err
		}
		settings["address"] = out.Address
	}
	if before == out.Address {
		return out, nil
	}
	b, _ := json.Marshal(settings)
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if err := q.IntegrationSetSettings(ctx, store.IntegrationSetSettingsParams{ID: conn.ID, Settings: b}); err != nil {
			return err
		}
		summary := "address " + firstNonEmpty(out.Address, "removed")
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "integration.address_changed", TargetType: "integration", TargetID: conn.ID.String(),
			TargetName: Name(conn.Kind) + " · " + conn.Label,
			Before:     map[string]any{"address": before}, After: map[string]any{"address": out.Address},
			Metadata: map[string]any{"summary": summary, "permission": string(authz.IntegrationsManage)}})
	})
	return out, err
}

func normalizeAddress(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if !strings.Contains(raw, "://") {
		raw = "https://" + raw
	}
	u, err := url.Parse(raw)
	if err != nil || u.Hostname() == "" || (u.Scheme != "http" && u.Scheme != "https") {
		return "", errors.New("invalid address")
	}
	return strings.TrimRight(u.Scheme+"://"+u.Host+u.EscapedPath(), "/"), nil
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

func capitalize(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

func isUnique(err error) bool {
	return err != nil && strings.Contains(err.Error(), "SQLSTATE 23505")
}
