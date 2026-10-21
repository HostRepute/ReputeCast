import { PanelTabs, TabPanel } from '@/components/ui/Tabs'
import { useT } from '@/i18n'
import { BlockTab } from './BlockTab'
import { ChecksTab } from './ChecksTab'
import { useBuilder, type RightTab } from './context'
import { m } from './messages'
import { PageTab } from './PageTab'

/** The right panel (PageBuilder): the Block, Page and Checks tabs. */
export function RightPanel() {
  const t = useT()
  const b = useBuilder()
  const items: { value: RightTab; label: string }[] = [
    { value: 'block', label: t(m.tabBlock) },
    { value: 'page', label: t(m.tabPage) },
    { value: 'checks', label: t(m.tabChecks) },
  ]
  return (
    <aside className="pb-right" aria-label={t(m.panelLabel)}>
      <div className="pb-sticky">
        <PanelTabs value={b.rightTab} onChange={b.setRightTab} items={items} label={t(m.panelLabel)}>
          <TabPanel value="block">
            <BlockTab />
          </TabPanel>
          <TabPanel value="page">
            <PageTab />
          </TabPanel>
          <TabPanel value="checks">
            <ChecksTab />
          </TabPanel>
        </PanelTabs>
      </div>
    </aside>
  )
}
