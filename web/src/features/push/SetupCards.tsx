import { Link, useNavigate } from '@tanstack/react-router'
import { useState, type ReactNode, type RefObject } from 'react'
import { $api, errorMessage } from '@/api/client'
import { Badge } from '@/components/ui/Badge'
import { Button, ButtonAnchor, ButtonLink } from '@/components/ui/Button'
import { Table, Td, Th } from '@/components/ui/Table'
import { toast } from '@/components/ui/Toast'
import { T, useT } from '@/i18n'
import { count, dayMonthDate, dayMonthYear, orList, pct } from '@/lib/format'
import { useWorkspace } from '@/lib/session'
import { bareHost, browserName, copyText, styleName, useInvalidatePush, type PushPromptCard, type PushPromptConfig, type PushSetup, type PushSite } from './common'
import { m } from './messages'
import { RotateDialog } from './RotateDialog'
import { useWhen } from './SetupPage'

const SITE_COLORS = ['#2a78d6', '#eb6834', '#1baf7a', '#eda100', '#8b94a3']

export function InfoIcon({ color = '#4B5563' }: { color?: string }) {
  return (
    <svg width="16" height="16" viewBox="0 0 16 16" fill="none" stroke={color} strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
      <circle cx="8" cy="8" r="6.2" />
      <path d="M8 7.2v3.8M8 5v.1" />
    </svg>
  )
}

function AlertIcon({ color }: { color: string }) {
  return (
    <svg width="16" height="16" viewBox="0 0 16 16" fill="none" stroke={color} strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
      <circle cx="8" cy="8" r="6.2" />
      <path d="M8 4.8v3.6M8 10.9v.1" />
    </svg>
  )
}

function GlobeIcon() {
  return (
    <svg width="15" height="15" viewBox="0 0 16 16" fill="none" stroke="#4B5563" strokeWidth="1.6" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
      <circle cx="8" cy="8" r="6.2" />
      <path d="M1.8 8h12.4M8 1.8c1.8 1.7 2.7 3.8 2.7 6.2S9.8 12.5 8 14.2C6.2 12.5 5.3 10.4 5.3 8S6.2 3.5 8 1.8z" />
    </svg>
  )
}

function EyeOffIcon() {
  return (
    <svg width="14" height="14" viewBox="0 0 16 16" fill="none" stroke="#5B6472" strokeWidth="1.6" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
      <path d="M2 2l12 12M6.6 6.6a2 2 0 0 0 2.8 2.8M4.2 4.3C2.9 5.2 2 6.5 1.5 8c1.1 2.9 3.6 4.5 6.5 4.5 1.3 0 2.5-.3 3.6-.9M8 3.5c2.9 0 5.4 1.6 6.5 4.5-.3.9-.8 1.7-1.4 2.4" />
    </svg>
  )
}

const BAD_WORKER = new Set(['missing', 'unreachable', 'not_a_worker'])

/** Sites whose service worker needs attention: missing, unreachable, or an old version. */
export function WorkerAlerts({ data }: { data: PushSetup }) {
  const sites = data.sites.filter((s) => s.kind === 'snippet' && (BAD_WORKER.has(s.worker.status) || s.worker.status === 'old_version'))
  if (!sites.length) return null
  return (
    <>
      {sites.map((s) => (
        <WorkerAlert key={s.id} site={s} data={data} />
      ))}
    </>
  )
}

