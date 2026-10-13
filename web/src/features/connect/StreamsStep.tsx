import { Link } from '@tanstack/react-router'
import { useState } from 'react'
import type { Schemas } from '@/api/client'
import { Badge } from '@/components/ui/Badge'
import { Button } from '@/components/ui/Button'
import { Note } from '@/components/ui/Note'
import { cn } from '@/lib/cn'
import { useWorkspace } from '@/lib/session'
import { failed, useSetupActions, type StepProps } from './parts'
import { continueClass, StepHeading, WizardFooter } from './Wizard'

type View = Schemas['ConnectStreamView']
type Choice = { enabled: boolean; role: View['role']; weight: number }

const ROLE_WORDS: Record<View['role'], string> = { primary: 'Primary', split: 'Weighted split', fallback: 'Fallback' }

/** 7.2 step 5 (SetupStreams): which streams the provider carries, and in which role. */
export function StreamsStep({ setup, go, reason, finished }: StepProps) {
  const { slug } = useWorkspace()
  const a = useSetupActions(slug, setup.id)
  const views = setup.streams.streams
  const [choices, setChoices] = useState<Record<string, Choice>>(() =>
    Object.fromEntries(views.map((v) => [v.stream, { enabled: v.enabled && v.allowed, role: v.role, weight: v.weight || 100 }])),
  )
  const set = (stream: string, patch: Partial<Choice>) => setChoices((c) => ({ ...c, [stream]: { ...c[stream]!, ...patch } }))
  const save = () =>
    a.streams.mutate(
      {
        streams: views.map((v) => {
          const c = choices[v.stream]!
          return { stream: v.stream, enabled: c.enabled, role: c.enabled ? c.role : undefined, weight: c.enabled && c.role === 'split' ? c.weight : undefined }
        }),
      },
      { onSuccess: () => go('contacts'), onError: failed("Couldn't save the streams", save) },
    )
  const none = !Object.values(choices).some((c) => c.enabled)
  return (
    <>
      <StepHeading title={`Which streams should ${setup.info.name} carry?`}>
        A stream is a kind of mail. Each one gets its own route, failover policy and configuration set, so a bad campaign never blocks a password reset.
      </StepHeading>
      {finished ? (
        <Note tone="neutral">
          This setup is finished: these are the streams it set up. Routing has its own page now.{' '}
          <Link to="/w/$workspace/providers/routing" params={{ workspace: slug }}>
            Open Routing
          </Link>
        </Note>
      ) : null}
      <div className="grid gap-3.5 max-md:grid-cols-1!" style={{ gridTemplateColumns: 'repeat(auto-fit, minmax(min(300px, 100%), 1fr))' }}>
        {views.map((v) => (
          <StreamCard
            key={v.stream}
            view={finished ? { ...v, hasRoute: false, current: [] } : v}
            choice={choices[v.stream]!}
            onChange={(p) => set(v.stream, p)}
            disabled={Boolean(reason) || finished}
            short={setup.info.short}
          />
        ))}
      </div>
      {setup.streams.tips.length || setup.streams.defaults ? (
        <section className="card flex flex-col gap-2.5 px-5 py-4" aria-labelledby="next-h">
          <h3 className="h2" id="next-h">
            Suggested next steps
          </h3>
          {setup.streams.tips.map((t) => (
            <div key={t} className="flex items-start gap-3 text-[13px] max-md:flex-wrap">
              <Badge tone="info">Tip</Badge>
              <span>{t}</span>
            </div>
          ))}
          {setup.streams.defaults ? (
            <div className="flex items-start gap-3 text-[13px] max-md:flex-wrap">
              <Badge tone="ok">Set</Badge>
              <span>{setup.streams.defaults}</span>
            </div>
          ) : null}
        </section>
      ) : null}
      <WizardFooter back={<Button onClick={() => go('test')}>Back</Button>}>
        <Button
          variant="primary"
          className={continueClass}
          onClick={finished ? () => go('contacts') : save}
          loading={a.streams.isPending}
          disabledReason={finished ? undefined : (reason ?? (none ? `Pick at least one stream for ${setup.info.short}.` : undefined))}
        >
          Continue to contacts
        </Button>
      </WizardFooter>
    </>
  )
}

function StreamCard({ view: v, choice: c, onChange, disabled, short }: { view: View; choice: Choice; onChange: (p: Partial<Choice>) => void; disabled: boolean; short: string }) {
  const on = c.enabled
  const roleLabel = !v.hasRoute || c.role === v.role ? v.roleLabel : c.role === 'split' ? `Split · ${c.weight}%` : ROLE_WORDS[c.role]
  const title = (
    <>
      <span className="text-[15px] font-bold">{v.name}</span>
      <br />
      <span className="k">{v.allowed ? v.description : (v.reason ?? v.description)}</span>
    </>
  )
  return (
    <section className={cn('card flex flex-col gap-3 px-5 py-4.5', on && 'border-accent bg-accent-tint')} aria-label={v.name}>
      <label className="cn-sw">
        <input type="checkbox" checked={on} onChange={(e) => onChange({ enabled: e.target.checked })} disabled={disabled || !v.allowed} />
        <span>{title}</span>
      </label>
      {on ? (
        <div className="flex flex-col gap-1 text-[13px]">
          {v.hasRoute ? (
            <label className="flex items-center justify-between gap-3">
              <span className="k">{`Next to ${v.current.join(', ')}`}</span>
              <select className="in h-[30px] w-auto" value={c.role} onChange={(e) => onChange({ role: e.target.value as View['role'] })} disabled={disabled}>
                {v.stream === 'broadcast' ? <option value="split">Weighted split</option> : null}
                <option value="primary">{`Primary, replacing ${v.current.join(', ')}`}</option>
                <option value="fallback">Fallback</option>
              </select>
            </label>
          ) : null}
          {v.hasRoute && c.role === 'split' ? (
            <label className="flex items-center justify-between gap-3">
              <span className="k">{`${short}'s share`}</span>
              <span className="flex items-center gap-1.5">
                <input
                  className="in h-[30px] w-[72px]"
                  type="number"
                  inputMode="numeric"
                  min={1}
                  max={99}
                  value={c.weight}
                  onChange={(e) => onChange({ weight: Math.max(1, Math.min(99, Number(e.target.value) || 1)) })}
                  disabled={disabled}
                />
                %
              </span>
            </label>
          ) : null}
          <div className="flex justify-between gap-3">
            <span className="k">Role</span>
            <Badge tone="purple">{roleLabel}</Badge>
          </div>
          {v.configurationSet ? (
            <div className="flex justify-between gap-3">
              <span className="k">Configuration set</span>
              <span className="mono">{v.configurationSet}</span>
            </div>
          ) : null}
          {v.failover ? (
            <div className="flex justify-between gap-3">
              <span className="k">Failover</span>
              <span>{v.failover}</span>
            </div>
          ) : null}
          {v.tracking ? (
            <div className="flex justify-between gap-3">
              <span className="k">Tracking</span>
              <span>{v.tracking}</span>
            </div>
          ) : null}
        </div>
      ) : null}
    </section>
  )
}
