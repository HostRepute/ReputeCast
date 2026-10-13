package accounts

import "testing"

func TestParseUserAgent(t *testing.T) {
	for _, c := range []struct {
		ua   string
		want Device
	}{
		{"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/141.0.0.0 Safari/537.36", Device{"Mac", "macOS", "Chrome 141"}},
		{"Mozilla/5.0 (iPhone; CPU iPhone OS 26_0 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/26.0 Mobile/15E148 Safari/604.1", Device{"iPhone", "iOS 26", "Safari 26"}},
		{"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/141.0.0.0 Safari/537.36 Edg/141.0.0.0", Device{"Windows PC", "Windows", "Edge 141"}},
		{"Mozilla/5.0 (X11; Linux x86_64; rv:143.0) Gecko/20100101 Firefox/143.0", Device{"Linux PC", "Linux", "Firefox 143"}},
		{"Mozilla/5.0 (Linux; Android 15; Pixel 9) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/141.0.0.0 Mobile Safari/537.36", Device{"Android phone", "Android 15", "Chrome 141"}},
		{"curl/8.7.1", Device{"", "", "curl"}},
	} {
		if got := ParseUserAgent(c.ua); got != c.want {
			t.Errorf("%s\n got %+v\nwant %+v", c.ua, got, c.want)
		}
	}
}

func TestDefaultPasskeyName(t *testing.T) {
	mac := "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/141.0.0.0 Safari/537.36"
	if got := defaultPasskeyName(mac, "platform", "iCloud Keychain"); got != "Mac · Touch ID" {
		t.Errorf("platform on a Mac: %q", got)
	}
	if got := defaultPasskeyName(mac, "cross-platform", "YubiKey 5"); got != "YubiKey 5" {
		t.Errorf("security key: %q", got)
	}
	if got := defaultPasskeyName("", "cross-platform", ""); got != "Security key" {
		t.Errorf("unknown key: %q", got)
	}
}
