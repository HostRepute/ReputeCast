package texting

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/phone"
	"github.com/hostrepute/reputecast/internal/sms"
	"github.com/hostrepute/reputecast/internal/store"
	"github.com/hostrepute/reputecast/internal/webhooks"
)

// Inbound texts (brief §3.4). A keyword matches when it's the whole reply,
// in any case:
//   - opt-out keywords withdraw SMS consent on every list, for every contact
//     with the number, put the number on the SMS suppression list as
//     "Opted out by text" and send one confirmation reply;
//   - opt-in keywords lift that suppression and bring back exactly the
//     consents the STOP removed, each recorded as "Opted back in by text";
//   - help keywords are answered with the help reply;
//   - YES confirms pending consent when the workspace asks for it;
//   - anything else is a reply: on the contact's timeline and in the report.
//
// When the provider handles keywords itself (rules: keywords answered by
// the provider), ReputeCast applies the same changes and sends no reply of
// its own. Keyword changes are automatic and high volume: they're kept in
// the consent evidence and the inbound log, not the audit log.

// inboundEffect is what ReputeCast did with an inbound text, as the Events
// tab and the timeline word it.
type inboundEffect struct {
	Lists    []string `json:"lists,omitempty"`
	Contacts int      `json:"contacts"`
	// Replied is true when ReputeCast sent the reply; ProviderAnswered when
	// the provider did.
	Replied          bool   `json:"replied"`
	ProviderAnswered bool   `json:"providerAnswered,omitempty"`
	Suppressed       bool   `json:"suppressed,omitempty"`
	Lifted           bool   `json:"lifted,omitempty"`
	ReplyError       string `json:"replyError,omitempty"`
	Unknown          bool   `json:"unknown,omitempty"`
}

