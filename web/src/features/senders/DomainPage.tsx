import { useQueryClient } from '@tanstack/react-query'
import { useNavigate } from '@tanstack/react-router'
import { useState } from 'react'
import { $api, errorMessage, http } from '@/api/client'
import { Page } from '@/components/shell/Page'
import { Button } from '@/components/ui/Button'
import { ConfirmDialog } from '@/components/ui/ConfirmDialog'
import { ErrorPanel, Skeleton } from '@/components/ui/Feedback'
import { Note } from '@/components/ui/Note'
import { Row } from '@/components/ui/Layout'
import { toast } from '@/components/ui/Toast'
import { DomainOptional, DomainRecordsCard, DomainTools, useDomainDetail, type DomainDetail } from '@/features/connect/DomainRecords'
import { failed } from '@/features/connect/parts'
import { settingsTabs } from '@/features/settings/SettingsTabs'
import { usePermissionReason, useWorkspace } from '@/lib/session'
import { sendersPath } from './SendersPage'
import './senders.css'

/**
 * A sending domain's DNS records in Settings › Senders, for every provider
 * that signs for it or one of them (?provider=…), checked live.
 */
export function DomainPage({ domainId, provider, onProvider }: { domainId: string; provider?: string; onProvider: (p?: string) => void }) {
  const { slug } = useWorkspace()
  const q = useDomainDetail(slug, domainId, provider)
  const reason = usePermissionReason('domains:manage')
  const d = q.data
  const crumbs = [
    { label: 'Settings', to: '/w/$workspace/settings' as const, params: { workspace: slug } },
    { label: 'Senders', to: '/w/$workspace/settings/senders' as const, params: { workspace: slug } },
    { label: d?.domain ?? 'Domain' },
  ]
  return (
    <Page title={d ? d.domain : 'Domain'} documentTitle={d ? `${d.domain} · Senders · Settings` : 'Senders · Settings'} tabs={settingsTabs(slug)} breadcrumb={crumbs}>
      {q.isError ? (
        <ErrorPanel title="Couldn't load the domain" actions={<Button onClick={() => void q.refetch()}>Retry</Button>}>
          {errorMessage(q.error)}
        </ErrorPanel>
      ) : !d ? (
        <section className="card flex flex-col gap-3 p-5" aria-busy="true">
          <span className="vh" role="status">
            Loading the records…
          </span>
          <Skeleton w="40%" h={16} />
          <Skeleton w="100%" h={220} />
        </section>
      ) : (
        <>
          <div className="flex flex-col gap-1.5">
            <h2 className="m-0 text-[20px] font-bold tracking-[-0.01em]">{`DNS records for ${d.domain}`}</h2>
            <span className="text-ink-2">
              {d.providers.length
                ? `Each provider signs with its own DKIM key, so each one needs its records. ${d.senders === 1 ? 'One sender uses' : `${d.senders} senders use`} this domain.`
                : 'No sending provider is set up for this domain yet. Connect one, or set it up for this domain from Sending providers.'}
            </span>
          </div>
          {d.providers.length > 1 ? (
            <nav className="tabs" aria-label="Providers">
              <button type="button" aria-pressed={!provider} onClick={() => onProvider(undefined)}>
                All providers
              </button>
              {d.providers.map((p) => (
                <button key={p.id} type="button" aria-pressed={provider === p.id} onClick={() => onProvider(p.id)}>
                  {p.name}
                </button>
              ))}
            </nav>
          ) : null}
          <DomainTools workspace={slug} detail={d} reason={reason} />
          <ProviderNotes detail={d} reason={reason} />
          <DomainRecordsCard detail={d} />
          <DomainOptional workspace={slug} detail={d} reason={reason} />
          <DomainSettings detail={d} reason={reason} />
        </>
      )}
    </Page>
  )
}

/**
 * Providers that have no records for the domain yet, each with the action
 * that asks the provider for them (through its domain API, or the records to
 * publish by hand).
 */
