import { defineStore } from 'pinia'
import { ref, watch } from 'vue'

export type PlayerMode = 'internal' | 'external' | 'ask'
const KEY = 'plenka.playerMode'

function read(): PlayerMode {
  try {
    const v = localStorage.getItem(KEY)
    if (v === 'internal' || v === 'external' || v === 'ask') return v
  } catch {
    /* storage unavailable */
  }
  return 'internal'
}

/** Device-local preferences (localStorage). */
export const usePrefs = defineStore('prefs', () => {
  const playerMode = ref<PlayerMode>(read())
  watch(playerMode, (v) => {
    try {
      localStorage.setItem(KEY, v)
    } catch {
      /* ignore */
    }
  })
  return { playerMode }
})
