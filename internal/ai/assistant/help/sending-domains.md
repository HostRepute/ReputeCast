# Sending domains: SPF, DKIM and DMARC
Keywords: dkim, spf, dmarc, domain, dns, authentication, sender
A sending domain needs DNS records at your DNS host. Settings › Senders lists each domain with the records to add: SPF (a TXT record that names who may send for the domain), DKIM (a public key in a TXT or CNAME record that signs each message) and DMARC (a TXT record at _dmarc that says what receivers do with mail that fails). ReputeCast checks the records and shows each as verified or missing; changes at a DNS host can take up to a day to appear.
