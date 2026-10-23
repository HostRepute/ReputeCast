package digests

import (
	"encoding/json"
	"fmt"
	"html"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/hostrepute/reputecast/internal/digests/feed"
	"github.com/hostrepute/reputecast/internal/i18n"
)

// The Feed items block (brief 3.1): a ReputeCast block of the composer in
// digest mode that repeats for each post of an issue. A digest's template
// stores it as one Raw HTML element carrying its settings,
//
//	<mj-raw><!-- rc:feed-items {"layout":"cards",…} --></mj-raw>
//
// which the composer draws with the preview's posts and an issue replaces
// with a section per post (RenderBlock), each marked css-class="rc-feed-item".

// Fields of a feed item the block can show.
const (
	FieldImage      = "image"
	FieldTitle      = "title"
	FieldDate       = "date"
	FieldAuthor     = "author"
	FieldCategories = "categories"
	FieldExcerpt    = "excerpt"
	FieldButton     = "button"
)

// AllFields are the fields in the inspector's order.
var AllFields = []string{FieldImage, FieldTitle, FieldDate, FieldAuthor, FieldCategories, FieldExcerpt, FieldButton}

// DigestFeedBlock is the Feed items block's settings.
type DigestFeedBlock struct {
	// Layout: list (title and excerpt), cards (the image above the text,
	// one post per row) or grid (two per row, stacked on mobile).
	Layout string   `json:"layout" enum:"list,cards,grid"`
	Fields []string `json:"fields" maxItems:"7" doc:"image, title (linked), date, author, categories, excerpt, button"`
	// ExcerptWords is the excerpt's length in words.
	ExcerptWords int    `json:"excerptWords" minimum:"5" maximum:"200"`
	ButtonText   string `json:"buttonText" maxLength:"60"`
	// Max is how many posts an issue shows, 1 to 20.
	Max   int    `json:"max" minimum:"1" maximum:"20"`
	Order string `json:"order" enum:"newest,oldest"`
	// Overflow is what happens to more new posts than fit: drop (send the
	// newest and leave out the rest) or keep (they count as new until sent).
	Overflow string `json:"overflow" enum:"drop,keep"`
	// Fallback is the image of posts without one (a library URL).
	Fallback     string `json:"fallback,omitempty" maxLength:"500"`
	FallbackName string `json:"fallbackName,omitempty" maxLength:"200"`
}

// DefaultBlock is the "Blog digest" starter's block.
func DefaultBlock() DigestFeedBlock {
	return DigestFeedBlock{Layout: "cards", Fields: []string{FieldImage, FieldTitle, FieldDate, FieldAuthor, FieldExcerpt, FieldButton},
		ExcerptWords: 40, ButtonText: "Read the post", Max: 6, Order: "newest", Overflow: "drop"}
}

// normalized fills what a stored block leaves out and clamps the numbers.
func (b DigestFeedBlock) normalized() DigestFeedBlock {
	d := DefaultBlock()
	if b.Layout != "list" && b.Layout != "cards" && b.Layout != "grid" {
		b.Layout = d.Layout
	}
	if b.Fields == nil {
		b.Fields = d.Fields
	}
	if b.ExcerptWords < 5 || b.ExcerptWords > 200 {
		b.ExcerptWords = d.ExcerptWords
	}
	if strings.TrimSpace(b.ButtonText) == "" {
		b.ButtonText = d.ButtonText
	}
	if b.Max < 1 || b.Max > 20 {
		b.Max = d.Max
	}
	if b.Order != "oldest" {
		b.Order = "newest"
	}
	if b.Overflow != "keep" {
		b.Overflow = "drop"
	}
	return b
}

func (b DigestFeedBlock) has(field string) bool { return slices.Contains(b.Fields, field) }

