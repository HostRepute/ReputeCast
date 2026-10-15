import { useState, type ReactNode } from 'react'
import { $api, errorMessage, http, type Schemas } from '@/api/client'
import { Icon } from '@/components/icons/Icon'
import { Page } from '@/components/shell/Page'
import { Badge } from '@/components/ui/Badge'
import { Button, ButtonAnchor, ButtonLink } from '@/components/ui/Button'
import { KeyValues } from '@/components/ui/Data'
import { Skeleton } from '@/components/ui/Feedback'
import { Menu, MenuItem } from '@/components/ui/Menu'
import { Note } from '@/components/ui/Note'
import { toast } from '@/components/ui/Toast'
import { copyText } from '@/features/access/parts'
import { T, formatList, useT } from '@/i18n'
import { ago, clock, dayMonthDate } from '@/lib/format'
import { useWorkspace } from '@/lib/session'
import { CallLog, statusText } from './CallLog'
import { ConnectDialog, Ev, receiveLines } from './ConnectDialog'
import { ChangeDialog, DisconnectDialog, RotateDialog, StopOldKeyDialog, TakeOverDialog } from './KeyDialogs'
import { m } from './messages'
import {
  base,
  entryLine,
  entryName,
  helpersText,
  itemLine,
  itemName,
  longDay,
  maintainer,
  PermChip,
  PermChips,
  platformUrl,
  reasonWhy,
  stamp,
  StatusBadge,
  tags,
  Tile,
  useApp,
  useRefreshHub,
  type App,
  type PlatformKind,
  type Reason,
} from './shared'

type Hook = Schemas['IntegrationManagedHook']
type Workflow = Schemas['IntegrationWorkflowRow']
type Dialog = 'connect' | 'change' | 'rotate' | 'take-over' | 'disconnect' | 'stop-old' | null

