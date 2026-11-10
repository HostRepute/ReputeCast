# French glossary (Français)

The words ReputeCast uses in French, so every screen, email and error
says the same thing the same way. It covers the Vietnamese glossary's
terms section by section, in the vocabulary French-speaking marketers
already know from the French interfaces of Mailchimp, Brevo and HubSpot.

> **Status: draft.** A native speaker reviews these words before French
> ships in a release (brief §8.17). Change a term here first, then in the
> translations (`web/src/**/messages.fr.ts`,
> `internal/<package>/messages/fr.json`).

How to add or complete a language: [`translating.md`](translating.md).

## Voice

- Plain and specific, like the English: say what happened and what to do.
  No exclamation marks, no jokes ("Oups"), and no "Veuillez" where the
  imperative says it: "Réessayez dans quelques minutes."
- Standard French as written in France, readable in Belgium, Switzerland
  and Canada. Avoid words only one country uses, and clipped words: write
  "e-mail" (plural "e-mails", "adresse e-mail"; the channel is "E-mail"),
  not "mail", "email" or "courriel"; "importer", not "téléverser"; never
  "appli", "dispo", "récap".
- Address the reader as **vous**, never "tu". Instructions use the
  imperative: "Choisissez une langue."
- Buttons, menu items and links that do something use the **infinitive**:
  "Enregistrer", "Envoyer la campagne", "Ajouter un contact",
  "Se déconnecter". Never the imperative ("Enregistrez") or a noun
  ("Enregistrement") on a control. A confirmation's title is an infinitive
  question: "Supprimer la campagne ?". What has happened takes a
  participle: "Campagne programmée", "Modifications enregistrées". An empty
  state: "Aucune campagne pour l’instant".
- **Sentence case**, as in English: only the first word and names take a
  capital. "Nouvelle campagne", not "Nouvelle Campagne". Days and months
  are lower case: "lundi", "octobre". A capital keeps its accent: "État",
  "À venir", "Échec".
- Overlines (table headers, group labels) show in capitals and keep their
  accents: "DESTINATAIRES", "ÉTAT", "DERNIÈRE ACTIVITÉ". Write them as the
  English source does (usually sentence case: "État"); the app sets the
  capitals.
- **Typography.** French puts a space before two-part punctuation, and that
  space must never break at the end of a line:
  - U+202F, the narrow no-break space (*espace fine insécable*), before
    `;` `!` `?`: "Supprimer la campagne ?"
  - U+00A0, the no-break space (*espace insécable*), before `:` and inside
    guillemets: "Langue : Français", "« October Product Update »"; also
    between a number you write and its unit or sign: "2 h", "102 Ko",
    "n° 143".
  - In the sources, write them as the escapes `\u202F` and `\u00A0` (the
    same in a `.ts` string and in JSON), never as a plain space or
    `&nbsp;`: `'Supprimer la campagne\u202F?'`,
    `"Langue\u00A0: {language}"`. The French examples in this glossary
    already carry the right characters; copy them as they are.
  - No space before `.` `,` `…` `)` or after `(`.
- **Apostrophe: always ’** (U+2019), in every elision: "l’audience",
  "aujourd’hui", "n’est pas". Never `'` in text. Besides matching the
  English sources, it keeps ICU safe: a `'` just before `{` or `#` starts
  an ICU quote, so `l'{name}` would print `{name}` as it is.
- **Quotations** in running text take « » with U+00A0 inside: quoted words,
  button names, workspace content ("la campagne « October Product Update »"),
  where the English has “ ” or " ". A quotation inside a quotation takes
  “ ”.
- **Numbers, dates and times** come from the region format, never typed by
  hand: fr-FR writes 187 312 · 99,2 % · 18,21 $US · mar. 14 oct. 2026 ·
  09:00. Write `{count}`, `{date}`, `{percent}`, and add no space of your
  own inside or around what the format writes.
- Relative times: "à l’instant", "il y a 4 min", "il y a 2 h", "hier",
  "dans 5 jours". Units without a period: "min", "h", "s", "Ko", "Mo"
  (KB, MB). "#" before a number is "n°": "n° 143", "{list} n° {issue}".
