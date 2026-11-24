import { useState } from 'react'
import type { RunRequest } from '../types'

interface Props {
  busy: boolean
  onSubmit: (req: RunRequest) => void
  onCancel: () => void
}

interface FormState {
  players: string
  runs: string
  workers: string
  chunk: string
  format: string
  seed: string
  tsh: string
}

const INITIAL: FormState = {
  players: '6',
  runs: '100000',
  workers: '0',
  chunk: '10000',
  format: 'csv',
  seed: '0',
  tsh: '0',
}

function parseIntField(errors: Record<string, string>, key: keyof FormState, label: string, raw: string, min: number, max: number): number {
  const v = Number(raw)
  if (raw.trim() === '' || !Number.isInteger(v) || v < min || v > max) {
    errors[key] = `${label} must be an integer between ${min} and ${max}`
    return 0
  }
  return v
}

export function validateForm(s: FormState): { req?: RunRequest; errors: Record<string, string> } {
  const errors: Record<string, string> = {}
  const players = parseIntField(errors, 'players', 'Players', s.players, 2, 10)
  const runs = parseIntField(errors, 'runs', 'Runs', s.runs, 1, 100_000_000)
  const workers = parseIntField(errors, 'workers', 'Workers', s.workers, 0, 1024)
  const chunk = parseIntField(errors, 'chunk', 'Chunk size', s.chunk, 1, 10_000_000)
  const seed = parseIntField(errors, 'seed', 'Seed', s.seed, 0, Number.MAX_SAFE_INTEGER)
  const tsh = parseIntField(errors, 'tsh', 'Top starting hands', s.tsh, 0, 1326)
  if (s.format !== 'csv' && s.format !== 'jsonl') {
    errors.format = 'Format must be csv or jsonl'
  }
  if (Object.keys(errors).length > 0) return { errors }
  return {
    errors,
    req: {
      players,
      runs,
      workers: workers === 0 ? undefined : workers,
      chunk: chunk === 10000 ? undefined : chunk,
      format: s.format === 'csv' ? undefined : s.format,
      seed: seed === 0 ? undefined : seed,
      top_starting_hands: tsh === 0 ? undefined : tsh,
    },
  }
}

export function NewRunForm({ busy, onSubmit, onCancel }: Props) {
  const [form, setForm] = useState<FormState>(INITIAL)
  const [errors, setErrors] = useState<Record<string, string>>({})

  const set = (key: keyof FormState) => (e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>) =>
    setForm((f) => ({ ...f, [key]: e.target.value }))

  const submit = (e: React.FormEvent) => {
    e.preventDefault()
    const { req, errors: errs } = validateForm(form)
    setErrors(errs)
    if (req) onSubmit(req)
  }

  const field = (
    key: keyof FormState,
    label: string,
    opts: { type?: string; placeholder?: string; hint?: string } = {},
  ) => (
    <div className="field">
      <label htmlFor={`f-${key}`}>{label}</label>
      <input id={`f-${key}`} type={opts.type ?? 'number'} value={form[key]} onChange={set(key)} placeholder={opts.placeholder} />
      {errors[key] && <div className="field-error">{errors[key]}</div>}
      {opts.hint && <div className="muted">{opts.hint}</div>}
    </div>
  )

  return (
    <form className="panel form" onSubmit={submit}>
      <h2>New simulation</h2>
      <div className="form-grid">
        {field('players', 'Players', { hint: '2–10' })}
        {field('runs', 'Runs', { hint: 'hands to simulate' })}
        {field('workers', 'Workers', { hint: '0 = auto (NumCPU)' })}
        {field('chunk', 'Chunk size', { hint: 'runs per chunk file' })}
        {field('seed', 'Seed', { hint: '0 = random' })}
        {field('tsh', 'Top starting hands', { hint: '0 = all (max 1326)' })}
        <div className="field">
          <label htmlFor="f-format">Format</label>
          <select id="f-format" value={form.format} onChange={set('format')}>
            <option value="csv">csv</option>
            <option value="jsonl">jsonl</option>
          </select>
          {errors.format && <div className="field-error">{errors.format}</div>}
        </div>
      </div>
      <div className="form-actions">
        <button type="submit" disabled={busy}>
          {busy ? 'Starting…' : 'Start simulation'}
        </button>
        <button type="button" onClick={onCancel}>
          Cancel
        </button>
      </div>
    </form>
  )
}
