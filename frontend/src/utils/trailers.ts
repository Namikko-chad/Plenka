// Trailers of a title: TMDB /videos results (YouTube only), merged over languages and
// sorted like Lampa shows them. Pure functions — see trailers.test.ts.

export interface Trailer {
  key: string // YouTube video id
  name: string
  type: string // Trailer, Teaser, Clip, Featurette, …
  lang: string // ISO 639-1, '' when unknown
  publishedAt: string | null
}

const TYPE_ORDER = ['Trailer', 'Teaser', 'Clip', 'Featurette', 'Behind the Scenes']

interface TmdbVideo {
  key?: unknown
  name?: unknown
  site?: unknown
  type?: unknown
  iso_639_1?: unknown
  published_at?: unknown
}

const str = (v: unknown) => (typeof v === 'string' ? v : '')

/**
 * Merges `/videos` answers (preferred language first), drops non-YouTube videos and
 * duplicates; order: language of the answer, type (trailers first), newest first.
 */
export function trailersFromTmdb(answers: unknown[]): Trailer[] {
  const seen = new Set<string>()
  const out: (Trailer & { rank: number })[] = []
  answers.forEach((a, rank) => {
    const results = (a as { results?: unknown })?.results
    if (!Array.isArray(results)) return
    for (const v of results as TmdbVideo[]) {
      const key = str(v.key)
      if (!key || str(v.site) !== 'YouTube' || seen.has(key)) continue
      seen.add(key)
      out.push({
        key,
        name: str(v.name) || str(v.type),
        type: str(v.type),
        lang: str(v.iso_639_1),
        publishedAt: str(v.published_at) || null,
        rank,
      })
    }
  })
  const typeRank = (t: string) => {
    const i = TYPE_ORDER.indexOf(t)
    return i < 0 ? TYPE_ORDER.length : i
  }
  out.sort(
    (a, b) =>
      a.rank - b.rank ||
      typeRank(a.type) - typeRank(b.type) ||
      (b.publishedAt ?? '').localeCompare(a.publishedAt ?? ''),
  )
  return out.map(({ rank: _rank, ...t }) => t)
}

export const youtubeThumb = (key: string) => `https://i.ytimg.com/vi/${encodeURIComponent(key)}/hqdefault.jpg`
export const youtubeWatch = (key: string) => `https://www.youtube.com/watch?v=${encodeURIComponent(key)}`
export const youtubeEmbed = (key: string) =>
  `https://www.youtube-nocookie.com/embed/${encodeURIComponent(key)}?autoplay=1&rel=0`
