import { describe, expect, it } from 'vitest'
import type { HistoryEntry, Media } from '@/api/types'
import { continueWatching, isFinished, lastForMedia, progressOf, resumeSeconds } from './history'

const movie = (id: number, type: Media['type'] = 'movie'): Media => ({
  id,
  type,
  title: `M${id}`,
  originalTitle: '',
  year: null,
  posterPath: null,
  backdropPath: null,
  overview: '',
  rating: 0,
})

let t = 0
const entry = (p: Partial<HistoryEntry>): HistoryEntry => ({
  hash: 'h',
  fileId: 1,
  fileName: 'f.mkv',
  torrentTitle: 't',
  media: null,
  positionMs: 1000,
  durationMs: 10_000,
  updatedAt: new Date(Date.UTC(2026, 0, 1, 0, 0, t++)).toISOString(),
  ...p,
})

describe('progress', () => {
  it('is 0 without duration and clamped to 1', () => {
    expect(progressOf({ positionMs: 5, durationMs: 0 })).toBe(0)
    expect(progressOf({ positionMs: 20, durationMs: 10 })).toBe(1)
    expect(isFinished({ positionMs: 96, durationMs: 100 })).toBe(true)
    expect(isFinished({ positionMs: 95, durationMs: 100 })).toBe(false)
    expect(isFinished({ positionMs: 0, durationMs: 0 })).toBe(false)
  })
})

describe('continueWatching', () => {
  it('keeps the newest entry per media and per hash, newest first', () => {
    const e1 = entry({ hash: 'a', fileId: 1, media: movie(1, 'tv') })
    const e2 = entry({ hash: 'a', fileId: 2, media: movie(1, 'tv') }) // newer episode
    const e3 = entry({ hash: 'x' })
    const e4 = entry({ hash: 'x', fileId: 7 }) // newer, same hash, no media
    const r = continueWatching([e1, e2, e3, e4])
    expect(r).toEqual([e4, e2])
  })
  it('distinguishes movie and tv with the same TMDB id', () => {
    const r = continueWatching([entry({ hash: 'a', media: movie(5, 'movie') }), entry({ hash: 'b', media: movie(5, 'tv') })])
    expect(r).toHaveLength(2)
  })
  it('skips a media whose newest entry is finished (older entries do not resurface)', () => {
    const old = entry({ hash: 'a', fileId: 1, media: movie(2) })
    const done = entry({ hash: 'a', fileId: 2, media: movie(2), positionMs: 9900 })
    expect(continueWatching([old, done])).toEqual([])
  })
  it('respects the limit', () => {
    const many = Array.from({ length: 30 }, (_, i) => entry({ hash: `h${i}` }))
    expect(continueWatching(many, 20)).toHaveLength(20)
  })
  it('groups the same media across different torrents', () => {
    const a = entry({ hash: 'a', media: movie(9) })
    const b = entry({ hash: 'b', media: movie(9) })
    expect(continueWatching([a, b])).toEqual([b])
  })
})

describe('lastForMedia / resumeSeconds', () => {
  it('finds the newest entry for a media', () => {
    const a = entry({ hash: 'a', media: movie(3) })
    const b = entry({ hash: 'b', media: movie(3) })
    expect(lastForMedia([b, a], { type: 'movie', id: 3 })).toBe(b)
    expect(lastForMedia([a], { type: 'tv', id: 3 })).toBeNull()
  })
  it('resumes 3 s earlier only past 10 s and before the end', () => {
    expect(resumeSeconds({ positionMs: 60_000, durationMs: 100_000 })).toBe(57)
    expect(resumeSeconds({ positionMs: 9_000, durationMs: 100_000 })).toBe(0)
    expect(resumeSeconds({ positionMs: 99_000, durationMs: 100_000 })).toBe(0)
    expect(resumeSeconds(null)).toBe(0)
  })
})
