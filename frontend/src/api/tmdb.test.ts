import { afterEach, describe, expect, it, vi } from 'vitest'
import { tmdbLanguage } from '@/i18n'
import { tmdb } from './tmdb'

afterEach(() => vi.unstubAllGlobals())

describe('tmdb', () => {
  it('asks in the interface language unless the caller picks one', async () => {
    const fetch = vi.fn(async () => new Response('{}', { status: 200 }))
    vi.stubGlobal('fetch', fetch)
    await tmdb('3/movie/popular', { page: 1 })
    await tmdb('/3/movie/1/videos', { language: 'en-US' })
    const urls = fetch.mock.calls.map((c) => String((c as unknown[])[0]))
    expect(urls[0]).toBe(`/api/tmdb/3/movie/popular?language=${tmdbLanguage}&page=1`)
    expect(urls[1]).toBe('/api/tmdb/3/movie/1/videos?language=en-US')
  })
})
