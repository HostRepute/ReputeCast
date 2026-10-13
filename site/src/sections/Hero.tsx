import { useRef } from 'react'
import { Count } from '../components/Count'
import { links } from '../content/links'
import { useProgress } from '../lib/motion'

const RECIPIENTS = 31_604
const SENT = 19_594

/**
 * Milliseconds until `offset` ms after the element's CSS animation started,
 * so the counters start exactly when the verdict badge pops even though the
 * script runs a little after first paint.
 */
function sinceCssStart(el: Element | null, offset: number): number {
  const anim = el && typeof el.getAnimations === 'function' ? el.getAnimations()[0] : undefined
  const now = typeof document !== 'undefined' ? document.timeline?.currentTime : null
  if (anim && typeof anim.startTime === 'number' && typeof now === 'number') return anim.startTime + offset - now
  return offset
}

const checks: { tone: 'ok' | 'warn'; label: string; text: string; action?: string }[] = [
  { tone: 'ok', label: 'Pass', text: 'news.acme.io clean on 84 of 84 blocklists' },
  { tone: 'ok', label: 'Pass', text: 'SPF, DKIM, DMARC aligned on SES and Postal' },
  { tone: 'ok', label: 'Pass', text: "64 KB, under Gmail's 102 KB clipping limit" },
  { tone: 'warn', label: 'Warn', text: '412 unengaged contacts in the audience', action: 'Exclude' },
  { tone: 'ok', label: 'Pass', text: 'Route SES 90% / Postal 10%, both healthy' },
]

export function Hero() {
  const verdict = useRef<HTMLSpanElement>(null)
  // Recipients and sent count up together when the verdict pops (2.5 s).
  const { p, live } = useProgress({ start: true, delay: () => sinceCssStart(verdict.current, 2500), duration: 1600 })
  const frac = Math.round(SENT * p) / RECIPIENTS
  return (
    <section className="relative overflow-hidden border-b border-line-2 bg-ground-2">
      {/* On phones the board shows only the small blob. */}
      <div className="blob top-[-140px] right-[-120px] h-[420px] w-[420px] max-md:hidden" aria-hidden="true" />
      <div className="blob bottom-[-110px] left-[-90px] h-[260px] w-[260px] [animation-delay:-6s]" aria-hidden="true" />
      <div className="wrap relative flex flex-wrap items-center gap-10 pt-18 pb-18">
        <div className="flex flex-[1_1_460px] flex-col gap-5">
          <span className="badge mut rise d1 self-start">
            <span className="dot text-accent" />
            Open source · self-hosted · built by HostRepute
          </span>
          <h1 className="rise d2 m-0 text-[52px] leading-[1.05] font-bold tracking-[-0.03em]">The newsletter platform that protects your inbox placement.</h1>
          <p className="lead rise d3 text-[19px]">
            ReputeCast tells you whether a campaign will reach the inbox before you send, and guards your sender reputation while it sends. It delivers through
            your own Amazon SES, Postmark, Mailgun, SendGrid, Resend, Postal or any SMTP server, with several providers at once and no lock-in.
          </p>
          <div className="rise d4 flex flex-col gap-2.5">
            <pre className="term">
              <span className="typed">docker compose -f {links.composeUrl} up -d</span>
              <span className="cursor" aria-hidden="true" />
            </pre>
            <span className="k">One command. Postgres and the app. Then open localhost:8080 and connect a provider in about 10 minutes.</span>
          </div>
          <div className="rise d5 flex flex-wrap gap-2.5">
            <a href={links.repo} className="btn pri">
              Star on GitHub
            </a>
            <a href={links.discord} className="btn">
              Join the Discord
            </a>
            <a href={links.demo} className="btn">
              Try the live demo
            </a>
          </div>
        </div>
        <div className="flex flex-[1_1_420px] flex-col gap-3">
          <div className="card scan rise d6 flex flex-col gap-3 px-[22px] py-5 shadow-[var(--shadow-hero)]">
            <div className="flex items-center justify-between gap-2.5 max-md:flex-wrap">
              <span className="font-bold">Send Readiness · Weekly Digest #142</span>
              <span className="badge warn verdict" ref={verdict}>
                <span className="dot" />
                Ready with 2 warnings
              </span>
            </div>
            {checks.map((c, i) => (
              <div key={c.text} className={`row chk c${i + 1}${i === 0 ? ' border-t-0' : ''}`}>
                <span className={`badge ${c.tone}`}>{c.label}</span>
                <span>{c.text}</span>
                {c.action ? (
                  // Part of the product illustration, not a control on this page.
                  <span className="btn ml-auto min-h-[30px] cursor-default px-2.5 text-[12.5px]" aria-hidden="true">
                    {c.action}
                  </span>
                ) : null}
              </div>
            ))}
            <div className="flex flex-wrap gap-4 border-t border-line-2 pt-1.5 text-[13.5px]">
              <span>
                <b className="n">
                  <Count final={RECIPIENTS} p={p} live={live} />
                </b>{' '}
                recipients
              </span>
              <span>
                <b className="n">$2.84</b> on SES
              </span>
              <span>
                <b className="n">34 min</b> at 14/s
              </span>
            </div>
          </div>
          <div className="card live flex flex-col gap-2 px-4.5 py-3.5">
            <div className="flex items-center justify-between gap-2.5 text-[13.5px] max-md:flex-wrap">
              <span className="flex items-center gap-2 font-bold">
                <span className="dot text-ok-bar" />
                Sending · live
              </span>
              <span className="k">SES 14 /s · Postal 2 /s</span>
            </div>
            <div className="bar flex gap-0.5" aria-hidden="true">
              <i className="rounded-[5px_0_0_5px] bg-chart-ses" style={{ width: `${(frac * 90).toFixed(1)}%` }} />
              <i className="rounded-none bg-chart-postal" style={{ width: `${(frac * 10).toFixed(1)}%` }} />
            </div>
            <div className="flex flex-wrap justify-between gap-1.5 text-[13px]">
              <span>
                <b className="n">
                  <Count final={SENT} p={p} live={live} />
                </b>{' '}
                of 31,604 sent
              </span>
              <span className="k">99.0% delivered · 0.60% bounced · 0.03% complaints</span>
            </div>
          </div>
          <div className="card held flex items-center gap-3 border-bad-line bg-bad-tint px-4.5 py-3.5 max-md:flex-wrap">
            <span className="badge bad">
              <span className="dot pulse rounded-full" />
              Auto-paused
            </span>
            <span className="text-[14px]">Outlook complaints hit 0.18% (limit 0.10%). 2,522 held, everything else still sending.</span>
          </div>
        </div>
      </div>
    </section>
  )
}
