package demo

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/emailaddr"
)

func init() { seedWebsiteComments = seedWebsiteCommentsImpl }

// Comments on the website's posts (M61, G-27; boards PublicPostComments,
// WebsiteComments and WebsiteCommentsStates): comments are on for
// signed-in members and a member's first comment waits for review. 23
// comments on two posts: "What shipped in October" has 13 published (Sam
// Rivera's pinned, with the publication's reply written by Sofia Rossi), 2
// waiting (Alex Murphy's first comment, Kenji W.'s with two links) and
// Brandboost Media's spam, hidden by Sofia; "Ask Acme" has 6 published and
// Omar Haddad's first comment waiting.

// demoCommenter is a reader who comments: their address and the name their
// comments show.
type demoCommenter struct{ email, first, last, display string }

var demoCommenters = map[string]demoCommenter{
	"sam":     {"sam.rivera@outlook.com", "Sam", "Rivera", "Sam Rivera"},
	"alex":    {"a.murphy@icloud.com", "Alex", "Murphy", "Alex Murphy"},
	"priya":   {"priya.n@gmail.com", "Priya", "Natarajan", "Priya Natarajan"},
	"ben":     {"ben@okoro.example", "Ben", "Okoro", "Ben Okoro"},
	"mateo":   {"mateo.silva@gmail.com", "Mateo", "Silva", "Mateo Silva"},
	"grace":   {"g.okafor@outlook.com", "Grace", "Okafor", "Grace Okafor"},
	"olga":    {"olga.bennett@gmail.com", "Olga", "Bennett", "Olga Bennett"},
	"diego":   {"d.alvarez@yahoo.com", "Diego", "Alvarez", "Diego Alvarez"},
	"lena":    {"lena.fischer@hotmail.com", "Lena", "Fischer", "Lena Fischer"},
	"hannah":  {"hannah@bergconsulting.de", "Hannah", "Berg", "Hannah Berg"},
	"tomasz":  {"t.nowak@gmail.com", "Tomasz", "Nowak", "Tomasz Nowak"},
	"kenji":   {"kenji.w@icloud.com", "Kenji", "Watanabe", "Kenji W."},
	"omar":    {"omar.haddad@gmail.com", "Omar", "Haddad", "Omar Haddad"},
	"spammer": {"hello@brandboost.example", "Brandboost", "Media", "Brandboost Media"},
}

// demoComment is one seeded comment: who (a commenter's key, or "team"),
// its parent's key, when (days before the design's today, at h:m New York
// time) and its status.
type demoComment struct {
	key, post, who, parent string
	day, h, m              int
	status, held           string
	pinned                 bool
	body                   string
}

