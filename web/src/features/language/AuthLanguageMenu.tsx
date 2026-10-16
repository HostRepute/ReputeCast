import { useRouterState } from '@tanstack/react-router'
import { Fragment } from 'react'
import { Icon } from '@/components/icons/Icon'
import { PSEUDO_LOCALE, pickableLanguages, useI18n, useT } from '@/i18n'
import { chooseVisitorLanguage, isVisitorPath } from '@/i18n/sync'
import { m } from './messages'
import './language.css'

/**
 * The language menu at the bottom of the sign-in, invitation and password
 * pages (connect-and-reach brief §6): every language the app has, each
 * named in its own language, as in the public pages' footer menu. The
 * choice is remembered in this browser and beats the invitation's language
 * and the browser's.
 */
export function AuthLanguageMenu({ languages }: { languages?: readonly { code: string }[] }) {
  const t = useT()
  const { locale } = useI18n()
  const path = useRouterState({ select: (s) => s.location.pathname })
  const options = pickableLanguages({ serverLanguages: languages?.map((l) => l.code), pseudo: true })
  if (!isVisitorPath(path) || options.length < 2) return null
  return (
    <nav className="lang-auth-menu" aria-label={t(m.menuLabel)}>
      <Icon name="language" size={14} />
      {options.map((l, i) => (
        <Fragment key={l.code}>
          {i > 0 ? <span aria-hidden="true">·</span> : null}
          <button
            type="button"
            lang={l.code === PSEUDO_LOCALE ? 'en' : l.code}
            aria-current={l.code === locale ? 'true' : undefined}
            onClick={() => void chooseVisitorLanguage(l.code)}
          >
            {l.name}
          </button>
        </Fragment>
      ))}
    </nav>
  )
}
