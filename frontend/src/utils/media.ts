import type { Media, MediaDetails, MediaType } from '@/api/types'

type Json = Record<string, unknown>

const str = (v: unknown): string => (typeof v === 'string' ? v : '')
const num = (v: unknown): number | null => (typeof v === 'number' && Number.isFinite(v) ? v : null)

/**
 * TMDB list item → Media. /trending and /search/multi carry `media_type`;
 * /movie/* and /tv/* don't, so the caller passes the type. People are skipped.
 */
export function mediaFromTmdb(j: Json, type?: MediaType): Media | null {
  const mt = j.media_type === 'movie' || j.media_type === 'tv' ? j.media_type : j.media_type == null ? type : undefined
  if (!mt || typeof j.id !== 'number') return null
  const isTv = mt === 'tv'
  const date = str(isTv ? j.first_air_date : j.release_date)
  const year = date.length >= 4 ? Number(date.slice(0, 4)) : NaN
  return {
    id: j.id,
    type: mt,
    title: str(isTv ? j.name : j.title),
    originalTitle: str(isTv ? j.original_name : j.original_title),
    year: Number.isFinite(year) ? year : null,
    posterPath: str(j.poster_path) || null,
    backdropPath: str(j.backdrop_path) || null,
    overview: str(j.overview),
    rating: num(j.vote_average) ?? 0,
  }
}

export function mediaListFromTmdb(j: unknown, type?: MediaType): Media[] {
  const results = (j as Json | null)?.results
  if (!Array.isArray(results)) return []
  return results.map((r) => mediaFromTmdb(r as Json, type)).filter((m): m is Media => m !== null)
}

const names = (v: unknown): string[] =>
  Array.isArray(v) ? v.map((x) => str((x as Json)?.name)).filter(Boolean) : []

export function detailsFromTmdb(j: Json, type: MediaType): MediaDetails | null {
  const base = mediaFromTmdb({ ...j, media_type: type })
  if (!base) return null
  const runtimes = Array.isArray(j.episode_run_time) ? (j.episode_run_time as unknown[]) : []
  const tagline = str(j.tagline).trim()
  const credits = j.credits as Json | undefined
  return {
    ...base,
    genres: names(j.genres),
    runtime: num(j.runtime) || num(runtimes[0]) || null,
    seasons: num(j.number_of_seasons),
    episodes: num(j.number_of_episodes),
    tagline: tagline || null,
    cast: names(credits?.cast).slice(0, 8),
    countries: names(j.production_countries),
  }
}

/** Strip details fields so only the history/torrent snapshot is stored. */
export function toMedia(m: Media): Media {
  return {
    id: m.id,
    type: m.type,
    title: m.title,
    originalTitle: m.originalTitle,
    year: m.year,
    posterPath: m.posterPath,
    backdropPath: m.backdropPath,
    overview: m.overview,
    rating: m.rating,
  }
}

/** Torrent `data` field → Media we saved there on add (null for foreign data). */
export function mediaFromTorrentData(data: string | undefined | null): Media | null {
  if (!data) return null
  try {
    const j = JSON.parse(data) as Json
    if (typeof j.id === 'number' && (j.type === 'movie' || j.type === 'tv') && typeof j.title === 'string') {
      return toMedia(j as unknown as Media)
    }
  } catch {
    /* not ours */
  }
  return null
}

export type ImageSize = 'w342' | 'w500' | 'w780' | 'w1280'

export function imageUrl(path: string | null | undefined, size: ImageSize): string | null {
  if (!path) return null
  return `/img/t/p/${size}${path.startsWith('/') ? '' : '/'}${path}`
}

export type Tone = 'ok' | 'error' | 'warning' | 'accent' | 'neutral' | 'active' | 'purple'

export function ratingTone(r: number): Tone {
  return r >= 7 ? 'ok' : r >= 5 ? 'warning' : 'neutral'
}

export function mediaKey(m: Pick<Media, 'type' | 'id'>): string {
  return `${m.type}:${m.id}`
}
