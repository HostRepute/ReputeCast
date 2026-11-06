import { Fragment, type CSSProperties } from 'react'
import { Icon, type IconName } from '../components/Icon'
import { LogoMark } from '../components/Marks'
import { links } from '../content/links'

export const APP_LANGUAGE_NAMES = ['English', 'French', 'German', 'Spanish', 'Vietnamese'] as const

const languageList = `${APP_LANGUAGE_NAMES.slice(0, -1).join(', ')} and ${APP_LANGUAGE_NAMES.at(-1)}`

type Feature = { id: string; line: string; icon: IconName; title: string; summary: string; items: string[]; docs: string }

export const groups: { name: string; features: Feature[] }[] = [
  {
    name: 'Create',
    features: [
      {
        id: 'composer',
        line: 'Composer',
        icon: 'edit',
        title: 'Drag-and-drop composer',
        summary: 'Emails that render in Outlook, built on GrapesJS with MJML.',
        items: [
          'A Write mode for plain typing',
          'Device, client and dark-mode previews',
          'Personalization with fallbacks',
          'A/B variants',
          'A brand kit',
          'Inline deliverability and accessibility warnings',
          'AI suggestions through your own model key',
        ],
        docs: 'campaigns/composer/',
      },
      {
        id: 'automations',
        line: 'Automations',
        icon: 'automations',
        title: 'Visual automations',
        summary: 'Flows on React Flow, with live stats on every step.',
        items: ['Up to three triggers', 'Delays, emails, texts, push and WhatsApp steps', 'Branches and A/B splits', 'Webhooks, list actions and goals', 'Test sends', 'Versioned publishing'],
        docs: 'automations/overview/',
      },
      {
        id: 'forms',
        line: 'Forms and pages',
        icon: 'forms',
        title: 'Forms and public pages',
        summary: 'Signup forms, and the pages readers land on.',
        items: [
          'Embedded, popup and hosted forms',
          'Spam protection',
          "A double opt-in that link scanners can't confirm",
          'A preference center',
          'Unsubscribe with a "fewer emails" option',
          'A public archive',
        ],
        docs: 'forms/signup-forms/',
      },
      {
        id: 'website',
        line: 'Website and shop',
        icon: 'desktop',
        title: 'Website, pages and a shop',
        summary: 'Publish, gate and sell from the same place.',
        items: ['A publication website with posts and members', 'A meter and a paywall', 'Landing, sales and webinar pages', 'A shop for digital products'],
        docs: 'forms/website/',
      },
    ],
  },
  {
    name: 'Reach',
    features: [
      {
        id: 'transactional',
        line: 'Transactional email',
        icon: 'transactional',
        title: 'Transactional email',
        summary: 'Receipts, resets and alerts through your own providers.',
        items: [
          'Templates written in the composer',
          'A send, batch and render API with idempotency keys',
          'An SMTP relay',
          'A message log and delivery alerts',
          'A stream that fails over and goes before broadcasts',
        ],
        docs: 'transactional/overview/',
      },
      {
        id: 'channels',
        line: 'SMS, push, WhatsApp',
        icon: 'mobile',
        title: 'SMS, web push and WhatsApp',
        summary: 'Reach readers beyond the inbox.',
        items: [
          'SMS through Twilio, Vonage, Plivo, Telnyx or AWS',
          'Routing by country, consent per list, keywords and quiet hours',
          'Web push with a prompt designer',
          "WhatsApp through Meta's Cloud API",
        ],
        docs: 'channels/sms/',
      },
      {
        id: 'languages',
        line: 'Languages',
        icon: 'language',
        title: "In your readers' languages",
        summary: 'Every reader in their own language.',
        items: [
          'Language versions of campaigns, automation emails and transactional templates',
          'Forms and pages in each language',
          `The app itself in ${languageList}`,
        ],
        docs: 'campaigns/languages/',
      },
      {
        id: 'setup',
        line: 'Provider setup',
        icon: 'providers',
        title: 'Guided provider setup',
        summary: 'Most providers take 10–15 minutes, DNS included.',
        items: [
          'SES via CloudFormation, with SNS and configuration sets created for you',
          'API keys for the rest',
          'A bounce mailbox for SMTP',
          'DNS records with live verification',
          'A test send',
        ],
        docs: 'providers/connect/',
      },
    ],
  },
  {
    name: 'Grow',
    features: [
      {
        id: 'earn',
        line: 'Grow and earn',
        icon: 'earn',
        title: 'Grow and earn',
        summary: 'Grow the list, and get paid for it.',
        items: ['Referrals', 'Recommendations with partner links', 'Paid subscriptions through Stripe', 'Offers and gift subscriptions', 'Sponsorships', 'Revenue attribution'],
        docs: 'earn/overview/',
      },
      {
        id: 'analytics',
        line: 'Analytics',
        icon: 'barChart',
        title: 'Honest analytics',
        summary: 'Numbers that count people, not bots.',
        items: [
          'Human opens apart from Apple Mail prefetches',
          'Clicks checked against security scanners',
          'A/B winners decided by significance',
          'A click map per variant',
          "Bounces in one taxonomy, with the provider's original code",
          'Results per mailbox and per sending provider',
        ],
        docs: 'campaigns/reports/',
      },
      {
        id: 'segments',
        line: 'Segments',
        icon: 'filter',
        title: 'Segments without SQL',
        summary: 'Precise audiences without writing SQL.',
        items: ['A condition builder with AND/OR groups', 'A live count as you build', 'The generated SQL one click away'],
        docs: 'audience/segments/',
      },
      {
        id: 'health',
        line: 'List health',
        icon: 'audience',
        title: 'List health',
        summary: "Know who still reads you, and stop mailing who doesn't.",
        items: ['Active, Cooling, At risk and Sunset tiers', 'A sunset policy with a dry run', 'One suppression list shared by every provider'],
        docs: 'audience/list-health/',
      },
    ],
  },
  {
    name: 'Run',
    features: [
      {
        id: 'teams',
        line: 'Teams and agencies',
        icon: 'key',
        title: 'Teams and agencies',
        summary: 'Everyone gets the access they need, and no more.',
        items: [
          'Roles with granular permissions',
          'Send approvals',
          'OIDC SSO and passkeys',
          'An audit log',
          'Client workspaces for agencies, with staff assignments, client sign-off and usage per client',
        ],
        docs: 'team/roles-and-permissions/',
      },
      {
        id: 'api',
        line: 'API, MCP, integrations',
        icon: 'code',
        title: 'API, MCP and integrations',
        summary: 'Connect it to your stack, and to AI agents.',
        items: [
          'An OpenAPI 3.1 REST API with scoped keys',
          'Signed webhooks',
          'An MCP server whose agents pass the same permission checks as people',
          'Zapier, Make and n8n apps and a WordPress plugin',
          'Shopify and WooCommerce stores',
        ],
        docs: 'developers/overview/',
      },
      {
        id: 'privacy',
        line: 'Privacy',
        icon: 'shield',
        title: 'Privacy built in',
        summary: 'Erasure that reaches every table.',
        items: [
          'Erasing a contact reaches every table that holds their data',
          "A contact's export is complete",
          'Open and click details and unconfirmed signups expire on the schedule you set',
        ],
        docs: 'deliverability/compliance/',
      },
      {
        id: 'migration',
        line: 'Migration',
        icon: 'import',
        title: 'Migration that keeps consent',
        summary: "Move in without losing anyone's consent.",
        items: ['Import from Brevo, Mailchimp, MailerLite, listmonk, Keila, CSV or XLSX', 'Unsubscribed and bounced contacts stay suppressed, forever'],
        docs: 'migrate/overview/',
      },
    ],
  },
]