function WorkerAlert({ site, data }: { site: PushSite; data: PushSetup }) {
  const t = useT()
  const when = useWhen()
  const w = site.worker
  const bad = BAD_WORKER.has(w.status)
  const status = w.status === 'unreachable' ? m.workerUnreachable : w.status === 'not_a_worker' ? m.workerNotAWorker : m.workerMissing
  const fileURL = data.snippet.workerUrl
  return (
    <section className={`card px-site-alert ${bad ? 'px-bad' : 'px-warn'}`} aria-label={site.host}>
      <div className="row1">
        <span className="mono font-semibold">{site.host}</span>
        {bad ? (
          <Badge tone="bad" dot>
            {t(status)}
          </Badge>
        ) : (
          <Badge tone="warn">{t(m.workerOld, { version: w.version ?? 0, current: w.current })}</Badge>
        )}
        <span className="k">
          {bad
            ? w.checkedAt
              ? t(m.checkedLine, { date: when(w.checkedAt), count: site.subscribers })
              : null
            : w.checkedAt
              ? t(m.foundLine, { date: when(w.checkedAt), scope: w.scope || '/' })
              : null}
        </span>
      </div>
      <div className={bad ? 'note b' : 'note w'} role={bad ? 'alert' : 'note'}>
        <AlertIcon color={bad ? '#B91C1C' : '#B45309'} />
        <div className="px-note-body">
          <b>
            {bad
              ? w.status === 'not_a_worker'
                ? t(m.notWorkerAlert, { url: w.url })
                : w.httpStatus
                  ? t(m.missingAlert, { url: w.url, status: w.httpStatus })
                  : t(m.missingNoStatus, { url: w.url })
              : t(m.oldTitle, { host: site.host })}
          </b>
          <div>
            {bad ? t(m.missingHeld, { host: site.host, count: site.subscribers }) : t(m.oldText, { version: w.version ?? 0, host: site.host })}
            {data.canManage ? (
              <div className="px-note-acts">
                <ButtonAnchor href={fileURL} download={data.snippet.workerFile} newTab={false} size="sm" icon="download">
                  {bad ? t(m.downloadFile, { file: data.snippet.workerFile }) : t(m.downloadVersion, { version: data.snippet.workerVersion })}
                </ButtonAnchor>
                <CheckButton site={site} variant={bad ? 'primary' : 'default'} label={t(m.checkAgain)} />
              </div>
            ) : null}
          </div>
        </div>
      </div>
    </section>
  )
}

/** Fetches the site's service worker again and says what it found. */
function CheckButton({ site, variant = 'default', label, icon }: { site: PushSite; variant?: 'default' | 'primary'; label: string; icon?: boolean }) {
  const t = useT()
  const { slug } = useWorkspace()
  const invalidate = useInvalidatePush()
  const check = $api.useMutation('post', '/api/v1/workspaces/{workspace}/push/sites/{siteId}/check')
  const run = () =>
    check.mutate(
      { params: { path: { workspace: slug, siteId: site.id } } },
      {
        onSuccess: (s) => {
          void invalidate()
          const ok = s.worker.status === 'found' || s.worker.status === 'built_in'
          const problem =
            s.worker.status === 'old_version'
              ? t(m.workerOld, { version: s.worker.version ?? 0, current: s.worker.current })
              : s.worker.status === 'unreachable'
                ? t(m.workerUnreachable)
                : s.worker.status === 'not_a_worker'
                  ? t(m.workerNotAWorker)
                  : t(m.workerMissing)
          toast({
            kind: ok ? 'success' : 'error',
            title: ok ? t(m.checkedOk, { host: s.host, version: s.worker.version ?? s.worker.current }) : t(m.checkedBad, { host: s.host, problem }),
          })
        },
        onError: (e) => toast({ kind: 'error', title: errorMessage(e) }),
      },
    )
  return (
    <Button size="sm" variant={variant} icon={icon ? 'refresh' : undefined} loading={check.isPending} busyLabel={t(m.checking)} onClick={run}>
      {label}
    </Button>
  )
}

/** The VAPID keys: the public key, the sealed private key, who made them and the contact address. */
export function KeysCard({ data }: { data: PushSetup }) {
  const t = useT()
  const [rotate, setRotate] = useState(false)
  const keys = data.keys
  if (!keys) return null
  return (
    <section className="card">
      <div className="px-head">
        <div>
          <h2 className="h2">{t(m.keysTitle)}</h2>
          <span className="k">{t(m.keysIntro)}</span>
        </div>
        {data.canManage ? (
          <div>
            <Button size="sm" variant="danger" onClick={() => setRotate(true)}>
              {t(m.rotateKeys)}
            </Button>
          </div>
        ) : null}
      </div>
      <div className="pad flex flex-col gap-3.5">
        <dl className="kv">
          <dt>{t(m.publicKey)}</dt>
          <dd>
            <div className="px-key">
              <span className="mono">{keys.publicKey}</span>
              <Button
                size="sm"
                iconOnly
                icon="copy"
                aria-label={t(m.copyPublicKey)}
                className="flex-none"
                onClick={() => void copyText(keys.publicKey).then((ok) => ok && toast({ kind: 'success', title: t(m.copied) }))}
              />
            </div>
          </dd>
          <dt>{t(m.privateKey)}</dt>
          <dd>
            <span className="px-mask">
              <EyeOffIcon />
              ••••••
              <Badge tone="neutral" size="sm">
                {t(m.sealed)}
              </Badge>
            </span>
          </dd>
          <dt>{t(m.created)}</dt>
          <dd>{t(m.createdBy, { date: dayMonthYear(keys.createdAt), name: keys.createdBy })}</dd>
          <dt>{t(m.contact)}</dt>
          <dd>
            <ContactValue data={data} contact={keys.contact} />
          </dd>
        </dl>
        <div className="note n" role="note">
          <InfoIcon />
          <div>{t(m.rotateNote, { count: data.subscribers.total })}</div>
        </div>
      </div>
      {rotate ? <RotateDialog open={rotate} onOpenChange={setRotate} /> : null}
    </section>
  )
}

