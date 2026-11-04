// Torrent engine settings (docs/en/31-rest-api.md §1.10).
import { request } from './http'
import type { EngineSettings } from './types'

const url = '/api/engine/settings'

export const engineApi = {
  settings: () => request<EngineSettings>(url),
  /** Returns the saved (clamped) settings; the engine restarts. */
  save: (s: EngineSettings) => request<EngineSettings>(url, { method: 'PUT', body: s, timeout: 30_000 }),
  /** Resets to defaults and returns them. */
  reset: () => request<EngineSettings>(url, { method: 'DELETE', timeout: 30_000 }),
}