/**
 * A $0 receipt. Its lines are radio buttons and CSS opens the picked one's details, so it works without
 * JavaScript; --c/--r place a line in its group's column, --c2/--r2 on tablets.
 */
export function AllFeatures() {
  const count = groups.reduce((n, g) => n + g.features.length, 0)
  const first = groups[0]?.features[0]?.id
  const at = (g: number, row: number) => ({ '--c': g + 1, '--r': row, '--c2': (g % 2) + 1, '--r2': row + (g < 2 ? 0 : 5) }) as CSSProperties
  return (
    <section className="wrap flex flex-col gap-6 pt-10 pb-10" aria-labelledby="all-features-h">
      <div className="reveal flex flex-col gap-2.5">
        <h2 className="sec" id="all-features-h">
          Everything a newsletter tool should have. All of it free.
        </h2>
        <p className="lead">No paid tier, no feature flags behind a licence key. If it&apos;s in the product, it&apos;s yours.</p>
      </div>
      <fieldset className="fx-set reveal">
        <legend className="vh">Pick a feature to see what it includes</legend>
        <div className="fx">
          <div className="fx-top">
            <span className="fx-brand">
              <LogoMark size={20} />
              ReputeCast
            </span>
            <span className="fx-hint" aria-hidden="true">
              Pick a line to see what&apos;s in it
            </span>
          </div>
          {groups.map((g, gi) => (
            <Fragment key={g.name}>
              <p className="fx-group" style={at(gi, 2)}>
                {g.name}
              </p>
              {g.features.map((f, fi) => (
                <Fragment key={f.id}>
                  <label htmlFor={`fx-${f.id}`} className="fx-row" style={at(gi, fi + 3)}>
                    <input type="radio" name="feature" id={`fx-${f.id}`} value={f.id} defaultChecked={f.id === first} />
                    <Icon name={f.icon} />
                    <span className="fx-name">{f.line}</span>
                    <span className="fx-dots" aria-hidden="true" />
                    <span className="fx-price" aria-hidden="true">
                      $0
                    </span>
                  </label>
                  <div className="fx-detail" style={at(gi, 0)}>
                    <div className="fx-about">
                      <div className="fx-kind">
                        <span className="fx-icon">
                          <Icon name={f.icon} size={22} />
                        </span>
                        {g.name}
                      </div>
                      <h3>{f.title}</h3>
                      <p className="fx-summary">{f.summary}</p>
                      <div className="fx-foot">
                        <span className="badge ok">
                          <span className="dot" />
                          Included, $0
                        </span>
                        <a href={`${links.docs}${f.docs}`} aria-label={`Read the docs: ${f.title}`}>
                          Read the docs
                          <Icon name="arrowRight" size={14} />
                        </a>
                      </div>
                    </div>
                    <ul className="fx-items">
                      {f.items.map((t) => (
                        <li key={t}>
                          <i />
                          <span>{t}</span>
                        </li>
                      ))}
                    </ul>
                  </div>
                </Fragment>
              ))}
            </Fragment>
          ))}
          <div className="fx-total">
            <span>Total</span>
            <b>$0.00</b>
            <span className="fx-note">
              A month, for all {count}. No paid tier, no licence key: AGPL-3.0.
            </span>
          </div>
        </div>
      </fieldset>
    </section>
  )
}
