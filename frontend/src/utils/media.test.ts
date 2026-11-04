import { describe, expect, it } from 'vitest'
import { detailsFromTmdb, imageUrl, mediaFromTmdb, mediaFromTorrentData, mediaListFromTmdb, ratingTone } from './media'

describe('mediaFromTmdb', () => {
  it('parses movies and tv from multi results, skipping people', () => {
    const list = mediaListFromTmdb({
      results: [
        { id: 1, media_type: 'movie', title: 'Матрица', original_title: 'The Matrix', release_date: '1999-03-31', vote_average: 8.2, poster_path: '/p.jpg' },
        { id: 2, media_type: 'tv', name: 'Тьма', original_name: 'Dark', first_air_date: '2017-12-01' },
        { id: 3, media_type: 'person', name: 'Keanu' },
      ],
    })
    expect(list).toHaveLength(2)
    expect(list[0]).toMatchObject({ id: 1, type: 'movie', title: 'Матрица', originalTitle: 'The Matrix', year: 1999, rating: 8.2, posterPath: '/p.jpg', backdropPath: null })
    expect(list[1]).toMatchObject({ id: 2, type: 'tv', title: 'Тьма', year: 2017, rating: 0 })
  })
  it('uses the given type when media_type is absent and handles missing dates', () => {
    expect(mediaFromTmdb({ id: 7, name: 'X', first_air_date: '' }, 'tv')).toMatchObject({ type: 'tv', year: null })
    expect(mediaFromTmdb({ id: 7, title: 'X' })).toBeNull()
    expect(mediaListFromTmdb(null)).toEqual([])
  })
})

describe('detailsFromTmdb', () => {
  it('extracts genres, runtime, cast (8), countries, tagline', () => {
    const d = detailsFromTmdb(
      {
        id: 5,
        name: 'S',
        first_air_date: '2020-01-01',
        genres: [{ name: 'Драма' }],
        episode_run_time: [50],
        number_of_seasons: 2,
        number_of_episodes: 16,
        tagline: '  ',
        production_countries: [{ name: 'Германия' }],
        credits: { cast: Array.from({ length: 12 }, (_, i) => ({ name: `A${i}` })) },
      },
      'tv',
    )!
    expect(d).toMatchObject({ type: 'tv', genres: ['Драма'], runtime: 50, seasons: 2, episodes: 16, tagline: null, countries: ['Германия'] })
    expect(d.cast).toHaveLength(8)
  })
})

describe('helpers', () => {
  it('builds proxied image URLs', () => {
    expect(imageUrl('/a.jpg', 'w342')).toBe('/img/t/p/w342/a.jpg')
    expect(imageUrl(null, 'w342')).toBeNull()
  })
  it('colours ratings', () => {
    expect([8, 6, 3].map(ratingTone)).toEqual(['ok', 'warning', 'neutral'])
  })
  it('reads our media JSON from torrent data and ignores foreign data', () => {
    const m = { id: 1, type: 'movie', title: 'T', originalTitle: '', year: 2000, posterPath: null, backdropPath: null, overview: '', rating: 1, extra: 1 }
    expect(mediaFromTorrentData(JSON.stringify(m))).toEqual({ ...m, extra: undefined })
    expect(mediaFromTorrentData('{"foo":true}')).toBeNull()
    expect(mediaFromTorrentData('not json')).toBeNull()
    expect(mediaFromTorrentData(undefined)).toBeNull()
  })
})
