import { fireEvent, screen, waitFor, within } from '@testing-library/react'
import type { Schemas } from '@/api/client'
import type { Overview, SignupForm } from '@/features/forms/common'
import { json, mockApi } from '@/test/api'
import { renderInWorkspace, workspaceMe } from '@/test/app'
import { FormLanguagesView } from './FormLanguagesView'

type CopyLanguage = Schemas['SignupFormCopyLanguage']

const form = {
  id: 'f1',
  name: 'Newsletter signup',
  kind: 'embedded',
  status: 'live',
  publicId: 'frm_8c2e1a',
  slug: 'newsletter-signup',
  hostedPage: true,
  hostedUrl: 'https://news.acme.io/subscribe',
  fields: [
    { key: 'email', label: '', required: true },
    { key: 'first_name', label: '', required: false },
  ],
  listIds: ['digest', 'updates'],
  defaultListIds: ['digest'],
  listChoice: true,
  doubleOptIn: true,
  consentText: 'I agree to receive emails from Acme. Unsubscribe any time with one click. See the privacy policy.',
  consentVersion: 3,
  spam: { honeypot: true, minSeconds: 3, turnstile: true, blockDisposable: true, rateLimitPerIp: 5 },
  afterSubmit: 'message',
  redirectUrl: '',
  allowedOrigins: ['acme.io'],
  content: {
    heading: 'The Acme newsletter',
    description: 'One email a week.',
    button: 'Subscribe',
    footnote: 'No spam, ever.',
    listsLabel: 'What would you like?',
    emailPlaceholder: 'you@company.com',
    namePlaceholder: 'Sam',
    popupDelay: 8,
    popupScroll: 0,
    popupRepeatDays: 14,
  },
} as unknown as SignupForm

const overview = {
  forms: [],
  lists: [
    { id: 'digest', name: 'Weekly Digest', description: '', public: true, subscribed: 31988 },
    { id: 'updates', name: 'Product Updates', description: '', public: true, subscribed: 14110 },
  ],
  customFields: [],
  hostedUrl: 'https://news.acme.io/subscribe',
  hostedFormId: 'f1',
  branding: { publisherName: 'Acme', accent: '#4338CA', onAccent: '#FFFFFF', accentText: '#4338CA', privacyUrl: 'https://acme.io/privacy', initial: 'A' },
} as unknown as Overview

const language = (patch: Partial<CopyLanguage> & Pick<CopyLanguage, 'code' | 'native'>): CopyLanguage => ({
  name: patch.code,
  isDefault: false,
  isSource: false,
  status: 'live',
  missing: 0,
  written: 5,
  total: 5,
  by: 'Sofia Rossi',
  at: '2026-10-08T12:00:00Z',
  copy: { heading: '', description: '', button: '', consentText: '', afterSubmit: '', fields: {} },
  consentVersion: 1,
  consentFor: 3,
  consentReview: false,
  labels: { shipped: false, written: 193, total: 193, by: 'Sofia Rossi', at: '2026-10-08T12:00:00Z' },
  defaultAfterSubmit: '',
  ...patch,
})

const copy = (review = false): Schemas['SignupFormCopy'] => ({
  formId: 'f1',
  languageMode: 'page',
  fixedLanguage: '',
  copyLanguage: 'en',
  fallback: 'en',
  consentVersion: review ? 4 : 3,
  consentChangedBy: 'Linh Nguyen',
  fields: [],
  languages: [
    language({
      code: 'en',
      native: 'English',
      isDefault: true,
      isSource: true,
      consentVersion: review ? 4 : 3,
      consentFor: review ? 4 : 3,
      by: '',
      copy: { heading: 'The Acme newsletter', description: 'One email a week.', button: 'Subscribe', consentText: form.consentText, afterSubmit: '', fields: {} },
      labels: { shipped: true, written: 193, total: 193, by: '' },
      defaultAfterSubmit: 'We sent you a link to confirm your subscription. It only takes a click.',
    }),
    language({
      code: 'es',
      native: 'Español',
      consentReview: review,
      copy: {
        heading: 'El boletín de Acme',
        description: 'Un correo a la semana.',
        button: 'Suscribirme',
        consentText: 'Acepto recibir correos de Acme. Consulta la política de privacidad.',
        afterSubmit: 'Revisa tu correo para confirmar.',
        fields: {},
      },
      evidence: { contactId: 'c1', name: 'Lucía Fernández', at: '2026-10-03T12:00:00Z', version: 'v1' },
    }),
    language({
      code: 'vi',
      native: 'Tiếng Việt',
      status: 'not_set_up',
      missing: 5,
      written: 0,
      by: '',
      at: undefined,
      consentVersion: 0,
      consentFor: 0,
      labels: { shipped: true, written: 193, total: 193, by: '' },
    }),
  ],
})

