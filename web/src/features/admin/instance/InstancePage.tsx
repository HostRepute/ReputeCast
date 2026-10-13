import { useQueryClient } from '@tanstack/react-query'
import { useMemo, useRef, useState } from 'react'
import { $api, errorMessage, type Schemas } from '@/api/client'
import { LogoMark } from '@/components/icons/Icon'
import { Badge } from '@/components/ui/Badge'
import { Button } from '@/components/ui/Button'
import { CardSection } from '@/components/ui/Card'
import { ErrorPanel, Skeleton } from '@/components/ui/Feedback'
import { Note } from '@/components/ui/Note'
import { toast } from '@/components/ui/Toast'
import { timezoneOptions } from '@/features/workspace/timezones'
import { ago, usDate } from '@/lib/format'
import { AdminPage } from '../AdminPage'
import { WorkspacesCard } from './WorkspacesCard'

const twoCards = 'grid items-start gap-5 [&>*]:min-w-0 grid-cols-[repeat(auto-fit,minmax(min(560px,100%),1fr))] max-md:grid-cols-1'

/** Instance admin › Instance & updates (InstanceSettings). */
export function InstancePage() {
  return (
    <AdminPage documentTitle="Instance & updates">
      <div className={twoCards}>
        <InstanceCard />
        <VersionCard />
      </div>
      <WorkspacesCard />
    </AdminPage>
  )
}

function InstanceCard() {
  const q = $api.useQuery('get', '/api/v1/admin/instance')
  return (
    <section className="card" aria-labelledby="inst-h">
      <CardSection title={<span id="inst-h">Instance</span>} caption="Name, logo, public URL and timezone" />
      {q.isError ? (
        <div className="pad">
          <ErrorPanel title="Couldn't load the instance settings" actions={<Button onClick={() => q.refetch()}>Retry</Button>}>
            {errorMessage(q.error)} Nothing was changed.
          </ErrorPanel>
        </div>
      ) : q.isPending ? (
        <div className="pad col gap-3.5" aria-busy="true">
          <Skeleton w="100%" h={56} r={8} />
          <Skeleton w="100%" h={56} r={8} />
          <Skeleton w="60%" h={48} r={8} />
          <Skeleton w="100%" h={110} r={10} />
        </div>
      ) : (
        <InstanceForm key={`${q.data.name}|${q.data.publicUrl}|${q.data.defaultTimezone}`} s={q.data} />
      )}
    </section>
  )
}

