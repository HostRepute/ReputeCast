# ReputeCast

The design system behind ReputeCast: the self-hosted newsletter platform that checks deliverability before you send and guards sender reputation while you send. It covers the app, the setup and migration wizards, the public pages subscribers see, the landing page and the GitHub banner. Internally it is called **Clearwater**: calm surfaces, honest numbers, one accent.

Built by HostRepute. Every token here was lifted from the clickable prototype, so a screen built with these values matches the prototype pixel for pixel.

## Principles

1. **The verdict first.** Every screen answers one question at the top: is my sending healthy, will this campaign land, what paused and why. Headline numbers and Pass / Warn / Block badges sit above everything else; detail is a scroll or a click away.
2. **Honest, never alarming.** Red means a real block or an active pause. Amber means something to look at before you send. Green is quiet. There is no colour for "fine, but buy the upgrade": the product is open source and the UI has nothing to sell.
3. **One accent.** Teal is the only brand colour in the chrome. Status colours carry meaning and are never decorative. Chart series have their own fixed palette and never borrow from status.
4. **Borders, not shadows.** Structure comes from 1px lines on white. Shadows only lift the few things that float: menus, modals, the one hero card.
5. **Numbers you can line up.** Tabular figures everywhere a number appears, units after the number, provider names spelled the way the provider spells them.
6. **The same page at every width.** Screens reflow; they do not hide features on phones. The sidebar becomes a rail at tablet width and a tab bar on phones, and every table either wraps its secondary cells or scrolls inside its card.

## Colour

Tokens are in `tokens.json`; the page's Colors view lists each with its use. Rules of thumb:

- **Text** uses `ink` (primary), `ink-2` (descriptions), `ink-3` (labels, metadata), `ink-4` (placeholders). Never colour body text with a status or chart colour; put a coloured dot or badge beside it instead.
- **Surfaces**: the app sits on `ground`; cards, the sidebar and the header are `surface` with a `line` border; inset panels inside a card use `ground-3`. Row dividers are `line-2`. Secondary buttons use `line-strong`; inputs use the darker `line-input` so their boundary meets 3:1.
- **Accent** (`accent`, `accent-hover`) is for the primary action on a screen, links, the active nav item (`accent-soft` fill) and the current wizard step. `accent-tint` and `accent-line` highlight ReputeCast's own column in comparisons and the "before you send" feature cards.
- **Status** comes in pairs of fill and text (`ok-bg`/`ok-fg`, `warn-bg`/`warn-fg`, `bad-bg`/`bad-fg`, `info-bg`/`info-fg`, `neutral-bg`/`neutral-fg`, `purple-bg`/`purple-fg`) plus a solid bar colour (`ok-bar`, `warn-bar`, `bad-bar`). A status badge always has a dot or icon and a word; colour alone is never the message. Alert cards use `bad-tint` with a `bad-line` border; success panels `ok-tint` with `ok-line`.
- **Dark surfaces** (`ink`, `dark-surface`, `dark-line`, `dark-text`, `dark-muted`) are reserved for code and terminal blocks and the landing page's install section. The app itself is light only in this version.
- **Public pages** belong to the publisher, not to ReputeCast: they take the workspace's brand colour (`publisher-accent`, sample Acme indigo) on a tinted ground and never show the teal.

### Chart palette

Validated for colour-vision deficiency with the dataviz palette checks. Assign in fixed order and never cycle:

| Job | Tokens, in order |
|---|---|
| Mailbox providers (categorical) | `chart-gmail`, `chart-outlook`, `chart-yahoo`, `chart-icloud`, `chart-other` |
| Sending providers (categorical) | `chart-ses`, `chart-postal`, `chart-postmark` |
| Engagement tiers (ordinal) | `chart-tier-1` Active → `chart-tier-4` Sunset |
| Opens | `chart-human` (ink) for human opens, `chart-machine` for Apple Mail prefetches |
| Tracks | `chart-track` behind every bar |

Bars are thin, rounded only at the data end, with a 2px surface gap between stacked segments. One axis per chart. Legends appear from two series up; values are labelled selectively, never on every point. Every chart has a table view.

## Type

IBM Plex Sans for everything, IBM Plex Mono for code, ids, DNS records and terminal lines. Both load from Google Fonts in the prototype; self-host them in production.

| Style | Use |
|---|---|
| `app-title` 24/1.2 bold, tight | Wizard and settings page titles |
| `app-h1` 20/1.25 bold | Screen title in the top bar |
| `app-h2` 15/1.4 semibold | Card and section headings |
| `app-body` 14/1.45 | Default text; `app-body-strong` for names and keys |
| `app-label` 13 semibold | Buttons, input labels, tabs |
| `app-caption` 12 medium | Metadata, badges, helper text, axes |
| `app-overline` 11 semibold, uppercase, +0.05em | Table headers, group labels |
| `app-number` 28/1.1 bold, tabular | Stat tiles |
| `mono-code` 12/1.6, `mono-inline` 13, `mono-terminal` 14/1.6 | Code panes, inline ids, terminal blocks |
| `mk-display` 52/1.05, `mk-h2` 34/1.15, `mk-lead` 18, `mk-body` 16, `mk-card-title` 17, `mk-button` 15 | Landing page; display drops to 34 and `mk-h2` to 26 on phones |

