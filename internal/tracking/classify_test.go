package tracking

import (
	"net/netip"
	"testing"
	"time"
)

const (
	chromeMac = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/129.0.0.0 Safari/537.36"
	iosMail   = "Mozilla/5.0 (iPhone; CPU iPhone OS 18_0 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/15E148"
	outlook   = "Mozilla/4.0 (compatible; ms-office; MSOffice 16)"
	gmailPx   = "Mozilla/5.0 (Windows NT 5.1; rv:11.0) Gecko Firefox/11.0 (via ggpht.com GoogleImageProxy)"
	yahooPx   = "YahooMailProxy; https://help.yahoo.com/kb/yahoo-mail-proxy-SLN28749.html"
)

func ip(s string) netip.Addr { return netip.MustParseAddr(s) }

func TestClassifyOpen(t *testing.T) {
	tests := []struct {
		name    string
		sig     Signal
		machine bool
		want    string // reason (machine) or client (human)
	}{
		{"person in Chrome", Signal{UserAgent: chromeMac, IP: ip("198.51.100.7"), SinceSend: 9 * time.Minute}, false, "Chrome on macOS"},
		{"person in Outlook", Signal{UserAgent: outlook, IP: ip("198.51.100.7"), SinceSend: time.Hour}, false, "Outlook"},
		{"person in Apple Mail on iPhone", Signal{UserAgent: iosMail, IP: ip("198.51.100.7"), SinceSend: time.Hour}, false, "Apple Mail on iOS"},
		{"Apple MPP by network", Signal{UserAgent: iosMail, IP: ip("17.58.101.4"), SinceSend: 20 * time.Second}, true, "Apple Mail Privacy Protection prefetch"},
		{"Apple MPP by agent", Signal{UserAgent: "Mozilla/5.0", IP: ip("203.0.113.9"), SinceSend: 30 * time.Minute}, true, "Apple Mail Privacy Protection prefetch"},
		{"Gmail proxy prefetch on arrival", Signal{UserAgent: gmailPx, IP: ip("66.249.84.1"), SinceSend: 4 * time.Second}, true, "Google image proxy prefetch"},
		{"Gmail proxy when someone opens", Signal{UserAgent: gmailPx, IP: ip("66.249.84.1"), SinceSend: 12 * time.Minute}, false, "Gmail"},
		{"Yahoo proxy prefetch", Signal{UserAgent: yahooPx, SinceSend: 10 * time.Second}, true, "Yahoo Mail image proxy prefetch"},
		{"Yahoo proxy later", Signal{UserAgent: yahooPx, SinceSend: time.Hour}, false, "Yahoo Mail"},
		{"scanner agent", Signal{UserAgent: "Mozilla/5.0 Barracuda Sentinel", SinceSend: time.Hour}, true, "Security scanner (Barracuda)"},
		{"script", Signal{UserAgent: "python-requests/2.32", SinceSend: time.Hour}, true, "Security scanner (a script)"},
		{"scanner network on arrival", Signal{UserAgent: chromeMac, IP: ip("40.107.22.10"), SinceSend: 30 * time.Second}, true, "Security scanner (Microsoft Defender)"},
		{"scanner network later is a person", Signal{UserAgent: chromeMac, IP: ip("40.107.22.10"), SinceSend: 3 * time.Hour}, false, "Chrome on macOS"},
		{"too fast", Signal{UserAgent: chromeMac, SinceSend: time.Second}, true, "Opened within 2 s of sending, likely a scanner"},
		{"empty agent", Signal{SinceSend: time.Hour}, true, "No user agent, likely a scanner"},
		{"HEAD", Signal{UserAgent: chromeMac, Method: "HEAD", SinceSend: time.Hour}, true, "HEAD request from a link scanner"},
		{"unknown timing", Signal{UserAgent: chromeMac, SinceSend: -1}, false, "Chrome on macOS"},
		// "bot" inside a device's name isn't a bot (AN-9).
		{"CUBOT phone", Signal{UserAgent: "Mozilla/5.0 (Linux; Android 11; CUBOT_X50) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/129.0.0.0 Mobile Safari/537.36",
			IP: ip("198.51.100.7"), SinceSend: time.Hour}, false, "Chrome on Android"},
		{"crawler", Signal{UserAgent: "Mozilla/5.0 (compatible; AhrefsBot/7.0; +http://ahrefs.com/robot/)", SinceSend: time.Hour}, true, "Security scanner (a bot)"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			v := ClassifyOpen(tc.sig)
			if v.Machine != tc.machine {
				t.Fatalf("machine = %v (%+v), want %v", v.Machine, v, tc.machine)
			}
			wantClass := ClassHuman
			switch {
			case tc.want == reasonApple:
				wantClass = ClassPrivacy
			case tc.machine:
				wantClass = ClassMachine
			}
			if v.Class != wantClass {
				t.Fatalf("class = %q, want %q", v.Class, wantClass)
			}
			got := v.Client
			if v.Machine {
				got = v.Reason
			}
			if got != tc.want {
				t.Fatalf("got %q, want %q", got, tc.want)
			}
		})
	}
}

