import { useEffect, useRef, useState, type CSSProperties, type ReactNode } from 'react'
import { Count, percent } from '../components/Count'
import { motionAllowed, useMotion, useProgress } from '../lib/motion'
import { useScrollable } from '../lib/scrollable'

type Edge = { id: string; left: number; top: number; width: number; height: number; h?: boolean }

const edges: Edge[] = [
  { id: 'e1', left: 379, top: 88, width: 2, height: 18 },
  { id: 'e2', left: 379, top: 160, width: 2, height: 18 },
  { id: 'e3', left: 379, top: 285, width: 2, height: 18 },
  { id: 'e5', left: 379, top: 392, width: 2, height: 26 },
  { id: 'e4', left: 159, top: 418, width: 444, height: 2, h: true },
  { id: 'e6', left: 159, top: 418, width: 2, height: 30 },
  { id: 'e7', left: 601, top: 418, width: 2, height: 30 },
  { id: 'e8', left: 159, top: 537, width: 2, height: 22 },
  { id: 'e9', left: 601, top: 520, width: 2, height: 18 },
  { id: 'e10', left: 601, top: 627, width: 2, height: 18 },
]

/** Pulses p5 and p8 run along the horizontal branch edge. */
const pulses = ['p1', 'p2', 'p3', 'p4', 'p5 h', 'p6', 'p7', 'p8 h', 'p9', 'p10', 'p11']

function Node({
  id,
  left,
  top,
  border,
  color,
  kind,
  title,
  input = true,
  output = true,
  selected,
  liveDot,
  children,
}: {
  id: string
  left: number
  top: number
  border?: string
  color: string
  kind: string
  title: string
  input?: boolean
  output?: boolean
  selected?: boolean
  liveDot?: boolean
  children?: ReactNode
}) {
  const style: CSSProperties = { left, top, borderColor: border }
  return (
    <div className={`fn ${id}${selected ? ' sel' : ''}`} style={style}>
      {input ? <span className="hb i" /> : null}
      {output ? <span className="hb o" /> : null}
      <div className="hd" style={{ color }}>
        <span className={liveDot ? 'dot live' : 'dot'} />
        {kind}
      </div>
      <div className="t">{title}</div>
      {children ? <div className="st">{children}</div> : null}
    </div>
  )
}

/**
 * The landing flow canvas: nodes draw in when the canvas scrolls into view
 * (or after 3.5 s), numbers count up, then one contact pulses through the
 * steps on an 11 s loop. The 760px stage scales to its card in CSS.
 */
export function FlowCanvas() {
  const wrap = useRef<HTMLDivElement>(null)
  const motion = useMotion()
  const [go, setGo] = useState(false)
  const scrollable = useScrollable(wrap)

  useEffect(() => {
    if (go) return
    // Reduced or paused motion: show the finished canvas right away.
    if (!motionAllowed(motion) || motion.settled) {
      setGo(true)
      return
    }
    const el = wrap.current
    let io: IntersectionObserver | undefined
    if (el && typeof IntersectionObserver !== 'undefined') {
      io = new IntersectionObserver(
        (entries) => {
          if (entries.some((e) => e.isIntersecting)) setGo(true)
        },
        { threshold: 0.2 },
      )
      io.observe(el)
    }
    const timer = window.setTimeout(() => setGo(true), 3500)
    return () => {
      io?.disconnect()
      window.clearTimeout(timer)
    }
  }, [go, motion])

  const { p, live } = useProgress({ start: go, delay: 900, duration: 1500 })
  const n = (final: number) => <Count final={final} p={p} live={live} />
  const pc = (final: number) => <Count final={final} p={p} live={live} format={percent} />

  return (
    <div className="reveal flex min-w-0 flex-[3_1_520px] flex-col overflow-hidden rounded-[14px] border border-line bg-white">
      <div
        ref={wrap}
        className="flowwrap rounded-none border-0"
        role="img"
        aria-label="Animated automation flow: joined list, delay, welcome email, branch on opens, two follow-ups, goal and exit"
        tabIndex={scrollable ? 0 : undefined}
      >
        <div className="flowbox">
          <div className={`flowstage${go ? ' go' : ''}`}>
            {pulses.map((cls) => (
              <span key={cls} className={`pu ${cls}`} aria-hidden="true" />
            ))}
            {edges.map((e) => (
              <div key={e.id} className={`fe ${e.id}${e.h ? ' h' : ''}`} style={{ left: e.left, top: e.top, width: e.width, height: e.height }} />
            ))}
            <span className="flab l1" style={{ left: 172, top: 407, background: '#DCFCE7', borderColor: '#86EFAC', color: '#166534' }}>
              Yes · 61%
            </span>
            <span className="flab l2" style={{ left: 614, top: 407, background: '#FEF3C7', borderColor: '#FDE68A', color: '#92400E' }}>
              No · 39%
            </span>
            <Node id="f1" left={280} top={16} border="#0F766E" color="#0F766E" kind="Trigger · joined list" title="Weekly Digest" input={false} liveDot>
              <span>
                <b className="n">{n(1284)}</b> entered · 30 d
              </span>
            </Node>
            <Node id="f2" left={280} top={106} color="#6B7280" kind="Delay" title="Wait 10 minutes" />
            <Node id="f3" left={280} top={178} color="#4a3aa7" kind="Send email · SES, failover Postmark" title={'Welcome · step 1 "You\'re in"'} selected>
              <span>
                <b className="n">{n(1276)}</b> sent
              </span>
              <span>
                <b className="n">{pc(61)}</b> human opens
              </span>
              <span>
                <b className="n">{pc(14)}</b> clicks
              </span>
            </Node>
            <Node id="f4" left={280} top={303} border="#F59E0B" color="#B45309" kind="Branch" title="Opened step 1 within 2 days?">
              <span>human opens only</span>
            </Node>
            <Node id="f5" left={60} top={448} color="#4a3aa7" kind="Send email" title={'Step 2 "Three things to try first"'}>
              <span>
                <b className="n">{n(778)}</b> sent
              </span>
              <span>
                <b className="n">{pc(57)}</b> opens
              </span>
            </Node>
            <Node id="f6" left={60} top={559} border="#0B1220" color="#0B1220" kind="Goal" title="Clicked a docs link" output={false}>
              <span>
                <b className="n">{n(312)}</b> reached · {pc(24)}
              </span>
            </Node>
            <Node id="f7" left={502} top={448} color="#6B7280" kind="Delay" title="Wait 3 days">
              <span>
                <b className="n">{n(498)}</b> waiting or passed
              </span>
            </Node>
            <Node id="f8" left={502} top={538} color="#4a3aa7" kind="Send email" title={'"Did you get our welcome?" · plain text'}>
              <span>
                <b className="n">{n(498)}</b> sent
              </span>
              <span>
                <b className="n">{pc(38)}</b> opens
              </span>
            </Node>
            <Node id="f9" left={502} top={645} border="#0B1220" color="#0B1220" kind="Exit" title="Leave the flow · stays on the list" output={false}>
              <span>
                <b className="n">{n(436)}</b> exited
              </span>
            </Node>
          </div>
        </div>
      </div>
      <div className="flowkey">
        <span>
          <span className="dot text-accent" />A contact moving through the flow; each step lights up as it arrives
        </span>
        <span>
          <span className="dot text-[#86BFB4]" />
          The No branch runs lighter: 39% of contacts
        </span>
        <span>
          <span className="dot text-[#1D4ED8]" />
          Selected step: edit it in the side panel
        </span>
        <span className="k ml-auto">Numbers are the last 30 days of this flow</span>
      </div>
    </div>
  )
}