var demoComments = []demoComment{
	// What shipped in October, and what we're retiring (9 Oct).
	{key: "sam", post: "what-shipped-in-october", who: "sam", day: 6, h: 10, m: 42, status: "published", pinned: true,
		body: "The v1 retirement date is what I needed. We still call /v1/flags from two cron jobs. Is there a way to see which of our keys called v1 in the last 30 days, so we know what's left to move?"},
	{key: "team", post: "what-shipped-in-october", who: "team", parent: "sam", day: 6, h: 11, m: 20, status: "published",
		body: "Yes: Settings › API keys has a “v1 calls, last 30 days” column, and each key's page lists the endpoints it called. We'll also email every account that still calls v1 on 2, 16 and 27 November."},
	{key: "priya1", post: "what-shipped-in-october", who: "priya", parent: "sam", day: 6, h: 12, m: 5, status: "published",
		body: "That column just saved us an afternoon. Two keys left, both from an old staging setup."},
	{key: "ben", post: "what-shipped-in-october", who: "ben", day: 6, h: 10, m: 58, status: "published",
		body: "Feature flags going GA is great news. Do flags created during the beta keep their history, or do they start fresh?"},
	{key: "mateo", post: "what-shipped-in-october", who: "mateo", parent: "ben", day: 6, h: 13, m: 31, status: "published",
		body: "Ours kept everything, including the rollout history from August. Only the Beta label went away."},
	{key: "grace", post: "what-shipped-in-october", who: "grace", day: 6, h: 14, m: 10, status: "published",
		body: "Streaming audit log exports to our SIEM was the last item on our security review. What format does it send, and how far behind does it run?"},
	{key: "olga", post: "what-shipped-in-october", who: "olga", parent: "grace", day: 6, h: 15, m: 2, status: "published",
		body: "We send ours to Splunk over HTTPS. It's JSON lines, and events arrive within about a minute."},
	{key: "diego", post: "what-shipped-in-october", who: "diego", day: 5, h: 8, m: 47, status: "published",
		body: "Thanks for seven weeks' notice on v1 instead of the usual two. That's enough to fit the move into a normal sprint."},
	{key: "lena", post: "what-shipped-in-october", who: "lena", day: 5, h: 16, m: 23, status: "published",
		body: "The Q4 roadmap mentions EU data residency. Is that this quarter or next?"},
	{key: "hannah1", post: "what-shipped-in-october", who: "hannah", parent: "lena", day: 5, h: 18, m: 40, status: "published",
		body: "They answered this on Tuesday's community call: the EU region comes first, in December."},
	{key: "spam", post: "what-shipped-in-october", who: "spammer", day: 4, h: 22, m: 14, status: "hidden",
		body: "Grow your newsletter to 50,000 subscribers in a week. Message us for our price list."},
	{key: "tomasz", post: "what-shipped-in-october", who: "tomasz", day: 3, h: 9, m: 15, status: "published",
		body: "Could the S3 export write one file per day instead of one per hour? Our retention rules work by day."},
	{key: "hannah2", post: "what-shipped-in-october", who: "hannah", day: 2, h: 11, m: 2, status: "published",
		body: "Flag cleanup reminders are the feature I didn't know I wanted. We had 300 flags nobody had touched since spring."},
	{key: "priya2", post: "what-shipped-in-october", who: "priya", day: 1, h: 17, m: 36, status: "published",
		body: "Do the v1 webhooks stop on 30 November too, or only the REST endpoints?"},
	{key: "kenji", post: "what-shipped-in-october", who: "kenji", day: 0, h: 8, m: 47, status: "pending", held: "links",
		body: "We wrote up our move from v1 to v2, with the scripts: https://kenji.dev/acme-v2-migration, and the checklist we used is at https://github.com/kwatanabe/acme-v2-checklist. Happy to answer questions here."},
	{key: "alex", post: "what-shipped-in-october", who: "alex", day: 0, h: 9, m: 21, status: "pending", held: "first_comment",
		body: "Is there a v1 to v2 migration guide for the Python SDK? The changelog only links to the REST docs."},
	// Ask Acme: your deliverability questions, answered (10 Oct, members).
	{key: "ask-grace", post: "ask-acme-deliverability-questions", who: "grace", day: 5, h: 10, m: 12, status: "published",
		body: "The Promotions answer matches what we saw: moving the logo made no difference, cutting the images from eight to two did."},
	{key: "ask-ben", post: "ask-acme-deliverability-questions", who: "ben", parent: "ask-grace", day: 5, h: 11, m: 30, status: "published",
		body: "Same here. Plain text with one image landed in Primary for about 70% of our Gmail test accounts."},
	{key: "ask-mateo", post: "ask-acme-deliverability-questions", who: "mateo", day: 5, h: 13, m: 5, status: "published",
		body: "On warming up a new IP: does the advice change if we only send 20,000 a week? Four weeks of ramp feels slow at that volume."},
	{key: "ask-olga", post: "ask-acme-deliverability-questions", who: "olga", parent: "ask-mateo", day: 5, h: 15, m: 48, status: "published",
		body: "At that volume we stayed on the shared pool and skipped the dedicated IP. Its reputation was better than a cold IP's."},
	{key: "ask-sam", post: "ask-acme-deliverability-questions", who: "sam", day: 4, h: 9, m: 30, status: "published",
		body: "The DMARC section finally explained the Google servers in our reports: they're calendar invites. Two years of wondering."},
	{key: "ask-lena", post: "ask-acme-deliverability-questions", who: "lena", day: 2, h: 14, m: 20, status: "published",
		body: "Could the next Ask Acme cover BIMI? We have the certificate and Gmail still shows our initials."},
	{key: "ask-omar", post: "ask-acme-deliverability-questions", who: "omar", day: 0, h: 10, m: 2, status: "pending", held: "first_comment",
		body: "Does a p=reject DMARC policy affect forwarded mail? Our support inbox forwards to a shared Gmail account and some messages never arrive."},
}

