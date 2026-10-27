import { useQueryClient } from '@tanstack/react-query'
import { useState } from 'react'
import { $api, errorMessage, type Schemas } from '@/api/client'
import { Page } from '@/components/shell/Page'
import { Badge, type Tone } from '@/components/ui/Badge'
import { Button, ButtonLink } from '@/components/ui/Button'
import { ErrorPanel, Skeleton } from '@/components/ui/Feedback'
import { Grid } from '@/components/ui/Layout'
import { EmptyRow, Table, Th } from '@/components/ui/Table'
import { toast } from '@/components/ui/Toast'
import { failed } from '@/features/connect/parts'
import { streamName, type Stream } from '@/features/providers/common'
import { settingsTabs } from '@/features/settings/SettingsTabs'
import { SmsShortLinkCard } from '@/features/sms/SendersCard'
import { T, useT } from '@/i18n'
import { common } from '@/i18n/common'
import { usePermissionReason, useWorkspace } from '@/lib/session'
import { DomainDialog } from './DomainDialog'
import { m } from './messages'
import { ReadOnlyNote } from './ReadOnlyNote'
import { SenderDialog } from './SenderDialog'
import './senders.css'

export type Overview = Schemas['SendersOverview']
export type Sender = Schemas['SenderIdentity']

export const sendersPath = '/api/v1/workspaces/{workspace}/senders' as const

const VERIFICATION_TONE: Record<Schemas['SenderVerification']['status'], Tone> = {
  verified: 'ok',
  pending: 'warn',
  failed: 'bad',
  not_set_up: 'neutral',
  not_used: 'neutral',
}

const CELL_TONE: Record<Schemas['SenderDomainCell']['tone'], Tone> = { ok: 'ok', warn: 'warn', bad: 'bad', neutral: 'neutral' }

/** 7.13 Settings › Senders (SettingsSenders): sender identities, domains and defaults. */
export function SendersPage() {
  const t = useT()
  const { slug } = useWorkspace()
  const q = $api.useQuery('get', sendersPath, { params: { path: { workspace: slug } } })
  const [editing, setEditing] = useState<Sender | 'new'>()
  const [addingDomain, setAddingDomain] = useState(false)
  const data = q.data
  const reason = usePermissionReason('domains:manage', data?.helpers)
  return (
    <Page
      title={t(m.settings)}
      documentTitle={t(m.documentTitle)}
      tabs={settingsTabs(slug)}
      badges={
        <Badge tone="ok" className="ml-auto">
          {t(m.ossBadge)}
        </Badge>
      }
      toolsAfterTabs
    >
      {q.isError ? (
        <ErrorPanel title={t(m.loadFailed)} actions={<Button onClick={() => void q.refetch()}>{t(common.retry)}</Button>}>
          {errorMessage(q.error)}
        </ErrorPanel>
      ) : (
        <>
          {data && !data.canManage ? <ReadOnlyNote msg={m.readOnlyText} helpers={data.helpers} /> : null}
          <section className="card" aria-labelledby="senders-h">
            <div className="sec">
              <div className="flex flex-col">
                <h2 className="h2" id="senders-h">
                  {t(m.identitiesTitle)}
                </h2>
                <span className="k">{t(m.identitiesLine)}</span>
              </div>
              <Button variant="primary" onClick={() => setEditing('new')} disabledReason={data ? reason : t(common.loading)}>
                {t(m.addSender)}
              </Button>
            </div>
            {data ? <SendersTable data={data} onEdit={setEditing} /> : <TableSkeleton rows={4} />}
          </section>

          <section className="card" aria-labelledby="domains-h">
            <div className="sec">
              <div className="flex flex-col">
                <h2 className="h2" id="domains-h">
                  {t(m.domainsTitle)}
                </h2>
                <span className="k">
                  {t(m.domainsLine)} {data ? <TransactionalHint data={data} /> : null}
                </span>
              </div>
              <Button onClick={() => setAddingDomain(true)} disabledReason={data ? reason : t(common.loading)}>
                {t(m.addDomain)}
              </Button>
            </div>
            {data ? <DomainsTable data={data} /> : <TableSkeleton rows={2} />}
          </section>
          {/* Phase 8: the short-link domain of texts, with its record and status. */}
          <SmsShortLinkCard />

          {data ? <DefaultsCard data={data} reason={reason} /> : null}
        </>
      )}
      {data ? (
        <>
          <SenderDialog
            key={editing === 'new' ? 'new' : editing?.id}
            open={editing !== undefined}
            onOpenChange={(o) => !o && setEditing(undefined)}
            sender={editing === 'new' ? undefined : editing}
            data={data}
          />
          <DomainDialog open={addingDomain} onOpenChange={setAddingDomain} />
        </>
      ) : null}
    </Page>
  )
}

