import { useNavigate } from '@tanstack/react-router'
import { useCallback, useState } from 'react'
import { useI18n, useT } from '@/i18n'
import { useComposer } from '@/features/composer/context'
import { AddVersionDialog, LanguageBanner, LanguageSwitcher, MissingLanguage, nameIn, useCampaignLanguages, useLanguagesCache } from './composer'
import { CompareView } from './compare'
import { mz } from './messages'

/**
 * The language version the composer has open: its code when it isn't the
 * source (the content endpoints' ?language=), else undefined.
 */
export function versionLanguage(c: { language?: string; sourceLanguage?: string }): string | undefined {
  return c.language && c.language !== c.sourceLanguage ? c.language : undefined
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
  const current = c.campaign.language || data.source
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

/** Above the canvas: where the version open stands. */
export function ComposerLanguageBanner() {
  const c = useComposer()
  const q = useCampaignLanguages(c.campaign.id)
  const go = useLanguageNav()
  const lang = versionLanguage(c.campaign)
  if (!q.data || q.data.versions.length < 2) return null
  return <LanguageBanner campaignId={c.campaign.id} data={q.data} language={lang ?? q.data.source} onCompare={() => void go(lang, 'compare')} />
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

/** Keeps the switcher in step after a save of the version open. */
export function useRefreshLanguages() {
  const c = useComposer()
  const cache = useLanguagesCache(c.campaign.id)
  return cache.refresh
}
