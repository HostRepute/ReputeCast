import { Fragment, type ReactNode } from 'react'
import { useI18n } from '.'
import { i18nKey } from './store'

/**
 * Re-renders the app when the language or a format changes. lib/format and
 * t() read the current language when they run, so the screens below are
 * mounted again (data stays in the query cache) and every date, number and
 * word comes out in the new language at once.
 */
export function I18nRoot({ children }: { children: ReactNode }) {
  const state = useI18n()
  return <Fragment key={i18nKey(state)}>{children}</Fragment>
}
