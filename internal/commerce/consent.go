package commerce

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/emailaddr"
	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/store"
	"github.com/hostrepute/reputecast/internal/stores"
)

// ConsentSourcePrefix starts the consent source of a store's records:
// "store:shopify:<store id>".
const ConsentSourcePrefix = "store:"

// ConsentSource is a store's consent records' source.
func ConsentSource(st store.Store) string {
	return ConsentSourcePrefix + st.Platform + ":" + st.ID.String()
}

// SourcePlatform reads the platform from a store consent source ("" when
// it isn't one).
func SourcePlatform(source string) string {
	rest, ok := strings.CutPrefix(source, ConsentSourcePrefix)
	if !ok {
		return ""
	}
	platform, _, _ := strings.Cut(rest, ":")
	return platform
}

// joined is what consent did for one customer.
type joined struct {
	contactID *uuid.UUID
	// created: the contact is new; joined: they joined the store's list now.
	created bool
	joined  bool
	// listName is the list they joined.
	listName string
}

// contactOf finds a live contact by email.
func (s *Service) contactOf(ctx context.Context, q *store.Queries, wsID uuid.UUID, email string) (store.CommerceContactByEmailRow, bool) {
	email = strings.ToLower(strings.TrimSpace(email))
	if email == "" {
		return store.CommerceContactByEmailRow{}, false
	}
	c, err := q.CommerceContactByEmail(ctx, store.CommerceContactByEmailParams{WorkspaceID: wsID, Email: email})
	if err != nil || c.ErasedAt != nil {
		return store.CommerceContactByEmailRow{}, false
	}
	return c, true
}

// consent applies a customer who accepted marketing: they become a contact
// (or stay the one they are) and join the store's list, with a consent
// record that says "Shopify: accepted marketing at checkout", its date and
// opt-in level. Nobody suppressed, unsubscribed from everything, or who
// left the store's list after this consent joins it again.
func (s *Service) consent(ctx context.Context, q *store.Queries, st store.Store, c stores.Customer, at time.Time) (joined, error) {
	out := joined{}
	email := strings.ToLower(strings.TrimSpace(c.Email))
	if email == "" || !strings.Contains(email, "@") {
		return out, nil
	}
	consentAt := at
	if c.Consent.UpdatedAt != nil && !c.Consent.UpdatedAt.IsZero() {
		consentAt = *c.Consent.UpdatedAt
	}
	contact, known := s.contactOf(ctx, q, st.WorkspaceID, email)
	reason, err := q.CommerceSuppressed(ctx, store.CommerceSuppressedParams{WorkspaceID: st.WorkspaceID, Email: email})
	if err != nil {
		return out, err
	}
	blocked := reason != "" || (known && contact.Status != "active")
	var contactID uuid.UUID
	switch {
	case known:
		contactID = contact.ID
	case blocked:
		// Suppressed and not a contact: nothing is kept.
		return out, nil
	default:
		evidence := mustJSON(map[string]any{"at": consentAt, "source": ConsentLabel(i18n.Default, st.Platform), "level": c.Consent.Level})
		id, err := q.CommerceContactInsert(ctx, store.CommerceContactInsertParams{ID: newID(), WorkspaceID: st.WorkspaceID, Email: email,
			FirstName: c.FirstName, LastName: c.LastName, Consent: "single", Source: ConsentSource(st), Country: c.Country,
			MailboxProvider: emailaddr.MailboxProvider(emailaddr.Domain(email)), ConsentEvidence: evidence, At: s.now()})
		if errors.Is(err, pgx.ErrNoRows) {
			// Someone else created them meanwhile.
			contact, known = s.contactOf(ctx, q, st.WorkspaceID, email)
			if !known {
				return out, nil
			}
			id = contact.ID
		} else if err != nil {
			return out, err
		} else {
			out.created = true
		}
		contactID = id
	}
	out.contactID = &contactID
	var joinedAt *time.Time
	if !blocked && st.ListID != nil {
		action := "subscribed"
		if prev, err := q.CommerceMembership(ctx, store.CommerceMembershipParams{ListID: *st.ListID, ContactID: contactID}); err == nil && prev.Status == "unsubscribed" {
			// They left the list before and gave consent again since.
			action = "resubscribed"
		}
		n, err := q.CommerceJoinList(ctx, store.CommerceJoinListParams{ListID: *st.ListID, ContactID: contactID, Source: ConsentSource(st),
			SubscribedAt: ptr(s.now()), At: s.now(), ConsentAt: consentAt})
		if err != nil {
			return out, err
		}
		if n > 0 {
			if err := q.CommerceConsentInsert(ctx, store.CommerceConsentInsertParams{ID: newID(), ContactID: contactID, ListID: st.ListID, Action: action,
				Source: ConsentSource(st), Detail: optInDetail(c.Consent.Level), Note: "", ConsentText: c.Consent.Text, At: consentAt}); err != nil {
				return out, err
			}
			out.joined = true
			joinedAt = ptr(s.now())
			if name, err := q.CommerceListName(ctx, *st.ListID); err == nil {
				out.listName = name
			}
		}
	}
	_, err = q.CommerceCustomerUpsert(ctx, store.CommerceCustomerUpsertParams{ID: newID(), StoreID: st.ID, WorkspaceID: st.WorkspaceID,
		ExternalID: customerKey(c, email), ContactID: &contactID, Email: email, FirstName: c.FirstName, LastName: c.LastName, Country: c.Country,
		Marketing: stores.ConsentSubscribed, OptInLevel: c.Consent.Level, ConsentAt: &consentAt, JoinedListAt: joinedAt,
		UpdatedAtStore: nonZero(c.UpdatedAt), At: s.now()})
	return out, err
}

