package app

import (
	"context"
	"regexp"
	"slices"
	"sort"
	"strings"
	"testing"

	"github.com/hostrepute/reputecast/internal/testdb"
)

// personalColumn names columns that hold, or point at, personal data about
// a contact: who they are, how to reach them, where they were, what they
// wrote. A table with one must be in an erasure source or exempted below.
var personalColumn = regexp.MustCompile(`^(contact_id|contact_ids|[a-z_]+_contact|email|[a-z_]+_email|emails|phone|[a-z_]+_phone|ip|[a-z_]+_ip|user_agent|` +
	`endpoint|p256dh|recipient|to_name|cc|bcc|first_name|last_name|buyer_name|buyer|recipient_name|billing_[a-z_]+|stripe_customer|stripe_customer_id|` +
	`referrer|landing_page|recovery_url|issued_to|address|comment)$`)

// erasureExempt are tables with personal-looking columns that erasing a
// contact leaves alone, and why.
var erasureExempt = map[string]string{
	// Suppression lists: kept so an erased person is never mailed or
	// texted again (review 2026-11, owner decision).
	"suppressions":                 "suppression list, kept on purpose",
	"channel_suppressions":         "SMS and WhatsApp suppression list, kept on purpose",
	"transactional_opt_outs":       "transactional opt-outs, kept on purpose",
	"provider_suppression_entries": "providers' suppression lists as last synced, kept on purpose",
	// Staff, their sign-ins and their notes: people of the team, not contacts.
	"users":                    "team members",
	"user_identities":          "team members' sign-in identities",
	"sessions":                 "team members' sessions",
	"sign_in_throttle":         "team sign-in throttling",
	"email_changes":            "team members' email changes",
	"invitations":              "team invitations",
	"agency_staff":             "agency staff",
	"notification_preferences": "a boolean (email notifications on), not an address",
	"approval_decisions":       "an approver's comment",
	"audit_events":             "the immutable audit log of what people and keys did",
	"api_keys":                 "where an API key was last used from",
	"api_key_rejections":       "refused API calls, a security log",
	// The workspace's own sending identity and test recipients.
	"senders":                 "the workspace's sender addresses",
	"placement_seeds":         "seed mailboxes for placement tests",
	"placement_copies":        "seed mailboxes for placement tests",
	"placement_tests":         "the test's sender address",
	"poll_test_recipients":    "staff test addresses for polls",
	"push_test_subscriptions": "staff browsers for push tests",
	"sms_test_numbers":        "staff numbers for SMS tests",
	"snds_daily":              "the workspace's sending IPs",
	"warmup_plans":            "the workspace's sending IPs",
	"workspace_sending_ips":   "the workspace's sending IPs",
	"hostrepute_connections":  "the HostRepute account that was connected",
	// Business settings, not people on the list.
	"earn_member_emails":   "a kind of member email, not an address",
	"earn_settings":        "the publisher's support address",
	"sponsors":             "a sponsor's business contact",
	"stores":               "the shop's web address",
	"wordpress_sites":      "the WordPress site's web address",
	"report_shares":        "a boolean (include the email), not an address",
	"referral_programs":    "a boolean (the same-IP rule), not an address",
	"recommendation_daily": "a count of recommendations shown in email",
	// Links to the scrubbed contact, for counts; nothing else personal.
	"form_submissions":         "form signup counts; no personal data beyond the scrubbed contact",
	"page_conversions":         "landing page conversion counts; the contact is scrubbed",
	"website_post_conversions": "web post conversion counts; the contact is scrubbed",
	// Sealed and short-lived.
	"partner_requests_out": "a sealed address sent to a partner, deleted after its retention (recommendations jobs)",
}

// TestErasureCoversPersonalData: every table with personal-data columns is
// covered by an erasure source (internal/app/wire_erasure.go) or exempted
// with a reason; every personal column of a covered table is erased or kept
// with a reason; no source names a table that doesn't exist.
func TestErasureCoversPersonalData(t *testing.T) {
	ctx := context.Background()
	pool := testdb.New(t)
	rows, err := pool.Query(ctx, `SELECT table_name, column_name FROM information_schema.columns
		WHERE table_schema = 'public' AND table_name NOT IN (SELECT table_name FROM information_schema.views WHERE table_schema = 'public')
		ORDER BY table_name, ordinal_position`)
	if err != nil {
		t.Fatal(err)
	}
	personal := map[string][]string{}
	tables := map[string]bool{}
	for rows.Next() {
		var tbl, col string
		if err := rows.Scan(&tbl, &col); err != nil {
			t.Fatal(err)
		}
		tables[tbl] = true
		if personalColumn.MatchString(col) {
			personal[tbl] = append(personal[tbl], col)
		}
	}
	rows.Close()

	covered := map[string]bool{}
	for _, src := range erasureRegistry().Sources() {
		for _, tb := range src.Tables {
			if !tables[tb.Name] {
				t.Errorf("source %s names table %s, which doesn't exist", src.Name, tb.Name)
				continue
			}
			if covered[tb.Name] {
				t.Errorf("table %s is in two sources", tb.Name)
			}
			covered[tb.Name] = true
			if _, ok := erasureExempt[tb.Name]; ok {
				t.Errorf("table %s is both erased by %s and exempt", tb.Name, src.Name)
			}
			if tb.Action == "delete" {
				continue
			}
			for _, col := range personal[tb.Name] {
				if !slices.Contains(tb.Columns, col) && tb.Kept[col] == "" {
					t.Errorf("%s.%s looks personal: erase it in source %s or say in Kept why it stays", tb.Name, col, src.Name)
				}
			}
		}
	}
	var missing []string
	for tbl, cols := range personal {
		if covered[tbl] {
			continue
		}
		if _, ok := erasureExempt[tbl]; ok {
			continue
		}
		missing = append(missing, tbl+" ("+strings.Join(cols, ", ")+")")
	}
	sort.Strings(missing)
	for _, m := range missing {
		t.Errorf("table %s has personal-data columns but no erasure source covers it; add it to its package's Erasure() or to erasureExempt with a reason", m)
	}
	for tbl := range erasureExempt {
		if !tables[tbl] {
			t.Errorf("exempt table %s doesn't exist", tbl)
		}
	}
}
