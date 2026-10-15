package transactional

import (
	"context"
	"encoding/json"
	"errors"
	"net/mail"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/providers"
	"github.com/hostrepute/reputecast/internal/routing"
)

// payload is a queued message as its provider will get it, with the
// sensitive values and the attachments. It is sealed with the instance key
// and deleted once a provider accepted the message (or every provider
// failed), so sensitive values are never kept.
type payload struct {
	From        mail.Address      `json:"from"`
	ReplyTo     *mail.Address     `json:"reply_to,omitempty"`
	To          mail.Address      `json:"to"`
	Cc          []mail.Address    `json:"cc,omitempty"`
	Bcc         []mail.Address    `json:"bcc,omitempty"`
	Subject     string            `json:"subject"`
	HTML        string            `json:"html"`
	Text        string            `json:"text"`
	Headers     map[string]string `json:"headers,omitempty"`
	Tags        map[string]string `json:"tags,omitempty"`
	Attachments []payloadFile     `json:"attachments,omitempty"`
}

type payloadFile struct {
	Filename    string `json:"filename"`
	ContentType string `json:"content_type"`
	ContentID   string `json:"content_id,omitempty"`
	Content     []byte `json:"content"`
}

func payloadAD(id uuid.UUID) string { return "transactional:" + id.String() + ":payload" }

func (s *Service) seal(id uuid.UUID, p payload) ([]byte, error) {
	if s.Box == nil {
		return nil, errors.New("no secret key to seal the message with")
	}
	b, err := json.Marshal(p)
	if err != nil {
		return nil, err
	}
	return s.Box.Seal(b, payloadAD(id))
}

// OpenPayload decodes a queued message for the sending engine's
// transactional outbox.
func (s *Service) OpenPayload(_ context.Context, id uuid.UUID, sealed []byte) (providers.Message, providers.MessageExtras, error) {
	if s.Box == nil {
		return providers.Message{}, providers.MessageExtras{}, errors.New("no secret key to open the message with")
	}
	b, err := s.Box.Open(sealed, payloadAD(id))
	if err != nil {
		return providers.Message{}, providers.MessageExtras{}, err
	}
	var p payload
	if err := json.Unmarshal(b, &p); err != nil {
		return providers.Message{}, providers.MessageExtras{}, err
	}
	m := providers.Message{ID: id, Stream: routing.StreamTransactional, From: p.From, ReplyTo: p.ReplyTo, To: p.To,
		Subject: p.Subject, HTML: p.HTML, Text: p.Text, Headers: p.Headers, Tags: p.Tags}
	x := providers.MessageExtras{Cc: p.Cc, Bcc: p.Bcc}
	for _, a := range p.Attachments {
		x.Attachments = append(x.Attachments, providers.Attachment{Filename: a.Filename, ContentType: a.ContentType, ContentID: a.ContentID, Content: a.Content})
	}
	return m, x, nil
}
