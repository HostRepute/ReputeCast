import { useNavigate } from '@tanstack/react-router'
import { useState } from 'react'
import { $api, errorMessage, http } from '@/api/client'
import { Button } from '@/components/ui/Button'
import { Dialog } from '@/components/ui/Dialog'
import { Skeleton } from '@/components/ui/Feedback'
import { Field, Input, OptionCard, Textarea } from '@/components/ui/Field'
import { Note } from '@/components/ui/Note'
import { toast } from '@/components/ui/Toast'
import { useT } from '@/i18n'
import { until } from '@/lib/format'
import { useCan, useMe, useWorkspace } from '@/lib/session'
import { m } from './messages'
import { dateTime, useRefreshDigest, type Digest } from './shared'

type Close = (open: boolean) => void

/** Turn on, or back on: the mode, who becomes the owner, and when the next issue goes. */
export function TurnOnDialog({ digest, onOpenChange }: { digest: Digest; onOpenChange: Close }) {
  const t = useT()
  const { slug } = useWorkspace()
  const refresh = useRefreshDigest()
  const canSend = useCan('campaigns:send')
  const [mode, setMode] = useState<'auto' | 'draft'>(canSend ? digest.mode : 'draft')
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState<string>()
  const when = digest.nextSend ? `${dateTime(digest.nextSend)} (${until(digest.nextSend)})` : t(m.nextIssueCellNoDate, { number: digest.sentIssues + 1, count: digest.schedule.count ?? 3 })
  const submit = async () => {
    setBusy(true)
    setError(undefined)
    try {
      await http.POST('/api/v1/workspaces/{workspace}/digests/{digestId}/turn-on', { params: { path: { workspace: slug, digestId: digest.id } }, body: { mode } })
      refresh()
      toast({ kind: 'success', title: t(m.turnedOn, { name: digest.name }) })
      onOpenChange(false)
    } catch (e) {
      setError(errorMessage(e))
    } finally {
      setBusy(false)
    }
  }
  return (
    <Dialog
      open
      onOpenChange={onOpenChange}
      title={t(m.turnOnTitle, { name: digest.name })}
      size="sm"
      footer={
        <>
          <Button onClick={() => onOpenChange(false)}>{t(m.cancel)}</Button>
          <Button variant="primary" loading={busy} onClick={() => void submit()}>
            {t(m.turnOnConfirm)}
          </Button>
        </>
      }
    >
      <div className="flex flex-col gap-2.5">
        <OptionCard
          name="digest-mode"
          value="auto"
          checked={mode === 'auto'}
          onChange={() => setMode('auto')}
          disabled={!canSend}
          title={t(m.modeAuto)}
          description={t(m.turnOnAuto, { when })}
        />
        <OptionCard name="digest-mode" value="draft" checked={mode === 'draft'} onChange={() => setMode('draft')} title={t(m.modeDraft)} description={t(m.turnOnDraft, { when })} />
        {!canSend ? <Note tone="info">{t(m.draftModeHint)}</Note> : null}
        <span className="k">{t(m.turnOnBecomesOwner)}</span>
        {error ? <Note tone="bad">{error}</Note> : null}
      </div>
    </Dialog>
  )
}

/** Pause, with a note for the team. */
export function PauseDialog({ digest, onOpenChange }: { digest: Digest; onOpenChange: Close }) {
  const t = useT()
  const { slug } = useWorkspace()
  const refresh = useRefreshDigest()
  const [note, setNote] = useState('')
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState<string>()
  const submit = async () => {
    setBusy(true)
    try {
      await http.POST('/api/v1/workspaces/{workspace}/digests/{digestId}/pause', { params: { path: { workspace: slug, digestId: digest.id } }, body: { note: note.trim() || undefined } })
      refresh()
      toast({ kind: 'success', title: t(m.paused, { name: digest.name }) })
      onOpenChange(false)
    } catch (e) {
      setError(errorMessage(e))
    } finally {
      setBusy(false)
    }
  }
  return (
    <Dialog
      open
      onOpenChange={onOpenChange}
      title={t(m.pauseTitle, { name: digest.name })}
      description={t(m.pauseBody)}
      size="sm"
      footer={
        <>
          <Button onClick={() => onOpenChange(false)}>{t(m.cancel)}</Button>
          <Button variant="primary" loading={busy} icon="pause" onClick={() => void submit()}>
            {t(m.pauseConfirm)}
          </Button>
        </>
      }
    >
      <Field label={t(m.pauseNote)} error={error}>
        <Textarea rows={3} maxLength={500} value={note} onChange={(e) => setNote(e.target.value)} />
      </Field>
    </Dialog>
  )
}

