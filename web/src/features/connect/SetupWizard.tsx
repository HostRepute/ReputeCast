import { useNavigate } from '@tanstack/react-router'
import { useEffect, useRef, useState } from 'react'
import { $api, ApiError, errorMessage, http } from '@/api/client'
import { Button, ButtonLink } from '@/components/ui/Button'
import { ErrorPanel, Skeleton } from '@/components/ui/Feedback'
import { Note } from '@/components/ui/Note'
import { toast } from '@/components/ui/Toast'
import { ReadOnlyNote } from '@/features/senders/ReadOnlyNote'
import { useT } from '@/i18n'
import { common } from '@/i18n/common'
import { deniedReason, useCan, useWorkspace } from '@/lib/session'
import { stepNumber, useSetup, type Setup, type SetupStep } from './api'
import { ConnectionAPI } from './ConnectionAPI'
import { ConnectionSES } from './ConnectionSES'
import { ConnectionSMTP } from './ConnectionSMTP'
import { ContactsStep } from './ContactsStep'
import { DomainStep } from './DomainStep'
import { DonePage } from './DonePage'
import { m } from './messages'
import { failed, useSetupActions, type StepProps } from './parts'
import { ms } from './setup.messages'
import { StreamsStep } from './StreamsStep'
import { TestStep } from './TestStep'
import { StepBadge, StepHeading, WizardBody, WizardLayout } from './Wizard'

export interface WizardSearch {
  setup?: string
  step?: SetupStep
}

/**
 * 7.2 steps 2–6 and the summary for one adapter. The URL names the setup
 * and the step (?setup=…&step=domain), so a step can be shared and the back
 * button walks back through the steps. Without a setup the wizard starts
 * one, or resumes the adapter's unfinished one.
 */
export function SetupWizard({ adapter, search, onSearch }: { adapter: string; search: WizardSearch; onSearch: (next: WizardSearch, replace?: boolean) => void }) {
  if (!search.setup) return <StartSetup adapter={adapter} onStarted={(s) => onSearch({ setup: s.id, step: s.status === 'finished' ? 'done' : s.step }, true)} />
  return <SetupScreens key={search.setup} setupId={search.setup} step={search.step} onStep={(step) => onSearch({ setup: search.setup, step })} />
}