func seedWebsiteCommentsImpl(ctx context.Context, s *Seeder) error {
	ws := s.WS()
	sofia := s.Users["sofia@acme.io"].ID
	if _, err := s.Tx.Exec(ctx, `UPDATE website_settings SET comments = 'members', comments_review = 'first', comments_notified_at = $2 WHERE workspace_id = $1`,
		ws.ID, s.designDay(0, 8, 50)); err != nil {
		return err
	}
	posts := map[string]uuid.UUID{}
	for _, slug := range []string{"what-shipped-in-october", "ask-acme-deliverability-questions"} {
		var id uuid.UUID
		if err := s.Tx.QueryRow(ctx, `SELECT id FROM website_posts WHERE workspace_id = $1 AND slug = $2 AND deleted_at IS NULL`, ws.ID, slug).Scan(&id); err != nil {
			return fmt.Errorf("the post %s: %w", slug, err)
		}
		posts[slug] = id
	}
	// The commenters are contacts who signed in to the site, with the name
	// their comments show.
	contacts := map[string]uuid.UUID{}
	for key, c := range demoCommenters {
		id, ok := s.contactByEmail(ctx, c.email)
		if !ok {
			// A commenter the other seeders didn't make takes the place of a
			// generated Weekly Digest subscriber with the same mailbox
			// provider (as Earn's paid members do), so every audience total
			// stays as the boards count it.
			var err error
			if id, err = s.standIn(ctx, c.email, c.first, c.last); err != nil {
				return fmt.Errorf("commenter %s: %w", c.email, err)
			}
		}
		contacts[key] = id
		if _, err := s.Tx.Exec(ctx, `INSERT INTO website_members (contact_id, workspace_id, first_signed_in, last_signed_in, last_method, sign_ins, display_name)
			VALUES ($1,$2,$3,$3,'link',1,$4) ON CONFLICT (contact_id) DO UPDATE SET display_name = EXCLUDED.display_name`,
			id, ws.ID, s.designDay(8, 9, 0), c.display); err != nil {
			return err
		}
	}
	ids := map[string]uuid.UUID{}
	for _, c := range demoComments {
		id := ID()
		ids[c.key] = id
		at := s.designDay(c.day, c.h, c.m)
		var parent, contact, user, moderator *uuid.UUID
		if c.parent != "" {
			p := ids[c.parent]
			parent = &p
		}
		team := c.who == "team"
		if team {
			user = &sofia
		} else {
			ct := contacts[c.who]
			contact = &ct
		}
		var published, pinned, moderated *time.Time
		switch c.status {
		case "published":
			published = &at
		case "hidden":
			mod := s.designDay(3, 8, 5)
			moderated, moderator = &mod, &sofia
		}
		if c.pinned {
			p := at.Add(4 * time.Hour)
			pinned = &p
		}
		if c.held != "" && c.status == "published" {
			// Published by a moderator after it waited.
			moderator = &sofia
		}
		if _, err := s.Tx.Exec(ctx, `INSERT INTO website_comments (id, workspace_id, post_id, parent_id, contact_id, team, user_id, body, status, held_reason,
			pinned_at, created_at, published_at, moderated_by, moderated_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15)`,
			id, ws.ID, posts[c.post], parent, contact, team, user, strings.TrimSpace(c.body), c.status, c.held, pinned, at, published, moderator, moderated); err != nil {
			return fmt.Errorf("comment %s: %w", c.key, err)
		}
	}
	// The pin, the team reply and the hide are in the audit log.
	for _, a := range []struct {
		action, key string
		at          time.Time
		after       map[string]any
	}{
		{"comment.pinned", "sam", s.designDay(6, 14, 42), map[string]any{"pinned": true}},
		{"comment.team_replied", "team", s.designDay(6, 11, 20), map[string]any{"status": "published"}},
		{"comment.hidden", "spam", s.designDay(3, 8, 5), map[string]any{"status": "hidden"}},
	} {
		after, _ := json.Marshal(a.after)
		if _, err := s.Tx.Exec(ctx, `INSERT INTO audit_events (id, workspace_id, occurred_at, actor_type, actor_id, actor_name, action, target_type, target_id,
			target_name, result, after) VALUES ($1,$2,$7,'user',$3,'Sofia Rossi',$4,'website_comment',$5,'What shipped in October, and what we''re retiring','allowed',$6)`,
			ID(), ws.ID, sofia, a.action, ids[a.key].String(), after, a.at); err != nil {
			return err
		}
	}
	return nil
}

// standIn turns the newest generated Weekly Digest subscriber with the same
// mailbox provider into a named reader, and returns its id.
func (s *Seeder) standIn(ctx context.Context, email, first, last string) (uuid.UUID, error) {
	ws := s.WS()
	var id uuid.UUID
	err := s.Tx.QueryRow(ctx, `SELECT c.id FROM contacts c
		JOIN list_memberships m ON m.contact_id = c.id AND m.list_id = $2 AND m.status = 'subscribed'
		WHERE c.workspace_id = $1 AND c.status = 'active' AND c.erased_at IS NULL AND c.first_name <> '' AND c.mailbox_provider = $3
		  AND NOT EXISTS (SELECT 1 FROM website_members wm WHERE wm.contact_id = c.id)
		  AND NOT EXISTS (SELECT 1 FROM paid_subscriptions ps WHERE ps.contact_id = c.id)
		ORDER BY c.created_at DESC, c.id DESC LIMIT 1`, ws.ID, s.Lists["Weekly Digest"], emailaddr.MailboxProvider(emailaddr.Domain(email))).Scan(&id)
	if err != nil {
		return uuid.Nil, err
	}
	_, err = s.Tx.Exec(ctx, `UPDATE contacts SET email = $2, first_name = $3, last_name = $4, role_address = $5 WHERE id = $1`,
		id, email, first, last, emailaddr.IsRole(email))
	return id, err
}