function ContactValue({ data, contact }: { data: PushSetup; contact: string }) {
  const t = useT()
  const { slug } = useWorkspace()
  const invalidate = useInvalidatePush()
  const [editing, setEditing] = useState(false)
  const [value, setValue] = useState(contact)
  const save = $api.useMutation('put', '/api/v1/workspaces/{workspace}/push/contact')
  if (!editing) {
    return (
      <>
        <span className="mono">{contact}</span>
        {data.canManage ? (
          <button
            type="button"
            className="px-lnk ml-1.5"
            onClick={() => {
              setValue(contact)
              setEditing(true)
            }}
          >
            {t(m.edit)}
          </button>
        ) : null}
      </>
    )
  }
  const submit = () =>
    save.mutate(
      { params: { path: { workspace: slug } }, body: { contact: value.trim() } },
      {
        onSuccess: () => {
          void invalidate()
          setEditing(false)
          toast({ kind: 'success', title: t(m.contactSaved) })
        },
      },
    )
  return (
    <form
      className="flex flex-col gap-1.5"
      onSubmit={(e) => {
        e.preventDefault()
        submit()
      }}
    >
      <label className="vh" htmlFor="push-contact">
        {t(m.contactLabel)}
      </label>
      <div className="flex flex-wrap items-center gap-2">
        <input
          id="push-contact"
          className="in mono min-w-0 flex-1"
          value={value}
          onChange={(e) => setValue(e.target.value)}
          aria-describedby="push-contact-hint"
          aria-invalid={save.isError || undefined}
          autoFocus
        />
        <Button size="sm" variant="primary" type="submit" loading={save.isPending}>
          {t(m.save)}
        </Button>
        <Button size="sm" onClick={() => setEditing(false)}>
          {t(m.cancel)}
        </Button>
      </div>
      {save.isError ? (
        <span className="err" role="alert">
          {errorMessage(save.error)}
        </span>
      ) : (
        <span className="k" id="push-contact-hint">
          {t(m.contactHint)}
        </span>
      )}
    </form>
  )
}

/** The frequency cap and quiet hours: guards for every notification. */
export function GuardsCard({ data }: { data: PushSetup }) {
  const t = useT()
  const { slug } = useWorkspace()
  const invalidate = useInvalidatePush()
  // The typed cap, until the saved one changes under it.
  const [cap, setCap] = useState(String(data.cap))
  const [savedCap, setSavedCap] = useState(data.cap)
  if (savedCap !== data.cap) {
    setSavedCap(data.cap)
    setCap(String(data.cap))
  }
  const save = $api.useMutation('put', '/api/v1/workspaces/{workspace}/push/settings')
  const put = (frequencyCap: number, quietHours: boolean) =>
    save.mutate(
      { params: { path: { workspace: slug } }, body: { frequencyCap, quietHours } },
      {
        onSuccess: () => void invalidate(),
        onError: (e) => {
          setCap(String(data.cap))
          toast({ kind: 'error', title: errorMessage(e) })
        },
      },
    )
  const n = Number(cap)
  const commitCap = () => {
    if (!Number.isInteger(n) || n < 1 || n > 24) {
      setCap(String(data.cap))
      return
    }
    if (n !== data.cap) put(n, data.quiet)
  }
  const ro = !data.canManage
  return (
    <section className="card">
      <div className="sec">
        <div className="flex min-w-0 flex-col gap-0.5">
          <h2 className="h2">{t(m.guardsTitle)}</h2>
          <span className="k">{t(m.guardsIntro)}</span>
        </div>
      </div>
      <div className="pad flex flex-col py-1.5">
        <div className="px-mrow flex-nowrap max-md:flex-wrap">
          <span className="tx">
            <b>{t(m.capTitle, { n: data.cap })}</b>
            <span className="k">{t(m.capText)}</span>
          </span>
          <span className="flex flex-none items-center gap-2">
            <input
              className="in w-14 text-center"
              inputMode="numeric"
              value={cap}
              aria-label={t(m.capLabel)}
              disabled={ro}
              onChange={(e) => setCap(e.target.value.replace(/[^0-9]/g, '').slice(0, 2))}
              onBlur={commitCap}
              onKeyDown={(e) => {
                if (e.key === 'Enter') commitCap()
              }}
            />
            <span className="k">{t(m.aDay)}</span>
          </span>
        </div>
        <div className="px-mrow flex-nowrap max-md:flex-wrap">
          <span className="tx">
            <b>{t(m.quietTitle, { window: data.window })}</b>
            <span className="k">{t(m.quietText)}</span>
          </span>
          <input
            type="checkbox"
            role="switch"
            className="tg flex-none"
            checked={data.quiet}
            disabled={ro || save.isPending}
            aria-label={t(m.quietLabel)}
            onChange={(e) => put(data.cap, e.target.checked)}
          />
        </div>
        <div className="px-mrow">
          <span className="k">{t(m.sameWindow)}</span>
          {ro ? null : (
            // The window lives in SMS › Routing & rules (X21).
            <a href={`/w/${slug}/providers/sms/rules`} className="px-lnk">
              {t(m.editWindow)}
            </a>
          )}
        </div>
      </div>
    </section>
  )
}

