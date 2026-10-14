const features = [
  {
    title: 'Drag-and-drop composer',
    text: 'Built on GrapesJS with MJML, so emails render in Outlook. Device, client and dark-mode previews, personalization with fallbacks, A/B variants, inline deliverability warnings.',
  },
  {
    title: 'Visual automations',
    text: 'Flows on React Flow: triggers, delays, emails, branches, A/B splits, webhooks, list actions and goals, with live stats on every step and versioned publishing.',
  },
  {
    title: 'Honest analytics',
    text: "Human opens separated from Apple Mail prefetches, a click map, bounces mapped to one taxonomy with the provider's original code, results per mailbox and per sending provider.",
  },
  {
    title: 'List health',
    text: 'Active, Cooling, At risk and Sunset tiers, a sunset policy, and one suppression list shared by every provider.',
  },
  {
    title: '10-minute provider setup',
    text: 'SES via CloudFormation with SNS and configuration sets created for you, API keys for the rest, a bounce mailbox for SMTP, DNS records with live verification, a test send.',
  },
  {
    title: 'Migration that keeps consent',
    text: 'Import from Brevo, Mailchimp, MailerLite, listmonk, Keila or CSV. Unsubscribed and bounced contacts stay suppressed, forever.',
  },
  {
    title: 'Segments without SQL',
    text: 'A condition builder with AND/OR groups and a live count. The generated SQL is one click away for people who want it.',
  },
  {
    title: 'Forms and public pages',
    text: 'Embedded, popup and hosted forms with double opt-in and spam protection; preference center, unsubscribe with a "fewer emails" option, public archive.',
  },
  {
    title: 'Teams, API, MCP',
    text: 'Roles with granular permissions, OIDC SSO, passkeys, scoped API keys, signed webhooks, an audit log, and an MCP server so AI agents can draft and report with approvals.',
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
