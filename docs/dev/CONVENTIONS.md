# ReputeCast engineering conventions

Read this before writing code. It explains how a screen is built end to end
(API, SQL, demo data, web, tests, visual QA) so every part of the product
looks and behaves the same. The reference implementation is the campaigns
list: `internal/campaigns/list.go`, `internal/store/queries/campaigns.sql`,
`internal/demo/seed_campaigns.go`, `web/src/features/campaigns/` and
`web/src/routes/w/$workspace/_app/campaigns/index.tsx`.

## 1. Repository map

| Path | What |
|---|---|
| `cmd/reputecast` | The single binary (cobra commands in `internal/cli`). |
| `internal/app` | Wiring. Domains plug in from their own `wire_<domain>.go` with an `init()` (see `register.go`). |
| `internal/api` | Huma helpers: `api.Get/Post/...`, access rules, problems (errors), `PrincipalFrom`, `WorkspaceFrom`, `UserFrom`. |
| `internal/authz` | Permission keys, built-in roles, escalation guard. |
| `internal/authn` | Passwords, tokens, TOTP, recovery codes, breach check. |
| `internal/audit` | `audit.Logger.Log/LogWith` — every state change writes an entry. |
| `internal/store` | sqlc output (do not edit) and `queries/<domain>.sql` (edit). |
| `internal/<domain>` | One package per domain: `Service` + `Register(huma.API, *Service)` + types. |
| `internal/providers` | The `Provider` contract and registry; adapters in `internal/providers/<name>`. |
| `internal/notify` | Create notifications (`notify.ToUser`, `notify.ToMembersWith`). |
| `internal/mailer` | System mail (`mailer.Sender`, `App.Mail`). |
| `internal/sending` | The send contract (`sending.Starter`, `App.Sending`). |
| `internal/linktoken` | Signed tokens in emails: open, click, unsubscribe, preferences, view online, confirm (`App.Links`). |
| `internal/publicurl` | Absolute public URLs: `/t/o`, `/t/c`, `/p/{workspace}/…`, `/r/{token}` (`App.URLs`). Never build these paths by hand. |
| `internal/demo` | The Acme sample data (`reputecast seed`). One file per domain. |
| `internal/apptest`, `internal/testdb` | Integration-test harness on real Postgres. |
| `migrations` | goose SQL migrations (embedded). |
| `web/src` | The React app (see §4). |
| `design/boards` | The prototype: one `*.dc.html` per artboard. **The UI source of truth.** |
| `design/design-system` | Clearwater tokens and the brand book (`README.md`). |

## 2. Backend

### Domain package shape

```go
// internal/widgets/widgets.go
type Service struct { Pool *pgxpool.Pool; Q *store.Queries; Audit *audit.Logger }

func (s *Service) List(ctx context.Context, ws *store.Workspace, ...) (WidgetList, error)

// internal/widgets/api.go
func Register(h huma.API, s *Service) {
	api.Get(h, api.Op{ID: "list-widgets", Path: "/workspaces/{workspace}/widgets",
		Summary: "List widgets", Tags: []string{"Widgets"}, Access: api.Workspace(authz.WidgetsView)},
		func(ctx context.Context, in *listWidgetsInput) (*api.Body[WidgetList], error) { ... })
}

// internal/app/wire_widgets.go
func init() { apiRegistrars = append(apiRegistrars, func(a *App, h huma.API) { widgets.Register(h, &widgets.Service{...}) }) }
```

Rules:

- **Every operation declares its access**: `api.Public`, `api.Authenticated`,
  `api.Pending` (allowed before setup is finished), `api.InstanceAdmin`,
  `api.Workspace(perm)` or `api.WorkspaceHuman(perm)` (no API keys/agents).
  Workspace operations have `{workspace}` (the slug) in the path and embed
  `api.WorkspaceInput`. The middleware has already loaded the workspace
  (`api.WorkspaceFrom(ctx)`) and checked the permission; check *extra*
  conditions in the service (scope, ownership, "can't grant what you don't hold").
- **Every state change writes an audit entry** in the same transaction:
  `s.Audit.LogWith(ctx, store.New(tx), audit.Event{Action: "routing.update", TargetType: "routing_rule", TargetID: …, Before: …, After: …})`.
  Actions are dotted `noun.verb`. Never log secrets.
