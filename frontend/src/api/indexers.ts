import { request } from './http'
import type { ConfigValue, Indexer, IndexerDetail, IndexerTestResult } from './types'

const base = (id: string) => `/api/indexers/${encodeURIComponent(id)}`

export const indexersApi = {
  list: () => request<Indexer[]>('/api/indexers').then((r) => r ?? []),
  get: (id: string) => request<IndexerDetail>(base(id)),
  update: (id: string, body: { enabled?: boolean; config?: Record<string, ConfigValue> }) =>
    request<IndexerDetail>(base(id), { method: 'PUT', body }),
  test: (id: string) => request<IndexerTestResult>(`${base(id)}/test`, { method: 'POST', timeout: 90_000 }),
}
