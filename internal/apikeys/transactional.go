package apikeys

import (
	"context"
	"fmt"
	"path"
	"slices"
	"strings"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/authn"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/store"
)

// transactionalKey is a send-only key's settings, as stored.
type transactionalKey struct {
	Preset        string
	Mode          string
	Templates     []string
	Categories    []uuid.UUID
	TestAddresses []string
	Rate          int
}

// prefix makes the key say what it is: rc_smtp_ for SMTP credentials,
// rc_test_ for test mode, rc_live_ otherwise.
func (t transactionalKey) prefix() string {
	switch {
	case t.Preset == PresetSMTP:
		return authn.SMTPPrefix
	case t.Mode == "test":
		return authn.TestPrefix
	}
	return authn.APIKeyPrefix
}

// transactionalInput checks a send-only key: transactional:send is its only
// permission, its templates and categories exist, its test addresses are
// addresses or patterns.
func (s *Service) transactionalInput(ctx context.Context, ws *store.Workspace, in APIKeyCreateInput) (transactionalKey, error) {
	t := in.Transactional
	if len(in.Permissions) != 1 || in.Permissions[0] != string(authz.TransactionalSend) {
		return transactionalKey{}, api.Invalid("body.permissions", "A send-only key holds transactional:send and nothing else.")
	}
	if len(in.Lists) > 0 {
		return transactionalKey{}, api.Invalid("body.lists", "Send-only keys aren't limited by list; limit them to templates instead.")
	}
	out := transactionalKey{Preset: t.Preset, Mode: t.Mode, Templates: []string{}, Categories: []uuid.UUID{}, TestAddresses: []string{}, Rate: t.RatePerSecond}
	if out.Preset != PresetSendOnly && out.Preset != PresetSMTP {
		return transactionalKey{}, api.Invalid("body.transactional.preset", "Pick a send-only key or an SMTP credential.")
	}
	if out.Mode == "" {
		out.Mode = "live"
	}
	if out.Rate <= 0 {
		out.Rate = 50
	}
	if len(t.Templates) > 0 {
		known := map[string]bool{}
		rows, err := s.Q.TxnRouteTemplates(ctx, ws.ID)
		if err != nil {
			return transactionalKey{}, err
		}
		for _, r := range rows {
			known[r.Key] = true
		}
		for _, k := range t.Templates {
			k = strings.ToLower(strings.TrimSpace(k))
			if !known[k] {
				return transactionalKey{}, api.Invalid("body.transactional.templates", fmt.Sprintf("There is no template with the ID %s.", k))
			}
			if !slices.Contains(out.Templates, k) {
				out.Templates = append(out.Templates, k)
			}
		}
	}
	if len(t.Categories) > 0 {
		cats := s.categoryNames(ctx, ws.ID)
		for _, c := range t.Categories {
			id, err := uuid.Parse(c)
			if err != nil || cats[id] == "" {
				return transactionalKey{}, api.Invalid("body.transactional.categories", "Pick categories from this workspace.")
			}
			if !slices.Contains(out.Categories, id) {
				out.Categories = append(out.Categories, id)
			}
		}
	}
	for _, a := range t.TestAddresses {
		a = strings.ToLower(strings.TrimSpace(a))
		if a == "" {
			continue
		}
		at := strings.LastIndexByte(a, '@')
		if at < 1 || at == len(a)-1 || strings.ContainsAny(a, " <>") {
			return transactionalKey{}, api.Invalid("body.transactional.testAddresses", fmt.Sprintf("“%s” isn't an address or a pattern such as qa+*@acme.io.", a))
		}
		if _, err := path.Match(a, "x@example.com"); err != nil {
			return transactionalKey{}, api.Invalid("body.transactional.testAddresses", fmt.Sprintf("“%s” isn't a valid pattern.", a))
		}
		if !slices.Contains(out.TestAddresses, a) {
			out.TestAddresses = append(out.TestAddresses, a)
		}
	}
	if out.Mode == "live" && len(out.TestAddresses) > 0 {
		return transactionalKey{}, api.Invalid("body.transactional.testAddresses", "Allowed test addresses belong to test-mode keys; live keys deliver to everyone.")
	}
	return out, nil
}

// transactionalScope says what a send-only key can send: "all templates",
// "password-reset, magic-link" or "category Billing".
func transactionalScope(t *APIKeyTransactional) string {
	switch {
	case len(t.Categories) > 0:
		names := make([]string, 0, len(t.Categories))
		for _, c := range t.Categories {
			names = append(names, c.Name)
		}
		if len(names) == 1 {
			return "category " + names[0]
		}
		return "categories " + strings.Join(names, ", ")
	case len(t.Templates) > 0:
		return strings.Join(t.Templates, ", ")
	}
	return "all templates"
}
