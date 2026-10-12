# ReputeCast — Add-on spec: accounts, teams, access and send approvals

## 0. How to use this document

Read all of it before you start.

- **If the ReputeCast prototype from the main spec already exists in this project:**
  - Add the screens in section 5 to it.
  - Apply the changes in section 6 to the existing screens.
  - Reuse the existing navigation, components and design system.
- **If it does not exist yet:** read the main spec (ReputeCast — Complete product spec) first and build both together.
- Same rules as the main spec:
  - Include empty, loading, error and success states, plus the specific states listed for each screen.
  - All copy in English.
  - Use the sample data in section 4 together with the main spec's sample data.
- **Build order:**
  1. 5.1 Sign in
  2. 5.8 Team
  3. 5.9 Roles & permissions
  4. 5.12 Send approvals
  5. 5.11 SSO setup
  6. 5.15 Connect HostRepute
  7. The rest, in order
  8. Then section 6

  If the work needs several passes, keep going until every screen exists.

---

## 1. Principles

1. **Everything here is free.** Every feature in this document is part of the free, open-source edition. Never show a paywall, upgrade prompt, lock icon, "Pro" or "Enterprise" badge, seat limit or workspace limit. The only paid products in the ReputeCast world are HostRepute and the hosted ReputeCast Cloud, and neither one gates a feature in the app.
2. **Self-hosted first.** One installation (an "instance") belongs to the user's company. When the first people sign in, a sending provider may not exist yet. So every flow that normally sends an email must also work without one:
   - Invites: a copyable invite link.
   - Password reset and 2FA reset: a CLI command.
3. **Access control protects sender reputation.** These are separate permissions:
   - who can edit a campaign
   - who can send it
   - who can stop a live send
   - who can touch provider credentials
   - who can export contacts

   Large or risky sends can require a second person's approval.
4. **Never lock everyone out.**
   - Owners keep a break-glass sign-in (password + 2FA), even when SSO is enforced.
   - The last Owner of a workspace can't be removed or demoted.
5. **No privilege escalation.** Nobody can grant a permission they don't hold, whether to a role, a member, an API key or an MCP agent.
6. **Write-only secrets.**
   - Provider credentials, OIDC client secrets and API keys can be replaced, but never read back after saving.
   - New API keys are shown exactly once.
7. **Explain missing permissions.** When the current member can't do something, hide the action or disable it with the reason and who can help. Example: "Sending needs the campaigns:send permission. Ask Marcus Chen or Linh Nguyen."

---

## 2. Terms

- **Instance:** one ReputeCast installation. Instance admins manage users, sign-in methods, SSO, security policies and workspaces for the whole instance.
- **Workspace:** an isolated space with its own lists, campaigns, automations, sending providers and team. Unlimited.
- **Member:** a user's membership in a workspace. Each member has exactly one role per workspace.
- **Role:** a named set of permissions. The built-in roles are Owner, Admin, Editor and Viewer. Custom roles are allowed.
- **Scope:** an optional restriction of a member's role to specific lists or sender domains. Example: "Editor, Beta Program list only".
- **Social sign-in:** Google, Microsoft or GitHub sign-in for people who were already invited. It never creates accounts on its own.
- **SSO:** an organization identity provider (IdP) connected through OpenID Connect. SSO can create accounts automatically, map groups to roles, and be enforced for an email domain.

---

## 3. Requirements

### 3.1 Sign-in methods
- **Email and password:**
  - Minimum length 12 by default.
  - Optional breached-password check, which can be turned off for offline installs.
- **Passkeys:** they work as a single sign-in step, and as a second factor.
- **Authenticator app (TOTP) for 2FA,** with 10 one-time recovery codes.
- **Social sign-in:** Google, Microsoft and GitHub. Each can be turned on separately. GitHub sign-in can be restricted to members of a GitHub organization.
- **SSO over OpenID Connect,** with presets for Google Workspace, Microsoft Entra ID, Okta, Keycloak and Authentik, plus generic OIDC.
- **Out of scope:** SAML and SCIM are not part of this version. Do not design them.

