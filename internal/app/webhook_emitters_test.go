package app

import (
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/hostrepute/reputecast/internal/webhooks"
)

// TestEveryWebhookEventHasAnEmitter (review 2026-11 EN-6): an event in the
// catalog people subscribe to must be emitted somewhere in production code,
// by its name or by a constant naming it. Twelve events were offered and
// never sent before.
func TestEveryWebhookEventHasAnEmitter(t *testing.T) {
	root := filepath.Join("..")
	constDef := regexp.MustCompile(`(?m)^\s*(\w+)\s*(?:string\s*)?=\s*"([a-z_]+\.[a-z_*]+)"`)
	type file struct{ path, text string }
	var files []file
	err := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if d.Name() == "demo" || d.Name() == "apptest" || d.Name() == "testdb" || d.Name() == "store" {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(p, ".go") || strings.HasSuffix(p, "_test.go") {
			return nil
		}
		b, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		files = append(files, file{p, string(b)})
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	// Constants naming an event: EmailSent = "email.sent".
	names := map[string][]string{}
	for _, f := range files {
		for _, m := range constDef.FindAllStringSubmatch(f.text, -1) {
			names[m[2]] = append(names[m[2]], m[1])
		}
	}
	catalog := filepath.Join("..", "webhooks", "webhooks.go")
	for _, ev := range webhooks.Events {
		lit := `"` + ev.Name + `"`
		found := false
		for _, f := range files {
			if f.path == catalog {
				continue
			}
			for _, line := range strings.Split(f.text, "\n") {
				if strings.Contains(line, lit) && !constDef.MatchString(line) {
					found = true
					break
				}
				for _, n := range names[ev.Name] {
					if regexp.MustCompile(`\b`+n+`\b`).MatchString(line) && !constDef.MatchString(line) {
						found = true
						break
					}
				}
				if found {
					break
				}
			}
			if found {
				break
			}
		}
		if !found {
			t.Errorf("webhook event %s is in the catalog but nothing emits it", ev.Name)
		}
	}
}
