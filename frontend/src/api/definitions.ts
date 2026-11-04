// Tracker definitions (docs/en/31-rest-api.md §1.6).
import { request } from './http'
import type { DefinitionsState } from './types'

export const definitionsApi = {
  get: () => request<DefinitionsState>('/api/definitions'),
  /** Runs (or joins) an update and waits for it — a download, so a long timeout. */
  update: () => request<DefinitionsState>('/api/definitions/update', { method: 'POST', timeout: 150_000 }),
}
