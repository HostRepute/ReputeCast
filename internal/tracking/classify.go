package tracking

import (
	"net/netip"
	"strings"
	"time"

	"github.com/hostrepute/reputecast/internal/useragent"
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
	// SinceDelivery is how long after the message was delivered, when
	// Delivered says that is known. Scanners act when mail arrives, which
	// can be minutes after it was sent, so the windows below are measured
	// from delivery when it is known (review AN-2).
	SinceDelivery time.Duration
	Delivered     bool
	// Net is the machine network the address belongs to, from the
	// instance's list (tracking_networks); nil checks the built-in ranges.
	Net *Network
	// Agents is the instance's list of machines' user agents
	// (tracking_user_agents); nil is the built-in list.
	Agents *useragent.List
	// Reported: the sending provider reported the open or click (the
	// workspace keeps the provider's tracking, review AN-4). A provider
	// that doesn't pass the user agent on says nothing about the request,
	// so a missing one isn't a machine's sign.
	Reported bool
}

// Network is a network of machines that open and click email.
type Network struct {
	// Class is "privacy" (a privacy proxy) or "scanner".
	Class string
	Name  string
}

// since is the time the windows are measured from: delivery when known,
// else the hand-over to the provider.
func (s Signal) since() time.Duration {
	if s.Delivered {
		return max(s.SinceDelivery, 0)
	}
	return s.SinceSend
}

// Classes of opens and clicks.
const (
	// ClassHuman: a person opened or clicked.
	ClassHuman = "human"
	// ClassPrivacy: a privacy proxy (Apple Mail Privacy Protection) fetched
	// the images on delivery. Whether a person read the email can't be
	// measured: these opens are neither a person's nor a scanner's.
	ClassPrivacy = "privacy"
	// ClassMachine: a scanner or a proxy prefetching on arrival opened it.
	ClassMachine = "machine"
	// ClassBot: a click no person made (scanners following every link).
	ClassBot = "bot"
)

// Verdict classifies an open or a click.
type Verdict struct {
	// Machine is true for opens and clicks no person made: privacy proxies
	// prefetching images, security scanners following links.
	Machine bool
	// Class is ClassHuman, ClassPrivacy or ClassMachine for an open,
	// ClassHuman or ClassBot for a click.
	Class string
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
)

// Timing thresholds, measured from delivery when it is known (else from
// the hand-over to the provider).
const (
	// openPrefetchWindow: image fetches through a mail provider's proxy
	// this soon after arrival are caching, not someone reading.
	openPrefetchWindow = 30 * time.Second
	// tooFastOpen: nobody opens an email within two seconds of its arrival.
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
	machine := func(reason string) Verdict { return Verdict{Machine: true, Class: ClassMachine, Reason: reason} }
	person := func(client string) Verdict { return Verdict{Class: ClassHuman, Client: client} }
	switch {
	case strings.EqualFold(s.Method, "HEAD"):
		return machine(reasonHead)
	case ua == "" && !s.Reported:
		return machine(reasonNoAgent)
	case ua == "Mozilla/5.0":
		// Apple Mail Privacy Protection: whether anyone read it can't be
		// told, so it's neither a person's open nor a scanner's.
		return Verdict{Machine: true, Class: ClassPrivacy, Reason: reasonApple}
	}
	if class, name := netOf(s); class == netPrivacy {
		reason := reasonApple
		if name != appleName {
			reason = "Privacy proxy prefetch (" + name + ")"
		}
		return Verdict{Machine: true, Class: ClassPrivacy, Reason: reason}
	}
	switch {
	case strings.Contains(low, "googleimageproxy"):
		if known(s) && s.since() < openPrefetchWindow {
			return machine(reasonGoogle)
		}
		return person("Gmail")
	case strings.Contains(low, "yahoomailproxy"):
		if known(s) && s.since() < openPrefetchWindow {
			return machine(reasonYahoo)
		}
		return person("Yahoo Mail")
	}
	if name := scannerAgent(s, ua); name != "" {
		return machine("Security scanner (" + name + ")")
	}
	if name := scannerNet(s); name != "" && known(s) && s.since() < scannerWindow {
		return machine("Security scanner (" + name + ")")
	}
	if known(s) && s.since() < tooFastOpen {
		return machine(reasonFastOpen)
	}
	return person(ClientName(ua))
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
	bot := func(reason string) Verdict { return Verdict{Machine: true, Class: ClassBot, Reason: reason} }
	switch {
	case strings.EqualFold(s.Method, "HEAD"):
		return bot(reasonHead)
	case ua == "" && !s.Reported:
		return bot(reasonNoAgent)
	}
	if name := scannerAgent(s, ua); name != "" {
		return bot("Link scanner (" + name + ")")
	}
	if name := scannerNet(s); name != "" && known(s) && s.since() < scannerWindow {
		return bot("Link scanner (" + name + ")")
	}
	if known(s) && s.since() < tooFastClick {
		return bot(reasonFastClick)
	}
	if burst {
		return bot(reasonBurst)
	}
	return Verdict{Class: ClassHuman, Client: ClientName(ua)}
}

// ClassifyHoneypot is a click on the hidden link: no person can see it.
func ClassifyHoneypot() Verdict {
	return Verdict{Machine: true, Class: ClassBot, Reason: reasonHoneypot}
}

func known(s Signal) bool { return s.since() >= 0 }

// netOf is the machine network of the request's address: the instance's
// list when the caller looked it up, else the built-in ranges.
func netOf(s Signal) (class, name string) {
	if s.Net != nil {
		return s.Net.Class, s.Net.Name
	}
	if !s.IP.IsValid() {
		return "", ""
	}
	ip := s.IP.Unmap()
	if appleNet.Contains(ip) {
		return netPrivacy, appleName
	}
	for _, n := range scannerNets {
		if n.net.Contains(ip) {
			return netScanner, n.name
		}
	}
	return "", ""
}

