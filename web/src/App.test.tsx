import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import App from './App'
import { api } from './api'
import type { RunInfo } from './types'

vi.mock('./api', () => ({
  api: {
    listRuns: vi.fn(),
    getRun: vi.fn(),
    getSummary: vi.fn(),
    getCategories: vi.fn(),
    getHands: vi.fn(),
    createRun: vi.fn(),
    getJob: vi.fn(),
    deleteRun: vi.fn(),
  },
}))

vi.mock('recharts', () => ({
  ResponsiveContainer: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  BarChart: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  Bar: () => null,
  XAxis: () => null,
  YAxis: () => null,
  Tooltip: () => null,
  CartesianGrid: () => null,
}))

const mocked = vi.mocked(api)

const runInfo: RunInfo = {
  id: '2026-01-01/[10:00:00]-2p-100r',
  date: '2026-01-01',
  dir: '[10:00:00]-2p-100r',
  status: 'done',
  players: 2,
  runs: 100,
  format: 'csv',
  chunks: [{ name: 'pre_flop_sim_chunk_0.csv', index: 0, format: 'csv', size_bytes: 4321 }],
  has_summary: true,
  manifest: {
    version: 1,
    players: 2,
    runs: 100,
    workers: 4,
    chunk_size: 10000,
    format: 'csv',
    seed: 42,
    top_starting_hands: 0,
    status: 'done',
    started_at: '2026-01-01T10:00:00Z',
    finished_at: '2026-01-01T10:00:01Z',
    duration_ms: 1000,
    hands: 200,
    chunks: 1,
  },
}

const summaryRows = [
  { pocket: 'AH KD', hands: 10, wins: 8, win_pct: 80 },
  { pocket: '2C 7D', hands: 12, wins: 1, win_pct: 8.33 },
]

const categories = [
  { hand_type: 'Pair', hands: 40, pct: 20 },
  { hand_type: 'High Card', hands: 160, pct: 80 },
]

const handRows = [
  { run_id: 0, seat: 0, pocket: ['AH', 'KD'], board: ['2C', '7D', '9H', 'JS', 'QC'], hand_type: 'Pair', won: true },
  { run_id: 0, seat: 1, pocket: ['2C', '7D'], board: ['2C', '7D', '9H', 'JS', 'QC'], hand_type: 'High Card', won: false },
]

beforeEach(() => {
  vi.clearAllMocks()
  mocked.listRuns.mockResolvedValue([runInfo])
  mocked.getRun.mockResolvedValue({ ...runInfo, files: [{ name: 'summary.csv', size_bytes: 100 }] })
  mocked.getSummary.mockResolvedValue(summaryRows)
  mocked.getCategories.mockResolvedValue(categories)
  mocked.getHands.mockResolvedValue({ total: 2, offset: 0, limit: 50, rows: handRows })
})

describe('App', () => {
  it('lists runs and opens a run detail with summary and hands', async () => {
    render(<App />)
    expect(await screen.findByText('[10:00:00]-2p-100r')).toBeInTheDocument()
    expect(screen.getByText('done')).toBeInTheDocument()
    expect(screen.getByText('200')).toBeInTheDocument() // hands column

    fireEvent.click(screen.getByText('[10:00:00]-2p-100r'))

    expect(await screen.findByText('Starting-hand win rates (2)')).toBeInTheDocument()
    expect(screen.getByText('Hand-by-hand results')).toBeInTheDocument()
    expect(mocked.getRun).toHaveBeenCalledWith(runInfo.id)
    expect(mocked.getSummary).toHaveBeenCalledWith(runInfo.id)
    expect(mocked.getCategories).toHaveBeenCalledWith(runInfo.id)
    expect(mocked.getHands).toHaveBeenCalledWith(runInfo.id, { offset: 0, limit: 50 })

    // summary table rows render pockets as cards (AH appears in summary + hands tables)
    expect(screen.getAllByTestId('card-AH').length).toBeGreaterThan(0)
    // hands explorer shows the board (shared by both seats) and result badges
    expect(screen.getAllByTestId('card-QC').length).toBeGreaterThan(0)
    expect(screen.getAllByText('won').length).toBeGreaterThan(0)
    expect(screen.getAllByText('lost').length).toBeGreaterThan(0)
    expect(screen.getByText(/showing 1–2 of 2 hands/)).toBeInTheDocument()
  })

  it('starts a new simulation from the form and navigates to it when the job finishes', async () => {
    mocked.createRun.mockResolvedValue({
      id: 'job1',
      status: 'running',
      request: { players: 6, runs: 50 },
      created_at: '2026-01-01T11:00:00Z',
    })
    mocked.getJob.mockResolvedValue({
      id: 'job1',
      status: 'done',
      run_id: runInfo.id,
      request: { players: 6, runs: 50 },
      created_at: '2026-01-01T11:00:00Z',
    })

    render(<App />)
    await screen.findByText('[10:00:00]-2p-100r')

    fireEvent.click(screen.getByText('+ New simulation'))
    fireEvent.change(screen.getByLabelText('Runs'), { target: { value: '50' } })
    fireEvent.click(screen.getByText('Start simulation'))

    await waitFor(() =>
      expect(mocked.createRun).toHaveBeenCalledWith(expect.objectContaining({ players: 6, runs: 50 })),
    )
    expect(await screen.findByText('Starting-hand win rates (2)')).toBeInTheDocument()
    expect(mocked.getJob).toHaveBeenCalledWith('job1')
  })

  it('shows an error banner when listing runs fails', async () => {
    mocked.listRuns.mockRejectedValue(new Error('connection refused'))
    render(<App />)
    expect(await screen.findByText(/connection refused/)).toBeInTheDocument()
  })

  it('navigates back to the list', async () => {
    render(<App />)
    fireEvent.click(await screen.findByText('[10:00:00]-2p-100r'))
    await screen.findByText('Hand-by-hand results')
    fireEvent.click(screen.getByText('← Back to runs'))
    expect(await screen.findByText('[10:00:00]-2p-100r')).toBeInTheDocument()
  })
})
