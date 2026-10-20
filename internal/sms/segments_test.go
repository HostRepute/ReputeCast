package sms

import (
	"encoding/json"
	"os"
	"slices"
	"testing"
)

// The vectors are shared with the web app's counter
// (web/src/features/sms/segments.test.ts), so the composer and the server
// always agree.
type segmentCase struct {
	Name       string    `json:"name"`
	Text       string    `json:"text"`
	Characters int       `json:"characters"`
	Encoding   Encoding  `json:"encoding"`
	Segments   int       `json:"segments"`
	Fill       []int     `json:"fill"`
	Left       int       `json:"left"`
	Outside    []Outside `json:"outside"`
	Fixed      *string   `json:"fixed"`
}

func loadSegmentCases(t *testing.T) []segmentCase {
	t.Helper()
	b, err := os.ReadFile("testdata/segments.json")
	if err != nil {
		t.Fatal(err)
	}
	var v struct {
		Cases []segmentCase `json:"cases"`
	}
	if err := json.Unmarshal(b, &v); err != nil {
		t.Fatal(err)
	}
	return v.Cases
}

func TestCountTextVectors(t *testing.T) {
	for _, tc := range loadSegmentCases(t) {
		t.Run(tc.Name, func(t *testing.T) {
			c := CountText(tc.Text)
			if c.Characters != tc.Characters || c.Encoding != tc.Encoding || c.Segments != tc.Segments || c.Left != tc.Left || !slices.Equal(c.Fill, tc.Fill) {
				t.Fatalf("got %d characters, %s, %d segments %v, %d left; want %d, %s, %d %v, %d",
					c.Characters, c.Encoding, c.Segments, c.Fill, c.Left, tc.Characters, tc.Encoding, tc.Segments, tc.Fill, tc.Left)
			}
			if tc.Outside != nil {
				if len(c.Outside) != len(tc.Outside) {
					t.Fatalf("outside %+v, want %+v", c.Outside, tc.Outside)
				}
				for i, o := range tc.Outside {
					g := c.Outside[i]
					if g.Char != o.Char || g.Kind != o.Kind || g.Replacement != o.Replacement || g.Count != o.Count || g.First != o.First {
						t.Fatalf("outside[%d] = %+v, want %+v", i, g, o)
					}
				}
			}
			if tc.Fixed != nil {
				if got := FixEncoding(tc.Text); got != *tc.Fixed {
					t.Fatalf("fixed %q, want %q", got, *tc.Fixed)
				}
				if CountText(*tc.Fixed).Encoding != GSM7 {
					t.Fatalf("the fix left %q outside GSM-7", *tc.Fixed)
				}
			}
		})
	}
}

func TestEveryLookalikeIsGSM7(t *testing.T) {
	for r, l := range lookalikes {
		if InGSM7(r) {
			t.Errorf("%q is in GSM-7 already", r)
		}
		for _, w := range l.with {
			if !InGSM7(w) {
				t.Errorf("replacement %q for %q is outside GSM-7", l.with, r)
			}
		}
	}
}

func TestFixKeepsEmoji(t *testing.T) {
	in := "Hi " + string(rune(0x1F44B)) + string(rune(0x2019))
	if got := FixEncoding(in); got != "Hi "+string(rune(0x1F44B))+"'" {
		t.Fatalf("got %q", got)
	}
}
