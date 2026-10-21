package i18n

import (
	"encoding/json"
	"testing"
	"testing/fstest"
	"time"
)

func TestKeptTextReadsInEachReadersLanguage(t *testing.T) {
	fsys := fstest.MapFS{
		"messages/en.json": {Data: []byte(`{"check.off":"Turned off on {at, date} at {at, time} by {name}: {count, plural, one {# recipient misses} other {# recipients miss}} it."}`)},
		"messages/vi.json": {Data: []byte(`{"check.off":"{name} đã tắt lúc {at, time} ngày {at, date}: {count, plural, other {# người nhận}} không thấy."}`)},
	}
	c, err := Load(fsys, "kepttest")
	if err != nil {
		t.Fatal(err)
	}
	at := time.Date(2026, 10, 13, 9, 5, 0, 0, time.FixedZone("ICT", 7*3600))
	args := Args{"at": at, "name": "Linh", "count": 31604}
	kept := c.M("check.off").Keep(args)
	b, err := json.Marshal(kept)
	if err != nil {
		t.Fatal(err)
	}
	var back Text
	if err := json.Unmarshal(b, &back); err != nil {
		t.Fatal(err)
	}
	// Read back from storage, it reads as the message formatted at the time
	// it was kept, in each reader's language and region.
	for _, l := range []Lang{For(English), For(Vietnamese), Resolve("vi", "en-US", "", "")} {
		if got, want := back.In(l), c.M("check.off").Text(l, args); got != want {
			t.Errorf("%v: %q, want %q", l, got, want)
		}
	}
	if got := back.In(For(Vietnamese)); got != "Linh đã tắt lúc 09:05 ngày 13 thg 10, 2026: 31.604 người nhận không thấy." {
		t.Errorf("vi: %q", got)
	}
	// Rows without a message, or with one a catalog no longer has, show the
	// English they stored.
	for _, gone := range []Text{{}, {ID: "kepttest.removed"}, {ID: "nowhere.check.off"}} {
		if got := gone.In(For(Vietnamese)); got != "" {
			t.Errorf("%+v: %q, want empty", gone, got)
		}
	}
	if !(Text{}).IsZero() || kept.IsZero() || kept.ID != "kepttest.check.off" {
		t.Errorf("kept %+v", kept)
	}
}
