package manage

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/riverqueue/river"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/jobs"
	"github.com/hostrepute/reputecast/internal/providers"
	"github.com/hostrepute/reputecast/internal/providers/state"
	"github.com/hostrepute/reputecast/internal/store"
)

// A provider's sending IPs (providers.ip_mode and ips, review GS-10) are
// what the canary's new-IP trigger, Send Readiness's IP blocklist rows,
// Deliverability and warm-up read. A provider whose API lists them
// (providers.IPReporter) is asked when it's connected, when its connection
// step is repeated and every day; for the others, people set them by hand.
// Where they come from is kept in providers.ip_source (state.IPSource).

// What asked for a provider's sending IPs, for the audit entry.
const (
	IPsOnConnect = "connect"
	IPsDaily     = "daily"
	IPsOnRequest = "request"
)

// ipCheckTimeout bounds one provider's answer.
const ipCheckTimeout = 30 * time.Second

// RefreshIPs asks a provider whose API reports its sending IPs for them and
// keeps the answer (RefreshIPsWith). The adapter is opened from the stored
// credentials; one that can't be opened is a check that failed.
func (s *Service) RefreshIPs(ctx context.Context, p store.Provider, trigger string) (store.Provider, error) {
	adapter, err := s.Factory.Open(ctx, p)
	if err != nil {
		if !state.Caps(p).ReportsIPs {
			return p, nil
		}
		return s.keepIPs(ctx, p, providers.SendingIPs{}, err, trigger)
	}
	return s.RefreshIPsWith(ctx, p, adapter, trigger)
}

// RefreshIPsWith asks adapter, the provider's, for the account's sending IPs
// and keeps providers.ip_mode and ips from the answer, each IP once and in
// address order: dedicated with the IPs it lists, or shared. A change is
// audited (provider.ips_update, by the system). A check that fails keeps
// the IPs as they were and records why; an adapter whose API doesn't list
// them changes nothing. The error is a storage failure only: the provider's
// own is in the returned row's ip_source.
func (s *Service) RefreshIPsWith(ctx context.Context, p store.Provider, adapter providers.Provider, trigger string) (store.Provider, error) {
	r, ok := adapter.(providers.IPReporter)
	if !ok {
		return p, nil
	}
	cctx, cancel := context.WithTimeout(ctx, ipCheckTimeout)
	got, err := r.SendingIPs(cctx)
	cancel()
	return s.keepIPs(ctx, p, got, err, trigger)
}

// keepIPs stores a provider's answer about its sending IPs, or why there is
// none (checkErr).
func (s *Service) keepIPs(ctx context.Context, p store.Provider, got providers.SendingIPs, checkErr error, trigger string) (store.Provider, error) {
	now := s.now()
	ips, bad := providers.CleanIPs(got.IPs)
	if len(bad) > 0 {
		s.log().WarnContext(ctx, "provider listed sending IPs that aren't addresses", "provider", p.ID, "entries", bad)
	}
	mode := "shared"
	if got.Dedicated && len(ips) > 0 {
		mode = "dedicated"
	} else {
		ips = []string{}
	}
	var out store.Provider
	err := db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		cur, err := q.ProvLockForIPs(ctx, p.ID)
		if err != nil {
			return err
		}
		out = cur
		src := state.IPSourceOf(cur)
		src.From, src.SetBy, src.SetByName, src.SetAt = state.IPsFromProvider, nil, "", nil
		if checkErr != nil {
			src.Error, src.ErrorText, src.ErrorAt = callMessage(checkErr), callText(checkErr), &now
			out.IpSource, _ = json.Marshal(src)
			return q.ProvSetIPSource(ctx, store.ProvSetIPSourceParams{ID: cur.ID, IpSource: out.IpSource})
		}
		src.CheckedAt, src.Error, src.ErrorText, src.ErrorAt = &now, "", i18n.Text{}, nil
		out.IpSource, _ = json.Marshal(src)
		if err := q.ProvSetIPs(ctx, store.ProvSetIPsParams{ID: cur.ID, IpMode: mode, Ips: ips, IpSource: out.IpSource}); err != nil {
			return err
		}
		out.IpMode, out.Ips = mode, ips
		if mode == cur.IpMode && slices.Equal(ips, cur.Ips) {
			return nil
		}
		wsID := cur.WorkspaceID
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &wsID, Actor: systemActor, Action: "provider.ips_update", TargetType: "provider",
			TargetID: cur.ID.String(), TargetName: cur.Name, Before: ipsState(cur.IpMode, cur.Ips), After: ipsState(mode, ips),
			Metadata: map[string]any{"source": state.IPsFromProvider, "trigger": trigger, "adapter": cur.Adapter}})
	})
	if errors.Is(err, pgx.ErrNoRows) {
		// Disconnected meanwhile: nothing to keep.
		return p, nil
	}
	return out, err
}

