import { pickableLanguages, useT } from '@/i18n'
import { m } from './messages'

/**
 * Team › Invite a member: "Invitation language", the instance default
 * unless picked (LanguageRegion board). The email and the page it opens
 * use it; once they sign in, members pick their own.
 */
export function InvitationLanguageField({
  value,
  onChange,
  languages,
  defaultLanguage,
  disabled,
}: {
  value: string
  onChange: (v: string) => void
  languages: readonly { code: string; name: string }[] | undefined
  defaultLanguage: string | undefined
  disabled?: boolean
}) {
  const t = useT()
  const def = defaultLanguage ?? 'en'
  const pickable = pickableLanguages({ serverLanguages: languages?.map((l) => l.code) }).map((l) => l.code as string)
  const options = (languages ?? []).filter((l) => l.code === def || l.code === value || pickable.includes(l.code))
  if (options.length < 2) return null
  return (
    <label className="lbl">
      {t(m.invitationLanguage)}
      <select className="in" value={value} onChange={(e) => onChange(e.target.value)} disabled={disabled}>
        {options.map((l) => (
          <option key={l.code} value={l.code} lang={l.code}>
            {l.code === def ? t(m.invitationDefault, { language: l.name }) : l.name}
          </option>
        ))}
      </select>
      <span className="k font-medium">{t(m.invitationHint)}</span>
    </label>
  )
}
