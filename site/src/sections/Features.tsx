const pillars = [
  {
    badge: 'Before you send',
    title: 'Send Readiness',
    text: "Blocklists across 80+ sources, SPF/DKIM/DMARC/BIMI against each provider's required records, spam triggers, Gmail's clipping limit, broken and blocklisted links, one-click unsubscribe, audience quality and routing health. Every check is Pass, Warn or Block, with a fix.",
  },
  {
    badge: 'While you send',
    title: 'Guarded sending',
    text: 'Live delivered, bounced and complained counts per mailbox provider and per sending provider. Bounces or complaints over your thresholds pause the campaign, or just one route, and ReputeCast explains what happened and offers resume, adjust, fail over or cancel.',
  },
  {
    badge: 'Any provider, several at once',
    title: 'Pluggable sending providers',
    text: 'Built-in adapters for SES, Mailgun, SendGrid, Postmark, Resend, SparkPost, Mailjet, Brevo, Azure, Postal and SMTP. Weighted splits for warm-up and migration, automatic failover, one shared suppression list, and an adapter SDK for anything else.',
  },
]

/** "Nothing else checks deliverability before and during a send." */
export function Features() {
  return (
    <section className="wrap flex flex-col gap-7 pt-18 pb-6" id="features" aria-labelledby="features-h">
      <div className="reveal flex flex-col gap-2.5">
        <h2 className="sec" id="features-h">
          Nothing else checks deliverability before and during a send.
        </h2>
        <p className="lead">Every newsletter tool sends. ReputeCast sends and keeps you in the inbox.</p>
      </div>
      <div className="grid grid-cols-[repeat(auto-fit,minmax(min(300px,100%),1fr))] gap-4">
        {pillars.map((f) => (
          <div key={f.title} className="feat hi reveal">
            <span className="badge ok self-start">{f.badge}</span>
            <h3>{f.title}</h3>
            <p>{f.text}</p>
          </div>
        ))}
      </div>
    </section>
  )
}
