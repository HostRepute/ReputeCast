import { useMemo, useState, type ReactNode } from 'react'
import { Badge, type Tone } from '@/components/ui/Badge'
import { Icon } from '@/components/icons/Icon'
import { formatList, useT } from '@/i18n'
import { cn } from '@/lib/cn'
import { useComposer } from '@/features/composer/context'
import { DEVICES } from '@/features/composer/editor/grapes'
import { FeedItemRow } from '../FeedItemRow'
import { useDigestMode } from './DigestMode'
import type { FeedBlock } from './feedComponent'
import { dc } from './messages'

/** The whole issue shows at 40%. */
export const WHOLE_ZOOM = 0.4
const MARKER = /<mj-raw>\s*<!--\s*rc:feed-items[\s\S]*?-->\s*<\/mj-raw>/

const kb = (n: number) => `${Math.max(1, Math.round(n / 1024))} KB`

/** Whether the block draws images (the list layout and a block without the Image field don't). */
function showsImages(b: FeedBlock): boolean {
  return b.layout !== 'list' && b.fields.includes('image')
}

/** A check of the whole issue: its icon, what it found and its badge. */
function Check({ ok, title, children, badge, tone }: { ok: boolean; title: string; children: ReactNode; badge: string; tone: Tone }) {
  return (
    <div className="flex items-start gap-2.5 border-t border-line-2 py-[9px] text-[12.5px]">
      <span className={cn('mt-0.5 shrink-0', ok ? 'text-ok-fg' : 'text-warn-fg')}>
        <Icon name={ok ? 'check' : 'alert'} size={14} />
      </span>
      <span className="min-w-0 flex-1">
        <b className="font-semibold">{title}</b>
        <br />
        <span className="k">{children}</span>
      </span>
      <Badge tone={tone}>{badge}</Badge>
    </div>
  )
}

/**
 * "Preview with 6 items" (DigestComposer): the whole issue at 40% beside
 * its checks, its size by number of posts and the posts it shows. The
 * canvas stays underneath, so picking another preview goes back to it.
 */