function SendersTable({ data, onEdit }: { data: Overview; onEdit: (s: Sender) => void }) {
  const t = useT()
  const { slug } = useWorkspace()
  return (
    <Table className="sd-tbl" fit caption={t(m.identitiesCaption)}>
      <thead>
        <tr>
          <Th>{t(m.colFrom)}</Th>
          <Th>{t(m.colReplyTo)}</Th>
          <Th>{t(m.colDefaultFor)}</Th>
          {data.providers.map((p) => (
            <Th key={p.id}>{p.name}</Th>
          ))}
          <Th>
            <span className="vh">{t(m.colActions)}</span>
          </Th>
        </tr>
      </thead>
      <tbody>
        {data.senders.length === 0 ? <EmptyRow colSpan={4 + data.providers.length}>{t(m.noSenders)}</EmptyRow> : null}
        {data.senders.map((s) => {
          const pending = s.verification.find((v) => v.status === 'pending' || v.status === 'failed')
          return (
            <tr key={s.id}>
              <td>
                <span className="font-semibold">{s.name}</span>
                <div className="k mono">{s.email}</div>
              </td>
              <td className="mono">{s.replyTo || s.email}</td>
              {s.defaultFor.length || s.transactional ? (
                <td>
                  <span className="flex flex-wrap items-center gap-1.5">
                    {/* A sender on a Transactional domain sends transactional mail, default or not. */}
                    {[...s.defaultFor.filter((d) => d !== 'transactional'), ...(s.transactional ? ['transactional'] : [])].map((d) => (
                      <Badge key={d} tone="purple">
                        {streamName[d as Stream] ?? d}
                      </Badge>
                    ))}
                    {s.smtpRelay ? <span className="k">{t(m.smtpRelay)}</span> : null}
                  </span>
                </td>
              ) : (
                <td className="k">{s.note}</td>
              )}
              {data.providers.map((p) => {
                const v = s.verification.find((x) => x.providerId === p.id)
                return <td key={p.id}>{v ? <Badge tone={VERIFICATION_TONE[v.status]}>{v.text}</Badge> : null}</td>
              })}
              <td>
                {!data.canManage ? (
                  <span className="k">–</span>
                ) : pending ? (
                  <ButtonLink size="sm" to="/w/$workspace/settings/senders/$domainId" params={{ workspace: slug, domainId: s.domainId }} search={{ provider: pending.providerId }}>
                    {t(m.verify)}
                  </ButtonLink>
                ) : (
                  <Button size="sm" onClick={() => onEdit(s)} aria-label={t(m.editSenderLabel, { name: s.name, email: s.email })}>
                    {t(common.edit)}
                  </Button>
                )}
              </td>
            </tr>
          )
        })}
      </tbody>
    </Table>
  )
}

function DomainsTable({ data }: { data: Overview }) {
  const t = useT()
  const { slug } = useWorkspace()
  return (
    <Table className="sd-tbl" fit caption={t(m.domainsCaption)}>
      <thead>
        <tr>
          <Th>{t(m.colDomain)}</Th>
          <Th>{t(m.colUsedBy)}</Th>
          <Th>{'DKIM'}</Th>
          <Th>{'SPF'}</Th>
          <Th>{'DMARC'}</Th>
          <Th>{t(m.colReturnPath)}</Th>
          <Th>{t(m.colTracking)}</Th>
          <Th>{'BIMI'}</Th>
          <Th>
            <span className="vh">{t(m.colRecords)}</span>
          </Th>
        </tr>
      </thead>
      <tbody>
        {data.domains.length === 0 ? <EmptyRow colSpan={9}>{t(m.noDomains)}</EmptyRow> : null}
        {data.domains.map((d) => (
          <tr key={d.id}>
            <td>
              <span className="mono font-semibold">{d.domain}</span>
              {d.note ? <div className="k">{d.note}</div> : null}
            </td>
            <td>{d.usedBy.length ? d.usedBy.join(' · ') : <span className="k">{t(m.noProviderYet)}</span>}</td>
            {[d.dkim, d.spf, d.dmarc, d.returnPath, d.tracking, d.bimi].map((c, i) => (
              <td key={i}>
                <Badge tone={CELL_TONE[c.tone]}>{c.text}</Badge>
              </td>
            ))}
            <td>
              <ButtonLink size="sm" to="/w/$workspace/settings/senders/$domainId" params={{ workspace: slug, domainId: d.id }} aria-label={t(m.recordsOf, { domain: d.domain })}>
                {t(m.records)}
              </ButtonLink>
            </td>
          </tr>
        ))}
      </tbody>
    </Table>
  )
}

