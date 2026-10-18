import { useState, type ReactNode } from 'react'
import { $api, type Schemas } from '@/api/client'
import { Badge } from '@/components/ui/Badge'
import { Button, ButtonLink } from '@/components/ui/Button'
import { ConfirmDialog } from '@/components/ui/ConfirmDialog'
import { Banner } from '@/components/ui/Note'
import { Segmented } from '@/components/ui/Tabs'
import { toast } from '@/components/ui/Toast'
import { useT } from '@/i18n'
import { clock, dayMonthDate, shortDate } from '@/lib/format'
import { deniedReason, useCan, useWorkspace } from '@/lib/session'
import { m } from './messages'
import { mr } from './referrals.messages'
import { GROW, useGrowRefresh } from './shared'

export type Program = Schemas['ReferralProgramDetail']
export type Period = '7d' | '30d' | 'all'

/** "31,988 subscribers, each with a personal link · on since Tue 23 Sep, turned on by Sofia Rossi · announced in Weekly Digest #139". */
export function programLine(t: ReturnType<typeof useT>, p: Program): string {
  if (p.status === 'off') {
    if (!p.startedAt) return t(mr.headerNeverOn, { step: p.setupStep })
    const when = p.statusAt ? `${dayMonthDate(p.statusAt)}, ${clock(p.statusAt)}` : ''
    return p.statusBy ? t(mr.headerOff, { date: when, by: p.statusBy }) : t(mr.headerOffNoBy, { date: when })
  }
  const since = p.startedAt ? shortDate(p.startedAt) : ''
  const base = p.startedBy ? t(mr.headerOn, { count: p.links, date: since, by: p.startedBy }) : t(mr.headerOnNoBy, { count: p.links, date: since })
  return p.announcedIn ? t(mr.headerAnnounced, { line: base, campaign: p.announcedIn }) : base
}

/** Turning a program off (confirmed) and on again. */
export function useProgramSwitch(p: Program) {
  const t = useT()
  const { slug } = useWorkspace()
  const refresh = useGrowRefresh()
  const [asking, setAsking] = useState<'on' | 'off'>()
  const off = $api.useMutation('post', `${GROW}/referrals/{program}/off`)
  const on = $api.useMutation('post', `${GROW}/referrals/{program}/on`)
  const dialog = (
    <ConfirmDialog
      open={asking !== undefined}
      onOpenChange={(o) => !o && setAsking(undefined)}
      tone={asking === 'off' ? 'warn' : 'ok'}
      badge={asking === 'off' ? t(mr.offConfirmAction) : t(mr.onConfirmAction)}
      title={asking === 'off' ? t(mr.offConfirmTitle, { list: p.listName }) : t(mr.onConfirmTitle, { list: p.listName })}
      description={asking === 'off' ? t(mr.offConfirmText) : t(mr.onConfirmText)}
      confirmLabel={asking === 'off' ? t(mr.offConfirmAction) : t(mr.onConfirmAction)}
      confirmVariant="primary"
      onConfirm={async () => {
        const params = { params: { path: { workspace: slug, program: p.id } } }
        if (asking === 'off') {
          await off.mutateAsync(params)
          toast({ title: t(mr.offDone), detail: t(mr.offDoneDetail) })
        } else {
          const r = await on.mutateAsync(params)
          toast({ kind: 'success', title: t(mr.onDone), detail: t(mr.onDoneDetail, { count: r.linksCreated || r.program.links }) })
        }
        setAsking(undefined)
        await refresh()
      }}
    />
  )
  return { ask: setAsking, dialog }
}

/**
 * The program's header card (Referrals › Results, ReferralStates): name,
 * status, the line under it, the period and Turn off / Turn on.
 */
export function ProgramHeader({ p, period, onPeriod, helpers }: { p: Program; period?: Period; onPeriod?: (p: Period) => void; helpers: string[] }) {
  const t = useT()
  const canEdit = useCan('grow:edit')
  const sw = useProgramSwitch(p)
  const on = p.status === 'on'
  return (
    <section className="card flex flex-wrap items-center gap-x-5 gap-y-3 px-[18px] py-3.5" aria-label={t(m.programNav)}>
      <div className="flex min-w-0 flex-[1_1_380px] flex-col gap-[3px]">
        <div className="flex flex-wrap items-center gap-2">
          <h2 className="h2">{t(mr.programName, { list: p.listName })}</h2>
          <Badge tone={on ? 'ok' : 'neutral'} dot>
            {on ? t(m.on) : t(m.off)}
          </Badge>
        </div>
        <span className="k">{programLine(t, p)}</span>
      </div>
      <div className="flex flex-wrap items-center gap-2.5">
        {period && onPeriod ? (
          <Segmented
            label={t(mr.period)}
            value={period}
            onChange={onPeriod}
            items={[
              { value: '7d', label: t(mr.period7) },
              { value: '30d', label: t(mr.period30) },
              { value: 'all', label: t(mr.periodAll) },
            ]}
          />
        ) : null}
        {canEdit ? (
          <Button variant={on ? undefined : 'primary'} onClick={() => sw.ask(on ? 'off' : 'on')}>
            {on ? t(mr.turnOff) : t(mr.turnOn)}
          </Button>
        ) : null}
      </div>
      {canEdit ? sw.dialog : null}
      {!canEdit && helpers.length === 0 ? <span className="vh">{deniedReason('grow:edit')}</span> : null}
    </section>
  )
}

/** "The referral program is off since 15 Oct, 09:12 (Sofia Rossi)." with Turn on and Settings. */
export function OffBanner({ p, settings = true }: { p: Program; settings?: boolean }) {
  const t = useT()
  const { slug } = useWorkspace()
  const canEdit = useCan('grow:edit')
  const sw = useProgramSwitch(p)
  if (p.status !== 'off' || !p.startedAt) return null
  const when = p.statusAt ? `${dayMonthDate(p.statusAt)}, ${clock(p.statusAt)}` : ''
  const actions: ReactNode = (
    <>
      {canEdit ? (
        <Button variant="primary" size="sm" onClick={() => sw.ask('on')}>
          {t(mr.turnOn)}
        </Button>
      ) : null}
      {settings ? (
        <ButtonLink to="/w/$workspace/grow/referrals/settings" params={{ workspace: slug }} search={{ program: p.id }} size="sm">
          {t(mr.offSettings)}
        </ButtonLink>
      ) : null}
    </>
  )
  return (
    <>
      <Banner tone="warn" title={p.statusBy ? t(mr.offTitle, { date: when, by: p.statusBy }) : t(mr.offTitleNoBy, { date: when })} actions={actions}>
        {t(mr.offText)}
      </Banner>
      {canEdit ? sw.dialog : null}
    </>
  )
}
