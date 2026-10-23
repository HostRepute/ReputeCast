package apikeys

import (
	"context"
	"path"
	"slices"
	"strings"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/authn"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/i18n"
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
		return transactionalKey{}, api.Invalid("body.permissions", say(ctx, "problem.sendOnlyPermission", nil))
	}
	if len(in.Lists) > 0 {
		return transactionalKey{}, api.Invalid("body.lists", say(ctx, "problem.sendOnlyLists", nil))
	}
	out := transactionalKey{Preset: t.Preset, Mode: t.Mode, Templates: []string{}, Categories: []uuid.UUID{}, TestAddresses: []string{}, Rate: t.RatePerSecond}
	if out.Preset != PresetSendOnly && out.Preset != PresetSMTP {
		return transactionalKey{}, api.Invalid("body.transactional.preset", say(ctx, "problem.preset", nil))
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
				return transactionalKey{}, api.Invalid("body.transactional.templates", say(ctx, "problem.template", i18n.Args{"id": k}))
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
				return transactionalKey{}, api.Invalid("body.transactional.categories", say(ctx, "problem.categories", nil))
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
			return transactionalKey{}, api.Invalid("body.transactional.testAddresses", say(ctx, "problem.testAddress", i18n.Args{"address": a}))
		}
		if _, err := path.Match(a, "x@example.com"); err != nil {
			return transactionalKey{}, api.Invalid("body.transactional.testAddresses", say(ctx, "problem.testPattern", i18n.Args{"address": a}))
		}
		if !slices.Contains(out.TestAddresses, a) {
			out.TestAddresses = append(out.TestAddresses, a)
		}
	}
	if out.Mode == "live" && len(out.TestAddresses) > 0 {
		return transactionalKey{}, api.Invalid("body.transactional.testAddresses", say(ctx, "problem.testLive", nil))
	}
	return out, nil
}

// transactionalScope says what a send-only key can send, kept for the
// audit log: "all templates", "password-reset, magic-link" (template IDs)
// or "category Billing".
func transactionalScope(t *APIKeyTransactional) any {
	switch {
	case len(t.Categories) > 0:
		names := make([]string, 0, len(t.Categories))
		for _, c := range t.Categories {
			names = append(names, c.Name)
		}
		if len(names) == 1 {
			return msgs.M("audit.scopeCategory").Keep(i18n.Args{"names": names[0]})
		}
		return msgs.M("audit.scopeCategories").Keep(i18n.Args{"names": strings.Join(names, ", ")})
	case len(t.Templates) > 0:
		return strings.Join(t.Templates, ", ")
	}
	return msgs.M("audit.scopeAllTemplates").Keep(nil)
}