// BlockItem is a post as the block shows it.
type BlockItem struct {
	ID         string
	Title      string
	URL        string
	Author     string
	Published  *time.Time
	Categories []string
	Excerpt    string
	// Image is the library copy (or the fallback); ImageAlt its alt text.
	Image    string
	ImageAlt string
	// AltFromTitle: the image has no alt text of its own, so the title is used.
	AltFromTitle bool
	// Fallback: the post has no image and shows the block's fallback image.
	Fallback bool
}

// withImage fills the item's image and alt text for a block: its own
// image, else the block's fallback; alt text from the image, else the title.
func (b DigestFeedBlock) withImage(it BlockItem) BlockItem {
	if it.Image == "" && b.Fallback != "" {
		it.Image, it.Fallback, it.ImageAlt = b.Fallback, true, ""
	}
	if it.Image != "" && strings.TrimSpace(it.ImageAlt) == "" {
		it.ImageAlt, it.AltFromTitle = it.Title, true
	}
	return it
}

// Select picks the posts an issue shows from its new items (newest first):
// at most Max, in the block's order. The rest are left out (drop) or stay
// new for the next issue (keep).
func (b DigestFeedBlock) Select(items []BlockItem) (shown, rest []BlockItem) {
	b = b.normalized()
	n := min(len(items), b.Max)
	if b.Overflow == "keep" {
		// The oldest new posts go first, so none waits forever.
		shown = append([]BlockItem{}, items[len(items)-n:]...)
		rest = append([]BlockItem{}, items[:len(items)-n]...)
	} else {
		shown = append([]BlockItem{}, items[:n]...)
		rest = append([]BlockItem{}, items[n:]...)
	}
	if b.Order == "oldest" {
		slices.Reverse(shown)
	} else {
		// Keep newest first whatever was picked.
		slices.SortStableFunc(shown, func(x, y BlockItem) int { return compareNewest(x, y) })
	}
	return shown, rest
}

func compareNewest(x, y BlockItem) int {
	switch {
	case x.Published == nil && y.Published == nil:
		return 0
	case x.Published == nil:
		return 1
	case y.Published == nil:
		return -1
	case x.Published.After(*y.Published):
		return -1
	case y.Published.After(*x.Published):
		return 1
	}
	return 0
}

const (
	plex     = `'IBM Plex Sans', Helvetica, Arial, sans-serif`
	teal     = "#0F766E"
	inkColor = "#0B1220"
	bodyText = "#374151"
	metaText = "#5B6472"
	divider  = "#E3E6EB"
)

// esc writes text for the email: HTML-escaped, and Liquid's braces as
// entities, so a post titled "{{ x }}" can't become a token.
func esc(s string) string { return braces.Replace(html.EscapeString(s)) }

var braces = strings.NewReplacer("{", "&#123;", "}", "&#125;")

// RenderBlock writes the block for an issue's posts as MJML sections, one
// per post (two per row in the grid), each marked css-class="rc-feed-item";
// each post's column is marked css-class="rc-feed-post" (the composer's
// canvas labels them). Dates are written in lang, the email's language.
func RenderBlock(b DigestFeedBlock, items []BlockItem, loc *time.Location, lang i18n.Lang) string {
	b = b.normalized()
	if loc == nil {
		loc = time.UTC
	}
	var out strings.Builder
	switch b.Layout {
	case "grid":
		for i := 0; i < len(items); i += 2 {
			pad := "18px 24px 4px"
			if i == 0 {
				pad = "28px 24px 4px"
			}
			fmt.Fprintf(&out, `<mj-section background-color="#ffffff" padding="%s" css-class="rc-feed-item">`, pad)
			fmt.Fprintf(&out, `<mj-column width="50%%" padding="0px 8px 0px 0px" css-class="rc-feed-post">%s</mj-column>`, gridCell(b, b.withImage(items[i]), loc, lang))
			if i+1 < len(items) {
				fmt.Fprintf(&out, `<mj-column width="50%%" padding="0px 0px 0px 8px" css-class="rc-feed-post">%s</mj-column>`, gridCell(b, b.withImage(items[i+1]), loc, lang))
			}
			out.WriteString(`</mj-section>`)
		}
	default:
		for i, it := range items {
			it = b.withImage(it)
			pad := "0px 24px 4px"
			if i == 0 {
				pad = "28px 24px 4px"
			}
			fmt.Fprintf(&out, `<mj-section background-color="#ffffff" padding="%s" css-class="rc-feed-item"><mj-column css-class="rc-feed-post">`, pad)
			if i > 0 {
				fmt.Fprintf(&out, `<mj-divider border-width="1px" border-color="%s" padding="14px 0px 18px"></mj-divider>`, divider)
			}
			if b.Layout == "list" {
				out.WriteString(listCell(b, it, loc, lang))
			} else {
				out.WriteString(cardCell(b, it, loc, lang))
			}
			out.WriteString(`</mj-column></mj-section>`)
		}
	}
	if len(items) > 0 {
		// Room under the last post.
		out.WriteString(`<mj-section background-color="#ffffff" padding="0px 24px 12px"><mj-column><mj-spacer height="4px"></mj-spacer></mj-column></mj-section>`)
	}
	return out.String()
}