Sentence case everywhere, including buttons and table headers (the overline style is the only uppercase). Headings state the fact, not the feature: "Nothing is sending right now", not "Status". Percentages keep one decimal (99.1%), counts use thousands separators (48,213), rates show their unit (14 /s).

## Spacing, radius, elevation

A 4px grid. Cards are padded 18px 20px, inset panels 12px 14px, rows 10px to 12px vertical. Content padding is 28px on desktop, 16px on tablets, 12px on phones. Gaps between cards are 20px, 16px and 12px at the same three widths.

Radius: 8px for controls, 10px for cards, 14px for marketing cards, pill for badges, bars and chips. The logo mark is a 26px rounded square (7px radius) in `accent`.

Elevation: `shadow-sm` for menus, `shadow-md` for hover lifts and dragged nodes, `shadow-lg` for modals and hovering feature cards, `shadow-hero` for the single floating card in the landing hero.

## Layout

- **App shell**: a 232px sidebar (`surface`, right border `line`) with the logo, the workspace switcher, the nav and a footer with the "New campaign" action; a top bar with the screen title, optional tabs and actions; a content column on `ground`.
- **Breakpoints**: at 1100px and below the sidebar becomes a 72px icon rail (labels hidden, 44px targets); at 767px and below it becomes a fixed bottom tab bar with 44px icon buttons and the content gets 72px of bottom padding. Tabs in the top bar scroll horizontally. Side panels in the composer and flow editor stack below the canvas at full width.
- **Wizards** (setup, migration) use a header with the step badge and an "Exit setup" link, a stepper card on the left (250px) and the step content on the right; on phones the stepper turns into a horizontal strip of step numbers.
- **Tables** wrap their secondary cells first (`min-width` on the key column, `white-space:normal` on descriptions); if a table still cannot fit, it scrolls inside its card, never the page.
- **Public pages** are a single 560px column centred on the publisher's ground; the archive is 720px.

## Components

**Buttons** `.btn`: 36px tall (30px small, 40px for the one primary wizard action, 44px on the landing page), 8px radius, `line-strong` border on white; `.pri` fills with `accent`; destructive actions use `bad-strong`. Icon-only buttons are 44px squares in the rail and tab bar. Hover darkens the fill; nothing moves in the app, only on the landing page.

**Badges** `.badge`: 22px pill, 12px semibold, a 8px dot or an icon before the word. Fills: ok, warn, bad, info, neutral, purple. Use them for state words only (Healthy, Warming up, Auto-paused, Scheduled, Step 4 of 6), never for counts or labels that are not states.

**Cards** `.card`: white, `line` border, 10px radius. A card has one heading (`app-h2`), optional caption and one action group in its top-right corner. Rows inside a card are separated by `line-2`.

**Inputs** `.in`: 36px tall, 8px radius, `line-input` border, label above in `app-label`, helper or error text below in `app-caption`. Error state: `bad-bar` border and a message that says what to change. Selects look like inputs. Checkboxes and radios are 16px, accent when checked.

**Tabs**: text tabs under the screen title, active tab in `ink` with a 2px `accent` underline; secondary tabs inside cards are pills.

**Nav**: 36px items, 8px radius, icon plus label; active item `accent-soft` fill with `accent` text. The rail shows only icons with a tooltip; the phone tab bar shows the five primary destinations.

**Stepper** (wizards): 26px numbered circles; done steps `ok-bg` with a check, current step `accent`, upcoming steps outlined `line-strong`.

**Status rows** (readiness, timeline): a badge, a sentence that states the finding, and the fix or detail link on the right. Blocked rows come first, then warnings, then passes.

**Stat tiles**: label in `app-caption`, number in `app-number`, a one-line comparison below (▲ 0.4 pts vs last send) in `ok-fg` or `bad-fg` with the arrow, never colour alone.

**Bars and tracks**: 10px to 14px tall, `chart-track` background, segments in chart or status colours with a 2px gap, rounded at the data end only.

**Banners**: full-width inside the content column, above everything; `bad-tint` with `bad-line` for an active pause or outage, `warn-bg` for a warning, `info-bg` for scheduled or informational notices. A banner always has an action.

**Empty, loading, error**: an empty state is a card with a sentence that says what will appear here and the one action that makes it appear. Loading uses grey skeleton blocks shaped like the content, never a spinner alone. Errors say what failed, what ReputeCast already did about it and what the person can do next.

**Code and terminal blocks**: `ink` or `dark-surface` background, `mono-code` or `mono-terminal`, tag / attribute / string colours from the `code-*` tokens, line numbers in `ink-4`.

## Patterns