const labels = (code: string) =>
  json({
    language: code,
    shipped: code !== 'es',
    written: 193,
    total: 193,
    by: '',
    labels: [
      { key: 'form.email', source: 'Email', arguments: [], text: code === 'es' ? 'Correo electrónico' : 'Email' },
      { key: 'form.privacy', source: 'privacy policy', arguments: [], text: code === 'es' ? 'política de privacidad' : 'privacy policy' },
    ],
  })

const editor = workspaceMe({ permissions: ['subscribers:view', 'forms:edit'] })

function api(data: Schemas['SignupFormCopy'], extra: Record<string, (r: Request) => Response | Promise<Response>> = {}) {
  mockApi({
    'GET /api/v1/workspaces/acme-newsletter/forms/f1/copy': () => json(data),
    'GET /api/v1/workspaces/acme-newsletter/public-pages/labels/es': () => labels('es'),
    'GET /api/v1/workspaces/acme-newsletter/public-pages/labels/en': () => labels('en'),
    'GET /api/v1/workspaces/acme-newsletter/public-pages/labels/vi': () => labels('vi'),
    'GET /api/v1/workspaces/acme-newsletter/public-pages/copy': () => json({ copyLanguage: 'en', default: 'en', fallback: 'en', languages: [], language: 'es', strings: [] }),
    ...extra,
  })
}

describe('the form builder in a language', () => {
  afterEach(() => mockApi({}))

  it('shows the Spanish copy as FormBuilderLanguages does, and saves it', async () => {
    let saved: Record<string, unknown> | undefined
    api(copy(), {
      'PUT /api/v1/workspaces/acme-newsletter/forms/f1/copy/es': async (r) => {
        saved = (await r.json()) as Record<string, unknown>
        return json(copy())
      },
    })
    await renderInWorkspace(<FormLanguagesView form={form} overview={overview} language="es" />, { me: editor, path: '/w/acme-newsletter/forms' })
    const switcher = await screen.findByRole('group', { name: 'Copy in' })
    expect(within(switcher).getByRole('button', { name: 'Español Live' })).toHaveAttribute('aria-pressed', 'true')
    expect(within(switcher).getByRole('button', { name: 'Tiếng Việt Not set up' })).toHaveAttribute('aria-pressed', 'false')
    expect(screen.getByLabelText('Heading')).toHaveValue('El boletín de Acme')
    expect(screen.getByText('Spanish · v1')).toBeInTheDocument()
    expect(screen.getByText('English is v3. Each signup keeps the exact text and the language it was shown in.')).toBeInTheDocument()
    expect(screen.getByRole('radio', { name: /Match the page it’s on/ })).toBeChecked()
    expect(screen.getByRole('link', { name: 'Lucía Fernández’s consent, 3 Oct' })).toBeInTheDocument()
    fireEvent.change(screen.getByLabelText('Heading'), { target: { value: 'El boletín semanal de Acme' } })
    fireEvent.click(screen.getByRole('button', { name: 'Save & publish' }))
    await waitFor(() => expect(saved?.heading).toBe('El boletín semanal de Acme'))
    expect(saved?.button).toBe('Suscribirme')
  })

  it('offers to start a language that isn’t set up from English', async () => {
    api(copy())
    await renderInWorkspace(<FormLanguagesView form={form} overview={overview} language="vi" />, { me: editor, path: '/w/acme-newsletter/forms' })
    const note = await screen.findByRole('note')
    expect(note).toHaveTextContent('Vietnamese copy isn’t set up')
    expect(screen.getByText('Ready, from ReputeCast’s translation')).toBeInTheDocument()
    fireEvent.click(within(note).getByRole('button', { name: 'Copy English' }))
    expect(screen.getByLabelText('Heading')).toHaveValue('The Acme newsletter')
    expect(screen.getByLabelText('After submit')).toHaveValue('We sent you a link to confirm your subscription. It only takes a click.')
  })

  it('flags a translation after the form’s own consent text changes', async () => {
    let kept = false
    api(copy(true), {
      'POST /api/v1/workspaces/acme-newsletter/forms/f1/copy/es/consent-reviewed': () => {
        kept = true
        return json(copy())
      },
    })
    await renderInWorkspace(<FormLanguagesView form={form} overview={overview} language="es" />, { me: editor, path: '/w/acme-newsletter/forms' })
    expect(await screen.findByText('Consent text · after the English text changes to v4')).toBeInTheDocument()
    expect(screen.getByText('v4 · changed by Linh Nguyen')).toBeInTheDocument()
    expect(screen.getByText('Review the translation')).toBeInTheDocument()
    fireEvent.click(screen.getByRole('button', { name: 'Spanish is still right' }))
    await waitFor(() => expect(kept).toBe(true))
  })
})
