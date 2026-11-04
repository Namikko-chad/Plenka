import { defineStore } from 'pinia'
import { ref } from 'vue'
import { tmdb } from '@/api/tmdb'
import { tmdbLanguage } from '@/i18n'
import type { Media, MediaDetails, MediaType } from '@/api/types'
import { msg } from '@/i18n'
import { detailsFromTmdb, mediaListFromTmdb } from '@/utils/media'
import { trailersFromTmdb, type Trailer } from '@/utils/trailers'

export interface HomeRow {
  id: string
  title: string
  path: string
  type?: MediaType
  items: Media[]
  state: 'loading' | 'ready' | 'error'
}

const ROWS: Omit<HomeRow, 'items' | 'state'>[] = [
  { id: 'trending', title: msg.home.trending, path: '3/trending/all/week' },
  { id: 'now', title: msg.home.nowPlaying, path: '3/movie/now_playing', type: 'movie' },
  { id: 'popMovies', title: msg.home.popularMovies, path: '3/movie/popular', type: 'movie' },
  { id: 'popTv', title: msg.home.popularTv, path: '3/tv/popular', type: 'tv' },
  { id: 'topMovies', title: msg.home.topMovies, path: '3/movie/top_rated', type: 'movie' },
  { id: 'topTv', title: msg.home.topTv, path: '3/tv/top_rated', type: 'tv' },
]

/** TMDB catalog: home rows, details (cached per session), search. */
export const useCatalog = defineStore('catalog', () => {
  const rows = ref<HomeRow[]>(ROWS.map((r) => ({ ...r, items: [], state: 'loading' })))
  const homeLoaded = ref(false)
  const lastError = ref<unknown>(null)
  const details = new Map<string, MediaDetails>()
  const trailers = new Map<string, Trailer[]>()

  async function loadHome() {
    rows.value = ROWS.map((r) => ({ ...r, items: [], state: 'loading' }))
    lastError.value = null
    await Promise.all(
      ROWS.map(async (r, i) => {
        try {
          const items = mediaListFromTmdb(await tmdb(r.path, { page: 1 }), r.type)
          rows.value[i] = { ...rows.value[i]!, items, state: 'ready' }
        } catch (e) {
          lastError.value = e
          rows.value[i] = { ...rows.value[i]!, state: 'error' }
        }
      }),
    )
    homeLoaded.value = true
  }

  async function getDetails(type: MediaType, id: number): Promise<MediaDetails> {
    const k = `${type}:${id}`
    const cached = details.get(k)
    if (cached) return cached
    const j = await tmdb(`3/${type}/${id}`, { append_to_response: 'credits' })
    const d = detailsFromTmdb(j, type)
    if (!d) throw new Error(msg.common.nothingFound)
    details.set(k, d)
    return d
  }

  /** YouTube videos of a title: the interface language first, then English. */
  async function getTrailers(type: MediaType, id: number): Promise<Trailer[]> {
    const k = `${type}:${id}`
    const cached = trailers.get(k)
    if (cached) return cached
    const path = `3/${type}/${id}/videos`
    const answers = await Promise.all(tmdbLanguage === 'en-US' ? [tmdb(path)] : [tmdb(path), tmdb(path, { language: 'en-US' })])
    const list = trailersFromTmdb(answers)
    trailers.set(k, list)
    return list
  }

  async function search(query: string, signal?: AbortSignal): Promise<Media[]> {
    const j = await tmdb('3/search/multi', { query, include_adult: false, page: 1 }, signal)
    return mediaListFromTmdb(j)
  }

  return { rows, homeLoaded, lastError, loadHome, getDetails, getTrailers, search }
})
