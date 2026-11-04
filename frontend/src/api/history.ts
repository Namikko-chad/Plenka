import { request } from './http'
import type { HistoryEntry, HistoryInput } from './types'

export const historyApi = {
  list: (limit = 300) => request<HistoryEntry[]>('/api/history', { query: { limit } }).then((r) => r ?? []),
  save: (e: HistoryInput, keepalive = false) =>
    request<HistoryEntry>('/api/history', { method: 'PUT', body: e, keepalive }),
  removeHash: (hash: string) => request<void>(`/api/history/${encodeURIComponent(hash)}`, { method: 'DELETE' }),
}
