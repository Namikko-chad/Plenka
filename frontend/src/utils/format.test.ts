import { describe, expect, it } from 'vitest'
import { formatBytes, formatDate, formatDateTime, formatDuration, formatSpeed, plural } from './format'

describe('formatBytes', () => {
  it('uses Russian units with sensible precision', () => {
    expect(formatBytes(0)).toBe('0 B')
    expect(formatBytes(512)).toBe('512 B')
    expect(formatBytes(1536)).toBe('1.5 KB')
    expect(formatBytes(150 * 1024 * 1024)).toBe('150 MB')
    expect(formatBytes(1.4 * 1024 ** 3)).toBe('1.4 GB')
    expect(formatBytes(2 * 1024 ** 4)).toBe('2.0 TB')
  })
  it('caps at TB and tolerates junk', () => {
    expect(formatBytes(5000 * 1024 ** 4)).toBe('5000 TB')
    expect(formatBytes(undefined)).toBe('0 B')
    expect(formatBytes(-5)).toBe('0 B')
  })
  it('formats speed per second', () => {
    expect(formatSpeed(2.5 * 1024 * 1024)).toBe('2.5 MB/s')
  })
})

describe('formatDuration', () => {
  it('omits hours when zero', () => {
    expect(formatDuration(65)).toBe('01:05')
    expect(formatDuration(3723)).toBe('1:02:03')
    expect(formatDuration(NaN)).toBe('00:00')
  })
})

describe('date formatting', () => {
  it('renders local time as YYYY-MM-DD HH:MM:SS', () => {
    const d = new Date(2026, 5, 15, 14, 30, 5) // local
    expect(formatDateTime(d)).toBe('2026-06-15 14:30:05')
    expect(formatDate(d)).toBe('2026-06-15')
  })
  it('converts UTC ISO strings to the browser zone', () => {
    const iso = '2026-11-03T06:00:00Z'
    const d = new Date(iso)
    const pad = (n: number) => String(n).padStart(2, '0')
    expect(formatDateTime(iso)).toBe(
      `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())} ${pad(d.getHours())}:00:00`,
    )
  })
  it('returns empty for missing / invalid / zero dates', () => {
    expect(formatDateTime(null)).toBe('')
    expect(formatDateTime('nope')).toBe('')
    expect(formatDate('0001-01-01T00:00:00Z')).toBe('')
  })
})

describe('plural', () => {
  it('picks Russian forms', () => {
    const f = (n: number) => plural(n, 'файл', 'файла', 'файлов')
    expect([1, 2, 5, 11, 21, 22, 25, 112].map(f)).toEqual([
      'файл', 'файла', 'файлов', 'файлов', 'файл', 'файла', 'файлов', 'файлов',
    ])
  })
})
