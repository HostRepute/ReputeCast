package emailaddr

import "strings"

// disposable lists temporary-inbox services. Addresses there expire and
// bounce, so the importer skips them by default. The list is small on
// purpose: it covers the services that show up in real exports.
var disposable = map[string]bool{}

func init() {
	for _, d := range strings.Fields(`
		10minutemail.com 10minutemail.net 20minutemail.com 33mail.com anonbox.net anonymbox.com binkmail.com
		bobmail.info burnermail.io chacuo.net cool.fr.nf courriel.fr.nf crazymailing.com deadaddress.com
		discard.email discardmail.com dispostable.com dropmail.me e4ward.com emailondeck.com emailtemporanea.net
		fakeinbox.com fakemail.net filzmail.com getairmail.com getnada.com grr.la guerrillamail.biz guerrillamail.com
		guerrillamail.de guerrillamail.info guerrillamail.net guerrillamail.org guerrillamailblock.com
		harakirimail.com incognitomail.org inboxbear.com inboxkitten.com jetable.org kasmail.com mail-temp.com
		mail.tm mailcatch.com maildrop.cc maildu.de mailexpire.com mailforspam.com mailinator.com mailinator.net
		mailinator2.com mailmetrash.com mailnesia.com mailnull.com mailsac.com mailtemp.info mailtothis.com
		meltmail.com mintemail.com moakt.com mohmal.com mt2015.com mytemp.email mytrashmail.com nada.email
		no-spam.ws nospam.ze.tc nowmymail.com objectmail.com one-time.email oneoffemail.com pokemail.net
		proxymail.eu rcpt.at receiveee.com sharklasers.com shitmail.me smellfear.com spam4.me spamavert.com
		spambog.com spambox.us spamcero.com spamex.com spamfree24.org spamgourmet.com spamhole.com spaml.com
		spamspot.com spamthis.co.uk tempail.com tempemail.net tempinbox.com tempmail.dev tempmail.net
		tempmail.plus tempmailo.com temp-mail.io temp-mail.org tempr.email tempsky.com thankyou2010.com
		thisisnotmyrealemail.com throwam.com throwawaymail.com tmail.ws tmpmail.net tmpmail.org trash-mail.com
		trash2009.com trashmail.at trashmail.com trashmail.de trashmail.me trashmail.net trashymail.com
		tyldd.com wegwerfemail.de wegwerfmail.de wh4f.org yopmail.com yopmail.fr yopmail.net zetmail.com
	`) {
		disposable[d] = true
	}
}

// IsDisposable reports whether domain belongs to a temporary-inbox service.
func IsDisposable(domain string) bool {
	return disposable[strings.ToLower(strings.TrimSpace(domain))]
}