func blockMeta(b DigestFeedBlock, it BlockItem, loc *time.Location, lang i18n.Lang, short bool) string {
	parts := []string{}
	if b.has(FieldDate) && it.Published != nil {
		form := datePost
		if short {
			form = dateCard
		}
		parts = append(parts, emailDate(lang, it.Published.In(loc), form))
	}
	if b.has(FieldAuthor) && it.Author != "" {
		parts = append(parts, esc(it.Author))
	}
	if len(parts) == 0 {
		return ""
	}
	return fmt.Sprintf(`<mj-text font-family="%s" font-size="12px" line-height="1.5" color="%s" padding="0px 0px 4px">%s</mj-text>`, plex, metaText, strings.Join(parts, " · "))
}

func blockTitle(b DigestFeedBlock, it BlockItem, size string, linkColor string, underline bool) string {
	if !b.has(FieldTitle) {
		return ""
	}
	deco := "none"
	if underline {
		deco = "underline"
	}
	t := esc(it.Title)
	if it.URL != "" {
		t = fmt.Sprintf(`<a href="%s" style="color:%s;text-decoration:%s">%s</a>`, esc(it.URL), linkColor, deco, t)
	}
	return fmt.Sprintf(`<mj-text font-size="%s" line-height="1.3" font-weight="700" color="%s" padding="0px 0px 6px"><h2 style="margin:0;font-size:%s;line-height:1.3;font-weight:700">%s</h2></mj-text>`,
		size, inkColor, size, t)
}

func blockCategories(b DigestFeedBlock, it BlockItem) string {
	if !b.has(FieldCategories) || len(it.Categories) == 0 {
		return ""
	}
	cats := make([]string, 0, len(it.Categories))
	for _, c := range it.Categories {
		cats = append(cats, esc(c))
	}
	return fmt.Sprintf(`<mj-text font-family="%s" font-size="12px" font-weight="600" color="%s" padding="0px 0px 6px">%s</mj-text>`, plex, teal, strings.Join(cats, " · "))
}

func blockExcerpt(b DigestFeedBlock, it BlockItem, words int) string {
	if !b.has(FieldExcerpt) || it.Excerpt == "" {
		return ""
	}
	return fmt.Sprintf(`<mj-text font-size="14px" line-height="1.55" color="%s" padding="0px 0px 12px">%s</mj-text>`, bodyText, esc(feed.Words(it.Excerpt, words)))
}

func blockButton(b DigestFeedBlock, it BlockItem) string {
	if !b.has(FieldButton) || it.URL == "" {
		return ""
	}
	return fmt.Sprintf(`<mj-button href="%s" align="left" background-color="%s" color="#ffffff" font-family="%s" font-size="13px" font-weight="600" border-radius="6px" inner-padding="9px 14px" padding="0px 0px 8px">%s</mj-button>`,
		esc(it.URL), teal, plex, esc(b.ButtonText))
}