func ipsState(mode string, ips []string) map[string]any {
	if ips == nil {
		ips = []string{}
	}
	return map[string]any{"ip_mode": mode, "ips": ips}
}

// callMessage is a failed call's message as the provider or adapter wrote
// it; callText the message the adapter kept beside it.
func callMessage(err error) string {
	var ce *providers.CallError
	if errors.As(err, &ce) && ce.Message != "" {
		return ce.Message
	}
	return err.Error()
}

func callText(err error) i18n.Text {
	var ce *providers.CallError
	if errors.As(err, &ce) && ce.Message != "" {
		return ce.Text
	}
	return i18n.Text{}
}

// IPsArgs is the daily check of the providers' sending IPs.
type IPsArgs struct{}

// Kind implements river.JobArgs.
func (IPsArgs) Kind() string { return "providers.ips_check" }

// InsertOpts runs one check at a time.
func (IPsArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{Queue: jobs.QueueDefault, UniqueOpts: river.UniqueOpts{ByPeriod: time.Hour}}
}

// IPsWorker asks every connected provider whose API reports its sending IPs
// for them. Sample providers are never called.
type IPsWorker struct {
	river.WorkerDefaults[IPsArgs]
	S *Service
}

// Timeout bounds one sweep.
func (w *IPsWorker) Timeout(*river.Job[IPsArgs]) time.Duration { return 10 * time.Minute }

// Work implements river.Worker.
func (w *IPsWorker) Work(ctx context.Context, _ *river.Job[IPsArgs]) error {
	provs, err := w.S.Q.ProvListAllActive(ctx)
	if err != nil {
		return err
	}
	var errs []error
	for _, p := range provs {
		if IsDemo(p) || !state.Caps(p).ReportsIPs {
			continue
		}
		out, err := w.S.RefreshIPs(ctx, p, IPsDaily)
		if err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", p.Name, err))
			continue
		}
		if src := state.IPSourceOf(out); src.Error != "" {
			w.S.log().WarnContext(ctx, "provider's sending IPs unchecked", "provider", p.ID, "err", src.Error)
		}
	}
	return errors.Join(errs...)
}

// ProviderIPsInput sets a provider's sending IPs by hand.
type ProviderIPsInput struct {
	// Mode is "dedicated" (IPs of the account's own, listed in IPs) or
	// "shared" (the provider's pool, checked at domain level).
	Mode string   `json:"mode" enum:"shared,dedicated"`
	IPs  []string `json:"ips,omitempty" doc:"The public IPv4 or IPv6 addresses mail leaves from, up to 64, when mode is dedicated"`
}

// SetIPs sets the sending IPs of a provider whose API doesn't list them
// (self-hosted Postal, SMTP, API providers without an IP API), audited.
// A provider whose API reports them is refused: its answer wins every day.
func (s *Service) SetIPs(ctx context.Context, ws *store.Workspace, id string, in ProviderIPsInput) (ProviderCard, error) {
	p, err := s.provider(ctx, ws, id)
	if err != nil {
		return ProviderCard{}, err
	}
	l := i18n.LangFrom(ctx)
	if state.Caps(p).ReportsIPs {
		return ProviderCard{}, api.Conflict(mIPsReported.Text(l, i18n.Args{"name": p.Name}))
	}
	ips, err := manualIPs(l, in)
	if err != nil {
		return ProviderCard{}, err
	}
	now := s.now()
	src := state.IPSource{From: state.IPsFromPerson, SetAt: &now}
	if pr := api.PrincipalFrom(ctx); pr != nil {
		src.SetByName = pr.Name
		if pr.Type == authz.ActorUser && pr.UserID != uuid.Nil {
			u := pr.UserID
			src.SetBy = &u
		}
	}
	b, _ := json.Marshal(src)
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		cur, err := q.ProvLockForIPs(ctx, p.ID)
		if err != nil {
			return api.MapError(err, "Provider")
		}
		if err := q.ProvSetIPs(ctx, store.ProvSetIPsParams{ID: cur.ID, IpMode: in.Mode, Ips: ips, IpSource: b}); err != nil {
			return err
		}
		return s.audit(ctx, q, ws, cur, "provider.ips_update", ipsState(cur.IpMode, cur.Ips), ipsState(in.Mode, ips), map[string]any{"source": state.IPsFromPerson})
	})
	if err != nil {
		return ProviderCard{}, err
	}
	return s.Card(ctx, ws, p.ID)
}

