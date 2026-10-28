package tracking

import (
	"context"
	"errors"
	"net/http"
	"net/netip"
	"strings"
	"sync"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/store"
)

// The networks of machines that open and click email (privacy proxies,
// mail-security scanners) are a list the instance keeps
// (tracking_networks): the built-in ranges come with releases, and
// instance admins add the ones they meet (review AN-2). The classifier
// reads the list from memory, refreshed every few minutes.

// netRefresh is how long the list is kept in memory.
const netRefresh = 5 * time.Minute

type netEntry struct {
	prefix netip.Prefix
	Network
}

type netCache struct {
	mu   sync.RWMutex
	at   time.Time
	list []netEntry
}

// networks returns the instance's list, read again when it is older than
// netRefresh; nil when it can't be read (the classifier then uses the
// built-in ranges).
func (s *Service) networks(ctx context.Context) []netEntry {
	s.nets.mu.RLock()
	list, at := s.nets.list, s.nets.at
	s.nets.mu.RUnlock()
	if list != nil && time.Since(at) < netRefresh {
		return list
	}
	rows, err := s.Q.TrackingNetworkList(ctx)
	if err != nil {
		s.log().WarnContext(ctx, "read tracking networks", "err", err)
		return list
	}
	fresh := make([]netEntry, 0, len(rows))
	for _, r := range rows {
		p, err := netip.ParsePrefix(r.Cidr)
		if err != nil {
			continue
		}
		fresh = append(fresh, netEntry{prefix: p.Masked(), Network: Network{Class: r.Class, Name: r.Name}})
	}
	s.nets.mu.Lock()
	s.nets.list, s.nets.at = fresh, time.Now()
	s.nets.mu.Unlock()
	return fresh
}

// forgetNetworks makes the next lookup read the list again.
func (s *Service) forgetNetworks() {
	s.nets.mu.Lock()
	s.nets.at = time.Time{}
	s.nets.mu.Unlock()
}

// networkOf is the machine network of an address: the most specific of
// the instance's networks holding it, an empty Network when none does,
// nil when the list can't be read.
func (s *Service) networkOf(ctx context.Context, ip netip.Addr) *Network {
	if !ip.IsValid() {
		return nil
	}
	list := s.networks(ctx)
	if list == nil {
		return nil
	}
	ip = ip.Unmap()
	best := -1
	out := &Network{}
	for _, n := range list {
		if n.prefix.Contains(ip) && n.prefix.Bits() > best {
			best = n.prefix.Bits()
			nw := n.Network
			out = &nw
		}
	}
	return out
}

// TrackingNetwork is one network of privacy proxies or mail scanners.
type TrackingNetwork struct {
	CIDR string `json:"cidr" example:"203.0.113.0/24"`
	Name string `json:"name"`
	// Class: privacy (a privacy proxy fetches images on delivery; its opens
	// are privacy-protected) or scanner (a mail-security scanner; its opens
	// and clicks on arrival are a machine's).
	Class string `json:"class" enum:"privacy,scanner"`
	// Source: builtin (comes with ReputeCast) or admin (added here).
	Source        string    `json:"source" enum:"builtin,admin"`
	Note          string    `json:"note"`
	CreatedAt     time.Time `json:"createdAt"`
	CreatedByName string    `json:"createdByName,omitempty"`
}

// TrackingNetworkList lists the networks.
type TrackingNetworkList struct {
	Items []TrackingNetwork `json:"items"`
}

// TrackingNetworkInput adds a network.
type TrackingNetworkInput struct {
	CIDR  string `json:"cidr" minLength:"3" maxLength:"64" doc:"An address or a network: 203.0.113.0/24, 2001:db8::/48"`
	Name  string `json:"name" minLength:"1" maxLength:"80" doc:"Who runs it: Mimecast, our mail gateway"`
	Class string `json:"class" enum:"privacy,scanner"`
	Note  string `json:"note,omitempty" maxLength:"300"`
}

// widest are the widest networks an admin may add: a wider one would take
// ordinary readers for machines.
const (
	widestV4 = 8
	widestV6 = 16
)

func parseNetwork(s string) (netip.Prefix, bool) {
	s = strings.TrimSpace(s)
	if !strings.Contains(s, "/") {
		a, err := netip.ParseAddr(s)
		if err != nil {
			return netip.Prefix{}, false
		}
		return netip.PrefixFrom(a.Unmap(), a.Unmap().BitLen()), true
	}
	p, err := netip.ParsePrefix(s)
	if err != nil {
		return netip.Prefix{}, false
	}
	if p.Addr().Is4In6() {
		p = netip.PrefixFrom(p.Addr().Unmap(), p.Bits()-96)
	}
	return p.Masked(), true
}

// Networks lists the instance's networks.
func (s *Service) Networks(ctx context.Context) (TrackingNetworkList, error) {
	rows, err := s.Q.TrackingNetworkList(ctx)
	if err != nil {
		return TrackingNetworkList{}, err
	}
	out := TrackingNetworkList{Items: make([]TrackingNetwork, 0, len(rows))}
	for _, r := range rows {
		out.Items = append(out.Items, TrackingNetwork{CIDR: r.Cidr, Name: r.Name, Class: r.Class, Source: r.Source, Note: r.Note, CreatedAt: r.CreatedAt,
			CreatedByName: r.CreatedByName})
	}
	return out, nil
}

