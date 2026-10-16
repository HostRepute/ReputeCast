import { useState } from 'react'
import { errorMessage, http } from '@/api/client'
import { Button } from '@/components/ui/Button'
import { toast } from '@/components/ui/Toast'
import { browserLanguage, languageName, useI18n, useT, useTIn, type AppLocale } from '@/i18n'
import { m } from './messages'
import { useMemberLanguage, useSaveMemberLanguage } from './useLanguage'
import './language.css'

const ANSWERED_KEY = 'rc.language.suggested'

function answered(): boolean {
  try {
    return window.localStorage.getItem(ANSWERED_KEY) !== null
  } catch {
    return false
  }
}

function answer(value: string) {
  try {
    window.localStorage.setItem(ANSWERED_KEY, value)
  } catch {
    // Without storage the suggestion comes back next time; nothing breaks.
  }
}

const Globe = () => (
  <svg width="18" height="18" viewBox="0 0 16 16" fill="none" stroke="#1D4ED8" strokeWidth="1.6" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
    <circle cx="8" cy="8" r="6.2" />
    <path d="M1.8 8h12.4M8 1.8c1.8 1.7 2.7 3.8 2.7 6.2S9.8 12.5 8 14.2C6.2 12.5 5.3 10.4 5.3 8S6.2 3.5 8 1.8z" />
  </svg>
)

const Cross = () => (
  <svg width="13" height="13" viewBox="0 0 16 16" fill="none" stroke="currentColor" strokeWidth="1.9" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
    <path d="M4 4l8 8M12 4l-8 8" />
  </svg>
)

/** Tells the server the suggestion was answered, so no other browser offers it again. */
function report(answer: 'use' | 'keep' | 'dismiss', language: AppLocale) {
  void http.PUT('/api/v1/me/language/suggestion', { body: { answer, language: language as 'en' | 'vi' } }).catch(() => {
    // This browser remembers the answer anyway.
  })
}

/**
 * "Your browser prefers Tiếng Việt. Use ReputeCast in Tiếng Việt?": offered
 * once per member, when the browser prefers a language the app has and the
 * member hasn't chosen one (connect-and-reach brief §3.7, LanguageRegion
 * board). The question also reads in the language it offers.
 */
export function LanguageSuggestion() {
  const { me, choices, languages } = useMemberLanguage()
  const { locale } = useI18n()
  const [hidden, setHidden] = useState(answered)
  const offer: AppLocale | undefined =
    me && choices.language === 'default' && !hidden && !me.user.languageSuggestionAnswered ? browserLanguage(languages) : undefined
  const show = offer !== undefined && offer !== locale
  const t = useT()
  const tIn = useTIn(show ? offer : undefined)
  const save = useSaveMemberLanguage()
  if (!show || !offer) return null
  const name = languageName(offer)

  const close = (how: 'keep' | 'dismiss') => {
    answer('dismissed')
    setHidden(true)
    report(how, offer)
  }
  const use = async () => {
    answer(offer)
    setHidden(true)
    report('use', offer)
    try {
      await save({ ...choices, language: offer })
      toast({ kind: 'success', title: t(m.switched, { language: name }) })
    } catch (e) {
      toast({ kind: 'error', title: t(m.saveFailed), detail: errorMessage(e) })
    }
  }
  return (
    <section className="card lang-suggest" aria-label={t(m.suggestLabel)}>
      <Globe />
      <div className="flex min-w-0 flex-col gap-[3px]" style={{ flex: '1 1 280px' }}>
        <span className="font-semibold">{t(m.suggestQuestion, { language: name })}</span>
        <span className="k" lang={offer} style={{ color: 'var(--ink-nav)' }}>
          {tIn(m.suggestQuestion, { language: name })}
        </span>
        <span className="k">{t(m.suggestOnce)}</span>
      </div>
      <div className="flex flex-wrap items-center gap-2">
        <Button variant="primary" size="sm" lang={offer} onClick={() => void use()}>
          {tIn(m.suggestUse, { language: name })}
        </Button>
        <Button size="sm" onClick={() => close('keep')}>
          {t(m.suggestKeep, { language: languageName(locale) })}
        </Button>
        <Button size="sm" iconOnly aria-label={t(m.dismiss)} onClick={() => close('dismiss')} style={{ width: 30, padding: 0 }}>
          <Cross />
        </Button>
      </div>
    </section>
  )
}