/** Skip an issue: nothing goes out, its posts stay new. */
export function SkipDialog({ digestId, issueId, number, onOpenChange }: { digestId: string; issueId: string; number: number; onOpenChange: Close }) {
  const t = useT()
  const { slug } = useWorkspace()
  const refresh = useRefreshDigest()
  const [note, setNote] = useState('')
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState<string>()
  const submit = async () => {
    setBusy(true)
    try {
      await http.POST('/api/v1/workspaces/{workspace}/digests/{digestId}/issues/{issueId}/skip', {
        params: { path: { workspace: slug, digestId, issueId } },
        body: { note: note.trim() || undefined },
      })
      refresh()
      toast({ kind: 'success', title: t(m.skipped) })
      onOpenChange(false)
    } catch (e) {
      setError(errorMessage(e))
    } finally {
      setBusy(false)
    }
  }
  return (
    <Dialog
      open
      onOpenChange={onOpenChange}
      title={t(m.skipTitle, { number })}
      description={t(m.skipBody)}
      size="sm"
      footer={
        <>
          <Button onClick={() => onOpenChange(false)}>{t(m.cancel)}</Button>
          <Button variant="primary" loading={busy} onClick={() => void submit()}>
            {t(m.skipConfirm)}
          </Button>
        </>
      }
    >
      <Field label={t(m.skipNote)} error={error}>
        <Input maxLength={500} value={note} onChange={(e) => setNote(e.target.value)} />
      </Field>
    </Dialog>
  )
}

/** Delete a digest that isn't on. */
export function DeleteDialog({ digest, onOpenChange }: { digest: Digest; onOpenChange: Close }) {
  const t = useT()
  const { slug } = useWorkspace()
  const navigate = useNavigate()
  const refresh = useRefreshDigest()
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState<string>()
  const submit = async () => {
    setBusy(true)
    try {
      await http.DELETE('/api/v1/workspaces/{workspace}/digests/{digestId}', { params: { path: { workspace: slug, digestId: digest.id } } })
      refresh()
      toast({ kind: 'success', title: t(m.deleted, { name: digest.name }) })
      onOpenChange(false)
      void navigate({ to: '/w/$workspace/campaigns', params: { workspace: slug }, search: { status: 'recurring' } })
    } catch (e) {
      setError(errorMessage(e))
    } finally {
      setBusy(false)
    }
  }
  return (
    <Dialog
      open
      onOpenChange={onOpenChange}
      title={t(m.deleteTitle, { name: digest.name })}
      description={t(m.deleteBody)}
      size="sm"
      dismissible={false}
      footer={
        <>
          <Button onClick={() => onOpenChange(false)}>{t(m.cancel)}</Button>
          <Button variant="danger-primary" loading={busy} onClick={() => void submit()}>
            {t(m.deleteConfirm)}
          </Button>
        </>
      }
    >
      {error ? <Note tone="bad">{error}</Note> : null}
    </Dialog>
  )
}

/** The next issue as a reader would get it now. */
export function PreviewDialog({ digest, onOpenChange }: { digest: Digest; onOpenChange: Close }) {
  const t = useT()
  const { slug } = useWorkspace()
  const q = $api.useQuery('get', '/api/v1/workspaces/{workspace}/digests/{digestId}/preview', { params: { path: { workspace: slug, digestId: digest.id } } })
  return (
    <Dialog open onOpenChange={onOpenChange} title={t(m.previewNext)} description={q.data ? q.data.subject : undefined} size="lg">
      {q.isError ? (
        <Note tone="bad">{errorMessage(q.error)}</Note>
      ) : !q.data ? (
        <Skeleton w="100%" h={420} />
      ) : (
        <iframe title={q.data.subject} srcDoc={q.data.html} sandbox="" className="h-[70vh] w-full rounded-md border border-line bg-white" />
      )}
    </Dialog>
  )
}

/** Send the next issue, as it would be built now, as a test. */
export function TestDialog({ digest, onOpenChange }: { digest: Digest; onOpenChange: Close }) {
  const t = useT()
  const { slug } = useWorkspace()
  const me = useMe()
  const [to, setTo] = useState(me.data?.user.email ?? '')
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState<string>()
  const submit = async () => {
    setBusy(true)
    setError(undefined)
    try {
      const { data } = await http.POST('/api/v1/workspaces/{workspace}/digests/{digestId}/test', {
        params: { path: { workspace: slug, digestId: digest.id } },
        body: { to: to.split(',').map((s) => s.trim()).filter(Boolean) },
      })
      toast({ kind: 'success', title: t(m.testSent, { count: data?.sent.length ?? 0, list: (data?.sent ?? []).join(', '), provider: data?.provider ?? '' }) })
      onOpenChange(false)
    } catch (e) {
      setError(errorMessage(e))
    } finally {
      setBusy(false)
    }
  }
  return (
    <Dialog
      open
      onOpenChange={onOpenChange}
      title={t(m.sendTest)}
      size="sm"
      footer={
        <>
          <Button onClick={() => onOpenChange(false)}>{t(m.cancel)}</Button>
          <Button variant="primary" loading={busy} disabled={!to.trim()} onClick={() => void submit()}>
            {t(m.sendTest)}
          </Button>
        </>
      }
    >
      <Field label={t(m.testTo)} hint={t(m.testHint)} error={error}>
        <Input type="email" multiple value={to} onChange={(e) => setTo(e.target.value)} />
      </Field>
    </Dialog>
  )
}
