import { afterEach, describe, expect, it, vi } from 'vitest'
import { errorText } from '@/utils/errors'
import { ApiError, request, withQuery } from './http'

afterEach(() => vi.unstubAllGlobals())

const respond = (status: number, body: string) =>
  vi.stubGlobal('fetch', vi.fn(async () => new Response(body || null, { status })))

describe('withQuery', () => {
  it('skips empty values and appends to existing queries', () => {
    expect(withQuery('/a', { q: 'x y', n: 1, e: '', u: undefined })).toBe('/a?q=x+y&n=1')
    expect(withQuery('/a?b=1', { c: true })).toBe('/a?b=1&c=true')
  })
})

describe('request', () => {
  it('parses JSON and tolerates empty bodies', async () => {
    respond(200, '{"ok":1}')
    await expect(request('/x')).resolves.toEqual({ ok: 1 })
    respond(200, '')
    await expect(request('/x')).resolves.toBeUndefined()
  })
  it('turns {error} into ApiError with the server message', async () => {
    respond(400, '{"error":"bad query"}')
    const e = await request('/api/search').catch((x: unknown) => x)
    expect(e).toBeInstanceOf(ApiError)
    expect(errorText(e)).toBe('bad query')
  })
  it('maps statuses and network failures to messages', async () => {
    respond(500, 'oops')
    expect(errorText(await request('/x').catch((x: unknown) => x))).toBe('Server error: 500')
    respond(503, '{"error":"TMDB key is not configured"}')
    expect(errorText(await request('/api/tmdb/3/movie/popular').catch((x: unknown) => x))).toBe(
      'TMDB is not configured — set the key in Settings',
    )
    vi.stubGlobal('fetch', vi.fn(async () => Promise.reject(new TypeError('Failed to fetch'))))
    expect(errorText(await request('/x').catch((x: unknown) => x))).toBe('No connection to the server')
  })
  it('times out', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn(
        (_: string, init: RequestInit) =>
          new Promise((_, reject) => init.signal?.addEventListener('abort', () => reject(new DOMException('', 'AbortError')))),
      ),
    )
    const e = await request('/slow', { timeout: 10 }).catch((x: unknown) => x)
    expect(e).toMatchObject({ kind: 'timeout' })
  })
  it('sends JSON bodies with the right method', async () => {
    const f = vi.fn(async () => new Response('{}', { status: 200 }))
    vi.stubGlobal('fetch', f)
    await request('/api/torrents', { method: 'POST', body: { link: 'abc' } })
    const [, init] = f.mock.calls[0] as unknown as [string, RequestInit]
    expect(init.method).toBe('POST')
    expect(init.body).toBe('{"link":"abc"}')
    expect((init.headers as Record<string, string>)['Content-Type']).toBe('application/json')
  })
})
