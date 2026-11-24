import { beforeEach, describe, expect, it, vi } from 'vitest'
import { api } from './api'

const fetchMock = vi.fn()

beforeEach(() => {
  fetchMock.mockReset()
  vi.stubGlobal('fetch', fetchMock)
})

function jsonResponse(body: unknown, status = 200) {
  return {
    ok: status >= 200 && status < 300,
    status,
    statusText: status === 200 ? 'OK' : 'Error',
    json: async () => body,
  } as Response
}

describe('api client', () => {
  it('listRuns GETs /api/runs and unwraps the runs array', async () => {
    fetchMock.mockResolvedValue(jsonResponse({ runs: [{ id: 'a' }, { id: 'b' }] }))
    const runs = await api.listRuns()
    expect(fetchMock).toHaveBeenCalledWith('/api/runs', undefined)
    expect(runs).toEqual([{ id: 'a' }, { id: 'b' }])
  })

  it('getRun encodes the run id as one path segment', async () => {
    fetchMock.mockResolvedValue(jsonResponse({ id: 'x' }))
    await api.getRun('2026-01-01/[10:00:00]-2p-100r')
    const [url] = fetchMock.mock.calls[0]
    expect(url).toBe(`/api/runs/${encodeURIComponent('2026-01-01/[10:00:00]-2p-100r')}`)
  })

  it('getHands builds the query string from filters', async () => {
    fetchMock.mockResolvedValue(jsonResponse({ total: 0, offset: 0, limit: 50, rows: [] }))
    await api.getHands('d/e', { run_id: 7, seat: 1, won: true, hand_type: 'Pair', offset: 100, limit: 50 })
    const [url] = fetchMock.mock.calls[0]
    expect(url).toBe(
      `/api/runs/${encodeURIComponent('d/e')}/hands?run_id=7&seat=1&won=true&hand_type=Pair&offset=100&limit=50`,
    )
  })

  it('getHands omits empty filters', async () => {
    fetchMock.mockResolvedValue(jsonResponse({ total: 0, offset: 0, limit: 50, rows: [] }))
    await api.getHands('d/e')
    const [url] = fetchMock.mock.calls[0]
    expect(url).toBe(`/api/runs/${encodeURIComponent('d/e')}/hands?`)
  })

  it('createRun POSTs JSON', async () => {
    fetchMock.mockResolvedValue(jsonResponse({ id: 'job1', status: 'running' }, 202))
    const job = await api.createRun({ players: 6, runs: 1000, format: 'jsonl' })
    const [url, init] = fetchMock.mock.calls[0]
    expect(url).toBe('/api/runs')
    expect(init.method).toBe('POST')
    expect(init.headers).toEqual({ 'Content-Type': 'application/json' })
    expect(JSON.parse(init.body as string)).toEqual({ players: 6, runs: 1000, format: 'jsonl' })
    expect(job.id).toBe('job1')
  })

  it('getJob encodes the job id', async () => {
    fetchMock.mockResolvedValue(jsonResponse({ id: 'abc/123', status: 'done' }))
    await api.getJob('abc/123')
    expect(fetchMock.mock.calls[0][0]).toBe('/api/jobs/abc%2F123')
  })

  it('deleteRun sends DELETE and accepts 204', async () => {
    fetchMock.mockResolvedValue({ ok: true, status: 204, statusText: 'No Content' } as Response)
    await api.deleteRun('d/e')
    const [url, init] = fetchMock.mock.calls[0]
    expect(url).toBe(`/api/runs/${encodeURIComponent('d/e')}`)
    expect(init.method).toBe('DELETE')
  })

  it('throws the server error message on failure', async () => {
    fetchMock.mockResolvedValue(jsonResponse({ error: 'invalid configuration: runs must be greater than 0' }, 400))
    await expect(api.createRun({ players: 2, runs: 0 })).rejects.toThrow(
      'invalid configuration: runs must be greater than 0',
    )
  })

  it('falls back to status text when the body is not JSON', async () => {
    fetchMock.mockResolvedValue({
      ok: false,
      status: 500,
      statusText: 'Internal Server Error',
      json: async () => {
        throw new Error('not json')
      },
    } as unknown as Response)
    await expect(api.listRuns()).rejects.toThrow('500 Internal Server Error')
  })
})
