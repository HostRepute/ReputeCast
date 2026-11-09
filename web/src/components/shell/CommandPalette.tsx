import { keepPreviousData } from '@tanstack/react-query'
import { useNavigate } from '@tanstack/react-router'
import { Dialog as RDialog } from 'radix-ui'
import { useEffect, useId, useMemo, useRef, useState, type ReactNode } from 'react'
import { $api, errorMessage, type Schemas } from '@/api/client'
import { t as tt, useT } from '@/i18n'
import { cn } from '@/lib/cn'
import { shortDate } from '@/lib/format'
import { useDebounced } from '@/lib/hooks'
import { useOptionalWorkspace } from '@/lib/session'
import { m as ma } from '@/features/appearance/messages'
import { useSaveTheme, useTheme } from '@/features/appearance/useTheme'
import { settingRows } from '@/features/settings/settingsIndex'
import { settingsTabs } from '@/features/settings/SettingsTabs'
import { m } from './messages'
import { NAV } from './nav'
import { mp } from './palette.messages'

type Kind = Schemas['SearchGroup']['kind']
type Hit = Schemas['SearchHit']

/** What an entry opens: a router destination. */
interface Dest {
  to: string
  params?: Record<string, string>
  search?: Record<string, string>
}

/** One row of the palette. */
export interface Entry {
  key: string
  kind: Kind | 'goto' | 'action'
  title: string
  detail: string
  dest: Dest
  /** Kept in Recent when opened (search results are; places aren't). */
  remember?: boolean
  /** A single setting: shown only for what is typed, not among the places to go. */
  setting?: boolean
  /** A command (the theme): runs instead of opening dest, and is found by its title or detail. */
  run?: () => void
}

/** The permission that opens each kind; Recent leaves out what the role can't open any more. */
const kindPerm: Record<Kind, string> = {
  campaigns: 'campaigns:view',
  contacts: 'subscribers:view',
  automations: 'automations:view',
  lists: 'subscribers:view',
  segments: 'subscribers:view',
  forms: 'subscribers:view',
  pages: 'website:view',
}

const kindGroup = { campaigns: mp.campaigns, contacts: mp.contacts, automations: mp.automations, lists: mp.lists, segments: mp.segments, forms: mp.forms, pages: mp.pages }
const kindWord = {
  campaigns: mp.kindCampaign,
  contacts: mp.kindContact,
  automations: mp.kindAutomation,
  lists: mp.kindList,
  segments: mp.kindSegment,
  forms: mp.kindForm,
  pages: mp.kindPage,
}
const kindIcon: Record<Kind | 'goto' | 'action', string> = { campaigns: 'C', contacts: '@', automations: 'A', lists: 'L', segments: 'S', forms: 'F', pages: 'P', goto: '→', action: '◐' }

const reports = { email: 'report', sms: 'sms-report', push: 'push-report', whatsapp: 'whatsapp-report' } as const
const editors = { sms: '/w/$workspace/campaigns/$campaignId/sms', push: '/w/$workspace/campaigns/$campaignId/push', whatsapp: '/w/$workspace/campaigns/$campaignId/whatsapp' } as const

/** Where a search result opens: a campaign where its status goes. */
export function destOf(kind: Kind, h: Hit, workspace: string): Dest {
  const p = { workspace }
  switch (kind) {
    case 'campaigns': {
      const params = { ...p, campaignId: h.id }
      const channel = (h.channel ?? 'email') as keyof typeof reports
      if (h.status === 'sending' || h.status === 'paused') return { to: '/w/$workspace/campaigns/$campaignId/live', params }
      if (h.status === 'sent' || h.status === 'cancelled' || h.status === 'failed') return { to: `/w/$workspace/campaigns/$campaignId/${reports[channel] ?? 'report'}`, params }
      return { to: channel in editors ? editors[channel as keyof typeof editors] : '/w/$workspace/campaigns/$campaignId/edit', params }
    }
    case 'contacts':
      return { to: '/w/$workspace/audience/contacts/$contactId', params: { ...p, contactId: h.id } }
    case 'automations':
      return { to: '/w/$workspace/automations/$automationId', params: { ...p, automationId: h.id } }
    case 'lists':
      return { to: '/w/$workspace/audience', params: p, search: { list: h.id } }
    case 'segments':
      return { to: '/w/$workspace/audience/segments/$segmentId', params: { ...p, segmentId: h.id } }
    case 'forms':
      return { to: '/w/$workspace/forms', params: p, search: { form: h.id } }
    case 'pages':
      return { to: '/w/$workspace/forms/pages/$pageId', params: { ...p, pageId: h.id } }
  }
}