/** The sites table and, for managers, Add a site. */
export function SitesCard({ data, addRef }: { data: PushSetup; addRef: RefObject<HTMLInputElement | null> }) {
  const t = useT()
  return (
    <section className="card">
      <div className="sec">
        <div className="flex min-w-0 flex-col gap-0.5">
          <h2 className="h2">{t(m.sitesTitle, { count: data.sites.length })}</h2>
          <span className="k">{t(m.sitesIntro)}</span>
        </div>
      </div>
      {data.sites.length ? (
        <Table aria-label={t(m.sitesTitle, { count: data.sites.length })}>
          <thead>
            <tr>
              <Th>{t(m.colSite)}</Th>
              <Th>{t(m.colSetup)}</Th>
              <Th>{t(m.colWorker)}</Th>
              <Th align="right">{t(m.colSubscribers)}</Th>
              <Th>{t(m.colPrompt)}</Th>
              <Th>
                <span className="vh">{t(m.colActions)}</span>
              </Th>
            </tr>
          </thead>
          <tbody>
            {data.sites.map((s) => (
              <SiteRow key={s.id} site={s} data={data} />
            ))}
          </tbody>
        </Table>
      ) : null}
      {data.canManage ? <AddSite data={data} addRef={addRef} /> : null}
      <div className="px-foot">
        <span className="k">{t(m.refusedNote)}</span>
      </div>
    </section>
  )
}

function SiteRow({ site, data }: { site: PushSite; data: PushSetup }) {
  const t = useT()
  const when = useWhen()
  const prompt = site.prompts.find((p) => p.status === 'live') ?? site.prompts[0]
  const builtIn = site.kind === 'publication'
  return (
    <tr>
      <Td>
        <div className="px-site">
          <span className="ic">
            <GlobeIcon />
          </span>
          <div className="px-two">
            <span className="host">{site.host}</span>
            <span className="k">
              {builtIn
                ? site.label || t(m.publicationSite)
                : site.snippetSeenAt
                  ? t(m.snippetInstalled, { label: site.label || site.host, date: dayMonthDate(site.snippetSeenAt) })
                  : t(m.snippetNotSeen, { label: site.label || site.host })}
            </span>
          </div>
        </div>
      </Td>
      <Td>
        {builtIn ? (
          <Badge tone="ok" dot>
            {t(m.builtIn)}
          </Badge>
        ) : (
          <Badge tone="neutral">{t(m.snippet)}</Badge>
        )}
      </Td>
      <Td>
        <WorkerCell site={site} when={when} />
      </Td>
      <Td num align="right">
        <span className="font-semibold">{count(site.subscribers)}</span>
      </Td>
      <Td>
        {prompt ? (
          <span className="px-two">
            <span>{t(styleName[prompt.style])}</span>
            <span className="k">
              {prompt.status === 'live' && prompt.liveSince
                ? t(m.promptLiveSince, { date: dayMonthDate(prompt.liveSince) })
                : prompt.status === 'paused'
                  ? t(m.promptPaused)
                  : t(m.promptDraft)}
            </span>
          </span>
        ) : (
          <span className="k">{t(m.promptNone)}</span>
        )}
      </Td>
      <Td align="right">
        <span className="inline-flex gap-1.5">
          {!builtIn && data.canManage ? <CheckButton site={site} label={t(m.check)} icon /> : null}
          {prompt ? (
            <OpenPrompt formId={prompt.formId} label={t(m.openPrompt)} />
          ) : data.canEditPrompts && data.canManage ? (
            <NewPromptButton siteId={site.id} label={t(m.newPrompt)} />
          ) : null}
        </span>
      </Td>
    </tr>
  )
}

