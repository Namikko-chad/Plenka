// Torrents (docs/en/31-rest-api.md §1.8).
import { ApiError, request } from './http'
import type { AddTorrent, CacheState, Torrent } from './types'

const base = (hash: string) => `/api/torrents/${encodeURIComponent(hash)}`

/** Raw bytes of a small torrent file (subtitles) through the stream endpoint. */
async function fileBytes(hash: string, fileId: number, signal?: AbortSignal): Promise<ArrayBuffer> {
  const url = `/api/stream/${encodeURIComponent(hash)}/${fileId}`
  let res: Response
  try {
    res = await fetch(url, { signal })
  } catch {
    throw new ApiError(signal?.aborted ? 'aborted' : 'network', signal?.aborted ? 'abort' : 'network', 0, url)
  }
  if (!res.ok) throw new ApiError('', 'http', res.status, url)
  return res.arrayBuffer()
}

export const torrentsApi = {
  fileBytes,
  list: () => request<Torrent[] | null>('/api/torrents').then((r) => r ?? []),
  add: (t: AddTorrent) => request<Torrent>('/api/torrents', { method: 'POST', body: t, timeout: 60_000 }),
  upload: (file: File, opts: { title?: string; poster?: string; data?: string } = {}) => {
    const fd = new FormData()
    fd.append('file', file)
    if (opts.title) fd.append('title', opts.title)
    if (opts.poster) fd.append('poster', opts.poster)
    if (opts.data) fd.append('data', opts.data)
    return request<Torrent>('/api/torrents/upload', { method: 'POST', body: fd, timeout: 60_000 })
  },
  get: (hash: string, signal?: AbortSignal) => request<Torrent>(base(hash), { timeout: 10_000, signal }),
  remove: (hash: string) => request<void>(base(hash), { method: 'DELETE' }),
  drop: (hash: string) => request<void>(`${base(hash)}/drop`, { method: 'POST' }),
  /**
   * Downloads the start and the end of every file to the server disk in the background,
   * this file first. Waits for the metadata, then 202; 507 when it does not fit.
   */
  preload: (hash: string, fileId: number) =>
    request<void>(`${base(hash)}/preload`, { method: 'POST', body: { fileId }, timeout: 75_000 }),
  /** Downloads the whole torrent to the server disk and keeps it there; 507 when it does not fit. */
  downloadAll: (hash: string) => request<Torrent>(`${base(hash)}/download`, { method: 'POST', timeout: 75_000 }),
  /** Stops a full download and deletes what it fetched, except the preloaded parts. */
  cancelDownload: (hash: string) => request<void>(`${base(hash)}/download`, { method: 'DELETE' }),
  cache: (hash: string, signal?: AbortSignal) => request<CacheState>(`${base(hash)}/cache`, { timeout: 10_000, signal }),
}
