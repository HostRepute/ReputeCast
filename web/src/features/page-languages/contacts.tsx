import { useQueryClient } from '@tanstack/react-query'
import { useState } from 'react'
import { $api, errorMessage, type Schemas } from '@/api/client'
import { toast } from '@/components/ui/Toast'
import { brandOf } from '@/features/languages/LanguagesCard'
import { languageLabel } from '@/features/languages/composerParts'
import { useI18n, useT } from '@/i18n'
import { dayMonthYear } from '@/lib/format'
import { usePermissionReason, useWorkspace } from '@/lib/session'
import { mc } from './messages'
import './page-languages.css'

type Translate = ReturnType<typeof useT>
type Profile = Schemas['ContactProfile']
type ContactLanguage = Schemas['ContactLanguage']
type Activity = Schemas['ContactActivityItem']

/** The workspace's languages (Settings › General), for filters and selects. */
export function useWorkspaceLanguages() {
  const { slug } = useWorkspace()
  return $api.useQuery('get', '/api/v1/workspaces/{workspace}/languages', { params: { path: { workspace: slug } } }, { staleTime: 60_000 })
}

/**
 * The Contacts filter's language select (Contacts): any language, each
 * workspace language, contacts with no language, and contacts with a
 * language the workspace doesn't offer.
 */
export function LanguageFilter({ value, onChange, className }: { value?: string; onChange: (v: string | undefined) => void; className?: string }) {
  const t = useT()
  const { locale } = useI18n()
  const langs = useWorkspaceLanguages()
  const codes = langs.data?.languages.map((l) => l.code) ?? []
  // A shared link may filter on a language the list doesn't have yet.
  if (value && value !== 'none' && value !== 'not_offered' && !codes.includes(value)) codes.push(value)
  return (
    <select className={className} aria-label={t(mc.filterLabel)} value={value ?? ''} onChange={(e) => onChange(e.target.value || undefined)}>
      <option value="">{t(mc.filterAny)}</option>
      {codes.map((code) => (
        <option key={code} value={code}>
          {languageLabel(code, locale)}
        </option>
      ))}
      <option value="none">{t(mc.filterNone)}</option>
      <option value="not_offered">{t(mc.filterNotOffered)}</option>
    </select>
  )
}

/** Where a contact's language came from ("Brevo import, 18 Sep 2026"). */
export function languageSource(l: ContactLanguage, t: Translate): string {
  if (!l.code || !l.source) return ''
  const date = l.setAt ? dayMonthYear(l.setAt) : ''
  switch (l.source) {
    case 'import':
      return t(mc.sourceImport, { source: l.sourceName, date })
    case 'form':
      if (!l.sourceName) return t(mc.sourceSubscribePage, { date })
      return l.browser ? t(mc.sourceForm, { form: l.sourceName, date }) : t(mc.sourceFormNotBrowser, { form: l.sourceName, date })
    case 'preference_center':
      return t(mc.sourcePreferences, { date })
    case 'api':
      return l.sourceName ? t(mc.sourceApiNamed, { name: l.sourceName, date }) : t(mc.sourceApi, { date })
    case 'team':
      return l.sourceName ? t(mc.sourceTeam, { name: l.sourceName, date }) : t(mc.sourceTeamUnknown, { date })
  }
  return ''
}

/**
 * The language row of the profile's Fields card (ContactProfile): a select
 * of the workspace's languages with where the language came from, and a
 * note when the contact's language isn't one of them (LanguageStates).
 */