function WorkerCell({ site, when }: { site: PushSite; when: (d: string) => string }) {
  const t = useT()
  const w = site.worker
  switch (w.status) {
    case 'built_in':
      return (
        <span className="px-two">
          <span>{t(m.servedByUs)}</span>
          <span className="k">{t(m.alwaysCurrent)}</span>
        </span>
      )
    case 'found':
      return (
        <span className="px-two">
          <span>{w.checkedAt ? t(m.workerFound, { date: when(w.checkedAt), version: w.version ?? w.current, scope: w.scope || '/' }) : t(m.workerUnchecked)}</span>
          <span className="mono k">{w.url}</span>
        </span>
      )
    case 'old_version':
      return (
        <span className="px-two">
          <Badge tone="warn" className="self-start">
            {t(m.workerOld, { version: w.version ?? 0, current: w.current })}
          </Badge>
          <span className="mono k">{w.url}</span>
        </span>
      )
    case 'unchecked':
      return (
        <span className="px-two">
          <span>{t(m.workerUnchecked)}</span>
          <span className="k">{t(m.workerUncheckedLine)}</span>
        </span>
      )
    default:
      return (
        <span className="px-two">
          <Badge tone="bad" dot className="self-start">
            {t(w.status === 'unreachable' ? m.workerUnreachable : w.status === 'not_a_worker' ? m.workerNotAWorker : m.workerMissing)}
          </Badge>
          <span className="mono k">{w.url}</span>
        </span>
      )
  }
}

/** Opens a push prompt in the designer (Forms & pages › Forms). */
function OpenPrompt({ formId, label, variant }: { formId: string; label: string; variant?: 'default' }) {
  const { slug } = useWorkspace()
  return (
    <ButtonLink size="sm" variant={variant} to="/w/$workspace/forms" params={{ workspace: slug }} search={{ prompt: formId }}>
      {label}
    </ButtonLink>
  )
}

/** Creates a push prompt (for a site, when given) and opens it in the designer. */
function NewPromptButton({ siteId, label, icon }: { siteId?: string; label: string; icon?: boolean }) {
  const t = useT()
  const { slug } = useWorkspace()
  const navigate = useNavigate()
  const create = $api.useMutation('post', '/api/v1/workspaces/{workspace}/push/prompts')
  return (
    <Button
      size="sm"
      icon={icon ? 'plus' : undefined}
      loading={create.isPending}
      onClick={() =>
        create.mutate(
          { params: { path: { workspace: slug } }, body: { name: t(m.newPushPrompt), siteId } },
          {
            onSuccess: (p) => void navigate({ to: '/w/$workspace/forms', params: { workspace: slug }, search: { prompt: p.formId } }),
            onError: (e) => toast({ kind: 'error', title: errorMessage(e) }),
          },
        )
      }
    >
      {label}
    </Button>
  )
}

