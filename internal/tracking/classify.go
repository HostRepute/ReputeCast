package tracking

import (
	"net/netip"
	"strings"
	"time"
)

// Signal is what a tracking request tells us about who made it.
type Signal struct {
	UserAgent string
	IP        netip.Addr
	// Method is the HTTP method; scanners often probe with HEAD.
	Method string
	// SinceSend is how long after the message was handed to the provider
	// the request came (negative when unknown).
	SinceSend time.Duration
}

// Verdict classifies an open or a click.
type Verdict struct {
	// Machine is true for opens and clicks no person made: privacy proxies
	// prefetching images, security scanners following links.
	Machine bool
	// Reason explains a machine verdict in the live feed's words
	// ("Apple Mail Privacy Protection prefetch").
	Reason string
	// Client is the mail client or browser, for human events
	// ("Chrome on macOS").
	Client string
}

var (
	// Apple's network (Mail Privacy Protection proxies fetch from it).
	appleNet = netip.MustParsePrefix("17.0.0.0/8")
	// Mail-security scanners that open and follow links on arrival:
	// Microsoft Defender for Office 365 / Exchange Online Protection,
	// Mimecast, Proofpoint and Barracuda egress ranges.
	scannerNets = []struct {
		net  netip.Prefix
		name string
	}{
		{netip.MustParsePrefix("40.92.0.0/15"), "Microsoft Defender"},
		{netip.MustParsePrefix("40.107.0.0/16"), "Microsoft Defender"},
		{netip.MustParsePrefix("52.100.0.0/14"), "Microsoft Defender"},
		{netip.MustParsePrefix("104.47.0.0/17"), "Microsoft Defender"},
		{netip.MustParsePrefix("91.220.42.0/24"), "Mimecast"},
		{netip.MustParsePrefix("207.211.30.0/24"), "Mimecast"},
		{netip.MustParsePrefix("205.139.110.0/24"), "Mimecast"},
		{netip.MustParsePrefix("148.163.128.0/19"), "Proofpoint"},
		{netip.MustParsePrefix("67.231.144.0/20"), "Proofpoint"},
		{netip.MustParsePrefix("64.235.144.0/20"), "Barracuda"},
		{netip.MustParsePrefix("209.222.80.0/21"), "Barracuda"},
	}
	// User agents of scanners, link checkers and HTTP libraries.
	scannerAgents = []struct{ needle, name string }{
		{"barracuda", "Barracuda"}, {"mimecast", "Mimecast"}, {"proofpoint", "Proofpoint"},
		{"safelinks", "Microsoft Defender"}, {"microsoft office protocol discovery", "Microsoft Defender"},
		{"trendmicro", "Trend Micro"}, {"trend micro", "Trend Micro"}, {"sophos", "Sophos"},
		{"fireeye", "FireEye"}, {"forcepoint", "Forcepoint"}, {"symantec", "Symantec"},
		{"ironport", "Cisco Secure Email"}, {"zscaler", "Zscaler"}, {"checkpoint", "Check Point"},
		{"headlesschrome", "a headless browser"}, {"phantomjs", "a headless browser"},
		{"python-requests", "a script"}, {"python-urllib", "a script"}, {"curl/", "a script"},
		{"wget/", "a script"}, {"go-http-client", "a script"}, {"okhttp", "a script"},
		{"java/", "a script"}, {"libwww", "a script"}, {"httpclient", "a script"},
		{"bot", "a bot"}, {"crawler", "a bot"}, {"spider", "a bot"}, {"slurp", "a bot"},
		{"linkexpanding", "a link preview"}, {"preview", "a link preview"},
	}
)

// Timing thresholds.
const (
	// openPrefetchWindow: image fetches through a mail provider's proxy
	// this soon after sending are caching on arrival, not someone reading.
	openPrefetchWindow = 30 * time.Second
	// tooFastOpen: nobody opens an email within two seconds of it leaving.
	tooFastOpen = 2 * time.Second
	// tooFastClick: scanners follow links within seconds of arrival.
	tooFastClick = 10 * time.Second
	// scannerWindow: scanner networks are only trusted as machines on
	// arrival; later traffic from them may be a person behind the gateway.
	scannerWindow = 2 * time.Minute
)

