import { useRef, type CSSProperties, type PointerEvent, type ReactNode, type RefObject } from 'react'
import { LogoMark } from '../components/Marks'
import { links } from '../content/links'
import { useHydrated } from '../lib/motion'
import { useScrollable } from '../lib/scrollable'

/**
 * The comparison is with paid platforms only (product decision: ReputeCast
 * marketing never compares itself with other open-source projects).
 */
export const PLATFORMS = ['ReputeCast', 'Mailchimp', 'Brevo', 'Kit', 'MailerLite', 'beehiiv'] as const
type Platform = (typeof PLATFORMS)[number]
export const RIVALS = PLATFORMS.slice(1)

export const slug = (p: Platform) => p.toLowerCase()

type Cell = string | { yes: string } | { yes: true } | { no: true } | { price: string; rest: string }
type Row = { label: string; cells: Cell[]; kind?: 'edge' | 'cost' }

const theirCloud = 'Their cloud'
const policy = 'Their sending policy applies to your account'
const exportable = 'Exportable; lives in their cloud'
const previews = 'Previews and test sends'

export const rows: Row[] = [
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
  { label: 'Several providers at once, weighted splits, failover', cells: [{ yes: true }, { no: true }, { no: true }, { no: true }, { no: true }, { no: true }] },
  {
    label: 'Pre-send deliverability check',
    cells: [{ yes: 'Send Readiness: blocklists, authentication, links, audience, routing, one verdict' }, 'Previews and content checks', previews, previews, previews, previews],
  },
  {
    label: 'When bounces or complaints spike',
    cells: ['Pauses the affected route, explains why, offers resume, adjust, fail over or cancel', policy, policy, policy, policy, policy],
  },
  {
    label: 'Automations',
    cells: [{ yes: 'visual flows' }, { yes: 'Customer Journeys' }, { yes: true }, { yes: 'visual automations' }, { yes: true }, { yes: true }],
  },
  { label: 'Segments', cells: ['Builder + SQL', 'Builder', 'Builder', 'Builder', 'Builder', 'Builder'] },
  {
    label: 'Users, roles and granular permissions, OIDC SSO, 2FA, passkeys, audit log',
    cells: ['All included, free', 'By plan', 'By plan', 'By plan', 'By plan', 'By plan'],
  },
  { label: 'Your data', cells: ['In your database; export or delete anything, any time', exportable, exportable, exportable, exportable, exportable] },
  {
    label: "Where it's better than ReputeCast",
    kind: 'edge',
    cells: [
      'Younger project; you run and update the server; a smaller template and integration library',
      'Huge template and integration library, nothing to host',
      'Live chat and a sales CRM next to email and SMS',
      "A creator network that recommends you to other creators' readers",
      'Simple, affordable, polished editor',
      'Boosts and an ad network that bring readers and sponsors to you',
    ],
  },
  {
    label: 'Typical monthly cost at 50,000 subscribers',
    kind: 'cost',
    cells: [
      { price: 'About $5', rest: 'on SES per 50,000 emails, plus a small server' },
      { price: 'About $350 and up', rest: 'on Standard, by contacts' },
      { price: 'About $70 to $250', rest: 'by email volume' },
      { price: 'About $300', rest: 'by subscribers' },
      { price: 'About $250 to $300', rest: 'by subscribers' },
      { price: 'About $100 to $250', rest: 'on Scale, by subscribers' },
    ],
  },
]

function renderCell(cell: Cell, p: Platform, kind: Row['kind']): ReactNode {
  const tag = kind === 'edge' ? <span className={`badge ${p === 'ReputeCast' ? 'mut' : 'warn'}`}>{p === 'ReputeCast' ? 'Our gaps' : 'Their edge'}</span> : null
  if (typeof cell === 'string') {
    return (
      <>
        {tag}
        <span>{cell}</span>
      </>
    )
  }
  if ('price' in cell) {
    return (
      <>
        <b className="cmp-price">{cell.price}</b> <span className="cmp-rest">{cell.rest}</span>
      </>
    )
  }
  if ('no' in cell) {
    return (
      <span className="cmp-mark no bare">
        <i />
        <span>No</span>
      </span>
    )
  }
  return (
    <span className={cell.yes === true ? 'cmp-mark yes bare' : 'cmp-mark yes'}>
      <i />
      <span>{cell.yes === true ? 'Yes' : cell.yes}</span>
    </span>
  )
}

function Step({ dir, root }: { dir: 1 | -1; root: RefObject<HTMLDivElement | null> }) {
  return (
    <button type="button" className="cmp-step" tabIndex={-1} aria-hidden="true" onClick={() => step(root.current, dir)}>
      {dir < 0 ? '‹' : '›'}
    </button>
  )
}

