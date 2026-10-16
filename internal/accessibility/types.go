package accessibility

import "time"

// Issue kinds.
const (
	KindAltText    = "alt_text"
	KindLinkNoText = "link_no_text"
	KindContrast   = "contrast"
	KindLinkText   = "link_text"
	KindHeading    = "heading"
	KindTextSize   = "text_size"
)

// Severities. Serious issues can block sending when the workspace says so;
// moderate issues only warn.
const (
	Serious  = "serious"
	Moderate = "moderate"
)

// Workspace policy choices for serious issues.
const (
	PolicyWarn  = "warn"
	PolicyBlock = "block"
)

// Minimum text size bounds (px).
const (
	DefaultMinTextPx = 12
	MinMinTextPx     = 12
	MaxMinTextPx     = 16
)

// AccessibilityLanguage is a language version that was checked.
type AccessibilityLanguage struct {
	Code   string `json:"code" example:"es"`
	Name   string `json:"name" example:"Spanish"`
	Source bool   `json:"source"`
}

// AccessibilityElement is the element an issue is about.
type AccessibilityElement struct {
	// Tag is the MJML component ("mj-image") or, inside a text block, the
	// HTML element ("a", "h3").
	Tag  string `json:"tag"`
	Role string `json:"role" enum:"image,linked_image,button,link,text,heading"`
	// Label is the button's label, the link's text in the first language,
	// the heading's text or the image's file name.
	Label string `json:"label,omitempty"`
	// Section is the label of the section it's in ("hero"), when the
	// section has one.
	Section string `json:"section,omitempty"`
	// Index locates an element inside a text block: the nth link or heading
	// of the component, from 0.
	Index int    `json:"index"`
	Href  string `json:"href,omitempty"`
	Src   string `json:"src,omitempty"`
	// Count is the number of elements the issue covers ("h3 × 2").
	Count int `json:"count"`
}

// AccessibilityColorSuggestion is a color that fixes a contrast issue.
type AccessibilityColorSuggestion struct {
	Color string  `json:"color" example:"#0F766E"`
	Ratio float64 `json:"ratio" doc:"Contrast ratio, rounded down to one decimal"`
	// Reason: the color of the email's other buttons, the nearest shade that
	// passes AA, or one that passes AAA.
	Reason string `json:"reason" enum:"other_buttons,nearest_aa,aaa"`
}

// AccessibilityHeadingChange is one heading a level fix changes.
type AccessibilityHeadingChange struct {
	Path  []int  `json:"path"`
	Index int    `json:"index" doc:"The nth heading of the text block, from 0"`
	Text  string `json:"text"`
	From  int    `json:"from"`
	To    int    `json:"to"`
}

// AccessibilityFixSpec says how an issue is fixed.
type AccessibilityFixSpec struct {
	Kind string `json:"kind" enum:"alt_text,link_text,color,heading,text_size"`
	// OneClick fixes need no words; they apply to every language version.
	OneClick bool `json:"oneClick"`
	// PerLanguage fixes are words, written in each language.
	PerLanguage bool `json:"perLanguage"`
	// Decorative is offered for images that aren't linked.
	Decorative bool `json:"decorative"`
	// Color fixes: the color that changes (Target, From), the one it is
	// compared with (Against), the ratio now and what it needs.
	Target      string                         `json:"target,omitempty" enum:"background,text"`
	From        string                         `json:"from,omitempty"`
	Against     string                         `json:"against,omitempty"`
	Ratio       float64                        `json:"ratio,omitempty"`
	Needs       float64                        `json:"needs,omitempty"`
	Large       bool                           `json:"large,omitempty"`
	Suggestions []AccessibilityColorSuggestion `json:"suggestions"`
	// Heading fixes: each heading's level now and after.
	Headings []AccessibilityHeadingChange `json:"headings"`
	// Text size fixes.
	FromPx float64 `json:"fromPx,omitempty"`
	ToPx   int     `json:"toPx,omitempty"`
	// Subject names what a one-click fix changes, as its preview lists it:
	// "Card titles", "Footer text".
	Subject string `json:"subject,omitempty"`
	// Links are the links a link-text fix changes (links that share their
	// words are one issue).
	Links []AccessibilityLinkTarget `json:"links"`
}

