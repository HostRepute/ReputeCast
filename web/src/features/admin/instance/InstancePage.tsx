import { useQueryClient } from '@tanstack/react-query'
import { useMemo, useRef, useState, type ReactNode } from 'react'
import { $api, ApiError, errorMessage, type Schemas } from '@/api/client'
import { LogoMark } from '@/components/icons/Icon'
import { Badge } from '@/components/ui/Badge'
import { Button } from '@/components/ui/Button'
import { CardSection } from '@/components/ui/Card'
import { ErrorPanel, Skeleton } from '@/components/ui/Feedback'
import { Note } from '@/components/ui/Note'
import { toast } from '@/components/ui/Toast'
import { timezoneOptions } from '@/features/workspace/timezones'
import { copyText } from '@/features/access/parts'
import { InstanceLanguageFields } from '@/features/language/InstanceLanguageFields'
import { m as lm } from '@/features/language/messages'
import { T, useT } from '@/i18n'
import { common } from '@/i18n/common'
import { ago, usDate } from '@/lib/format'
import { AdminPage } from '../AdminPage'
import { RelayAdminCard } from '@/features/transactional/RelayAdminCard'
import { WorkspacesCard } from './WorkspacesCard'
import { AdminAgencyCard } from '@/features/agency/AdminParts'
import { m } from './messages'

const twoCards = 'grid items-start gap-5 [&>*]:min-w-0 grid-cols-[repeat(auto-fit,minmax(min(560px,100%),1fr))] max-md:grid-cols-1'

const mono = (c: ReactNode) => <span className="mono">{c}</span>

/** Instance admin › Instance & updates (InstanceSettings). */
export function InstancePage() {
  const t = useT()
  return (
    <AdminPage documentTitle={t(m.documentTitle)}>
      <div className={twoCards}>
        <InstanceCard />
        <VersionCard />
        <RelayAdminCard />
      </div>
      {/* Agency mode (agency accounts). */}
      <AdminAgencyCard />
      <WorkspacesCard />
    </AdminPage>
  )
}

function InstanceCard() {
  const t = useT()
  const q = $api.useQuery('get', '/api/v1/admin/instance')
  // What the last save that moved the public URL did to provider webhooks
  // (the form remounts after a save, so it lives here).
  const [moved, setMoved] = useState<Schemas['PublicURLWebhooks']>()
  return (
    <section className="card" aria-labelledby="inst-h">
      <CardSection title={<span id="inst-h">{t(m.instanceTitle)}</span>} caption={t(lm.instanceCaption)} />
      {q.isError ? (
        <div className="pad">
          <ErrorPanel title={t(m.loadFailed)} actions={<Button onClick={() => q.refetch()}>{t(common.retry)}</Button>}>
            {errorMessage(q.error)} {t(m.nothingChanged)}
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
        <InstanceForm
          key={`${q.data.name}|${q.data.publicUrl}|${q.data.defaultTimezone}|${q.data.defaultLanguage}|${q.data.defaultRegionFormat}`}
          s={q.data}
          moved={moved}
          onMoved={setMoved}
        />
      )}
    </section>
  )
}