/** Add a site: the address, then the snippet, the file and the import line for it. */
function AddSite({ data, addRef }: { data: PushSetup; addRef: RefObject<HTMLInputElement | null> }) {
  const t = useT()
  const { slug } = useWorkspace()
  const invalidate = useInvalidatePush()
  const [address, setAddress] = useState('')
  const add = $api.useMutation('post', '/api/v1/workspaces/{workspace}/push/sites')
  const host = bareHost(address) || bareHost(t(m.siteAddressPlaceholder))
  const origin = `https://${host}`
  const s = data.snippet
  const [invalid, setInvalid] = useState(false)
  const submit = (value = address) => {
    if (!value.trim()) {
      setInvalid(true)
      addRef.current?.focus()
      return
    }
    setInvalid(false)
    add.mutate(
      { params: { path: { workspace: slug } }, body: { address: value.trim() } },
      {
        onSuccess: (site) => {
          void invalidate()
          setAddress('')
          toast({ kind: 'success', title: t(m.siteAdded, { host: site.host }) })
        },
      },
    )
  }
  const snippet = `<script async src="${s.script}"\n  data-workspace="${slug}"\n  data-site="${host}"></script>`
  const importLine = `importScripts("${s.importUrl}");`
  return (
    <div className="px-add">
      <div className="px-add-row">
        <div className="flex flex-col gap-0.5">
          <h3 className="h2">{t(m.addTitle)}</h3>
          <span className="k">{t(m.addIntro)}</span>
        </div>
        <form
          className="flex flex-wrap items-end gap-2"
          onSubmit={(e) => {
            e.preventDefault()
            submit()
          }}
        >
          <div className="lbl">
            <label htmlFor="push-site-address">{t(m.siteAddress)}</label>
            <input
              ref={addRef}
              id="push-site-address"
              className="in w-[260px] max-w-full"
              value={address}
              placeholder={t(m.siteAddressPlaceholder)}
              onChange={(e) => {
                setAddress(e.target.value)
                setInvalid(false)
              }}
              aria-invalid={add.isError || invalid || undefined}
              aria-describedby={add.isError || invalid ? 'push-site-error' : undefined}
              autoComplete="off"
              spellCheck={false}
            />
          </div>
          <Button variant="primary" type="submit" loading={add.isPending} busyLabel={t(m.adding)}>
            {t(m.addSiteButton)}
          </Button>
        </form>
      </div>
      {add.isError || invalid ? (
        <span className="err" id="push-site-error" role="alert">
          {invalid ? t(m.siteAddressMissing) : errorMessage(add.error)}
        </span>
      ) : null}
      {data.website ? (
        <div className="note i" role="note">
          <InfoIcon color="#1E40AF" />
          <div className="flex flex-wrap items-center justify-between gap-2">
            <span>{t(m.addWebsiteNote, { host: bareHost(data.website) })}</span>
            <Button size="sm" onClick={() => submit(data.website)} loading={add.isPending}>
              {t(m.addWebsite, { host: bareHost(data.website) })}
            </Button>
          </div>
        </div>
      ) : null}
      <div className="flex flex-col">
        <Step n={1} title={t(m.step1Title)} text={t(m.step1Text)}>
          <CodeBox lang={t(m.html)} label={t(m.copySnippet)} copy={snippet}>
            &lt;<span className="c5">script</span>
            <span className="c1"> async src</span>=<span className="c2">&quot;{s.script}&quot;</span>
            <span className="c1">{'\n  '}data-workspace</span>=<span className="c2">&quot;{slug}&quot;</span>
            <span className="c1">{'\n  '}data-site</span>=<span className="c2">&quot;{host}&quot;</span>&gt;&lt;/<span className="c5">script</span>&gt;
          </CodeBox>
        </Step>
        <Step n={2} title={t(m.step2Title)} text={t(m.step2Text)}>
          <div className="px-file">
            <svg width="20" height="20" viewBox="0 0 16 16" fill="none" stroke="#4B5563" strokeWidth="1.4" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
              <path d="M9.5 1.5H4A1.5 1.5 0 0 0 2.5 3v10A1.5 1.5 0 0 0 4 14.5h8a1.5 1.5 0 0 0 1.5-1.5V5.5z" />
              <path d="M9.5 1.5v4h4M5.5 8.5h5M5.5 11h5" />
            </svg>
            <span className="flex min-w-0 flex-[1_1_140px] flex-col gap-px">
              <span className="mono text-[12.5px] font-semibold">{s.workerFile}</span>
              <span className="k">
                <T
                  msg={m.workerFileLine}
                  values={{ size: t(m.sizeKb, { n: Math.max(1, Math.round(s.workerBytes / 1024)) }), version: s.workerVersion, url: <span className="mono">{`${origin}/${s.workerFile}`}</span> }}
                />
              </span>
            </span>
            <ButtonAnchor href={s.workerUrl} download={s.workerFile} newTab={false} size="sm" icon="download">
              {t(m.download)}
            </ButtonAnchor>
          </div>
        </Step>
        <Step n={3} title={t(m.step3Title)} text={t(m.step3Text)}>
          <CodeBox lang={t(m.javascript)} label={t(m.copyImport)} copy={importLine}>
            <span className="cm">{t(m.importComment)}</span>
            {'\n'}
            <span className="c5">importScripts</span>(<span className="c2">&quot;{s.importUrl}&quot;</span>);
          </CodeBox>
        </Step>
      </div>
    </div>
  )
}

function Step({ n, title, text, children }: { n: number; title: string; text: string; children: ReactNode }) {
  return (
    <div className="px-step">
      <div className="lead">
        <span className="px-num" aria-hidden="true">
          {n}
        </span>
        <span className="flex min-w-0 flex-col gap-0.5">
          <span className="font-semibold">{title}</span>
          <span className="k text-ink-2">{text}</span>
        </span>
      </div>
      <div className="body">{children}</div>
    </div>
  )
}

