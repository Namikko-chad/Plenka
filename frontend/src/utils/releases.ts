import type { IndexerStatus, Media, Release, SearchResponse } from '@/api/types'

export type Quality = '4K' | '1080p' | '720p' | 'SD'
const QUALITIES: Quality[] = ['4K', '1080p', '720p', 'SD']

export function qualityOf(title: string): Quality {
  const t = title.toLowerCase()
  if (/2160p|\b4k\b|\buhd\b/.test(t)) return '4K'
  if (t.includes('1080')) return '1080p'
  if (t.includes('720')) return '720p'
  return 'SD'
}

export function isHdr(title: string): boolean {
  return /\b(hdr(10)?|dolby ?vision|dv)\b/i.test(title)
}

export function dedupKey(r: Pick<Release, 'infoHash' | 'title' | 'size'>): string {
  return r.infoHash ? r.infoHash.toLowerCase() : `${r.title}|${r.size}`
}

/** Search queries for a title: localized and original, year only for movies; deduped. */
export function buildQueries(m: Pick<Media, 'type' | 'title' | 'originalTitle' | 'year'>): string[] {
  const year = m.type === 'movie' && m.year ? ` ${m.year}` : ''
  const out: string[] = []
  for (const name of [m.title, m.originalTitle]) {
    const q = `${name.trim()}${year}`.trim()
    if (name.trim() && !out.some((x) => x.toLowerCase() === q.toLowerCase())) out.push(q)
  }
  return out
}

/**
 * Merges several /api/search responses: releases deduped by infoHash (else title+size)
 * keeping the best-seeded copy; indexer statuses combined per id (ok wins, max results).
 */
export function mergeSearch(responses: SearchResponse[]): SearchResponse {
  const byKey = new Map<string, Release>()
  const idx = new Map<string, IndexerStatus>()
  for (const r of responses) {
    for (const rel of r.results ?? []) {
      const k = dedupKey(rel)
      const prev = byKey.get(k)
      if (!prev || rel.seeders > prev.seeders) byKey.set(k, rel)
    }
    for (const s of r.indexers ?? []) {
      const prev = idx.get(s.id)
      if (!prev) {
        idx.set(s.id, { ...s })
      } else {
        const ok = prev.status === 'ok' || s.status === 'ok'
        idx.set(s.id, {
          ...prev,
          status: ok ? 'ok' : 'error',
          results: Math.max(prev.results, s.results),
          error: ok ? '' : prev.error || s.error,
          elapsedMs: Math.max(prev.elapsedMs, s.elapsedMs),
        })
      }
    }
  }
  return { results: sortReleases([...byKey.values()], 'seeders'), indexers: [...idx.values()] }
}

export type ReleaseSort = 'seeders' | 'size' | 'date'

export function sortReleases(list: Release[], by: ReleaseSort): Release[] {
  const date = (r: Release) => (r.publishDate ? Date.parse(r.publishDate) || 0 : 0)
  const key: Record<ReleaseSort, (r: Release) => number> = {
    seeders: (r) => r.seeders,
    size: (r) => r.size,
    date,
  }
  const f = key[by]
  return [...list].sort((a, b) => f(b) - f(a) || b.seeders - a.seeders)
}

/** Qualities present in the list, in canonical order. */
export function presentQualities(list: Release[]): Quality[] {
  const set = new Set(list.map((r) => qualityOf(r.title)))
  return QUALITIES.filter((q) => set.has(q))
}

export function seedTone(seeders: number): 'ok' | 'warning' | 'error' {
  return seeders >= 10 ? 'ok' : seeders > 0 ? 'warning' : 'error'
}
