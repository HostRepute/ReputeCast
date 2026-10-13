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
import { settingsTabs } from '@/features/settings/SettingsTabs'
import { usePermissionReason, useWorkspace } from '@/lib/session'
import { DomainDialog } from './DomainDialog'
import { ReadOnlyNote } from './ReadOnlyNote'
import { SenderDialog } from './SenderDialog'
import './senders.css'

export type Overview = Schemas['SendersOverview']
export type Sender = Schemas['SenderIdentity']

export const sendersPath = '/api/v1/workspaces/{workspace}/senders' as const

const STREAM_NAMES: Record<string, string> = { broadcast: 'Broadcasts', automation: 'Automations', system: 'System' }

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
  const { slug } = useWorkspace()
  const q = $api.useQuery('get', sendersPath, { params: { path: { workspace: slug } } })
  const [editing, setEditing] = useState<Sender | 'new'>()
  const [addingDomain, setAddingDomain] = useState(false)
  const data = q.data
  const reason = usePermissionReason('domains:manage', data?.helpers)
  return (
    <Page
      title="Settings"
      documentTitle="Senders · Settings"
      tabs={settingsTabs(slug)}
      badges={
        <Badge tone="ok" className="ml-auto">
          Open-source edition · every feature included
        </Badge>
      }
      toolsAfterTabs
    >
      {q.isError ? (
        <ErrorPanel title="Couldn't load the senders" actions={<Button onClick={() => void q.refetch()}>Retry</Button>}>
          {errorMessage(q.error)}
        </ErrorPanel>
      ) : (
        <>
          {data && !data.canManage ? (
            <ReadOnlyNote perm="domains:manage" helpers={data.helpers}>
              You can view the senders, domains and their records.
            </ReadOnlyNote>
          ) : null}
          <section className="card" aria-labelledby="senders-h">
            <div className="sec">
              <div className="flex flex-col">
                <h2 className="h2" id="senders-h">
                  Sender identities
                </h2>
                <span className="k">Verification is per sending provider: each one signs with its own DKIM key.</span>
              </div>
              <Button variant="primary" onClick={() => setEditing('new')} disabledReason={data ? reason : 'Loading…'}>
                Add sender
              </Button>
            </div>
            {data ? <SendersTable data={data} onEdit={setEditing} /> : <TableSkeleton rows={4} />}
          </section>

          <section className="card" aria-labelledby="domains-h">
            <div className="sec">
              <div className="flex flex-col">
                <h2 className="h2" id="domains-h">
                  Domains
                </h2>
                <span className="k">DNS checked every 30 minutes; a change here shows up in Send Readiness within the hour.</span>
              </div>
              <Button onClick={() => setAddingDomain(true)} disabledReason={data ? reason : 'Loading…'}>
                Add domain
              </Button>
            </div>
            {data ? <DomainsTable data={data} /> : <TableSkeleton rows={2} />}
          </section>

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
  const { slug } = useWorkspace()
  return (
    <Table className="sd-tbl" caption="Sender identities">
      <thead>
        <tr>
          <Th>From</Th>
          <Th>Reply-to</Th>
          <Th>Default for</Th>
          {data.providers.map((p) => (
            <Th key={p.id}>{p.name}</Th>
          ))}
          <Th>
            <span className="vh">Actions</span>
          </Th>
        </tr>
      </thead>
      <tbody>
        {data.senders.length === 0 ? (
          <EmptyRow colSpan={4 + data.providers.length}>No senders yet. Add the address your mail comes from.</EmptyRow>
        ) : null}
        {data.senders.map((s) => {
          const pending = s.verification.find((v) => v.status === 'pending' || v.status === 'failed')
          return (
            <tr key={s.id}>
              <td>
                <span className="font-semibold">{s.name}</span>
                <div className="k mono">{s.email}</div>
              </td>
              <td className="mono">{s.replyTo || s.email}</td>
              {s.defaultFor.length ? (
                <td>
                  <span className="flex gap-1.5">
                    {s.defaultFor.map((d) => (
                      <Badge key={d} tone="purple">
                        {STREAM_NAMES[d] ?? d}
                      </Badge>
                    ))}
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
                    Verify
                  </ButtonLink>
                ) : (
                  <Button size="sm" onClick={() => onEdit(s)} aria-label={`Edit ${s.name} <${s.email}>`}>
                    Edit
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
  const { slug } = useWorkspace()
  return (
    <Table className="sd-tbl" caption="Sending domains">
      <thead>
        <tr>
          <Th>Domain</Th>
          <Th>Used by</Th>
          <Th>DKIM</Th>
          <Th>SPF</Th>
          <Th>DMARC</Th>
          <Th>Return-path</Th>
          <Th>Tracking domain</Th>
          <Th>BIMI</Th>
          <Th>
            <span className="vh">Records</span>
          </Th>
        </tr>
      </thead>
      <tbody>
        {data.domains.length === 0 ? <EmptyRow colSpan={9}>No sending domains yet. Add the domain in your From addresses.</EmptyRow> : null}
        {data.domains.map((d) => (
          <tr key={d.id}>
            <td>
              <span className="mono font-semibold">{d.domain}</span>
              {d.note ? <div className="k">{d.note}</div> : null}
            </td>
            <td>{d.usedBy.length ? d.usedBy.join(' · ') : <span className="k">No provider yet</span>}</td>
            {[d.dkim, d.spf, d.dmarc, d.returnPath, d.tracking, d.bimi].map((c, i) => (
              <td key={i}>
                <Badge tone={CELL_TONE[c.tone]}>{c.text}</Badge>
              </td>
            ))}
            <td>
              <ButtonLink size="sm" to="/w/$workspace/settings/senders/$domainId" params={{ workspace: slug, domainId: d.id }} aria-label={`Records of ${d.domain}`}>
                Records
              </ButtonLink>
            </td>
          </tr>
        ))}
      </tbody>
    </Table>
  )
}

function DefaultsCard({ data, reason }: { data: Overview; reason?: string }) {
  const { slug } = useWorkspace()
  const qc = useQueryClient()
  const d = data.defaults
  const [senderId, setSenderId] = useState(d.defaultSenderId)
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
          replyHandling: 'forward',
          replyTo: reply || undefined,
          footerAddress: footer.trim(),
          trackingDomain: tracking,
        },
      },
      {
        onSuccess: () => {
          void qc.invalidateQueries({ queryKey: ['get', sendersPath] })
          toast({ title: 'Defaults saved' })
        },
        onError: failed("Couldn't save the defaults", submit),
      },
    )
  }
  return (
    <section className="card flex flex-col gap-3 px-5 py-4.5" aria-labelledby="defaults-h">
      <h2 className="h2" id="defaults-h">
        Defaults
      </h2>
      <Grid min={240} gap={12}>
        <label className="lbl">
          Default From for new campaigns
          <select className="in" value={senderId} onChange={(e) => setSenderId(e.target.value)} disabled={Boolean(reason)}>
            {data.senders.map((s) => (
              <option key={s.id} value={s.id}>{`${s.name} · ${s.email}`}</option>
            ))}
          </select>
        </label>
        <label className="lbl">
          Reply handling
          <select className="in" value={reply} onChange={(e) => setReply(e.target.value)} disabled={Boolean(reason)}>
            {(d.replyOptions.length ? d.replyOptions : [d.replyTo].filter(Boolean)).map((a) => (
              <option key={a} value={a}>{`Forward replies to ${a}`}</option>
            ))}
            <option value="">{"Use each sender's reply-to"}</option>
          </select>
        </label>
        <label className="lbl">
          Footer address
          <input className="in" value={footer} onChange={(e) => setFooter(e.target.value)} disabled={Boolean(reason)} />
        </label>
        <label className="lbl">
          Link tracking domain
          <select className="in" value={tracking} onChange={(e) => setTracking(e.target.value)} disabled={Boolean(reason)}>
            {d.trackingOptions.map((t) => (
              <option key={t.domain} value={t.domain}>{`${t.domain} · ${t.verified ? 'verified' : 'not verified yet'}`}</option>
            ))}
            <option value="">Provider default</option>
          </select>
        </label>
      </Grid>
      <div className="flex gap-2">
        <Button variant="primary" onClick={submit} loading={save.isPending} disabledReason={reason}>
          Save
        </Button>
      </div>
    </section>
  )
}

function TableSkeleton({ rows }: { rows: number }) {
  return (
    <div className="flex flex-col gap-3 px-4.5 py-4" aria-busy="true">
      <span className="vh" role="status">
        Loading…
      </span>
      {Array.from({ length: rows }, (_, i) => (
        <Skeleton key={i} w="100%" h={28} />
      ))}
    </div>
  )
}
