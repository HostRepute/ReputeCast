import { Badge } from '@/components/ui/Badge'
import { Icon } from '@/components/icons/Icon'
import { useT } from '@/i18n'
import { clock, shortDate } from '@/lib/format'
import { cn } from '@/lib/cn'
import { m } from './messages'
import type { FeedItem } from './shared'

/** "Wed 15 Oct, 07:30" for today's and yesterday's posts, "Tue 14 Oct" for older ones (as the boards date them). */
export function postDate(d: string, withTime: boolean): string {
  return withTime ? `${shortDate(d)}, ${clock(d)}` : shortDate(d)
}

/**
 * A post of the feed (design system "Feed item row"): thumbnail, title, date
 * and author, and a badge: New, Sent in #2, Left out.
 */
export function FeedItemRow({ item, withTime, showCategory, className }: { item: FeedItem; withTime?: boolean; showCategory?: boolean; className?: string }) {
  const t = useT()
  const date = item.published ? postDate(item.published, Boolean(withTime)) : ''
  const category = item.categories[0]
  const meta =
    showCategory && category
      ? item.author
        ? t(m.postMetaCategory, { date, author: item.author, category })
        : t(m.postMetaCategoryOnly, { date, category })
      : item.author
        ? t(m.postMeta, { date, author: item.author })
        : date
  return (
    <div className={cn('flex items-center gap-3 py-2.5', className)}>
      <span
        className="grid h-9 w-9 shrink-0 place-items-center overflow-hidden rounded-md border border-line bg-ground-2 text-ink-3"
        role="img"
        aria-label={item.image ? t(m.featuredImage, { title: item.title }) : t(m.noFeaturedImage, { title: item.title })}
      >
        {item.image ? <img src={item.image} alt="" className="h-full w-full object-cover" loading="lazy" /> : <Icon name="image" size={16} />}
      </span>
      <div className="flex min-w-0 flex-[1_1_160px] flex-col gap-0.5">
        <span className="font-semibold leading-snug">{item.title}</span>
        <span className="k">{meta}</span>
      </div>
      <FeedItemBadge item={item} />
    </div>
  )
}

export function FeedItemBadge({ item }: { item: FeedItem }) {
  const t = useT()
  switch (item.status) {
    case 'new':
      return <Badge tone="info">{t(m.badgeNew)}</Badge>
    case 'sent':
      return <Badge tone="ok">{t(m.badgeSentIn, { number: item.sentIn ?? 0 })}</Badge>
    case 'left_out':
      return <Badge tone="neutral">{t(m.badgeLeftOut)}</Badge>
    default:
      return <Badge tone="neutral">{t(m.badgeBefore)}</Badge>
  }
}