function InstanceForm({
  s,
  moved,
  onMoved,
}: {
  s: Schemas['InstanceSettings']
  moved?: Schemas['PublicURLWebhooks']
  onMoved: (m: Schemas['PublicURLWebhooks'] | undefined) => void
}) {
  const t = useT()
  const qc = useQueryClient()
  const [name, setName] = useState(s.name)
  const [url, setUrl] = useState(s.publicUrl || s.configPublicUrl || '')
  const [tz, setTz] = useState(s.defaultTimezone)
  const [language, setLanguage] = useState(s.defaultLanguage ?? 'en')
  const [region, setRegion] = useState(s.defaultRegionFormat ?? 'standard')
  const [error, setError] = useState<{ field?: 'name' | 'url' | 'tz'; message: string }>()
  const [check, setCheck] = useState(s.urlCheck)
  const zones = useMemo(() => timezoneOptions(tz), [tz])
  const save = $api.useMutation('put', '/api/v1/admin/instance')
  const checkUrl = $api.useMutation('post', '/api/v1/admin/instance/check-url')
  const refresh = () =>
    Promise.all([qc.invalidateQueries({ queryKey: ['get', '/api/v1/admin/instance'] }), qc.invalidateQueries({ queryKey: ['meta'] }), qc.invalidateQueries({ queryKey: ['me'] })])
  // The problem names its field; its sentence reads in the app language.
  const fieldOf = (e: unknown): 'name' | 'url' | 'tz' | undefined => {
    const loc = e instanceof ApiError ? e.problem.errors?.[0]?.location : undefined
    return loc === 'body.publicUrl' || loc === 'body.url' ? 'url' : loc === 'body.defaultTimezone' ? 'tz' : loc === 'body.name' ? 'name' : undefined
  }
  return (
    <div className="pad col gap-3.5">
      <div className="grid grid-cols-2 gap-3.5 max-md:grid-cols-1">
        <label className="lbl">
          {t(m.nameLabel)}
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
          {t(m.timezoneLabel)}
          <select className="in" value={tz} onChange={(e) => setTz(e.target.value)} aria-invalid={error?.field === 'tz' || undefined}>
            {zones.map((z) => (
              <option key={z.value} value={z.value}>
                {z.label}
              </option>
            ))}
          </select>
        </label>
      </div>
      <InstanceLanguageFields language={language} region={region} onLanguage={setLanguage} onRegion={setRegion} />
      <label className="lbl">
        {t(m.publicUrlLabel)}
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
            {t(m.check)}
          </Button>
        </div>
        <span className="k font-medium" id="inst-url-hint">
          {t(m.publicUrlHint)}
        </span>
      </label>
      {s.configPublicUrl && trimSlash(url) !== trimSlash(s.configPublicUrl) ? (
        <Note tone="warn" label={t(m.serverSetting)}>
          <T msg={m.serverSettingBody} values={{ setting: 'http.public_url', url: s.configPublicUrl, env: 'REPUTECAST_HTTP__PUBLIC_URL', mono }} />
        </Note>
      ) : null}
      {moved && (moved.reregistering > 0 || moved.manual.length > 0) ? <WebhookMove moved={moved} /> : null}
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
              const r = await save.mutateAsync({
                body: {
                  name: name.trim(),
                  publicUrl: url.trim(),
                  defaultTimezone: tz,
                  defaultLanguage: language as Schemas['InstanceSettingsInput']['defaultLanguage'],
                  defaultRegionFormat: region as Schemas['InstanceSettingsInput']['defaultRegionFormat'],
                },
              })
              // A save that moved nothing clears the last move's notice.
              onMoved(r.webhooks)
              await refresh()
              toast({ kind: 'success', title: t(m.saved) })
            } catch (e) {
              setError({ field: fieldOf(e), message: errorMessage(e) })
            }
          }}
        >
          {t(common.save)}
        </Button>
      </div>
    </div>
  )
}

const trimSlash = (u: string) => u.trim().replace(/\/+$/, '')

/** What moving the public URL did to provider webhooks. */
function WebhookMove({ moved }: { moved: Schemas['PublicURLWebhooks'] }) {
  const t = useT()
  const n = moved.reregistering
  return (
    <Note tone={moved.manual.length > 0 ? 'warn' : 'ok'} label={t(m.webhooksLabel)}>
      {n > 0 ? <>{t(m.reregistering, { count: n })} </> : null}
      {moved.manual.length > 0 ? (
        <>
          {t(m.manualProviders, { count: moved.manual.length })}
          <ul className="mt-1.5 flex flex-col gap-1.5">
            {moved.manual.map((w) => (
              <li key={w.url} className="flex flex-wrap items-center gap-x-2 gap-y-1">
                <span className="font-semibold">{w.provider}</span>
                <span className="k">{w.workspace}</span>
                <span className="mono min-w-0 break-all">{w.url}</span>
                <Button
                  size="sm"
                  onClick={async () => {
                    const ok = await copyText(w.url)
                    toast(ok ? { kind: 'success', title: t(m.webhookCopied) } : { kind: 'error', title: t(m.copyFailed), detail: t(m.copyFailedDetail) })
                  }}
                >
                  {t(common.copy)}
                </Button>
              </li>
            ))}
          </ul>
        </>
      ) : null}
    </Note>
  )
}

