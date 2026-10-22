import type { ReactNode } from 'react'
import { useInstanceMeta } from '@/components/shell/AuthLayout'
import { ButtonLink } from '@/components/ui/Button'
import { Grid } from '@/components/ui/Layout'
import { cn } from '@/lib/cn'
import { count } from '@/lib/format'
import { useWorkspace } from '@/lib/session'
import type { Schemas } from '@/api/client'
import { useT } from '@/i18n'
import { m } from './messages'
import { PendingNextCard } from './NextCampaignCard'
import { PendingHealthCard } from './SenderHealthCard'

function Step({ n, current, title, children, action }: { n: number; current?: boolean; title: string; children: ReactNode; action?: ReactNode }) {
  const t = useT()
  return (
    <div className="flex items-start gap-4 border-t border-line-2 px-5 py-4.5">
      <span
        className={cn('inline-flex h-[30px] w-[30px] flex-none items-center justify-center rounded-full text-[13px] font-bold', current ? 'bg-accent text-white' : 'bg-line-2 text-ink-3')}
        aria-hidden="true"
      >
        {n}
      </span>
      <div className={cn('flex flex-[1_1_300px] flex-col', current ? 'gap-1.5' : 'gap-1')}>
        <span className={cn('text-[15px] font-semibold', !current && 'text-ink-3')}>
          <span className="vh">{t(m.step, { n, current: current ? 'yes' : 'no' })} </span>
          {title}
        </span>
        {children}
      </div>
      {action}
    </div>
  )
}

/** The first-run Overview: connect a provider first (OverviewEmpty board). */
export function FirstRun({ audience }: { audience?: Schemas['OverviewAudience'] }) {
  const t = useT()
  const { slug, me } = useWorkspace()
  const meta = useInstanceMeta()
  const params = { workspace: slug }
  const lists = audience?.lists.map((l) => l.name) ?? []
  const subscribers = audience?.subscribers ?? 0
  return (
    <>
      <section className="card flex flex-col" aria-labelledby="ov-welcome-h">
        <div className="flex flex-col gap-2 px-6 pb-5 pt-[26px]">
          <span className="badge info self-start">{t(m.selfHosted, { version: meta.data?.version ?? 'v1.0' })}</span>
          <h2 id="ov-welcome-h" className="m-0 text-[24px] font-bold tracking-[-0.02em]">
            {t(m.welcome, { name: me.workspace.name })}
          </h2>
          <p className="m-0 max-w-[620px] text-ink-2">{t(m.welcomeLead)}</p>
        </div>
        <Step n={1} current title={t(m.stepConnect)}>
          <span className="text-ink-2">{t(m.stepConnectLine)}</span>
          <div className="mt-1.5 flex flex-wrap gap-2.5">
            <ButtonLink variant="primary" to="/w/$workspace/providers/connect" params={params}>
              {t(m.chooseProvider)}
            </ButtonLink>
            <ButtonLink to="/w/$workspace/providers/adapters" params={params}>
              {t(m.browseAdapters)}
            </ButtonLink>
          </div>
        </Step>
        <Step n={2} title={t(m.stepVerify)}>
          <span className="k">{t(m.stepVerifyLine)}</span>
        </Step>
        <Step
          n={3}
          title={t(m.stepImport)}
          action={
            <ButtonLink size="sm" to="/w/$workspace/import" params={params}>
              {t(m.importNow)}
            </ButtonLink>
          }
        >
          <span className="k">{t(m.stepImportLine)}</span>
        </Step>
      </section>
      <Grid min={300} gap={16}>
        <PendingHealthCard />
        <PendingNextCard />
        <section className="card flex flex-col gap-2.5 border-dashed px-5 py-4.5" aria-labelledby="ov-subs-p">
          <h3 className="h2 text-ink-3" id="ov-subs-p">
            {t(m.subscribers)}
          </h3>
          <span className="n text-[28px] text-[#6B7280]">{count(subscribers)}</span>
          <span className="k">
            {!lists.length
              ? t(m.subscribersNoLists)
              : t(subscribers ? m.subscribersWaiting : m.listsReady, { n: lists.length, lists: lists.join(', ') })}
          </span>
        </section>
      </Grid>
    </>
  )
}
