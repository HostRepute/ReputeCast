import { useT } from '@/i18n'
import { EmptyState } from '../ui/Feedback'
import { m } from './messages'

export function NotBuilt({ screen }: { screen: string }) {
  const t = useT()
  return <EmptyState title={t(m.notBuiltTitle, { screen })}>{t(m.notBuiltBody)}</EmptyState>
}
