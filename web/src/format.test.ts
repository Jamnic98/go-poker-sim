import { describe, expect, it } from 'vitest'
import { cardParts, formatBytes, formatDateTime, formatDuration, formatPct, isRedSuit } from './format'

describe('cardParts', () => {
  it('splits rank and suit', () => {
    expect(cardParts('AH')).toEqual({ rank: 'A', suit: 'H' })
    expect(cardParts('10D')).toEqual({ rank: '10', suit: 'D' })
    expect(cardParts('2C')).toEqual({ rank: '2', suit: 'C' })
  })
})

describe('isRedSuit', () => {
  it('hearts and diamonds are red', () => {
    expect(isRedSuit('H')).toBe(true)
    expect(isRedSuit('D')).toBe(true)
    expect(isRedSuit('C')).toBe(false)
    expect(isRedSuit('S')).toBe(false)
  })
})

describe('formatPct', () => {
  it('formats with two decimals', () => {
    expect(formatPct(83.25)).toBe('83.25%')
    expect(formatPct(100)).toBe('100.00%')
    expect(formatPct(0)).toBe('0.00%')
  })
})

describe('formatDuration', () => {
  it('handles missing values', () => {
    expect(formatDuration(undefined)).toBe('—')
  })
  it('formats ms, s and m', () => {
    expect(formatDuration(832)).toBe('832ms')
    expect(formatDuration(1240)).toBe('1.24s')
    expect(formatDuration(125_000)).toBe('2m 5s')
  })
})

describe('formatBytes', () => {
  it('formats sizes', () => {
    expect(formatBytes(512)).toBe('512 B')
    expect(formatBytes(2048)).toBe('2.0 KB')
    expect(formatBytes(5 * 1024 * 1024)).toBe('5.0 MB')
    expect(formatBytes(3 * 1024 ** 3)).toBe('3.00 GB')
  })
})

describe('formatDateTime', () => {
  it('handles missing and invalid values', () => {
    expect(formatDateTime(undefined)).toBe('—')
    expect(formatDateTime('not-a-date')).toBe('not-a-date')
  })
  it('parses ISO timestamps', () => {
    const out = formatDateTime('2026-01-01T10:00:00Z')
    expect(out).not.toBe('2026-01-01T10:00:00Z')
    expect(out.length).toBeGreaterThan(5)
  })
})