// manualIPs checks the IPs a person entered: for dedicated IPs, up to
// MaxSendingIPs public addresses, kept each once and in address order; for
// a shared pool, none. A problem names the entry by its place in the list.
func manualIPs(l i18n.Lang, in ProviderIPsInput) ([]string, error) {
	entered, at := []string{}, []int{}
	for i, s := range in.IPs {
		if s = strings.TrimSpace(s); s != "" {
			entered, at = append(entered, s), append(at, i)
		}
	}
	switch {
	case in.Mode != "dedicated" && len(entered) > 0:
		return nil, api.Invalid("body.ips", mIPsShared.Text(l, nil))
	case in.Mode != "dedicated":
		return []string{}, nil
	case len(entered) == 0:
		return nil, api.Invalid("body.ips", mIPsEnter.Text(l, nil))
	case len(entered) > providers.MaxSendingIPs:
		return nil, api.Invalid("body.ips", mIPsTooMany.Text(l, i18n.Args{"max": providers.MaxSendingIPs}))
	}
	for i, s := range entered {
		a, ok := providers.ParseIP(s)
		if !ok {
			return nil, api.Invalid(fmt.Sprintf("body.ips[%d]", at[i]), mIPsNotAddress.Text(l, i18n.Args{"value": s}))
		}
		if !PublicIP(a) {
			return nil, api.Invalid(fmt.Sprintf("body.ips[%d]", at[i]), mIPsNotPublic.Text(l, i18n.Args{"ip": a.String()}))
		}
	}
	ips, _ := providers.CleanIPs(entered)
	return ips, nil
}

// CheckIPs asks a provider whose API reports its sending IPs for them now
// ("Check again" after a check failed), audited. Sample providers are never
// called. The card says what the provider answered, or why it couldn't.
func (s *Service) CheckIPs(ctx context.Context, ws *store.Workspace, id string) (ProviderCard, error) {
	p, err := s.provider(ctx, ws, id)
	if err != nil {
		return ProviderCard{}, err
	}
	if !state.Caps(p).ReportsIPs {
		return ProviderCard{}, api.Conflict(mIPsNotReported.Text(i18n.LangFrom(ctx), i18n.Args{"name": p.Name}))
	}
	if !IsDemo(p) {
		if p, err = s.RefreshIPs(ctx, p, IPsOnRequest); err != nil {
			return ProviderCard{}, err
		}
	}
	result := "ok"
	if state.IPSourceOf(p).Error != "" {
		result = "error"
	}
	if err := s.audit(ctx, s.Q, ws, p, "provider.ips_check", nil, nil, map[string]any{"result": result}); err != nil {
		return ProviderCard{}, err
	}
	return s.Card(ctx, ws, p.ID)
}

// ipSourceCard is where a provider's sending IPs come from, for the card.
func ipSourceCard(l i18n.Lang, p store.Provider) ProviderIPSource {
	src := state.IPSourceOf(p)
	out := ProviderIPSource{From: "none", Editable: !state.Caps(p).ReportsIPs}
	switch {
	case !out.Editable:
		out.From, out.CheckedAt = "provider", src.CheckedAt
		if src.From == state.IPsFromProvider && src.Error != "" {
			out.Error, out.ErrorAt = providers.TextIn(src.Error, src.ErrorText, l), src.ErrorAt
		}
	case src.From == state.IPsFromPerson:
		out.From, out.SetBy, out.SetAt = "manual", src.SetByName, src.SetAt
	}
	return out
}
