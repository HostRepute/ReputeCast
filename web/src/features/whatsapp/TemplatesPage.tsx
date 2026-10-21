import { $api, errorMessage } from '@/api/client'
import { Badge } from '@/components/ui/Badge'
import { Button, ButtonAnchor } from '@/components/ui/Button'
import { ErrorPanel, Loading, Skeleton } from '@/components/ui/Feedback'
import { Table, Td, Th } from '@/components/ui/Table'
import { toast } from '@/components/ui/Toast'
import { useT } from '@/i18n'
import { clock, dayMonthDate } from '@/lib/format'
import { useWorkspace } from '@/lib/session'
import { useWhatsApp, WhatsAppFrame, ReadOnlyNote } from './AccountPage'
import { Body, categoryBadge, MANAGER_URL, statusBadge, throwIfForbidden, useInvalidateWhatsApp, VarLine, type Overview, type Template } from './common'
import { m } from './messages'

/** Sending providers › WhatsApp › Templates (board WhatsAppChannel). */
export function WhatsAppTemplatesPage() {
  const t = useT()
  const q = useWhatsApp()
  throwIfForbidden(q.error)
  const data = q.data
  return (
    <WhatsAppFrame doc={t(m.docTitleTemplates)} data={data}>
      {q.isError ? (
        <ErrorPanel title={t(m.loadFailed)} actions={<Button onClick={() => void q.refetch()}>{t(m.retry)}</Button>}>
          {errorMessage(q.error)} {t(m.loadFailedText)}
        </ErrorPanel>
      ) : !data ? (
        <Loading label={t(m.loading)}>
          <Skeleton w="100%" h={260} />
        </Loading>
      ) : (
        <>
          {data.canManage ? null : <ReadOnlyNote helpers={data.helpers} />}
          {data.templates
            .filter((x) => x.status === 'rejected')
            .map((x) => (
              <Rejected key={x.id} tpl={x} canManage={data.canManage} />
            ))}
          <TemplatesCard data={data} />
        </>
      )}
    </WhatsAppFrame>
  )
}

function SyncButton({ label }: { label?: string }) {
  const t = useT()
  const { slug } = useWorkspace()
  const invalidate = useInvalidateWhatsApp()
  const sync = $api.useMutation('post', '/api/v1/workspaces/{workspace}/whatsapp/templates/sync')
  return (
    <Button
      size="sm"
      icon="refresh"
      loading={sync.isPending}
      busyLabel={t(m.syncing)}
      onClick={() =>
        sync.mutate(
          { params: { path: { workspace: slug } } },
          {
            onSuccess: (r) => {
              void invalidate()
              toast({
                kind: 'success',
                title:
                  r.added.length || r.changed.length || r.removed.length
                    ? t(m.syncedChanges, { added: r.added.length, changed: r.changed.length, removed: r.removed.length })
                    : t(m.synced, { count: r.templates }),
              })
            },
            onError: (e) => toast({ kind: 'error', title: errorMessage(e) }),
          },
        )
      }
    >
      {label ?? t(m.syncNow)}
    </Button>
  )
}

/** The templates table: synced from Meta, created in WhatsApp Manager. */
export function TemplatesCard({ data }: { data: Overview }) {
  const t = useT()
  const synced = data.account?.templatesSyncedAt
  return (
    <section className="card">
      <div className="px-head">
        <div>
          <h2 className="h2">{t(m.templatesTitle, { count: data.templates.length })}</h2>
          <span className="k">{t(m.templatesIntro)}</span>
        </div>
        <div>
          {data.canManage && data.account ? <SyncButton /> : null}
          <ButtonAnchor size="sm" icon="external" href={MANAGER_URL}>
            {t(m.openManager)}
          </ButtonAnchor>
        </div>
      </div>
      <Table aria-label={t(m.templatesTitle, { count: data.templates.length })}>
        <thead>
          <tr>
            <Th>{t(m.colTemplate)}</Th>
            <Th>{t(m.colCategory)}</Th>
            <Th>{t(m.colLanguage)}</Th>
            <Th>{t(m.colStatus)}</Th>
            <Th>{t(m.colBody)}</Th>
            <Th>{t(m.colButtons)}</Th>
          </tr>
        </thead>
        <tbody>
          {data.templates.length ? (
            data.templates.map((x) => <Row key={x.id} tpl={x} />)
          ) : (
            <tr>
              <td colSpan={6} className="py-8 text-center text-ink-3">
                {t(m.noTemplates)}
              </td>
            </tr>
          )}
        </tbody>
      </Table>
      <div className="px-foot">
        <span className="k">{synced ? t(m.templatesFoot, { date: `${dayMonthDate(synced)}, ${clock(synced)}` }) : t(m.templatesFootNever)}</span>
      </div>
    </section>
  )
}

