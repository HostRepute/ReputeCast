package placement

import (
	"context"
	"encoding/json"
	"fmt"
	"hash/fnv"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/placement/seedbox"
	"github.com/hostrepute/reputecast/internal/store"
)

// demoBox answers for a seed of the demo data. The demo's mailboxes don't
// exist, so a copy "arrives" a few seconds after its hand-off, in the
// folder a mailbox provider would plausibly pick, with the headers it
// would write. Nothing leaves ReputeCast to read it.
type demoBox struct {
	s    *Service
	seed store.PlacementSeed
}

func (b *demoBox) Close() error { return nil }

// Folders returns the provider's folders as the demo seeds list them.
func (b *demoBox) Folders(context.Context) ([]seedbox.Folder, error) {
	return DemoFolders(b.seed.Provider, b.seed.Auth), nil
}

// DemoFolders are the folders a seed of provider shows (the seeder uses them too).
func DemoFolders(provider, auth string) []seedbox.Folder {
	p, _ := seedbox.ProviderOf(provider)
	out := []seedbox.Folder{}
	switch {
	case seedbox.IsMicrosoft(provider) && auth == seedbox.AuthOAuth:
		out = append(out, seedbox.Folder{Role: seedbox.RoleInbox, Name: "inbox", Label: "Focused"},
			seedbox.Folder{Role: seedbox.RoleInbox, Name: "inbox-other", Label: "Other"})
	default:
		out = append(out, seedbox.Folder{Role: seedbox.RoleInbox, Name: "INBOX", Label: p.InboxName})
	}
	if seedbox.IsGoogle(provider) {
		for _, t := range p.Tabs[1:] {
			out = append(out, seedbox.Folder{Role: seedbox.RoleTab, Name: "INBOX", Label: t})
		}
	}
	spam := p.SpamName
	name := spam
	if seedbox.IsGoogle(provider) {
		name = "[Gmail]/Spam"
	}
	return append(out, seedbox.Folder{Role: seedbox.RoleSpam, Name: name, Label: spam})
}

// DemoOutcome is where a demo copy lands and how long it takes.
type DemoOutcome struct {
	Role, Folder, Tab string
	After             time.Duration
}

// DemoLanding picks a demo copy's folder: Gmail sorts newsletters into
// Promotions, Microsoft into Focused or Other, a warming-up IP lands in
// Outlook's and Yahoo's spam, and transactional mail goes to the inbox.
func DemoLanding(seedProvider, stream string, warming bool, token string) DemoOutcome {
	h := fnv.New32a()
	_, _ = h.Write([]byte(token))
	after := time.Duration(8+h.Sum32()%40) * time.Second
	p, _ := seedbox.ProviderOf(seedProvider)
	inbox := DemoOutcome{Role: seedbox.RoleInbox, Folder: p.InboxName, After: after}
	transactional := stream == "transactional" || stream == "system"
	switch seedProvider {
	case seedbox.Gmail:
		if transactional {
			inbox.Tab = "Primary"
			return inbox
		}
		return DemoOutcome{Role: seedbox.RoleTab, Folder: "INBOX", Tab: "Promotions", After: after}
	case seedbox.GoogleWorkspace:
		inbox.Tab = "Primary"
		return inbox
	case seedbox.Outlook:
		if warming && !transactional {
			return DemoOutcome{Role: seedbox.RoleSpam, Folder: p.SpamName, After: after + 30*time.Second}
		}
		inbox.Tab = "Focused"
		return inbox
	case seedbox.Microsoft365:
		inbox.Tab = "Other"
		return inbox
	case seedbox.Yahoo:
		if warming && !transactional {
			return DemoOutcome{Role: seedbox.RoleSpam, Folder: p.SpamName, After: 2*time.Minute + 10*time.Second}
		}
	}
	return inbox
}

// Find returns the demo copies that have "arrived" by now.
func (b *demoBox) Find(ctx context.Context, tokens []string, _ time.Time) ([]seedbox.Found, error) {
	copies, err := b.s.Q.PlacementCopiesByTokens(ctx, tokens)
	if err != nil {
		return nil, err
	}
	now := b.s.now()
	out := []seedbox.Found{}
	tests := map[string]store.PlacementTest{}
	for _, c := range copies {
		if c.HandedAt == nil || c.SeedID == nil || *c.SeedID != b.seed.ID {
			continue
		}
		t, ok := tests[c.TestID.String()]
		if !ok {
			if t, err = b.s.Q.PlacementTestByID(ctx, c.TestID); err != nil {
				continue
			}
			tests[c.TestID.String()] = t
		}
		ps := snapshots(t)[providerKey(c.ProviderID)]
		o := DemoLanding(c.SeedProvider, t.Stream, ps.WarmDay > 0, c.Token)
		at := c.HandedAt.Add(o.After)
		if at.After(now) {
			continue
		}
		raw := DemoHeaders(DemoMail{SeedProvider: c.SeedProvider, Host: ps.Detail, IP: ps.IP, From: t.FromEmail, MessageID: c.ID.String(), Token: c.Token, At: at})
		out = append(out, seedbox.Found{Token: c.Token, Role: o.Role, Folder: o.Folder, Tab: o.Tab, ReceivedAt: at,
			Header: seedbox.ParseHeader([]byte(raw)), Ref: seedbox.Ref{Folder: o.Folder}})
	}
	return out, nil
}

