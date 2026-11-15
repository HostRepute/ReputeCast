import { fireEvent, screen, waitFor, within } from '@testing-library/react'
import { afterEach, expect, it } from 'vitest'
import { json, mockApi } from '@/test/api'
import { renderInWorkspace, workspaceMe } from '@/test/app'
import { BudgetHolds } from './BudgetHolds'

const base = '/api/v1/workspaces/acme-newsletter/ai/budget-holds'
let unmock: (() => void) | undefined
afterEach(() => unmock?.())

it('requires verified counts and explicit confirmation; a rejected decision preserves the form', async () => {
  const decisions: unknown[] = []
  unmock = mockApi({
    [`GET ${base}`]: () => json({ items: [{ id: 'request-1', model: 'Original model', provider: 'compatible', createdAt: '2026-01-01T09:00:00Z', reconcileAfter: '2026-01-01T09:10:00Z', reservedTokens: 2100, reservedCostMicros: 51000 }] }),
    [`POST ${base}/request-1/reconcile`]: async (r) => { decisions.push(await r.json()); return json({ status: 409, detail: 'Already reconciled. Refresh to see the recorded usage.' }, 409) },
  })
  await renderInWorkspace(<BudgetHolds />, { me: workspaceMe({ permissions: ['integrations:view', 'integrations:manage'] }) })
  fireEvent.click(await screen.findByRole('button', { name: 'Reconcile usage' }))
  const dialog = within(screen.getByRole('dialog'))
  const save = dialog.getByRole('button', { name: 'Reconcile usage' })
  expect(save).toBeDisabled()
  fireEvent.change(dialog.getByLabelText('Verified input tokens'), { target: { value: '17' } })
  fireEvent.change(dialog.getByLabelText('Verified output tokens'), { target: { value: '11' } })
  expect(save).toBeDisabled()
  expect(decisions).toEqual([])
  fireEvent.click(dialog.getByRole('checkbox'))
  fireEvent.click(save)
  expect(await dialog.findByRole('alert')).toHaveTextContent('Already reconciled')
  expect(decisions).toEqual([{ inputTokens: 17, outputTokens: 11, confirmed: true }])
  expect(dialog.getByLabelText('Verified input tokens')).toHaveValue(17)
  expect(dialog.getByLabelText('Verified output tokens')).toHaveValue(11)
  fireEvent.click(dialog.getByRole('button', { name: 'Cancel' }))
  await waitFor(() => expect(screen.queryByRole('dialog')).not.toBeInTheDocument())
})
