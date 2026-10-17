import { useMemo, useState } from 'react'
import { $api, ApiError, errorMessage, type Schemas } from '@/api/client'
import { Badge } from '@/components/ui/Badge'
import { Button } from '@/components/ui/Button'
import { Dialog } from '@/components/ui/Dialog'
import { ErrorPanel, Skeleton } from '@/components/ui/Feedback'
import { toast } from '@/components/ui/Toast'
import { useInvalidateForms } from '@/features/forms/common'
import { languageLabel } from '@/features/languages/composerParts'
import { useI18n, useT } from '@/i18n'
import { cn } from '@/lib/cn'
import { usePermissionReason, useWorkspace } from '@/lib/session'
import { mf, ml } from './messages'
import './page-languages.css'

export type CopyStatus = 'live' | 'incomplete' | 'not_set_up'
export type CopyLabels = Schemas['PublicCopyLabels']
export type PageLabel = Schemas['PublicPageLabel']

/** A language of the "Copy in" switcher. */
export interface SwitcherLanguage {
  code: string
  native: string
  status: CopyStatus
  missing: number
}

/** Live · Incomplete · n to write · Not set up. */
export function CopyStatusBadge({ status, missing, size = 'row' }: { status: CopyStatus; missing: number; size?: 'chip' | 'row' }) {
  const t = useT()
  const cls = size === 'chip' ? 'pl-badge-chip' : 'pl-badge-row'
  if (status === 'live') {
    return (
      <Badge tone="ok" className={cls}>
        {t(mf.statusLive)}
      </Badge>
    )
  }
  if (status === 'incomplete') {
    return (
      <Badge tone="warn" className={cls}>
        {t(mf.statusIncomplete, { count: missing })}
      </Badge>
    )
  }
  return (
    <Badge tone="neutral" className={cls}>
      {t(mf.statusNotSetUp)}
    </Badge>
  )
}

/**
 * "Copy in": each workspace language in its own name with its copy status
 * (FormBuilderLanguages). Pressing one shows its copy.
 */
export function CopySwitcher({ languages, value, onPick }: { languages: SwitcherLanguage[]; value: string; onPick: (code: string) => void }) {
  const t = useT()
  return (
    <div className="pl-chips" role="group" aria-label={t(mf.copyIn)}>
      {languages.map((l) => {
        const on = l.code === value
        return (
          <button key={l.code} type="button" className={cn('pl-chip', on && 'on')} aria-pressed={on} onClick={() => onPick(l.code)}>
            <span lang={l.code}>{l.native}</span>{' '}
            <CopyStatusBadge status={l.status} missing={l.missing} size="chip" />
          </button>
        )
      })}
    </div>
  )
}

/** The language's name in the app's language: "Spanish". */
export function useLanguageName(): (code: string) => string {
  const { locale } = useI18n()
  return (code: string) => languageLabel(code, locale)
}

/**
 * "Built-in labels · Spanish" (FormBuilderLanguages): collapsed, with how
 * many are written and who wrote them; ReputeCast's own languages are
 * Ready. Expanding offers the editor.
 */
export function BuiltinLabelsSection({ code, labels, className }: { code: string; labels: CopyLabels; className?: string }) {
  const t = useT()
  const name = useLanguageName()(code)
  const [editing, setEditing] = useState(false)
  if (labels.shipped) return <ShippedLabelsRow code={code} className={className} />
  return (
    <div className={cn('pl-labels', className)}>
      <details className="pl-details">
        <summary>
          <svg width="13" height="13" viewBox="0 0 16 16" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
            <path d="m6 4 4 4-4 4" />
          </svg>
          {t(mf.labelsTitle, { language: name })}
          <span className="k pl-labels-count">{t(mf.labelsCount, { written: labels.written, total: labels.total })}</span>
        </summary>
        <div className="pl-details-body">
          <Button size="sm" onClick={() => setEditing(true)}>
            {t(mf.labelsEdit, { language: name })}
          </Button>
        </div>
      </details>
      <span className="k">{labels.written > 0 && labels.by ? t(mf.labelsWrittenBy, { name: labels.by, language: name }) : t(mf.labelsToWrite, { language: name })}</span>
      {editing ? <LabelsDialog code={code} onClose={() => setEditing(false)} /> : null}
    </div>
  )
}

