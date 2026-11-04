import { describe, expect, it } from 'vitest'
import { trailersFromTmdb, youtubeWatch } from './trailers'

const v = (key: string, type: string, lang: string, date: string, site = 'YouTube') => ({
  key,
  name: `${type} ${key}`,
  site,
  type,
  iso_639_1: lang,
  published_at: date,
})

describe('trailersFromTmdb', () => {
  it('merges languages, drops duplicates and other sites, sorts by language, type, date', () => {
    const ru = { results: [v('t1', 'Teaser', 'ru', '2026-01-01'), v('r1', 'Trailer', 'ru', '2026-01-01')] }
    const en = {
      results: [
        v('e0', 'Clip', 'en', '2026-03-01'),
        v('e1', 'Trailer', 'en', '2026-01-01'),
        v('e2', 'Trailer', 'en', '2026-02-01'),
        v('r1', 'Trailer', 'en', '2026-01-01'),
        v('vm', 'Trailer', 'en', '2026-01-01', 'Vimeo'),
      ],
    }
    expect(trailersFromTmdb([ru, en]).map((t) => t.key)).toEqual(['r1', 't1', 'e2', 'e1', 'e0'])
  })
  it('tolerates broken answers', () => {
    expect(trailersFromTmdb([null, {}, { results: [{ site: 'YouTube' }] }])).toEqual([])
  })
  it('builds YouTube links', () => {
    expect(youtubeWatch('a b')).toBe('https://www.youtube.com/watch?v=a%20b')
  })
})
