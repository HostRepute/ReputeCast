import type { Schemas } from '@/api/client'
import { Icon } from '@/components/icons/Icon'
import { T, formatList, useT } from '@/i18n'
import { useComposer } from '@/features/composer/context'
import { nameIn, useEmailLanguages } from './composer'
import { mx } from './messages'
import './languages.css'

type PublishLanguage = Schemas['TransactionalPublishLanguage']
type LanguageChange = Schemas['TransactionalLanguageChange']

/**
 * The publish dialog's languages (TransactionalPublish): "Publishes English,
 * Spanish and German together", then any outdated version, sent as it is,
 * and any draft, whose recipients get the fallback.
 */
export function PublishLanguages({ languages, changes }: { languages?: PublishLanguage[] | null; changes?: LanguageChange[] | null }) {
  const t = useT()
  if (!languages || languages.length < 2) return null
  const source = languages[0]!
  const sent = languages.filter((l) => l.status !== 'draft')
  return (
    <div className="flex flex-col gap-2">
      <span className="font-semibold">{t(mx.publishTogether, { languages: formatList(sent.map((l) => l.name)), count: sent.length })}</span>
      {changes?.length ? (
        <ul className="lng-changes">
          {changes.map((c) => (
            <li key={c.language}>{t(mx.languageChange, { language: c.name, change: c.change })}</li>
          ))}
        </ul>
      ) : null}
      {languages
        .filter((l) => l.status === 'outdated')
        .map((l) => (
          <div key={l.language} className="note w" role="note">
            <Icon name="alertCircle" size={16} />
            <span>{t(mx.publishOutdated, { language: l.name, count: l.attention, source: source.name })}</span>
          </div>
        ))}
      {languages
        .filter((l) => l.status === 'draft')
        .map((l) => (
          <div key={l.language} className="note n" role="note">
            <Icon name="info" size={16} />
            <span>{t(mx.publishDraft, { language: l.name, fallback: l.getsName })}</span>
          </div>
        ))}
    </div>
  )
}

/**
 * The API tab's word on languages (TransactionalSetup): send `language` to
 * pick a version; without it, the linked contact's language, then the
 * fallback. Nothing for a template in one language.
 */
export function ApiLanguageNote() {
  const c = useComposer()
  const q = useEmailLanguages({ kind: 'template', id: c.campaign.id })
  const data = q.data
  const versions = (data?.versions ?? []).filter((v) => v.status !== 'missing' && v.status !== 'source')
  if (!data || !versions.length) return null
  return (
    <span className="k">
      <T
        msg={mx.apiLanguage}
        values={{ codes: formatList(versions.map((v) => v.language)), fallback: nameIn(data, data.fallback), mono: (chunks) => <span className="mono">{chunks}</span> }}
      />
    </span>
  )
}