// AccessibilityLinkTarget is one link of a link-text issue.
type AccessibilityLinkTarget struct {
	Path  []int  `json:"path"`
	Index int    `json:"index" doc:"The nth link of the text block, from 0"`
	Href  string `json:"href"`
}

// AccessibilityIssueVersion is what an issue looks like in one language.
type AccessibilityIssueVersion struct {
	Lang string `json:"lang"`
	// Text is the link's text in this language ("Haz clic aquí").
	Text string `json:"text,omitempty"`
	// Suggestion is the linked page's title, offered when the page is in
	// this language.
	Suggestion string `json:"suggestion,omitempty"`
}

// AccessibilityIssue is one finding, listed once with every language it's in.
type AccessibilityIssue struct {
	// ID is stable while the email keeps its structure ("contrast:6-0-0").
	ID       string `json:"id"`
	Kind     string `json:"kind" enum:"alt_text,link_no_text,contrast,link_text,heading,text_size"`
	Severity string `json:"severity" enum:"serious,moderate"`
	// Title and Detail are the finding in English (API clients, MCP agents,
	// Send Readiness); the app words them from the fields below.
	Title  string `json:"title"`
	Detail string `json:"detail,omitempty"`
	// Tag is the short label on the canvas ("No alt text").
	Tag string `json:"tag"`
	// Short names the issue inside a sentence: "the hero image's alt text",
	// "“Click here”", "a bare URL", "11 px footer text".
	Short     string   `json:"short"`
	Languages []string `json:"languages"`
	// Path is the element-index path from mj-body of the (first) element;
	// Paths lists every element the issue covers. Empty for HTML-only emails.
	Path  []int   `json:"path"`
	Paths [][]int `json:"paths"`
	// Location is the element's path as the panel shows it:
	// ["mjml", "mj-body", "mj-section · hero", "mj-image"].
	Location []string                    `json:"location"`
	Element  AccessibilityElement        `json:"element"`
	Fix      AccessibilityFixSpec        `json:"fix"`
	Versions []AccessibilityIssueVersion `json:"versions"`
}

// AccessibilityCounts are the passing checks' numbers.
type AccessibilityCounts struct {
	// Languages are the codes lang was set to, one per version ("en", "es").
	Languages []string `json:"languages"`
	Dir       string   `json:"dir" enum:"ltr"`
	// LayoutTables carry role="presentation"; RoleAdded of them were marked
	// by ReputeCast; DataTables kept their meaning.
	LayoutTables int `json:"layoutTables"`
	RoleAdded    int `json:"roleAdded"`
	DataTables   int `json:"dataTables"`
	Images       int `json:"images"`
	// ImagesOK have alt text or are marked decorative.
	ImagesOK int `json:"imagesOk"`
	Links    int `json:"links"`
	// LinksOK say where they go.
	LinksOK int `json:"linksOk"`
	// OverImage counts texts over a background image, which can't be
	// checked for contrast ("Check it by eye"); they count as no issue.
	OverImage int `json:"overImage"`
}

// AccessibilityReport is the accessibility of an email, every language
// version checked together.
type AccessibilityReport struct {
	Issues    []AccessibilityIssue    `json:"issues"`
	Serious   int                     `json:"serious"`
	Moderate  int                     `json:"moderate"`
	Languages []AccessibilityLanguage `json:"languages"`
	Counts    AccessibilityCounts     `json:"counts"`
	// Policy is the workspace's: under "block", serious issues block sending.
	Policy     string    `json:"policy" enum:"warn,block"`
	MinTextPx  int       `json:"minTextPx"`
	CheckedAt  time.Time `json:"checkedAt"`
	DurationMs int       `json:"durationMs"`
	// Mode is how the email is written: issues of an HTML-only email have
	// no element path, so its fixes happen in the code.
	Mode string `json:"mode" enum:"mjml,html"`
}

// Summary is "6 issues: 2 serious, 4 moderate" or "No issues".
func (r AccessibilityReport) Summary() string {
	n := len(r.Issues)
	if n == 0 {
		return "No issues"
	}
	return plural(n, "issue", "issues") + ": " + count(r.Serious) + " serious, " + count(r.Moderate) + " moderate"
}
