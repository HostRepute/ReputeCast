import { Link } from '@tanstack/react-router'
import type { Component } from 'grapesjs'
import type { ReactNode } from 'react'
import { $api, type Schemas } from '@/api/client'
import { T, useT } from '@/i18n'
import { useComposer } from './context'
import type { BlockDef } from './editor/blocks'
import { syncedSectionOf } from './editor/synced'
import { mbk } from './brandkit.messages'

/*
 * The brand kit in the composer (CO-11, ComposerEmpty · Composer · ComposerWrite):
 * the gallery's "Start from your brand kit" banner, the synced Header and
 * Footer as blocks, and synced sections kept read-only on the canvas (they
 * change in Settings › Brand kit, and every draft holding them follows).
 */

/** The workspace's brand kit (defaults until it's saved). */
export function useBrandKit() {
  const c = useComposer()
  return $api.useQuery('get', '/api/v1/workspaces/{workspace}/brand-kit', { params: { path: { workspace: c.slug } } }, { staleTime: 60_000 })
}

/** "Start from your brand kit" (ComposerEmpty), until the kit is saved. */
export function BrandKitBanner() {
  const t = useT()
  const c = useComposer()
  const kit = useBrandKit().data
  if (!kit || kit.saved) return null
  return (
    <div className="bkbanner">
      <span className="bki" aria-hidden="true">
        <svg width="18" height="18" viewBox="0 0 16 16" fill="none" stroke="currentColor" strokeWidth="1.6" strokeLinecap="round" strokeLinejoin="round">
          <path d="M8 1.8a6.2 6.2 0 1 0 0 12.4c.9 0 1.3-.6 1.3-1.2 0-.8-.6-1.1-.6-1.8 0-.7.6-1.1 1.3-1.1h1.3a2.9 2.9 0 0 0 2.9-2.9C14.2 4.4 11.4 1.8 8 1.8z" />
          <circle cx="5" cy="7" r=".8" />
          <circle cx="8" cy="4.8" r=".8" />
          <circle cx="11" cy="7" r=".8" />
        </svg>
      </span>
      <span className="bkt">
        <T msg={mbk.banner} values={{ b: (chunks: ReactNode) => <b>{chunks}</b> }} />
      </span>
      <Link to="/w/$workspace/settings/brand-kit" params={{ workspace: c.slug }} className="btn">
        {t(mbk.setUp)}
      </Link>
    </div>
  )
}

/** "Shown in default colours until you set up the brand kit", over the starters. */
export function useDefaultColoursNote(): string | null {
  const t = useT()
  const kit = useBrandKit().data
  return kit && !kit.saved ? t(mbk.defaultColours) : null
}

/** The synced Header and Footer as blocks (ReputeCast blocks, "Synced"). */
export function syncedBlockDefs(kit: Schemas['BrandKit'] | undefined, t: ReturnType<typeof useT>): BlockDef[] {
  return (kit?.synced ?? []).map((b) => ({
    id: `rc-synced-${b.kind}`,
    label: b.kind === 'header' ? t(mbk.syncedHeader) : t(mbk.syncedFooter),
    badge: t(mbk.synced),
    group: 'reputecast' as const,
    icon: b.kind === 'header' ? 'rcHeader' : 'rcFooter',
    kind: 'section' as const,
    content: b.mjml,
    keywords: 'synced brand kit header footer',
  }))
}

/** In the settings of a synced section's block: what it is and where it changes (ComposerWrite · "Edit in Brand kit"). */
export function SyncedNotice({ component }: { component: Component | null }) {
  const t = useT()
  const c = useComposer()
  const synced = syncedSectionOf(component)
  if (!synced) return null
  return (
    <div className="note i" role="note" style={{ margin: '12px 16px 0' }}>
      <div className="flex flex-col gap-1.5">
        <span>
          <b>{synced.kind === 'header' ? t(mbk.syncedHeaderNotice) : t(mbk.syncedFooterNotice)}</b> {t(mbk.syncedNoticeBody)}
        </span>
        <Link to="/w/$workspace/settings/brand-kit" params={{ workspace: c.slug }} className="font-semibold underline">
          {t(mbk.editInBrandKit)}
        </Link>
      </div>
    </div>
  )
}