/** "Built-in labels · Vietnamese · Ready, from ReputeCast's translation". */
export function ShippedLabelsRow({ code, className }: { code: string; className?: string }) {
  const t = useT()
  const name = useLanguageName()(code)
  return (
    <div className={cn('pl-shipped', className)}>
      <span className="flex min-w-0 flex-col gap-0.5">
        <b className="font-semibold">{t(mf.labelsTitle, { language: name })}</b>
        <span className="k">{t(mf.labelsShipped)}</span>
      </span>
      <Badge tone="ok">{t(mf.labelsReady)}</Badge>
    </div>
  )
}

/** The editor of a language's built-in labels, in a dialog. */
function LabelsDialog({ code, onClose }: { code: string; onClose: () => void }) {
  const t = useT()
  const { slug } = useWorkspace()
  const name = useLanguageName()(code)
  const q = $api.useQuery('get', '/api/v1/workspaces/{workspace}/public-pages/labels/{language}', { params: { path: { workspace: slug, language: code } } })
  return (
    <Dialog
      open
      onOpenChange={(o) => !o && onClose()}
      title={t(ml.title, { language: name })}
      description={t(ml.description, { language: name, example: '{publisher}' })}
      size="lg"
      footer={q.data ? undefined : <Button onClick={onClose}>{t(ml.cancel)}</Button>}
    >
      {q.isError ? (
        <ErrorPanel title={t(mf.loadFailed)} actions={<Button onClick={() => q.refetch()}>{t(mf.retry)}</Button>}>
          {errorMessage(q.error)}
        </ErrorPanel>
      ) : !q.data ? (
        <div className="flex flex-col gap-2" aria-busy="true">
          <span className="vh" role="status">
            {t(ml.loading)}
          </span>
          <Skeleton h={52} />
          <Skeleton h={52} />
          <Skeleton h={52} />
        </div>
      ) : (
        <LabelsEditor code={code} data={q.data} onClose={onClose} />
      )}
    </Dialog>
  )
}

