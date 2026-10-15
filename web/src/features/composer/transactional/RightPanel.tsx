import { PanelTabs, TabPanel } from '@/components/ui/Tabs'
import { useT } from '@/i18n'
import { ApiTab } from './ApiTab'
import { ChecksTab } from './ChecksTab'
import { DesignTab } from './DesignTab'
import { m } from './messages'
import { useTx } from './state'
import { TestDataTab } from './TestDataTab'
import type { TxTab } from './types'
import { VariablesTab } from './VariablesTab'

/** Right panel (TransactionalTemplate): Design · Variables · Test data · Checks · API. */
export function TxRightPanel() {
  const t = useT()
  const tx = useTx()
  const issues = (tx.checks?.blocks ?? 0) + (tx.checks?.warnings ?? 0)
  const items: { value: TxTab; label: string }[] = [
    { value: 'design', label: t(m.tabDesign) },
    { value: 'variables', label: t(m.tabVariables) },
    { value: 'testdata', label: t(m.tabTestData) },
    { value: 'checks', label: issues ? t(m.tabChecksCount, { count: issues }) : t(m.tabChecks) },
    { value: 'api', label: t(m.tabApi) },
  ]
  return (
    <aside className="cpanel right txe-right" aria-label={t(m.panelLabel)}>
      <PanelTabs value={tx.tab} onChange={tx.setTab} label={t(m.panelLabel)} items={items} className="flex flex-1 flex-col">
        <TabPanel value="design" className="flex flex-1 flex-col" tabIndex={-1}>
          <DesignTab />
        </TabPanel>
        <TabPanel value="variables" className="flex flex-1 flex-col" tabIndex={-1}>
          <VariablesTab />
        </TabPanel>
        <TabPanel value="testdata" className="flex flex-1 flex-col" tabIndex={-1}>
          <TestDataTab />
        </TabPanel>
        <TabPanel value="checks" className="flex flex-1 flex-col" tabIndex={-1}>
          <ChecksTab />
        </TabPanel>
        <TabPanel value="api" className="flex flex-1 flex-col" tabIndex={-1}>
          <ApiTab />
        </TabPanel>
      </PanelTabs>
    </aside>
  )
}
