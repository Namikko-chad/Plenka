import { request } from './http'
import type { SearchResponse } from './types'

export function searchReleases(
  q: string,
  opts: { indexers?: string[]; cat?: number[]; signal?: AbortSignal; timeoutSec?: number } = {},
): Promise<SearchResponse> {
  return request<SearchResponse>('/api/search', {
    query: { q, indexers: opts.indexers?.join(','), cat: opts.cat?.join(',') },
    signal: opts.signal,
    // the server waits up to its own search timeout for slow trackers
    timeout: ((opts.timeoutSec ?? 30) + 15) * 1000,
  }).then((r) => ({ results: r?.results ?? [], indexers: r?.indexers ?? [] }))
}
