import { Link } from '@tanstack/react-router'
import type { Schemas } from '@/api/client'
import { ScoreRing } from '@/components/ui/Data'
import { useWorkspace } from '@/lib/session'
import { useT } from '@/i18n'
import { m } from './messages'
import { reasonDot, scoreColors } from './words'

/** Where a reason's link goes. */
function ReasonAction({ a }: { a: Schemas['OverviewHealthAction'] }) {
  const { slug } = useWorkspace()
  const params = { workspace: slug }
  switch (a.kind) {
    case 'providers':
      return (
        <Link to="/w/$workspace/providers" params={params}>
          {a.label}
        </Link>
      )
    case 'warmup':
      return (
        <Link to="/w/$workspace/providers/warmup" params={params}>
          {a.label}
        </Link>
      )
    case 'integrations':
      return (
        <Link to="/w/$workspace/settings/integrations/hostrepute" params={params}>
          {a.label}
        </Link>
      )
  }
  return (
    <Link to="/w/$workspace/deliverability" params={params}>
      {a.label}
    </Link>
  )
}

/** "Sender health": the score ring and the reasons behind it (Main board). */
export function SenderHealthCard({ h }: { h: Schemas['OverviewSenderHealth'] }) {
  const t = useT()
  const { slug } = useWorkspace()
  const colors = scoreColors[h.tone] ?? scoreColors.neutral!
  const verdict = t(m.verdict, { label: h.label })
  return (
    <section className="card flex flex-col gap-4 px-[22px] py-5" aria-labelledby="ov-health-h">
      <div className="flex items-center justify-between gap-3 max-md:flex-wrap">
        <h2 className="h2" id="ov-health-h">
          {t(m.senderHealth)}
        </h2>
        <Link to="/w/$workspace/deliverability" params={{ workspace: slug }} className="text-[13px] font-medium">
          {t(m.deliverabilityDetails)}
        </Link>
      </div>
      <div className="flex flex-wrap items-center gap-5">
        <ScoreRing score={h.score} label={t(m.scoreLabel, { score: h.score, verdict: verdict.toLowerCase() })} verdict={verdict.toUpperCase()} verdictColor={colors.word} color={colors.ring} />
        <ul className="m-0 flex flex-[1_1_220px] list-none flex-col gap-2 p-0 text-[13px]">
          {h.reasons.map((r, i) => (
            <li key={`${r.key}-${i}`} className="flex items-start gap-2">
              <span className="dot mt-1.5" style={{ color: reasonDot[r.tone] }} aria-hidden="true" />
              <span>
                <span className="vh">{t(r.tone === 'ok' ? m.reasonGood : r.tone === 'bad' ? m.reasonProblem : m.reasonWarning)} </span>
                {r.text}
                {r.action ? (
                  <>
                    {' · '}
                    <ReasonAction a={r.action} />
                  </>
                ) : null}
              </span>
            </li>
          ))}
        </ul>
      </div>
    </section>
  )
}

/** First run: the score appears once a sending domain is verified (OverviewEmpty). */
export function PendingHealthCard() {
  const t = useT()
  return (
    <section className="card flex flex-col gap-2.5 border-dashed px-5 py-4.5" aria-labelledby="ov-health-p">
      <h3 className="h2 text-ink-3" id="ov-health-p">
        {t(m.senderHealth)}
      </h3>
      <div className="flex items-center gap-3.5 max-md:flex-wrap">
        <svg width="64" height="64" viewBox="0 0 64 64" aria-hidden="true">
          <circle cx="32" cy="32" r="26" fill="none" stroke="#EEF0F3" strokeWidth="7" />
          <text x="32" y="37" textAnchor="middle" fontFamily="IBM Plex Sans, sans-serif" fontSize="16" fontWeight="700" fill="#9AA3B2">
            –
          </text>
        </svg>
        <span className="k">{t(m.healthPending)}</span>
      </div>
    </section>
  )
}
