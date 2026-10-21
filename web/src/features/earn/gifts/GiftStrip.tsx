import { Link } from '@tanstack/react-router'
import type { ReactNode } from 'react'
import { $api } from '@/api/client'
import { T, useT } from '@/i18n'
import { useWorkspace } from '@/lib/session'
import { EARN } from '../EarnPage'
import { money } from '../format'
import { mof } from '../offers/messages'
import { GiftIcon } from './parts'

const bold = (c: ReactNode) => <b className="font-semibold">{c}</b>
const mono = (c: ReactNode) => <span className="mono">{c}</span>

/** Plans & offers' "Gift options" line (EarnPlans): the options and the gift page, linking to Gifts. */
export function GiftStrip() {
  const t = useT()
  const { slug } = useWorkspace()
  const q = $api.useQuery('get', `${EARN}/gifts`, { params: { path: { workspace: slug } } })
  const v = q.data
  if (!v) return null
  const options = v.options.map((o) => t(mof.giftsOption, { tier: o.short, months: o.months, price: money(o.amountCents, o.currency, 'auto') })).join(' · ')
  return (
    <section className="card flex flex-wrap items-center gap-x-[18px] gap-y-2.5 px-[18px] py-3.5" aria-label={t(mof.giftsLink)}>
      <span className="flex-none text-ink-nav">
        <GiftIcon size={18} />
      </span>
      <span className="min-w-0 flex-[1_1_420px]">
        {v.options.length ? (
          <T msg={mof.giftsStrip} values={{ b: bold, mono, options, on: v.page.on ? 'true' : 'false', host: v.page.host }} />
        ) : (
          <T msg={mof.giftsNone} values={{ b: bold }} />
        )}
      </span>
      <Link to="/w/$workspace/earn/gifts" params={{ workspace: slug }} className="lnk">
        {t(mof.giftsLink)}
      </Link>
    </section>
  )
}
