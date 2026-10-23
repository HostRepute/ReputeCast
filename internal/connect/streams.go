package connect

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/providers"
	"github.com/hostrepute/reputecast/internal/providers/state"
	"github.com/hostrepute/reputecast/internal/routing"
	"github.com/hostrepute/reputecast/internal/store"
)

// ConnectStreamView is one stream card of step 5.
type ConnectStreamView struct {
	Stream      string `json:"stream" enum:"broadcast,automation,system,transactional"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Enabled     bool   `json:"enabled"`
	// Allowed is false when the provider's terms exclude the stream
	// (bulk marketing); Reason says why.
	Allowed bool   `json:"allowed"`
	Reason  string `json:"reason,omitempty"`
	// HasRoute: another provider already carries the stream, so the role
	// can be chosen (primary, split, fallback).
	HasRoute bool     `json:"hasRoute"`
	Current  []string `json:"current"`
	Role     string   `json:"role" enum:"primary,split,fallback"`
	Weight   int      `json:"weight"`
	// RoleLabel is the badge ("Primary · 100%").
	RoleLabel        string `json:"roleLabel"`
	ConfigurationSet string `json:"configurationSet"`
	Failover         string `json:"failover"`
	// Tracking is shown for Broadcasts ("ReputeCast only; SES tracking off").
	Tracking string `json:"tracking"`
}

// ConnectStreams is step 5.
type ConnectStreams struct {
	Streams []ConnectStreamView `json:"streams"`
	Saved   bool                `json:"saved"`
	// Tips are the "Suggested next steps"; Defaults the line about
	// throttles and thresholds.
	Tips     []string `json:"tips"`
	Defaults string   `json:"defaults"`
}

// policyKey names a failover policy for the messages (failover.*).
func policyKey(p routing.Policy) string {
	switch p {
	case routing.PauseAndAsk:
		return "pause"
	case routing.WarmOnly:
		return "warm"
	}
	return "auto"
}

// streamsView builds step 5 from the saved routing and the setup's choices.
func (s *Service) streamsView(ctx context.Context, l i18n.Lang, ws *store.Workspace, d providers.Descriptor, c adapterCopy, p store.Provider, st State) (ConnectStreams, error) {
	out := ConnectStreams{Streams: []ConnectStreamView{}, Tips: []string{}}
	set, err := routing.Load(ctx, s.Q, ws.ID)
	if err != nil {
		return out, err
	}
	provs, err := s.Q.ProvListActive(ctx, ws.ID)
	if err != nil {
		return out, err
	}
	short := map[uuid.UUID]string{}
	for _, pr := range provs {
		short[pr.ID] = state.Short(pr.Name, pr.Adapter)
	}
	var cfg struct {
		Sets map[string]string `json:"configuration_sets"`
	}
	_ = json.Unmarshal(p.Config, &cfg)
	caps := state.Caps(p)
	track := state.TrackingOf(p)
	if st.Streams != nil {
		out.Saved = st.Streams.Saved
	}
	for _, stream := range routing.Streams {
		key := string(stream)
		v := ConnectStreamView{Stream: key, Name: streamName(l, key), Description: mStreamDescription.Text(l, i18n.Args{"stream": key}), Allowed: true, Current: []string{}}
		rule, has := set.Default(stream)
		mine := false
		if has {
			for _, sp := range rule.Splits {
				if sp.ProviderID == p.ID {
					mine = true
					continue
				}
				label := short[sp.ProviderID]
				if len(rule.Splits) > 1 {
					label += " " + strconv.Itoa(sp.Weight) + "%"
				}
				v.Current = append(v.Current, label)
			}
			v.HasRoute = len(v.Current) > 0
		}
		if stream == providers.StreamBroadcast && !caps.BulkAllowed() {
			v.Allowed = false
			v.Reason = mStreamsNoBulk.Text(l, i18n.Args{"name": d.Name})
			if caps.Bulk == providers.BulkBroadcastStreams {
				v.Reason = mStreamsBroadcastOnly.Text(l, i18n.Args{"name": d.Name})
			}
		}
		// Defaults: streams nobody carries yet, and the ones it already
		// carries; a second provider is offered as a fallback.
		v.Enabled = v.Allowed && (!v.HasRoute || mine)
		v.Role, v.Weight = "primary", 100
		if v.HasRoute && !mine {
			v.Role = "fallback"
			if stream == providers.StreamBroadcast {
				v.Role, v.Weight = "split", 10
			}
		}
		if st.Streams != nil {
			if ch, ok := st.Streams.Streams[key]; ok {
				v.Enabled = ch.Enabled && v.Allowed
				if ch.Role != "" {
					v.Role = ch.Role
				}
				if ch.Weight > 0 {
					v.Weight = ch.Weight
				}
			}
		}
		role := v.Role
		if role == "primary" && stream == providers.StreamBroadcast {
			role = "primaryAll"
		}
		v.RoleLabel = mRoleLabel.Text(l, i18n.Args{"role": role, "weight": strconv.Itoa(v.Weight)})
		v.ConfigurationSet = cfg.Sets[key]
		policy := policyKey(set.Policy(stream))
		if stream == providers.StreamBroadcast {
			v.Failover = mFailoverPolicy.Text(l, i18n.Args{"policy": policy})
			switch {
			case !track.OpenPixel && !track.LinkRewrite:
				v.Tracking = mStreamsTrackingOff.Text(l, i18n.Args{"short": c.Short})
			default:
				v.Tracking = mStreamsTrackingOn.Text(l, i18n.Args{"short": c.Short})
			}
		} else {
			fallback := ""
			if has && rule.Fallback != nil && *rule.Fallback != p.ID {
				fallback = short[*rule.Fallback]
			}
			if v.Role == "primary" && has && len(rule.Splits) > 0 && rule.Splits[0].ProviderID != p.ID && fallback == "" {
				fallback = short[rule.Splits[0].ProviderID]
			}
			switch {
			case v.Role == "fallback":
				v.Failover = mFailoverTakesOver.Text(l, i18n.Args{"policy": policy, "current": i18n.AndList(l.Locale, v.Current)})
			case fallback != "":
				v.Failover = mFailoverFallsBack.Text(l, i18n.Args{"policy": policy, "fallback": fallback})
			default:
				v.Failover = mFailoverNoFallback.Text(l, i18n.Args{"policy": policy})
			}
		}
		out.Streams = append(out.Streams, v)
	}
	second := "Postmark"
	if d.Adapter == "postmark" {
		second = "SES"
	}
	out.Tips = append(out.Tips, mTipsSecond.Text(l, i18n.Args{"second": second, "short": c.Short}), mTipsSplit.Text(l, nil))
	out.Defaults = s.defaultsLine(ctx, l, ws)
	return out, nil
}

// defaultsLine is the "Set" tip: throttles and auto-pause thresholds.
func (s *Service) defaultsLine(ctx context.Context, l i18n.Lang, ws *store.Workspace) string {
	def, err := s.sendingDefaults(ctx, ws)
	if err != nil {
		return ""
	}
	parts := []string{}
	for _, mb := range []string{"outlook", "yahoo", "icloud"} {
		if v := def.Throttles[mb]; v > 0 {
			parts = append(parts, mThrottleItem.Text(l, i18n.Args{"mailbox": routing.MailboxName(mb), "rate": v}))
		}
	}
	return mStreamsDefaults.Text(l, i18n.Args{"hasThrottles": yesNo(len(parts) > 0), "throttles": strings.Join(parts, ", "), "bounce": def.Bounce, "complaint": def.Complaint})
}

// ConnectStreamChoiceInput is one stream of step 5.
type ConnectStreamChoiceInput struct {
	Stream  string `json:"stream" enum:"broadcast,automation,system,transactional"`
	Enabled bool   `json:"enabled"`
	Role    string `json:"role,omitempty" enum:"primary,split,fallback"`
	Weight  int    `json:"weight,omitempty" minimum:"1" maximum:"99"`
}

// ConnectStreamsInput assigns the provider to streams.
type ConnectStreamsInput struct {
	Streams []ConnectStreamChoiceInput `json:"streams" maxItems:"4"`
}

// SaveStreams assigns the provider to the chosen streams through the
// routing rules (primary, a weighted split or the fallback) and records the
// choice on the setup.
func (s *Service) SaveStreams(ctx context.Context, ws *store.Workspace, rawID string, in ConnectStreamsInput) (ConnectSetup, error) {
	cur, err := s.setup(ctx, ws, rawID)
	if err != nil {
		return ConnectSetup{}, err
	}
	p, err := s.provider(ctx, ws, cur)
	if err != nil {
		return ConnectSetup{}, err
	}
	cfg, err := s.Routing.Get(ctx, ws)
	if err != nil {
		return ConnectSetup{}, err
	}
	l := i18n.LangFrom(ctx)
	caps := state.Caps(p)
	choices := map[string]StreamChoice{}
	save := routing.RoutingSaveInput{}
	for i, ch := range in.Streams {
		choices[ch.Stream] = StreamChoice{Enabled: ch.Enabled, Role: ch.Role, Weight: ch.Weight}
		if !ch.Enabled {
			continue
		}
		if ch.Stream == "broadcast" && !caps.BulkAllowed() {
			return ConnectSetup{}, api.Invalid(fmt.Sprintf("body.streams[%d].enabled", i), mProblemNoBroadcasts.Text(l, i18n.Args{"name": p.Name}))
		}
		var view *routing.RoutingStreamView
		for j := range cfg.Streams {
			if cfg.Streams[j].Stream == ch.Stream {
				view = &cfg.Streams[j]
			}
		}
		next := routing.RoutingStreamInput{Stream: ch.Stream, Splits: []routing.RoutingSplit{}, Conditions: []routing.RoutingCondition{},
			Failover: string(routing.DefaultPolicy(providers.Stream(ch.Stream)))}
		if view != nil {
			next.Splits = slices.Clone(view.Splits)
			next.Fallback, next.Failover = view.Fallback, view.Failover
			for _, c := range view.Conditions {
				next.Conditions = append(next.Conditions, c.RoutingCondition)
			}
		}
		pid := p.ID.String()
		role := ch.Role
		if role == "" || len(next.Splits) == 0 {
			role = "primary"
		}
		switch role {
		case "primary":
			old := ""
			if len(next.Splits) > 0 && next.Splits[0].ProviderID != pid {
				old = next.Splits[0].ProviderID
			}
			next.Splits = []routing.RoutingSplit{{ProviderID: pid, Weight: 100}}
			if next.Fallback == pid {
				next.Fallback = ""
			}
			if ch.Stream != "broadcast" && next.Fallback == "" && old != "" {
				next.Fallback = old
			}
		case "split":
			if ch.Stream != "broadcast" {
				return ConnectSetup{}, api.Invalid(fmt.Sprintf("body.streams[%d].role", i), mProblemSplitBroadcastsOnly.Text(l, nil))
			}
			w := ch.Weight
			if w <= 0 {
				w = 10
			}
			next.Splits = splitWith(next.Splits, pid, w)
		case "fallback":
			if len(next.Splits) == 1 && next.Splits[0].ProviderID == pid {
				return ConnectSetup{}, api.Invalid(fmt.Sprintf("body.streams[%d].role", i), mProblemAlreadyCarries.Text(l, i18n.Args{"name": p.Name}))
			}
			next.Fallback = pid
		}
		save.Streams = append(save.Streams, next)
	}
	if len(save.Streams) > 0 {
		if _, err := s.Routing.Save(ctx, ws, save); err != nil {
			var pr *api.Problem
			if errors.As(err, &pr) {
				return ConnectSetup{}, pr
			}
			return ConnectSetup{}, err
		}
		if s.OnRoutingChange != nil {
			s.OnRoutingChange()
		}
	}
	now := s.now()
	st, err := s.update(ctx, ws, cur.ID, "provider.setup_streams", map[string]any{"provider": p.Name, "streams": in.Streams}, func(_ *store.ProviderSetup, state *State) error {
		state.Streams = &StreamsStep{Saved: true, At: &now, Streams: choices}
		return nil
	})
	if err != nil {
		return ConnectSetup{}, err
	}
	return s.View(ctx, ws, st)
}

// splitWith gives pid weight w and scales the other shares to 100 - w
// (largest remainder, so the weights add up to 100).
func splitWith(splits []routing.RoutingSplit, pid string, w int) []routing.RoutingSplit {
	others := []routing.RoutingSplit{}
	total := 0
	for _, sp := range splits {
		if sp.ProviderID != pid {
			others = append(others, sp)
			total += sp.Weight
		}
	}
	rest := 100 - w
	out := []routing.RoutingSplit{}
	if total == 0 || len(others) == 0 {
		return []routing.RoutingSplit{{ProviderID: pid, Weight: 100}}
	}
	type part struct {
		i    int
		frac float64
	}
	sum := 0
	parts := []part{}
	for i, o := range others {
		exact := float64(o.Weight) * float64(rest) / float64(total)
		n := int(exact)
		sum += n
		out = append(out, routing.RoutingSplit{ProviderID: o.ProviderID, Weight: n})
		parts = append(parts, part{i, exact - float64(n)})
	}
	slices.SortFunc(parts, func(a, b part) int {
		switch {
		case a.frac > b.frac:
			return -1
		case a.frac < b.frac:
			return 1
		}
		return a.i - b.i
	})
	for k := 0; sum < rest && k < len(parts); k++ {
		out[parts[k].i].Weight++
		sum++
	}
	kept := out[:0]
	for _, o := range out {
		if o.Weight > 0 {
			kept = append(kept, o)
		}
	}
	return append(kept, routing.RoutingSplit{ProviderID: pid, Weight: w})
}