/** A result's second line: "Campaign · sent Oct 14 · 48,213 recipients". */
export function detailOf(kind: Kind, h: Hit): string {
  const parts = [tt(kindWord[kind])]
  if (kind === 'campaigns') {
    if (h.at && ['sent', 'sending', 'paused', 'cancelled', 'failed'].includes(h.status ?? '')) parts.push(tt(mp.campaignSent, { date: shortDate(h.at) }))
    else if (h.at && h.status === 'scheduled') parts.push(tt(mp.campaignScheduled, { date: shortDate(h.at) }))
    else if (h.status) parts.push(h.status)
    if (h.count) parts.push(tt(mp.recipients, { count: h.count }))
  } else if (kind === 'contacts') {
    if (h.detail) parts.push(h.detail)
  } else if (kind === 'lists') {
    if (h.count !== undefined) parts.push(tt(mp.subscribed, { count: h.count }))
  } else if (kind === 'segments') {
    if (h.count !== undefined) parts.push(tt(mp.contactsCount, { count: h.count }))
  } else if (h.status) {
    parts.push(h.status)
  }
  return parts.join(' · ')
}

// ── Recent ───────────────────────────────────────────────────────────────

const RECENT_MAX = 8
const recentKey = (workspace: string) => `rc.palette.recent.${workspace}`

function readRecent(workspace: string): Entry[] {
  try {
    const v = JSON.parse(window.localStorage.getItem(recentKey(workspace)) ?? '[]') as Entry[]
    return Array.isArray(v) ? v.slice(0, RECENT_MAX) : []
  } catch {
    return []
  }
}

function remember(workspace: string, e: Entry) {
  try {
    const next = [e, ...readRecent(workspace).filter((r) => r.key !== e.key)].slice(0, RECENT_MAX)
    window.localStorage.setItem(recentKey(workspace), JSON.stringify(next))
  } catch {
    // Private mode or full storage: Recent just stays as it was.
  }
}

// ── Opening ──────────────────────────────────────────────────────────────

const OPEN_EVENT = 'rc:open-palette'

/** Opens the palette from anywhere (a search button in a header). */
export function openPalette() {
  window.dispatchEvent(new Event(OPEN_EVENT))
}

/** Highlights the text in a title, case-insensitively. */
function Marked({ text, q }: { text: string; q: string }) {
  const i = q ? text.toLowerCase().indexOf(q.toLowerCase()) : -1
  if (i < 0) return <>{text}</>
  return (
    <>
      {text.slice(0, i)}
      <mark>{text.slice(i, i + q.length)}</mark>
      {text.slice(i + q.length)}
    </>
  )
}

/**
 * The command palette on ⌘K (CommandPalette board; review 2026-11 FE-1):
 * one search over campaigns, contacts, automations, lists, segments, forms
 * and pages, the places the role can open, and Recent. The server searches
 * only the kinds the role can open.
 */
export function CommandPalette({ open, onOpenChange }: { open: boolean; onOpenChange: (o: boolean) => void }) {
  const ws = useOptionalWorkspace()
  useEffect(() => {
    const show = () => onOpenChange(true)
    window.addEventListener(OPEN_EVENT, show)
    return () => window.removeEventListener(OPEN_EVENT, show)
  }, [onOpenChange])
  if (!ws) return null
  return (
    <RDialog.Root open={open} onOpenChange={onOpenChange}>
      <RDialog.Portal>
        <RDialog.Overlay className="ovl pal-ovl">
          <RDialog.Content className="pal" aria-describedby={undefined} onOpenAutoFocus={(e) => e.preventDefault()}>
            {open ? <Palette workspace={ws.slug} permissions={ws.me.permissions} close={() => onOpenChange(false)} /> : null}
          </RDialog.Content>
        </RDialog.Overlay>
      </RDialog.Portal>
    </RDialog.Root>
  )
}

