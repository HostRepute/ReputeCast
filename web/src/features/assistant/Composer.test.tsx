import { fireEvent, render, screen } from '@testing-library/react'
import { Composer } from './Composer'

function setup(answering = false) {
  const onSend = vi.fn()
  const onStop = vi.fn()
  render(<Composer value="xin chào" onChange={() => {}} onSend={onSend} onStop={onStop} answering={answering} placeholder="Ask" />)
  return { onSend, onStop, area: screen.getByRole('textbox') }
}

describe('Composer', () => {
  it('sends on Enter, never while an input method composes, and adds a line with Shift', () => {
    const { onSend, area } = setup()
    fireEvent.keyDown(area, { key: 'Enter', isComposing: true })
    fireEvent.keyDown(area, { key: 'Enter', keyCode: 229 })
    fireEvent.keyDown(area, { key: 'Enter', shiftKey: true })
    expect(onSend).not.toHaveBeenCalled()
    fireEvent.keyDown(area, { key: 'Enter' })
    expect(onSend).toHaveBeenCalledTimes(1)
  })

  it('says at the composer why the server didn’t take the message, and the text stays', () => {
    const note = 'This message contains an email address. Remove it before sending.'
    render(<Composer value="Write to linh@acme.io" onChange={() => {}} onSend={() => {}} onStop={() => {}} answering={false} note={note} placeholder="Ask" />)
    expect(screen.getByRole('alert')).toHaveTextContent(note)
    expect(screen.getByRole('textbox')).toHaveValue('Write to linh@acme.io')
    expect(screen.getByRole('textbox')).toHaveAccessibleDescription(new RegExp(`^${note}`))
  })

  it('turns Send into Stop while the assistant answers', () => {
    const { onSend, area } = setup(true)
    fireEvent.keyDown(area, { key: 'Enter' })
    expect(onSend).not.toHaveBeenCalled()
    expect(screen.getByRole('button', { name: 'Stop' })).toBeInTheDocument()
  })
})