function CodeBox({ lang, label, copy, children }: { lang: string; label: string; copy: string; children: ReactNode }) {
  const t = useT()
  return (
    <div className="px-cbx">
      <div className="px-cbh">
        <span className="lang">{lang}</span>
        <button
          type="button"
          className="cp"
          aria-label={label}
          onClick={() => void copyText(copy).then((ok) => ok && toast({ kind: 'success', title: t(m.copied) }))}
        >
          <svg width="13" height="13" viewBox="0 0 16 16" fill="none" stroke="currentColor" strokeWidth="1.6" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
            <rect x="5" y="5" width="9" height="9" rx="1.5" />
            <path d="M11 5V3.5A1.5 1.5 0 0 0 9.5 2h-6A1.5 1.5 0 0 0 2 3.5v6A1.5 1.5 0 0 0 3.5 11H5" />
          </svg>
          {t(m.copy)}
        </button>
      </div>
      <pre>{children}</pre>
    </div>
  )
}

/** Subscribers by browser, by site, and linked to contacts or anonymous. */
export function SubscribersCard({ data }: { data: PushSetup }) {
  const t = useT()
  const { slug } = useWorkspace()
  const s = data.subscribers
  const top = Math.max(1, ...s.byBrowser.map((b) => b.count))
  return (
    <section className="card">
      <div className="sec">
        <div className="flex min-w-0 flex-col gap-0.5">
          <h2 className="h2">{t(m.subscribersTitle, { count: count(s.total) })}</h2>
        </div>
        {s.highlight ? (
          <div className="flex flex-wrap items-center gap-2">
            <Link className="px-lnk" to="/w/$workspace/audience" params={{ workspace: slug }} search={{ list: s.highlight.listId }}>
              {t(m.highlight, { count: count(s.highlight.count), list: s.highlight.name })}
            </Link>
          </div>
        ) : null}
      </div>
      {s.total ? (
        <div className="pad flex flex-col gap-[18px]">
          <div className="flex flex-col gap-2.5">
            <span className="k px-sub">{t(m.byBrowser)}</span>
            <div className="px-bars">
              {s.byBrowser.map((b) => (
                <div key={b.key} className="flex flex-col gap-1">
                  <div className="row-h">
                    <span>{t(browserName(b.key))}</span>
                    <span className="num font-semibold">{count(b.count)}</span>
                  </div>
                  <div className="px-bar">
                    <i style={{ width: `${((b.count / top) * 100).toFixed(1)}%` }} />
                  </div>
                </div>
              ))}
            </div>
          </div>
          <div className="flex flex-col gap-2">
            <span className="k px-sub">{t(m.bySite)}</span>
            <div className="flex flex-col gap-2.5">
              <div className="px-split" role="img" aria-label={s.bySite.map((b) => t(m.siteLegend, { host: b.key, count: count(b.count) })).join(', ')}>
                {s.bySite.map((b, i) => (b.count ? <i key={b.key} style={{ flex: b.count, background: SITE_COLORS[i % SITE_COLORS.length] }} /> : null))}
              </div>
              <div className="px-legend">
                {s.bySite.map((b, i) => (
                  <span key={b.key}>
                    <i style={{ background: SITE_COLORS[i % SITE_COLORS.length] }} />
                    {t(m.siteLegend, { host: b.key, count: count(b.count) })}
                  </span>
                ))}
              </div>
            </div>
          </div>
          <div className="flex flex-col gap-2">
            <span className="k px-sub">{t(m.linkedTitle)}</span>
            <div className="flex flex-col gap-2.5">
              <div className="px-split" role="img" aria-label={`${t(m.linkedLegend, { count: count(s.linked) })}, ${t(m.anonymousLegend, { count: count(s.anonymous) })}`}>
                {s.linked ? <i style={{ flex: s.linked, background: 'var(--accent)' }} /> : null}
                {s.anonymous ? <i style={{ flex: s.anonymous, background: 'var(--chart-other)' }} /> : null}
              </div>
              <div className="px-legend">
                <span>
                  <i style={{ background: 'var(--accent)' }} />
                  {t(m.linkedLegend, { count: count(s.linked) })}
                </span>
                <span>
                  <i style={{ background: 'var(--chart-other)' }} />
                  {t(m.anonymousLegend, { count: count(s.anonymous) })}
                </span>
              </div>
            </div>
            <span className="k text-ink-2">{t(m.linkedNote)}</span>
          </div>
        </div>
      ) : (
        <div className="pad">
          <span className="k">{t(m.noSubscribers)}</span>
        </div>
      )}
    </section>
  )
}

