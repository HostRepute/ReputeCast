import { useMemo, useState } from 'react'
import { $api, errorMessage, type Schemas } from '@/api/client'
import { problemReference } from '@/components/shell/SystemScreens'
import { Page } from '@/components/shell/Page'
import { Badge } from '@/components/ui/Badge'
import { Button, ButtonAnchor, ButtonLink } from '@/components/ui/Button'
import { ErrorPanel, Skeleton } from '@/components/ui/Feedback'
import { Field, Input, Textarea } from '@/components/ui/Field'
import { Note } from '@/components/ui/Note'
import { toast } from '@/components/ui/Toast'
import { FormsTabs } from '@/features/forms/common'
import { useT, type MessageDescriptor } from '@/i18n'
import { dayMonthYear } from '@/lib/format'
import { useWorkspace } from '@/lib/session'
import { throwIfForbidden, useRefreshSms } from './common'
import { m } from './terms.messages'
import './sms.css'

type Content = Schemas['SmsTermsContent']
type Key = keyof Content

const sections: { key: Key; label: MessageDescriptor; required?: boolean; rows?: number }[] = [
  { key: 'intro', label: m.intro, rows: 3 },
  { key: 'program', label: m.program, required: true },
  { key: 'frequency', label: m.frequency },
  { key: 'cost', label: m.cost, required: true },
  { key: 'help', label: m.help, required: true },
  { key: 'optOut', label: m.optOut, required: true },
  { key: 'senders', label: m.senders },
  { key: 'privacy', label: m.privacy },
  { key: 'carriers', label: m.carriers },
]

const links: { key: Key; label: MessageDescriptor; hint?: MessageDescriptor; mono?: boolean }[] = [
  { key: 'privacyUrl', label: m.privacyUrl, mono: true },
  { key: 'helpNumber', label: m.helpNumber, mono: true },
  { key: 'helpEmail', label: m.helpEmail, mono: true },
  { key: 'address', label: m.address, hint: m.addressHint },
]

/** Forms & pages › Text message terms: edit and publish the terms page. */
export function SmsTermsPage() {
  const t = useT()
  const { slug } = useWorkspace()
  const q = $api.useQuery('get', '/api/v1/workspaces/{workspace}/sms/terms', { params: { path: { workspace: slug } } })
  const ch = $api.useQuery('get', '/api/v1/workspaces/{workspace}/channels', { params: { path: { workspace: slug } } })
  throwIfForbidden(q.error)
  const sms = ch.data?.channels.find((c) => c.channel === 'sms')
  return (
    <Page title={t(m.title)} filters={<FormsTabs />}>
      {q.isError ? (
        <ErrorPanel title={t(m.errorTitle)} reference={problemReference(q.error)} actions={<Button onClick={() => void q.refetch()}>{t(m.retry)}</Button>}>
          {errorMessage(q.error)}
        </ErrorPanel>
      ) : !q.data ? (
        <div className="card flex flex-col gap-3 p-5" aria-busy="true" aria-label={t(m.loading)}>
          <Skeleton w={200} h={18} />
          <Skeleton h={80} />
          <Skeleton h={80} />
        </div>
      ) : (
        <>
          {sms && !sms.setUp ? (
            <Note tone="neutral">
              <span className="flex flex-wrap items-center gap-3">
                <span>{t(m.noSms)}</span>
                <ButtonLink size="sm" to="/w/$workspace/providers/connect/sms" params={{ workspace: slug }}>
                  {t(m.addProvider)}
                </ButtonLink>
              </span>
            </Note>
          ) : null}
          <Editor key={`${q.data.version}`} data={q.data} />
        </>
      )}
    </Page>
  )
}