- **Errors** are RFC 9457 problems. Use `api.NotFound("Campaign")`,
  `api.Conflict(…)`, `api.Invalid("body.email", "…")`, `api.Forbidden(perm, helpers)`,
  or `api.Err(status, code, detail).WithMeta(…)`. Wrap DB errors with
  `api.MapError(err, "Campaign")` (pgx.ErrNoRows → 404). Messages are
  sentences that say what happened and what to do, in the product voice.
- **Huma types**: exported type names must be unique across the whole API
  (they become OpenAPI schema names): prefer `CampaignListItem` over `Item`.
  Optional request fields need `omitempty` (otherwise Huma marks them required).
  Use `enum:"a,b"`, `minimum`, `maxLength`, `format:"email"` to validate.
  **Never return nil slices** (they serialize as `null`; the TS types say arrays):
  initialize with `[]T{}` or `make`.
- **Secrets are write-only**: encrypt with `a.Secrets.Seal(b, "provider:<id>:credentials")`
  (associated data names the row and field), never return them, show
  “••••” + a Replace action in the UI.
- **Transactions**: `db.Tx(ctx, pool, func(tx pgx.Tx) error { q := store.New(tx); … })`.
- **IDs** are UUIDv7: `uuid.Must(uuid.NewV7())`.
- **Times** are `timestamptz`; the API returns RFC 3339; the web formats them
  in the workspace timezone.
- **Background work** uses River: add workers with `river.AddWorker(a.JobReg.Workers, …)`
  from a `jobRegistrars` init in your wire file. Pick a queue from `internal/jobs`.
- **No new dependencies** outside `docs/TECH_STACK.md` without a reason in the PR.

### SQL and migrations

- Queries live in `internal/store/queries/<domain>.sql`; run `go tool sqlc generate`.
- **Names are global.** sqlc query names share one package (`store`), demo
  helpers share `package demo`, and API schema names share one OpenAPI
  document. Prefix with your domain: `AutomationProviders` not
  `WorkspaceProviders`, `ComposerCustomFields`, `deliverabilityDemoKey`,
  `ImportFieldOption`. (Huma falls back to a package prefix on a schema-name
  clash, but the web types then change name — pick unique names up front.)
- Nullability: sqlc can't infer it through `LEFT JOIN LATERAL` or aggregates.
  Wrap with `coalesce(x, 0)::bigint` / `coalesce(x, '')::text` so pgx never
  scans NULL into a non-pointer.
- Use `sqlc.narg(name)` for optional filters, `@name::type` for typed params.
- Migrations: goose files in `migrations/`. **Number ranges are reserved per
  workstream** (see `docs/dev/WORKSTREAMS.md`) so parallel branches never
  collide. Postgres 16–18: no PG18-only syntax. Every migration has a `Down`.
- The base schema (00001–00010) already has tables for every domain; extend it
  with `ALTER TABLE` in your range rather than creating parallel tables.

### Demo data

The demo (`reputecast seed`) must make every designed screen look like its
board. Each workstream owns `internal/demo/seed_<domain>.go` and assigns its
seeder in `init()` (`func init() { seedAudience = seedAudienceImpl }`); the
order is fixed in `demo.go`. Use `s.Ago(d)` / `s.designDay(daysAgo, h, m)`
for times (relative to now), `demo.Split` so breakdowns add up exactly, and
the shared maps (`s.Users`, `s.Lists`, `s.Providers`, `s.Campaigns`…) to link
data. Rows that need IDs from a seeder later in the order go in
`s.After(func(ctx, s) error)`, which runs once every seeder is done. Before
inserting a shared row (a domain, a contact, a campaign), look it up in the
maps first: another seeder may already have created it. Use the sample data from the specs and the boards (names, numbers,
copy). Keep totals internally consistent: when a board's numbers contradict
each other, keep the per-row numbers and let totals be computed.

### Tests

- Unit tests next to the code; integration tests use
  `env := apptest.New(t); env.SeedDemo(); c := env.Client(); c.SignInDemo("linh@acme.io")`
  and call the real HTTP API. Cover: happy path, validation, permission denied
  (sign in as `aisha@acme.io`, the Viewer, or `daniel@acme.io`, the Content
  Writer), audit entry written, no secret in responses.
