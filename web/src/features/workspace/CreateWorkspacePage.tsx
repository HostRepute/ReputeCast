import { useQueryClient } from '@tanstack/react-query'
import { Link, useNavigate } from '@tanstack/react-router'
import { useId, useMemo, useState, type FormEvent } from 'react'
import { ApiError, errorMessage, http } from '@/api/client'
import { AuthLayout } from '@/components/shell/AuthLayout'
import { Button } from '@/components/ui/Button'
import { Note } from '@/components/ui/Note'
import { useT } from '@/i18n'
import { common } from '@/i18n/common'
import { meQueryKey, useMe } from '@/lib/session'
import { m } from './messages'
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
  const t = useT()
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
      setError({ field: 'name', message: t(m.createNameRequired) })
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
    <AuthLayout title={t(m.createTitle)} labelledBy="h-new-ws">
      <div className="flex flex-col gap-1">
        <h1 id="h-new-ws">{t(m.createTitle)}</h1>
        <span className="k">{t(m.createIntro)}</span>
      </div>
      <form className="flex flex-col gap-4" onSubmit={submit} noValidate>
        <label className="lbl">
          {t(m.nameLabel)}
          <input
            className="in"
            type="text"
            value={name}
            onChange={(e) => setName(e.target.value)}
            maxLength={80}
            placeholder={t(m.namePlaceholder)}
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
          {t(m.timezoneLabel)}
          <select className="in" value={tz} onChange={(e) => setTz(e.target.value)}>
            {zones.map((z) => (
              <option key={z.value} value={z.value}>
                {z.label}
              </option>
            ))}
          </select>
          <span className="k font-medium">{t(m.createTimezoneHint)}</span>
        </label>
        {error && !error.field ? (
          <Note tone="bad" label={t(m.notCreated)}>
            {error.message}
          </Note>
        ) : null}
        <Button type="submit" variant="primary" loading={busy}>
          {t(m.createButton)}
        </Button>
      </form>
      <div className="links">
        {back ? (
          <Link to="/w/$workspace" params={{ workspace: back }}>
            {t(m.backTo, { name: me?.workspaces[0]?.name ?? '' })}
          </Link>
        ) : (
          <Link to="/">{t(common.back)}</Link>
        )}
      </div>
    </AuthLayout>
  )
}
