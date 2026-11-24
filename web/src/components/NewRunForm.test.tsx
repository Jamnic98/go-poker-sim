import { fireEvent, render, screen } from '@testing-library/react'
import { describe, expect, it, vi } from 'vitest'
import { NewRunForm, validateForm } from './NewRunForm'

describe('validateForm', () => {
  const base = { players: '6', runs: '1000', workers: '0', chunk: '10000', format: 'csv', seed: '0', tsh: '0' }

  it('accepts defaults and maps them to a request', () => {
    const { req, errors } = validateForm(base)
    expect(errors).toEqual({})
    expect(req).toEqual({ players: 6, runs: 1000, workers: undefined, chunk: undefined, format: undefined, seed: undefined, top_starting_hands: undefined })
  })

  it('rejects out-of-range players and zero runs', () => {
    const { errors } = validateForm({ ...base, players: '1', runs: '0' })
    expect(errors.players).toBeTruthy()
    expect(errors.runs).toBeTruthy()
  })

  it('rejects a bad format', () => {
    const { errors } = validateForm({ ...base, format: 'xml' })
    expect(errors.format).toBeTruthy()
  })

  it('keeps explicit optional values', () => {
    const { req, errors } = validateForm({ ...base, workers: '8', chunk: '500', format: 'jsonl', seed: '42', tsh: '20' })
    expect(errors).toEqual({})
    expect(req).toEqual({ players: 6, runs: 1000, workers: 8, chunk: 500, format: 'jsonl', seed: 42, top_starting_hands: 20 })
  })
})

describe('NewRunForm component', () => {
  it('shows validation errors and does not submit invalid input', () => {
    const onSubmit = vi.fn()
    render(<NewRunForm busy={false} onSubmit={onSubmit} onCancel={() => {}} />)
    fireEvent.change(screen.getByLabelText('Runs'), { target: { value: '0' } })
    fireEvent.click(screen.getByText('Start simulation'))
    expect(onSubmit).not.toHaveBeenCalled()
    expect(screen.getByText(/Runs must be an integer/)).toBeInTheDocument()
  })

  it('submits parsed values', () => {
    const onSubmit = vi.fn()
    render(<NewRunForm busy={false} onSubmit={onSubmit} onCancel={() => {}} />)
    fireEvent.change(screen.getByLabelText('Players'), { target: { value: '4' } })
    fireEvent.change(screen.getByLabelText('Runs'), { target: { value: '250' } })
    fireEvent.change(screen.getByLabelText('Seed'), { target: { value: '7' } })
    fireEvent.click(screen.getByText('Start simulation'))
    expect(onSubmit).toHaveBeenCalledWith(expect.objectContaining({ players: 4, runs: 250, seed: 7 }))
  })

  it('disables the submit button while busy', () => {
    render(<NewRunForm busy={true} onSubmit={() => {}} onCancel={() => {}} />)
    expect(screen.getByText('Starting…')).toBeDisabled()
  })
})
