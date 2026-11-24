import { useCallback, useEffect, useRef, useState } from 'react'
import { api } from './api'
import { NewRunForm } from './components/NewRunForm'
import { RunDetail } from './components/RunDetail'
import { RunsTable } from './components/RunsTable'
import type { Job, RunInfo, RunRequest } from './types'

type View = { kind: 'list' } | { kind: 'detail'; id: string }

const JOB_POLL_MS = 750

export default function App() {
  const [view, setView] = useState<View>({ kind: 'list' })
  const [runs, setRuns] = useState<RunInfo[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [showForm, setShowForm] = useState(false)
  const [submitting, setSubmitting] = useState(false)
  const [job, setJob] = useState<Job | null>(null)
  const pollRef = useRef<number | null>(null)

  const refresh = useCallback(async () => {
    try {
      setRuns(await api.listRuns())
      setError(null)
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e))
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    refresh()
  }, [refresh])

  useEffect(
    () => () => {
      if (pollRef.current !== null) clearTimeout(pollRef.current)
    },
    [],
  )

  const pollJob = useCallback(
    async (jobId: string) => {
      try {
        const j = await api.getJob(jobId)
        setJob(j)
        if (j.status === 'running') {
          pollRef.current = window.setTimeout(() => pollJob(jobId), JOB_POLL_MS)
          return
        }
        await refresh()
        if (j.status === 'done' && j.run_id) {
          setView({ kind: 'detail', id: j.run_id })
        } else if (j.status === 'failed') {
          setError(`simulation failed: ${j.error ?? 'unknown error'}`)
        }
      } catch (e) {
        setError(e instanceof Error ? e.message : String(e))
      }
    },
    [refresh],
  )

  const handleCreate = async (req: RunRequest) => {
    setSubmitting(true)
    setError(null)
    try {
      const j = await api.createRun(req)
      setJob(j)
      setShowForm(false)
      pollJob(j.id)
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e))
    } finally {
      setSubmitting(false)
    }
  }

  const handleDelete = async (id: string) => {
    if (!window.confirm(`Delete run ${id}? This removes the results directory.`)) return
    try {
      await api.deleteRun(id)
      if (view.kind === 'detail' && view.id === id) setView({ kind: 'list' })
      await refresh()
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e))
    }
  }

  return (
    <div className="app">
      <header>
        <h1>♠ go-poker-sim dashboard</h1>
        {job && job.status === 'running' && (
          <div className="banner">simulation running (job {job.id.slice(0, 8)}…) — it will appear in the list when done</div>
        )}
        {job && job.status === 'failed' && <div className="banner bad">simulation failed: {job.error}</div>}
        {error && (
          <div className="error">
            {error} <button onClick={() => setError(null)}>×</button>
          </div>
        )}
      </header>

      {view.kind === 'list' ? (
        <main>
          <div className="toolbar">
            <button className="primary" onClick={() => setShowForm((s) => !s)}>
              + New simulation
            </button>
            <button onClick={refresh}>Refresh</button>
          </div>
          {showForm && (
            <NewRunForm busy={submitting} onSubmit={handleCreate} onCancel={() => setShowForm(false)} />
          )}
          <RunsTable
            runs={runs}
            loading={loading}
            onSelect={(id) => setView({ kind: 'detail', id })}
            onDelete={handleDelete}
          />
        </main>
      ) : (
        <main>
          <RunDetail id={view.id} onBack={() => setView({ kind: 'list' })} />
        </main>
      )}
    </div>
  )
}
