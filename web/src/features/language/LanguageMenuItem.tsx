import { Link } from '@tanstack/react-router'
import { DropdownMenu } from 'radix-ui'
import { Icon } from '@/components/icons/Icon'
import { languageName, useI18n, useT } from '@/i18n'
import { m } from './messages'

/** The user menu's "Language: English", which opens Language & region (ShellUserMenu board). */
export function LanguageMenuItem({ slug }: { slug: string }) {
  const t = useT()
  const { locale } = useI18n()
  return (
    <DropdownMenu.Item asChild>
      <Link to="/w/$workspace/account" params={{ workspace: slug }} hash="language">
        <Icon name="language" />
        {t(m.menuItem, { language: languageName(locale) })}
      </Link>
    </DropdownMenu.Item>
  )
}