// Network classes (tracking_networks.class).
const (
	netPrivacy = "privacy"
	netScanner = "scanner"
	appleName  = "Apple Mail Privacy Protection"
)

// scannerAgent names the machine a user agent belongs to, by the
// instance's list or else the built-in one (whole product tokens, so
// "CUBOT" phones aren't bots), or "".
func scannerAgent(s Signal, ua string) string { return s.Agents.Bot(ua) }

func scannerNet(s Signal) string {
	if class, name := netOf(s); class == netScanner {
		return name
	}
	return ""
}

// Mail apps and browsers, systems and devices as ClientParts reads them
// (the campaign report's breakdowns keep these keys).
const (
	AppAppleMail   = "apple_mail"
	AppGmail       = "gmail"
	AppOutlook     = "outlook"
	AppYahooMail   = "yahoo_mail"
	AppThunderbird = "thunderbird"
	AppChrome      = "chrome"
	AppSafari      = "safari"
	AppFirefox     = "firefox"
	AppEdge        = "edge"
	AppOther       = "other"

	OSiOS      = "ios"
	OSAndroid  = "android"
	OSWindows  = "windows"
	OSMacOS    = "macos"
	OSChromeOS = "chromeos"
	OSLinux    = "linux"
	OSUnknown  = "unknown"

	DeviceMobile  = "mobile"
	DeviceTablet  = "tablet"
	DeviceDesktop = "desktop"
	DeviceUnknown = "unknown"
)

// ClientInfo is a user agent's mail app or browser, system and kind of
// device, as keys.
type ClientInfo struct {
	App, OS, Device string
}

// ClientParts reads a user agent. Gmail's and Yahoo's image proxies fetch
// for the reader, so they tell the app and nothing about the system or the
// device (and their address is the proxy's): only they read as gmail and
// yahoo_mail.
func ClientParts(ua string) ClientInfo {
	low := strings.ToLower(ua)
	switch {
	case strings.Contains(low, "googleimageproxy"):
		return ClientInfo{App: AppGmail, OS: OSUnknown, Device: DeviceUnknown}
	case strings.Contains(low, "yahoomailproxy"):
		return ClientInfo{App: AppYahooMail, OS: OSUnknown, Device: DeviceUnknown}
	}
	c := ClientInfo{App: AppOther, OS: OSUnknown, Device: DeviceUnknown}
	switch {
	case strings.Contains(low, "microsoft outlook") || strings.Contains(low, "ms-office") || strings.Contains(low, "outlook-") || strings.Contains(low, "microsoft office"):
		c.App = AppOutlook
	case strings.Contains(low, "thunderbird"):
		c.App = AppThunderbird
	case strings.Contains(low, "edg/") || strings.Contains(low, "edge/"):
		c.App = AppEdge
	case strings.Contains(low, "firefox/"):
		c.App = AppFirefox
	case strings.Contains(low, "chrome/") || strings.Contains(low, "crios/"):
		c.App = AppChrome
	case strings.Contains(low, "safari/") && strings.Contains(low, "version/"):
		c.App = AppSafari
	case strings.Contains(low, "applewebkit") && (strings.Contains(low, "iphone") || strings.Contains(low, "ipad") || strings.Contains(low, "macintosh")):
		c.App = AppAppleMail
	}
	switch {
	case strings.Contains(low, "iphone") || strings.Contains(low, "ipod"):
		c.OS, c.Device = OSiOS, DeviceMobile
	case strings.Contains(low, "ipad"):
		c.OS, c.Device = OSiOS, DeviceTablet
	case strings.Contains(low, "android"):
		// Android phones say "Mobile"; tablets don't.
		c.OS, c.Device = OSAndroid, DeviceTablet
		if strings.Contains(low, "mobile") {
			c.Device = DeviceMobile
		}
	case strings.Contains(low, "windows"):
		c.OS, c.Device = OSWindows, DeviceDesktop
	case strings.Contains(low, "mac os x") || strings.Contains(low, "macintosh"):
		c.OS, c.Device = OSMacOS, DeviceDesktop
	case strings.Contains(low, "; cros"):
		// "(X11; CrOS x86_64 …)"; not the "cros" inside "Microsoft".
		c.OS, c.Device = OSChromeOS, DeviceDesktop
	case strings.Contains(low, "linux"):
		c.OS, c.Device = OSLinux, DeviceDesktop
	}
	return c
}

// AppName is an app key's product name ("" for other).
func AppName(key string) string {
	switch key {
	case AppAppleMail:
		return "Apple Mail"
	case AppGmail:
		return "Gmail"
	case AppOutlook:
		return "Outlook"
	case AppYahooMail:
		return "Yahoo Mail"
	case AppThunderbird:
		return "Thunderbird"
	case AppChrome:
		return "Chrome"
	case AppSafari:
		return "Safari"
	case AppFirefox:
		return "Firefox"
	case AppEdge:
		return "Edge"
	}
	return ""
}

// OSName is a system key's product name ("" for unknown).
func OSName(key string) string {
	switch key {
	case OSiOS:
		return "iOS"
	case OSAndroid:
		return "Android"
	case OSWindows:
		return "Windows"
	case OSMacOS:
		return "macOS"
	case OSChromeOS:
		return "ChromeOS"
	case OSLinux:
		return "Linux"
	}
	return ""
}

// ClientName turns a user agent into "Chrome on macOS", "Outlook on
// Windows", "Apple Mail on iOS".
func ClientName(ua string) string {
	c := ClientParts(ua)
	app, os := AppName(c.App), OSName(c.OS)
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