export function ContactLanguageField({ p }: { p: Profile }) {
  const t = useT()
  const { locale } = useI18n()
  const { slug, me } = useWorkspace()
  const qc = useQueryClient()
  const reason = usePermissionReason('subscribers:edit')
  const update = $api.useMutation('patch', '/api/v1/workspaces/{workspace}/contacts/{contact}')
  const l = p.language
  const [value, setValue] = useState(l.code)
  const name = p.firstName || p.name || p.email
  const label = (code: string) => t(mc.option, { language: languageLabel(code, locale), code })
  const change = async (code: string) => {
    setValue(code)
    try {
      await update.mutateAsync({ params: { path: { workspace: slug, contact: p.id } }, body: { language: code } })
      await qc.invalidateQueries({ queryKey: ['get', '/api/v1/workspaces/{workspace}/contacts/{contact}'] })
      toast({ kind: 'success', title: t(mc.saved), detail: t(mc.savedDetail, { name, language: languageLabel(code, locale) }) })
    } catch (e) {
      setValue(l.code)
      toast({ kind: 'error', title: t(mc.saveFailed), detail: errorMessage(e) })
    }
  }
  const source = languageSource(l, t)
  return (
    <>
      <div className="pkv items-center">
        <label className="k" htmlFor="cp-lang">
          {t(mc.fieldLabel)}
        </label>
        <span className="flex min-w-0 flex-col items-end gap-[3px]">
          <select
            id="cp-lang"
            className="cp-lang"
            value={value}
            aria-disabled={reason || update.isPending ? true : undefined}
            aria-describedby={reason ? 'cp-lang-why' : undefined}
            title={reason ?? undefined}
            // Without subscribers:edit the list opens but the value stays
            // (it's controlled).
            onChange={(e) => (reason || update.isPending ? undefined : void change(e.target.value))}
          >
            {l.options.map((o) => (
              <option key={o.code} value={o.code}>
                {label(o.code)}
              </option>
            ))}
            {!l.code ? <option value="">{t(mc.optionNone)}</option> : null}
          </select>
          {source ? <span className="k">{source}</span> : null}
          {reason ? (
            <span className="vh" id="cp-lang-why">
              {reason}
            </span>
          ) : null}
        </span>
      </div>
      {l.code && !l.offered ? (
        <div className="note w cp-lang-note" role="note">
          <WarnIcon />
          <div>{t(mc.notOffered, { language: languageLabel(l.code, locale), code: l.code, fallback: languageLabel(l.gets, locale), name, workspace: brandOf(me.workspace.name) })}</div>
        </div>
      ) : null}
    </>
  )
}

/** The warning sign of the boards' warn notes. */
export function WarnIcon() {
  return (
    <svg width="16" height="16" viewBox="0 0 16 16" fill="none" stroke="#B45309" strokeWidth={1.8} strokeLinecap="round" strokeLinejoin="round" aria-hidden="true" className="flex-none">
      <circle cx="8" cy="8" r="6.2" />
      <path d="M8 4.8v3.6M8 10.9v.1" />
    </svg>
  )
}

/**
 * The language version a delivery sent, for the activity line ("Weekly
 * Digest #141 · English via Amazon SES"), and why when the contact got the
 * fallback ("Portuguese isn't one of Acme's languages").
 */
export function deliveryLanguage(it: Activity, t: Translate, locale: string, workspace: string): { label?: string; fallback: boolean; why?: string } {
  if (!it.language) return { fallback: false }
  const language = languageLabel(it.language, locale)
  const contact = it.contactLanguageName || ''
  switch (it.languageReason) {
    case 'not_offered':
      return { label: t(mc.deliveredFallback, { language }), fallback: true, why: contact ? t(mc.reasonNotOffered, { language: contact, workspace }) : undefined }
    case 'no_language':
      return { label: t(mc.deliveredFallback, { language }), fallback: true, why: t(mc.reasonNoLanguage) }
    case 'missing':
      return { label: t(mc.deliveredFallback, { language }), fallback: true, why: contact ? t(mc.reasonMissing, { language: contact }) : undefined }
    case 'draft':
      return { label: t(mc.deliveredFallback, { language }), fallback: true, why: contact ? t(mc.reasonDraft, { language: contact }) : undefined }
    case 'held':
      return { label: t(mc.deliveredFallback, { language }), fallback: true, why: contact ? t(mc.reasonHeld, { language: contact }) : undefined }
  }
  return { label: language, fallback: false }
}

/** "consent text v1 · Spanish": the version and the language it was shown in. */
export function consentVersionText(version: string, language: string, t: Translate, locale: string): string {
  return t(mc.consentVersionLanguage, { version, language: languageLabel(language, locale) })
}