// ClassifyOpen decides whether an open pixel fetch was a person.
//
// Machine opens, in order:
//  1. HEAD requests and empty user agents.
//  2. Apple Mail Privacy Protection: requests from Apple's network
//     (17.0.0.0/8) or with the bare "Mozilla/5.0" user agent its proxies
//     send. They fetch every image on delivery whether or not anyone reads.
//  3. Gmail's GoogleImageProxy and Yahoo's YahooMailProxy within 30 s of
//     sending (caching on arrival); later fetches are people opening in
//     Gmail or Yahoo Mail and count as human.
//  4. Security scanners and HTTP libraries (user agent), and scanner
//     networks within two minutes of sending.
//  5. Anything within two seconds of sending.
func ClassifyOpen(s Signal) Verdict {
	ua := strings.TrimSpace(s.UserAgent)
	low := strings.ToLower(ua)
	switch {
	case strings.EqualFold(s.Method, "HEAD"):
		return Verdict{Machine: true, Reason: reasonHead}
	case ua == "":
		return Verdict{Machine: true, Reason: reasonNoAgent}
	case s.IP.IsValid() && appleNet.Contains(s.IP.Unmap()), ua == "Mozilla/5.0":
		return Verdict{Machine: true, Reason: reasonApple}
	case strings.Contains(low, "googleimageproxy"):
		if known(s) && s.SinceSend < openPrefetchWindow {
			return Verdict{Machine: true, Reason: reasonGoogle}
		}
		return Verdict{Client: "Gmail"}
	case strings.Contains(low, "yahoomailproxy"):
		if known(s) && s.SinceSend < openPrefetchWindow {
			return Verdict{Machine: true, Reason: reasonYahoo}
		}
		return Verdict{Client: "Yahoo Mail"}
	}
	if name := scannerAgent(low); name != "" {
		return Verdict{Machine: true, Reason: "Security scanner (" + name + ")"}
	}
	if name := scannerNet(s.IP); name != "" && known(s) && s.SinceSend < scannerWindow {
		return Verdict{Machine: true, Reason: "Security scanner (" + name + ")"}
	}
	if known(s) && s.SinceSend < tooFastOpen {
		return Verdict{Machine: true, Reason: reasonFastOpen}
	}
	return Verdict{Client: ClientName(ua)}
}

// ClassifyClick decides whether a click was a person. Bot clicks still get
// redirected; they are logged but never counted as clicks.
//
// Bot clicks: HEAD requests, empty or scanner user agents, scanner networks
// within two minutes of sending, clicks within ten seconds of sending, and
// bursts (burst is true when the same message already clicked several
// different links within the last few seconds, as scanners that follow
// every link do).
func ClassifyClick(s Signal, burst bool) Verdict {
	ua := strings.TrimSpace(s.UserAgent)
	low := strings.ToLower(ua)
	switch {
	case strings.EqualFold(s.Method, "HEAD"):
		return Verdict{Machine: true, Reason: reasonHead}
	case ua == "":
		return Verdict{Machine: true, Reason: reasonNoAgent}
	}
	if name := scannerAgent(low); name != "" {
		return Verdict{Machine: true, Reason: "Link scanner (" + name + ")"}
	}
	if name := scannerNet(s.IP); name != "" && known(s) && s.SinceSend < scannerWindow {
		return Verdict{Machine: true, Reason: "Link scanner (" + name + ")"}
	}
	if known(s) && s.SinceSend < tooFastClick {
		return Verdict{Machine: true, Reason: reasonFastClick}
	}
	if burst {
		return Verdict{Machine: true, Reason: reasonBurst}
	}
	return Verdict{Client: ClientName(ua)}
}

func known(s Signal) bool { return s.SinceSend >= 0 }

func scannerAgent(low string) string {
	for _, a := range scannerAgents {
		if strings.Contains(low, a.needle) {
			return a.name
		}
	}
	return ""
}

func scannerNet(ip netip.Addr) string {
	if !ip.IsValid() {
		return ""
	}
	ip = ip.Unmap()
	for _, n := range scannerNets {
		if n.net.Contains(ip) {
			return n.name
		}
	}
	return ""
}

// ClientName turns a user agent into "Chrome on macOS", "Outlook on
// Windows", "Apple Mail on iOS".
func ClientName(ua string) string {
	low := strings.ToLower(ua)
	app := ""
	switch {
	case strings.Contains(low, "googleimageproxy"):
		return "Gmail"
	case strings.Contains(low, "yahoomailproxy"):
		return "Yahoo Mail"
	case strings.Contains(low, "microsoft outlook") || strings.Contains(low, "ms-office") || strings.Contains(low, "outlook-") || strings.Contains(low, "microsoft office"):
		app = "Outlook"
	case strings.Contains(low, "thunderbird"):
		app = "Thunderbird"
	case strings.Contains(low, "edg/") || strings.Contains(low, "edge/"):
		app = "Edge"
	case strings.Contains(low, "firefox/"):
		app = "Firefox"
	case strings.Contains(low, "chrome/") || strings.Contains(low, "crios/"):
		app = "Chrome"
	case strings.Contains(low, "safari/") && strings.Contains(low, "version/"):
		app = "Safari"
	case strings.Contains(low, "applewebkit") && (strings.Contains(low, "iphone") || strings.Contains(low, "ipad") || strings.Contains(low, "macintosh")):
		app = "Apple Mail"
	}
	os := ""
	switch {
	case strings.Contains(low, "iphone") || strings.Contains(low, "ipad") || strings.Contains(low, "ipod"):
		os = "iOS"
	case strings.Contains(low, "android"):
		os = "Android"
	case strings.Contains(low, "windows"):
		os = "Windows"
	case strings.Contains(low, "mac os x") || strings.Contains(low, "macintosh"):
		os = "macOS"
	case strings.Contains(low, "cros"):
		os = "ChromeOS"
	case strings.Contains(low, "linux"):
		os = "Linux"
	}
	switch {
	case app != "" && os != "":
		return app + " on " + os
	case app != "":
		return app
	case os != "":
		return "A mail client on " + os
	}
	return clientUnknown
}
