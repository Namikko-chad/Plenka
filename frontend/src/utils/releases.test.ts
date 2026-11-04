import { describe, expect, it } from 'vitest'
import type { Release, SearchResponse } from '@/api/types'
import { buildQueries, dedupKey, isHdr, mergeSearch, presentQualities, qualityOf, seedTone, sortReleases } from './releases'

const rel = (p: Partial<Release>): Release => ({
  title: 'T',
  tracker: 'Rutor',
  trackerId: 'rutor',
  size: 1,
  seeders: 0,
  peers: 0,
  link: 'magnet:?',
  magnet: '',
  infoHash: '',
  details: '',
  publishDate: null,
  category: [],
  categoryDesc: '',
  ...p,
})

describe('quality / HDR detection', () => {
  it('detects quality from the title', () => {
    expect(qualityOf('Dune (2021) 2160p WEB-DL')).toBe('4K')
    expect(qualityOf('Dune 4K HDR')).toBe('4K')
    expect(qualityOf('Dune UHD BDRemux')).toBe('4K')
    expect(qualityOf('Dune BDRip 1080p')).toBe('1080p')
    expect(qualityOf('Dune 720p')).toBe('720p')
    expect(qualityOf('Dune DVDRip')).toBe('SD')
  })
  it('does not see 4K inside other words', () => {
    expect(qualityOf('Mega4kids collection')).toBe('SD')
  })
  it('detects HDR and Dolby Vision as words', () => {
    expect(isHdr('Film 2160p HDR10')).toBe(true)
    expect(isHdr('Film 2160p HDR10+')).toBe(true)
    expect(isHdr('Film HDRip')).toBe(false)
    expect(isHdr('Film 2160p HDR')).toBe(true)
    expect(isHdr('Film Dolby Vision')).toBe(true)
    expect(isHdr('Film DolbyVision')).toBe(true)
    expect(isHdr('Film [DV]')).toBe(true)
    expect(isHdr('DVDRip')).toBe(false)
  })
  it('lists present qualities in canonical order', () => {
    expect(presentQualities([rel({ title: '720p' }), rel({ title: '2160p' }), rel({ title: 'x 720p' })])).toEqual(['4K', '720p'])
  })
})

describe('buildQueries', () => {
  it('adds the year for movies and dedupes equal titles', () => {
    expect(buildQueries({ type: 'movie', title: 'Матрица', originalTitle: 'The Matrix', year: 1999 })).toEqual([
      'Матрица 1999',
      'The Matrix 1999',
    ])
    expect(buildQueries({ type: 'movie', title: 'Dune', originalTitle: 'dune', year: 2021 })).toEqual(['Dune 2021'])
  })
  it('omits the year for series and skips empty titles', () => {
    expect(buildQueries({ type: 'tv', title: 'Тьма', originalTitle: '', year: 2017 })).toEqual(['Тьма'])
  })
})

describe('mergeSearch', () => {
  const a: SearchResponse = {
    results: [
      rel({ title: 'A', infoHash: 'ABC', seeders: 5 }),
      rel({ title: 'B', size: 10, seeders: 1 }),
    ],
    indexers: [
      { id: 'rutor', name: 'Rutor', status: 'ok', results: 2, error: '', elapsedMs: 100 },
      { id: 'kz', name: 'Kinozal', status: 'error', results: 0, error: 'timeout', elapsedMs: 3000 },
    ],
  }
  const b: SearchResponse = {
    results: [
      rel({ title: 'A (other title)', infoHash: 'abc', seeders: 9 }),
      rel({ title: 'B', size: 10, seeders: 0 }),
      rel({ title: 'B', size: 11, seeders: 3 }),
    ],
    indexers: [
      { id: 'rutor', name: 'Rutor', status: 'ok', results: 3, error: '', elapsedMs: 200 },
      { id: 'kz', name: 'Kinozal', status: 'ok', results: 1, error: '', elapsedMs: 900 },
    ],
  }

  it('dedupes by infoHash (case-insensitive) else title+size, keeping the best seeded', () => {
    const r = mergeSearch([a, b])
    expect(r.results.map((x) => [x.title, x.seeders])).toEqual([
      ['A (other title)', 9],
      ['B', 3],
      ['B', 1],
    ])
  })
  it('combines indexer statuses: ok wins, max results', () => {
    const r = mergeSearch([a, b])
    expect(r.indexers.find((i) => i.id === 'kz')).toMatchObject({ status: 'ok', results: 1, error: '' })
    expect(r.indexers.find((i) => i.id === 'rutor')).toMatchObject({ results: 3, elapsedMs: 200 })
  })
  it('keeps the error when every query failed for an indexer', () => {
    const r = mergeSearch([a, { results: [], indexers: [{ ...a.indexers[1]!, error: 'again' }] }])
    expect(r.indexers.find((i) => i.id === 'kz')).toMatchObject({ status: 'error', error: 'timeout' })
  })
  it('dedupKey falls back to title|size', () => {
    expect(dedupKey({ infoHash: '', title: 'X', size: 5 })).toBe('X|5')
  })
})

describe('sortReleases', () => {
  const list = [
    rel({ title: 'old big', size: 100, seeders: 1, publishDate: '2020-01-01T00:00:00Z' }),
    rel({ title: 'new small', size: 10, seeders: 5, publishDate: '2026-01-01T00:00:00Z' }),
    rel({ title: 'no date', size: 50, seeders: 50, publishDate: null }),
  ]
  it('sorts by seeders, size, date (newest first, missing last)', () => {
    expect(sortReleases(list, 'seeders').map((r) => r.title)).toEqual(['no date', 'new small', 'old big'])
    expect(sortReleases(list, 'size').map((r) => r.title)).toEqual(['old big', 'no date', 'new small'])
    expect(sortReleases(list, 'date').map((r) => r.title)).toEqual(['new small', 'old big', 'no date'])
  })
  it('colours seeders', () => {
    expect([10, 3, 0].map(seedTone)).toEqual(['ok', 'warning', 'error'])
  })
})
