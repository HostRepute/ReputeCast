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

  it('turns Send into Stop while the assistant answers', () => {
    const { onSend, area } = setup(true)
    fireEvent.keyDown(area, { key: 'Enter' })
    expect(onSend).not.toHaveBeenCalled()
    expect(screen.getByRole('button', { name: 'Stop' })).toBeInTheDocument()
  })
})
