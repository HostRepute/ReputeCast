export const APP_LANGUAGE_NAMES = ['English', 'French', 'German', 'Spanish', 'Vietnamese'] as const

const languageList = `${APP_LANGUAGE_NAMES.slice(0, -1).join(', ')} and ${APP_LANGUAGE_NAMES.at(-1)}`

const features = [
  {
    title: 'Drag-and-drop composer',
    text: 'Built on GrapesJS with MJML, so emails render in Outlook. A Write mode, device, client and dark-mode previews, personalization with fallbacks, A/B variants, a brand kit, inline deliverability and accessibility warnings, and AI suggestions through your own model key.',
  },
  {
    title: 'Visual automations',
    text: 'Flows on React Flow: up to three triggers, delays, emails, texts, push and WhatsApp steps, branches, A/B splits, webhooks, list actions and goals, with live stats on every step, test sends and versioned publishing.',
  },
  {
    title: 'Honest analytics',
    text: "Human opens apart from Apple Mail prefetches, clicks checked against security scanners, A/B winners decided by significance, a click map per variant, bounces in one taxonomy with the provider's original code, results per mailbox and per sending provider.",
  },
  {
    title: 'List health',
    text: 'Active, Cooling, At risk and Sunset tiers, a sunset policy with a dry run, and one suppression list shared by every provider.',
  },
  {
    title: 'Segments without SQL',
    text: 'A condition builder with AND/OR groups and a live count. The generated SQL is one click away for people who want it.',
  },
  {
    title: 'Forms and public pages',
    text: 'Embedded, popup and hosted forms with spam protection and a double opt-in that link scanners can\'t confirm; preference center, unsubscribe with a "fewer emails" option, public archive.',
  },
  {
    title: 'Transactional email',
    text: 'Templates written in the composer, a send, batch and render API with idempotency keys, an SMTP relay, a message log and delivery alerts, on a stream that fails over and goes before broadcasts.',
  },
  {
    title: 'SMS, web push and WhatsApp',
    text: "SMS through Twilio, Vonage, Plivo, Telnyx or AWS, with routing by country, consent per list, keywords and quiet hours; web push with a prompt designer; WhatsApp through Meta's Cloud API.",
  },
  {
    title: 'Grow and earn',
    text: 'Referrals, recommendations with partner links, paid subscriptions through Stripe, offers, gift subscriptions, sponsorships and revenue attribution.',
  },
  {
    title: 'Website, pages and a shop',
    text: 'A publication website with posts, members, a meter and a paywall; landing, sales and webinar pages; a shop for digital products.',
  },
  {
    title: "In your readers' languages",
    text: `Language versions of campaigns, automation emails and transactional templates, forms and pages in each language, and the app itself in ${languageList}.`,
  },
  {
    title: 'Guided provider setup',
    text: 'SES via CloudFormation with SNS and configuration sets created for you, API keys for the rest, a bounce mailbox for SMTP, DNS records with live verification, a test send. Most take 10–15 minutes.',
  },
  {
    title: 'Migration that keeps consent',
    text: 'Import from Brevo, Mailchimp, MailerLite, listmonk, Keila, CSV or XLSX. Unsubscribed and bounced contacts stay suppressed, forever.',
  },
  {
    title: 'Privacy built in',
    text: "Erasing a contact reaches every table that holds their data, a contact's export is complete, and open and click details and unconfirmed signups expire on the schedule you set.",
  },
  {
    title: 'Teams and agencies',
    text: 'Roles with granular permissions, send approvals, OIDC SSO, passkeys and an audit log. Agencies run client workspaces with staff assignments, client sign-off and usage per client.',
  },
  {
    title: 'API, MCP and integrations',
    text: 'An OpenAPI 3.1 REST API, scoped keys, signed webhooks, and an MCP server whose agents pass the same permission checks as people. Zapier, Make and n8n apps, a WordPress plugin, and Shopify and WooCommerce stores.',
  },
]

export function AllFeatures() {
  return (
    <section className="wrap flex flex-col gap-6 pt-10 pb-10" aria-labelledby="all-features-h">
      <div className="reveal flex flex-col gap-2.5">
        <h2 className="sec" id="all-features-h">
          Everything a newsletter tool should have. All of it free.
        </h2>
        <p className="lead">No paid tier, no feature flags behind a licence key. If it&apos;s in the product, it&apos;s yours.</p>
      </div>
      <div className="grid grid-cols-[repeat(auto-fit,minmax(min(260px,100%),1fr))] gap-3.5">
        {features.map((f) => (
          <div key={f.title} className="feat reveal">
            <h3>{f.title}</h3>
            <p>{f.text}</p>
          </div>
        ))}
      </div>
    </section>
  )
}
