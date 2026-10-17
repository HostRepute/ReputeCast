import { useNavigate } from '@tanstack/react-router'
import { useMemo, useState, type ReactNode } from 'react'
import { $api, ApiError, errorMessage, type Schemas } from '@/api/client'
import { Page } from '@/components/shell/Page'
import { Button } from '@/components/ui/Button'
import { UnsavedChangesGuard } from '@/components/ui/ConfirmDialog'
import { ErrorPanel, Skeleton } from '@/components/ui/Feedback'
import { toast } from '@/components/ui/Toast'
import { FormsTabs, hostPath, useInvalidateForms } from '@/features/forms/common'
import { useT } from '@/i18n'
import { usePermissionReason, useWorkspace } from '@/lib/session'
import { BuiltinLabelsSection, CopySwitcher, useLanguageName, type CopyStatus } from './copy'
import { useWorkspaceLanguages } from './contacts'
import { mf, mp } from './messages'

type PageCopy = Schemas['PublicPageCopy']
type CopyString = Schemas['PublicPageCopyString']
type List = Schemas['PublicPageList']

/** Which page tab: what of the pages' own text it edits. */
export type PageGroup = 'subscribe' | 'preferences' | 'unsubscribe' | 'archive'

const groupPages: Record<PageGroup, CopyString['page'][]> = {
  subscribe: ['subscribe', 'confirm', 'lists', 'privacy'],
  preferences: ['preferences'],
  unsubscribe: ['unsubscribe'],
  archive: [],
}

/** The pages' copy in every language, with one language's strings. */
export function usePageCopy(language?: string) {
  const { slug } = useWorkspace()
  return $api.useQuery('get', '/api/v1/workspaces/{workspace}/public-pages/copy', { params: { path: { workspace: slug }, query: { language } } })
}

/**
 * "Copy in" on the page tabs (spec 6, Forms & pages): each workspace
 * language with the status of the pages' copy, for a workspace with more
 * than one language. Picking one edits the page's text in it.
 */
export function PageLanguageSection({ value }: { value?: string }) {
  const t = useT()
  const name = useLanguageName()
  const navigate = useNavigate()
  const langs = useWorkspaceLanguages()
  const copy = usePageCopy()
  if ((langs.data?.languages.length ?? 0) < 2 || !copy.data) return null
  const d = copy.data
  const current = value && d.languages.some((l) => l.code === value) ? value : d.copyLanguage
  const pick = (code: string) => {
    void navigate({ to: '.', search: (s: Record<string, unknown>) => ({ ...s, language: code === d.copyLanguage ? undefined : code }) })
  }
  return (
    <div className="fb-sec">
      <span className="k font-semibold">{t(mp.copyIn)}</span>
      <CopySwitcher languages={d.languages.map((l) => ({ code: l.code, native: l.native, status: l.status as CopyStatus, missing: l.missing }))} value={current} onPick={pick} />
      {current === d.copyLanguage ? <span className="k">{t(mp.source, { language: name(d.copyLanguage) })}</span> : null}
    </div>
  )
}

/**
 * A page tab in a language other than the pages' own: the page's text in
 * it next to the original, its status, and its built-in labels, collapsed;
 * the preview on the right.
 */
export function PageCopyView({
  group,
  language,
  lists,
  preview,
  previewLabel,
  previewURL,
  extraBar,
}: {
  group: PageGroup
  language: string
  lists: List[]
  preview: (device: 'desktop' | 'mobile') => ReactNode
  previewLabel: string
  previewURL: string
  extraBar?: ReactNode
}) {
  const t = useT()
  const q = usePageCopy(language)
  if (q.isError) {
    return (
      <Page title={t(mf.pageTitle)} filters={<FormsTabs />}>
        <ErrorPanel title={t(mp.loadFailed)} actions={<Button onClick={() => q.refetch()}>{t(mp.retry)}</Button>}>
          {errorMessage(q.error)}
        </ErrorPanel>
      </Page>
    )
  }
  if (!q.data || q.data.language !== language) {
    return (
      <Page title={t(mf.pageTitle)} filters={<FormsTabs />} contentClassName="fb-main">
        <div className="fb-cols" aria-busy="true">
          <aside className="fb-left">
            <div className="fb-sec">
              <Skeleton h={18} w="40%" />
              <Skeleton h={64} />
            </div>
            <div className="fb-sec">
              <Skeleton h={120} />
            </div>
          </aside>
          <section className="fb-center">
            <div className="fb-stage wide">
              <Skeleton h={520} r={10} />
            </div>
          </section>
        </div>
      </Page>
    )
  }
  const strings = q.data.strings.filter((s) => groupPages[group].includes(s.page))
  return (
    <CopyEditor
      key={`${language}:${JSON.stringify(strings.map((s) => s.text))}`}
      data={q.data}
      strings={strings}
      language={language}
      lists={lists}
      preview={preview}
      previewLabel={previewLabel}
      previewURL={previewURL}
      extraBar={extraBar}
    />
  )
}

