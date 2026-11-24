import { formatDuration } from '../format'
import type { RunInfo } from '../types'
import { StatusBadge } from './StatusBadge'

interface Props {
  runs: RunInfo[]
  loading: boolean
  onSelect: (id: string) => void
  onDelete: (id: string) => void
}

export function RunsTable({ runs, loading, onSelect, onDelete }: Props) {
  if (loading) return <p className="muted">Loading runs…</p>
  if (runs.length === 0) return <p className="muted">No runs yet — start a simulation above.</p>
  return (
    <table className="runs">
      <thead>
        <tr>
          <th>Run</th>
          <th>Status</th>
          <th>Players</th>
          <th>Runs</th>
          <th>Hands</th>
          <th>Format</th>
          <th>Seed</th>
          <th>Duration</th>
          <th>Chunks</th>
          <th></th>
        </tr>
      </thead>
      <tbody>
        {runs.map((r) => (
          <tr key={r.id}>
            <td>
              <button className="link" onClick={() => onSelect(r.id)}>
                {r.dir}
              </button>
              <div className="muted">{r.date}</div>
            </td>
            <td>
              <StatusBadge status={r.status} />
            </td>
            <td>{r.players}</td>
            <td>{r.runs.toLocaleString()}</td>
            <td>{r.manifest?.hands !== undefined ? r.manifest.hands.toLocaleString() : '—'}</td>
            <td>{r.format ?? '—'}</td>
            <td>{r.manifest?.seed ?? '—'}</td>
            <td>{formatDuration(r.manifest?.duration_ms)}</td>
            <td>{r.chunks.length}</td>
            <td>
              <button className="danger" onClick={() => onDelete(r.id)}>
                Delete
              </button>
            </td>
          </tr>
        ))}
      </tbody>
    </table>
  )
}