function InstanceForm({ s }: { s: Schemas['InstanceSettings'] }) {
  const qc = useQueryClient()
  const [name, setName] = useState(s.name)
  const [url, setUrl] = useState(s.publicUrl || s.configPublicUrl || '')
  const [tz, setTz] = useState(s.defaultTimezone)
  const [error, setError] = useState<{ field?: 'name' | 'url' | 'tz'; message: string }>()
  const [check, setCheck] = useState(s.urlCheck)
  const zones = useMemo(() => timezoneOptions(tz), [tz])
  const save = $api.useMutation('put', '/api/v1/admin/instance')
  const checkUrl = $api.useMutation('post', '/api/v1/admin/instance/check-url')
  const refresh = () =>
    Promise.all([qc.invalidateQueries({ queryKey: ['get', '/api/v1/admin/instance'] }), qc.invalidateQueries({ queryKey: ['meta'] }), qc.invalidateQueries({ queryKey: ['me'] })])
  const fieldOf = (msg: string): 'name' | 'url' | 'tz' | undefined => (/url|address/i.test(msg) ? 'url' : /timezone/i.test(msg) ? 'tz' : /name/i.test(msg) ? 'name' : undefined)
  return (
    <div className="pad col gap-3.5">
      <div className="grid grid-cols-2 gap-3.5 max-md:grid-cols-1">
        <label className="lbl">
          Instance name
          <input
            className="in"
            type="text"
            value={name}
            maxLength={80}
            onChange={(e) => setName(e.target.value)}
            aria-invalid={error?.field === 'name' || undefined}
            aria-describedby={error?.field === 'name' ? 'inst-err' : undefined}
          />
        </label>
        <label className="lbl">
          Default timezone
          <select className="in" value={tz} onChange={(e) => setTz(e.target.value)} aria-invalid={error?.field === 'tz' || undefined}>
            {zones.map((z) => (
              <option key={z.value} value={z.value}>
                {z.label}
              </option>
            ))}
          </select>
        </label>
      </div>
      <label className="lbl">
        Public URL
        <div className="row flex-nowrap gap-2">
          <input
            className="in mono flex-1"
            value={url}
            onChange={(e) => setUrl(e.target.value)}
            inputMode="url"
            autoComplete="off"
            spellCheck={false}
            aria-invalid={error?.field === 'url' || undefined}
            aria-describedby={error?.field === 'url' ? 'inst-err' : 'inst-url-hint'}
          />
          <Button
            loading={checkUrl.isPending}
            onClick={async () => {
              setError(undefined)
              try {
                const r = await checkUrl.mutateAsync({ body: { url: url.trim() || undefined } })
                setCheck(r.urlCheck)
              } catch (e) {
                setError({ field: 'url', message: errorMessage(e) })
              }
            }}
          >
            Check
          </Button>
        </div>
        <span className="k font-medium" id="inst-url-hint">
          Used for SSO redirect URIs, sign-in links and system emails. Provider webhooks, tracking links and public pages use the server&apos;s http.public_url; keep
          both on the same address.
        </span>
      </label>
      {s.configPublicUrl && trimSlash(url) !== trimSlash(s.configPublicUrl) ? (
        <Note tone="warn" label="Server setting">
          The server runs with http.public_url = <span className="mono">{s.configPublicUrl}</span>. Provider webhooks, tracking links and public pages keep using it
          until you set REPUTECAST_HTTP__PUBLIC_URL to the same address and restart.
        </Note>
      ) : null}
      <LogoRow s={s} />
      {check ? <URLCheckPanel check={check} /> : null}
      {error ? (
        <span className="err" id="inst-err" role="alert">
          {error.message}
        </span>
      ) : null}
      <div className="row justify-end">
        <Button
          size="sm"
          variant="primary"
          loading={save.isPending}
          onClick={async () => {
            setError(undefined)
            try {
              await save.mutateAsync({ body: { name: name.trim(), publicUrl: url.trim(), defaultTimezone: tz } })
              await refresh()
              toast({ kind: 'success', title: 'Instance settings saved' })
            } catch (e) {
              const msg = errorMessage(e)
              setError({ field: fieldOf(msg), message: msg })
            }
          }}
        >
          Save
        </Button>
      </div>
    </div>
  )
}

const trimSlash = (u: string) => u.trim().replace(/\/+$/, '')

function URLCheckPanel({ check }: { check: Schemas['URLCheck'] }) {
  const lines: [string, Schemas['CheckLine']][] = [
    ['DNS', check.dns],
    ['TLS', check.tls],
    ['Reachable', check.reachable],
  ]
  const tone = (st: string) => (st === 'ok' ? 'ok' : st === 'warn' ? 'warn' : st === 'bad' ? 'bad' : 'neutral')
  return (
    <div className="card flex flex-col gap-2 bg-[#F9FAFB] px-3.5 py-3" role="status" aria-label="URL check result">
      <span className="k">Public URL checked {ago(check.checkedAt)}</span>
      {lines.map(([label, l]) => (
        <div key={label} className="row gap-2">
          <Badge tone={tone(l.status)}>{label}</Badge>
          <span>{l.detail}</span>
        </div>
      ))}
    </div>
  )
}

