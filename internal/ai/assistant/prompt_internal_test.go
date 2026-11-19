package assistant

import (
	"os"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/ai/llm"
	"github.com/hostrepute/reputecast/internal/store"
)

// Every screen the assistant describes or links to is a route of the web
// app.
func TestScreensAreRoutes(t *testing.T) {
	raw, err := os.ReadFile("../../../web/src/routeTree.gen.ts")
	if err != nil {
		t.Fatal(err)
	}
	for _, s := range screens {
		if !strings.Contains(string(raw), "fullPath: '"+s.Route+"'") {
			t.Errorf("screen %s: %s isn't a route of the web app", s.Key, s.Route)
		}
		if s.Param != "" && !strings.Contains(s.Route, "$"+s.Param) {
			t.Errorf("screen %s: its route has no $%s", s.Key, s.Param)
		}
	}
	if got := searchHelp("How do I set up DKIM?"); len(got) == 0 || got[0].ID != "sending-domains" {
		t.Fatalf("searching help for DKIM: %+v", got)
	}
}

// History goes back to the model that wrote it as it was; to another model
// without the provider's own parts (Gemini's thoughts and signatures), and
// tool calls never without their results.
func TestReplayStripsThoughtsForAnotherModel(t *testing.T) {
	writer, other := uuid.New(), uuid.New()
	m := store.AiAssistantMessage{Role: "assistant", ModelID: &writer}
	c := messageContent{Text: "Answer", Replay: &llm.Message{Role: "assistant", Parts: []llm.Part{
		{Text: "thinking", Thought: true}, {Text: "Answer", ThoughtSignature: "sig"}, {ToolCall: &llm.ToolCall{ID: "1", Name: "search_help"}}}}}
	s := &Service{}
	same, _ := s.replay(m, c, "tool", writer)
	if len(same.Parts) != 3 || same.Parts[1].ThoughtSignature != "sig" {
		t.Fatalf("to the same model: %+v", same.Parts)
	}
	another, _ := s.replay(m, c, "tool", other)
	if len(another.Parts) != 2 || another.Parts[0].Thought || another.Parts[0].ThoughtSignature != "" || another.Parts[1].ToolCall == nil {
		t.Fatalf("to another model: %+v", another.Parts)
	}
	unanswered, _ := s.replay(m, c, "user", writer)
	if len(unanswered.Parts) != 2 || unanswered.Parts[1].ToolCall != nil {
		t.Fatalf("calls without their results: %+v", unanswered.Parts)
	}
	if llm.ContextTokens(llm.Anthropic, "claude-haiku-4-5-20251001") != 200_000 || llm.ContextTokens(llm.Compatible, "llama3.3:70b") != 32_768 {
		t.Fatal("context windows")
	}
}
