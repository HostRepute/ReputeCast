import { useQueryClient } from '@tanstack/react-query'
import { useNavigate } from '@tanstack/react-router'
import { useState } from 'react'
import { http } from '@/api/client'
import { Button } from '@/components/ui/Button'
import { Dialog } from '@/components/ui/Dialog'
import { Field, Input } from '@/components/ui/Field'
import { toast } from '@/components/ui/Toast'
import { problemMessage } from '@/features/connect/parts'
import { usePermissionReason, useWorkspace } from '@/lib/session'
import { sendersPath } from './SendersPage'

/** Add a sending domain; its records open next. */
export function DomainDialog({ open, onOpenChange }: { open: boolean; onOpenChange: (o: boolean) => void }) {
  const { slug } = useWorkspace()
  const qc = useQueryClient()
  const navigate = useNavigate()
  const reason = usePermissionReason('domains:manage')
  const [domain, setDomain] = useState('')
  const [note, setNote] = useState('')
  const [error, setError] = useState<string>()
  const [busy, setBusy] = useState(false)
  const add = async () => {
    setBusy(true)
    setError(undefined)
    try {
      const { data } = await http.POST('/api/v1/workspaces/{workspace}/domains', {
        params: { path: { workspace: slug } },
        body: { domain: domain.trim().toLowerCase(), note: note.trim() || undefined },
      })
      await qc.invalidateQueries({ queryKey: ['get', sendersPath] })
      const created = data!
      toast({
        title: `${created.domain.domain} added`,
        detail: created.notes.length
          ? created.notes.map((n) => (n.text.startsWith(n.provider) ? n.text : `${n.provider}: ${n.text}`)).join(' ')
          : 'Add its records at your DNS host; ReputeCast checks them as they appear.',
      })
      onOpenChange(false)
      setDomain('')
      setNote('')
      void navigate({ to: '/w/$workspace/settings/senders/$domainId', params: { workspace: slug, domainId: created.domain.id } })
    } catch (e) {
      setError(problemMessage(e))
    } finally {
      setBusy(false)
    }
  }
  return (
    <Dialog
      open={open}
      onOpenChange={onOpenChange}
      size="sm"
      title="Add domain"
      description="The domain in your From addresses, e.g. news.example.com. Each connected provider that can set it up through its API gets it, with the DNS records to add."
      footer={
        <>
          <Button onClick={() => onOpenChange(false)}>Cancel</Button>
          <Button variant="primary" onClick={() => void add()} loading={busy} disabledReason={reason ?? (domain.trim() ? undefined : 'Enter the domain first.')}>
            Add domain
          </Button>
        </>
      }
    >
      <Field label="Domain" error={error}>
        <Input
          className="mono"
          value={domain}
          onChange={(e) => setDomain(e.target.value)}
          placeholder="news.example.com"
          autoComplete="off"
          spellCheck={false}
          autoCapitalize="none"
          onKeyDown={(e) => (e.key === 'Enter' && domain.trim() ? void add() : undefined)}
        />
      </Field>
      <Field label="What it's for" optional hint="Shown under the domain, e.g. primary sending domain.">
        <Input value={note} onChange={(e) => setNote(e.target.value)} />
      </Field>
    </Dialog>
  )
}