function LogoRow({ s }: { s: Schemas['InstanceSettings'] }) {
  const qc = useQueryClient()
  const file = useRef<HTMLInputElement>(null)
  const upload = $api.useMutation('put', '/api/v1/admin/instance/logo')
  const remove = $api.useMutation('delete', '/api/v1/admin/instance/logo')
  const refresh = () => Promise.all([qc.invalidateQueries({ queryKey: ['get', '/api/v1/admin/instance'] }), qc.invalidateQueries({ queryKey: ['meta'] })])
  const pick = async (f: File) => {
    if (!/^image\/(svg\+xml|png)$/.test(f.type)) {
      toast({ kind: 'error', title: 'Upload an SVG or PNG', detail: `${f.name} is ${f.type || 'another type'}` })
      return
    }
    const data = await new Promise<string>((resolve, reject) => {
      const r = new FileReader()
      r.onload = () => resolve(String(r.result))
      r.onerror = () => reject(r.error)
      r.readAsDataURL(f)
    })
    try {
      await upload.mutateAsync({ body: { data } })
      await refresh()
      toast({ kind: 'success', title: 'Logo updated', detail: 'shown on the sign-in page and in team emails' })
    } catch (e) {
      toast({ kind: 'error', title: "Couldn't use that logo", detail: errorMessage(e) })
    }
  }
  return (
    <div className="row items-center gap-4">
      {s.logoSvg ? (
        <img src={`data:image/svg+xml;charset=utf-8,${encodeURIComponent(s.logoSvg)}`} width={48} height={48} alt="Instance logo" className="rounded-[10px]" />
      ) : (
        <span className="inline-flex" title="The ReputeCast mark">
          <LogoMark size={48} />
        </span>
      )}
      <div className="col gap-1">
        <div className="row gap-2">
          <Button size="sm" loading={upload.isPending} onClick={() => file.current?.click()}>
            Upload logo
          </Button>
          <Button
            size="sm"
            loading={remove.isPending}
            disabledReason={s.logoSvg ? undefined : 'Already using the ReputeCast mark'}
            onClick={async () => {
              try {
                await remove.mutateAsync({})
                await refresh()
                toast({ kind: 'success', title: 'Using the ReputeCast mark' })
              } catch (e) {
                toast({ kind: 'error', title: "Couldn't remove the logo", detail: errorMessage(e) })
              }
            }}
          >
            Use ReputeCast mark
          </Button>
          <input
            ref={file}
            type="file"
            accept="image/svg+xml,image/png"
            className="vh"
            tabIndex={-1}
            aria-hidden="true"
            onChange={(e) => {
              const f = e.target.files?.[0]
              e.target.value = ''
              if (f) void pick(f)
            }}
          />
        </div>
        <span className="k">Shown on the sign-in page and in team emails · SVG or PNG, square</span>
      </div>
    </div>
  )
}

function VersionCard() {
  const q = $api.useQuery('get', '/api/v1/admin/updates')
  return (
    <section className="card" aria-labelledby="ver-h">
      <CardSection title={<span id="ver-h">Version &amp; updates</span>} caption="Current version, latest release and how to upgrade" />
      {q.isError ? (
        <div className="pad">
          <ErrorPanel title="Couldn't load the version" actions={<Button onClick={() => q.refetch()}>Retry</Button>}>
            {errorMessage(q.error)}
          </ErrorPanel>
        </div>
      ) : q.isPending ? (
        <div className="pad col gap-3.5" aria-busy="true">
          <Skeleton w="70%" h={60} r={8} />
          <Skeleton w="100%" h={120} r={8} />
          <Skeleton w="100%" h={60} r={8} />
        </div>
      ) : (
        <Version v={q.data} />
      )}
    </section>
  )
}

