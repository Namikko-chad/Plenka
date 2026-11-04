import { defineStore } from 'pinia'
import { ref } from 'vue'
import { ApiError } from '@/api/http'
import { torrentsApi } from '@/api/torrents'
import type { AddTorrent, HistoryEntry, Torrent } from '@/api/types'

/** Saved torrents of the engine (Torrents tab). */
export const useTorrents = defineStore('torrents', () => {
  const list = ref<Torrent[]>([])
  const loaded = ref(false)
  const error = ref<unknown>(null)

  async function refresh() {
    try {
      list.value = await torrentsApi.list()
      loaded.value = true
      error.value = null
    } catch (e) {
      error.value = e
    }
  }

  function upsert(t: Torrent) {
    const i = list.value.findIndex((x) => x.hash === t.hash)
    if (i >= 0) list.value.splice(i, 1, t)
    else list.value.unshift(t)
  }

  async function add(t: AddTorrent): Promise<Torrent> {
    const st = await torrentsApi.add(t)
    upsert(st)
    return st
  }

  async function upload(file: File, title?: string): Promise<Torrent> {
    const st = await torrentsApi.upload(file, { title })
    upsert(st)
    return st
  }

  async function drop(hash: string) {
    await torrentsApi.drop(hash)
    const t = list.value.find((x) => x.hash === hash)
    if (t) upsert({ ...t, stat: 5, statString: 'Torrent in db', activePeers: 0, downloadSpeed: 0, uploadSpeed: 0 })
  }

  async function preload(hash: string, fileId: number) {
    await torrentsApi.preload(hash, fileId)
  }

  async function downloadAll(hash: string) {
    upsert(await torrentsApi.downloadAll(hash))
  }

  async function cancelDownload(hash: string) {
    await torrentsApi.cancelDownload(hash)
    const t = list.value.find((x) => x.hash === hash)
    if (t) upsert({ ...t, fullDownload: false })
  }

  async function remove(hash: string) {
    await torrentsApi.remove(hash)
    list.value = list.value.filter((x) => x.hash !== hash)
  }

  /**
   * Polls `get` every second until the torrent has metadata (`files`).
   * A 404 re-adds the torrent with `revive` (it was removed from the engine's list).
   */
  async function waitForFiles(
    hash: string,
    opts: { signal: AbortSignal; revive?: AddTorrent; timeoutMs?: number; onStatus?: (s: Torrent) => void },
  ): Promise<Torrent | null> {
    const deadline = Date.now() + (opts.timeoutMs ?? 90_000)
    let st: Torrent
    try {
      st = await torrentsApi.get(hash, opts.signal)
    } catch (e) {
      if (!(e instanceof ApiError && e.status === 404) || !opts.revive) throw e
      st = await add(opts.revive)
    }
    opts.onStatus?.(st)
    while (!st.files.length) {
      if (Date.now() > deadline) return null
      await sleep(1000, opts.signal)
      st = await torrentsApi.get(hash, opts.signal)
      opts.onStatus?.(st)
    }
    upsert(st)
    return st
  }

  return { list, loaded, error, refresh, add, upload, drop, preload, downloadAll, cancelDownload, remove, waitForFiles }
})

function sleep(ms: number, signal: AbortSignal): Promise<void> {
  return new Promise((resolve, reject) => {
    if (signal.aborted) return reject(new DOMException('aborted', 'AbortError'))
    const t = setTimeout(resolve, ms)
    signal.addEventListener('abort', () => (clearTimeout(t), reject(new DOMException('aborted', 'AbortError'))), {
      once: true,
    })
  })
}

/** Re-add payload for a hash the engine no longer has, built from its history entry. */
export function reviveFromHistory(hash: string, h: HistoryEntry | null | undefined): AddTorrent {
  const m = h?.media
  return {
    link: hash,
    title: m?.title || h?.torrentTitle || hash,
    poster: m?.posterPath ? `${location.origin}/img/t/p/w342${m.posterPath}` : '',
    data: m ? JSON.stringify(m) : '',
  }
}