func TestClassifyClick(t *testing.T) {
	tests := []struct {
		name    string
		sig     Signal
		burst   bool
		machine bool
	}{
		{"person", Signal{UserAgent: chromeMac, SinceSend: 10 * time.Minute}, false, false},
		{"safe links scanner", Signal{UserAgent: "Mozilla/5.0 SafeLinks", SinceSend: time.Hour}, false, true},
		{"within ten seconds", Signal{UserAgent: chromeMac, SinceSend: 3 * time.Second}, false, true},
		{"every link at once", Signal{UserAgent: chromeMac, SinceSend: 2 * time.Minute}, true, true},
		{"scanner network on arrival", Signal{UserAgent: chromeMac, IP: ip("148.163.130.2"), SinceSend: 40 * time.Second}, false, true},
		{"HEAD probe", Signal{UserAgent: chromeMac, Method: "HEAD", SinceSend: time.Hour}, false, true},
		{"headless browser", Signal{UserAgent: "Mozilla/5.0 HeadlessChrome/120", SinceSend: time.Hour}, false, true},
		{"person on a CUBOT phone", Signal{UserAgent: "Mozilla/5.0 (Linux; Android 10; CUBOT KING KONG 5 Pro) Chrome/129.0 Mobile", SinceSend: time.Hour}, false, false},
		{"link preview", Signal{UserAgent: "Slackbot-LinkExpanding 1.0 (+https://api.slack.com/robots)", SinceSend: time.Hour}, false, true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			v := ClassifyClick(tc.sig, tc.burst)
			if v.Machine != tc.machine {
				t.Fatalf("machine = %v (%+v), want %v", v.Machine, v, tc.machine)
			}
			if want := map[bool]string{true: ClassBot, false: ClassHuman}[tc.machine]; v.Class != want {
				t.Fatalf("class = %q, want %q", v.Class, want)
			}
		})
	}
}

func TestClientName(t *testing.T) {
	cases := map[string]string{
		chromeMac: "Chrome on macOS",
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:131.0) Gecko/20100101 Firefox/131.0":                                              "Firefox on Windows",
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/129.0.0.0 Safari/537.36 Edg/129.0.0.0": "Edge on Windows",
		"Mozilla/5.0 (Macintosh; Intel Mac OS X 14_6) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/18.0 Safari/605.1.15":            "Safari on macOS",
		"Mozilla/5.0 (Linux; Android 14; Pixel 8) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/129.0.0.0 Mobile Safari/537.36":         "Chrome on Android",
		"Mozilla/5.0 (X11; Linux x86_64; rv:128.0) Gecko/20100101 Thunderbird/128.3.0":                                                  "Thunderbird on Linux",
		"curl-like/1.0":        "Unknown mail client",
		gmailPx:                "Gmail",
		"SomeClient (Windows)": "A mail client on Windows",
	}
	for ua, want := range cases {
		if got := ClientName(ua); got != want {
			t.Errorf("ClientName(%q) = %q, want %q", ua, got, want)
		}
	}
}
