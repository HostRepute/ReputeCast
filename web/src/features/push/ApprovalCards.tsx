import type { Schemas } from '@/api/client'
import { useT } from '@/i18n'
import { andList, money } from '@/lib/format'
import { Body } from '@/features/whatsapp/common'
import { m } from './approval.messages'
import { Preview } from './Previews'
import { views } from './cuts'
import './push.css'
import './composer.css'
import '@/features/whatsapp/whatsapp.css'


/** What an approver reviews for a push campaign: the notification as two platforms show it, and where it goes. */
export function PushApprovalCard({ push }: { push: Schemas['ApprovalPushSnapshot'] }) {
  const t = useT()
  const n = { title: push.title, message: push.message, host: push.host ?? push.sites[0] ?? '', iconUrl: push.icon, imageUrl: push.image, buttons: push.buttons.map((b) => b.label) }
  const vs = views(push.title, push.message).filter((v) => v.platform.key === 'chrome_windows' || v.platform.key === 'chrome_android')
  const ttl = push.ttlMinutes % 60 === 0 ? t(m.hours, { n: push.ttlMinutes / 60 }) : t(m.minutes, { n: push.ttlMinutes })
  return (
    <section className="card flex min-w-0 flex-col" aria-labelledby="ap-push-h">
      <div className="sec">
        <div className="flex min-w-0 flex-col gap-0.5">
          <h2 className="h2" id="ap-push-h">
            {t(m.pushTitle)}
          </h2>
          <span className="k">{t(m.pushCaption, { n: push.recipients, sites: andList(push.sites), schedule: push.schedule })}</span>
        </div>
      </div>
      <div className="pad flex flex-col gap-3">
        <div className="pc-pcols">
          {vs.map((v) => (
            <div key={v.platform.key} className="pc-pcol">
              <Preview v={v} n={n} />
            </div>
          ))}
        </div>
        <dl className="kv">
          <dt>{t(m.link)}</dt>
          <dd className="mono break-all">{push.link.replace(/^https?:\/\//, '')}</dd>
          <dt>{t(m.buttons)}</dt>
          <dd>{push.buttons.length ? push.buttons.map((b) => b.label).join(' · ') : t(m.none)}</dd>
          <dt>{t(m.expiry)}</dt>
          <dd>{ttl}</dd>
          {push.tag ? (
            <>
              <dt>{t(m.tag)}</dt>
              <dd className="mono">{push.tag}</dd>
            </>
          ) : null}
        </dl>
      </div>
    </section>
  )
}

/** What an approver reviews for a WhatsApp campaign: the message as the preview contact reads it, its variables and its cost. */
export function WhatsAppApprovalCard({ wa }: { wa: Schemas['ApprovalWhatsAppSnapshot'] }) {
  const t = useT()
  return (
    <section className="card flex min-w-0 flex-col" aria-labelledby="ap-wa-h">
      <div className="sec">
        <div className="flex min-w-0 flex-col gap-0.5">
          <h2 className="h2" id="ap-wa-h">
            {t(m.waTitle)}
          </h2>
          <span className="k">
            {t(m.waCaption, { template: wa.template, category: wa.category, language: wa.language, n: wa.recipients, schedule: wa.schedule })}
          </span>
        </div>
      </div>
      <div className="pad flex flex-wrap items-start gap-5">
        <div className="wa-chat flex-[1_1_300px]">
          <div className="hd">
            <span className="wa-mark" aria-hidden="true">
              {(wa.business || 'A').charAt(0).toUpperCase()}
            </span>
            <span className="flex min-w-0 flex-col">
              <b className="text-[14px]">{wa.business}</b>
              <span className="k mono">{wa.number}</span>
            </span>
          </div>
          <div className="bd">
            <div className="bub">{wa.preview || <Body text={wa.body} />}</div>
            {wa.buttons.map((b, i) => (
              <span key={i} className="qr">
                {b}
              </span>
            ))}
          </div>
        </div>
        <dl className="kv flex-[1_1_260px]">
          <dt>{t(m.business)}</dt>
          <dd>
            {wa.business} · <span className="mono">{wa.number}</span>
          </dd>
          <dt>{t(m.variables)}</dt>
          <dd className="flex flex-col gap-0.5">
            {wa.variables.length
              ? wa.variables.map((v) => (
                  <span key={v.key}>
                    {v.source.startsWith('“')
                      ? t(m.varText, { key: v.key, text: v.source })
                      : v.fallback
                        ? t(m.varFieldFallback, { key: v.key, field: v.source, fallback: v.fallback })
                        : t(m.varField, { key: v.key, field: v.source })}
                  </span>
                ))
              : t(m.none)}
          </dd>
          <dt>{t(m.cost)}</dt>
          <dd>{t(m.costLine, { amount: money(wa.cost) })}</dd>
        </dl>
      </div>
    </section>
  )
}
