import { useRef, type ReactNode } from 'react'
import { useScrollable } from '../lib/scrollable'

/**
 * The comparison is with paid platforms only (product decision: ReputeCast
 * marketing never compares itself with other open-source projects).
 */
export const PLATFORMS = ['ReputeCast', 'Mailchimp', 'Brevo', 'Kit', 'MailerLite', 'beehiiv'] as const

const yes = (
  <>
    <span aria-hidden="true">✓</span>
    <span className="vh">Yes</span>
  </>
)
const no = (
  <>
    <span aria-hidden="true">–</span>
    <span className="vh">No</span>
  </>
)
const check = (text: string) => (
  <>
    <span aria-hidden="true">✓</span> {text}
  </>
)

type Cell = ReactNode | { c: ReactNode } | { k: ReactNode }
type Row = { label: string; cells: Cell[] }

const theirCloud = 'Their cloud'
const policy = 'Their sending policy applies to your account'
const exportable = 'Exportable; lives in their cloud'
const previews = 'Previews and test sends'

const rows: Row[] = [
  { label: 'Where it runs', cells: ['Your server, your Postgres. Open source, free.', theirCloud, theirCloud, theirCloud, theirCloud, theirCloud] },
  {
    label: 'Who sends the mail',
    cells: [
      'Your own providers: SES, Postmark, Mailgun, SendGrid, Resend, Postal or any SMTP',
      'Their shared pools; dedicated IP as a paid add-on',
      'Their infrastructure; dedicated IPs available',
      'Their infrastructure',
      'Their infrastructure',
      'Their infrastructure',
    ],
  },
  { label: 'Several providers at once, weighted splits, failover', cells: [{ c: yes }, { c: no }, { c: no }, { c: no }, { c: no }, { c: no }] },
  {
    label: 'Pre-send deliverability check',
    cells: [check('Send Readiness: blocklists, authentication, links, audience, routing, one verdict'), 'Previews and content checks', previews, previews, previews, previews],
  },
  {
    label: 'When bounces or complaints spike',
    cells: ['Pauses the affected route, explains why, offers resume, adjust, fail over or cancel', policy, policy, policy, policy, policy],
  },
  {
    label: 'Automations',
    cells: [{ c: check('visual flows') }, { c: check('Customer Journeys') }, { c: yes }, { c: check('visual automations') }, { c: yes }, { c: yes }],
  },
  { label: 'Segments', cells: [{ c: 'Builder + SQL' }, { c: 'Builder' }, { c: 'Builder' }, { c: 'Builder' }, { c: 'Builder' }, { c: 'Builder' }] },
  {
    label: 'Users, roles and granular permissions, OIDC SSO, 2FA, passkeys, audit log',
    cells: [{ c: 'All included, free' }, { c: 'By plan' }, { c: 'By plan' }, { c: 'By plan' }, { c: 'By plan' }, { c: 'By plan' }],
  },
  { label: 'Your data', cells: ['In your database; export or delete anything, any time', exportable, exportable, exportable, exportable, exportable] },
  {
    label: "Where it's better than ReputeCast",
    cells: [
      { k: 'Younger project; you run the server; no SMS or push' },
      { k: 'Huge template and integration library, nothing to host' },
      { k: 'Email, SMS, chat and CRM in one place' },
      { k: 'Creator tools: paid newsletters, recommendations, a creator network' },
      { k: 'Simple, affordable, polished editor' },
      { k: 'Built for growth: referrals, boosts, an ad network' },
    ],
  },
  {
    label: 'Typical monthly cost at 50,000 subscribers',
    cells: [
      { k: 'Your provider: about $5 on SES per 50,000 emails, plus a small server' },
      { k: 'About $350 and up on Standard, by contacts' },
      { k: 'About $70 to $250, by email volume' },
      { k: 'About $300, by subscribers' },
      { k: 'About $250 to $300, by subscribers' },
      { k: 'About $100 to $250 on Scale, by subscribers' },
    ],
  },
]

function renderCell(cell: Cell, i: number) {
  if (cell && typeof cell === 'object' && 'c' in cell) {
    return (
      <td key={i} className="c">
        {cell.c}
      </td>
    )
  }
  if (cell && typeof cell === 'object' && 'k' in cell) {
    return (
      <td key={i} className="k">
        {cell.k}
      </td>
    )
  }
  return <td key={i}>{cell as ReactNode}</td>
}

export function Compare() {
  const scroller = useRef<HTMLDivElement>(null)
  const scrollable = useScrollable(scroller)
  return (
    <section className="wrap flex flex-col gap-5 pt-10 pb-10" id="compare" aria-labelledby="compare-h">
      <div className="reveal flex flex-col gap-2.5">
        <h2 className="sec" id="compare-h">
          An honest comparison
        </h2>
        <p className="lead">
          These are good products. Here&apos;s where each one shines and where ReputeCast is different. List prices change often, so check each provider; corrections
          welcome on GitHub.
        </p>
      </div>
      <div className="card reveal overflow-hidden">
        {/* relative: the visually hidden labels in cells stay inside the
            scroller instead of widening the page on phones. */}
        <div
          ref={scroller}
          className="relative overflow-x-auto"
          tabIndex={scrollable ? 0 : undefined}
          role={scrollable ? 'region' : undefined}
          aria-label={scrollable ? 'Comparison table, scrolls sideways' : undefined}
        >
          <table className="tbl">
            <caption className="vh">ReputeCast compared with Mailchimp, Brevo, Kit, MailerLite and beehiiv</caption>
            <thead>
              <tr>
                <th scope="col">
                  <span className="vh">Feature</span>
                </th>
                {PLATFORMS.map((p) => (
                  <th scope="col" key={p}>
                    {p}
                  </th>
                ))}
              </tr>
            </thead>
            <tbody>
              {rows.map((r) => (
                <tr key={r.label}>
                  <th scope="row" className="rowh">
                    {r.label}
                  </th>
                  {r.cells.map(renderCell)}
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </div>
    </section>
  )
}