// applyInbound acts on a text someone sent to one of the workspace's numbers.
func (s *Service) applyInbound(ctx context.Context, ws *store.Workspace, p store.SmsProvider, rawID uuid.UUID, e sms.Event) error {
	now := s.now()
	_ = s.Q.SmsProviderTouchInbound(ctx, store.SmsProviderTouchInboundParams{At: now, ID: p.ID})
	from := e.From
	if n, err := phone.Parse(e.From, ""); err == nil {
		from = n.E164
	}
	r, err := s.rules(ctx, s.Q, ws)
	if err != nil {
		return err
	}
	kind, word := r.Keywords.Match(e.Body, r.RequireYes)
	providerAnswered := false
	if e.OptOut != "" {
		// The provider acted on a keyword itself (Twilio's Advanced Opt-Out).
		providerAnswered = true
		switch strings.ToUpper(e.OptOut) {
		case "STOP":
			kind = sms.KeywordOptOut
		case "START":
			kind = sms.KeywordOptIn
		case "HELP":
			kind = sms.KeywordHelp
		}
		if word == "" {
			word = strings.ToUpper(e.OptOut)
		}
	}
	contacts, err := s.Q.ChanContactsByPhone(ctx, store.ChanContactsByPhoneParams{WorkspaceID: ws.ID, Phone: from})
	if err != nil {
		return err
	}
	ids := make([]uuid.UUID, 0, len(contacts))
	for _, c := range contacts {
		ids = append(ids, c.ID)
	}
	var last *store.ChannelMessage
	if m, err := s.Q.ChanLastTextTo(ctx, store.ChanLastTextToParams{WorkspaceID: ws.ID, Recipient: from}); err == nil {
		last = &m
	}
	typed := strings.TrimSpace(e.Body)
	if len([]rune(typed)) > 20 {
		typed = string([]rune(typed)[:20])
	}
	if typed == "" {
		typed = word
	}
	eff := inboundEffect{Contacts: len(ids), ProviderAnswered: providerAnswered, Unknown: len(ids) == 0}
	reply := ""
	err = s.tx(ctx, func(q *store.Queries, tx pgx.Tx) error {
		var err error
		switch kind {
		case sms.KeywordOptOut:
			eff.Lists, err = s.optOutNumber(ctx, q, tx, ws, p, from, typed, ids, last, now)
			eff.Suppressed = true
			reply = r.OptOutReply
		case sms.KeywordOptIn:
			eff.Lists, eff.Lifted, err = s.optInNumber(ctx, q, tx, ws, from, word, ids, now)
			reply = r.OptInReply
		case sms.KeywordHelp:
			reply = r.HelpReply
		case sms.KeywordYes:
			eff.Lists, err = s.confirmNumber(ctx, q, tx, ws, from, ids, now)
		}
		return err
	})
	if err != nil {
		return err
	}
	var replyID *uuid.UUID
	if reply != "" && r.KeywordsBy != "provider" && !providerAnswered {
		m, err := s.reply(ctx, ws, p, from, e.To, reply)
		if err != nil {
			eff.ReplyError = err.Error()
		} else {
			replyID, eff.Replied = &m.ID, true
		}
	}
	in := store.SmsInboundInsertParams{ID: uuid.Must(uuid.NewV7()), WorkspaceID: ws.ID, ProviderID: &p.ID, Phone: from, Sender: e.To, Body: e.Body, Kind: kind,
		Keyword: word, ContactIds: ids, Effect: jsonOf(eff), ReplyMessageID: replyID, RawEventID: &rawID, ReceivedAt: e.OccurredAt}
	if last != nil {
		in.MessageID, in.CampaignID, in.AutomationID = &last.ID, last.CampaignID, last.AutomationID
	}
	err = s.tx(ctx, func(q *store.Queries, tx pgx.Tx) error {
		if err := q.SmsInboundInsert(ctx, in); err != nil {
			return err
		}
		evKind, hook := "", ""
		data := map[string]any{"phone": from, "keyword": word, "lists": eff.Lists, "received_at": e.OccurredAt}
		switch kind {
		case sms.KeywordOptOut:
			evKind, hook = "opted_out", "sms.opted_out"
		case sms.KeywordOptIn:
			evKind, hook = "opted_in", "sms.opted_in"
		case sms.KeywordHelp:
			evKind = "help"
		case sms.KeywordReply:
			evKind, hook = "replied", "sms.replied"
			data["body"] = e.Body
		}
		if last != nil && evKind != "" {
			if err := q.ChanMessageEventInsert(ctx, store.ChanMessageEventInsertParams{ID: uuid.Must(uuid.NewV7()), WorkspaceID: ws.ID, MessageID: last.ID, Kind: evKind,
				Detail: jsonOf(map[string]any{"keyword": word, "inbound": in.ID.String()}), RawEventID: &rawID, OccurredAt: e.OccurredAt}); err != nil {
				return err
			}
		}
		if hook != "" {
			if last != nil {
				data = s.hookData(*last, data)
			}
			if len(ids) > 0 {
				cids := make([]string, 0, len(ids))
				for _, id := range ids {
					cids = append(cids, id.String())
				}
				data["contact_ids"] = cids
			}
			if _, err := webhooks.Emit(ctx, tx, ws.ID, ws.Slug, hook, data); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return err
	}
	if kind == sms.KeywordOptOut {
		s.syncAdd(ctx, ws, from)
		if last != nil && last.CampaignID != nil && last.Kind == "campaign" {
			s.checkAutoPause(ctx, ws, *last.CampaignID)
		}
	}
	if kind == sms.KeywordOptIn && eff.Lifted {
		s.syncRemove(ctx, ws, from)
	}
	return nil
}

// optOutNumber withdraws SMS consent on every list for every contact with
// the number and suppresses it.
func (s *Service) optOutNumber(ctx context.Context, q *store.Queries, tx pgx.Tx, ws *store.Workspace, p store.SmsProvider, ph, typed string,
	contacts []uuid.UUID, last *store.ChannelMessage, now time.Time) ([]string, error) {
	lists := []string{}
	for _, cid := range contacts {
		rows, err := tx.Query(ctx, `UPDATE channel_consents cc SET status = 'opted_out', source = 'keyword', keyword = $3, stopped_at = $4, changed_at = $4
			FROM lists l WHERE l.id = cc.list_id AND cc.contact_id = $1 AND cc.workspace_id = $2 AND cc.channel = 'sms' AND cc.status IN ('subscribed','pending')
			RETURNING cc.list_id, l.name`, cid, ws.ID, strings.ToUpper(typed), now)
		if err != nil {
			return nil, err
		}
		type ln struct {
			id   uuid.UUID
			name string
		}
		changed := []ln{}
		for rows.Next() {
			var x ln
			if err := rows.Scan(&x.id, &x.name); err != nil {
				rows.Close()
				return nil, err
			}
			changed = append(changed, x)
		}
		rows.Close()
		for _, x := range changed {
			if !contains(lists, x.name) {
				lists = append(lists, x.name)
			}
			if err := q.ChanConsentEventInsert(ctx, store.ChanConsentEventInsertParams{ID: uuid.Must(uuid.NewV7()), WorkspaceID: ws.ID, ContactID: cid, ListID: &x.id,
				Channel: "sms", Action: "opted_out", Phone: ph, Source: "keyword", Page: "Text message", Keyword: strings.ToUpper(typed),
				Note: "Opted out by text", OccurredAt: now}); err != nil {
				return nil, err
			}
		}
	}
	// A number suppressed for another reason now carries the opt-out.
	if _, err := tx.Exec(ctx, `UPDATE channel_suppressions SET removed_at = $3, removed_reason = 'Opted out by text'
		WHERE workspace_id = $1 AND channel = 'sms' AND phone = $2 AND removed_at IS NULL AND reason <> 'opted_out'`, ws.ID, ph, now); err != nil {
		return nil, err
	}
	sup := store.ChanSuppressionInsertParams{ID: uuid.Must(uuid.NewV7()), WorkspaceID: ws.ID, Channel: "sms", Phone: ph, Reason: "opted_out",
		Keyword: strings.ToUpper(typed), OriginalCode: typed, Source: p.Name, SourceProviderID: &p.ID, CreatedAt: now}
	if last != nil {
		sup.CampaignID, sup.AutomationID, sup.Context = last.CampaignID, last.AutomationID, contextOf(last.Kind)
	}
	return lists, q.ChanSuppressionInsert(ctx, sup)
}

// optInNumber lifts a text opt-out and brings back the consents it removed.
func (s *Service) optInNumber(ctx context.Context, q *store.Queries, tx pgx.Tx, ws *store.Workspace, ph, word string, contacts []uuid.UUID,
	now time.Time) ([]string, bool, error) {
	lists := []string{}
	for _, cid := range contacts {
		rows, err := tx.Query(ctx, `UPDATE channel_consents cc SET status = 'subscribed', source = 'keyword', keyword = $3, stopped_at = NULL, changed_at = $4
			FROM lists l WHERE l.id = cc.list_id AND l.deleted_at IS NULL AND cc.contact_id = $1 AND cc.workspace_id = $2 AND cc.channel = 'sms'
			AND cc.status = 'opted_out' AND cc.stopped_at IS NOT NULL
			RETURNING cc.list_id, l.name`, cid, ws.ID, word, now)
		if err != nil {
			return nil, false, err
		}
		type ln struct {
			id   uuid.UUID
			name string
		}
		changed := []ln{}
		for rows.Next() {
			var x ln
			if err := rows.Scan(&x.id, &x.name); err != nil {
				rows.Close()
				return nil, false, err
			}
			changed = append(changed, x)
		}
		rows.Close()
		for _, x := range changed {
			if !contains(lists, x.name) {
				lists = append(lists, x.name)
			}
			if err := q.ChanConsentEventInsert(ctx, store.ChanConsentEventInsertParams{ID: uuid.Must(uuid.NewV7()), WorkspaceID: ws.ID, ContactID: cid, ListID: &x.id,
				Channel: "sms", Action: "opted_in", Phone: ph, Source: "keyword", Page: "Text message", Keyword: word, Note: "Opted back in by text",
				OccurredAt: now}); err != nil {
				return nil, false, err
			}
		}
	}
	tag, err := tx.Exec(ctx, `UPDATE channel_suppressions SET removed_at = $3, removed_reason = 'Opted back in by text'
		WHERE workspace_id = $1 AND channel = 'sms' AND phone = $2 AND removed_at IS NULL AND reason IN ('opted_out','provider_list')`, ws.ID, ph, now)
	if err != nil {
		return nil, false, err
	}
	return lists, tag.RowsAffected() > 0, nil
}

// confirmNumber turns pending SMS consent into consent after a YES.
func (s *Service) confirmNumber(ctx context.Context, q *store.Queries, tx pgx.Tx, ws *store.Workspace, ph string, contacts []uuid.UUID,
	now time.Time) ([]string, error) {
	lists := []string{}
	for _, cid := range contacts {
		rows, err := tx.Query(ctx, `UPDATE channel_consents cc SET status = 'subscribed', changed_at = $3
			FROM lists l WHERE l.id = cc.list_id AND cc.contact_id = $1 AND cc.workspace_id = $2 AND cc.channel = 'sms' AND cc.status = 'pending'
			RETURNING cc.list_id, l.name`, cid, ws.ID, now)
		if err != nil {
			return nil, err
		}
		type ln struct {
			id   uuid.UUID
			name string
		}
		changed := []ln{}
		for rows.Next() {
			var x ln
			if err := rows.Scan(&x.id, &x.name); err != nil {
				rows.Close()
				return nil, err
			}
			changed = append(changed, x)
		}
		rows.Close()
		for _, x := range changed {
			if !contains(lists, x.name) {
				lists = append(lists, x.name)
			}
			if err := q.ChanConsentEventInsert(ctx, store.ChanConsentEventInsertParams{ID: uuid.Must(uuid.NewV7()), WorkspaceID: ws.ID, ContactID: cid, ListID: &x.id,
				Channel: "sms", Action: "confirmed", Phone: ph, Source: "keyword", Page: "Text message", Keyword: "YES", Note: "Confirmed by replying YES",
				OccurredAt: now}); err != nil {
				return nil, err
			}
		}
	}
	return lists, nil
}

// reply answers a keyword at once, from the number that was texted. Replies
// to keywords aren't held by quiet hours: carriers expect them right away.
func (s *Service) reply(ctx context.Context, ws *store.Workspace, p store.SmsProvider, to, via, text string) (store.ChannelMessage, error) {
	g, err := s.graph(ctx, s.Q, ws)
	if err != nil {
		return store.ChannelMessage{}, err
	}
	var sender *store.SmsSendersRow
	for _, id := range g.order {
		sn := g.senders[id]
		if sn.ProviderID == p.ID && sn.Value == via {
			sender = &sn
			break
		}
	}
	return s.sendOne(ctx, ws, g, oneText{kind: "reply", recipient: Recipient{Phone: to}, text: Text{Body: text}, sender: sender})
}