function StartSetup({ adapter, onStarted }: { adapter: string; onStarted: (s: Setup) => void }) {
  const t = useT()
  const { slug } = useWorkspace()
  const can = useCan('providers:manage')
  // Who can connect providers, for people who can't.
  const catalog = $api.useQuery('get', '/api/v1/workspaces/{workspace}/connect/catalog', { params: { path: { workspace: slug } } }, { enabled: !can })
  const [error, setError] = useState<unknown>()
  const started = useRef(false)
  const start = () => {
    started.current = true
    setError(undefined)
    http
      .POST('/api/v1/workspaces/{workspace}/connect/setups', { params: { path: { workspace: slug } }, body: { adapter } })
      .then(({ data }) => onStarted(data!))
      .catch(setError)
  }
  useEffect(() => {
    if (can && !started.current) start()
    // Starts once per visit; a retry is the person's choice.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [can])
  const unknown = error instanceof ApiError && error.status === 404
  return (
    <WizardLayout title={t(m.title)} badge={<StepBadge n={2} />}>
      <WizardBody current={2}>
        {!can ? (
          <>
            <StepHeading title={t(m.title)} />
            <ReadOnlyNote msg={m.readOnlyCatalog} helpers={catalog.data?.helpers} />
            <div>
              <ButtonLink to="/w/$workspace/providers/connect" params={{ workspace: slug }}>
                {t(ms.backToCatalog)}
              </ButtonLink>
            </div>
          </>
        ) : error ? (
          <ErrorPanel
            title={unknown ? t(ms.unknownAdapter) : t(ms.startFailed)}
            actions={
              <>
                {unknown ? null : <Button onClick={start}>{t(common.tryAgain)}</Button>}
                <ButtonLink to="/w/$workspace/providers/connect" params={{ workspace: slug }}>
                  {t(ms.backToCatalog)}
                </ButtonLink>
              </>
            }
          >
            {unknown ? t(ms.unknownAdapterBody) : errorMessage(error)}
          </ErrorPanel>
        ) : (
          <div className="flex flex-col gap-3" aria-busy="true">
            <span className="vh" role="status">
              {t(ms.starting)}
            </span>
            <Skeleton w="50%" h={28} />
            <Skeleton w="80%" h={14} />
            <Skeleton w="100%" h={220} r={10} />
          </div>
        )}
      </WizardBody>
    </WizardLayout>
  )
}

function SetupScreens({ setupId, step, onStep }: { setupId: string; step?: SetupStep; onStep: (s: SetupStep) => void }) {
  const t = useT()
  const { slug } = useWorkspace()
  const navigate = useNavigate()
  const q = useSetup(slug, setupId)
  const a = useSetupActions(slug, setupId)
  const can = useCan('providers:manage')
  const setup = q.data
  if (!setup) {
    return (
      <WizardLayout title={t(m.title)} badge={<StepBadge n={2} />}>
        <WizardBody current={2}>
          {q.isError ? (
            <ErrorPanel
              title={q.error instanceof ApiError && q.error.status === 404 ? t(ms.notFound) : t(ms.loadFailed)}
              actions={
                <>
                  <Button onClick={() => void q.refetch()}>{t(common.retry)}</Button>
                  <ButtonLink to="/w/$workspace/providers/connect" params={{ workspace: slug }}>
                    {t(ms.backToCatalog)}
                  </ButtonLink>
                </>
              }
            >
              {errorMessage(q.error)}
            </ErrorPanel>
          ) : (
            <div className="flex flex-col gap-3" aria-busy="true">
              <span className="vh" role="status">
                {t(ms.loading)}
              </span>
              <Skeleton w="50%" h={28} />
              <Skeleton w="80%" h={14} />
              <Skeleton w="100%" h={220} r={10} />
            </div>
          )}
        </WizardBody>
      </WizardLayout>
    )
  }
  const view = viewStep(setup, step)
  if (view === 'done' && setup.done) return <DonePage setup={setup} />
  const reason = can && setup.status !== 'abandoned' ? undefined : setup.status === 'abandoned' ? t(ms.cancelledReason) : deniedReason('providers:manage', setup.helpers)
  const go = (next: SetupStep) => {
    onStep(next)
    // The setup remembers where it was, so it resumes there.
    if (!reason && setup.status === 'open' && next !== 'done' && next !== setup.step) a.step.mutate({ step: next as Exclude<SetupStep, 'done'> })
  }
  const saveLater = () =>
    a.step.mutate(
      { step: view as Exclude<SetupStep, 'done'> },
      {
        onSuccess: () => {
          toast({ title: t(ms.savedForLater, { provider: setup.info.name }) })
          void navigate({ to: '/w/$workspace/providers', params: { workspace: slug } })
        },
        onError: failed(t(ms.saveFailed)),
      },
    )
  const props: StepProps = { setup, go, saveLater, reason, finished: setup.status === 'finished' }
  return (
    <WizardLayout title={setup.info.title} badge={<StepBadge n={stepNumber[view]} />}>
      <WizardBody current={stepNumber[view]} setup={setup}>
        {setup.status === 'abandoned' ? <Note tone="warn">{t(ms.cancelledNote)}</Note> : null}
        {!can && setup.status !== 'abandoned' ? (
          <ReadOnlyNote msg={m.readOnlySetup} helpers={setup.helpers} />
        ) : null}
        <StepScreen view={view} props={props} />
      </WizardBody>
    </WizardLayout>
  )
}

function StepScreen({ view, props }: { view: SetupStep; props: StepProps }) {
  switch (view) {
    case 'domain':
      return <DomainStep {...props} />
    case 'test':
      return <TestStep {...props} />
    case 'streams':
      return <StreamsStep {...props} />
    case 'contacts':
      return <ContactsStep {...props} />
  }
  switch (props.setup.info.flow) {
    case 'ses':
      return <ConnectionSES {...props} />
    case 'smtp':
      return <ConnectionSMTP {...props} />
  }
  return <ConnectionAPI {...props} />
}

/**
 * The step to show: the one in the URL, else where the setup is. Steps after
 * the connection need a connected provider, and the summary a finished setup.
 */
export function viewStep(setup: Pick<Setup, 'step' | 'status' | 'providerId'>, step?: SetupStep): SetupStep {
  const want = step ?? (setup.status === 'finished' ? 'done' : setup.step)
  if (want === 'done') return setup.status === 'finished' ? 'done' : setup.providerId ? 'contacts' : 'connection'
  if (want !== 'connection' && !setup.providerId) return 'connection'
  return want
}