// AddNetwork adds a network to the list (audited).
func (s *Service) AddNetwork(ctx context.Context, in TrackingNetworkInput) (TrackingNetwork, error) {
	l := i18n.LangFrom(ctx)
	p, ok := parseNetwork(in.CIDR)
	if !ok {
		return TrackingNetwork{}, api.Invalid("body.cidr", msgs.M("problem.networkInvalid").Text(l, nil))
	}
	if (p.Addr().Is4() && p.Bits() < widestV4) || (p.Addr().Is6() && p.Bits() < widestV6) {
		return TrackingNetwork{}, api.Invalid("body.cidr", msgs.M("problem.networkTooWide").Text(l, i18n.Args{"v4": widestV4, "v6": widestV6}))
	}
	name, note := strings.TrimSpace(in.Name), strings.TrimSpace(in.Note)
	var by *uuid.UUID
	if pr := api.PrincipalFrom(ctx); pr != nil && pr.UserID != uuid.Nil {
		u := pr.UserID
		by = &u
	}
	var out TrackingNetwork
	err := db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		cidr, err := q.TrackingNetworkAdd(ctx, store.TrackingNetworkAddParams{Cidr: p, Name: name, Class: in.Class, Note: note, CreatedBy: by})
		if errors.Is(err, pgx.ErrNoRows) {
			return api.Conflict(msgs.M("problem.networkListed").Text(l, i18n.Args{"cidr": p.String()}))
		}
		if err != nil {
			return err
		}
		out = TrackingNetwork{CIDR: cidr, Name: name, Class: in.Class, Source: "admin", Note: note, CreatedAt: s.now()}
		if p := api.PrincipalFrom(ctx); p != nil {
			out.CreatedByName = p.Name
		}
		return s.Audit.LogWith(ctx, q, audit.Event{Action: "tracking_network.create", TargetType: "tracking_network", TargetID: cidr, TargetName: name,
			After: map[string]any{"cidr": cidr, "name": name, "class": in.Class, "note": note}})
	})
	if err != nil {
		return TrackingNetwork{}, err
	}
	s.forgetNetworks()
	return out, nil
}

// RemoveNetwork takes a network an admin added off the list (audited).
func (s *Service) RemoveNetwork(ctx context.Context, cidr string) error {
	l := i18n.LangFrom(ctx)
	p, ok := parseNetwork(cidr)
	if !ok {
		return api.Invalid("query.cidr", msgs.M("problem.networkInvalid").Text(l, nil))
	}
	err := db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		row, err := q.TrackingNetworkDelete(ctx, p)
		if errors.Is(err, pgx.ErrNoRows) {
			return api.Err(http.StatusNotFound, api.CodeNotFound, msgs.M("problem.networkNotRemovable").Text(l, i18n.Args{"cidr": p.String()}))
		}
		if err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{Action: "tracking_network.delete", TargetType: "tracking_network", TargetID: row.Cidr, TargetName: row.Name,
			Before: map[string]any{"cidr": row.Cidr, "name": row.Name, "class": row.Class, "note": row.Note}})
	})
	if err != nil {
		return err
	}
	s.forgetNetworks()
	return nil
}

// RegisterNetworks adds the instance admin's operations on the bot filter's
// lists: the networks and the user agents (agents.go).
func RegisterNetworks(h huma.API, s *Service) {
	tags := []string{"Instance admin · Tracking"}
	registerUserAgents(h, s, tags)
	base := "/admin/tracking-networks"
	api.Get(h, api.Op{ID: "list-tracking-networks", Path: base, Summary: "Networks of privacy proxies and mail scanners whose opens and clicks aren't people's",
		Tags: tags, Access: api.InstanceAdmin},
		func(ctx context.Context, _ *struct{}) (*api.Body[TrackingNetworkList], error) {
			out, err := s.Networks(ctx)
			return api.OK(out), err
		})
	api.Post(h, api.Op{ID: "add-tracking-network", Path: base, Summary: "Add a network of a privacy proxy or a mail scanner (audited)",
		Description: "Opens from a privacy proxy's network count as privacy-protected; opens and clicks from a scanner's network on a message's arrival count as a machine's.",
		Tags:        tags, Access: api.InstanceAdmin, Status: http.StatusCreated},
		func(ctx context.Context, in *struct{ Body TrackingNetworkInput }) (*api.Body[TrackingNetwork], error) {
			out, err := s.AddNetwork(ctx, in.Body)
			return api.OK(out), err
		})
	api.Delete(h, api.Op{ID: "remove-tracking-network", Path: base, Summary: "Remove a network an admin added (audited)", Tags: tags, Access: api.InstanceAdmin},
		func(ctx context.Context, in *struct {
			CIDR string `query:"cidr" required:"true" maxLength:"64" doc:"The network, as listed: 203.0.113.0/24"`
		}) (*api.Empty, error) {
			return api.NoContent, s.RemoveNetwork(ctx, in.CIDR)
		})
}
