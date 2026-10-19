import { useQueryClient } from '@tanstack/react-query'
import type { Component, Editor } from 'grapesjs'
import { createContext, useCallback, useContext, useMemo, useState, type ReactNode } from 'react'
import { $api, http } from '@/api/client'
import { Icon } from '@/components/icons/Icon'
import { useT } from '@/i18n'
import { clock, dayMonthDate } from '@/lib/format'
import { useCan, useWorkspace } from '@/lib/session'
import { useComposer, type Campaign } from '@/features/composer/context'
import { markerOf, sectionOf, tagOf } from '@/features/composer/editor/labels'
import { conditional, ruleOf } from '@/features/conditional/rules'
import { useConditional } from '@/features/conditional/state'
import { useWebsite, type CampaignWeb, type Post, type PostInput } from '../api'
import { mc } from './messages'
import { PreviewOnWeb } from './PreviewOnWeb'
import { useWebSettingsOf, WebPanel, type WebSettings } from './WebPanel'

/**
 * The campaign composer's web side (ComposerPublishToWeb · campaign mode):
 * while the website is on, the Web chip in the header, the Web tab, the
 * Email · Web switch over the canvas, Email only badges on the canvas and
 * "Show on" in a block's Visibility.
 */
interface WebMode {
  web: CampaignWeb
  settings: WebSettings
  host: string
  /** The canvas shows the email (badges on email-only blocks) or the post as the web shows it. */
  canvas: 'email' | 'web'
  setCanvas: (v: 'email' | 'web') => void
}

const Ctx = createContext<WebMode | null>(null)

/** The web side of the campaign being edited; null while the website is off (or for a member who can't see it). */
export function useWebMode(): WebMode | null {
  return useContext(Ctx)
}

const CAMPAIGN_WEB = '/api/v1/workspaces/{workspace}/campaigns/{campaignId}/web' as const

/**
 * Loads the campaign's web state once the website is on, and provides it to
 * the composer's parts. The tree under it never changes shape (the canvas
 * keeps its editor): only the context's value does.
 */
export function WebModeProvider({ campaign, children }: { campaign: Campaign; children: ReactNode }) {
  const { slug } = useWorkspace()
  const qc = useQueryClient()
  const canView = useCan('website:view')
  const overview = useWebsite(canView && !campaign.digestRole)
  const on = overview.data?.status === 'on'
  const web = $api.useQuery('get', CAMPAIGN_WEB, { params: { path: { workspace: slug, campaignId: campaign.id } } }, { enabled: on && canView, refetchOnWindowFocus: false })
  const [state, setState] = useState<CampaignWeb | null>(null)
  const current = state ?? web.data ?? null
  const save = useCallback(
    async (input: PostInput): Promise<Post> => {
      const { data } = await http.PUT(CAMPAIGN_WEB, { params: { path: { workspace: slug, campaignId: campaign.id } }, body: input })
      setState(data!)
      void qc.invalidateQueries({ predicate: (q) => typeof q.queryKey[1] === 'string' && ((q.queryKey[1] as string).includes('/campaigns-web') || (q.queryKey[1] as string).includes('/website/posts')) })
      return data!.post
    },
    [slug, campaign.id, qc],
  )
  const ready = Boolean(on && current?.siteOn && overview.data)
  const settings = useWebSettingsOf({ post: ready ? current!.post : undefined, save })
  const [canvas, setCanvas] = useState<'email' | 'web'>('email')
  const host = overview.data ? overview.data.address.domain || overview.data.address.host : ''
  const value = useMemo(() => (ready && settings && current ? { web: current, settings, host, canvas, setCanvas } : null), [ready, settings, current, host, canvas])
  return <Ctx.Provider value={value}>{children}</Ctx.Provider>
}

/** "🌐 Web · ● Publishes with the send": opens the Web tab. */
export function WebChip() {
  const t = useT()
  const c = useComposer()
  const w = useWebMode()
  if (!w) return null
  const d = w.settings.draft
  const p = w.settings.post
  let state: string
  if (!d.webOn) state = t(mc.chipOff)
  else if (p.status === 'published') state = t(mc.chipPublished)
  else if (p.status === 'unpublished') state = t(mc.chipUnpublished)
  else if (d.publishMode === 'at' && d.publishAt) state = t(mc.chipScheduled, { date: `${dayMonthDate(d.publishAt)}, ${clock(d.publishAt)}` })
  else state = t(mc.chipWithSend)
  const open = c.rightTab === 'web'
  return (
    <button type="button" className={`wchip${open ? ' on' : ''}`} aria-label={t(mc.chipLabel, { state })} aria-pressed={open} onClick={() => c.setRightTab('web')}>
      <Icon name="globe" size={14} />
      {t(mc.web)}
      <span className="wcs">
        <span className="dot" aria-hidden="true" style={{ color: d.webOn ? (p.status === 'published' ? 'var(--ok-bar)' : 'var(--info-fg)') : 'var(--ink-4)' }} />
        {state}
      </span>
    </button>
  )
}

