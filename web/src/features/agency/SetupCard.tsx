import { Icon } from '@/components/icons/Icon'
import { Badge } from '@/components/ui/Badge'
import { useT } from '@/i18n'
import type { AgencyClientRow, AgencyPulse } from './api'
import { m } from './messages'
import { ClientTile, hhmm, isToday } from './parts'

type Setup = NonNullable<AgencyPulse['setup']>

const steps: Record<string, { label: typeof m.stepProvider; hint: typeof m.stepProviderHint }> = {
  provider: { label: m.stepProvider, hint: m.stepProviderHint },
  domain: { label: m.stepDomain, hint: m.stepDomainHint },
  invite: { label: m.stepInvite, hint: m.stepInviteHint },
  shopify: { label: m.stepShopify, hint: m.stepShopifyHint },
}

/** A new client's setup: the steps its template asked for (AgencyStates). */
export function SetupCard({ row, setup }: { row: AgencyClientRow; setup: Setup }) {
  const t = useT()
  const today = isToday(row.createdAt)
  return (
    <section className="card" aria-label={row.workspace.name}>
      <div className="flex flex-wrap items-center gap-2.5 px-[18px] pt-3.5 pb-2.5">
        <ClientTile ws={row.workspace} size={28} />
        <b className="text-[14px]">{row.workspace.name}</b>
        <Badge tone="neutral">{t(m.settingUp)}</Badge>
        <span className="k ml-auto">
          {t(m.setupCardMeta, { manager: row.accountManager?.name ?? '', when: today ? `${t(m.today).toLowerCase()}, ${hhmm(row.createdAt)}` : hhmm(row.createdAt) })}
        </span>
      </div>
      <div className="flex items-center gap-2.5 px-[18px] pb-2.5">
        <span className="bar flex-1" aria-hidden="true">
          <i style={{ width: `${setup.total ? (setup.done / setup.total) * 100 : 0}%`, background: '#0F766E' }} />
        </span>
        <span className="k">{t(m.setupSteps, { done: setup.done, total: setup.total })}</span>
      </div>
      <ol className="m-0 list-none p-0">
        {setup.steps.map((s, i) => {
          const def = steps[s.key]
          if (!def) return null
          return (
            <li key={s.key} className="flex items-center gap-2.5 border-t border-line-2 px-[18px] py-2.5">
              <span
                className="inline-flex h-[22px] w-[22px] flex-none items-center justify-center rounded-full border text-[11px] font-semibold"
                style={s.done ? { background: '#DCFCE7', borderColor: '#BBF7D0', color: '#166534' } : { borderColor: '#D4D8DF', color: '#4B5563' }}
              >
                {s.done ? <Icon name="check" size={12} /> : i + 1}
              </span>
              <span className="flex-1">{t(def.label)}</span>
              <span className="k text-right">{s.done ? t(m.stepDone) : s.key === 'domain' && s.detail ? s.detail : t(def.hint)}</span>
            </li>
          )
        })}
      </ol>
    </section>
  )
}
