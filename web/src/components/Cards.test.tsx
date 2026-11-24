import { render, screen } from '@testing-library/react'
import { describe, expect, it } from 'vitest'
import { Cards } from './Cards'

describe('Cards', () => {
  it('renders rank and suit symbol for each token', () => {
    render(<Cards tokens={['AH', '10D']} />)
    const ah = screen.getByTestId('card-AH')
    expect(ah).toHaveTextContent('A')
    expect(ah).toHaveTextContent('♥')
    const tenD = screen.getByTestId('card-10D')
    expect(tenD).toHaveTextContent('10')
    expect(tenD).toHaveTextContent('♦')
  })

  it('marks hearts and diamonds red, clubs and spades black', () => {
    render(<Cards tokens={['AH', '2S']} />)
    expect(screen.getByTestId('card-AH').className).toContain('red')
    expect(screen.getByTestId('card-2S').className).not.toContain('red')
  })
})
