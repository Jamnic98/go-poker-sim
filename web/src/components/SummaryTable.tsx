import { useMemo, useState } from 'react'
import { formatPct } from '../format'
import type { PocketRow } from '../types'
import { Cards } from './Cards'

type SortKey = 'pocket' | 'hands' | 'wins' | 'win_pct'

const PAGE_SIZE = 25

export function SummaryTable({ rows }: { rows: PocketRow[] }) {
  const [query, setQuery] = useState('')
  const [sortKey, setSortKey] = useState<SortKey>('win_pct')
  const [sortDesc, setSortDesc] = useState(true)
  const [page, setPage] = useState(0)

  const filtered = useMemo(() => {
    const q = query.trim().toUpperCase()
    const base = q ? rows.filter((r) => r.pocket.toUpperCase().includes(q)) : rows
    const sorted = [...base].sort((a, b) => {
      let cmp: number
      if (sortKey === 'pocket') cmp = a.pocket.localeCompare(b.pocket)
      else cmp = a[sortKey] - b[sortKey]
      return sortDesc ? -cmp : cmp
    })
    return sorted
  }, [rows, query, sortKey, sortDesc])

  const pages = Math.max(1, Math.ceil(filtered.length / PAGE_SIZE))
  const safePage = Math.min(page, pages - 1)
  const pageRows = filtered.slice(safePage * PAGE_SIZE, (safePage + 1) * PAGE_SIZE)

  const toggleSort = (key: SortKey) => {
    if (key === sortKey) {
      setSortDesc((d) => !d)
    } else {
      setSortKey(key)
      setSortDesc(key !== 'pocket')
    }
    setPage(0)
  }

  const header = (key: SortKey, label: string) => (
    <th onClick={() => toggleSort(key)} className="sortable">
      {label}
      {sortKey === key ? (sortDesc ? ' ▼' : ' ▲') : ''}
    </th>
  )

  return (
    <div>
      <div className="toolbar">
        <input
          placeholder="Filter pocket, e.g. AH or AA…"
          value={query}
          onChange={(e) => {
            setQuery(e.target.value)
            setPage(0)
          }}
        />
        <span className="muted">
          {filtered.length} pocket{filtered.length === 1 ? '' : 's'}
        </span>
      </div>
      <table>
        <thead>
          <tr>
            {header('pocket', 'Pocket')}
            {header('hands', 'Hands')}
            {header('wins', 'Wins')}
            {header('win_pct', 'Win %')}
          </tr>
        </thead>
        <tbody>
          {pageRows.map((r) => (
            <tr key={r.pocket}>
              <td>
                <Cards tokens={r.pocket.split(' ')} />
              </td>
              <td>{r.hands.toLocaleString()}</td>
              <td>{r.wins.toLocaleString()}</td>
              <td>
                <div className="pct-cell">
                  <div className="pct-bar" style={{ width: `${Math.min(100, r.win_pct)}%` }} />
                  <span>{formatPct(r.win_pct)}</span>
                </div>
              </td>
            </tr>
          ))}
        </tbody>
      </table>
      <div className="pager">
        <button disabled={safePage === 0} onClick={() => setPage(safePage - 1)}>
          ← Prev
        </button>
        <span className="muted">
          page {safePage + 1} / {pages}
        </span>
        <button disabled={safePage >= pages - 1} onClick={() => setPage(safePage + 1)}>
          Next →
        </button>
      </div>
    </div>
  )
}