/** The Web tab of the campaign composer's right panel. */
export function WebTabPanel() {
  const c = useComposer()
  const w = useWebMode()
  if (!w) return null
  const hero = heroAlt(c.editor)
  return (
    <WebPanel
      ws={w.settings}
      mode="campaign"
      host={w.host}
      canEdit={c.canEdit && c.campaign.access.editable}
      campaign={{ lists: w.web.audienceLists, scheduledAt: w.web.scheduledAt, sentAt: w.web.sentAt, status: w.web.campaignStatus, followsAudience: w.web.followsAudience, heroAlt: hero }}
      onShowHidden={() => {
        w.setCanvas('email')
        c.setRightTab('style')
      }}
    />
  )
}

/** The alt text of the email's first image (the post's cover by default). */
function heroAlt(editor: Editor | null): string | undefined {
  const img = editor?.getWrapper()?.findType('mj-image')[0] as Component | undefined
  const alt = img ? String((img.getAttributes() as Record<string, unknown>).alt ?? '') : ''
  return alt || undefined
}

/** "Email · Web" over the canvas, with how many blocks are email only. */
export function EmailWebToggle() {
  const t = useT()
  const c = useComposer()
  const w = useWebMode()
  // eslint-disable-next-line react-hooks/exhaustive-deps -- rev covers canvas changes
  const count = useMemo(() => emailOnlyBlocks(c.editor).length, [c.editor, c.rev])
  if (!w || !w.settings.draft.webOn) return null
  return (
    <>
      <div className="seg" role="group" aria-label={t(mc.emailWeb)}>
        <button type="button" aria-pressed={w.canvas === 'email'} onClick={() => w.setCanvas('email')}>
          {t(mc.email)}
        </button>
        <button type="button" aria-pressed={w.canvas === 'web'} onClick={() => w.setCanvas('web')}>
          <Icon name="globe" size={13} />
          {t(mc.web)}
        </button>
      </div>
      {count ? <span className="k">{t(mc.emailOnlyCount, { count })}</span> : null}
    </>
  )
}

/** The post as the web shows it, over the canvas (the canvas stays loaded underneath). */
export function WebCanvasView() {
  const w = useWebMode()
  const [as, setAs] = useState<'visitor' | 'free' | 'paid'>('paid')
  if (!w || w.canvas !== 'web' || !w.settings.draft.webOn) return null
  return (
    <div
      className="wcanvas"
      // Under the canvas toolbar, so Email · Web stays at hand.
      ref={(el) => {
        const tools = el?.parentElement?.querySelector<HTMLElement>('.cmp-tools')
        if (el) el.style.top = `${tools ? tools.offsetHeight : 0}px`
      }}
    >
      <PreviewOnWeb post={w.settings.post} as={as} onAs={setAs} onBack={() => w.setCanvas('email')} asMain={false} />
    </div>
  )
}

type EmailOnly = { comp: Component; reason: 'email' | 'default' | 'conditional'; words?: string }

/** The blocks the web leaves out: Show on Email only, the footer, view-online and referral blocks, and conditional blocks. */
export function emailOnlyBlocks(editor: Editor | null, words?: (id: string) => string | undefined): EmailOnly[] {
  const body = editor?.getWrapper()?.findType('mj-body')[0] as Component | undefined
  if (!body) return []
  const out: EmailOnly[] = []
  const walk = (comp: Component) => {
    const tag = tagOf(comp)
    if (!tag) {
      comp.components().forEach((x: Component) => walk(x))
      return
    }
    const attrs = (comp.getAttributes() ?? {}) as Record<string, unknown>
    const showOn = String(attrs['rc-show-on'] ?? '')
    if (showOn === 'email') {
      out.push({ comp, reason: 'email' })
      return
    }
    if (showOn !== 'web' && (tag === 'mj-section' || tag === 'mj-wrapper' || tag === 'mj-hero')) {
      const marker = markerOf(comp)
      const html = String(comp.toHTML?.() ?? '')
      if (marker === 'footer' || marker === 'referral' || marker === 'referral-block' || marker === 'view-online' || marker === 'view-in-browser' || /unsubscribe_url|view_in_browser_url|referral_url/.test(html)) {
        out.push({ comp, reason: 'default' })
        return
      }
    }
    const rule = ruleOf(comp)
    if (conditional(rule)) {
      out.push({ comp, reason: 'conditional', words: words?.(rule.id) })
      return
    }
    comp.components().forEach((x: Component) => walk(x))
  }
  walk(body)
  return out
}

interface Box {
  top: number
  left: number
  width: number
  height: number
}

