import { links } from '../content/links'

const checks = [
  { label: 'Domain blocklists', value: '0 of 84 listed' },
  { label: 'DMARC alignment, 7 days', value: '99.6%' },
  { label: 'Gmail domain reputation', value: 'High' },
]

/** "Built by HostRepute" with the domain health card. */
export function HostRepute() {
  return (
    <section className="wrap flex flex-wrap items-center gap-8 pt-16 pb-16" aria-labelledby="hostrepute-h">
      <div className="reveal flex flex-[1_1_380px] flex-col gap-3">
        <span className="badge mut self-start">Built by HostRepute</span>
        <h2 className="sec" id="hostrepute-h">
          Made by the people who watch sender reputation for a living.
        </h2>
        <p className="lead">
          HostRepute monitors domains and IPs across 80+ blocklists, Google Postmaster and Microsoft SNDS. ReputeCast is its sibling: the same checks, built into the
          moment you press send. Connecting HostRepute is optional and free; without it, ReputeCast still uses your providers&apos; data and built-in checks.
        </p>
        <div className="flex flex-wrap gap-2.5">
          <a href={links.hostrepute} className="btn">
            About HostRepute
          </a>
          <a href={links.demoWithoutHostRepute} className="btn">
            See what works without it
          </a>
        </div>
      </div>
      <div className="card reveal flex flex-[1_1_380px] flex-col gap-3 p-[22px]">
        <div className="flex items-center justify-between max-md:flex-wrap">
          <span className="font-bold">news.acme.io</span>
          <span className="badge ok">
            <span className="dot" />
            96 · Good
          </span>
        </div>
        {checks.map((c, i) => (
          <div key={c.label} className={`row${i === 0 ? ' border-t-0' : ''}`}>
            <span className="flex-1">{c.label}</span>
            <span className="n">{c.value}</span>
          </div>
        ))}
        <div className="row">
          <span className="flex-1">Postal IP 203.0.113.25</span>
          <span className="badge warn">Listed on 1 · delisting requested</span>
        </div>
      </div>
    </section>
  )
}
