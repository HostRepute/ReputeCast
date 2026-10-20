import { useId } from 'react'
import { $api } from '@/api/client'
import { formatList, useT } from '@/i18n'
import { count } from '@/lib/format'
import { useComposer } from '@/features/composer/context'
import { mc } from './composer.messages'
import { tierChoice, useEarnComposer } from './PaidVisibility'
import './composer.css'

export type Who = '' | 'paid' | 'free'

/**
 * Who on this list (ComposerEarnBlocks, brief §3.7): everyone, paid members
 * only (any paid tier or a tier and above) or free subscribers only, with
 * how many each reaches and why some paid members don't. Shown once the
 * workspace has a tier.
 */
export function WhoOnList({ lists, who, tier, onChange }: { lists: string[]; who: Who; tier: string; onChange: (who: Who, tier: string) => void }) {
  const t = useT()
  const id = useId()
  const { slug, context } = useComposer()
  const tiers = useEarnComposer().data?.tiers ?? []
  const list = lists.length === 1 ? lists[0] : undefined
  const q = $api.useQuery(
    'get',
    '/api/v1/workspaces/{workspace}/earn/audience',
    { params: { path: { workspace: slug }, query: { list: list ?? '', tier: who === 'paid' && tier ? tier : undefined } } },
    { enabled: !!list && tiers.length > 0 },
  )
  if (!tiers.length) return null
  const d = list ? q.data : undefined
  const listName = context.lists.find((l) => l.id === list)?.name ?? ''
  const options: { value: Who; label: string; n?: number }[] = [
    { value: '', label: t(mc.whoEveryone), n: d?.everyone },
    { value: 'paid', label: t(mc.whoPaid), n: d?.paid },
    { value: 'free', label: t(mc.whoFree), n: d?.free },
  ]
  const left: string[] = []
  if (d) {
    if (d.leftOut.unsubscribed) left.push(t(mc.leftUnsubscribed, { count: d.leftOut.unsubscribed, list: listName }))
    if (d.leftOut.notOnList) left.push(t(mc.leftNotOn, { count: d.leftOut.notOnList, list: listName }))
    if (d.leftOut.suppressed) left.push(t(mc.leftSuppressed, { count: d.leftOut.suppressed }))
    if (d.leftOut.inactive) left.push(t(mc.leftInactive, { count: d.leftOut.inactive }))
  }
  return (
    <fieldset className="ea-who">
      <legend>{t(mc.who)}</legend>
      {options.map((o) => (
        <label key={o.value} className={`ea-popt${who === o.value ? ' on' : ''}`}>
          <input type="radio" name={id} checked={who === o.value} onChange={() => onChange(o.value, o.value === 'paid' ? tier : '')} />
          <span className="flex-1">{o.label}</span>
          <span className="num">{o.n != null ? count(o.n) : ''}</span>
        </label>
      ))}
      {who === 'paid' ? (
        <label className="flex flex-col gap-1 text-[12.5px] font-semibold">
          {t(mc.paidTier)}
          <select className="in font-normal" value={tier} onChange={(e) => onChange('paid', e.target.value)}>
            <option value="">{t(mc.anyPaidTier)}</option>
            {tiers.map((x) => (
              <option key={x.id} value={x.id}>
                {tierChoice(t, x)}
              </option>
            ))}
          </select>
        </label>
      ) : null}
      {who && lists.length > 1 ? <span className="k">{t(mc.oneList)}</span> : null}
      {who === 'paid' && d ? (
        <div className="ea-who-sum">
          <span className="num font-semibold">{t(mc.recipients, { count: d.paid })}</span>
          <span className="k">
            {t(mc.withAccess, { count: d.withAccess })} {left.length ? t(mc.leftOut, { list: formatList(left) }) : ''}
          </span>
          <span className="k">{t(mc.includesAudience, { pastDue: d.pastDue })}</span>
        </div>
      ) : null}
    </fieldset>
  )
}