### 3.2 SSO
- **Configured at instance level.** Each IdP has its own:
  - issuer URL, with discovery
  - client ID and client secret
  - scopes
  - claim mapping: email, email_verified, name, groups
- **Allowed email domains.**
- **Just-in-time (JIT) account creation:** a default workspace and role for new users.
- **Group-to-role mapping:**
  - Each mapping is IdP group → workspace → role, with optional scope.
  - Mappings are re-applied at every sign-in.
  - Group mapping only works when the IdP sends a groups claim (Entra ID, Okta, Keycloak, Authentik). For Google Workspace, map by email domain or by individual user instead.
  - Entra ID sends group object IDs by default, so let admins paste an ID and give it a readable label.
- **Domain enforcement:** password sign-in is turned off for users of a domain (e.g. acme.io), except break-glass Owners.
- **Test sign-in:** shows the exact claims the IdP returned and which roles they would produce, without signing anyone in.

### 3.3 Security policies (instance)
- **Allowed sign-in methods.**
- **2FA enforcement:** for everyone, or only for members who hold a sensitive permission (see 3.7). Includes a grace period.
- **Sessions:** lifetime and idle timeout.
- **Lockout:** sign-in rate limiting and temporary lockout.
- **Email domain allowlist** for JIT accounts.

### 3.4 Workspaces
- Unlimited workspaces, with a workspace switcher.
- Create a workspace.
- Workspace settings: name, default sender, timezone.
- Transfer ownership.
- Delete a workspace: typed confirmation, with an offer to export everything first.

### 3.5 Team
- **Invite members:**
  - By email, or by copyable link.
  - Set a role, an optional scope and an expiry (default 7 days).
- **Manage members:** change role or scope, deactivate, remove.
- **Deactivating a member:**
  - Keeps their history.
  - Signs them out everywhere.
  - Asks whether to keep or revoke the API keys they created.

### 3.6 Roles
- **Built-in roles:**
  - **Owner:** everything, including transferring ownership and deleting the workspace.
  - **Admin:** everything except transferring ownership and deleting the workspace.
  - **Editor:** view everything, plus:
    - edit campaigns, templates, automations, segments (without SQL) and forms
    - send, pause and publish
    - import contacts

    Editors cannot manage providers, export contacts or manage the team.
  - **Viewer:** read-only. Never sees secrets.
- **Built-in roles are read-only.** Duplicate one to customize it.
- **Custom roles:** any combination of the permissions in 3.7.
- **Permission dependencies:** a permission includes the matching view permission (e.g. edit includes view).

### 3.7 Permissions
The UI shows plain-language names. The keys in brackets are also used for API keys and MCP agents. Sensitive permissions are marked with ⚑ and should be clearly flagged in the UI.

- **Campaigns:**
  - view (campaigns:view)
  - edit (campaigns:edit)
  - send or schedule (campaigns:send) ⚑
  - pause or cancel a live send (campaigns:pause)
  - approve (campaigns:approve) ⚑

  Anyone with campaigns:pause can stop a live send. Resuming a send requires campaigns:send.
- **Templates:**
  - view (templates:view)
  - edit (templates:edit)
- **Automations:**
  - view (automations:view)
  - edit (automations:edit)
  - publish (automations:publish) ⚑
- **Contacts:**
  - view (subscribers:view)
  - create, as used by signup forms and the API (subscribers:create)
  - edit (subscribers:edit)
  - import (subscribers:import)
  - export (subscribers:export) ⚑
  - erase personal data (subscribers:delete) ⚑
- **Lists and segments:**
  - manage lists (lists:manage)
  - edit segments (segments:edit)
  - write SQL segments (segments:sql) ⚑
- **Forms and pages:**
  - edit (forms:edit)
- **Reports:**
  - view (reports:view)
  - create public read-only links (reports:share)
- **Deliverability:**
  - view (deliverability:view)
  - connect HostRepute (integrations:hostrepute)
