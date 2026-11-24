import { useCallback, useEffect, useState } from 'react'
import { api, type HandsQuery } from '../api'
import type { CategoryRow, HandsResponse } from '../types'
import { Cards } from './Cards'

const PAGE_SIZE = 50

interface Props {
  runID: string
  players: number
  categories: CategoryRow[]
}

export function HandsExplorer({ runID, players, categories }: Props) {
  const [runId, setRunId] = useState('')
  const [seat, setSeat] = useState('')
  const [won, setWon] = useState('')
  const [handType, setHandType] = useState('')
  const [offset, setOffset] = useState(0)
  const [data, setData] = useState<HandsResponse | null>(null)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const load = useCallback(
    async (off: number) => {
      setLoading(true)
      setError(null)
      try {
        const q: HandsQuery = { offset: off, limit: PAGE_SIZE }
        if (runId.trim() !== '') q.run_id = Number(runId)
        if (seat !== '') q.seat = Number(seat)
        if (won !== '') q.won = won === 'true'
        if (handType !== '') q.hand_type = handType
        const d = await api.getHands(runID, q)
        setData(d)
        setOffset(off)
      } catch (e) {
        setError(e instanceof Error ? e.message : String(e))
      } finally {
        setLoading(false)
      }
    },
    [runID, runId, seat, won, handType],
  )

  useEffect(() => {
    load(0)
  }, [load])

  const winners = data?.rows.filter((r) => r.won).length ?? 0
  const isSingleRun = runId.trim() !== '' && data !== null

  return (
    <div>
      <div className="toolbar filters">
        <label>
          Run ID
          <input
            type="number"
            min={0}
            value={runId}
            onChange={(e) => setRunId(e.target.value)}
            placeholder="e.g. 42"
            onKeyDown={(e) => e.key === 'Enter' && load(0)}
          />
        </label>
        <label>
          Seat
          <select value={seat} onChange={(e) => setSeat(e.target.value)}>
            <option value="">any</option>
            {Array.from({ length: players }, (_, i) => (
              <option key={i} value={i}>
                {i}
              </option>
            ))}
          </select>
        </label>
        <label>
          Result
          <select value={won} onChange={(e) => setWon(e.target.value)}>
            <option value="">any</option>
            <option value="true">won</option>
            <option value="false">lost</option>
          </select>
        </label>
        <label>
          Hand type
          <select value={handType} onChange={(e) => setHandType(e.target.value)}>
            <option value="">any</option>
            {categories.map((c) => (
              <option key={c.hand_type} value={c.hand_type}>
                {c.hand_type}
              </option>
            ))}
          </select>
        </label>
        <button onClick={() => load(0)} disabled={loading}>
          Apply
        </button>
        <button
          onClick={() => {
            setRunId('')
            setSeat('')
            setWon('')
            setHandType('')
            load(0)
          }}
        >
          Reset
        </button>
      </div>

      {error && <div className="error">{error}</div>}
      {loading && <p className="muted">Loading hands…</p>}

      {data && (
        <>
          <div className="toolbar">
            <span className="muted">
              showing {data.offset + 1}–{data.offset + data.rows.length} of {data.total.toLocaleString()} hands
            </span>
            {isSingleRun && (
              <span className="muted">
                {winners} winner{winners === 1 ? '' : 's'}
                {winners > 1 ? ' — split pot' : ''}
              </span>
            )}
          </div>
          <table>
            <thead>
              <tr>
                <th>Run</th>
                <th>Seat</th>
                <th>Pocket</th>
                <th>Board</th>
                <th>Hand</th>
                <th>Result</th>
              </tr>
            </thead>
            <tbody>
              {data.rows.map((r, i) => (
                <tr key={`${r.run_id}-${r.seat}-${i}`}>
                  <td>{r.run_id}</td>
                  <td>{r.seat}</td>
                  <td>
                    <Cards tokens={r.pocket} />
                  </td>
                  <td>
                    <Cards tokens={r.board} />
                  </td>
                  <td>{r.hand_type}</td>
                  <td>
                    <span className={r.won ? 'badge ok' : 'badge'}>{r.won ? 'won' : 'lost'}</span>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
          <div className="pager">
            <button disabled={offset === 0 || loading} onClick={() => load(Math.max(0, offset - PAGE_SIZE))}>
              ← Prev
            </button>
            <span className="muted">
              offset {offset.toLocaleString()}
            </span>
            <button
              disabled={loading || offset + PAGE_SIZE >= data.total}
              onClick={() => load(offset + PAGE_SIZE)}
            >
              Next →
            </button>
          </div>
        </>
      )}
    </div>
  )
}
