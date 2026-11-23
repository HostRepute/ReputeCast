package assistant

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/ai/llm"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/store"
)

// Answers saved before they kept their sources count everything the
// thread's tools read before them (fail-closed): one from before the first
// read stays, one after it is withheld once that read's source can't be
// read. An answer that kept its sources is judged by them alone; a summary
// is made without a withheld answer's text.
func TestHistoryOfLegacyAnswers(t *testing.T) {
	campaign := toolSource{Kind: "campaign", ID: uuid.Must(uuid.NewV7()), Permission: authz.CampaignsView}
	turn := uuid.Must(uuid.NewV7())
	msg := func(seq int32, role string) store.AiAssistantMessage {
		return store.AiAssistantMessage{ID: uuid.Must(uuid.NewV7()), Role: role, Sequence: seq, TurnID: &turn}
	}
	result, _ := json.Marshal(map[string]any{"subject": "SECRET-SUBJECT"})
	read := llm.Part{ToolResult: &llm.ToolResult{ID: "c1", Name: "read_attached", Content: result}}
	src := campaign
	src.CallID = "c1"
	messages := []store.AiAssistantMessage{msg(1, "user"), msg(2, "assistant"), msg(3, "tool"), msg(4, "assistant"), msg(5, "assistant"), msg(6, "user")}
	contents := []messageContent{
		{Text: "Hi"},
		{Text: "Hello before any read."},
		{Results: []llm.Part{read}, Tools: []AIThreadToolRow{{CallID: "c1", Name: "read_attached", OK: true, Label: "Spring sale"}}, Sources: []toolSource{src}},
		{Text: "It says SECRET-SUBJECT."},
		{Text: "A later answer that read nothing.", SourcesKnown: true},
		{Text: "And now?"},
	}
	key := campaign
	r := &readable{owner: &authz.Principal{Type: authz.ActorUser}, known: map[toolSource]bool{key: false}}
	h, err := r.history(context.Background(), messages, contents)
	if err != nil {
		t.Fatal(err)
	}
	if h.held[1] || !h.held[3] || h.held[4] || h.readable || !h.calls[2]["c1"] {
		t.Fatalf("held %v, calls %v, readable %v", h.held, h.calls, h.readable)
	}
	if v := h.view(3, messages[3], contents[3]); !v.Withheld || v.Text != "" {
		t.Fatalf("the legacy answer's view: %+v", v)
	}
	if v := h.view(2, messages[2], contents[2]); v.Tools[0].Label != "" || v.Tools[0].Code != "no_longer_available" {
		t.Fatalf("the read's row: %+v", v.Tools)
	}
	req, used := summaryRequest("", messages, contents, []int{0, 1, 2, 3, 4}, h)
	body, _ := json.Marshal(req)
	if strings.Contains(string(body), "SECRET-SUBJECT") || !strings.Contains(string(body), "Hello before any read.") || len(used) != 0 {
		t.Fatalf("the summary's input: %s (from %v)", body, used)
	}
	// A result saved before results kept their source can't be read: the
	// legacy answers after it are withheld too.
	contents[2].Sources = nil
	r = &readable{owner: &authz.Principal{Type: authz.ActorUser}, known: map[toolSource]bool{}}
	if h, err = r.history(context.Background(), messages, contents); err != nil {
		t.Fatal(err)
	}
	if h.held[1] || !h.held[3] || h.held[4] || h.readable || !h.calls[2]["c1"] {
		t.Fatalf("without the result's source: held %v, calls %v, readable %v", h.held, h.calls, h.readable)
	}
}

// A summary made from an answer carries what the answer carried: the
// title of an attached campaign it repeated.
func TestSummaryCarriesItsAnswersSources(t *testing.T) {
	campaign := toolSource{Kind: "campaign", ID: uuid.Must(uuid.NewV7()), Permission: authz.CampaignsView}
	turn := uuid.Must(uuid.NewV7())
	msg := func(seq int32, role string) store.AiAssistantMessage {
		return store.AiAssistantMessage{ID: uuid.Must(uuid.NewV7()), Role: role, Sequence: seq, TurnID: &turn}
	}
	messages := []store.AiAssistantMessage{msg(1, "user"), msg(2, "assistant"), msg(3, "user"), msg(4, "assistant")}
	contents := []messageContent{
		{Text: "What did I attach?"},
		{Text: "You attached Spring sale.", Sources: []toolSource{campaign}, SourcesKnown: true},
		{Text: "Thanks"},
		{Text: "You're welcome.", SourcesKnown: true},
	}
	r := &readable{owner: &authz.Principal{Type: authz.ActorUser}, known: map[toolSource]bool{campaign: true}}
	h, err := r.history(context.Background(), messages, contents)
	if err != nil {
		t.Fatal(err)
	}
	if _, used := summaryRequest("", messages, contents, []int{0, 1, 2, 3}, h); len(used) != 1 || used[0] != campaign {
		t.Fatalf("the summary carries %v", used)
	}
}