- **Send Readiness** is the signature pattern: a verdict badge (Ready, Ready with warnings, Blocked), a list of checks in Block → Warn → Pass order, each with a fix, and a footer with recipients, cost and time. The same block appears in the composer, before scheduling and in the landing hero.
- **Guarded sending**: a live bar per sending provider, counters that only ever go up, and an auto-pause banner that names the threshold, the number held and the choices (resume, adjust, fail over, cancel).
- **Provider health** everywhere a provider is named: a badge (Healthy, Warming up, Degraded, Outage, Webhooks silent) and the number that explains it.
- **Routing** is shown as a weighted bar per mail kind (Broadcasts, Automations, System) in the sending-provider colours, with the fallback written out.
- **Consent is visible**: migration and import screens always show how many contacts stay suppressed and why.

## Access, teams and approvals

- **Permission-aware controls.** When the current member can't do something, hide the action or disable it with the reason and who can help ("Sending needs campaigns:send. Ask Marcus Chen or Linh Nguyen."). Disabled buttons keep `aria-disabled` and a `title`; banners use `role="status"`.
- **Sensitive permissions** carry a flag (⚑, `warn-bar` colour) wherever they appear: role editor, member detail, API keys, MCP agents. Holders must use two-factor; the UI shows the grace period.
- **Write-only secrets.** Provider credentials, OIDC client secrets and API keys show as masked fields with a Replace action and are never read back. New keys appear once, in a dialog that requires a "copied" confirmation.
- **Deliberate destructive actions.** Delete workspace, transfer ownership, emergency bypass and two-factor resets use a dialog with a typed name or a required reason, list consequences first, and offer the safer alternative (export first, keep API keys).
- **Approval states** reuse the Send Readiness verdict banner: warn for "approval required" and "waiting", success for "approved", danger for "invalidated" and "blocked", always with the diff of what changed.
- **Shell.** Workspace switcher under the brand, notification bell with an unread count and the user menu in the top bar; instance admins get an "Instance admin" entry with its own tab set.

## Motion

Motion belongs on the landing page and in live screens; the rest of the app stays still except for hover fills and skeletons.

- Entrances rise 18px and fade in over `duration-slow` with `ease-out-soft`, staggered 90ms to 100ms.
- Readiness checks tick in one by one; badges pop (scale 0.5 → 1.12 → 1) over `duration-base`.
- Bars grow from zero on first paint; counters count up over `duration-count`; the auto-pause dot pulses every `duration-loop`.
- Scroll reveals use `animation-timeline: view()` where supported and simply show otherwise.
- `prefers-reduced-motion: reduce` turns everything off and renders the final state.

## Accessibility

The prototype was audited against WCAG 2.2 AA with scripted checks on every artboard (text contrast of every text node, control boundaries, target sizes, accessible names, landmarks, reflow at 320px) and fixed where it failed. The rules that keep it that way:

- **Contrast**: every text token on every surface is at least 4.5:1 (3:1 for 24px+ or 18.66px bold). `ink-4` is for placeholders and dark surfaces only. Bars and control borders meet 3:1 (`warn-bar`, `line-input`).
- **Not colour alone**: badges carry a word or icon; stat deltas carry an arrow; links inside running text are underlined; errors are sentences next to the field with `aria-invalid` and `aria-describedby`.
- **Focus**: `:focus-visible` draws a 2px `focus` ring with 2px offset on everything; it is never removed. Dialogs trap focus and return it on close.
- **Targets**: 44px on touch widths, 36px with a mouse, never under 24px for standalone controls; inline links in sentences are exempt.
- **Structure**: one `h1` per screen (visually hidden when the top bar shows a breadcrumb), `header`, `main`, `nav` with labels, `th scope="col"`, decorative SVGs `aria-hidden`, a skip link to `#main`.
- **Names**: every input, select and icon button has a label or `aria-label`; condition builders name field, operator and value per row.
- **Motion**: everything respects `prefers-reduced-motion`; auto-playing motion longer than five seconds (marquee, flow pulses) has a visible Pause control.
- **Reflow**: screens work at 320px with no horizontal page scroll; data tables scroll inside their card.
- **Status**: live counters and toasts use `role="status"` or `aria-live="polite"`; errors use `role="alert"`.
- **Auth**: no cognitive tests; passkeys and SSO offered; two-factor codes can be pasted; session timeouts warn two minutes ahead and keep unsaved work.

## Logo

An envelope whose flap is a check, on a 32-unit grid: teal `accent` square with radius 8, white 2.4-unit strokes with round caps. Files live under `assets/Logo` (mark, ink, white and mono versions, horizontal and stacked lockups, app icon, avatar, favicon). Clear space is half the mark's height; minimum sizes are 16px for the mark and 120px for the horizontal lockup. The wordmark is IBM Plex Sans Bold with -2 percent tracking; outline it to paths before sending files to print.

## Voice

English, plain, specific. Say what happened and what to do: "Outlook complaints hit 0.18% (limit 0.10%). 2,436 held, everything else still sending." Numbers before adjectives, provider names exact, no exclamation marks, no "oops".

## Responsive artboards

The prototype canvas carries desktop (1440), tablet (834) and phone (390) versions of the key screens. Tablet and phone boards reuse the desktop markup; only the CSS in this system changes, which is the point.