function Version({ v }: { v: Schemas['InstanceVersion'] }) {
  const qc = useQueryClient()
  const toggle = $api.useMutation('put', '/api/v1/admin/updates')
  const latest = v.latest
  const installedLine = v.release
    ? `Released ${v.builtAt ? usDate(v.builtAt) : 'date unknown'}`
    : `Development build${v.commit ? ` · ${v.commit.slice(0, 7)}` : ''}`
  return (
    <div className="pad col gap-3.5">
      <div className="row items-start gap-[18px]">
        <div className="col flex-[1_1_240px] gap-0.5">
          <span className="k">Installed</span>
          <span className="mono text-[22px] font-bold">{v.installed}</span>
          <span className="k">
            {installedLine}
            {v.checkEnabled && v.checkAllowed && v.checkedAt ? ` · checked for updates ${ago(v.checkedAt)}` : ''}
          </span>
        </div>
        <div className="col flex-[1_1_240px] gap-0.5">
          <span className="k">Latest release</span>
          {latest ? (
            <>
              <span className="row gap-2">
                <span className={`mono text-[22px] font-bold ${v.updateAvailable ? 'text-accent' : ''}`}>{latest.tag}</span>
                {v.updateAvailable ? <Badge tone="ok">Update available</Badge> : latest.demo ? <Badge tone="neutral">Demo data</Badge> : <Badge tone="ok">Up to date</Badge>}
              </span>
              <span className="k">
                Released {usDate(latest.publishedAt)} ·{' '}
                {latest.demo ? (
                  // The demo's sample release isn't on GitHub: say so instead of linking.
                  'sample release from the demo data'
                ) : (
                  <a href={latest.url} target="_blank" rel="noopener noreferrer">
                    changelog on GitHub
                  </a>
                )}
              </span>
            </>
          ) : (
            <>
              <span className="mono text-[22px] font-bold text-ink-3">–</span>
              <span className="k">{v.checkEnabled && v.checkAllowed ? 'Not checked yet' : 'The update check is off'}</span>
            </>
          )}
        </div>
      </div>
      {v.checkError && v.checkEnabled ? (
        <Note tone="warn" label="Check failed">
          {v.checkError} The next check runs tomorrow; nothing else is affected.
        </Note>
      ) : null}
      {latest && latest.notes.length > 0 ? (
        <div className="note i">
          <span className="col gap-1.5">
            <span className="font-semibold">What&apos;s new in {latest.tag}</span>
            <ul className="m-0 flex list-disc flex-col gap-[3px] pl-[18px]">
              {latest.notes.map((n) => (
                <li key={n}>{n}</li>
              ))}
            </ul>
            {latest.note ? <span className="k">{latest.note}</span> : null}
          </span>
        </div>
      ) : null}
      <div className="col gap-1.5">
        <span className="text-[12px] font-semibold text-ink-2">Upgrade on the server</span>
        <pre className="code">{v.upgradeCommand}</pre>
        <span className="k">
          Back up Postgres first: <span className="mono">docker compose exec postgres pg_dump -U reputecast -Fc reputecast &gt; reputecast.dump</span>. Rollback: pin the
          image to {v.release ? v.installed : 'the version you ran before'} and restart.
        </span>
      </div>
      {/* eslint-disable-next-line jsx-a11y/label-has-associated-control -- the label wraps its control; the title and description inside are its text */}
      <label className="row flex-nowrap gap-3 border-t border-line-2 pt-1.5">
        <input
          type="checkbox"
          className="tg"
          role="switch"
          checked={v.checkEnabled}
          disabled={!v.checkAllowed || toggle.isPending}
          onChange={async (e) => {
            try {
              await toggle.mutateAsync({ body: { enabled: e.target.checked } })
              await qc.invalidateQueries({ queryKey: ['get', '/api/v1/admin/updates'] })
              toast({ kind: 'success', title: e.target.checked ? 'Update check on' : 'Update check off', detail: e.target.checked ? 'checks github.com once a day' : 'no outbound calls' })
            } catch (err) {
              toast({ kind: 'error', title: "Couldn't change the update check", detail: errorMessage(err) })
            }
          }}
        />
        <span className="col gap-px">
          <span className="font-semibold">Check GitHub releases once a day</span>
          <span className="k">
            {v.checkAllowed
              ? 'Fetches only the latest version number from github.com. ReputeCast never sends usage data, telemetry or crash reports anywhere; this is the only outbound call it makes on its own.'
              : 'Turned off in the server configuration (updates.check = false). ReputeCast never sends usage data, telemetry or crash reports anywhere.'}
          </span>
        </span>
      </label>
    </div>
  )
}