/** "⊘ Email only" on each block the web leaves out, while the canvas shows the email. */
export function WebOverlays({ host, tick, boxOf }: { host: HTMLElement | null; tick: number; boxOf: (editor: Editor, el: Element | null | undefined, host: HTMLElement | null) => Box | null }) {
  const t = useT()
  const c = useComposer()
  const w = useWebMode()
  const cond = useConditional()
  const ed = c.editor
  const placed = useMemo(() => {
    if (!ed || !w || !w.settings.draft.webOn || w.canvas !== 'email') return []
    const out: { key: string; box: Box; label: string; comp: Component }[] = []
    for (const b of emailOnlyBlocks(ed, (id) => cond?.blockInfo(id)?.words)) {
      const box = boxOf(ed, b.comp.getEl(), host)
      if (!box) continue
      out.push({ key: b.comp.cid, box, comp: b.comp, label: b.reason === 'conditional' && b.words ? `${t(mc.emailOnlyBadge)} · ${b.words}` : t(mc.emailOnlyBadge) })
    }
    return out
    // eslint-disable-next-line react-hooks/exhaustive-deps -- tick follows the canvas
  }, [ed, host, tick, w?.canvas, w?.settings.draft.webOn, cond?.report])
  if (!placed.length) return null
  return (
    <>
      {placed.map((p) => (
        <span key={p.key} aria-hidden="true">
          <span className="wonly-box" style={{ top: p.box.top, left: p.box.left, width: p.box.width, height: p.box.height }} />
          <button
            type="button"
            className="wonly-badge"
            style={{ top: p.box.top + 6, left: p.box.left + p.box.width - 8 }}
            tabIndex={-1}
            onClick={() => {
              ed?.select(sectionOf(p.comp) ?? p.comp)
              c.setRightTab('settings')
            }}
          >
            <EyeOffSmall />
            {p.label}
          </button>
        </span>
      ))}
    </>
  )
}

function EyeOffSmall() {
  return (
    <svg width="12" height="12" viewBox="0 0 16 16" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
      <path d="M2 2l12 12M6.6 6.6a2 2 0 0 0 2.8 2.8M4.2 4.3C2.6 5.3 1.5 8 1.5 8s2.5 4.5 6.5 4.5c1.3 0 2.4-.4 3.3-1M7 3.6c.3 0 .7-.1 1-.1 4 0 6.5 4.5 6.5 4.5s-.5 1-1.5 2.1" />
    </svg>
  )
}

/** Where Show on is offered: sections and content blocks. */
const SHOW_ON = new Set(['mj-section', 'mj-wrapper', 'mj-hero', 'mj-text', 'mj-image', 'mj-button', 'mj-divider', 'mj-spacer', 'mj-social', 'mj-navbar', 'mj-raw', 'mj-table'])

/**
 * "Show on · Email and web · Email only · Web only" in a block's Visibility
 * (ComposerPublishToWeb · Composer › Visibility, "Show on"), while the
 * website is on.
 */
export function ShowOnProp({ c: comp }: { c: Component }) {
  const t = useT()
  const ctx = useComposer()
  const w = useWebMode()
  const cond = useConditional()
  if (!w || !SHOW_ON.has(tagOf(comp))) return null
  const attrs = (comp.getAttributes() ?? {}) as Record<string, unknown>
  const raw = String(attrs['rc-show-on'] ?? '')
  const kind = defaultKind(comp)
  const value: 'both' | 'email' | 'web' = raw === 'email' || raw === 'web' ? raw : raw === 'both' ? 'both' : kind ? 'email' : 'both'
  const set = (v: 'both' | 'email' | 'web') => {
    const next = { ...attrs }
    // A footer and its kin are Email only by default: "both" is written to say otherwise.
    if (v === (kind ? 'email' : 'both')) delete next['rc-show-on']
    else next['rc-show-on'] = v
    comp.setAttributes(next)
  }
  const rule = ruleOf(comp)
  const words = rule && conditional(rule) ? cond?.blockInfo(rule.id)?.words : undefined
  return (
    <div className="prop wshow" role="group" aria-labelledby={`show-on-${comp.cid}`}>
      <span className="t" id={`show-on-${comp.cid}`}>
        {t(mc.showOn)}
      </span>
      <div className="seg" role="group" aria-label={t(mc.showOn)}>
        {(['both', 'email', 'web'] as const).map((v) => (
          <button key={v} type="button" aria-pressed={value === v} disabled={!ctx.canEdit} onClick={() => set(v)}>
            {v === 'both' ? t(mc.showBoth) : v === 'email' ? t(mc.showEmail) : t(mc.showWeb)}
          </button>
        ))}
      </div>
      {words ? (
        <div className="wshow-note">
          <b>{t(mc.showOnCondition, { name: words })}</b>
          <span className="k">{t(mc.showOnConditionText)}</span>
        </div>
      ) : null}
      <span className="k">{t(mc.showOnDefaults)}</span>
    </div>
  )
}

/** Whether a block is Email only by default (a footer, view-online or referral block). */
function defaultKind(comp: Component): boolean {
  const sec = tagOf(comp) === 'mj-section' || tagOf(comp) === 'mj-wrapper' ? comp : null
  if (!sec) return false
  const marker = markerOf(sec)
  if (marker === 'footer' || marker === 'referral' || marker === 'referral-block' || marker === 'view-online' || marker === 'view-in-browser') return true
  return /unsubscribe_url|view_in_browser_url|referral_url/.test(String(sec.toHTML?.() ?? ''))
}
