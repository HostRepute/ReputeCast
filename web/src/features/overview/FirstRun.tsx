import type { ReactNode } from 'react'
import { useInstanceMeta } from '@/components/shell/AuthLayout'
import { ButtonLink } from '@/components/ui/Button'
import { Grid } from '@/components/ui/Layout'
import { cn } from '@/lib/cn'
import { count } from '@/lib/format'
import { useWorkspace } from '@/lib/session'
import type { Schemas } from '@/api/client'
import { PendingNextCard } from './NextCampaignCard'
import { PendingHealthCard } from './SenderHealthCard'

function Step({ n, current, title, children, action }: { n: number; current?: boolean; title: string; children: ReactNode; action?: ReactNode }) {
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
          <span className="vh">Step {n}{current ? ', current' : ''}: </span>
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
  const { slug, me } = useWorkspace()
  const meta = useInstanceMeta()
  const params = { workspace: slug }
  const lists = audience?.lists.map((l) => l.name) ?? []
  return (
    <>
      <section className="card flex flex-col" aria-labelledby="ov-welcome-h">
        <div className="flex flex-col gap-2 px-6 pb-5 pt-[26px]">
          <span className="badge info self-start">Self-hosted · {meta.data?.version ?? 'v1.0'} · all features included</span>
          <h2 id="ov-welcome-h" className="m-0 text-[24px] font-bold tracking-[-0.02em]">
            Welcome to {me.workspace.name}
          </h2>
          <p className="m-0 max-w-[620px] text-ink-2">
            ReputeCast sends through a sending provider you control. Connect one now, and this page fills up with sender health, provider limits and your first campaign's readiness.
          </p>
        </div>
        <Step n={1} current title="Connect a sending provider">
          <span className="text-ink-2">Amazon SES, Postmark, Mailgun, SendGrid, Resend, Brevo, your own Postal server or any SMTP server. About 10 minutes.</span>
          <div className="mt-1.5 flex flex-wrap gap-2.5">
            <ButtonLink variant="primary" to="/w/$workspace/providers/connect" params={params}>
              Choose a provider
            </ButtonLink>
            <ButtonLink to="/w/$workspace/providers/adapters" params={params}>
              Browse community adapters
            </ButtonLink>
          </div>
        </Step>
        <Step n={2} title="Verify your sending domain">
          <span className="k">DKIM, return-path, SPF and DMARC records, verified live. Unlocked after step 1.</span>
        </Step>
        <Step
          n={3}
          title="Import contacts"
          action={
            <ButtonLink size="sm" to="/w/$workspace/import" params={params}>
              Import now
            </ButtonLink>
          }
        >
          <span className="k">From Brevo, Mailchimp, MailerLite, listmonk, Keila or CSV. Unsubscribed and bounced contacts stay suppressed.</span>
        </Step>
      </section>
      <Grid min={300} gap={16}>
        <PendingHealthCard />
        <PendingNextCard />
        <section className="card flex flex-col gap-2.5 border-dashed px-5 py-4.5" aria-labelledby="ov-subs-p">
          <h3 className="h2 text-ink-3" id="ov-subs-p">
            Subscribers
          </h3>
          <span className="n text-[28px] text-[#6B7280]">{count(audience?.total ?? 0)}</span>
          <span className="k">
            {lists.length
              ? `${lists.length} ${lists.length === 1 ? 'list is' : 'lists are'} ready to receive contacts: ${lists.join(', ')}.`
              : 'Create a list to collect contacts, or import them with their lists.'}
          </span>
        </section>
      </Grid>
    </>
  )
}
