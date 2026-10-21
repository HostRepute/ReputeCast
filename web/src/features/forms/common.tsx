import { useQueryClient } from '@tanstack/react-query'
import { useRouterState } from '@tanstack/react-router'
import { $api, ApiError, type Schemas } from '@/api/client'
import { Badge } from '@/components/ui/Badge'
import { NavTabs } from '@/components/ui/Tabs'
import { useT } from '@/i18n'
import { useWorkspace } from '@/lib/session'
import { m as tabs } from './tabs.messages'
import './forms.css'

export type Overview = Schemas['SignupFormsOverview']
export type SignupForm = Schemas['SignupForm']
export type FormInput = Schemas['SignupFormInput']
export type PublicPages = Schemas['PublicPagesSettings']
export type Branding = Schemas['PublicBranding']

/** The forms, lists, fields and branding behind every Forms & public pages tab. */
export function useFormsOverview() {
  const { slug } = useWorkspace()
  return $api.useQuery('get', '/api/v1/workspaces/{workspace}/forms', { params: { path: { workspace: slug } } })
}

/** The public pages' branding, options, domain and Turnstile set-up. */
export function usePublicPages() {
  const { slug } = useWorkspace()
  return $api.useQuery('get', '/api/v1/workspaces/{workspace}/public-pages', { params: { path: { workspace: slug } } })
}

/** Refreshes every query of this area after a change. */
export function useInvalidateForms() {
  const qc = useQueryClient()
  return () =>
    qc.invalidateQueries({
      predicate: (q) => {
        const k = q.queryKey[1]
        return typeof k === 'string' && (k.includes('/forms') || k.includes('/public-pages'))
      },
    })
}

/**
 * The Forms & pages tabs, inline after the screen title (they navigate):
 * Signup forms · Website · Pages · Branding & domain (design expansion
 * phase 7, FormBuilder and FormsStates). Surveys open under Signup forms;
 * the built-in pages' editors (subscribe page, preference center,
 * unsubscribe page, archive) under Pages.
 */
export function FormsTabs() {
  const t = useT()
  const { slug } = useWorkspace()
  const path = useRouterState({ select: (s) => s.location.pathname })
  const onSurvey = path.includes('/forms/surveys/')
  const builtin = /\/forms\/(subscribe|preferences|unsubscribe|archive)(\/|$)/.test(path)
  const params = { workspace: slug }
  return (
    <NavTabs
      label={t(tabs.label)}
      className="ml-3 max-md:ml-0"
      tabs={[
        { label: t(tabs.signupForms), to: '/w/$workspace/forms', params, exact: true, current: onSurvey || undefined },
        { label: t(tabs.website), to: '/w/$workspace/forms/website', params },
        { label: t(tabs.pages), to: '/w/$workspace/forms/pages', params, current: builtin || undefined },
        { label: t(tabs.branding), to: '/w/$workspace/forms/branding', params },
      ]}
    />
  )
}

/** The form's state in the top bar: Published · news.acme.io/subscribe. */
export function FormStatusBadge({ form }: { form: SignupForm }) {
  const where = hostPath(form.hostedUrl)
  switch (form.status) {
    case 'live':
      return (
        <Badge tone="ok" dot>
          Published · {where}
        </Badge>
      )
    case 'paused':
      return (
        <Badge tone="warn" dot>
          Paused · {where}
        </Badge>
      )
    default:
      return (
        <Badge tone="neutral" dot>
          Draft · not published yet
        </Badge>
      )
  }
}

/** "news.acme.io/subscribe" for "https://news.acme.io/subscribe". */
export function hostPath(url: string): string {
  return url.replace(/^https?:\/\//, '').replace(/\/$/, '')
}

/** A load error that is a missing permission goes to the shell's 403 state. */
export function throwIfForbidden(error: unknown) {
  if (error instanceof ApiError && error.status === 403) throw error
}

/** Copies text and says so; falls back to selecting it for the person. */
export async function copyText(text: string): Promise<boolean> {
  try {
    await navigator.clipboard.writeText(text)
    return true
  } catch {
    return false
  }
}

/** The publisher's mark: the uploaded logo, else the initial on an ink tile. */
export function PublisherMark({ branding, className, logoClassName }: { branding: Branding; className: string; logoClassName: string }) {
  if (branding.logo) {
    return (
      <span className={`${className} ${logoClassName}`} aria-hidden="true">
        <img src={branding.logo.url} alt="" />
      </span>
    )
  }
  return (
    <span className={className} aria-hidden="true">
      {branding.initial}
    </span>
  )
}

export const builtinFieldLabels: Record<string, string> = {
  email: 'Email',
  first_name: 'First name',
  last_name: 'Last name',
  company: 'Company',
}

/** A field's label: the form's own, else the built-in or custom field's name. */
export function fieldLabel(f: Schemas['SignupFormField'], customs: Schemas['SignupFormCustomField'][]): string {
  if (f.label) return f.label
  if (builtinFieldLabels[f.key]) return builtinFieldLabels[f.key]!
  return customs.find((c) => c.key === f.key)?.label ?? f.key
}

/** The input a form edits, from the stored form. */
export function inputOf(f: SignupForm): FormInput {
  return {
    name: f.name,
    // The builder never loads a push prompt (X22): those open in their designer.
    kind: f.kind === 'push' ? 'embedded' : f.kind,
    fields: f.fields.map((x) => ({ ...x })),
    listIds: [...f.listIds],
    defaultListIds: [...f.defaultListIds],
    listChoice: f.listChoice,
    doubleOptIn: f.doubleOptIn,
    consentText: f.consentText,
    spam: { ...f.spam },
    afterSubmit: f.afterSubmit,
    redirectUrl: f.redirectUrl,
    allowedOrigins: [...f.allowedOrigins],
    content: { ...f.content },
    publish: true,
  }
}

/** Whether two inputs differ (the Discard and leave guards). */
export function sameInput(a: FormInput, b: FormInput): boolean {
  return JSON.stringify(a) === JSON.stringify(b)
}

/** The embed snippet, laid out as on the board. */
export function embedSnippet(publicId: string, script: string): string {
  return `<div data-reputecast-form="${publicId}"></div>\n<script async\n  src="${script}">\n</script>`
}
