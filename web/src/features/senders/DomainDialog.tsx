import { useQueryClient } from '@tanstack/react-query'
import { useNavigate } from '@tanstack/react-router'
import { useState } from 'react'
import { http } from '@/api/client'
import { Button } from '@/components/ui/Button'
import { Dialog } from '@/components/ui/Dialog'
import { Field, Input } from '@/components/ui/Field'
import { toast } from '@/components/ui/Toast'
import { problemMessage } from '@/features/connect/parts'
import { useT } from '@/i18n'
import { common } from '@/i18n/common'
import { usePermissionReason, useWorkspace } from '@/lib/session'
import { m } from './messages'
import { sendersPath } from './SendersPage'

/** Add a sending domain; its records open next. */
export function DomainDialog({ open, onOpenChange }: { open: boolean; onOpenChange: (o: boolean) => void }) {
  const t = useT()
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
        title: t(m.domainAdded, { domain: created.domain.domain }),
        detail: created.notes.length
          ? created.notes.map((n) => (n.text.startsWith(n.provider) ? n.text : t(m.providerNote, { provider: n.provider, note: n.text }))).join(' ')
          : t(m.domainAddedDetail),
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
      title={t(m.addDomainTitle)}
      description={t(m.addDomainDescription)}
      footer={
        <>
          <Button onClick={() => onOpenChange(false)}>{t(common.cancel)}</Button>
          <Button variant="primary" onClick={() => void add()} loading={busy} disabledReason={reason ?? (domain.trim() ? undefined : t(m.enterDomainFirst))}>
            {t(m.addDomainSubmit)}
          </Button>
        </>
      }
    >
      <Field label={t(m.domain)} error={error}>
        <Input
          className="mono"
          value={domain}
          onChange={(e) => setDomain(e.target.value)}
          placeholder={'news.example.com'}
          autoComplete="off"
          spellCheck={false}
          autoCapitalize="none"
          onKeyDown={(e) => (e.key === 'Enter' && domain.trim() ? void add() : undefined)}
        />
      </Field>
      <Field label={t(m.whatFor)} optional hint={t(m.whatForHint)}>
        <Input value={note} onChange={(e) => setNote(e.target.value)} />
      </Field>
    </Dialog>
  )
}
