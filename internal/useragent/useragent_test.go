package useragent

import "testing"

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