/** The prompts with their 30-day numbers, linked to the designer. */
export function PromptsCard({ data }: { data: PushSetup }) {
  const t = useT()
  return (
    <section className="card">
      <div className="px-head">
        <div>
          <h2 className="h2">{t(m.promptsTitle, { count: data.prompts.length })}</h2>
          <span className="k">{t(m.promptsIntro)}</span>
        </div>
        {data.canEditPrompts ? (
          <div>
            <NewPromptButton label={t(m.newPushPrompt)} icon />
          </div>
        ) : null}
      </div>
      {data.prompts.length ? (
        data.prompts.map((p, i) => <PromptBlock key={p.formId} prompt={p} first={i === 0} />)
      ) : (
        <div className="pad">
          <span className="k">{t(m.noPrompts)}</span>
        </div>
      )}
    </section>
  )
}

function PromptBlock({ prompt, first }: { prompt: PushPromptCard; first: boolean }) {
  const t = useT()
  const st = prompt.stats
  const bell = prompt.config.style === 'bell'
  return (
    <div className="px-prompt" style={first ? { borderTop: 0 } : undefined}>
      <div className="px-ph">
        <span className="nm">
          {prompt.site ? (
            <>
              <span className="mono font-semibold">{prompt.site}</span>
              <span className="k" aria-hidden="true">
                ·
              </span>
            </>
          ) : null}
          <span className="font-semibold">{t(styleName[prompt.config.style])}</span>
          {prompt.status === 'live' ? (
            <Badge tone="ok" dot>
              {t(m.live)}
            </Badge>
          ) : prompt.status === 'paused' ? (
            <Badge tone="warn">{t(m.promptPaused)}</Badge>
          ) : (
            <Badge tone="neutral">{t(m.promptDraft)}</Badge>
          )}
        </span>
        <OpenPrompt formId={prompt.formId} label={t(m.openInDesigner)} />
      </div>
      <span className="k text-ink-2">{promptSummary(t, prompt.config)}</span>
      <span className="k px-sub mt-0.5">
        {st.sinceSnippet ? t(m.sinceSnippet, { date: dayMonthDate(st.since) }) : t(m.last30)}
      </span>
      <div className="px-funnel">
        <div>
          <span className="k">{t(bell ? m.numBell : m.numShown)}</span>
          <span className="v">{count(bell ? st.bellClicked : st.shown)}</span>
        </div>
        <div>
          <span className="k">{t(m.numAllow)}</span>
          <span className="v">{count(st.allowed)}</span>
          <span className="k">{pct(st.allowedPct)}</span>
        </div>
        <div>
          <span className="k">{t(m.numGranted)}</span>
          <span className="v">{count(st.granted)}</span>
          <span className="k">{pct(st.grantedPct)}</span>
        </div>
        <div>
          <span className="k">{t(m.numNotNow)}</span>
          <span className="v">{count(st.notNow)}</span>
          <span className="k">{pct(st.notNowPct)}</span>
        </div>
        <div>
          <span className="k">{t(m.numClosed)}</span>
          <span className="v">{count(st.closed)}</span>
          <span className="k">{pct(st.closedPct)}</span>
        </div>
      </div>
      <span className="k">{t(bell ? m.sharesBell : m.sharesShown)}</span>
    </div>
  )
}

/** "“Get new posts as notifications” · after 30 seconds or half a post · asks again after 14 days". */
export function promptSummary(t: ReturnType<typeof useT>, c: PushPromptConfig): string {
  if (c.style === 'bell') return t(m.bellSummary, { title: c.title, corner: t(c.corner === 'bottom_left' ? m.cornerLeft : m.cornerRight) })
  const parts: string[] = []
  if (c.when.seconds.on) parts.push(t(m.whenSeconds, { n: c.when.seconds.n }))
  if (c.when.scroll.on) parts.push(c.when.scroll.n === 50 ? t(m.whenHalf) : t(m.whenScroll, { n: c.when.scroll.n }))
  if (c.when.posts.on) parts.push(t(m.whenPosts, { n: c.when.posts.n }))
  if (c.when.button) parts.push(t(m.whenButton))
  const when = orList(parts)
  return t(c.style === 'dialog' ? m.dialogSummary : m.barSummary, { title: c.title, when, days: c.askAgainDays })
}