function Row({ tpl: x }: { tpl: Template }) {
  const t = useT()
  const c = categoryBadge[x.category]
  const s = statusBadge[x.status]
  return (
    <tr>
      <Td>
        <span className="mono text-[12.5px] font-semibold">{x.name}</span>
      </Td>
      <Td>
        <Badge tone={c.tone}>{t(c.msg)}</Badge>
      </Td>
      <Td nowrap>{x.languageLabel}</Td>
      <Td>
        <span className="flex flex-col items-start gap-[3px]">
          <Badge tone={s.tone}>{t(s.msg)}</Badge>
          {x.status === 'pending' && x.statusSince ? <span className="k whitespace-normal">{t(m.pendingSince, { date: dayMonthDate(x.statusSince) })}</span> : null}
          {x.status === 'rejected' && x.reason ? <span className="k whitespace-normal text-bad-fg!">{t(m.rejectedReason, { reason: x.reason })}</span> : null}
        </span>
      </Td>
      <Td wrap>
        <span className="flex min-w-[240px] flex-col gap-[3px]">
          <span>
            <Body text={x.body} />
          </span>
          {x.variables.length ? (
            <span className="k">
              <VarLine vars={x.variables} />
            </span>
          ) : null}
        </span>
      </Td>
      <Td>{x.buttons.length ? x.buttons.map((b) => b.text).join(', ') : '–'}</Td>
    </tr>
  )
}

/** A rejected template, with Meta's reason and where to fix it. */
function Rejected({ tpl: x, canManage }: { tpl: Template; canManage: boolean }) {
  const t = useT()
  const c = categoryBadge[x.category]
  const qr = x.buttons.filter((b) => b.type === 'quick_reply')
  const exampleReason = x.reasonCode === 'missing_examples' || /sample|example/i.test(x.reason ?? '')
  return (
    <section className="card flex flex-col gap-3 p-[18px]" style={{ borderColor: 'var(--bad-line)' }} aria-label={x.name}>
      <div className="flex flex-wrap items-center gap-2.5">
        <span className="mono font-semibold">{x.name}</span>
        <Badge tone={c.tone}>{t(c.msg)}</Badge>
        <Badge tone="bad">{t(m.stRejected)}</Badge>
        <span className="k">{x.statusSince ? t(m.rejectedOn, { language: x.languageLabel, date: dayMonthDate(x.statusSince) }) : x.languageLabel}</span>
      </div>
      <div className="flex flex-col gap-1.5 rounded-lg border border-line bg-ground-3 px-3.5 py-2.5">
        <span>
          <Body text={x.body} />
        </span>
        <span className="k">
          <VarLine vars={x.variables} />
          {qr.map((b) => ` · ${t(m.quickReply, { text: b.text })}`)}
        </span>
      </div>
      <div className="note b" role="alert">
        <svg width="16" height="16" viewBox="0 0 16 16" fill="none" stroke="#B91C1C" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
          <circle cx="8" cy="8" r="6.2" />
          <path d="M8 4.8v3.6M8 10.9v.1" />
        </svg>
        <div className="px-note-body">
          {x.reason ? <b>{t(m.rejectedReason, { reason: x.reason })}</b> : null}
          <div>
            {t(exampleReason ? m.rejectedHelp : m.rejectedHelpOther)}
            <div className="px-note-acts">
              <ButtonAnchor size="sm" icon="external" href={MANAGER_URL}>
                {t(m.fixInManager)}
              </ButtonAnchor>
              {canManage ? <SyncButton /> : null}
            </div>
          </div>
        </div>
      </div>
    </section>
  )
}
