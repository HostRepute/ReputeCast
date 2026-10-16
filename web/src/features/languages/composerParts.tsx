import { useNavigate } from '@tanstack/react-router'
import type { Component, Editor } from 'grapesjs'
import { useCallback, useEffect, useLayoutEffect, useMemo, useState, type ReactNode } from 'react'
import { $api } from '@/api/client'
import { Badge } from '@/components/ui/Badge'
import { Button } from '@/components/ui/Button'
import { useI18n, useT } from '@/i18n'
import { clock, dayMonthDate } from '@/lib/format'
import { useCan } from '@/lib/session'
import { useComposer } from '@/features/composer/context'
import { pathOf, sectionOf } from '@/features/composer/editor/labels'
import { componentAt } from '@/features/composer/fixes'
import { kb } from '@/features/composer/format'
import {
  AddVersionDialog,
  CompareIcon,
  LanguageBanner,
  LanguageSwitcher,
  MissingLanguage,
  nameIn,
  useCampaignLanguages,
  useLanguagesCache,
  useVersionActions,
} from './composer'
import { CompareFooter, CompareView, compareLanguage, diffWords, excerpt } from './compare'
import { mz } from './messages'

/**
 * The language version the composer has open: its code when it isn't the
 * source (the content endpoints' ?language=), else undefined.
 */
export function versionLanguage(c: { language?: string; sourceLanguage?: string }): string | undefined {
  return c.language && c.language !== c.sourceLanguage ? c.language : undefined
}

/**
 * The composer's save state without its version number, on a language
 * version (ComposerLanguages: "Draft · autosaved 11:20"): a version counts
 * its own saves, which would read as the email's.
 */
export function withoutVersion(text: string, onVersion: boolean): string {
  return onVersion ? text.replace(/ · v\d+$/, '') : text
}

/** Opens the composer on a language version, or its compare view, after saving. */
export function useLanguageNav() {
  const c = useComposer()
  const navigate = useNavigate()
  return useCallback(
    async (language: string | undefined, view?: 'compare', select?: string) => {
      if (c.canEdit && c.status.kind === 'dirty') await c.save()
      const source = c.campaign.sourceLanguage
      void navigate({
        to: '/w/$workspace/campaigns/$campaignId/edit',
        params: { workspace: c.slug, campaignId: c.campaign.id },
        search: { language: language && language !== source ? language : undefined, view, select } as never,
        ignoreBlocker: true,
      })
    },
    [c, navigate],
  )
}

/** The name of the language version open, or undefined for the source. */
export function useVersionName(): string | undefined {
  const c = useComposer()
  const lang = versionLanguage(c.campaign)
  const q = useCampaignLanguages(c.campaign.id)
  return lang ? nameIn(q.data, lang) : undefined
}

/** The header's switcher, with Add a language version (ComposerLanguages). */
export function ComposerLanguageSwitcher() {
  const c = useComposer()
  const q = useCampaignLanguages(c.campaign.id)
  const go = useLanguageNav()
  const [adding, setAdding] = useState(false)
  const data = q.data
  if (!data || data.versions.length < 2) return null
  const current = c.missingLanguage || c.campaign.language || data.source
  return (
    <>
      <LanguageSwitcher
        campaignName={c.name || c.campaign.name}
        data={data}
        language={current}
        onPick={(l) => void go(l)}
        onCompare={() => void go(current === data.source ? undefined : current, 'compare')}
        onAdd={() => setAdding(true)}
      />
      {adding ? (
        <AddVersionDialog
          campaignId={c.campaign.id}
          campaignName={c.name || c.campaign.name}
          data={data}
          open
          onOpenChange={setAdding}
          onAdded={(l) => void go(l)}
        />
      ) : null}
    </>
  )
}

/** Above the email on the canvas, as wide as it: where the version open stands. */
export function ComposerLanguageBanner({ width }: { width: number }) {
  const c = useComposer()
  const q = useCampaignLanguages(c.campaign.id)
  const go = useLanguageNav()
  const lang = versionLanguage(c.campaign)
  if (!q.data || q.data.versions.length < 2) return null
  return (
    <div className="lng-banner-wrap" style={{ width }}>
      <LanguageBanner campaignId={c.campaign.id} data={q.data} language={lang ?? q.data.source} onCompare={() => void go(lang, 'compare')} />
    </div>
  )
}

