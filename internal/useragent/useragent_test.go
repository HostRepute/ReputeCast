package useragent

import (
	"strings"
	"testing"
)

func TestBot(t *testing.T) {
	cases := map[string]string{
		"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/129.0.0.0 Safari/537.36": "",
		// Phones and words with "bot" inside aren't bots.
		"Mozilla/5.0 (Linux; Android 11; CUBOT_X50) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/129.0.0.0 Mobile Safari/537.36": "",
		"Mozilla/5.0 (Linux; Android 10; CUBOT KING KONG 5 Pro) Chrome/129.0 Mobile":                                              "",
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64) Abbott-Kiosk/2.1 Chrome/129.0":                                                 "",
		"Mozilla/5.0 (Linux; Android 14; Robotics-Tab) Chrome/129.0":                                                              "",
		"Mozilla/5.0 (compatible; Googlebot/2.1; +http://www.google.com/bot.html)":                                                AnyBot,
		"Mozilla/5.0 (compatible; AhrefsBot/7.0; +http://ahrefs.com/robot/)":                                                      AnyBot,
		"Mozilla/5.0 (compatible; bingbot/2.0)":                                                                                   AnyBot,
		"Mozilla/5.0 (compatible; Baiduspider/2.0)":                                                                               AnyBot,
		"Mozilla/5.0 (compatible; Yahoo! Slurp)":                                                                                  AnyBot,
		"Slackbot-LinkExpanding 1.0 (+https://api.slack.com/robots)":                                                              Preview,
		"facebookexternalhit/1.1":                                                                                                 Preview,
		"Mozilla/5.0 (Windows NT 10.0) SkypeUriPreview Preview/0.5":                                                               Preview,
		"python-requests/2.32":           Script,
		"curl/8.5.0":                     Script,
		"Mozilla/5.0 Barracuda Sentinel": "Barracuda",
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64) HeadlessChrome/120.0.0.0": Headless,
		"Microsoft Office Protocol Discovery":                                "Microsoft Defender",
		"":                                                                   "",
	}
	for ua, want := range cases {
		if got := Bot(ua); got != want {
			t.Errorf("Bot(%q) = %q, want %q", ua, got, want)
		}
	}
}

// TestList: an instance's list is the built-in one plus what its admins
// add: a machine's pattern anywhere in the user agent, and a person's
// device that a word rule would take for a machine (review AN-2).
func TestList(t *testing.T) {
	gateway := "Mozilla/5.0 (Windows NT 10.0) AcmeGate-LinkCheck/3.1"
	kiosk := "Mozilla/5.0 (Linux; Android 13; RoboKiosk) Chrome/129.0 Mobile"
	if Bot(gateway) != "" || Bot(kiosk) != "" {
		t.Fatalf("built in: gateway %q, kiosk %q", Bot(gateway), Bot(kiosk))
	}
	kioskBot := "Mozilla/5.0 (Linux; Android 13; KioskBot) Chrome/129.0 Mobile"
	if Bot(kioskBot) != AnyBot {
		t.Fatalf("built in: %q", Bot(kioskBot))
	}
	l := NewList(append(Builtin(), Pattern{Text: "AcmeGate-LinkCheck", Kind: KindScanner, Name: "Our mail gateway"},
		Pattern{Text: "kioskbot", Kind: KindPerson, Name: "Lobby kiosks"}, Pattern{Text: "ok-client/", Kind: KindScript}))
	for ua, want := range map[string]string{
		gateway:  "Our mail gateway",
		kioskBot: "",
		"Mozilla/5.0 (compatible; Googlebot/2.1)": AnyBot,
		"ok-client/1.0":                  Script,
		"Mozilla/5.0 Barracuda Sentinel": "Barracuda",
		"Mozilla/5.0 (Linux; Android 11; CUBOT_X50) Chrome/129.0 Mobile": "",
	} {
		if got := l.Bot(ua); got != want {
			t.Errorf("list Bot(%q) = %q, want %q", ua, got, want)
		}
	}
	// A list without the built-in patterns doesn't know them.
	if got := NewList(nil).Bot("Mozilla/5.0 Barracuda Sentinel"); got != "" {
		t.Errorf("an empty list found %q", got)
	}
	var none *List
	if got := none.Bot("curl/8.5.0"); got != Script {
		t.Errorf("a nil list is the built-in one: %q", got)
	}
}

// TestPatternChecks: what an admin may add. A machine's pattern found in a
// browser's or mail client's user agent would take people for machines; a
// person's is one word, as the rules read user agents.
func TestPatternChecks(t *testing.T) {
	for _, p := range []string{"mozilla", "Chrome", "safari/", "gecko", "outlook", "googleimageproxy", "iphone", "windows nt"} {
		if !TakesPeople(p) {
			t.Errorf("%q would take people for machines", p)
		}
	}
	for _, p := range []string{"acmegate", "barracuda", "linkcheck/", "python-requests"} {
		if TakesPeople(p) {
			t.Errorf("%q isn't found in people's user agents", p)
		}
	}
	if !Word("kioskbot") || !Word("KioskBot") || Word("kiosk bot") || Word("kiosk-bot") || Word("") {
		t.Error("a person's pattern is one word of letters and digits")
	}
	for _, p := range Builtin() {
		if p.Text != strings.ToLower(p.Text) || len(p.Text) < 3 || TakesPeople(p.Text) {
			t.Errorf("built-in pattern %q", p.Text)
		}
		if p.Kind == KindPerson && !Word(p.Text) {
			t.Errorf("built-in person %q isn't one word", p.Text)
		}
	}
}
