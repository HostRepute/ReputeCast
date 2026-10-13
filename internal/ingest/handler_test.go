package ingest

import (
	"encoding/json"
	"net/http"
	"testing"
)

func TestKeepHeadersDropsSecrets(t *testing.T) {
	h := http.Header{}
	h.Set("Authorization", "Bearer s3cret")
	h.Set("Cookie", "session=1")
	h.Set("X-ReputeCast-Secret", "s3cret")
	h.Set("X-Hook-Token", "s3cret")
	h.Set("Svix-Id", "msg_1")
	h.Set("Aeg-Event-Type", "Notification")
	var kept map[string]string
	if err := json.Unmarshal(keepHeaders(h, "x-reputecast-secret", "X-HOOK-TOKEN"), &kept); err != nil {
		t.Fatal(err)
	}
	for _, k := range []string{"Authorization", "Cookie", "X-Reputecast-Secret", "X-Hook-Token"} {
		if _, ok := kept[k]; ok {
			t.Errorf("%s must not be stored: %v", k, kept)
		}
	}
	if kept["Svix-Id"] != "msg_1" || kept["Aeg-Event-Type"] != "Notification" {
		t.Errorf("headers parsing needs must be kept: %v", kept)
	}
}