- **Sending:**
  - view providers (providers:view)
  - manage providers, credentials, routing and failover (providers:manage) ⚑
  - manage domains and senders (domains:manage) ⚑
- **Team:**
  - view (team:view)
  - invite (team:invite)
  - manage roles and remove members (team:manage) ⚑
- **Developer:**
  - API keys (apikeys:manage) ⚑
  - webhooks (webhooks:manage)
  - MCP agents (mcp:manage) ⚑
- **Audit:**
  - view the audit log (audit:view)
- **Workspace:**
  - manage settings (workspace:manage)

### 3.8 Send approvals
- **Approval policy (per workspace).** Approval is required when any enabled rule matches:
  - a Broadcast audience of at least N recipients
  - the route includes a provider that is Warming up, or that was connected in the last 14 days
  - Send Readiness is "Ready with warnings"
  - the author lacks campaigns:send
  - the audience includes a list marked "Requires approval"

  Optionally, publishing an automation can also require approval.
- **Approvers:** members with campaigns:approve.
  - 1 or 2 approvals required.
  - Authors can't approve their own campaign.
- **What an approval covers:** a snapshot of the campaign:
  - content and subject
  - audience definition
  - sending route
  - schedule

  The approval is invalidated if any of these change, or if the audience count moves by more than 10%. The approver then sees exactly what changed.
- **Recheck at send time.** Send Readiness runs again when the send starts. If the result is Blocked, the send doesn't start and both author and approvers are notified.
- **Emergency bypass.** Owners can send without approval by entering a reason. The bypass is logged, and every approver is notified.

### 3.9 API keys and MCP agents
- **Scoped permissions:** both use the permission keys from 3.7, with an optional list scope. They can never exceed the creator's own permissions.
- **Lifecycle:**
  - optional expiry
  - shown once at creation
  - last-used time and IP
  - rotate and revoke
- **MCP agents** additionally show their recent actions.

### 3.10 Audit log
- **What it records:**
  - who: a user, API key, MCP agent or the system
  - did what, to what, and in which workspace
  - from which IP and user agent
  - when
  - whether the action was allowed or denied
- **Details:** before/after values for every change.
- **Entries are immutable:** they can't be edited or deleted from the UI.
- **Filters, export and retention:**
  - filter by actor, action, target and result
  - export to CSV or JSON
  - stream to a webhook
  - retention setting (default 365 days)
- **Two levels:** instance-level and workspace-level views.

### 3.11 Connect HostRepute
- **OAuth connection:** "Connect with HostRepute" opens HostRepute's consent page on app.hostrepute.com and returns to ReputeCast. Pasting a HostRepute API key is the alternative.
- **Data shared:**
  - Sent from ReputeCast: sending domains and dedicated IPs to monitor.
  - Received by ReputeCast: blacklist status across 80+ sources, Google Postmaster Tools, Microsoft SNDS, DMARC reports and alerts.
  - Never shared: contacts and email content.
- **After connecting:**
  - monitored assets with sync status
  - "Add to monitoring" for assets not yet monitored
  - granted scopes
  - HostRepute alerts delivered into ReputeCast notifications
  - Disconnect, with a clear list of what stops working

### 3.12 Notifications
- **In-app notification center.** It covers:
  - approval requested, approved, rejected or invalidated
  - send auto-paused
  - provider down
  - webhooks silent
  - invite accepted
  - HostRepute alerts
- **Per-member preferences:** in-app and/or email for each event type.

### 3.13 Instance updates
- **Version & updates page:** current version, latest release, changelog and upgrade instructions (docker compose pull).
- **Update check:** it contacts GitHub releases and can be turned off. ReputeCast never sends usage data anywhere.

---

## 4. Sample data

### Instance
- **Name and URL:** "Acme ReputeCast" at https://reputecast.acme.io.
- **Version:** v1.4.2. Update v1.5.0 is available.

### Workspaces
- **Acme Newsletter:** the workspace from the main spec. 48,213 subscribers, 6 members.
- **Acme Engineering Blog:** 6,420 subscribers, 3 members.

