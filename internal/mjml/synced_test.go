package mjml

import (
	"strings"
	"testing"
)

// CO-11: synced blocks in a document.

const syncedDoc = `<mjml><mj-body>
<mj-section rc-synced="header"><mj-column><mj-image src="/a.png"></mj-image></mj-column></mj-section>
<mj-section><mj-column><mj-text>Hello</mj-text></mj-column></mj-section>
<!-- rc:footer -->
<mj-section rc-synced="footer" padding="8px"><mj-column><mj-text>Old footer {{ unsubscribe_url }}</mj-text></mj-column></mj-section>
</mj-body></mjml>`

func TestSyncedBlocks(t *testing.T) {
	if got := SyncedKinds(syncedDoc); strings.Join(got, ",") != "header,footer" {
		t.Fatalf("kinds: %v", got)
	}
	block := `<mj-section rc-synced="footer"><mj-column><mj-text>New footer {{ unsubscribe_url }}</mj-text></mj-column></mj-section>`
	out, ok := ReplaceSynced(syncedDoc, "footer", block)
	if !ok || strings.Contains(out, "Old footer") || !strings.Contains(out, "New footer") || !strings.Contains(out, `<mj-image src="/a.png">`) {
		t.Fatalf("replace: %v\n%s", ok, out)
	}
	if again, ok := ReplaceSynced(out, "footer", block); ok || again != out {
		t.Fatal("replacing with the same block changes nothing")
	}
	if _, ok := ReplaceSynced(syncedDoc, "banner", block); ok {
		t.Fatal("a kind the document doesn't hold")
	}
	// The compiled HTML leaves the marker out.
	if res := Compile(out); len(res.Errors) > 0 || strings.Contains(res.HTML, "rc-synced") || !strings.Contains(res.HTML, "New footer") {
		t.Fatalf("compile: %v", res.Errors)
	}

	marked, ok := MarkSynced(`  <mj-section padding="0px"><mj-column></mj-column></mj-section> `, "header")
	if !ok || !strings.HasPrefix(marked, `<mj-section rc-synced="header" padding="0px">`) {
		t.Fatalf("mark: %v %s", ok, marked)
	}
	if _, ok := MarkSynced(`<mj-section></mj-section><mj-section></mj-section>`, "footer"); ok {
		t.Fatal("two sections aren't one block")
	}
	if _, ok := MarkSynced(`<mj-text>Hi</mj-text>`, "footer"); ok {
		t.Fatal("a text isn't a section")
	}
}
