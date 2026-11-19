import { screen } from '@testing-library/react'
import { describe, expect, it } from 'vitest'
import { renderInWorkspace } from '@/test/app'
import { buildItems, Conversation, type Item } from './Conversation'
import { initialThreadState } from './reducer'

// What the server leaves out once the person can't read a source any more:
// a tool row reads "no longer available to you", and a summary made from
// it is the server's placeholder, which the panel shows as the marker alone.

const at = '2026-11-18T10:00:00Z'

describe('a conversation whose sources the person can no longer read', () => {
  it('labels the rows no longer available, built-in and MCP tools alike', async () => {
    const items: Item[] = [
      {
        kind: 'answer',
        key: 'a1',
        turnId: 't1',
        texts: [],
        tools: [
          { callId: 'c1', name: 'read_report', ok: false, code: 'no_longer_available', ms: 0 },
          { callId: 'c2', name: 'mcp__linear__search_issues', label: 'Linear · search_issues', ok: false, code: 'no_longer_available', ms: 0 },
        ],
      },
    ]
    await renderInWorkspace(<Conversation items={items} />)
    expect(screen.getByRole('button', { name: 'Read report · no longer available to you' })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Linear · search_issues · no longer available to you' })).toBeInTheDocument()
  })

  it('shows a left-out summary as the marker, without its text', async () => {
    const placeholder = 'An earlier part of this conversation used something the person can no longer read, so it is left out.'
    const items = buildItems({ ...initialThreadState, messages: [{ id: 's1', role: 'summary', text: placeholder, createdAt: at }, { id: 'u1', role: 'user', text: 'Next?', createdAt: at }] })
    await renderInWorkspace(<Conversation items={items} />)
    expect(screen.getByRole('separator', { name: 'Earlier messages are summarized' })).toBeInTheDocument()
    expect(screen.queryByText(/left out/)).toBeNull()
  })
})
