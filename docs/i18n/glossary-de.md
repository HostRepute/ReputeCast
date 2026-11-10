# German glossary (Deutsch)

The words ReputeCast uses in German, so every screen, email and error says
the same thing the same way. It follows the Vietnamese glossary's sections
term for term, and the German that email-marketing tools (Mailchimp, Brevo,
CleverReach, HubSpot) and the services ReputeCast connects to (Google,
Microsoft, AWS, Stripe, Shopify, WordPress) already use in their German
interfaces, so a German marketer meets the words they know.

> **Status: draft.** A native speaker reviews these words before German
> ships in a release (brief §8.17). Change a term here first, then in the
> translations (`web/src/**/messages.de.ts`,
> `internal/<package>/messages/de.json`).

How to add or complete a language: [`translating.md`](translating.md).

## Voice

- Plain and specific, like the English: say what happened and what to do.
  No exclamation marks.
- Standard German as written in Germany, readable in Austria and
  Switzerland: "ß" as in Germany ("Größe", "schließen"), no regional words
  ("Januar", not "Jänner").
- Address the reader formally as **Sie**, with a capital in every form:
  Sie, Ihr, Ihre, Ihnen ("Fragen Sie Ihren Admin."). Never "du". Watch
  the lower-case "sie" (they, it): it must not be read as the reader.
  "Bitte" is rare: an instruction with "Sie" is polite enough ("Geben Sie
  einen Namen ein."), and buttons never carry it.
- Buttons, menu items and links that do something are infinitives, with
  the verb last: "Speichern", "Kampagne senden", "Kontakt hinzufügen",
  "<link>Einstellungen öffnen</link>". Never an imperative with "Sie" on a
  button ("Senden Sie die Kampagne").
- Sentence case. German nouns keep their capital; otherwise only the first
  word and names start with a capital: "Neue Kampagne", "Letzte
  Aktivität", not "Neue Kampagne Erstellen".
- What went wrong says what didn't happen: "Couldn't save the campaign" is
  "Die Kampagne konnte nicht gespeichert werden."; a toast may drop the
  article ("Kampagne nicht gespeichert"). Progress ends in "…" with no
  space: "Wird gespeichert…".
- Quotation marks are German: „…“ (U+201E, U+201C), and ‚…‘ inside them.
  A campaign's name in a sentence goes between „ “: „October Product
  Update“ wurde automatisch pausiert.
- Neutral wording for people: "Person", "Kontakt", "Mitglied",
  "Teammitglied", or a sentence with "wer" ("Wer {language} liest, …").
  Never a gender star, colon, capital I or slash (Nutzer*innen,
  Nutzer:innen, NutzerInnen, Nutzer/-innen), and no participles made into
  nouns (Abonnierende, Lesende). Role nouns that are the established terms
  of the trade stay as they are: Absender, Empfänger, Abonnent, Besucher,
  Inhaber.
- Plurals use `one` and `other`:
  `{count, plural, one {# Kampagne} other {# Kampagnen}}`. `one` is only
  the number 1 ("21 Kampagnen"). `=1 {…}` matches 1 exactly where the
  wording changes ("eine Kampagne").
- Numbers and dates come from the region format, never typed by hand:
  187.312 · 99,2 % · 18,21 $ · Di., 14. Okt. 2026 · 09:00. "at {time}" is
  "um {time}" and "on {date}" is "am {date}"; never add "Uhr" after
  `{time}`, which may be a 12-hour time ("9:00 AM").
- Relative times: "gerade eben", "vor 4 Minuten", "vor 2 Stunden",
  "gestern", "in 5 Tagen". In tight places "Min." and "Std." ("vor 4
  Min."), as the region format abbreviates them.
- Overlines (table headers, group labels) are written like any label
  ("Empfänger", "Letzte Aktivität"); the app sets them in capitals:
  "EMPFÄNGER", "LETZTE AKTIVITÄT". In capitals "ß" becomes "SS"
  ("GRÖSSE"), as CSS does; where you type capitals yourself ("Match ALL"),
  write "SS" too, never "ẞ".
- "&" stays in labels where the English has it ("Konto & Sicherheit",
  "Prüfen & senden"); sentences say "und". Ranges take an en dash: "0–10",
  "Mo–Fr".
- English words that stay are German nouns: they start with a capital and
  join German words with a hyphen ("Bounce-Rate", "Webhook-Endpunkt",
  "Opt-in-Bestätigung"). Always "E-Mail" (die E-Mail, die E-Mails), never
  "Email", "eMail" or "Mail" on its own.

### Length

German runs 30–40% longer than English and is the longest of ReputeCast's
languages. Screens make room as `docs/dev/CONVENTIONS.md` ›
[Text expansion](../dev/CONVENTIONS.md#text-expansion) describes, up to the
pseudo-locale's +40%; staying inside that is the translator's job.

- Of two correct words, take the shorter: "Versandanbieter", not
  "Versanddienstleister"; "Limit", not "Obergrenze"; "Rolle ändern", not
  "Die Rolle ändern".
- Tabs, buttons, table headers, badges, stat tiles and the sidebar take the
  short variant a table gives ("short: …"). Drop articles there and prefer
  a noun ("Speichern fehlgeschlagen") to a sentence.
- Buttons and badges never wrap. Table headers and stat-tile labels take at
  most two lines and break only at a space or after a hyphen: one long word
  stays on one line and widens its column. In a header, two words ("Letzte
  Aktivität") or a hyphenated compound ("Bounce-Rate") beat a long closed
  compound.
- Compounds up to about 20 letters are written closed, as Duden writes them
  ("Absenderreputation", "Sperrliste"). A longer one is better as a phrase
  that can wrap ("Einstellungen des Arbeitsbereichs") or a shorter word.
- A compound with an English word, an abbreviation, a number or a name is
  hyphenated throughout: "E-Mail-Adresse", "Double-Opt-in-Bestätigung",
  "Hard-Bounce-Rate", "DKIM-Eintrag", "A/B-Test", "2FA-Code",
  "SES-Anteil", "Warm-up-Plan". It can break after each hyphen, which
  helps a narrow layout.
- Look at German at 1440 and 390 px, on the stress-test screens
  (Campaigns, Review & send, Settings › Team). Text that spills or is cut
  off is a bug: shorten the word, don't squeeze the layout.

### Language names

The app passes `{language}` and `{languages}` as names in the app language:
"Englisch", "Spanisch", "Deutsch (Deutschland)", or a list ("Spanisch und
Deutsch"). German language names are nouns, so they start with a capital
everywhere and nothing changes inside a sentence. Write `{language}` where
the sentence needs it and never type a language name around it.

A German name can't be declined or joined in an argument, and it may be a
list, so build the sentence around the name: never "{language}-Version",
"{language}e", "ins {language}e", "aus dem {language}en" or
"{language}sprachig". "In {language}" is "auf {language}".

| English | Deutsch |
|---|---|
| Add {language} | {language} hinzufügen |
| {language} version · No {language} version | Version auf {language} · Keine Version auf {language} |
| Add a {language} version to {campaign} | Version auf {language} zu {campaign} hinzufügen |
| Written in {language} · Alt text in {language} | Verfasst auf {language} · Alternativtext auf {language} |
| Language: {language} · Composer · {language} | Sprache: {language} · Editor · {language} |
| {language} readers get {fallback} | Kontakte mit {language} erhalten die Version auf {fallback}; in a longer sentence: Wer {language} liest, erhält die Version auf {fallback}; in a badge: erhalten {fallback} |
| {language} visitors | Besucher mit {language} |
| translated from {language} | Ausgangssprache: {language}; in a sentence: übersetzt aus der Version auf {language} |
| Mark {language} as translated | {language} als übersetzt markieren |
| {languages} is missing · {languages} are missing | `{count, plural, one {{languages} fehlt} other {{languages} fehlen}}` |
| Instance default ({language}) | Standard der Instanz ({language}) |

## Never translated

Workspace content (campaign, list, segment, template, form and custom role
names such as "Content Writer"), people's names and email addresses,
permission keys (`campaigns:send`), audit action names, API error codes,
provider names and codes (Amazon SES, Postal, Postmark, SES), product names
(ReputeCast, HostRepute, Gmail, Outlook), DNS records and record types (SPF,
DKIM, DMARC, BIMI, CNAME, MAIL FROM), headers (List-Unsubscribe), code
samples, CLI output, "MCP", "2FA", "TOTP", "IdP", "SSO", "JIT", "A/B",
"HTML", "CSS", "API", "Webhook".

What stays in English is a German noun all the same, with one gender for
everyone: die API, der Webhook (die Webhooks), das Token, die Payload, der
Header, das Tag (die Tags), der Link, der Button, der Bounce,
das Opt-in, das Tracking, das Routing, der Stream, die Sandbox, der Feed,
das Dashboard, der Prompt, das Tool, der Agent (des Agenten), das Snippet,
das Warm-up, der Score, die Domain.

## App shell and navigation

| English | Deutsch |
|---|---|
| Overview | Übersicht |
| Campaigns | Kampagnen |
| Automations | Automationen |
| Transactional | Transaktional |
| Audience | Zielgruppe |
| Grow | Wachstum |
| Earn | Einnahmen |
| Forms & pages | Formulare & Seiten |
| Deliverability | Zustellbarkeit |
| Sending providers | Versandanbieter |
| Settings | Einstellungen |
| More | Mehr |
| Skip to content | Zum Inhalt springen |
| Import contacts | Kontakte importieren |
| HostRepute connected | HostRepute verbunden |
| synced 4 min ago | vor 4 Min. synchronisiert |
| Account & security | Konto & Sicherheit |
| Notification preferences | Benachrichtigungseinstellungen (short, in the user menu: Benachrichtigungen) |
| Workspace settings | Einstellungen des Arbeitsbereichs |
| Instance admin | Instanzverwaltung |
| admins only | nur für Admins |
| Sign out | Abmelden |
| Language: English | Sprache: Deutsch (the item names the current language in its own name) |
| unexpected error · server log | unerwarteter Fehler · Serverprotokoll |

The Audience section is the **Zielgruppe**, as in Mailchimp's German
interface: "Empfänger" are a send's recipients, and "Kontakte" is the
section's first tab. A sending provider is a **Versandanbieter**, the
shorter of the two German words. Grow is **Wachstum**; Audience › Growth is
**Zuwachs**, so the two don't read alike.

## Settings tabs

| English | Deutsch |
|---|---|
| General | Allgemein |
| Senders | Absender |
| Providers | Anbieter |
| Team | Team |
| Roles | Rollen |
| Approvals | Freigaben |
| Keys & MCP | Schlüssel & MCP |
| Webhooks | Webhooks |
| Integrations | Integrationen |
| Audit log | Audit-Protokoll |

## Roles and people

| English | Deutsch |
|---|---|
| Owner | Inhaber |
| Admin | Admin |
| Editor | Bearbeiter |
| Viewer | Betrachter |
| custom (role) | eigene ("Eigene Rolle") |
| instance admin (the area, a label: "Instance admin", "· instance admin") | Instanzverwaltung |
| instance admin (a person, in a sentence: "Ask your instance admin") | Instanz-Admin ("Fragen Sie einen Instanz-Admin") |
| break-glass account | Notfallkonto |
| member | Mitglied (Teammitglied where a reader with an account could be meant: see Earn) |
| invitee | eingeladene Person |
| workspace | Arbeitsbereich |
| instance | Instanz |
| team | Team |

## Statuses

| English | Deutsch |
|---|---|
| Draft | Entwurf |
| Draft · blocked | Entwurf · blockiert |
| Scheduled | Geplant |
| Sending | Im Versand |
| Sent | Gesendet |
| Active | Aktiv |
| Invited | Eingeladen |
| Deactivated | Deaktiviert |
| On · Off | An · Aus |
| Via IdP | Über IdP |
| Required | Erforderlich |
| Pass · Warn · Block | Bestanden · Warnung · Blockiert |
| Ready · Ready with warnings · Blocked | Bereit · Bereit mit Warnungen · Blockiert |
| Healthy | Fehlerfrei |
| Warming up | Im Warm-up |
| New | Neu |
| Not saved | Nicht gespeichert |
| Development | Entwicklung |
| Auto-paused | Automatisch pausiert (badge: Auto-Pause) |
| Pending approval · Approved | Freigabe ausstehend · Freigegeben |
| Cancelled · Failed | Abgebrochen · Fehlgeschlagen |

## Campaigns

| English | Deutsch |
|---|---|
| Campaign | Kampagne |
| New campaign | Neue Kampagne |
| All · Drafts · Scheduled · Sending · Sent | Alle · Entwürfe · Geplant · Im Versand · Gesendet |
| Search campaigns | Kampagnen durchsuchen |
| Sent · last 30 days | Gesendet · letzte 30 Tage |
| Cost · last 30 days | Kosten · letzte 30 Tage |
| Recipients | Empfänger |
| Delivered | Zugestellt |
| Human opens | Menschliche Öffnungen (short: Öffnungen) |
| Clicks | Klicks |
| click-to-open | Klicks pro Öffnung |
| bounces · complaints | Bounces · Beschwerden |
| machine opens shown separately | maschinelle Öffnungen separat ausgewiesen |
| Readiness | Bereitschaft (the column; the checks are the "Versandbereitschaft") |
| 3 blocking issues | 3 blockierende Probleme |
| Edit · Report · Duplicate | Bearbeiten · Bericht · Duplizieren |
| not set | nicht festgelegt |
| created 08:55 | erstellt um 08:55 |
| edited 09:31 by Linh Nguyen | bearbeitet von Linh Nguyen um 09:31 |
| All subscribers | Alle Abonnenten |
| Broadcasts route | Route Massenversand |
| A/B subject, winner B | A/B-Betreff, Gewinner: B |
| auto-paused once, Yahoo deferrals | einmal automatisch pausiert, Zurückstellungen bei Yahoo |
| Showing 9 of 48 | 9 von 48 |
| Previous · Next | Vorherige · Nächste |
| Live monitor | Live-Monitor |
| imported from Brevo · Brevo · imported | importiert aus Brevo · Brevo · importiert |
| segment (in an audience: 2 segments) | Segment (2 Segmente) |

## Review & send

| English | Deutsch |
|---|---|
| Review & send | Prüfen & senden |
| Back to editor | Zurück zum Editor |
| Run checks again | Erneut prüfen |
| Draft · autosaved 2 min ago | Entwurf · vor 2 Min. automatisch gespeichert |
| Fix both warnings | Beide Warnungen beheben |
| Schedule send to 31,604 recipients | Versand an 31.604 Empfänger planen |
| 27 checks passed, 2 warnings, 0 blocks | 27 Prüfungen bestanden, 2 Warnungen, 0 blockiert |
| Sender reputation | Absenderreputation |
| blocklist | Blockliste |
| suppression list (shared) | Sperrliste (gemeinsam) |
| role addresses | Rollenadressen |
| Duplicates | Duplikate |
| Final recipients | Endgültige Empfänger |
| Route split | Routenaufteilung |
| Content · Links · Compliance | Inhalt · Links · Compliance |
| Audience quality | Qualität der Zielgruppe |
| Routing · stream | Routing · Stream |
| Broadcasts stream | Stream Massenversand |
| Send time · Send now · Schedule | Versandzeit · Jetzt senden · Planen |
| Best time per recipient | Beste Zeit je Empfänger |
| Test email · Send test | Test-E-Mail · Test senden |
| Auto-pause thresholds | Schwellenwerte für Auto-Pause |
| Bounce rate · Complaint rate | Bounce-Rate · Beschwerderate |
| per mailbox provider | je Postfachanbieter |
| warm-up (day 9 of 21) | Warm-up (Tag 9 von 21) |
| 14 /s | 14/s |
| Cost · Time | Kosten · Dauer |

### Send Readiness's checks

The findings the server writes for each check (`internal/readiness`).

| English | Deutsch | Note |
|---|---|---|
| check (a row) · passed · not checked · timed out | Prüfung · bestanden · nicht geprüft · Zeitüberschreitung | |
| the verdict waits for this one | das Ergebnis wartet auf diese Prüfung | |
| I understand … (a warning's confirmation) | Mir ist bewusst: … | as the web's "Mir ist bewusst: {title}." |
| unengaged (contacts) | ohne Interaktion | no open or click in the sunset rule's days |
| Sunset tier · At risk (names) | Stufe Auslaufend · Gefährdet | tier names keep their capitals |
| spam trigger · spam heuristic | Spam-Trigger | "trips 3 spam heuristics": "löst 3 Spam-Trigger aus" |
| all caps · repeated punctuation · urgency phrasing · money words · click-bait | nur Großbuchstaben · wiederholte Satzzeichen · Dringlichkeitsfloskeln · Geldbegriffe · Clickbait | |
| image-to-text ratio | Bild-Text-Verhältnis | |
| Gmail's 102 KB clipping limit · clips | Kürzungsgrenze von Gmail (102 KB) · kürzt | Gmail's German says „Nachricht gekürzt“ |
| inline(d) CSS · minify | Inline-CSS · minifizieren | |
| one-click unsubscribe (headers) | Abmeldung mit einem Klick (Header) | "Header" stays, as in German mail docs |
| postal address | Postanschrift | |
| bulk-sender rules | Regeln für Massenabsender | Gmail and Yahoo's; "Massenabsender" is Google's German word |
| URL shortener · redirects through | URL-Kürzer · leitet weiter über | |
| responds · doesn't respond | erreichbar · nicht erreichbar | a link |
| review thresholds · pause level (SES) | Prüfschwellen · Pausierungsschwelle | |
| no enforcement | keine Maßnahmen | an account's standing |
| terms allow bulk marketing | Bedingungen erlauben Massenmarketing | |
| production access | Produktionszugriff | as in Sending; AWS's German word |
| double-tracked · rewrites links | doppelt getrackt · schreibt Links um | |
| aligned (DMARC, SPF) | ausgerichtet | |
| Easy DKIM CNAME · selector | Easy-DKIM-CNAME · Selektor | names and record types stay; "DKIM-Selektor" as German hosts write it |
| custom MAIL FROM | eigene MAIL-FROM-Domain | |
| Conditional content · conditional block · version | Bedingte Inhalte · bedingter Block · Version | |
| Visibility rule | Sichtbarkeitsregel | |
| poll · survey · answer page | Abstimmung · Umfrage · Antwortseite | |
| risky · catch-all · disposable (addresses) | riskant · Catch-all · Wegwerfadresse | |
| inactive (contacts) | inaktiv | At risk and Sunset together |
| paid members · trial · complimentary · gift (access) | zahlende Mitglieder · Testphase · Gratiszugang · Geschenk | as Earn has it |
| Sponsor block · creative · booking · Sponsored (label) | Sponsor-Block · Werbemittel · Buchung · Anzeige | |
| Upgrade block · offer | Upgrade-Block · Angebot | |
| page copy · preference center | Seitentext · Präferenzcenter | |
| bare URL · skipped heading level | ausgeschriebene URL · übersprungene Überschriftenebene | accessibility |
| linked image | verlinktes Bild | |
| self-hosted (pricing) · plan | selbst gehostet · je nach Tarif | a provider's price |

## Sending and deliverability

Sending providers & routing, the connect wizard, Settings › Senders and
Providers, Deliverability, HostRepute and the live monitor. In alphabetical
order of the English.

| English | Deutsch | Note |
|---|---|---|
| account health (a provider's) | Kontozustand | |
| adapter · community adapter · adapter registry | Adapter · Community-Adapter · Adapter-Verzeichnis | |
| aggregate report (DMARC) | Sammelbericht | |
| alignment (DMARC) | Ausrichtung | "DMARC-Ausrichtung" |
| asset (a domain or IP HostRepute monitors) | Asset | |
| backfill (statuses from an API) | nachladen | "Status über die Mailgun-API nachladen" |
| blocklist · listed · clean | Blockliste · gelistet · nicht gelistet | never "Sperrliste", which is the suppression list |
| bounce mailbox | Bounce-Postfach | SMTP providers |
| bounce · hard bounce · soft bounce | Bounce · Hard Bounce · Soft Bounce | the rate is "Bounce-Rate"; in compounds "Hard-Bounce-Rate" |
| Broadcasts · Automations · System · Transactional (streams) | Massenversand · Automationen · System · Transaktional | names: they follow the noun, as German interfaces name a menu: "Stream System", "Route Massenversand" |
| bulk marketing | Massenmarketing | a provider's terms |
| cap (daily, today's) | Limit | "Limit heute" |
| catalog (of providers) | Katalog | the wizard's first step |
| condition (routing) | Bedingung | |
| credentials (a provider's) | Zugangsdaten | |
| dedicated IP · shared pool | dedizierte IP · gemeinsamer Pool | |
| defer · deferral | zurückstellen · Zurückstellung | "Yahoo stellt Zustellungen zurück" |
| degraded · down | beeinträchtigt · ausgefallen | provider statuses |
| delisting | Delisting | "Delisting beantragen" |
| event log · raw event · normalized event | Ereignisprotokoll · Rohereignis · normalisiertes Ereignis | |
| failover · fallback · primary | Failover · Ersatz- · Haupt- | "Ersatzanbieter", "Hauptanbieter"; "failed over to" is "umgeschaltet auf" |
| feedback loop | Feedback-Loop | |
| flow (an automation's, in sending text) | Automationsablauf | "Stream" is a stream: "E-Mails aus Automationsabläufen"; on the Automations screens a flow is "Ablauf" |
| guard (Warm-up page) | Schutzgrenze | "Bounce-Rate · Schutzgrenze 2 %" |
| held (recipients) | zurückgehalten | |
| incident · outage | Störung · Ausfall | |
| live sending monitor | Live-Versandmonitor | "Live monitor" is "Live-Monitor" |
| maintainer · unmaintained (adapter) | Maintainer · nicht mehr gepflegt | |
| mapping (webhook, JSONPath) | Zuordnung | |
| message id | Nachrichten-ID | |
| monitoring · monitored | Überwachung · überwacht | HostRepute |
| nameserver | Nameserver | |
| polling (instead of webhooks) | Abfrage | "fragt die Event-API jede Minute ab" |
| Postmaster compliance status | Compliance-Status | Google Postmaster Tools v2 has no domain or IP reputation |
| production access (SES) | Produktionszugriff | the SES sandbox stays "Sandbox" |
| provisioning | Einrichtung | SNS topics and configuration sets |
| pulled in · pushed out · two-way (suppression sync) | abgerufen · übertragen · beidseitig | |
| quota | Kontingent | |
| read-only link (a live send) | Ansichtslink | |
| recommended (an option, a badge) | empfohlen | "IMAP (empfohlen)"; a recommendation's heading is "Empfehlung" |
| region (AWS) | Region | AWS's own word; not the region format ("Regionsformat") |
| reputation · domain reputation · IP reputation | Reputation · Domain-Reputation · IP-Reputation | |
| reroute | umleiten | |
| reverse DNS | Reverse DNS | |
| route tester | Routentester | |
| route · routing · weighted split | Route · Routing · gewichtete Aufteilung | |
| secret (webhook signing) | Signaturschlüssel | "Signaturschlüssel für Webhooks" |
| seed list | Testliste | one address of yours at each mailbox provider |
| self-test (of an endpoint) | Selbsttest | |
| Send Readiness | Versandbereitschaft | |
| share (of a split · of a send) | Anteil | |
| slowdown (warm-up) | Verlangsamung | |
| SNDS filter result: Green · Yellow · Red | Grün · Gelb · Rot | |
| spam rate · spam trap | Spamrate · Spamfalle | |
| spam trap hit | Spamfallen-Treffer | |
| stale (data) · cached results | veraltet · zwischengespeicherte Ergebnisse | |
| stream | Stream | der Stream, die Streams |
| suppression sync | Sperrlisten-Synchronisierung | |
| throttle · throttling | drosseln · Drosselung | |
| tracking · tracking domain | Tracking · Tracking-Domain | |
| verdict (HostRepute): Good · Fair · Poor · Bad | Gut · Mäßig · Schwach · Schlecht | |
| warm-up · warmed up · warm-up guard · ramp | Warm-up · aufgewärmt · Warm-up-Schutz · Steigerung | |
| webhook endpoint · registration | Webhook-Endpunkt · Registrierung | |
| write-only (a key) | nicht auslesbar | "kann ersetzt, aber nie wieder angezeigt werden" |
| zone file | Zonendatei | |

"Sandbox", "configuration set", "SNS topic", "IAM", "ARN", "CloudFormation",
"stack", "return-path", "DSN", "ARF", "rua", "JSONPath", "manifest.json",
"SDK", "Tunnel", "Reverse Proxy", "Token", "Postmaster", "SNDS" and "JMRP"
stay as they are ("SNS-Topic", "Configuration Set" in a German sentence), and
so do the AWS field names ("Access key ID", "Secret access key", "RoleArn"),
the SES region names, a provider's own stream names (Postmark's "Broadcast"
stream: "Stream Broadcast") and the results in DNS records and reports
("pass", "fail", "p=quarantine"). An endpoint is an "Endpunkt", and
idempotency "Idempotenz".

### What adapters and the connect flow write

The server's words about a provider: the catalog's cards and chips, an
adapter's fields and terms, the account's standing, the stepper and the
test's timeline.

| English | Deutsch | Note |
|---|---|---|
| Production · Sandbox (an account's standing) | Produktion · Sandbox | "Produktion · keine Einschränkungen" |
| Two-way · Pull only · Push only · Local only (suppression sync) | Beidseitig · Nur abrufen · Nur übertragen · Nur lokal | the Adapters tab |
| self-hosted | selbst gehostet | "Postal (selbst gehostet)" |
| Any SMTP server · Generic HTTP adapter | Beliebiger SMTP-Server · Generischer HTTP-Adapter | catalog names |
| Batch 50 · Quota discovery · Sandbox detection (chips) | Batch 50 · Kontingent-Erkennung · Sandbox-Erkennung | |
| Dedicated IPs, paid · Pro plan | Dedizierte IPs, kostenpflichtig · Tarif Pro | the plan's name stays |
| read-only call | reiner Leseaufruf | "mit einem reinen Leseaufruf geprüft" |
| Round trip 2.4 s | Hin und zurück 2,4 s | the test send |
| Mailpit (development) | Mailpit (Entwicklung) | |
| a price in an adapter's words | 0,10 $ pro 1.000 E-Mails | decimal comma, dot for thousands, $ after the amount with a space, as de-DE writes it |

A provider's names for its keys, plans, menus and stream types stay as the
provider writes them ("Server API token", "Private API key", "Account SID",
"Auth token", "Full access", "Broadcasts API", "Scale plan"), and so do its
event names ("Delivery", "SpamComplaint") and its error codes.

### What the sending engine, Deliverability and HostRepute write

The live monitor's incident panel (likely causes, recommendations and the
ways to continue), its notifications and feed, Deliverability's account
health and check runs, the HostRepute connection, route notes on a
contact's timeline and why a transactional message failed.

| English | Deutsch | Note |
|---|---|---|
| "October Product Update" auto-paused · Outlook | „October Product Update“ automatisch pausiert · Outlook | a campaign's name between „ “ |
| built-in daily check (blocklists) | integrierte tägliche Prüfung | without HostRepute |
| check run · lookups that didn't answer | Prüflauf · unbeantwortete Abfragen | |
| connection errors · throttling errors · API errors | Verbindungsfehler · Drosselungsfehler · API-Fehler | "Postal meldet seit 3 Minuten Verbindungsfehler" |
| decision requested · waiting for a decision | Entscheidung angefragt · wartet auf Ihre Entscheidung | a held share |
| health check · still failing · answering again | Zustandsprüfung · weiterhin fehlerhaft · antwortet wieder | the outage log's lines |
| never-engaged · lapsed (contacts) | nie interagiert · nicht mehr aktiv | |
| placement ("protects placement") | Platzierung | "schützt die Platzierung im Posteingang"; as in Placement tests |
| Reputation dashboard (SES) | Reputations-Dashboard | AWS's German name |
| Resume as is · Resume with changes · Cancel the Outlook share · Retry Postal · Move 1,204 to SES | Unverändert fortsetzen · Mit Änderungen fortsetzen · Outlook-Anteil abbrechen · Postal erneut versuchen · 1.204 zu SES verschieben | the incident panel's ways to continue; "(empfohlen)" after the recommended one |
| route note: SES share of the 90/10 split · by rule · instead of · failed over to | SES-Anteil der Aufteilung 90/10 · per Regel · statt · umgeschaltet auf | |
| scope (a HostRepute token's) | Scope | the scopes' keys stay: monitor:read |
| Send held (a scheduled send) | Versand zurückgehalten | |
| status page (a provider's) | Statusseite | |
| web process (a self-hosted provider's) | Web-Prozess | |

The providers' own replies stay as they come: SMTP codes and texts
("421 4.7.0 TSS04"), API errors, the outage log's error lines
("ECONNREFUSED mta.acme.io:443") and a bounce's diagnostic.

## Transactional templates

| English | Deutsch |
|---|---|
| transactional template · transactional email | Transaktionsvorlage · Transaktions-E-Mail |
| draft · publish · v3 live | Entwurf · veröffentlichen · v3 live |
| version · Publish again | Version · Erneut veröffentlichen |
| variable · declare · required · optional | Variable · deklarieren · erforderlich · optional |
| Sensitive | Vertraulich |
| sample value · default | Beispielwert · Standardwert |
| test data (set) | Testdaten(satz) |
| render · test renders | rendern · Test-Renderings |
| Checks · Ready to publish | Prüfungen · Bereit zum Veröffentlichen |
| breaking change | inkompatible Änderung |
| Subject · Preheader | Betreff · Vorschautext |
| canvas | Arbeitsfläche |
| Visual · Text (views) | Visuell · Text |
| plain text | Nur-Text |
| call (an API call) | Aufruf |
| Essential · Optional (categories) | Notwendig · Optional |
| autosaved 09:41 · conflict | um 09:41 automatisch gespeichert · Konflikt |
| Import templates · Needs a manual fix | Vorlagen importieren · Manuelle Korrektur nötig |

"Payload", "Liquid", "MJML", "Mustachio", "Handlebars", "cURL" and "JSON"
stay as they are.

## Composer

| English | Deutsch |
|---|---|
| A/B test · variant | A/B-Test · Variante |
| block · Blocks panel | Block · Blöcke |
| code view | Codeansicht |
| column · 2 columns | Spalte · 2 Spalten |
| compile · compiled HTML | kompilieren · kompiliertes HTML |
| composer | Editor („Editor verlassen?“) |
| dark mode · light | Dunkelmodus · hell |
| footer · header (of an email) | Fußzeile · Kopfzeile |
| Inspector | Eigenschaften |
| layer · layer tree | Ebene · Ebenenbaum |
| padding · border radius | Innenabstand · Eckenradius |
| personalization token · Liquid tag | Platzhalter · Liquid-Tag |
| raw HTML | HTML-Code |
| sample · winner (A/B) | Testgruppe · Gewinner |
| section (of an email) · saved section | Abschnitt · gespeicherter Abschnitt |
| Send Readiness | Versandbereitschaft (the checks before a send, as a name; "runs again" = „wird erneut geprüft“) |
| spam heuristic | Spam-Trigger |
| bare URL · link text · says nothing (a link's text) | ausgeschriebene URL · Linktext · nichtssagend |
| broken link · shortened link | defekter Link · Kurzlink (a URL shortener is Send Readiness's „URL-Kürzer“) |
| inline CSS | Inline-CSS (Gmail's clipping limit is Send Readiness's „Kürzungsgrenze“) |
| contact field · personalization token's fallback | Kontaktfeld · Ersatzwert |
| plain-text version · postal address | Textversion · Postanschrift |
| Line 3: (a compile or layout problem) · doesn't compile | Zeile 3: · lässt sich nicht kompilieren |
| title (of an article card, a post) | Titel („Betreff“ is the email's subject, „Überschrift“ a heading) |
| Email preferences link · View in browser link | Link zu den E-Mail-Einstellungen · Link „Im Browser ansehen“ |
| starter (template) | Startvorlage |
| starters: Blank · Weekly digest · Product update · Plain text · Announcement · Product grid · Blog digest | Leer · Wochenüberblick · Produktneuigkeiten · Nur Text · Ankündigung · Produktraster · Blog-Überblick |

MJML tags (mj-section, mj-body) stay as they are. Section names in the
server's sentences use the composer's words: Kopfzeile, Fußzeile, Hero,
2 Spalten, Text, Bild, Button. A block's sample content is written for the
email's readers, in the app language: the first-name fallback "there" is
"zusammen" („Hallo zusammen“).

## Automations

| English | Deutsch |
|---|---|
| A/B split · weight | A/B-Aufteilung · Gewichtung |
| automation (one, or the section) | Automation („Neue Automation“) |
| Branch · Yes · No | Verzweigung · Ja · Nein |
| Delay (step) · Wait 10 minutes | Verzögerung · 10 Minuten warten |
| double opt-in · double opt-in confirmed only | Double-Opt-in · nur mit bestätigtem Double-Opt-in |
| edge (a connection on the canvas) | Verbindung |
| exit · goal · goal rate | Ausstieg · Ziel · Zielquote |
| fail over · failover | umschalten · Failover |
| Filter (step) | Filter |
| flow · in the flow · leave the flow | Ablauf · im Ablauf · Ablauf verlassen („Automationsablauf“ where the kind needs saying; „Stream“ is only a stream: „Stream Automationen“) |
| List action | Listenaktion |
| method (HTTP) | Methode |
| override (a route) | überschreiben; "route override" on a step is „eigene Route“ |
| re-entry | Wiedereintritt |
| retire · retiring (a version) | stilllegen · wird stillgelegt |
| send window | Versandfenster |
| signing secret · write-only | Signaturschlüssel · nicht auslesbar |
| stats · Live stats | Statistik · Live-Statistik |
| step · trigger · action · condition | Schritt · Auslöser · Aktion · Bedingung |
| web push | Web-Push |
| path (of a branch or split) · connection | Pfad · Verbindung |
| starters: Welcome series · Re-engagement · Onboarding · Abandoned cart | Willkommensserie · Reaktivierung · Onboarding · Warenkorbabbruch |
| Send Readiness content checks | Inhaltsprüfungen der Versandbereitschaft |
| webhook delivery log · failed calls | Webhook-Zustellprotokoll · fehlgeschlagene Aufrufe |
| route label: SES, failover Postmark | SES, Failover: Postmark |
| Removed provider | Entfernter Anbieter |
| webinar | Webinar |
| version summary: created from a blank canvas · created from the X template · copied from X · first published version | aus leerer Arbeitsfläche erstellt · aus der Vorlage X erstellt · aus X kopiert · erste veröffentlichte Version |

## Approvals

| English | Deutsch |
|---|---|
| after exclusions | nach Ausschlüssen |
| approval inbox | Freigabe-Eingang |
| approval request · approver · author | Freigabeanfrage · freigebende Person · Autor (column: „Freigabe durch“) |
| Approve · Reject · Request changes | Freigeben · Ablehnen · Änderungen anfordern |
| Bypassed · emergency bypass | Umgangen · Notfall-Umgehung |
| comment · Discussion · evidence | Kommentar · Diskussion · Nachweis |
| Contacts in flight | Kontakte im Ablauf |
| Invalidated · Approval invalidated | Ungültig · Freigabe ungültig |
| median wait | Wartezeit (Median); a median, not an average: never „durchschnittlich“ or „mittlere“ |
| Pending · Rejected | Ausstehend · Abgelehnt |
| policy · rule · Rules matched | Richtlinie · Regel · Zutreffende Regeln |
| Recheck | Erneut prüfen |
| snapshot | Momentaufnahme |
| after approval (a change) | nach der Freigabe („Empfänger +14 % seit der Freigabe“; said once) |
| rules: audience ≥ 10,000 · warming-up provider · new provider · author lacks send · list requires approval · client sign-off | Zielgruppe ≥ 10.000 · Anbieter im Warm-up · neuer Anbieter · Autor ohne Sendeberechtigung · Liste erfordert Freigabe · Kundenfreigabe |
| automation publish · post publish · page publish · template publish (rules) | Automation veröffentlichen · Beitrag veröffentlichen · Seite veröffentlichen · Vorlage veröffentlichen |
| a send held · suppressed | zurückgehaltener Versand · gesperrt |
| online · active 2 h ago (an approver) | online · vor 2 Std. aktiv |
| Best time within 24 h of | Beste Zeit innerhalb von 24 Std. ab |
| Only people can… (not agents or keys) | Nur Menschen können … |

## Content languages

| English | Deutsch |
|---|---|
| language version · Source | Sprachversion · Original |
| Default language · Fallback language | Standardsprache · Ersatzsprache |
| Up to date · Outdated · Draft · Missing · Held | Aktuell · Veraltet · Entwurf · Fehlt · Zurückgehalten |
| Mark as translated · Mark as up to date | Als übersetzt markieren · Als aktuell markieren |
| readers (of a language) | Kontakte mit {language} („1.468 Kontakte mit Deutsch“); in a sentence „wer {language} liest“ |
| page copy · built-in labels | Seitentexte · Standardtexte |
| Results per language · click map | Ergebnisse je Sprache · Klickkarte |
| no language · other languages (results) | ohne Sprache · andere Sprachen |
| compare view's blocks: Header · Footer · Raw HTML · Social links · Two cards · 3 columns · Hero image · Button · Intro · The email | Kopfzeile · Fußzeile · HTML-Code · Social-Media-Links · Zwei Karten · 3 Spalten · Hero-Bild · Button · Einleitung · Die ganze E-Mail |
| Headline (a block) · no alt text | Schlagzeile („Überschrift“ is a heading) · kein Alternativtext |
| version labels: copied from · layout of · applied from · block copied from · changed by an agent · edited in the flow editor · translated from | kopiert aus · Layout von · übernommen aus · Block kopiert aus · von einem Agenten geändert · im Ablauf-Editor bearbeitet · übersetzt aus |
| a language's changes after approval: added · marked translated · back to draft · sent again · edited | hinzugefügt · als übersetzt markiert · zurück auf Entwurf · erneut gesendet · bearbeitet |
| copy (a form's or page's own text) · Copy in | Text · Text auf („Text auf {language}“) |
| Live · Incomplete · Not set up (copy) | Live · Unvollständig · Nicht eingerichtet |
| string (text to write) | Text |
| visitors | Besucher („Besucher mit {language}“) |

A content language's name comes from the browser's or CLDR's names in the
app language ("Spanisch"), capitalised as German nouns are; the app's own
picker names each language in itself.

## Conditional content

| English | Deutsch |
|---|---|
| combination (of blocks) · version | Kombination · Version |
| conditional block | bedingter Block |
| Gmail clips · clipping size | Gmail kürzt · Kürzungsgrenze |
| includes · excludes | umfasst · umfasst nicht |
| is · isn't · contains · is set · is empty | ist · ist nicht · enthält · ist gesetzt · ist leer (as automations' conditions) |
| is at least · is at most · is between … and … | ist mindestens · ist höchstens · liegt zwischen … und … |
| is greater than · is less than · is before · is after | ist größer als · ist kleiner als · liegt vor · liegt nach |
| is in · is not in | ist in · ist nicht in |
| Match ALL / ANY of these | ALLE / MINDESTENS EINE davon erfüllen (the uppercase words stay uppercase; the two-way switch alone: ALLE · EINE; "AND" is "UND") |
| operator | Operator |
| Preview as · Show why | Vorschau als · Grund anzeigen |
| Visibility (who sees a block) | Sichtbarkeit |

## Accessibility

| English | Deutsch |
|---|---|
| Accessibility | Barrierefreiheit |
| issue · serious · moderate | Problem · schwerwiegend · mittel |
| alt text · decorative image | Alternativtext (short: Alt-Text) · dekoratives Bild |
| contrast · Passes AA · Fails AA | Kontrast · AA erfüllt · AA nicht erfüllt |
| Heading 1 · Paragraph · Text role | Überschrift 1 · Absatz · Textrolle |
| layout table | Layouttabelle |
| screen reader | Screenreader |
| Warn · Block (the policy) | Warnen · Blockieren |
| skipped heading level · card titles | übersprungene Überschriftenebene · Kartentitel |

"Heading" is "Überschrift", never "Betreff", which is an email's subject,
nor "Titel", which is a post's or page's title. Content languages in a
sentence use their German names, which are nouns: "auf Deutsch", "auf
Spanisch".

## Earn

Payments, paid subscriptions, offers, gifts, sponsors and revenue. A
"paid member" is a **zahlendes Mitglied**; where a member of the team
could be meant on the same screen, call them **Teammitglied**. A reader's
Stripe subscription is an **Abo** (**Bezahlabo** where it must be
explicit), and it is **gekündigt**, never "abgemeldet": "abmelden" is
unsubscribing from a newsletter.

| English | Deutsch |
|---|---|
| access (paid) · With access | Zugang · Mit Zugang |
| access rank · Rank 2 | Zugangsstufe · Stufe 2 |
| Access rules | Zugangsregeln |
| Active · Trialing · Past due · Cancels soon · Canceled · Ended (paid status) | Aktiv · In Testphase · Überfällig · Endet bald · Gekündigt · Beendet |
| archive · Archived (a tier) | archivieren · Archiviert |
| attribution · attribution window · credited to (an email) | Zuordnung · Zuordnungszeitraum · zugeordnet zu |
| benefit (of a tier) | Vorteil |
| billing email · billing period · Manage billing | Rechnungs-E-Mail · Abrechnungszeitraum · Abrechnung verwalten |
| book · booking · Booked (a sponsor in an issue) | buchen · Buchung · Gebucht |
| buyer · recipient (of a gift) | Käufer · Empfänger |
| cancel page · gift page · upgrade page · welcome page | Kündigungsseite · Geschenkseite · Upgrade-Seite · Willkommensseite |
| charge (a card) | belasten (die Karte); "nothing is charged" is „es wird nichts berechnet“ |
| checkout · test checkout | Checkout · Test-Checkout ("Stripe Checkout" is a name) |
| churn · churn rate | Kündigungen · Kündigungsrate |
| click token | Klick-Token |
| Complimentary · complimentary access | Gratis · Gratiszugang |
| conversion · conversion API | Conversion · Conversion-API |
| coupon · promotion code · code | Gutschein · Aktionscode · Code |
| creative (a sponsor's) | Werbemittel |
| customer portal | Kundenportal |
| deliver · delivery date (a gift) | zustellen · Zustelldatum |
| discount · 20% off | Rabatt · 20 % Rabatt |
| dispute (a card payment's) | Anfechtung (never „Beschwerde“, a spam complaint) |
| Earn's tabs: Overview · Plans & offers · Paid members · Gifts · Sponsors · Products · Settings | Übersicht · Tarife & Angebote · Zahlende Mitglieder · Geschenke · Sponsoren · Produkte · Einstellungen |
| expansion · contraction · reactivated (MRR movement) | Erweiterung · Verringerung · reaktiviert |
| Free (the card) · free subscriber | Kostenlos · kostenloser Abonnent |
| free-to-paid | von kostenlos zu zahlend |
| gift · gift option | Geschenk · Geschenkoption |
| headline (an upgrade page's, a creative's) | Schlagzeile; a subject is „Betreff“, a heading „Überschrift“ |
| highlight (a tier) · Most popular | hervorheben · Am beliebtesten |
| hold · on hold · release (an issue) | reservieren · reserviert · Reservierung aufheben |
| in person (a gift handed over) | persönlich |
| issue (of a newsletter: #143) | Ausgabe (Ausgabe #143); an accessibility issue is „Problem“ |
| legacy price | bisheriger Preis |
| lifetime value | Kundenwert |
| live · test (a key, a mode, payments) | Live · Test („Live-Modus“, „Testmodus“); "no real payments" is „keine echten Zahlungen“ |
| live (a page, an offer or a template version that's running) | live |
| map (a Stripe price to a tier) | zuordnen |
| member email · membership | Mitglieder-E-Mail · Mitgliedschaft |
| merchant of record | Merchant of Record (the party that sells in its own name for you; the term stays, as payment providers' German pages keep it) |
| monthly · yearly · Monthly · Yearly (the choice) · /month | monatlich · jährlich · Monatlich · Jährlich · /Monat |
| MRR · ARR · MRR movement | MRR · ARR · MRR-Entwicklung |
| Needs you (Earn's overview) | Wartet auf Sie |
| normal price (without an offer) | regulärer Preis |
| offer · signup · retention · referral reward | Angebot · Anmeldeangebot · Halteangebot · Empfehlungsprämie |
| Overdue · Unpaid · Paid (a sponsor's payment) | Überfällig · Unbezahlt · Bezahlt |
| paid member | zahlendes Mitglied |
| paid status · paid-only | Zahlstatus · nur für zahlende Mitglieder |
| Paid subscriptions (what Earn sells) | Bezahlabos |
| paid-through date | bezahlt bis |
| payment provider | Zahlungsanbieter |
| payment terms (a sponsor's) | Zahlungsbedingungen |
| payouts | Auszahlungen |
| plan · Plans & offers | Tarif · Tarife & Angebote |
| Public · Hidden (a tier's visibility) | Öffentlich · Verborgen |
| receipt · invoice | Beleg · Rechnung |
| record (a sponsor's payment, by hand) | erfassen |
| redeem · redeem link (a gift) | einlösen · Einlöse-Link |
| redeem · redemption (an offer code) | einlösen · Einlösung |
| refund | Rückerstattung · erstatten |
| renew · renewal | verlängern · Verlängerung |
| restricted key · secret key | eingeschränkter Schlüssel · geheimer Schlüssel (Stripe's German words) |
| results so far · final (a sponsor report's) | bisherige Ergebnisse · endgültig |
| revenue · income | Umsatz · Einnahmen |
| roll (a Stripe key) | rotieren |
| seller ("you're the seller") | Verkäufer (in a sentence: „Sie verkaufen selbst“) |
| signing secret (a webhook's) · endpoint | Signaturschlüssel (as Automations has it) · Endpunkt |
| Since · Member since (a paid member's) | Seit · Mitglied seit |
| slot · placement | Werbeplatz · Platzierung |
| sponsor · sponsorship · Sponsored (the block's label) | Sponsor · Sponsoring · Anzeige (the label German readers and law expect on paid content) |
| sponsor report | Sponsorenbericht |
| subscription (a reader's, billed by Stripe) | Abo (Bezahlabo where it must be explicit) |
| Terms of sale · Support address | Verkaufsbedingungen · Support-Adresse |
| tier · paid tier | Tarif · Bezahltarif |
| timeline (a paid member's) | Verlauf |
| trial · free trial | Testphase · kostenlose Testphase |
| upgrade · Upgrade block | Upgrade · Upgrade-Block |
| webhook silent · gap check | Webhook stumm · Lückenprüfung |
| write-only · sealed (a key, a secret) | nicht auslesbar · versiegelt |

Some English messages choose a word for English grammar only: an article
(`{article, select, an {an} other {a}} {tier}`) or a possessive
(`{ends, select, s {{sponsor}’} other {{sponsor}’s}}`). German keeps the
argument and puts what it needs in its `other` branch. The article belongs
to the German noun and its case, not to the name: `{article, select, other
{den}} Tarif {tier}`. A possessive is "von" with the name, since a German genitive "-s"
can't be added to any name safely: `die Buchung von {ends, select, other
{{sponsor}}}`. Where the English counts with `#`, keep
`{count, plural, one {# …} other {# …}}` rather than a bare `{count}`, so
the number is grouped by the region format (48.213).

## Website and pages

The publication website, its posts, the page builder and the public pages'
own words. Words a reader sees follow the built-in labels of the public
pages (`internal/pagelabels`): "Newsletter", "Ausgabe", "Archiv",
"E-Mail-Einstellungen". The built-in labels address readers with "Sie",
like the app; a workspace that prefers "du" writes its own page copy.

| English | Deutsch | Note |
|---|---|---|
| About (page, template) | Über uns | |
| Access (Website › Access) | Zugang | |
| archive · Archive (the page, the link) | Archiv · Archiv | "Archivseite" when the page itself is meant |
| band (subscribe band) | Streifen | a full-width strip of the home page: "Anmeldestreifen" |
| block · layers | Block · Ebenen | the page builder's |
| built-in page | Standardseite | like "Standardtexte" |
| byline · By {name} | Autorenzeile · Von {name} | |
| canonical URL | kanonische URL | as Google's German tools say it |
| card (paywall card, a post's card) | Karte | |
| certificate | Zertifikat | |
| countdown | Countdown | |
| cover (image) | Titelbild | |
| excerpt | Auszug | |
| FAQ (block) | FAQ | |
| featured · Feature on home | hervorgehoben · Auf der Startseite hervorheben | |
| feed · RSS feed | Feed · RSS-Feed | |
| free preview · Free preview ends here (the post's block) | kostenlose Vorschau · Hier endet die kostenlose Vorschau | |
| Hero (block) | Hero | the block's name, as in the composer |
| home page | Startseite | |
| issue (of a newsletter) | Ausgabe | "48 Ausgaben", "Neueste Ausgabe lesen" |
| lead magnet | Lead-Magnet | a free file for confirmed subscribers |
| live (the website) · live (a page, a form, a version) | online · live | as Earn and the signup forms have it |
| member · free member · paid member (of the site) · membership | Mitglied · kostenloses Mitglied · zahlendes Mitglied · Mitgliedschaft | a reader who signs in, as Earn has it; a member of the team is "Teammitglied" here |
| meter | kostenloses Lesekontingent | "Lesekontingent" as a heading |
| navigation · header · footer | Navigation · Kopfbereich · Fußbereich | a site's; an email has "Kopfzeile" and "Fußzeile" |
| page · Pages | Seite · Seiten | |
| page builder | Seiteneditor | |
| Pay what you want | Freie Preiswahl | |
| paywall | Paywall | |
| post · web-only post | Beitrag · Beitrag nur im Web | |
| publish · unpublish · unschedule | veröffentlichen · Veröffentlichung zurückziehen · Planung aufheben | Unpublished: Zurückgezogen |
| redirect | Weiterleitung | |
| referrer (where views come from) | Verweisquelle | a person who refers is "empfehlende Person" |
| registration (event) | Registrierung | "Anmeldung" is a signup |
| sales page | Verkaufsseite | |
| signups · upgrades · views · visitors | Anmeldungen · Upgrades · Aufrufe · Besucher | |
| site · website | Website | |
| sitemap | Sitemap | |
| slug | URL-Pfad | the page's or tag's part of the address |
| social image | Vorschaubild | the image a shared link shows |
| store product (block) | Shop-Produkt | |
| tab (a panel's: the Web tab) | Tab | „Tab Web“ |
| tag | Tag | das Tag, die Tags |
| tagline | Slogan | |
| testimonial | Testimonial | |
| theme | Theme | as WordPress's German says it |
| tier · paid tier | Tarif · Bezahltarif | as Earn has it; a tier's name, such as "Insider", stays as it is |
| title (a post's, a page's, the site's) · SEO title | Titel · SEO-Titel | "Betreff" is an email's subject ("Betreff der E-Mail" beside a title) |
| visibility · Public · Members · Paid | Sichtbarkeit · Öffentlich · Mitglieder · Zahlende | |
| visitor | Besucher | someone who isn't signed in |
| webinar | Webinar | |

## Team

| English | Deutsch |
|---|---|
| Team | Team |
| Invite member | Mitglied einladen |
| Members | Mitglieder |
| Everyone with access to Acme Newsletter | Alle mit Zugriff auf Acme Newsletter |
| Search name or email | Name oder E-Mail-Adresse suchen |
| Any role · Any status · Any 2FA | Alle Rollen · Alle Status · 2FA: alle |
| Role · scope | Rolle · Bereich |
| Sign-in | Anmeldung |
| Last active | Zuletzt aktiv |
| Manage · Reactivate | Verwalten · Reaktivieren |
| Password · passkey | Passwort · Passkey |
| just now · yesterday | gerade eben · gestern |
| 2 selected · Bulk: | 2 ausgewählt · Für Auswahl: |
| Change role · Require 2FA now · Deactivate | Rolle ändern · 2FA jetzt verlangen · Deaktivieren |
| Required · off, 3 days of grace left | Erforderlich · aus, noch 3 Tage Schonfrist |
| deactivated Sep 30, 2026 | deaktiviert am 30. Sept. 2026 |
| Pending invitations · 1 | Offene Einladungen · 1 |
| Links last 7 days by default | Links gelten standardmäßig 7 Tage |
| Invitee · Sent · Expires | Eingeladene Person · Gesendet · Läuft ab |
| invited by Linh Nguyen | eingeladen von Linh Nguyen |
| Beta Program list only | nur Liste Beta Program |
| in 5 days | in 5 Tagen |
| Resend · Copy link · Revoke | Erneut senden · Link kopieren · Widerrufen |

## Language and region

| English | Deutsch |
|---|---|
| Language & region | Sprache & Region |
| App language | App-Sprache |
| Instance default (English) | Standard der Instanz (English) |
| Region format | Regionsformat |
| Follows the app language | Wie die App-Sprache |
| Time · 24-hour · 12-hour | Uhrzeit · 24 Stunden · 12 Stunden |
| Week starts on · Monday · Sunday | Woche beginnt am · Montag · Sonntag |
| From the region format | Aus dem Regionsformat |
| Preview | Vorschau |
| Help translate ReputeCast | Beim Übersetzen von ReputeCast helfen |
| Save · Undo | Speichern · Rückgängig |
| Profile | Profil |
| timezone | Zeitzone |
| quiet hours | Ruhezeiten |
| Settings › General | Einstellungen › Allgemein |
| Invitation language | Sprache der Einladung |
| Default language · Default region format | Standardsprache · Standard-Regionsformat |

Languages are always named in their own language, in every interface
language: English, Tiếng Việt, Deutsch (Deutschland). "ReputeCast standard"
is a name and stays in English.

## Account and security

| English | Deutsch |
|---|---|
| Full name | Vollständiger Name |
| Upload photo · Remove | Foto hochladen · Entfernen |
| Change email · verification link | E-Mail-Adresse ändern · Bestätigungslink |
| Current password · New password | Aktuelles Passwort · Neues Passwort |
| security key · platform passkey | Sicherheitsschlüssel · Geräte-Passkey |
| Rename · Keep it | Umbenennen · Behalten |
| Time-based codes (TOTP) | Zeitbasierte Codes (TOTP) |
| Set up · Replace app | Einrichten · App ersetzen |
| Regenerate codes | Codes neu erzeugen |
| 7 of 10 remain | 7 von 10 übrig |
| Connected sign-ins | Verknüpfte Anmeldungen |
| Link · Unlink · Managed | Verknüpfen · Trennen · Verwaltet |
| session · Active sessions · This session | Sitzung · Aktive Sitzungen · Diese Sitzung |
| Sign out all other sessions | Alle anderen Sitzungen abmelden |
| Device · Browser · IP · location (approx.) | Gerät · Browser · IP · Standort (ungefähr) |
| My workspaces · Leave workspace | Meine Arbeitsbereiche · Arbeitsbereich verlassen |
| In-app · All events | In der App · Alle Ereignisse |
| MCP agent | MCP-Agent |

## Instance admin, keys, webhooks and the audit log

| English | Deutsch | Note |
|---|---|---|
| actor (audit) · Any actor type | Akteur · Alle Akteurtypen | who did it: a member, a key, an agent or the system |
| agent capability: Allow · Deny · Approve · Pause only (Settings › MCP) | Erlauben · Verweigern · Freigabe · Nur pausieren | the matrix's cells; „Freigabe“ as the caption quotes it |
| All retained (date range) | Gesamte Aufbewahrungsdauer | |
| app registration (Entra) · app integration (Okta) | App-Registrierung · App-Integration | „die App-Registrierung in Entra“ |
| batch (of streamed entries) | Batch | as in "Batch-Endpunkt" |
| break-glass Owner | Inhaber mit Notfallzugang | never „Notfall-Inhaber“ |
| break-glass password · break-glass passkey | Notfallpasswort · Notfall-Passkey | what a break-glass Owner keeps while SSO is enforced |
| claim · Client ID · client secret · Scopes (OIDC) | Claim · Client ID · Client Secret · Scopes | named as the identity provider's console names them |
| danger zone | Gefahrenbereich | |
| delivery (webhook) · Delivery log | Zustellung · Zustellprotokoll | |
| Development build | Entwicklungs-Build | |
| digest (email) · Hourly digest · Daily digest | Zusammenfassung · Stündliche Zusammenfassung · Tägliche Zusammenfassung | notification preferences |
| discovery · issuer · Issuer URL (OIDC) | Discovery · Aussteller · Aussteller-URL (Issuer) | |
| Effective permissions | Effektive Berechtigungen | a member's page; the column is "Effektiv" |
| endpoint · payload | Endpunkt · Payload | |
| enforce (SSO, two-factor) · enforcement | erzwingen · Erzwingung | "Enforce SSO for acme.io": „SSO für acme.io erzwingen“ |
| entry (audit) · Entry details | Eintrag · Details zum Eintrag | |
| Failing · Paused (an endpoint) | Fehlerhaft · Pausiert | "Healthy" is "Fehlerfrei" |
| Force sign-out | Abmeldung erzwingen | |
| Generic OIDC (a preset) | Generisches OIDC | the IdP's short name in sentences is "IdP" |
| Given (approvals a member gave) | Erteilt | |
| group mapping · group object ID | Gruppenzuordnung · Objekt-ID der Gruppe | |
| hourly cleanup | stündliche Bereinigung | |
| identity provider | Identitätsanbieter | "IdP" stays |
| immutable (audit entries) | unveränderlich | |
| instance-level (entries) · Instance level | auf Instanzebene · Instanzebene | |
| JIT account creation | automatische Kontoerstellung (JIT) | |
| just-in-time account · JIT default role | JIT-Konto · JIT-Standardrolle | |
| Locked (sign-in) · Unlock sign-in | Gesperrt · Anmeldung entsperren | a lockout is "Anmeldesperre" |
| Make instance admin · Remove instance admin | Zum Instanz-Admin machen · Als Instanz-Admin entfernen | |
| overage link (Entra's groups) | Overage-Link | Entra's own word |
| preset (an SSO provider's, a key's) | Voreinstellung | never „Vorlage“, a template |
| proposal (an agent asked a person) · proposal recorded | Vorschlag · Vorschlag erfasst | "Ops asked to pause …": „Ops schlägt vor, … zu pausieren“ |
| provider status: degraded · down | beeinträchtigt · ausgefallen | as Channels and SMS' Beeinträchtigt · Ausgefallen |
| re-authentication | erneute Authentifizierung | |
| Redirect URI | Umleitungs-URI | as Entra's German console |
| reference ID · request ID | Referenz-ID · Anfrage-ID | |
| result: Allowed · Denied · Error | Erlaubt · Verweigert · Fehler | |
| retention | Aufbewahrungsdauer | |
| revoke · rotate · replace (a key or secret) | widerrufen · rotieren · ersetzen | |
| secret · signing secret | geheimer Schlüssel · Signaturschlüssel | "secrets" in general: „vertrauliche Zugangsdaten“ |
| Send approvals (Settings › General) | Freigabe vor dem Versand | |
| Sending (notification category) | Versand | so it isn't read as the status "Im Versand" |
| SIEM or warehouse | SIEM oder Data Warehouse | |
| single sign-on | Single Sign-on | "SSO" stays |
| stream (audit entries, noun) · stream (verb) · streaming endpoint | Stream · streamen · Streaming-Endpunkt | the System stream is "Stream System" |
| test sign-in | Testanmeldung | |
| Then allowed | Danach erlaubt | a sign-in that was rejected, then allowed |
| token (an agent's) | Token | "agent token": „Token des Agenten“ |
| tool (MCP) | Tool | tool names (get_campaign_report) stay as they are |
| Trusted from Okta (two-factor) | Über Okta abgesichert | the IdP enforces the second factor |
| typed confirmation | Bestätigung durch Eingabe des Namens | |
| Up to date · Update available (a release) | Auf dem neuesten Stand · Update verfügbar | language versions keep "Aktuell" |
| user (an account on the instance) | Konto | Instance admin › Users is "Konten"; a workspace's people are "Mitglieder" |
| Workspace ID | Arbeitsbereichs-ID | |
| workspace switcher | Arbeitsbereichsauswahl | |

## Channels and SMS

| English | Deutsch |
|---|---|
| 10DLC · local · mobile · toll-free (number kinds) | 10DLC · Ortsnetz · Mobil · gebührenfrei („Ortsnetznummer“, „Mobilnummer“, „gebührenfreie Nummer“) |
| adapter · manifest · contract test suite | Adapter · Manifest · Contract-Test-Suite (developer words, kept) |
| auto-pause · Auto-paused | automatische Pause (Auto-Pause) · Automatisch pausiert |
| auto-pause reasons: Too many texts failed · Carriers filtered too many texts · Too many people opted out | Zu viele SMS fehlgeschlagen · Netzbetreiber haben zu viele SMS gefiltert · Zu viele Abmeldungen |
| brand (in a 10DLC registration) · brand prefix | Marke · Markenpräfix |
| budget · spend · price per segment | Budget · Ausgaben · Preis pro SMS-Teil |
| budget used up · budget at 80% | Budget aufgebraucht · Budget zu 80 % genutzt („SMS-Budget für Oktober aufgebraucht“) |
| carrier | Netzbetreiber |
| channel · Email · SMS · Push · WhatsApp | Kanal · E-Mail · SMS · Push · WhatsApp |
| consent · SMS consent · consent text · evidence | Einwilligung · SMS-Einwilligung · Einwilligungstext · Nachweis |
| credentials · Live · Test credentials | Zugangsdaten · Live · Test-Zugangsdaten |
| credentials rejected · isn't answering · is back (a provider) | Zugangsdaten abgelehnt · antwortet nicht · wieder erreichbar |
| delivery receipt | Zustellbestätigung (what a carrier reports back; "Last receipt" is „Letzte Bestätigung“) |
| fallback (of a route) | Ersatz |
| filtered by carriers | von Netzbetreibern gefiltert |
| held · skipped | zurückgehalten · übersprungen |
| inbound (text) · reply | eingehende SMS · Antwort |
| keyword | Schlüsselwort (STOP, START, HELP and YES stay in English: people text them) |
| landline · not in service · invalid number · unreachable | Festnetz · nicht vergeben · ungültige Nummer · nicht erreichbar |
| link preview | Linkvorschau (a messaging app's machine click, never a person) |
| local time · "Berlin local" (a reply's time) | Ortszeit · „Ortszeit Berlin“ |
| no answer · invalid credentials (what a provider answered a call) | keine Antwort · ungültige Zugangsdaten (the HTTP status before them stays: „401 Unauthorized · ungültige Zugangsdaten“) |
| opt in · opt-in confirmation | einwilligen · Opt-in-Bestätigung |
| opt out · opt-outs · opt-out text · opt-out reply | abmelden · Abmeldungen · Abmeldehinweis · Abmeldeantwort |
| opt-out sync · Two-way · Read only | Abmelde-Synchronisierung · Beidseitig · Nur lesen |
| opt-out list (a provider's own) | Abmeldeliste („Abmeldeliste von Twilio“) |
| outside quiet hours (a text waits or is skipped) | außerhalb des Versandfensters (the window texts may go out in, not „außerhalb der Ruhezeiten“) |
| provider status: Healthy · Degraded · Down | Fehlerfrei · Beeinträchtigt · Ausgefallen |
| quiet hours · sending window | Ruhezeiten · Versandfenster |
| raw event · normalized | Rohereignis · normalisiert |
| receipts silent (none came back while texts went out) | Bestätigungen bleiben aus |
| registration · registered | Registrierung · registriert |
| route · Routing & rules · route tester | Route · Routing & Regeln · Routentester |
| segment (of a text) | SMS-Teil (the billed part of 160 GSM-7 or 70 UCS-2 characters; never „Segment“, an audience segment) |
| sender ID · alphanumeric sender ID | Absenderkennung · alphanumerische Absenderkennung |
| short-link domain · public shortener | Kurzlink-Domain · öffentlicher URL-Kürzer |
| signature · signing secret | Signatur · Signaturschlüssel |
| status callback (Twilio) | Status-Callback |
| suppression list (SMS) | SMS-Sperrliste |
| test text · test number | Test-SMS · Testnummer |
| text (a message) · to text | SMS (die SMS, die SMS) · per SMS senden |
| Text message terms | SMS-Bedingungen |
| throughput · 14 texts a second | Durchsatz · 14 SMS pro Sekunde |
| unschedule · a send (of a campaign) | Planung aufheben · Versand („Dieser Versand ist beendet“) |

"GSM-7", "UCS-2", "Unicode", "VoIP" and "Emoji" stay as they are. A
phone number, a sender ID and a country code are data: never translated.

## Web push and WhatsApp

| English | Deutsch |
|---|---|
| access token · permanent access token · system user | Zugriffstoken · dauerhaftes Zugriffstoken · Systemnutzer |
| Allow · Not now · Granted (a prompt's numbers) | Zulassen · Nicht jetzt · Erteilt |
| anonymous · linked (a subscription) | anonym · verknüpft (mit einem Kontakt) |
| app secret · verify token · callback URL · phone number ID | App-Geheimcode · Verifizierungstoken · Callback-URL · Telefonnummer-ID |
| banner (a notification's style on a desktop) | Banner |
| Bell button · Dialog · Slide-in bar (prompt styles) | Glocken-Button · Dialog · Einblendleiste |
| business account · display name | Unternehmenskonto · Anzeigename ("WhatsApp Business Account" is a name) |
| category (of a template): Marketing · Utility · Authentication | Kategorie: Marketing · Utility · Authentifizierung (as WhatsApp Manager names them) |
| conversation · customer service window | Konversation · Kundenservice-Fenster (short: „24-Stunden-Fenster“, never the quiet-hours „Versandfenster“) |
| expiry · Expires after | Ablauf · Läuft ab nach |
| free-form reply · quick reply | Freitext-Antwort · Schnellantwort |
| frequency cap | Häufigkeitslimit |
| header · body · footer · buttons (of a template) | Kopfzeile · Textkörper · Fußzeile · Buttons |
| Home Screen (iPhone, iPad) | Home-Bildschirm |
| icon · badge · image (of a notification) | Symbol · Kennzeichen · Bild |
| In review · In appeal · submit (a template) | In Prüfung · Einspruch läuft · zur Prüfung einreichen |
| messaging limit · rolling 24 hours | Nachrichtenlimit · letzte 24 Stunden |
| notification (one push message) | Benachrichtigung |
| permission prompt (the browser's own) | Berechtigungsabfrage (des Browsers) |
| prompt · prompt designer | Abfrage · Abfrage-Designer (an AI's prompt is „Prompt“) |
| publication site | Website der Publikation |
| push (in a sentence) · Push (the channel) | Push-Benachrichtigung · Push |
| push service (the browser maker's) | Push-Dienst |
| quality rating · High · Medium · Low | Qualitätsbewertung · Hoch · Mittel · Niedrig |
| rate card · Edit rates | Preisliste · Preise bearbeiten |
| read receipts | Lesebestätigungen (WhatsApp's own German) |
| Rotate keys · public key · private key | Schlüssel rotieren · öffentlicher Schlüssel · privater Schlüssel |
| Sealed | Versiegelt |
| Shown (a count) · Shown (an event) · Gone · removed | Angezeigt · Angezeigt · Abgelaufen · entfernt |
| site (where readers subscribe) · scope (of a service worker) | Website · Scope |
| snippet | Snippet |
| subscriber · subscription | Push-Abonnent · Push-Abonnement |
| Sync now | Jetzt synchronisieren |
| Tapped (a WhatsApp button) | Angetippt |
| template (WhatsApp) | Vorlage (Nachrichtenvorlage where it must be explicit) |
| token (that a link carries to know the reader) | Token |

"Service Worker", "VAPID", "Meta", "Graph API" and Meta's own screens and
paths (WhatsApp Manager, Meta for Developers › … › Configuration) stay as
they are.

## Surveys and polls

| English | Deutsch |
|---|---|
| 0–10 scale · Emoji scale · faces | Skala 0–10 · Emoji-Skala · Gesichter |
| answer · answer page | Antwort · Antwortseite |
| Breakdown · Break down by | Aufschlüsselung · Aufschlüsseln nach |
| completion rate · drop-off · started · completed | Abschlussrate · Abbruch · begonnen · abgeschlossen |
| Embedded · Popup · Hosted page (form kinds) | Eingebettet · Pop-up · Seite auf ReputeCast |
| link scanner · security gateway · prefetching proxy | Link-Scanner · Sicherheits-Gateway · Prefetch-Proxy (machine visits, never a person) |
| Not likely · Very likely (an NPS scale's ends) | Unwahrscheinlich · Sehr wahrscheinlich |
| Open · Closed (a survey or a poll) | Offen · Geschlossen |
| Owner (the forms list's column) | Verantwortlich (not the role „Inhaber“) |
| poll (the block) · a poll | Abstimmung · eine Abstimmung |
| promoters · passives · detractors | Promotoren · Passive · Kritiker ("NPS" and "Net Promoter Score" stay) |
| question · respondent · response | Frage · befragte Person (plural: Befragte) · Antwort |
| submission | Einsendung |
| survey | Umfrage |
| Top answer | Häufigste Antwort |
| vote · voter | Stimme · abstimmende Person (plural: Abstimmende) |

## Notifications

| English | Deutsch |
|---|---|
| digest (recurring) · issue (of a digest) | wiederkehrender Newsletter · Ausgabe |
| event type (a row of Notification preferences) | Ereignistyp |
| Everyone with campaigns:pause | Alle mit campaigns:pause |
| paid tier (Earn) | Bezahltarif |
| placement test · seed mailbox | Platzierungstest · Testpostfach |
| Sending (the notification center's tab) | Versand |
| verification credits | Prüf-Credits |

## Agency accounts

| English | Deutsch |
|---|---|
| Account manager · Agency manager (staff levels) | Kundenbetreuung · Agenturleitung |
| agency · Agency (the area) · agency mode | Agentur · Agentur · Agenturmodus |
| assign · assignment | zuweisen · Zuweisung |
| client · client workspace | Kunde · Kunden-Arbeitsbereich |
| client people (a client's own people) | Personen des Kunden (never the agency team) |
| Client approver · Client viewer (roles) | Freigeber (Kunde) · Betrachter (Kunde) (role names are nouns, like Bearbeiter and Betrachter; in a sentence an approver is „freigebende Person“) |
| client sign-off | Kundenfreigabe |
| copy (of a library item) · behind | Kopie · veraltet (the library has a newer version) |
| flat (rate) · monthly fee | pauschal · Monatsgebühr |
| incident · Needs attention | Störung · Handlungsbedarf |
| level (of staff) | Stufe |
| library · shared library · library item | Bibliothek · gemeinsame Bibliothek · Bibliothekselement |
| provider cost · plan | Anbieterkosten · Tarif |
| rebill · rebill rate · to rebill | weiterberechnen · Weiterberechnungssatz · weiterzuberechnen |
| saved section | gespeicherter Abschnitt |
| sender health · good · fair · poor | Absenderzustand · gut · mäßig · schwach |
| setup steps · Setting up | Einrichtungsschritte · Wird eingerichtet |
| staff | Agenturteam |
| usage | Nutzung |
| workspace template · apply (a template) | Arbeitsbereichsvorlage · anwenden |

## Grow, Growth and scoring

| English | Deutsch | Note |
|---|---|---|
| awaiting confirmation | wartet auf Bestätigung | a referral whose subscriber hasn't confirmed |
| band (score) · Cold · Warm · Hot | Stufe · Kalt · Warm · Heiß | the names are fixed, like the English |
| burst · shared IP · numbered addresses | Häufung · gemeinsame IP · nummerierte Adressen | the signals that hold a referral |
| cap (a rule's) | Limit | "6 a week": „6 pro Woche“ |
| claim · claim link | einlösen · Einlöse-Link | a coupon reward |
| cohort retention | Kohortenbindung | Audience › Growth |
| counted · not counted | gezählt · nicht gezählt | "Count" and "Don't count" in Review: „Zählen“, „Nicht zählen“ |
| coupon (Stripe) | Gutschein | |
| decay | Abklingen | |
| double opt-in | Double-Opt-in | two-factor is „Zwei-Faktor-Authentifizierung“ |
| engagement tier · Active · Cooling · At risk · Sunset | Engagement-Stufe · Aktiv · Nachlassend · Gefährdet · Auslaufend | List health's tiers, as Scoring names them; the sunset policy is „Auslaufregel“ |
| field condition | Feldbedingung | |
| fulfill · fulfilled · To fulfill | aushändigen · ausgehändigt · Auszuhändigen | a manual reward; "sent" stays „gesendet“ |
| Growth (Audience › Growth) | Zuwachs | so it doesn't read as Grow („Wachstum“) |
| held for review | zur Prüfung zurückgehalten | short: „zurückgehalten“ |
| hold period · in hold | Wartezeit · in der Wartezeit | "the 7-day hold": „die Wartezeit von 7 Tagen“ |
| hosted (form, page) | auf ReputeCast · von ReputeCast gehostet | |
| invite page | Einladungsseite | |
| join · departure | Zugang · Abgang | Growth's joined and left: „Zugänge“, „Abgänge“ |
| leaderboard | Bestenliste | |
| List health | Listenqualität | as Growth and Scoring refer to it |
| milestone · milestone email | Meilenstein · Meilenstein-E-Mail | "Meilenstein 3" |
| net (growth) | netto · Nettozuwachs | |
| newsletter | Newsletter | der Newsletter |
| partner · partner code · partner link · partner profile · partnership | Partner · Partnercode · Partnerlink · Partnerprofil · Partnerschaft | |
| recommendation · recommend · Recommending you | Empfehlung · empfehlen · Empfehlen Ihren Newsletter | |
| referral · referrer · referral link · referral program | Weiterempfehlung · empfehlende Person · Empfehlungslink · Empfehlungsprogramm | "Weiterempfehlung" keeps a referral apart from a newsletter's „Empfehlung“ |
| referral hub | Prämienseite | the referrer's own page |
| Referral · Recommendations · Reward block | Weiterempfehlungs-Block · Empfehlungs-Block · Prämien-Block | the composer's Grow blocks |
| referring page · referring domain | verweisende Seite · verweisende Domain | never „empfehlen“, which is a referral |
| review queue · Review (tab) | Prüfwarteschlange · Prüfung | |
| reward | Prämie | |
| rule · points | Regel · Punkte | |
| score · scoring | Score · Scoring | |
| self-referral | Selbstempfehlung | |
| soft bounce · hard bounce | Soft Bounce · Hard Bounce | |
| source (growth) | Quelle | |
| spread (of scores) | Verteilung | |
| subscribe request | Anmeldeanfrage | what one instance sends a partner |
| threshold | Schwellenwert | |
| token (Liquid) | Platzhalter | as the composer's personalization tokens |
| trigger (automation) | Auslöser | |

## Audience, imports, forms and reports

| English | Deutsch | Note |
|---|---|---|
| accent color · page background | Akzentfarbe · Seitenhintergrund | public page branding |
| Added by the team (a contact's source) | Vom Team hinzugefügt | a suppression added by hand is „Manuell“ |
| allowed domains | erlaubte Domains | the sites a form may be embedded on |
| archive (public page) · issue | Archiv · Ausgabe | as subscribers read them: „Newsletter-Archiv“, „Nächste Ausgabe“ |
| audience · merge field · cleaned (Mailchimp) | Zielgruppe · Merge-Feld · cleaned | Mailchimp's own words: "Zielgruppe" as its German screens say it; the status "cleaned" as its API returns it |
| batch (an import's) | Batch | the live log: „Batch 3 · 500 Zeilen“ |
| bounce types (a report's) | Hard · Soft · Blockiert | „Hard · Empfänger existiert nicht“, „Soft · Postfach voll“ |
| branding | Branding | Forms & pages › „Branding & Domain“ |
| catch-all domain · known complainers | Catch-all-Domain · Adressen mit Spam-Beschwerden | a verification service's risky groups |
| column · map fields | Spalte · Felder zuordnen | an import's step: „Felder zuordnen“ |
| confirmation link | Bestätigungslink | |
| consent · consent text · consent evidence | Einwilligung · Einwilligungstext · Einwilligungsnachweis | the GDPR's German word ("DSGVO") |
| custom domain | eigene Domain | Branding & domain |
| data center (Mailchimp) | Rechenzentrum | the API key's suffix, like us21 |
| deferral (provider) | Zurückstellung | "deferrals expired": „Zurückstellungen abgelaufen“ |
| disposable domain · role address · likely typo | Wegwerf-Domain · Rollenadresse · vermutlich Tippfehler | an import's risky groups |
| embed code · script embed · embedded (form) | Einbettungscode · Skript-Einbettung · eingebettet | |
| erase (a contact) | endgültig löschen | a GDPR erasure; a plain delete stays „löschen“ |
| excerpt | Auszug | an archived issue's |
| export data (GDPR) | Daten exportieren | |
| Forms & public pages | Formulare & öffentliche Seiten | |
| hosted page (a form type) | Seite auf ReputeCast | as "hosted" above |
| Import from {tool} (a source) | Import aus {tool} | a contact's or suppression's source: „Import aus Mailchimp“ |
| live log (an import's) | Live-Protokoll | |
| merge (a duplicate) | zusammenführen | "Merge · fills first name": „Zusammenführen · ergänzt Vornamen“ |
| Migration (where a suppression came from) | Migration | an import from another email tool |
| offer a download | einen Download anbieten | the file a form gives after confirmation |
| popup (form) | Pop-up | |
| preference center | Präferenzcenter | in the app; subscribers read „E-Mail-Einstellungen“ |
| publisher (name) | Herausgeber | heads every public page |
| re-permission | Re-Permission | "re-permission flow": „Re-Permission-Ablauf“, as German marketing says it |
| read-only link | Ansichtslink | a shared report's |
| RSS feed | RSS-Feed | |
| self-hosted (a provider) | selbst gehostet | „Postal (selbst gehostet)“ |
| sender health | Absenderzustand | the Overview's score; reputation is „Reputation“ |
| signup · submission (form) | Anmeldung · Einsendung | "Views · Submissions": „Aufrufe · Einsendungen“ |
| single opt-in | Single-Opt-in | double opt-in is „Double-Opt-in“ |
| site key · secret key (Turnstile) | Website-Schlüssel · geheimer Schlüssel | "Turnstile" stays |
| source status (an import's: subscribed, cleaned…) | as the source writes it | the consent table and the consent history keep the tool's own word |
| spam trap (audience, imports) | Spamfalle | the same word as Deliverability's |
| subscribe page · confirmation page · unsubscribe page | Anmeldeseite · Bestätigungsseite · Abmeldeseite | |
| suppressed | gesperrt | the list is „Sperrliste“ |
| System stream · System mail | Stream System · System-E-Mail | the stream's name follows the noun |
| tag | Tag | das Tag, die Tags |
| title (archived issue) | Titel | „Betreff“ is an email's subject, „Überschrift“ a heading |
| workbook · sheet (XLSX) | Arbeitsmappe · Arbeitsblatt | as Excel's German names them |

## Products, stores and transactional email

| English | Deutsch | Note |
|---|---|---|
| abandoned checkout · abandoned cart | abgebrochener Checkout · abgebrochener Warenkorb | as Shopify's German admin says it |
| accepted marketing · didn't accept | Marketing zugestimmt · nicht zugestimmt | a store customer's consent |
| attribution · credited to | Umsatzzuordnung · zugeordnet | "Attribution window": „Zuordnungszeitraum“ |
| Bought a product (trigger · condition) | Produkt gekauft · Hat Produkt gekauft | |
| buyer · customer | Käufer · Kunde | a product's buyer; a store's customer |
| checkout · at checkout | Checkout · beim Checkout | "Stripe Checkout" keeps its name |
| click token | Klick-Token | |
| collection (store) | Kollektion | |
| compare-at price | Vergleichspreis | |
| cover (product) | Titelbild | |
| delivery · delivery email · delivery template | Auslieferung · Auslieferungs-E-Mail · Auslieferungsvorlage | a digital product's download link; a delivered email is still „Zugestellt“ |
| dispute | Anfechtung | never „Beschwerde“, which is a spam complaint |
| download link · downloads | Download-Link · Downloads | "4 of 5 downloads left": „noch 4 von 5 Downloads“ |
| failover · primary · fallback | Failover · Haupt- · Ersatz- | a route's providers |
| Gross · Net | Gesamt · Nach Erstattungen | product revenue before and after refunds; not „Brutto · Netto“, which German readers take as with and without VAT |
| idempotency key | Idempotenzschlüssel | |
| list offer | Newsletter-Angebot | the checkbox next to a product's Buy button |
| live · test (payments, keys, modes) | Live · Test (as Earn has it) | the message log's live updates are „Live“ |
| median delivery | Zustellzeit (Median) | |
| message (transactional) · Messages (tab) | Nachricht · Nachrichten | |
| order | Bestellung | |
| Order placed · First order · Placed an order | Bestellung aufgegeben · Erste Bestellung · Hat bestellt | store triggers and the goal |
| Pay what you want · One-time | Freie Preiswahl · Einmalig | a product's price |
| product · digital product | Produkt · digitales Produkt | |
| Queued · Deferred · Bounced · Complained · Rejected | In Warteschlange · Zurückgestellt · Bounce · Beschwerde · Abgelehnt | a message's status; Delivered is „Zugestellt“ |
| Raw (SMTP) | Rohformat (SMTP) | mail an app sends already rendered |
| reconciliation | Abgleich | a store's nightly check |
| refund · refunded | Rückerstattung · erstattet | |
| revoke access · restore access | Zugriff entziehen · Zugriff wiederherstellen | a product order's download |
| route · Transactional route | Route · Route Transaktional | as „Route Massenversand“ |
| sales page · default product page · shop | Verkaufsseite · Standard-Produktseite · Shop | |
| send-only key · SMTP credential | Nur-Senden-Schlüssel · SMTP-Zugangsdaten | "SMTP relay" keeps its name |
| single opt-in | Single-Opt-in | double opt-in is „Double-Opt-in“ |
| store · connect · disconnect · sync | Shop · verbinden · trennen · synchronisieren | Shopify and WooCommerce; their menu paths and field names stay in English, as the setup steps name them |
| suppressed · opted out | gesperrt · abgemeldet | why a message wasn't sent |
| webhook topic | Thema | orders/create stays as it is |
| write-only · sealed | nicht auslesbar · versiegelt | a secret that's never shown again |

## Recurring digests

| English | Deutsch | Note |
|---|---|---|
| approval rule · approver | Freigaberegel · freigebende Person | as in Approvals |
| archive (public) | öffentliches Archiv | where sent issues are listed |
| back catalogue | ältere Beiträge | a feed's older posts |
| category · tag (of a post) | Kategorie · Schlagwort | WordPress's own German words |
| Collecting · Draft ready · Not sent (an issue) | Sammelt · Entwurf bereit · Nicht gesendet | an issue's own states; the others are the campaign words |
| Create a draft · Send automatically · draft mode | Entwurf erstellen · Automatisch senden · Entwurfsmodus | the two sending modes |
| excerpt · featured image | Textauszug · Beitragsbild | as WordPress's German says them |
| fallback image | Ersatzbild | |
| feed · feed address | Feed · Feed-Adresse | as Integrations; RSS 2.0, Atom 1.0 and JSON Feed 1.1 keep their names |
| feed item · Feed items (the block) | Eintrag · Feed-Einträge | |
| feed token | Feed-Platzhalter | a Liquid token: „Platzhalter“, as in Grow |
| held · release (an issue) | zurückgehalten · freigeben | "Release issue": „Ausgabe freigeben“ |
| hours before (preparation) | Stunden vor dem Versand | "{count} hours before": „{count} Std. vor dem Versand“, so it doesn't read as a time of day |
| issue · Next issue · issue #143 | Ausgabe · Nächste Ausgabe · Ausgabe #143 | as a magazine's issue, and as the public pages say it |
| late limit | Verspätungsgrenze | how late an issue may still go out |
| layout: List · Cards · Grid | Liste · Karten · Raster | the Feed items block's layouts |
| left out (by a filter) | ausgeschlossen | "Left out by a filter": „Durch einen Filter ausgeschlossen“ |
| owner (of a digest) | verantwortliche Person | the member issues send under; not the Owner role („Inhaber“) |
| percentage points (pts) | Prozentpunkte | "{delta} pts": „{delta} Pp.“ |
| post | Beitrag | |
| prepare · Prepared at 07:00 | vorbereiten · Vorbereitet um 07:00 | |
| recurring digest · digest | wiederkehrender Newsletter | as Integrations; the Recurring tab is „Wiederkehrend“ |
| Regular (campaign) | Standard | the other choice in New campaign |
| sending window | Versandfenster | as Automations' send window |
| skip · skipped | überspringen · übersprungen | |
| SSRF guard | SSRF-Schutz | |
| starter (a template to begin from) | Startvorlage | a starter's name ("Blog digest") stays as it is; what a new digest's starter writes into the email (subject, preheader) follows the workspace's content language |
| starting point | Ausgangspunkt | |
| turn on · pause (a digest) | aktivieren · pausieren | |
| weekdays (Monday to Friday) | Montag bis Freitag | short: „Mo–Fr“, the region format's own abbreviations |

## AI assistant

| English | Deutsch | Note |
|---|---|---|
| accept · edit · reject (a suggestion) | übernehmen · bearbeiten · verwerfen | |
| Accepted with edits · No decision | Mit Änderungen übernommen · Keine Entscheidung | outcomes in the activity log |
| activity log (AI) | KI-Aktivitätsprotokoll | the audit log stays „Audit-Protokoll“ |
| AI assistant · Assist (the menu) | KI-Assistent · Assistent | "KI", never "AI" |
| AI budget | KI-Budget | |
| cap · monthly cap | Limit · Monatslimit | in dollars or tokens; "No cap": „Kein Limit“ |
| condition schema · segment builder | Bedingungsschema · Segment-Editor | |
| deployment (Azure) | Bereitstellung | Azure's German word |
| describe the segment · Draft (its button) | Segment beschreiben · Entwerfen | what it drafts is a „Entwurf“ |
| feature (of the assistant) | Funktion | |
| Image input · Structured output | Bildeingabe · Strukturierte Ausgabe | what a model can do |
| input · output (tokens, prices) | Input · Output | „Ausgabe“ is a newsletter's issue |
| instruction (to rewrite) | Anweisung | "Rewrite with an instruction": „Mit Anweisung umschreiben“ |
| keep · discard (a summary) | behalten · verwerfen | "kept summaries": „behaltene Zusammenfassungen“ |
| model · connect a model | Modell · Modell verbinden | model names stay as they are |
| private address | private Adresse | a server on a private network |
| prompt · test prompt | Prompt · Test-Prompt | |
| refused by the model | vom Modell abgelehnt | |
| resets (the usage) | wird zurückgesetzt | "resets {date}": „wird am {date} zurückgesetzt“ |
| rewrite · shorten · friendlier · more formal · more direct | umschreiben · kürzen · freundlicher · förmlicher · direkter | |
| subject ideas | Betreffideen | |
| suggestion | Vorschlag | |
| summary (of results) | Zusammenfassung | "Summarize results": „Ergebnisse zusammenfassen“ |
| token (of a model) | Token | what the provider bills; a Liquid token is „Platzhalter“ |
| tone | Tonfall | "Rewrite, shorten and tone": „Umschreiben, kürzen, Tonfall ändern“ |
| usage | Nutzung | |

## Placement tests

| English | Deutsch | Note |
|---|---|---|
| app password · app-specific password | App-Passwort · app-spezifisches Passwort | Google's and Apple's own words |
| arrived late | verspätet angekommen | |
| copy (of the test email) | Kopie | |
| deferral | Zurückstellung | as Campaigns' „Zurückstellung“ |
| folder · tab | Ordner · Tab | a mailbox's own names (Promotions, Junk Email) stay as it gives them |
| folders: Sent · Draft · Archive · Trash | Gesendet · Entwürfe · Archiv · Papierkorb | the folders a test never reads |
| hand-off | Übergabe | when the sending provider takes a copy |
| headers | Header | "All headers": „Alle Header“; header names stay |
| Health (Deliverability › Health) | Zustand | as Integrations' Health |
| Healthy · Warming up · Degraded · Down · Paused · Sandbox (a provider) | Fehlerfrei · Im Warm-up · Beeinträchtigt · Ausgefallen · Pausiert · Sandbox | |
| Inbox · Tab · Spam · Not arrived | Posteingang · Tab · Spam · Nicht angekommen | the four results |
| median time to arrive | Ankunftszeit (Median) | a median, as Approvals' median wait |
| pass · fail · aligned (authentication) | bestanden · nicht bestanden · ausgerichtet | SPF, DKIM and DMARC keep their names |
| placement · placement test | Platzierung · Platzierungstest | where a copy lands |
| Queued (a copy) | In Warteschlange | |
| remove (a seed) | entfernen | deleting copies stays „löschen“ |
| round (of copies) | Runde | "Round 2": „Runde 2“ |
| sample (the seeds) | Testgruppe | as an A/B test's sample |
| seed · seed mailbox | Testpostfach | mailboxes that exist only for tests |
| stop waiting | nicht mehr warten | |
| test (one run) | Test | "Tests · 12" |
| test token | Test-Token | what marks a copy as a test's |
| vs (providers compared) | vs. | „SES vs. Postal“ |
| warm-up cap | Warm-up-Limit | a warming-up provider's daily cap |
| warm-up planner · warm-up plan | Warm-up-Planer · Warm-up-Plan | |

## Address verification

| English | Deutsch | Note |
|---|---|---|
| abuse · catch-all domain · disposable · role address | Beschwerderisiko · Catch-all-Domain · Wegwerfadresse · Rollenadresse | why an address is risky; tags such as verify:catch-all stay as written |
| Add credits · Low credits | Guthaben aufladen · Wenig Guthaben | |
| balance · credits | Guthaben · Credits | the prepaid credits at the service |
| bulk check · real-time check | Massenprüfung · Echtzeitprüfung | |
| check (one address) | Prüfung | "{price} a check": „{price} pro Prüfung“ |
| data region | Datenregion | |
| Found by | Gefunden durch | which check found a risky address |
| imports (of contacts) | Importe | "Bulk checks and imports": „Massenprüfungen und Importe“ |
| job · Resume | Auftrag · Fortsetzen | |
| spam trap | Spamfalle | |
| Uses (of a service) | Nutzungen | |
| Valid · Invalid · Risky · Unknown | Gültig · Ungültig · Riskant · Unbekannt | the four results |
| verification service · verify | Prüfdienst · prüfen | a domain is „verifiziert“ |

## Words used everywhere

| English | Deutsch |
|---|---|
| contact | Kontakt |
| subscriber | Abonnent (die Abonnenten); "Kontakt" when any contact is meant |
| list | Liste |
| segment | Segment |
| template | Vorlage |
| sending provider | Versandanbieter |
| mailbox provider | Postfachanbieter |
| sender | Absender |
| domain | Domain |
| record (DNS) | Eintrag ("DNS-Eintrag", "DKIM-Eintrag") |
| verified | verifiziert |
| sign in · sign out | anmelden · abmelden |
| password · passkey | Passwort · Passkey |
| authenticator app | Authenticator-App |
| recovery code | Wiederherstellungscode |
| two-factor | Zwei-Faktor-Authentifizierung (2FA) |
| invitation · invite | Einladung · einladen |
| permission | Berechtigung |
| Ask Marcus Chen or Linh Nguyen. | Fragen Sie Marcus Chen oder Linh Nguyen. |
| notification | Benachrichtigung |
| approval | Freigabe |
| retry · try again | Erneut versuchen |
| cancel · close · back | Abbrechen · Schließen · Zurück |
| copy · copied | Kopieren · Kopiert |
| optional · required | optional · erforderlich |
| audience | Zielgruppe |
| subscribe · subscription (to a newsletter) | abonnieren · Abonnement |
| unsubscribe · unsubscribes · unsubscribe rate | abmelden · Abmeldungen · Abmelderate |
| open rate · click rate | Öffnungsrate · Klickrate |
| draft · scheduled | Entwurf · geplant |
| suppression list | Sperrliste |
| overview | Übersicht |

"Abmelden" is both signing out of ReputeCast and unsubscribing from a
newsletter, as in every German email tool; the screen tells them apart.
Where one sentence could mean either, say "vom Newsletter abmelden" or
"von ReputeCast abmelden". A paid subscription is cancelled ("kündigen"),
never "abgemeldet".
