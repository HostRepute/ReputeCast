Read CLAUDE.md first, then everything in docs/spec/ and design/. Don't write any code yet.

1. Make an inventory of the design: every screen and every state, mapped to the spec section it implements (e.g. "7.5 Review & send", "5.12 Send approvals"). List the design tokens and shared components you find. Tell me about any conflicts between the design and the spec.

2. Write docs/ROADMAP.md. It must contain:
   - Small milestones, each shippable and testable on its own, in this order:
     - M0 Skeleton:
       - repo layout, Makefile, docker-compose (Postgres + Mailpit)
       - goose, sqlc, River
       - chi + Huma with /healthz and OpenAPI
       - Vite app embedded with go:embed
       - openapi-typescript codegen
       - CI with lint + test
       - GoReleaser
     - M1 Install and first sign-in:
       - `reputecast install`
       - sign-in with password + TOTP + passkeys
       - sessions
       - Account & security
     - M2 Workspaces, team, invitations, roles and permissions (internal/authz), audit log
     - M3 Sending providers:
       - Provider interface and adapter contract tests
       - SES, SMTP and Postmark adapters
       - webhook ingest
       - shared suppression list
       - streams and routing
     - M4 Senders and domains: DNS checks, Send Readiness
     - M5 Audience: contacts, lists, custom fields, segments, importer
     - M6 Composer:
       - GrapesJS + MJML wrapper
       - Liquid tokens
       - template storage
     - M7 Review & send, send approvals, live sending monitor, auto-pause
     - M8 Campaign reports and honest analytics
     - M9 Automations with React Flow
     - M10 Forms and public pages
     - M11 SSO (OIDC), social sign-in, API keys, MCP server, webhooks
     - M12 Remaining adapters: Mailgun, SendGrid, Resend, SparkPost, Mailjet, Brevo, Azure, Postal, generic HTTP
     - M13 Deliverability, warm-up planner, Connect HostRepute
     - M14 Migration importers, landing page, docs, v1.0 release
   - For each milestone: its goal, the spec sections and design screens it covers, the DB tables, the API endpoints with their permission keys, the frontend routes, the tests, and acceptance criteria.
   - Adjust the order or split milestones if you find a better plan, but explain why.

3. Stop and show me the inventory and the roadmap. After I approve, start M0 on a new branch.
