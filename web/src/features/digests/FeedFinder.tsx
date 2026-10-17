import { Link } from '@tanstack/react-router'
import { useState, type ReactNode } from 'react'
import { errorMessage, http, type Schemas } from '@/api/client'
import { Badge } from '@/components/ui/Badge'
import { Button } from '@/components/ui/Button'
import { Icon } from '@/components/icons/Icon'
import { toast } from '@/components/ui/Toast'
import { T, useT } from '@/i18n'
import { useWorkspace } from '@/lib/session'
import { FeedItemRow } from './FeedItemRow'
import { s } from './settingsMessages'
import { dateTime } from './shared'

type Found = Schemas['DigestFound']
type Site = Schemas['DigestSite']

const mono = (c: ReactNode[]) => <span className="mono">{c}</span>

/** "blog.acme.io" from an address as typed. */
export function hostOf(address: string): string {
  const v = address.trim()
  try {
    return new URL(/^[a-z]+:\/\//i.test(v) ? v : `https://${v}`).host
  } catch {
    return v
  }
}

const kb = (n: number) => `${Math.max(1, Math.round(n / 1024))} KB`

/**
 * Finding a feed from an address (DigestSetupStates): POST …/digests/find
 * reads it through the outbound guard and finds the feed a page links to,
 * follows a moved feed, or says why it couldn't.
 */
export function useFeedFinder() {
  const { slug } = useWorkspace()
  const [finding, setFinding] = useState<string | null>(null)
  const [found, setFound] = useState<Found | null>(null)
  const find = async (address: string) => {
    if (!address.trim()) return
    setFinding(address)
    setFound(null)
    try {
      const { data } = await http.POST('/api/v1/workspaces/{workspace}/digests/find', { params: { path: { workspace: slug } }, body: { address } })
      setFound(data ?? null)
    } catch (e) {
      toast({ kind: 'error', title: errorMessage(e) })
    } finally {
      setFinding(null)
    }
  }
  return { finding, found, find, clear: () => setFound(null) }
}

/** The badge in the Feed card's header while a find is shown. */
export function FoundBadge({ found }: { found: Found }) {
  const t = useT()
  if (found.ok && found.moved) return <Badge tone="warn">{t(s.movedBadge)}</Badge>
  if (found.ok) {
    return (
      <Badge tone="ok" dot>
        {t(s.foundBadge)}
      </Badge>
    )
  }
  return (
    <Badge tone="bad" dot>
      {found.error?.kind === 'not_feed' ? t(s.notFeedBadge) : t(s.readFailedBadge)}
    </Badge>
  )
}

/** "Find feed", beside the address. */
export function FindButton({ busy, disabled, onFind }: { busy: boolean; disabled?: boolean; onFind: () => void }) {
  const t = useT()
  return (
    <button type="button" className="btn" style={{ minWidth: 112 }} disabled={busy || disabled} aria-busy={busy || undefined} onClick={onFind}>
      {busy ? <span className="spin" aria-hidden="true" /> : null}
      {busy ? t(s.finding) : t(s.findFeed)}
    </button>
  )
}

/** What the find found, under the source choice: the feed, a move, or why it couldn't be read. */
export function FoundNote({
  address,
  finding,
  found,
  sites,
  onUse,
  onKeep,
  onRetry,
  onUseSite,
}: {
  address: string
  finding: string | null
  found: Found | null
  sites: Site[]
  onUse: (url: string) => void
  onKeep: () => void
  onRetry: () => void
  onUseSite: () => void
}) {
  const t = useT()
  const [preview, setPreview] = useState(false)
  if (finding) return <span className="k text-ink-2">{t(s.findingBody, { host: hostOf(finding) })}</span>
  if (!found) return null
  const host = hostOf(address)
  if (found.ok && found.moved) {
    const first = found.hops[0]
    return (
      <>
        <Note tone="w" icon="alert" title={t(s.movedTitle)}>
          <T msg={s.movedBody} values={{ status: first?.status ?? 301, url: found.moved, mono }} />
          <Actions>
            <Button size="sm" variant="primary" onClick={() => onUse(found.moved!)}>
              {t(s.useNew)}
            </Button>
            <Button size="sm" onClick={onKeep}>
              {t(s.keepOld)}
            </Button>
          </Actions>
        </Note>
        <Hops found={found} />
      </>
    )
  }
  if (found.ok) {
    const fromPage = Boolean(found.page) && found.url !== address.trim()
    return (
      <>
        <Note tone="g" icon="check" title={fromPage ? t(s.foundTitle, { host }) : t(s.foundBadge)}>
          <T msg={fromPage ? s.foundBody : s.foundDirectBody} values={{ url: found.url ?? '', kind: found.kind || 'rss', count: found.items, host, mono }} />
          <Actions>
            {fromPage && found.url ? (
              <Button size="sm" variant="primary" onClick={() => onUse(found.url!)}>
                {t(s.useFeed)}
              </Button>
            ) : null}
            {found.preview.length ? (
              <Button size="sm" aria-expanded={preview} onClick={() => setPreview(!preview)}>
                {preview ? t(s.hideItems) : t(s.previewItems)}
              </Button>
            ) : null}
          </Actions>
        </Note>
        {preview ? (
          <div className="flex flex-col rounded-lg border border-line px-3.5">
            {found.preview.map((it, i) => (
              <FeedItemRow key={it.id} item={it} withTime={i < 2} className="border-b border-line-2 last:border-b-0" />
            ))}
          </div>
        ) : null}
        {found.hops.some((h) => h.status >= 300 && h.status < 400) ? <Hops found={found} /> : null}
      </>
    )
  }
  const e = found.error
  const notFeed = e?.kind === 'not_feed'
  // The server's sentence, ended, before what it means for the digest.
  const detail = (e?.detail ?? '').trim()
  const ended = !detail || /[.!?…]$/.test(detail) ? detail : `${detail}.`
  return (
    <Note tone="b" icon="alert" title={notFeed ? t(s.notFeedTitle) : t(s.firstReadTitle)}>
      {notFeed ? ended : t(s.firstReadBody, { detail: ended })}
      <Actions>
        <Button size="sm" onClick={onRetry}>
          {t(s.tryAgain)}
        </Button>
        {notFeed && e?.url ? (
          <a className="btn sm" href={e.url} target="_blank" rel="noreferrer noopener">
            {t(s.openAddress)}
          </a>
        ) : sites.length ? (
          <Button size="sm" onClick={onUseSite}>
            {t(s.useSite)}
          </Button>
        ) : null}
      </Actions>
      {notFeed && e?.lines.length ? (
        <ol className="mono m-0 mt-2 list-none overflow-x-auto rounded-md border border-line bg-surface p-0 text-[12px]" aria-label={t(s.notFeedTitle)}>
          {e.lines.map((line, i) => (
            <li key={i} className="flex gap-3 px-3 py-0.5">
              <span className="w-4 shrink-0 text-right text-ink-3">{i + 1}</span>
              <span className="whitespace-pre">{line}</span>
            </li>
          ))}
        </ol>
      ) : null}
    </Note>
  )
}

function Note({ tone, icon, title, children }: { tone: 'g' | 'w' | 'b' | 'i'; icon: 'check' | 'alert' | 'info'; title: string; children: ReactNode }) {
  return (
    <div className={`note ${tone}`} role="note" style={{ flexWrap: 'nowrap' }}>
      <span className="mt-0.5 shrink-0">
        <Icon name={icon} size={16} />
      </span>
      <div className="flex min-w-0 flex-[1_1_auto] flex-col gap-[3px]">
        <span className="font-bold">{title}</span>
        <div>{children}</div>
      </div>
    </div>
  )
}

function Actions({ children }: { children: ReactNode }) {
  return <div className="mt-2 flex flex-wrap items-center gap-2">{children}</div>
}

/** "Redirects followed · up to 3": each address answered on the way to the feed. */
function Hops({ found }: { found: Found }) {
  const t = useT()
  return (
    <div className="flex flex-col gap-0.5">
      <span className="lbl">{t(s.hops)}</span>
      <ol className="m-0 flex list-none flex-col p-0">
        {found.hops.map((h, i) => (
          <li key={i} className="flex items-center gap-3 border-b border-line-2 py-2 last:border-b-0">
            <span className={`grid h-7 w-7 shrink-0 place-items-center rounded-full ${h.status >= 300 ? 'bg-info-tint text-info-fg' : 'bg-ok-tint text-ok-fg'}`}>
              <Icon name={h.status >= 300 ? 'arrowRight' : 'check'} size={13} />
            </span>
            <span className="flex min-w-0 flex-1 flex-col">
              <span className="truncate font-semibold">{h.url.replace(/^https?:\/\//, '')}</span>
              <span className="k">
                {h.status >= 300 ? t(s.hopRedirect, { status: String(h.status) }) : t(s.hopFeed, { kind: found.kind || 'rss', count: found.items, size: kb(found.bytes) })}
              </span>
            </span>
            <span className="k shrink-0">{t(s.hopNumber, { number: i + 1 })}</span>
          </li>
        ))}
      </ol>
    </div>
  )
}

/** A connected WordPress site that stopped reporting, or whose plugin is outdated (DigestSetupStates). */
export function SiteStatusNote({ site }: { site: Site }) {
  const t = useT()
  const { slug } = useWorkspace()
  if (site.status === 'silent') {
    return (
      <Note tone="w" icon="alert" title={t(s.siteSilentTitle)}>
        {site.lastReportAt ? t(s.siteSilentBody, { host: site.host, date: dateTime(site.lastReportAt) }) : t(s.siteSilentNever, { host: site.host })}
        <Actions>
          <Link to="/w/$workspace/settings/integrations/wordpress" params={{ workspace: slug }} className="btn sm">
            {t(s.openSite, { host: site.host })}
          </Link>
        </Actions>
      </Note>
    )
  }
  if (site.status === 'outdated') {
    return (
      <Note tone="i" icon="info" title={t(s.siteOutdatedTitle)}>
        {t(s.siteOutdatedBody, { latest: site.latestPlugin, version: site.pluginVersion })}
      </Note>
    )
  }
  return null
}

/** "WordPress · plugin 1.2.0" with the site's state. */
export function SiteBadge({ site }: { site: Site }) {
  const t = useT()
  if (site.status === 'silent') {
    return (
      <Badge tone="warn" dot>
        {t(s.siteSilentBadge)}
      </Badge>
    )
  }
  if (site.status === 'outdated') return <Badge tone="info">{t(s.siteOutdatedBadge)}</Badge>
  return <span className="k">{t(s.pluginLine, { version: site.pluginVersion || '–' })}</span>
}

