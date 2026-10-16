import { useNavigate } from '@tanstack/react-router'
import { useEffect, useRef, useState } from 'react'
import { $api, ApiError, http } from '@/api/client'
import { ForbiddenState, NotFoundState, problemReference, ServerErrorState } from '@/components/shell/SystemScreens'
import { ButtonLink } from '@/components/ui/Button'
import { useT } from '@/i18n'
import { useDocumentTitle } from '@/lib/hooks'
import { useCan, useWorkspace } from '@/lib/session'
import { ComposerSkeleton } from '../ComposerPage'
import { mt } from './page.messages'
import { TransactionalComposer, type TxSearch } from './TransactionalComposer'
import { useOpenTemplateLanguage } from '@/features/languages/composerParts'
import '../composer.css'

/** 403, 404 and 500 of the full-screen editor, in a centred frame (as the composer's). */
function EditorError({ error, path, retry }: { error: unknown; path: string; retry?: () => void }) {
  const t = useT()
  const { slug } = useWorkspace()
  const back = (
    <ButtonLink to="/w/$workspace/transactional" params={{ workspace: slug }}>
      {t(mt.backToTemplates)}
    </ButtonLink>
  )
  const status = error instanceof ApiError ? error.status : 0
  useDocumentTitle(t(status === 404 ? mt.titleNotFound : status === 403 ? mt.titleForbidden : mt.titleBroken))
  let body
  if (error instanceof ApiError && status === 403 && error.problem.permission) {
    body = <ForbiddenState permission={error.problem.permission} title={path.endsWith('/new') ? t(mt.cantCreate) : t(mt.cantOpen)} readOnly={back} />
  } else if (status === 404) {
    body = (
      <NotFoundState
        path={path}
        reference={problemReference(error)}
        primary={
          <ButtonLink variant="primary" to="/w/$workspace/transactional" params={{ workspace: slug }}>
            {t(mt.goToTemplates)}
          </ButtonLink>
        }
      >
        {t(mt.notFoundBody)}
      </NotFoundState>
    )
  } else {
    body = <ServerErrorState reference={problemReference(error)} onRetry={retry ?? (() => window.location.reload())} back={back} />
  }
  return (
    <div className="cmp rc-app items-center" style={{ padding: '56px 16px' }}>
      <main id="main" className="w-full max-w-[640px]">
        {body}
      </main>
    </div>
  )
}

/**
 * /w/$workspace/transactional/templates/$templateId: loads the template,
 * Settings › Transactional (senders, product name), the categories and, for
 * members who can read campaigns, the composer's context (the footer
 * address). Then the editor.
 */
export function TransactionalTemplatePage({ templateId, search, onSearch }: { templateId: string; search: TxSearch; onSearch: (s: TxSearch) => void }) {
  const t = useT()
  const { slug } = useWorkspace()
  const canCampaigns = useCan('campaigns:view')
  // ?language= opens a language version (internal/languages).
  const lang = useOpenTemplateLanguage(templateId, search.language)
  const path = { params: { path: { workspace: slug, template: templateId }, query: lang.load ? { language: lang.load } : undefined } }
  const tpl = $api.useQuery('get', '/api/v1/workspaces/{workspace}/transactional/templates/{template}', path, {
    refetchOnWindowFocus: false,
    retry: (n, e) => !(e as unknown as ApiError)?.status && n < 2,
    enabled: lang.ready,
  })
  const settings = $api.useQuery('get', '/api/v1/workspaces/{workspace}/transactional/settings', { params: { path: { workspace: slug } } }, { staleTime: 60_000 })
  const categories = $api.useQuery('get', '/api/v1/workspaces/{workspace}/transactional/categories', { params: { path: { workspace: slug } } }, { staleTime: 60_000 })
  const context = $api.useQuery('get', '/api/v1/workspaces/{workspace}/composer', { params: { path: { workspace: slug } } }, { staleTime: 60_000, enabled: canCampaigns })
  useDocumentTitle(tpl.data ? t(mt.documentTitle, { name: tpl.data.name }) : t(mt.documentTitleLoading))
  if (tpl.isError) return <EditorError error={tpl.error} path={`/transactional/templates/${templateId}`} retry={() => void tpl.refetch()} />
  const waiting = !tpl.data || settings.isPending || categories.isPending || (canCampaigns && context.isPending)
  if (waiting) return <ComposerSkeleton title={t(mt.opening)} detail={t(mt.openingDetail)} />
  return (
    <TransactionalComposer
      key={`${tpl.data.id}:${tpl.data.language}`}
      template={tpl.data}
      settings={settings.data}
      categories={categories.data?.items ?? []}
      context={context.data}
      search={search}
      onSearch={onSearch}
    />
  )
}

/**
 * /w/$workspace/transactional/templates/new: creates the draft (blank, from a
 * starter, or a copy of another template), then opens it.
 */
export function NewTransactionalTemplatePage({ starter, from, name }: { starter?: string; from?: string; name?: string }) {
  const t = useT()
  const { slug } = useWorkspace()
  const navigate = useNavigate()
  const started = useRef(false)
  const [error, setError] = useState<unknown>(null)
  const [attempt, setAttempt] = useState(0)
  useDocumentTitle(t(mt.newTitle))
  useEffect(() => {
    if (started.current) return
    started.current = true
    http
      .POST('/api/v1/workspaces/{workspace}/transactional/templates', {
        params: { path: { workspace: slug } },
        body: { name: name?.trim() || t(mt.untitled), starter, fromTemplate: from },
      })
      .then(({ data }) => {
        if (data) void navigate({ to: '/w/$workspace/transactional/templates/$templateId', params: { workspace: slug, templateId: data.id }, replace: true })
      })
      .catch((e: unknown) => setError(e))
  }, [slug, starter, from, name, navigate, attempt, t])
  if (error)
    return (
      <EditorError
        error={error}
        path="/transactional/templates/new"
        retry={() => {
          started.current = false
          setError(null)
          setAttempt((n) => n + 1)
        }}
      />
    )
  return <ComposerSkeleton title={t(mt.creating)} detail={t(mt.creatingDetail)} />
}
