import { useId } from 'react'
import { CardSection } from '@/components/ui/Card'
import { useT } from '@/i18n'
import { cn } from '@/lib/cn'
import { THEME_CHOICES, type Theme, type ThemeChoice } from '@/lib/theme'
import { m } from './messages'
import { useSaveTheme, useTheme } from './useTheme'
import './appearance.css'

export const themeLabel = { system: m.system, light: m.light, dark: m.dark } as const
export const themeIcon = { system: 'desktop', light: 'sun', dark: 'moon' } as const

/** A small picture of the app in one theme: the sidebar, a title, a card with its button. */
function Picture({ theme }: { theme: Theme }) {
  return (
    <span className={cn('tp', theme === 'dark' ? 'keep-dark' : 'keep-light')}>
      <span className="tp-side">
        <i />
        <i />
        <i />
      </span>
      <span className="tp-main">
        <span className="tp-title" />
        <span className="tp-card">
          <i />
          <i />
          <b />
        </span>
      </span>
    </span>
  )
}

/** The theme a choice shows: Same as device is half light, half dark. */
export function ThemePicture({ choice }: { choice: ThemeChoice }) {
  return (
    <span className={cn('tpic', choice === 'system' && 'split')} aria-hidden="true">
      {choice === 'dark' ? <Picture theme="dark" /> : <Picture theme="light" />}
      {choice === 'system' ? <Picture theme="dark" /> : null}
    </span>
  )
}

/**
 * Account & security › Appearance (M54): Same as device, Light or Dark.
 * The theme changes as soon as one is picked and is saved to the account,
 * so it follows the member to every browser; a refused save goes back with
 * a message, as switches do.
 */
export function AppearanceCard() {
  const t = useT()
  const { choice, theme } = useTheme()
  const save = useSaveTheme()
  const heading = useId()
  const name = useId()
  const detail = {
    system: t(m.systemDetail, { theme: t(theme === 'dark' ? m.nowDark : m.nowLight) }),
    light: t(m.lightDetail),
    dark: t(m.darkDetail),
  }
  return (
    <section className="card" id="appearance" aria-labelledby={heading}>
      <CardSection id={heading} title={t(m.title)} caption={t(m.caption)} />
      <div className="pad flex flex-col gap-3">
        <fieldset className="theme-opts">
          <legend className="vh">{t(m.legend)}</legend>
          {THEME_CHOICES.map((c) => (
            <label key={c} className="opt theme-opt">
              <ThemePicture choice={c} />
              <span className="theme-opt-t">
                {/* The whole card is the target; the radio is named by its title and described by its sentence. */}
                <input type="radio" name={name} value={c} checked={choice === c} onChange={() => void save(c)} aria-labelledby={`${name}-${c}-t`} aria-describedby={`${name}-${c}-d`} />
                <span className="flex min-w-0 flex-col gap-0.5">
                  <b className="text-[13.5px]" id={`${name}-${c}-t`}>
                    {t(themeLabel[c])}
                  </b>
                  <span className="k" id={`${name}-${c}-d`}>
                    {detail[c]}
                  </span>
                </span>
              </span>
            </label>
          ))}
        </fieldset>
        <p className="k">{t(m.savedNote)}</p>
      </div>
    </section>
  )
}
