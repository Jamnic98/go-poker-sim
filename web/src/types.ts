export interface Manifest {
  version: number
  players: number
  runs: number
  workers: number
  chunk_size: number
  format: string
  seed: number
  top_starting_hands: number
  status: string
  started_at: string
  finished_at?: string
  duration_ms?: number
  hands?: number
  chunks?: number
  error?: string
}

export interface ChunkInfo {
  name: string
  index: number
  format: string
  size_bytes: number
}

export interface RunInfo {
  id: string
  date: string
  dir: string
  status: string
  players: number
  runs: number
  format?: string
  chunks: ChunkInfo[]
  has_summary: boolean
  manifest?: Manifest
}

export interface FileInfo {
  name: string
  size_bytes: number
}

export interface RunDetail extends RunInfo {
  files: FileInfo[]
}

export interface PocketRow {
  pocket: string
  hands: number
  wins: number
  win_pct: number
}

export interface CategoryRow {
  hand_type: string
  hands: number
  pct: number
}

export interface HandRow {
  run_id: number
  seat: number
  pocket: string[]
  board: string[]
  hand_type: string
  won: boolean
}

export interface HandsResponse {
  total: number
  offset: number
  limit: number
  rows: HandRow[]
}

export interface RunRequest {
  players: number
  runs: number
  workers?: number
  chunk?: number
  format?: string
  seed?: number
  top_starting_hands?: number
}

export interface Job {
  id: string
  status: string
  request: RunRequest
  run_id?: string
  error?: string
  created_at: string
  finished_at?: string
}