function DefaultsCard({ data, reason }: { data: Overview; reason?: string }) {
  const t = useT()
  const { slug } = useWorkspace()
  const qc = useQueryClient()
  const d = data.defaults
  const [senderId, setSenderId] = useState(d.defaultSenderId)
  const [txSenderId, setTxSenderId] = useState(d.transactionalSenderId ?? '')
  const txSenders = data.senders.filter((x) => x.transactional)
  // Where replies go: one address for every sender, or "" for each sender's own reply-to.
  const [reply, setReply] = useState(d.replyTo)
  const [footer, setFooter] = useState(d.footerAddress)
  const [tracking, setTracking] = useState(d.trackingDomain)
  const save = $api.useMutation('put', '/api/v1/workspaces/{workspace}/senders/defaults')
  const submit = () => {
    save.mutate(
      {
        params: { path: { workspace: slug } },
        body: {
          defaultSenderId: senderId || undefined,
          transactionalSenderId: txSenderId || undefined,
          replyHandling: 'forward',
          replyTo: reply || undefined,
          footerAddress: footer.trim(),
          trackingDomain: tracking,
        },
      },
      {
        onSuccess: () => {
          void qc.invalidateQueries({ queryKey: ['get', sendersPath] })
          toast({ title: t(m.defaultsSaved) })
        },
        onError: failed(t(m.defaultsFailed), submit),
      },
    )
  }
  return (
    <section className="card flex flex-col gap-3 px-5 py-4.5" aria-labelledby="defaults-h">
      <h2 className="h2" id="defaults-h">
        {t(m.defaultsTitle)}
      </h2>
      <Grid min={240} gap={12}>
        <label className="lbl">
          {t(m.defaultFrom)}
          <select className="in" value={senderId} onChange={(e) => setSenderId(e.target.value)} disabled={Boolean(reason)}>
            {data.senders.map((s) => (
              <option key={s.id} value={s.id}>{`${s.name} · ${s.email}`}</option>
            ))}
          </select>
        </label>
        {txSenders.length ? (
          <label className="lbl">
            {t(m.defaultFromTx)}
            <select className="in" value={txSenderId} onChange={(e) => setTxSenderId(e.target.value)} disabled={Boolean(reason)}>
              {txSenderId ? null : <option value="">{t(m.chooseSender)}</option>}
              {txSenders.map((x) => (
                <option key={x.id} value={x.id}>{`${x.name} · ${x.email}`}</option>
              ))}
            </select>
          </label>
        ) : null}
        <label className="lbl">
          {t(m.replyHandling)}
          <select className="in" value={reply} onChange={(e) => setReply(e.target.value)} disabled={Boolean(reason)}>
            {(d.replyOptions.length ? d.replyOptions : [d.replyTo].filter(Boolean)).map((a) => (
              <option key={a} value={a}>
                {t(m.forwardReplies, { address: a })}
              </option>
            ))}
            <option value="">{t(m.ownReplyTo)}</option>
          </select>
        </label>
        <label className="lbl">
          {t(m.footerAddress)}
          <input className="in" value={footer} onChange={(e) => setFooter(e.target.value)} disabled={Boolean(reason)} />
        </label>
        <label className="lbl">
          {t(m.trackingDomain)}
          <select className="in" value={tracking} onChange={(e) => setTracking(e.target.value)} disabled={Boolean(reason)}>
            {d.trackingOptions.map((o) => (
              <option key={o.domain} value={o.domain}>
                {t(m.trackingOption, { domain: o.domain, verified: o.verified ? 'yes' : 'no' })}
              </option>
            ))}
            <option value="">{t(m.providerDefault)}</option>
          </select>
        </label>
      </Grid>
      <div className="flex gap-2">
        <Button variant="primary" onClick={submit} loading={save.isPending} disabledReason={reason}>
          {t(common.save)}
        </Button>
      </div>
    </section>
  )
}

/** The hint for transactional mail: its own subdomain keeps its reputation apart (SettingsSenders). */
function TransactionalHint({ data }: { data: Overview }) {
  const t = useT()
  const tx = data.domains.find((d) => d.transactional)
  return tx ? <T msg={m.txHintHas} values={{ domain: tx.domain, mono: (c) => <span className="mono">{c}</span> }} /> : <>{t(m.txHintGive)}</>
}

function TableSkeleton({ rows }: { rows: number }) {
  const t = useT()
  return (
    <div className="flex flex-col gap-3 px-4.5 py-4" aria-busy="true">
      <span className="vh" role="status">
        {t(common.loading)}
      </span>
      {Array.from({ length: rows }, (_, i) => (
        <Skeleton key={i} w="100%" h={28} />
      ))}
    </div>
  )
}
