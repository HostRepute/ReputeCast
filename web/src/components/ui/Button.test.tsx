import { render, screen } from '@testing-library/react'
import { Button } from './Button'

describe('Button', () => {
  it('spans its container without losing its flex layout', () => {
    render(<Button block>Save</Button>)
    const button = screen.getByRole('button', { name: 'Save' })
    expect(button).toHaveClass('btn', 'wide')
    expect(button).not.toHaveClass('block')
  })
})
