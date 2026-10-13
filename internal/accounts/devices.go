package accounts

import (
	"regexp"
	"strings"
)

// Device describes the browser and computer a session was created from.
type Device struct {
	Device  string // "Mac", "iPhone", "Windows PC"
	OS      string // "macOS", "iOS 18", "Windows"
	Browser string // "Chrome 141", "Safari 18", "Edge 141"
}

var (
	reIOS     = regexp.MustCompile(`(?:iPhone|CPU) OS (\d+)`)
	reAndroid = regexp.MustCompile(`Android (\d+)`)
	reCrOS    = regexp.MustCompile(`CrOS`)
	reEdge    = regexp.MustCompile(`Edg(?:e|A|iOS)?/(\d+)`)
	reOpera   = regexp.MustCompile(`(?:OPR|OPiOS)/(\d+)`)
	reFirefox = regexp.MustCompile(`(?:Firefox|FxiOS)/(\d+)`)
	reChrome  = regexp.MustCompile(`(?:HeadlessChrome|Chrome|CriOS)/(\d+)`)
	reSafari  = regexp.MustCompile(`Version/(\d+)(?:\.\d+)*.*Safari/`)
	reSamsung = regexp.MustCompile(`SamsungBrowser/(\d+)`)
)

// ParseUserAgent extracts a readable device, OS and browser from a user
// agent string. It is deliberately small: the session list only needs a
// recognisable label, and browsers freeze most details in the UA anyway
// (macOS always reports 10.15, Windows 10 and 11 look the same).
func ParseUserAgent(ua string) Device {
	var d Device
	switch {
	case strings.Contains(ua, "iPhone"):
		d.Device, d.OS = "iPhone", "iOS"
		if m := reIOS.FindStringSubmatch(ua); m != nil {
			d.OS = "iOS " + m[1]
		}
	case strings.Contains(ua, "iPad"):
		d.Device, d.OS = "iPad", "iPadOS"
		if m := reIOS.FindStringSubmatch(ua); m != nil {
			d.OS = "iPadOS " + m[1]
		}
	case strings.Contains(ua, "Android"):
		d.Device, d.OS = "Android phone", "Android"
		if !strings.Contains(ua, "Mobile") {
			d.Device = "Android tablet"
		}
		if m := reAndroid.FindStringSubmatch(ua); m != nil {
			d.OS = "Android " + m[1]
		}
	case strings.Contains(ua, "Macintosh") || strings.Contains(ua, "Mac OS X"):
		d.Device, d.OS = "Mac", "macOS"
	case strings.Contains(ua, "Windows"):
		d.Device, d.OS = "Windows PC", "Windows"
	case reCrOS.MatchString(ua):
		d.Device, d.OS = "Chromebook", "ChromeOS"
	case strings.Contains(ua, "Linux"):
		d.Device, d.OS = "Linux PC", "Linux"
	}
	switch {
	case reEdge.MatchString(ua):
		d.Browser = "Edge " + reEdge.FindStringSubmatch(ua)[1]
	case reOpera.MatchString(ua):
		d.Browser = "Opera " + reOpera.FindStringSubmatch(ua)[1]
	case reSamsung.MatchString(ua):
		d.Browser = "Samsung Internet " + reSamsung.FindStringSubmatch(ua)[1]
	case reFirefox.MatchString(ua):
		d.Browser = "Firefox " + reFirefox.FindStringSubmatch(ua)[1]
	case reChrome.MatchString(ua):
		d.Browser = "Chrome " + reChrome.FindStringSubmatch(ua)[1]
	case reSafari.MatchString(ua):
		d.Browser = "Safari " + reSafari.FindStringSubmatch(ua)[1]
	case strings.Contains(ua, "Safari"):
		d.Browser = "Safari"
	case ua != "":
		name, _, _ := strings.Cut(ua, "/")
		if len(name) > 40 {
			name = name[:40]
		}
		d.Browser = name
	}
	return d
}
