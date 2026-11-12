import { Link } from '@tanstack/react-router'
import { type ReactNode } from 'react'
import { $api, errorMessage, type Schemas } from '@/api/client'
import { Badge } from '@/components/ui/Badge'
import { Button } from '@/components/ui/Button'
import { toast } from '@/components/ui/Toast'
import { T, formatList, useT } from '@/i18n'
import { dayMonthDate } from '@/lib/format'
import { useWorkspace } from '@/lib/session'
import { m } from './messages'

const mono = (c: ReactNode[]) => <span className="mono">{c}</span>

/**
 * The store page's "Discount codes" row (IntegrationsShopify,
 * DiscountCodesStates · store not allowed): whether Shopify lets
 * ReputeCast write discounts, who allowed it, the pools that create codes
 * there, and Allow discount codes, which asks Shopify again.
 */
export function StoreDiscountsRow({ page }: { page: Schemas['CommerceStorePage'] }) {
  const t = useT()
  const { slug } = useWorkspace()
  const d = page.discounts
  const allow = $api.useMutation('post', '/api/v1/workspaces/{workspace}/stores/{id}/allow-discounts')
  if (!d.supported) return null
  const a = (c: ReactNode[]) => (
    <Link to="/w/$workspace/campaigns/codes" params={{ workspace: slug }}>
      {c}
    </Link>
  )
  const run = async () => {
    try {
      const out = await allow.mutateAsync({ params: { path: { workspace: slug, id: page.store.id } } })
      window.location.assign(out.url)
    } catch (e) {
      toast({ kind: 'error', title: errorMessage(e) })
    }
  }
  let detail: ReactNode
  if (!d.granted) {
    detail = t(m.storeNotAllowed, { store: page.store.name })
  } else if (d.allowedBy && d.allowedAt && d.pools.length > 0) {
    detail = <T msg={m.storeAllowedBy} values={{ who: d.allowedBy, date: dayMonthDate(d.allowedAt), pools: formatList(d.pools), count: d.pools.length, mono, a }} />
  } else if (d.allowedBy && d.allowedAt) {
    detail = <T msg={m.storeAllowedNoPools} values={{ who: d.allowedBy, date: dayMonthDate(d.allowedAt), mono, a }} />
  } else {
    detail = <T msg={m.storeAllowedUnknown} values={{ mono, a }} />
  }
  return (
    <div className="flex flex-col gap-2 border-t border-line-2 pt-3">
      <div className="flex flex-wrap items-start justify-between gap-3.5">
        <span className="flex min-w-0 flex-[1_1_300px] flex-col gap-0.5">
          <b className="font-semibold">{t(m.storeRow)}</b>
          <span className="k">{detail}</span>
        </span>
        {d.granted ? (
          <Badge tone="ok" dot>
            {t(m.storeAllowedBadge)}
          </Badge>
        ) : page.canManage ? (
          <Button size="sm" variant="primary" loading={allow.isPending} onClick={() => void run()}>
            {t(m.allowDiscounts)}
          </Button>
        ) : (
          <Badge>{t(m.storeNotAllowedBadge)}</Badge>
        )}
      </div>
      {!d.granted ? (
        <span className="k">
          <T msg={m.storeAllowHint} values={{ mono }} />
        </span>
      ) : null}
    </div>
  )
}