/** In place of the canvas: the language picked has no version (LanguageStates). */
export function ComposerMissingLanguage({ language }: { language: string }) {
  const c = useComposer()
  const q = useCampaignLanguages(c.campaign.id)
  const go = useLanguageNav()
  const [adding, setAdding] = useState(false)
  if (!q.data) return null
  return (
    <>
      <MissingLanguage data={q.data} language={language} onAdd={() => setAdding(true)} />
      {adding ? (
        <AddVersionDialog
          campaignId={c.campaign.id}
          campaignName={c.name || c.campaign.name}
          data={q.data}
          open
          preselect={language}
          onOpenChange={setAdding}
          onAdded={(l) => void go(l)}
        />
      ) : null}
    </>
  )
}

/** The compare view in the composer (ComposerCompare). */
export function ComposerCompare() {
  const c = useComposer()
  const q = useCampaignLanguages(c.campaign.id)
  const go = useLanguageNav()
  const lang = versionLanguage(c.campaign)
  if (!q.data) return null
  return (
    <CompareView
      campaignId={c.campaign.id}
      data={q.data}
      language={lang ?? ''}
      onLanguage={(l) => void go(l, 'compare')}
      onBack={() => void go(lang)}
      onShow={(index) => void go(lang, undefined, String(index))}
    />
  )
}

/** The status bar while comparing: "German · 71 KB of 102 KB · 2 of 9 blocks changed in English since v6". */
export function ComposerCompareFooter({ children }: { children?: ReactNode }) {
  const c = useComposer()
  const q = useCampaignLanguages(c.campaign.id)
  const lang = versionLanguage(c.campaign) ?? ''
  if (!q.data) return null
  // The size is the version's when the canvas holds the one compared.
  const size = c.checks && compareLanguage(q.data, lang) === lang ? kb(c.checks.sizeBytes) : undefined
  return (
    <CompareFooter campaignId={c.campaign.id} data={q.data} language={lang} size={size}>
      {children}
    </CompareFooter>
  )
}

/** The status bar's "German · 71 KB of 102 KB". */
export function useVersionStatus(size: string): string | undefined {
  const t = useT()
  const name = useVersionName()
  return name ? t(mz.statusLine, { language: name, size }) : undefined
}

/**
 * What the composer loads for ?language=: that version's content, or the
 * source with "No … version" when it has none. Without ?language= nothing
 * waits.
 */
export function useOpenLanguage(campaignId: string, language?: string): { ready: boolean; load?: string; missing?: string } {
  const q = useCampaignLanguages(campaignId)
  if (!language) return { ready: true }
  if (q.isPending) return { ready: false }
  const v = q.data?.versions.find((x) => x.language === language)
  if (!v || v.status === 'source') return { ready: true }
  if (v.status === 'missing') return { ready: true, missing: language }
  return { ready: true, load: language }
}

/** "One of each language" in the test-send picker. */
export const EACH_LANGUAGE = '*'

/** The test-send body for a language pick. */
export function testLanguageBody(language?: string): { language?: string; eachLanguage?: boolean } {
  if (language === EACH_LANGUAGE) return { eachLanguage: true }
  return language ? { language } : {}
}

/** Send test: one language, or one of each (ComposerPreview). Nothing for one language. */
export function TestLanguagePicker({ value, onChange }: { value?: string; onChange: (v: string) => void }) {
  const c = useComposer()
  const t = useT()
  const q = useCampaignLanguages(c.campaign.id)
  const versions = (q.data?.versions ?? []).filter((v) => v.status !== 'missing')
  if (versions.length < 2) return null
  return (
    <label className="lbl px-2.5 py-1.5">
      {t(mz.testOne)}
      <select className="in" value={value || q.data!.source} onChange={(e) => onChange(e.target.value)}>
        {versions.map((v) => (
          <option key={v.language} value={v.language}>
            {v.name}
          </option>
        ))}
        <option value={EACH_LANGUAGE}>{t(mz.testEach)}</option>
      </select>
    </label>
  )
}

const names = new Map<string, Intl.DisplayNames>()