function CopyEditor({
  data,
  strings,
  language,
  lists,
  preview,
  previewLabel,
  previewURL,
  extraBar,
}: {
  data: PageCopy
  strings: CopyString[]
  language: string
  lists: List[]
  preview: (device: 'desktop' | 'mobile') => ReactNode
  previewLabel: string
  previewURL: string
  extraBar?: ReactNode
}) {
  const t = useT()
  const { slug } = useWorkspace()
  const name = useLanguageName()
  const invalidate = useInvalidateForms()
  const reason = usePermissionReason('forms:edit')
  const readOnly = Boolean(reason)
  const lang = data.languages.find((l) => l.code === language)!
  const initial = useMemo(() => Object.fromEntries(strings.map((s) => [s.key, s.text])), [strings])
  const [draft, setDraft] = useState<Record<string, string>>(initial)
  const [started, setStarted] = useState(lang.status !== 'not_set_up')
  const [device, setDevice] = useState<'desktop' | 'mobile'>('desktop')
  const [errors, setErrors] = useState<Record<string, string>>({})
  const save = $api.useMutation('put', '/api/v1/workspaces/{workspace}/public-pages/copy/{language}')
  const changed = Object.fromEntries(Object.entries(draft).filter(([k, v]) => v !== initial[k]))
  const dirty = Object.keys(changed).length > 0
  const fallback = name(data.fallback)
  const source = name(data.copyLanguage)
  const listName = (id?: string) => lists.find((l) => l.id === id)?.name ?? ''

  const submit = async () => {
    setErrors({})
    try {
      const out = await save.mutateAsync({ params: { path: { workspace: slug, language } }, body: { strings: changed } })
      const now = out.languages.find((l) => l.code === language)
      toast({
        kind: 'success',
        title: t(mp.saved, { language: name(language) }),
        detail: now?.status === 'live' ? t(mp.savedLive, { language: name(language) }) : t(mp.savedIncomplete, { count: now?.missing ?? 0, language: name(language) }),
      })
      await invalidate()
    } catch (e) {
      const fieldErrors: Record<string, string> = {}
      if (e instanceof ApiError) {
        for (const x of e.problem.errors ?? []) {
          if (x.location?.startsWith('body.strings.') && x.message) fieldErrors[x.location.slice('body.strings.'.length)] = x.message
        }
      }
      setErrors(fieldErrors)
      toast({ kind: 'error', title: t(mp.saveFailed, { language: name(language) }), detail: errorMessage(e) })
      throw e
    }
  }

  const label = (s: CopyString): string => {
    const list = listName(s.listId)
    if (s.key === 'subscribe.heading') return t(mp.heading)
    if (s.key === 'subscribe.description') return t(mp.description)
    if (s.key === 'subscribe.footnote') return t(mp.footnote)
    if (s.key.startsWith('subscribe.list.')) return list
    if (s.key.startsWith('confirm.list.')) return list
    if (s.key.startsWith('confirm.offer.')) return t(mp.confirmOffer, { list })
    if (s.key.startsWith('preferences.list.')) return list
    if (s.key === 'unsubscribe.fewerTitle') return t(mp.fewerTitle)
    if (s.key === 'unsubscribe.fewerText') return t(mp.fewerText)
    if (s.key === 'unsubscribe.fewerButton') return t(mp.fewerButton)
    if (s.key.endsWith('.name')) return t(mp.listName, { list })
    if (s.key.endsWith('.description')) return t(mp.listDescription, { list })
    if (s.key === 'privacyUrl') return t(mp.privacy)
    return s.key
  }
  const long = (s: CopyString) => s.key.endsWith('description') || s.key === 'subscribe.footnote' || s.key === 'unsubscribe.fewerText'
  const sections: { title: string; items: CopyString[] }[] = []
  const add = (title: string, test: (s: CopyString) => boolean) => {
    const items = strings.filter(test)
    if (items.length) sections.push({ title, items })
  }
  add(t(mp.groupSubscribe), (s) => ['subscribe.heading', 'subscribe.description', 'subscribe.footnote'].includes(s.key))
  add(t(mp.groupSubscribeLists), (s) => s.key.startsWith('subscribe.list.'))
  add(t(mp.groupConfirm), (s) => s.key.startsWith('confirm.'))
  add(t(mp.groupPreferences), (s) => s.key.startsWith('preferences.list.'))
  add(t(mp.groupUnsubscribe), (s) => s.key.startsWith('unsubscribe.'))
  add(t(mp.groupLists), (s) => s.page === 'lists')
  add(t(mp.groupPrivacy), (s) => s.page === 'privacy')

  const copySource = () => {
    setDraft((d) => ({ ...d, ...Object.fromEntries(strings.filter((s) => !s.optional && !(d[s.key] ?? '').trim()).map((s) => [s.key, s.source])) }))
    setStarted(true)
  }

  return (
    <Page
      title={t(mf.pageTitle)}
      filters={<FormsTabs />}
      actions={
        dirty ? (
          <>
            <Button onClick={() => setDraft(initial)}>{t(mp.discard)}</Button>
            <Button variant="primary" onClick={() => void submit().catch(() => undefined)} loading={save.isPending} busyLabel={t(mp.saving)} disabledReason={reason}>
              {t(mp.save)}
            </Button>
          </>
        ) : null
      }
      contentClassName="fb-main"
    >
      <div className="fb-cols">
        <aside className="fb-left" aria-label={t(mf.settingsAside)}>
          <PageLanguageSection value={language} />
          {lang.status !== 'live' && (started || lang.status === 'incomplete') ? (
            <div className="fb-sec">
              <div className="note w" role="note">
                <span className="pl-note-body">{t(mp.incomplete, { language: name(language), fallback, count: lang.missing })}</span>
              </div>
            </div>
          ) : null}
          {!started ? (
            <div className="fb-sec">
              <div className="note w" role="note">
                <svg width="16" height="16" viewBox="0 0 16 16" fill="none" stroke="#B45309" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
                  <circle cx="8" cy="8" r="6.2" />
                  <path d="M8 4.8v3.6M8 10.9v.1" />
                </svg>
                <div className="pl-note-body flex flex-col gap-1">
                  <span className="font-bold">{t(mp.notSetUpTitle, { language: name(language) })}</span>
                  <div>
                    {lang.labels.shipped ? t(mp.notSetUpText, { language: name(language), fallback }) : t(mp.notSetUpTextLabels, { language: name(language), fallback })}
                    {strings.length ? (
                      <div className="mt-2 flex flex-wrap gap-2">
                        <Button size="sm" variant="primary" onClick={copySource} disabledReason={reason}>
                          {t(mp.copySource, { language: source })}
                        </Button>
                        <Button size="sm" onClick={() => setStarted(true)} disabledReason={reason}>
                          {t(mp.fromScratch)}
                        </Button>
                      </div>
                    ) : null}
                  </div>
                </div>
              </div>
            </div>
          ) : null}
          {started && strings.length === 0 ? (
            <div className="fb-sec">
              <span className="k">{t(mp.noOwnText, { language: name(language) })}</span>
            </div>
          ) : null}
          {started
            ? sections.map((sec) => (
                <div key={sec.title} className="fb-sec">
                  <span className="k">{sec.title}</span>
                  {sec.items.map((s) => {
                    const id = `pc-${s.key.replace(/[^a-z0-9]/gi, '-')}`
                    const err = errors[s.key]
                    const common = {
                      id,
                      className: 'fb-in',
                      lang: language,
                      value: draft[s.key] ?? '',
                      placeholder: s.source,
                      disabled: readOnly,
                      'aria-invalid': err ? true : undefined,
                      'aria-describedby': `${id}-src${err ? ` ${id}-err` : ''}`,
                      onChange: (e: { target: { value: string } }) => setDraft((d) => ({ ...d, [s.key]: e.target.value })),
                    }
                    return (
                      <div key={s.key} className="lbl">
                        <label htmlFor={id}>
                          {label(s)}
                          {s.optional ? <span className="k font-normal"> {t(mp.optional)}</span> : null}
                        </label>
                        {long(s) ? <textarea {...common} maxLength={600} /> : <input {...common} maxLength={600} />}
                        <span className="k font-normal" id={`${id}-src`}>
                          {t(mp.sourceText, { language: source, text: s.source })}
                        </span>
                        {err ? (
                          <span className="err" id={`${id}-err`} role="alert">
                            {err}
                          </span>
                        ) : null}
                      </div>
                    )
                  })}
                </div>
              ))
            : null}
          <div className="fb-sec last">
            <BuiltinLabelsSection code={language} labels={lang.labels} />
          </div>
        </aside>
        <section className="fb-center" aria-label={t(mp.preview)}>
          <div className="fb-bar">
            <span className="k">{t(mp.previewWhere, { where: previewURL ? hostPath(previewURL) : previewLabel })}</span>
            <div className="ml-auto flex flex-wrap items-center gap-2.5">
              {extraBar}
              <div className="fb-seg" role="group" aria-label={t(mf.previewSize)}>
                <button type="button" aria-pressed={device === 'desktop'} onClick={() => setDevice('desktop')}>
                  {t(mf.desktop)}
                </button>
                <button type="button" aria-pressed={device === 'mobile'} onClick={() => setDevice('mobile')}>
                  {t(mf.mobile)}
                </button>
              </div>
            </div>
          </div>
          {preview(device)}
        </section>
      </div>
      <UnsavedChangesGuard dirty={dirty && !readOnly} description={t(mp.leaveText, { language: name(language) })} saveLabel={t(mp.saveAndLeave)} onSave={submit} />
    </Page>
  )
}
