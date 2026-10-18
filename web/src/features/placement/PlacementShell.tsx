import { useNavigate } from '@tanstack/react-router'
import { useState, type ReactNode } from 'react'
import { Page } from '@/components/shell/Page'
import { Badge } from '@/components/ui/Badge'
import { Button, ButtonLink } from '@/components/ui/Button'
import { useT } from '@/i18n'
import { useWorkspace } from '@/lib/session'
import { m } from './messages'
import { RunDialog, type RunPreset } from './RunDialog'
import { DeliverabilityTabs, PlacementPills, usePlacementAccess, useTests, type TestList } from './shared'

/**
 * Deliverability › Placement tests: the header (section tabs, Seed
 * mailboxes, Run placement test), the Tests · Seed mailboxes pills and the
 * Run a placement test dialog.
 */
export function PlacementShell({
  current,
  documentTitle,
  preset,
  children,
}: {
  current: 'tests' | 'seeds'
  documentTitle: string
  /** Opens Run a placement test with what an entry point knows. */
  preset?: RunPreset
  children: ReactNode
}) {
  const t = useT()
  const q = useTests()
  const d = q.data
  const navigate = useNavigate()
  const { slug } = useWorkspace()
  const [run, setRun] = useState(false)
  const open = run || Boolean(preset)
  const close = (next: boolean) => {
    if (next) return
    setRun(false)
    // An entry point's preset lives in the URL; closing drops it.
    if (preset) void navigate({ to: '/w/$workspace/deliverability/placement', params: { workspace: slug }, search: {}, replace: true })
  }
  return (
    <Page title={t(m.deliverability)} documentTitle={documentTitle} filters={<DeliverabilityTabs />} actions={d ? <HeaderActions d={d} onRun={() => setRun(true)} /> : null}>
      <PlacementPills current={current} tests={d?.tests.length} seeds={d?.seedsTotal} />
      {children}
      {open && d?.canRun ? <RunDialog open onOpenChange={close} preset={preset ?? { origin: 'placement' }} /> : null}
    </Page>
  )
}

function HeaderActions({ d, onRun }: { d: TestList; onRun: () => void }) {
  const t = useT()
  const { slug } = useWorkspace()
  const access = usePlacementAccess(d)
  if (access.readOnly) return <Badge tone="neutral">{t(m.readOnly)}</Badge>
  const attention = access.canManage ? d.attention.length : 0
  return (
    <>
      <ButtonLink to="/w/$workspace/deliverability/placement/seeds" params={{ workspace: slug }}>
        {attention ? <span className="dot text-warn-bar" aria-hidden="true" /> : null}
        {attention ? t(m.seedsButtonAttention, { count: d.seedsTotal, attention }) : t(m.seedsButton, { count: d.seedsTotal })}
      </ButtonLink>
      <Button variant="primary" onClick={onRun} disabledReason={access.runReason}>
        {t(m.runTest)}
      </Button>
    </>
  )
}
