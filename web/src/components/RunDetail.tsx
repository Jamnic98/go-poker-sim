import { useEffect, useState } from 'react'
import {
  Bar,
  BarChart,
  CartesianGrid,
  ResponsiveContainer,
  Tooltip,
  XAxis,
  YAxis,
} from 'recharts'
import { api } from '../api'
import { formatBytes, formatDateTime, formatDuration } from '../format'
import type { CategoryRow, PocketRow, RunDetail as RunDetailData } from '../types'
import { HandsExplorer } from './HandsExplorer'
import { StatusBadge } from './StatusBadge'
import { SummaryTable } from './SummaryTable'

interface Props {
  id: string
  onBack: () => void
}

export function RunDetail({ id, onBack }: Props) {
  const [detail, setDetail] = useState<RunDetailData | null>(null)
  const [summary, setSummary] = useState<PocketRow[]>([])
  const [categories, setCategories] = useState<CategoryRow[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    let cancelled = false
    setLoading(true)
    Promise.all([api.getRun(id), api.getSummary(id), api.getCategories(id)])
      .then(([d, s, c]) => {
        if (cancelled) return
        setDetail(d)
        setSummary(s)
        setCategories(c)
        setError(null)
      })
      .catch((e) => {
        if (!cancelled) setError(e instanceof Error ? e.message : String(e))
      })
      .finally(() => {
        if (!cancelled) setLoading(false)
      })
    return () => {
      cancelled = true
    }
  }, [id])

  if (loading) return <p className="muted">Loading run…</p>
  if (error) return <div className="error">{error}</div>
  if (!detail) return null

  const m = detail.manifest
  const topPockets = summary.slice(0, 20).map((r) => ({ pocket: r.pocket, 'win %': r.win_pct }))
  const catData = categories.map((c) => ({ name: c.hand_type, '%': c.pct }))

  return (
    <div>
      <div className="toolbar">
        <button onClick={onBack}>← Back to runs</button>
        <h2>{detail.dir}</h2>
        <StatusBadge status={detail.status} />
      </div>

      <div className="panel chips">
        <span>players: {detail.players}</span>
        <span>runs: {detail.runs.toLocaleString()}</span>
        <span>hands: {m?.hands?.toLocaleString() ?? detail.runs * detail.players}</span>
        <span>format: {detail.format ?? '—'}</span>
        <span>seed: {m?.seed ?? '—'}</span>
        <span>workers: {m?.workers ?? '—'}</span>
        <span>chunk: {m?.chunk_size ?? '—'}</span>
        <span>duration: {formatDuration(m?.duration_ms)}</span>
        <span>started: {formatDateTime(m?.started_at)}</span>
        {m?.error && <span className="badge bad">error: {m.error}</span>}
      </div>

      <div className="charts">
        <div className="panel">
          <h3>Hand categories</h3>
          <ResponsiveContainer width="100%" height={320}>
            <BarChart data={catData} layout="vertical" margin={{ left: 16 }}>
              <CartesianGrid strokeDasharray="3 3" stroke="#2a3350" />
              <XAxis type="number" tick={{ fill: '#9aa4c0', fontSize: 12 }} unit="%" />
              <YAxis type="category" dataKey="name" tick={{ fill: '#e6e9f0', fontSize: 12 }} width={120} />
              <Tooltip />
              <Bar dataKey="%" fill="#4f8ef7" />
            </BarChart>
          </ResponsiveContainer>
        </div>
        <div className="panel">
          <h3>Top 20 starting pockets by win %</h3>
          <ResponsiveContainer width="100%" height={320}>
            <BarChart data={topPockets} layout="vertical" margin={{ left: 16 }}>
              <CartesianGrid strokeDasharray="3 3" stroke="#2a3350" />
              <XAxis type="number" domain={[0, 100]} tick={{ fill: '#9aa4c0', fontSize: 12 }} unit="%" />
              <YAxis type="category" dataKey="pocket" tick={{ fill: '#e6e9f0', fontSize: 12 }} width={70} />
              <Tooltip />
              <Bar dataKey="win %" fill="#3ecf8e" />
            </BarChart>
          </ResponsiveContainer>
        </div>
      </div>

      <div className="panel">
        <h3>Files</h3>
        <table>
          <thead>
            <tr>
              <th>Name</th>
              <th>Size</th>
            </tr>
          </thead>
          <tbody>
            {detail.files.map((f) => (
              <tr key={f.name}>
                <td>{f.name}</td>
                <td>{formatBytes(f.size_bytes)}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>

      <div className="panel">
        <h3>Starting-hand win rates ({summary.length})</h3>
        <SummaryTable rows={summary} />
      </div>

      <div className="panel">
        <h3>Hand-by-hand results</h3>
        <HandsExplorer runID={id} players={detail.players} categories={categories} />
      </div>
    </div>
  )
}