/** "Beta application" from "Beta application (Typeform) → ReputeCast contact", for the narrow number tile. */
function shortName(name: string): string {
  return name.split(/\s[(→]/)[0] ?? name
}

const isToday = (d: string) => dayMonthDate(d) === dayMonthDate(new Date())

/** Settings › Integrations › Zapier (Make and n8n use the same page): IntegrationsZapier, IntegrationsStates. */
export function AppPage({ kind }: { kind: PlatformKind }) {
  const { slug } = useWorkspace()
  const t = useT()
  const q = useApp(kind)
  const name = t(entryName[kind])
  const [dialog, setDialog] = useState<Dialog>(null)
  const app = q.data
  const close = (o: boolean) => !o && setDialog(null)
  return (
    <Page
      title={name}
      documentTitle={`${name} · ${t(m.title)}`}
      breadcrumb={[
        { label: t(m.settings), to: '/w/$workspace/settings', params: { workspace: slug } },
        { label: t(m.title), to: '/w/$workspace/settings/integrations', params: { workspace: slug } },
        { label: name },
      ]}
    >
      {q.isError ? (
        <AppError kind={kind} detail={errorMessage(q.error)} onRetry={() => void q.refetch()} />
      ) : !app ? (
        <AppSkeleton kind={kind} />
      ) : (
        <>
          <Header app={app} onDialog={setDialog} />
          {!app.canManage ? (
            <div className="note n" role="status">
              <Badge tone="neutral">{t(m.readOnly)}</Badge>
              <span>{app.helpers.length ? t(m.readOnlyApp, { name, helpers: helpersText(app.helpers) }) : t(m.readOnlyAppAlone, { name })}</span>
            </div>
          ) : null}
          {app.status === 'not_connected' ? (
            <NotConnected app={app} onConnect={() => setDialog('connect')} />
          ) : (
            <Connected app={app} onDialog={setDialog} />
          )}
          {dialog === 'connect' || dialog === null ? (
            <ConnectDialog kind={kind} open={dialog === 'connect'} onOpenChange={close} onDone={() => setDialog(null)} />
          ) : null}
          {app.connectionId ? (
            <>
              {dialog === 'change' ? <ChangeDialog app={app} open onOpenChange={close} /> : null}
              {dialog === 'rotate' ? <RotateDialog connectionId={app.connectionId} kind={kind} name={name} oldMasked={app.key?.masked ?? ''} open onOpenChange={close} /> : null}
              {dialog === 'take-over' ? <TakeOverDialog connectionId={app.connectionId} name={name} open onOpenChange={close} /> : null}
              {dialog === 'disconnect' ? <DisconnectDialog connectionId={app.connectionId} kind={kind} name={name} open onOpenChange={close} /> : null}
              {dialog === 'stop-old' && app.oldKey ? <StopOldKeyDialog connectionId={app.connectionId} masked={app.oldKey.masked} open onOpenChange={close} /> : null}
            </>
          ) : null}
        </>
      )}
    </Page>
  )
}

/** Logo, name, status, who connected it, and the actions (IntegrationsZapier's `.dhead`). */
function Header({ app, onDialog }: { app: App; onDialog: (d: Dialog) => void }) {
  const t = useT()
  const name = t(entryName[app.kind])
  const who = t(maintainer[app.kind])
  const by = app.connectedBy?.name ?? ''
  const host = app.address ? app.address.replace(/^https?:\/\//, '') : ''
  const sub =
    app.status === 'not_connected' || !app.connectedAt
      ? t(m.subNotConnected, { kind: app.kind, maintainer: who, name })
      : isToday(app.connectedAt)
        ? t(m.subConnectedToday, { maintainer: who, name: by, time: clock(app.connectedAt) })
        : host
          ? t(m.subConnectedHost, { maintainer: who, name: by, date: longDay(app.connectedAt), host })
          : t(m.subConnected, { maintainer: who, name: by, date: longDay(app.connectedAt) })
  const url = platformUrl(app.kind, app.address)
  const stopped = app.reasons.some((r) => r.code === 'key_revoked' || r.code === 'key_expired')
  return (
    <section className="card ih-dhead" aria-label={name}>
      <div className="ih-dmain">
        <Tile k={app.kind} size="lg" />
        <div className="ih-dtext">
          <div className="row gap-2.5">
            <h2 className="ih-ttl">{name}</h2>
            <StatusBadge status={app.status} />
          </div>
          <span className="text-[13px] text-ink-2">{sub}</span>
          <p>{t(entryLine[app.kind])}</p>
        </div>
      </div>
      <div className="ih-dact">
        {app.status === 'not_connected' ? (
          app.canManage ? (
            <Button variant="primary" onClick={() => onDialog('connect')}>
              {t(m.connectPlatform, { name })}
            </Button>
          ) : null
        ) : (
          <>
            {url ? (
              <ButtonAnchor href={url} iconRight="external">
                {t(m.openPlatform, { name })}
                <span className="vh">{t(m.newTab)}</span>
              </ButtonAnchor>
            ) : null}
            {app.canManage ? (
              <>
                {stopped ? (
                  <Button variant="primary" onClick={() => onDialog('connect')}>
                    {t(m.reconnect)}
                  </Button>
                ) : (
                  <>
                    <Button onClick={() => onDialog('change')}>{t(m.changeWhat, { name })}</Button>
                    <Button onClick={() => onDialog('rotate')}>{t(m.rotateKey)}</Button>
                  </>
                )}
                <Menu
                  label={t(m.moreActions, { name })}
                  trigger={
                    <button type="button" className="ibtn" aria-label={t(m.moreActions, { name })}>
                      <Icon name="more" size={16} />
                    </button>
                  }
                >
                  <MenuItem onSelect={() => onDialog('take-over')}>{t(m.takeOver)}</MenuItem>
                  <MenuItem className="text-bad-fg" onSelect={() => onDialog('disconnect')}>
                    {t(m.disconnect)}
                  </MenuItem>
                </Menu>
              </>
            ) : null}
          </>
        )}
      </div>
    </section>
  )
}

/** Not connected: what it can do, connecting in three steps, and Connect (IntegrationsStates). */
function NotConnected({ app, onConnect }: { app: App; onConnect: () => void }) {
  const t = useT()
  const name = t(entryName[app.kind])
  const r = app.reachability
  const needsPublic = app.kind !== 'n8n'
  return (
    <div className="ih-two">
      <CanDo app={app} />
      <section className="card" aria-labelledby="h-steps">
        <div className="sec">
          <h2 className="h2" id="h-steps">
            {t(m.stepsTitle)}
          </h2>
        </div>
        <div className="pad flex flex-col gap-3.5">
          <ol className="ih-ol">
            <li>
              <span>
                <span className="font-semibold">{t(m.stepChoose, { name })}</span>
                <br />
                <span className="k">{t(m.stepChooseBody, { name })}</span>
              </span>
            </li>
            <li>
              <span>
                <span className="font-semibold">{t(m.stepPaste, { name })}</span>
                <br />
                <span className="k">{t(m.stepPasteBody, { name })}</span>
              </span>
            </li>
            <li>
              <span>
                <span className="font-semibold">{t(m.stepWait, { name })}</span>
                <br />
                <span className="k">{t(m.stepWaitBody, { name })}</span>
              </span>
            </li>
          </ol>
          {needsPublic ? (
            r.reachable ? (
              <Note tone="info">{r.checkedAt ? t(m.reachableNote, { name, url: r.url, ago: ago(r.checkedAt) }) : t(m.reachableUnchecked, { name, url: r.url })}</Note>
            ) : (
              <Note tone="bad">{t(m.unreachableBody, { name, url: r.url })}</Note>
            )
          ) : (
            <Note tone="info">{t(m.n8nNetworkNote, { url: r.url })}</Note>
          )}
          <div className="row gap-2.5">
            {app.canManage ? (
              <Button variant="primary" onClick={onConnect}>
                {t(m.connectPlatform, { name })}
              </Button>
            ) : null}
            <span className="k">{t(m.freeLine, { kind: app.kind, name })}</span>
          </div>
        </div>
      </section>
    </div>
  )
}

function Connected({ app, onDialog }: { app: App; onDialog: (d: Dialog) => void }) {
  const t = useT()
  const name = t(entryName[app.kind])
  const waiting = app.status === 'waiting'
  const failing = app.reasons.find((r) => r.code === 'deliveries_failing')
  const keyProblems = app.reasons.filter((r) => r.code !== 'deliveries_failing')
  return (
    <>
      {keyProblems.map((r, i) => (
        <ProblemCard key={i} app={app} r={r} onDialog={onDialog} />
      ))}
      {failing && failing.retrying && failing.nextAt ? (
        <Note tone="bad" role="alert">
          <span className="font-semibold">{t(m.failingNoteTitle)}</span>
          <div>
            {t(m.failingNoteBody, {
              time: failing.since ? stamp(failing.since) : '',
              host: failing.host ?? '',
              status: failing.status ?? '',
              statusText: failing.statusText ?? '',
              retrying: failing.retrying,
              next: stamp(failing.nextAt),
              name,
            })}
          </div>
        </Note>
      ) : null}
      <Stats app={app} />
      {waiting ? (
        <div className="ih-two">
          <WaitingCard app={app} />
          <ConnectionCard app={app} onDialog={onDialog} />
        </div>
      ) : (
        <>
          {app.webhooks.length ? <HooksCard app={app} /> : null}
          <WorkflowsCard app={app} />
          {app.connectionId ? <CallLog connectionId={app.connectionId} kind={app.kind} errorsFirst={app.numbers.errors7d > 0} /> : null}
          <div className="ih-two">
            <CanDo app={app} onChange={app.canManage ? () => onDialog('change') : undefined} />
            <div className="ih-col">
              <ConnectionCard app={app} onDialog={onDialog} />
              {app.webhooks.length ? null : <HooksCard app={app} />}
              <ReceivesCard app={app} />
            </div>
          </div>
        </>
      )}
      {app.canManage ? (
        <section className="card ih-dz" aria-label={t(m.disconnectTitle, { name })}>
          <div className="flex min-w-0 flex-[1_1_420px] flex-col gap-0.5">
            <h2 className="h2">{t(m.disconnectTitle, { name })}</h2>
            <span className="k text-ink-2">{t(m.disconnectLine, { kind: app.kind, count: app.workflows.length })}</span>
          </div>
          <Button variant="danger" onClick={() => onDialog('disconnect')}>
            {t(m.disconnectButton, { name })}
          </Button>
        </section>
      ) : null}
    </>
  )
}

/** A stopped or soon-stopping key: what happened, the calls rejected since, and the way out (IntegrationsStates). */
function ProblemCard({ app, r, onDialog }: { app: App; r: Reason; onDialog: (d: Dialog) => void }) {
  const t = useT()
  const name = t(entryName[app.kind])
  const time = r.since ? stamp(r.since) : ''
  let title = reasonWhy(t, r)
  let body = ''
  let foot = ''
  let action: ReactNode = null
  let tone: 'bad' | 'warn' = 'bad'
  switch (r.code) {
    case 'key_revoked':
      title = t(m.revokedTitle)
      body = r.by ? t(m.revokedBy, { by: r.by, time, rejected: r.rejected ?? 0, name }) : t(m.revokedAnon, { time, rejected: r.rejected ?? 0, name })
      foot = t(m.rejectedFoot, { kind: app.kind })
      action = app.canManage ? (
        <Button size="sm" variant="primary" onClick={() => onDialog('connect')}>
          {t(m.reconnect)}
        </Button>
      ) : null
      break
    case 'key_expired':
      title = t(m.expiredTitle)
      body = t(m.expiredBody, { time, rejected: r.rejected ?? 0, name })
      foot = t(m.expiredFoot, { name })
      action = app.canManage ? (
        <Button size="sm" variant="primary" onClick={() => onDialog('connect')}>
          {t(m.reconnect)}
        </Button>
      ) : null
      break
    case 'owner_left':
      tone = 'warn'
      title = t(m.ownerLeftTitle)
      body = t(m.ownerLeftBody, { by: r.by ?? '', time, name })
      foot = t(m.ownerLeftFoot)
      action = app.canManage ? (
        <Button size="sm" variant="primary" onClick={() => onDialog('take-over')}>
          {t(m.takeOver)}
        </Button>
      ) : null
      break
    case 'key_expiring':
      tone = 'warn'
      title = t(m.expiringTitle)
      body = t(m.expiringBody, { date: r.since ? dayMonthDate(r.since) : '', name })
      break
    case 'calls_failing':
      tone = 'warn'
      title = t(m.callsFailingTitle)
      body = t(m.callsFailingBody, { failed: r.failed ?? 0, calls: r.calls ?? 0, since: time, name })
      break
  }
  const rejected = r.code === 'key_revoked' || r.code === 'key_expired' || r.code === 'owner_left' ? app.rejected : []
  return (
    <section className="card" aria-label={title}>
      <div className="sec">
        <div className="row gap-3">
          <Tile k={app.kind} size="sm" />
          <div className="flex min-w-0 flex-col gap-0.5">
            <span className="row gap-2">
              <span className="font-semibold">{name}</span>
              <StatusBadge status="attention" />
            </span>
            <span className="k">{title}</span>
          </div>
        </div>
        {action}
      </div>
      <div className="pad flex flex-col gap-3">
        <Note tone={tone} role={tone === 'bad' ? 'alert' : 'status'}>
          {body}
        </Note>
        {rejected.length ? (
          <div className="flex flex-col">
            {rejected.map((c, i) => (
              <div key={i} className="row flex-nowrap gap-3 border-t border-line-2 py-2 first:border-t-0">
                <span className="mono k">{stamp(c.at)}</span>
                <span className="min-w-0 flex-1 truncate">{c.workflow || t(m.unnamed, { kind: app.kind })}</span>
                <Badge tone="bad">
                  {c.status} {c.code}
                </Badge>
              </div>
            ))}
          </div>
        ) : null}
        {foot ? <span className="k">{foot}</span> : null}
      </div>
    </section>
  )
}

/** The four numbers: calls, errors, last call, managed webhooks. */
function Stats({ app }: { app: App }) {
  const t = useT()
  const n = app.numbers
  const waiting = app.status === 'waiting'
  const errorParts = [n.fixed7d ? t(m.statFixed, { count: n.fixed7d, kind: app.kind }) : '', n.consentKept7d ? t(m.statConsentKept, { count: n.consentKept7d }) : ''].filter(Boolean)
  const delivered = app.webhooks.reduce((s, h) => s + h.delivered24h, 0)
  return (
    <div className="ih-stats">
      <Stat label={t(m.statCalls)} value={String(n.calls7d)} sub={waiting || !n.calls7d ? t(m.statNoneYet) : t(m.statCallsFrom, { kind: app.kind, count: app.workflows.length })} />
      <Stat
        label={t(m.statErrors)}
        value={String(n.errors7d)}
        sub={waiting || !n.calls7d ? t(m.statNoneYet) : errorParts.length ? errorParts.join(' · ') : t(m.statNone)}
      />
      <Stat
        label={t(m.statLastCall)}
        value={n.lastCallAt ? stamp(n.lastCallAt) : '—'}
        sub={
          n.lastCallAt
            ? n.lastCallWorkflow
              ? t(m.statLastCallFrom, { ago: ago(n.lastCallAt), workflow: shortName(n.lastCallWorkflow) })
              : ago(n.lastCallAt)
            : app.key
              ? t(m.statWaitingSince, { time: stamp(app.key.createdAt) })
              : ''
        }
      />
      <Stat
        label={t(m.statHooks)}
        value={String(n.managedWebhooks)}
        sub={n.managedWebhooks ? t(m.statHooksDelivered, { count: delivered }) : waiting ? t(m.statHooksNone, { kind: app.kind }) : t(m.statHooksActionsOnly, { kind: app.kind })}
      />
    </div>
  )
}

function Stat({ label, value, sub }: { label: string; value: string; sub: string }) {
  return (
    <div className="card ih-stat">
      <span className="k">{label}</span>
      <span className="ih-v">{value}</span>
      <span className="ih-s">{sub}</span>
    </div>
  )
}

/** What a workflow uses, in words: "Create or update a contact" over "Beta Program · tag beta-applicant · double opt-in". */
function UseText({ use, kind, inline }: { use: Workflow['uses'][number]; kind: PlatformKind; inline?: boolean }) {
  const t = useT()
  if (use.event && use.item === 'record_event') {
    const values = { ...tags, event: use.event }
    if (use.starts?.length) return <T msg={m.useEventStarts} values={{ ...values, flows: formatList(use.starts), count: use.starts.length }} />
    if (use.goalOf?.length) return <T msg={m.useEventGoal} values={{ ...values, flows: formatList(use.goalOf) }} />
    return <T msg={m.useEvent} values={values} />
  }
  const name = itemName(t, use.item)
  const trigger = use.item.includes('.')
  if (trigger) return <>{t(m.useTrigger, { name })}</>
  const parts = [
    ...(use.lists ?? []),
    use.tags?.length ? (use.item === 'add_tag' ? use.tags.join(', ') : t(m.useTags, { count: use.tags.length, tags: use.tags.join(', ') })) : '',
    use.optIn ? t(m.useOptIn, { optIn: use.optIn }) : '',
    use.template ?? '',
  ].filter(Boolean)
  void kind
  if (inline) return <>{parts.length ? `${name} · ${parts.join(' · ')}` : name}</>
  return (
    <div className="col gap-px">
      <span>{name}</span>
      {parts.length ? <span className="k">{parts.join(' · ')}</span> : null}
    </div>
  )
}

/** Zaps (scenarios, workflows) using ReputeCast: seen in the last 30 days, with calls, errors and last run. */
function WorkflowsCard({ app }: { app: App }) {
  const t = useT()
  const kind = app.kind
  const rows = app.workflows
  const countCell = (w: Workflow) => (w.trigger && !w.calls7d ? t(m.deliveriesCount, { count: w.deliveries7d }) : t(m.callsCount, { count: w.calls7d }))
  const errorsCell = (w: Workflow) =>
    w.errors7d ? (
      <>
        <span className="num font-semibold">{w.errors7d}</span>
        <div className="k whitespace-nowrap">{w.consentKept7d === w.errors7d ? t(m.consentKept) : w.fixedAt ? t(m.fixedAt, { time: clock(w.fixedAt) }) : null}</div>
      </>
    ) : (
      <span className="k">0</span>
    )
  return (
    <section className="card" aria-labelledby="h-workflows">
      <div className="sec">
        <div className="flex min-w-0 flex-col gap-0.5">
          <h2 className="h2" id="h-workflows">
            {t(m.workflowsTitle, { kind, count: rows.length })}
          </h2>
          <span className="k">{t(m.workflowsSub, { kind })}</span>
        </div>
      </div>
      {rows.length === 0 ? (
        <div className="pad k">{t(m.workflowsEmpty, { kind })}</div>
      ) : (
        <>
          <div className="ih-no-phone">
            <div className="tscroll">
              <table className="tbl" aria-label={t(m.workflowsTitle, { kind, count: rows.length })}>
                <thead>
                  <tr>
                    <th scope="col">{t(m.colWorkflow, { kind })}</th>
                    <th scope="col">{t(m.colUses)}</th>
                    <th scope="col" className="r">
                      {t(m.colCalls7d)}
                    </th>
                    <th scope="col" className="r">
                      {t(m.colErrors)}
                    </th>
                    <th scope="col">{t(m.colLastRun)}</th>
                  </tr>
                </thead>
                <tbody>
                  {rows.map((w) => (
                    <tr key={w.key}>
                      <td className="!whitespace-normal">
                        <span className="font-semibold">{w.unnamed ? t(m.unnamed, { kind }) : w.name}</span>
                      </td>
                      <td className="!whitespace-normal">
                        {w.uses.map((u, i) => (
                          <UseText key={i} use={u} kind={kind} />
                        ))}
                      </td>
                      <td className="r">
                        <span className="num">{w.trigger && !w.calls7d ? w.deliveries7d : w.calls7d}</span>
                      </td>
                      <td className="r">{errorsCell(w)}</td>
                      <td className="nw mono">{stamp(w.lastRunAt)}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          </div>
          <div className="ih-only-phone ih-list">
            {rows.map((w) => (
              <div key={w.key} className="ih-it">
                <span className="font-semibold">{w.unnamed ? t(m.unnamed, { kind }) : w.name}</span>
                <span className="k text-ink-nav">
                  {w.uses.map((u, i) => (
                    <UseText key={i} use={u} kind={kind} inline />
                  ))}
                </span>
                <span className="k">
                  {w.trigger && !w.calls7d ? countCell(w) : t(m.workflowLine, { calls: w.calls7d, errors: w.errors7d, time: stamp(w.lastRunAt) })}
                </span>
              </div>
            ))}
          </div>
          <div className="ih-foot">
            <span className="k">{t(m.workflowsFoot, { kind })}</span>
          </div>
        </>
      )}
    </section>
  )
}

/** What it can do: every trigger and action with its permission, Available or Off (and why). */
function CanDo({ app, onChange }: { app: App; onChange?: () => void }) {
  const t = useT()
  const name = t(entryName[app.kind])
  const connected = app.status !== 'not_connected'
  const available = app.items.filter((i) => i.available).length
  const triggers = app.items.filter((i) => i.kind === 'trigger')
  const actions = app.items.filter((i) => i.kind === 'action')
  const state = (i: App['items'][number]) => {
    if (!connected) return <span className="k">{i.sensitive ? t(m.offByDefault) : t(m.onByDefault)}</span>
    if (i.available) return <Badge tone="ok">{t(m.available)}</Badge>
    return (
      <>
        <Badge tone="neutral">{t(m.off)}</Badge>
        <span className="k">
          {i.offReason === 'owner_lacks' ? t(m.offOwnerLacks, { perm: i.missing ?? '', owner: i.owner ?? '' }) : t(m.offTurnedOff)}
        </span>
      </>
    )
  }
  const row = (i: App['items'][number]) => (
    <div key={i.key} className="ih-prow">
      <div className="ih-pl">
        <span className="ih-pn">
          {itemName(t, i.key)}
          {i.new ? <Badge tone="info">{t(m.newBadge)}</Badge> : null}
        </span>
        <span className={`ih-pd${i.kind === 'trigger' ? ' mono' : ''}`}>{i.kind === 'trigger' ? i.event : itemLine(t, i.key)}</span>
      </div>
      <div className="ih-pp">
        {i.permissions.map((p) => (
          <PermChip key={p} perm={p} />
        ))}
        {i.sensitive ? (
          <span className="flag" title={t(m.sensitivePermission)}>
            <span aria-hidden="true">⚑</span>
            {t(m.sensitive)}
          </span>
        ) : null}
      </div>
      <div className="ih-st">{state(i)}</div>
    </div>
  )
  return (
    <section className="card" aria-labelledby="h-cando">
      <div className="sec">
        <div className="flex min-w-0 flex-col gap-0.5">
          <h2 className="h2" id="h-cando">
            {t(m.canDoTitle, { name })}
          </h2>
          <span className="k">
            {connected ? t(m.canDoSub, { available, total: app.items.length, by: app.connectedBy?.name ?? '' }) : t(m.canDoSubNotConnected)}
          </span>
        </div>
      </div>
      <div className="pad ih-prl !pt-1">
        <div className="ih-phd">
          {t(m.triggers)} <span className="k">{t(m.triggersHint, { name })}</span>
        </div>
        {triggers.map(row)}
        <div className="ih-phd">
          {t(m.actions)} <span className="k">{t(m.actionsHint)}</span>
        </div>
        {actions.map(row)}
      </div>
      {connected ? (
        <div className="ih-foot">
          <span className="k">{t(m.canDoFoot)}</span>
          {onChange ? (
            <Button size="sm" onClick={onChange}>
              {t(m.changeWhat, { name })}
            </Button>
          ) : null}
        </div>
      ) : null}
    </section>
  )
}

/** The connection: instance URL, the key, its permissions and list scope. */
function ConnectionCard({ app, onDialog }: { app: App; onDialog: (d: Dialog) => void }) {
  const { slug } = useWorkspace()
  const t = useT()
  const name = t(entryName[app.kind])
  const k = app.key
  if (!k) return null
  const created = isToday(k.createdAt)
    ? t(m.createdTodayBy, { time: clock(k.createdAt), name: k.owner.name })
    : t(m.createdBy, { date: longDay(k.createdAt), name: k.owner.name })
  const used = k.lastUsedAt ? [stamp(k.lastUsedAt), k.lastUsedIp, k.lastUsedClient].filter(Boolean).join(' · ') : t(m.never)
  return (
    <section className="card" aria-labelledby="h-connection">
      <div className="sec">
        <h2 className="h2" id="h-connection">
          {t(m.connectionTitle)}
        </h2>
      </div>
      <div className="pad flex flex-col gap-3">
        <KeyValues
          items={[
            [
              t(m.instanceUrl),
              <div className="ih-inl">
                <span className="mono">{app.instanceUrl}</span>
                <button
                  type="button"
                  className="ibtn"
                  aria-label={t(m.copyInstanceUrl)}
                  onClick={async () => {
                    const ok = await copyText(app.instanceUrl)
                    toast({ kind: ok ? 'success' : 'error', title: t(m.copiedToast) })
                  }}
                >
                  <Icon name="copy" size={14} />
                </button>
              </div>,
            ],
            [
              t(m.key),
              <>
                <span className="font-semibold">{k.name}</span> <span className="mono k">{k.masked}</span>
              </>,
            ],
            [t(m.created), created],
            [t(m.lastUsed), used],
            [t(m.expires), k.expiresAt ? longDay(k.expiresAt) : t(m.never)],
            [t(m.permissions), <PermChips perms={k.permissions} sensitive={k.sensitive} />],
            [t(m.listScope), k.scope.all || !k.scope.label ? t(m.allLists) : k.scope.label],
          ]}
        />
        <Note tone="neutral" className="font-normal [font-variant-numeric:normal]">
          {isToday(k.createdAt) ? t(m.shownOnceAt, { time: clock(k.createdAt) }) : t(m.shownOnceOn, { date: dayMonthDate(k.createdAt) })}
        </Note>
        {app.oldKey?.graceUntil ? (
          <Note tone="info">
            <div>{t(m.oldKeyWorks, { masked: app.oldKey.masked, time: stamp(app.oldKey.graceUntil), name })}</div>
            {app.canManage ? (
              <div className="mt-2">
                <Button size="sm" variant="danger" onClick={() => onDialog('stop-old')}>
                  {t(m.stopOldKey)}
                </Button>
              </div>
            ) : null}
          </Note>
        ) : null}
      </div>
      <div className="ih-foot">
        <span className="k">{t(m.managedInKeys, { name })}</span>
        <ButtonLink size="sm" variant="ghost" className="ih-lnk" to="/w/$workspace/settings/keys" params={{ workspace: slug }}>
          {t(m.keysAndMcp)}
        </ButtonLink>
      </div>
    </section>
  )
}

/** Waiting for the first call, right after connecting (IntegrationsStates). */
function WaitingCard({ app }: { app: App }) {
  const t = useT()
  const name = t(entryName[app.kind])
  const k = app.key
  const on = app.items.filter((i) => i.on).length
  return (
    <section className="card" aria-labelledby="h-waiting">
      <div className="sec">
        <h2 className="h2" id="h-waiting">
          {t(m.waitingTitle, { name })}
        </h2>
        <Badge tone="info" dot>
          {t(m.listening)}
        </Badge>
      </div>
      <div className="pad flex flex-col gap-2">
        <div className="flex flex-col" aria-live="polite">
          <Ev
            icon="ok"
            title={t(m.evKeyCreated)}
            sub={k ? t(m.evKeyCreatedSub, { key: k.name, count: on, name: k.owner.name }) : undefined}
            time={k ? clock(k.createdAt) : undefined}
          />
          <Ev icon="live" title={t(m.evWaiting, { name })} sub={t(m.evWaitingUsually, { name })} />
          <Ev icon="none" title={t(m.evConnected)} sub={t(m.evConnectedPage)} />
        </div>
        <span className="k">{t(m.waitingFoot, { name })}</span>
      </div>
    </section>
  )
}

/** "What Zapier receives": the data of the triggers in use. */
function ReceivesCard({ app }: { app: App }) {
  const t = useT()
  const name = t(entryName[app.kind])
  const lines = receiveLines(t, app.receives)
  return (
    <section className="card" aria-labelledby="h-receives">
      <div className="sec">
        <h2 className="h2" id="h-receives">
          {t(m.receivesTitle, { name })}
        </h2>
      </div>
      <div className="pad flex flex-col gap-1">
        {lines.length ? (
          <>
            <ul className="m-0 flex list-disc flex-col gap-1 pl-[18px] text-[13px] text-ink-nav">
              {lines.map((l, i) => (
                <li key={i}>{l}</li>
              ))}
            </ul>
            <span className="k">{t(m.receivesFoot, { kind: app.kind })}</span>
          </>
        ) : (
          <>
            <span className="font-semibold">{t(m.receivesNothing, { kind: app.kind })}</span>
            <span className="k text-ink-2">{t(m.receivesNothingSub, { kind: app.kind })}</span>
          </>
        )}
      </div>
    </section>
  )
}

/** Webhooks the integration manages: event, endpoint (masked), created, deliveries, status; failing and dead-letter details. */
function HooksCard({ app }: { app: App }) {
  const { slug } = useWorkspace()
  const t = useT()
  const name = t(entryName[app.kind])
  const hooks = app.webhooks
  const one = hooks.length === 1 ? hooks[0] : undefined
  const sub = one?.workflow ? t(m.hooksSubOne, { kind: app.kind, workflow: one.workflow, name }) : hooks.length ? t(m.hooksSubMany, { name }) : ''
  return (
    <section className="card" aria-labelledby="h-hooks">
      <div className="sec">
        <div className="flex min-w-0 flex-col gap-0.5">
          <h2 className="h2" id="h-hooks">
            {t(m.hooksTitle, { name, count: hooks.length })}
          </h2>
          {sub ? <span className="k">{sub}</span> : null}
        </div>
      </div>
      {hooks.length === 0 ? (
        <div className="pad row flex-nowrap items-start gap-3">
          <Icon name="webhook" size={18} className="mt-0.5 flex-none text-ink-3" />
          <span className="text-ink-nav">{t(m.hooksEmpty, { kind: app.kind })}</span>
        </div>
      ) : (
        <>
          <div className="ih-no-phone">
            <div className="tscroll">
              <table className="tbl" aria-label={t(m.hooksTitle, { name, count: hooks.length })}>
                <thead>
                  <tr>
                    <th scope="col">{t(m.colEvent)}</th>
                    <th scope="col">{t(m.colEndpoint)}</th>
                    <th scope="col">{t(m.colCreated)}</th>
                    <th scope="col">{t(m.colDeliveries24h)}</th>
                    <th scope="col">{t(m.colStatus)}</th>
                    <th scope="col">
                      <span className="vh">{t(m.deliveryLog)}</span>
                    </th>
                  </tr>
                </thead>
                <tbody>
                  {hooks.map((h) => (
                    <tr key={h.id}>
                      <td>
                        <span className="ih-mk">{h.event}</span>
                      </td>
                      <td>
                        <span className="mono">{h.endpoint}</span>
                      </td>
                      <td className="nw">{dayMonthDate(h.createdAt)}</td>
                      <td>
                        <span className="num">
                          {t(m.hookDelivered, { count: h.delivered24h })}
                          {h.retrying ? (
                            <>
                              {' · '}
                              <span className="font-semibold text-bad-fg">{t(m.hookRetrying, { count: h.retrying })}</span>
                            </>
                          ) : null}
                        </span>
                      </td>
                      <td>
                        <HookStatus h={h} />
                      </td>
                      <td className="r">
                        <ButtonLink size="sm" to="/w/$workspace/settings/webhooks" params={{ workspace: slug }}>
                          {t(m.deliveryLog)}
                        </ButtonLink>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          </div>
          <div className="ih-only-phone ih-list">
            {hooks.map((h) => (
              <div key={h.id} className="ih-it">
                <div className="row justify-between gap-2">
                  <span className="ih-mk">{h.event}</span>
                  <HookStatus h={h} />
                </div>
                <span className="mono">{h.endpoint}</span>
                <span className="k">
                  {h.retrying
                    ? t(m.hookCreatedLineRetrying, { date: dayMonthDate(h.createdAt), delivered: h.delivered24h, retrying: h.retrying })
                    : t(m.hookCreatedLine, { date: dayMonthDate(h.createdAt), delivered: h.delivered24h })}
                </span>
              </div>
            ))}
          </div>
          {hooks
            .filter((h) => h.status === 'failing' || h.status === 'dead_letters')
            .map((h) => (h.status === 'failing' ? <FailingDetail key={h.id} h={h} /> : <DeadLetters key={h.id} app={app} h={h} />))}
        </>
      )}
    </section>
  )
}

function HookStatus({ h }: { h: Hook }) {
  const t = useT()
  switch (h.status) {
    case 'failing':
      return (
        <Badge tone="bad" dot>
          {t(m.hookFailing, { status: h.lastStatus ?? '', time: h.failingSince ? stamp(h.failingSince) : '' })}
        </Badge>
      )
    case 'dead_letters':
      return (
        <Badge tone="warn" dot>
          {t(m.hookDead, { count: h.dead })}
        </Badge>
      )
    case 'new':
      return <Badge tone="neutral">{t(m.hookNew)}</Badge>
  }
  return (
    <Badge tone="ok" dot>
      {t(m.hookHealthy)}
    </Badge>
  )
}

/** Since when, the retries and the next try (IntegrationsStates, "Webhook deliveries failing"). */
function FailingDetail({ h }: { h: Hook }) {
  const t = useT()
  const host = h.endpoint.replace(/^https?:\/\//, '').split('/')[0] ?? ''
  const a = h.lastAttempt
  return (
    <div className="pad flex flex-col gap-3 border-t border-line-2">
      <KeyValues
        items={[
          [t(m.failingSinceLabel), t(m.failingSince, { time: h.failingSince ? stamp(h.failingSince) : '', host, status: h.lastStatus ?? '', statusText: h.statusText ?? statusText(h.lastStatus ?? 0) })],
          [t(m.retryingLabel), t(m.retryingValue, { count: h.retrying, time: h.nextAt ? stamp(h.nextAt) : '' })],
          [t(m.thenLabel), t(m.thenValue, { max: a?.maxAttempts ?? 8 })],
        ]}
      />
      {a ? (
        <div className="flex flex-col">
          <Ev
            icon="bad"
            title={t(m.attemptLine, { n: a.attempt, max: a.maxAttempts, status: a.status ?? '', statusText: a.statusText ?? '' })}
            sub={t(m.attemptSub, { event: a.event, count: a.queued })}
            time={a.at ? clock(a.at) : undefined}
          />
          {h.nextAt ? <Ev icon="none" title={t(m.nextTry)} sub={t(m.nextTrySub, { count: h.retrying })} time={clock(h.nextAt)} /> : null}
        </div>
      ) : null}
    </div>
  )
}

/** Deliveries that gave up: replay one, or all once the tool is back (IntegrationsStates, "dead-letter queue"). */
function DeadLetters({ app, h }: { app: App; h: Hook }) {
  const { slug } = useWorkspace()
  const t = useT()
  const name = t(entryName[app.kind])
  const q = $api.useQuery('get', `${base}/connections/{id}/webhooks/{webhookId}/dead-letters`, {
    params: { path: { workspace: slug, id: app.connectionId ?? '', webhookId: h.id } },
  })
  const refresh = useRefreshHub()
  const [busy, setBusy] = useState<string | null>(null)
  const host = h.endpoint.replace(/^https?:\/\//, '').split('/')[0] ?? ''
  const items = q.data?.items ?? []

  async function replay(deliveryId?: string) {
    setBusy(deliveryId ?? 'all')
    try {
      const res = deliveryId
        ? await http.POST('/api/v1/workspaces/{workspace}/webhooks/deliveries/{deliveryId}/retry', { params: { path: { workspace: slug, deliveryId } } })
        : await http.POST('/api/v1/workspaces/{workspace}/webhooks/retry-failed', { params: { path: { workspace: slug } }, body: { webhookId: h.id } })
      await refresh()
      void q.refetch()
      toast({ title: t(m.replayed, { count: res.data?.queued ?? 0 }) })
    } catch (e) {
      toast({ kind: 'error', title: t(m.replayError), detail: errorMessage(e) })
    } finally {
      setBusy(null)
    }
  }

  const max = h.lastAttempt?.maxAttempts ?? 8
  return (
    <div className="pad flex flex-col gap-3 border-t border-line-2">
      <Note tone={h.recoveredAt ? 'ok' : 'warn'}>
        <span className="font-semibold">{h.recoveredAt ? t(m.backTitle, { name }) : t(m.deadTitle)}</span>
        <div>
          {h.recoveredAt
            ? t(m.backBody, { host, time: stamp(h.recoveredAt), status: h.lastStatus ?? 200, count: h.dead, max })
            : t(m.deadBody, { count: h.dead, max })}
          {app.canManage ? (
            <div className="row mt-2 gap-2">
              <Button size="sm" variant="primary" onClick={() => replay()} loading={busy === 'all'}>
                {t(m.replayAll, { count: h.dead })}
              </Button>
              <ButtonLink size="sm" to="/w/$workspace/settings/webhooks" params={{ workspace: slug }}>
                {t(m.deliveryLog)}
              </ButtonLink>
            </div>
          ) : null}
        </div>
      </Note>
      {items.length ? (
        <>
          <div className="ih-no-phone">
            <div className="tscroll">
              <table className="tbl" aria-label={t(m.deadTitle)}>
                <thead>
                  <tr>
                    <th scope="col">{t(m.colFirstTry)}</th>
                    <th scope="col">{t(m.colEvent)}</th>
                    <th scope="col">{t(m.colContact)}</th>
                    <th scope="col">{t(m.colLastAnswer)}</th>
                    <th scope="col">{t(m.colKeptUntil)}</th>
                    <th scope="col">
                      <span className="vh">{t(m.replay)}</span>
                    </th>
                  </tr>
                </thead>
                <tbody>
                  {items.map((d) => (
                    <tr key={d.id}>
                      <td className="nw mono">{stamp(d.firstTryAt)}</td>
                      <td>
                        <span className="ih-mk">{d.event}</span>
                      </td>
                      <td>
                        <span className="mono">{d.email ?? t(m.aContact)}</span>
                      </td>
                      <td>
                        {d.lastAnswer ? (
                          <Badge tone="bad">
                            {d.lastAnswer} {d.statusText ?? statusText(d.lastAnswer)}
                          </Badge>
                        ) : (
                          <span className="k">{d.error}</span>
                        )}
                      </td>
                      <td className="nw">{longDay(d.keptUntil)}</td>
                      <td className="r">
                        {app.canManage ? (
                          <Button size="sm" onClick={() => replay(d.id)} loading={busy === d.id}>
                            {t(m.replay)}
                          </Button>
                        ) : null}
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          </div>
          <div className="ih-only-phone ih-list">
            {items.map((d) => (
              <div key={d.id} className="ih-it">
                <div className="row justify-between gap-2">
                  <span className="mono">{d.email ?? t(m.aContact)}</span>
                  {d.lastAnswer ? <Badge tone="bad">{d.lastAnswer}</Badge> : null}
                </div>
                <span className="k">{t(m.deadLine, { event: d.event, first: stamp(d.firstTryAt), until: dayMonthDate(d.keptUntil) })}</span>
              </div>
            ))}
          </div>
        </>
      ) : null}
    </div>
  )
}

/** Loading: the header, three numbers and rows (IntegrationsStates). */
function AppSkeleton({ kind }: { kind: PlatformKind }) {
  const t = useT()
  return (
    <section className="card flex flex-col gap-4 p-5" aria-busy="true">
      <span className="vh" role="status">
        {t(m.loadingApp, { name: t(entryName[kind]) })}
      </span>
      <div className="row gap-3">
        <Skeleton w={52} h={52} r={12} />
        <div className="flex flex-col gap-2">
          <Skeleton w={140} h={18} />
          <Skeleton w={260} />
        </div>
      </div>
      <div className="ih-stats">
        {[0, 1, 2].map((i) => (
          <div key={i} className="card ih-stat gap-2">
            <Skeleton w={90} />
            <Skeleton w={60} h={22} />
          </div>
        ))}
      </div>
      {[0, 1, 2].map((i) => (
        <div key={i} className="row gap-3">
          <Skeleton w={110} />
          <Skeleton w={260} />
          <Skeleton w={80} h={22} r={999} />
        </div>
      ))}
    </section>
  )
}

function AppError({ kind, detail, onRetry }: { kind: PlatformKind; detail: string; onRetry: () => void }) {
  const { slug } = useWorkspace()
  const t = useT()
  const name = t(entryName[kind])
  return (
    <section className="card flex flex-col items-start gap-2.5 px-5 py-4.5" role="alert">
      <span className="row gap-2 font-semibold">
        <Icon name="alertCircle" size={18} className="text-bad-fg" />
        {t(m.loadAppError, { name })}
      </span>
      <span className="k">{t(m.loadAppErrorBody, { detail, name })}</span>
      <div className="row gap-2">
        <Button variant="primary" onClick={onRetry}>
          {t(m.retry)}
        </Button>
        <ButtonLink to="/w/$workspace" params={{ workspace: slug }}>
          {t(m.statusOnOverview)}
        </ButtonLink>
      </div>
    </section>
  )
}