function Options({ name, legend, options, className }: { name: string; legend: string; options: { id: string; label: string }[]; className: string }) {
  return (
    <fieldset className={`cmp-seg ${className}`}>
      <legend className="vh">{legend}</legend>
      <div className="cmp-opts" style={{ '--n': options.length } as CSSProperties}>
        <span className="cmp-thumb" aria-hidden="true" />
        {options.map((o, i) => (
          <label key={o.id} htmlFor={`cmp-${o.id}`}>
            <input type="radio" name={name} id={`cmp-${o.id}`} value={o.id} defaultChecked={i === 0} />
            {o.label}
          </label>
        ))}
      </div>
    </fieldset>
  )
}

function step(root: HTMLElement | null, by: number) {
  const radios = [...(root?.querySelectorAll<HTMLInputElement>('input[name="cmp-rival"]') ?? [])]
  const at = radios.findIndex((r) => r.checked)
  const next = radios[(at + by + radios.length) % radios.length]
  if (next) next.checked = true
}

/**
 * Radio buttons pick the rival and the view and CSS shows the picked columns,
 * so it works without JavaScript. The arrows and the swipe are for pointers.
 */
export function Compare() {
  const root = useRef<HTMLDivElement>(null)
  const scroller = useRef<HTMLDivElement>(null)
  const scrollable = useScrollable(scroller)
  const hydrated = useHydrated()
  const swipe = useRef<{ x: number; y: number } | null>(null)
  const headToHead = () => root.current?.querySelector<HTMLInputElement>('#cmp-vs')?.checked === true
  const onPointerDown = (e: PointerEvent) => {
    swipe.current = e.pointerType === 'touch' && headToHead() ? { x: e.clientX, y: e.clientY } : null
  }
  const onPointerUp = (e: PointerEvent) => {
    const start = swipe.current
    swipe.current = null
    if (!start) return
    const dx = e.clientX - start.x
    if (Math.abs(dx) > 60 && Math.abs(dx) > 2 * Math.abs(e.clientY - start.y)) step(root.current, dx < 0 ? 1 : -1)
  }
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
      <div ref={root} className="cmp card reveal">
        <div className="cmp-bar">
          <span className="cmp-cap" aria-hidden="true">
            Compare with
          </span>
          <Options name="cmp-rival" legend="Compare ReputeCast with" className="cmp-rivals" options={RIVALS.map((p) => ({ id: slug(p), label: p }))} />
          <Options
            name="cmp-view"
            legend="View"
            className="cmp-views"
            options={[
              { id: 'vs', label: 'Head to head' },
              { id: 'all', label: 'All six' },
            ]}
          />
        </div>
        <div
          ref={scroller}
          className="cmp-scroll"
          tabIndex={scrollable ? 0 : undefined}
          role={scrollable ? 'region' : undefined}
          aria-label={scrollable ? 'Comparison table, scrolls sideways' : undefined}
          onPointerDown={onPointerDown}
          onPointerUp={onPointerUp}
          onPointerCancel={() => (swipe.current = null)}
        >
          <div className="cmp-table" role="table" aria-label="ReputeCast compared with Mailchimp, Brevo, Kit, MailerLite and beehiiv">
            <div role="rowgroup">
              <div role="row" className="cmp-row cmp-head">
                <div role="columnheader" className="cmp-label">
                  <span className="vh">Feature</span>
                  <span className="cmp-vs" aria-hidden="true">
                    vs
                  </span>
                </div>
                {PLATFORMS.map((p) => (
                  <div role="columnheader" key={p} className="cmp-cell" data-p={slug(p)}>
                    {p === 'ReputeCast' ? (
                      <span className="cmp-name">
                        <LogoMark size={22} />
                        ReputeCast
                      </span>
                    ) : (
                      <>
                        {hydrated ? <Step dir={-1} root={root} /> : null}
                        <span className="cmp-name">{p}</span>
                        {hydrated ? <Step dir={1} root={root} /> : null}
                      </>
                    )}
                  </div>
                ))}
              </div>
            </div>
            <div role="rowgroup">
              {rows.map((r, i) => (
                <div role="row" key={r.label} className="cmp-row" data-kind={r.kind} style={{ '--i': i + 1 } as CSSProperties}>
                  <div role="rowheader" className="cmp-label">
                    {r.label}
                  </div>
                  {r.cells.map((c, j) => (
                    <div role="cell" key={PLATFORMS[j]} className="cmp-cell" data-p={slug(PLATFORMS[j]!)}>
                      {renderCell(c, PLATFORMS[j]!, r.kind)}
                    </div>
                  ))}
                </div>
              ))}
            </div>
          </div>
        </div>
        <div className="cmp-foot">
          <a href="#cost">Work out your own cost</a>
          <a href={links.newIssue}>Suggest a correction</a>
        </div>
      </div>
    </section>
  )
}
