import type { Schemas } from '@/api/client'
import { Table } from '@/components/ui/Table'
import { useT } from '@/i18n'
import { count, money } from '@/lib/format'
import { m } from './approval.messages'
import { CountryChips, countryName, price } from './common'
import './sms.css'

type Snapshot = Schemas['ApprovalSMSSnapshot']

/**
 * What an approver reviews for a text campaign (brief §6, the approval
 * review): the text as it goes out, with the rules' locked parts, its
 * length, and the senders and cost per country at request time.
 */
export function SmsApprovalCard({ sms }: { sms: Snapshot }) {
  const t = useT()
  return (
    <section className="card flex min-w-0 flex-col" aria-labelledby="ap-sms-h">
      <div className="sec">
        <div className="flex min-w-0 flex-col gap-0.5">
          <h2 className="h2" id="ap-sms-h">
            {t(m.title)}
          </h2>
          <span className="k">{t(m.caption, { chars: count(sms.chars), segments: sms.segments, encoding: sms.encoding === 'ucs2' ? 'UCS-2' : 'GSM-7' })}</span>
        </div>
      </div>
      <div className="pad flex flex-col gap-3">
        <div className="flex flex-col" aria-label={t(m.bubble)} role="group">
          <p className="sms-bub m-0">
            {sms.prefix ? <span className="fx">{sms.prefix} </span> : null}
            {sms.body}
            {sms.optOut ? <span className="fx"> {sms.optOut}</span> : null}
          </p>
        </div>
        {sms.countries.length ? (
          <div className="overflow-x-auto">
            <Table caption={t(m.countries)}>
              <thead>
                <tr>
                  <th scope="col">{t(m.country)}</th>
                  <th scope="col">{t(m.sender)}</th>
                  <th scope="col" className="r">
                    {t(m.recipients)}
                  </th>
                  <th scope="col" className="r">
                    {t(m.price)}
                  </th>
                  <th scope="col" className="r">
                    {t(m.cost)}
                  </th>
                </tr>
              </thead>
              <tbody>
                {sms.countries.map((c) => (
                  <tr key={c.country}>
                    <td>
                      <span className="inline-flex items-center gap-2">
                        <CountryChips codes={[c.country]} />
                        {countryName(c.country)}
                      </span>
                    </td>
                    <td>
                      <span className="mono">{c.sender}</span> <span className="k">· {c.provider}</span>
                    </td>
                    <td className="r n">{c.skipped ? <span className="k">{t(m.skipped)}</span> : count(c.recipients)}</td>
                    <td className="r n">{price(c.price)}</td>
                    <td className="r n">{money(c.cost)}</td>
                  </tr>
                ))}
              </tbody>
            </Table>
          </div>
        ) : null}
        <div className="sms-sum border-t border-line-2 pt-2">
          <span className="font-semibold">{t(m.total, { recipients: sms.recipients, cost: money(sms.cost) })}</span>
          {sms.schedule ? <span className="k">{t(m.schedule, { schedule: sms.schedule })}</span> : null}
        </div>
      </div>
    </section>
  )
}
