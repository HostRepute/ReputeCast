import { fireEvent, screen, waitFor, within } from '@testing-library/react'
import type { Schemas } from '@/api/client'
import { json, mockApi } from '@/test/api'
import { renderInWorkspace, workspaceMe } from '@/test/app'
import { SendingRulesPage } from './RulesPage'

type View = Schemas['SendingRulesView']
type Rule = Schemas['SendingRule']

const rule = (patch: Partial<Rule> & Pick<Rule, 'id' | 'name' | 'kind' | 'operator' | 'value'>): Rule => ({
  severity: 'warn',
  appliesTo: 'broadcast',
  enabled: true,
  message: '',
  description: '',
  addedBy: 'Marcus Chen',
  addedAt: '2026-09-23T14:05:00Z',
  ...patch,
})

/** The SettingsSendingRules board's rules and defaults. */
const board = (canManage: boolean): View => ({
  rules: [
    rule({
      id: 'r1',
      name: 'Links need UTM tags',
      kind: 'link_param',
      operator: 'on_every',
      value: 'utm_campaign',
      description: 'Every link has a utm_campaign query parameter',
      triggered: { at: '2026-10-13T13:40:00Z', campaign: 'Weekly Digest #142', campaignId: 'c142', count: 2, fixed: false },
    }),
    rule({
      id: 'r2',
      name: 'No URL shorteners',
      kind: 'link_domain',
      operator: 'not_one_of',
      value: 'shorteners',
      severity: 'block',
      appliesTo: 'both',
      description: 'No link points to bit.ly, t.co, tinyurl.com or 37 other shortener domains',
      triggered: { at: '2026-10-02T15:20:00Z', campaign: 'Feature flags preview', count: 1, fixed: true },
    }),
    rule({ id: 'r4', name: 'Footer must include the postal address', kind: 'contains_text', operator: 'footer_contains', value: '500 Market St', severity: 'block', enabled: false }),
  ],
  defaults: { excludeUnengaged: true, typedMultiple: 1.5 },
  sunsetDays: 180,
  canManage,
  managers: ['Linh Nguyen', 'Marcus Chen'],
  reason: canManage ? undefined : "Your role can't change the sending rules: Linh Nguyen or Marcus Chen can.",
})

const owner = workspaceMe({ permissions: ['workspace:manage', 'campaigns:view'], workspace: { ...workspaceMe().workspace, roleKey: 'owner', roleName: 'Owner' } })
const path = '/w/acme-newsletter/settings/sending-rules'

describe('Settings › Sending rules', () => {
  afterEach(() => mockApi({}))

  it('lists the rules as SettingsSendingRules does', async () => {
    mockApi({ 'GET /api/v1/workspaces/acme-newsletter/sending-rules': () => json(board(true)) })
    await renderInWorkspace(<SendingRulesPage />, { me: owner, path })
    const table = await screen.findByRole('table', { name: 'Workspace rules, 2 on' })
    const rows = within(table).getAllByRole('row')
    expect(rows[1]).toHaveTextContent('Links need UTM tags')
    expect(rows[1]).toHaveTextContent('Every link has a utm_campaign query parameter')
    expect(rows[1]).toHaveTextContent('Weekly Digest #142 · 2 links')
    expect(rows[2]).toHaveTextContent('Feature flags preview · fixed before sending')
    expect(rows[3]).toHaveTextContent('Never')
    expect(within(rows[3]!).getByRole('switch', { name: 'Footer must include the postal address' })).not.toBeChecked()
    expect(within(rows[2]!).getByLabelText('Severity, No URL shorteners')).toHaveValue('block')
    expect(screen.getByText('Owners and Admins can change rules: Linh Nguyen and Marcus Chen.')).toBeInTheDocument()
    expect(screen.getByRole('switch', { name: /^Exclude unengaged contacts per sunset policy \(180 days\)/ })).toBeChecked()
  })

  it('is read-only without the right to change rules, and says who can', async () => {
    mockApi({ 'GET /api/v1/workspaces/acme-newsletter/sending-rules': () => json(board(false)) })
    await renderInWorkspace(<SendingRulesPage />, { path })
    await screen.findByRole('table', { name: 'Workspace rules, 2 on' })
    expect(screen.getAllByLabelText(/^Severity, /)[0]).toBeDisabled()
    expect(screen.getByRole('button', { name: 'Edit Links need UTM tags' })).toBeDisabled()
    expect(document.body.textContent).not.toContain('workspace:manage')
  })

  it('adds a rule with its condition, severity and scope', async () => {
    let posted: unknown
    mockApi({
      'GET /api/v1/workspaces/acme-newsletter/sending-rules': () => json(board(true)),
      'POST /api/v1/workspaces/acme-newsletter/sending-rules': async (r) => {
        posted = await r.json()
        return json(rule({ id: 'r5', name: 'Subject under 70 characters', kind: 'subject_length', operator: 'at_most', value: '70' }), 201)
      },
    })
    await renderInWorkspace(<SendingRulesPage />, { me: owner, path })
    const form = await screen.findByRole('region', { name: 'Add a rule' })
    fireEvent.change(within(form).getByLabelText('Rule name'), { target: { value: 'Subject under 70 characters' } })
    fireEvent.change(within(form).getByLabelText('Check'), { target: { value: 'subject_length' } })
    fireEvent.change(within(form).getByLabelText('Operator'), { target: { value: 'at_most' } })
    fireEvent.change(within(form).getByLabelText('Characters'), { target: { value: '70' } })
    fireEvent.click(within(form).getByRole('radio', { name: /Block/ }))
    fireEvent.click(within(form).getByRole('button', { name: 'Add rule' }))
    await waitFor(() => expect(posted).toBeDefined())
    expect(posted).toMatchObject({ name: 'Subject under 70 characters', kind: 'subject_length', operator: 'at_most', value: '70', severity: 'block', appliesTo: 'both', enabled: true })
    expect(await screen.findByText('Rule added: Subject under 70 characters')).toBeInTheDocument()
  })

  it('saves the audience defaults', async () => {
    let saved: unknown
    mockApi({
      'GET /api/v1/workspaces/acme-newsletter/sending-rules': () => json(board(true)),
      'PUT /api/v1/workspaces/acme-newsletter/send-defaults': async (r) => {
        saved = await r.json()
        return json({ ...board(true), defaults: { excludeUnengaged: false, typedMultiple: 2 } })
      },
    })
    await renderInWorkspace(<SendingRulesPage />, { me: owner, path })
    fireEvent.click(await screen.findByRole('switch', { name: /^Exclude unengaged contacts per sunset policy \(180 days\)/ }))
    const multiple = screen.getByLabelText('Ask for a typed confirmation when the audience is more than')
    fireEvent.change(multiple, { target: { value: '25' } })
    fireEvent.click(screen.getByRole('button', { name: 'Save' }))
    expect(await screen.findByRole('alert')).toHaveTextContent('1.1')
    fireEvent.change(multiple, { target: { value: '2' } })
    fireEvent.click(screen.getByRole('button', { name: 'Save' }))
    await waitFor(() => expect(saved).toEqual({ excludeUnengaged: false, typedMultiple: 2 }))
  })
})