- Run with the compose Postgres: `REPUTECAST_TEST_DATABASE_URL=postgres://reputecast:reputecast@localhost:5433/postgres?sslmode=disable go test ./...`.
  When several people (or worktrees) test against the same server, add
  `-p 4`: each package's app keeps a small pool, but many packages at once
  can still exhaust `max_connections` ("too many clients").

## 3. API contract → web types

After changing any operation:

```bash
go run ./cmd/reputecast openapi > web/openapi.json
cd web && pnpm api:types
```

Call the API only through the generated client: `$api.useQuery('get', '/api/v1/…', { params })`,
`$api.useMutation('post', '/api/v1/…')`, or `http.GET/POST` in loaders and
handlers. Never hand-write API types; derive them: `type Item = Schemas['CampaignListItem']`.

## 4. Web app

### Structure

- Routes are files under `web/src/routes` (TanStack Router, generated tree):
  - `w/$workspace/_app/...` — screens with the sidebar (`<Page>` renders the top bar).
  - `w/$workspace/<name>/...` — full-screen editors and wizards (composer,
    flow editor, setup and migration wizards).
  - top level — sign-in, setup, invite, reset (`AuthLayout`).
- Screen code lives in `web/src/features/<domain>/`; route files stay thin
  (search-param validation with zod + render the feature component).
- Shared UI is in `web/src/components/ui` (Button, Badge, Card, Note, Banner,
  Field/Input/Select/Checkbox/Toggle/OptionCard, Table/Th/Td, NavTabs/PillTabs/
  PanelTabs/Segmented, Dialog, Menu/Popover/Tooltip, Skeleton/EmptyState/
  ErrorPanel/Spinner, toast, Bar/StackBar/StatTile/KeyValues/ScoreRing/Sparkline,
  Avatar, Row/Stack/Grid) and `components/shell` (AppShell, Page, AuthLayout,
  FullPageState). Use them; extend them when a board needs a variant; don't
  fork lookalikes.

### Porting a board

1. Open the board source (`design/boards/<Board>.dc.html`) and the rendered
   board (`http://localhost:8765/<Board>.dc.html`, see §6). Read its
   `<helmet><style>`: shared classes (`.card`, `.btn`, `.badge`, `.tbl`, `.k`,
   `.n`, `.note`, `.kv`, `.chip`, `.sec`, `.dlg`…) already exist in
   `web/src/styles/app.css` with the same values. Board-specific classes are
   ported into the feature (Tailwind utilities, or a feature CSS file using the
   `var(--token)` colours) with **exact** values.