### Members of Acme Newsletter
| Name | Email | Role | Sign-in methods | 2FA | Last active |
|---|---|---|---|---|---|
| Linh Nguyen | linh@acme.io | Owner, instance admin, break-glass | password, passkey | TOTP | now |
| Marcus Chen | marcus@acme.io | Admin | Google Workspace SSO | via IdP | 12 min ago |
| Sofia Rossi | sofia@acme.io | Editor | Google Workspace SSO | via IdP | 2 h ago |
| Daniel Park | daniel@acme.io | Content Writer (custom role) | Google Workspace SSO | via IdP | 25 min ago |
| Aisha Bello | aisha@acme.io | Viewer | password | off, grace period ends in 3 days | yesterday |
| Tom Becker | tom@acme.io | Admin | (deactivated) | | deactivated Sep 30, 2026 |

The Content Writer custom role can edit campaigns and templates, view reports and request approval. It cannot send, export contacts or touch providers.

### Pending invite
- **Invitee:** jo@contractor.dev, invited as Editor scoped to the Beta Program list.
- **Status:** expires in 5 days. Invited by Linh Nguyen; the email was sent via Postmark.

### SSO
- **Google Workspace (acme.io): enabled.**
  - JIT is on: new users join Acme Newsletter as Viewer.
  - Password sign-in is turned off for @acme.io, except the break-glass Owner Linh Nguyen.
- **Microsoft Entra ID: setup in progress.** It is used to show group mapping:
  - group "Marketing" (object ID 3f2a…c91) → Acme Newsletter, Editor
  - group "Eng Leads" (object ID 8b7d…02e) → Acme Engineering Blog, Admin
- **GitHub sign-in: enabled,** restricted to members of the GitHub organization "acme-eng".

### Security policies
- **2FA:** required for members holding any ⚑ permission, with a 7-day grace period.
- **Sessions:** 14-day lifetime, 12-hour idle timeout.
- **Lockout:** after 10 failed attempts, sign-in is locked for 15 minutes.

### Approval policy (Acme Newsletter)
- **Rules:**
  - audience ≥ 10,000 recipients
  - route includes a Warming-up provider
  - author lacks campaigns:send
- **Approvals:** 1 approval from Owner or Admin.

### Pending approval
- **Campaign:** "November Product Update" by Daniel Park, requested 2 hours ago.
- **Audience:** 47,912 recipients.
- **Route:** SES 43,121, Postal 4,791 (Warming up).
- **Send Readiness:** Ready with warnings.
  - 1 link redirects through a URL shortener.
  - 3.1% of the audience has not engaged in more than 180 days.
- **Estimated cost:** SES $4.31; Postal $0, because it is self-hosted.
- **Scheduled:** Nov 4, 2026 at 09:00, America/New_York.
- **Rules matched:** all three.
- **Approvers:** Marcus Chen or Linh Nguyen.

### API keys
| Key | Permissions | Scope | Last used | Expiry |
|---|---|---|---|---|
| Website signup form | subscribers:create | Weekly Digest only | 3 min ago | none |
| Data warehouse sync | reports:view, subscribers:view | all lists | yesterday | Jan 1, 2027 |

### MCP agents
- **"Claude — analytics":** reports:view and deliverability:view.
- **"Ops agent":** campaigns:pause and providers:view. It can stop a send but never start one.

### HostRepute
- **Account:** connected to the HostRepute team "Acme Inc".
- **Monitored:** news.acme.io, acme.io and 203.0.113.25.
- **Not yet monitored:** mta.acme.io.
- **Last sync:** 4 min ago.

### Recent audit events
- Marcus Chen changed Broadcast routing from SES 95% / Postal 5% to SES 90% / Postal 10%.
- Linh Nguyen created API key "Website signup form".
- Daniel Park requested approval for "November Product Update".
- Denied: Aisha Bello tried to export contacts (missing subscribers:export).
- Sign-in rejected for kim@gmail.com: email domain not allowed for Google Workspace SSO.
- MCP agent "Claude — analytics" read the report for "October Product Update".
- System auto-paused "October Product Update": Outlook complaint rate reached 0.18%.