function URLCheckPanel({ check }: { check: Schemas['URLCheck'] }) {
  const t = useT()
  const lines: [string, Schemas['CheckLine']][] = [
    ['DNS', check.dns],
    ['TLS', check.tls],
    [t(m.checkReachable), check.reachable],
  ]
  const tone = (st: string) => (st === 'ok' ? 'ok' : st === 'warn' ? 'warn' : st === 'bad' ? 'bad' : 'neutral')
  return (
    <div className="card flex flex-col gap-2 bg-ground-3 px-3.5 py-3" role="status" aria-label={t(m.urlCheckLabel)}>
      <span className="k">{t(m.urlChecked, { ago: ago(check.checkedAt) })}</span>
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
  const t = useT()
  const qc = useQueryClient()
  const file = useRef<HTMLInputElement>(null)
  const upload = $api.useMutation('put', '/api/v1/admin/instance/logo')
  const remove = $api.useMutation('delete', '/api/v1/admin/instance/logo')
  const refresh = () => Promise.all([qc.invalidateQueries({ queryKey: ['get', '/api/v1/admin/instance'] }), qc.invalidateQueries({ queryKey: ['meta'] })])
  const pick = async (f: File) => {
    if (!/^image\/(svg\+xml|png)$/.test(f.type)) {
      toast({ kind: 'error', title: t(m.logoWrongType), detail: t(m.logoWrongTypeDetail, { name: f.name, hasType: f.type ? 'yes' : 'no', type: f.type }) })
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
      toast({ kind: 'success', title: t(m.logoUpdated), detail: t(m.logoUpdatedDetail) })
    } catch (e) {
      toast({ kind: 'error', title: t(m.logoFailed), detail: errorMessage(e) })
    }
  }
  return (
    <div className="row items-center gap-4">
      {s.logoSvg ? (
        <img src={`data:image/svg+xml;charset=utf-8,${encodeURIComponent(s.logoSvg)}`} width={48} height={48} alt={t(m.logoAlt)} className="rounded-[10px]" />
      ) : (
        <span className="inline-flex" title={t(m.markTitle)}>
          <LogoMark size={48} />
        </span>
      )}
      <div className="col gap-1">
        <div className="row gap-2">
          <Button size="sm" loading={upload.isPending} onClick={() => file.current?.click()}>
            {t(m.uploadLogo)}
          </Button>
          <Button
            size="sm"
            loading={remove.isPending}
            disabledReason={s.logoSvg ? undefined : t(m.alreadyMark)}
            onClick={async () => {
              try {
                await remove.mutateAsync({})
                await refresh()
                toast({ kind: 'success', title: t(m.usingMark) })
              } catch (e) {
                toast({ kind: 'error', title: t(m.removeLogoFailed), detail: errorMessage(e) })
              }
            }}
          >
            {t(m.useMark)}
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
        <span className="k">{t(m.logoHint)}</span>
      </div>
    </div>
  )
}

function VersionCard() {
  const t = useT()
  const q = $api.useQuery('get', '/api/v1/admin/updates')
  return (
    <section className="card" aria-labelledby="ver-h">
      <CardSection title={<span id="ver-h">{t(m.versionTitle)}</span>} caption={t(m.versionCaption)} />
      {q.isError ? (
        <div className="pad">
          <ErrorPanel title={t(m.versionLoadFailed)} actions={<Button onClick={() => q.refetch()}>{t(common.retry)}</Button>}>
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
  const t = useT()
  const qc = useQueryClient()
  const toggle = $api.useMutation('put', '/api/v1/admin/updates')
  const latest = v.latest
  const installedLine = v.release
    ? v.builtAt
      ? t(m.releasedOn, { date: usDate(v.builtAt) })
      : t(m.releasedUnknown)
    : v.commit
      ? t(m.developmentBuildCommit, { commit: v.commit.slice(0, 7) })
      : t(m.developmentBuild)
  return (
    <div className="pad col gap-3.5">
      <div className="row items-start gap-[18px]">
        <div className="col flex-[1_1_240px] gap-0.5">
          <span className="k">{t(m.installed)}</span>
          <span className="mono text-[22px] font-bold">{v.installed}</span>
          <span className="k">
            {installedLine}
            {v.checkEnabled && v.checkAllowed && v.checkedAt ? ` · ${t(m.checkedForUpdates, { ago: ago(v.checkedAt) })}` : ''}
          </span>
        </div>
        <div className="col flex-[1_1_240px] gap-0.5">
          <span className="k">{t(m.latestRelease)}</span>
          {latest ? (
            <>
              <span className="row gap-2">
                <span className={`mono text-[22px] font-bold ${v.updateAvailable ? 'text-accent' : ''}`}>{latest.tag}</span>
                {v.updateAvailable ? (
                  <Badge tone="ok">{t(m.updateAvailable)}</Badge>
                ) : latest.demo ? (
                  <Badge tone="neutral">{t(m.demoData)}</Badge>
                ) : (
                  <Badge tone="ok">{t(m.upToDate)}</Badge>
                )}
              </span>
              <span className="k">
                {latest.demo ? (
                  // The demo's sample release isn't on GitHub: say so instead of linking.
                  t(m.releasedDemo, { date: usDate(latest.publishedAt) })
                ) : (
                  <T
                    msg={m.releasedChangelog}
                    values={{
                      date: usDate(latest.publishedAt),
                      link: (c) => (
                        <a href={latest.url} target="_blank" rel="noopener noreferrer">
                          {c}
                        </a>
                      ),
                    }}
                  />
                )}
              </span>
            </>
          ) : (
            <>
              <span className="mono text-[22px] font-bold text-ink-3">–</span>
              <span className="k">{v.checkEnabled && v.checkAllowed ? t(m.notChecked) : t(m.checkOff)}</span>
            </>
          )}
        </div>
      </div>
      {v.checkError && v.checkEnabled ? (
        <Note tone="warn" label={t(m.checkFailed)}>
          {v.checkError} {t(m.nextCheck)}
        </Note>
      ) : null}
      {latest && latest.notes.length > 0 ? (
        <div className="note i">
          <span className="col gap-1.5">
            <span className="font-semibold">{t(m.whatsNew, { tag: latest.tag })}</span>
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
        <span className="text-[12px] font-semibold text-ink-2">{t(m.upgradeTitle)}</span>
        <pre className="code">{v.upgradeCommand}</pre>
        <span className="k">
          <T
            msg={m.backupNote}
            values={{
              command: 'docker compose exec postgres pg_dump -U reputecast -Fc reputecast > reputecast.dump',
              release: v.release ? 'yes' : 'no',
              version: v.installed,
              mono,
            }}
          />
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
              toast({
                kind: 'success',
                title: e.target.checked ? t(m.updateCheckOn) : t(m.updateCheckOff),
                detail: e.target.checked ? t(m.updateCheckOnDetail) : t(m.updateCheckOffDetail),
              })
            } catch (err) {
              toast({ kind: 'error', title: t(m.updateCheckFailed), detail: errorMessage(err) })
            }
          }}
        />
        <span className="col gap-px">
          <span className="font-semibold">{t(m.updateCheckToggle)}</span>
          <span className="k">{v.checkAllowed ? t(m.updateCheckAllowed) : t(m.updateCheckDisallowed, { setting: 'updates.check = false' })}</span>
        </span>
      </label>
    </div>
  )
}