// optInDetail is the opt-in level, as the consent record says it.
func optInDetail(level string) string {
	switch level {
	case "confirmed_opt_in":
		return "confirmed opt-in"
	case "single_opt_in":
		return "single opt-in"
	}
	return "opt-in level unknown"
}

// customerKey is the customer's id in the store, or their email for a guest.
func customerKey(c stores.Customer, email string) string {
	if c.ID != "" {
		return c.ID
	}
	return "guest:" + email
}

func nonZero(t time.Time) *time.Time {
	if t.IsZero() {
		return nil
	}
	return &t
}

// link keeps a customer who didn't accept marketing but is a contact: only
// the link (their orders show on the profile). Anyone else isn't kept.
func (s *Service) link(ctx context.Context, q *store.Queries, st store.Store, c stores.Customer, contactID uuid.UUID) error {
	state := c.Consent.State
	if state == "" {
		state = stores.ConsentNotSubscribed
	}
	_, err := q.CommerceCustomerUpsert(ctx, store.CommerceCustomerUpsertParams{ID: newID(), StoreID: st.ID, WorkspaceID: st.WorkspaceID,
		ExternalID: customerKey(c, strings.ToLower(c.Email)), ContactID: &contactID, Marketing: state, OptInLevel: "", UpdatedAtStore: nonZero(c.UpdatedAt),
		At: s.now()})
	return err
}

// leave takes a customer who unsubscribed in the store off its list.
func (s *Service) leave(ctx context.Context, q *store.Queries, st store.Store, contactID uuid.UUID, at time.Time) (bool, error) {
	if st.ListID == nil {
		return false, nil
	}
	n, err := q.CommerceLeaveList(ctx, store.CommerceLeaveListParams{ListID: *st.ListID, ContactID: contactID, At: at})
	if err != nil || n == 0 {
		return false, err
	}
	err = q.CommerceConsentInsert(ctx, store.CommerceConsentInsertParams{ID: newID(), ContactID: contactID, ListID: st.ListID, Action: "unsubscribed",
		Source: ConsentSource(st), At: at})
	return err == nil, err
}