function Editor({ data }: { data: Schemas['SmsTerms'] }) {
  const t = useT()
  const { slug } = useWorkspace()
  const refresh = useRefreshSms()
  const [content, setContent] = useState<Content>(data.content)
  const [errors, setErrors] = useState<Partial<Record<Key, string>>>({})
  const dirty = useMemo(() => JSON.stringify(content) !== JSON.stringify(data.content), [content, data.content])
  const ro = !data.canEdit
  const next = data.publishedAt ? data.version + 1 : data.version
  const publish = $api.useMutation('put', '/api/v1/workspaces/{workspace}/sms/terms', {
    onSuccess: (d) => {
      void refresh()
      toast({ kind: 'success', title: t(m.published, { version: d.version }), detail: t(m.publishedDetail, { version: d.version }) })
    },
    onError: (e) => toast({ kind: 'error', title: errorMessage(e) }),
  })
  const set = (k: Key, v: string) => setContent((c) => ({ ...c, [k]: v }))
  const submit = () => {
    const e: Partial<Record<Key, string>> = {}
    for (const s of sections) if (s.required && !String(content[s.key] ?? '').trim()) e[s.key] = t(m.required)
    setErrors(e)
    if (Object.keys(e).length) return
    publish.mutate({ params: { path: { workspace: slug } }, body: content })
  }
  return (
    <section className="card">
      <div className="sec">
        <div className="flex min-w-0 flex-col gap-0.5">
          <h2 className="h2">{t(m.heading)}</h2>
          <span className="k">
            {data.publishedAt
              ? data.publishedBy
                ? t(m.version, { version: data.version, date: dayMonthYear(data.publishedAt), name: data.publishedBy })
                : t(m.versionNoBy, { version: data.version, date: dayMonthYear(data.publishedAt) })
              : t(m.draft)}
          </span>
        </div>
        <div className="flex flex-wrap items-center gap-2">
          {dirty ? <Badge tone="warn">{t(m.unsaved)}</Badge> : null}
          <ButtonAnchor size="sm" href={data.url} newTab>
            {t(m.open)}
          </ButtonAnchor>
          <Button size="sm" variant="primary" disabled={ro || (!dirty && Boolean(data.publishedAt))} loading={publish.isPending} onClick={submit}>
            {t(m.publish, { version: next })}
          </Button>
        </div>
      </div>
      <div className="pad flex flex-col gap-4">
        <span className="text-ink-2">{t(m.lead)}</span>
        {ro ? <Note tone="neutral">{t(m.readOnly)}</Note> : null}
        <h3 className="h2">{t(m.sections)}</h3>
        {sections.map((s) => {
          const gen = data.generated[s.key] ?? ''
          const cur = String(content[s.key] ?? '')
          return (
            <div key={s.key} className="flex flex-col gap-1">
              <Field label={t(s.label)} error={errors[s.key]}>
                <Textarea rows={s.rows ?? 2} disabled={ro} value={cur} onChange={(e) => set(s.key, e.target.value)} />
              </Field>
              {!ro && gen && cur !== gen ? (
                <div>
                  <button type="button" className="sms-lbtn text-[12.5px]" onClick={() => set(s.key, gen)}>
                    {t(m.reset)}
                  </button>
                </div>
              ) : null}
            </div>
          )
        })}
        <h3 className="h2">{t(m.links)}</h3>
        <div className="grid gap-3.5" style={{ gridTemplateColumns: 'repeat(auto-fit, minmax(min(260px, 100%), 1fr))' }}>
          {links.map((l) => (
            <Field key={l.key} label={t(l.label)} hint={l.hint ? t(l.hint) : undefined}>
              <Input className={l.mono ? 'mono' : undefined} disabled={ro} value={String(content[l.key] ?? '')} onChange={(e) => set(l.key, e.target.value)} />
            </Field>
          ))}
        </div>
        {!ro ? (
          <div>
            <Button size="sm" onClick={() => setContent({ ...data.generated, address: content.address, privacyUrl: content.privacyUrl, helpEmail: content.helpEmail })}>
              {t(m.resetAll)}
            </Button>
          </div>
        ) : null}
      </div>
    </section>
  )
}
