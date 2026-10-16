import { PSEUDO_LOCALE, regionOptions, useT } from '@/i18n'
import { m } from './messages'
import { useMemberLanguage } from './useLanguage'

/** The languages an instance default or an invitation can be in: never the pseudo-locale. */
export function useInstanceLanguages() {
  const { languages } = useMemberLanguage()
  return languages.filter((l) => l.code !== PSEUDO_LOCALE)
}

/**
 * Default language and Default region format (Instance admin › Instance &
 * updates › Instance; InstanceSettings board), with the line on where they
 * apply.
 */
export function InstanceLanguageFields({
  language,
  region,
  onLanguage,
  onRegion,
  showRegion = true,
}: {
  language: string
  region: string
  onLanguage: (v: string) => void
  onRegion: (v: string) => void
  /** The first sign-in asks for the language only; the region follows it. */
  showRegion?: boolean
}) {
  const t = useT()
  const languages = useInstanceLanguages()
  if (!languages.some((l) => l.code === language)) languages.push({ code: language as 'en', name: language })
  return (
    <div className={showRegion ? 'grid grid-cols-2 gap-3.5 max-md:grid-cols-1' : 'grid grid-cols-1 gap-3.5'}>
      <label className="lbl">
        {t(m.defaultLanguage)}
        <select className="in" value={language} onChange={(e) => onLanguage(e.target.value)}>
          {languages.map((l) => (
            <option key={l.code} value={l.code} lang={l.code}>
              {l.name}
            </option>
          ))}
        </select>
      </label>
      {showRegion ? (
        <label className="lbl">
          {t(m.defaultRegion)}
          <select className="in" value={region} onChange={(e) => onRegion(e.target.value)}>
            {regionOptions().map((r) => (
              <option key={r.code} value={r.code} lang={r.code === 'standard' ? 'en' : r.code}>
                {r.name}
              </option>
            ))}
          </select>
        </label>
      ) : null}
      <span className="k" style={{ gridColumn: '1/-1', fontWeight: 500, marginTop: -6 }}>
        {t(m.defaultsHint)}
      </span>
    </div>
  )
}
