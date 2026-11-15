import { Fragment, useEffect, useRef, useState, type CSSProperties } from 'react'
import { Count } from '../components/Count'
import { Icon } from '../components/Icon'
import { links } from '../content/links'
import { motionAllowed, useHydrated, useMotion, useProgress } from '../lib/motion'

/** The limits a workspace's agent runs start with (internal/ai/agent.go). */
export const AGENT_RUN_LIMITS = { toolCalls: 40, costUSD: 2 }

const points = ['You keep or discard every draft', 'Spending limits on every run', 'MCP assistants can propose drafts too']

const brief = 'Win back readers who went quiet: three emails over ten days.'.split(' ')

const drafts = [
  { kind: 'Segment', name: 'Team · quiet 60 days', note: '214 people' },
  { kind: 'Flow', name: 'Team re-engagement', note: 'Unpublished' },
  { kind: 'Email 1', name: 'A good time to catch up', note: 'Day 0' },
  { kind: 'Email 2', name: 'A small thank-you from Acme', note: 'Day 5' },
  { kind: 'Email 3', name: 'Still want to hear from us?', note: 'Day 10' },
]

/** When a step of the run appears, in seconds after it starts (site.css times the rest). */
const at = (s: number) => ({ '--at': `${s}s` }) as CSSProperties

const cents = (n: number) => `$${(n / 100).toFixed(2)}`

/** A button in the picture, not a control on this page: hidden from assistive technology, and it takes no pointer. */
function Pictured({ label, primary }: { label: string; primary?: boolean }) {
  return (
    <span className={`btn${primary ? ' pri ag-keep' : ''} pointer-events-none min-h-8 px-3 text-[13px]`} aria-hidden="true">
      {label}
    </span>
  )
}

/**
 * "An AI agent that drafts, never sends.": one agent run, drafted in about
 * four seconds once a third of it is in view: the brief, the drafts one by
 * one, Ready to review, an MCP client's proposal and the lock on sending.
 * The prerendered page, reduced motion and paused motion show it finished;
 * data-live tells the CSS fallback that the script has taken over.
 */
export function Agent() {
  const run = useRef<HTMLDivElement>(null)
  const motion = useMotion()
  const hydrated = useHydrated()
  const animate = motionAllowed(motion) && !motion.settled
  const [started, setStarted] = useState(false)
  const go = started || (hydrated && !animate)

  useEffect(() => {
    if (started || !animate) return
    const el = run.current
    if (!el || typeof IntersectionObserver === 'undefined') {
      const timer = window.setTimeout(() => setStarted(true), 0)
      return () => window.clearTimeout(timer)
    }
    const io = new IntersectionObserver(
      (entries) => {
        if (entries.some((e) => e.isIntersecting)) setStarted(true)
      },
      { threshold: 0.35 },
    )
    io.observe(el)
    return () => io.disconnect()
  }, [started, animate])

  // The run's usage counts up as its review appears.
  const { p, live } = useProgress({ start: go, delay: 2300, duration: 1000 })

  return (
    <section className="wrap flex flex-wrap items-center gap-10 pt-10 pb-10" id="agent" aria-labelledby="agent-h">
      <div className="reveal flex flex-[1_1_340px] flex-col gap-3.5">
        <h2 className="sec" id="agent-h">
          An AI agent that drafts, never sends.
        </h2>
        <p className="lead">Describe what you need. It plans and drafts campaigns, posts, segments and email flows on your own model key, then waits for you.</p>
        <ul className="m-0 flex list-none flex-col gap-2.5 p-0 text-[15px] leading-[1.5] font-medium">
          {points.map((t) => (
            <li key={t}>
              <span className="cmp-mark yes">
                <i />
                <span>{t}</span>
              </span>
            </li>
          ))}
        </ul>
        <div className="flex flex-wrap gap-2.5 pt-1">
          <a href={links.modelGuide} className="btn pri">
            Connect your own model
          </a>
        </div>
      </div>
      <div ref={run} className={`ag flex flex-[1.3_1_500px] flex-col gap-3${go ? ' go' : ''}`} data-live={hydrated && animate ? '' : undefined}>
        <div className="card reveal ag-scan flex flex-col gap-3.5 px-[22px] py-5 shadow-[var(--shadow-hero)]">
          <div className="flex flex-wrap items-center justify-between gap-2.5">
            <span className="flex items-center gap-2 font-bold">
              <span className="inline-flex text-accent">
                <Icon name="sparkle" />
              </span>
              Agent run · Team re-engagement
            </span>
            <span className="ag-state">
              {/* The state while it drafts; the finished state is the one read out. */}
              <span className="badge warn ag-drafting" aria-hidden="true">
                <span className="dot" />
                Drafting
              </span>
              <span className="badge ok ag-ready">
                <span className="dot" />
                Ready to review
              </span>
            </span>
          </div>
          <div className="ag-brief">
            <span className="k">Your brief</span>
            <p className="m-0">
              {brief.map((w, i) => (
                <Fragment key={i}>
                  {i > 0 ? ' ' : null}
                  <span className="ag-w" style={{ '--w': i } as CSSProperties}>
                    {w}
                  </span>
                </Fragment>
              ))}
            </p>
          </div>
          <div>
            {drafts.map((d, i) => (
              <div key={d.kind} className={`row ag-in${i === 0 ? ' border-t-0' : ''}`} style={at(1.3 + i * 0.2)}>
                <span className="cmp-mark yes ag-check">
                  <i />
                </span>
                <span className="min-w-0 flex-1">
                  <b>{d.kind}</b> {d.name}
                </span>
                <span className="k whitespace-nowrap">{d.note}</span>
              </div>
            ))}
          </div>
          <div className="ag-in flex flex-wrap items-center gap-2 border-t border-line-2 pt-3" style={at(2.3)}>
            <Pictured label="Keep bundle" primary />
            <Pictured label="Discard" />
            <span className="k ml-auto">
              <b className="n">
                <Count final={18} p={p} live={live} />
              </b>{' '}
              of {AGENT_RUN_LIMITS.toolCalls} tool calls ·{' '}
              <b className="n">
                <Count final={42} p={p} live={live} format={cents} />
              </b>{' '}
              of {cents(AGENT_RUN_LIMITS.costUSD * 100)}
            </span>
          </div>
        </div>
        <div className="card ag-in ml-7 flex flex-col gap-1.5 px-[18px] py-3.5 max-md:ml-0" style={at(3)}>
          <div className="flex flex-wrap items-center justify-between gap-2.5 text-[14px]">
            <span className="flex items-center gap-2 font-bold">
              <span className="inline-flex text-purple-fg">
                <Icon name="link" />
              </span>
              Editorial assistant · over MCP
            </span>
            <span className="badge warn">
              <span className="dot" />
              Waiting for you
            </span>
          </div>
          <span className="text-[14px]">
            Proposes a subject for Weekly Digest #143: <b>What&apos;s new at Acme this week</b>
          </span>
        </div>
        <div className="card ag-in flex items-center gap-3 border-ok-line bg-ok-tint px-[18px] py-3.5" style={at(3.5)}>
          <span className="badge ok flex-none">
            <Icon name="lock" size={13} />
            Locked
          </span>
          <span className="text-[14px]">Sending, scheduling and publishing stay with people.</span>
        </div>
      </div>
    </section>
  )
}
