import { defineStore } from 'pinia'
import { computed, ref } from 'vue'
import { historyApi } from '@/api/history'
import type { HistoryEntry, HistoryInput, Media } from '@/api/types'
import { continueWatching, groupKey, lastForMedia } from '@/utils/history'

const key = (hash: string, fileId: number) => `${hash}:${fileId}`

/** Watch history shared with the Flutter app (server-side). */
export const useHistory = defineStore('history', () => {
  const entries = ref<HistoryEntry[]>([])
  const loaded = ref(false)
  const loading = ref(false)

  const byKey = computed(() => new Map(entries.value.map((e) => [key(e.hash, e.fileId), e])))
  const continueList = computed(() => continueWatching(entries.value, 20))

  async function load() {
    loading.value = true
    try {
      entries.value = await historyApi.list(300)
      loaded.value = true
    } catch {
      /* "continue watching" stays empty; the next ensure() retries */
    } finally {
      loading.value = false
    }
  }

  async function ensure() {
    if (!loaded.value && !loading.value) await load()
  }

  function entry(hash: string, fileId: number): HistoryEntry | undefined {
    return byKey.value.get(key(hash, fileId))
  }

  function lastFor(m: Pick<Media, 'type' | 'id'>) {
    return lastForMedia(entries.value, m)
  }

  function upsert(e: HistoryEntry) {
    const k = key(e.hash, e.fileId)
    entries.value = [e, ...entries.value.filter((x) => key(x.hash, x.fileId) !== k)]
  }

  /** Saves progress; the local copy is updated optimistically. */
  async function save(input: HistoryInput, keepalive = false) {
    upsert({ ...input, updatedAt: new Date().toISOString() })
    try {
      const saved = await historyApi.save(input, keepalive)
      if (saved && saved.hash) upsert(saved)
    } catch {
      /* best effort: progress saving must never interrupt playback */
    }
  }

  /** Removes a "continue watching" item: every torrent watched for that media. */
  async function removeGroup(e: HistoryEntry) {
    const g = groupKey(e)
    const hashes = [...new Set(entries.value.filter((x) => groupKey(x) === g).map((x) => x.hash))]
    await Promise.all(hashes.map((h) => historyApi.removeHash(h)))
    entries.value = entries.value.filter((x) => !hashes.includes(x.hash))
  }

  return { entries, loaded, loading, continueList, load, ensure, entry, lastFor, save, removeGroup }
})
