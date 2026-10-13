import { useNavigate } from '@tanstack/react-router'
import { useState } from 'react'
import { $api, errorMessage, type Schemas } from '@/api/client'
import { Badge } from '@/components/ui/Badge'
import { Button } from '@/components/ui/Button'
import { ErrorPanel, Skeleton } from '@/components/ui/Feedback'
import { count } from '@/lib/format'
import { useWorkspace } from '@/lib/session'
import { failed, useSetupActions, type StepProps } from './parts'
import { continueClass, StepHeading, WizardFooter } from './Wizard'

type Contacts = Schemas['ConnectContacts']

const WORDS = ['No', 'One', 'Two', 'Three', 'Four', 'Five', 'Six']

/** 7.2 step 6 (SetupContacts): import, pull suppressions, or start empty. */
export function ContactsStep({ setup, go, reason, finished }: StepProps) {
  const { slug } = useWorkspace()
  const q = $api.useQuery('get', '/api/v1/workspaces/{workspace}/connect/setups/{setupId}/contacts', { params: { path: { workspace: slug, setupId: setup.id } } })
  return (
    <>
      <StepHeading title="Bring your contacts">
        {`Import keeps consent intact: unsubscribed, bounced and complained contacts become suppressions and are never re-subscribed.${
          q.data?.suppressions.available ? ` Existing suppressions can also be pulled from ${setup.info.name}.` : ''
        }`}
      </StepHeading>
      {q.data ? (
        <ContactsBody setup={setup} contacts={q.data} go={go} reason={reason} finished={finished} />
      ) : q.isError ? (
        <ErrorPanel title="Couldn't load the import options" actions={<Button onClick={() => void q.refetch()}>Retry</Button>}>
          {errorMessage(q.error)}
        </ErrorPanel>
      ) : (
        <div className="grid gap-3" style={{ gridTemplateColumns: 'repeat(auto-fit, minmax(min(220px, 100%), 1fr))' }} aria-busy="true">
          <span className="vh" role="status">
            Loading the import options…
          </span>
          {[0, 1, 2, 3, 4, 5].map((i) => (
            <Skeleton key={i} h={92} r={10} />
          ))}
        </div>
      )}
    </>
  )
}

function ContactsBody({ setup, contacts: c, go, reason, finished }: { setup: Schemas['ConnectSetup']; contacts: Contacts; go: StepProps['go']; reason?: string; finished: boolean }) {
  const { slug } = useWorkspace()
  const navigate = useNavigate()
  const a = useSetupActions(slug, setup.id)
  const sup = c.suppressions
  const [pull, setPull] = useState(sup.available && (c.pull || !setup.finishedAt))
  const finish = (body: Omit<Schemas['ConnectFinishInput'], 'pullSuppressions'>, then?: () => void) =>
    a.finish.mutate({ ...body, pullSuppressions: pull && sup.available }, { onSuccess: () => (then ? then() : go('done')), onError: failed("Couldn't finish the setup") })
  const lists = c.suggested.length ? c.suggested : c.lists
  return (
    <>
      <div className="grid gap-3 max-md:grid-cols-[repeat(auto-fit,minmax(140px,1fr))]!" style={{ gridTemplateColumns: 'repeat(auto-fit, minmax(min(220px, 100%), 1fr))' }}>
        {c.sources.map((s) => (
          <button
            key={s.key}
            type="button"
            className="cn-src"
            onClick={() =>
              reason || finished
                ? void navigate({ to: '/w/$workspace/import', params: { workspace: slug } })
                : finish({ startEmpty: false, source: s.key as Schemas['ConnectFinishInput']['source'] }, () => void navigate({ to: '/w/$workspace/import', params: { workspace: slug } }))
            }
          >
            <span className="font-bold">{s.name}</span>
            <span className="k">{s.detail}</span>
            <Badge tone={s.tone} className="self-start">
              {s.badge}
            </Badge>
          </button>
        ))}
      </div>

      {sup.available ? (
        <section className="card flex flex-col gap-3 px-5 py-4.5" aria-labelledby="sup-h">
          <div className="flex flex-wrap items-center justify-between gap-2">
            <h3 className="h2" id="sup-h">{`Pull existing suppressions from ${sup.provider}`}</h3>
            <Badge tone="ok" dot>
              Recommended
            </Badge>
          </div>
          <span className="text-[13px] text-ink-2">{`Your ${sup.short} account-level suppression list has ${count(sup.total)} ${sup.total === 1 ? 'address' : 'addresses'} from earlier sending. Importing them now means ReputeCast never tries them, on any provider.`}</span>
          <label className="flex items-center gap-2.5 text-[13px]">
            <input type="checkbox" checked={pull} onChange={(e) => setPull(e.target.checked)} disabled={Boolean(reason) || finished} />
            {`Import ${count(sup.total)} suppressed ${sup.total === 1 ? 'address' : 'addresses'} from ${sup.short} (bounces ${count(sup.bounces)}, complaints ${count(sup.complaints)})`}
          </label>
        </section>
      ) : sup.error ? (
        <section className="card flex flex-col gap-1.5 px-5 py-4.5">
          <h3 className="h2">{`Suppressions from ${sup.provider}`}</h3>
          <span className="text-[13px] text-ink-2">{sup.error}</span>
        </section>
      ) : null}

      <section className="card flex flex-col gap-2.5 border-dashed px-5 py-4.5" aria-labelledby="empty-h">
        <h3 className="h2" id="empty-h">
          Or start empty
        </h3>
        <span className="text-[13px] text-ink-2">
          {c.suggested.length
            ? `Create the lists now and grow them with signup forms, the API or a later import. ${WORDS[c.suggested.length] ?? c.suggested.length} ${c.suggested.length === 1 ? 'list is' : 'lists are'} suggested from your workspace name; rename them any time.`
            : `Keep the workspace's ${c.lists.length === 1 ? 'list' : `${c.lists.length} lists`} and grow them with signup forms, the API or a later import.`}
        </span>
        {lists.length ? (
          <div className="flex flex-wrap gap-2">
            {lists.map((l) => (
              <Badge key={l} tone="neutral">
                {l}
              </Badge>
            ))}
          </div>
        ) : null}
        <Button className="self-start" onClick={() => finish({ startEmpty: true })} loading={a.finish.isPending} disabledReason={finished ? 'This setup is finished.' : reason}>
          Start empty and finish
        </Button>
      </section>

      <WizardFooter back={<Button onClick={() => go('streams')}>Back</Button>}>
        <Button variant="primary" className={continueClass} onClick={() => (finished ? go('done') : finish({ startEmpty: false }))} loading={a.finish.isPending} disabledReason={finished ? undefined : reason}>
          Finish setup
        </Button>
      </WizardFooter>
    </>
  )
}