/** A language code's name in the app's language ("pt-BR" → "Portuguese"). */
export function languageLabel(code: string, locale: string): string {
  const base = code.toLowerCase().split(/[-_]/)[0] ?? code
  try {
    let n = names.get(locale)
    if (!n) {
      n = new Intl.DisplayNames([locale], { type: 'language' })
      names.set(locale, n)
    }
    return n.of(base) ?? base
  } catch {
    return base
  }
}

/**
 * "Preview as" names each contact's language and the version they get
 * (ComposerLanguages): "Lucía Fernández · Spanish", "Phạm Minh Anh ·
 * Vietnamese, gets English". Undefined for an email in one language.
 */
export function usePreviewAsLabel(): (ct: { name: string; email: string; language: string }) => string | undefined {
  const c = useComposer()
  const t = useT()
  const { locale } = useI18n()
  const q = useCampaignLanguages(c.campaign.id)
  return (ct) => {
    const data = q.data
    if (!data || data.versions.length < 2) return undefined
    const who = ct.name || ct.email
    const base = ct.language.toLowerCase().split(/[-_]/)[0] ?? ''
    if (!base) return t(mz.previewAsGets, { name: who, language: t(mz.noLanguage), gets: nameIn(data, data.fallback) })
    const v = data.versions.find((x) => x.language === base)
    const own = v?.name ?? languageLabel(base, locale)
    const gets = v ? v.gets : data.fallback
    return gets === base ? t(mz.previewAs, { name: who, language: own }) : t(mz.previewAsGets, { name: who, language: own, gets: nameIn(data, gets) })
  }
}

const AS_KEY = (campaignId: string) => `rc-preview-as:${campaignId}`
const baseOf = (code: string) => code.toLowerCase().split(/[-_]/)[0] ?? ''

/**
 * The canvas toolbar's "Preview as" (ComposerLanguages): a contact of each
 * language and the version they get. Picking one opens that version; the
 * pick is kept for the visit, so it survives the switch.
 */
export function CanvasPreviewAs() {
  const c = useComposer()
  const t = useT()
  const canView = useCan('subscribers:view')
  const q = useCampaignLanguages(c.campaign.id)
  const label = usePreviewAsLabel()
  const go = useLanguageNav()
  const [picked, setPicked] = useState(() => {
    try {
      return window.sessionStorage.getItem(AS_KEY(c.campaign.id)) ?? ''
    } catch {
      return ''
    }
  })
  const many = (q.data?.versions.length ?? 0) > 1
  const contacts = $api.useQuery(
    'get',
    '/api/v1/workspaces/{workspace}/campaigns/{campaignId}/languages/preview-contacts',
    { params: { path: { workspace: c.slug, campaignId: c.campaign.id } } },
    { enabled: canView && many, staleTime: 60_000 },
  )
  const items = contacts.data?.items ?? []
  const data = q.data
  if (!data || !many || !items.length) return null
  const current = c.missingLanguage || versionLanguage(c.campaign) || data.source
  const chosen =
    items.find((x) => x.id === picked && (x.gets === current || baseOf(x.language) === current)) ??
    items.find((x) => x.offered && baseOf(x.language) === current) ??
    items.find((x) => x.gets === current)
  return (
    <label className="lng-pvas">
      <span>{t(mz.previewAsLabel)}</span>
      <select
        className="in"
        value={chosen?.id ?? ''}
        onChange={(e) => {
          const ct = items.find((x) => x.id === e.target.value)
          if (!ct) return
          setPicked(ct.id)
          try {
            window.sessionStorage.setItem(AS_KEY(c.campaign.id), ct.id)
          } catch {
            /* storage unavailable */
          }
          if (ct.gets !== current) void go(ct.gets === data.source ? undefined : ct.gets)
        }}
      >
        {chosen ? null : (
          <option value="" disabled>
            {t(mz.previewAsPick)}
          </option>
        )}
        {items.map((ct) => (
          <option key={ct.id} value={ct.id}>
            {label(ct) ?? (ct.name || ct.email)}
          </option>
        ))}
      </select>
    </label>
  )
}

interface Box {
  top: number
  left: number
  width: number
  height: number
}

