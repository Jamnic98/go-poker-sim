import type {
  CategoryRow,
  HandsResponse,
  Job,
  PocketRow,
  RunDetail,
  RunInfo,
  RunRequest,
} from './types'

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const res = await fetch(path, init)
  if (!res.ok) {
    let msg = `${res.status} ${res.statusText}`
    try {
      const body = await res.json()
      if (body && typeof body.error === 'string') msg = body.error
    } catch {
      // keep the status-based message
    }
    throw new Error(msg)
  }
  if (res.status === 204) return undefined as T
  return (await res.json()) as T
}

const enc = (runID: string) => encodeURIComponent(runID)

export interface HandsQuery {
  run_id?: number
  seat?: number
  won?: boolean
  hand_type?: string
  offset?: number
  limit?: number
}

function handsQueryString(q: HandsQuery): string {
  const p = new URLSearchParams()
  if (q.run_id !== undefined) p.set('run_id', String(q.run_id))
  if (q.seat !== undefined) p.set('seat', String(q.seat))
  if (q.won !== undefined) p.set('won', String(q.won))
  if (q.hand_type) p.set('hand_type', q.hand_type)
  if (q.offset !== undefined) p.set('offset', String(q.offset))
  if (q.limit !== undefined) p.set('limit', String(q.limit))
  return p.toString()
}

export const api = {
  listRuns: () => request<{ runs: RunInfo[] }>('/api/runs').then((r) => r.runs),
  getRun: (id: string) => request<RunDetail>(`/api/runs/${enc(id)}`),
  getSummary: (id: string) =>
    request<{ rows: PocketRow[] }>(`/api/runs/${enc(id)}/summary`).then((r) => r.rows),
  getCategories: (id: string) =>
    request<{ rows: CategoryRow[] }>(`/api/runs/${enc(id)}/categories`).then((r) => r.rows),
  getHands: (id: string, q: HandsQuery = {}) =>
    request<HandsResponse>(`/api/runs/${enc(id)}/hands?${handsQueryString(q)}`),
  createRun: (req: RunRequest) =>
    request<Job>('/api/runs', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(req),
    }),
  getJob: (jobId: string) => request<Job>(`/api/jobs/${encodeURIComponent(jobId)}`),
  deleteRun: (id: string) => request<void>(`/api/runs/${enc(id)}`, { method: 'DELETE' }),
}
