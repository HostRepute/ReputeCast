package importer

import (
	"net/url"
	"strings"
)

// An embedded player or card can't run in a post (the site serves no
// frames or scripts) or in an email: it becomes a link to it, worded by
// what readers do there.
type platform struct {
	Name string
	// Verb picks the link's words: watch, listen, view or read.
	Verb string
}

// platforms by host (without "www.").
var platforms = map[string]platform{
	"youtube.com": {"YouTube", "watch"}, "m.youtube.com": {"YouTube", "watch"}, "youtube-nocookie.com": {"YouTube", "watch"}, "youtu.be": {"YouTube", "watch"},
	"vimeo.com": {"Vimeo", "watch"}, "player.vimeo.com": {"Vimeo", "watch"},
	"tiktok.com": {"TikTok", "watch"}, "loom.com": {"Loom", "watch"},
	"open.spotify.com": {"Spotify", "listen"}, "soundcloud.com": {"SoundCloud", "listen"}, "w.soundcloud.com": {"SoundCloud", "listen"},
	"podcasts.apple.com": {"Apple Podcasts", "listen"}, "embed.podcasts.apple.com": {"Apple Podcasts", "listen"},
	"twitter.com": {"X", "view"}, "mobile.twitter.com": {"X", "view"}, "x.com": {"X", "view"}, "platform.twitter.com": {"X", "view"},
	"instagram.com": {"Instagram", "view"}, "bsky.app": {"Bluesky", "view"}, "facebook.com": {"Facebook", "view"},
	"linkedin.com": {"LinkedIn", "view"}, "codepen.io": {"CodePen", "view"}, "gist.github.com": {"GitHub", "view"},
	"substack.com": {"Substack", "read"},
}

func hostOf(u *url.URL) string { return strings.TrimPrefix(strings.ToLower(u.Hostname()), "www.") }

// platformOf names the platform of an address; ok is false for a site
// that isn't a known one (it is named by its host).
func platformOf(raw string) (platform, bool) {
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" {
		return platform{}, false
	}
	h := hostOf(u)
	if p, ok := platforms[h]; ok {
		return p, true
	}
	if strings.HasSuffix(h, ".substack.com") {
		return platforms["substack.com"], true
	}
	return platform{Name: h, Verb: "view"}, false
}

// embedLink is where readers follow an embed to: the platform's page, not
// its player ("youtube.com/embed/ID" → "youtube.com/watch?v=ID").
func embedLink(u *url.URL) string {
	h := hostOf(u)
	seg := strings.Split(strings.Trim(u.Path, "/"), "/")
	switch {
	case (h == "youtube.com" || h == "youtube-nocookie.com" || h == "m.youtube.com") && len(seg) >= 2 && seg[0] == "embed" && seg[1] != "":
		return "https://www.youtube.com/watch?v=" + url.QueryEscape(seg[1])
	case h == "player.vimeo.com" && len(seg) >= 2 && seg[0] == "video":
		return "https://vimeo.com/" + url.PathEscape(seg[1])
	case h == "open.spotify.com" && len(seg) >= 3 && seg[0] == "embed":
		return "https://open.spotify.com/" + strings.Join(seg[1:], "/")
	case h == "w.soundcloud.com":
		if v := u.Query().Get("url"); strings.HasPrefix(v, "https://") {
			return v
		}
	case h == "embed.podcasts.apple.com":
		c := *u
		c.Host, c.RawQuery = "podcasts.apple.com", ""
		return c.String()
	case h == "instagram.com" && len(seg) >= 2 && (seg[0] == "p" || seg[0] == "reel"):
		return "https://www.instagram.com/" + seg[0] + "/" + url.PathEscape(seg[1]) + "/"
	case h == "loom.com" && len(seg) >= 2 && seg[0] == "embed":
		return "https://www.loom.com/share/" + url.PathEscape(seg[1])
	}
	c := *u
	c.Fragment = ""
	return c.String()
}
