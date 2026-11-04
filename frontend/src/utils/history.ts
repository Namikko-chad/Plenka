import type { HistoryEntry, Media } from '@/api/types'
import { mediaKey } from './media'

export function progressOf(e: Pick<HistoryEntry, 'positionMs' | 'durationMs'>): number {
  if (!(e.durationMs > 0)) return 0
  return Math.min(1, Math.max(0, e.positionMs / e.durationMs))
}

/** Watched to the end: less than 5 % left (credits). */
export function isFinished(e: Pick<HistoryEntry, 'positionMs' | 'durationMs'>): boolean {
  return e.durationMs > 0 && progressOf(e) > 0.95
}

/** Group key: the media, or the torrent when there is no media. */
export function groupKey(e: HistoryEntry): string {
  return e.media ? mediaKey(e.media) : `hash:${e.hash}`
}

const time = (e: HistoryEntry) => Date.parse(e.updatedAt) || 0

/**
 * «Continue watching»: the newest entry per media (or per hash), skipped if that
 * entry is finished, newest first, at most `limit`.
 */
export function continueWatching(entries: HistoryEntry[], limit = 20): HistoryEntry[] {
  const sorted = [...entries].sort((a, b) => time(b) - time(a))
  const seen = new Set<string>()
  const out: HistoryEntry[] = []
  for (const e of sorted) {
    const k = groupKey(e)
    if (seen.has(k)) continue
    seen.add(k)
    if (isFinished(e)) continue
    out.push(e)
    if (out.length >= limit) break
  }
  return out
}

/** Last watched file of a movie / series. */
export function lastForMedia(entries: HistoryEntry[], m: Pick<Media, 'type' | 'id'>): HistoryEntry | null {
  const k = mediaKey(m)
  let best: HistoryEntry | null = null
  for (const e of entries) {
    if (e.media && mediaKey(e.media) === k && (!best || time(e) > time(best))) best = e
  }
  return best
}

/** Resume position in seconds: 3 s before the saved point, only if worth resuming. */
export function resumeSeconds(e: Pick<HistoryEntry, 'positionMs' | 'durationMs'> | null | undefined): number {
  if (!e || e.positionMs <= 10_000 || isFinished(e)) return 0
  return (e.positionMs - 3000) / 1000
}
