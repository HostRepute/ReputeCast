package ai

import "testing"

// The assistant's HostRepute card takes diagnosis questions only, in the
// app's five languages; setup and how-to questions go to the model. The
// one-shot rewrite's check stays as broad as it was.
func TestAssistantReputationDiagnosis(t *testing.T) {
	for text, want := range map[string]bool{
		"How do I set up DKIM?":                         false,
		"Cách cấu hình DKIM?":                           false,
		"What should our DMARC policy be?":              false,
		"Wie richte ich SPF für unsere Domain ein?":     false,
		"¿Cómo configuro DMARC?":                        false,
		"Comment configurer DKIM et SPF ?":              false,
		"Which deliverability settings matter most?":    false,
		"Summarize the October report":                  false,
		"Why are my emails going to spam?":              true,
		"Tại sao email vào thư rác?":                    true,
		"Are we on a blocklist?":                        true,
		"Is our domain blacklisted?":                    true,
		"What is our sender score?":                     true,
		"Check Postmaster Tools for us":                 true,
		"Do we hit spam traps?":                         true,
		"Sind wir auf einer Sperrliste?":                true,
		"Warum landen unsere Mails im Spam?":            true,
		"¿Por qué mis correos llegan a spam?":           true,
		"¿Estamos en una lista negra?":                  true,
		"Pourquoi nos emails arrivent dans les spams ?": true,
		"Sommes-nous sur une liste noire ?":             true,
		"Uy tín người gửi của chúng tôi thế nào?":       true,
	} {
		if got := AsksAboutReputation(text); got != want {
			t.Errorf("%q: %v, want %v", text, got, want)
		}
	}
	if !asksAboutReputation("How do I set up DKIM?") {
		t.Error("the one-shot rewrite's check changed")
	}
}