---

## 5. New screens

### 5.1 Sign in
- **Branding:** the instance name and logo, set by the instance admin.
- **Identifier-first flow:** the user enters an email first.
  - If the email's domain enforces SSO, they go straight to the IdP.
  - Otherwise they see the password field.
- **Other options:**
  - "Sign in with a passkey", which needs no email
  - social sign-in buttons for the providers that are enabled
  - "Forgot password"

States to show:
- default sign-in
- SSO redirect for linh@acme.io's domain, with the break-glass "Sign in with password" link for Owners
- wrong password (a generic message that doesn't reveal whether the account exists)
- too many attempts (locked for 15 minutes)
- account deactivated
- IdP returned an error, with a reference ID the admin can look up in the audit log
- "No account for this email. Ask an admin to invite you." (JIT off or domain not allowed)

### 5.2 Two-factor challenge
- **Methods:** authenticator code, "Use a passkey instead" and "Use a recovery code".
- **Recovery code use:** using a recovery code shows how many remain.

States to show:
- wrong code
- the last recovery code used, prompting the user to generate new ones

### 5.3 Password reset
- **Normal flow:** a reset email via the System stream.
- **No System provider connected yet:** explain that an instance admin can reset the password from the server with:

  ```
  reputecast user reset-password --email aisha@acme.io
  ```

- **SSO-only users:** they are told their password is managed by their IdP.

### 5.4 Accept invitation
- **What the invitee sees:** workspace, inviter, role and scope.
- **Joining:** set a password or continue with SSO or social sign-in, then an offer to add a passkey and set up 2FA. If policy requires 2FA, the grace period is shown.

States to show:
- expired invite
- invite revoked
- invite for a different email than the account that is signed in

### 5.5 First sign-in and instance setup
This is the first sign-in of the Owner created on the server by `reputecast install`, using a temporary password printed in the terminal or set via environment variables.

1. Replace the temporary password.
2. Add a passkey or an authenticator app. This step is required for the first Owner.
3. Save the recovery codes.
4. Instance basics:
   - instance name
   - public URL. It is used for provider webhooks, SSO redirects and public pages. Check its DNS and TLS and show the result.
   - default timezone
   - turn the update check on or off
5. Create the first workspace.
6. Continue into "Connect a sending provider" (main spec 7.2).

### 5.6 Account & security (personal)
- **Profile:** name, avatar, timezone, email change (with verification).
- **Password:** change it. This section is hidden for SSO-only users.
- **Passkeys:** add, rename, remove. Show when each was created and last used.
- **Authenticator app:** set up with a QR code and a manual key, then confirm with a code. Recovery codes are shown once, with download and print. Regenerate codes.
- **Connected sign-ins:** Google, Microsoft, GitHub and SSO identities. Link and unlink. The last remaining sign-in method can't be unlinked.
- **Active sessions:** device, browser, IP, approximate location and last active. The current session is marked. Sign out one session, or all others.
- **My workspaces:** each workspace with my role, and "Leave workspace".
- **Notification preferences.**

### 5.7 Workspace switcher and workspace settings
- **Switcher:** lists every workspace with my role, plus "Create workspace".
- **Settings:** name, default sender, timezone and the approval policy (5.12).
- **Danger zone:**
  - transfer ownership
  - delete workspace, with a typed confirmation and an "Export everything first" offer

### 5.8 Team (members and invitations)
- **Members list:**
  - name and email
  - role, plus scope
  - sign-in methods
  - 2FA status: On, Off, Via IdP, or "Required, grace ends in N days"
  - last active
  - status: Active, Invited or Deactivated
  - filters and bulk role change
- **Invite flow:**
  - multiple emails
  - role
  - optional scope (lists, sender domains)
  - expiry
  - personal message
- **Invite fallbacks and hints:**
  - If no System-stream provider exists: "Email can't be sent yet. Copy the invite links."
  - If SSO JIT covers the email's domain: "acme.io users can also join automatically with Google Workspace."
- **Pending invites:** resend, copy link, revoke.
- **Member detail:**
  - effective permissions, including where each one comes from: role, SSO group mapping or scope
  - recent activity, from the audit log
  - API keys they created
  - approvals they gave
- **Deactivate:** shows the consequences, and asks whether to keep or revoke the member's API keys.

States to show:
- Aisha (Viewer) opening Team: read-only, no invite button, with the reason
- trying to demote the last Owner
- jo@contractor.dev pending

### 5.9 Roles & permissions
- **Roles list:** member count per role. Built-in roles are read-only, with "Duplicate to customize".
- **Role editor:**
  - name and description
  - permissions grouped by area, with plain-language descriptions
  - ⚑ marks on sensitive permissions
  - automatic dependencies (edit includes view)
- **Comparison:** compare roles side by side.
- **Impact preview before saving:** e.g. "4 members and 1 API key will lose campaigns:send."
- **Escalation guard:** a permission you don't hold can't be added; explain why.

States to show:
- creating the "Content Writer" role
- the comparison view
- the escalation guard

### 5.10 Instance admin
This area is only visible to instance admins.

- **Users:** all accounts across workspaces.
  - Columns: workspaces, instance-admin flag, sign-in methods, 2FA, last sign-in, status.
  - Actions:
    - deactivate
    - force sign-out
    - reset 2FA (requires a reason, which is logged)
    - make instance admin
    - send a password reset, or show the CLI command
- **Workspaces:** member count, subscribers, providers and created date. Create and delete.
- **Authentication:**
  - allowed sign-in methods
  - social sign-in providers
  - SSO providers (link to 5.11)
  - password policy, including the breached-password check toggle
  - 2FA enforcement and grace period
  - session lifetime and idle timeout
  - lockout
  - email domain allowlist
- **Instance settings:** name, logo, public URL (with its check), default timezone.
- **Version & updates:** current version, latest release, changelog, upgrade command, update-check toggle and the "no usage data" statement.

### 5.11 SSO provider setup
1. **Pick a preset:** Google Workspace, Microsoft Entra ID, Okta, Keycloak, Authentik or generic OIDC.
2. **Connection:**
   - Copy the redirect URI, which is built from the public URL.
   - Enter the issuer URL, then "Test discovery".
   - Enter the client ID and client secret (write-only).
   - Set the scopes.
3. **Claim mapping:** email, email_verified, name, groups (with a configurable claim name).
4. **Access:**
   - allowed email domains
   - JIT on or off, with the default workspace and role
   - group → workspace → role (+ scope) mappings
   - domain enforcement, with the break-glass Owners listed
5. **Test sign-in:** a debug view of the returned claims, plus the resulting accounts and roles.

States to show:
- Google Workspace enabled, with a note that group mapping is unavailable for Google Workspace
- Entra ID setup with group object IDs and labels
- discovery failed (issuer unreachable or wrong URL)
- groups claim missing from the test sign-in
- turning on domain enforcement, which confirms who keeps password access

### 5.12 Send approvals
- **Policy (in workspace settings):** the rules from 3.8, toggled on or off, with N editable. Choose approvers and the number of approvals required. Optionally require approval for automation publishing. Owners can use emergency bypass.
- **Author's side (Review & send):**
  - If any rule matches, "Send" becomes "Request approval".
  - Show which rules matched.
  - The author can add a note for approvers.
- **Approval review (for the approver):**
  - campaign preview: subject, preheader, content
  - Send Readiness result with evidence
  - audience count and definition
  - route breakdown and estimated cost per provider
  - schedule
  - author's note
  - what changed since the last approved version, if any
  - comment thread
  - actions: Approve, Request changes, Reject
- **Approval inbox:** pending, approved, changes requested, rejected and invalidated, with filters.

States to show:
- Marcus reviewing "November Product Update"
- approved, then invalidated because Daniel edited the subject (with the diff)
- the send-time recheck turning Blocked after approval
- Linh using the emergency bypass, with the reason required

### 5.13 API keys & MCP agents
This screen replaces the version in the main spec's Settings.

- **Create:** name, permissions (picked from the 3.7 keys), optional list scope, optional expiry. The key is shown once, with copy.
- **List:** permissions, scope, creator, last used (time and IP), expiry. Rotate and revoke.
- **MCP agents:** the same controls, plus recent agent actions.

States to show:
- the one-time key display
- an expired key
- trying to grant a permission the creator doesn't hold

### 5.14 Audit log
- **Event list:** filters for actor type, actor, action, target, workspace, result and date range.
- **Event detail:** before/after values, IP, user agent and request ID.
- **Export and streaming:** export to CSV or JSON, stream to a webhook, retention setting.
- **Two views:** workspace-level and instance-level.

States to show:
- the sample events
- the detail of the routing change
- a denied event

### 5.15 Connect HostRepute
Entry points:
- the Deliverability "not connected" state
- Settings › Integrations
- an optional card on the Overview

The flow:
1. **Before connecting:**
   - what HostRepute adds
   - exactly what data is shared and what is never shared
   - "Connect with HostRepute" (OAuth), or "Use an API key instead"
2. **OAuth:** the consent step happens on app.hostrepute.com. Show only a simple "Redirecting to HostRepute…" handoff and the return.
3. **Connected:**
   - HostRepute account and team
   - granted scopes
   - monitored domains and IPs with sync status
   - assets not yet monitored, with "Add to monitoring"
   - last sync
   - alerts routing into ReputeCast notifications
   - Disconnect, which lists what stops working

States to show:
- not connected
- connected (sample data)
- token revoked or expired → Reconnect
- HostRepute's own monitoring limit reached for this team. Show it honestly and link to HostRepute. ReputeCast itself stays fully usable.

### 5.16 Notification center
- **Notification list:** the events from 3.12, each linking to its screen. Unread and read, with "Mark all as read".
- **Preferences:** a shortcut to the member's notification preferences.

---

## 6. Changes to existing screens

- **App shell:** add a workspace switcher, the notification center and a user menu (Account & security, Sign out). Instance admins also see an entry to the Instance admin area.
- **Permission-aware screens:** apply principle 7 everywhere. Show one full example: Daniel Park (Content Writer) in the campaign composer, where "Review & send" leads to "Request approval".
- **Campaign composer (main spec 7.4):** personalization tokens use Liquid syntax. Replace the earlier example with `{{ first_name | default: "there" }}`.
- **Review & send (main spec 7.5):** add these states:
  - "Approval required" (the rules matched)
  - "Waiting for approval"
  - "Approved by Marcus Chen, 10:42"
  - "Approval invalidated: subject changed"
- **Live sending monitor (main spec 7.6):**
  - Show who started the send and who approved it.
  - Pausing needs campaigns:pause.
  - Resuming needs campaigns:send.
- **Sending providers & routing (main spec 7.3):**
  - Members with only providers:view see status and routing but never credentials.
  - Credential fields are write-only.
- **Deliverability (main spec 7.8):** the "HostRepute not connected" state leads into 5.15.
- **Audience (main spec 7.9):**
  - Export requires subscribers:export and is logged.
  - SQL segments require segments:sql.
- **Settings (main spec 7.13):** split into Workspace settings (general, team, roles, approvals, API keys & MCP, webhooks, integrations, audit log) and the Instance admin area (5.10).
- **Landing page (main spec section 8):** add team features to the honest comparison, using only these verified facts:
  - ReputeCast: users, roles, granular permissions, OIDC SSO, 2FA, passkeys and audit log are all free.
  - listmonk: multi-user roles, granular permissions and OIDC are free.
  - Notifuse self-hosted (v40+, BSL licence): granular permissions need a paid licence. OIDC SSO and audit logs need the $1,990/year Enterprise licence.
