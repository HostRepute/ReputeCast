# Spanish glossary (Español)

The words ReputeCast uses in Spanish, so every screen, email and error
says the same thing the same way. It follows the Vietnamese glossary
([`glossary-vi.md`](glossary-vi.md)) section by section, with every term it
settles, and takes the Spanish that email-marketing tools (Mailchimp, Brevo,
HubSpot) and the services ReputeCast connects to (Google, Apple, Microsoft,
Meta, Stripe, Shopify, WordPress) already use. The design has no Spanish
stress-test boards yet: check Spanish screens in the room the pseudo-locale
(+40%) makes.

> **Status: draft.** These are the first Spanish words in ReputeCast. A
> native speaker reviews them before Spanish ships in a release (brief
> §8.17). Change a term here first, then in the translations
> (`web/src/**/messages.es.ts`, `internal/<package>/messages/es.json`).

How to add or complete a language: [`translating.md`](translating.md).

## Voice

- Plain and specific, like the English: say what happened and what to do.
  No exclamation marks, so no "¡". Questions keep both marks: "¿Salir del
  editor?".
- Neutral international Spanish that reads naturally in Spain and in Latin
  America. Where the two differ, use the word both understand (the table
  below) and no regional idioms.
- Address the reader as **tú**: "Revisa tu bandeja de entrada", "Puedes
  cambiarlo más tarde". Never "usted", never "vosotros": for a group, name it
  ("Todo el equipo puede verlo").
- Buttons, menu items, a row's actions and links that do something use the
  **infinitive**: "Guardar", "Enviar campaña", "Añadir contacto", "Volver al
  editor". Never the imperative ("Guarda", "Envía"). Sentences (instructions,
  help text, errors, empty states) address tú: "Añade un remitente para
  poder enviar.", "Corrige los avisos y vuelve a intentarlo." A dialog's
  question uses the infinitive: "¿Eliminar la lista Beta Program?".
- Sentence case, as in English. Only the first word and names start with a
  capital: "Nueva campaña", not "Nueva Campaña". Days and months are lower
  case in a sentence ("el lunes", "en octubre", as the region format writes
  them); a label or a choice that is only a day starts with a capital like
  any label ("Lunes").
- Inclusive, without slashes. Never "los/las", "usuarios/as", "@" or "-x".
  Prefer words without gender: "personas", "miembros del equipo",
  "contactos", "quien…" ("Pídeselo a quien administra la instancia"), "Todas
  las personas con acceso". Role names are labels, the same for everyone, in
  the form Google Workspace uses: Propietario, Administrador, Editor, Lector
  ("Linh tiene el rol Propietario").
