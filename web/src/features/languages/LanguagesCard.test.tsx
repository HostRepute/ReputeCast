import { fireEvent, screen, within } from '@testing-library/react'
import type { Schemas } from '@/api/client'
import { json, mockApi } from '@/test/api'
import { renderInWorkspace, workspaceMe } from '@/test/app'
import { LanguagesCard } from './LanguagesCard'

type Lang = Schemas['WorkspaceLanguage']

const lang = (patch: Partial<Lang> & Pick<Lang, 'code' | 'name'>): Lang => ({
  native: '',
  codes: [patch.code],
  moreCodes: false,
  contacts: 0,
  own: 0,
  isDefault: false,
  isFallback: false,
  pageCopy: { status: 'live', missing: 0, by: 'Sofia Rossi', at: '2026-10-08T15:40:00Z', builtinLabels: true },
  addedBy: 'Linh Nguyen',
  addedAt: '2026-10-06T14:05:00Z',
  ...patch,
})

/** The SettingsLanguages board's languages. */
const board = (canManage: boolean): Schemas['WorkspaceLanguages'] => ({
  default: 'en',
  fallback: 'en',
  totalContacts: 48213,
  noLanguage: 5508,
  notOffered: 12,
  notOfferedCodes: ['pt-BR', 'fr', 'it'],
  canManage,
  helpers: canManage ? [] : ['Marcus Chen', 'Linh Nguyen'],
  languages: [
    lang({ code: 'en', name: 'English', contacts: 41020, own: 35500, isDefault: true, isFallback: true, addedBy: '', addedAt: undefined,
      pageCopy: { status: 'live', missing: 0, by: '', builtinLabels: true } }),
    lang({ code: 'es', name: 'Spanish', native: 'Español', codes: ['es', 'es-MX', 'es-ES'], moreCodes: true, contacts: 4180, own: 4180 }),
    lang({ code: 'de', name: 'German', native: 'Deutsch', codes: ['de', 'de-AT', 'de-CH'], moreCodes: true, contacts: 2240, own: 2240 }),
    lang({ code: 'vi', name: 'Vietnamese', native: 'Tiếng Việt', contacts: 773, own: 773, addedAt: '2026-10-13T20:45:00Z',
      pageCopy: { status: 'not_set_up', missing: 0, by: '', builtinLabels: true } }),
  ],
})

const manager = workspaceMe({ permissions: ['workspace:manage', 'campaigns:edit'], workspace: { ...workspaceMe().workspace, roleKey: 'owner', roleName: 'Owner' } })

describe('Settings › General › Languages', () => {
  afterEach(() => mockApi({}))

  it('lists the languages as SettingsLanguages does', async () => {
    mockApi({ 'GET /api/v1/workspaces/acme-newsletter/languages': () => json(board(true)) })
    await renderInWorkspace(<LanguagesCard />, { me: manager, path: '/w/acme-newsletter/settings' })
    const table = await screen.findByRole('table', { name: 'Workspace languages' })
    const rows = within(table).getAllByRole('row')
    expect(rows[1]).toHaveTextContent('English')
    expect(rows[1]).toHaveTextContent('Default')
    expect(rows[1]).toHaveTextContent('41,020')
    expect(rows[2]).toHaveTextContent('35,500 English · 5,508 with no language · 12 other languages (pt-BR, fr, it) · all get English')
    expect(within(table).getByText('es, es-MX, es-ES…')).toBeInTheDocument()
    expect(within(table).getByText('Built-in labels are ready. Acme’s own text isn’t written, so visitors see English.')).toBeInTheDocument()
    expect(within(table).getByRole('link', { name: 'Write Vietnamese page copy' })).toBeInTheDocument()
    expect(screen.getByText('48,213 contacts in all · a regional code matches its language: es-MX gets Spanish')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Add language' })).toBeInTheDocument()
    expect(screen.getByLabelText('Fallback language')).not.toBeDisabled()
  })

  it('is read-only without workspace:manage, and says who can change it', async () => {
    mockApi({ 'GET /api/v1/workspaces/acme-newsletter/languages': () => json(board(false)) })
    await renderInWorkspace(<LanguagesCard />, { path: '/w/acme-newsletter/settings' })
    const note = await screen.findByRole('note')
    expect(note).toHaveTextContent('Changing workspace languages needs workspace:manage. Ask Marcus Chen or Linh Nguyen.')
    expect(screen.queryByRole('button', { name: 'Add language' })).not.toBeInTheDocument()
    expect(screen.getByLabelText('Default language')).toBeDisabled()
    expect(screen.getAllByRole('link', { name: 'View page copy' }).length).toBeGreaterThan(0)
  })

  it('shows what a new fallback moves before saving it', async () => {
    mockApi({
      'GET /api/v1/workspaces/acme-newsletter/languages': () => json(board(true)),
      'GET /api/v1/workspaces/acme-newsletter/languages/fallback-impact': () => json({ fallback: 'es', moved: 5520, noLanguage: 5508, notOffered: 12 }),
    })
    await renderInWorkspace(<LanguagesCard />, { me: manager, path: '/w/acme-newsletter/settings' })
    fireEvent.change(await screen.findByLabelText('Fallback language'), { target: { value: 'es' } })
    expect(await screen.findByText('5,520 contacts would get Spanish where a Spanish version exists.')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Save the fallback' })).toBeInTheDocument()
  })

  it('offers languages to add, and refuses right-to-left ones', async () => {
    mockApi({
      'GET /api/v1/workspaces/acme-newsletter/languages': () => json(board(true)),
      'GET /api/v1/workspaces/acme-newsletter/languages/catalog': (r) =>
        json({
          items: new URL(r.url).searchParams.get('q')
            ? [{ code: 'ar', name: 'Arabic', native: 'العربية', rtl: true, regions: [], contacts: 0, contactCodes: [], builtinLabels: false, added: false }]
            : [
                { code: 'pt', name: 'Portuguese', native: 'Português', rtl: false, regions: ['pt-BR', 'pt-PT'], contacts: 7, contactCodes: ['pt-BR'], builtinLabels: false, added: false },
                { code: 'fr', name: 'French', native: 'Français', rtl: false, regions: ['fr-CA', 'fr-BE'], contacts: 3, contactCodes: ['fr'], builtinLabels: false, added: false },
              ],
        }),
    })
    await renderInWorkspace(<LanguagesCard />, { me: manager, path: '/w/acme-newsletter/settings' })
    fireEvent.click(await screen.findByRole('button', { name: 'Add language' }))
    const dialog = await screen.findByRole('dialog', { name: 'Add a language' })
    expect(await within(dialog).findByText('Contacts with Portuguese: 7, all pt-BR · No built-in page labels yet: about 60 short strings to write once')).toBeInTheDocument()
    expect(within(dialog).getByRole('button', { name: 'Add Portuguese' })).toBeInTheDocument()
    fireEvent.change(within(dialog).getByLabelText('Search languages'), { target: { value: 'ar' } })
    expect(await within(dialog).findByText('Right-to-left languages aren’t supported yet. Contacts with Arabic get English.')).toBeInTheDocument()
    expect(within(dialog).getByRole('radio')).toBeDisabled()
    expect(within(dialog).getByRole('button', { name: 'Add language' })).toBeDisabled()
  })
})
