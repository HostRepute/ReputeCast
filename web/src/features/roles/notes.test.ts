import { t } from '@/i18n'
import { editorNote, groupNote, matrixNote } from './notes'

describe('role notes', () => {
  it('says people-only permissions are kept from keys and agents', () => {
    const p = { key: 'integrations:manage', peopleOnly: true }
    expect(t(editorNote(p).aside!)).toBe("people only: keys and agents can't hold it")
    expect(editorNote(p).line).toBeUndefined()
    expect(t(matrixNote(p).suffix!)).toBe('· people only')
  })

  it('gives the AI permission its own line instead', () => {
    const p = { key: 'ai:use', peopleOnly: true }
    expect(editorNote(p).aside).toBeUndefined()
    expect(t(editorNote(p).line!)).toBe("Subject ideas, rewrites, translations and alt text with the workspace's own model. Not offered to API keys and agents.")
    expect(matrixNote(p).suffix).toBeUndefined()
    expect(t(matrixNote(p).line!)).toBe("Not sensitive. API keys and MCP agents can't hold it.")
  })

  it('adds nothing to other permissions', () => {
    expect(editorNote({ key: 'subscribers:verify', peopleOnly: false })).toEqual({})
    expect(matrixNote({ key: 'campaigns:send', peopleOnly: false })).toEqual({})
  })

  it("adds Earn's note about the Sponsor and Upgrade blocks", () => {
    expect(t(groupNote('Earn')!)).toBe('Adding Sponsor and Upgrade blocks and paid-only sections to drafts needs only campaigns:edit; prices stay hidden without earn:view.')
    expect(groupNote('Grow')).toBeUndefined()
  })
})
