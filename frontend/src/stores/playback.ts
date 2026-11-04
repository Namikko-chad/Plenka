import { defineStore } from 'pinia'
import { ref } from 'vue'
import type { Media } from '@/api/types'
import router from '@/router'
import { useHistory } from './history'
import { usePrefs } from './prefs'

export interface PlayTarget {
  hash: string
  fileId: number
  fileName: string
  torrentTitle: string
  media: Media | null
}

/** Starts playback in the built-in or an external player according to preferences. */
export const usePlayback = defineStore('playback', () => {
  /** Target shown in the external-player sheet. */
  const external = ref<PlayTarget | null>(null)
  /** Pending "Open with?" choice. */
  const asking = ref<{ target: PlayTarget; replace: boolean } | null>(null)

  function openInternal(t: PlayTarget, replace: boolean) {
    const to = { name: 'player', params: { hash: t.hash, fileId: String(t.fileId) } }
    return replace ? router.replace(to) : router.push(to)
  }

  function openExternal(t: PlayTarget) {
    // The external player's position is unknown, but the entry keeps "Continue watching".
    const history = useHistory()
    const prev = history.entry(t.hash, t.fileId)
    void history.save({
      hash: t.hash,
      fileId: t.fileId,
      fileName: t.fileName,
      torrentTitle: t.torrentTitle,
      media: t.media,
      positionMs: prev?.positionMs ?? 0,
      durationMs: prev?.durationMs ?? 0,
    })
    external.value = t
  }

  function start(t: PlayTarget, opts: { replace?: boolean } = {}) {
    const mode = usePrefs().playerMode
    if (mode === 'external') return openExternal(t)
    if (mode === 'ask') {
      asking.value = { target: t, replace: !!opts.replace }
      return
    }
    return openInternal(t, !!opts.replace)
  }

  function choose(kind: 'internal' | 'external' | null) {
    const a = asking.value
    asking.value = null
    if (!a || !kind) return
    if (kind === 'internal') void openInternal(a.target, a.replace)
    else openExternal(a.target)
  }

  return { external, asking, start, choose, openExternal }
})