/** Position of an element of the canvas frame, relative to `host` (as CanvasArea's overlays). */
function boxOf(editor: Editor, el: Element | null | undefined, host: HTMLElement | null): Box | null {
  if (!el || !host || !el.isConnected) return null
  const frame = editor.Canvas.getFrameEl()
  if (!frame) return null
  const zoom = editor.Canvas.getZoomDecimal()
  const fr = frame.getBoundingClientRect()
  const hr = host.getBoundingClientRect()
  const r = el.getBoundingClientRect()
  if (!r.width && !r.height) return null
  return { top: fr.top - hr.top + r.top * zoom, left: fr.left - hr.left + r.left * zoom, width: r.width * zoom, height: r.height * zoom }
}

/** Room above a tagged block, in the canvas only (never in what is saved). */
const TAG_ROOM = 'rc-lang-tag'
const TAG_CSS = `[data-${TAG_ROOM}]::before{content:"";display:block;height:38px}`

/**
 * Tags on the canvas over the version's blocks to look at (ComposerLanguages):
 * "Changed in English" for an outdated block, "Still in English" for a
 * draft's block not translated yet. As on the boards, each tag gets its own
 * row above the block's content: the canvas makes room for it, so it never
 * covers the email.
 */
export function LanguageTags({ host, tick }: { host: HTMLElement | null; tick: number }) {
  const c = useComposer()
  const t = useT()
  const q = useCampaignLanguages(c.campaign.id)
  const lang = versionLanguage(c.campaign)
  const v = lang ? q.data?.versions.find((x) => x.language === lang) : undefined
  const ed = c.editor
  const targets = useMemo(() => {
    if (!ed || !v) return []
    const out: { key: string; target: Component; still: boolean }[] = []
    const add = (i: number, still: boolean) => {
      const target = componentAt(c, [i])
      if (target) out.push({ key: `${still ? 's' : 'c'}${i}`, target, still })
    }
    v.changedIndexes.forEach((i) => add(i, false))
    v.stillIndexes.forEach((i) => add(i, true))
    return out
  }, [ed, v, tick]) // eslint-disable-line react-hooks/exhaustive-deps
  useLayoutEffect(() => {
    const doc = ed?.Canvas.getDocument()
    if (!doc?.head) return
    if (!doc.getElementById(TAG_ROOM)) {
      const style = doc.createElement('style')
      style.id = TAG_ROOM
      style.textContent = TAG_CSS
      doc.head.appendChild(style)
    }
    const els = targets.map((x) => x.target.getEl()).filter((el): el is HTMLElement => Boolean(el))
    els.forEach((el) => el.setAttribute(`data-${TAG_ROOM}`, ''))
    doc.querySelectorAll(`[data-${TAG_ROOM}]`).forEach((el) => {
      if (!els.includes(el as HTMLElement)) el.removeAttribute(`data-${TAG_ROOM}`)
    })
  }, [ed, targets])
  useEffect(
    () => () => {
      ed?.Canvas.getDocument()
        ?.querySelectorAll(`[data-${TAG_ROOM}]`)
        .forEach((el) => el.removeAttribute(`data-${TAG_ROOM}`))
    },
    [ed],
  )
  if (!q.data || !ed || !targets.length) return null
  const source = nameIn(q.data, q.data.source)
  return (
    <>
      {targets.map(({ key, target, still }) => {
        const box = boxOf(ed, target.getEl(), host)
        if (!box) return null
        return (
          <button
            key={key}
            type="button"
            className={still ? 'wtag n' : 'wtag w'}
            style={{ top: box.top + 7, left: box.left + 24 }}
            onClick={() => {
              ed.select(target)
              c.setRightTab('settings')
            }}
          >
            <svg width="12" height="12" viewBox="0 0 16 16" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true" focusable="false">
              <path d={still ? 'M3 8h10' : 'M8 3v6M8 12v.5'} />
            </svg>
            {still ? t(mz.stateStill, { source }) : t(mz.stateChanged, { source })}
          </button>
        )
      })}
    </>
  )
}

/**
 * The inspector's "This block in German" (ComposerLanguages): where the
 * selected block of a version stands, what changed in the source since it
 * was translated, Compare and Mark this block as up to date.
 */