function Palette({ workspace, permissions, close }: { workspace: string; permissions: readonly string[]; close: () => void }) {
  const t = useT()
  const navigate = useNavigate()
  const id = useId()
  const input = useRef<HTMLInputElement>(null)
  const [q, setQ] = useState('')
  const term = useDebounced(q.trim(), 200)
  const can = (perm: string) => permissions.includes(perm)
  const search = $api.useQuery(
    'get',
    '/api/v1/workspaces/{workspace}/search',
    { params: { path: { workspace }, query: { q: term } } },
    { enabled: term.length >= 2, placeholderData: keepPreviousData, staleTime: 15_000 },
  )
  useEffect(() => input.current?.focus(), [])
  const theme = useTheme()
  const saveTheme = useSaveTheme()

  // The places the role can open: sections, New campaign and settings tabs.
  const places = useMemo<Entry[]>(() => {
    const sectionPerm: Record<string, string | undefined> = {
      campaigns: 'campaigns:view',
      automations: 'automations:view',
      transactional: 'transactional:view',
      audience: 'subscribers:view',
      grow: 'grow:view',
      earn: 'earn:view',
      forms: 'subscribers:view',
      deliverability: 'deliverability:view',
      providers: 'providers:view',
    }
    const out: Entry[] = []
    if (can('campaigns:edit')) out.push({ key: 'goto:new-campaign', kind: 'goto', title: tt(mp.newCampaign), detail: tt(m.navCampaigns), dest: { to: '/w/$workspace/campaigns/new', params: { workspace } } })
    for (const n of NAV) {
      const perm = sectionPerm[n.key]
      if (perm && !can(perm)) continue
      out.push({ key: `goto:${n.key}`, kind: 'goto', title: tt(n.label), detail: '', dest: { to: n.to, params: { workspace } } })
    }
    for (const tab of settingsTabs(workspace)) {
      if (typeof tab.label === 'string' && tab.to) out.push({ key: `goto:${tab.to}`, kind: 'goto', title: tab.label, detail: tt(mp.settings), dest: { to: tab.to, params: { workspace } } })
    }
    // Single settings by their own words (review 2026-11 FE-10); only
    // found by typing, after the tabs.
    for (const row of settingRows) {
      const title = tt(row.label)
      out.push({ key: `setting:${row.to}:${title}`, kind: 'goto', title, detail: tt(mp.settingIn, { tab: tt(row.tab) }), dest: { to: row.to, params: { workspace } }, setting: true })
    }
    return out
  }, [workspace, permissions]) // eslint-disable-line react-hooks/exhaustive-deps

  const typed = q.trim()
  const groups: { label: string; entries: Entry[] }[] = []
  if (!typed) {
    const recent = readRecent(workspace).filter((e) => e.kind !== 'goto' && can(kindPerm[e.kind as Kind]))
    if (recent.length) groups.push({ label: t(mp.recent), entries: recent })
    groups.push({ label: t(mp.goTo), entries: places.filter((p) => !p.setting).slice(0, 6) })
  } else {
    if (term === typed || search.data) {
      for (const g of search.data?.groups ?? []) {
        groups.push({
          label: t(mp.groupCount, { kind: t(kindGroup[g.kind]), count: g.items.length }),
          entries: g.items.map((h) => ({ key: `${g.kind}:${h.id}`, kind: g.kind, title: h.title, detail: detailOf(g.kind, h), dest: destOf(g.kind, h, workspace), remember: true })),
        })
      }
    }
    const low = typed.toLowerCase()
    const placeHits = places.filter((p) => !p.setting && p.title.toLowerCase().includes(low)).slice(0, 5)
    if (placeHits.length) groups.push({ label: t(mp.goTo), entries: placeHits })
    const settingHits = places.filter((p) => p.setting && p.title.toLowerCase().includes(low)).slice(0, 5)
    if (settingHits.length) groups.push({ label: t(mp.settings), entries: settingHits })
    // The theme (M54): "dark", "light", "theme" or "appearance" find the other two choices.
    const themeHits = (['dark', 'light', 'system'] as const)
      .filter((c) => c !== theme.choice)
      .map<Entry>((c) => ({
        key: `action:theme:${c}`,
        kind: 'action',
        title: t(c === 'dark' ? ma.useDark : c === 'light' ? ma.useLight : ma.useSystem),
        detail: t(ma.paletteDetail),
        dest: { to: '/w/$workspace/account', params: { workspace } },
        run: () => saveTheme(c, { announce: true }),
      }))
      .filter((e) => `${e.title} ${e.detail}`.toLowerCase().includes(low))
    if (themeHits.length) groups.push({ label: t(ma.title), entries: themeHits })
  }
  const flat = groups.flatMap((g) => g.entries)
  // The highlight starts over on the first row whenever the rows change.
  const rowsKey = `${typed}|${search.dataUpdatedAt}`
  const [sel, setSel] = useState({ key: '', i: 0 })
  const current = sel.key === rowsKey ? Math.min(sel.i, Math.max(flat.length - 1, 0)) : 0
  const setActive = (next: number | ((i: number) => number)) => setSel({ key: rowsKey, i: typeof next === 'function' ? next(current) : next })

  const go = (e: Entry) => {
    if (e.run) {
      close()
      e.run()
      return
    }
    if (e.remember) remember(workspace, e)
    close()
    void navigate({ to: e.dest.to, params: e.dest.params, search: e.dest.search } as never)
  }
  const onKey = (ev: React.KeyboardEvent) => {
    if (ev.key === 'ArrowDown') {
      ev.preventDefault()
      setActive((i) => (flat.length ? (i + 1) % flat.length : 0))
    } else if (ev.key === 'ArrowUp') {
      ev.preventDefault()
      setActive((i) => (flat.length ? (i - 1 + flat.length) % flat.length : 0))
    } else if (ev.key === 'Enter') {
      const e = flat[current]
      if (e) {
        ev.preventDefault()
        go(e)
      }
    }
  }
  const optionId = (i: number) => `${id}-o${i}`
  const searching = typed.length >= 2 && (term !== typed || search.isFetching) && !search.data
  const nothing = typed.length >= 2 && term === typed && search.isSuccess && !flat.length
  let index = -1
  let status: ReactNode = null
  if (search.isError) status = <p className="pal-empty bad">{t(mp.failed, { error: errorMessage(search.error) })}</p>
  else if (nothing)
    status = (
      <div className="pal-empty">
        <b>{t(mp.noResults, { q: typed })}</b>
        <span className="k">{t(mp.noResultsHint)}</span>
      </div>
    )
  else if (searching && !flat.length) status = <p className="pal-empty k">{t(mp.searching)}</p>
  return (
    <>
      <RDialog.Title className="vh">{t(mp.label)}</RDialog.Title>
      <div className="pal-q">
        <svg width="18" height="18" viewBox="0 0 16 16" fill="none" stroke="currentColor" strokeWidth="1.6" aria-hidden="true">
          <circle cx="7" cy="7" r="4.5" />
          <path d="m10.5 10.5 3 3" />
        </svg>
        <input
          ref={input}
          role="combobox"
          aria-expanded="true"
          aria-controls={`${id}-list`}
          aria-autocomplete="list"
          aria-activedescendant={flat.length ? optionId(current) : undefined}
          aria-label={t(mp.placeholder)}
          aria-describedby={`${id}-d`}
          placeholder={t(mp.placeholder)}
          value={q}
          onChange={(e) => setQ(e.target.value)}
          onKeyDown={onKey}
          autoComplete="off"
          spellCheck={false}
        />
        <kbd className="max-md:hidden" aria-hidden="true">
          {t(mp.close)}
        </kbd>
      </div>
      <span className="vh" id={`${id}-d`}>
        {t(mp.description)}
      </span>
      <div className="pal-list" id={`${id}-list`} role="listbox" aria-label={t(mp.label)}>
        {groups.map((g, gi) => (
          <div key={gi} role="group" aria-labelledby={`${id}-g${gi}`}>
            <div className="pal-grp" id={`${id}-g${gi}`} role="presentation">
              {g.label}
            </div>
            {g.entries.map((e) => {
              index++
              const i = index
              return (
                <div
                  key={e.key}
                  id={optionId(i)}
                  role="option"
                  tabIndex={-1}
                  aria-selected={i === current}
                  className={cn('pal-opt', i === current && 'on')}
                  onMouseMove={() => setActive(i)}
                  onClick={() => go(e)}
                  onKeyDown={(ev) => ev.key === 'Enter' && go(e)}
                >
                  <span className="pal-ico" aria-hidden="true">
                    {kindIcon[e.kind]}
                  </span>
                  <span className="pal-t">
                    <b>
                      <Marked text={e.title} q={typed} />
                    </b>
                    {e.detail ? <span>{e.detail}</span> : null}
                  </span>
                </div>
              )
            })}
          </div>
        ))}
        {status}
      </div>
      <div className="pal-foot max-md:hidden" aria-hidden="true">
        <span>
          <kbd>↑</kbd> <kbd>↓</kbd> {t(mp.toMove)}
        </span>
        <span>
          <kbd>↵</kbd> {t(mp.toOpen)}
        </span>
        <span>
          <kbd>{t(mp.close)}</kbd> {t(mp.toClose)}
        </span>
        <span className="ml-auto">{t(mp.onlyYours)}</span>
      </div>
      <span className="vh" role="status" aria-live="polite">
        {typed.length >= 2 && term === typed && search.isSuccess ? t(mp.results, { count: flat.length }) : ''}
      </span>
    </>
  )
}
