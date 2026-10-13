import { useState } from 'react'
import { $api, errorMessage, type Schemas } from '@/api/client'
import { Badge } from '@/components/ui/Badge'
import { Button } from '@/components/ui/Button'
import { Skeleton } from '@/components/ui/Feedback'
import { Grid } from '@/components/ui/Layout'
import { toast } from '@/components/ui/Toast'
import { count } from '@/lib/format'
import { usePermissionReason, useWorkspace } from '@/lib/session'
import { useInvalidateForms } from './common'

/** "Loading · building the preview": two cards shaped like the builder. */
export function FormsLoading({ label = 'Loading the signup forms' }: { label?: string }) {
  return (
    <Grid min={300} gap={16} aria-busy="true">
      <span className="vh" role="status">
        {label}
      </span>
      <div className="card flex flex-col gap-2.5 p-4">
        <Skeleton w={140} h={16} />
        <Skeleton h={34} />
        <Skeleton h={34} />
        <Skeleton h={46} w="60%" />
      </div>
      <div className="card flex flex-col gap-2.5 p-4">
        <Skeleton w={120} h={16} />
        <Skeleton h={80} />
        <Skeleton h={30} w="40%" />
      </div>
    </Grid>
  )
}

/** "No signup forms yet": the hosted page already works; make a form for your site. */
export function FormsEmpty({ hostedUrl, hostedLabel, onCreate, createReason }: { hostedUrl: string; hostedLabel: string; onCreate: () => void; createReason?: string }) {
  return (
    <section className="card flex flex-col items-center gap-2.5 border-dashed px-7 py-8 text-center">
      <h2 className="m-0 text-[17px] font-bold">No signup forms yet</h2>
      <p className="max-w-[520px] text-ink-2">
        The hosted subscribe page at {hostedLabel} already works. Create an embedded or popup form to collect signups on your own site.
      </p>
      <div className="flex flex-wrap justify-center gap-2.5">
        <Button variant="primary" onClick={onCreate} disabledReason={createReason}>
          Create a form
        </Button>
        <a className="btn" href={hostedUrl} target="_blank" rel="noreferrer">
          Open hosted page
        </a>
      </div>
    </section>
  )
}

/**
 * "41 submissions from staging.acme.io were rejected today": allow the site,
 * or keep it blocked (the alert goes away until it submits again tomorrow).
 */
export function BlockedOriginAlert({ form, blocked }: { form: Schemas['SignupForm']; blocked: Schemas['SignupFormBlockedOrigin'] }) {
  const { slug } = useWorkspace()
  const reason = usePermissionReason('forms:edit')
  const invalidate = useInvalidateForms()
  const params = { path: { workspace: slug, form: form.id } }
  const allow = $api.useMutation('post', '/api/v1/workspaces/{workspace}/forms/{form}/allowed-origins')
  const keep = $api.useMutation('post', '/api/v1/workspaces/{workspace}/forms/{form}/blocked-origins/keep')
  const run = async (which: 'allow' | 'keep') => {
    try {
      if (which === 'allow') {
        await allow.mutateAsync({ params, body: { origin: blocked.origin } })
        toast({ kind: 'success', title: `${blocked.origin} can submit "${form.name}" now`, detail: 'It was added to the allowed domains.' })
      } else {
        await keep.mutateAsync({ params, body: { origin: blocked.origin } })
        toast({ kind: 'success', title: `${blocked.origin} stays blocked`, detail: 'This alert comes back only if it submits again tomorrow.' })
      }
      await invalidate()
    } catch (e) {
      toast({ kind: 'error', title: "Couldn't change the allowed domains", detail: errorMessage(e), retry: () => void run(which) })
    }
  }
  const n = blocked.count
  return (
    <div role="alert" className="card fb-alert alert-bad">
      <div className="flex items-center gap-2.5">
        <Badge tone="bad">Blocked</Badge>
        <span className="font-semibold">
          {count(n)} {n === 1 ? 'submission' : 'submissions'} from {blocked.origin} {n === 1 ? 'was' : 'were'} rejected today
        </span>
      </div>
      <span className="k">
        {blocked.origin} isn't an allowed domain for form "{form.name}". Add it, or keep it blocked if those are test submissions.
      </span>
      <div className="flex gap-2 max-md:flex-wrap">
        <Button size="sm" variant="primary" onClick={() => void run('allow')} loading={allow.isPending} disabledReason={reason}>
          Allow {blocked.origin}
        </Button>
        <Button size="sm" onClick={() => void run('keep')} loading={keep.isPending} disabledReason={reason}>
          Keep blocked
        </Button>
      </div>
    </div>
  )
}

/**
 * "Cloudflare rejected the site key (error 110200)": spam protection fell
 * back to the honeypot; paste the right key. The secret stays as stored.
 */
export function TurnstileAlert({ turnstile }: { turnstile: Schemas['PublicTurnstile'] }) {
  const { slug } = useWorkspace()
  const reason = usePermissionReason('forms:edit')
  const invalidate = useInvalidateForms()
  const save = $api.useMutation('put', '/api/v1/workspaces/{workspace}/public-pages/turnstile')
  const [key, setKey] = useState('')
  const [error, setError] = useState<string>()
  const submit = async () => {
    const siteKey = key.trim()
    if (!siteKey || save.isPending) return
    setError(undefined)
    try {
      const out = await save.mutateAsync({ params: { path: { workspace: slug } }, body: { siteKey } })
      await invalidate()
      if (out.turnstile.status === 'ok') {
        setKey('')
        toast({ kind: 'success', title: 'Turnstile site key saved', detail: 'Cloudflare accepted the keys; suspicious signups get the check again.' })
      } else {
        setError(`${out.turnstile.error}. Check the key in the Cloudflare dashboard.`)
      }
    } catch (e) {
      setError(errorMessage(e))
    }
  }
  return (
    <div role="alert" className="card fb-alert alert-warn">
      <div className="flex items-center gap-2.5">
        <Badge tone="warn">Turnstile</Badge>
        <span className="font-semibold">{turnstile.error || 'Cloudflare rejected the Turnstile keys'}</span>
      </div>
      <span className="k">
        Spam protection fell back to the honeypot and timing check. Paste the key from the Cloudflare dashboard; the secret key is stored encrypted.
      </span>
      <label className="lbl">
        Turnstile site key
        <input
          className="fb-in mono in-warn"
          placeholder="0x4AAAAAAA…"
          value={key}
          disabled={Boolean(reason)}
          title={reason}
          aria-invalid={error ? true : undefined}
          aria-describedby={error ? 'turnstile-key-err' : undefined}
          onChange={(e) => setKey(e.target.value)}
          onBlur={() => void submit()}
          onKeyDown={(e) => {
            if (e.key === 'Enter') {
              e.preventDefault()
              void submit()
            }
          }}
        />
      </label>
      {error ? (
        <span className="err" id="turnstile-key-err" role="alert">
          {error}
        </span>
      ) : null}
    </div>
  )
}