2. Reproduce the DOM structure, copy and order exactly. Inline styles become
   Tailwind utilities with the same values (`gap-2.5` = 10px, `px-4.5` = 18px,
   `text-[12.5px]`, `rounded-[10px]`), colours via tokens (`text-ink-3`,
   `bg-ground`, `border-line-2`, `bg-ok-tint`); hex values only when a board
   uses a colour that has no token.
   **Box model:** the boards use the browser default — `content-box`, except
   buttons, selects, tables and button-like inputs (search, checkbox, radio,
   submit…), which are `border-box`; text inputs and textareas are
   `content-box` unless a class says otherwise (`.in` is `border-box`). The app restores exactly that default in `app.css`
   (Tailwind's preflight would make everything `border-box`), so a width,
   height or flex-basis copied from a board renders at the same size. Where a
   board sets `box-sizing:border-box` itself (inline or in a class), add
   `box-border`. Don't compensate sizes by hand; measure both pages when in
   doubt.
   **Links:** as on the boards, links in running text, spans and table cells
   are underlined and standalone links get a 24px target (rules in
   `app.css`). A board link with an inline `color` is a name, not running
   text, and has no underline (and no hover colour): add `no-underline` and
   the colour class (`text-ink`).
3. Replace sample data with API data. Copy that is not data stays literal.
4. Reproduce the responsive behaviour. The boards get it from global rules in
   their CSS; in the app use the primitives:
   - `<Row>`: a flex row that wraps on phones (the boards wrap every flex row
     that isn't a column at ≤767px). Use `nowrap` where the board's row has
     `nowrap`, a height or `gap:2px`.
   - `<Grid min={380}>` (= `repeat(auto-fit,minmax(380px,1fr))`): one column on
     phones when min ≥ 260, else 140px tracks. `<Grid cols={4}>`: 4 equal columns
     that become 140px auto-fit tracks on phones.
   - Fixed widths 300–420px become `max-md:w-auto max-md:min-w-0 max-md:max-w-full`.
   - Tables scroll inside their card (`<Table>` does it); key column min-width
     and `white-space` as on the board. At ≤767px badges and buttons in cells
     stay on one line by default (the team and access boards have
     `.tbl td .badge, .tbl td .btn {white-space:nowrap}`); boards without that
     rule let them wrap: pass `<Table phoneWrap>`.
   - At ≤1100px the shell is a rail and the content padding is 16px; at ≤767px
     the bottom tab bar appears, padding is 12px and gaps 12px.
   Check every board at 1440, 834 and 390 (the canvas has Tablet/Phone boards
   for most screens).
5. Every state: the board's `…States` boards (loading = skeletons shaped like
   the content, never a spinner alone; empty = a sentence + the one action;
   error = what failed, what ReputeCast did, what to do), plus domain states
   (outage, paused, blocked, read-only for Viewers…). States come from real
   API conditions: `isPending`, `isError`, empty arrays, entity fields.
6. Motion only where the design system allows it (README › Motion): rise,
   pop, grow, pulse, counters, flow pulses. Respect `prefers-reduced-motion`
   (global rule in app.css). Hover states as on the boards.

### Behaviour rules

- **Permission-aware**: `useCan(perm)` / `usePermissionReason(perm)`; disabled
  controls use `<Button disabledReason={reason}>` (stays focusable,
  `aria-disabled`, explains who can help). The API enforces it too.
- **Dates/numbers** only through `lib/format` (`count`, `pct`, `rate`, `money`,
  `shortDate`, `clock`, `ago`, `until`) — they use the workspace timezone and
  the design's formats ("Tue 14 Oct · 09:00", "48,213", "99.1%", "0.04%").
- **Data** with TanStack Query via `$api`. Invalidate the affected queries
  after mutations. Optimistic updates only where the design shows instant feedback.
- **Feedback**: `toast({ kind, title, detail, action })` for results that
  aren't visible on the page; errors next to fields (`Field error`),
  form-level summaries for several errors.
- **Destructive actions** use `Dialog` with consequences listed first, the
  safer alternative, and a typed confirmation or reason where the design asks.
- **Forms**: react-hook-form + zod when a form has more than a couple of
  fields; errors are sentences next to the field with `aria-invalid`.
- **URLs** hold UI state that should survive reload/share (filters, tabs, page).

### Text and app languages

The app comes in English and Vietnamese (design expansion, phase 2,
`docs/spec/reputecast-design-addon-connect-reach.md` §3.7). Every string a
person reads goes through `web/src/i18n`; screens written before that are
being converted.

- **Messages** are ICU MessageFormat, declared next to the code that uses
  them. A feature has a `messages.ts`, or `<screen>.messages.ts` when it's
  large:

  ```ts
  export const m = defineMessages('transactional', {
    title: 'Transactional',
    sent: '{count, plural, one {# message} other {# messages}} sent',
    sentTo: 'Sent to <b>{name}</b> via {provider}',
  })
  ```

  Use `useT()` for plain text (`t(m.sent, { count })`), `<T msg={m.sentTo}
  values={{ name, provider, b: (c) => <b>{c}</b> }} />` for markup, and `t()`
  from `@/i18n` outside React. Shared words are in `common` (`@/i18n/common`).
- **IDs** are `namespace.key`: the namespace is the feature (`transactional`,
  `integrations.catalog`), the key says what the text is for (`emptyTitle`,
  `publishButton`), not what it says. Never reuse a message for a different
  purpose because the English happens to match.
- **Whole sentences only.** Never build a sentence from pieces or join
  translated fragments; put the variables inside one message. Plurals use
  `{n, plural, one {…} other {…}}` (Vietnamese has only `other`). Lists use
  `formatList`, sorting uses `compareText`.
- **Numbers and dates** still go through `lib/format`; pass the formatted
  string into the message. `{n, plural}` and `#` follow the region format.
- **Translations** sit next to the source as `messages.vi.ts` (or
  `<screen>.messages.vi.ts`), default-exporting `translate(m, { … })`. Keys
  are checked against the source; a missing one shows English.
- **Never translated:** workspace content (campaign, list, template and form
  names), permission keys, audit action names, API error codes, provider
  names and codes, DNS records, code samples and CLI output.
- **Server text** a member reads (problem details with their fix, "who can
  help", notifications, member emails, the invitation page's role
  description) comes from the package's `messages/<locale>.json` catalog
  through `internal/i18n`, in the request's language (`i18n.LangFrom(ctx)`).
  API clients get English unless they send `Accept-Language`.
- **Checks:** `pnpm run i18n:check` lists what each language misses, by file,
  and fails on broken ICU or a translation that uses an argument or a tag
  its English doesn't have. It writes `src/i18n/status.gen.ts`: outside
  development the language picker offers only complete languages. Before a
  screen is done, look at it in the pseudo-locale (Account & security ›
  Language & region › Pseudo (+40%), in development): text without ⟦ ⟧
  skipped `web/src/i18n`, and a missing ⟧ is text that was cut off.
  `docs/i18n/translating.md` is the translators' guide.

### Text expansion

Translations run longer than English: Vietnamese by about a quarter, the
pseudo-locale by 40%. The stress-test boards (`StressCampaigns…`,
`StressReview…`, `StressTeam…`) settle how screens make room:

1. **Never cut off** verdicts, status badges, numbers, buttons, permission
   keys, error messages or anything in a dialog. They wrap, or the layout
   makes room.
2. **Ellipsis, with the full text on hover and focus,** for user content in
   table cells (campaign, list and member names, email addresses) and the
   middle of breadcrumbs: wrap the cell's link in `components/ui/Truncate`.
3. **At most two lines** for table headers, stat-tile labels, sidebar labels
   and card captions. `Th` measures a translated header on two lines, so its
   column never gets narrower than that and doesn't squeeze its numbers;
   stat-tile labels use `line-clamp-2` with a `title`.
4. **Buttons don't wrap.** A button group that doesn't fit moves to its own
   row, as the page header's actions and the composer header's do.
5. **Badges don't wrap;** the cell wraps around them.
6. **Tabs scroll sideways** (`NavTabs`, `FilterTabs`), as in English.
7. **Line height of at least 1.4** for body text in Vietnamese, whose
   diacritics stack (ệ, ở, Ữ); overlines stay uppercase. Anything clipped
   (`overflow: hidden`) keeps room above and below for the marks.

Rules that change a layout for longer text live in
`web/src/styles/expansion.css` and apply only when the app isn't in English
(`:root:not([lang='en'])`), so English keeps the boards' layout. The board's
proposal for text 40% longer (cell padding of 10 px, a 160–170 px name
column, row actions stacked) applies to the pseudo-locale until a language
that long is added. Never select elements by their text (`[aria-label=…]`,
`:has-text`): it breaks in every other language.

### Accessibility (WCAG 2.2 AA, as audited on the boards)

One `h1` per screen (the top bar's title), landmarks (`header`, `main#main`,
`nav` with labels), `th scope="col"`, every control labelled (visible label or
`aria-label`), icon-only buttons with `aria-label`, decorative SVG
`aria-hidden`, charts with `role="img"` + `aria-label` and a table or list
view, status text not by colour alone (badges carry a word), live regions for
counters and toasts (`role="status"`), errors with `role="alert"`, 44px touch
targets on phones, focus visible everywhere, dialogs trap and return focus
(Radix), no horizontal page scroll at 320px.

### SEO and documents

App routes set the document title through `<Page title>` / `AuthLayout`
("Campaigns · Acme Newsletter · ReputeCast") and are `noindex`. Public pages
(server-rendered, `/p/…`) and the landing site have full metadata (title,
description, canonical, Open Graph, JSON-LD where relevant).

## 5. Running locally

```bash
docker compose up -d                       # Postgres (PG_PORT, 5433 here) + Mailpit
go run ./cmd/reputecast seed               # REPUTECAST_ENV=development + DB URL, see Makefile
make dev                                   # API :8080 (air) + web :5173 (Vite proxy → API)
```

Parallel workstreams use their own database and ports (see WORKSTREAMS.md):

```bash
createdb -h localhost -p 5433 -U reputecast reputecast_<ws>
export REPUTECAST_ENV=development REPUTECAST_DATABASE__URL=postgres://reputecast:reputecast@localhost:5433/reputecast_<ws>?sslmode=disable
go run ./cmd/reputecast seed
REPUTECAST_HTTP__ADDR=:81NN REPUTECAST_HTTP__PUBLIC_URL=http://localhost:52NN REPUTECAST_AUTH__COOKIE_SECURE=false go run ./cmd/reputecast serve --auto-migrate &
cd web && WEB_PORT=52NN REPUTECAST_BACKEND=http://localhost:81NN pnpm dev &
```

Demo sign-in: `linh@acme.io` / `reputecast-demo` + TOTP secret `JBSWY3DPEHPK3PXP`
(Owner, instance admin); `aisha@acme.io` / `reputecast-demo` (Viewer, no 2FA).

## 6. Visual QA (required for every screen)

```bash
cd web
node scripts/shot.ts --app http://localhost:52NN --path /w/acme-newsletter/campaigns --design Campaigns --widths 1440,834,390 --out ../tmp/shots/<ws>
```

It signs in as the demo Owner, captures the app at each width and the design
board at the same widths (design server: `http://localhost:8765`, started with
`python3 -m http.server 8765 -d design/boards`; README › Development says
which boards also need the canvas runtime). Open the
PNGs (Read tool) side by side and fix every difference in layout, spacing,
type, colour, wrapping, alignment and copy that isn't caused by data. Then
check the phone and tablet boards (`<Board>Phone`, `<Board>Tablet`) and every
state board. A screen is done when a reviewer can't tell the app screenshot
from the board except for live data.

## 7. Definition of done for a screen

- [ ] Matches its boards at 1440/834/390, all states implemented.
- [ ] Real API (typed), permissions enforced server-side and explained client-side.
- [ ] Audit entries for every change; secrets never returned.
- [ ] Demo data makes it look like the board.
- [ ] Go integration test for the API; unit tests for non-trivial logic (Go or Vitest).
- [ ] `go vet ./...`, `go test ./...`, `pnpm lint`, `pnpm typecheck`, `pnpm test`, `pnpm build` pass.
- [ ] Accessibility checklist above, and `pnpm a11y --only <its boards>` (axe-core, WCAG 2.0–2.2 A and AA) reports nothing.

## 8. Git

Work on your branch and commit often. Commit messages follow
[Conventional Commits 1.0](https://www.conventionalcommits.org/en/v1.0.0/):
`type(scope): description`.

- **Type:**
  - `feat`: a new capability.
  - `fix`: a bug, or a screen that doesn't match its board.
  - `docs`: documentation, the roadmap and the design inventory.
  - `test`: tests and the QA scripts.
  - `refactor`: same behaviour, different code.
  - `perf`: faster or lighter.
  - `style`: formatting and lint only.
  - `build`: Makefile, Dockerfile, release and dependencies.
  - `ci`: workflows.
  - `chore`: generated files, branch syncs and the rest.
  - Mark a breaking change with `!` after the type or scope, plus a `BREAKING CHANGE:` footer.
- **Scope:** optional. Use the feature or package the change is about, in lower case: `audience`, `sending`, `composer`, `providers`, `approvals`, `forms`, `site` and so on.
- **Description:** imperative, lower case after the colon, no full stop. The whole first line is at most 100 characters. Explain why in the body.
- **No attribution trailers:** no `Co-Authored-By` lines or other AI attribution.

```
fix(campaigns): count only blocking issues for a blocked draft

The list said "5 blocking issues" where Review & send says 3: it counted
the warnings too.
```

Don't edit files owned by another workstream except the shared, append-only
ones listed in WORKSTREAMS.md. Generated files (`internal/store/*.go`,
`web/openapi.json`, `web/src/api/schema.d.ts`, `web/src/routeTree.gen.ts`) are
regenerated, never hand-edited.
