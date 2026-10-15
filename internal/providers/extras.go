package providers

import (
	"context"
	"net/mail"
)

// Attachment is a file sent with a message (a PDF invoice). ContentID makes
// it an inline part that the HTML references as cid:<ContentID>.
type Attachment struct {
	Filename    string
	ContentType string
	ContentID   string
	Content     []byte
}

// MessageExtras is what a transactional message can carry beyond the base
// Message: copies and attachments.
type MessageExtras struct {
	Cc          []mail.Address
	Bcc         []mail.Address
	Attachments []Attachment
}

// Empty reports whether the extras add nothing.
func (e MessageExtras) Empty() bool {
	return len(e.Cc) == 0 && len(e.Bcc) == 0 && len(e.Attachments) == 0
}

// ExtrasSender is implemented by adapters that can send copies and
// attachments (an optional interface: core code type-asserts before using
// it, and a message with extras only goes to a provider that has it).
// extras[i] belongs to msgs[i].
type ExtrasSender interface {
	SendWithExtras(ctx context.Context, msgs []Message, extras []MessageExtras) ([]SendResult, error)
}

// SendWithExtras sends msgs through p, with their extras when p supports
// them. It refuses (per message, permanently) a message with extras that p
// can't carry, rather than dropping its attachments.
func SendWithExtras(ctx context.Context, p Provider, msgs []Message, extras []MessageExtras) ([]SendResult, error) {
	has := false
	for _, e := range extras {
		if !e.Empty() {
			has = true
			break
		}
	}
	if !has {
		return p.Send(ctx, msgs)
	}
	if es, ok := p.(ExtrasSender); ok {
		return es.SendWithExtras(ctx, msgs, extras)
	}
	plain := make([]Message, 0, len(msgs))
	idx := make([]int, 0, len(msgs))
	out := make([]SendResult, len(msgs))
	for i, m := range msgs {
		if i < len(extras) && !extras[i].Empty() {
			out[i] = SendResult{MessageID: m.ID, Err: &SendError{Code: "Unsupported", Message: "this provider's adapter can't send copies or attachments"}}
			continue
		}
		plain = append(plain, m)
		idx = append(idx, i)
	}
	if len(plain) == 0 {
		return out, nil
	}
	res, err := p.Send(ctx, plain)
	if err != nil {
		return nil, err
	}
	for j, r := range res {
		if j < len(idx) {
			out[idx[j]] = r
		}
	}
	return out, nil
}
