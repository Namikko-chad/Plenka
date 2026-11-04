import { tmdbLanguage } from '@/i18n'
import { request } from './http'

/** TMDB JSON through the server proxy: path like `3/movie/popular`, in the interface language
 * unless `query.language` says otherwise. */
export function tmdb<T = Record<string, unknown>>(
  path: string,
  query?: Record<string, string | number | boolean>,
  signal?: AbortSignal,
): Promise<T> {
  return request<T>(`/api/tmdb/${path.replace(/^\//, '')}`, { query: { language: tmdbLanguage, ...query }, signal })
}
