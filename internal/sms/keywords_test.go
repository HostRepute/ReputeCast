package sms

import (
	"errors"
	"testing"
)

func TestKeywordMatch(t *testing.T) {
	k := DefaultKeywords
	cases := []struct {
		body       string
		yes        bool
		kind, word string
	}{
		{"STOP", false, KeywordOptOut, "STOP"},
		{"Stop", false, KeywordOptOut, "STOP"},
		{"  stop \n", false, KeywordOptOut, "STOP"},
		{"unsubscribe", false, KeywordOptOut, "UNSUBSCRIBE"},
		{"StopAll", false, KeywordOptOut, "STOPALL"},
		{"START", false, KeywordOptIn, "START"},
		{"unstop", false, KeywordOptIn, "UNSTOP"},
		{"help", false, KeywordHelp, "HELP"},
		{"Info", false, KeywordHelp, "INFO"},
		// Only the whole reply counts.
		{"Please stop texting me", false, KeywordReply, ""},
		{"STOP!", false, KeywordReply, ""},
		{"Is there parking?", false, KeywordReply, ""},
		// YES confirms consent only when the workspace asks for it.
		{"yes", false, KeywordReply, ""},
		{"Yes", true, KeywordYes, "YES"},
		{"", false, KeywordReply, ""},
	}
	for _, c := range cases {
		kind, word := k.Match(c.body, c.yes)
		if kind != c.kind || word != c.word {
			t.Errorf("Match(%q) = %s %q, want %s %q", c.body, kind, word, c.kind, c.word)
		}
	}
	// Words in other languages can be added.
	k2 := Keywords{OptOut: append(append([]string{}, k.OptOut...), "arret", "STOPP"), OptIn: k.OptIn, Help: k.Help}.Normalized()
	if err := k2.Validate(); err != nil {
		t.Fatal(err)
	}
	if kind, word := k2.Match("Arret", false); kind != KeywordOptOut || word != "ARRET" {
		t.Fatalf("ARRET: %s %s", kind, word)
	}
}

func TestKeywordValidate(t *testing.T) {
	bad := []struct {
		k    Keywords
		want error
	}{
		{Keywords{OptOut: []string{"STOP", "two words"}, Help: []string{"HELP"}}, ErrKeywordFormat},
		{Keywords{OptOut: []string{"STOP"}, OptIn: []string{"stop"}, Help: []string{"HELP"}}, ErrKeywordDuplicate},
		{Keywords{OptOut: []string{"END"}, Help: []string{"HELP"}}, ErrKeywordRequired},
		{Keywords{OptOut: []string{"STOP"}, Help: []string{"INFO"}}, ErrKeywordRequired},
		{Keywords{OptOut: []string{"STOP", " "}, Help: []string{"HELP"}}, ErrKeywordEmpty},
	}
	for _, b := range bad {
		if err := b.k.Validate(); !errors.Is(err, b.want) {
			t.Errorf("%+v: %v, want %v", b.k, err, b.want)
		}
	}
	if err := DefaultKeywords.Validate(); err != nil {
		t.Fatal(err)
	}
}