export function LanguageBlockPanel() {
  const c = useComposer()
  const t = useT()
  const go = useLanguageNav()
  const lang = versionLanguage(c.campaign)
  const q = useCampaignLanguages(c.campaign.id)
  const section = sectionOf(c.selected)
  const index = section ? (pathOf(section)[0] ?? -1) : -1
  const cmp = $api.useQuery(
    'get',
    '/api/v1/workspaces/{workspace}/campaigns/{campaignId}/languages/{language}/compare',
    { params: { path: { workspace: c.slug, campaignId: c.campaign.id, language: lang ?? '' } } },
    { enabled: Boolean(lang) && index >= 0 },
  )
  const mark = useVersionActions(c.campaign.id)
  const data = q.data
  if (!lang || !data || index < 0 || !cmp.data) return null
  const row = cmp.data.rows.find((r) => r.version?.index === index)
  const v = data.versions.find((x) => x.language === lang)
  if (!row || !v) return null
  const source = nameIn(data, data.source)
  const name = nameIn(data, lang)
  const b =
    row.state === 'changed'
      ? { tone: 'warn' as const, text: t(mz.stateChanged, { source }) }
      : row.state === 'still'
        ? { tone: 'neutral' as const, text: t(mz.stateStill, { source }) }
        : row.state === 'only'
          ? { tone: 'neutral' as const, text: t(mz.stateOnly, { language: name }) }
          : row.state === 'removed'
            ? { tone: 'warn' as const, text: t(mz.stateRemoved, { source }) }
            : { tone: 'ok' as const, text: t(mz.stateUpToDate) }
  return (
    <div className="flex flex-col gap-2.5" style={{ paddingTop: 12, borderTop: '1px solid var(--line-2)' }}>
      <div className="row" style={{ justifyContent: 'space-between', gap: 8 }}>
        <span style={{ fontWeight: 600 }}>{t(mz.blockIn, { language: name })}</span>
        <Badge tone={b.tone} style={{ height: 20, fontSize: 11 }}>
          {b.text}
        </Badge>
      </div>
      <dl className="kv">
        {cmp.data.translatedBy && cmp.data.translatedAt ? (
          <>
            <dt>{t(mz.translatedRow)}</dt>
            <dd>
              {t(mz.translatedFrom, {
                source,
                version: row.fromVersion || cmp.data.translatedFrom,
                name: cmp.data.translatedBy,
                date: `${dayMonthDate(cmp.data.translatedAt)} ${clock(cmp.data.translatedAt)}`,
              })}
            </dd>
          </>
        ) : null}
        {row.state === 'changed' && row.changed ? (
          <>
            <dt>{t(mz.sourceNowRow, { source })}</dt>
            <dd>{t(mz.sourceNow, { version: cmp.data.sourceVersion, name: row.changed.by, date: `${dayMonthDate(row.changed.at)} ${clock(row.changed.at)}` })}</dd>
          </>
        ) : null}
      </dl>
      {row.state === 'changed' && row.before && row.source ? (
        <div className="diff" style={{ fontFamily: 'var(--font-sans, inherit)', fontSize: 12.5 }}>
          <div className="ctx" style={{ background: 'var(--ground-3)', fontWeight: 600 }}>
            {t(mz.diffHead, { source, from: row.fromVersion, to: cmp.data.sourceVersion })}
          </div>
          <div className="ctx">
            {excerpt(diffWords(row.before, row.source.text)).map((p, i) =>
              p.kind === 'same' ? (
                <span key={i}>{p.text}</span>
              ) : (
                <span key={i} className={p.kind === 'add' ? 'lng-da' : 'lng-dl'}>
                  {p.text}
                </span>
              ),
            )}
          </div>
        </div>
      ) : null}
      {row.state === 'changed' ? (
        <div className="flex flex-wrap gap-1.5">
          <Button size="sm" onClick={() => void go(lang, 'compare')}>
            <CompareIcon />
            {t(mz.compareButton)}
          </Button>
          {data.canEdit ? (
            <Button size="sm" loading={mark.busy} onClick={() => mark.markUpToDate(v, [row.key])}>
              {t(mz.markBlock)}
            </Button>
          ) : null}
        </div>
      ) : null}
    </div>
  )
}

/** Keeps the switcher in step after a save of the version open. */
export function useRefreshLanguages() {
  const c = useComposer()
  const cache = useLanguagesCache(c.campaign.id)
  return cache.refresh
}
