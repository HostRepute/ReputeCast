import { useId, useState } from 'react'
import { useT } from '@/i18n'
import { count } from '@/lib/format'
import { useDigestMode } from '@/features/digests/composer/DigestMode'
import { useComposer } from './context'
import { RouteButton } from './Header'
import { ToolIcon } from './icons'
import { AudiencePicker, SenderPicker } from './Pickers'
import { SubjectBar } from './SubjectBar'
import { mh } from './chrome.messages'

const routeColors: Record<string, string> = { ses: 'var(--chart-ses)', postal: 'var(--chart-postal)', postmark: 'var(--chart-postmark)' }

/**
 * The Inbox strip (CO-14, Composer board): the subject, preheader, audience
 * and route on one line; Edit opens them with the inbox preview and the
 * "Sending to" drawer (audience, sender, route).
 */
export function InboxStrip({ phoneSheet }: { phoneSheet?: boolean }) {
  const c = useComposer()
  const t = useT()
  const id = useId()
  const digest = useDigestMode()
  // A new draft opens it: the subject comes first.
  const [open, setOpen] = useState(() => !c.subject.trim() && !c.isEmpty)
  const route = c.campaign.route.shares.length ? c.campaign.route : c.context.broadcastRoute
  const heuristics = c.checks?.subjectHeuristics ?? []
  const panel = `${id}-panel`
  const shown = open || phoneSheet
  return (
    <>
      {phoneSheet ? null : (
        <div className="istrip" role="region" aria-label={t(mh.inbox)}>
          <span className="is-l">
            <ToolIcon name="mail" size={14} strokeWidth={1.6} />
            {t(mh.inbox)}
          </span>
          <p className="is-sum">
            <span className="is-i sub">
              <span className="is-k">{t(mh.inboxSubject)}</span>{' '}
              {c.subject.trim() ? <span className="is-v">{c.subject}</span> : <span className="is-v none">{t(mh.inboxNone)}</span>}
            </span>
            {heuristics.length && c.subject.trim() ? <span className="is-flag">{t(mh.subjectFlag, { count: heuristics.length })}</span> : null}
            {c.preheader.trim() ? (
              <>
                <span className="is-sep" aria-hidden="true">
                  ·
                </span>
                <span className="is-i pre">
                  <span className="is-k">{t(mh.inboxPreheader)}</span>{' '}
                  <span className="is-v">{c.preheader}</span>
                </span>
              </>
            ) : null}
            {digest ? null : (
              <>
                <span className="is-sep" aria-hidden="true">
                  ·
                </span>
                <span className="is-i to">
                  <span className="is-k">{t(mh.inboxTo)}</span>{' '}
                  <span className="is-v">{c.campaign.audienceCount != null ? `${c.campaign.audienceLabel} · ${count(c.campaign.audienceCount)}` : c.campaign.audienceLabel}</span>
                </span>
              </>
            )}
            {route.shares.length ? (
              <>
                <span className="is-sep" aria-hidden="true">
                  ·
                </span>
                <span className="is-i via">
                  <span className="is-k">{t(mh.inboxVia)}</span>{' '}
                  <span className="is-v">
                    {route.shares.map((s, i) => (
                      <span key={s.providerId}>
                        {i ? ' / ' : ''}
                        <span className="dot" style={{ color: routeColors[s.adapter] ?? 'var(--chart-other)', marginRight: 4 }} aria-hidden="true" />
                        {s.short} {Math.round(s.percent)}%
                      </span>
                    ))}
                  </span>
                </span>
              </>
            ) : null}
          </p>
          <button type="button" className="btn sm is-edit" aria-expanded={open} aria-controls={open ? panel : undefined} onClick={() => setOpen((o) => !o)}>
            {open ? t(mh.inboxDone) : t(mh.inboxEdit)}
          </button>
        </div>
      )}
      {shown ? (
        <div id={panel} className="ipanel" role="region" aria-label={t(mh.inboxPanel)}>
          <SubjectBar />
          {digest ? null : (
            <div className="ip-send" role="group" aria-labelledby={`${id}-sendto`}>
              <h2 className="ip-h" id={`${id}-sendto`}>
                {t(mh.sendingTo)}
              </h2>
              <div className="lbl">
                <span>{t(mh.audience)}</span>
                <AudiencePicker />
              </div>
              <div className="lbl">
                <span>{t(mh.from)}</span>
                <SenderPicker />
              </div>
              <div className="lbl">
                <span>{t(mh.route)}</span>
                <RouteButton />
                {c.campaign.access.canManageRouting ? null : <span className="k">{t(mh.routeWho)}</span>}
              </div>
            </div>
          )}
        </div>
      ) : null}
    </>
  )
}