function LabelsEditor({ code, data, onClose }: { code: string; data: Schemas['PublicPageLabels']; onClose: () => void }) {
  const t = useT()
  const { slug } = useWorkspace()
  const name = useLanguageName()(code)
  const invalidate = useInvalidateForms()
  const reason = usePermissionReason('forms:edit')
  const readOnly = Boolean(reason) || data.shipped
  const initial = useMemo(() => Object.fromEntries(data.labels.map((l) => [l.key, l.text])), [data])
  const [texts, setTexts] = useState<Record<string, string>>(initial)
  const [query, setQuery] = useState('')
  const [missingOnly, setMissingOnly] = useState(false)
  const [errors, setErrors] = useState<Record<string, string>>({})
  const save = $api.useMutation('put', '/api/v1/workspaces/{workspace}/public-pages/labels/{language}')
  const written = data.labels.filter((l) => (texts[l.key] ?? '').trim() !== '').length
  const q = query.trim().toLowerCase()
  const shown = data.labels.filter(
    (l) =>
      (!missingOnly || (texts[l.key] ?? '').trim() === '') &&
      (!q || l.source.toLowerCase().includes(q) || (texts[l.key] ?? '').toLowerCase().includes(q) || l.key.toLowerCase().includes(q)),
  )
  const copyEnglish = () => {
    const empty = data.labels.filter((l) => (texts[l.key] ?? '').trim() === '')
    setTexts((x) => ({ ...x, ...Object.fromEntries(empty.map((l) => [l.key, l.source])) }))
    toast({ kind: 'info', title: t(ml.copyEnglishDone, { count: empty.length }), detail: t(ml.copyEnglishDetail) })
  }
  const submit = async () => {
    const changed = Object.fromEntries(Object.entries(texts).filter(([k, v]) => v !== initial[k]))
    setErrors({})
    try {
      const out = await save.mutateAsync({ params: { path: { workspace: slug, language: code } }, body: { labels: changed } })
      await invalidate()
      toast({ kind: 'success', title: t(ml.saved, { language: name }), detail: t(ml.savedDetail, { written: out.written, total: out.total }) })
      onClose()
    } catch (e) {
      const fieldErrors: Record<string, string> = {}
      if (e instanceof ApiError) {
        for (const x of e.problem.errors ?? []) {
          if (x.location?.startsWith('body.labels.') && x.message) fieldErrors[x.location.slice('body.labels.'.length)] = x.message
        }
      }
      setErrors(fieldErrors)
      toast({ kind: 'error', title: t(ml.saveFailed), detail: errorMessage(e) })
    }
  }
  if (data.shipped) {
    return <p className="text-ink-2">{t(ml.shipped, { language: name })}</p>
  }
  return (
    <div className="flex flex-col gap-3">
      <div className="pl-label-tools">
        <input
          className="fb-in pl-label-filter"
          type="search"
          aria-label={t(ml.filter)}
          placeholder={t(ml.filter)}
          value={query}
          onChange={(e) => setQuery(e.target.value)}
        />
        <label className="fb-check">
          <input type="checkbox" checked={missingOnly} onChange={(e) => setMissingOnly(e.target.checked)} />
          {t(ml.missingOnly)}
        </label>
        <Button size="sm" onClick={copyEnglish} disabledReason={reason}>
          {t(ml.copyEnglish)}
        </Button>
        <span className="k ml-auto" aria-live="polite">
          {t(ml.count, { written, total: data.total })}
        </span>
      </div>
      {shown.length === 0 ? (
        <p className="k">{q ? t(ml.none, { query }) : t(ml.allWritten)}</p>
      ) : (
        <ul className="pl-label-list">
          {shown.map((l) => {
            const id = `pl-label-${l.key.replace(/[^a-z0-9]/gi, '-')}`
            const err = errors[l.key]
            return (
              <li key={l.key} className="pl-label">
                <label htmlFor={id} className="pl-label-source">
                  {t(ml.english, { text: l.source })}
                </label>
                {l.note || l.arguments.length ? (
                  <span className="k" id={`${id}-note`}>
                    {[l.note, l.arguments.length ? t(ml.uses, { args: l.arguments.join(' ') }) : ''].filter(Boolean).join(' · ')}
                  </span>
                ) : null}
                <textarea
                  id={id}
                  className="fb-in pl-label-input"
                  lang={code}
                  rows={l.source.length > 70 ? 3 : 1}
                  value={texts[l.key] ?? ''}
                  disabled={readOnly}
                  aria-invalid={err ? true : undefined}
                  aria-describedby={[l.note || l.arguments.length ? `${id}-note` : '', err ? `${id}-err` : ''].filter(Boolean).join(' ') || undefined}
                  onChange={(e) => setTexts((x) => ({ ...x, [l.key]: e.target.value }))}
                />
                {err ? (
                  <span className="err" id={`${id}-err`} role="alert">
                    {err}
                  </span>
                ) : null}
              </li>
            )
          })}
        </ul>
      )}
      <div className="flex flex-wrap justify-end gap-2 border-t border-line-2 pt-3">
        <Button onClick={onClose}>{t(ml.cancel)}</Button>
        <Button variant="primary" onClick={() => void submit()} loading={save.isPending} busyLabel={t(ml.saving)} disabledReason={reason}>
          {t(ml.save)}
        </Button>
      </div>
    </div>
  )
}