- **Plurals.** French has the CLDR categories `one`, `many` and `other`.
  `one` covers 0 and 1, so 0 is singular: "0 campagne", "1 campagne",
  "2 campagnes". Write `{count, plural, one {# campagne} other {# campagnes}}`,
  and let the words that agree follow: `one {# campagne envoyée} other
  {# campagnes envoyées}`. `many` is only for round millions ("1 000 000 de
  campagnes"): leave it out, it falls back to `other`. Keep `=0 {…}` where
  the English has it ("Aucune campagne"), and keep `#` rather than a bare
  `{count}`, so the number is grouped by the region format.
- **Everyone, without midpoints.** No inclusive-writing midpoints, slashes
  or brackets ("utilisateur·rice", "invité(e)", "abonné/e"). Prefer words
  that don't mark gender: "personnes", "membres de l’équipe", "contacts",
  "destinataires", "responsable", "propriétaire", "admin". Where a role or
  a fixed term has a gender (abonné, approbateur, éditeur, observateur,
  parrain), use the masculine form, as French software does. Don't make a
  sentence agree with a person: "Invitation envoyée à {name}", not
  "{name} a été invité".
- **Words before an argument.** The app can't elide or agree for a value
  it fills in: "de {name}" gives "de Acme" where French writes "d’Acme",
  and "le {list}" can't know the name's gender. Bring a value in with a
  noun, or a word that never elides: "la campagne « {name} »",
  "la liste {list}", "par {name}", "pour {name}", "l’équipe {publisher}",
  or after a colon. Numbers are safe: "sur {total}".
- **Language names.** `{language}` and `{languages}` come as names: with a
  capital where they stand alone or start a sentence, which includes after
  a colon or "·" ("Anglais", "Langue : Anglais"; a list capitalises its
  first name only: "Anglais, espagnol et allemand"), and in lower case
  once the app sees the argument inside a sentence ("Version en anglais").
  The app sets the case; write `{language}` as it is. French needs an article
  before a language name ("l’anglais", "le français") and the article
  depends on the name, so never write "le {language}", "l’{language}",
  "du {language}" or "de {language}". Use a construction that needs none:
  - "en {language}": "Version en {language}" (Version en anglais),
    "Lecteurs en {language}", "Traduire en {language}",
    "Texte alternatif en {language}", "Contacts en {language} : {count}".
  - "{language} :" at the start of a label: "{language} : manquante".
  - "langue : {language}" after a colon: "Langue par défaut : {language}",
    rather than "{language} est la langue par défaut".
  - Add {language} → "Ajouter la langue : {language}"; Remove {language}? →
    "Retirer la langue : {language} ?"; {language} added. → "Langue
    ajoutée : {language}."; {language} version → "Version en {language}";
    {language} readers get {fallback} → "Les lecteurs en {language}
    reçoivent la version en {fallback}"; translated from {language} →
    "traduite de la version en {language}".
  - To name the language again in the same message, say "cette langue":
    "Ajoutez de nouveau cette langue pour retrouver ses versions."
- **Length.** French runs about 15–30% longer than English. Stay within
  about 30% of the English where you can: buttons and badges don't wrap,
  table headers get two lines (`docs/dev/CONVENTIONS.md` › "Text
  expansion"). Where this glossary gives a short form ("short: …"), use it
  on buttons, tabs, badges and table headers, and the full term in
  sentences. Don't abbreviate with a period ("Param."); drop words instead.

## Never translated

Workspace content (campaign, list, segment, template, form and custom role
names such as "Content Writer"), people's names and email addresses,
permission keys (`campaigns:send`), audit action names, API error codes,
provider names and codes (Amazon SES, Postal, Postmark, SES), product names
(ReputeCast, HostRepute, Gmail, Outlook), DNS records and record types (SPF,
DKIM, DMARC, BIMI, CNAME, MAIL FROM), headers (List-Unsubscribe), code
samples, CLI output, "MCP", "2FA", "TOTP", "IdP", "SSO", "JIT", "A/B",
"HTML", "CSS", "API", "webhook".

Their gender in a French sentence: "l’API" and "la 2FA" are feminine; "le
webhook" (plural "webhooks"), "le SSO", "l’IdP" and "le MCP" are
masculine. "GDPR" is the French "RGPD".

## App shell and navigation

| English | Français |
|---|---|
| Overview | Vue d’ensemble |
| Campaigns | Campagnes |
| Automations | Automatisations |
| Transactional | Transactionnel |
| Audience | Audience |
| Grow | Croissance |
| Earn | Monétisation |
| Forms & pages | Formulaires et pages ("&" is "et" in French labels) |
| Deliverability | Délivrabilité |
| Sending providers | Fournisseurs d’envoi |
| Settings | Paramètres |
| More | Plus |
| Skip to content | Aller au contenu |
| Import contacts | Importer des contacts |
| HostRepute connected | HostRepute connecté |
| synced 4 min ago | synchronisé il y a 4 min |
| Account & security | Compte et sécurité |
| Notification preferences | Préférences de notification |
| Workspace settings | Paramètres de l’espace de travail (short: Paramètres de l’espace) |
| Instance admin | Administration de l’instance (short: Administration) |
| admins only | admins uniquement |
| Sign out | Se déconnecter |
| Language: English | Langue : Français (the item names the current language in its own name) |
| unexpected error · server log | erreur inattendue · journal du serveur |

## Settings tabs

| English | Français |
|---|---|
| General | Général |
| Senders | Expéditeurs |
| Providers | Fournisseurs |
| Team | Équipe |
| Roles | Rôles |
| Approvals | Approbations |
| Keys & MCP | Clés et MCP |
| Webhooks | Webhooks |
| Integrations | Intégrations |
| Audit log | Journal d’audit |

## Roles and people

| English | Français |
|---|---|
| Owner | Propriétaire |
| Admin | Admin |
| Editor | Éditeur (the role; a publication's publisher is also "éditeur", on public pages only) |
| Viewer | Observateur (not "Lecteur": "lecteurs" are a newsletter's readers) |
| custom (role) | personnalisé |
| instance admin (the area, a label: "Instance admin", "· instance admin") | administration de l’instance |
| instance admin (a person, in a sentence: "Ask your instance admin") | admin de l’instance ("Demandez à l’admin de votre instance") |
| break-glass account | compte d’urgence |
| member | membre |
| invitee | personne invitée |
| workspace | espace de travail (short: espace) |
| instance | instance |
| team | équipe |

## Statuses

A status agrees with the noun it describes: campagne, version,
approbation, formule and adresse are feminine ("Envoyée", "Programmée");
membre, compte, fournisseur, contrôle, message and abonnement are
masculine ("Actif", "Invité"). "Brouillon", "Échec" and "Avertissement"
are nouns and never change.

| English | Français |
|---|---|
| Draft | Brouillon |
| Draft · blocked | Brouillon · bloqué |
| Scheduled | Programmée |
| Sending | Envoi en cours (short: En cours) |
| Sent | Envoyée |
| Active | Actif (a member, a provider); Active (an automation, a route) |
| Invited | Invité |
| Deactivated | Désactivé |
| On · Off | Activé · Désactivé |
| Via IdP | Via l’IdP |
| Required | Obligatoire |
| Pass · Warn · Block | Réussi · Avertissement · Bloquant |
| Ready · Ready with warnings · Blocked | Prête · Prête, avec avertissements · Bloquée |
| Healthy | Opérationnel |
| Warming up | En chauffe |
| New | Nouveau (Nouvelle for a campaign, a route, a version) |
| Not saved | Non enregistré |
| Development | Développement |
| Auto-paused | En pause auto (in a sentence: "mise en pause automatiquement") |
| Pending approval · Approved | En attente d’approbation · Approuvée (short: À approuver) |
| Cancelled · Failed | Annulée · Échec |

## Campaigns

| English | Français |
|---|---|
| Campaign | Campagne |
| New campaign | Nouvelle campagne |
| All · Drafts · Scheduled · Sending · Sent | Toutes · Brouillons · Programmées · Envoi en cours · Envoyées |
| Search campaigns | Rechercher des campagnes |
| Sent · last 30 days | Envois · 30 derniers jours |
| Cost · last 30 days | Coût · 30 derniers jours |
| Recipients | Destinataires |
| Delivered | Délivrés |
| Human opens | Ouvertures humaines |
| Clicks | Clics |
| click-to-open | taux de réactivité (clicks ÷ opens: "réactivité {cto}") |
| bounces · complaints | rebonds · plaintes |
| machine opens shown separately | ouvertures automatiques comptées à part |
| Readiness | Contrôle (the result of the contrôle avant envoi) |
| 3 blocking issues | 3 problèmes bloquants |
| Edit · Report · Duplicate | Modifier · Rapport · Dupliquer |
| not set | non défini |
| created 08:55 | créée à 08:55 |
| edited 09:31 by Linh Nguyen | modifiée à 09:31 par Linh Nguyen |
| All subscribers | Tous les abonnés |
| Broadcasts route | Route Diffusions |
| A/B subject, winner B | Objet A/B, gagnant : B |
| auto-paused once, Yahoo deferrals | mise en pause automatiquement une fois, reports Yahoo |
| Showing 9 of 48 | 9 sur 48 affichées |
| Previous · Next | Précédent · Suivant |
| Live monitor | Suivi en direct |
| imported from Brevo · Brevo · imported | importée depuis Brevo · Brevo · importée |
| segment (in an audience: 2 segments) | segment (2 segments) |

## Review & send

| English | Français |
|---|---|
| Review & send | Vérifier et envoyer |
| Back to editor | Retour à l’éditeur |
| Run checks again | Relancer les contrôles |
| Draft · autosaved 2 min ago | Brouillon · enregistré automatiquement il y a 2 min |
| Fix both warnings | Corriger les deux avertissements |
| Schedule send to 31,604 recipients | Programmer l’envoi à 31 604 destinataires |
| 27 checks passed, 2 warnings, 0 blocks | 27 contrôles réussis, 2 avertissements, 0 blocage |
| Sender reputation | Réputation de l’expéditeur |
| blocklist | liste de blocage |
| suppression list (shared) | liste de suppression (partagée) |
| role addresses | adresses génériques |
| Duplicates | Doublons |
| Final recipients | Destinataires finaux |
| Route split | Répartition entre routes |
| Content · Links · Compliance | Contenu · Liens · Conformité |
| Audience quality | Qualité de l’audience |
| Routing · stream | Routage · flux |
| Broadcasts stream | Flux Diffusions |
| Send time · Send now · Schedule | Heure d’envoi · Envoyer maintenant · Programmer |
| Best time per recipient | Meilleure heure par destinataire |
| Test email · Send test | E-mail de test · Envoyer un test |
| Auto-pause thresholds | Seuils de pause automatique |
| Bounce rate · Complaint rate | Taux de rebond · Taux de plainte |
| per mailbox provider | par fournisseur de messagerie (short: par messagerie) |
| warm-up (day 9 of 21) | chauffe (jour 9 sur 21) |
| 14 /s | 14/s |
| Cost · Time | Coût · Durée |

### Send Readiness's checks

The findings the server writes for each check (`internal/readiness`).
Send Readiness is the "contrôle avant envoi", and one of its checks a
"contrôle".

| English | Français | Note |
|---|---|---|
| check (a row) · passed · not checked · timed out | contrôle · réussi · non vérifié · délai dépassé | |
| the verdict waits for this one | le verdict attend ce contrôle | |
| I understand … (a warning's confirmation) | Je comprends… | as the web's "I understand: {title}.": "Je comprends : {title}." |
| unengaged (contacts) | non engagés | no open or click in the sunset policy's days |
| Sunset tier · At risk (names) | niveau Arrêt progressif · À risque | tier names keep their capitals |
| spam trigger · spam heuristic | signal de spam | "trips 3 spam heuristics": "déclenche 3 signaux de spam" |
| all caps · repeated punctuation · urgency phrasing · money words · click-bait | tout en majuscules · ponctuation répétée · formules d’urgence · termes liés à l’argent · piège à clics | |
| image-to-text ratio | ratio image/texte | |
| Gmail's 102 KB clipping limit · clips | limite de troncature de Gmail (102 Ko) · tronque | KB is "Ko" |
| inline(d) CSS · minify | CSS en ligne · minifier | |
| one-click unsubscribe (headers) | (en-têtes de) désabonnement en un clic | header names stay as they are |
| postal address | adresse postale | |
| bulk-sender rules | règles pour les expéditeurs de gros volumes | Gmail and Yahoo's |
| URL shortener · redirects through | raccourcisseur d’URL · redirige via | |
| responds · doesn't respond | répond · ne répond pas | a link |
| review thresholds · pause level (SES) | seuils d’examen · niveau de suspension | |
| no enforcement | aucune mesure appliquée | an account's standing |
| terms allow bulk marketing | les conditions autorisent le marketing de masse | |
| production access | accès production | as in Sending |
| double-tracked · rewrites links | suivi en double · réécrit les liens | |
| aligned (DMARC, SPF) | aligné | |
| Easy DKIM CNAME · selector | CNAME Easy DKIM · sélecteur | Easy DKIM keeps its name |
| custom MAIL FROM | MAIL FROM personnalisé | |
| Conditional content · conditional block · version | Contenu conditionnel · bloc conditionnel · version | |
| Visibility rule | règle de visibilité | |
| poll · survey · answer page | sondage · questionnaire · page de réponse | |
| risky · catch-all · disposable (addresses) | à risque · catch-all · jetable | |
| inactive (contacts) | inactifs | At risk and Sunset together |
| paid members · trial · complimentary · gift (access) | abonnés payants · essai · offert · cadeau | as Earn has it |
| Sponsor block · creative · booking · Sponsored (label) | bloc Sponsor · création · réservation · Sponsorisé | |
| Upgrade block · offer | bloc Passage au payant · offre | |
| page copy · preference center | textes de la page · centre de préférences | |
| bare URL · skipped heading level | URL brute · niveau de titre sauté | accessibility |
| linked image | image avec lien | |
| self-hosted (pricing) · plan | auto-hébergé · selon le forfait | a provider's price |

## Sending and deliverability

Sending providers & routing, the connect wizard, Settings › Senders and
Providers, Deliverability, HostRepute and the live monitor. In alphabetical
order of the English.

| English | Français | Note |
|---|---|---|
| account health (a provider's) | santé du compte | |
| adapter · community adapter · adapter registry | adaptateur · adaptateur communautaire · registre d’adaptateurs | |
| aggregate report (DMARC) | rapport agrégé | |
| alignment (DMARC) | alignement | "alignement DMARC" |
| asset (a domain or IP HostRepute monitors) | ressource | never "actif", which reads as money |
| backfill (statuses from an API) | rattrapage | "Rattraper les statuts depuis l’API Mailgun" |
| blocklist · listed · clean | liste de blocage · listé · non listé | not "liste noire" |
| bounce mailbox | boîte de réception des rebonds | SMTP providers |
| bounce · hard bounce · soft bounce | rebond · rebond définitif · rebond temporaire | the rate is "taux de rebond"; as in the Grow section |
| Broadcasts · Automations · System · Transactional (streams) | Diffusions · Automatisations · Système · Transactionnel | names, so capitalised in a sentence: "le flux Système" |
| bulk marketing | marketing de masse | a provider's terms |
| cap (daily, today's) | plafond | "Plafond du jour" |
| catalog (of providers) | catalogue | the wizard's first step |
| condition (routing) | condition | |
| credentials (a provider's) | identifiants | |
| dedicated IP · shared pool | IP dédiée · pool d’IP partagé | |
| defer · deferral | différer · report | "Yahoo diffère la livraison"; the status is "Différé" |
| degraded · down | dégradé · en panne | provider statuses |
| delisting | délistage | "Demander le délistage" |
| event log · raw event · normalized event | journal des événements · événement brut · événement normalisé | |
| failover · fallback · primary | basculement · secours · principal | "fournisseur de secours", "fournisseur principal"; a fallback language or value is "de repli" |
| feedback loop | boucle de rétroaction | |
| flow (an automation's, in sending text) | scénario | "flux" is a stream: "les e-mails envoyés par des scénarios automatisés"; on the Automations screens a flow is "scénario" too |
| guard (Warm-up page) | seuil de protection | "Taux de rebond · seuil de protection 2 %" |
| held (recipients) | retenus | |
| incident · outage | incident · panne | |
| live sending monitor | suivi des envois en direct | "Live monitor" is "Suivi en direct" |
| maintainer · unmaintained (adapter) | mainteneur · non maintenu | |
| mapping (webhook, JSONPath) | correspondance | not "mappage" |
| message id | ID du message | |
| monitoring · monitored | surveillance · surveillé | HostRepute |
| nameserver | serveur de noms | |
| polling (instead of webhooks) | interrogation périodique | "interroge l’API des événements toutes les minutes" |
| Postmaster compliance status | état de conformité | Google Postmaster Tools v2 has no domain or IP reputation |
| production access (SES) | accès production | the SES sandbox stays "sandbox" |
| provisioning | provisionnement | SNS topics and configuration sets |
| pulled in · pushed out · two-way (suppression sync) | importées · exportées · bidirectionnelle | addresses, so feminine ("Importées" as a column); the sync is "bidirectionnelle" |
| quota | quota | |
| read-only link (a live send) | lien en lecture seule | |
| recommended (an option, a badge) | recommandé | "IMAP (recommandé)"; a recommendation's heading is "Recommandation" |
| region (AWS) | région | AWS's own word; the region format is "format régional" |
| reputation · domain reputation · IP reputation | réputation · réputation du domaine · réputation de l’IP | |
| reroute | rerouter | |
| reverse DNS | DNS inverse | |
| route tester | testeur de routes | |
| route · routing · weighted split | route · routage · répartition pondérée | |
| secret (webhook signing) | secret | "secret de signature du webhook" |
| seed list | liste témoin | one address of yours at each mailbox provider |
| self-test (of an endpoint) | autotest | |
| Send Readiness | Contrôle avant envoi | lower case inside a sentence: "le contrôle avant envoi" |
| share (of a split · of a send) | part | |
| slowdown (warm-up) | ralentissement | |
| SNDS filter result: Green · Yellow · Red | Vert · Jaune · Rouge | |
| spam rate · spam trap | taux de spam · piège à spam | |
| spam trap hit | piège à spam touché | |
| stale (data) · cached results | obsolète · résultats en cache | |
| stream | flux | in Digests and on the website, "flux" is a feed: say "flux d’envoi" if both could be meant |
| suppression sync | synchronisation de la liste de suppression | |
| throttle · throttling | limitation de débit | |
| tracking · tracking domain | suivi · domaine de suivi | |
| verdict (HostRepute): Good · Fair · Poor · Bad | Bon · Moyen · Faible · Mauvais | |
| warm-up · warmed up · warm-up guard · ramp | chauffe · chauffe terminée · protection de chauffe · montée en charge | "Warming up" is "En chauffe" |
| webhook endpoint · registration | endpoint du webhook · enregistrement | |
| write-only (a key) | en écriture seule | "it can be replaced but never read back" |
| zone file | fichier de zone | |

"sandbox", "configuration set", "SNS topic", "IAM", "ARN", "CloudFormation",
"stack", "return-path", "DSN", "ARF", "rua", "JSONPath", "manifest.json",
"SDK", "tunnel", "reverse proxy", "endpoint", "idempotency", "Postmaster",
"SNDS" and "JMRP" stay as they are (masculine: "le sandbox", "l’endpoint"),
and so do the AWS field names ("Access key ID", "Secret access key",
"RoleArn"), the SES region names, a provider's own stream names (Postmark's
"Broadcast" stream: "le flux Broadcast") and the results in DNS records and
reports ("pass", "fail", "p=quarantine"). An API or HostRepute token is a
"jeton".

### What adapters and the connect flow write

The server's words about a provider: the catalog's cards and chips, an
adapter's fields and terms, the account's standing, the stepper and the
test's timeline.

| English | Français | Note |
|---|---|---|
| Production · Sandbox (an account's standing) | Production · Sandbox | "Production · sans restriction" |
| Two-way · Pull only · Push only · Local only (suppression sync) | Bidirectionnelle · Import seul · Export seul · Locale uniquement | the Adapters tab; agrees with "synchronisation" |
| self-hosted | auto-hébergé | "Postal (auto-hébergé)" |
| Any SMTP server · Generic HTTP adapter | Tout serveur SMTP · Adaptateur HTTP générique | catalog names |
| Batch 50 · Quota discovery · Sandbox detection (chips) | Lots de 50 · Détection du quota · Détection du sandbox | |
| Dedicated IPs, paid · Pro plan | IP dédiées, payantes · forfait Pro | the plan's name stays |
| read-only call | appel en lecture seule | "vérifié par un appel en lecture seule" |
| Round trip 2.4 s | Aller-retour 2,4 s | the test send |
| Mailpit (development) | Mailpit (développement) | |
| a price in an adapter's words | 0,10 $US pour 1 000 e-mails | decimal comma, U+202F between thousands, U+00A0 before $US |

A provider's names for its keys, plans, menus and stream types stay as the
provider writes them ("Server API token", "Private API key", "Account SID",
"Auth token", "Full access", "Broadcasts API", "Scale plan"), and so do its
event names ("Delivery", "SpamComplaint") and its error codes.

### What the sending engine, Deliverability and HostRepute write

The live monitor's incident panel (likely causes, recommendations and the
ways to continue), its notifications and feed, Deliverability's account
health and check runs, the HostRepute connection, route notes on a
contact's timeline and why a transactional message failed.

| English | Français | Note |
|---|---|---|
| "October Product Update" auto-paused · Outlook | « October Product Update » mise en pause automatiquement · Outlook | a campaign's name between « », agreeing with "campagne" |
| built-in daily check (blocklists) | contrôle quotidien intégré | without HostRepute |
| check run · lookups that didn't answer | exécution du contrôle · requêtes sans réponse | |
| connection errors · throttling errors · API errors | erreurs de connexion · erreurs de limitation de débit · erreurs d’API | "Postal renvoie des erreurs de connexion depuis 3 min" |
| decision requested · waiting for a decision | décision demandée · en attente de votre décision | a held share |
| health check · still failing · answering again | vérification de l’état · toujours en échec · répond de nouveau | the outage log's lines |
| never-engaged · lapsed (contacts) | jamais engagés · désengagés | |
| placement ("protects placement") | placement en boîte de réception | as in Placement tests |
| Reputation dashboard (SES) | Tableau de bord de la réputation | |
| Resume as is · Resume with changes · Cancel the Outlook share · Retry Postal · Move 1,204 to SES | Reprendre tel quel · Reprendre avec des modifications · Annuler la part Outlook · Réessayer Postal · Transférer 1 204 vers SES | the incident panel's ways to continue; "(recommandé)" after the recommended one |
| route note: SES share of the 90/10 split · by rule · instead of · failed over to | SES : part de la répartition 90/10 · par règle · au lieu de · basculé vers | a provider after a colon or as a label, never after "de" |
| scope (a HostRepute token's) | portée | the scopes' keys stay: monitor:read |
| Send held (a scheduled send) | Envoi retenu | |
| status page (a provider's) | page d’état | |
| web process (a self-hosted provider's) | processus web | |

The providers' own replies stay as they come: SMTP codes and texts
("421 4.7.0 TSS04"), API errors, the outage log's error lines
("ECONNREFUSED mta.acme.io:443") and a bounce's diagnostic.

## Transactional templates

| English | Français |
|---|---|
| transactional template · transactional email | modèle transactionnel · e-mail transactionnel |
| draft · publish · v3 live | brouillon · publier · v3 en ligne |
| version · Publish again | version · Republier |
| variable · declare · required · optional | variable · déclarer · obligatoire · facultative |
| Sensitive | Sensible |
| sample value · default | valeur d’exemple · par défaut |
| test data (set) | (jeu de) données de test |
| render · test renders | rendu · rendus de test |
| Checks · Ready to publish | Contrôles · Prêt à publier |
| breaking change | changement incompatible |
| Subject · Preheader | Objet · Texte d’aperçu |
| canvas | canevas |
| Visual · Text (views) | Visuel · Texte |
| plain text | texte brut |
| call (an API call) | appel |
| Essential · Optional (categories) | Essentiels · Facultatifs |
| autosaved 09:41 · conflict | enregistré automatiquement à 09:41 · conflit |
| Import templates · Needs a manual fix | Importer des modèles · Correction manuelle requise |

"payload", "Liquid", "MJML", "Mustachio", "Handlebars", "cURL" and "JSON"
stay as they are.

## Composer

| English | Français |
|---|---|
| A/B test · variant | test A/B · variante |
| block · Blocks panel | bloc · Blocs |
| code view | vue code |
| column · 2 columns | colonne · 2 colonnes |
| compile · compiled HTML | compiler · HTML compilé |
| composer | éditeur ("Quitter l’éditeur ?") |
| dark mode · light | mode sombre · clair |
| footer · header (of an email) | pied de page · en-tête |
| Inspector | Propriétés (the panel) |
| layer · layer tree | calque · arborescence des calques |
| padding · border radius | marge intérieure · arrondi des angles |
| personalization token · Liquid tag | variable de personnalisation · balise Liquid |
| raw HTML | HTML brut |
| sample · winner (A/B) | échantillon · gagnant |
| section (of an email) · saved section | section · section enregistrée |
| Send Readiness | Contrôle avant envoi (the checks before a send, as a name; "runs again" = "est relancé") |
| spam heuristic | signal de spam |
| bare URL · link text · says nothing (a link's text) | URL brute · texte du lien · non explicite |
| broken link · shortened link | lien cassé · lien raccourci (a URL shortener is Send Readiness's "raccourcisseur d’URL") |
| inline CSS | CSS en ligne (Gmail's clipping limit is Send Readiness's "limite de troncature") |
| contact field · personalization token's fallback | champ de contact · valeur de repli |
| plain-text version · postal address | version texte brut · adresse postale |
| Line 3: (a compile or layout problem) · doesn't compile | Ligne 3 : · ne compile pas |
| title (of an article card, a post) | titre (an email's subject is always "objet") |
| Email preferences link · View in browser link | Lien vers les préférences e-mail · Lien vers la version en ligne |
| starter (template) | modèle de départ |
| starters: Blank · Weekly digest · Product update · Plain text · Announcement · Product grid · Blog digest | Vierge · Récapitulatif hebdomadaire · Nouveautés produit · Texte brut · Annonce · Grille de produits · Récapitulatif du blog |

MJML tags (mj-section, mj-body) stay as they are. Section names in the
server's sentences use the composer's words: en-tête, pied de page, hero,
2 colonnes, texte, image, bouton. A block's sample content is written for
the email's readers, in the app language: the first-name fallback "there"
is "à vous" ("Bonjour à vous").

## Automations

| English | Français |
|---|---|
| A/B split · weight | Répartition A/B · pondération |
| automation (one, or the section) | automatisation ("Nouvelle automatisation") |
| Branch · Yes · No | Embranchement · Oui · Non |
| Delay (step) · Wait 10 minutes | Délai · Attendre 10 minutes |
| double opt-in · double opt-in confirmed only | double opt-in · double opt-in confirmé uniquement |
| edge (a connection on the canvas) | liaison |
| exit · goal · goal rate | sortie · objectif · taux d’atteinte de l’objectif |
| fail over · failover | basculer · basculement |
| Filter (step) | Filtre |
| flow · in the flow · leave the flow | scénario · dans le scénario · quitter le scénario ("scénario automatisé" where the kind needs saying; "flux" is only a stream: "flux Automatisations") |
| List action | Action sur la liste |
| method (HTTP) | méthode |
| override (a route) | remplacement; "route override" on a step is "route spécifique" |
| re-entry | réentrée |
| retire · retiring (a version) | retirer · retrait en cours |
| send window | plage d’envoi |
| signing secret · write-only | secret de signature · en écriture seule |
| stats · Live stats | statistiques · Statistiques en direct |
| step · trigger · action · condition | étape · déclencheur · action · condition |
| web push | notification push web |
| path (of a branch or split) · connection | branche · liaison |
| starters: Welcome series · Re-engagement · Onboarding · Abandoned cart | Série de bienvenue · Réengagement · Prise en main · Panier abandonné |
| Send Readiness content checks | contrôles de contenu avant envoi |
| webhook delivery log · failed calls | journal des livraisons du webhook · appels en échec |
| route label: SES, failover Postmark | SES, secours Postmark |
| Removed provider | Fournisseur supprimé |
| webinar | webinaire |
| version summary: created from a blank canvas · created from the X template · copied from X · first published version | créée à partir d’un canevas vierge · créée à partir du modèle X · copiée depuis X · première version publiée |

## Approvals

| English | Français |
|---|---|
| after exclusions | après exclusions |
| approval inbox | boîte des approbations |
| approval request · approver · author | demande d’approbation · approbateur · auteur |
| Approve · Reject · Request changes | Approuver · Refuser · Demander des modifications |
| Bypassed · emergency bypass | Contournée · contournement d’urgence |
| comment · Discussion · evidence | commentaire · Discussion · preuve |
| Contacts in flight | Contacts en cours de scénario |
| Invalidated · Approval invalidated | Invalidée · Approbation invalidée |
| median wait | attente médiane (a median, not an average: never "moyenne") |
| Pending · Rejected | En attente · Refusée |
| policy · rule · Rules matched | politique · règle · Règles correspondantes |
| Recheck | Revérifier |
| snapshot | instantané |
| after approval (a change) | après approbation ("le nombre de destinataires a changé de +14 % après approbation"; said once) |
| rules: audience ≥ 10,000 · warming-up provider · new provider · author lacks send · list requires approval · client sign-off | audience ≥ 10 000 · fournisseur en chauffe · nouveau fournisseur · auteur sans droit d’envoi · liste soumise à approbation · validation client |
| automation publish · post publish · page publish · template publish (rules) | publication d’automatisation · publication d’article · publication de page · publication de modèle |
| a send held · suppressed | envoi retenu · exclu (on the suppression list; never "supprimé", which reads as deleted) |
| online · active 2 h ago (an approver) | en ligne · actif il y a 2 h |
| Best time within 24 h of | Meilleure heure dans les 24 h suivant |
| Only people can… (not agents or keys) | Seules des personnes peuvent… |

## Content languages

| English | Français |
|---|---|
| language version · Source | version linguistique · Source |
| Default language · Fallback language | Langue par défaut · Langue de repli |
| Up to date · Outdated · Draft · Missing · Held | À jour · Obsolète · Brouillon · Manquante · Retenue (agreeing with "version") |
| Mark as translated · Mark as up to date | Marquer comme traduite · Marquer comme à jour |
| readers (of a language) | lecteurs ("Lecteurs en {language}") |
| page copy · built-in labels | textes de la page · libellés intégrés |
| Results per language · click map | Résultats par langue · carte des clics |
| no language · other languages (results) | sans langue · autres langues |
| compare view's blocks: Header · Footer · Raw HTML · Social links · Two cards · 3 columns · Hero image · Button · Intro · The email | En-tête · Pied de page · HTML brut · Liens sociaux · Deux cartes · 3 colonnes · Image hero · Bouton · Introduction · L’e-mail entier |
| Headline (a block) · no alt text | Grand titre ("Titre" is a heading) · sans texte alternatif |
| version labels: copied from · layout of · applied from · block copied from · changed by an agent · edited in the flow editor · translated from | copiée depuis · mise en page d’après · appliquée depuis · bloc copié depuis · modifiée par un agent · modifiée dans l’éditeur de scénario · traduite de (with a language: "traduite de la version en {language}") |
| a language's changes after approval: added · marked translated · back to draft · sent again · edited | ajoutée · marquée comme traduite · repassée en brouillon · renvoyée · modifiée |
| copy (a form's or page's own text) · Copy in | textes · Textes en ("Textes en {language}") |
| Live · Incomplete · Not set up (copy) | En ligne · Incomplet · Non configuré |
| string (text to write) | chaîne |
| visitors | visiteurs ("Visiteurs en {language}") |

A content language's name comes from the browser's or CLDR's names in the
app language ("Espagnol" on its own, "espagnol" in a sentence), and the
app's own picker names each language in itself. Write it after "en", never
after an article (the Voice section).

## Conditional content

| English | Français |
|---|---|
| combination (of blocks) · version | combinaison · version |
| conditional block | bloc conditionnel |
| Gmail clips · clipping size | Gmail tronque · taille de troncature |
| includes · excludes | inclut · exclut |
| is · isn't · contains · is set · is empty | est · n’est pas · contient · est renseigné · est vide (as automations' conditions) |
| is at least · is at most · is between … and … | est au moins · est au plus · est entre … et … |
| is greater than · is less than · is before · is after | est supérieur à · est inférieur à · est avant · est après |
| is in · is not in | fait partie de · ne fait pas partie de |
| Match ALL / ANY of these | Remplir TOUTES / AU MOINS UNE de ces conditions (the uppercase words stay uppercase) |
| operator | opérateur |
| Preview as · Show why | Prévisualiser en tant que · Voir pourquoi |
| Visibility (who sees a block) | Visibilité |

## Accessibility

| English | Français |
|---|---|
| Accessibility | Accessibilité |
| issue · serious · moderate | problème · grave · modéré |
| alt text · decorative image | texte alternatif · image décorative |
| contrast · Passes AA · Fails AA | contraste · Conforme AA · Non conforme AA |
| Heading 1 · Paragraph · Text role | Titre 1 · Paragraphe · Rôle du texte |
| layout table | tableau de mise en page |
| screen reader | lecteur d’écran |
| Warn · Block (the policy) | Avertir · Bloquer |
| skipped heading level · card titles | niveau de titre sauté · titres de cartes |

"Heading" is "titre", as in word processors ("Titre 1"); an email's
subject is always "objet", never "titre". Content languages in a sentence
follow "en", in lower case: "en allemand", "en espagnol". A link "that
says nothing" is "non explicite", the word of the French accessibility
standard (RGAA).

## Earn

Payments, paid subscriptions, offers, gifts, sponsors and revenue. A
"paid member" is an **abonné payant**, never a "membre", which is a team
member. A reader's Stripe subscription is an **abonnement**: it is
"résilié" (cancelled), while a contact who stops the emails "se désabonne".
Earn's plans and tiers are **formules**; a sending provider's plan is a
"forfait".

| English | Français |
|---|---|
| access (paid) · With access | accès · Avec accès |
| access rank · Rank 2 | niveau d’accès · Niveau 2 |
| Access rules | Règles d’accès |
| Active · Trialing · Past due · Cancels soon · Canceled · Ended (paid status) | Actif · En essai · Impayé · Résiliation prévue · Résilié · Terminé (agreeing with "abonnement") |
| archive · Archived (a tier) | archiver · Archivée |
| attribution · attribution window · credited to (an email) | attribution · fenêtre d’attribution · attribué à |
| benefit (of a tier) | avantage |
| billing email · billing period · Manage billing | e-mail de facturation · période de facturation · Gérer la facturation |
| book · booking · Booked (a sponsor in an issue) | réserver · réservation · Réservé |
| buyer · recipient (of a gift) | acheteur · destinataire |
| cancel page · gift page · upgrade page · welcome page | page de résiliation · page cadeau · page de passage au payant · page de bienvenue |
| charge (a card) | débiter (la carte); "nothing is charged" is "aucun montant n’est débité" |
| checkout · test checkout | paiement · paiement de test ("Stripe Checkout" is a name) |
| churn · churn rate | attrition · taux d’attrition |
| click token | jeton de clic |
| Complimentary · complimentary access | Offert · accès offert |
| conversion · conversion API | conversion · API de conversion |
| coupon · promotion code · code | coupon · code promo · code |
| creative (a sponsor's) | création |
| customer portal | portail client |
| deliver · delivery date (a gift) | remettre · date de remise |
| discount · 20% off | réduction · −20 % |
| dispute (a card payment's) | litige |
| Earn's tabs: Overview · Plans & offers · Paid members · Gifts · Sponsors · Products · Settings | Vue d’ensemble · Formules et offres · Abonnés payants · Cadeaux · Sponsors · Produits · Paramètres |
| expansion · contraction · reactivated (MRR movement) | expansion · contraction · réactivé |
| Free (the card) · free subscriber | Gratuit · abonné gratuit |
| free-to-paid | passage du gratuit au payant |
| gift · gift option | cadeau · option cadeau |
| headline (an upgrade page's, a creative's) | titre; a subject is "objet" |
| highlight (a tier) · Most popular | mettre en avant · Le plus populaire |
| hold · on hold · release (an issue) | option · sous option · lever l’option (as in ad sales: "poser une option") |
| in person (a gift handed over) | en main propre |
| issue (of a newsletter: #143) | numéro (n° 143); an accessibility issue is "problème" |
| legacy price | ancien tarif |
| lifetime value | valeur vie client |
| live · test (a key, a mode, payments) | production · test ("mode production", "clé de production"); "no real payments" is "aucun paiement réel" |
| live (a page, an offer or a template version that's running) | en ligne |
| map (a Stripe price to a tier) | associer |
| member email · membership | e-mail de l’abonné · abonnement |
| merchant of record | marchand officiel |
| monthly · yearly · Monthly · Yearly (the choice) · /month | mensuel · annuel · Mensuel · Annuel · /mois |
| MRR · ARR · MRR movement | MRR · ARR · variation du MRR |
| Needs you (Earn's overview) | À traiter |
| normal price (without an offer) | prix normal |
| offer · signup · retention · referral reward | offre · offre d’inscription · offre de fidélisation · récompense de parrainage |
| Overdue · Unpaid · Paid (a sponsor's payment) | En retard · Impayé · Payé |
| paid member | abonné payant |
| paid status · paid-only | statut payant · réservé aux abonnés payants |
| Paid subscriptions (what Earn sells) | Abonnements payants |
| paid-through date | payé jusqu’au |
| payment provider | prestataire de paiement |
| payment terms (a sponsor's) | conditions de paiement |
| payouts | virements |
| plan · Plans & offers | formule · Formules et offres |
| Public · Hidden (a tier's visibility) | Publique · Masquée |
| receipt · invoice | reçu · facture |
| record (a sponsor's payment, by hand) | enregistrer |
| redeem · redeem link (a gift) | activer · lien d’activation |
| redeem · redemption (an offer code) | utiliser · utilisation |
| refund | rembourser · remboursement |
| renew · renewal | renouveler · renouvellement |
| restricted key · secret key | clé restreinte · clé secrète |
| results so far · final (a sponsor report's) | résultats provisoires · définitifs |
| revenue · income | revenus · recettes |
| roll (a Stripe key) | renouveler |
| seller ("you're the seller") | vendeur |
| signing secret (a webhook's) · endpoint | secret de signature (as Automations has it) · endpoint |
| Since · Member since (a paid member's) | Depuis · Abonné depuis |
| slot · placement | emplacement |
| sponsor · sponsorship · Sponsored (the block's label) | sponsor · sponsoring · Sponsorisé ("parrainage" is a referral) |
| sponsor report | rapport sponsor |
| subscription (a reader's, billed by Stripe) | abonnement |
| Terms of sale · Support address | Conditions de vente · Adresse d’assistance |
| tier · paid tier | formule · formule payante |
| timeline (a paid member's) | chronologie |
| trial · free trial | essai · essai gratuit |
| upgrade · Upgrade block | passage au payant · bloc Passage au payant (a button: "Passer au payant") |
| webhook silent · gap check | webhook silencieux · contrôle des événements manquants |
| write-only · sealed (a key, a secret) | en écriture seule · scellé |

Some English messages choose a word for English grammar only: an article
(`{article, select, an {an} other {a}} {tier}`) or a possessive
(`{ends, select, s {{sponsor}’} other {{sponsor}’s}}`). French keeps the
argument and writes the whole phrase in its `other` branch, brought in by
a noun so nothing has to elide: `{article, select, other {la formule
{tier}}}`, `{ends, select, other {la création du sponsor {sponsor}}}`.
Where the English counts with `#`, keep `{count, plural, one {# …} other
{# …}}` rather than a bare `{count}`, so the number is grouped by the
region format (48 213).

## Website and pages

The publication website, its posts, the page builder and the public pages'
own words. Words a reader sees follow the built-in labels of the public
pages (`internal/pagelabels`): "newsletter", "numéro", "Archives",
"Préférences e-mail".

| English | Français | Note |
|---|---|---|
| About (page, template) | À propos | |
| Access (Website › Access) | Accès | |
| archive · Archive (the page, the link) | archives · Archives | always plural in French; "page des archives" when the page itself is meant |
| band (subscribe band) | bandeau | a full-width strip of the home page |
| block · layers | bloc · calques | the page builder's |
| built-in page | page intégrée | like "libellés intégrés" |
| byline · By {name} | signature · Par {name} | |
| canonical URL | URL canonique | as Google's tools say it |
| card (paywall card, a post's card) | encart | "carte" is a layout card ("Deux cartes") |
| certificate | certificat | |
| countdown | compte à rebours | |
| cover (image) | image de couverture | |
| excerpt | extrait | |
| FAQ (block) | FAQ | |
| featured · Feature on home | à la une · Mettre à la une | |
| feed · RSS feed | flux · flux RSS | |
| free preview · Free preview ends here (the post's block) | aperçu gratuit · Fin de l’aperçu gratuit | |
| Hero (block) | Hero | the block's name, as in the composer |
| home page | page d’accueil | |
| issue (of a newsletter) | numéro | "48 numéros", "Lire le dernier numéro" |
| lead magnet | cadeau d’inscription | a free file for confirmed subscribers |
| live (the website) · live (a page, a form, a version) | en ligne · en ligne | as Earn and the signup forms have it |
| member · free member · paid member (of the site) · membership | abonné · abonné gratuit · abonné payant · abonnement | a reader who signs in, as Earn has it; "membre" is a team member |
| meter | quota d’articles gratuits | "Quota d’articles" as a heading |
| navigation · header · footer | navigation · en-tête · pied de page | |
| page · Pages | page · Pages | |
| page builder | éditeur de pages | |
| Pay what you want | Prix libre | |
| paywall | paywall | masculine: "le paywall" |
| post · web-only post | article · article web uniquement | |
| publish · unpublish · unschedule | publier · dépublier · déprogrammer | Unpublished: Dépublié |
| redirect | redirection | |
| referrer (where views come from) | site référent | a person who refers is a "parrain" |
| registration (event) | inscription | "inscription à l’événement" where a newsletter signup could be meant |
| sales page | page de vente | |
| signups · upgrades · views · visitors | inscriptions · passages au payant · vues · visiteurs | |
| site · website | site · site web | |
| sitemap | plan du site | |
| slug | slug | the page's or tag's part of the address |
| social image | image de partage | the image a shared link shows |
| store product (block) | produit de la boutique | |
| tab (a panel's: the Web tab) | onglet | |
| tag | tag | |
| tagline | slogan | |
| testimonial | témoignage | |
| theme | thème | |
| tier · paid tier | formule · formule payante | as Earn has it; a tier's name, such as "Insider", stays as it is |
| title (a post's, a page's, the site's) · SEO title | titre · titre SEO | "objet" is an email's subject |
| visibility · Public · Members · Paid | visibilité · Public · Abonnés · Payant | |
| visitor | visiteur | someone who isn't signed in |
| webinar | webinaire | |

## Team

| English | Français |
|---|---|
| Team | Équipe |
| Invite member | Inviter un membre |
| Members | Membres |
| Everyone with access to Acme Newsletter | Toutes les personnes ayant accès à Acme Newsletter |
| Search name or email | Rechercher un nom ou un e-mail |
| Any role · Any status · Any 2FA | Tous les rôles · Tous les états · 2FA : tous |
| Role · scope | Rôle · portée |
| Sign-in | Connexion |
| Last active | Dernière activité |
| Manage · Reactivate | Gérer · Réactiver |
| Password · passkey | Mot de passe · clé d’accès |
| just now · yesterday | à l’instant · hier |
| 2 selected · Bulk: | 2 sélectionnés · Action groupée : |
| Change role · Require 2FA now · Deactivate | Changer le rôle · Exiger la 2FA maintenant · Désactiver |
| Required · off, 3 days of grace left | Obligatoire · désactivée, encore 3 jours de délai de grâce |
| deactivated Sep 30, 2026 | désactivé le 30 sept. 2026 |
| Pending invitations · 1 | Invitations en attente · 1 |
| Links last 7 days by default | Les liens sont valables 7 jours par défaut |
| Invitee · Sent · Expires | Personne invitée · Envoyée · Expiration |
| invited by Linh Nguyen | invitation envoyée par Linh Nguyen |
| Beta Program list only | liste Beta Program uniquement |
| in 5 days | dans 5 jours |
| Resend · Copy link · Revoke | Renvoyer · Copier le lien · Révoquer |

## Language and region

| English | Français |
|---|---|
| Language & region | Langue et région |
| App language | Langue de l’application |
| Instance default (English) | Langue de l’instance (English) |
| Region format | Format régional |
| Follows the app language | Suit la langue de l’application |
| Time · 24-hour · 12-hour | Heure · 24 h · 12 h |
| Week starts on · Monday · Sunday | Premier jour de la semaine · Lundi · Dimanche (capital only because each stands alone as a choice; "lundi" in a sentence) |
| From the region format | Selon le format régional |
| Preview | Aperçu |
| Help translate ReputeCast | Aider à traduire ReputeCast |
| Save · Undo | Enregistrer · Annuler |
| Profile | Profil |
| timezone | fuseau horaire |
| quiet hours | heures de silence |
| Settings › General | Paramètres › Général |
| Invitation language | Langue des invitations |
| Default language · Default region format | Langue par défaut · Format régional par défaut |

Languages are always named in their own language, in every interface
language: English, Tiếng Việt, Deutsch (Deutschland), Français (France).
"ReputeCast standard" is a name and stays in English.

## Account and security

| English | Français |
|---|---|
| Full name | Nom complet |
| Upload photo · Remove | Importer une photo · Supprimer |
| Change email · verification link | Modifier l’adresse e-mail · lien de vérification |
| Current password · New password | Mot de passe actuel · Nouveau mot de passe |
| security key · platform passkey | clé de sécurité · clé d’accès de l’appareil |
| Rename · Keep it | Renommer · Conserver |
| Time-based codes (TOTP) | Codes temporaires (TOTP) |
| Set up · Replace app | Configurer · Changer d’application |
| Regenerate codes | Régénérer les codes |
| 7 of 10 remain | 7 codes restants sur 10 |
| Connected sign-ins | Connexions associées |
| Link · Unlink · Managed | Associer · Dissocier · Géré |
| session · Active sessions · This session | session · Sessions actives · Cette session |
| Sign out all other sessions | Déconnecter toutes les autres sessions |
| Device · Browser · IP · location (approx.) | Appareil · Navigateur · IP · localisation (approx.) |
| My workspaces · Leave workspace | Mes espaces de travail · Quitter l’espace de travail |
| In-app · All events | Dans l’application · Tous les événements |
| MCP agent | agent MCP |

## Instance admin, keys, webhooks and the audit log

| English | Français | Note |
|---|---|---|
| actor (audit) · Any actor type | acteur · Tous les types d’acteurs | who did it: a member, a key, an agent or the system |
| agent capability: Allow · Deny · Approve · Pause only (Settings › MCP) | Autoriser · Refuser · Approuver · Pause uniquement | the matrix's cells; « Approuver » as the caption quotes it |
| All retained (date range) | Toute la période conservée | |
| app registration (Entra) · app integration (Okta) | inscription d’application · intégration d’application | "the Entra app registration": "l’inscription d’application Entra", as Entra's French console says it |
| batch (of streamed entries) | lot | as in "endpoint par lots" |
| break-glass Owner | Propriétaire avec compte d’urgence | never "Propriétaire d’urgence" |
| break-glass password · break-glass passkey | mot de passe d’urgence · clé d’accès d’urgence | what a break-glass Owner keeps while SSO is enforced |
| claim · Client ID · client secret · Scopes (OIDC) | claim · ID client · secret client · Scopes | as identity providers' French consoles name them; "claim" and "scope" stay, the OIDC words |
| danger zone | Zone de danger | |
| delivery (webhook) · Delivery log | livraison · Journal des livraisons | |
| Development build | Version de développement | |
| digest (email) · Hourly digest · Daily digest | e-mail de synthèse · Synthèse horaire · Synthèse quotidienne | notification preferences; a recurring digest of posts is a "récapitulatif" |
| discovery · issuer · Issuer URL (OIDC) | discovery · émetteur · URL de l’émetteur (issuer) | |
| Effective permissions | Autorisations effectives | a member's page; the column is "Effectives" |
| endpoint · payload | endpoint · payload | not translated |
| enforce (SSO, two-factor) · enforcement | imposer · obligation | "Enforce SSO for acme.io": "Imposer le SSO pour acme.io" |
| entry (audit) · Entry details | entrée · Détails de l’entrée | |
| Failing · Paused (an endpoint) | En échec · En pause | "Healthy" is "Opérationnel" |
| Force sign-out | Forcer la déconnexion | |
| Generic OIDC (a preset) | OIDC générique | the IdP's short name in sentences is "IdP" |
| Given (approvals a member gave) | Traitées | approvals and rejections alike |
| group mapping · group object ID | correspondance des groupes · ID d’objet du groupe | |
| hourly cleanup | nettoyage horaire | |
| identity provider | fournisseur d’identité | "IdP" stays |
| immutable (audit entries) | non modifiables | |
| instance-level (entries) · Instance level | au niveau de l’instance · Niveau instance | |
| JIT account creation | création de compte à la volée (JIT) | |
| just-in-time account · JIT default role | compte JIT · rôle JIT par défaut | |
| Locked (sign-in) · Unlock sign-in | Bloqué · Débloquer la connexion | a lockout is "blocage de la connexion" |
| Make instance admin · Remove instance admin | Nommer admin de l’instance · Retirer le rôle d’admin de l’instance | |
| overage link (Entra's groups) | lien de dépassement (overage) | Entra's own word, as its French docs give it |
| preset (an SSO provider's, a key's) | préréglage | |
| proposal (an agent asked a person) · proposal recorded | proposition · proposition enregistrée | "Ops asked to pause …": "Ops propose de mettre en pause…" |
| provider status: degraded · down | dégradé · en panne | as Channels and SMS' Dégradé · En panne |
| re-authentication | réauthentification | |
| Redirect URI | URI de redirection | |
| reference ID · request ID | ID de référence · ID de requête | |
| result: Allowed · Denied · Error | Autorisé · Refusé · Erreur | |
| retention | durée de conservation | |
| revoke · rotate · replace (a key or secret) | révoquer · renouveler · remplacer | |
| secret · signing secret | secret · secret de signature | "secrets" in general: "secrets" |
| Send approvals (Settings › General) | Approbation des envois | |
| Sending (notification category) | Envois | so it isn't read as the status "Envoi en cours" |
| SIEM or warehouse | SIEM ou entrepôt de données | |
| single sign-on | authentification unique | "SSO" stays |
| stream (audit entries, noun) · stream (verb) · streaming endpoint | flux · diffuser en continu · endpoint de streaming | the System stream is "le flux Système" |
| test sign-in | connexion de test | |
| Then allowed | Puis autorisée | a sign-in that was rejected, then allowed |
| token (an agent's) | jeton | "agent token": "jeton de l’agent" |
| tool (MCP) | outil | tool names (get_campaign_report) stay as they are |
| Trusted from Okta (two-factor) | Assurée par Okta | the IdP enforces the second factor |
| typed confirmation | confirmation par saisie | |
| Up to date · Update available (a release) | À jour · Mise à jour disponible | language versions use "À jour" too |
| user (an account on the instance) | utilisateur | a workspace's people are "membres"; "compte" where it reads as well |
| Workspace ID | ID de l’espace de travail | |
| workspace switcher | sélecteur d’espace de travail | |

## Channels and SMS

| English | Français |
|---|---|
| 10DLC · local · mobile · toll-free (number kinds) | 10DLC · local · mobile · gratuit ("numéro mobile", "numéro gratuit") |
| adapter · manifest · contract test suite | adaptateur · manifest · suite de tests de contrat |
| auto-pause · Auto-paused | pause automatique · En pause auto |
| auto-pause reasons: Too many texts failed · Carriers filtered too many texts · Too many people opted out | Trop de SMS en échec · Trop de SMS filtrés par les opérateurs · Trop de désinscriptions |
| brand (in a 10DLC registration) · brand prefix | marque · préfixe de marque |
| budget · spend · price per segment | budget · dépenses · prix par segment SMS |
| budget used up · budget at 80% | budget épuisé · budget utilisé à 80 % ("Budget SMS épuisé pour octobre") |
| carrier | opérateur |
| channel · Email · SMS · Push · WhatsApp | canal · E-mail · SMS · Push · WhatsApp |
| consent · SMS consent · consent text · evidence | consentement · consentement SMS · texte de consentement · preuve |
| credentials · Live · Test credentials | identifiants · Production · Identifiants de test |
| credentials rejected · isn't answering · is back (a provider) | identifiants refusés · ne répond pas · de nouveau disponible |
| delivery receipt | accusé de réception (what a carrier reports back; "Last receipt" is "Dernier accusé de réception") |
| fallback (of a route) | secours |
| filtered by carriers | filtré par les opérateurs |
| held · skipped | retenu · ignoré |
| inbound (text) · reply | SMS entrant · réponse |
| keyword | mot-clé (STOP, START, HELP and YES stay in English: people text them) |
| landline · not in service · invalid number · unreachable | fixe · hors service · numéro non valide · injoignable |
| link preview | aperçu du lien (a messaging app's machine click, never a person) |
| local time · "Berlin local" (a reply's time) | heure locale · « heure de Berlin » |
| no answer · invalid credentials (what a provider answered a call) | pas de réponse · identifiants non valides (the HTTP status before them stays: "401 Unauthorized · identifiants non valides") |
| opt in · opt-in confirmation | s’inscrire · SMS de confirmation d’inscription |
| opt out · opt-outs · opt-out text · opt-out reply | se désinscrire · désinscriptions · mention de désinscription · réponse de désinscription |
| opt-out sync · Two-way · Read only | synchronisation des désinscriptions · Bidirectionnelle · Lecture seule |
| opt-out list (a provider's own) | liste de désinscription ("Liste de désinscription Twilio") |
| outside quiet hours (a text waits or is skipped) | hors de la plage d’envoi (the window texts may go out in, not "hors des heures de silence") |
| provider status: Healthy · Degraded · Down | Opérationnel · Dégradé · En panne |
| quiet hours · sending window | heures de silence · plage d’envoi |
| raw event · normalized | événement brut · normalisé |
| receipts silent (none came back while texts went out) | aucun accusé de réception |
| registration · registered | enregistrement · enregistré |
| route · Routing & rules · route tester | route · Routage et règles · testeur de routes |
| segment (of a text) | segment SMS (the billed part of 160 GSM-7 or 70 UCS-2 characters; never a bare "segment", which is an audience segment) |
| sender ID · alphanumeric sender ID | ID d’expéditeur · ID d’expéditeur alphanumérique |
| short-link domain · public shortener | domaine de liens courts · raccourcisseur public |
| signature · signing secret | signature · secret de signature |
| status callback (Twilio) | callback de statut |
| suppression list (SMS) | liste de suppression SMS |
| test text · test number | SMS de test · numéro de test |
| text (a message) · to text | SMS · envoyer un SMS |
| Text message terms | Conditions des SMS |
| throughput · 14 texts a second | débit · 14 SMS par seconde |
| unschedule · a send (of a campaign) | déprogrammer · envoi ("Cet envoi est terminé") |

"GSM-7", "UCS-2", "Unicode", "VoIP" and "emoji" stay as they are. A
phone number, a sender ID and a country code are data: never translated.
An email unsubscribe is "désabonnement"; leaving SMS (opt out) is
"désinscription".

## Web push and WhatsApp

| English | Français |
|---|---|
| access token · permanent access token · system user | jeton d’accès · jeton d’accès permanent · utilisateur système |
| Allow · Not now · Granted (a prompt's numbers) | Autoriser · Plus tard · Accordées |
| anonymous · linked (a subscription) | anonyme · associé (à un contact) |
| app secret · verify token · callback URL · phone number ID | clé secrète de l’app · jeton de vérification · URL de rappel · ID du numéro de téléphone (as Meta's French screens say them) |
| banner (a notification's style on a desktop) | bannière |
| Bell button · Dialog · Slide-in bar (prompt styles) | Bouton cloche · Boîte de dialogue · Barre coulissante |
| business account · display name | compte professionnel · nom d’affichage ("WhatsApp Business Account" is a name) |
| category (of a template): Marketing · Utility · Authentication | catégorie : Marketing · Utilitaire · Authentification |
| conversation · customer service window | conversation · fenêtre de service client (short: "fenêtre de 24 h", never the quiet hours' "plage d’envoi") |
| expiry · Expires after | expiration · Expire après |
| free-form reply · quick reply | réponse libre · réponse rapide |
| frequency cap | plafond de fréquence |
| header · body · footer · buttons (of a template) | en-tête · corps · pied de page · boutons |
| Home Screen (iPhone, iPad) | écran d’accueil |
| icon · badge · image (of a notification) | icône · badge · image |
| In review · In appeal · submit (a template) | En cours d’examen · En appel · soumettre |
| messaging limit · rolling 24 hours | limite de messagerie · 24 heures glissantes |
| notification (one push message) | notification |
| permission prompt (the browser's own) | demande d’autorisation (du navigateur) |
| prompt · prompt designer | invite · éditeur d’invites (an AI prompt is a "prompt") |
| publication site | site de la publication |
| push (in a sentence) · Push (the channel) | notification push · Push |
| push service (the browser maker's) | service push |
| quality rating · High · Medium · Low | note de qualité · Élevée · Moyenne · Faible |
| rate card · Edit rates | grille tarifaire · Modifier les tarifs |
| read receipts | confirmations de lecture (WhatsApp's own French) |
| Rotate keys · public key · private key | Renouveler les clés · clé publique · clé privée |
| Sealed | Scellé |
| Shown (a count) · Shown (an event) · Gone · removed | Affichages · Affichée · Expirée · supprimée |
| site (where readers subscribe) · scope (of a service worker) | site · portée |
| snippet | extrait de code |
| subscriber · subscription | abonné · abonnement |
| Sync now | Synchroniser maintenant |
| Tapped (a WhatsApp button) | Touché |
| template (WhatsApp) | modèle ("modèle de message" where it must be explicit) |
| token (that a link carries to know the reader) | jeton d’identification |

"service worker", "VAPID", "Meta", "Graph API" and Meta's own screens and
paths (WhatsApp Manager, Meta for Developers › … › Configuration) stay as
they are.

## Surveys and polls

| English | Français |
|---|---|
| 0–10 scale · Emoji scale · faces | échelle de 0 à 10 · échelle d’emojis · visages |
| answer · answer page | réponse · page de réponse |
| Breakdown · Break down by | Ventilation · Ventiler par |
| completion rate · drop-off · started · completed | taux de complétion · abandon · commencé · terminé |
| Embedded · Popup · Hosted page (form kinds) | Intégré · Pop-up · Page hébergée |
| link scanner · security gateway · prefetching proxy | scanner de liens · passerelle de sécurité · proxy de préchargement (machine visits, never a person) |
| Not likely · Very likely (an NPS scale's ends) | Peu probable · Très probable |
| Open · Closed (a survey or a poll) | Ouvert · Fermé |
| Owner (the forms list's column) | Responsable (not the role "Propriétaire") |
| poll (the block) · a poll | Sondage · un sondage |
| promoters · passives · detractors | promoteurs · passifs · détracteurs ("NPS" and "Net Promoter Score" stay) |
| question · respondent · response | question · répondant · réponse |
| submission | soumission |
| survey | questionnaire |
| Top answer | Réponse la plus choisie |
| vote · voter | vote · votant |

## Notifications

| English | Français |
|---|---|
| digest (recurring) · issue (of a digest) | récapitulatif récurrent · numéro |
| sending summary (by email, M57) · weekly · monthly | bilan des envois (par e-mail) · bilan hebdomadaire · mensuel des envois (a member's own email of the workspace's numbers; never "synthèse", the notification digest, or "récapitulatif") |
| event type (a row of Notification preferences) | type d’événement |
| Everyone with campaigns:pause | Toutes les personnes disposant de l’autorisation campaigns:pause |
| paid tier (Earn) | formule payante |
| placement test · seed mailbox | test de placement · boîte témoin |
| Sending (the notification center's tab) | Envois |
| verification credits | crédits de vérification |

## Agency accounts

| English | Français |
|---|---|
| Account manager · Agency manager (staff levels) | Responsable de compte · Responsable d’agence |
| agency · Agency (the area) · agency mode | agence · Agence · mode agence |
| assign · assignment | affecter · affectation |
| client · client workspace | client · espace de travail client |
| client people (a client's own people) | personnes du client (never agency staff) |
| Client approver · Client viewer (roles) | Approbateur client · Observateur client |
| client sign-off | validation client (distinct from an "approbation") |
| copy (of a library item) · behind | copie · pas à jour (the library has a newer version) |
| flat (rate) · monthly fee | forfaitaire · frais mensuels |
| incident · Needs attention | incident · Attention requise |
| level (of staff) | niveau |
| library · shared library · library item | bibliothèque · bibliothèque partagée · élément de bibliothèque |
| provider cost · plan | coût du fournisseur · forfait |
| rebill · rebill rate · to rebill | refacturer · taux de refacturation · à refacturer |
| saved section | section enregistrée |
| sender health · good · fair · poor | santé de l’expéditeur · bonne · moyenne · faible |
| setup steps · Setting up | étapes de configuration · Configuration en cours |
| staff | personnel de l’agence |
| usage | consommation |
| workspace template · apply (a template) | modèle d’espace de travail · appliquer |

## Grow, Growth and scoring

| English | Français | Note |
|---|---|---|
| awaiting confirmation | en attente de confirmation | a referral whose subscriber hasn't confirmed |
| band (score) · Cold · Warm · Hot | niveau · Froid · Tiède · Chaud | the names are fixed, like the English |
| burst · shared IP · numbered addresses | pic · IP partagée · adresses numérotées | the signals that hold a referral |
| cap (a rule's) | plafond | "6 a week": "6 par semaine" |
| claim · claim link | récupérer · lien de récupération | a coupon reward |
| cohort retention | rétention par cohorte | Audience › Growth |
| counted · not counted | compté · non compté | "Count" and "Don't count" in Review: "Compter", "Ne pas compter" |
| coupon (Stripe) | coupon | |
| decay | décroissance | |
| double opt-in | double opt-in | two-factor is "authentification à deux facteurs" |
| engagement tier · Active · Cooling · At risk · Sunset | niveau d’engagement · Actif · En baisse · À risque · Arrêt progressif | List health's tiers, as Scoring names them; the sunset policy is "politique d’arrêt progressif" |
| field condition | condition de champ | |
| fulfill · fulfilled · To fulfill | remettre · remis · À remettre | a manual reward; "sent" stays "envoyé" |
| Growth (Audience › Growth) | Évolution | so it doesn't read as Grow ("Croissance") |
| held for review | retenu pour examen | short: "retenu" |
| hold period · in hold | délai de validation · en validation | "the 7-day hold": "le délai de validation de 7 jours" |
| hosted (form, page) | hébergé (sur ReputeCast) | |
| invite page | page d’invitation | |
| join · departure | arrivée · départ | Growth's joined and left: "Arrivées", "Départs" |
| leaderboard | classement | |
| List health | Santé de la liste | as Growth and Scoring refer to it |
| milestone · milestone email | palier · e-mail de palier | "Palier 3" |
| net (growth) | croissance nette | |
| newsletter | newsletter | feminine: "la newsletter" |
| partner · partner code · partner link · partner profile · partnership | partenaire · code partenaire · lien partenaire · profil partenaire · partenariat | |
| recommendation · recommend · Recommending you | recommandation · recommander · Vous recommandent | |
| referral · referrer · referral link · referral program | parrainage · parrain · lien de parrainage · programme de parrainage | "12 parrainages"; the person referred is a "filleul" |
| referral hub | espace parrainage | the referrer's own page |
| Referral · Recommendations · Reward block | bloc Parrainage · bloc Recommandations · bloc Récompense | the composer's Grow blocks |
| referring page · referring domain | page référente · domaine référent | never "parrain", which is a person who refers |
| review queue · Review (tab) | file d’examen · Examen | |
| reward | récompense | |
| rule · points | règle · points | |
| score · scoring | score · scoring | |
| self-referral | auto-parrainage | |
| soft bounce · hard bounce | rebond temporaire · rebond définitif | |
| source (growth) | source | |
| spread (of scores) | répartition | |
| subscribe request | demande d’abonnement | what one instance sends a partner |
| threshold | seuil | |
| token (Liquid) | variable | as transactional templates' variables |
| trigger (automation) | déclencheur | |

## Audience, imports, forms and reports

| English | Français | Note |
|---|---|---|
| accent color · page background | couleur d’accent · arrière-plan de la page | public page branding |
| Added by the team (a contact's source) | Ajouté par l’équipe | a suppression added by hand is "Manuelle" |
| allowed domains | domaines autorisés | the sites a form may be embedded on |
| archive (public page) · issue | archives · numéro | as subscribers read them: "Archives de la newsletter", "Numéro suivant" |
| audience · merge field · cleaned (Mailchimp) | audience · champ de fusion · nettoyé | Mailchimp's own words, as its French screens show them |
| batch (an import's) | lot | the live log: "Lot 3 · 500 lignes" |
| bounce types (a report's) | Définitif · Temporaire · Bloqué | "Définitif · destinataire inexistant", "Temporaire · boîte pleine" |
| branding | image de marque | Forms & pages › "Image de marque et domaine" |
| catch-all domain · known complainers | domaine catch-all · auteurs de plaintes connus | a verification service's risky groups |
| column · map fields | colonne · associer les champs | an import's step: "Associer les champs" |
| confirmation link | lien de confirmation | |
| consent · consent text · consent evidence | consentement · texte de consentement · preuve de consentement | |
| custom domain | domaine personnalisé | Branding & domain |
| data center (Mailchimp) | centre de données | the API key's suffix, like us21 |
| deferral (provider) | report | "deferrals expired": "reports expirés" |
| disposable domain · role address · likely typo | domaine jetable · adresse générique · faute de frappe probable | an import's risky groups |
| embed code · script embed · embedded (form) | code d’intégration · intégration par script · intégré | |
| erase (a contact) | effacer définitivement | a GDPR (RGPD) erasure; a plain delete stays "supprimer" |
| excerpt | extrait | an archived issue's |
| export data (GDPR) | exporter les données | |
| Forms & public pages | Formulaires et pages publiques | |
| hosted page (a form type) | page hébergée | as "hosted" above |
| Import from {tool} (a source) | Import depuis {tool} | a contact's or suppression's source: "Import depuis Mailchimp" |
| live log (an import's) | journal en direct | |
| merge (a duplicate) | fusionner | "Merge · fills first name": "Fusionner · complète le prénom" |
| Migration (where a suppression came from) | Migration | an import from another email tool |
| offer a download | proposer un téléchargement | the file a form gives after confirmation |
| popup (form) | pop-up | |
| preference center | centre de préférences | in the app; subscribers read "page de préférences" |
| publisher (name) | éditeur | heads every public page: "Nom de l’éditeur"; not the Editor role |
| re-permission | redemande de consentement | "re-permission flow": "scénario de redemande de consentement" |
| read-only link | lien en lecture seule | a shared report's |
| RSS feed | flux RSS | |
| self-hosted (a provider) | auto-hébergé | "Postal (auto-hébergé)" |
| sender health | santé de l’expéditeur | the Overview's score; reputation is "réputation" |
| signup · submission (form) | inscription · soumission | "Views · Submissions": "Vues · Soumissions" |
| single opt-in | opt-in simple | double opt-in stays "double opt-in" |
| site key · secret key (Turnstile) | clé de site · clé secrète | "Turnstile" stays |
| source status (an import's: subscribed, cleaned…) | as the source writes it | the consent table and the consent history keep the tool's own word |
| spam trap (audience, imports) | piège à spam | the suppression list's and the importer's word, as everywhere |
| subscribe page · confirmation page · unsubscribe page | page d’inscription · page de confirmation · page de désabonnement | |
| suppressed | exclu | on the "liste de suppression"; never "supprimé", which reads as deleted |
| System stream · System mail | flux Système · e-mails Système | the stream's name keeps its capital |
| tag | tag | |
| title (archived issue) | titre | "objet" is an email's subject |
| workbook · sheet (XLSX) | classeur · feuille | as Excel's French names them |

## Products, stores and transactional email

| English | Français | Note |
|---|---|---|
| abandoned checkout · abandoned cart | paiement abandonné · panier abandonné | as Shopify's French admin says it |
| accepted marketing · didn't accept | accepte le marketing · n’accepte pas | a store customer's consent |
| attribution · credited to | attribution · attribué à | "Attribution window": "Fenêtre d’attribution" |
| Bought a product (trigger · condition) | Achète un produit · A acheté un produit | |
| buyer · customer | acheteur · client | a product's buyer; a store's customer |
| checkout · at checkout | paiement · au moment du paiement | "Stripe Checkout" keeps its name |
| click token | jeton de clic | |
| collection (store) | collection | |
| compare-at price | prix avant réduction | |
| cover (product) | image de couverture | |
| delivery · delivery email · delivery template | livraison · e-mail de livraison · modèle d’e-mail de livraison | a digital product's download link; a delivered email is still "Délivré" |
| dispute | litige | never "plainte", which is a spam complaint |
| download link · downloads | lien de téléchargement · téléchargements | "4 of 5 downloads left": "4 téléchargements restants sur 5" |
| failover · primary · fallback | basculement · principal · secours | a route's providers |
| Gross · Net | Brut · Net | product revenue before and after refunds |
| idempotency key | clé d’idempotence | |
| list offer | proposition d’inscription | the checkbox next to a product's Buy button |
| live · test (payments, keys, modes) | production · test (as Earn has it) | the message log's live updates are "En direct" |
| median delivery | délai de livraison médian | |
| message (transactional) · Messages (tab) | message · Messages | |
| order | commande | |
| Order placed · First order · Placed an order | Commande passée · Première commande · A passé une commande | store triggers and the goal |
| Pay what you want · One-time | Prix libre · Paiement unique | a product's price |
| product · digital product | produit · produit numérique | |
| Queued · Deferred · Bounced · Complained · Rejected | En file d’attente · Différé · Rebond · Plainte · Rejeté | a message's status; Delivered is "Délivré" |
| Raw (SMTP) | Brut (SMTP) | mail an app sends already rendered |
| reconciliation | rapprochement | a store's nightly check |
| refund · refunded | remboursement · remboursé | |
| revoke access · restore access | révoquer l’accès · rétablir l’accès | a product order's download |
| route · Transactional route | route · route Transactionnel | as "Route Diffusions" |
| sales page · default product page · shop | page de vente · page produit par défaut · boutique | |
| send-only key · SMTP credential | clé d’envoi uniquement · identifiants SMTP | "SMTP relay" keeps its name |
| single opt-in | opt-in simple | double opt-in stays "double opt-in" |
| store · connect · disconnect · sync | boutique · connecter · déconnecter · synchroniser | Shopify and WooCommerce; their menu paths and field names stay as their screens give them |
| suppressed · opted out | exclu · désinscrit | why a message wasn't sent |
| webhook topic | sujet | orders/create stays as it is |
| write-only · sealed | en écriture seule · scellé | a secret that's never shown again |

## Recurring digests

| English | Français | Note |
|---|---|---|
| approval rule · approver | règle d’approbation · approbateur | as in Approvals |
| archive (public) | archives publiques | where sent issues are listed |
| back catalogue | anciens articles | a feed's older posts |
| category · tag (of a post) | catégorie · étiquette | WordPress's own French words |
| Collecting · Draft ready · Not sent (an issue) | Collecte en cours · Brouillon prêt · Non envoyé | an issue's own states, agreeing with "numéro"; the others are the campaign words |
| Create a draft · Send automatically · draft mode | Créer un brouillon · Envoyer automatiquement · mode brouillon | the two sending modes |
| excerpt · featured image | extrait · image mise en avant | featured image as WordPress says it |
| fallback image | image de repli | |
| feed · feed address | flux · adresse du flux | as Integrations; RSS 2.0, Atom 1.0 and JSON Feed 1.1 keep their names |
| feed item · Feed items (the block) | élément · Éléments du flux | |
| feed token | variable de flux | a Liquid token: "variable", as in Grow |
| held · release (an issue) | retenu · libérer | "Release issue": "Libérer ce numéro" |
| hours before (preparation) | heures avant l’envoi | "{count} hours before": "{count, plural, one {# heure avant l’envoi} other {# heures avant l’envoi}}"; "3 h avant" alone could read as a clock time |
| issue · Next issue · issue #143 | numéro · Prochain numéro · n° 143 | as a magazine's issue, and as the public pages say it |
| late limit | retard maximal | how late an issue may still go out |
| layout: List · Cards · Grid | Liste · Cartes · Grille | the Feed items block's layouts |
| left out (by a filter) | écarté | "Left out by a filter": "Écarté par un filtre" |
| owner (of a digest) | responsable | the member issues send under; not the Owner role |
| percentage points (pts) | points (pts) | "{delta} pts": "{delta} pts" |
| post | article | |
| prepare · Prepared at 07:00 | préparer · Préparé à 07:00 | |
| recurring digest · digest | récapitulatif récurrent · récapitulatif | as Integrations; the Recurring tab is "Récurrents" |
| Regular (campaign) | Standard | the other choice in New campaign |
| sending window | plage d’envoi | as Automations' send window |
| skip · skipped | ignorer · ignoré | |
| SSRF guard | protection SSRF | |
| starter (a template to begin from) | modèle de départ | a starter's name ("Blog digest") stays as it is; what a new digest's starter writes into the email (subject, preheader) follows the workspace's content language |
| starting point | point de départ | |
| turn on · pause (a digest) | activer · mettre en pause | |
| weekdays (Monday to Friday) | du lundi au vendredi | short: "lun.–ven.", the region format's own abbreviations |

## AI assistant

| English | Français | Note |
|---|---|---|
| accept · edit · reject (a suggestion) | accepter · modifier · refuser | |
| Accepted with edits · No decision | Acceptée avec modifications · Aucune décision | outcomes in the activity log, agreeing with "suggestion" |
| activity log (AI) | journal d’activité de l’IA | the audit log stays "journal d’audit" |
| AI assistant · Assist (the menu) | assistant IA · Assistant | |
| AI budget | budget IA | |
| cap · monthly cap | plafond · plafond mensuel | in dollars or tokens; "No cap": "Aucun plafond" |
| condition schema · segment builder | schéma de conditions · éditeur de segments | |
| deployment (Azure) | déploiement | |
| describe the segment · Draft (its button) | décrire le segment · Rédiger | what it drafts is a "brouillon" |
| feature (of the assistant) | fonctionnalité | |
| Image input · Structured output | Entrée image · Sortie structurée | what a model can do |
| input · output (tokens, prices) | entrée · sortie | |
| instruction (to rewrite) | consigne | "Rewrite with an instruction": "Réécrire selon une consigne" |
| keep · discard (a summary) | conserver · rejeter | "kept summaries": "résumés conservés" |
| model · connect a model | modèle · connecter un modèle | "modèle d’IA" where an email template could be meant; model names stay as they are |
| private address | adresse privée | a server on a private network |
| prompt · test prompt | prompt · prompt de test | a web push prompt is an "invite" |
| refused by the model | refusé par le modèle | |
| resets (the usage) | réinitialisé | "resets {date}": "réinitialisé le {date}" |
| rewrite · shorten · friendlier · more formal · more direct | réécrire · raccourcir · plus chaleureux · plus formel · plus direct | |
| subject ideas | idées d’objet | |
| suggestion | suggestion | |
| summary (of results) | résumé | "Summarize results": "Résumer les résultats" |
| token (of a model) | token | what the provider bills; a Liquid token is "variable", an API token "jeton" |
| tone | ton | "Rewrite, shorten and tone": "Réécrire, raccourcir et changer le ton" |
| usage | consommation | |

## Placement tests

| English | Français | Note |
|---|---|---|
| app password · app-specific password | mot de passe d’application · mot de passe pour app | Google's and Apple's own words |
| arrived late | arrivé en retard | |
| copy (of the test email) | copie | |
| deferral | report | as Campaigns' "reports" |
| folder · tab | dossier · onglet | a mailbox's own names (Promotions, Junk Email) stay as it gives them |
| folders: Sent · Draft · Archive · Trash | Envoyés · Brouillons · Archives · Corbeille | the folders a test never reads |
| hand-off | prise en charge | when the sending provider takes a copy |
| headers | en-têtes | "All headers": "Tous les en-têtes"; header names stay |
| Health (Deliverability › Health) | Santé | as Integrations' Health |
| Healthy · Warming up · Degraded · Down · Paused · Sandbox (a provider) | Opérationnel · En chauffe · Dégradé · En panne · En pause · Sandbox | |
| Inbox · Tab · Spam · Not arrived | Boîte de réception · Onglet · Spam · Non arrivé | the four results |
| median time to arrive | délai d’arrivée médian | a median, as Approvals' median wait |
| pass · fail · aligned (authentication) | réussi · échec · aligné | SPF, DKIM and DMARC keep their names |
| placement · placement test | placement · test de placement | where a copy lands |
| Queued (a copy) | En file d’attente | |
| remove (a seed) | retirer | deleting copies stays "supprimer" |
| round (of copies) | vague | "Round 2": "Vague 2" |
| sample (the seeds) | échantillon | as an A/B test's sample |
| seed · seed mailbox | boîte témoin | mailboxes that exist only for tests |
| stop waiting | arrêter d’attendre | |
| test (one run) | test | "Tests · 12" |
| test token | jeton de test | what marks a copy as a test's |
| vs (providers compared) | face à | "SES vs Postal": "SES face à Postal" |
| warm-up cap | plafond de chauffe | a warming-up provider's daily cap |
| warm-up planner · warm-up plan | planificateur de chauffe · plan de chauffe | |

## Address verification

| English | Français | Note |
|---|---|---|
| abuse · catch-all domain · disposable · role address | plaintes fréquentes · domaine catch-all · jetable · adresse générique | why an address is risky; tags such as verify:catch-all stay as written |
| Add credits · Low credits | Ajouter des crédits · Crédits bientôt épuisés | |
| balance · credits | solde · crédits | the prepaid credits at the service |
| bulk check · real-time check | vérification en masse · vérification en temps réel | |
| check (one address) | vérification | "{price} a check": "{price} par vérification" |
| data region | région des données | |
| Found by | Détecté par | which check found a risky address |
| imports (of contacts) | imports de contacts | "Bulk checks and imports": "Vérifications en masse et imports de contacts" |
| job · Resume | tâche · Reprendre | |
| spam trap | piège à spam | |
| Uses (of a service) | Utilisations | |
| Valid · Invalid · Risky · Unknown | Valide · Non valide · À risque · Inconnue | the four results, agreeing with "adresse" |
| verification service · verify | service de vérification · vérifier | |

## Words used everywhere

| English | Français |
|---|---|
| contact | contact |
| subscriber | abonné |
| list | liste |
| segment | segment |
| template | modèle |
| sending provider | fournisseur d’envoi |
| mailbox provider | fournisseur de messagerie (short: messagerie) |
| sender | expéditeur |
| domain | domaine |
| record (DNS) | enregistrement |
| verified | vérifié |
| sign in · sign out | se connecter · se déconnecter (the act: connexion · déconnexion) |
| password · passkey | mot de passe · clé d’accès |
| authenticator app | application d’authentification |
| recovery code | code de récupération |
| two-factor | authentification à deux facteurs (2FA) |
| invitation · invite | invitation · inviter |
| permission | autorisation |
| Ask Marcus Chen or Linh Nguyen. | Demandez à Marcus Chen ou à Linh Nguyen. |
| notification | notification |
| approval | approbation |
| retry · try again | réessayer |
| cancel · close · back | annuler · fermer · retour |
| copy · copied | copier · copié |
| optional · required | facultatif · obligatoire |
| unsubscribe | se désabonner · désabonnement |
| open rate · click rate | taux d’ouverture · taux de clic |
| email (the word) | e-mail (plural e-mails; "adresse e-mail") |
| subject (of an email) | objet |