export function WholeIssue() {
  const t = useT()
  const c = useComposer()
  const mode = useDigestMode()
  const r = mode?.render
  const [height, setHeight] = useState(1200)
  // The template as on the canvas, with the preview's posts in place of the Feed items block.
  const html = useMemo(() => {
    if (!r || !c.ge) return ''
    const src = c.currentMjml()
    return c.ge.compile(MARKER.test(src) ? src.replace(MARKER, r.mjml) : src).html
  }, [r, c.ge, c.rev]) // eslint-disable-line react-hooks/exhaustive-deps
  if (!mode || !r) return null
  const width = parseInt(DEVICES[c.device].width, 10)
  const ch = r.checks
  const over = ch.maxBytes > ch.limit
  const subject = r.subjects.find((x) => x.items === r.count)?.text ?? r.subject
  const noImage = showsImages(mode.block) ? r.items.filter((x) => x.fallback || !x.image) : []
  const titles = formatList(noImage.map((x) => `“${x.title}”`))
  const biggest = Math.max(ch.limit, ...r.sizes.map((s) => s.bytes))
  const fit = ch.fitPosts
  const sizesNote =
    fit === 0
      ? t(dc.sizesNone, { limit: kb(ch.limit) })
      : fit > ch.maxPosts
        ? t(dc.sizesRoom, { limit: kb(ch.limit), fit, max: ch.maxPosts })
        : fit === ch.maxPosts
          ? t(dc.sizesExact, { limit: kb(ch.limit), fit, max: ch.maxPosts })
          : t(dc.sizesOver, { limit: kb(ch.limit), fit, max: ch.maxPosts })
  return (
    <div className="flex min-h-full flex-wrap items-stretch bg-surface">
      <div className="flex flex-[1_0_auto] justify-center bg-ground-2" style={{ padding: '24px 20px', maxWidth: width * WHOLE_ZOOM + 200 }}>
        <div className="shrink-0 overflow-hidden bg-surface shadow-[0_1px_3px_rgba(11,18,32,.12)]" style={{ width: width * WHOLE_ZOOM, height: height * WHOLE_ZOOM }}>
          {html ? (
            <iframe
              title={t(dc.wholeIssue, { count: r.count })}
              srcDoc={html}
              sandbox="allow-same-origin"
              tabIndex={-1}
              className="block border-0"
              style={{ width, height, transform: `scale(${WHOLE_ZOOM})`, transformOrigin: 'top left', pointerEvents: 'none' }}
              onLoad={(e) => {
                const doc = e.currentTarget.contentDocument
                if (doc?.body) setHeight(Math.max(400, doc.documentElement.scrollHeight))
              }}
            />
          ) : null}
        </div>
      </div>
      <div className="flex min-w-0 flex-[999_1_360px] flex-col gap-5 border-l border-line" style={{ padding: '18px 18px 24px' }}>
        <section className="flex flex-col" aria-label={t(dc.checksWith, { count: r.count })}>
          <span className="mb-1 font-semibold">{t(dc.checksWith, { count: r.count })}</span>
          <Check
            ok={!over}
            title={t(over ? dc.checkSizeOver : dc.checkSize, { count: ch.maxPosts, size: kb(ch.maxBytes), limit: kb(ch.limit) })}
            badge={over ? t(dc.over) : t(dc.pass)}
            tone={over ? 'bad' : 'ok'}
          >
            {t(dc.sizeHint)}
          </Check>
          <Check ok title={t(dc.subjectCheck, { count: r.count })} badge={t(dc.pass)} tone="ok">
            {t(dc.subjectChars, { text: subject, count: [...subject].length })}
          </Check>
          {noImage.length ? (
            <Check
              ok={false}
              title={t(dc.noImage, { count: noImage.length })}
              badge={noImage.some((x) => x.fallback) ? t(dc.fromTitle) : t(dc.noFallback)}
              tone="warn"
            >
              {noImage.some((x) => x.fallback) ? t(dc.noImageFallback, { titles, count: noImage.length }) : t(dc.noImageNone, { titles, count: noImage.length })}
            </Check>
          ) : null}
          <Check ok title={t(dc.linksCheck, { count: ch.links })} badge={t(dc.pass)} tone="ok">
            {t(dc.linksHint)}
          </Check>
        </section>
        {r.sizes.length ? (
          <section className="flex max-w-[640px] flex-col gap-2.5" aria-label={t(dc.sizes)}>
            <span className="font-semibold">{t(dc.sizes)}</span>
            <div className="flex flex-col gap-2">
              {r.sizes.map((s) => {
                const big = s.bytes > ch.limit
                return (
                  <div key={s.posts} className="grid items-center gap-2.5 text-[12.5px]" style={{ gridTemplateColumns: '64px minmax(0,1fr) 64px' }}>
                    <span className={s.posts === ch.maxPosts ? 'font-bold' : 'font-medium'}>{t(dc.sizesPosts, { count: s.posts })}</span>
                    <div className="bar" style={{ height: 8 }} role="img" aria-label={kb(s.bytes)}>
                      <i style={{ width: `${Math.min(100, (s.bytes / biggest) * 100).toFixed(1)}%`, background: big ? '#C2410C' : 'var(--accent)' }} />
                    </div>
                    <span className={cn('num text-right font-semibold', big ? 'text-[#C2410C]' : 'text-ink')}>{kb(s.bytes)}</span>
                  </div>
                )
              })}
            </div>
            <span className="k">{sizesNote}</span>
          </section>
        ) : null}
        <section className="flex flex-col gap-0.5" aria-label={t(dc.postsTitle)}>
          <span className="font-semibold">{t(dc.postsTitle)}</span>
          <span className="k">{t(dc.postsSub, { count: r.items.length, next: mode.digest?.next?.posts ?? 0 })}</span>
          <div className="mt-1 flex flex-col">
            {r.items.map((x, i) => (
              <FeedItemRow key={x.id} item={x} withTime={i === 0} className="border-b border-line-2" />
            ))}
          </div>
        </section>
      </div>
    </div>
  )
}

/**
 * Under the canvas, when the preview has posts without an image (DigestComposer
 * "A post without an image"): the block's fallback image is used, with the
 * post's title as alt text.
 */
export function AltTextNote() {
  const t = useT()
  const mode = useDigestMode()
  const r = mode?.render
  if (!mode || !r || mode.preview === 'six' || !showsImages(mode.block)) return null
  const fallback = r.items.map((x, i) => ({ x, n: i + 1 })).filter(({ x }) => x.fallback)
  if (!fallback.length) return null
  const fromFeed = r.items.map((x, i) => ({ x, n: i + 1 })).filter(({ x }) => x.image && !x.altFromTitle)
  const fromTitle = r.items.map((x, i) => ({ x, n: i + 1 })).filter(({ x }) => x.image && x.altFromTitle)
  const file = mode.block.fallbackName || mode.block.fallback?.split('/').pop() || ''
  return (
    <div className="flex flex-col gap-2.5" style={{ marginTop: 16 }}>
      <section className="note w" role="note" aria-label={t(dc.altTitle)}>
        <span className="text-warn-fg">
          <Icon name="alert" size={16} />
        </span>
        <div className="flex min-w-0 flex-col gap-1">
          <span className="font-bold">{t(dc.altTitle)}</span>
          <span>
            {t(dc.altBody, {
              titles: formatList(fallback.map(({ x }) => `“${x.title}”`)),
              count: fallback.length,
              file,
              source: mode.digest?.source ?? 'url',
            })}
          </span>
        </div>
      </section>
      <div className="flex flex-wrap gap-2">
        {fromFeed.length ? <Badge tone="ok">{t(dc.altFeed, { count: fromFeed.length, items: formatList(fromFeed.map(({ n }) => String(n))) })}</Badge> : null}
        {fromTitle.length ? <Badge tone="warn">{t(dc.altFromTitleItems, { count: fromTitle.length, items: formatList(fromTitle.map(({ n }) => String(n))) })}</Badge> : null}
      </div>
    </div>
  )
}