// Delete "removes" the demo copies: it counts the ones that arrived.
func (b *demoBox) Delete(ctx context.Context, tokens []string, since time.Time) (int, error) {
	found, err := b.Find(ctx, tokens, since)
	return len(found), err
}

func providerKey(id *uuid.UUID) string {
	if id == nil {
		return ""
	}
	return id.String()
}

// DemoMail describes a demo copy for its headers.
type DemoMail struct {
	SeedProvider string
	// Host and IP are the sending server's.
	Host, IP string
	From     string
	// MessageID is the Message-ID's left part (the copy's id by default).
	MessageID string
	Token     string
	// Selector and SPFDomain are the DKIM selector and the Return-Path
	// domain ("7kq2n3x", "bounce.news.acme.io").
	Selector, SPFDomain string
	At                  time.Time
}

// DemoHeaders writes the headers a mailbox provider would show for a demo
// copy: its Authentication-Results (all passing), the Received line of the
// hand-over, the Message-ID and the test token.
func DemoHeaders(m DemoMail) string {
	domain := "example.com"
	if i := strings.LastIndexByte(m.From, '@'); i >= 0 {
		domain = m.From[i+1:]
	}
	host, ip := m.Host, m.IP
	if host == "" || !strings.Contains(host, ".") {
		host = "mail." + domain
	}
	if ip == "" {
		ip = "198.51.100.24"
	}
	selector, spf := firstNonEmpty(m.Selector, "rc2026"), firstNonEmpty(m.SPFDomain, "bounce."+domain)
	date := m.At.UTC().Format(time.RFC1123Z)
	var auth, received string
	switch {
	case seedbox.IsGoogle(m.SeedProvider):
		auth = fmt.Sprintf("mx.google.com;\r\n       dkim=pass header.i=@%s header.s=%s;\r\n       spf=pass (google.com: domain of bounce@%s designates %s as permitted sender) smtp.mailfrom=bounce@%s;\r\n       dmarc=pass (p=QUARANTINE sp=QUARANTINE dis=NONE) header.from=%s",
			domain, selector, spf, ip, spf, domain)
		received = fmt.Sprintf("from %s (%s. [%s])\r\n        by mx.google.com with ESMTPS id 6a1803df08f44-6f5d3e2b9b1si1144906d6.112;\r\n        %s", host, host, ip, date)
	case seedbox.IsMicrosoft(m.SeedProvider):
		auth = fmt.Sprintf("spf=pass (sender IP is %s)\r\n smtp.mailfrom=%s; dkim=pass (signature was verified)\r\n header.d=%s;dmarc=pass action=none header.from=%s;\r\n compauth=pass reason=100", ip, spf, domain, domain)
		received = fmt.Sprintf("from %s (%s) by\r\n AM4PEPF00027A6C.mail.protection.outlook.com\r\n (10.167.16.90) with Microsoft SMTP Server\r\n (version=TLS1_3); %s", host, ip, date)
	case m.SeedProvider == seedbox.Yahoo || m.SeedProvider == seedbox.AOL:
		auth = fmt.Sprintf("atlas220.free.mail.ne1.yahoo.com;\r\n dkim=pass header.i=@%s header.s=%s;\r\n spf=pass smtp.mailfrom=%s;\r\n dmarc=pass(p=QUARANTINE) header.from=%s;", domain, selector, spf, domain)
		received = fmt.Sprintf("from %s (EHLO %s) (%s)\r\n by atlas220.free.mail.ne1.yahoo.com with ESMTPS; %s", host, host, ip, date)
	case m.SeedProvider == seedbox.ICloud:
		auth = fmt.Sprintf("dmarc.icloud.com; dmarc=pass header.from=%s;\r\n dkim=pass header.d=%s header.s=%s; spf=pass smtp.mailfrom=%s", domain, domain, selector, spf)
		received = fmt.Sprintf("from %s ([%s]) by mx6.mail.icloud.com with ESMTPS; %s", host, ip, date)
	default:
		by := "mx.gmx.net"
		if m.SeedProvider == seedbox.WebDE {
			by = "mx.web.de"
		}
		auth = fmt.Sprintf("%s; dkim=pass header.d=%s header.s=%s;\r\n spf=pass smtp.mailfrom=%s; dmarc=pass header.from=%s", by, domain, selector, spf, domain)
		received = fmt.Sprintf("from %s ([%s]) by %s with ESMTPS; %s", host, ip, by, date)
	}
	id := firstNonEmpty(m.MessageID, m.Token)
	return "Authentication-Results: " + auth + "\r\n" +
		"Received: " + received + "\r\n" +
		fmt.Sprintf("DKIM-Signature: v=1; a=rsa-sha256; d=%s; s=%s; c=relaxed/relaxed\r\n", domain, selector) +
		fmt.Sprintf("Message-ID:\r\n <%s@%s>\r\n", id, domain) +
		seedbox.TestHeader + ": " + m.Token + "\r\n"
}

// DemoAuth is a demo copy's parsed authentication results (the seeder
// stores it with the copy).
func DemoAuth(raw string) json.RawMessage {
	b, _ := json.Marshal(seedbox.ParseHeader([]byte(raw)).AuthResults())
	return b
}

// DemoHop is a demo copy's hand-over hop.
func DemoHop(raw string) json.RawMessage {
	hop := seedbox.ParseHeader([]byte(raw)).HandOver()
	if hop == nil {
		return json.RawMessage(`{}`)
	}
	b, _ := json.Marshal(hop)
	return b
}