function ProviderNotes({ detail: d, reason }: { detail: DomainDetail; reason?: string }) {
  const { slug } = useWorkspace()
  const qc = useQueryClient()
  const [busy, setBusy] = useState<string>()
  const fetchRecords = async (providerId: string, provider: string) => {
    setBusy(providerId)
    try {
      const { data } = await http.POST('/api/v1/workspaces/{workspace}/domains/{domainId}/providers/{providerId}', {
        params: { path: { workspace: slug, domainId: d.id, providerId } },
      })
      await Promise.all([qc.invalidateQueries({ queryKey: ['get', '/api/v1/workspaces/{workspace}/domains/{domainId}'] }), qc.invalidateQueries({ queryKey: ['get', sendersPath] })])
      const note = data?.notes.find((n) => n.providerId === providerId && !n.provider)
      toast({ title: `Fetched ${provider}'s records for ${d.domain}`, detail: note?.text ?? 'Publish them at your DNS host; ReputeCast checks them every 30 seconds.' })
    } catch (e) {
      failed(`Couldn't fetch ${provider}'s records`, () => void fetchRecords(providerId, provider))(e)
    } finally {
      setBusy(undefined)
    }
  }
  const missing = d.notes.filter((n) => n.provider)
  if (!missing.length) return null
  return (
    <div className="flex flex-col gap-2">
      {missing.map((n) => (
        <Note key={n.providerId} tone="info">
          <span className="flex flex-wrap items-center gap-x-3 gap-y-2">
            <span>{n.text}</span>
            <Button size="sm" onClick={() => void fetchRecords(n.providerId, n.provider)} loading={busy === n.providerId} busyLabel="Fetching…" disabledReason={reason}>
              {`Fetch ${n.provider}'s records`}
            </Button>
          </span>
        </Note>
      ))}
    </div>
  )
}

function DomainSettings({ detail: d, reason }: { detail: DomainDetail; reason?: string }) {
  const { slug } = useWorkspace()
  const qc = useQueryClient()
  const navigate = useNavigate()
  const [note, setNote] = useState(d.note)
  const [deleting, setDeleting] = useState(false)
  const update = $api.useMutation('patch', '/api/v1/workspaces/{workspace}/domains/{domainId}')
  const saveNote = () =>
    update.mutate(
      { params: { path: { workspace: slug, domainId: d.id } }, body: { note: note.trim() } },
      {
        onSuccess: () => {
          void qc.invalidateQueries({ queryKey: ['get', sendersPath] })
          toast({ title: 'Saved' })
        },
        onError: failed("Couldn't save the note", saveNote),
      },
    )
  return (
    <section className="card flex flex-col gap-3 px-5 py-4.5" aria-labelledby="domain-settings-h">
      <h2 className="h2" id="domain-settings-h">
        Domain
      </h2>
      <Row gap={12} align="flex-end">
        <label className="lbl flex-[1_1_320px]">
          What it&apos;s for
          <input className="in" value={note} onChange={(e) => setNote(e.target.value)} disabled={Boolean(reason)} placeholder="primary sending domain" />
        </label>
        <Button onClick={saveNote} loading={update.isPending} disabledReason={reason ?? (note.trim() === d.note ? 'Nothing changed.' : undefined)}>
          Save
        </Button>
        <Button
          variant="danger"
          className="ml-auto"
          onClick={() => setDeleting(true)}
          disabledReason={reason ?? (d.senders > 0 ? `${d.senders === 1 ? 'A sender uses' : `${d.senders} senders use`} this domain. Delete them first.` : undefined)}
        >
          Delete domain
        </Button>
      </Row>
      <ConfirmDialog
        open={deleting}
        onOpenChange={setDeleting}
        badge="Delete domain"
        title={`Delete ${d.domain}?`}
        description="ReputeCast stops checking its records. The records stay at your DNS host and at each provider until you remove them there."
        confirmLabel="Delete domain"
        busyLabel="Deleting…"
        onConfirm={async () => {
          await http.DELETE('/api/v1/workspaces/{workspace}/domains/{domainId}', { params: { path: { workspace: slug, domainId: d.id } } })
          await qc.invalidateQueries({ queryKey: ['get', sendersPath] })
          toast({ kind: 'deleted', title: `${d.domain} deleted` })
          void navigate({ to: '/w/$workspace/settings/senders', params: { workspace: slug } })
        }}
      />
    </section>
  )
}