func blockImage(b DigestFeedBlock, it BlockItem, width int) string {
	if !b.has(FieldImage) || it.Image == "" {
		return ""
	}
	href := ""
	if it.URL != "" {
		href = fmt.Sprintf(` href="%s"`, esc(it.URL))
	}
	return fmt.Sprintf(`<mj-image src="%s" alt="%s" width="%dpx"%s padding="0px 0px 12px"></mj-image>`, esc(it.Image), esc(it.ImageAlt), width, href)
}

func cardCell(b DigestFeedBlock, it BlockItem, loc *time.Location, lang i18n.Lang) string {
	return blockImage(b, it, 552) + blockMeta(b, it, loc, lang, false) + blockTitle(b, it, "17px", inkColor, false) + blockCategories(b, it) + blockExcerpt(b, it, b.ExcerptWords) + blockButton(b, it)
}

func listCell(b DigestFeedBlock, it BlockItem, loc *time.Location, lang i18n.Lang) string {
	return blockTitle(b, it, "17px", teal, true) + blockMeta(b, it, loc, lang, false) + blockCategories(b, it) + blockExcerpt(b, it, b.ExcerptWords)
}

func gridCell(b DigestFeedBlock, it BlockItem, loc *time.Location, lang i18n.Lang) string {
	return blockImage(b, it, 264) + blockMeta(b, it, loc, lang, true) + blockTitle(b, it, "17px", inkColor, false) + blockCategories(b, it) + blockExcerpt(b, it, max(10, b.ExcerptWords/2)) + blockButton(b, it)
}

// blockMarker finds the Feed items block of a template.
var blockMarker = regexp.MustCompile(`(?s)<mj-raw(?:\s[^>]*)?>\s*<!--\s*rc:feed-items(\s+\{.*?\})?\s*-->\s*</mj-raw>`)

// jsonDash is a hyphen written as a JSON escape (backslash u002d), so the
// marker's JSON never holds "--".
var jsonDash = string([]byte{0x5c, 0x75, 0x30, 0x30, 0x32, 0x64})

// MarkerFor writes a block's settings as the template's marker. The JSON
// never contains "--", which would end the comment.
func MarkerFor(b DigestFeedBlock) string {
	raw, _ := json.Marshal(b.normalized())
	s := string(raw)
	for strings.Contains(s, "--") {
		s = strings.ReplaceAll(s, "--", "-"+jsonDash)
	}
	return "<mj-raw><!-- rc:feed-items " + s + " --></mj-raw>"
}

// BlockOf reads a template's Feed items block; ok is false without one.
func BlockOf(mjml string) (DigestFeedBlock, bool) {
	m := blockMarker.FindStringSubmatch(mjml)
	if m == nil {
		return DefaultBlock(), false
	}
	b := DefaultBlock()
	if strings.TrimSpace(m[1]) != "" {
		var stored DigestFeedBlock
		if json.Unmarshal([]byte(strings.TrimSpace(m[1])), &stored) == nil {
			b = stored
		}
	}
	return b.normalized(), true
}

// ExpandBlock replaces a template's Feed items block with the issue's
// posts. The block's place is kept with a marker naming its layout, so the
// composer labels the posts of an issue ("Feed item 1 of 3").
func ExpandBlock(mjml string, items []BlockItem, loc *time.Location, lang i18n.Lang) string {
	b, ok := BlockOf(mjml)
	if !ok {
		return mjml
	}
	loc0 := blockMarker.FindStringIndex(mjml)
	marker := fmt.Sprintf(`<mj-raw><!-- rc:feed-issue {"layout":%q,"items":%d} --></mj-raw>`, b.Layout, len(items))
	return mjml[:loc0[0]] + marker + RenderBlock(b, items, loc, lang) + mjml[loc0[1]:]
}