- Agreement. A status word agrees with the noun it describes on that
  screen: a campaign, a version or an issue is feminine ("Programada",
  "Enviada"), a member, a contact or an email masculine ("Activo",
  "Entregado"). An argument doesn't tell you its gender, so never make an
  adjective agree with `{name}`: put the noun before it ("la campaña
  “{name}”") or use "se" ("Se envió “{name}”").
- Plurals: Spanish has the CLDR categories `one`, `many` and `other`. Write
  `one` and `other`: `{count, plural, one {# campaña} other {# campañas}}`.
  `many` isn't needed: it only applies to millions in compact notation and
  falls back to `other`. Keep `#` in both branches, and add `=0 {…}` where
  zero reads differently ("ninguna campaña").
- Numbers and dates come from the region format, never typed by hand. In
  es-ES: 187.312 · 1204 (four digits take no dot) · 99,2 % · 18,21 US$ ·
  mié, 14 oct 2026 · 9:00. Before a time write "a las {time}".
- Relative times, as the app writes them: "ahora", "hace 4 minutos", "hace 2
  horas", "ayer", "dentro de 5 días"; short: "hace 4 min", "2 h".
- Overlines (table headers, group labels) stay uppercase and keep their
  accents: "DESTINATARIOS", "ÚLTIMA ACTIVIDAD".
- Language names: the app passes `{language}` and `{languages}` in the form
  that stands alone, with a capital ("Inglés") on its own, at the start of a
  sentence and after a colon or "·", and lowercases them inside a sentence
  ("Añadir inglés", "Versión en inglés", "Los lectores en alemán reciben la
  versión en inglés"). Write `{language}` and never type a language name or
  an article around it. Prefer "en {language}" or no article ("Añadir
  {language}", "Textos en {language}"); never "el {language}". A list comes
  joined, with "y/e" and "o/u" already chosen, and only its first name
  capitalised where it stands alone ("Inglés, español y alemán").
- Punctuation: “ ” around a name ("la campaña “October Product Update”"),
  never straight quotes; the ellipsis as one character (…); "·" between items,
  as in English; "n.º" for número. Units and short forms take no period and
  no plural: "2 min", "4 h", "14/s", "102 KB".
- Length: Spanish runs 20–30% longer than English. Stay within about 30%
  where you can. Where the natural word is long, the tables give a short form
  ("short:") for tabs, buttons and table headers. Buttons don't wrap and are
  never abbreviated with a period.
- English words that stay (webhook, endpoint, token, payload, sandbox, feed,
  spam, catch-all, slug, prompt) are written in roman type, lower case unless
  they're names, with Spanish gender and plurals: el webhook, los webhooks,
  el endpoint, el token, el feed, la API, la URL (las URL), la IP (las IP),
  el SMS (los SMS).

Common words, chosen so they read the same in Spain and Latin America:

| English | Español |
|---|---|
| email (a message) · email address | correo · dirección de correo; "Correo electrónico" as a form field's label; never "email", "e-mail" or "mail" |
| file | archivo (never "fichero") |
| click · to click | clic · hacer clic (never "pinchar", "cliquear") |
| delete · remove | eliminar (gone for good) · quitar (out of a list, a group, a role); "borrar definitivamente" is only a GDPR erasure |
| upload · download | subir · descargar |
| settings · default · by default | configuración (never "ajustes") · predeterminado · por defecto |
| cost | costo (never "coste") |
| live (happening now) | en tiempo real (never "en directo", "en vivo") |
| computer · mobile phone | dispositivo or equipo (never "ordenador", "computadora") · móvil (never "celular") |
| manage | gestionar |
| app | aplicación; "app" only in a provider's own words (Meta, Apple) or where space is tight |
| tap | tocar (never "pulsar") |
| warning | aviso (never "advertencia") |
| report | informe (never "reporte") |
| period | periodo |

## Never translated

Workspace content (campaign, list, segment, template, form and custom role
names such as "Content Writer"), people's names and email addresses,
permission keys (`campaigns:send`), audit action names, API error codes,
provider names and codes (Amazon SES, Postal, Postmark, SES), product names
(ReputeCast, HostRepute, Gmail, Outlook), DNS records and record types (SPF,
DKIM, DMARC, BIMI, CNAME, MAIL FROM), headers (List-Unsubscribe), code
samples, CLI output, "MCP", "2FA", "TOTP", "IdP", "SSO", "JIT", "A/B",
"HTML", "CSS", "API", "webhook".

## App shell and navigation

| English | Español |
|---|---|
| Overview | Resumen |
| Campaigns | Campañas |
| Automations | Automatizaciones |
| Transactional | Transaccional |
| Audience | Audiencia |
| Grow | Crecimiento |
| Earn | Monetización |
| Forms & pages | Formularios y páginas |
| Deliverability | Entregabilidad |
| Sending providers | Proveedores de envío (short: Proveedores) |
| Settings | Configuración |
| More | Más |
| Skip to content | Ir al contenido |
| Import contacts | Importar contactos |
| HostRepute connected | HostRepute conectado |
| synced 4 min ago | sincronizado hace 4 min |
| Account & security | Cuenta y seguridad |
| Notification preferences | Preferencias de notificaciones (short: Notificaciones) |
| Workspace settings | Configuración del espacio de trabajo (short: Espacio de trabajo) |
| Instance admin | Administración de la instancia (short: Administración) |
| admins only | solo administradores |
| Sign out | Cerrar sesión |
| Language: English | Idioma: Español (the item names the current language in its own name) |
| unexpected error · server log | error inesperado · registro del servidor |

## Settings tabs

| English | Español |
|---|---|
| General | General |
| Senders | Remitentes |
| Providers | Proveedores |
| Team | Equipo |
| Roles | Roles |
| Approvals | Aprobaciones |
| Keys & MCP | Claves y MCP |
| Webhooks | Webhooks |
| Integrations | Integraciones |
| Audit log | Registro de auditoría (short: Auditoría) |

## Roles and people

| English | Español |
|---|---|
| Owner | Propietario |
| Admin | Administrador (short: Admin) |
| Editor | Editor |
| Viewer | Lector |
| custom (role) | personalizado |
| instance admin (the area, a label: "Instance admin", "· instance admin") | administración de la instancia |
| instance admin (a person, in a sentence: "Ask your instance admin") | quien administra la instancia ("Pídeselo a quien administra la instancia") |
| break-glass account | cuenta de emergencia |
| member | miembro ("el miembro", "la miembro"; "miembros del equipo" where it must be explicit) |
| invitee | persona invitada |
| workspace | espacio de trabajo |
| instance | instancia |
| team | equipo |

## Statuses

| English | Español |
|---|---|
| Draft | Borrador |
| Draft · blocked | Borrador · bloqueado |
| Scheduled | Programada |
| Sending | Enviando |
| Sent | Enviada |
| Active | Activo (a member; a feminine noun takes "Activa") |
| Invited | Invitado |
| Deactivated | Desactivado |
| On · Off | Activado · Desactivado (short: Sí · No) |
| Via IdP | Vía IdP |
| Required | Obligatoria (two-factor: la verificación en dos pasos) |
| Pass · Warn · Block | Correcto · Aviso · Bloqueo |
| Ready · Ready with warnings · Blocked | Preparada · Preparada, con avisos · Bloqueada (short: Con avisos); never "Lista" alone, which reads as a list ("Lista para publicar" is clear) |
| Healthy | Operativo |
| Warming up | En calentamiento |
| New | Nuevo |
| Not saved | Sin guardar |
| Development | Desarrollo |
| Auto-paused | En pausa automática |
| Pending approval · Approved | Pendiente de aprobación (short: Pendiente) · Aprobada |
| Cancelled · Failed | Cancelada · Fallida |

## Campaigns

| English | Español |
|---|---|
| Campaign | Campaña |
| New campaign | Nueva campaña |
| All · Drafts · Scheduled · Sending · Sent | Todas · Borradores · Programadas · Enviando · Enviadas |
| Search campaigns | Buscar campañas |
| Sent · last 30 days | Enviados · últimos 30 días |
| Cost · last 30 days | Costo · últimos 30 días |
| Recipients | Destinatarios |
| Delivered | Entregados |
| Human opens | Aperturas humanas (short: Aperturas) |
| Clicks | Clics |
| click-to-open | clics por apertura |
| bounces · complaints | rebotes · quejas |
| machine opens shown separately | aperturas automáticas, contadas aparte |
| Readiness | Revisión previa (short: Revisión) |
| 3 blocking issues | 3 problemas bloqueantes |
| Edit · Report · Duplicate | Editar · Informe · Duplicar |
| not set | sin definir |
| created 08:55 | creada a las 8:55 |
| edited 09:31 by Linh Nguyen | Linh Nguyen la editó a las 9:31 |
| All subscribers | Todos los suscriptores |
| Broadcasts route | Ruta Envíos masivos |
| A/B subject, winner B | Asunto A/B, gana la B |
| auto-paused once, Yahoo deferrals | pausada automáticamente una vez por aplazamientos de Yahoo |
| Showing 9 of 48 | Mostrando 9 de 48 |
| Previous · Next | Anterior · Siguiente |
| Live monitor | Monitor en tiempo real (short: Monitor) |
| imported from Brevo · Brevo · imported | importada de Brevo · Brevo · importada |
| segment (in an audience: 2 segments) | segmento (2 segmentos) |

## Review & send

| English | Español |
|---|---|
| Review & send | Revisar y enviar |
| Back to editor | Volver al editor |
| Run checks again | Volver a comprobar |
| Draft · autosaved 2 min ago | Borrador · guardado automáticamente hace 2 min |
| Fix both warnings | Corregir ambos avisos |
| Schedule send to 31,604 recipients | Programar envío a 31.604 destinatarios |
| 27 checks passed, 2 warnings, 0 blocks | 27 comprobaciones correctas, 2 avisos, 0 bloqueos |
| Sender reputation | Reputación del remitente |
| blocklist | lista de bloqueo |
| suppression list (shared) | lista de supresión (compartida) |
| role addresses | direcciones genéricas (info@, ventas@) |
| Duplicates | Duplicados |
| Final recipients | Destinatarios finales |
| Route split | Reparto entre rutas |
| Content · Links · Compliance | Contenido · Enlaces · Cumplimiento |
| Audience quality | Calidad de la audiencia |
| Routing · stream | Enrutamiento · flujo |
| Broadcasts stream | Flujo Envíos masivos |
| Send time · Send now · Schedule | Hora de envío · Enviar ahora · Programar |
| Best time per recipient | Mejor hora para cada destinatario |
| Test email · Send test | Correo de prueba · Enviar prueba |
| Auto-pause thresholds | Umbrales de pausa automática |
| Bounce rate · Complaint rate | Tasa de rebote · Tasa de quejas |
| per mailbox provider | por proveedor de buzón |
| warm-up (day 9 of 21) | calentamiento (día 9 de 21) |
| 14 /s | 14/s |
| Cost · Time | Costo · Tiempo |

### Send Readiness's checks

The findings the server writes for each check (`internal/readiness`).

| English | Español | Note |
|---|---|---|
| check (a row) · passed · not checked · timed out | comprobación · correcta · sin comprobar · tiempo agotado | |
| the verdict waits for this one | el veredicto espera a esta comprobación | |
| I understand … (a warning's confirmation) | Entiendo … | as the web's "I understand: {title}.": "Entiendo: {title}." |
| unengaged (contacts) | sin interacción | no open or click in the sunset policy's days |
| Sunset tier · At risk (names) | grupo En retirada · En riesgo | tier names keep their capitals |
| spam trigger · spam heuristic | indicio de spam | "trips 3 spam heuristics": "activa 3 indicios de spam" |
| all caps · repeated punctuation · urgency phrasing · money words · click-bait | todo en mayúsculas · puntuación repetida · tono de urgencia · palabras sobre dinero · cebo de clics | |
| image-to-text ratio | proporción entre imagen y texto | |
| Gmail's 102 KB clipping limit · clips | límite de recorte de 102 KB de Gmail · recorta | |
| inline(d) CSS · minify | CSS en línea · minificar | |
| one-click unsubscribe (headers) | (cabeceras de) baja con un clic | header names stay: List-Unsubscribe |
| postal address | dirección postal | |
| bulk-sender rules | requisitos para remitentes masivos | Gmail's and Yahoo's |
| URL shortener · redirects through | acortador de URL · redirige a través de | |
| responds · doesn't respond | responde · no responde | a link |
| review thresholds · pause level (SES) | umbrales de revisión · nivel de pausa | |
| no enforcement | sin restricciones | an account's standing |
| terms allow bulk marketing | las condiciones permiten el marketing masivo | |
| production access | acceso de producción | as in Sending |
| double-tracked · rewrites links | con doble seguimiento · reescribe los enlaces | |
| aligned (DMARC, SPF) | alineado | |
| Easy DKIM CNAME · selector | CNAME de Easy DKIM · selector | names and record words stay |
| custom MAIL FROM | MAIL FROM personalizado | |
| Conditional content · conditional block · version | Contenido condicional · bloque condicional · versión | |
| Visibility rule | regla de visibilidad | |
| poll · survey · answer page | votación · encuesta · página de respuesta | |
| risky · catch-all · disposable (addresses) | de riesgo · catch-all · desechable | |
| inactive (contacts) | inactivos | At risk and Sunset together |
| paid members · trial · complimentary · gift (access) | suscriptores de pago · periodo de prueba · acceso de cortesía · regalo | as Earn has it |
| Sponsor block · creative · booking · Sponsored (label) | bloque Patrocinador · creatividad · reserva · Patrocinado | |
| Upgrade block · offer | bloque Mejora · oferta | |
| page copy · preference center | textos de la página · centro de preferencias | |
| bare URL · skipped heading level | URL desnuda · nivel de encabezado omitido | accessibility |
| linked image | imagen enlazada | |
| self-hosted (pricing) · plan | autoalojado · según el plan | a provider's price |

## Sending and deliverability

Sending providers & routing, the connect wizard, Settings › Senders and
Providers, Deliverability, HostRepute and the live monitor. In alphabetical
order of the English.

| English | Español | Note |
|---|---|---|
| account health (a provider's) | estado de la cuenta | |
| adapter · community adapter · adapter registry | adaptador · adaptador de la comunidad · registro de adaptadores | |
| aggregate report (DMARC) | informe agregado | |
| alignment (DMARC) | alineación | "alineación DMARC" |
| asset (a domain or IP HostRepute monitors) | recurso | never "activo", the status |
| backfill (statuses from an API) | recuperar | "Recuperar los estados desde la API de Mailgun" |
| blocklist · listed · clean | lista de bloqueo · en la lista · limpio | |
| bounce mailbox | buzón de rebotes | SMTP providers |
| bounce · hard bounce · soft bounce | rebote · rebote permanente · rebote temporal | the rate is "tasa de rebote"; never "rebote duro/suave"; as in the Grow section |
| Broadcasts · Automations · System · Transactional (streams) | Envíos masivos · Automatizaciones · Sistema · Transaccional | names, so capitalised in a sentence: "el flujo Sistema" |
| bulk marketing | marketing masivo | a provider's terms |
| cap (daily, today's) | límite | "Límite de hoy" |
| catalog (of providers) | catálogo | the wizard's first step |
| condition (routing) | condición | |
| credentials (a provider's) | credenciales | |
| dedicated IP · shared pool | IP dedicada · grupo de IP compartido | "IP" is feminine: "la IP", "las IP" |
| defer · deferral | aplazar · aplazamiento | "Yahoo aplaza la entrega" |
| degraded · down | degradado · caído | provider statuses |
| delisting | exclusión de la lista | "Solicitar la exclusión" |
| event log · raw event · normalized event | registro de eventos · evento sin procesar · evento normalizado | |
| failover · fallback · primary | conmutación por error · respaldo · principal | "proveedor de respaldo", "proveedor principal"; short: "conmutación" |
| feedback loop | bucle de retroalimentación | |
| flow (an automation's, in sending text) | recorrido | "flujo" is a stream: "correos enviados por los recorridos de automatización"; on the Automations screens a flow is "recorrido" |
| guard (Warm-up page) | umbral de protección | "Tasa de rebote · umbral de protección 2 %" |
| held (recipients) | retenidos | |
| incident · outage | incidente · caída | |
| live sending monitor | monitor de envíos en tiempo real | "Live monitor" is "Monitor en tiempo real" |
| maintainer · unmaintained (adapter) | mantenedor · sin mantenimiento | |
| mapping (webhook, JSONPath) | asignación | |
| message id | ID del mensaje | |
| monitoring · monitored | supervisión · supervisado | HostRepute; never "monitoreo" or "monitorización" |
| nameserver | servidor de nombres | |
| polling (instead of webhooks) | consulta periódica | "consulta la API de eventos cada minuto" |
| Postmaster compliance status | estado de cumplimiento | Google Postmaster Tools v2 has no domain or IP reputation |
| production access (SES) | acceso de producción | the SES sandbox stays "sandbox" |
| provisioning | aprovisionamiento | SNS topics and configuration sets |
| pulled in · pushed out · two-way (suppression sync) | recibidas · enviadas · bidireccional | suppressions, from and to the provider |
| quota | cuota | |
| read-only link (a live send) | enlace de solo lectura | |
| recommended (an option, a badge) | recomendado | "IMAP (recomendado)"; a recommendation's heading is "Recomendación" |
| region (AWS) | región | AWS's own word; not the region format |
| reputation · domain reputation · IP reputation | reputación · reputación del dominio · reputación de la IP | |
| reroute | reenrutar | |
| reverse DNS | DNS inverso | |
| route tester | probador de rutas | |
| route · routing · weighted split | ruta · enrutamiento · reparto ponderado | |
| secret (webhook signing) | secreto | "secreto de firma del webhook" |
| seed list | lista de buzones de prueba | one address of yours at each mailbox provider |
| self-test (of an endpoint) | autocomprobación | |
| Send Readiness | revisión previa | the checks before a send; lower case in a sentence: "la revisión previa vuelve a ejecutarse" |
| share (of a split · of a send) | proporción · parte | |
| slowdown (warm-up) | ralentización | |
| SNDS filter result: Green · Yellow · Red | Verde · Amarillo · Rojo | |
| spam rate · spam trap | tasa de spam · trampa de spam | |
| spam trap hit | impacto en una trampa de spam | |
| stale (data) · cached results | desactualizados · resultados en caché | |
| stream | flujo | |
| suppression sync | sincronización de supresiones | |
| throttle · throttling | limitar la velocidad · limitación de velocidad | |
| tracking · tracking domain | seguimiento · dominio de seguimiento | |
| verdict (HostRepute): Good · Fair · Poor · Bad | Bueno · Aceptable · Deficiente · Malo | |
| warm-up · warmed up · warm-up guard · ramp | calentamiento · calentamiento completado · protección del calentamiento · progresión | |
| webhook endpoint · registration | endpoint del webhook · registro | |
| write-only (a key) | solo escritura | "se puede reemplazar, pero nunca volver a leer" |
| zone file | archivo de zona | |

"sandbox", "configuration set", "SNS topic", "IAM", "ARN", "CloudFormation",
"stack", "return-path", "DSN", "ARF", "rua", "JSONPath", "manifest.json",
"SDK", "endpoint", "token", "Postmaster", "SNDS" and "JMRP" stay as they
are, and so do the AWS field names ("Access key ID", "Secret access key",
"RoleArn"), the SES region names, a provider's own stream names (Postmark's
"Broadcast" stream: "flujo Broadcast") and the results in DNS records and
reports ("pass", "fail", "p=quarantine"). Spanish has its own words for a
few that other languages keep: "túnel", "proxy inverso", "idempotencia".

### What adapters and the connect flow write

The server's words about a provider: the catalog's cards and chips, an
adapter's fields and terms, the account's standing, the stepper and the
test's timeline.

| English | Español | Note |
|---|---|---|
| Production · Sandbox (an account's standing) | Producción · Sandbox | "Producción · sin restricciones" |
| Two-way · Pull only · Push only · Local only (suppression sync) | Bidireccional · Solo recibir · Solo enviar · Solo local | the Adapters tab |
| self-hosted | autoalojado | "Postal (autoalojado)" |
| Any SMTP server · Generic HTTP adapter | Cualquier servidor SMTP · Adaptador HTTP genérico | catalog names |
| Batch 50 · Quota discovery · Sandbox detection (chips) | Lotes de 50 · Detección de cuota · Detección de sandbox | |
| Dedicated IPs, paid · Pro plan | IP dedicadas, de pago · plan Pro | the plan's name stays |
| read-only call | llamada de solo lectura | "comprobado con una llamada de solo lectura" |
| Round trip 2.4 s | Ida y vuelta: 2,4 s | the test send |
| Mailpit (development) | Mailpit (desarrollo) | |
| a price in an adapter's words | 0,10 US$ por cada 1000 correos | decimal comma, no dot in four digits, US$ after the amount, as the region format writes them |

A provider's names for its keys, plans, menus and stream types stay as the
provider writes them ("Server API token", "Private API key", "Account SID",
"Auth token", "Full access", "Broadcasts API", "Scale plan"), and so do its
event names ("Delivery", "SpamComplaint") and its error codes.

### What the sending engine, Deliverability and HostRepute write

The live monitor's incident panel (likely causes, recommendations and the
ways to continue), its notifications and feed, Deliverability's account
health and check runs, the HostRepute connection, route notes on a
contact's timeline and why a transactional message failed.

| English | Español | Note |
|---|---|---|
| "October Product Update" auto-paused · Outlook | “October Product Update” en pausa automática · Outlook | a campaign's name between “ ” |
| built-in daily check (blocklists) | comprobación diaria integrada | without HostRepute |
| check run · lookups that didn't answer | ejecución de comprobaciones · consultas sin respuesta | |
| connection errors · throttling errors · API errors | errores de conexión · errores de limitación de velocidad · errores de API | "Postal devolvió errores de conexión durante 3 minutos" |
| decision requested · waiting for a decision | decisión solicitada · esperando tu decisión | a held share |
| health check · still failing · answering again | comprobación de estado · sigue fallando · vuelve a responder | the outage log's lines |
| never-engaged · lapsed (contacts) | sin interacción nunca · han dejado de interactuar | |
| placement ("protects placement") | ubicación en la bandeja | as in Placement tests |
| Reputation dashboard (SES) | Panel de reputación | |
| Resume as is · Resume with changes · Cancel the Outlook share · Retry Postal · Move 1,204 to SES | Reanudar sin cambios · Reanudar con cambios · Cancelar la parte de Outlook · Reintentar Postal · Mover 1204 a SES | the incident panel's ways to continue; "(recomendado)" after the recommended one |
| route note: SES share of the 90/10 split · by rule · instead of · failed over to | Parte de SES en el reparto 90/10 · por regla · en lugar de · conmutó a | |
| scope (a HostRepute token's) | ámbito | the scopes' keys stay: monitor:read |
| Send held (a scheduled send) | Envío retenido | |
| status page (a provider's) | página de estado | |
| web process (a self-hosted provider's) | proceso web | |

The providers' own replies stay as they come: SMTP codes and texts
("421 4.7.0 TSS04"), API errors, the outage log's error lines
("ECONNREFUSED mta.acme.io:443") and a bounce's diagnostic.

## Transactional templates

| English | Español |
|---|---|
| transactional template · transactional email | plantilla transaccional · correo transaccional |
| draft · publish · v3 live | borrador · publicar · v3 publicada |
| version · Publish again | versión · Volver a publicar |
| variable · declare · required · optional | variable · declarar · obligatoria · opcional |
| Sensitive | Confidencial (never "sensible", a false friend) |
| sample value · default | valor de ejemplo · predeterminado |
| test data (set) | (conjunto de) datos de prueba |
| render · test renders | renderizar · renderizados de prueba |
| Checks · Ready to publish | Comprobaciones · Lista para publicar |
| breaking change | cambio incompatible |
| Subject · Preheader | Asunto · Preencabezado |
| canvas | lienzo |
| Visual · Text (views) | Visual · Texto |
| plain text | texto sin formato |
| call (an API call) | llamada |
| Essential · Optional (categories) | Esenciales · Opcionales |
| autosaved 09:41 · conflict | guardado automáticamente a las 9:41 · conflicto |
| Import templates · Needs a manual fix | Importar plantillas · Requiere corrección manual |

"payload", "Liquid", "MJML", "Mustachio", "Handlebars", "cURL" and "JSON"
stay as they are.

## Composer

| English | Español |
|---|---|
| A/B test · variant | prueba A/B · variante |
| block · Blocks panel | bloque · Bloques |
| code view | vista de código |
| column · 2 columns | columna · 2 columnas |
| compile · compiled HTML | compilar · HTML compilado |
| composer | editor ("¿Salir del editor?") |
| dark mode · light | modo oscuro · claro |
| footer · header (of an email) | pie · cabecera |
| Inspector | Propiedades |
| layer · layer tree | capa · árbol de capas |
| padding · border radius | relleno · radio del borde |
| personalization token · Liquid tag | etiqueta de personalización · etiqueta Liquid |
| raw HTML | HTML puro |
| sample · winner (A/B) | muestra · ganadora ("la variante ganadora") |
| section (of an email) · saved section | sección · sección guardada |
| Send Readiness | revisión previa (the checks before a send; "runs again": "vuelve a ejecutarse") |
| spam heuristic | indicio de spam |
| bare URL · link text · says nothing (a link's text) | URL desnuda · texto del enlace · no dice nada |
| broken link · shortened link | enlace roto · enlace acortado (a URL shortener is the revisión previa's "acortador de URL") |
| inline CSS | CSS en línea (Gmail's clipping limit is the revisión previa's "límite de recorte") |
| contact field · personalization token's fallback | campo del contacto · valor alternativo |
| plain-text version · postal address | versión de texto sin formato · dirección postal |
| Line 3: (a compile or layout problem) · doesn't compile | Línea 3: · no compila |
| title (of an article card, a post) | título ("asunto" is the email's subject) |
| Email preferences link · View in browser link | Enlace a las preferencias de correo · Enlace para ver en el navegador |
| starter (template) | plantilla inicial |
| starters: Blank · Weekly digest · Product update · Plain text · Announcement · Product grid · Blog digest | En blanco · Resumen semanal · Novedades del producto · Texto sin formato · Anuncio · Cuadrícula de productos · Resumen del blog |

MJML tags (mj-section, mj-body) stay as they are. Section names in the
server's sentences use the composer's words: cabecera, pie, hero,
2 columnas, texto, imagen, botón. A block's sample content is written for
the email's readers, in the app language: the first-name fallback "there"
is "tú" ("Hola, tú,"), one gender-neutral word (to confirm in the native
review).

## Automations

| English | Español |
|---|---|
| A/B split · weight | División A/B · peso |
| automation (one, or the section) | automatización ("Nueva automatización") |
| Branch · Yes · No | Bifurcación · Sí · No |
| Delay (step) · Wait 10 minutes | Espera · Esperar 10 minutos |
| double opt-in · double opt-in confirmed only | doble confirmación · solo con la doble confirmación completada |
| edge (a connection on the canvas) | conexión |
| exit · goal · goal rate | salida · objetivo · tasa de objetivo cumplido |
| fail over · failover | conmutar por error · conmutación por error |
| Filter (step) | Filtro |
| flow · in the flow · leave the flow | recorrido · en el recorrido · salir del recorrido ("recorrido de automatización" where the kind needs saying; "flujo" is only a stream: "flujo Automatizaciones") |
| List action | Acción de lista |
| method (HTTP) | método |
| override (a route) | anular; "route override" on a step is "ruta propia" |
| re-entry | reingreso |
| retire · retiring (a version) | retirar · retirándose |
| send window | franja de envío |
| signing secret · write-only | secreto de firma · solo escritura |
| stats · Live stats | estadísticas · Estadísticas en tiempo real |
| step · trigger · action · condition | paso · disparador · acción · condición |
| web push | notificación push web |
| path (of a branch or split) · connection | rama · conexión |
| starters: Welcome series · Re-engagement · Onboarding · Abandoned cart | Serie de bienvenida · Reactivación · Primeros pasos · Carrito abandonado |
| Send Readiness content checks | comprobaciones de contenido de la revisión previa |
| webhook delivery log · failed calls | registro de entregas del webhook · llamadas fallidas |
| route label: SES, failover Postmark | SES, respaldo Postmark |
| Removed provider | Proveedor eliminado |
| webinar | seminario web |
| version summary: created from a blank canvas · created from the X template · copied from X · first published version | creada desde un lienzo en blanco · creada desde la plantilla X · copiada de X · primera versión publicada |

## Approvals

| English | Español |
|---|---|
| after exclusions | tras las exclusiones |
| approval inbox | bandeja de aprobaciones |
| approval request · approver · author | solicitud de aprobación · aprobador · autor (labels; in a sentence "quien aprueba", "quien la creó") |
| Approve · Reject · Request changes | Aprobar · Rechazar · Solicitar cambios |
| Bypassed · emergency bypass | Omitida · omisión de emergencia |
| comment · Discussion · evidence | comentario · Conversación · evidencia |
| Contacts in flight | Contactos en el recorrido |
| Invalidated · Approval invalidated | Invalidada · Aprobación invalidada |
| median wait | espera mediana (a median, not an average: never "media") |
| Pending · Rejected | Pendiente · Rechazada |
| policy · rule · Rules matched | política · regla · Reglas que se cumplen |
| Recheck | Volver a comprobar |
| snapshot | instantánea |
| after approval (a change) | tras la aprobación ("los destinatarios cambiaron un +14 % tras la aprobación"; said once) |
| rules: audience ≥ 10,000 · warming-up provider · new provider · author lacks send · list requires approval · client sign-off | audiencia ≥ 10.000 · proveedor en calentamiento · proveedor nuevo · el autor no puede enviar · la lista requiere aprobación · visto bueno del cliente |
| automation publish · post publish · page publish · template publish (rules) | publicación de automatización · publicación de artículo · publicación de página · publicación de plantilla |
| a send held · suppressed | envío retenido · suprimido |
| online · active 2 h ago (an approver) | en línea · activo hace 2 h |
| Best time within 24 h of | Mejor hora en las 24 h siguientes a |
| Only people can… (not agents or keys) | Solo las personas pueden… |

## Content languages

| English | Español |
|---|---|
| language version · Source | versión de idioma · Original |
| Default language · Fallback language | Idioma predeterminado · Idioma alternativo |
| Up to date · Outdated · Draft · Missing · Held | Actualizada · Desactualizada · Borrador · Falta · Retenida |
| Mark as translated · Mark as up to date | Marcar como traducida · Marcar como actualizada |
| readers (of a language) | lectores ("los lectores en {language}"; the role Lector is capitalised) |
| page copy · built-in labels | textos de la página · etiquetas integradas |
| Results per language · click map | Resultados por idioma · mapa de clics |
| no language · other languages (results) | sin idioma · otros idiomas |
| compare view's blocks: Header · Footer · Raw HTML · Social links · Two cards · 3 columns · Hero image · Button · Intro · The email | Cabecera · Pie · HTML puro · Enlaces a redes sociales · Dos tarjetas · 3 columnas · Imagen hero · Botón · Introducción · El correo completo |
| Headline (a block) · no alt text | Titular ("Encabezado" is a heading) · sin texto alternativo |
| version labels: copied from · layout of · applied from · block copied from · changed by an agent · edited in the flow editor · translated from | copiada de · diseño de · aplicada desde · bloque copiado de · modificada por un agente · editada en el editor del recorrido · traducida de |
| a language's changes after approval: added · marked translated · back to draft · sent again · edited | añadida · marcada como traducida · de vuelta a borrador · reenviada · editada |
| copy (a form's or page's own text) · Copy in | textos · Textos en ("Textos en {language}") |
| Live · Incomplete · Not set up (copy) | En uso · Incompletos · Sin configurar |
| string (text to write) | cadena |
| visitors | visitantes ("Visitantes que leen en {language}") |

A content language's name comes from the browser's or CLDR's names in
the app language, which Spanish writes in lower case ("inglés", "español de
México"); the app capitalises it where it stands alone ("Inglés"). The
app's own picker names each language in itself.

## Conditional content

| English | Español |
|---|---|
| combination (of blocks) · version | combinación · versión |
| conditional block | bloque condicional |
| Gmail clips · clipping size | Gmail recorta · tamaño de recorte |
| includes · excludes | incluye · excluye |
| is · isn't · contains · is set · is empty | es · no es · contiene · tiene valor · está vacío (as automations' conditions) |
| is at least · is at most · is between … and … | es como mínimo · es como máximo · está entre … y … |
| is greater than · is less than · is before · is after | es mayor que · es menor que · es anterior a · es posterior a |
| is in · is not in | está en · no está en |
| Match ALL / ANY of these | Cumplir TODAS / ALGUNA de estas condiciones (the uppercase words stay uppercase) |
| operator | operador |
| Preview as · Show why | Vista previa como · Mostrar el motivo |
| Visibility (who sees a block) | Visibilidad |

## Accessibility

| English | Español |
|---|---|
| Accessibility | Accesibilidad |
| issue · serious · moderate | problema · grave · moderado |
| alt text · decorative image | texto alternativo · imagen decorativa |
| contrast · Passes AA · Fails AA | contraste · Cumple AA · No cumple AA |
| Heading 1 · Paragraph · Text role | Encabezado 1 · Párrafo · Rol del texto |
| layout table | tabla de diseño |
| screen reader | lector de pantalla |
| Warn · Block (the policy) | Avisar · Bloquear |
| skipped heading level · card titles | nivel de encabezado omitido · títulos de tarjeta |

"Heading" is "Encabezado", never "Título" (a post's or a page's title) nor
"Asunto" (an email's subject). Content languages in a sentence use their
standard Spanish names, in lower case: "alemán", "español".

## Earn

Payments, paid subscriptions, offers, gifts, sponsors and revenue. A
"paid member" is a **suscriptor de pago**, never a "miembro", which is a
team member. A reader's Stripe subscription is a **suscripción de pago**:
ending it is "cancelar", never "darse de baja", which is unsubscribing from
the emails.

| English | Español |
|---|---|
| access (paid) · With access | acceso · Con acceso |
| access rank · Rank 2 | rango de acceso · Rango 2 |
| Access rules | Reglas de acceso |
| Active · Trialing · Past due · Cancels soon · Canceled · Ended (paid status) | Activa · En periodo de prueba (short: En prueba) · Vencida · Se cancela pronto · Cancelada · Finalizada |
| archive · Archived (a tier) | archivar · Archivado |
| attribution · attribution window · credited to (an email) | atribución · periodo de atribución · atribuido a |
| benefit (of a tier) | ventaja |
| billing email · billing period · Manage billing | correo de facturación · periodo de facturación · Gestionar facturación |
| book · booking · Booked (a sponsor in an issue) | reservar · reserva · Reservado |
| buyer · recipient (of a gift) | comprador · destinatario |
| cancel page · gift page · upgrade page · welcome page | página de cancelación · página de regalo · página de mejora · página de bienvenida |
| charge (a card) | cobrar; "nothing is charged" is "no se cobra nada" |
| checkout · test checkout | pago · pago de prueba ("Stripe Checkout" is a name) |
| churn · churn rate | cancelaciones · tasa de cancelación |
| click token | token de clic |
| Complimentary · complimentary access | De cortesía · acceso de cortesía |
| conversion · conversion API | conversión · API de conversiones |
| coupon · promotion code · code | cupón · código promocional · código |
| creative (a sponsor's) | creatividad |
| customer portal | portal del cliente |
| deliver · delivery date (a gift) | entregar · fecha de entrega |
| discount · 20% off | descuento · 20 % de descuento |
| dispute (a card payment's) | disputa |
| Earn's tabs: Overview · Plans & offers · Paid members · Gifts · Sponsors · Products · Settings | Resumen · Planes y ofertas · Suscriptores de pago · Regalos · Patrocinadores · Productos · Configuración |
| expansion · contraction · reactivated (MRR movement) | expansión · contracción · reactivados |
| Free (the card) · free subscriber | Gratis · suscriptor gratuito |
| free-to-paid | de gratuito a de pago; "free-to-paid conversion" is "conversión a suscripción de pago" |
| gift · gift option | regalo · opción de regalo |
| headline (an upgrade page's, a creative's) | titular; a subject is "asunto", a heading "encabezado" |
| highlight (a tier) · Most popular | destacar · Más popular |
| hold · on hold · release (an issue) | reserva provisional · con reserva provisional · liberar |
| in person (a gift handed over) | en persona |
| issue (of a newsletter: #143) | edición (edición #143); an accessibility issue is "problema" |
| legacy price | precio anterior |
| lifetime value | valor del ciclo de vida |
| live · test (a key, a mode, payments) | real · de prueba ("modo real", "clave de prueba"); "no real payments" is "sin pagos reales" |
| live (a page, an offer or a template version that's running) | publicado; an offer is "activa" |
| map (a Stripe price to a tier) | asignar |
| member email · membership | correo para suscriptores · suscripción |
| merchant of record | comerciante registrado (merchant of record) |
| monthly · yearly · Monthly · Yearly (the choice) · /month | mensual · anual · Mensual · Anual · /mes |
| MRR · ARR · MRR movement | MRR · ARR · evolución del MRR |
| Needs you (Earn's overview) | Requiere tu atención |
| normal price (without an offer) | precio normal |
| offer · signup · retention · referral reward | oferta · oferta de suscripción · oferta de retención · recompensa por referidos |
| Overdue · Unpaid · Paid (a sponsor's payment) | Vencido · Sin pagar · Pagado |
| paid member | suscriptor de pago |
| paid status · paid-only | estado de pago · solo de pago |
| Paid subscriptions (what Earn sells) | Suscripciones de pago |
| paid-through date | pagado hasta |
| payment provider | proveedor de pagos |
| payment terms (a sponsor's) | condiciones de pago |
| payouts | transferencias (to the bank, as Stripe says it) |
| plan · Plans & offers | plan · Planes y ofertas |
| Public · Hidden (a tier's visibility) | Público · Oculto |
| receipt · invoice | recibo · factura |
| record (a sponsor's payment, by hand) | registrar |
| redeem · redeem link (a gift) | canjear · enlace de canje |
| redeem · redemption (an offer code) | usar · uso |
| refund | reembolso |
| renew · renewal | renovar · renovación |
| restricted key · secret key | clave restringida · clave secreta |
| results so far · final (a sponsor report's) | resultados provisionales · definitivos |
| revenue · income | ingresos · ganancias |
| roll (a Stripe key) | rotar |
| seller ("you're the seller") | vendedor ("tú eres quien vende") |
| signing secret (a webhook's) · endpoint | secreto de firma (as Automations has it) · endpoint |
| Since · Member since (a paid member's) | Desde · Suscripción desde |
| slot · placement | espacio · ubicación |
| sponsor · sponsorship · Sponsored (the block's label) | patrocinador · patrocinio · Patrocinado |
| sponsor report | informe de patrocinio |
| subscription (a reader's, billed by Stripe) | suscripción de pago |
| Terms of sale · Support address | Condiciones de venta · Dirección de soporte |
| tier · paid tier | nivel · nivel de pago |
| timeline (a paid member's) | cronología |
| trial · free trial | periodo de prueba · prueba gratuita ("de prueba" alone is test mode) |
| upgrade · Upgrade block | mejora (to upgrade: "mejorar") · bloque Mejora |
| webhook silent · gap check | webhook sin eventos · comprobación de eventos perdidos |
| write-only · sealed (a key, a secret) | solo escritura · sellado |

Some English messages choose a word for English grammar only: an article
(`{article, select, an {an} other {a}} {tier}`) or a possessive
(`{ends, select, s {{sponsor}’} other {{sponsor}’s}}`). Spanish keeps the
argument and puts the name alone in its `other` branch, with the Spanish
words around the select: "la creatividad de `{ends, select, other
{{sponsor}}}`", "el nivel `{article, select, other {{tier}}}`". Where the
English counts with `#`, keep `{count, plural, one {# …} other {# …}}` rather
than a bare `{count}`, so the number is grouped by the region format (48.213).

## Website and pages

The publication website, its posts, the page builder and the public pages'
own words. Words a reader sees follow the built-in labels of the public
pages (`internal/pagelabels`): "boletín", "edición", "Archivo",
"Preferencias de correo".

| English | Español | Note |
|---|---|---|
| About (page, template) | Acerca de | |
| Access (Website › Access) | Acceso | |
| archive · Archive (the page, the link) | archivo · Archivo | the archive as a whole; "página de archivo" when the page itself is meant |
| band (subscribe band) | franja | a full-width strip of the home page |
| block · layers | bloque · capas | the page builder's |
| built-in page | página integrada | like "etiquetas integradas" |
| byline · By {name} | firma · Por {name} | |
| canonical URL | URL canónica | as Google's tools say it |
| card (paywall card, a post's card) | tarjeta | "etiqueta" is a tag |
| certificate | certificado | |
| countdown | temporizador | never "cuenta atrás" or "cuenta regresiva" |
| cover (image) | imagen de portada | |
| excerpt | extracto | |
| FAQ (block) | Preguntas frecuentes | |
| featured · Feature on home | destacado · Destacar en la página de inicio | |
| feed · RSS feed | feed · feed RSS | never "fuente", which is a font |
| free preview · Free preview ends here (the post's block) | vista previa gratuita · Aquí termina la vista previa gratuita | |
| Hero (block) | Hero | the block's name, as in the composer |
| home page | página de inicio | |
| issue (of a newsletter) | edición | "48 ediciones", "Leer la última edición"; never "número", which is a phone number or a count |
| lead magnet | regalo de suscripción | a free file for confirmed subscribers |
| live (the website) · live (a page, a form, a version) | en línea · publicado | as Earn and the signup forms have it |
| member · free member · paid member (of the site) · membership | suscriptor · suscriptor gratuito · suscriptor de pago · suscripción | a reader who signs in, as Earn has it; "miembro" is a team member |
| meter | límite de lectura gratuita | "Límite de lectura" as a heading |
| navigation · header · footer | navegación · cabecera · pie de página | |
| page · Pages | página · Páginas | |
| page builder | constructor de páginas | |
| Pay what you want | Paga lo que quieras | the pricing model's name |
| paywall | muro de pago | |
| post · web-only post | artículo · artículo solo web | "publicación" is the act of publishing |
| publish · unpublish · unschedule | publicar · anular la publicación · cancelar la programación | Unpublished: "Publicación anulada" |
| redirect | redirección | |
| referrer (where views come from) | origen | a person who refers is a "referidor" |
| registration (event) | inscripción | "suscripción" is a signup |
| sales page | página de venta | |
| signups · upgrades · views · visitors | suscripciones · mejoras · vistas · visitantes | |
| site · website | sitio · sitio web | |
| sitemap | mapa del sitio | |
| slug | slug | the page's or tag's part of the address |
| social image | imagen para redes sociales | the image a shared link shows |
| store product (block) | producto de la tienda | |
| tab (a panel's: the Web tab) | pestaña | "etiqueta" is a tag |
| tag | etiqueta | |
| tagline | eslogan | |
| testimonial | testimonio | |
| theme | tema | |
| tier · paid tier | nivel · nivel de pago | as Earn has it; a tier's name, such as "Insider", stays as it is |
| title (a post's, a page's, the site's) · SEO title | título · título SEO | "asunto" is an email's subject ("asunto del correo" beside a title) |
| visibility · Public · Members · Paid | visibilidad · Público · Suscriptores · De pago | |
| visitor | visitante | someone who isn't signed in |
| webinar | seminario web | |

## Team

| English | Español |
|---|---|
| Team | Equipo |
| Invite member | Invitar a un miembro (short: Invitar) |
| Members | Miembros |
| Everyone with access to Acme Newsletter | Todas las personas con acceso a Acme Newsletter |
| Search name or email | Buscar por nombre o correo |
| Any role · Any status · Any 2FA | Todos los roles · Todos los estados · 2FA: todos |
| Role · scope | Rol · ámbito |
| Sign-in | Acceso (the column; "inicio de sesión" in a sentence) |
| Last active | Última actividad |
| Manage · Reactivate | Gestionar · Reactivar |
| Password · passkey | Contraseña · llave de acceso |
| just now · yesterday | ahora · ayer |
| 2 selected · Bulk: | 2 seleccionados · En bloque: |
| Change role · Require 2FA now · Deactivate | Cambiar rol · Exigir 2FA ahora · Desactivar |
| Required · off, 3 days of grace left | Obligatoria · desactivada, quedan 3 días de gracia |
| deactivated Sep 30, 2026 | desactivado el 30 sept 2026 |
| Pending invitations · 1 | Invitaciones pendientes · 1 |
| Links last 7 days by default | Los enlaces caducan a los 7 días por defecto |
| Invitee · Sent · Expires | Persona invitada · Enviada · Caduca |
| invited by Linh Nguyen | invitación de Linh Nguyen |
| Beta Program list only | solo la lista Beta Program |
| in 5 days | dentro de 5 días |
| Resend · Copy link · Revoke | Reenviar · Copiar enlace · Revocar |

## Language and region

| English | Español |
|---|---|
| Language & region | Idioma y región |
| App language | Idioma de la aplicación |
| Instance default (English) | Predeterminado de la instancia (English) |
| Region format | Formato regional |
| Follows the app language | Según el idioma de la aplicación |
| Time · 24-hour · 12-hour | Hora · 24 horas · 12 horas |
| Week starts on · Monday · Sunday | La semana empieza el · Lunes · Domingo (choices; "el lunes" in a sentence) |
| From the region format | Según el formato regional |
| Preview | Vista previa |
| Help translate ReputeCast | Ayudar a traducir ReputeCast |
| Save · Undo | Guardar · Deshacer |
| Profile | Perfil |
| timezone | zona horaria |
| quiet hours | horas de silencio |
| Settings › General | Configuración › General |
| Invitation language | Idioma de la invitación |
| Default language · Default region format | Idioma predeterminado · Formato regional predeterminado |

Languages are always named in their own language, in every interface
language: English, Español, Tiếng Việt, Deutsch (Deutschland). "ReputeCast
standard" is a name and stays in English.

## Account and security

| English | Español |
|---|---|
| Full name | Nombre completo |
| Upload photo · Remove | Subir foto · Quitar |
| Change email · verification link | Cambiar correo · enlace de verificación |
| Current password · New password | Contraseña actual · Contraseña nueva |
| security key · platform passkey | llave de seguridad · llave de acceso del dispositivo |
| Rename · Keep it | Cambiar nombre · Conservar |
| Time-based codes (TOTP) | Códigos temporales (TOTP) |
| Set up · Replace app | Configurar · Cambiar de aplicación |
| Regenerate codes | Regenerar códigos |
| 7 of 10 remain | Quedan 7 de 10 |
| Connected sign-ins | Cuentas vinculadas |
| Link · Unlink · Managed | Vincular · Desvincular · Gestionada |
| session · Active sessions · This session | sesión · Sesiones activas · Esta sesión |
| Sign out all other sessions | Cerrar las demás sesiones |
| Device · Browser · IP · location (approx.) | Dispositivo · Navegador · IP · ubicación (aprox.) |
| My workspaces · Leave workspace | Mis espacios de trabajo · Salir del espacio de trabajo |
| In-app · All events | En la aplicación · Todos los eventos |
| MCP agent | agente MCP |

## Instance admin, keys, webhooks and the audit log

| English | Español | Note |
|---|---|---|
| actor (audit) · Any actor type | actor · Todos los tipos de actor | who did it: a member, a key, an agent or the system |
| agent capability: Allow · Deny · Approve · Pause only (Settings › MCP) | Permitir · Denegar · Aprobar · Solo pausar | the matrix's cells; “Aprobar” as the caption quotes it |
| All retained (date range) | Todo el periodo conservado | |
| app registration (Entra) · app integration (Okta) | registro de aplicación · integración de aplicación | “the Entra app registration”: “el registro de aplicación de Entra” |
| batch (of streamed entries) | lote | as in "endpoint por lotes" |
| break-glass Owner | Propietario con cuenta de emergencia | never "Propietario de emergencia" |
| break-glass password · break-glass passkey | contraseña de emergencia · llave de acceso de emergencia | what a break-glass Owner keeps while SSO is enforced |
| claim · Client ID · client secret · Scopes (OIDC) | claim · Client ID · client secret · Scopes | named as the identity provider's console names them |
| danger zone | Zona de peligro | |
| delivery (webhook) · Delivery log | entrega · Registro de entregas | |
| Development build | Compilación de desarrollo | |
| digest (email) · Hourly digest · Daily digest | correo de resumen · Resumen cada hora · Resumen diario | notification preferences |
| discovery · issuer · Issuer URL (OIDC) | discovery · emisor · URL del emisor (issuer) | |
| Effective permissions | Permisos efectivos | a member's page; the column is "Efectivos" |
| endpoint · payload | endpoint · payload | not translated |
| enforce (SSO, two-factor) · enforcement | exigir · obligatoriedad | "Enforce SSO for acme.io": "Exigir SSO para acme.io" |
| entry (audit) · Entry details | entrada · Detalles de la entrada | |
| Failing · Paused (an endpoint) | Con errores · En pausa | "Healthy" is "Operativo" |
| Force sign-out | Forzar el cierre de sesión | |
| Generic OIDC (a preset) | OIDC genérico | the IdP's short name in sentences is “IdP” |
| Given (approvals a member gave) | Emitidas | |
| group mapping · group object ID | asignación de grupos · ID de objeto del grupo | |
| hourly cleanup | limpieza cada hora | |
| identity provider | proveedor de identidad | "IdP" stays |
| immutable (audit entries) | inmutable | |
| instance-level (entries) · Instance level | de la instancia · Nivel de instancia | |
| JIT account creation | creación automática de cuentas (JIT) | |
| just-in-time account · JIT default role | cuenta JIT · rol predeterminado de JIT | |
| Locked (sign-in) · Unlock sign-in | Bloqueado · Desbloquear el acceso | a lockout is "bloqueo de acceso" |
| Make instance admin · Remove instance admin | Dar la administración de la instancia · Quitar la administración de la instancia | |
| overage link (Entra's groups) | enlace de overage | Entra's own word |
| preset (an SSO provider's, a key's) | configuración predefinida | |
| proposal (an agent asked a person) · proposal recorded | propuesta · propuesta registrada | “Ops asked to pause …”: “Ops propone pausar …” |
| provider status: degraded · down | degradado · caído | as Channels and SMS' Degradado · Caído |
| re-authentication | reautenticación | |
| Redirect URI | URI de redirección | |
| reference ID · request ID | ID de referencia · ID de solicitud | |
| result: Allowed · Denied · Error | Permitido · Denegado · Error | |
| retention | conservación | |
| revoke · rotate · replace (a key or secret) | revocar · rotar · reemplazar | |
| secret · signing secret | secreto · secreto de firma | "secrets" in general: "datos secretos" |
| Send approvals (Settings › General) | Aprobación antes del envío | |
| Sending (notification category) | Envíos | so it isn't read as the status "Enviando" |
| SIEM or warehouse | SIEM o almacén de datos | |
| single sign-on | inicio de sesión único | "SSO" stays |
| stream (audit entries, noun) · stream (verb) · streaming endpoint | flujo · transmitir · endpoint de transmisión | the System stream is "flujo Sistema" |
| test sign-in | inicio de sesión de prueba | |
| Then allowed | Luego permitido | a sign-in that was rejected, then allowed |
| token (an agent's) | token | "agent token": "token del agente" |
| tool (MCP) | herramienta | tool names (get_campaign_report) stay as they are |
| Trusted from Okta (two-factor) | Avalada por Okta | the IdP enforces the second factor |
| typed confirmation | confirmación escribiendo el nombre | |
| Up to date · Update available (a release) | Actualizado · Actualización disponible | language versions keep "Actualizada" |
| user (an account on the instance) | usuario | a workspace's people are "miembros" |
| Workspace ID | ID del espacio de trabajo | |
| workspace switcher | selector de espacio de trabajo | |

## Channels and SMS

| English | Español |
|---|---|
| 10DLC · local · mobile · toll-free (number kinds) | 10DLC · local · móvil · gratuito ("número móvil", "número gratuito") |
| adapter · manifest · contract test suite | adaptador · manifest · conjunto de pruebas de contrato (developer words) |
| auto-pause · Auto-paused | pausa automática · En pausa automática |
| auto-pause reasons: Too many texts failed · Carriers filtered too many texts · Too many people opted out | Demasiados mensajes fallidos · Los operadores filtraron demasiados mensajes · Demasiadas bajas |
| brand (in a 10DLC registration) · brand prefix | marca · prefijo de marca |
| budget · spend · price per segment | presupuesto · gasto · precio por fragmento |
| budget used up · budget at 80% | presupuesto agotado · presupuesto al 80 % ("Presupuesto de SMS de octubre agotado") |
| carrier | operador |
| channel · Email · SMS · Push · WhatsApp | canal · Correo · SMS · Push · WhatsApp |
| consent · SMS consent · consent text · evidence | consentimiento · consentimiento para SMS · texto de consentimiento · evidencia |
| credentials · Live · Test credentials | credenciales · Reales · Credenciales de prueba |
| credentials rejected · isn't answering · is back (a provider) | credenciales rechazadas · no responde · vuelve a funcionar |
| delivery receipt | confirmación de entrega (what a carrier reports back; "Last receipt" is "Última confirmación") |
| fallback (of a route) | respaldo |
| filtered by carriers | filtrado por los operadores |
| held · skipped | retenido · omitido |
| inbound (text) · reply | entrante · respuesta |
| keyword | palabra clave (STOP, START, HELP and YES stay in English: people text them) |
| landline · not in service · invalid number · unreachable | fijo · fuera de servicio · número no válido · inaccesible |
| link preview | vista previa del enlace (a messaging app's machine click, never a person) |
| local time · "Berlin local" (a reply's time) | hora local · "hora de Berlín" |
| no answer · invalid credentials (what a provider answered a call) | sin respuesta · credenciales no válidas (the HTTP status before them stays: "401 Unauthorized · credenciales no válidas") |
| opt in · opt-in confirmation | darse de alta · confirmación de alta |
| opt out · opt-outs · opt-out text · opt-out reply | darse de baja · bajas · texto de baja · respuesta de baja |
| opt-out sync · Two-way · Read only | sincronización de bajas · Bidireccional · Solo lectura |
| opt-out list (a provider's own) | lista de bajas ("Lista de bajas de Twilio") |
| outside quiet hours (a text waits or is skipped) | fuera de la franja de envío (the window texts may go out in, not "fuera de las horas de silencio") |
| provider status: Healthy · Degraded · Down | Operativo · Degradado · Caído |
| quiet hours · sending window | horas de silencio · franja de envío |
| raw event · normalized | evento sin procesar · normalizado |
| receipts silent (none came back while texts went out) | sin confirmaciones de entrega |
| registration · registered | registro · registrado |
| route · Routing & rules · route tester | ruta · Enrutamiento y reglas · probador de rutas |
| segment (of a text) | fragmento (the billed part of 160 GSM-7 or 70 UCS-2 characters; never "segmento", an audience segment) |
| sender ID · alphanumeric sender ID | ID de remitente · ID de remitente alfanumérico |
| short-link domain · public shortener | dominio de enlaces cortos · acortador público |
| signature · signing secret | firma · secreto de firma |
| status callback (Twilio) | callback de estado |
| suppression list (SMS) | lista de supresión de SMS |
| test text · test number | SMS de prueba · número de prueba |
| text (a message) · to text | mensaje ("SMS" where it must be explicit) · enviar un SMS |
| Text message terms | Condiciones de los SMS |
| throughput · 14 texts a second | rendimiento · 14 mensajes por segundo |
| unschedule · a send (of a campaign) | cancelar la programación · envío ("Este envío ha terminado") |

"GSM-7", "UCS-2", "Unicode", "VoIP" and "emoji" stay as they are. A
phone number, a sender ID and a country code are data: never translated.

## Web push and WhatsApp

| English | Español |
|---|---|
| access token · permanent access token · system user | token de acceso · token de acceso permanente · usuario del sistema |
| Allow · Not now · Granted (a prompt's numbers) | Permitir · Ahora no · Concedido |
| anonymous · linked (a subscription) | anónima · vinculada (a un contacto) |
| app secret · verify token · callback URL · phone number ID | clave secreta de la app · token de verificación · URL de devolución de llamada · identificador del número de teléfono (Meta's own Spanish) |
| banner (a notification's style on a desktop) | banner |
| Bell button · Dialog · Slide-in bar (prompt styles) | Botón de campana · Cuadro de diálogo · Barra deslizante |
| business account · display name | cuenta de empresa · nombre visible ("WhatsApp Business Account" is a name) |
| category (of a template): Marketing · Utility · Authentication | categoría: Marketing · Utilidad · Autenticación |
| conversation · customer service window | conversación · ventana de atención al cliente (short: "ventana de 24 horas", never the quiet-hours "franja de envío") |
| expiry · Expires after | caducidad · Caduca después de |
| free-form reply · quick reply | respuesta libre · respuesta rápida |
| frequency cap | límite de frecuencia |
| header · body · footer · buttons (of a template) | encabezado · cuerpo · pie de página · botones (Meta's own Spanish) |
| Home Screen (iPhone, iPad) | pantalla de inicio |
| icon · badge · image (of a notification) | icono · insignia · imagen |
| In review · In appeal · submit (a template) | En revisión · En apelación · enviar a revisión |
| messaging limit · rolling 24 hours | límite de mensajes · últimas 24 horas |
| notification (one push message) | notificación |
| permission prompt (the browser's own) | solicitud de permiso (del navegador) |
| prompt · prompt designer | aviso · diseñador de avisos (an AI prompt is "prompt") |
| publication site | sitio de la publicación |
| push (in a sentence) · Push (the channel) | notificación push · Push |
| push service (the browser maker's) | servicio push |
| quality rating · High · Medium · Low | calificación de calidad · Alta · Media · Baja |
| rate card · Edit rates | tarifas · Editar tarifas |
| read receipts | confirmaciones de lectura (WhatsApp's own Spanish) |
| Rotate keys · public key · private key | Rotar claves · clave pública · clave privada |
| Sealed | Sellado |
| Shown (a count) · Shown (an event) · Gone · removed | Mostradas · Mostrada · Caducada · eliminada |
| site (where readers subscribe) · scope (of a service worker) | sitio · ámbito |
| snippet | fragmento de código (never "fragmento" alone, an SMS segment) |
| subscriber · subscription | suscriptor · suscripción |
| Sync now | Sincronizar ahora |
| Tapped (a WhatsApp button) | Tocado |
| template (WhatsApp) | plantilla ("plantilla de mensaje" where it must be explicit) |
| token (that a link carries to know the reader) | token de identificación |

"service worker", "VAPID", "Meta", "Graph API" and Meta's own screens and
paths (WhatsApp Manager, Meta for Developers › … › Configuration) stay as
they are.

## Surveys and polls

| English | Español |
|---|---|
| 0–10 scale · Emoji scale · faces | escala de 0 a 10 · escala de emojis · caras |
| answer · answer page | respuesta · página de respuesta |
| Breakdown · Break down by | Desglose · Desglosar por |
| completion rate · drop-off · started · completed | tasa de finalización · abandono · iniciadas · completadas |
| Embedded · Popup · Hosted page (form kinds) | Insertado · Emergente · Página en ReputeCast |
| link scanner · security gateway · prefetching proxy | escáner de enlaces · pasarela de seguridad · proxy de precarga (machine visits, never a person) |
| Not likely · Very likely (an NPS scale's ends) | Nada probable · Muy probable |
| Open · Closed (a survey or a poll) | Abierta · Cerrada |
| Owner (the forms list's column) | Responsable (not the role "Propietario") |
| poll (the block) · a poll | Votación · votación |
| promoters · passives · detractors | promotores · pasivos · detractores ("NPS" and "Net Promoter Score" stay) |
| question · respondent · response | pregunta · persona encuestada · respuesta |
| submission | envío |
| survey | encuesta |
| Top answer | Respuesta más elegida |
| vote · voter | voto · votante |

## Notifications

| English | Español |
|---|---|
| digest (recurring) · issue (of a digest) | boletín periódico · edición |
| sending summary (by email, M57) · weekly · monthly | informe de envíos (por correo) · informe semanal · mensual de envíos (a member's own email of the workspace's numbers; never "resumen", which is the notification digest) |
| event type (a row of Notification preferences) | tipo de evento |
| Everyone with campaigns:pause | Todas las personas con campaigns:pause |
| paid tier (Earn) | nivel de pago |
| placement test · seed mailbox | prueba de ubicación · buzón de prueba |
| Sending (the notification center's tab) | Envíos |
| verification credits | créditos de verificación |

## Agency accounts

| English | Español |
|---|---|
| Account manager · Agency manager (staff levels) | Responsable de cuentas · Responsable de agencia |
| agency · Agency (the area) · agency mode | agencia · Agencia · modo agencia |
| assign · assignment | asignar · asignación |
| client · client workspace | cliente · espacio de trabajo del cliente |
| client people (a client's own people) | personas del cliente (never agency staff) |
| Client approver · Client viewer (roles) | Aprobador del cliente · Lector del cliente |
| client sign-off | visto bueno del cliente |
| copy (of a library item) · behind | copia · desactualizada (the library has a newer version) |
| flat (rate) · monthly fee | fija · cuota mensual |
| incident · Needs attention | incidente · Requiere atención |
| level (of staff) | nivel |
| library · shared library · library item | biblioteca · biblioteca compartida · elemento de la biblioteca |
| provider cost · plan | costo del proveedor · plan |
| rebill · rebill rate · to rebill | refacturar · tarifa de refacturación · por refacturar |
| saved section | sección guardada |
| sender health · good · fair · poor | salud del remitente · buena · aceptable · deficiente |
| setup steps · Setting up | pasos de configuración · En configuración |
| staff | personal |
| usage | uso |
| workspace template · apply (a template) | plantilla de espacio de trabajo · aplicar |

## Grow, Growth and scoring

| English | Español | Note |
|---|---|---|
| awaiting confirmation | pendiente de confirmación | a referral whose subscriber hasn't confirmed |
| band (score) · Cold · Warm · Hot | tramo (tramo de puntuación) · Frío · Templado · Caliente | the names are fixed, like the English |
| burst · shared IP · numbered addresses | pico · IP compartida · direcciones numeradas | the signals that hold a referral |
| cap (a rule's) | límite | "6 a week": "6 por semana" |
| claim · claim link | canjear · enlace de canje | a coupon reward |
| cohort retention | retención por cohortes | Audience › Growth |
| counted · not counted | contado · no contado | "Count" and "Don't count" in Review: "Contar", "No contar" |
| coupon (Stripe) | cupón | |
| decay | decaimiento | |
| double opt-in | doble confirmación | chosen over "doble opt-in": it says what happens, and subscribers read it too; two-factor is "verificación en dos pasos" |
| engagement tier · Active · Cooling · At risk · Sunset | grupo de interacción · Activo · Enfriándose · En riesgo · En retirada | List health's tiers, as Scoring names them; the sunset policy is "política de retirada" |
| field condition | condición de campo | |
| fulfill · fulfilled · To fulfill | entregar · entregada · Por entregar | a manual reward; "sent" stays "enviada" |
| Growth (Audience › Growth) | Evolución | so it doesn't read as Grow ("Crecimiento") |
| held for review | retenido para revisión | short: "retenido" |
| hold period · in hold | periodo de espera · en espera | "the 7-day hold": "el periodo de espera de 7 días" |
| hosted (form, page) | en ReputeCast · alojado en ReputeCast | |
| invite page | página de invitación | |
| join · departure | alta · baja | Growth's joined and left: "Altas", "Bajas" |
| leaderboard | clasificación | |
| List health | Salud de la lista | as Growth and Scoring refer to it |
| milestone · milestone email | hito · correo de hito | "Hito 3" |
| net (growth) | neto · crecimiento neto | |
| newsletter | boletín | never "newsletter" in the interface |
| partner · partner code · partner link · partner profile · partnership | socio · código de socio · enlace de socio · perfil de socio · colaboración | |
| recommendation · recommend · Recommending you | recomendación · recomendar · Te recomiendan | |
| referral · referrer · referral link · referral program | referido · referidor · enlace de referido · programa de referidos | a referral is the subscriber who joined; the referrer shared the link |
| referral hub | página de recompensas | the referrer's own page |
| Referral · Recommendations · Reward block | bloque Referidos · bloque Recomendaciones · bloque Recompensa | the composer's Grow blocks |
| referring page · referring domain | página de origen · dominio de origen | never "referido", which is a referral |
| review queue · Review (tab) | cola de revisión · Revisión | |
| reward | recompensa | |
| rule · points | regla · puntos | |
| score · scoring | puntuación · sistema de puntuación | |
| self-referral | autorreferido | |
| soft bounce · hard bounce | rebote temporal · rebote permanente | |
| source (growth) | origen | |
| spread (of scores) | distribución | |
| subscribe request | solicitud de suscripción | what one instance sends a partner |
| threshold | umbral | |
| token (Liquid) | variable | as transactional templates' variables |
| trigger (automation) | disparador | |

## Audience, imports, forms and reports

| English | Español | Note |
|---|---|---|
| accent color · page background | color de énfasis · fondo de la página | public page branding |
| Added by the team (a contact's source) | Añadido por el equipo | a suppression added by hand is "Manual" |
| allowed domains | dominios permitidos | the sites a form may be embedded on |
| archive (public page) · issue | página de archivo · edición | as subscribers read them: "Archivo del boletín", "Próxima edición" |
| audience · merge field · cleaned (Mailchimp) | audience · merge field · cleaned | Mailchimp's own words, as its export and API write them |
| batch (an import's) | lote | the live log: "Lote 3 · 500 filas" |
| bounce types (a report's) | Permanente · Temporal · Bloqueado | "Permanente · el destinatario no existe", "Temporal · buzón lleno" |
| branding | marca | Forms & pages › "Marca y dominio" |
| catch-all domain · known complainers | dominio catch-all · personas que suelen quejarse | a verification service's risky groups |
| column · map fields | columna · asignar campos | an import's step: "Asignar campos" |
| confirmation link | enlace de confirmación | |
| consent · consent text · consent evidence | consentimiento · texto de consentimiento · evidencia de consentimiento | |
| custom domain | dominio personalizado | Branding & domain |
| data center (Mailchimp) | centro de datos | the API key's suffix, like us21 |
| deferral (provider) | aplazamiento | "deferrals expired": "aplazamientos caducados" |
| disposable domain · role address · likely typo | dominio desechable · dirección genérica · posible errata | an import's risky groups |
| embed code · script embed · embedded (form) | código de inserción · inserción con script · insertado | |
| erase (a contact) | borrar definitivamente | a GDPR erasure; a plain delete stays "eliminar" |
| excerpt | extracto | an archived issue's |
| export data (GDPR) | exportar datos | |
| Forms & public pages | Formularios y páginas públicas | |
| hosted page (a form type) | página en ReputeCast | as "hosted" above |
| Import from {tool} (a source) | Importación de {tool} | a contact's or suppression's source: "Importación de Mailchimp" |
| live log (an import's) | registro en tiempo real | |
| merge (a duplicate) | combinar | "Merge · fills first name": "Combinar · completa el nombre" |
| Migration (where a suppression came from) | Migración | an import from another email tool |
| offer a download | ofrecer una descarga | the file a form gives after confirmation |
| popup (form) | emergente | |
| preference center | centro de preferencias | in the app; subscribers read "página de preferencias" |
| publisher (name) | editor | heads every public page; the organization that publishes, not the role Editor |
| re-permission | reconfirmación | "re-permission flow": "proceso de reconfirmación" |
| read-only link | enlace de solo lectura | a shared report's |
| RSS feed | feed RSS | |
| self-hosted (a provider) | autoalojado | "Postal (autoalojado)" |
| sender health | salud del remitente | the Overview's score; reputation is "reputación" |
| signup · submission (form) | suscripción · envío | "Views · Submissions": "Vistas · Envíos" |
| single opt-in | suscripción simple | double opt-in is "doble confirmación" |
| site key · secret key (Turnstile) | clave del sitio · clave secreta | "Turnstile" stays |
| source status (an import's: subscribed, cleaned…) | as the source writes it | the consent table and the consent history keep the tool's own word |
| spam trap (audience, imports) | trampa de spam | |
| subscribe page · confirmation page · unsubscribe page | página de suscripción · página de confirmación · página de baja | |
| suppressed | suprimido | on the suppression list ("lista de supresión"); never "eliminado" |
| System stream · System mail | flujo Sistema · correo de Sistema | the stream's name keeps its capital |
| tag | etiqueta | |
| title (archived issue) | título | "asunto" is an email's subject, "encabezado" a heading |
| workbook · sheet (XLSX) | libro · hoja | as Excel's Spanish names them |

## Products, stores and transactional email

| English | Español | Note |
|---|---|---|
| abandoned checkout · abandoned cart | pago abandonado · carrito abandonado | as Shopify's Spanish admin says it |
| accepted marketing · didn't accept | acepta marketing · no acepta | a store customer's consent |
| attribution · credited to | atribución · atribuido a | "Attribution window": "Periodo de atribución" |
| Bought a product (trigger · condition) | Compra un producto · Ha comprado un producto | |
| buyer · customer | comprador · cliente | a product's buyer; a store's customer |
| checkout · at checkout | pago · al pagar | "Stripe Checkout" keeps its name |
| click token | token de clic | |
| collection (store) | colección | |
| compare-at price | precio de comparación | |
| cover (product) | imagen de portada | |
| delivery · delivery email · delivery template | entrega · correo de entrega · plantilla de entrega | a digital product's download link; a delivered email is still "Entregado" |
| dispute | disputa | never "queja", which is a spam complaint |
| download link · downloads | enlace de descarga · descargas | "4 of 5 downloads left": "quedan 4 de 5 descargas" |
| failover · primary · fallback | conmutación por error · principal · respaldo | a route's providers |
| Gross · Net | Bruto · Neto | product revenue before and after refunds |
| idempotency key | clave de idempotencia | |
| list offer | oferta de lista | the checkbox next to a product's Buy button |
| live · test (payments, keys, modes) | real · de prueba (as Earn has it) | the message log's live updates are "En tiempo real" |
| median delivery | tiempo mediano de entrega | |
| message (transactional) · Messages (tab) | mensaje · Mensajes | |
| order | pedido | |
| Order placed · First order · Placed an order | Pedido realizado · Primer pedido · Ha realizado un pedido | store triggers and the goal |
| Pay what you want · One-time | Paga lo que quieras · Pago único | a product's price |
| product · digital product | producto · producto digital | |
| Queued · Deferred · Bounced · Complained · Rejected | En cola · Aplazado · Rebotado · Con queja · Rechazado | a message's status; Delivered is "Entregado" |
| Raw (SMTP) | Sin procesar (SMTP) | mail an app sends already rendered |
| reconciliation | conciliación | a store's nightly check |
| refund · refunded | reembolso · reembolsado | |
| revoke access · restore access | revocar el acceso · restablecer el acceso | a product order's download |
| route · Transactional route | ruta · ruta Transaccional | as "Ruta Envíos masivos" |
| sales page · default product page · shop | página de venta · página de producto predeterminada · tienda | |
| send-only key · SMTP credential | clave solo de envío · credencial SMTP | "SMTP relay" keeps its name |
| single opt-in | suscripción simple | double opt-in is "doble confirmación" |
| store · connect · disconnect · sync | tienda · conectar · desconectar · sincronizar | Shopify and WooCommerce; their menu paths and field names stay as they write them |
| suppressed · opted out | suprimido · dado de baja | why a message wasn't sent |
| webhook topic | tema | orders/create stays as it is |
| write-only · sealed | solo escritura · sellado | a secret that's never shown again |

## Recurring digests

| English | Español | Note |
|---|---|---|
| approval rule · approver | regla de aprobación · aprobador | as in Approvals |
| archive (public) | archivo público | where sent issues are listed |
| back catalogue | artículos anteriores | a feed's older posts |
| category · tag (of a post) | categoría · etiqueta | WordPress's own Spanish words |
| Collecting · Draft ready · Not sent (an issue) | Recopilando · Borrador listo · No enviada | an issue's own states; the others are the campaign words |
| Create a draft · Send automatically · draft mode | Crear un borrador · Enviar automáticamente · modo borrador | the two sending modes |
| excerpt · featured image | extracto · imagen destacada | featured image as WordPress says it |
| fallback image | imagen alternativa | |
| feed · feed address | feed · dirección del feed | as Integrations; RSS 2.0, Atom 1.0 and JSON Feed 1.1 keep their names |
| feed item · Feed items (the block) | elemento · Elementos del feed | |
| feed token | variable del feed | a Liquid token: "variable", as in Grow |
| held · release (an issue) | retenida · liberar | "Release issue": "Liberar esta edición" |
| hours before (preparation) | horas antes del envío | "{count} hours before": "{count} horas antes del envío", never "a las 3" |
| issue · Next issue · issue #143 | edición · Próxima edición · edición #143 | as the public pages say it |
| late limit | límite de retraso | how late an issue may still go out |
| layout: List · Cards · Grid | Lista · Tarjetas · Cuadrícula | the Feed items block's layouts |
| left out (by a filter) | excluido | "Left out by a filter": "Excluido por un filtro" |
| owner (of a digest) | responsable | the member issues send under; not the Owner role |
| percentage points (pts) | puntos porcentuales (p. p.) | "{delta} pts": "{delta} p. p." |
| post | artículo | |
| prepare · Prepared at 07:00 | preparar · Preparada a las 7:00 | |
| recurring digest · digest | boletín periódico | as Integrations; the Recurring tab is "Periódicos" |
| Regular (campaign) | Normal | the other choice in New campaign |
| sending window | franja de envío | as Automations' send window |
| skip · skipped | omitir · omitida | |
| SSRF guard | protección contra SSRF | |
| starter (a template to begin from) | plantilla inicial | a starter's name ("Blog digest") stays as it is; what a new digest's starter writes into the email (subject, preheader) follows the workspace's content language |
| starting point | punto de partida | |
| turn on · pause (a digest) | activar · pausar | |
| weekdays (Monday to Friday) | de lunes a viernes | short: "lun–vie", the region format's own abbreviations |

## AI assistant

| English | Español | Note |
|---|---|---|
| accept · edit · reject (a suggestion) | aceptar · editar · rechazar | |
| Accepted with edits · No decision | Aceptada con cambios · Sin decisión | outcomes in the activity log |
| activity log (AI) | registro de actividad de IA | the audit log stays "registro de auditoría" |
| AI assistant · Assist (the menu) | asistente de IA · Asistente | |
| AI budget | presupuesto de IA | |
| cap · monthly cap | límite · límite mensual | in dollars or tokens; "No cap": "Sin límite" |
| condition schema · segment builder | esquema de condiciones · generador de segmentos | |
| deployment (Azure) | implementación | as Azure's Spanish says it |
| describe the segment · Draft (its button) | describir el segmento · Redactar | what it drafts is a "borrador" |
| feature (of the assistant) | función | |
| Image input · Structured output | Entrada de imágenes · Salida estructurada | what a model can do |
| input · output (tokens, prices) | entrada · salida | |
| instruction (to rewrite) | instrucción | "Rewrite with an instruction": "Reescribir con una instrucción" |
| keep · discard (a summary) | conservar · descartar | "kept summaries": "resúmenes conservados" |
| model · connect a model | modelo · conectar un modelo | model names stay as they are |
| private address | dirección privada | a server on a private network |
| prompt · test prompt | prompt · prompt de prueba | "el prompt"; a push prompt is "aviso" |
| refused by the model | rechazado por el modelo | |
| resets (the usage) | se restablece | "resets {date}": "se restablece el {date}" |
| rewrite · shorten · friendlier · more formal · more direct | reescribir · acortar · más cercano · más formal · más directo | |
| subject ideas | ideas de asunto | |
| suggestion | sugerencia | |
| summary (of results) | resumen | "Summarize results": "Resumir los resultados" |
| token (of a model) | token | what the provider bills; a Liquid token is "variable" |
| tone | tono | "Rewrite, shorten and tone": "Reescribir, acortar y cambiar el tono" |
| usage | uso | |

## Placement tests

| English | Español | Note |
|---|---|---|
| app password · app-specific password | contraseña de aplicación · contraseña específica de app | Google's and Apple's own words |
| arrived late | llegó tarde | |
| copy (of the test email) | copia | |
| deferral | aplazamiento | as Campaigns' "aplazamiento" |
| folder · tab | carpeta · pestaña | a mailbox's own names (Promotions, Junk Email) stay as it gives them |
| folders: Sent · Draft · Archive · Trash | Enviados · Borradores · Archivo · Papelera | the folders a test never reads |
| hand-off | traspaso | when the sending provider takes a copy |
| headers | cabeceras | "All headers": "Todas las cabeceras"; header names stay |
| Health (Deliverability › Health) | Estado | as Integrations' Health |
| Healthy · Warming up · Degraded · Down · Paused · Sandbox (a provider) | Operativo · En calentamiento · Degradado · Caído · En pausa · Sandbox | |
| Inbox · Tab · Spam · Not arrived | Bandeja de entrada · Pestaña · Spam · No llegó | the four results |
| median time to arrive | tiempo mediano de llegada | a median, as Approvals' median wait |
| pass · fail · aligned (authentication) | correcto · fallido · alineado | SPF, DKIM and DMARC keep their names |
| placement · placement test | ubicación · prueba de ubicación | where a copy lands |
| Queued (a copy) | En cola | |
| remove (a seed) | quitar | deleting copies stays "eliminar" |
| round (of copies) | ronda | "Round 2": "Ronda 2" |
| sample (the seeds) | muestra | as an A/B test's sample |
| seed · seed mailbox | buzón de prueba | mailboxes that exist only for tests |
| stop waiting | dejar de esperar | |
| test (one run) | prueba | "Tests · 12": "Pruebas · 12" |
| test token | token de prueba | what marks a copy as a test's |
| vs (providers compared) | frente a | "SES vs Postal": "SES frente a Postal" |
| warm-up cap | límite de calentamiento | a warming-up provider's daily cap |
| warm-up planner · warm-up plan | planificador de calentamiento · plan de calentamiento | |

## Address verification

| English | Español | Note |
|---|---|---|
| abuse · catch-all domain · disposable · role address | abuso · dominio catch-all · desechable · dirección genérica | why an address is risky; tags such as verify:catch-all stay as written |
| Add credits · Low credits | Añadir créditos · Quedan pocos créditos | |
| balance · credits | saldo · créditos | the prepaid credits at the service |
| bulk check · real-time check | verificación masiva · verificación en tiempo real | |
| check (one address) | verificación | "{price} a check": "{price} por verificación" |
| data region | región de datos | |
| Found by | Detectado por | which check found a risky address |
| imports (of contacts) | importaciones | "Bulk checks and imports": "Verificaciones masivas e importaciones" |
| job · Resume | tarea · Reanudar | |
| spam trap | trampa de spam | |
| Uses (of a service) | Usos | |
| Valid · Invalid · Risky · Unknown | Válida · No válida · De riesgo · Desconocida | the four results |
| verification service · verify | servicio de verificación · verificar | |

## Words used everywhere

| English | Español |
|---|---|
| contact | contacto |
| subscriber | suscriptor |
| list | lista |
| segment | segmento |
| template | plantilla |
| sending provider | proveedor de envío |
| mailbox provider | proveedor de buzón (Gmail, Outlook, Yahoo) |
| sender | remitente |
| domain | dominio |
| record (DNS) | registro |
| verified | verificado |
| sign in · sign out | iniciar sesión · cerrar sesión |
| password · passkey | contraseña · llave de acceso |
| authenticator app | aplicación de autenticación |
| recovery code | código de recuperación |
| two-factor | verificación en dos pasos (2FA) |
| invitation · invite | invitación · invitar |
| permission | permiso |
| Ask Marcus Chen or Linh Nguyen. | Pídeselo a Marcus Chen o Linh Nguyen. (the app joins the names and writes "u" before an "o") |
| notification | notificación |
| approval | aprobación |
| retry · try again | reintentar · volver a intentarlo |
| cancel · close · back | cancelar · cerrar · volver |
| copy · copied | copiar · copiado |
| optional · required | opcional · obligatorio |
| unsubscribe · unsubscribed | darse de baja · dado de baja; the noun is "baja" ("enlace de baja") |
| open · open rate | apertura · tasa de apertura |
| click · click rate | clic · tasa de clics |
| bounce · complaint | rebote · queja |
| dashboard | panel |
