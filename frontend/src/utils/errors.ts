import { ApiError } from '@/api/http'
import { msg } from '@/i18n'

/** Any thrown value → user-facing message in the UI language. */
export function errorText(e: unknown): string {
  if (e instanceof ApiError) {
    if (e.kind === 'network') return msg.errors.network
    if (e.kind === 'timeout') return msg.errors.timeout
    if (e.status === 503 && e.url.includes('/api/tmdb/')) return msg.errors.tmdb
    return e.message || msg.errors.server(e.status)
  }
  if (e instanceof Error && e.message) return e.message
  return msg.errors.unknown
}
