import { links } from '../content/links'
import { FlowCanvas } from './FlowCanvas'

const facts = [
  'Live numbers on every node: entered, sent, human opens, clicks, goal reached.',
  'Branches use human opens only, so Apple Mail prefetches never fork a flow.',
  'Each email step picks its route: SES first, Postmark as failover, with auto-pause if complaints spike.',
  'Publish a new version and choose whether contacts mid-flow finish on v3 or move to v4.',
]

/** "Automations you can watch." with the animated flow canvas. */
export function Automations() {
  return (
    <section className="wrap flex flex-col gap-5 pt-10 pb-10" id="automations" aria-labelledby="automations-h">
      <div className="reveal flex flex-col gap-2.5">
        <h2 className="sec" id="automations-h">
          Automations you can watch.
        </h2>
        <p className="lead">
          Build flows on a canvas: triggers, delays, emails, branches, A/B splits and goals. Every step shows live numbers from real events, and every email goes
          through the same readiness checks and guarded routes as a campaign.
        </p>
      </div>
      <div className="flex flex-wrap items-stretch gap-6">
        <div className="card reveal flex max-w-[340px] flex-[1_1_280px] flex-col gap-3.5 p-[22px] max-md:max-w-none!">
          <h3 className="m-0 text-[17px] font-bold">What the canvas tells you</h3>
          <ol className="m-0 flex list-none flex-col gap-3.5 p-0">
            {facts.map((f, i) => (
              <li key={f} className="flex items-start gap-2.5 text-[14px] max-md:flex-wrap">
                <span className="badge ok flex-none" aria-hidden="true">
                  {i + 1}
                </span>
                <span>{f}</span>
              </li>
            ))}
          </ol>
          <div className="mt-auto flex flex-wrap gap-2.5 pt-1.5">
            <a href={links.demoFlowEditor} className="btn pri min-h-10">
              Open the flow editor
            </a>
            <a href={links.demoAutomations} className="btn min-h-10">
              See automations
            </a>
          </div>
        </div>
        <FlowCanvas />
      </div>
    </section>
  )
}
