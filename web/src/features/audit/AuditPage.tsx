import { Page } from '@/components/shell/Page'
import { NotBuilt } from '@/components/shell/NotBuilt'
import { settingsTabs } from '@/features/settings/SettingsTabs'
import { useWorkspace } from '@/lib/session'
import type { AuditSearch } from '@/routes/w/$workspace/_app/settings/audit'

export function AuditPage({ search: _search }: { search: AuditSearch }) {
  const { slug } = useWorkspace()
  return (
    <Page title="Settings" tabs={settingsTabs(slug)}>
      <NotBuilt screen="Audit log" />
    </Page>
  )
}
