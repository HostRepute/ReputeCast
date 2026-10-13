import { Link } from '@tanstack/react-router'
import type { Schemas } from '@/api/client'
import { Badge } from '@/components/ui/Badge'
import { ButtonLink } from '@/components/ui/Button'
import { useWorkspace } from '@/lib/session'
import { Ticks } from './text'
import { WizardLayout } from './Wizard'

type Row = Schemas['ConnectDoneRow']

const ROW_BADGE: Record<Row['status'], { tone: 'ok' | 'neutral' | 'bad' | 'warn'; label: string }> = {
  done: { tone: 'ok', label: 'Done' },
  skipped: { tone: 'neutral', label: 'Skipped' },
  failed: { tone: 'bad', label: 'Failed' },
  pending: { tone: 'warn', label: 'Pending' },
}

/** 7.2 summary (SetupDone): what was configured and what to do next. */
export function DonePage({ setup }: { setup: Schemas['ConnectSetup'] }) {
  const { slug } = useWorkspace()
  const done = setup.done!
  const minutes = done.minutes === 1 ? '1 minute' : `${done.minutes} minutes`
  return (
    <WizardLayout title="Connect a sending provider" badge={<Badge tone="ok">Done</Badge>} exit={false}>
      <main id="main" tabIndex={-1} className="mx-auto box-border flex w-full max-w-[860px] flex-col gap-[18px] px-7 py-10 max-md:px-3 max-md:py-6">
        <div className="flex flex-wrap items-center gap-[18px]">
          <svg width="56" height="56" viewBox="0 0 40 40" aria-hidden="true" className="pop">
            <circle cx="20" cy="20" r="19" fill="#BBF7D0" />
            <path d="m12 20.5 5.5 5.5L28 15" stroke="#166534" strokeWidth="3" strokeLinecap="round" strokeLinejoin="round" fill="none" />
          </svg>
          <div className="flex flex-col gap-1">
            <h2 className="m-0 text-[26px] font-bold tracking-[-0.02em]">{done.title}</h2>
            <span className="text-ink-2">
              {`Setup took ${minutes}. Everything below is live`}
              {done.domain ? `, and the Overview now shows sender health for ${done.domain}.` : '.'}
            </span>
          </div>
        </div>

        <section className="card flex flex-col" aria-labelledby="configured-h">
          <div className="flex items-center justify-between gap-2 px-4.5 py-3.5">
            <h3 className="h2" id="configured-h">
              What was configured
            </h3>
            <span className="k">all reversible from Sending providers</span>
          </div>
          {done.rows.map((r) => (
            <div key={r.lead} className="cn-row lg">
              <Badge tone={ROW_BADGE[r.status].tone}>{ROW_BADGE[r.status].label}</Badge>
              <span className="min-w-0">
                <strong>{r.lead}</strong>
                <Ticks text={r.text} />
                {r.importLink ? (
                  <>
                    {' '}
                    <Link to="/w/$workspace/import" params={{ workspace: slug }}>
                      Import now
                    </Link>
                  </>
                ) : null}
              </span>
            </div>
          ))}
        </section>

        <section className="card flex flex-col gap-3 px-5 py-4.5" aria-labelledby="recommended-h">
          <h3 className="h2" id="recommended-h">
            Recommended before your first campaign
          </h3>
          <div className="grid gap-3 max-md:grid-cols-1!" style={{ gridTemplateColumns: 'repeat(auto-fit, minmax(min(240px, 100%), 1fr))' }}>
            <div className="flex flex-col gap-1.5 rounded-lg border border-line px-3.5 py-3">
              <span className="font-semibold">Add another provider</span>
              <span className="k">A second provider for System mail, or a Postal server for a weighted split. Several can run at once.</span>
              <ButtonLink size="sm" variant="primary" className="self-start" to="/w/$workspace/providers/connect" params={{ workspace: slug }}>
                Add another provider
              </ButtonLink>
            </div>
            <div className="flex flex-col gap-1.5 rounded-lg border border-line px-3.5 py-3">
              <span className="font-semibold">Connect HostRepute</span>
              <span className="k">
                {done.hostRepute ? 'Connected: 84 blocklists, Google Postmaster and Microsoft SNDS show inside Send Readiness.' : 'Optional, free: 84 blocklists, Google Postmaster and Microsoft SNDS inside Send Readiness.'}
              </span>
              <ButtonLink size="sm" className="self-start" to={done.hostRepute ? '/w/$workspace/deliverability' : '/w/$workspace/deliverability/connect'} params={{ workspace: slug }}>
                {done.hostRepute ? 'Open' : 'Connect'}
              </ButtonLink>
            </div>
            <div className="flex flex-col gap-1.5 rounded-lg border border-line px-3.5 py-3">
              <span className="font-semibold">Create a signup form</span>
              <span className="k">Double opt-in is on by default; confirmations go out on the System stream.</span>
              <ButtonLink size="sm" className="self-start" to="/w/$workspace/forms" params={{ workspace: slug }}>
                Build a form
              </ButtonLink>
            </div>
          </div>
        </section>

        <div className="flex flex-wrap justify-end gap-2.5">
          <ButtonLink to="/w/$workspace/providers/connect" params={{ workspace: slug }}>
            Add another provider
          </ButtonLink>
          <ButtonLink variant="primary" className="min-h-10 px-4.5" to="/w/$workspace" params={{ workspace: slug }}>
            Go to Overview
          </ButtonLink>
        </div>
      </main>
    </WizardLayout>
  )
}
