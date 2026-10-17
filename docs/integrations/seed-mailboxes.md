# Seed mailboxes and the sign-in apps

Placement tests (Deliverability › Placement tests) send one copy of an email to each seed mailbox the workspace owns, then read where it landed. This page says how seeds connect, what ReputeCast reads, and what Google's and Microsoft's app rules mean for a self-hosted instance. The rules were checked in October 2026 (brief §8 conflict 4); check them again before relying on them.

## How a seed connects

| Mailbox provider | Connect with | How ReputeCast reads it |
|---|---|---|
| Gmail, Google Workspace | An IMAP app password, or OAuth through the instance's Google sign-in app | IMAP (XOAUTH2 for OAuth). Gmail's tabs come from its labels (`X-GM-RAW`). |
| Outlook.com, Microsoft 365 | OAuth through the instance's Microsoft sign-in app | Microsoft Graph, which tells Focused from Other. Over IMAP both show as Inbox. |
| Yahoo, AOL | An IMAP app password | IMAP |
| iCloud | An app-specific password (the Apple ID needs two-factor authentication) | IMAP |
| GMX, Web.de | The account password, with IMAP turned on in the mailbox's settings | IMAP |

- ReputeCast opens only the inbox, the spam folder and Gmail's tabs. Sent, Drafts, Archive and Trash are never read.
- It looks only for messages that carry a test token (in the Message-ID and the `X-ReputeCast-Test` header), and deletes copies by that token (Gmail: to its Trash).
- Passwords and OAuth tokens are sealed with the instance's secret key and are never returned by the API. Adding, reconnecting and removing a seed needs `placement:manage`, which people can hold and agents can't.
- Use mailboxes that exist only for tests: a seed's credentials can read the whole mailbox.

## The sign-in apps (Instance admin › Authentication)

Seeds reuse the instance's Google and Microsoft sign-in apps; there is no separate app to register. Each app needs a second redirect URI next to the sign-in one:

```
<public URL>/api/v1/placement/oauth/callback
```

The consent uses PKCE and a single-use state that names the workspace and the person who started it; the callback checks that person still holds `placement:manage`.

### Google

Add the scope `https://mail.google.com/` to the app's consent screen (IMAP with XOAUTH2 needs it). Google classes it as a restricted scope. What that means:

- **Apps in Testing.** While the consent screen's publishing status is Testing, Google issues refresh tokens that expire after 7 days, unless the app asks for nothing beyond name, email and profile. A seed connected this way signs in again every week; ReputeCast shows it as "Sign-in expired" and notifies everyone with `placement:manage`.
- **Unverified apps in production.** People see Google's unverified-app screen first, and the app is limited to 100 new users until it's verified.
- **Verification.** A restricted scope needs Google's restricted-scope review and, when the data passes through a server, a security assessment by a Google-approved assessor at least every 12 months. That is out of reach for most self-hosted instances.
- **Internal apps.** An app whose user type is Internal serves only the Google Workspace organization it belongs to and needs no verification, so a Google Workspace seed in that organization connects without these limits. A gmail.com seed can't use an Internal app.

So: connect gmail.com seeds with an app password (Google account › Security › 2-Step Verification › App passwords; 2-Step Verification must be on). Connect Google Workspace seeds with OAuth through an Internal app, or with an app password where the organization allows them. The add dialog recommends the app password for gmail.com.

### Microsoft

Register the app for accounts in any organizational directory and personal Microsoft accounts (Accounts: "Personal and work accounts", tenant `common`) so Outlook.com seeds can sign in, and add the delegated Microsoft Graph permissions `Mail.ReadWrite` and `offline_access` (with `openid` and `email`).

- **No passwords over IMAP.** Exchange Online turned Basic authentication off for IMAP and POP in every tenant from 1 October 2022, app passwords included, and it can't be turned back on. Microsoft did the same for Outlook.com personal accounts in September 2024. Outlook.com and Microsoft 365 seeds therefore connect with OAuth; the add dialog says so when the app password is picked for them.
- **Consent in other tenants.** Where a tenant has risk-based step-up consent, its users can't consent to a multitenant app registered after 8 November 2020 that asks for more than sign-in unless the publisher is verified (Microsoft's publisher verification program). Tenants that restrict user consent need an admin to consent as well. A Microsoft 365 seed in such a tenant needs its admin to approve the app once; Outlook.com seeds aren't affected.

## Other providers

- **Yahoo and AOL**: Account security › Generate app password.
- **iCloud**: appleid.apple.com › Sign-In and Security › App-Specific Passwords. Revoking the password, or changing the Apple ID's password, makes the sign-in fail (`AUTHENTICATIONFAILED`); ReputeCast marks the seed and skips it until it's reconnected.
- **GMX and Web.de**: turn on POP3/IMAP access in the mailbox's settings, then use the account password.

## Sources

- Google, [Using OAuth 2.0 to access Google APIs](https://developers.google.com/identity/protocols/oauth2) (refresh token expiration)
- Google Cloud Help, [Unverified apps](https://support.google.com/cloud/answer/7454865)
- Google, [Restricted scope verification](https://developers.google.com/identity/protocols/oauth2/production-readiness/restricted-scope-verification)
- Microsoft Learn, [Deprecation of Basic authentication in Exchange Online](https://learn.microsoft.com/en-us/exchange/clients-and-mobile-in-exchange-online/deprecation-of-basic-authentication-exchange-online)
- Microsoft Learn, [Publisher verification overview](https://learn.microsoft.com/en-us/entra/identity-platform/publisher-verification-overview)
