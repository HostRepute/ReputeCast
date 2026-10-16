import { useQueryClient } from '@tanstack/react-query'
import { useState } from 'react'
import { $api, errorMessage, type Schemas } from '@/api/client'
import { Badge, type Tone } from '@/components/ui/Badge'
import { Button } from '@/components/ui/Button'
import { Dialog } from '@/components/ui/Dialog'
import { Skeleton } from '@/components/ui/Feedback'
import { Checkbox, Field, Input, Textarea } from '@/components/ui/Field'
import { toast } from '@/components/ui/Toast'
import { useT } from '@/i18n'
import { dayMonthDate } from '@/lib/format'
import { useWorkspace } from '@/lib/session'
import { ma, mc } from './messages'

type Lang = Schemas['EmailLanguage']
type Translate = ReturnType<typeof useT>

const languagesPath = '/api/v1/workspaces/{workspace}/automation-emails/{id}/languages' as const

/** A language's status badge (DSExpansion › Language chip). */
function statusBadge(l: Lang, fallbackName: string, t: Translate): { tone: Tone; text: string } {
  if (l.held) return { tone: 'neutral', text: t(mc.held, { language: fallbackName }) }
  switch (l.status) {
    case 'source':
      return { tone: 'info', text: t(mc.source) }
    case 'up_to_date':
      return { tone: 'ok', text: t(mc.upToDate) }
    case 'outdated':
      return { tone: 'warn', text: t(mc.outdatedBlocks, { count: l.attention }) }
    case 'draft':
      return { tone: 'neutral', text: t(mc.draft) }
  }
  return { tone: 'neutral', text: t(mc.missingGets, { language: fallbackName }) }
}

function detail(l: Lang, fallbackName: string, t: Translate): string {
  switch (l.status) {
    case 'up_to_date':
      return l.translatedBy && l.translatedAt ? t(ma.translatedBy, { name: l.translatedBy, date: dayMonthDate(l.translatedAt) }) : ''
    case 'outdated':
      return l.changedBlocks.length === 1
        ? t(ma.outdatedBlock, { block: (l.changedBlocks[0] ?? '').split(' · ')[0]?.toLowerCase() ?? '', name: l.translatedBy || t(ma.someone) })
        : t(ma.outdatedBlocks, { count: l.attention, name: l.translatedBy || t(ma.someone) })
    case 'draft':
      return t(ma.draftDetail, { language: l.name, fallback: fallbackName })
    case 'missing':
      return t(ma.missingDetail, { language: l.name, fallback: fallbackName })
  }
  return ''
}

/**
 * The Send email step's languages (AutomationLanguages): each version's
 * status, "Edit in composer" for the one you pick, and its subject. Writing
 * a language version makes the email's next version, which this step then
 * sends once the flow is published.
 */
export function StepLanguages({
  emailId,
  version,
  disabledReason,
  onEditSource,
  onVersion,
}: {
  emailId: string
  version: number
  /** Why the step can't be edited (read-only, a live version). */
  disabledReason?: string
  onEditSource: () => void
  /** The email's new version, after a language version was written. */
  onVersion: (version: number) => void
}) {
  const { slug } = useWorkspace()
  const t = useT()
  const q = $api.useQuery('get', languagesPath, { params: { path: { workspace: slug, id: emailId }, query: { version } } })
  const [pick, setPick] = useState<string>()
  const [editing, setEditing] = useState<string>()
  const data = q.data
  if (q.isPending) {
    return (
      <div className="fe-prop" aria-busy="true">
        <span className="tt">{t(ma.title)}</span>
        <Skeleton h={12} />
        <Skeleton w="70%" h={12} />
      </div>
    )
  }
  if (!data || data.versions.length < 2) return null
  const fallbackName = data.versions.find((v) => v.language === data.fallback)?.name ?? data.fallback
  const chosen = data.versions.find((v) => v.language === pick) ?? data.versions.find((v) => v.status === 'outdated') ?? data.versions[0]!
  const isSource = chosen.status === 'source'
  const missing = chosen.status === 'missing'
  const edit = () => (isSource ? onEditSource() : setEditing(chosen.language))
  return (
    <>
      <div className="fe-prop" role="group" aria-labelledby="p-languages">
        <span className="tt" id="p-languages">
          {t(ma.title)}
        </span>
        <div className="flex flex-col">
          {data.versions.map((l) => {
            const b = statusBadge(l, fallbackName, t)
            const d = detail(l, fallbackName, t)
            return (
              <div key={l.language} className="flex flex-col gap-px py-1.5" style={{ borderTop: '1px solid var(--line-2)' }}>
                <span className="flex flex-wrap items-center gap-2">
                  <b style={{ fontWeight: 600 }}>{l.name}</b>
                  <Badge tone={b.tone} style={{ height: 20, fontSize: 11 }}>
                    {b.text}
                  </Badge>
                </span>
                {d ? <span className="k">{d}</span> : null}
              </div>
            )
          })}
        </div>
        <div className="flex flex-wrap items-end gap-2">
          <label className="lbl" style={{ flex: '1 1 140px' }}>
            {t(ma.editIn)}
            <select className="in" value={chosen.language} onChange={(e) => setPick(e.target.value)}>
              {data.versions.map((l) => (
                <option key={l.language} value={l.language}>
                  {l.status === 'missing' ? t(ma.addLanguage, { language: l.name }) : l.name}
                </option>
              ))}
            </select>
          </label>
          <Button size="sm" disabledReason={isSource ? undefined : disabledReason} onClick={edit}>
            {missing ? t(ma.addLanguage, { language: chosen.name }) : t(ma.editLanguage, { language: chosen.name })}
          </Button>
        </div>
      </div>
      {chosen.subject && !missing ? (
        <div className="fe-prop">
          <span className="tt">{t(ma.subjectIn, { language: chosen.name })}</span>
          <span style={{ fontSize: 13 }} lang={chosen.language}>
            {chosen.subject}
          </span>
        </div>
      ) : null}
      {editing ? (
        <LanguageEmailDialog
          emailId={emailId}
          version={version}
          language={data.versions.find((v) => v.language === editing)!}
          onClose={() => setEditing(undefined)}
          onSaved={(v) => {
            setEditing(undefined)
            onVersion(v)
          }}
        />
      ) : null}
    </>
  )
}

