import { DropdownMenu } from 'radix-ui'
import { useId } from 'react'
import { Icon } from '@/components/icons/Icon'
import { useT } from '@/i18n'
import { isThemeChoice, THEME_CHOICES } from '@/lib/theme'
import { themeIcon, themeLabel } from './AppearanceCard'
import { m } from './messages'
import { useSaveTheme, useTheme } from './useTheme'
import './appearance.css'

/**
 * The user menu's theme row (M54): "Theme" and Same as device, Light and
 * Dark as three radio items, one tap each. The menu stays open, so the
 * change is seen before it closes.
 */
export function ThemeMenuItem() {
  const t = useT()
  const { choice } = useTheme()
  const save = useSaveTheme()
  const label = useId()
  return (
    <div className="mtheme">
      <DropdownMenu.Label className="k" id={label}>
        {t(m.menuLabel)}
      </DropdownMenu.Label>
      <DropdownMenu.RadioGroup value={choice} onValueChange={(v) => isThemeChoice(v) && save(v)} aria-labelledby={label}>
        {THEME_CHOICES.map((c) => (
          <DropdownMenu.RadioItem key={c} value={c} aria-label={t(themeLabel[c])} title={t(themeLabel[c])} onSelect={(e) => e.preventDefault()}>
            <Icon name={themeIcon[c]} size={16} />
          </DropdownMenu.RadioItem>
        ))}
      </DropdownMenu.RadioGroup>
    </div>
  )
}
