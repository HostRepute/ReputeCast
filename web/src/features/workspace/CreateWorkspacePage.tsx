import { useQueryClient } from '@tanstack/react-query'
import { Link, useNavigate } from '@tanstack/react-router'
import { useId, useMemo, useState, type FormEvent } from 'react'
import { ApiError, errorMessage, http } from '@/api/client'
import { AuthLayout } from '@/components/shell/AuthLayout'
import { Button } from '@/components/ui/Button'
import { Note } from '@/components/ui/Note'
import { meQueryKey, useMe } from '@/lib/session'
import { timezoneOptions } from './timezones'

function browserZone() {
  try {
    return Intl.DateTimeFormat().resolvedOptions().timeZone || 'UTC'
  } catch {
    return 'UTC'
  }
}

/**
 * Create a workspace (A5 §3.4): a name and a timezone; the creator becomes
 * its Owner. Workspaces are unlimited.
 */
export function CreateWorkspacePage() {
  const me = useMe().data
  const navigate = useNavigate()
  const qc = useQueryClient()
  const [name, setName] = useState('')
  const [tz, setTz] = useState(() => me?.user.timezone || browserZone())
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState<{ field?: 'name'; message: string }>()
  const zones = useMemo(() => timezoneOptions(tz), [tz])
  const errId = useId()
  const back = me?.workspaces[0]?.slug
  const submit = async (e: FormEvent) => {
    e.preventDefault()
    if (!name.trim()) {
      setError({ field: 'name', message: 'Give the workspace a name, for example your newsletter or brand.' })
      return
    }
    setBusy(true)
    setError(undefined)
    try {
      const { data } = await http.POST('/api/v1/workspaces', { body: { name: name.trim(), timezone: tz } })
      await qc.invalidateQueries({ queryKey: meQueryKey })
      void navigate({ to: '/w/$workspace', params: { workspace: data!.slug } })
    } catch (err) {
      const field = err instanceof ApiError && err.problem.errors?.some((x) => x.location === 'body.name') ? 'name' : undefined
      setError({ field, message: errorMessage(err) })
    } finally {
      setBusy(false)
    }
  }
  return (
    <AuthLayout title="Create a workspace" labelledBy="h-new-ws">
      <div className="flex flex-col gap-1">
        <h1 id="h-new-ws">Create a workspace</h1>
        <span className="k">
          You become its Owner. Each workspace keeps its own contacts, campaigns, providers and team; the suppression list is shared across the instance.
        </span>
      </div>
      <form className="flex flex-col gap-4" onSubmit={submit} noValidate>
        <label className="lbl">
          Workspace name
          <input
            className="in"
            type="text"
            value={name}
            onChange={(e) => setName(e.target.value)}
            maxLength={80}
            placeholder="Acme Newsletter"
            autoFocus
            aria-invalid={error?.field === 'name' || undefined}
            aria-describedby={error?.field === 'name' ? errId : undefined}
          />
          {error?.field === 'name' ? (
            <span className="err" id={errId} role="alert">
              {error.message}
            </span>
          ) : null}
        </label>
        <label className="lbl">
          Timezone
          <select className="in" value={tz} onChange={(e) => setTz(e.target.value)}>
            {zones.map((z) => (
              <option key={z.value} value={z.value}>
                {z.label}
              </option>
            ))}
          </select>
          <span className="k font-medium">Schedules and reports use this timezone. You can change it later in Settings.</span>
        </label>
        {error && !error.field ? (
          <Note tone="bad" label="Not created">
            {error.message}
          </Note>
        ) : null}
        <Button type="submit" variant="primary" loading={busy}>
          Create workspace
        </Button>
      </form>
      <div className="links">
        {back ? (
          <Link to="/w/$workspace" params={{ workspace: back }}>
            Back to {me?.workspaces[0]?.name}
          </Link>
        ) : (
          <Link to="/">Back</Link>
        )}
      </div>
    </AuthLayout>
  )
}
