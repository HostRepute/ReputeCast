// Package channels is the channel model shared by the ways ReputeCast
// reaches people (design expansion phase 8): email, SMS, push and WhatsApp.
//
// It holds what every channel after email has in common, so push and
// WhatsApp (X22) slot in next to SMS without reworking it:
//   - the channel switch of Sending providers, from a registry of channels
//     and whether each is set up in a workspace;
//   - quiet hours in the recipient's time zone (quiet.go);
//   - consent per list and per channel, with its evidence;
//   - the message log of texts, notifications and WhatsApp messages, kept
//     90 days in its own tables, never in the audit log.
//
// Email keeps its own model (list memberships, consent events, messages);
// this package never changes it.
package channels

import (
	"context"
	"slices"
	"sync"

	"github.com/google/uuid"
)

// Channel is a way to reach people.
type Channel string

// Channels, in the switch's order.
const (
	Email    Channel = "email"
	SMS      Channel = "sms"
	Push     Channel = "push"
	WhatsApp Channel = "whatsapp"
)

// All lists the channels in the order the app shows them.
var All = []Channel{Email, SMS, Push, WhatsApp}

// Valid reports whether c is a known channel.
func Valid(c Channel) bool { return slices.Contains(All, c) }

// Info registers a channel's setup state for the switch.
type Info struct {
	Channel Channel
	// SetUp reports whether the workspace can send on the channel: a
	// connected provider for email, SMS and WhatsApp, keys and a site for
	// push.
	SetUp func(ctx context.Context, workspaceID uuid.UUID) (bool, error)
}

var (
	mu       sync.RWMutex
	registry = map[Channel]Info{}
)

// Register adds a channel; each channel's domain calls it from its wiring.
// A channel without an entry is shown as not built yet.
func Register(i Info) {
	mu.Lock()
	defer mu.Unlock()
	registry[i.Channel] = i
}

// Lookup returns a channel's entry.
func Lookup(c Channel) (Info, bool) {
	mu.RLock()
	defer mu.RUnlock()
	i, ok := registry[c]
	return i, ok
}

// ChannelStatus is one channel in the switch.
type ChannelStatus struct {
	Channel Channel `json:"channel" enum:"email,sms,push,whatsapp"`
	// Available is false for a channel this build doesn't have yet.
	Available bool `json:"available"`
	// SetUp is true once the workspace can send on it.
	SetUp bool `json:"setUp"`
}

// Statuses returns every channel's state in a workspace. A channel whose
// check fails counts as not set up, so the switch still renders.
func Statuses(ctx context.Context, workspaceID uuid.UUID) []ChannelStatus {
	out := make([]ChannelStatus, 0, len(All))
	for _, c := range All {
		s := ChannelStatus{Channel: c}
		if i, ok := Lookup(c); ok && i.SetUp != nil {
			s.Available = true
			if up, err := i.SetUp(ctx, workspaceID); err == nil {
				s.SetUp = up
			}
		}
		out = append(out, s)
	}
	return out
}

// SwitchShown is the rule for the channel switch: it appears once a second
// channel is set up. Until then the header offers "Add a channel".
func SwitchShown(statuses []ChannelStatus) bool {
	n := 0
	for _, s := range statuses {
		if s.SetUp {
			n++
		}
	}
	return n >= 2
}