/**
 * Editing one language version of an automation email in the flow editor:
 * its subject, preheader and HTML, like the email itself. Saving makes the
 * email's next version.
 */
function LanguageEmailDialog({
  emailId,
  version,
  language,
  onClose,
  onSaved,
}: {
  emailId: string
  version: number
  language: Lang
  onClose: () => void
  onSaved: (version: number) => void
}) {
  const { slug } = useWorkspace()
  const t = useT()
  const qc = useQueryClient()
  const missing = language.status === 'missing'
  // A missing language starts from the email's own content.
  const source = $api.useQuery('get', '/api/v1/workspaces/{workspace}/automation-emails/{id}', { params: { path: { workspace: slug, id: emailId }, query: { version } } }, { enabled: missing })
  const own = $api.useQuery(
    'get',
    '/api/v1/workspaces/{workspace}/automation-emails/{id}/languages/{language}',
    { params: { path: { workspace: slug, id: emailId, language: language.language }, query: { version } } },
    { enabled: !missing },
  )
  const loaded = missing ? source.data : own.data
  const save = $api.useMutation('put', '/api/v1/workspaces/{workspace}/automation-emails/{id}/languages/{language}')
  const [edits, setEdits] = useState<{ subject: string; preheader: string; html: string }>()
  const [translated, setTranslated] = useState(language.status !== 'draft' && !missing)
  const form = edits ?? { subject: loaded?.subject ?? '', preheader: loaded?.preheader ?? '', html: loaded?.html ?? '' }
  const submit = () =>
    save.mutate(
      { params: { path: { workspace: slug, id: emailId, language: language.language } }, body: { version, ...form, translated } },
      {
        onSuccess: (r) => {
          void qc.invalidateQueries({ queryKey: ['get', languagesPath] })
          void qc.invalidateQueries({ queryKey: ['get', '/api/v1/workspaces/{workspace}/automations/options'] })
          toast({ title: t(ma.saved, { language: language.name, version: r.version }), detail: t(ma.savedDetail) })
          onSaved(r.version)
        },
        onError: (e) => toast({ kind: 'error', title: t(ma.saveFailed, { language: language.name }), detail: errorMessage(e) }),
      },
    )
  return (
    <Dialog
      open
      onOpenChange={(o) => !o && onClose()}
      size="lg"
      title={t(ma.dialogTitle, { language: language.name })}
      description={t(ma.dialogDescription, { language: language.name, version })}
      footer={
        <>
          <Button onClick={onClose}>{t(ma.cancel)}</Button>
          <Button variant="primary" loading={save.isPending} disabled={!loaded} onClick={submit}>
            {t(ma.saveNew)}
          </Button>
        </>
      }
    >
      {loaded ? (
        <>
          <Field label={t(ma.subject)}>
            <Input lang={language.language} value={form.subject} onChange={(e) => setEdits({ ...form, subject: e.target.value })} />
          </Field>
          <Field label={t(ma.preheader)} optional>
            <Input lang={language.language} value={form.preheader} onChange={(e) => setEdits({ ...form, preheader: e.target.value })} />
          </Field>
          <Field label={t(ma.html)} hint={t(ma.htmlHint, { example: `{{ first_name | default: "there" }}` })}>
            <Textarea className="mono min-h-[260px] text-[12px]" spellCheck={false} value={form.html} onChange={(e) => setEdits({ ...form, html: e.target.value })} />
          </Field>
          <Checkbox
            label={t(ma.translated, { language: language.name })}
            hint={t(ma.translatedHint, { language: language.name })}
            checked={translated}
            onChange={(e) => setTranslated(e.target.checked)}
          />
        </>
      ) : (
        <Skeleton w="100%" h={320} />
      )}
    </Dialog>
  )
}
